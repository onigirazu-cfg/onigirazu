package engine

import (
	"context"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/plugins"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Callback plugins get playbook, play and per-host task events. Task events
// come from parallel host workers, so plugins must be safe for concurrent
// use. A callback error is logged and never stops the run.

// SetCallbacks sets the callback plugins that receive execution events
func (e *ExecutionEngine) SetCallbacks(callbacks []plugins.CallbackPlugin) {
	e.callbacks = callbacks
}

func (e *ExecutionEngine) callback(event string, call func(plugins.CallbackPlugin) error) {
	for _, cb := range e.callbacks {
		if err := call(cb); err != nil {
			e.logger.Warn("callback plugin %s: %s: %v", cb.GetName(), event, err)
		}
	}
}

func (e *ExecutionEngine) callbackPlaybookStart(ctx context.Context, playbook *types.Playbook) {
	e.callback("playbook start", func(cb plugins.CallbackPlugin) error { return cb.OnPlaybookStart(ctx, playbook) })
}

func (e *ExecutionEngine) callbackPlaybookEnd(ctx context.Context, playbook *types.Playbook, success bool, duration time.Duration) {
	e.callback("playbook end", func(cb plugins.CallbackPlugin) error {
		return cb.OnPlaybookEnd(ctx, playbook, success, duration)
	})
}

func (e *ExecutionEngine) callbackPlayStart(ctx context.Context, play *types.Play) {
	e.callback("play start", func(cb plugins.CallbackPlugin) error { return cb.OnPlayStart(ctx, play) })
}

func (e *ExecutionEngine) callbackPlayEnd(ctx context.Context, play *types.Play, success bool, duration time.Duration) {
	e.callback("play end", func(cb plugins.CallbackPlugin) error { return cb.OnPlayEnd(ctx, play, success, duration) })
}

func (e *ExecutionEngine) callbackTaskStart(ctx context.Context, task *types.Task, host types.Host) {
	e.callback("task start", func(cb plugins.CallbackPlugin) error { return cb.OnTaskStart(ctx, task, host) })
}

func (e *ExecutionEngine) callbackTaskEnd(ctx context.Context, task *types.Task, host types.Host, result types.TaskResult) {
	e.callback("task end", func(cb plugins.CallbackPlugin) error { return cb.OnTaskEnd(ctx, task, host, result) })
}
