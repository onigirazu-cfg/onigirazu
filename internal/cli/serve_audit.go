package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// serve_audit: the action log — every run, approve, reject, cancel, pause and resume
// and every refused attempt, one JSON line each in data_dir/audit.log

// serveAction is one entry of the action log
type serveAction struct {
	Time   time.Time `json:"time"`
	User   string    `json:"user"`
	Action string    `json:"action"`
	Job    string    `json:"job,omitempty"`
	Run    string    `json:"run,omitempty"`
	OK     bool      `json:"ok"`
	Reason string    `json:"reason,omitempty"`
	Addr   string    `json:"addr,omitempty"`
}

// logAction appends an entry; refused attempts are also printed
func (s *server) logAction(a serveAction) {
	s.auditMu.Lock()
	defer s.auditMu.Unlock()
	if a.Time.IsZero() {
		a.Time = time.Now().UTC()
	}
	if !a.OK {
		s.mu.Lock()
		fmt.Fprintf(s.out, "serve: audit: %s %s %s refused: %s\n", a.User, a.Action, a.Job, a.Reason)
		s.mu.Unlock()
	}
	f, err := os.OpenFile(filepath.Join(s.cfg.DataDir, "audit.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "serve: audit: %v\n", err)
		return
	}
	if err := json.NewEncoder(f).Encode(a); err != nil {
		fmt.Fprintf(os.Stderr, "serve: audit: %v\n", err)
	}
	if err := f.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "serve: audit: %v\n", err)
	}
}

type serveActionFilter struct {
	User, Action, Job, Run string
	OK                     *bool
	From, To               time.Time
	Limit                  int
	Before                 *int64
}

// actions scans complete lines backwards, retaining only matching entries.
func (s *server) actions(filter serveActionFilter) ([]serveAction, string) {
	s.auditMu.Lock()
	defer s.auditMu.Unlock()
	out := make([]serveAction, 0)
	if filter.Limit <= 0 {
		return out, ""
	}
	f, err := os.Open(filepath.Join(s.cfg.DataDir, "audit.log")) // #nosec G304 -- our own data directory
	if err != nil {
		return out, ""
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return out, ""
	}
	end := info.Size()
	if filter.Before != nil {
		end = min(end, *filter.Before)
	}
	const chunk = 64 * 1024
	var carry []byte
	var oldest int64
	for end > 0 {
		start := max(end-chunk, 0)
		buf := make([]byte, end-start)
		if _, err := f.ReadAt(buf, start); err != nil && !errors.Is(err, io.EOF) {
			return out, ""
		}
		buf = append(buf, carry...)
		data := buf
		lineEnd := len(data)
		if lineEnd > 0 && data[lineEnd-1] == '\n' {
			lineEnd--
		}
		for lineEnd > 0 {
			i := bytes.LastIndexByte(data[:lineEnd], '\n')
			if i < 0 && start > 0 {
				break
			}
			var a serveAction
			if json.Unmarshal(data[i+1:lineEnd], &a) == nil {
				if !filter.From.IsZero() && a.Time.Before(filter.From) {
					return out, ""
				}
				if (filter.User == "" || a.User == filter.User) &&
					(filter.Action == "" || a.Action == filter.Action) &&
					(filter.Job == "" || a.Job == filter.Job) &&
					(filter.Run == "" || a.Run == filter.Run) &&
					(filter.OK == nil || a.OK == *filter.OK) &&
					(filter.To.IsZero() || !a.Time.After(filter.To)) {
					if len(out) == filter.Limit {
						return out, strconv.FormatInt(oldest, 10)
					}
					out = append(out, a)
					oldest = start + int64(i+1)
				}
			}
			lineEnd = i
		}
		// Keep the cut first line for the next chunk, including its delimiter.
		carry = data[:min(max(lineEnd+1, 0), len(data))]
		end = start
	}
	return out, ""
}

// actionCursor accepts decimal offsets at line boundaries.
func (s *server) actionCursor(value string) (int64, error) {
	before, err := strconv.ParseInt(value, 10, 64)
	invalid := errors.New("invalid before cursor")
	if err != nil || before < 0 || strconv.FormatInt(before, 10) != value {
		return 0, invalid
	}
	if before == 0 {
		return before, nil
	}
	f, err := os.Open(filepath.Join(s.cfg.DataDir, "audit.log")) // #nosec G304 -- our own data directory
	if err != nil {
		return 0, invalid
	}
	defer f.Close()
	var previous [1]byte
	if _, err := f.ReadAt(previous[:], before-1); err != nil || previous[0] != '\n' {
		return 0, invalid
	}
	return before, nil
}

// servePageOptions shares the default page size and RFC 3339 date validation.
func servePageOptions(r *http.Request) (limit int, from, to time.Time, err error) {
	q := r.URL.Query()
	limit, _ = strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	limit = min(limit, 1000)
	for _, field := range []struct {
		name   string
		target *time.Time
	}{{"from", &from}, {"to", &to}} {
		if q.Has(field.name) {
			*field.target, err = time.Parse(time.RFC3339, q.Get(field.name))
			if err != nil {
				return limit, from, to, fmt.Errorf("invalid %s: expected RFC 3339", field.name)
			}
		}
	}
	return
}

// requestAction is the entry a request describes: who, what, which job or run
func (s *server) requestAction(r *http.Request, id serveIdentity) serveAction {
	addr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		addr = r.RemoteAddr
	}
	a := serveAction{User: id.User, Addr: addr}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	a.Action = parts[len(parts)-1]
	if a.Action == "rerun-failed" {
		a.Action = "rerun_failed"
	}
	if r.URL.Path == "/api/inventory/refresh" {
		a.Action = "inventory_refresh"
	}
	if len(parts) == 4 {
		switch parts[1] {
		case "jobs":
			a.Job = parts[2]
		case "runs":
			a.Run = parts[2]
			s.mu.Lock()
			for job := range s.results {
				if run := s.resultByID(job, a.Run); run != nil && (a.Action == "cancel" || a.Action == "rerun_failed" || run.Status == "pending") {
					a.Job = job
					break
				}
			}
			s.mu.Unlock()
		}
	}
	return a
}
