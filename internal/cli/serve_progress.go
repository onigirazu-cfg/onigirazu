package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/engine"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type runObserverKey struct{}

func withRunObserver(ctx context.Context, obs engine.ExecutionObserverI) context.Context {
	return context.WithValue(ctx, runObserverKey{}, obs)
}

const progressLimit = 1000

type progressEvent struct {
	Seq        uint64    `json:"seq"`
	At         time.Time `json:"at"`
	Type       string    `json:"type"`
	Planned    int       `json:"planned,omitempty"`
	Items      int       `json:"items,omitempty"`
	Hosts      []string  `json:"hosts,omitempty"`
	Play       string    `json:"play,omitempty"`
	Task       string    `json:"task,omitempty"`
	Host       string    `json:"host,omitempty"`
	Status     string    `json:"status,omitempty"`
	DurationMS int64     `json:"duration_ms,omitempty"`
}

type hostProgress struct {
	Planned       int    `json:"planned"`
	Task          string `json:"task"`
	Done          bool   `json:"done"`
	OK            int    `json:"ok"`
	Changed       int    `json:"changed"`
	Failed        int    `json:"failed"`
	Skipped       int    `json:"skipped"`
	TasksFinished int    `json:"tasks_finished"`
	// a running loop: its size and the items started; the loop is one task
	// with one result, the worst of its items
	Items      int  `json:"items,omitempty"`
	Item       int  `json:"item,omitempty"`
	Loop       bool `json:"loop,omitempty"`
	loopStatus string
}

// loopRank orders item results: a loop reports its worst one
var loopRank = map[string]int{"": 0, "skipped": 1, "ok": 2, "ignored": 2, "changed": 3, "failed": 4}

func (h *hostProgress) count(status string) {
	h.TasksFinished++
	switch status {
	case "ok", "ignored":
		h.OK++
	case "changed":
		h.Changed++
	case "failed":
		h.Failed++
	case "skipped":
		h.Skipped++
	}
}

type progressSnapshot struct {
	Type          string                  `json:"type"`
	Seq           uint64                  `json:"seq"`
	Hosts         map[string]hostProgress `json:"hosts"`
	TasksFinished int                     `json:"tasks_finished"`
	TasksPlanned  int                     `json:"tasks_planned"`
	Status        string                  `json:"status,omitempty"`
}

type runProgress struct {
	mu          sync.Mutex
	mask        *strings.Replacer
	play        string
	seq         uint64
	events      [progressLimit]progressEvent
	size        uint64
	hosts       map[string]hostProgress
	planned     int
	finished    int
	status      string
	subscribers map[chan progressEvent]struct{}
}

func newRunProgress(mask *strings.Replacer) *runProgress {
	if mask == nil {
		mask = strings.NewReplacer()
	}
	return &runProgress{mask: mask, hosts: make(map[string]hostProgress), subscribers: make(map[chan progressEvent]struct{})}
}

// emit holds only the progress lock; slow readers lose their subscription and replay.
func (p *runProgress) emit(e progressEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.status != "" {
		return
	}
	if e.Type == "play_start" {
		p.play = p.mask.Replace(e.Play)
	}
	if e.Type == "plan" {
		e.Play = p.mask.Replace(e.Play)
		hosts := make([]string, len(e.Hosts))
		for i, host := range e.Hosts {
			hosts[i] = p.mask.Replace(host)
			h := p.hosts[hosts[i]]
			h.Planned += e.Planned
			p.hosts[hosts[i]] = h
			p.planned += e.Planned
		}
		e.Hosts = hosts
	} else {
		e.Play = p.play
	}
	e.Task, e.Host = p.mask.Replace(e.Task), p.mask.Replace(e.Host)
	p.seq++
	e.Seq, e.At = p.seq, time.Now().UTC()
	if e.Host != "" {
		h := p.hosts[e.Host]
		switch e.Type {
		case "loop_start":
			h.Task, h.Done, h.Loop, h.loopStatus, h.Items, h.Item = e.Task, false, true, "", e.Items, 0
		case "loop_end":
			if h.loopStatus == "" {
				h.loopStatus = "skipped" // no items ran
			}
			e.Status = h.loopStatus
			h.count(e.Status)
			p.finished++
			h.Task, h.Done, h.Loop, h.loopStatus, h.Items, h.Item = "", true, false, "", 0, 0
		case "task_start":
			if h.Loop {
				h.Item++
			} else {
				h.Task, h.Done = e.Task, false
			}
		case "task_end":
			if h.Loop {
				if loopRank[e.Status] > loopRank[h.loopStatus] {
					h.loopStatus = e.Status
				}
				break
			}
			h.Task, h.Done = "", true
			h.count(e.Status)
			p.finished++
		}
		p.hosts[e.Host] = h
	}
	if e.Type == "end" {
		p.status = e.Status
		for host, h := range p.hosts {
			h.Task, h.Done, h.Loop, h.Items, h.Item = "", true, false, 0, 0
			p.hosts[host] = h
		}
	}
	p.events[(p.seq-1)%progressLimit] = e
	if p.size < progressLimit {
		p.size++
	}
	for ch := range p.subscribers {
		select {
		case ch <- e:
			if e.Type == "end" {
				close(ch)
				delete(p.subscribers, ch)
			}
		default:
			close(ch)
			delete(p.subscribers, ch)
		}
	}
}

func (p *runProgress) OnExecutionStart(string, int, int) {}
func (p *runProgress) OnPlayStart(name string, _, _ int) {
	p.emit(progressEvent{Type: "play_start", Play: name})
}
func (p *runProgress) OnPlayPlan(play string, hosts []string, tasks int) {
	p.emit(progressEvent{Type: "plan", Play: play, Hosts: hosts, Planned: tasks})
}
func (p *runProgress) OnPlanAdjust(task, host string, delta int) {
	p.emit(progressEvent{Type: "plan", Task: task, Hosts: []string{host}, Planned: delta})
}
func (p *runProgress) OnLoopStart(task, host string, items int) {
	p.emit(progressEvent{Type: "loop_start", Task: task, Host: host, Items: items})
}
func (p *runProgress) OnLoopEnd(task, host string) {
	p.emit(progressEvent{Type: "loop_end", Task: task, Host: host})
}
func (p *runProgress) OnPlayEnd(_ string, _ int, _ bool, duration time.Duration) {
	p.emit(progressEvent{Type: "play_end", DurationMS: duration.Milliseconds()})
}
func (p *runProgress) OnTaskStart(task, host string) {
	p.emit(progressEvent{Type: "task_start", Task: task, Host: host})
}
func (p *runProgress) OnTaskEnd(r *types.TaskResult) {
	status := "ok"
	switch {
	case r.Ignored:
		status = "ignored"
	case r.Failed:
		status = "failed"
	case r.Skipped:
		status = "skipped"
	case r.Changed:
		status = "changed"
	}
	p.emit(progressEvent{Type: "task_end", Task: r.TaskName, Host: r.Host, Status: status, DurationMS: r.Duration.Milliseconds()})
}
func (p *runProgress) OnExecutionEnd(*types.PlaybookResult, time.Duration) {}
func (p *runProgress) OnError(string, string, string)                      {}
func (p *runProgress) finish(status string)                                { p.emit(progressEvent{Type: "end", Status: status}) }

// subscribe takes the snapshot, replay and subscription under one lock.
func (p *runProgress) subscribe(after uint64) (progressSnapshot, []progressEvent, bool, chan progressEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	snapshot := progressSnapshot{Type: "snapshot", Seq: p.seq, Hosts: make(map[string]hostProgress, len(p.hosts)), TasksFinished: p.finished, TasksPlanned: p.planned, Status: p.status}
	for host, h := range p.hosts {
		snapshot.Hosts[host] = h
	}
	first := p.seq - p.size + 1
	gap := after < first-1
	events := make([]progressEvent, 0, p.size)
	for seq := first; seq <= p.seq; seq++ {
		if seq > after {
			events = append(events, p.events[(seq-1)%progressLimit])
		}
	}
	ch := make(chan progressEvent, 64)
	if p.status == "" {
		p.subscribers[ch] = struct{}{}
	} else {
		close(ch)
	}
	return snapshot, events, gap, ch
}

func (p *runProgress) unsubscribe(ch chan progressEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.subscribers[ch]; ok {
		delete(p.subscribers, ch)
		close(ch)
	}
}

// retainProgress is called with the server lock after a run settles.
func (s *server) retainProgress(id, status string) {
	s.progress[id].finish(status)
	s.finishedProgress = append(s.finishedProgress, id)
	if len(s.finishedProgress) > 20 {
		delete(s.progress, s.finishedProgress[0])
		s.finishedProgress = s.finishedProgress[1:]
	}
}

func (s *server) progressHandler(w http.ResponseWriter, r *http.Request, id string) {
	after := uint64(0)
	if value := r.URL.Query().Get("after"); value != "" {
		var err error
		after, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid after"})
			return
		}
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unavailable"})
		return
	}
	ping := time.NewTimer(15 * time.Second)
	defer ping.Stop()
	encoder := json.NewEncoder(w)
	send := func(value interface{}) bool {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Accel-Buffering", "no")
		if err := encoder.Encode(value); err != nil {
			return false
		}
		flusher.Flush()
		ping.Reset(15 * time.Second)
		return true
	}
	poll := time.NewTicker(100 * time.Millisecond)
	defer poll.Stop()
	var p *runProgress
	waiting := false
	for p == nil {
		s.mu.Lock()
		p = s.progress[id]
		status := ""
		for _, list := range s.results {
			for _, run := range list {
				if run.ID == id {
					status = run.Status
					break
				}
			}
		}
		s.mu.Unlock()
		if p != nil {
			break
		}
		if status != "pending" && status != "scheduled" && status != "queued" && status != "running" && status != statusCancelling {
			if waiting {
				send(progressEvent{Seq: 1, Type: "end", At: time.Now().UTC(), Status: status})
				return
			}
			message := "no progress for this run"
			if status == "" {
				message = "no such result"
			}
			writeJSON(w, http.StatusNotFound, map[string]string{"error": message})
			return
		}
		if !waiting {
			if !send(progressSnapshot{Type: "snapshot", Hosts: map[string]hostProgress{}}) {
				return
			}
			waiting = true
		}
		select {
		case <-r.Context().Done():
			return
		case <-ping.C:
			if !send(map[string]string{"type": "ping"}) {
				return
			}
		case <-poll.C:
		}
	}
	snapshot, events, gap, ch := p.subscribe(after)
	defer p.unsubscribe(ch)
	if !send(snapshot) {
		return
	}
	if gap && !send(map[string]string{"type": "gap"}) {
		return
	}
	for _, e := range events {
		if !send(e) {
			return
		}
	}
	if snapshot.Status != "" {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ping.C:
			if !send(map[string]string{"type": "ping"}) {
				return
			}
		case e, open := <-ch:
			if !open {
				return
			}
			if !send(e) || e.Type == "end" {
				return
			}
		}
	}
}
