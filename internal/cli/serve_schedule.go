package cli

import (
	"fmt"
	"strings"
	"time"
)

type serveWindow struct {
	days       [7]bool
	start, end int
}

func parseServeWindows(specs []string, zone string) ([]serveWindow, *time.Location, error) {
	if zone == "" {
		zone = "UTC"
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return nil, nil, fmt.Errorf("windows_tz: %w", err)
	}
	var windows []serveWindow
	for _, spec := range specs {
		fields := strings.Fields(spec)
		if len(fields) != 2 {
			return nil, nil, fmt.Errorf("windows: invalid %q (expected <days> HH:MM-HH:MM)", spec)
		}
		times := strings.Split(fields[1], "-")
		if len(times) != 2 {
			return nil, nil, fmt.Errorf("windows: invalid time range %q", spec)
		}
		var w serveWindow
		w.start, err = parseServeClock(times[0])
		if err != nil {
			return nil, nil, fmt.Errorf("windows: %w", err)
		}
		w.end, err = parseServeClock(times[1])
		if err != nil {
			return nil, nil, fmt.Errorf("windows: %w", err)
		}
		if w.start == w.end {
			return nil, nil, fmt.Errorf("windows: start and end must differ in %q", spec)
		}
		w.days, err = parseServeDays(fields[0])
		if err != nil {
			return nil, nil, fmt.Errorf("windows: %w", err)
		}
		windows = append(windows, w)
	}
	return windows, loc, nil
}

func windowContains(w serveWindow, t time.Time) bool {
	minute := t.Hour()*60 + t.Minute()
	day := int(t.Weekday())
	if w.end > w.start {
		return w.days[day] && minute >= w.start && minute < w.end
	}
	return (w.days[day] && minute >= w.start) || (w.days[(day+6)%7] && minute < w.end)
}

func (j *ServeJob) windowOpen(now time.Time) bool {
	for _, w := range j.windows {
		if windowContains(w, now.In(j.windowsLoc)) {
			return true
		}
	}
	return false
}

// nextWindow scans real minutes so missing and repeated local times follow the
// zone's clock; two weeks plus a day covers a weekly window a DST change skipped.
// Without a start in that span it answers a day later: the run is checked again
// then, never let through outside its windows
func (j *ServeJob) nextWindow(now time.Time) time.Time {
	t := now.Truncate(time.Minute).Add(time.Minute)
	for end := t.Add(15 * 24 * time.Hour); t.Before(end); t = t.Add(time.Minute) {
		for _, w := range j.windows {
			if windowContains(w, t.In(j.windowsLoc)) && !windowContains(w, t.Add(-time.Minute).In(j.windowsLoc)) {
				return t.UTC()
			}
		}
	}
	return now.Add(24 * time.Hour).UTC()
}

func (j *ServeJob) gated(mode string) bool {
	return j.Kind == "apply" && mode != "check" && len(j.windows) > 0
}

// deferUntil combines a caller's lower bound with the maintenance windows.
func (j *ServeJob) deferUntil(r *JobResult, now time.Time) *time.Time {
	target := now
	if r.NotBefore != nil && r.NotBefore.After(target) {
		target = *r.NotBefore
	}
	if j.gated(r.Mode) && !j.windowOpen(target) {
		target = j.nextWindow(target)
	}
	if target.After(now) {
		return &target
	}
	return nil
}

// scheduledRun includes a pending run that reserved a future request; mu is held.
func (s *server) scheduledRun(job string) *JobResult {
	for _, r := range s.results[job] {
		if r.Status == "scheduled" || (r.Status == "pending" && r.NotBefore != nil) {
			return r
		}
	}
	return nil
}

// schedule records a deferral; the caller manages the job slot and holds mu.
func (s *server) schedule(r *JobResult, at time.Time) {
	r.Status = "scheduled"
	r.NotBefore = &at
	r.Summary = "scheduled for " + at.Format(time.RFC3339)
}

func (s *server) releaseScheduled(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopping != nil { // they stay scheduled for the next start
		return
	}
	for i := range s.cfg.Jobs {
		j := &s.cfg.Jobs[i]
		list := s.results[j.Name]
		for i := len(list) - 1; i >= 0; i-- { // oldest first: deferrals start in request order
			r := list[i]
			if r.Status != "scheduled" || (r.NotBefore != nil && now.Before(*r.NotBefore)) {
				continue
			}
			if s.running[j.Name] {
				if r.Summary != "waiting for the job to be free" {
					r.Summary = "waiting for the job to be free"
					s.saveResult(r)
				}
				continue
			}
			if at := j.deferUntil(r, now); at != nil {
				s.schedule(r, *at)
				s.saveResult(r)
				continue
			}
			select {
			case s.queue <- queued{job: j, run: r}:
				r.Status, r.Summary = "queued", ""
				s.running[j.Name] = true
				s.saveResult(r)
			default:
			}
		}
	}
}

func (s *server) scheduledCount() int {
	count := 0
	for _, list := range s.results {
		for _, r := range list {
			if r.Status == "scheduled" {
				count++
			}
		}
	}
	return count
}
