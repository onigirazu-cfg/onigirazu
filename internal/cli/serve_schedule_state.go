package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ScheduleReasonScheduled       = "scheduled"
	ScheduleReasonMissedSkipped   = "missed_skipped"
	ScheduleReasonMissedRunOnce   = "missed_run_once"
	ScheduleReasonStateUnreadable = "state_unreadable"
	ScheduleReasonNew             = "new"
)

type scheduleRecord struct {
	Schedule string     `json:"schedule"`
	Last     time.Time  `json:"last,omitzero"`
	Next     time.Time  `json:"next"`
	Reason   string     `json:"reason"`
	MissedAt *time.Time `json:"missed_at,omitempty"`
	Outcome  string     `json:"outcome,omitempty"`
	// Keep the original phase even if serve restarts during the catch-up stagger.
	AfterCatchUp *time.Time `json:"after_catch_up,omitempty"`
}

type scheduleFile struct {
	Version int                        `json:"version"`
	Jobs    map[string]*scheduleRecord `json:"jobs"`
}

func copyScheduleTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	copy := *t
	return &copy
}

func (j *ServeJob) firstSlotAfter(now time.Time) time.Time {
	if j.Every > 0 {
		return now.Add(j.Every)
	}
	return j.nextAt(now)
}

// nextInterval skips whole intervals in one step, keeping the original phase.
func nextInterval(slot time.Time, every time.Duration, now time.Time) time.Time {
	if slot.After(now) {
		return slot
	}
	elapsed := now.Sub(slot)
	// Add the remainder to now, avoiding an overflowing interval product.
	return now.Add(every - elapsed%every)
}

func (j *ServeJob) followingSlot(due, now time.Time) time.Time {
	if j.Every > 0 {
		return nextInterval(due, j.Every, now)
	}
	return j.nextAt(now)
}

func planStart(j *ServeJob, rec *scheduleRecord, history *time.Time, unreadable bool, now time.Time) (next time.Time, reason string, missedAt *time.Time, fireNow bool) {
	if unreadable {
		return j.firstSlotAfter(now), ScheduleReasonStateUnreadable, nil, false
	}
	var last, due time.Time
	if rec != nil {
		last = rec.Last
	}
	if last.IsZero() && history != nil {
		last = *history
	}
	if rec != nil && rec.Schedule == j.scheduleSignature() && !rec.Next.IsZero() {
		due = rec.Next
	} else if !last.IsZero() {
		if j.Every > 0 {
			due = last.Add(j.Every)
		} else {
			due = j.nextAt(last)
		}
	}
	if due.IsZero() {
		return j.firstSlotAfter(now), ScheduleReasonNew, nil, j.missed == "run_once"
	}
	if due.After(now) {
		bound := j.firstSlotAfter(now)
		if due.After(bound) {
			due = bound
		}
		return due, ScheduleReasonScheduled, nil, false
	}
	next = j.followingSlot(due, now)
	missedAt = &due
	if rec != nil && rec.Schedule == j.scheduleSignature() && rec.AfterCatchUp != nil {
		next = j.followingSlot(*rec.AfterCatchUp, now)
		if rec.MissedAt != nil {
			missedAt = copyScheduleTime(rec.MissedAt)
		}
	}
	if j.missed == "skip" {
		return next, ScheduleReasonMissedSkipped, missedAt, false
	}
	return next, ScheduleReasonMissedRunOnce, missedAt, true
}

// readSchedule keeps damaged state as evidence and never falls back to history.
func (s *server) readSchedule(now time.Time) (map[string]*scheduleRecord, bool) {
	path := filepath.Join(s.cfg.DataDir, "schedule.json")
	data, err := os.ReadFile(path) // #nosec G304 -- our own data directory
	if os.IsNotExist(err) {
		return nil, false
	}
	var state scheduleFile
	if err == nil {
		err = json.Unmarshal(data, &state)
		if err == nil && (state.Version != 1 || state.Jobs == nil) {
			err = errors.New("invalid schedule state version or jobs")
		}
		if err == nil {
			for name, rec := range state.Jobs {
				if rec == nil || rec.Next.IsZero() || rec.Schedule == "" || (rec.AfterCatchUp != nil && rec.AfterCatchUp.IsZero()) {
					err = fmt.Errorf("invalid schedule record for %s", name)
					break
				}
			}
		}
	}
	if err == nil {
		return state.Jobs, false
	}
	fmt.Fprintf(os.Stderr, "serve: %s unreadable: %v\n", path, err)
	corrupt := path + ".corrupt-" + now.UTC().Format("20060102T150405.000000000Z")
	if renameErr := os.Rename(path, corrupt); renameErr != nil {
		fmt.Fprintf(os.Stderr, "serve: preserve %s: %v\n", path, renameErr)
	}
	return nil, true
}

// planSchedules runs before the ticker starts; all plan state is protected by mu.
func (s *server) planSchedules(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, unreadable := s.readSchedule(now)
	s.scheduleState = make(map[string]*scheduleRecord, len(s.cfg.Jobs))
	s.nextRun = make(map[string]time.Time, len(s.cfg.Jobs))
	for i := range s.cfg.Jobs {
		j := &s.cfg.Jobs[i]
		if !j.hasSchedule() {
			continue
		}
		var history *time.Time
		for _, result := range s.results[j.Name] {
			if result.Trigger == "schedule" && (history == nil || result.Requested.After(*history)) {
				requested := result.Requested
				history = &requested
			}
		}
		old := records[j.Name]
		if old != nil && old.AfterCatchUp != nil && j.missed == "skip" {
			// a catch-up staged under run_once is dropped once the policy is skip
			dropped := *old
			dropped.Next, dropped.AfterCatchUp = *old.AfterCatchUp, nil
			old = &dropped
		}
		next, reason, missedAt, fire := planStart(j, old, history, unreadable, now)
		rec := &scheduleRecord{Schedule: j.scheduleSignature(), Next: next, Reason: reason, MissedAt: missedAt}
		if old != nil {
			rec.Last, rec.Outcome = old.Last, old.Outcome
		}
		if fire {
			// A catch-up consumes one staggered request, then resumes the original phase.
			rec.AfterCatchUp = copyScheduleTime(&next)
			rec.Next = now.Add(time.Duration(10+i*5) * time.Second)
		} else if old != nil && old.Schedule == rec.Schedule && old.Next.Equal(next) && old.AfterCatchUp != nil {
			rec.AfterCatchUp = copyScheduleTime(old.AfterCatchUp)
			rec.Reason, rec.MissedAt = old.Reason, copyScheduleTime(old.MissedAt)
		}
		s.scheduleState[j.Name] = rec
		s.nextRun[j.Name] = rec.Next
	}
	s.saveSchedule()
}

// saveSchedule is called with mu held, like saveResult.
func (s *server) saveSchedule() {
	data, err := json.Marshal(scheduleFile{Version: 1, Jobs: s.scheduleState})
	if err == nil {
		path := filepath.Join(s.cfg.DataDir, "schedule.json")
		tmp := path + ".tmp"
		err = os.WriteFile(tmp, data, 0o600)
		if err == nil {
			err = os.Rename(tmp, path)
		}
		if err != nil {
			err = errors.Join(err, removeResultFile(tmp))
		}
	}
	// a schedule write clears only its own error, never a result write error
	if err != nil {
		s.storageError(fmt.Errorf("schedule: %w", err))
		fmt.Fprintf(s.out, "serve: schedule not saved: %v\n", err)
	} else if strings.HasPrefix(s.storeErr, "schedule: ") {
		s.storageError(nil)
	}
}

func (s *server) scheduleTickDuration() time.Duration {
	if s.schedulerTick > 0 {
		return s.schedulerTick
	}
	return 5 * time.Second
}

// advanceSchedule is pure; a firing consumes its planned slot even on rejection.
func advanceSchedule(j *ServeJob, rec scheduleRecord, now time.Time, tick time.Duration) (scheduleRecord, bool, bool) {
	bound := j.firstSlotAfter(now)
	if rec.Next.After(bound) && (rec.AfterCatchUp == nil || bound.Before(*rec.AfterCatchUp)) {
		rec.Next, rec.Reason, rec.MissedAt = bound, ScheduleReasonScheduled, nil
		rec.AfterCatchUp = nil
		return rec, false, true
	}
	if now.Before(rec.Next) {
		return rec, false, false
	}
	slot := rec.Next
	lateLimit := time.Minute
	if j.Every > 0 && 2*tick > lateLimit {
		lateLimit = 2 * tick
	}
	if rec.AfterCatchUp == nil && j.missed == "skip" && now.Sub(slot) > lateLimit {
		rec.Next, rec.Reason, rec.MissedAt = j.followingSlot(slot, now), ScheduleReasonMissedSkipped, &slot
		return rec, false, true
	}
	rec.Last = slot
	if rec.AfterCatchUp != nil {
		rec.Next = j.followingSlot(*rec.AfterCatchUp, now)
		rec.AfterCatchUp = nil
		if rec.Reason == ScheduleReasonNew {
			rec.Reason, rec.MissedAt = ScheduleReasonScheduled, nil
		}
		return rec, true, true
	}
	rec.Reason, rec.MissedAt = ScheduleReasonScheduled, nil
	if j.Every > 0 {
		rec.Next = slot.Add(j.Every)
		if !rec.Next.After(now) {
			missed := rec.Next
			rec.Next = j.followingSlot(missed, now)
			rec.MissedAt = &missed
			if j.missed == "skip" {
				rec.Reason = ScheduleReasonMissedSkipped
			} else {
				rec.Reason = ScheduleReasonMissedRunOnce
			}
		}
	} else {
		rec.Next = j.nextAt(now)
		if now.Sub(slot) > time.Minute {
			rec.Reason, rec.MissedAt = ScheduleReasonMissedRunOnce, &slot
		}
	}
	return rec, true, true
}

func (s *server) scheduleTick(now time.Time) {
	s.mu.Lock()
	var due []*ServeJob
	changed := false
	for i := range s.cfg.Jobs {
		j := &s.cfg.Jobs[i]
		rec := s.scheduleState[j.Name]
		if rec == nil {
			continue
		}
		updated, fire, change := advanceSchedule(j, *rec, now, s.scheduleTickDuration())
		if !change {
			continue
		}
		*rec = updated
		s.nextRun[j.Name] = rec.Next
		changed = true
		if fire {
			due = append(due, j)
		}
	}
	if changed {
		s.saveSchedule()
	}
	s.mu.Unlock()
	for _, j := range due {
		result, err := s.request(j, "schedule", nil)
		s.mu.Lock()
		rec := s.scheduleState[j.Name]
		switch {
		case result == nil && err != nil:
			rec.Outcome = "skipped: " + err.Error()
		case result != nil:
			rec.Outcome = "requested " + result.ID
			if result.Status == "pending" || result.Status == "scheduled" {
				rec.Outcome += " (" + result.Status + ")"
			}
			if err != nil { // accepted, but its record was not saved
				rec.Outcome += " (not saved: " + err.Error() + ")"
			}
		}
		s.saveSchedule()
		s.mu.Unlock()
	}
}
