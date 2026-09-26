package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Handlers, as in Ansible: a task that changed something notifies handlers
// for its host; notified handlers run in the order they are defined, only on
// the hosts that notified them, after pre_tasks, after roles and tasks, after
// post_tasks, and at meta: flush_handlers. meta: end_host and end_play stop
// hosts for the rest of the play.

type handlerEntry struct {
	task types.Task
	vars map[string]interface{}
}

type playHandlers struct {
	mu       sync.Mutex
	entries  []handlerEntry
	notified map[string]map[string]bool // notify name -> hosts
	ended    map[string]bool            // hosts stopped by end_host/end_play
	hosts    []types.Host               // hosts of the play
}

// startPlayHandlers resets the handler state for a play
func (e *ExecutionEngine) startPlayHandlers(hosts []types.Host, handlers []types.Task, vars map[string]interface{}) {
	e.handlers = &playHandlers{
		notified: map[string]map[string]bool{},
		ended:    map[string]bool{},
		hosts:    hosts,
	}
	e.addHandlers(handlers, vars)
}

// addHandlers registers handlers (a role's, with the role's variables)
func (e *ExecutionEngine) addHandlers(handlers []types.Task, vars map[string]interface{}) {
	h := e.handlers
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, t := range handlers {
		h.entries = append(h.entries, handlerEntry{task: t, vars: vars})
	}
}

// notifyHandlers records that host notified the given handlers
func (e *ExecutionEngine) notifyHandlers(names []string, host string) {
	h := e.handlers
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, name := range names {
		if h.notified[name] == nil {
			h.notified[name] = map[string]bool{}
		}
		h.notified[name][host] = true
	}
}

// hostEnded tells whether meta end_host/end_play stopped a host
func (e *ExecutionEngine) hostEnded(host string) bool {
	h := e.handlers
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.ended[host]
}

func (e *ExecutionEngine) endHosts(names ...string) {
	h := e.handlers
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, n := range names {
		h.ended[n] = true
	}
}

// takeNotified returns, per handler entry, the hosts (of the given ones)
// that notified it, and forgets those notifications
func (h *playHandlers) takeNotified(hosts []types.Host) [][]types.Host {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([][]types.Host, len(h.entries))
	for i, entry := range h.entries {
		for _, host := range hosts {
			if h.ended[host.Name] {
				continue
			}
			if h.notified[entry.task.Name][host.Name] || (entry.task.Listen != "" && h.notified[entry.task.Listen][host.Name]) {
				out[i] = append(out[i], host)
			}
		}
	}
	for _, host := range hosts {
		for _, set := range h.notified {
			delete(set, host.Name)
		}
	}
	return out
}

// flushHandlers runs the handlers notified by the given hosts. A handler may
// notify another one, which then runs in a further pass.
func (e *ExecutionEngine) flushHandlers(ctx context.Context, hosts []types.Host, playResult *types.PlayResult) error {
	h := e.handlers
	if h == nil {
		return nil
	}
	for pass := 0; pass < 10; pass++ {
		toRun := h.takeNotified(hosts)
		ran := false
		for i, targets := range toRun {
			if len(targets) == 0 {
				continue
			}
			ran = true
			entry := h.entries[i]
			e.logger.Debug("Running handler %s on %d host(s)", entry.task.Name, len(targets))
			if err := e.executeTask(ctx, &entry.task, targets, entry.vars, playResult); err != nil && !entry.task.IgnoreErrors {
				return fmt.Errorf("handler '%s' failed: %w", entry.task.Name, err)
			}
		}
		if !ran {
			return nil
		}
	}
	return fmt.Errorf("handlers kept notifying each other")
}

// metaAction is the action of a meta task: "meta: flush_handlers"
func metaAction(task *types.Task) string {
	for _, key := range []string{"free_form", "_raw_params", "action"} {
		if s, ok := task.Args[key].(string); ok {
			return s
		}
	}
	return ""
}

// runMeta performs a per-host meta action; flush_handlers is handled for all
// hosts of the task in executeTask
func (e *ExecutionEngine) runMeta(task *types.Task, host *types.Host, playResult *types.PlayResult) error {
	result := types.TaskResult{
		TaskName: task.Name, Host: host.Name, Module: "meta", Success: true,
		Output: map[string]interface{}{"msg": metaAction(task)},
	}
	switch action := metaAction(task); action {
	case "noop", "flush_handlers", "clear_host_errors", "refresh_inventory", "reset_connection":
	case "end_host":
		e.endHosts(host.Name)
	case "end_play":
		names := []string{}
		if e.handlers != nil {
			for _, h := range e.handlers.hosts {
				names = append(names, h.Name)
			}
		}
		e.endHosts(append(names, host.Name)...)
	default:
		result.Success, result.Failed = false, true
		result.Error = fmt.Sprintf("unsupported meta action %q", action)
	}
	return e.finishTask(task, host, result, playResult)
}
