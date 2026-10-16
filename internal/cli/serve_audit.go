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
	"strings"
	"time"
)

// serve_audit: the action log — every run, approve, reject, pause and resume
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

// actions is the tail of the log, newest first: the file is read backwards
// in chunks until enough lines are in hand, so a log that grew for years
// costs the same as a young one
func (s *server) actions(limit int) []serveAction {
	s.auditMu.Lock()
	defer s.auditMu.Unlock()
	out := make([]serveAction, 0)
	if limit <= 0 {
		return out
	}
	f, err := os.Open(filepath.Join(s.cfg.DataDir, "audit.log")) // #nosec G304 -- our own data directory
	if err != nil {
		return out
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return out
	}
	const chunk = 64 * 1024
	end := info.Size()
	var data []byte
	for end > 0 {
		start := max(end-chunk, 0)
		buf := make([]byte, end-start)
		if _, err := f.ReadAt(buf, start); err != nil && !errors.Is(err, io.EOF) {
			return out
		}
		data = append(buf, data...)
		end = start
		complete := bytes.Count(data, []byte{'\n'})
		if end > 0 {
			complete-- // the first line may be cut
		}
		if complete >= limit {
			break
		}
	}
	if end > 0 { // drop the cut first line
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[i+1:]
		}
	}
	lines := bytes.Split(data, []byte{'\n'})
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		var a serveAction
		if json.Unmarshal(lines[i], &a) == nil {
			out = append(out, a)
		}
	}
	return out
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
	if len(parts) == 4 {
		switch parts[1] {
		case "jobs":
			a.Job = parts[2]
		case "runs":
			a.Run = parts[2]
			s.mu.Lock()
			for job, run := range s.pending {
				if run.ID == a.Run {
					a.Job = job
					break
				}
			}
			s.mu.Unlock()
		}
	}
	return a
}
