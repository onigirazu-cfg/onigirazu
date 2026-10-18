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
	"strings"
	"testing"
	"time"
)

func schedulingServer(t *testing.T, job ServeJob) *server {
	t.Helper()
	if job.Name == "" {
		job.Name = "patch"
	}
	if job.Kind == "" {
		job.Kind = "apply"
	}
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Jobs: []ServeJob{job}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func schedulingHTTP(s *server, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	s.mux().ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}

func TestServeWindowParsingAndNext(t *testing.T) {
	for _, tc := range []struct {
		spec, zone, now string
		open            bool
		next            string
	}{
		{"mon-fri 09:00-17:00", "UTC", "2026-10-16T16:59:00Z", true, "2026-10-19T09:00:00Z"},
		{"sat,sun 23:00-01:00", "UTC", "2026-10-19T00:30:00Z", true, "2026-10-24T23:00:00Z"},
		{"* 09:00-10:00", "UTC", "2026-10-18T10:00:00Z", false, "2026-10-19T09:00:00Z"},
		{"mon 23:00-01:00", "UTC", "2026-10-20T01:00:00Z", false, "2026-10-26T23:00:00Z"},
		{"sun 03:00-04:00", "Europe/Madrid", "2026-03-28T12:00:00Z", false, "2026-03-29T01:00:00Z"},
		{"sun 02:30-03:30", "Europe/Madrid", "2026-10-24T12:00:00Z", false, "2026-10-25T00:30:00Z"},
		{"sun 02:30-03:30", "Europe/Madrid", "2026-10-25T00:45:00Z", true, "2026-10-25T01:30:00Z"},
	} {
		t.Run(tc.spec+tc.now, func(t *testing.T) {
			s := schedulingServer(t, ServeJob{Windows: []string{tc.spec}, WindowsTZ: tc.zone})
			j := &s.cfg.Jobs[0]
			now, err := time.Parse(time.RFC3339, tc.now)
			if err != nil {
				t.Fatal(err)
			}
			if got := j.windowOpen(now); got != tc.open {
				t.Fatalf("open = %v, want %v", got, tc.open)
			}
			if got := j.nextWindow(now).Format(time.RFC3339); got != tc.next {
				t.Fatalf("next = %s, want %s", got, tc.next)
			}
		})
	}
	for _, spec := range []string{"", "monday 09:00-10:00", "mon-foo 09:00-10:00", "mon,,tue 09:00-10:00", "mon 25:00-26:00", "mon 9:00-10:00", "* 09:00-09:00", "* 09:00", "* 09:00-10:00 extra"} {
		if _, _, err := parseServeWindows([]string{spec}, "UTC"); err == nil {
			t.Errorf("accepted invalid window %q", spec)
		}
	}
	if _, _, err := parseServeWindows(nil, "No/Such_Zone"); err == nil {
		t.Error("accepted invalid zone")
	}
	for _, fragment := range []string{"windows: [\"mon 25:00-26:00\"]", "windows_tz: No/Such_Zone"} {
		p := filepath.Join(t.TempDir(), "serve.yml")
		if err := os.WriteFile(p, []byte("jobs:\n  - name: patch\n    kind: apply\n    playbook: patch.yml\n    "+fragment+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadServeConfig(p); err == nil || !strings.Contains(err.Error(), "windows") {
			t.Fatalf("invalid config: %v", err)
		}
	}
}

func TestServeRunLaterAPI(t *testing.T) {
	s := schedulingServer(t, ServeJob{})
	at := time.Now().UTC().Add(time.Hour)
	body := fmt.Sprintf(`{"not_before":%q}`, at.Format(time.RFC3339Nano))
	w := schedulingHTTP(s, "POST", "/api/jobs/patch/run", body)
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || w.Code != 202 || response["scheduled"] == "" {
		t.Fatalf("schedule: %d %s (%v)", w.Code, w.Body.String(), err)
	}
	r := s.resultByID("patch", response["run"])
	if r.Status != "scheduled" || r.NotBefore == nil || !r.NotBefore.Equal(at) || s.running["patch"] {
		t.Fatalf("scheduled run: %+v", r)
	}
	if w := schedulingHTTP(s, "POST", "/api/jobs/patch/run", body); w.Code != 409 || !strings.Contains(w.Body.String(), "a run is already scheduled for this job") {
		t.Fatalf("duplicate: %d %s", w.Code, w.Body.String())
	}
	for _, value := range []string{time.Now().Add(-time.Hour).Format(time.RFC3339Nano), time.Now().Add(31 * 24 * time.Hour).Format(time.RFC3339Nano), "tomorrow", ""} {
		if w := schedulingHTTP(s, "POST", "/api/jobs/patch/run", fmt.Sprintf(`{"not_before":%q}`, value)); w.Code != 400 {
			t.Errorf("invalid %q: %d %s", value, w.Code, w.Body.String())
		}
	}
	if w := schedulingHTTP(s, "GET", "/api/me", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"scheduled":1`) {
		t.Fatalf("me: %d %s", w.Code, w.Body.String())
	}
	if w := schedulingHTTP(s, "GET", "/api/jobs", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"scheduled":{`) || !strings.Contains(w.Body.String(), at.Format(time.RFC3339Nano)) {
		t.Fatalf("jobs: %d %s", w.Code, w.Body.String())
	}
	if w := schedulingHTTP(s, "GET", "/metrics", ""); !strings.Contains(w.Body.String(), "onigirazu_serve_scheduled 1") {
		t.Fatalf("metrics: %s", w.Body.String())
	}
	actions := serveAuditEntries(s, 10)
	found := false
	for _, a := range actions {
		if a.OK && strings.Contains(a.Reason, "not_before="+at.Format(time.RFC3339Nano)) {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing audit time: %+v", actions)
	}
	payload, err := s.notifyBody(eventScheduled, &s.cfg.Jobs[0], resultCopy(r), "json")
	if err != nil || !strings.Contains(string(payload), `"event":"scheduled"`) || !strings.Contains(string(payload), at.Format(time.RFC3339Nano)) {
		t.Fatalf("notification: %s %v", payload, err)
	}
	stopped, err := s.cancel(r.ID, "operator", "later stopped")
	if err != nil || stopped.Status != statusCancelled || stopped.Cancel == nil {
		t.Fatalf("cancel: %+v %v", stopped, err)
	}
	s.releaseScheduled(at.Add(time.Second))
	if len(s.queue) != 0 {
		t.Fatal("a stopped run was queued")
	}
}

func TestServeRunLaterDueAndBusy(t *testing.T) {
	s := schedulingServer(t, ServeJob{})
	j := &s.cfg.Jobs[0]
	at := time.Now().Add(time.Hour)
	later, err := s.request(j, "test", &RunParams{NotBefore: &at})
	if err != nil {
		t.Fatal(err)
	}
	immediate, err := s.request(j, "schedule", nil)
	if err != nil || immediate.Status != "queued" {
		t.Fatalf("immediate while scheduled: %+v %v", immediate, err)
	}
	s.releaseScheduled(at.Add(time.Second))
	r := s.resultByID(j.Name, later.ID)
	if r.Status != "scheduled" || r.Summary != "waiting for the job to be free" || !s.running[j.Name] {
		t.Fatalf("busy: %+v", r)
	}
	<-s.queue
	s.running[j.Name] = false
	s.releaseScheduled(at.Add(time.Second))
	if r.Status != "queued" || !s.running[j.Name] || len(s.queue) != 1 {
		t.Fatalf("released: %+v", r)
	}
	if q := <-s.queue; q.run.ID != later.ID {
		t.Fatalf("queued wrong run: %+v", q.run)
	}
}

func TestServeRunLaterScheduler(t *testing.T) {
	s := schedulingServer(t, ServeJob{})
	s.schedulerTick = 10 * time.Millisecond
	at := time.Now().Add(100 * time.Millisecond)
	r, err := s.request(&s.cfg.Jobs[0], "test", &RunParams{NotBefore: &at})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); s.scheduler(ctx) }()
	defer func() { cancel(); <-done }()
	select {
	case q := <-s.queue:
		if q.run.ID != r.ID || time.Now().Before(at) {
			t.Fatalf("released wrong run or too early: %+v", q.run)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("scheduled run was not released")
	}
}

func TestServeRunLaterApprovalAndRestart(t *testing.T) {
	for _, secret := range []bool{false, true} {
		t.Run(fmt.Sprint(secret), func(t *testing.T) {
			s := schedulingServer(t, ServeJob{Approval: "required", ApprovalTimeout: time.Hour, Adhoc: ServeAdhoc{ExtraVars: []string{"token"}, SecretVars: []string{"token"}}})
			at := time.Now().Add(time.Hour)
			p := &RunParams{NotBefore: &at}
			if secret {
				p.ExtraVars = map[string]string{"token": "private-value"}
			}
			r, err := s.request(&s.cfg.Jobs[0], "test", p)
			if err != nil || r.Status != "pending" {
				t.Fatalf("pending: %+v %v", r, err)
			}
			r, err = s.approve(r.ID, "operator", "approved")
			if err != nil || r.Status != "scheduled" || s.running["patch"] {
				t.Fatalf("approval: %+v %v", r, err)
			}
			restarted, err := newServer(s.cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			r = restarted.resultByID("patch", r.ID)
			if secret {
				if r.Status != "error" || !strings.Contains(r.Summary, "secret parameters are not kept") {
					t.Fatalf("secret restart: %+v", r)
				}
				return
			}
			if r.Status != "scheduled" || r.NotBefore == nil || !r.NotBefore.Equal(at) || restarted.running["patch"] {
				t.Fatalf("restart: %+v", r)
			}
			restarted.releaseScheduled(at.Add(time.Second))
			if r.Status != "queued" || len(restarted.queue) != 1 {
				t.Fatalf("rearmed: %+v", r)
			}
		})
	}
	s := schedulingServer(t, ServeJob{Approval: "required", ApprovalTimeout: time.Hour})
	at := time.Now().Add(time.Hour)
	r, err := s.request(&s.cfg.Jobs[0], "test", &RunParams{NotBefore: &at})
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Second)
	s.resultByID("patch", r.ID).NotBefore = &past
	approved, err := s.approve(r.ID, "operator", "")
	if err != nil || approved.Status != "queued" {
		t.Fatalf("late approval: %+v %v", approved, err)
	}
}

func TestServeWindowGating(t *testing.T) {
	now := time.Now().UTC()
	outside := now.Add(2*time.Hour).Format("15:04") + "-" + now.Add(3*time.Hour).Format("15:04")
	inside := now.Add(-time.Hour).Format("15:04") + "-" + now.Add(time.Hour).Format("15:04")
	for _, tc := range []struct{ kind, mode, window, status string }{
		{"apply", "", outside, "scheduled"}, {"apply", "check", outside, "queued"}, {"drift", "", outside, "queued"}, {"apply", "", inside, "queued"},
	} {
		t.Run(tc.kind+tc.mode+tc.status, func(t *testing.T) {
			s := schedulingServer(t, ServeJob{Kind: tc.kind, Windows: []string{"* " + tc.window}})
			j := &s.cfg.Jobs[0]
			r, err := s.request(j, "schedule", &RunParams{Mode: tc.mode})
			if err != nil || r.Status != tc.status {
				t.Fatalf("gated: %+v %v", r, err)
			}
			if tc.status == "scheduled" && (r.NotBefore == nil || !r.NotBefore.Equal(j.nextWindow(now))) {
				t.Fatalf("next window: %+v", r)
			}
		})
	}
	s := schedulingServer(t, ServeJob{Windows: []string{"* " + inside}})
	r, err := s.request(&s.cfg.Jobs[0], "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	windows, loc, err := parseServeWindows([]string{"* " + outside}, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	s.cfg.Jobs[0].windows, s.cfg.Jobs[0].windowsLoc = windows, loc
	s.runJob = func(context.Context, *ServeJob, string) (*JobResult, error) {
		t.Error("worker started outside window")
		return nil, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); s.worker(ctx) }()
	defer func() { cancel(); <-done }()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		got := resultCopy(s.resultByID("patch", r.ID))
		busy := s.running["patch"]
		s.mu.Unlock()
		if got.Status == "scheduled" {
			if busy || got.NotBefore == nil {
				t.Fatalf("pickup gating: %+v", got)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("worker did not reschedule outside window")
}

func TestServeRunLaterNotifyOptIn(t *testing.T) {
	for _, opted := range []bool{false, true} {
		t.Run(fmt.Sprint(opted), func(t *testing.T) {
			target := NotifyTarget{URL: "http://notify.test/scheduled", Format: "json"}
			if opted {
				target.On = []string{eventScheduled, eventCancelled}
			}
			s := schedulingServer(t, ServeJob{Notify: []NotifyTarget{target}, NotifyOK: true})
			messages := make(chan []byte, 4)
			s.notifyClient = &http.Client{Transport: serveNotifyTransport(func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				messages <- body
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
			})}
			at := time.Now().Add(time.Hour)
			r, err := s.request(&s.cfg.Jobs[0], "test", &RunParams{NotBefore: &at})
			if err != nil {
				t.Fatal(err)
			}
			s.notifyWait.Wait()
			if opted {
				if len(messages) != 1 {
					t.Fatalf("scheduled messages: %d", len(messages))
				}
				body := <-messages
				if !strings.Contains(string(body), `"event":"scheduled"`) || !strings.Contains(string(body), at.Format(time.RFC3339Nano)) {
					t.Fatalf("scheduled payload: %s", body)
				}
			} else if len(messages) != 0 {
				t.Fatal("scheduled event was not opt-in")
			}
			if _, err := s.cancel(r.ID, "operator", "cancel future"); err != nil {
				t.Fatal(err)
			}
			s.notifyWait.Wait()
			if len(messages) != 1 || !strings.Contains(string(<-messages), `"event":"`+statusCancelled+`"`) {
				t.Fatal("scheduled cancel did not notify")
			}
		})
	}
}

func TestServeWindowApprovalAndJobs(t *testing.T) {
	now := time.Now().UTC()
	spec := "* " + now.Add(2*time.Hour).Format("15:04") + "-" + now.Add(3*time.Hour).Format("15:04")
	s := schedulingServer(t, ServeJob{Approval: "required", ApprovalTimeout: time.Hour, Windows: []string{spec}})
	r, err := s.request(&s.cfg.Jobs[0], "test", nil)
	if err != nil || r.Status != "pending" {
		t.Fatalf("pending: %+v %v", r, err)
	}
	r, err = s.approve(r.ID, "operator", "window approval")
	if err != nil || r.Status != "scheduled" || r.NotBefore == nil || !r.NotBefore.Equal(s.cfg.Jobs[0].nextWindow(now)) {
		t.Fatalf("window approval: %+v %v", r, err)
	}
	w := schedulingHTTP(s, "GET", "/api/jobs", "")
	var jobs []jobView
	if err := json.Unmarshal(w.Body.Bytes(), &jobs); err != nil || len(jobs) != 1 {
		t.Fatalf("jobs: %s %v", w.Body.String(), err)
	}
	j := jobs[0]
	if len(j.Windows) != 1 || j.Windows[0] != spec || j.WindowsTZ != "UTC" || j.NextWindow == nil || !j.NextWindow.Equal(*r.NotBefore) {
		t.Fatalf("window view: %+v", j)
	}
}

func TestServeRunLaterKeepsActiveSlotAndView(t *testing.T) {
	s := schedulingServer(t, ServeJob{})
	j := &s.cfg.Jobs[0]
	active, err := s.request(j, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(time.Hour)
	later, err := s.request(j, "test", &RunParams{NotBefore: &at})
	if err != nil {
		t.Fatal(err)
	}
	w := schedulingHTTP(s, "GET", "/api/jobs", "")
	var jobs []jobView
	if err := json.Unmarshal(w.Body.Bytes(), &jobs); err != nil || len(jobs) != 1 {
		t.Fatalf("jobs: %s %v", w.Body.String(), err)
	}
	if jobs[0].Active == nil || jobs[0].Active.ID != active.ID || jobs[0].Scheduled == nil || jobs[0].Scheduled.ID != later.ID || !jobs[0].Queued || !jobs[0].Running {
		t.Fatalf("active and scheduled: %+v", jobs[0])
	}
	if _, err := s.cancel(later.ID, "operator", "cancel future"); err != nil {
		t.Fatal(err)
	}
	if !s.running[j.Name] || s.resultByID(j.Name, active.ID).Status != "queued" {
		t.Fatal("scheduled cancellation released active run's slot")
	}
	if _, err := s.request(j, "test", nil); err != errJobBusy {
		t.Fatalf("busy job accepted another immediate run: %v", err)
	}
	if len(s.queue) != 1 {
		t.Fatalf("queue length: %d", len(s.queue))
	}
}

func TestServeRunLaterRetention(t *testing.T) {
	s := schedulingServer(t, ServeJob{})
	s.cfg.Retain = 1
	j := &s.cfg.Jobs[0]
	at := time.Now().Add(time.Hour)
	later, err := s.request(j, "test", &RunParams{NotBefore: &at})
	if err != nil {
		t.Fatal(err)
	}
	finished, err := s.request(j, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	<-s.queue
	r := s.resultByID(j.Name, finished.ID)
	r.Status = "ok"
	s.running[j.Name] = false
	if err := s.store(r); err != nil {
		t.Fatal(err)
	}
	if s.resultByID(j.Name, later.ID) == nil {
		t.Fatal("retention removed scheduled run")
	}
	restarted, err := newServer(s.cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	saved := restarted.resultByID(j.Name, later.ID)
	if saved == nil || saved.Status != "scheduled" || saved.NotBefore == nil || !saved.NotBefore.Equal(at) {
		t.Fatalf("retained scheduled run: %+v", saved)
	}
	if newest := restarted.resultByID(j.Name, finished.ID); newest == nil || newest.Status != "ok" {
		t.Fatalf("retained finished run: %+v", newest)
	}
	restarted.releaseScheduled(at.Add(time.Second))
	if saved.Status != "queued" || len(restarted.queue) != 1 {
		t.Fatalf("retained run was not rearmed: %+v", saved)
	}
}

func TestServeRunLaterApprovedNotification(t *testing.T) {
	s := schedulingServer(t, ServeJob{Approval: "required", ApprovalTimeout: time.Hour, Notify: []NotifyTarget{{URL: "http://notify.test/approval", Format: "json", On: []string{eventScheduled}}}})
	messages := make(chan []byte, 1)
	s.notifyClient = &http.Client{Transport: serveNotifyTransport(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		messages <- body
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}
	at := time.Now().Add(time.Hour)
	r, err := s.request(&s.cfg.Jobs[0], "test", &RunParams{NotBefore: &at})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.approve(r.ID, "reviewer", "approved for tonight"); err != nil {
		t.Fatal(err)
	}
	s.notifyWait.Wait()
	if len(messages) != 1 {
		t.Fatalf("scheduled notification count: %d", len(messages))
	}
	var message struct {
		Event string    `json:"event"`
		Run   JobResult `json:"run"`
	}
	body := <-messages
	if err := json.Unmarshal(body, &message); err != nil {
		t.Fatal(err)
	}
	approval := message.Run.Approval
	if message.Event != eventScheduled || message.Run.Status != "scheduled" || approval == nil || approval.Decision != "approved" || approval.By != "reviewer" || approval.At == nil || approval.Comment != "approved for tonight" {
		t.Fatalf("approval notification: %s", body)
	}
}

func TestServeWindowSpringDSTAndOverlap(t *testing.T) {
	now := time.Date(2026, time.March, 28, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct{ spec, next string }{
		{"sun 02:30-03:30", "2026-03-29T01:00:00Z"},
		{"sun 02:15-02:45", "2026-04-05T00:15:00Z"},
	} {
		s := schedulingServer(t, ServeJob{Windows: []string{tc.spec}, WindowsTZ: "Europe/Madrid"})
		if got := s.cfg.Jobs[0].nextWindow(now).Format(time.RFC3339); got != tc.next {
			t.Errorf("%s next = %s, want %s", tc.spec, got, tc.next)
		}
	}
	s := schedulingServer(t, ServeJob{Windows: []string{"* 00:00-13:00", "* 12:00-01:00"}})
	j := &s.cfg.Jobs[0]
	if !j.windowOpen(now) {
		t.Fatal("overlapping windows should be open")
	}
	next := j.nextWindow(now)
	if next.IsZero() || !next.After(now) || !j.windowOpen(next) {
		t.Fatalf("next start under continuous coverage: %v", next)
	}
}

func TestServeRunLaterAutomaticDeferralCollision(t *testing.T) {
	for _, approval := range []bool{false, true} {
		t.Run(fmt.Sprint(approval), func(t *testing.T) {
			now := time.Now().UTC()
			open := "* " + now.Add(-time.Hour).Format("15:04") + "-" + now.Add(time.Hour).Format("15:04")
			closed := "* " + now.Add(2*time.Hour).Format("15:04") + "-" + now.Add(3*time.Hour).Format("15:04")
			s := schedulingServer(t, ServeJob{Windows: []string{open}, ApprovalTimeout: time.Hour})
			j := &s.cfg.Jobs[0]
			at := now.Add(24 * time.Hour)
			future, err := s.request(j, "test", &RunParams{NotBefore: &at})
			if err != nil {
				t.Fatal(err)
			}
			if approval {
				j.Approval = "required"
			}
			immediate, err := s.request(j, "test", nil)
			if err != nil {
				t.Fatal(err)
			}
			expected := "queued"
			if approval {
				expected = "pending"
			}
			if immediate.Status != expected {
				t.Fatalf("immediate: %+v", immediate)
			}
			windows, loc, err := parseServeWindows([]string{closed}, "UTC")
			if err != nil {
				t.Fatal(err)
			}
			j.windows, j.windowsLoc = windows, loc
			if approval {
				immediate, err = s.approve(immediate.ID, "reviewer", "approved after window")
				if err != nil || immediate.Status != "scheduled" {
					t.Fatalf("approval collision: %+v %v", immediate, err)
				}
			} else {
				s.runJob = func(context.Context, *ServeJob, string) (*JobResult, error) {
					t.Error("run started outside maintenance window")
					return nil, nil
				}
				ctx, cancel := context.WithCancel(context.Background())
				done := make(chan struct{})
				go func() { defer close(done); s.worker(ctx) }()
				deadline := time.Now().Add(time.Second)
				scheduled := false
				for time.Now().Before(deadline) {
					s.mu.Lock()
					status := s.resultByID(j.Name, immediate.ID).Status
					s.mu.Unlock()
					if status == "scheduled" {
						scheduled = true
						break
					}
					time.Sleep(time.Millisecond)
				}
				cancel()
				<-done
				if !scheduled {
					t.Fatal("worker did not preserve automatic deferral")
				}
			}
			w := schedulingHTTP(s, "GET", "/api/jobs", "")
			var jobs []jobView
			if err := json.Unmarshal(w.Body.Bytes(), &jobs); err != nil || len(jobs) != 1 {
				t.Fatalf("jobs: %s %v", w.Body.String(), err)
			}
			if len(jobs[0].ScheduledRuns) != 2 || jobs[0].Scheduled == nil || jobs[0].Active != nil || jobs[0].Running {
				t.Fatalf("collision jobs view: %s", w.Body.String())
			}
			ids := map[string]bool{future.ID: false, immediate.ID: false}
			for _, r := range jobs[0].ScheduledRuns {
				if _, ok := ids[r.ID]; !ok || r.Status != "scheduled" || r.NotBefore == nil || r.Report != nil {
					t.Fatalf("unexpected collision run: %+v", r)
				}
				ids[r.ID] = true
			}
			if !ids[future.ID] || !ids[immediate.ID] {
				t.Fatalf("missing deferral: %v", ids)
			}
			body := fmt.Sprintf(`{"not_before":%q}`, at.Add(time.Hour).Format(time.RFC3339Nano))
			if w := schedulingHTTP(s, "POST", "/api/jobs/patch/run", body); w.Code != 409 || !strings.Contains(w.Body.String(), "a run is already scheduled for this job") {
				t.Fatalf("explicit collision: %d %s", w.Code, w.Body.String())
			}
			check, err := s.request(j, "test", &RunParams{Mode: "check"})
			if err != nil || check.Status != "queued" {
				t.Fatalf("check with deferrals: %+v %v", check, err)
			}
			for _, id := range []string{future.ID, immediate.ID} {
				stopped, err := s.cancel(id, "operator", "cancel deferral")
				if err != nil || stopped.Status != statusCancelled {
					t.Fatalf("cancel deferral: %+v %v", stopped, err)
				}
				if !s.running[j.Name] || s.resultByID(j.Name, check.ID).Status != "queued" {
					t.Fatal("canceling deferral released check run's slot")
				}
			}
			if len(s.queue) != 1 || s.scheduledCount() != 0 {
				t.Fatalf("post-cancel queue=%d scheduled=%d", len(s.queue), s.scheduledCount())
			}
		})
	}
}

// a weekly window that a DST change skipped is found two weeks on, and a
// window that never opens keeps the run deferred instead of letting it run
func TestServeWindowNeverFailsOpen(t *testing.T) {
	s := schedulingServer(t, ServeJob{Windows: []string{"sun 02:15-02:45"}, WindowsTZ: "Europe/Madrid"})
	j := &s.cfg.Jobs[0]
	from := time.Date(2026, time.March, 22, 4, 0, 0, 0, time.UTC)
	if got := j.nextWindow(from).Format(time.RFC3339); got != "2026-04-05T00:15:00Z" {
		t.Fatalf("next after the skipped Sunday: %s", got)
	}
	j.windows = []serveWindow{{start: 60, end: 120}} // no day: never open
	if at := j.deferUntil(&JobResult{}, from); at == nil || !at.After(from) {
		t.Fatalf("a window that never opens let the run through: %v", at)
	}
}

func TestServeRetainCountsFinishedOnly(t *testing.T) {
	s := schedulingServer(t, ServeJob{})
	s.cfg.Retain = 1
	j := &s.cfg.Jobs[0]
	now := time.Now().UTC()
	older := &JobResult{ID: "older", Job: j.Name, Status: "ok", Requested: now.Add(-2 * time.Hour)}
	newest := &JobResult{ID: "newest", Job: j.Name, Status: "ok", Requested: now.Add(-time.Hour)}
	at := now.Add(time.Hour)
	later := &JobResult{ID: "later", Job: j.Name, Status: "scheduled", NotBefore: &at, Requested: now}
	s.results[j.Name] = []*JobResult{later, newest, older}
	if err := s.pruneResults(j.Name, now); err != nil {
		t.Fatal(err)
	}
	if s.resultByID(j.Name, "later") == nil || s.resultByID(j.Name, "newest") == nil || s.resultByID(j.Name, "older") != nil {
		t.Fatalf("kept: %+v", s.results[j.Name])
	}
}

func TestServeWindowDeferralsNextToAScheduledRun(t *testing.T) {
	now := time.Now().UTC()
	closed := "* " + now.Add(2*time.Hour).Format("15:04") + "-" + now.Add(3*time.Hour).Format("15:04")
	s := schedulingServer(t, ServeJob{Windows: []string{closed}})
	j := &s.cfg.Jobs[0]
	first, err := s.request(j, "api:alice", nil)
	if err != nil || first.Status != "scheduled" {
		t.Fatalf("first deferral: %+v %v", first, err)
	}
	second, err := s.request(j, "api:bob", nil)
	if err != nil || second.Status != "scheduled" {
		t.Fatalf("a caller's run outside the window is deferred too: %+v %v", second, err)
	}
	if _, err := s.request(j, "schedule", nil); !errors.Is(err, errAlreadyScheduled) {
		t.Fatalf("the schedule keeps one deferral: %v", err)
	}
	at := now.Add(time.Hour)
	if _, err := s.request(j, "api:carol", &RunParams{NotBefore: &at}); !errors.Is(err, errAlreadyScheduled) {
		t.Fatalf("one explicit run for later: %v", err)
	}
	// both due at once: the older request goes first
	open := "* " + now.Add(-time.Hour).Format("15:04") + "-" + now.Add(time.Hour).Format("15:04")
	j.windows, j.windowsLoc, _ = parseServeWindows([]string{open}, "UTC")
	for _, r := range []*JobResult{first, second} {
		stored := s.resultByID(j.Name, r.ID)
		past := now.Add(-time.Minute)
		stored.NotBefore = &past
	}
	s.releaseScheduled(now)
	q := <-s.queue
	if q.run.ID != first.ID || s.resultByID(j.Name, second.ID).Status != "scheduled" {
		t.Fatalf("release order: queued %s", q.run.ID)
	}
}
