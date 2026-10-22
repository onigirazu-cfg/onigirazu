package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/engine"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestRunObserverBufferAndMask(t *testing.T) {
	p := newRunProgress(strings.NewReplacer("private-value", "***"))
	p.OnPlayStart("play private-value", 1, 1)
	statuses := []types.TaskResult{
		{TaskName: "task private-value", Host: "local", Error: "secret-error", Output: map[string]interface{}{"stdout": "secret-out", "diff": "secret-diff"}, Duration: 3 * time.Millisecond},
		{Host: "local", Changed: true}, {Host: "local", Failed: true}, {Host: "local", Skipped: true}, {Host: "local", Failed: true, Ignored: true},
	}
	for i := range statuses {
		p.OnTaskStart(statuses[i].TaskName, "local")
		p.OnTaskEnd(&statuses[i])
	}
	snapshot, events, gap, ch := p.subscribe(0)
	p.unsubscribe(ch)
	if gap || snapshot.Seq != 11 || snapshot.TasksFinished != 5 {
		t.Fatalf("snapshot: %+v gap=%v", snapshot, gap)
	}
	h := snapshot.Hosts["local"]
	if h.OK != 2 || h.Changed != 1 || h.Failed != 1 || h.Skipped != 1 || !h.Done || h.TasksFinished != 5 {
		t.Fatalf("host: %+v", h)
	}
	for i, e := range events {
		if e.Seq != uint64(i+1) {
			t.Fatalf("sequence: %+v", events)
		}
	}
	if events[2].DurationMS != 3 {
		t.Fatalf("duration: %+v", events[2])
	}
	data, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"secret-error", "secret-out", "secret-diff", "private-value"} {
		if bytes.Contains(data, []byte(secret)) {
			t.Fatalf("leaked %s: %s", secret, data)
		}
	}
	if !bytes.Contains(data, []byte("task ***")) {
		t.Fatalf("task not masked: %s", data)
	}
	_, _, _, slow := p.subscribe(0)
	started := time.Now()
	for i := 0; i < 10000; i++ {
		p.OnTaskStart("bounded", "local")
	}
	if time.Since(started) > 2*time.Second {
		t.Fatal("callbacks blocked on a slow subscriber")
	}
	p.mu.Lock()
	_, subscribed := p.subscribers[slow]
	p.mu.Unlock()
	if subscribed {
		t.Fatal("full subscriber retained")
	}
	for range slow {
	}
	snapshot, events, gap, ch = p.subscribe(0)
	p.unsubscribe(ch)
	if !gap || len(events) != progressLimit || events[0].Seq != snapshot.Seq-progressLimit+1 {
		t.Fatalf("ring: %d %+v gap=%v", len(events), snapshot, gap)
	}
	p.finish("ok")
	p.OnTaskStart("too late", "local")
	snapshot, events, _, ch = p.subscribe(snapshot.Seq)
	if snapshot.Status != "ok" || len(events) != 1 || events[0].Type != "end" {
		t.Fatalf("finish: %+v %+v", snapshot, events)
	}
	if _, open := <-ch; open {
		t.Fatal("finished subscription open")
	}
}

func TestServeProgressPlan(t *testing.T) {
	p := newRunProgress(strings.NewReplacer("private-value", "***"))
	p.OnPlayStart("play private-value", 1, 1)
	_, _, _, live := p.subscribe(0)
	defer p.unsubscribe(live)
	hosts := []string{"host private-value", "other"}
	p.OnPlayPlan("play private-value", hosts, 4)
	event := <-live
	if event.Type != "plan" || event.Play != "play ***" || event.Host != "" || event.Planned != 4 || len(event.Hosts) != 2 || event.Hosts[0] != "host ***" {
		t.Fatalf("plan event: %+v", event)
	}
	if hosts[0] != "host private-value" {
		t.Fatal("plan mutated caller's hosts")
	}
	// Later plays and serial batches add to the existing totals.
	p.OnPlayPlan("next play", []string{"host private-value"}, 3)
	p.OnPlayPlan("verify", []string{"verify host"}, 0)
	p.OnTaskEnd(&types.TaskResult{Host: "host private-value"})
	snapshot, events, _, ch := p.subscribe(0)
	p.unsubscribe(ch)
	if snapshot.TasksPlanned != 11 || snapshot.TasksFinished != 1 || snapshot.Hosts["host ***"].Planned != 7 || snapshot.Hosts["other"].Planned != 4 || snapshot.Hosts["verify host"].Planned != 0 {
		t.Fatalf("planned snapshot: %+v", snapshot)
	}
	data, err := json.Marshal(struct {
		Snapshot progressSnapshot
		Events   []progressEvent
	}{snapshot, events})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("private-value")) || !bytes.Contains(data, []byte(`"hosts":["host ***","other"]`)) {
		t.Fatalf("plan masking: %s", data)
	}
	p.finish("ok")
	p.OnPlayPlan("late", []string{"other"}, 100)
	snapshot, _, _, ch = p.subscribe(0)
	p.unsubscribe(ch)
	if snapshot.TasksPlanned != 11 {
		t.Fatalf("plan after finish: %+v", snapshot)
	}
}

func progressTestServer(t *testing.T) *server {
	t.Helper()
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Retain: 5, Auth: ServeAuth{UserHeader: "X-User"}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestServeProgressReplayAndErrors(t *testing.T) {
	s := progressTestServer(t)
	p := newRunProgress(strings.NewReplacer())
	for i := 0; i < 1005; i++ {
		p.OnTaskStart("task", "local")
	}
	p.finish("ok")
	s.progress["kept"] = p
	s.results["job"] = []*JobResult{{ID: "kept", Status: "ok"}, {ID: "old", Status: "ok"}}
	for _, tc := range []struct {
		path  string
		code  int
		gap   bool
		count int
	}{
		{"kept/events?after=1003", 200, false, 4}, {"kept/events?after=1", 200, true, 1002}, {"old/events", 404, false, 0}, {"unknown/events", 404, false, 0}, {"kept/events?after=bad", 400, false, 0},
	} {
		t.Run(tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/results/"+tc.path, nil)
			req.Header.Set("X-User", "viewer")
			s.mux().ServeHTTP(w, req)
			if w.Code != tc.code {
				t.Fatalf("%d: %s", w.Code, w.Body.String())
			}
			if tc.code != 200 {
				if strings.HasPrefix(tc.path, "old/") && !strings.Contains(w.Body.String(), "no progress for this run") {
					t.Fatal(w.Body.String())
				}
				return
			}
			if w.Header().Get("Content-Type") != "application/x-ndjson" || !w.Flushed {
				t.Fatalf("not flushed NDJSON: %+v", w)
			}
			lines := strings.Split(strings.TrimSpace(w.Body.String()), "\n")
			if len(lines) != tc.count || !strings.Contains(lines[0], `"type":"snapshot"`) || strings.Contains(w.Body.String(), `"type":"gap"`) != tc.gap {
				t.Fatalf("replay: %d lines, gap=%v", len(lines), tc.gap)
			}
			if !strings.Contains(lines[len(lines)-1], `"type":"end"`) {
				t.Fatal("missing end")
			}
			if !tc.gap && !strings.Contains(lines[1], `"seq":1004`) {
				t.Fatal("wrong replay cursor")
			}
		})
	}
}

// Each Flush delivers complete lines without needing a TCP listener.
type progressWriter struct {
	header http.Header
	buffer bytes.Buffer
	lines  chan []byte
	ctx    context.Context
}

func (w *progressWriter) Header() http.Header         { return w.header }
func (w *progressWriter) WriteHeader(int)             {}
func (w *progressWriter) Write(b []byte) (int, error) { return w.buffer.Write(b) }
func (w *progressWriter) Flush() {
	b := bytes.Clone(w.buffer.Bytes())
	w.buffer.Reset()
	select {
	case w.lines <- b:
	case <-w.ctx.Done():
	}
}
func progressLine(t *testing.T, lines <-chan []byte) map[string]interface{} {
	t.Helper()
	select {
	case line := <-lines:
		var event map[string]interface{}
		if err := json.Unmarshal(line, &event); err != nil {
			t.Fatal(err)
		}
		return event
	case <-time.After(5 * time.Second):
		t.Fatal("no stream line")
		return nil
	}
}

func TestServeProgressLiveAndDisconnect(t *testing.T) {
	for _, queued := range []bool{false, true} {
		t.Run(fmt.Sprint(queued), func(t *testing.T) {
			s := progressTestServer(t)
			p := newRunProgress(strings.NewReplacer())
			s.results["job"] = []*JobResult{{ID: "live", Status: "queued"}}
			if !queued {
				s.progress["live"] = p
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			w := &progressWriter{header: make(http.Header), lines: make(chan []byte, 10), ctx: ctx}
			req := httptest.NewRequest("GET", "/api/results/live/events", nil).WithContext(ctx)
			req.Header.Set("X-User", "viewer")
			done := make(chan struct{})
			go func() { defer close(done); s.mux().ServeHTTP(w, req) }()
			if progressLine(t, w.lines)["type"] != "snapshot" {
				t.Fatal("missing snapshot")
			}
			if queued {
				s.mu.Lock()
				s.progress["live"] = p
				s.results["job"][0].Status = "running"
				s.mu.Unlock()
				if progressLine(t, w.lines)["type"] != "snapshot" {
					t.Fatal("missing running snapshot")
				}
			}
			p.OnTaskStart("live task", "host")
			if event := progressLine(t, w.lines); event["type"] != "task_start" || event["task"] != "live task" {
				t.Fatalf("event: %+v", event)
			}
			if queued {
				p.finish("ok")
				if progressLine(t, w.lines)["type"] != "end" {
					t.Fatal("missing end")
				}
			} else {
				cancel()
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("stream did not close")
			}
			p.mu.Lock()
			defer p.mu.Unlock()
			if len(p.subscribers) != 0 {
				t.Fatal("subscriber leaked")
			}
		})
	}
}

func TestServeProgressRetention(t *testing.T) {
	s := progressTestServer(t)
	for i := 0; i < 22; i++ {
		id := fmt.Sprint(i)
		s.progress[id] = newRunProgress(strings.NewReplacer())
		s.retainProgress(id, "ok")
	}
	if len(s.progress) != 20 || s.progress["0"] != nil || s.progress["1"] != nil || s.progress["21"] == nil {
		t.Fatal("wrong retained runs")
	}
}

func TestRunObserverLocalhostIsolation(t *testing.T) {
	dir := t.TempDir()
	inventory := filepath.Join(dir, "hosts.yml")
	if err := os.WriteFile(inventory, []byte("all:\n  hosts:\n    localhost:\n      ansible_connection: local\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for _, name := range []string{"first", "second"} {
		playbook := filepath.Join(dir, name+".yml")
		if err := os.WriteFile(playbook, []byte("- name: "+name+"\n  hosts: localhost\n  gather_facts: false\n  tasks:\n    - name: "+name+" task\n      command: \"true\"\n      changed_when: false\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := newRunProgress(strings.NewReplacer())
			ctx, cancel := context.WithTimeout(withRunObserver(context.Background(), p), 20*time.Second)
			defer cancel()
			obs, ok := ctx.Value(runObserverKey{}).(engine.ExecutionObserverI)
			if !ok || obs != p {
				t.Error("observer lost in context")
				return
			}
			result, err := runPlaybookContext(ctx, []string{playbook, "-i", inventory, "--state", filepath.Join(dir, name+"-state.json")})
			if err != nil || result == nil {
				t.Errorf("execute %s: %v", name, err)
				return
			}
			snapshot, events, _, ch := p.subscribe(0)
			if snapshot.TasksPlanned != 1 || snapshot.Hosts["localhost"].Planned != 1 {
				t.Errorf("missing localhost plan for %s: %+v", name, snapshot)
			}
			defer p.unsubscribe(ch)
			starts, ends := 0, 0
			for _, e := range events {
				if e.Type == "task_start" || e.Type == "task_end" {
					if e.Task != name+" task" {
						t.Errorf("crossed events in %s: %+v", name, e)
					}
					if e.Type == "task_start" {
						starts++
					} else {
						ends++
					}
				}
			}
			if starts != 1 || ends != 1 {
				t.Errorf("missing task events for %s: %+v", name, events)
			}
		}()
	}
	wg.Wait()
}

func TestServeProgressWorkerContext(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 5, Jobs: []ServeJob{{Name: "echo", Adhoc: ServeAdhoc{ExtraVars: []string{"token"}, SecretVars: []string{"token"}}}}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	s.runJob = func(ctx context.Context, j *ServeJob, _ string) (*JobResult, error) {
		obs, ok := ctx.Value(runObserverKey{}).(engine.ExecutionObserverI)
		if !ok {
			return nil, fmt.Errorf("missing observer")
		}
		obs.OnPlayStart("play "+j.ExtraVars["token"], 1, 1)
		obs.OnTaskStart("task "+j.ExtraVars["token"], "local")
		obs.OnTaskEnd(&types.TaskResult{TaskName: "task " + j.ExtraVars["token"], Host: "local", Failed: true, Error: "secret-error", Output: map[string]interface{}{"stdout": "secret-out"}})
		// The worker's final status is authoritative even after the engine finishes.
		obs.OnExecutionEnd(&types.PlaybookResult{}, time.Millisecond)
		return &JobResult{Status: "failed"}, nil
	}
	run, err := s.request(&cfg.Jobs[0], "test", &RunParams{ExtraVars: map[string]string{"token": "private-value"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := contextTimeout(t)
	done := make(chan struct{})
	go func() { defer close(done); s.worker(ctx) }()
	defer func() { cancel(); <-done }()
	serveWaitActivity(t, s, 0, 0)
	s.mu.Lock()
	p := s.progress[run.ID]
	s.mu.Unlock()
	if p == nil {
		t.Fatal("progress not retained")
	}
	snapshot, events, _, ch := p.subscribe(0)
	defer p.unsubscribe(ch)
	if snapshot.Status != "failed" || events[len(events)-1].Type != "end" || events[len(events)-1].Status != "failed" {
		t.Fatalf("wrong final progress: %+v %+v", snapshot, events)
	}
	data, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private-value") || !strings.Contains(string(data), "task ***") || !strings.Contains(string(data), "play ***") {
		t.Fatalf("worker masking: %s", data)
	}
}

func TestServeProgressPendingRejected(t *testing.T) {
	s := progressTestServer(t)
	s.results["job"] = []*JobResult{{ID: "pending", Status: "pending"}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &progressWriter{header: make(http.Header), lines: make(chan []byte, 10), ctx: ctx}
	req := httptest.NewRequest("GET", "/api/results/pending/events", nil).WithContext(ctx)
	req.Header.Set("X-User", "viewer")
	done := make(chan struct{})
	go func() { defer close(done); s.mux().ServeHTTP(w, req) }()
	if progressLine(t, w.lines)["type"] != "snapshot" {
		t.Fatal("missing waiting snapshot")
	}
	s.mu.Lock()
	s.results["job"][0].Status = "rejected"
	s.mu.Unlock()
	event := progressLine(t, w.lines)
	if event["type"] != "end" || event["status"] != "rejected" {
		t.Fatalf("pending end: %+v", event)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("rejected stream did not end")
	}
}
