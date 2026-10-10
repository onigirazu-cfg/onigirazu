package cli

import (
	"context"
	"encoding/json"
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
	if cfg.Listen != ":8086" || cfg.Jobs[0].Kind != "drift" || !filepath.IsAbs(cfg.Jobs[0].Playbook) || cfg.Jobs[1].Profile != "ssh" {
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
	if code, body := do("GET", "/metrics", nil); code != 200 || !strings.Contains(string(body), `onigirazu_serve_host_status{host="web1",job="drift-site"} 1`) {
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
