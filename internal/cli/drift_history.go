package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Drift history: every drift check is kept in ~/.onigirazu/drift-history, so
// a report can say since when a task has drifted on a host, and --history
// lists past checks of a playbook.

const keepDriftReports = 500

func driftHistoryDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".onigirazu", "drift-history"), nil
}

// historyEntry is a stored report with the playbook it checked
type historyEntry struct {
	PlaybookPath string `json:"playbook_path"`
	DriftReport
}

// loadDriftHistory returns the stored checks of a playbook, oldest first
func loadDriftHistory(dir, playbookPath string) ([]historyEntry, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	var out []historyEntry
	for _, f := range files {
		data, err := os.ReadFile(f) // #nosec G304 -- our own history directory
		if err != nil {
			continue
		}
		var e historyEntry
		if json.Unmarshal(data, &e) == nil && e.PlaybookPath == playbookPath {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CheckedAt.Before(out[j].CheckedAt) })
	return out, nil
}

// annotateSince sets, for every drifting task, the time of the earliest check
// in the unbroken run of checks (up to this one) where it drifted too. A
// check that did not look at the host (another --limit) does not break the run.
func annotateSince(r *DriftReport, history []historyEntry) {
	for host, items := range r.Drift {
		for i := range items {
			since := r.CheckedAt
			for k := len(history) - 1; k >= 0; k-- {
				h := history[k]
				if !checkedHost(&h.DriftReport, host) {
					continue
				}
				if !hasDriftItem(&h.DriftReport, host, items[i].Task) {
					break
				}
				since = h.CheckedAt
			}
			t := since
			items[i].Since = &t
		}
	}
}

func checkedHost(r *DriftReport, host string) bool {
	if _, ok := r.Drift[host]; ok {
		return true
	}
	if _, ok := r.Errors[host]; ok {
		return true
	}
	for _, h := range r.InSync {
		if h == host {
			return true
		}
	}
	return false
}

func hasDriftItem(r *DriftReport, host, task string) bool {
	for _, it := range r.Drift[host] {
		if it.Task == task {
			return true
		}
	}
	return false
}

// saveDriftHistory stores a check and keeps the newest keepDriftReports
func saveDriftHistory(dir, playbookPath string, r *DriftReport) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(historyEntry{PlaybookPath: playbookPath, DriftReport: *r})
	if err != nil {
		return err
	}
	name := filepath.Join(dir, fmt.Sprintf("%d.json", r.CheckedAt.UnixNano()))
	if err := os.WriteFile(name, data, 0o600); err != nil {
		return err
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	sort.Strings(files) // names are nanosecond timestamps of equal length
	for len(files) > keepDriftReports {
		_ = os.Remove(files[0])
		files = files[1:]
	}
	return nil
}

// writeDriftHistory lists past checks of a playbook, newest first
func writeDriftHistory(w io.Writer, playbook string, history []historyEntry) {
	if len(history) == 0 {
		fmt.Fprintf(w, "No drift checks of %s yet\n", playbook)
		return
	}
	fmt.Fprintf(w, "%-20s  %5s  %7s  %5s  %6s  %s\n", "CHECKED", "HOSTS", "DRIFTED", "TASKS", "ERRORS", "")
	for i := len(history) - 1; i >= 0; i-- {
		h := history[i]
		var hosts []string
		for name := range h.Drift {
			hosts = append(hosts, name)
		}
		sort.Strings(hosts)
		note := strings.Join(hosts, ", ")
		if h.Fixed {
			note += " (fixed)"
		}
		fmt.Fprintf(w, "%-20s  %5d  %7d  %5d  %6d  %s\n", h.CheckedAt.Local().Format("2006-01-02 15:04:05"),
			h.Hosts, len(h.Drift), h.DriftTasks, len(h.Errors), note)
	}
}

// sinceText is how long ago t was, for the text report
func sinceText(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "new"
	case d < time.Hour:
		return fmt.Sprintf("since %dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("since %dh", int(d.Hours()))
	}
	return "since " + t.Local().Format("2006-01-02")
}
