// Package asyncjob keeps the tasks started with async and poll: 0
package asyncjob

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// job is a task started with async and poll: 0; it runs in the
// background of this process until it ends or its async seconds pass
type job struct {
	host   string
	done   chan struct{}
	result types.TaskResult
}

var (
	mu   sync.Mutex
	jobs = map[string]*job{}
	seq  atomic.Int64
)

// Start runs fn in the background for host and returns its job id
func Start(host string, fn func() types.TaskResult) string {
	jid := fmt.Sprintf("j%d.%d", time.Now().UnixNano()%1_000_000_000_000, seq.Add(1))
	j := &job{host: host, done: make(chan struct{})}
	mu.Lock()
	jobs[jid] = j
	mu.Unlock()
	go func() {
		defer close(j.done)
		j.result = fn()
	}()
	return jid
}

// Wait waits for the background jobs still running (each ends
// within its async seconds) and returns how many it waited for
func Wait(ctx context.Context) int {
	mu.Lock()
	var running []*job
	for _, j := range jobs {
		select {
		case <-j.done:
		default:
			running = append(running, j)
		}
	}
	mu.Unlock()
	for _, j := range running {
		select {
		case <-j.done:
		case <-ctx.Done():
			return len(running)
		}
	}
	return len(running)
}

// Status is a job's state: ok false when host has no job jid; done with
// the task's result once it ended
func Status(host, jid string) (result types.TaskResult, done, ok bool) {
	mu.Lock()
	j, ok := jobs[jid]
	mu.Unlock()
	if !ok || j.host != host {
		return types.TaskResult{}, false, false
	}
	select {
	case <-j.done:
		return j.result, true, true
	default:
		return types.TaskResult{}, false, true
	}
}

// Cleanup forgets host's job jid and reports whether there was one
func Cleanup(host, jid string) bool {
	mu.Lock()
	defer mu.Unlock()
	if j, ok := jobs[jid]; ok && j.host == host {
		delete(jobs, jid)
		return true
	}
	return false
}
