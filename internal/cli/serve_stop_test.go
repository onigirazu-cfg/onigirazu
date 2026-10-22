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
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// stoppingServe runs workers as runServe does: kill cancels runs, done
// closes when every worker is back
type stoppingServe struct {
	s       *server
	started <-chan string
	release map[string]chan struct{}
	kill    context.CancelFunc
	done    chan struct{}
}

func newStoppingServe(t *testing.T, concurrency int) *stoppingServe {
	t.Helper()
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 5, Concurrency: concurrency}
	release := map[string]chan struct{}{}
	for _, name := range []string{"a", "b", "c"} {
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
	ctx, kill := contextTimeout(t)
	var workers sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		workers.Add(1)
		go func() { defer workers.Done(); s.worker(ctx) }()
	}
	done := make(chan struct{})
	go func() { workers.Wait(); close(done) }()
	t.Cleanup(func() { kill(); <-done })
	return &stoppingServe{s: s, started: started, release: release, kill: kill, done: done}
}

// stop runs drainStop in the background; the channel closes when it returns
func (ss *stoppingServe) stop(timeout time.Duration, again <-chan os.Signal) <-chan struct{} {
	returned := make(chan struct{})
	go func() { ss.s.drainStop(timeout, again, ss.kill, ss.done); close(returned) }()
	return returned
}

func (ss *stoppingServe) result(t *testing.T, job, id string) *JobResult {
	t.Helper()
	ss.s.mu.Lock()
	defer ss.s.mu.Unlock()
	return resultCopy(ss.s.resultByID(job, id))
}

func waitStopping(t *testing.T, s *server) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		s.mu.Lock()
		stopping := s.stopping != nil
		s.mu.Unlock()
		if stopping {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("stop did not begin")
		}
		time.Sleep(time.Millisecond)
	}
}

func returnedWithin(t *testing.T, ch <-chan struct{}, d time.Duration, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(d):
		t.Fatalf("%s: drainStop did not return", what)
	}
}

func TestServeStopDrainsRunningKeepsQueued(t *testing.T) {
	ss := newStoppingServe(t, 1)
	s := ss.s
	running, err := s.request(&s.cfg.Jobs[0], "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	serveStart(t, ss.started)
	waiting, err := s.request(&s.cfg.Jobs[1], "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	returned := ss.stop(time.Hour, nil)
	waitStopping(t, s)

	// no new runs while it stops; the page still answers and says why
	if _, err := s.request(&s.cfg.Jobs[2], "test", nil); !errors.Is(err, errServeStopping) {
		t.Fatalf("request while stopping: %v", err)
	}
	w := httptest.NewRecorder()
	s.mux().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/jobs/c/run", nil))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "serve is stopping") {
		t.Fatalf("run while stopping: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	s.mux().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	var me struct {
		Running  int        `json:"running"`
		Queued   int        `json:"queued"`
		Stopping *serveStop `json:"stopping"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &me); err != nil || me.Stopping == nil || me.Running != 1 || me.Queued != 1 ||
		me.Stopping.Deadline.Sub(me.Stopping.Since) != time.Hour {
		t.Fatalf("/api/me while stopping: %s %v", w.Body.String(), err)
	}
	select {
	case <-returned:
		t.Fatal("drainStop returned with a run still going")
	case <-time.After(50 * time.Millisecond):
	}

	close(ss.release["a"])
	returnedWithin(t, returned, 3*time.Second, "after the run finished")
	serveNoStart(t, ss.started)
	if got := ss.result(t, "a", running.ID); got.Status != "ok" {
		t.Fatalf("drained run: %+v", got)
	}
	if got := ss.result(t, "b", waiting.ID); got.Status != "queued" {
		t.Fatalf("queued run: %+v", got)
	}

	// the next start queues it again
	s2, err := newServer(s.cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.resultByID("b", waiting.ID); got == nil || got.Status != "queued" || !s2.running["b"] || len(s2.queue) != 1 {
		t.Fatalf("queued run after restart: %+v queue=%d", got, len(s2.queue))
	}
	if q := <-s2.queue; q.run.ID != waiting.ID || q.job.Name != "b" {
		t.Fatalf("requeued: %+v", q.run)
	}
}

func TestServeStopTimeoutCancels(t *testing.T) {
	ss := newStoppingServe(t, 1)
	r, err := ss.s.request(&ss.s.cfg.Jobs[0], "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	serveStart(t, ss.started)
	begin := time.Now()
	returnedWithin(t, ss.stop(100*time.Millisecond, nil), 3*time.Second, "at the timeout")
	if time.Since(begin) < 100*time.Millisecond {
		t.Fatal("canceled before shutdown_timeout")
	}
	if got := ss.result(t, "a", r.ID); got.Status != "error" || got.Summary != "interrupted: serve stopping" {
		t.Fatalf("timed out stop: %+v", got)
	}
}

func TestServeStopSecondSignalCancels(t *testing.T) {
	ss := newStoppingServe(t, 1)
	r, err := ss.s.request(&ss.s.cfg.Jobs[0], "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	serveStart(t, ss.started)
	again := make(chan os.Signal, 1)
	returned := ss.stop(time.Hour, again)
	waitStopping(t, ss.s)
	again <- os.Interrupt
	returnedWithin(t, returned, 3*time.Second, "after the second signal")
	if got := ss.result(t, "a", r.ID); got.Status != "error" || got.Summary != "interrupted: serve stopping" {
		t.Fatalf("second signal: %+v", got)
	}
}

func TestServeStopZeroTimeoutCancelsAtOnce(t *testing.T) {
	ss := newStoppingServe(t, 1)
	r, err := ss.s.request(&ss.s.cfg.Jobs[0], "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	serveStart(t, ss.started)
	returnedWithin(t, ss.stop(0, nil), 3*time.Second, "with shutdown_timeout 0")
	if got := ss.result(t, "a", r.ID); got.Status != "error" || got.Summary != "interrupted: serve stopping" {
		t.Fatalf("immediate stop: %+v", got)
	}
}

func TestServeStopIdleReturnsAndKeepsScheduledAndPending(t *testing.T) {
	ss := newStoppingServe(t, 1)
	s := ss.s
	s.cfg.Jobs[2].Approval = "required"
	s.cfg.Jobs[2].ApprovalTimeout = time.Hour
	later := time.Now().Add(time.Hour)
	scheduled, err := s.request(&s.cfg.Jobs[0], "test", &RunParams{NotBefore: &later})
	if err != nil || scheduled.Status != "scheduled" {
		t.Fatalf("scheduled: %+v %v", scheduled, err)
	}
	pending, err := s.request(&s.cfg.Jobs[2], "test", nil)
	if err != nil || pending.Status != "pending" {
		t.Fatalf("pending: %+v %v", pending, err)
	}
	returnedWithin(t, ss.stop(time.Hour, nil), 3*time.Second, "with nothing running")
	// a due scheduled run is not released and an approval does not queue
	s.releaseScheduled(later.Add(time.Minute))
	if _, err := s.approve(pending.ID, "alice", ""); !errors.Is(err, errServeStopping) {
		t.Fatalf("approve while stopping: %v", err)
	}
	serveNoStart(t, ss.started)
	if got := ss.result(t, "a", scheduled.ID); got.Status != "scheduled" {
		t.Fatalf("scheduled after stop: %+v", got)
	}
	if got := ss.result(t, "c", pending.ID); got.Status != "pending" || got.Approval.Decision != "" {
		t.Fatalf("pending after stop: %+v", got)
	}
}

func TestServeShutdownTimeoutConfig(t *testing.T) {
	for _, tc := range []struct {
		name, config, wantError string
		want                    time.Duration
	}{
		{"default", "jobs: [{name: patch, playbook: patch.yml}]", "", defaultShutdownTimeout},
		{"explicit", "shutdown_timeout: 30m\njobs: [{name: patch, playbook: patch.yml}]", "", 30 * time.Minute},
		{"zero", "shutdown_timeout: 0s\njobs: [{name: patch, playbook: patch.yml}]", "", 0},
		{"negative", "shutdown_timeout: -1s\njobs: [{name: patch, playbook: patch.yml}]", "shutdown_timeout must be >= 0", 0},
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
				return
			}
			if err != nil || cfg.shutdownTimeout() != tc.want {
				t.Fatalf("shutdown_timeout = %v (%v), want %v", cfg.shutdownTimeout(), err, tc.want)
			}
			unit, err := renderServeUnit(serveUnitOptions{Exec: "/usr/bin/onigirazu", ServeConfig: path, ShutdownTimeout: cfg.shutdownTimeout()})
			if err != nil || !strings.Contains(unit, "KillMode=mixed\n") ||
				!strings.Contains(unit, "TimeoutStopSec="+strconv.Itoa(int((tc.want+serveStopMargin).Seconds()))+"\n") {
				t.Fatalf("unit: %s %v", unit, err)
			}
		})
	}
}

func TestServeStopKeepsDueScheduleSlot(t *testing.T) {
	now := scheduleStateTime("2026-10-22T12:17:00Z")
	s := schedulingServer(t, ServeJob{Every: time.Hour})
	s.planSchedules(now)
	slot := s.scheduleState["patch"].Next
	s.stopping = &serveStop{Since: now, Deadline: now.Add(time.Hour)}
	s.scheduleTick(slot.Add(time.Second))
	if rec := s.scheduleState["patch"]; len(s.queue) != 0 || !rec.Next.Equal(slot) || !s.nextRun["patch"].Equal(slot) {
		t.Fatalf("slot consumed while stopping: queue=%d record=%+v", len(s.queue), rec)
	}
	data, err := os.ReadFile(filepath.Join(s.cfg.DataDir, "schedule.json"))
	if err != nil || !strings.Contains(string(data), slot.Format(time.RFC3339)) {
		t.Fatalf("saved schedule: %s %v", data, err)
	}
}

func TestServeRestartRequeuesBacklogLongerThanQueue(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Retain: 5}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	n := cap(s.queue) + 1 // a paused worker holds one run outside the queue
	base := time.Now().UTC()
	for i := 0; i < n; i++ {
		name := "job" + strconv.Itoa(i)
		cfg.Jobs = append(cfg.Jobs, ServeJob{Name: name, Kind: "drift"})
		s.saveResult(&JobResult{ID: name + "-1", Job: name, Kind: "drift", Status: "queued", Requested: base.Add(time.Duration(i) * time.Second)})
	}
	restored, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.queue) != n {
		t.Fatalf("requeued %d of %d", len(restored.queue), n)
	}
	for i := 0; i < n; i++ {
		if q := <-restored.queue; q.run.Job != "job"+strconv.Itoa(i) || q.run.Status != "queued" {
			t.Fatalf("position %d: %+v", i, q.run)
		}
	}
}
