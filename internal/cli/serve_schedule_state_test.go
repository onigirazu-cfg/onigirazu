package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func scheduleStateTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return t
}

func TestServeSchedulePlanStart(t *testing.T) {
	now := scheduleStateTime("2026-10-22T12:17:00Z")
	ptr := func(v time.Time) *time.Time { return &v }
	tests := []struct {
		name       string
		job        ServeJob
		record     *scheduleRecord
		history    *time.Time
		unreadable bool
		next       time.Time
		reason     string
		missed     *time.Time
		fire       bool
	}{
		{name: "restart before due", job: ServeJob{Kind: "apply", Every: 6 * time.Hour}, record: &scheduleRecord{Schedule: "every=6h0m0s", Next: now.Add(2 * time.Hour)}, next: now.Add(2 * time.Hour), reason: ScheduleReasonScheduled},
		{name: "weekly missed skip keeps phase", job: ServeJob{Kind: "apply", Every: 168 * time.Hour}, record: &scheduleRecord{Schedule: "every=168h0m0s", Next: now.Add(-72 * time.Hour)}, next: now.Add(96 * time.Hour), reason: ScheduleReasonMissedSkipped, missed: ptr(now.Add(-72 * time.Hour))},
		{name: "missed once keeps phase", job: ServeJob{Kind: "drift", Every: time.Hour}, record: &scheduleRecord{Schedule: "every=1h0m0s", Next: now.Add(-5 * time.Hour)}, next: now.Add(time.Hour), reason: ScheduleReasonMissedRunOnce, missed: ptr(now.Add(-5 * time.Hour)), fire: true},
		{name: "thirty days of minute slots", job: ServeJob{Kind: "drift", Every: time.Minute}, record: &scheduleRecord{Schedule: "every=1m0s", Next: now.Add(-30 * 24 * time.Hour)}, next: now.Add(time.Minute), reason: ScheduleReasonMissedRunOnce, missed: ptr(now.Add(-30 * 24 * time.Hour)), fire: true},
		{name: "upgrade history", job: ServeJob{Kind: "apply", Every: 168 * time.Hour}, history: ptr(now.Add(-48 * time.Hour)), next: now.Add(120 * time.Hour), reason: ScheduleReasonScheduled},
		{name: "new drift", job: ServeJob{Kind: "drift", Every: time.Hour}, next: now.Add(time.Hour), reason: ScheduleReasonNew, fire: true},
		{name: "new apply", job: ServeJob{Kind: "apply", Every: time.Hour}, next: now.Add(time.Hour), reason: ScheduleReasonNew},
		{name: "interval changed", job: ServeJob{Kind: "apply", Every: 2 * time.Hour}, record: &scheduleRecord{Schedule: "every=1h0m0s", Last: now.Add(-time.Hour), Next: now.Add(8 * time.Hour)}, next: now.Add(time.Hour), reason: ScheduleReasonScheduled},
		{name: "backward clock", job: ServeJob{Kind: "apply", Every: time.Hour}, record: &scheduleRecord{Schedule: "every=1h0m0s", Next: now.Add(10 * 24 * time.Hour)}, next: now.Add(time.Hour), reason: ScheduleReasonScheduled},
		{name: "unreadable ignores record and history", job: ServeJob{Kind: "drift", Every: time.Hour}, record: &scheduleRecord{Schedule: "every=1h0m0s", Next: now.Add(-time.Hour)}, history: ptr(now.Add(-24 * time.Hour)), unreadable: true, next: now.Add(time.Hour), reason: ScheduleReasonStateUnreadable},
		{name: "at new apply", job: ServeJob{Kind: "apply", At: []string{"fri 04:00"}}, next: scheduleStateTime("2026-10-23T04:00:00Z"), reason: ScheduleReasonNew},
		{name: "at changed uses last", job: ServeJob{Kind: "apply", At: []string{"fri 04:00"}}, record: &scheduleRecord{Schedule: "at=thu 04:00;tz=UTC", Last: now.Add(-time.Hour), Next: now.Add(8 * time.Hour)}, next: scheduleStateTime("2026-10-23T04:00:00Z"), reason: ScheduleReasonScheduled},
		{name: "at backward clock", job: ServeJob{Kind: "apply", At: []string{"fri 04:00"}}, record: &scheduleRecord{Schedule: "at=fri 04:00;tz=UTC", Next: now.Add(10 * 24 * time.Hour)}, next: scheduleStateTime("2026-10-23T04:00:00Z"), reason: ScheduleReasonScheduled},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.job.Name = "example"
			if tc.job.Kind == "" {
				tc.job.Kind = "apply"
			}
			if err := tc.job.prepareSchedule(""); err != nil {
				t.Fatal(err)
			}
			next, reason, missed, fire := planStart(&tc.job, tc.record, tc.history, tc.unreadable, now)
			if !next.Equal(tc.next) || reason != tc.reason || fire != tc.fire {
				t.Fatalf("plan=(%s, %s, fire=%v), want (%s, %s, fire=%v)", next, reason, fire, tc.next, tc.reason, tc.fire)
			}
			if (missed == nil) != (tc.missed == nil) || (missed != nil && !missed.Equal(*tc.missed)) {
				t.Fatalf("missed=%v, want %v", missed, tc.missed)
			}
		})
	}
}

func TestServeScheduleRuntimeClockChanges(t *testing.T) {
	initial := scheduleStateTime("2026-10-22T12:17:00Z")
	for _, policy := range []string{"skip", "run_once"} {
		t.Run(policy, func(t *testing.T) {
			s := schedulingServer(t, ServeJob{Every: 168 * time.Hour, Missed: policy})
			slot := initial.Add(168 * time.Hour)
			data, err := json.Marshal(map[string]any{"version": 1, "jobs": map[string]any{"patch": scheduleRecord{Schedule: s.cfg.Jobs[0].scheduleSignature(), Next: slot}}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(s.cfg.DataDir, "schedule.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			s.planSchedules(initial)
			now := slot.Add(3 * 24 * time.Hour)
			s.scheduleTick(now)
			rec := s.scheduleState["patch"]
			wantReason := ScheduleReasonMissedSkipped
			wantQueue := 0
			if policy == "run_once" {
				wantReason = ScheduleReasonScheduled
				wantQueue = 1
			}
			if len(s.queue) != wantQueue || rec.Reason != wantReason || !rec.Next.Equal(slot.Add(168*time.Hour)) || (policy == "skip" && rec.MissedAt == nil) {
				t.Fatalf("queue=%d record=%+v", len(s.queue), rec)
			}
			s.scheduleTick(now.Add(time.Second))
			if len(s.queue) != wantQueue {
				t.Fatal("repeated the catch-up request")
			}
		})
	}
	for _, job := range []ServeJob{{Every: time.Hour}, {At: []string{"fri 04:00"}}} {
		t.Run("backward/"+job.Every.String(), func(t *testing.T) {
			s := schedulingServer(t, job)
			s.planSchedules(initial)
			now := initial.Add(-10 * 24 * time.Hour)
			s.scheduleTick(now)
			want := now.Add(job.Every)
			if len(job.At) > 0 {
				want = s.cfg.Jobs[0].nextAt(now)
			}
			if !s.nextRun["patch"].Equal(want) || s.scheduleState["patch"].Reason != ScheduleReasonScheduled || len(s.queue) != 0 {
				t.Fatalf("next=%v want=%v record=%+v", s.nextRun["patch"], want, s.scheduleState["patch"])
			}
		})
	}
}

func TestServeScheduleCorruptState(t *testing.T) {
	now := scheduleStateTime("2026-10-22T12:17:00Z")
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Jobs: []ServeJob{
		{Name: "patch", Kind: "drift", Every: time.Hour},
		{Name: "weekly", Kind: "apply", Every: 168 * time.Hour},
		{Name: "anchored", Kind: "verify", At: []string{"* 04:00"}},
	}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.cfg.DataDir, "schedule.json")
	bad := []byte("{not json")
	if err := os.WriteFile(path, bad, 0600); err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return now }
	s.planSchedules(s.now())
	for i := 0; i < 5; i++ {
		s.scheduleTick(now.Add(time.Duration(i) * time.Second))
	}
	if len(s.queue) != 0 {
		t.Fatalf("queue=%d", len(s.queue))
	}
	for name, rec := range s.scheduleState {
		if rec.Reason != ScheduleReasonStateUnreadable {
			t.Fatalf("job=%s record=%+v", name, rec)
		}
	}
	corrupt, err := filepath.Glob(path + ".corrupt-*")
	if err != nil || len(corrupt) != 1 {
		t.Fatalf("preserved files=%v err=%v", corrupt, err)
	}
	got, err := os.ReadFile(corrupt[0])
	if err != nil || string(got) != string(bad) {
		t.Fatalf("preserved=%q err=%v", got, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Version int                        `json:"version"`
		Jobs    map[string]json.RawMessage `json:"jobs"`
	}
	if err := json.Unmarshal(data, &state); err != nil || state.Version != 1 || len(state.Jobs) != 3 {
		t.Fatalf("replacement=%s err=%v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("mode=%v err=%v", info, err)
	}
}

func TestServeScheduleRestartPersistenceAndAPI(t *testing.T) {
	now := scheduleStateTime("2026-10-22T12:17:00Z")
	s := schedulingServer(t, ServeJob{Every: 168 * time.Hour})
	s.now = func() time.Time { return now }
	s.planSchedules(s.now())
	slot := s.nextRun["patch"]
	now = slot
	s.scheduleTick(s.now())
	if len(s.queue) != 1 {
		t.Fatalf("first firing queue=%d", len(s.queue))
	}
	next := s.nextRun["patch"]
	// Keep the completed result as an upgrade/history source as well as schedule state.
	first := <-s.queue
	s.resultByID("patch", first.run.ID).Status = "ok"
	s.running["patch"] = false
	if err := s.store(s.resultByID("patch", first.run.ID)); err != nil {
		t.Fatal(err)
	}
	restart := func(at time.Time) *server {
		t.Helper()
		got, err := newServer(s.cfg, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		got.now = func() time.Time { return at }
		got.planSchedules(got.now())
		got.scheduleTick(got.now())
		return got
	}
	checkAPI := func(got *server, reason string, want time.Time) {
		t.Helper()
		w := schedulingHTTP(got, "GET", "/api/jobs", "")
		var jobs []struct {
			Next    time.Time `json:"next"`
			Reason  string    `json:"next_reason"`
			Last    time.Time `json:"last_scheduled"`
			Outcome string    `json:"last_outcome"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &jobs); err != nil || w.Code != 200 || len(jobs) != 1 {
			t.Fatalf("API=%s err=%v", w.Body.String(), err)
		}
		if !jobs[0].Next.Equal(want) || jobs[0].Reason != reason || !jobs[0].Last.Equal(slot) || !strings.HasPrefix(jobs[0].Outcome, "requested ") {
			t.Fatalf("API=%s", w.Body.String())
		}
	}
	before := restart(slot.Add(time.Hour))
	if len(before.queue) != 0 {
		t.Fatal("restart before due fired")
	}
	checkAPI(before, ScheduleReasonScheduled, next)
	after := restart(next.Add(time.Hour))
	if len(after.queue) != 0 {
		t.Fatal("restart after due fired apply")
	}
	checkAPI(after, ScheduleReasonMissedSkipped, next.Add(168*time.Hour))
}

func TestServeScheduleCatchUpApprovalAndWindows(t *testing.T) {
	now := time.Now().UTC()
	closed := "* " + now.Add(2*time.Hour).Format("15:04") + "-" + now.Add(3*time.Hour).Format("15:04")
	for _, tc := range []struct {
		name   string
		job    ServeJob
		status string
	}{
		{"approval", ServeJob{Kind: "drift", Every: time.Hour, Approval: "required", ApprovalTimeout: time.Hour}, "pending"},
		{"window", ServeJob{Every: time.Hour, Missed: "run_once", Windows: []string{closed}}, "scheduled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := schedulingServer(t, tc.job)
			s.planSchedules(now)
			slot := s.nextRun["patch"]
			s.scheduleTick(slot)
			for i := 1; i < 10; i++ {
				s.scheduleTick(slot.Add(time.Duration(i) * time.Second))
			}
			if len(s.queue) != 0 || len(s.results["patch"]) != 1 || s.results["patch"][0].Status != tc.status {
				t.Fatalf("queue=%d results=%+v", len(s.queue), s.results["patch"])
			}
			rec := s.scheduleState["patch"]
			if !strings.Contains(rec.Outcome, "("+tc.status+")") {
				t.Fatalf("outcome=%q", rec.Outcome)
			}
		})
	}
	t.Run("recovered pending", func(t *testing.T) {
		s := schedulingServer(t, ServeJob{Kind: "drift", Every: time.Hour, Approval: "required", ApprovalTimeout: time.Hour})
		r, err := s.request(&s.cfg.Jobs[0], "test", nil)
		if err != nil {
			t.Fatal(err)
		}
		restored, err := newServer(s.cfg, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		restored.planSchedules(now)
		slot := restored.nextRun["patch"]
		restored.scheduleTick(slot)
		restored.scheduleTick(slot.Add(time.Second))
		rec := restored.scheduleState["patch"]
		if !rec.Last.Equal(slot) || !strings.HasPrefix(rec.Outcome, "skipped: ") || len(restored.results["patch"]) != 1 || restored.results["patch"][0].ID != r.ID || len(restored.queue) != 0 {
			t.Fatalf("record=%+v results=%+v", rec, restored.results["patch"])
		}
	})
}

func TestServeScheduleStartupCatchUpConsumesOneSlot(t *testing.T) {
	now := scheduleStateTime("2026-10-22T12:17:00Z")
	s := schedulingServer(t, ServeJob{Kind: "drift", Every: time.Hour})
	missed := now.Add(-5*time.Hour - 10*time.Minute)
	state := map[string]any{"version": 1, "jobs": map[string]any{"patch": scheduleRecord{Schedule: "every=1h0m0s", Next: missed}}}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.cfg.DataDir, "schedule.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	s.planSchedules(now)
	fire := s.nextRun["patch"]
	if !fire.Equal(now.Add(10*time.Second)) || len(s.queue) != 0 {
		t.Fatalf("startup fire=%s queue=%d", fire, len(s.queue))
	}
	s.scheduleTick(fire.Add(-time.Nanosecond))
	if len(s.queue) != 0 {
		t.Fatal("catch-up fired before the stagger")
	}
	s.scheduleTick(fire)
	want := missed.Add(6 * time.Hour)
	rec := s.scheduleState["patch"]
	if len(s.queue) != 1 || !rec.Next.Equal(want) || rec.Reason != ScheduleReasonMissedRunOnce || rec.MissedAt == nil || !rec.MissedAt.Equal(missed) {
		t.Fatalf("queue=%d record=%+v want next=%s", len(s.queue), rec, want)
	}
	for i := 1; i <= 10; i++ {
		s.scheduleTick(fire.Add(time.Duration(i) * time.Second))
	}
	if len(s.queue) != 1 {
		t.Fatal("catch-up fired more than once")
	}
}

func TestServeScheduleRuntimeLateness(t *testing.T) {
	initial := scheduleStateTime("2026-10-22T12:17:00Z")
	for _, tc := range []struct {
		name   string
		job    ServeJob
		late   time.Duration
		fire   bool
		reason string
	}{
		{"interval normal tick", ServeJob{Every: time.Hour}, 5 * time.Second, true, ScheduleReasonScheduled},
		{"interval skip late", ServeJob{Every: time.Hour}, 2 * time.Minute, false, ScheduleReasonMissedSkipped},
		{"interval once late", ServeJob{Kind: "drift", Every: time.Hour}, 2 * time.Hour, true, ScheduleReasonMissedRunOnce},
		{"at normal tick", ServeJob{At: []string{"* 04:00"}}, 5 * time.Second, true, ScheduleReasonScheduled},
		{"at skip late", ServeJob{At: []string{"* 04:00"}}, 2 * time.Minute, false, ScheduleReasonMissedSkipped},
		{"at once late", ServeJob{Kind: "drift", Missed: "run_once", At: []string{"* 04:00"}}, 2 * time.Minute, true, ScheduleReasonMissedRunOnce},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := schedulingServer(t, tc.job)
			// Establish a future normal slot rather than a new-job catch-up.
			job := &s.cfg.Jobs[0]
			slot := initial.Add(time.Hour)
			if len(job.At) > 0 {
				slot = job.nextAt(initial)
			}
			data, err := json.Marshal(map[string]any{"version": 1, "jobs": map[string]any{"patch": scheduleRecord{Schedule: job.scheduleSignature(), Next: slot}}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(s.cfg.DataDir, "schedule.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			s.planSchedules(initial)
			now := slot.Add(tc.late)
			s.scheduleTick(now)
			rec := s.scheduleState["patch"]
			if (len(s.queue) == 1) != tc.fire || rec.Reason != tc.reason || !rec.Next.After(now) {
				t.Fatalf("queue=%d record=%+v", len(s.queue), rec)
			}
			if tc.reason == ScheduleReasonMissedSkipped && (rec.MissedAt == nil || !rec.MissedAt.Equal(slot)) {
				t.Fatalf("missed=%v want=%s", rec.MissedAt, slot)
			}
		})
	}
}

func TestServeScheduleAdvance(t *testing.T) {
	now := scheduleStateTime("2026-10-22T12:17:00Z")
	for _, tc := range []struct {
		name         string
		job          ServeJob
		slot         time.Time
		tick         time.Duration
		next         time.Time
		reason       string
		fire, change bool
		missed       *time.Time
	}{
		{name: "before due", job: ServeJob{Every: time.Hour}, slot: now.Add(time.Minute), next: now.Add(time.Minute), reason: ScheduleReasonScheduled},
		{name: "normal interval tick keeps phase", job: ServeJob{Every: time.Hour}, slot: now.Add(-5 * time.Second), next: now.Add(time.Hour - 5*time.Second), reason: ScheduleReasonScheduled, fire: true, change: true},
		{name: "skip minute intervals", job: ServeJob{Every: time.Minute}, slot: now.Add(-30 * 24 * time.Hour), next: now.Add(time.Minute), reason: ScheduleReasonMissedSkipped, change: true},
		{name: "once minute intervals", job: ServeJob{Kind: "drift", Every: time.Minute}, slot: now.Add(-30 * 24 * time.Hour), next: now.Add(time.Minute), reason: ScheduleReasonMissedRunOnce, fire: true, change: true},
		{name: "slow ticker tolerates two ticks", job: ServeJob{Every: time.Hour}, slot: now.Add(-90 * time.Second), tick: time.Minute, next: now.Add(time.Hour - 90*time.Second), reason: ScheduleReasonScheduled, fire: true, change: true},
		{name: "backward interval", job: ServeJob{Every: time.Hour}, slot: now.Add(10 * 24 * time.Hour), next: now.Add(time.Hour), reason: ScheduleReasonScheduled, change: true},
		{name: "backward at", job: ServeJob{At: []string{"fri 04:00"}}, slot: now.Add(10 * 24 * time.Hour), next: scheduleStateTime("2026-10-23T04:00:00Z"), reason: ScheduleReasonScheduled, change: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.job.Name = "example"
			if tc.job.Kind == "" {
				tc.job.Kind = "apply"
			}
			if err := tc.job.prepareSchedule(""); err != nil {
				t.Fatal(err)
			}
			tick := tc.tick
			if tick == 0 {
				tick = 5 * time.Second
			}
			before := scheduleRecord{Schedule: tc.job.scheduleSignature(), Next: tc.slot, Reason: ScheduleReasonScheduled}
			got, fire, changed := advanceSchedule(&tc.job, before, now, tick)
			if fire != tc.fire || changed != tc.change || !got.Next.Equal(tc.next) || got.Reason != tc.reason {
				t.Fatalf("advance=%+v fire=%v changed=%v", got, fire, changed)
			}
			if fire && !got.Last.Equal(tc.slot) {
				t.Fatalf("last=%s want planned slot=%s", got.Last, tc.slot)
			}
			if !fire && !got.Last.IsZero() {
				t.Fatalf("skipped slot marked fired: %s", got.Last)
			}
		})
	}
}

func TestServeScheduleRestartDuringCatchUp(t *testing.T) {
	now := scheduleStateTime("2026-10-22T12:17:00Z")
	for _, restartDelay := range []time.Duration{5 * time.Second, 20 * time.Second} {
		t.Run(restartDelay.String(), func(t *testing.T) {
			s := schedulingServer(t, ServeJob{Kind: "drift", Every: time.Hour})
			due := now.Add(-5*time.Hour - 10*time.Minute)
			data, err := json.Marshal(scheduleFile{Version: 1, Jobs: map[string]*scheduleRecord{"patch": {Schedule: "every=1h0m0s", Next: due}}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(s.cfg.DataDir, "schedule.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			s.planSchedules(now)
			restarted, err := newServer(s.cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			restarted.planSchedules(now.Add(restartDelay))
			missed := restarted.scheduleState["patch"].MissedAt
			if missed == nil || !missed.Equal(due) {
				t.Fatalf("missed=%v want=%v", missed, due)
			}
			restarted.scheduleTick(restarted.nextRun["patch"])
			if len(restarted.queue) != 1 || !restarted.nextRun["patch"].Equal(due.Add(6*time.Hour)) {
				t.Fatalf("queue=%d record=%+v", len(restarted.queue), restarted.scheduleState["patch"])
			}
		})
	}
}

func TestServeScheduleCatchUpBackwardClock(t *testing.T) {
	now := scheduleStateTime("2026-10-22T12:17:00Z")
	for _, every := range []time.Duration{time.Second, time.Hour} {
		j := ServeJob{Every: every, Kind: "drift"}
		if err := j.prepareSchedule(""); err != nil {
			t.Fatal(err)
		}
		following := now.Add(every)
		rec := scheduleRecord{Next: now.Add(10 * time.Second), AfterCatchUp: &following, Reason: ScheduleReasonNew}
		if _, fire, changed := advanceSchedule(&j, rec, now, 5*time.Second); fire || changed {
			t.Fatalf("startup stagger lost for %v", every)
		}
		backward := now.Add(-10 * 24 * time.Hour)
		got, fire, changed := advanceSchedule(&j, rec, backward, 5*time.Second)
		if fire || !changed || !got.Next.Equal(backward.Add(every)) || got.AfterCatchUp != nil {
			t.Fatalf("backward: %+v fire=%v changed=%v", got, fire, changed)
		}
	}
}

func TestServeScheduleRequestUsesSchedulerClock(t *testing.T) {
	s := schedulingServer(t, ServeJob{Kind: "drift", Every: time.Hour, Approval: "required", ApprovalTimeout: time.Hour})
	now := scheduleStateTime("2026-11-22T12:17:00Z")
	s.now = func() time.Time { return now }
	s.planSchedules(s.now())
	now = s.nextRun["patch"]
	s.scheduleTick(s.now())
	s.expirePending(now.Add(time.Second))
	r := s.pending["patch"]
	if r == nil || !r.Requested.Equal(now) || !r.Approval.Expires.Equal(now.Add(time.Hour)) {
		t.Fatalf("pending=%+v", r)
	}
}

func TestServeScheduleStateWriteFailureAndPruning(t *testing.T) {
	s := schedulingServer(t, ServeJob{Every: time.Hour})
	now := scheduleStateTime("2026-10-22T12:17:00Z")
	path := filepath.Join(s.cfg.DataDir, "schedule.json")
	// A directory blocks the temporary file write.
	if err := os.Mkdir(path+".tmp", 0700); err != nil {
		t.Fatal(err)
	}
	s.planSchedules(now)
	if s.storeErr == "" || s.nextRun["patch"].IsZero() {
		t.Fatalf("error=%q state=%+v", s.storeErr, s.scheduleState)
	}
	if err := os.Remove(path + ".tmp"); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.saveSchedule()
	s.mu.Unlock()
	if s.storeErr != "" {
		t.Fatalf("error not cleared: %s", s.storeErr)
	}
	s.cfg.Jobs[0].Every = 0
	s.planSchedules(now)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var state scheduleFile
	if err := json.Unmarshal(data, &state); err != nil || len(state.Jobs) != 0 {
		t.Fatalf("state=%s err=%v", data, err)
	}
}

func TestServeScheduleReviewFixes(t *testing.T) {
	now := scheduleStateTime("2026-10-22T12:17:00Z")
	writeState := func(t *testing.T, dir string, rec *scheduleRecord) {
		t.Helper()
		data, err := json.Marshal(scheduleFile{Version: 1, Jobs: map[string]*scheduleRecord{"patch": rec}})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "schedule.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("staged catch-up dropped under skip", func(t *testing.T) {
		s := schedulingServer(t, ServeJob{Kind: "apply", Every: time.Hour})
		following := now.Add(40 * time.Minute)
		writeState(t, s.cfg.DataDir, &scheduleRecord{Schedule: "every=1h0m0s", Next: now.Add(5 * time.Second), AfterCatchUp: &following, Reason: ScheduleReasonMissedRunOnce})
		s.planSchedules(now)
		s.scheduleTick(now.Add(10 * time.Second))
		if len(s.queue) != 0 || !s.nextRun["patch"].Equal(following) {
			t.Fatalf("queue=%d record=%+v", len(s.queue), s.scheduleState["patch"])
		}
	})
	t.Run("zero after_catch_up is unreadable", func(t *testing.T) {
		s := schedulingServer(t, ServeJob{Kind: "apply", Every: time.Hour})
		var zero time.Time
		writeState(t, s.cfg.DataDir, &scheduleRecord{Schedule: "every=1h0m0s", Next: now.Add(-time.Hour), AfterCatchUp: &zero})
		s.planSchedules(now)
		if s.scheduleState["patch"].Reason != ScheduleReasonStateUnreadable {
			t.Fatalf("record=%+v", s.scheduleState["patch"])
		}
	})
	t.Run("schedule write keeps a result write error", func(t *testing.T) {
		s := schedulingServer(t, ServeJob{Every: time.Hour})
		s.storeErr = "results: disk full"
		s.planSchedules(now)
		if s.storeErr != "results: disk full" {
			t.Fatalf("error=%q", s.storeErr)
		}
	})
	t.Run("accepted but unsaved run is not skipped", func(t *testing.T) {
		s := schedulingServer(t, ServeJob{Every: time.Hour})
		s.now = func() time.Time { return now }
		s.planSchedules(now)
		if err := os.RemoveAll(filepath.Join(s.cfg.DataDir, "results")); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(s.cfg.DataDir, "results"), nil, 0600); err != nil {
			t.Fatal(err)
		}
		s.scheduleTick(s.nextRun["patch"])
		got := s.scheduleState["patch"].Outcome
		if len(s.queue) != 1 || !strings.HasPrefix(got, "requested ") || !strings.Contains(got, "not saved") {
			t.Fatalf("queue=%d outcome=%q", len(s.queue), got)
		}
	})
}
