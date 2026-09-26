# Callback Plugins

A callback plugin receives events while `apply` (and `plan`/`drift`, which run through
it) executes a playbook: for logging, metrics, notifications and similar. How plugins
are configured, built and loaded is described in [PLUGIN_INTEGRATION.md](PLUGIN_INTEGRATION.md).

## Events

```go
type CallbackPlugin interface {
    Plugin
    OnPlaybookStart(ctx context.Context, playbook *types.Playbook) error
    OnPlaybookEnd(ctx context.Context, playbook *types.Playbook, success bool, duration time.Duration) error
    OnPlayStart(ctx context.Context, play *types.Play) error
    OnPlayEnd(ctx context.Context, play *types.Play, success bool, duration time.Duration) error
    OnTaskStart(ctx context.Context, task *types.Task, host types.Host) error
    OnTaskEnd(ctx context.Context, task *types.Task, host types.Host, result types.TaskResult) error
    OnTaskRetry(ctx context.Context, task *types.Task, host types.Host, attempt int, err error) error
}
```

| Event | When |
|-------|------|
| `OnPlaybookStart` / `OnPlaybookEnd` | once per playbook run; `success` is false if the run failed |
| `OnPlayStart` / `OnPlayEnd` | around each play |
| `OnTaskStart` / `OnTaskEnd` | around each task on each host; `result` has `Success`, `Failed`, `Changed`, `Skipped`, `Output`, `Error` |
| `OnTaskRetry` | currently never called; implement it as a no-op |

## Behaviour

- Task events come from parallel host workers: a plugin must be safe for concurrent use.
- A returned error is logged as a warning (`callback plugin NAME: EVENT: …`) and never
  stops the run.
- With several callback plugins the order in which they are called is not defined.
- `Initialize(ctx, config)` gets the `config` map from `plugins.yml`. `Cleanup` is not
  called at the end of the run, so flush files in `OnPlaybookEnd`.
- Callbacks run inline with task execution; keep them fast.

## Example

Embed `plugins.BaseCallbackPlugin` (all events are no-ops) and override what you need:

```go
//go:build plugin

package main

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type counter struct {
    *plugins.BaseCallbackPlugin
    mu      sync.Mutex
    changed int
    failed  int
}

func (c *counter) OnTaskEnd(ctx context.Context, task *types.Task, host types.Host, r types.TaskResult) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    if r.Failed {
        c.failed++
    } else if r.Changed {
        c.changed++
    }
    return nil
}

func (c *counter) OnPlaybookEnd(ctx context.Context, pb *types.Playbook, success bool, d time.Duration) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    fmt.Printf("%s: %d changed, %d failed in %s\n", pb.Name, c.changed, c.failed, d)
    return nil
}

func NewPlugin() plugins.Plugin {
    return &counter{BaseCallbackPlugin: plugins.NewBaseCallbackPlugin("counter", "1.0.0", "Counts task results")}
}
```

```yaml
# plugins.yml next to the playbook
plugins:
  - name: counter
    type: callback
    enabled: true
    path: counter.so
```

A fuller example that collects timing metrics: `examples/plugins/callback_metrics.go`.
