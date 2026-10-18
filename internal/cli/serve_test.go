package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/audit"
	"github.com/onigirazu-cfg/onigirazu/internal/comply"
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
	s.runJob = func(_ context.Context, j *ServeJob, _ string) (*JobResult, error) {
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
	s.runJob = func(_ context.Context, j *ServeJob, _ string) (*JobResult, error) {
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
	s.runJob = func(_ context.Context, _ *ServeJob, _ string) (*JobResult, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		r := s.results["patch"][0]
		if r.Status != "running" || r.Started.Before(r.Requested) {
			t.Errorf("running record: %+v", r)
		}
		return nil, errors.New("execution failed")
	}
	done := make(chan struct{})
	go func() { defer close(done); s.worker(ctx) }()
	defer func() { cancel(); <-done }()
	serveWaitActivity(t, s, 0, 0)
	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.runJob = func(_ context.Context, j *ServeJob, _ string) (*JobResult, error) {
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
	if entries := serveAuditEntries(s, 1); len(entries) != 1 || entries[0].OK || entries[0].Action != "pause" || !strings.Contains(entries[0].Reason, "not saved") {
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
	s.runJob = func(_ context.Context, j *ServeJob, _ string) (*JobResult, error) {
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
			s.runJob = func(_ context.Context, _ *ServeJob, _ string) (*JobResult, error) {
				return &JobResult{Status: tc.status, Report: json.RawMessage(`{"secret":"report"}`)}, tc.err
			}
			if _, err := s.request(&cfg.Jobs[0], "schedule", nil); err != nil {
				t.Fatal(err)
			}
			done := make(chan struct{})
			go func() { defer close(done); s.worker(ctx) }()
			defer func() { cancel(); <-done }()
			serveWaitActivity(t, s, 0, 0)
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
		{"missing environment inside the url", `["http://notify.test/hooks/${SERVE_TEST_HOOK_MISSING}"]`, false},
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
	s.runJob = func(_ context.Context, _ *ServeJob, _ string) (*JobResult, error) {
		return &JobResult{Status: "ok"}, nil
	}
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
	if got := serveAuditEntries(s, 100); len(got) != 0 {
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

func TestServeAdhocInventory(t *testing.T) {
	dir := t.TempDir()
	inventory := `all:
  children:
    fleet:
      children:
        web:
          hosts:
            web2: {}
            web1:
              ansible_password: inventory-secret
        db:
          hosts:
            db1: {}
    empty:
      hosts: {}
`
	path := filepath.Join(dir, "hosts.yml")
	if err := os.WriteFile(path, []byte(inventory), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &ServeConfig{DataDir: filepath.Join(dir, "data"), Retain: 10,
		Auth: ServeAuth{UserHeader: "X-User", GroupsHeader: "X-Groups", Operators: []string{"admins"}},
		Jobs: []ServeJob{
			{Name: "patch", Inventory: path, Limit: "web", Adhoc: ServeAdhoc{Limit: true}, Operators: []string{"alice"}},
			{Name: "fixed", Inventory: path},
			{Name: "fleet", Inventory: path, Adhoc: ServeAdhoc{Limit: true}},
			{Name: "broken", Inventory: filepath.Join(dir, "missing.yml"), Adhoc: ServeAdhoc{Limit: true}},
		},
	}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	handler := s.mux()
	for _, tc := range []struct {
		name, user, path string
		code             int
		want             string
	}{
		{"listing", "alice", "patch/inventory", 200, `{"hosts":["web1","web2"],"groups":[{"name":"web","hosts":["web1","web2"]}]}`},
		{"group past the limit", "alice", "patch/inventory?limit=fleet", 400, "outside the job's limit"},
		{"unlimited", "alice", "fleet/inventory", 200, `{"hosts":["db1","web1","web2"],"groups":[{"name":"db","hosts":["db1"]},{"name":"fleet","hosts":["db1","web1","web2"]},{"name":"web","hosts":["web1","web2"]}]}`},
		{"preview", "alice", "patch/inventory?limit=web2,web1", 200, `{"limit":"web2,web1","hosts":["web1","web2"]}`},
		{"group preview", "alice", "patch/inventory?limit=web", 200, `{"limit":"web","hosts":["web1","web2"]}`},
		{"empty preview", "alice", "patch/inventory?limit=", 200, `{"limit":"","hosts":["web1","web2"]}`},
		{"unlimited empty preview", "alice", "fleet/inventory?limit=", 200, `{"limit":"","hosts":["db1","web1","web2"]}`},
		{"outside", "alice", "patch/inventory?limit=db1", 400, "outside the job's limit"},
		{"no match", "alice", "patch/inventory?limit=missing", 400, "matches no host"},
		{"characters", "alice", "patch/inventory?limit=web%0A", 400, "characters a host pattern cannot have"},
		{"fixed", "alice", "fixed/inventory", 400, `{"error":"this job does not accept a limit"}`},
		{"other operator", "bob", "patch/inventory", 403, `{"error":"not an operator of this job"}`},
		{"viewer", "viewer", "patch/inventory?limit=web1", 403, `{"error":"not an operator of this job"}`},
		{"denied before load", "viewer", "broken/inventory", 403, `{"error":"not an operator of this job"}`},
		{"load failure", "alice", "broken/inventory", 500, "failed to load inventory"},
		{"preview load failure", "alice", "broken/inventory?limit=web1", 500, "failed to load inventory"},
		{"validate before load", "alice", "broken/inventory?limit=web%0A", 400, "characters a host pattern cannot have"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/jobs/"+tc.path, nil)
			req.Header.Set("X-User", tc.user)
			if tc.user != "viewer" {
				req.Header.Set("X-Groups", "admins")
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			body := strings.TrimSpace(w.Body.String())
			if w.Code != tc.code || !strings.Contains(body, tc.want) {
				t.Fatalf("got %d %s, want %d %s", w.Code, body, tc.code, tc.want)
			}
			if strings.HasPrefix(tc.want, "{") && body != tc.want {
				t.Fatalf("unexpected JSON fields: %s", body)
			}
			if strings.Contains(body, "ansible_password") || strings.Contains(body, "inventory-secret") {
				t.Fatalf("inventory variables exposed: %s", body)
			}
		})
	}
	if len(serveAuditEntries(s, 100)) != 0 || len(s.results) != 0 || len(s.queue) != 0 {
		t.Fatal("inventory requests created audit actions or runs")
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
	s.runJob = func(_ context.Context, j *ServeJob, _ string) (*JobResult, error) {
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
	actions := serveAuditEntries(s, 100)
	if len(actions) != 1 || actions[0].Run != id || actions[0].Reason != "limit=web1 reboot=true" {
		t.Fatalf("actions: %+v", actions)
	}
	for _, tc := range []struct{ job, body, message string }{
		{"patch", `{"limit":"db1"}`, "outside the job's limit"},
		{"patch", `{"limit":"nope"}`, "matches no host"},
		{"patch", `{"extra_vars":{"serial":"1"}}`, "not allowed"},
		{"patch", `{"limit":5}`, ""},
		{"patch", `{"hosts":["web1"]}`, "unknown field"},
		{"patch", `{"limit":"web1\nforged log line"}`, "characters a host pattern cannot have"},
		{"patch", `{"limit":"web1; rm -rf /"}`, "characters a host pattern cannot have"},
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

func TestServeExecuteDoesNotWriteGlobals(t *testing.T) {
	// a job's inventory and state file are passed as apply arguments: the
	// package globals must never be touched by a job's run
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
	if _, err := s.execute(context.Background(), j, ""); err == nil {
		t.Fatal("a missing playbook must fail")
	}
	if !slices.Equal(inventoryPaths, []string{"orig,"}) || statePath != "" {
		t.Fatalf("globals written by execute: %v %q", inventoryPaths, statePath)
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

func TestServeConcurrencyConfig(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  int
	}{{"", 1}, {"concurrency: 0\n", 1}, {"concurrency: 2\n", 2}, {"concurrency: -1\n", 0}} {
		t.Run(strings.TrimSpace(tc.value), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "serve.yml")
			if err := os.WriteFile(path, []byte(tc.value+testServeYML), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := loadServeConfig(path)
			if tc.want == 0 {
				if err == nil || !strings.Contains(err.Error(), "concurrency") {
					t.Fatalf("want concurrency error, got %v", err)
				}
			} else if err != nil || cfg.Concurrency != tc.want {
				t.Fatalf("want concurrency %d, got %+v, %v", tc.want, cfg, err)
			}
		})
	}
}

// blockedServe reports starts and keeps each job running until released or canceled.
func blockedServe(t *testing.T, concurrency int) (*server, <-chan string, map[string]chan struct{}, context.CancelFunc) {
	t.Helper()
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 5, Concurrency: concurrency,
		Metrics: ServeMetric{Labels: map[string]string{"env": "test"}}}
	release := map[string]chan struct{}{}
	for _, name := range []string{"a", "b", "c", "d"} {
		cfg.Jobs = append(cfg.Jobs, ServeJob{Name: name, Kind: "drift"})
		release[name] = make(chan struct{})
	}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan string, len(cfg.Jobs))
	s.runJob = func(ctx context.Context, j *ServeJob, _ string) (*JobResult, error) {
		started <- j.Name
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release[j.Name]:
			return &JobResult{Started: time.Now().UTC(), Status: "ok"}, nil
		}
	}
	ctx, cancel := contextTimeout(t)
	var workers sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		workers.Add(1)
		go func() { defer workers.Done(); s.worker(ctx) }()
	}
	t.Cleanup(func() { cancel(); workers.Wait() })
	return s, started, release, cancel
}

func serveStart(t *testing.T, started <-chan string) string {
	t.Helper()
	select {
	case name := <-started:
		return name
	case <-time.After(3 * time.Second):
		t.Fatal("job did not start")
		return ""
	}
}

func serveNoStart(t *testing.T, started <-chan string) {
	t.Helper()
	select {
	case name := <-started:
		t.Fatalf("unexpected start: %s", name)
	case <-time.After(50 * time.Millisecond):
	}
}

func serveWaitActivity(t *testing.T, s *server, running, queued int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		s.mu.Lock()
		r, q := s.activity()
		s.mu.Unlock()
		if r == running && q == queued {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("activity = %d running / %d queued, want %d / %d", r, q, running, queued)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestServeConcurrentWorkers(t *testing.T) {
	for _, concurrency := range []int{1, 2} {
		t.Run(fmt.Sprint(concurrency), func(t *testing.T) {
			s, started, release, _ := blockedServe(t, concurrency)
			for i := 0; i < 3; i++ {
				if _, err := s.request(&s.cfg.Jobs[i], "test", nil); err != nil {
					t.Fatal(err)
				}
			}
			active := map[string]bool{}
			for i := 0; i < concurrency; i++ {
				active[serveStart(t, started)] = true
			}
			if !active["a"] || (concurrency == 2 && !active["b"]) {
				t.Fatalf("first jobs = %v", active)
			}
			serveWaitActivity(t, s, concurrency, 3-concurrency)
			if concurrency == 2 {
				s.mu.Lock()
				ordered := !s.results["a"][0].Started.After(s.results["b"][0].Started)
				s.mu.Unlock()
				if !ordered {
					t.Fatal("jobs marked running out of FIFO order")
				}
			}
			serveNoStart(t, started)
			if _, err := s.request(&s.cfg.Jobs[0], "again", nil); !errors.Is(err, errJobBusy) {
				t.Fatalf("duplicate job: %v", err)
			}
			w := httptest.NewRecorder()
			s.mux().ServeHTTP(w, httptest.NewRequest("GET", "/api/me", nil))
			var status struct{ Running, Queued, Concurrency int }
			if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
				t.Fatal(err)
			}
			if status.Running != concurrency || status.Queued != 3-concurrency || status.Concurrency != concurrency {
				t.Fatalf("status: %+v", status)
			}
			w = httptest.NewRecorder()
			s.metricsHandler(w, httptest.NewRequest("GET", "/metrics", nil))
			for _, metric := range []string{"running", "concurrency"} {
				want := fmt.Sprintf("onigirazu_serve_%s{env=\"test\"} %d\n", metric, concurrency)
				if !strings.Contains(w.Body.String(), want) {
					t.Fatalf("missing %s in %s", want, w.Body.String())
				}
			}
			close(release["a"])
			want := "b"
			if concurrency == 2 {
				want = "c"
			}
			if got := serveStart(t, started); got != want {
				t.Fatalf("next job = %s, want %s", got, want)
			}
			close(release["b"])
			if concurrency == 1 {
				if got := serveStart(t, started); got != "c" {
					t.Fatalf("last job = %s", got)
				}
			}
			close(release["c"])
			serveWaitActivity(t, s, 0, 0)
		})
	}
}

func TestServeConcurrentPause(t *testing.T) {
	s, started, release, _ := blockedServe(t, 2)
	for i := range s.cfg.Jobs {
		if _, err := s.request(&s.cfg.Jobs[i], "test", nil); err != nil {
			t.Fatal(err)
		}
	}
	active := []string{serveStart(t, started), serveStart(t, started)}
	if _, err := s.setPause(true, "test", "maintenance"); err != nil {
		t.Fatal(err)
	}
	for _, name := range active {
		close(release[name])
	}
	serveWaitActivity(t, s, 0, 2)
	serveNoStart(t, started)
	if _, err := s.setPause(false, "test", ""); err != nil {
		t.Fatal(err)
	}
	resumed := []string{serveStart(t, started), serveStart(t, started)}
	serveWaitActivity(t, s, 2, 0)
	for _, name := range resumed {
		close(release[name])
	}
	serveWaitActivity(t, s, 0, 0)
}

func TestServeCancelRunning(t *testing.T) {
	s, started, _, cancel := blockedServe(t, 2)
	r, err := s.request(&s.cfg.Jobs[0], "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	serveStart(t, started)
	cancel()
	serveWaitActivity(t, s, 0, 0)
	s.mu.Lock()
	stored := resultCopy(s.resultByID("a", r.ID))
	s.mu.Unlock()
	if stored.Status != "error" || stored.Summary != "interrupted: serve stopping" {
		t.Fatalf("canceled run: %+v", stored)
	}
	data, err := os.ReadFile(filepath.Join(s.cfg.DataDir, "results", r.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var persisted JobResult
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Status != stored.Status || persisted.Summary != stored.Summary {
		t.Fatalf("persisted run: %+v", persisted)
	}
}

func serveCancelRequest(s *server, id, user string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/runs/"+id+"/cancel", strings.NewReader(`{"comment":"maintenance"}`))
	r.Header.Set("X-User", user)
	if user != "viewer" {
		r.Header.Set("X-Groups", "admins")
	}
	s.mux().ServeHTTP(w, r)
	return w
}

func TestServeCancelQueued(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Retain: 10,
		Auth: ServeAuth{UserHeader: "X-User", GroupsHeader: "X-Groups", Operators: []string{"admins"}},
		Jobs: []ServeJob{{Name: "patch", Kind: "apply", Operators: []string{"alice"}}}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	j := &s.cfg.Jobs[0]
	run, err := s.request(j, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []string{"bob", "viewer"} {
		w := serveCancelRequest(s, run.ID, user)
		if w.Code != 403 {
			t.Fatalf("denied: %d %s", w.Code, w.Body.String())
		}
		a := serveAuditEntries(s, 1)[0]
		if a.OK || a.Action != "cancel" || a.Job != j.Name || a.Run != run.ID || a.User != user {
			t.Fatalf("denied audit: %+v", a)
		}
	}
	w := serveCancelRequest(s, run.ID, "alice")
	var got JobResult
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || got.Status != statusCancelled || got.Summary != statusCancelled+" by alice: maintenance" || got.Cancel == nil || got.Cancel.By != "alice" || got.Cancel.At.IsZero() || got.Cancel.Comment != "maintenance" {
		t.Fatalf("queued cancel: %d %s", w.Code, w.Body.String())
	}
	a := serveAuditEntries(s, 1)[0]
	if !a.OK || a.Action != "cancel" || a.Reason != "maintenance" || a.Run != run.ID || a.Job != j.Name {
		t.Fatalf("audit: %+v", a)
	}
	if s.running[j.Name] {
		t.Fatal("canceled job still busy")
	}
	metrics := httptest.NewRecorder()
	s.metricsHandler(metrics, nil)
	if !strings.Contains(metrics.Body.String(), `onigirazu_serve_job_last_status{job="patch",kind="apply"} 4`) {
		t.Fatalf("queued cancel metrics: %s", metrics.Body.String())
	}
	w = serveCancelRequest(s, "missing", "alice")
	if w.Code != 404 {
		t.Fatalf("missing: %d", w.Code)
	}
	w = serveCancelRequest(s, run.ID, "alice")
	if w.Code != 409 {
		t.Fatalf("finished: %d", w.Code)
	}
	// A replacement must stay busy when the old queue entry is skipped.
	next, err := s.request(j, "schedule", nil)
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	s.runJob = func(ctx context.Context, _ *ServeJob, _ string) (*JobResult, error) {
		close(started)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release:
		}
		return &JobResult{Started: time.Now(), Status: "ok"}, nil
	}
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); s.worker(ctx) }()
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		stop()
		<-done
	})
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("replacement not started")
	}
	s.mu.Lock()
	busy := s.running[j.Name]
	old := resultCopy(s.resultByID(j.Name, run.ID))
	s.mu.Unlock()
	if !busy || old.Status != statusCancelled {
		t.Fatalf("old entry changed state: busy=%v old=%+v", busy, old)
	}
	if _, err := s.request(j, "test", nil); !errors.Is(err, errJobBusy) {
		t.Fatalf("replacement overlaps: %v", err)
	}
	close(release)
	serveWaitActivity(t, s, 0, 0)
	s.mu.Lock()
	status := s.resultByID(j.Name, next.ID).Status
	s.mu.Unlock()
	if status != "ok" {
		t.Fatalf("replacement: %s", status)
	}
	j.Approval, j.ApprovalTimeout = "required", time.Hour
	pending, err := s.request(j, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	w = serveCancelRequest(s, pending.ID, "alice")
	if w.Code != 409 || !strings.Contains(w.Body.String(), "reject it") {
		t.Fatalf("pending: %d %s", w.Code, w.Body.String())
	}
}

func TestServeCancelOperatorRunning(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		t.Run(fmt.Sprintf("shutdown=%v", shutdown), func(t *testing.T) {
			var mu sync.Mutex
			var events []string
			client := &http.Client{Transport: serveNotifyTransport(func(r *http.Request) (*http.Response, error) {
				var body struct {
					Event string `json:"event"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					return nil, err
				}
				mu.Lock()
				events = append(events, r.URL.Path+":"+body.Event)
				mu.Unlock()
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
			})}
			s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Retain: 10, Jobs: []ServeJob{{Name: "patch", Kind: "apply"}},
				Notify: []NotifyTarget{{URL: "http://notify.test/default", Format: "json"}, {URL: "http://notify.test/cancel", Format: "json", On: []string{eventCancelled}}}}, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			s.notifyClient = client
			// A canceled partial result must not replace this host verdict.
			s.store(&JobResult{ID: "previous", Job: "patch", Status: "ok", Hosts: map[string]string{"web": "ok"}})
			started, observed, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
			s.runJob = func(ctx context.Context, _ *ServeJob, _ string) (*JobResult, error) {
				close(started)
				<-ctx.Done()
				close(observed)
				<-release
				return &JobResult{Hosts: map[string]string{"web": "failed"}, Report: json.RawMessage(`{"partial":true}`)}, ctx.Err()
			}
			ctx, stop := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() { defer close(done); s.worker(ctx) }()
			t.Cleanup(func() {
				select {
				case <-release:
				default:
					close(release)
				}
				stop()
				<-done
			})
			run, err := s.request(&s.cfg.Jobs[0], "test", nil)
			if err != nil {
				t.Fatal(err)
			}
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("not started")
			}
			if shutdown {
				stop()
			} else {
				w := serveCancelRequest(s, run.ID, "alice")
				if w.Code != 202 || !strings.Contains(w.Body.String(), `"status":"`+statusCancelling+`"`) {
					t.Fatalf("running cancel: %d %s", w.Code, w.Body.String())
				}
				w = httptest.NewRecorder()
				s.mux().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/results/"+run.ID, nil))
				if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":"`+statusCancelling+`"`) {
					t.Fatalf("read active result: %d %s", w.Code, w.Body.String())
				}
				w = httptest.NewRecorder()
				s.metricsHandler(w, nil)
				if !strings.Contains(w.Body.String(), "onigirazu_serve_running 1\n") {
					t.Fatalf("stopping count: %s", w.Body.String())
				}
				if v := s.jobsView(serveIdentity{Operator: true})[0]; v.Active == nil || v.Active.Status != statusCancelling || !v.Running {
					t.Fatalf("active: %+v", v)
				}
			}
			select {
			case <-observed:
			case <-time.After(3 * time.Second):
				t.Fatal("context not canceled")
			}
			close(release)
			serveWaitActivity(t, s, 0, 0)
			stop()
			<-done
			s.notifyWait.Wait()
			s.mu.Lock()
			got := resultCopy(s.resultByID("patch", run.ID))
			busy := s.running["patch"]
			cancels := len(s.cancels)
			s.mu.Unlock()
			if busy || cancels != 0 || got.Duration <= 0 || got.Hosts["web"] != "failed" || string(got.Report) != `{"partial":true}` {
				t.Fatalf("partial result: %+v busy=%v", got, busy)
			}
			if shutdown {
				if got.Status != "error" || got.Summary != "interrupted: serve stopping" || got.Cancel != nil {
					t.Fatalf("shutdown: %+v", got)
				}
				return
			}
			if got.Status != statusCancelled || got.Summary != statusCancelled+" by anonymous: maintenance" || got.Cancel == nil {
				t.Fatalf("canceled: %+v", got)
			}
			stored := httptest.NewRecorder()
			s.mux().ServeHTTP(stored, httptest.NewRequest(http.MethodGet, "/api/results/"+run.ID, nil))
			var record JobResult
			if err := json.Unmarshal(stored.Body.Bytes(), &record); err != nil || stored.Code != 200 || record.Status != statusCancelled || record.Hosts["web"] != "failed" || string(record.Report) != string(got.Report) {
				t.Fatalf("stored API result: %d %s (%v)", stored.Code, stored.Body.String(), err)
			}
			mu.Lock()
			sent := slices.Clone(events)
			mu.Unlock()
			if !slices.Equal(sent, []string{"/cancel:" + statusCancelled}) {
				t.Fatalf("notifications: %v", sent)
			}
			hosts := s.hostsView()
			if len(hosts) != 1 || hosts[0].Status != "ok" {
				t.Fatalf("host verdict: %+v", hosts)
			}
			w := httptest.NewRecorder()
			s.metricsHandler(w, nil)
			if !strings.Contains(w.Body.String(), `onigirazu_serve_job_last_status{job="patch",kind="apply"} 4`) || strings.Contains(w.Body.String(), `onigirazu_serve_host_status{host="web",job="patch"} 2`) {
				t.Fatalf("cancel metrics: %s", w.Body.String())
			}
			s2, err := newServer(s.cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			loaded := s2.resultByID("patch", run.ID)
			if loaded == nil || loaded.Cancel == nil || loaded.Status != statusCancelled || string(loaded.Report) != string(got.Report) {
				t.Fatalf("persisted: %+v", loaded)
			}
			if _, err := s.request(&s.cfg.Jobs[0], "schedule", nil); err != nil {
				t.Fatalf("schedule after cancel: %v", err)
			}
		})
	}
}

func TestServeCancelRestartAndNotify(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Retain: 10, Jobs: []ServeJob{{Name: "patch"}}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	r := &JobResult{ID: "interrupted", Job: "patch", Status: statusCancelling, Cancel: &RunCancel{By: "alice", At: time.Now()}, Hosts: map[string]string{"web": "failed"}}
	s.store(r)
	if len(s.hostsView()) != 0 {
		t.Fatal("stopping run has host verdicts")
	}
	s2, err := newServer(s.cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	got := s2.resultByID("patch", r.ID)
	if got.Status != statusCancelled || got.Summary != "interrupted: serve restarted" || got.Cancel.By != "alice" || s2.running["patch"] {
		t.Fatalf("restart: %+v", got)
	}
	if len(s2.hostsView()) != 0 {
		t.Fatal("canceled run has host verdicts")
	}
	if err := validateNotifyTargets([]NotifyTarget{{URL: "http://notify.test", On: []string{eventCancelled}}}); err != nil {
		t.Fatal(err)
	}
	body, err := s.notifyBody(eventCancelled, &s.cfg.Jobs[0], got, "mattermost")
	if err != nil || !strings.Contains(string(body), ":no_entry: **patch** "+statusCancelled+" by alice — interrupted: serve restarted") {
		t.Fatalf("headline: %s %v", body, err)
	}
	var sent int
	s.notifyClient = &http.Client{Transport: serveNotifyTransport(func(_ *http.Request) (*http.Response, error) {
		sent++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	j := &s.cfg.Jobs[0]
	j.NotifyOK = true
	j.Notify = []NotifyTarget{{URL: "http://notify.test", Format: "json"}}
	s.notify(eventCancelled, j, resultCopy(got))
	s.notifyWait.Wait()
	if sent != 1 {
		t.Fatalf("notify_always: %d", sent)
	}
}

func TestServeCancelFinished(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Retain: 10, Jobs: []ServeJob{{Name: "patch"}}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"ok", "drift", "failed", "error", "rejected", "expired", statusCancelled, statusCancelling, statusTimedOut} {
		t.Run(status, func(t *testing.T) {
			r := &JobResult{ID: status, Job: "patch", Status: status}
			s.store(r)
			w := serveCancelRequest(s, r.ID, "alice")
			if w.Code != http.StatusConflict || r.Status != status || r.Cancel != nil {
				t.Fatalf("finished cancellation: %d %s", w.Code, w.Body.String())
			}
			a := serveAuditEntries(s, 1)[0]
			if a.OK || a.Action != "cancel" || a.Run != r.ID || a.Reason != errRunNotCancellable.Error() {
				t.Fatalf("refused audit: %+v", a)
			}
		})
	}
}

func TestServeAuditExecutionIdentity(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 5, Jobs: []ServeJob{{Name: "identity", Kind: "apply"}}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	s.audit, err = audit.NewStorage(t.TempDir(), logger.NewWithWriter(false, io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	const executionID = "execution-identity"
	const playbookHash = "0123456789abcdef"
	if err := s.audit.SaveRecord(&audit.ExecutionRecord{ID: executionID, PlaybookPath: "site.yml", StartTime: time.Now().UTC(), Status: audit.StatusSuccess}); err != nil {
		t.Fatal(err)
	}
	s.runJob = func(_ context.Context, _ *ServeJob, _ string) (*JobResult, error) {
		return &JobResult{Status: "ok", AuditExecutionID: executionID, PlaybookHash: playbookHash}, nil
	}
	requested, err := s.request(&cfg.Jobs[0], "schedule", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); s.worker(ctx) }()
	defer func() { cancel(); <-done }()
	serveWaitActivity(t, s, 0, 0)
	data, err := os.ReadFile(filepath.Join(cfg.DataDir, "results", requested.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var persisted JobResult
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.AuditExecutionID != executionID {
		t.Fatalf("persisted execution identity: %+v", persisted)
	}
	if persisted.PlaybookHash != playbookHash {
		t.Fatalf("persisted playbook hash: %+v", persisted)
	}
	handler := s.mux()
	for _, path := range []string{"/api/jobs/identity/results", "/api/results/" + requested.ID} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"audit_execution_id":"`+executionID+`"`) {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/api/runs", "/api/runs/" + executionID} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"serve_run":"`+requested.ID+`"`) {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
	}
}

func TestServeResultStorageFailureAndRecovery(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 50, Jobs: []ServeJob{{Name: "storage", Kind: "drift"}}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(cfg.DataDir, "results")
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := &JobResult{ID: "storage-run", Job: "storage", Requested: time.Now(), Status: "ok"}
	if err := s.store(r); err == nil || s.resultByID(r.Job, r.ID) == nil {
		t.Fatalf("failed write lost record or returned no error: %v", err)
	}
	checkStatus := func(wantError bool) {
		t.Helper()
		me := httptest.NewRecorder()
		s.mux().ServeHTTP(me, httptest.NewRequest(http.MethodGet, "/api/me", nil))
		var status struct {
			StorageError string `json:"storage_error"`
		}
		if err := json.Unmarshal(me.Body.Bytes(), &status); err != nil || (status.StorageError != "") != wantError {
			t.Fatalf("storage state: %s, %v", me.Body.String(), err)
		}
		metrics := httptest.NewRecorder()
		s.metricsHandler(metrics, nil)
		value := "0"
		if wantError {
			value = "1"
		}
		if !strings.Contains(metrics.Body.String(), "onigirazu_serve_storage_error "+value+"\n") {
			t.Fatalf("storage metric: %s", metrics.Body.String())
		}
	}
	checkStatus(true)
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := s.store(r); err != nil {
		t.Fatal(err)
	}
	checkStatus(false)
	data, err := os.ReadFile(filepath.Join(dir, r.ID+".json"))
	if err != nil || !json.Valid(data) {
		t.Fatalf("saved record: %s, %v", data, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != r.ID+".json" {
		t.Fatalf("temporary file left behind: %v, %v", entries, err)
	}
	// A rename onto a directory fails after the temporary file was written.
	if err := os.Remove(filepath.Join(dir, r.ID+".json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, r.ID+".json"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := s.store(r); err == nil {
		t.Fatal("rename failure was ignored")
	}
	if _, err := os.Stat(filepath.Join(dir, r.ID+".json.tmp")); !os.IsNotExist(err) {
		t.Fatalf("temporary file not cleaned up: %v", err)
	}
}

func TestServeRetainAge(t *testing.T) {
	for _, startup := range []bool{false, true} {
		t.Run(fmt.Sprintf("startup=%v", startup), func(t *testing.T) {
			cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 50, Jobs: []ServeJob{{Name: "retention", Kind: "drift"}}}
			s, err := newServer(cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			old := time.Now().Add(-48 * time.Hour)
			for _, status := range []string{"ok", "rejected", "expired", "pending", "queued", "running"} {
				r := &JobResult{ID: status, Job: "retention", Requested: old, Started: old, Status: status}
				if status == "pending" {
					r.Approval = &RunApproval{Expires: time.Now().Add(time.Hour)}
				}
				if err := s.store(r); err != nil {
					t.Fatal(err)
				}
			}
			cfg.RetainAge = 24 * time.Hour
			if startup {
				s, err = newServer(cfg, io.Discard)
				if err != nil {
					t.Fatal(err)
				}
			} else if err := s.store(&JobResult{ID: "new", Job: "retention", Requested: time.Now(), Status: "ok"}); err != nil {
				t.Fatal(err)
			}
			for _, status := range []string{"ok", "rejected", "expired"} {
				if s.resultByID("retention", status) != nil {
					t.Errorf("old %s record kept in memory", status)
				}
				if _, err := os.Stat(filepath.Join(cfg.DataDir, "results", status+".json")); !os.IsNotExist(err) {
					t.Errorf("old %s file kept: %v", status, err)
				}
			}
			for _, status := range []string{"pending", "queued", "running"} {
				if s.resultByID("retention", status) == nil {
					t.Errorf("active %s pruned by age", status)
				}
			}
		})
	}
}

func blockServeResultDir(t *testing.T, s *server) {
	t.Helper()
	dir := filepath.Join(s.cfg.DataDir, "results")
	if err := os.Rename(dir, dir+"-saved"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestServeStorageFailureHTTP(t *testing.T) {
	for _, action := range []string{"request", "approve", "reject", "expired"} {
		t.Run(action, func(t *testing.T) {
			cfg := &ServeConfig{DataDir: t.TempDir(), Jobs: []ServeJob{{Name: "guarded", Kind: "apply", Approval: "required", ApprovalTimeout: time.Hour}}}
			s, err := newServer(cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			path := "/api/jobs/guarded/run"
			wantStatus := "pending"
			if action != "request" {
				r, err := s.request(&cfg.Jobs[0], "schedule", nil)
				if err != nil {
					t.Fatal(err)
				}
				path = "/api/runs/" + r.ID + "/" + action
				switch action {
				case "approve":
					wantStatus = "queued"
				case "reject":
					wantStatus = "rejected"
				case "expired":
					path = "/api/runs/" + r.ID + "/approve"
					wantStatus = "expired"
					s.pending["guarded"].Approval.Expires = time.Now().Add(-time.Second)
				}
			}
			blockServeResultDir(t, s)
			res := httptest.NewRecorder()
			s.mux().ServeHTTP(res, httptest.NewRequest(http.MethodPost, path, nil))
			var body struct {
				JobResult
				Error string `json:"error"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if res.Code != http.StatusInternalServerError || body.Error == "" || body.ID == "" || body.Status != wantStatus {
				t.Fatalf("HTTP storage failure: %d %s", res.Code, res.Body.String())
			}
		})
	}
}

func TestServeWorkerKeepsUnsavedResult(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Jobs: []ServeJob{{Name: "worker", Kind: "drift"}}}
	var logs strings.Builder
	s, err := newServer(cfg, &logs)
	if err != nil {
		t.Fatal(err)
	}
	s.runJob = func(context.Context, *ServeJob, string) (*JobResult, error) {
		return &JobResult{Started: time.Now(), Status: "ok", Summary: "completed in memory"}, nil
	}
	r, err := s.request(&cfg.Jobs[0], "schedule", nil)
	if err != nil {
		t.Fatal(err)
	}
	blockServeResultDir(t, s)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.worker(ctx)
	}()
	deadline := time.After(5 * time.Second)
	for {
		s.mu.Lock()
		finished := !s.running["worker"]
		if finished {
			saved := s.resultByID("worker", r.ID)
			if saved == nil || saved.Status != "ok" || saved.Summary != "completed in memory" || s.storeErr == "" {
				t.Errorf("worker lost unsaved result: %+v", saved)
			}
			if !strings.Contains(logs.String(), "serve: worker: result not saved:") {
				t.Errorf("worker did not log storage failure: %s", logs.String())
			}
		}
		s.mu.Unlock()
		if finished {
			break
		}
		select {
		case <-deadline:
			t.Fatal("worker did not finish")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	<-done
}

func TestServeRetentionRemovalFailure(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 1}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	old := &JobResult{ID: "old", Job: "prune", Requested: time.Now(), Status: "rejected"}
	if err := s.store(old); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg.DataDir, "results", "old.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(path, "keep")
	if err := os.WriteFile(child, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	latest := &JobResult{ID: "latest", Job: "prune", Requested: time.Now(), Status: "ok"}
	if err := s.store(latest); err == nil || s.storeErr == "" || s.resultByID("prune", "old") == nil {
		t.Fatalf("prune failure was hidden or retry lost: %v", err)
	}
	if err := os.Remove(child); err != nil {
		t.Fatal(err)
	}
	if err := s.store(latest); err != nil || s.storeErr != "" || s.resultByID("prune", "old") != nil {
		t.Fatalf("prune recovery: %v, %s", err, s.storeErr)
	}
}

func TestServeSecretConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.yml")
	for _, tc := range []struct {
		vars  string
		valid bool
	}{
		{`[reboot, {name: token, secret: true}, {name: label, secret: false}]`, true},
		{`[{secret: true}]`, false},
		{`[token, {name: token, secret: true}]`, false},
		{`[42]`, false},
	} {
		if err := os.WriteFile(path, []byte("retain_age: 168h\njobs:\n  - name: patch\n    playbook: patch.yml\n    adhoc:\n      extra_vars: "+tc.vars+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := loadServeConfig(path)
		if !tc.valid {
			if err == nil {
				t.Fatalf("accepted %s", tc.vars)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		adhoc := cfg.Jobs[0].Adhoc
		if cfg.RetainAge != 168*time.Hour || !slices.Equal(adhoc.ExtraVars, []string{"reboot", "token", "label"}) || !slices.Equal(adhoc.SecretVars, []string{"token"}) {
			t.Fatalf("config: %+v", cfg)
		}
		data, err := json.Marshal(adhoc)
		if err != nil || !strings.Contains(string(data), `"extra_vars":["reboot","token","label"]`) || !strings.Contains(string(data), `"secret_vars":["token"]`) {
			t.Fatalf("compatible adhoc JSON: %s (%v)", data, err)
		}
	}
}

func TestServeSecretParameters(t *testing.T) {
	const secret = "private-token-87194"
	notifications := make(chan string, 8)
	transport := serveNotifyTransport(func(r *http.Request) (*http.Response, error) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		notifications <- string(data)
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 50, Jobs: []ServeJob{{
		Name: "patch", Kind: "apply", Playbook: "patch.yml", Approval: "required", ApprovalTimeout: time.Hour,
		ExtraVars: map[string]string{"keep": "default"},
		Adhoc:     ServeAdhoc{ExtraVars: []string{"token", "reboot"}, SecretVars: []string{"token"}},
	}}}
	for _, format := range []string{"json", "mattermost"} {
		cfg.Notify = append(cfg.Notify, NotifyTarget{URL: "http://notify.test/secrets", Format: format, On: []string{eventPending, eventStart, eventSuccess}})
	}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	s.notifyClient = &http.Client{Transport: transport}
	ran := make(chan map[string]string, 1)
	s.runJob = func(_ context.Context, j *ServeJob, _ string) (*JobResult, error) {
		ran <- j.ExtraVars
		return &JobResult{Started: time.Now().UTC(), Status: "ok"}, nil
	}
	handler := s.mux()
	do := func(method, path, body string, want int) string {
		t.Helper()
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		if w.Code != want {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), secret) {
			t.Fatalf("secret in %s", path)
		}
		return w.Body.String()
	}
	var response map[string]string
	if err := json.Unmarshal([]byte(do("POST", "/api/jobs/patch/run", `{"extra_vars":{"token":"`+secret+`","reboot":"true"}}`, http.StatusAccepted)), &response); err != nil {
		t.Fatal(err)
	}
	id := response["run"]
	s.mu.Lock()
	actual := s.secrets[id]["token"]
	masked := s.resultByID("patch", id).Params.ExtraVars["token"]
	s.mu.Unlock()
	if actual != secret || masked != "***" {
		t.Fatal("secret not separated from stored parameters")
	}
	checkMasked := func() {
		t.Helper()
		for _, path := range []string{"/api/results/" + id, "/api/jobs/patch/results", "/api/jobs/patch/results/" + id, "/api/jobs", "/api/audit"} {
			if body := do("GET", path, "", http.StatusOK); !strings.Contains(body, "***") {
				t.Fatalf("missing mask in %s: %s", path, body)
			}
		}
		for _, path := range []string{filepath.Join(cfg.DataDir, "results", id+".json"), filepath.Join(cfg.DataDir, "audit.log")} {
			data, err := os.ReadFile(path)
			if err != nil || strings.Contains(string(data), secret) || !strings.Contains(string(data), "***") {
				t.Fatalf("unmasked file %s: %v", path, err)
			}
		}
	}
	checkMasked()
	s.notifyWait.Wait()
	do("POST", "/api/runs/"+id+"/approve", "", http.StatusOK)
	ctx, cancel := contextTimeout(t)
	done := make(chan struct{})
	go func() { defer close(done); s.worker(ctx) }()
	defer func() { cancel(); <-done; s.notifyWait.Wait() }()
	select {
	case vars := <-ran:
		if vars["token"] != secret || vars["reboot"] != "true" || vars["keep"] != "default" {
			t.Fatal("effective job lost actual secret or ordinary variables")
		}
	case <-ctx.Done():
		t.Fatal("job did not run")
	}
	serveWaitActivity(t, s, 0, 0)
	s.mu.Lock()
	remaining := len(s.secrets)
	s.mu.Unlock()
	if remaining != 0 {
		t.Fatal("completed secret retained in memory")
	}
	checkMasked()
	s.notifyWait.Wait()
	for i := 0; i < 6; i++ {
		select {
		case payload := <-notifications:
			if strings.Contains(payload, secret) || !strings.Contains(payload, "***") {
				t.Fatalf("unmasked notification: %s", payload)
			}
		case <-ctx.Done():
			t.Fatal("missing notification")
		}
	}
}

func TestServeSecretCleanupAndRestart(t *testing.T) {
	for _, action := range []string{"reject", "expire", "restart-pending", "restart-queued", "cancel-running", "cancel-queued", "queue-full"} {
		t.Run(action, func(t *testing.T) {
			cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 50, Jobs: []ServeJob{{Name: "patch", Kind: "apply", Playbook: "patch.yml", Approval: "required", ApprovalTimeout: time.Hour,
				Adhoc: ServeAdhoc{ExtraVars: []string{"token"}, SecretVars: []string{"token"}}}}}
			s, err := newServer(cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if action == "queue-full" {
				cfg.Jobs[0].Approval = ""
				for i := 0; i < cap(s.queue); i++ {
					s.queue <- queued{}
				}
			}
			r, err := s.request(&cfg.Jobs[0], "test", &RunParams{ExtraVars: map[string]string{"token": "private-value"}})
			if action == "queue-full" {
				if !errors.Is(err, errQueueFull) || len(s.secrets) != 0 {
					t.Fatalf("queue refusal: %v; secrets %d", err, len(s.secrets))
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch action {
			case "reject":
				if _, err := s.reject(r.ID, "test", ""); err != nil {
					t.Fatal(err)
				}
			case "expire":
				s.expirePending(time.Now().Add(2 * time.Hour))
			case "restart-pending", "restart-queued":
				if action == "restart-queued" {
					if _, err := s.approve(r.ID, "test", ""); err != nil {
						t.Fatal(err)
					}
				}
				restarted, err := newServer(cfg, io.Discard)
				if err != nil {
					t.Fatal(err)
				}
				stored := restarted.resultByID("patch", r.ID)
				if stored == nil || stored.Status != "error" || stored.Summary != "interrupted: serve restarted (secret parameters are not kept)" || len(restarted.secrets) != 0 || len(restarted.pending) != 0 || restarted.running["patch"] {
					t.Fatalf("restarted run: %+v", stored)
				}
				data, err := os.ReadFile(filepath.Join(cfg.DataDir, "results", r.ID+".json"))
				if err != nil || !strings.Contains(string(data), stored.Summary) {
					t.Fatalf("restart transition not persisted: %v", err)
				}
				return
			case "cancel-running", "cancel-queued":
				if _, err := s.approve(r.ID, "test", ""); err != nil {
					t.Fatal(err)
				}
				started := make(chan struct{})
				s.runJob = func(ctx context.Context, j *ServeJob, _ string) (*JobResult, error) {
					if j.ExtraVars["token"] != "private-value" {
						t.Error("missing execution secret")
					}
					close(started)
					<-ctx.Done()
					return nil, ctx.Err()
				}
				if action == "cancel-queued" {
					if _, err := s.setPause(true, "test", ""); err != nil {
						t.Fatal(err)
					}
				}
				ctx, cancel := contextTimeout(t)
				done := make(chan struct{})
				go func() { defer close(done); s.worker(ctx) }()
				if action == "cancel-running" {
					select {
					case <-started:
					case <-ctx.Done():
						t.Fatal("job did not start")
					}
				}
				cancel()
				<-done
			}
			if len(s.secrets) != 0 {
				t.Fatal("secret retained after " + action)
			}
		})
	}
}

func TestServeSecretRestartMetadata(t *testing.T) {
	for _, secret := range []bool{false, true} {
		t.Run(fmt.Sprint(secret), func(t *testing.T) {
			cfg := &ServeConfig{DataDir: t.TempDir(), Jobs: []ServeJob{{Name: "job", Approval: "required", ApprovalTimeout: time.Hour, Adhoc: ServeAdhoc{ExtraVars: []string{"label"}}}}}
			if secret {
				cfg.Jobs[0].Adhoc.SecretVars = []string{"label"}
			}
			s, err := newServer(cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			r, err := s.request(&cfg.Jobs[0], "test", &RunParams{ExtraVars: map[string]string{"label": "***"}})
			if err != nil {
				t.Fatal(err)
			}
			cfg.Jobs[0].Adhoc.SecretVars = nil
			restarted, err := newServer(cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			want := "pending"
			if secret {
				want = "error"
			}
			if stored := restarted.resultByID("job", r.ID); stored == nil || stored.Status != want {
				t.Fatalf("restart secret=%v: %+v", secret, stored)
			}
		})
	}
}

func TestServeSecretExecutionEcho(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(fmt.Sprint(failure), func(t *testing.T) {
			cfg := &ServeConfig{DataDir: t.TempDir(), Jobs: []ServeJob{{Name: "echo", Adhoc: ServeAdhoc{ExtraVars: []string{"token", "pin"}, SecretVars: []string{"token", "pin"}}}}}
			var logs strings.Builder
			s, err := newServer(cfg, &logs)
			if err != nil {
				t.Fatal(err)
			}
			const secret = "quoted\"secret\\value"
			s.runJob = func(_ context.Context, j *ServeJob, _ string) (*JobResult, error) {
				value := j.ExtraVars["token"]
				if failure {
					return nil, fmt.Errorf("invalid token %s", value)
				}
				// the value as a string and, digits only, as a JSON number
				report := []byte(`{"output":` + strconv.Quote(value) + `,"pin":4711}`)
				return &JobResult{Status: "ok", Summary: "used " + value, Report: report}, nil
			}
			r, err := s.request(&cfg.Jobs[0], "test", &RunParams{ExtraVars: map[string]string{"token": secret, "pin": "4711"}})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := contextTimeout(t)
			done := make(chan struct{})
			go func() { defer close(done); s.worker(ctx) }()
			defer func() { cancel(); <-done }()
			serveWaitActivity(t, s, 0, 0)
			data, err := os.ReadFile(filepath.Join(cfg.DataDir, "results", r.ID+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var stored JobResult
			if err := json.Unmarshal(data, &stored); err != nil {
				t.Fatal(err)
			}
			s.mu.Lock()
			logText := logs.String()
			s.mu.Unlock()
			if strings.Contains(stored.Summary, secret) || !strings.Contains(stored.Summary, "***") || strings.Contains(logText, secret) {
				t.Fatal("secret echoed in stored summary or worker log")
			}
			if !failure && string(stored.Report) != `{"output":"***","pin":"***"}` {
				t.Fatalf("echoed report not masked: %s", stored.Report)
			}
		})
	}
}

func TestServeCancelStorageFailure(t *testing.T) {
	for _, status := range []string{"queued", "running"} {
		t.Run(status, func(t *testing.T) {
			s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Jobs: []ServeJob{{Name: "patch"}}}, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			r := &JobResult{ID: "cancel-storage", Job: "patch", Status: status}
			if err := s.store(r); err != nil {
				t.Fatal(err)
			}
			s.running[r.Job] = true
			s.secrets[r.ID] = map[string]string{"token": "private-value"}
			stopped := false
			s.cancels[r.ID] = func() { stopped = true }
			// A directory at the result path makes the atomic rename fail.
			path := filepath.Join(s.cfg.DataDir, "results", r.ID+".json")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
			w := serveCancelRequest(s, r.ID, "alice")
			if w.Code != http.StatusInternalServerError || s.storeErr == "" || !strings.Contains(w.Body.String(), `"error"`) {
				t.Fatalf("cancel storage failure: %d %s (%s)", w.Code, w.Body.String(), s.storeErr)
			}
			if status == "queued" {
				if r.Status != statusCancelled || s.running[r.Job] || len(s.secrets) != 0 {
					t.Fatal("queued cancellation did not clear activity and secrets")
				}
			} else if r.Status != statusCancelling || !stopped || len(s.secrets[r.ID]) == 0 {
				t.Fatal("running cancellation did not stop execution and retain masking secrets")
			}
		})
	}
}

func TestServeWorkerExitKeepsCancellingSecrets(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir()}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	r := &JobResult{ID: "still-stopping", Job: "patch", Status: statusCancelling}
	s.results[r.Job] = []*JobResult{r}
	s.secrets[r.ID] = map[string]string{"token": "private-value"}
	ctx, stop := context.WithCancel(context.Background())
	stop()
	s.worker(ctx)
	if len(s.secrets[r.ID]) == 0 {
		t.Fatal("exiting worker removed secrets needed to mask another cancelling run") //nolint:misspell // public API spelling
	}
}

func TestServeHostResults(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Auth: ServeAuth{Token: "tok", UserHeader: "X-User"}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i := range 205 {
		job := "drift"
		if i%2 == 0 {
			job = "verify"
		}
		s.results[job] = append(s.results[job], &JobResult{ID: strconv.Itoa(i), Job: job, Kind: job, Started: now.Add(time.Duration(i) * time.Minute), Status: "failed", Summary: "one failure", Hosts: map[string]string{"web/a": "ok", "other": "failed"}, Report: json.RawMessage(`{"secret":"large report"}`)})
	}
	s.results["irrelevant"] = []*JobResult{{ID: "ignored", Started: now.Add(999 * time.Hour), Hosts: map[string]string{"other": "ok"}}}
	handler := s.mux()
	get := func(path, token string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	if w := get("/api/hosts/web%2Fa/results", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated: %d", w.Code)
	}
	viewerReq := httptest.NewRequest(http.MethodGet, "/api/hosts/web%2Fa/results", nil)
	viewerReq.Header.Set("X-User", "viewer")
	viewerResponse := httptest.NewRecorder()
	handler.ServeHTTP(viewerResponse, viewerReq)
	if viewerResponse.Code != http.StatusOK {
		t.Fatalf("viewer history: %d", viewerResponse.Code)
	}
	w := get("/api/hosts/web%2Fa/results", "tok")
	var rows []hostResultView
	if w.Code != http.StatusOK {
		t.Fatalf("host results: %d %s", w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 200 || rows[0].ID != "204" || rows[199].ID != "5" {
		t.Fatalf("limit/order: %+v", rows)
	}
	for i, row := range rows {
		if row.HostStatus != "ok" || row.Status != "failed" || row.Kind != row.Job || row.Summary != "one failure" || row.Started.IsZero() {
			t.Fatalf("row: %+v", row)
		}
		if i > 0 && !rows[i-1].Started.After(row.Started) {
			t.Fatal("not newest first")
		}
	}
	if strings.Contains(w.Body.String(), "report") || strings.Contains(w.Body.String(), "other") {
		t.Fatalf("unexpected fields: %s", w.Body.String())
	}
	if w := get("/api/hosts/missing/results", "tok"); w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("missing: %d %s", w.Code, w.Body.String())
	}
}

func TestServeJobsActiveTiming(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Jobs: []ServeJob{{Name: "job", Kind: "drift", Every: time.Minute}}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	requested := time.Now().UTC().Add(-time.Minute)
	started := requested.Add(10 * time.Second)
	last := &JobResult{ID: "last", Job: "job", Status: "ok", Started: requested.Add(-time.Hour)}
	active := &JobResult{ID: "active", Job: "job", Status: "queued", Requested: requested, Started: started, Hosts: map[string]string{"web": "ok"}, Report: json.RawMessage(`{"x":1}`)}
	s.results["job"] = []*JobResult{active, last}
	s.running["job"] = true
	for _, status := range []string{"queued", "running"} {
		active.Status = status
		w := httptest.NewRecorder()
		s.mux().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/jobs", nil))
		var jobs []jobView
		if err := json.Unmarshal(w.Body.Bytes(), &jobs); err != nil {
			t.Fatal(err)
		}
		if len(jobs) != 1 {
			t.Fatalf("jobs: %+v", jobs)
		}
		j := jobs[0]
		if j.Active == nil || j.Active.ID != active.ID || !j.Active.Requested.Equal(requested) || !j.Active.Started.Equal(started) || j.Active.Status != status || j.Active.Report != nil || !j.Running || j.Queued != (status == "queued") || j.Every != "1m0s" || j.Last == nil || j.Last.ID != last.ID {
			t.Fatalf("active timing: %+v", j)
		}
	}
	copy := s.jobsView(serveIdentity{})[0].Active
	copy.Hosts["web"] = "failed"
	if active.Hosts["web"] != "ok" {
		t.Fatal("active hosts alias server state")
	}
	s.running["job"] = false
	if s.jobsView(serveIdentity{})[0].Active != nil {
		t.Fatal("inactive job exposes active record")
	}
}

func TestServeCheckRun(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 50, Concurrency: 1, Jobs: []ServeJob{
		{Name: "patch", Kind: "apply", Approval: "required", ApprovalTimeout: time.Hour},
		{Name: "guarded", Kind: "apply", CheckApproval: "required", ApprovalTimeout: time.Hour},
		{Name: "drift", Kind: "drift"}, {Name: "verify", Kind: "verify"}, {Name: "comply", Kind: "comply"},
	}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	handler := s.mux()
	post := func(job, body string, code int) string {
		t.Helper()
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("POST", "/api/jobs/"+job+"/run", strings.NewReader(body)))
		if w.Code != code {
			t.Fatalf("%s: %d %s, want %d", job, w.Code, w.Body.String(), code)
		}
		return w.Body.String()
	}
	for _, kind := range []string{"drift", "verify", "comply"} {
		if body := post(kind, `{"mode":"check"}`, 400); !strings.Contains(body, "only apply jobs have a check mode") {
			t.Fatal(body)
		}
	}
	post("patch", `{"mode":"unknown"}`, 400)
	post("patch", `{"mode":"check"}`, 202)
	post("patch", `{"mode":"check"}`, 409)
	if r := s.results["patch"][0]; r.Status != "queued" || r.Approval != nil || r.Mode != "check" {
		t.Fatalf("check queued: %+v", r)
	}
	post("guarded", `{"mode":"check"}`, 202)
	if r := s.pending["guarded"]; r == nil || r.Mode != "check" {
		t.Fatalf("check pending: %+v", r)
	}
	ran := make(chan string, 1)
	s.runJob = func(_ context.Context, j *ServeJob, mode string) (*JobResult, error) {
		ran <- mode
		return &JobResult{Started: time.Now().UTC(), Status: "drift", Hosts: map[string]string{"web1": "drift"}}, nil
	}
	ctx, cancel := contextTimeout(t)
	done := make(chan struct{})
	go func() { defer close(done); s.worker(ctx) }()
	defer func() { cancel(); <-done }()
	select {
	case mode := <-ran:
		if mode != "check" {
			t.Fatalf("execution mode: %q", mode)
		}
	case <-ctx.Done():
		t.Fatal("check did not execute")
	}
	for {
		s.mu.Lock()
		busy := s.running["patch"]
		s.mu.Unlock()
		if !busy {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("check did not finish")
		case <-time.After(time.Millisecond):
		}
	}
	reloaded, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if r := reloaded.results["patch"][0]; r.Mode != "check" || r.Status != "drift" {
		t.Fatalf("persisted check: %+v", r)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/audit", nil))
	if !strings.Contains(w.Body.String(), `"reason":"mode=check"`) {
		t.Fatalf("check audit: %s", w.Body.String())
	}
	post("patch", `{"mode":"apply"}`, 202)
	if r := s.pending["patch"]; r == nil || r.Mode != "" {
		t.Fatalf("normal run still needs approval: %+v", r)
	}
}

func TestServeCheckMetrics(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Retain: 50}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	normal := &JobResult{Job: "patch", Kind: "apply", Status: "ok", Started: time.Unix(100, 0), Hosts: map[string]string{"web1": "ok"}}
	check := &JobResult{Job: "patch", Kind: "apply", Mode: "check", Status: "drift", Started: time.Unix(200, 0), Hosts: map[string]string{"web1": "drift"}}
	s.results["patch"] = []*JobResult{check, normal}
	if s.lastHostResult("patch") != normal || s.lastExecuted("patch") != check {
		t.Fatal("check must be visible in history but excluded from host verdicts")
	}
	w := httptest.NewRecorder()
	s.metricsHandler(w, httptest.NewRequest("GET", "/metrics", nil))
	for _, want := range []string{
		`onigirazu_serve_job_last_status{job="patch",kind="apply"} 0`,
		`onigirazu_serve_job_last_run_timestamp_seconds{job="patch"} 100`,
		`onigirazu_serve_host_status{host="web1",job="patch"} 0`,
	} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("missing %s in %s", want, w.Body.String())
		}
	}
	s.results["patch"] = []*JobResult{check}
	w = httptest.NewRecorder()
	s.metricsHandler(w, httptest.NewRequest("GET", "/metrics", nil))
	if strings.Contains(w.Body.String(), `job="patch"`) || s.lastHostResult("patch") != nil {
		t.Fatalf("check-only metrics: %s", w.Body.String())
	}
}

func TestServeCheckNotification(t *testing.T) {
	s := &server{cfg: &ServeConfig{}}
	r := &JobResult{Mode: "check", Status: "drift"}
	for _, format := range []string{"json", "mattermost"} {
		body, err := s.notifyBody(eventDrift, &ServeJob{Name: "patch", Kind: "apply"}, r, format)
		if err != nil || !strings.Contains(string(body), "(check)") {
			t.Fatalf("%s: %s %v", format, body, err)
		}
		if format == "json" && !strings.Contains(string(body), `"mode":"check"`) {
			t.Fatalf("missing notification mode: %s", body)
		}
	}
}

func TestServeCheckConfig(t *testing.T) {
	for _, value := range []string{"", "required", "always"} {
		t.Run(value, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "serve.yml")
			data := "jobs:\n  - name: patch\n    kind: apply\n    playbook: patch.yml\n    check_approval: " + value + "\n"
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := loadServeConfig(path)
			if value == "always" {
				if err == nil || !strings.Contains(err.Error(), "check_approval") {
					t.Fatalf("invalid check approval: %v", err)
				}
			} else if err != nil || cfg.Jobs[0].CheckApproval != value {
				t.Fatalf("check approval %q: %+v %v", value, cfg, err)
			}
		})
	}
}

func TestServeCheckParams(t *testing.T) {
	dir := t.TempDir()
	inventory := filepath.Join(dir, "hosts.yml")
	if err := os.WriteFile(inventory, []byte("all:\n  hosts:\n    web1: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &server{}
	job := &ServeJob{Kind: "apply", Inventory: inventory, Adhoc: ServeAdhoc{Limit: true, ExtraVars: []string{"reboot"}}}
	p, err := s.runParams(httptest.NewRequest("POST", "/", strings.NewReader(`{"mode":"check","limit":"web1","extra_vars":{"reboot":"true"}}`)), job)
	if err != nil {
		t.Fatal(err)
	}
	if p.Mode != "check" || p.Limit != "web1" || !slices.Equal(p.Hosts, []string{"web1"}) || p.ExtraVars["reboot"] != "true" {
		t.Fatalf("check parameters: %+v", p)
	}
}

func TestServeRunTimeout(t *testing.T) {
	for _, tc := range []struct {
		name               string
		server, job        time.Duration
		cancel, paused     bool
		shutdown, finished bool
		late               bool
	}{
		{name: "job", job: 50 * time.Millisecond},
		{name: "server default", server: 50 * time.Millisecond},
		{name: "job override", server: time.Hour, job: 50 * time.Millisecond},
		{name: "operator cancel", job: time.Hour, cancel: true},
		{name: "shutdown", job: time.Hour, shutdown: true},
		{name: "late operator cancel", job: 50 * time.Millisecond, cancel: true, late: true},
		{name: "late shutdown", job: 50 * time.Millisecond, shutdown: true, late: true},
		{name: "queued", job: 50 * time.Millisecond, paused: true},
		{name: "finished", job: 50 * time.Millisecond, finished: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Retain: 10, RunTimeout: tc.server,
				Jobs: []ServeJob{{Name: "patch", Kind: "apply", Timeout: tc.job}}}, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if tc.paused {
				if _, err := s.setPause(true, "test", "waiting"); err != nil {
					t.Fatal(err)
				}
			}
			started := make(chan time.Duration, 1)
			observed, release := make(chan struct{}), make(chan struct{}, 1)
			s.runJob = func(ctx context.Context, _ *ServeJob, _ string) (*JobResult, error) {
				deadline, ok := ctx.Deadline()
				if !ok {
					return nil, errors.New("missing deadline")
				}
				started <- time.Until(deadline)
				result := &JobResult{Started: time.Now().UTC(), Status: "ok", AuditExecutionID: "exec-1", Hosts: map[string]string{"web": "failed"}, Report: json.RawMessage(`{"partial":true}`)}
				if tc.paused {
					select {
					case <-time.After(10 * time.Millisecond):
						return result, nil
					case <-ctx.Done():
						return result, ctx.Err()
					}
				}
				<-ctx.Done()
				if tc.late {
					close(observed)
					select {
					case <-release:
					case <-time.After(3 * time.Second):
					}
				}
				if tc.finished {
					return result, nil
				}
				return result, ctx.Err()
			}
			ctx, stop := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() { defer close(done); s.worker(ctx) }()
			t.Cleanup(func() { stop(); <-done })
			run, err := s.request(&s.cfg.Jobs[0], "test", nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.paused {
				time.Sleep(150 * time.Millisecond)
				s.mu.Lock()
				status := s.resultByID("patch", run.ID).Status
				s.mu.Unlock()
				if status != "queued" {
					t.Fatalf("paused status: %s", status)
				}
				if _, err := s.setPause(false, "test", ""); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case remaining := <-started:
				if !tc.cancel && !tc.shutdown && remaining > 50*time.Millisecond {
					t.Fatalf("timeout override ignored: %s", remaining)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("run did not start")
			}
			if tc.late {
				select {
				case <-observed:
				case <-time.After(3 * time.Second):
					t.Fatal("deadline not observed")
				}
			}
			if tc.cancel {
				if w := serveCancelRequest(s, run.ID, "alice"); w.Code != http.StatusAccepted {
					t.Fatalf("cancel: %d %s", w.Code, w.Body.String())
				}
			}
			if tc.shutdown {
				stop()
			}
			release <- struct{}{}
			serveWaitActivity(t, s, 0, 0)
			s.mu.Lock()
			got := resultCopy(s.resultByID("patch", run.ID))
			busy := s.running["patch"]
			lastHost := s.lastHostResult("patch")
			s.mu.Unlock()
			want := statusTimedOut
			switch {
			case tc.cancel && !tc.late:
				want = statusCancelled
			case tc.shutdown && !tc.late:
				want = "error"
			case tc.paused, tc.finished:
				want = "ok"
			}
			if got.Status != want || busy || got.Hosts["web"] != "failed" || string(got.Report) != `{"partial":true}` || got.AuditExecutionID != "exec-1" {
				t.Fatalf("result: %+v, running=%v", got, busy)
			}
			if want == statusTimedOut {
				if got.Summary != "timed out after 50ms" || lastHost != nil { // partial hosts are no verdicts
					t.Fatalf("timeout result: %+v, host verdict: %+v", got, lastHost)
				}
				w := httptest.NewRecorder()
				s.metricsHandler(w, nil)
				if strings.Contains(w.Body.String(), `onigirazu_serve_host_status{host="web"`) {
					t.Fatalf("partial host counted: %s", w.Body.String())
				}
				for _, line := range []string{`onigirazu_serve_job_last_status{job="patch",kind="apply"} 5`} {
					if !strings.Contains(w.Body.String(), line) {
						t.Fatalf("missing %s: %s", line, w.Body.String())
					}
				}
				body, err := s.notifyBody(eventFailure, &s.cfg.Jobs[0], got, "mattermost")
				if err != nil || !strings.Contains(string(body), "timed out") || strings.Contains(string(body), "failed —") {
					t.Fatalf("timeout notification: %s (%v)", body, err)
				}
			}
			w := httptest.NewRecorder()
			s.mux().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/jobs", nil))
			if tc.job > 0 && !strings.Contains(w.Body.String(), `"timeout":"`+tc.job.String()+`"`) {
				t.Fatalf("job timeout missing: %s", w.Body.String())
			}
		})
	}
}

func TestServeRunTimeoutConfig(t *testing.T) {
	for _, tc := range []struct {
		name, config, wantError string
	}{
		{"server negative", "run_timeout: -1s\njobs: [{name: patch, playbook: patch.yml}]", "run_timeout must be >= 0"},
		{"job negative", "jobs: [{name: patch, playbook: patch.yml, timeout: -1s}]", "job patch: timeout must be >= 0"},
		{"valid", "run_timeout: 1m\njobs: [{name: patch, playbook: patch.yml, timeout: 50ms}]", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "serve.yml")
			if err := os.WriteFile(path, []byte(tc.config), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := loadServeConfig(path)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected %q, got %v", tc.wantError, err)
				}
			} else if err != nil || cfg.RunTimeout != time.Minute || cfg.Jobs[0].Timeout != 50*time.Millisecond {
				t.Fatalf("config: %+v (%v)", cfg, err)
			}
		})
	}
}

func TestServeExecuteKeepsPartialResult(t *testing.T) {
	dir := t.TempDir()
	playbook := filepath.Join(dir, "slow.yml")
	inventory := filepath.Join(dir, "hosts.yml")
	if err := os.WriteFile(playbook, []byte("- hosts: localhost\n  gather_facts: false\n  tasks:\n    - name: quick\n      command: \"true\"\n    - name: hang\n      shell: \"sleep 30; true\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inventory, []byte("all:\n  hosts:\n    localhost:\n      ansible_connection: local\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := newServer(&ServeConfig{DataDir: filepath.Join(dir, "data"), Retain: 5}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := time.Now()
	r, err := s.execute(ctx, &ServeJob{Name: "slow", Kind: "apply", Playbook: playbook, Inventory: inventory}, "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want the deadline, got %v", err)
	}
	if time.Since(started) > 20*time.Second {
		t.Fatal("the hung task was not stopped")
	}
	if r == nil || r.Hosts["localhost"] == "" || len(r.Report) == 0 {
		t.Fatalf("partial result lost: %+v", r)
	}
	content, err := os.ReadFile(playbook)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := fmt.Sprintf("%x", sha256.Sum256(content))[:16]
	if r.PlaybookHash != wantHash {
		t.Fatalf("playbook hash: got %q, want %q", r.PlaybookHash, wantHash)
	}
}

func TestServeCheckRunSkipsVerify(t *testing.T) {
	dir := t.TempDir()
	playbook := filepath.Join(dir, "site.yml")
	inventory := filepath.Join(dir, "hosts.yml")
	missing := filepath.Join(dir, "never-created")
	if err := os.WriteFile(playbook, []byte("- hosts: localhost\n  gather_facts: false\n  tasks:\n    - name: quick\n      command: \"true\"\n      changed_when: false\n  verify:\n    - file: {path: "+missing+"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inventory, []byte("all:\n  hosts:\n    localhost:\n      ansible_connection: local\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := newServer(&ServeConfig{DataDir: filepath.Join(dir, "data"), Retain: 5}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	j := &ServeJob{Name: "site", Kind: "apply", Playbook: playbook, Inventory: inventory}
	check, err := s.execute(context.Background(), j, "check")
	if err != nil || check.Status != "ok" || check.Hosts["localhost"] != "ok" {
		t.Fatalf("check run: %+v %v", check, err)
	}
	apply, err := s.execute(context.Background(), j, "")
	if err != nil || apply.Status != "failed" {
		t.Fatalf("a normal run keeps its verify: %+v %v", apply, err)
	}
}

func TestServeJobHistoryAndRecent(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Auth: ServeAuth{UserHeader: "X-User"}, Jobs: []ServeJob{{Name: "site", Kind: "drift"}}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i := range 510 {
		status, mode := "ok", ""
		if i%2 == 0 {
			status, mode = "drift", "check"
		}
		s.results["site"] = append(s.results["site"], &JobResult{
			ID: strconv.Itoa(i), Job: "site", Kind: "drift", Status: status, Mode: mode,
			Requested: now.Add(-time.Duration(i+1) * time.Minute), Started: now.Add(-time.Duration(i) * time.Minute),
			Duration: 2.5, Summary: "three hosts", Trigger: "schedule",
			Hosts:  map[string]string{"web": "ok", "db": "failed", "edge": "unreachable"},
			Params: &RunParams{ExtraVars: map[string]string{"private": "value"}}, Report: json.RawMessage(`{"large":"report"}`),
		})
	}
	s.results["site"] = append([]*JobResult{{ID: "pending", Job: "site", Status: "pending", Requested: now.Add(time.Minute)}}, s.results["site"]...)
	handler := s.mux()
	get := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-User", "viewer")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	for _, tc := range []struct {
		query string
		count int
	}{{"", 50}, {"?limit=2", 2}, {"?limit=999", 500}, {"?limit=500", 500}} {
		t.Run("history"+tc.query, func(t *testing.T) {
			w := get("/api/jobs/site/history" + tc.query)
			var rows []map[string]json.RawMessage
			if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &rows) != nil {
				t.Fatalf("viewer history: %d %s", w.Code, w.Body.String())
			}
			if len(rows) != tc.count || string(rows[0]["id"]) != `"pending"` || string(rows[1]["id"]) != `"0"` {
				t.Fatalf("history limit/order: %s", w.Body.String())
			}
			for _, row := range rows {
				for _, key := range []string{"report", "hosts"} {
					if _, ok := row[key]; ok {
						t.Fatalf("heavy field %s in history: %s", key, w.Body.String())
					}
				}
			}
			for _, key := range []string{"status", "mode", "trigger", "requested", "started", "duration_seconds", "summary", "hosts_count", "failed_hosts"} {
				if _, ok := rows[1][key]; !ok {
					t.Errorf("missing history field %s", key)
				}
			}
			if string(rows[1]["params"]) != `{"extra_vars":{"private":"value"}}` {
				t.Fatalf("history params: %s", rows[1]["params"])
			}
			if string(rows[1]["hosts_count"]) != "3" || string(rows[1]["failed_hosts"]) != "2" || string(rows[1]["duration_seconds"]) != "2.5" {
				t.Fatalf("history counts/duration: %s", w.Body.String())
			}
		})
	}
	w := get("/api/jobs")
	var jobs []struct {
		Recent []struct {
			Status string `json:"status"`
			Mode   string `json:"mode"`
		} `json:"recent"`
	}
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &jobs) != nil || len(jobs) != 1 || len(jobs[0].Recent) != 20 {
		t.Fatalf("recent: %d %s", w.Code, w.Body.String())
	}
	for i, row := range jobs[0].Recent {
		wantStatus, wantMode := "ok", ""
		if i%2 == 0 {
			wantStatus, wantMode = "drift", "check"
		}
		if row.Status != wantStatus || row.Mode != wantMode {
			t.Errorf("recent row %d: %+v", i, row)
		}
	}
	if w := get("/api/jobs/missing/history"); w.Code != http.StatusNotFound {
		t.Errorf("unknown job history: %d", w.Code)
	}
}

func TestServeJobCompare(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Auth: ServeAuth{UserHeader: "X-User"}, Jobs: []ServeJob{{Name: "site", Kind: "drift"}, {Name: "other", Kind: "apply"}}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	report := func(items []DriftItem) json.RawMessage {
		t.Helper()
		data, err := json.Marshal(DriftReport{Drift: map[string][]DriftItem{"web": items}})
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	a := &JobResult{ID: "a", Job: "site", Kind: "drift", Status: "drift", Started: time.Now().UTC().Add(-time.Hour), PlaybookHash: "same", Hosts: map[string]string{"web": "drift", "old": "ok", "unchanged": "ok"}, Report: report([]DriftItem{
		{Key: "stable", Play: "p", Task: "old label", Module: "copy"},
		{Key: "removed", Play: "p", Task: "same label", Module: "copy"},
		{Play: "legacy", Task: "fallback", Module: "copy"},
		{Key: "fallback-key", Play: "legacy", Task: "missing new key", Module: "copy"},
	})}
	b := &JobResult{ID: "b", Job: "site", Kind: "apply", Status: "ok", Started: time.Now().UTC(), PlaybookHash: "same", Hosts: map[string]string{"web": "ok", "new": "drift", "unchanged": "ok"}, Report: report([]DriftItem{
		{Key: "stable", Play: "renamed", Task: "new label", Module: "copy"},
		{Key: "added", Play: "p", Task: "same label", Module: "copy"},
		{Key: "new-key", Play: "legacy", Task: "fallback", Module: "copy"},
		{Play: "legacy", Task: "missing new key", Module: "copy"},
	})}
	s.results["site"] = []*JobResult{b, a,
		{ID: "pending", Job: "site", Kind: "drift", Status: "pending", Report: a.Report},
		{ID: "no-report", Job: "site", Kind: "drift", Status: "ok"},
		{ID: "verify", Job: "site", Kind: "verify", Status: "ok", Report: a.Report},
		{ID: "comply", Job: "site", Kind: "comply", Status: "ok", Report: a.Report},
		{ID: "malformed", Job: "site", Kind: "drift", Status: "ok", Report: json.RawMessage(`{"drift":`)},
		{ID: "null", Job: "site", Kind: "drift", Status: "ok", Report: json.RawMessage(`null`)},
	}
	s.results["other"] = []*JobResult{{ID: "other-run", Job: "other", Kind: "apply", Status: "ok", Report: a.Report}}
	handler := s.mux()
	get := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-User", "viewer")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	type comparedItem struct {
		Host, Key, Play, Task, Module string
	}
	for _, tc := range []struct {
		aHash, hash string
		same        bool
	}{{"same", "same", true}, {"same", "different", false}, {"same", "", false}, {"", "", false}} {
		a.PlaybookHash = tc.aHash
		b.PlaybookHash = tc.hash
		w := get("/api/jobs/site/compare?a=a&b=b")
		var comparison struct {
			A, B struct {
				ID, Status   string
				PlaybookHash string `json:"playbook_hash"`
				Started      time.Time
			}
			SamePlaybook       bool `json:"same_playbook"`
			Hosts              []struct{ Host, A, B string }
			Appeared, Resolved []comparedItem
		}
		if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &comparison) != nil {
			t.Fatalf("viewer compare: %d %s", w.Code, w.Body.String())
		}
		if comparison.A.ID != "a" || comparison.B.ID != "b" || comparison.A.Started.IsZero() || comparison.A.Status != "drift" || comparison.B.Status != "ok" || comparison.A.PlaybookHash != tc.aHash || comparison.B.PlaybookHash != tc.hash || comparison.SamePlaybook != tc.same {
			t.Fatalf("comparison metadata: %s", w.Body.String())
		}
		gotHosts := make(map[string][2]string, len(comparison.Hosts))
		for _, host := range comparison.Hosts {
			gotHosts[host.Host] = [2]string{host.A, host.B}
		}
		if len(gotHosts) != 3 || gotHosts["web"] != [2]string{"drift", "ok"} || gotHosts["old"] != [2]string{"ok", ""} || gotHosts["new"] != [2]string{"", "drift"} {
			t.Fatalf("verdict differences: %s", w.Body.String())
		}
		if len(comparison.Appeared) != 1 || comparison.Appeared[0] != (comparedItem{"web", "added", "p", "same label", "copy"}) || len(comparison.Resolved) != 1 || comparison.Resolved[0].Key != "removed" {
			t.Fatalf("key/fallback matching: %s", w.Body.String())
		}
	}
	for _, tc := range []struct {
		path string
		code int
	}{
		{"/api/jobs/site/compare?a=missing&b=b", 404},
		{"/api/jobs/site/compare?a=a&b=missing", 404},
		{"/api/jobs/missing/compare?a=a&b=b", 404},
		{"/api/jobs/site/compare?a=other-run&b=b", 400},
		{"/api/jobs/site/compare?a=a&b=pending", 400},
		{"/api/jobs/site/compare?a=a&b=no-report", 400},
		{"/api/jobs/site/compare?a=a&b=verify", 400},
		{"/api/jobs/site/compare?a=a&b=comply", 400},
		{"/api/jobs/site/compare?a=a&b=malformed", 400},
		{"/api/jobs/site/compare?a=a&b=null", 400},
		{"/api/jobs/site/compare?a=a", 400},
		{"/api/jobs/site/compare?a=a&b=a", 400},
	} {
		if w := get(tc.path); w.Code != tc.code {
			t.Errorf("%s: got %d, want %d: %s", tc.path, w.Code, tc.code, w.Body.String())
		}
	}
	if w := get("/api/jobs/site/compare?a=b&b=a"); w.Code != http.StatusOK {
		t.Errorf("reverse comparison: %d %s", w.Code, w.Body.String())
	}
	a.Report = json.RawMessage(`{"drift":{"web":[{"key":"shared","play":"p","task":"task"}]}}`)
	b.Report = json.RawMessage(`{"drift":{"db":[{"key":"shared","play":"p","task":"task"}]}}`)
	w := get("/api/jobs/site/compare?a=a&b=b")
	var scoped struct{ Appeared, Resolved []comparedItem }
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &scoped) != nil || len(scoped.Appeared) != 1 || len(scoped.Resolved) != 1 || scoped.Appeared[0].Host != "db" || scoped.Resolved[0].Host != "web" {
		t.Fatalf("keys must be scoped by host: %d %s", w.Code, w.Body.String())
	}
}

func serveAuditEntries(s *server, limit int) []serveAction {
	entries, _ := s.actions(serveActionFilter{Limit: limit})
	return entries
}

func TestServeRunsFiltersAndPaging(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Auth: ServeAuth{UserHeader: "X-User"}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	s.audit, err = audit.NewStorage(t.TempDir(), logger.NewWithWriter(false, io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 6 {
		host, playbook, status := "web1", "site.yml", audit.StatusSuccess
		if i%2 == 1 {
			host, playbook, status = "db1", "db.yml", audit.StatusFailure
		}
		id := strconv.Itoa(i)
		err := s.audit.SaveRecord(&audit.ExecutionRecord{ID: id, PlaybookPath: playbook, StartTime: base.Add(time.Duration(i) * time.Hour), Status: status, Plays: []audit.PlayExecution{{Hosts: []string{host}}}})
		if err != nil {
			t.Fatal(err)
		}
		if i%2 == 0 {
			s.results["deploy"] = append(s.results["deploy"], &JobResult{ID: "serve-" + id, AuditExecutionID: id})
		}
	}
	handler := s.mux()
	for _, tc := range []struct{ query, ids, next string }{
		{"", "5,4,3,2,1,0", ""},
		{"host=web1", "4,2,0", ""}, {"playbook=db", "5,3,1", ""},
		{"status=failure", "5,3,1", ""}, {"job=deploy", "4,2,0", ""},
		{"from=" + url.QueryEscape(base.Add(3*time.Hour).Format(time.RFC3339)), "5,4,3", ""},
		{"to=" + url.QueryEscape(base.Add(2*time.Hour).Format(time.RFC3339)), "2,1,0", ""},
		{"limit=2", "5,4", "2"}, {"limit=2&offset=2", "3,2", "4"}, {"limit=2&offset=4", "1,0", ""},
		{"job=deploy&limit=1&offset=1", "2", "2"}, {"job=deploy&limit=1&offset=2", "0", ""},
		{"job=deploy&status=failure", "", ""}, {"offset=99", "", ""}, {"job=missing", "", ""},
	} {
		t.Run(tc.query, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/runs?"+tc.query, nil)
			r.Header.Set("X-User", "viewer")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			var rows []struct {
				ID       string `json:"id"`
				ServeRun string `json:"serve_run"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil || w.Code != 200 {
				t.Fatalf("%d %s: %v", w.Code, w.Body.String(), err)
			}
			ids := make([]string, 0, len(rows))
			for _, row := range rows {
				ids = append(ids, row.ID)
			}
			if strings.Join(ids, ",") != tc.ids || w.Header().Get("X-Next-Offset") != tc.next {
				t.Fatalf("ids %v next %q, want %s / %s", ids, w.Header().Get("X-Next-Offset"), tc.ids, tc.next)
			}
			if strings.Contains(tc.query, "job=deploy") && len(rows) > 0 && rows[0].ServeRun != "serve-"+rows[0].ID {
				t.Fatalf("missing serve link: %v", rows)
			}
		})
	}
	for _, query := range []string{"from=bad", "to=bad", "offset=abc", "offset=-1", "offset=99999999999999999999999999"} {
		r := httptest.NewRequest("GET", "/api/runs?"+query, nil)
		r.Header.Set("X-User", "viewer")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("%s: %d", query, w.Code)
		}
	}
}

func TestServeAuditFiltersAndPaging(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Auth: ServeAuth{UserHeader: "X-User"}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var entries []serveAction
	for i := range 137 {
		a := serveAction{Time: base.Add(time.Duration(i) * time.Second), User: "alice", Action: "run", Job: "deploy", Run: strconv.Itoa(i), OK: true, Reason: strings.Repeat("x", 2000)}
		if i%2 == 1 {
			a.User, a.Action, a.Job, a.OK = "bob", "cancel", "other", false
		}
		if i == 68 {
			a.Reason = strings.Repeat("x", 70000)
		}
		entries = append(entries, a)
		s.logAction(a)
	}
	handler := s.mux()
	get := func(query string) (*httptest.ResponseRecorder, []serveAction) {
		t.Helper()
		r := httptest.NewRequest("GET", "/api/audit?"+query, nil)
		r.Header.Set("X-User", "viewer")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		var got []serveAction
		if w.Code == 200 {
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
		}
		return w, got
	}
	for _, tc := range []struct {
		query string
		match func(serveAction) bool
	}{
		{"user=alice", func(a serveAction) bool { return a.User == "alice" }},
		{"action=cancel", func(a serveAction) bool { return a.Action == "cancel" }},
		{"job=deploy", func(a serveAction) bool { return a.Job == "deploy" }},
		{"run=68", func(a serveAction) bool { return a.Run == "68" }},
		{"ok=false", func(a serveAction) bool { return !a.OK }},
		{"ok=true", func(a serveAction) bool { return a.OK }},
		{"from=" + url.QueryEscape(base.Add(60*time.Second).Format(time.RFC3339)), func(a serveAction) bool { return !a.Time.Before(base.Add(60 * time.Second)) }},
		{"to=" + url.QueryEscape(base.Add(60*time.Second).Format(time.RFC3339)), func(a serveAction) bool { return !a.Time.After(base.Add(60 * time.Second)) }},
		{"user=alice&ok=false", func(a serveAction) bool { return false }},
	} {
		t.Run(tc.query, func(t *testing.T) {
			w, got := get(tc.query + "&limit=1000")
			var want []serveAction
			for i := len(entries) - 1; i >= 0; i-- {
				if tc.match(entries[i]) {
					want = append(want, entries[i])
				}
			}
			if w.Code != 200 || len(got) != len(want) {
				t.Fatalf("status %d count %d want %d", w.Code, len(got), len(want))
			}
			for i := range got {
				if got[i].Run != want[i].Run {
					t.Fatalf("entry %d: %s want %s", i, got[i].Run, want[i].Run)
				}
			}
			if w.Header().Get("X-Next-Cursor") != "" {
				t.Fatal("unexpected cursor")
			}
		})
	}
	for _, filter := range []string{"", "user=alice&", "from=" + url.QueryEscape(base.Add(30*time.Second).Format(time.RFC3339)) + "&"} {
		var want []string
		for i := len(entries) - 1; i >= 0; i-- {
			if filter == "user=alice&" && entries[i].User != "alice" {
				continue
			}
			if strings.HasPrefix(filter, "from=") && i < 30 {
				continue
			}
			want = append(want, entries[i].Run)
		}
		var got []string
		cursor := ""
		for page := 0; page < 20; page++ {
			query := filter + "limit=17"
			if cursor != "" {
				query += "&before=" + cursor
			}
			w, rows := get(query)
			if w.Code != 200 {
				t.Fatalf("paging: %d %s", w.Code, w.Body.String())
			}
			for _, a := range rows {
				got = append(got, a.Run)
			}
			cursor = w.Header().Get("X-Next-Cursor")
			if cursor == "" {
				break
			}
		}
		if cursor != "" || !slices.Equal(got, want) {
			t.Fatalf("paging %s: got %v, want %v, cursor %s", filter, got, want, cursor)
		}
	}
	for _, query := range []string{"before=bad", "before=", "before=1", "before=-1", "before=99999999999999999999999", "before=999999999", "ok=1", "ok=TRUE", "from=bad", "to=bad"} {
		w, _ := get(query)
		if w.Code != 400 {
			t.Fatalf("%s: %d", query, w.Code)
		}
	}
	w, got := get("before=0")
	if w.Code != 200 || len(got) != 0 || w.Header().Get("X-Next-Cursor") != "" {
		t.Fatal("before zero")
	}
}

func TestServeVarDeclarations(t *testing.T) {
	for _, tc := range []struct{ declaration, rule string }{
		{`{name: v, type: float}`, "type"},
		{`{name: v, type: choice}`, "choices"},
		{`{name: v, choices: [a]}`, "choices"},
		{`{name: v, type: choice, choices: []}`, "choices"},
		{`{name: v, pattern: '['}`, "pattern"},
		{`{name: v, type: int, min: 5, max: 2}`, "min"},
		{`{name: v, min: 1}`, "type int"},
		{`{name: v, type: bool, secret: true}`, "secret"},
		{`{name: v, type: choice, choices: [a], secret: true}`, "secret"},
		{`{name: v, type: int, default: secret-invalid-default, secret: true}`, "invalid default"},
		{`{name: v, type: int, min: 2, default: 1}`, ">= 2"},
		{`{name: v, type: bool, default: yes}`, "true or false"},
		{`{name: v, type: choice, choices: [a], default: b}`, "choices"},
		{`{name: v, pattern: '[a-z]+', default: A}`, "pattern"},
	} {
		t.Run(tc.declaration, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "serve.yml")
			data := "jobs:\n  - name: patch\n    playbook: patch.yml\n    adhoc:\n      extra_vars: [" + tc.declaration + "]\n"
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := loadServeConfig(path)
			if err == nil || !strings.Contains(err.Error(), tc.rule) || !strings.Contains(err.Error(), `"v"`) || strings.Contains(err.Error(), "secret-invalid-default") {
				t.Fatalf("declaration error: %v", err)
			}
		})
	}
}

func TestServeTypedVars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "serve.yml")
	data := `jobs:
  - name: patch
    kind: apply
    playbook: patch.yml
    approval: required
    adhoc:
      extra_vars:
        - legacy
        - {name: old_secret, secret: true}
        - {name: count, type: int, min: 1, max: 3, default: 2, description: Batch size}
        - {name: reboot, type: bool, default: false}
        - {name: wave, type: choice, choices: [blue, green]}
        - {name: tag, pattern: '[a-z]+', required: true}
        - {name: pin, secret: true, pattern: '[0-9]+', default: '1234'}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
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
	handler := s.mux()
	for _, tc := range []struct{ vars, rule string }{
		{`"count":"0","tag":"ok"`, ">= 1"},
		{`"count":"4","tag":"ok"`, "<= 3"},
		{`"count":"1.5","tag":"ok"`, "integer"},
		{`"reboot":"yes please","tag":"ok"`, "true or false"},
		{`"wave":"red","tag":"ok"`, "choices"},
		{`"tag":"ok!"`, "pattern"},
		{`"pin":"secret-invalid-value","tag":"ok"`, "pattern"},
		{`"legacy":"ok"`, "required"},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("POST", "/api/jobs/patch/run", strings.NewReader(`{"extra_vars":{`+tc.vars+`}}`)))
		var response map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if w.Code != 400 || !strings.Contains(response["error"], tc.rule) || strings.Contains(w.Body.String(), "secret-invalid-value") {
			t.Fatalf("validation: %d %s", w.Code, w.Body.String())
		}
	}
	for _, body := range []string{"", "{}"} {
		if _, err := s.runParams(httptest.NewRequest("POST", "/", strings.NewReader(body)), s.job("patch")); err == nil || !strings.Contains(err.Error(), "required") {
			t.Fatalf("empty body: %v", err)
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/jobs", nil))
	var view []jobView
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	a := view[0].Adhoc
	if len(a.ExtraVars) != 7 || !slices.Equal(a.SecretVars, []string{"old_secret", "pin"}) || len(a.Vars) != 7 || a.Vars[6].Default != nil || strings.Contains(w.Body.String(), "1234") {
		t.Fatalf("jobs metadata: %s", w.Body.String())
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("POST", "/api/jobs/patch/run", strings.NewReader(`{"extra_vars":{"tag":"ok","wave":"green","legacy":"unchanged"}}`)))
	if w.Code != 202 {
		t.Fatalf("default launch: %d %s", w.Code, w.Body.String())
	}
	run := s.results["patch"][0]
	if run.Params.ExtraVars["count"] != "2" || run.Params.ExtraVars["reboot"] != "false" || run.Params.ExtraVars["pin"] != "***" || s.secrets[run.ID]["pin"] != "1234" {
		t.Fatalf("defaults: %+v", run.Params)
	}
	// Defaults also apply when the entire request body is absent.
	j := *s.job("patch")
	j.Adhoc.Vars = slices.Clone(j.Adhoc.Vars)
	j.Adhoc.Vars[5].Required = false
	p, err := s.runParams(httptest.NewRequest("POST", "/", nil), &j)
	if err != nil || p.ExtraVars["count"] != "2" {
		t.Fatalf("empty body defaults: %+v %v", p, err)
	}
}

func TestServeRerunFailed(t *testing.T) {
	for _, scenario := range []string{"success", "secret", "forbidden", "no failed", "not executed", "busy", "approval", "window", "outside limit", "stale var"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			inv := filepath.Join(dir, "hosts.yml")
			if err := os.WriteFile(inv, []byte("web:\n  hosts:\n    web1: {}\n    web2: {}\n    web3: {}\ndb:\n  hosts:\n    db1: {}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			j := ServeJob{Name: "patch", Kind: "apply", Inventory: inv, Playbook: "patch.yml", Limit: "web", ApprovalTimeout: time.Hour,
				Adhoc: ServeAdhoc{ExtraVars: []string{"reboot"}, Vars: []adhocVar{{Name: "reboot", Type: "bool"}}}}
			cfg := &ServeConfig{DataDir: dir, Retain: 50, Jobs: []ServeJob{j}}
			cfg.Auth = ServeAuth{UserHeader: "X-User", GroupsHeader: "X-Groups", Operators: []string{"ops"}}
			switch scenario {
			case "forbidden":
				cfg.Jobs[0].Operators = []string{"bob"}
			case "approval":
				cfg.Jobs[0].CheckApproval = "required"
			case "window":
				cfg.Jobs[0].Windows = []string{strings.ToLower(time.Now().UTC().Add(24*time.Hour).Format("Mon")) + " 00:00-00:01"}
				var err error
				cfg.Jobs[0].windows, cfg.Jobs[0].windowsLoc, err = parseServeWindows(cfg.Jobs[0].Windows, "UTC")
				if err != nil {
					t.Fatal(err)
				}
			}
			s, err := newServer(cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			parent := &JobResult{ID: "parent", Job: "patch", Kind: "apply", Mode: "check", Status: "failed", Hosts: map[string]string{"web1": "failed", "web2": "unreachable", "web3": "ok"}, Params: &RunParams{ExtraVars: map[string]string{"reboot": "false"}}}
			code := 202
			if scenario == "window" {
				parent.Mode = ""
			}
			switch scenario {
			case "secret":
				parent.Params.SecretVars = []string{"token"}
				code = 400
			case "forbidden":
				code = 403
			case "no failed":
				parent.Hosts = map[string]string{"web1": "ok"}
				code = 400
			case "not executed":
				parent.Status = "pending"
				code = 400
			case "busy":
				s.running["patch"] = true
				code = 409
			case "outside limit":
				parent.Hosts["db1"] = "failed"
				code = 400
			case "stale var": // valid when the parent ran, not under today's declaration
				parent.Params.ExtraVars["reboot"] = "maybe"
				code = 400
			}
			s.results["patch"] = []*JobResult{parent}
			req := httptest.NewRequest("POST", "/api/runs/parent/rerun-failed", nil)
			req.Header.Set("X-User", "alice")
			req.Header.Set("X-Groups", "ops")
			w := httptest.NewRecorder()
			s.mux().ServeHTTP(w, req)
			if w.Code != code {
				t.Fatalf("rerun: %d %s, want %d", w.Code, w.Body.String(), code)
			}
			if scenario == "secret" && w.Body.String() != "{\"error\":\"the run had secret parameters; start it again from the form\"}\n" {
				t.Fatalf("secret error: %s", w.Body.String())
			}
			if code == 202 {
				child := s.results["patch"][0]
				if child.Parent != "parent" || child.Trigger != "rerun:alice" || child.Mode != parent.Mode || !slices.Equal(child.Params.Hosts, []string{"web1", "web2"}) || child.Params.ExtraVars["reboot"] != "false" {
					t.Fatalf("child: %+v params: %+v", child, child.Params)
				}
				saved, err := os.ReadFile(filepath.Join(dir, "results", child.ID+".json"))
				if err != nil || !strings.Contains(string(saved), `"parent": "parent"`) && !strings.Contains(string(saved), `"parent":"parent"`) {
					t.Fatalf("stored parent: %s %v", saved, err)
				}
				if scenario == "window" && child.Status != "scheduled" {
					t.Fatalf("window: %s", child.Status)
				}
				if scenario == "approval" && child.Status != "pending" {
					t.Fatalf("approval: %s", child.Status)
				}
				effective, err := s.effectiveJob(s.job("patch"), child.Params, child.ID)
				if err != nil || effective.Limit != "web1,web2" {
					t.Fatalf("effective hosts: %+v %v", effective, err)
				}
			}
			log, err := os.ReadFile(filepath.Join(dir, "audit.log"))
			if err != nil || !strings.Contains(string(log), `"action":"rerun_failed"`) || !strings.Contains(string(log), "parent=parent") {
				t.Fatalf("audit: %s %v", log, err)
			}
		})
	}
}

func TestServeComplyKeepsPartialResult(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	profile, inventory := writeComplyRunFiles(t, dir, true)
	s, err := newServer(&ServeConfig{DataDir: filepath.Join(dir, "data"), Retain: 5}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	pushes := 0
	oldTransport := http.DefaultTransport
	http.DefaultTransport = serveNotifyTransport(func(r *http.Request) (*http.Response, error) {
		pushes++
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	s.cfg.Metrics.Push = "http://metrics.test/push"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	r, err := s.execute(ctx, &ServeJob{Name: "partial", Kind: "comply", Profile: profile, Inventory: inventory}, "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline, got %v", err)
	}
	if r == nil || r.AuditExecutionID == "" || r.Hosts["localhost"] != "ok" || len(r.Report) == 0 {
		t.Fatalf("partial result lost: %+v", r)
	}
	var report comply.Report
	if err := json.Unmarshal(r.Report, &report); err != nil {
		t.Fatal(err)
	}
	if report.Passed != 1 || report.Failed != 0 {
		t.Fatalf("partial checks: %+v", report)
	}
	if pushes != 0 {
		t.Fatalf("stopped run pushed metrics %d times", pushes)
	}
}

func TestServeApplyIncludesDiff(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	playbook := filepath.Join(dir, "site.yml")
	inventory := filepath.Join(dir, "hosts.yml")
	target := filepath.Join(dir, "managed.txt")
	if err := os.WriteFile(target, []byte("before\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(playbook, []byte("- hosts: localhost\n  gather_facts: false\n  tasks:\n    - name: config\n      copy:\n        dest: "+target+"\n        content: 'after'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inventory, []byte("all:\n  hosts:\n    localhost:\n      ansible_connection: local\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := newServer(&ServeConfig{DataDir: filepath.Join(dir, "data"), Retain: 5}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	j := &ServeJob{Name: "diff", Kind: "apply", Playbook: playbook, Inventory: inventory}
	for _, mode := range []string{"check", "apply"} {
		r, err := s.execute(context.Background(), j, mode)
		if err != nil {
			t.Fatal(err)
		}
		var report DriftReport
		if err := json.Unmarshal(r.Report, &report); err != nil {
			t.Fatal(err)
		}
		items := report.Drift["localhost"]
		if len(items) != 1 || !strings.Contains(items[0].Diff, "-before") || !strings.Contains(items[0].Diff, "+after") {
			t.Fatalf("%s diff: %+v", mode, items)
		}
		data, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if mode == "check" && string(data) != "before\n" {
			t.Fatalf("check modified file: %s", data)
		}
		if mode == "apply" && string(data) != "after" {
			t.Fatalf("apply did not modify file: %s", data)
		}
	}
}

func TestServeComplyWithoutCompletedChecks(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	profile, inventory := writeComplyRunFiles(t, dir, false)
	if err := os.WriteFile(profile, []byte("name: pending\nbecome: false\ncontrols:\n  - id: slow\n    title: slow command\n    check: {command: {cmd: 'sleep 30; true'}}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := newServer(&ServeConfig{DataDir: filepath.Join(dir, "data"), Retain: 5}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := s.execute(ctx, &ServeJob{Name: "pending", Kind: "comply", Profile: profile, Inventory: inventory}, "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline, got %v", err)
	}
	if r == nil || r.AuditExecutionID == "" || len(r.Hosts) != 0 || len(r.Report) == 0 {
		t.Fatalf("stopped run without checks: %+v", r)
	}
	var report comply.Report
	if err := json.Unmarshal(r.Report, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Hosts) != 0 || report.Passed != 0 || report.Failed != 0 {
		t.Fatalf("unexpected verdict: %+v", report)
	}
}

func TestServeProxyAuth(t *testing.T) {
	if got := splitGroups("ops|Fleet Operators| web "); !slices.Equal(got, []string{"ops", "Fleet Operators", "web"}) {
		t.Fatalf("pipe list: %q", got)
	}
	if got := splitGroups("ops, Domain Admins"); !slices.Equal(got, []string{"ops", "Domain Admins"}) {
		t.Fatalf("comma list: %q", got)
	}
	if got := splitGroups(""); len(got) != 0 {
		t.Fatalf("empty: %q", got)
	}
	if _, err := parseTrustedProxies([]string{"10.0.0.0/8", "192.0.2.10", "::1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := parseTrustedProxies([]string{"proxy.example"}); err == nil {
		t.Fatal("a host name is not an address")
	}
	for _, tc := range []struct {
		name    string
		trusted []string
		remote  string
		code    int
	}{
		{"no list trusts every peer", nil, "192.0.2.1:4000", 200},
		{"peer in the list", []string{"10.0.0.0/8"}, "10.1.2.3:4000", 200},
		{"single address", []string{"192.0.2.10"}, "192.0.2.10:4000", 200},
		{"peer outside the list", []string{"10.0.0.0/8"}, "192.0.2.1:4000", 401},
		{"IPv4 over IPv6", []string{"10.0.0.0/8"}, "[::ffff:10.0.0.7]:4000", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &ServeConfig{DataDir: t.TempDir(), Auth: ServeAuth{UserHeader: "X-Authentik-Username", GroupsHeader: "X-Authentik-Groups",
				Operators: []string{"Fleet Operators"}, TrustedProxies: tc.trusted, SignOutURL: "/outpost.goauthentik.io/sign_out"}}
			s, err := newServer(cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("GET", "/api/me", nil)
			req.RemoteAddr = tc.remote
			req.Header.Set("X-Authentik-Username", "alice")
			req.Header.Set("X-Authentik-Groups", "staff|Fleet Operators")
			w := httptest.NewRecorder()
			s.mux().ServeHTTP(w, req)
			if w.Code != tc.code {
				t.Fatalf("%d %s, want %d", w.Code, w.Body.String(), tc.code)
			}
			if tc.code == 200 && (!strings.Contains(w.Body.String(), `"operator":true`) || !strings.Contains(w.Body.String(), `"sign_out_url":"/outpost.goauthentik.io/sign_out"`)) {
				t.Fatalf("identity: %s", w.Body.String())
			}
		})
	}
}
