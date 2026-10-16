package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/logger"
)

const testServeYML = `
title: test fleet
auth:
  token: tok
  user_header: X-User
  groups_header: X-Groups
  operators: [admins]
jobs:
  - name: drift-site
    playbook: site.yml
    inventory: hosts.yml
    every: 30m
  - name: ssh
    kind: comply
    profile: ssh
`

func TestServeConfigAndAPI(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "serve.yml")
	if err := os.WriteFile(p, []byte(testServeYML), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadServeConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = filepath.Join(dir, "data")
	if cfg.ApprovalTimeout != time.Hour || cfg.Jobs[0].ApprovalTimeout != time.Hour || cfg.Listen != ":8086" || cfg.Jobs[0].Kind != "drift" || !filepath.IsAbs(cfg.Jobs[0].Playbook) || cfg.Jobs[1].Profile != "ssh" {
		t.Errorf("defaults: %+v", cfg)
	}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ran := make(chan string, 4)
	s.runJob = func(j *ServeJob) (*JobResult, error) {
		ran <- j.Name
		return &JobResult{Job: j.Name, Kind: j.Kind, Started: time.Now().UTC(), Status: "drift", Summary: "1 drifting",
			Hosts: map[string]string{"web1": "drift", "web2": "ok"}, Report: json.RawMessage(`{"x":1}`)}, nil
	}
	srv := httptest.NewServer(s.mux())
	defer srv.Close()
	ctx, cancel := contextTimeout(t)
	defer cancel()
	go s.worker(ctx)

	do := func(method, path string, headers map[string]string) (int, []byte) {
		req, _ := http.NewRequest(method, srv.URL+path, nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		return res.StatusCode, body
	}
	if code, _ := do("GET", "/api/jobs", nil); code != http.StatusUnauthorized {
		t.Errorf("no credentials: %d", code)
	}
	if code, body := do("GET", "/api/me", map[string]string{"X-User": "denys", "X-Groups": "users"}); code != 200 || !strings.Contains(string(body), `"operator":false`) {
		t.Errorf("viewer: %d %s", code, body)
	}
	if code, _ := do("POST", "/api/jobs/drift-site/run", map[string]string{"X-User": "denys", "X-Groups": "users"}); code != http.StatusForbidden {
		t.Errorf("a viewer cannot run jobs: %d", code)
	}
	if code, _ := do("POST", "/api/jobs/drift-site/run", map[string]string{"X-User": "denys", "X-Groups": "admins,users"}); code != http.StatusAccepted {
		t.Errorf("an operator by group runs jobs: %d", code)
	}
	select {
	case name := <-ran:
		if name != "drift-site" {
			t.Errorf("ran %s", name)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the job did not run")
	}
	time.Sleep(100 * time.Millisecond)
	code, body := do("GET", "/api/jobs", map[string]string{"Authorization": "Bearer tok"})
	if code != 200 || !strings.Contains(string(body), `"status":"drift"`) || strings.Contains(string(body), `"x":1`) {
		t.Errorf("jobs view carries the last result without its report: %d %s", code, body)
	}
	code, body = do("GET", "/api/hosts", map[string]string{"Authorization": "Bearer tok"})
	if code != 200 || !strings.Contains(string(body), `"host":"web1","status":"drift"`) {
		t.Errorf("hosts: %d %s", code, body)
	}
	var results []JobResult
	_, body = do("GET", "/api/jobs/drift-site/results", map[string]string{"Authorization": "Bearer tok"})
	if err := json.Unmarshal(body, &results); err != nil || len(results) != 1 || results[0].Trigger != "api:denys" {
		t.Fatalf("results: %v %s", err, body)
	}
	if code, body := do("GET", "/api/jobs/drift-site/results/"+results[0].ID, map[string]string{"Authorization": "Bearer tok"}); code != 200 || !strings.Contains(string(body), `"x":1`) {
		t.Errorf("one result with its report: %d %s", code, body)
	}
	if code, body := do("GET", "/metrics", nil); code != 200 || !strings.Contains(string(body), `onigirazu_serve_host_status{host="web1",job="drift-site"} 1`) || strings.Contains(string(body), `onigirazu_serve_job_pending{job="drift-site"}`) {
		t.Errorf("metrics: %d %s", code, body)
	}
	if code, body := do("GET", "/", nil); code != 200 || !strings.Contains(string(body), "<title>test fleet") {
		t.Errorf("ui: %d", code)
	}
	// results survive a restart
	s2, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(s2.results["drift-site"]) != 1 {
		t.Errorf("results reloaded from disk: %d", len(s2.results["drift-site"]))
	}
}

func TestServeOpenWithoutAuth(t *testing.T) {
	cfg := &ServeConfig{Listen: ":0", Title: "x", DataDir: t.TempDir(), Retain: 5}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.identify(httptest.NewRequest("GET", "/api/jobs", nil))
	if err != nil || !id.Operator {
		t.Errorf("no auth configured: everyone is an operator, got %+v %v", id, err)
	}
}

func contextTimeout(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func TestServeApproval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "serve.yml")
	config := strings.Split(testServeYML, "jobs:")[0] + `
approval_timeout: 2h
jobs:
  - name: patch
    kind: apply
    playbook: x.yml
    approval: required
    approval_timeout: 1h
`
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadServeConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ApprovalTimeout != 2*time.Hour || cfg.Jobs[0].ApprovalTimeout != time.Hour {
		t.Fatalf("timeouts: %+v", cfg)
	}
	cfg.DataDir = filepath.Join(dir, "data")
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ran := make(chan struct{}, 4)
	s.runJob = func(j *ServeJob) (*JobResult, error) {
		ran <- struct{}{}
		return &JobResult{Started: time.Now().UTC(), Status: "drift", Summary: "drifting", Hosts: map[string]string{"web": "drift"}, Report: json.RawMessage(`{"x":1}`)}, nil
	}
	ctx, cancel := contextTimeout(t)
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); s.worker(ctx) }()
	defer func() { cancel(); <-done }()
	srv := httptest.NewServer(s.mux())
	defer srv.Close()
	do := func(method, path, body, role string) (int, []byte) {
		t.Helper()
		req, err := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		if role != "" {
			req.Header.Set("X-User", "denys")
			req.Header.Set("X-Groups", role)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode, data
	}
	request := func() string {
		t.Helper()
		code, body := do("POST", "/api/jobs/patch/run", "", "admins")
		var response map[string]string
		if err := json.Unmarshal(body, &response); err != nil || code != 202 || response["pending"] == "" || response["run"] != response["pending"] {
			t.Fatalf("request: %d %s (%v)", code, body, err)
		}
		return response["pending"]
	}
	id := request()
	select {
	case <-ran:
		t.Fatal("pending job ran")
	default:
	}
	if code, body := do("GET", "/api/jobs", "", "users"); code != 200 || !strings.Contains(string(body), `"pending":{"id":"`) || !strings.Contains(string(body), `"status":"pending"`) || !strings.Contains(string(body), `"approval":"required"`) || !strings.Contains(string(body), `"running":false`) {
		t.Fatalf("pending view: %d %s", code, body)
	}
	if code, _ := do("POST", "/api/jobs/patch/run", "", "admins"); code != 409 {
		t.Fatalf("busy: %d", code)
	}
	if code, _ := do("POST", "/api/runs/"+id+"/approve", "", "users"); code != 403 {
		t.Fatalf("viewer: %d", code)
	}
	if code, _ := do("POST", "/api/runs/"+id+"/approve", "", ""); code != 401 {
		t.Fatalf("anonymous: %d", code)
	}
	if code, _ := do("POST", "/api/runs/bogus/approve", "", "admins"); code != 404 {
		t.Fatalf("missing: %d", code)
	}
	if code, body := do("GET", "/metrics", "", ""); code != 200 || !strings.Contains(string(body), `onigirazu_serve_job_pending{job="patch"} 1`) {
		t.Fatalf("pending metric: %d %s", code, body)
	}
	if code, body := do("POST", "/api/runs/"+id+"/approve", `{"comment":"go"}`, "admins"); code != 200 || strings.Contains(string(body), `"report"`) {
		t.Fatalf("approve: %d %s", code, body)
	}
	select {
	case <-ran:
	case <-time.After(3 * time.Second):
		t.Fatal("approved job did not run")
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		code, body := do("GET", "/api/results/"+id, "", "users")
		var r JobResult
		if err := json.Unmarshal(body, &r); err != nil || code != 200 {
			t.Fatalf("result: %d %s (%v)", code, body, err)
		}
		if r.Status == "drift" {
			if r.Approval == nil || r.Approval.Decision != "approved" || r.Approval.By != "denys" || r.Approval.At == nil || r.Approval.Comment != "go" || string(r.Report) != `{"x":1}` || r.Requested.IsZero() {
				t.Fatalf("approved result: %s", body)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("unfinished: %s", body)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, body := do("GET", "/api/jobs", "", "users"); strings.Contains(string(body), `"pending"`) || !strings.Contains(string(body), `"status":"drift"`) {
		t.Fatalf("last executed: %s", body)
	}
	id = request()
	if _, body := do("GET", "/api/hosts", "", "users"); !strings.Contains(string(body), `"host":"web","status":"drift"`) {
		t.Fatalf("pending hid hosts: %s", body)
	}
	if _, body := do("GET", "/api/jobs", "", "users"); !strings.Contains(string(body), `"last":`) || !strings.Contains(string(body), `"status":"drift"`) {
		t.Fatalf("pending hid last result: %s", body)
	}
	if code, body := do("POST", "/api/runs/"+id+"/reject", `{"comment":"wait"}`, "admins"); code != 200 || !strings.Contains(string(body), `"status":"rejected"`) || !strings.Contains(string(body), `"by":"denys"`) || !strings.Contains(string(body), `"comment":"wait"`) {
		t.Fatalf("reject: %d %s", code, body)
	}
	rejected := id
	id = request()
	if id == rejected {
		t.Fatalf("a run id was reused within a second: %s", id)
	}
	expired := s.expirePending(time.Now().Add(2 * time.Hour))
	if len(expired) != 1 || expired[0].ID != id || expired[0].Status != "expired" || expired[0].Approval.Decision != "expired" || expired[0].Approval.By != "" || expired[0].Approval.At == nil {
		t.Fatalf("expired: %+v", expired)
	}
	id = request()
	interrupted := &JobResult{ID: "old-running", Job: "other", Started: time.Now().UTC(), Status: "running"}
	data, err := json.Marshal(interrupted)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(cfg.DataDir, "results", interrupted.ID+".json")
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	gone := &JobResult{ID: "gone-1", Job: "gone", Started: time.Now().UTC(), Status: "pending", Approval: &RunApproval{Required: true, Expires: time.Now().Add(time.Hour)}}
	data, err = json.Marshal(gone)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.DataDir, "results", gone.ID+".json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	s2, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(s2.results["patch"]) != 4 || s2.pending["patch"] == nil || s2.pending["patch"].ID != id || !s2.running["patch"] {
		t.Fatal("pending run lost on restart")
	}
	r := s2.results["other"][0]
	if r.Status != "error" || r.Summary != "interrupted: serve restarted" || r.Requested != r.Started {
		t.Fatalf("interrupted: %+v", r)
	}
	if g := s2.results["gone"][0]; g.Status != "error" || s2.pending["gone"] != nil || s2.running["gone"] {
		t.Fatalf("a pending run of a removed job: %+v", g)
	}
	data, err = os.ReadFile(file)
	if err != nil || !strings.Contains(string(data), `"status":"error"`) {
		t.Fatalf("interrupted file: %s %v", data, err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(config, "approval: required", "approval: maybe", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadServeConfig(path); err == nil || !strings.Contains(err.Error(), `job patch: approval "maybe" (required or empty)`) {
		t.Fatalf("invalid approval: %v", err)
	}
}

func TestServeApprovalQueueFull(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 5, Jobs: []ServeJob{{Name: "patch", Kind: "apply", Approval: "required", ApprovalTimeout: time.Hour}, {Name: "drift", Kind: "drift"}}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	s.queue = make(chan queued)
	if _, err := s.request(&cfg.Jobs[1], "schedule", nil); err != errQueueFull || len(s.results["drift"]) != 0 || s.running["drift"] {
		t.Fatalf("full queue stored a request: %v", err)
	}
	pending, err := s.request(&cfg.Jobs[0], "schedule", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.approve(pending.ID, "denys", "go"); err != errQueueFull {
		t.Fatalf("approve full queue: %v", err)
	}
	r := s.pending["patch"]
	if r == nil || r.Status != "pending" || r.Approval.Decision != "" || r.Approval.At != nil || !s.running["patch"] {
		t.Fatalf("full queue changed pending run: %+v", r)
	}
	file := filepath.Join(cfg.DataDir, "results", pending.ID+".json")
	data, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(data), `"status":"pending"`) {
		t.Fatalf("pending file: %s %v", data, err)
	}
	// a decision after the deadline expires the run instead of queueing it
	s.pending["patch"].Approval.Expires = time.Now().Add(-time.Second)
	if _, err := s.approve(pending.ID, "denys", "late"); !errors.Is(err, errRunExpired) || s.pending["patch"] != nil || s.running["patch"] || s.results["patch"][0].Status != "expired" {
		t.Fatalf("late approval: %v %+v", err, s.results["patch"][0])
	}
	pending, err = s.request(&cfg.Jobs[0], "schedule", nil)
	if err != nil {
		t.Fatal(err)
	}
	s.queue = make(chan queued, 1)
	approved, err := s.approve(pending.ID, "denys", "go")
	if err != nil || approved.Status != "queued" || len(s.results["patch"]) != 2 {
		t.Fatalf("retry: %+v %v", approved, err)
	}
	s2, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if restored := s2.results["patch"][0]; restored.Status != "error" || restored.Approval.Decision != "approved" {
		t.Fatalf("queued restart: %+v", restored)
	}
	// Expired pending runs are persisted during restart.
	r.Status = "pending"
	r.Approval = &RunApproval{Required: true, Expires: time.Now().Add(-time.Hour)}
	s.store(r)
	s3, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if s3.pending["patch"] != nil || s3.running["patch"] || s3.resultByID("patch", r.ID).Status != "expired" {
		t.Fatal("restart did not expire pending run")
	}
}

func TestServeRunIDsUnique(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 1, Jobs: []ServeJob{{Name: "p", Kind: "apply", Approval: "required", ApprovalTimeout: time.Hour}}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for i := 0; i < 5; i++ { // request + reject within one second, retain 1 prunes the rejected records
		r, err := s.request(&cfg.Jobs[0], "schedule", nil)
		if err != nil {
			t.Fatal(err)
		}
		if seen[r.ID] {
			t.Fatalf("id %s given twice", r.ID)
		}
		seen[r.ID] = true
		if _, err := s.reject(r.ID, "denys", ""); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.results["p"]) != 1 {
		t.Fatalf("retain: %d", len(s.results["p"]))
	}
}

func TestServeWorkerError(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 5, Jobs: []ServeJob{{Name: "patch", Kind: "apply"}}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	requested, err := s.request(&cfg.Jobs[0], "schedule", nil)
	if err != nil {
		t.Fatal(err)
	}
	if requested.Status != "queued" || requested.Requested != requested.Started {
		t.Fatalf("queued record: %+v", requested)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.runJob = func(_ *ServeJob) (*JobResult, error) {
		defer cancel()
		s.mu.Lock()
		defer s.mu.Unlock()
		r := s.results["patch"][0]
		if r.Status != "running" || r.Started.Before(r.Requested) {
			t.Errorf("running record: %+v", r)
		}
		return nil, errors.New("execution failed")
	}
	s.worker(ctx)
	list := s.results["patch"]
	if len(list) != 1 || list[0].ID != requested.ID || list[0].Requested != requested.Requested || list[0].Trigger != "schedule" || list[0].Status != "error" || list[0].Summary != "execution failed" || s.running["patch"] {
		t.Fatalf("failed record: %+v", list)
	}
}

func TestServePause(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "serve.yml")
	config := testServeYML + `
  - name: patch
    kind: apply
    playbook: patch.yml
    approval: required
`
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadServeConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = filepath.Join(dir, "data")
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ran := make(chan string, 4)
	s.runJob = func(j *ServeJob) (*JobResult, error) {
		ran <- j.Name
		return &JobResult{Started: time.Now().UTC(), Status: "drift", Summary: "1 drifting", Hosts: map[string]string{"web1": "drift"}}, nil
	}
	ctx, cancel := contextTimeout(t)
	done := make(chan struct{})
	go func() { defer close(done); s.worker(ctx) }()
	defer func() { cancel(); <-done }()
	handler := s.mux()
	do := func(method, path, body, role string, code int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("X-User", "denys")
		req.Header.Set("X-Groups", role)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != code {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		return w.Body.Bytes()
	}
	contains := func(body []byte, want string) {
		t.Helper()
		if !strings.Contains(string(body), want) {
			t.Fatalf("want %s in %s", want, body)
		}
	}
	waitRun := func(want string) {
		t.Helper()
		select {
		case name := <-ran:
			if name != want {
				t.Fatalf("ran %s, want %s", name, want)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("%s did not run", want)
		}
		deadline := time.Now().Add(3 * time.Second)
		for {
			s.mu.Lock()
			busy := s.running[want]
			s.mu.Unlock()
			if !busy {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s did not finish", want)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	noRun := func() {
		t.Helper()
		select {
		case name := <-ran:
			t.Fatalf("%s ran while paused", name)
		case <-time.After(300 * time.Millisecond):
		}
	}
	do("POST", "/api/pause", "", "users", 403)
	do("POST", "/api/resume", "", "users", 403)
	body := do("POST", "/api/pause", `{"reason":"maintenance"}`, "admins", 200)
	contains(body, `"paused":true`)
	contains(body, `"reason":"maintenance"`)
	contains(do("GET", "/api/pause", "", "users", 200), `"reason":"maintenance"`)
	contains(do("GET", "/api/me", "", "users", 200), `"paused":true`)
	contains(do("GET", "/metrics", "", "", 200), "# TYPE onigirazu_serve_paused gauge\nonigirazu_serve_paused 1\n")
	file := filepath.Join(cfg.DataDir, "pause.json")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	contains(data, `"paused":true`)
	info, err := os.Stat(file)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("pause file permissions: %v %v", info, err)
	}
	do("POST", "/api/jobs/drift-site/run", "", "admins", 202)
	noRun()
	var jobs []jobView
	body = do("GET", "/api/jobs", "", "users", 200)
	if err := json.Unmarshal(body, &jobs); err != nil || !jobs[0].Running || !jobs[0].Queued || jobs[0].Name != "drift-site" {
		t.Fatalf("queued jobs: %s (%v)", body, err)
	}
	s2, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/pause", nil)
	req.Header.Set("Authorization", "Bearer tok")
	s2.mux().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("restart pause: %d", w.Code)
	}
	contains(w.Body.Bytes(), `"paused":true`)
	contains(w.Body.Bytes(), `"reason":"maintenance"`)
	// Updating a pause must not strand a worker on the old channel.
	body = do("POST", "/api/pause", `{"reason":"updated"}`, "admins", 200)
	contains(body, `"reason":"updated"`)
	body = do("POST", "/api/resume", "", "admins", 200)
	contains(body, `"paused":false`)
	contains(body, `"by":"denys"`)
	if strings.Contains(string(body), `"reason"`) {
		t.Fatalf("resume kept reason: %s", body)
	}
	waitRun("drift-site")
	contains(do("GET", "/metrics", "", "", 200), "onigirazu_serve_paused 0\n")
	if again := do("POST", "/api/resume", "", "admins", 200); string(again) != string(body) {
		t.Fatalf("resume was not a no-op: %s -> %s", body, again)
	}
	data, err = os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	contains(data, `"paused":false`)
	do("POST", "/api/pause", "", "admins", 200)
	body = do("POST", "/api/jobs/patch/run", "", "admins", 202)
	var response map[string]string
	if err := json.Unmarshal(body, &response); err != nil || response["pending"] == "" {
		t.Fatalf("pending request: %s (%v)", body, err)
	}
	id := response["pending"]
	do("POST", "/api/runs/"+id+"/approve", "", "admins", 200)
	noRun()
	contains(do("GET", "/api/results/"+id, "", "users", 200), `"status":"queued"`)
	do("POST", "/api/resume", "", "admins", 200)
	waitRun("patch")
	// Pending runs still expire during a pause, including scheduled requests.
	do("POST", "/api/pause", "", "admins", 200)
	pending, err := s.request(s.job("ssh"), "schedule", nil)
	if err != nil || pending.Status != "queued" {
		t.Fatalf("scheduled while paused: %+v %v", pending, err)
	}
	noRun()
	pending, err = s.request(s.job("patch"), "schedule", nil)
	if err != nil {
		t.Fatal(err)
	}
	expired := s.expirePending(time.Now().Add(2 * time.Hour))
	if len(expired) != 1 || expired[0].ID != pending.ID || expired[0].Status != "expired" {
		t.Fatalf("expired while paused: %+v", expired)
	}
	cfg.Metrics.Labels = map[string]string{"env": "test"}
	contains(do("GET", "/metrics", "", "", 200), `onigirazu_serve_paused{env="test"} 1`)
	// a pause file that cannot be read pauses the server at start, until an operator resumes
	if err := os.WriteFile(file, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	s3, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if p, _ := s3.paused(); !p.Paused || !strings.Contains(p.Reason, "unreadable") || p.By != "serve" {
		t.Fatalf("unreadable pause file: %+v", p)
	}
	if _, err := s3.setPause(false, "denys", ""); err != nil {
		t.Fatal(err)
	}
	if p, _ := s3.paused(); p.Paused {
		t.Fatalf("resume after a broken file: %+v", p)
	}
	// a pause that cannot be written still holds in this process, with the error
	do("POST", "/api/resume", "", "admins", 200)
	waitRun("ssh") // queued while paused above
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(file, 0o700); err != nil { // the rename onto a directory fails
		t.Fatal(err)
	}
	body = do("POST", "/api/pause", `{"reason":"disk"}`, "admins", 500)
	contains(body, `"paused":true`)
	contains(body, `not saved`)
	if entries := s.actions(1); len(entries) != 1 || entries[0].OK || entries[0].Action != "pause" || !strings.Contains(entries[0].Reason, "not saved") {
		t.Fatalf("a pause that was not saved is logged as such: %+v", entries)
	}
	contains(do("GET", "/api/pause", "", "users", 200), `"paused":true`)
	do("POST", "/api/jobs/drift-site/run", "", "admins", 202)
	noRun()
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	do("POST", "/api/resume", "", "admins", 200)
	waitRun("drift-site")
}

func TestServeNotify(t *testing.T) {
	var mu sync.Mutex
	var messages, failedMessages [][]byte
	record := func(dst *[][]byte, code int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" || !json.Valid(body) {
				t.Errorf("invalid notification: %s %s %s", r.Method, r.Header.Get("Content-Type"), body)
			}
			mu.Lock()
			*dst = append(*dst, body)
			mu.Unlock()
			w.WriteHeader(code)
		}
	}
	recorder := httptest.NewServer(record(&messages, 200))
	defer recorder.Close()
	broken := httptest.NewServer(record(&failedMessages, 500))
	defer broken.Close()
	t.Setenv("SERVE_NOTIFY_URL", recorder.URL+"/secret")
	config := fmt.Sprintf(`
url: http://fleet.test/
auth:
  user_header: X-User
  groups_header: X-Groups
  operators: [admins]
notify:
  - url: %s
    format: json
    on: [start, success, failure]
jobs:
  - name: patch
    kind: apply
    playbook: patch.yml
    limit: web
    approval: required
    notify:
      - url: ${SERVE_NOTIFY_URL}
        on: [pending, start, success, rejected, expired]
  - name: drift-site
    playbook: site.yml
    notify: ["%s"]
`, broken.URL, recorder.URL)
	dir := t.TempDir()
	path := filepath.Join(dir, "serve.yml")
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadServeConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Jobs[0].Notify[0].URL != recorder.URL+"/secret" || cfg.Jobs[1].Notify[0].Format != "mattermost" {
		t.Fatalf("notification defaults/expansion: %+v", cfg.Jobs)
	}
	cfg.DataDir = filepath.Join(dir, "data")
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	s.notifyClient = &http.Client{Timeout: time.Second}
	ctx, cancel := contextTimeout(t)
	defer cancel()
	started := make(chan struct{}, 1)
	finish := make(chan struct{}, 1)
	s.runJob = func(j *ServeJob) (*JobResult, error) {
		started <- struct{}{}
		select {
		case <-finish:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		status := "ok"
		if j.Kind == "drift" {
			status = "drift"
		}
		return &JobResult{Started: time.Now().UTC(), Duration: 12.3, Status: status, Summary: "1 in sync, 0 changed, 0 failed", Hosts: map[string]string{"web1": status}, Report: json.RawMessage(`{"large":"report"}`)}, nil
	}
	done := make(chan struct{})
	go func() { defer close(done); s.worker(ctx) }()
	defer func() { cancel(); <-done; s.notifyWait.Wait() }()
	handler := s.mux()
	do := func(method, path, body string, want int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("X-User", "denys")
		req.Header.Set("X-Groups", "admins")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		return w.Body.Bytes()
	}
	request := func(job string) string {
		t.Helper()
		body := do("POST", "/api/jobs/"+job+"/run", "", 202)
		var response map[string]string
		if err := json.Unmarshal(body, &response); err != nil || response["run"] == "" {
			t.Fatalf("request: %s (%v)", body, err)
		}
		return response["run"]
	}
	waitStart := func() {
		t.Helper()
		select {
		case <-started:
		case <-time.After(3 * time.Second):
			t.Fatal("worker did not start")
		}
		s.notifyWait.Wait()
	}
	waitFinal := func(id, status string) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for {
			body := do("GET", "/api/results/"+id, "", 200)
			var result JobResult
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatal(err)
			}
			if result.Status == status {
				s.notifyWait.Wait()
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("unfinished run: %s", body)
			}
			time.Sleep(time.Millisecond)
		}
	}
	checkMessage := func(index int, event, id, want string) {
		t.Helper()
		s.notifyWait.Wait()
		mu.Lock()
		defer mu.Unlock()
		if len(messages) <= index {
			t.Fatalf("missing %s notification: %q", event, messages)
		}
		var msg struct {
			Username    string `json:"username"`
			Text        string `json:"text"`
			Attachments []struct {
				Title string `json:"title"`
				Link  string `json:"title_link"`
				Text  string `json:"text"`
				Color string `json:"color"`
			} `json:"attachments"`
		}
		body := messages[index]
		if err := json.Unmarshal(body, &msg); err != nil || msg.Username != "onigirazu" || !strings.Contains(msg.Text, want) || len(msg.Attachments) != 1 {
			t.Fatalf("message: %s (%v)", body, err)
		}
		a := msg.Attachments[0]
		if !strings.HasSuffix(a.Title, " — "+event) || a.Link != "http://fleet.test/#run/"+id || !strings.Contains(a.Text, "trigger: api:denys") || a.Color == "" {
			t.Fatalf("attachment: %s", body)
		}
		if event == eventSuccess && (!strings.Contains(a.Text, "hosts: 1 ok") || !strings.Contains(a.Text, "duration: 12.3s") || !strings.Contains(a.Text, "run: "+a.Link)) {
			t.Fatalf("success details: %s", body)
		}
		if event == eventPending && (!strings.Contains(msg.Text, "**patch**") || !strings.Contains(a.Text, "hosts: web") || !strings.Contains(a.Text, "expires:")) {
			t.Fatalf("pending details: %s", body)
		}
	}
	id := request("patch")
	checkMessage(0, eventPending, id, "waits for approval")
	do("POST", "/api/runs/"+id+"/approve", "", 200)
	waitStart()
	checkMessage(1, eventStart, id, "started")
	finish <- struct{}{}
	waitFinal(id, "ok")
	checkMessage(2, eventSuccess, id, "ok —")
	mu.Lock()
	failed := append([][]byte(nil), failedMessages...)
	mu.Unlock()
	if len(failed) != 2 {
		t.Fatalf("500 target: %q", failed)
	}
	for i, event := range []string{eventStart, eventSuccess} {
		var msg struct {
			Event string                                     `json:"event"`
			Job   struct{ Name, Kind, Target, Limit string } `json:"job"`
			Run   JobResult                                  `json:"run"`
			URL   string                                     `json:"url"`
			Text  string                                     `json:"text"`
		}
		if err := json.Unmarshal(failed[i], &msg); err != nil || msg.Event != event || msg.Job.Name != "patch" || msg.Job.Kind != "apply" || msg.Job.Limit != "web" || msg.Job.Target != cfg.Jobs[0].Playbook || msg.URL != "http://fleet.test/#run/"+id || msg.Run.ID != id || msg.Run.Report != nil || strings.Contains(msg.Text, "**") || strings.Contains(msg.Text, ":white_check_mark:") {
			t.Fatalf("generic JSON: %s (%v)", failed[i], err)
		}
	}
	id = request("patch")
	checkMessage(3, eventPending, id, "waits for approval")
	do("POST", "/api/runs/"+id+"/reject", `{"comment":"maintenance"}`, 200)
	checkMessage(4, eventRejected, id, "rejected by denys: maintenance")
	id = request("drift-site")
	waitStart()
	finish <- struct{}{}
	waitFinal(id, "drift")
	checkMessage(5, eventDrift, id, "**drift-site** drift")
	s.notifyWait.Wait()
	mu.Lock()
	count := len(messages)
	mu.Unlock()
	if count != 6 {
		t.Fatalf("default events sent extra messages: %d", count)
	}
	id = request("patch")
	checkMessage(6, eventPending, id, "waits for approval")
	s.expirePending(time.Now().Add(2 * time.Hour))
	checkMessage(7, eventExpired, id, "expired without approval")
	for _, invalid := range []string{"on: [sometimes]", "format: xml", `url: ""`} {
		t.Run(invalid, func(t *testing.T) {
			data := "notify:\n  - url: " + recorder.URL + "\n    " + invalid + "\n"
			if invalid == `url: ""` {
				data = "notify: [{url: \"\"}]\n"
			}
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := loadServeConfig(path); err == nil {
				t.Fatalf("accepted %s", invalid)
			}
		})
	}
}

func TestServeNotifyRouting(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status string
		err    error
		always bool
		on     []string
		want   string
	}{
		{name: "default success", status: "ok"},
		{name: "legacy success", status: "ok", always: true, want: eventSuccess},
		{name: "explicit empty", status: "ok", always: true, on: []string{}},
		{name: "explicit failure", status: "ok", always: true, on: []string{eventFailure}},
		{name: "failed", status: "failed", want: eventFailure},
		{name: "error", err: errors.New("execution failed"), want: eventFailure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var received []string
			transport := serveNotifyTransport(func(r *http.Request) (*http.Response, error) {
				var msg struct {
					Event string    `json:"event"`
					URL   string    `json:"url"`
					Run   JobResult `json:"run"`
				}
				if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
					t.Error(err)
				}
				if msg.URL != "" || msg.Run.Report != nil {
					t.Errorf("unexpected link or report: %+v", msg)
				}
				mu.Lock()
				received = append(received, r.URL.Path+":"+msg.Event)
				mu.Unlock()
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
			})
			cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 5,
				Notify: []NotifyTarget{{URL: "http://notify.test/global", Format: "json", On: tc.on}},
				Jobs: []ServeJob{{Name: "patch", Kind: "apply", NotifyOK: tc.always,
					Notify: []NotifyTarget{{URL: "http://notify.test/job", Format: "json", On: tc.on}}}},
			}
			s, err := newServer(cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			s.notifyClient = &http.Client{Timeout: time.Second, Transport: transport}
			ctx, cancel := contextTimeout(t)
			defer cancel()
			s.runJob = func(_ *ServeJob) (*JobResult, error) {
				cancel()
				return &JobResult{Status: tc.status, Report: json.RawMessage(`{"secret":"report"}`)}, tc.err
			}
			if _, err := s.request(&cfg.Jobs[0], "schedule", nil); err != nil {
				t.Fatal(err)
			}
			s.worker(ctx)
			s.notifyWait.Wait()
			mu.Lock()
			defer mu.Unlock()
			if tc.want == "" {
				if len(received) != 0 {
					t.Fatalf("unexpected notifications: %v", received)
				}
			} else if len(received) != 2 || !slices.Contains(received, "/global:"+tc.want) || !slices.Contains(received, "/job:"+tc.want) {
				t.Fatalf("targets did not add up: %v", received)
			}
		})
	}
}

type serveNotifyTransport func(*http.Request) (*http.Response, error)

func (f serveNotifyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestServeNotifyConfig(t *testing.T) {
	t.Setenv("SERVE_TEST_HOOK", "http://notify.test/secret")
	for _, tc := range []struct {
		name, notify string
		valid        bool
	}{
		{"string", `["${SERVE_TEST_HOOK}"]`, true},
		{"mapping", `[{url: "${SERVE_TEST_HOOK}", format: json, on: [start, success]}]`, true},
		{"event", `[{url: "http://notify.test/secret", on: [sometimes]}]`, false},
		{"format", `[{url: "http://notify.test/secret", format: xml}]`, false},
		{"empty", `[{url: ""}]`, false},
		{"missing environment", `["${SERVE_TEST_HOOK_MISSING}"]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "serve.yml")
			config := "notify: " + tc.notify + "\njobs:\n  - name: patch\n    playbook: patch.yml\n    notify: " + tc.notify + "\n"
			if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := loadServeConfig(path)
			if !tc.valid {
				if err == nil {
					t.Fatal("accepted invalid target")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, target := range []NotifyTarget{cfg.Notify[0], cfg.Jobs[0].Notify[0]} {
				if target.URL != "http://notify.test/secret" || target.Format == "" {
					t.Fatalf("target defaults: %+v", target)
				}
			}
		})
	}
}

func TestServeOperatorsPerJob(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "serve.yml")
	config := `auth:
  token: tok
  tokens: [{name: ci, token: "${SERVE_CI_TOKEN}", groups: [tier2]}]
  user_header: X-User
  groups_header: X-Groups
  operators: [admins, tier0]
jobs:
  - name: cattle
    playbook: cattle.yml
  - name: dc
    playbook: dc.yml
    approval: required
    operators: [tier0]
  - name: web
    playbook: web.yml
    operators: [ci, alice]
`
	t.Setenv("SERVE_CI_TOKEN", "citok")
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadServeConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = filepath.Join(dir, "data")
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := contextTimeout(t)
	s.runJob = func(_ *ServeJob) (*JobResult, error) { return &JobResult{Status: "ok"}, nil }
	done := make(chan struct{})
	go func() { defer close(done); s.worker(ctx) }()
	defer func() { cancel(); <-done }()
	handler := s.mux()
	var want []serveAction
	do := func(method, route, user, groups, token, body string, code int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, route, strings.NewReader(body))
		req.RemoteAddr = "[2001:db8::1]:1234"
		req.Header.Set("X-User", user)
		req.Header.Set("X-Groups", groups)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != code {
			t.Fatalf("%s %s as %s/%s: %d %s", method, route, user, token, w.Code, w.Body.String())
		}
		return w.Body.Bytes()
	}
	post := func(route, user, groups, token, body string, code int) string {
		t.Helper()
		data := do("POST", route, user, groups, token, body, code)
		var response struct {
			Run   string `json:"run"`
			ID    string `json:"id"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatal(err)
		}
		a := serveAction{User: user, Action: route[strings.LastIndex(route, "/")+1:], Run: response.Run, OK: code < 400, Reason: response.Error, Addr: "2001:db8::1"}
		if token == "tok" {
			a.User = "token"
		} else if token == "citok" {
			a.User = "ci"
		}
		if strings.HasPrefix(route, "/api/jobs/") {
			a.Job = strings.Split(route, "/")[3]
		}
		if strings.HasPrefix(route, "/api/runs/") {
			a.Run, a.Job = strings.Split(route, "/")[3], "dc"
		}
		if code != 401 {
			want = append(want, a)
		}
		if code < 400 && ((response.Run != "" && !strings.Contains(string(data), `"pending":`)) || strings.HasSuffix(route, "/approve")) {
			deadline := time.Now().Add(3 * time.Second)
			for {
				s.mu.Lock()
				r := s.resultByID(a.Job, a.Run)
				finished := r != nil && r.Status == "ok"
				s.mu.Unlock()
				if finished {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("run did not finish")
				}
				time.Sleep(time.Millisecond)
			}
		}
		return response.Run
	}
	req := httptest.NewRequest("GET", "/api/jobs", nil)
	req.Header.Set("X-User", "alice")
	req.Header.Set("X-Groups", "admins|Team, users")
	id, err := s.identify(req)
	if err != nil || id.User != "alice" || !id.Operator || !slices.Equal(id.Groups, []string{"admins", "Team", "users"}) || !id.matches([]string{"TEAM"}) || !id.matches([]string{"ALICE"}) {
		t.Fatalf("proxy identity: %+v %v", id, err)
	}
	post("/api/jobs/cattle/run", "alice", "admins", "", "", 202)
	post("/api/jobs/dc/run", "alice", "admins", "", "", 403)
	post("/api/jobs/web/run", "alice", "admins", "", "", 202)
	run := post("/api/jobs/dc/run", "bob", "tier0,admins", "", "", 202)
	post("/api/runs/"+run+"/approve", "alice", "admins", "", "", 403)
	post("/api/runs/"+run+"/approve", "bob", "tier0,admins", "", `{"comment":"go"}`, 200)
	want[len(want)-1].Reason = "go"
	post("/api/jobs/cattle/run", "", "", "tok", "", 202)
	post("/api/jobs/web/run", "", "", "tok", "", 403)
	post("/api/jobs/web/run", "", "", "citok", "", 202)
	post("/api/jobs/dc/run", "", "", "citok", "", 403)
	data := do("GET", "/api/jobs", "alice", "admins", "", "", 200)
	var jobs []jobView
	if err := json.Unmarshal(data, &jobs); err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 3 || !jobs[0].Allowed || jobs[1].Allowed || !jobs[2].Allowed || !slices.Equal(jobs[1].Operators, []string{"tier0"}) {
		t.Fatalf("jobs: %s", data)
	}
	// A named token's groups can match a job, but a viewer's user name cannot grant operator access.
	if !s.allowed(serveIdentity{User: "ci", Groups: []string{"TIER0"}, Operator: true}, &cfg.Jobs[1]) || s.allowed(serveIdentity{User: "alice"}, &cfg.Jobs[2]) {
		t.Fatal("job matching")
	}
	// Token identities take precedence over proxy headers.
	req.Header.Set("Authorization", "Bearer citok")
	id, err = s.identify(req)
	if err != nil || id.User != "ci" || !slices.Equal(id.Groups, []string{"tier2"}) {
		t.Fatalf("token identity: %+v %v", id, err)
	}
	run = post("/api/jobs/dc/run", "bob", "tier0", "", "", 202)
	post("/api/runs/"+run+"/reject", "alice", "admins", "", "", 403)
	post("/api/runs/"+run+"/reject", "bob", "tier0", "", `{"comment":"wait"}`, 200)
	want[len(want)-1].Reason = "wait"
	post("/api/jobs/web/run", "viewer", "users", "", "", 403)
	post("/api/pause", "viewer", "users", "", "", 403)
	post("/api/resume", "viewer", "users", "", "", 403)
	post("/api/runs/"+run+"/approve", "viewer", "users", "", "", 403)
	want[len(want)-1].Job = ""
	post("/api/runs/"+run+"/reject", "viewer", "users", "", "", 403)
	want[len(want)-1].Job = ""
	post("/api/pause", "alice", "admins", "", `{"reason":"maintenance"}`, 200)
	want[len(want)-1].Reason = "maintenance"
	post("/api/resume", "alice", "admins", "", "", 200)
	post("/api/jobs/web/run", "", "", "", "", 401)
	data = do("GET", "/api/audit", "viewer", "users", "", "", 200)
	var actions []serveAction
	if err := json.Unmarshal(data, &actions); err != nil {
		t.Fatal(err)
	}
	if len(actions) != len(want) {
		t.Fatalf("audit count %d, want %d: %s", len(actions), len(want), data)
	}
	for i, a := range actions {
		expected := want[len(want)-1-i]
		if a.Time.IsZero() || a.Time.Location() != time.UTC {
			t.Fatalf("audit time: %v", a.Time)
		}
		if i > 0 && actions[i-1].Time.Before(a.Time) {
			t.Fatal("audit order")
		}
		a.Time = time.Time{}
		if a != expected {
			t.Fatalf("audit %d: %+v, want %+v", i, a, expected)
		}
	}
	data = do("GET", "/api/audit?limit=2", "viewer", "users", "", "", 200)
	if err := json.Unmarshal(data, &actions); err != nil || len(actions) != 2 {
		t.Fatalf("audit limit: %s (%v)", data, err)
	}
	data, err = os.ReadFile(filepath.Join(cfg.DataDir, "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(want) {
		t.Fatalf("audit lines: %d", len(lines))
	}
	for _, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Fatalf("invalid line: %s", line)
		}
	}
	info, err := os.Stat(filepath.Join(cfg.DataDir, "audit.log"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("audit mode: %v %v", info, err)
	}
	do("POST", "/api/runs/unknown/approve", "alice", "admins", "", "", 404)
	for _, tokens := range []string{`[{token: tok}]`, `[{name: ci}]`, `[{name: ci, token: tok}, {name: ci, token: other}]`, `[{name: ci, token: "${SERVE_MISSING_TOKEN}"}]`} {
		t.Run(tokens, func(t *testing.T) {
			if err := os.WriteFile(path, []byte("auth:\n  tokens: "+tokens+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := loadServeConfig(path); err == nil {
				t.Fatal("accepted invalid named token")
			}
		})
	}
	// A configuration with only named tokens still requires an identity.
	s.cfg.Auth = ServeAuth{Tokens: []ServeToken{{Name: "ci", Token: "citok"}}}
	if _, err := s.identify(httptest.NewRequest("GET", "/api/jobs", nil)); !errors.Is(err, errServeAuth) {
		t.Fatalf("named-only auth: %v", err)
	}
}

func TestServeAudit(t *testing.T) {
	var out strings.Builder
	s, err := newServer(&ServeConfig{DataDir: t.TempDir()}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.actions(100); len(got) != 0 {
		t.Fatalf("empty audit: %v", got)
	}
	for i := range 1005 {
		s.logAction(serveAction{User: "ci", Action: "run", Job: "web", Run: fmt.Sprint(i), OK: true, Reason: strings.Repeat("x", 2000)}) // long lines: the tail read cuts the first one
	}
	if out.Len() != 0 {
		t.Fatalf("successful actions printed: %s", out.String())
	}
	s.logAction(serveAction{User: "alice", Action: "run", Job: "dc", Reason: "not an operator of this job"})
	if out.String() != "serve: audit: alice run dc refused: not an operator of this job\n" {
		t.Fatalf("refusal output: %s", out.String())
	}
	handler := s.mux()
	for _, tc := range []struct {
		query string
		count int
	}{
		{"", 100}, {"?limit=2", 2}, {"?limit=5000", 1000}, {"?limit=bad", 100}, {"?limit=-1", 100},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/audit"+tc.query, nil))
		var actions []serveAction
		if err := json.Unmarshal(w.Body.Bytes(), &actions); err != nil || w.Code != 200 || len(actions) != tc.count {
			t.Fatalf("audit %s: %d %s (%v)", tc.query, w.Code, w.Body.String(), err)
		}
		if actions[0].OK || actions[0].User != "alice" || actions[1].Run != "1004" {
			t.Fatalf("audit tail: %v", actions[:2])
		}
	}
}

func TestServeAdhocRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "serve.yml")
	config := `jobs:
  - name: patch
    kind: apply
    playbook: patch.yml
    inventory: hosts.yml
    limit: web
    extra_vars: {reboot: "false", keep: default}
    adhoc: {limit: true, extra_vars: [reboot]}
  - name: fixed
    kind: apply
    playbook: patch.yml
  - name: wave
    kind: apply
    playbook: patch.yml
    inventory: hosts.yml
    adhoc: {limit: true}
    approval: required
  - name: local
    kind: apply
    playbook: patch.yml
    adhoc: {limit: true}
`
	inventory := "web:\n  hosts:\n    web1: {}\n    web2: {}\ndb:\n  hosts:\n    db1: {}\n"
	for name, data := range map[string]string{"serve.yml": config, "hosts.yml": inventory, "inventory.yml": inventory, "patch.yml": "- hosts: all\n  tasks: []\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := loadServeConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DataDir = filepath.Join(dir, "data")
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ran := make(chan ServeJob, 8)
	s.runJob = func(j *ServeJob) (*JobResult, error) {
		ran <- *j
		return &JobResult{Started: time.Now().UTC(), Status: "ok"}, nil
	}
	ctx, cancel := contextTimeout(t)
	done := make(chan struct{})
	go func() { defer close(done); s.worker(ctx) }()
	defer func() { cancel(); <-done; s.notifyWait.Wait() }()
	handler := s.mux()
	do := func(method, path, body string, code int) []byte {
		t.Helper()
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		if w.Code != code {
			t.Fatalf("%s %s: %d %s, want %d", method, path, w.Code, w.Body.String(), code)
		}
		return w.Body.Bytes()
	}
	post := func(job, body string) string {
		t.Helper()
		data := do("POST", "/api/jobs/"+job+"/run", body, 202)
		var response map[string]string
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatal(err)
		}
		return response["run"]
	}
	wait := func(name string) ServeJob {
		t.Helper()
		var job ServeJob
		select {
		case job = <-ran:
			if job.Name != name {
				t.Fatalf("ran %s, want %s", job.Name, name)
			}
		case <-ctx.Done():
			t.Fatal("job did not run")
		}
		for {
			s.mu.Lock()
			busy := s.running[name]
			s.mu.Unlock()
			if !busy {
				return job
			}
			select {
			case <-ctx.Done():
				t.Fatal("job did not finish")
			case <-time.After(time.Millisecond):
			}
		}
	}
	id := post("patch", `{"limit":"web1","extra_vars":{"reboot":"true"}}`)
	job := wait("patch")
	if job.Limit != "web1" || job.ExtraVars["reboot"] != "true" || job.ExtraVars["keep"] != "default" {
		t.Fatalf("effective job: %+v", job)
	}
	if cfg.Jobs[0].Limit != "web" || cfg.Jobs[0].ExtraVars["reboot"] != "false" {
		t.Fatalf("configured job changed: %+v", cfg.Jobs[0])
	}
	data := do("GET", "/api/results/"+id, "", 200)
	var result JobResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.Params == nil || result.Params.Limit != "web1" || result.Params.ExtraVars["reboot"] != "true" || !slices.Equal(result.Params.Hosts, []string{"web1"}) {
		t.Fatalf("params: %s", data)
	}
	actions := s.actions(100)
	if len(actions) != 1 || actions[0].Run != id || actions[0].Reason != "limit=web1 reboot=true" {
		t.Fatalf("actions: %+v", actions)
	}
	for _, tc := range []struct{ job, body, message string }{
		{"patch", `{"limit":"db1"}`, "outside the job's limit"},
		{"patch", `{"limit":"nope"}`, "matches no host"},
		{"patch", `{"extra_vars":{"serial":"1"}}`, "not allowed"},
		{"patch", `{"limit":5}`, ""},
		{"patch", `{"hosts":["web1"]}`, "unknown field"},
		{"patch", `{} {}`, "one JSON body"},
		{"patch", `{"extra_vars":{"reboot":true}}`, ""},
		{"fixed", `{"limit":"web1"}`, "does not accept a limit"},
	} {
		data = do("POST", "/api/jobs/"+tc.job+"/run", tc.body, 400)
		if !strings.Contains(string(data), tc.message) {
			t.Fatalf("%s: %s", tc.body, data)
		}
	}
	post("fixed", "")
	wait("fixed")
	id = post("wave", `{"limit":"db1"}`)
	data = do("GET", "/api/jobs", "", 200)
	var jobs []jobView
	if err := json.Unmarshal(data, &jobs); err != nil {
		t.Fatal(err)
	}
	if jobs[0].Adhoc == nil || !jobs[0].Adhoc.Limit || !slices.Equal(jobs[0].Adhoc.ExtraVars, []string{"reboot"}) || jobs[1].Adhoc != nil {
		t.Fatalf("adhoc view: %s", data)
	}
	if jobs[2].Pending == nil || jobs[2].Pending.Params == nil || !slices.Equal(jobs[2].Pending.Params.Hosts, []string{"db1"}) {
		t.Fatalf("pending view: %s", data)
	}
	select {
	case j := <-ran:
		t.Fatalf("pending ran: %s", j.Name)
	default:
	}
	do("POST", "/api/runs/"+id+"/approve", "", 200)
	if job := wait("wave"); job.Limit != "db1" {
		t.Fatalf("approved limit: %s", job.Limit)
	}
	// A group resolves to sorted hosts; variables alone keep the job's limit.
	hosts, err := s.resolveLimit(&cfg.Jobs[0], "web")
	if err != nil || !slices.Equal(hosts, []string{"web1", "web2"}) {
		t.Fatalf("group hosts: %v %v", hosts, err)
	}
	post("patch", `{"extra_vars":{"reboot":"true"}}`)
	if job := wait("patch"); job.Limit != "web" || job.ExtraVars["reboot"] != "true" {
		t.Fatalf("variables only: %+v", job)
	}
	post("patch", "{}")
	if job := wait("patch"); job.Limit != "web" || job.ExtraVars["reboot"] != "false" {
		t.Fatalf("plain run: %+v", job)
	}
	for _, event := range []string{eventPending, eventStart} {
		body, err := s.notifyBody(event, &cfg.Jobs[0], &result, "mattermost")
		if err != nil || !strings.Contains(string(body), "hosts: web1") || !strings.Contains(string(body), "vars: reboot=true") {
			t.Fatalf("notification: %s %v", body, err)
		}
	}
	// a group runs as the validated host names, not as the pattern
	post("patch", `{"limit":"web"}`)
	if job := wait("patch"); job.Limit != "web1,web2" {
		t.Fatalf("group limit: %s", job.Limit)
	}
	// a job without an inventory finds the one next to its playbook, as apply does
	post("local", `{"limit":"db1"}`)
	if job := wait("local"); job.Limit != "db1" {
		t.Fatalf("discovered inventory: %s", job.Limit)
	}
	// an inventory that changed while the run waited cannot widen it: the hosts are checked again at start
	for _, tc := range []struct{ limit, inventory string }{
		{"web2", "web:\n  hosts:\n    web1: {}\n"},                                         // the host is gone
		{"web1", "web1:\n  hosts:\n    web1: {}\n    db1: {}\n"},                           // a group now carries the host's name
		{"db1", "db:\n  hosts:\n    db1: {}\n    db2: {}\nweb:\n  hosts:\n    web1: {}\n"}, // unchanged for db1: runs
	} {
		if err := os.WriteFile(filepath.Join(dir, "hosts.yml"), []byte(inventory), 0o600); err != nil {
			t.Fatal(err)
		}
		id = post("wave", `{"limit":"`+tc.limit+`"}`)
		if err := os.WriteFile(filepath.Join(dir, "hosts.yml"), []byte(tc.inventory), 0o600); err != nil {
			t.Fatal(err)
		}
		do("POST", "/api/runs/"+id+"/approve", "", 200)
		if tc.limit == "db1" {
			if job := wait("wave"); job.Limit != "db1" {
				t.Fatalf("unchanged host: %s", job.Limit)
			}
			continue
		}
		for {
			s.mu.Lock()
			r := s.resultByID("wave", id)
			status, summary := r.Status, r.Summary
			s.mu.Unlock()
			if status == "error" {
				if !strings.Contains(summary, "no longer fit") {
					t.Fatalf("changed inventory: %s", summary)
				}
				break
			}
			select {
			case j := <-ran:
				t.Fatalf("ran on a changed inventory (%s): %+v", tc.limit, j)
			case <-ctx.Done():
				t.Fatal("run did not end")
			case <-time.After(time.Millisecond):
			}
		}
	}
}

func TestServeInventorySources(t *testing.T) {
	lg, err := logger.NewEnhancedLogger("error", "text", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if got := inventorySources(lg, "", nil); !slices.Equal(got, []string{"localhost,"}) { // a comply job without inventory
		t.Fatalf("no playbook, no paths: %v", got)
	}
	if got := inventorySources(lg, "/nowhere/x.yml", []string{"a", "b"}); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("paths win: %v", got)
	}
	if got := inventorySources(lg, filepath.Join(t.TempDir(), "x.yml"), nil); !slices.Equal(got, []string{"localhost,"}) {
		t.Fatalf("nothing next to the playbook: %v", got)
	}
}

func TestServeExecuteRestoresGlobals(t *testing.T) {
	// a job's inventory and state file are set for its run only: the next
	// job must not inherit them (nor must a limit be validated against them)
	prevInventory, prevState := inventoryPaths, statePath
	defer func() { inventoryPaths, statePath = prevInventory, prevState }()
	inventoryPaths, statePath = []string{"orig,"}, ""
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 5}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(s.inventory, []string{"orig,"}) {
		t.Fatalf("server inventory: %v", s.inventory)
	}
	j := &ServeJob{Name: "x", Kind: "apply", Playbook: filepath.Join(cfg.DataDir, "missing.yml"), Inventory: filepath.Join(cfg.DataDir, "missing-hosts.yml")}
	if _, err := s.execute(j); err == nil {
		t.Fatal("a missing playbook must fail")
	}
	if !slices.Equal(inventoryPaths, []string{"orig,"}) || statePath != "" {
		t.Fatalf("globals not restored: %v %q", inventoryPaths, statePath)
	}
	if _, err := os.Stat(filepath.Join(cfg.DataDir, "state")); err != nil {
		t.Fatalf("state dir: %v", err)
	}
}

func TestServeExtraVarsArg(t *testing.T) {
	// a caller's value with spaces or = must stay one variable (a k=v list is split on spaces)
	vars, err := parseExtraVars([]string{extraVarsArg(map[string]string{"reboot": "true ansible_host=evil", "serial": "1"})})
	if err != nil || len(vars) != 2 || vars["reboot"] != "true ansible_host=evil" || vars["serial"] != "1" {
		t.Fatalf("extra vars: %v %v", vars, err)
	}
}
