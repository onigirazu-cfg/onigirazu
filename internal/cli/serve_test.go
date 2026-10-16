package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	if _, err := s.request(&cfg.Jobs[1], "schedule"); err != errQueueFull || len(s.results["drift"]) != 0 || s.running["drift"] {
		t.Fatalf("full queue stored a request: %v", err)
	}
	pending, err := s.request(&cfg.Jobs[0], "schedule")
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
	pending, err = s.request(&cfg.Jobs[0], "schedule")
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
		r, err := s.request(&cfg.Jobs[0], "schedule")
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
	requested, err := s.request(&cfg.Jobs[0], "schedule")
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
	pending, err := s.request(s.job("ssh"), "schedule")
	if err != nil || pending.Status != "queued" {
		t.Fatalf("scheduled while paused: %+v %v", pending, err)
	}
	noRun()
	pending, err = s.request(s.job("patch"), "schedule")
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
	contains(do("GET", "/api/pause", "", "users", 200), `"paused":true`)
	do("POST", "/api/jobs/drift-site/run", "", "admins", 202)
	noRun()
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	do("POST", "/api/resume", "", "admins", 200)
	waitRun("drift-site")
}
