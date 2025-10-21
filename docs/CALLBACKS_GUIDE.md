# Onigirazu Callbacks Guide

Complete guide to using and creating callback plugins in Onigirazu.

## 📋 Table of Contents

- [What are Callbacks?](#what-are-callbacks)
- [Callback Events](#callback-events)
  - [Playbook Events](#playbook-events)
  - [Play Events](#play-events)
  - [Task Events](#task-events)
- [Using Callbacks](#using-callbacks)
- [Creating Callback Plugins](#creating-callback-plugins)
- [Real-World Examples](#real-world-examples)
- [Advanced Patterns](#advanced-patterns)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## What are Callbacks?

Callbacks are plugin hooks that fire during playbook execution. They allow you to:

- **Monitor execution**: Track when playbooks, plays, and tasks start/end
- **Collect metrics**: Gather performance data and statistics
- **Log events**: Record execution details for audit trails
- **Send notifications**: Alert external systems of execution events
- **Implement custom workflows**: React to execution milestones

### Callback Architecture

```
┌─────────────────────────────────────────────────────┐
│          Onigirazu Core Engine                      │
│                                                     │
│  ┌────────────────────────────────────────────┐   │
│  │  Playbook Executor                        │   │
│  │  ┌──────────────────────────────────────┐ │   │
│  │  │ For each playbook:                   │ │   │
│  │  │  → OnPlaybookStart()                 │ │   │
│  │  │                                      │ │   │
│  │  │  → For each play:                    │ │   │
│  │  │     → OnPlayStart()                  │ │   │
│  │  │     → For each task:                 │ │   │
│  │  │        → For each host:              │ │   │
│  │  │           → OnTaskStart()            │ │   │
│  │  │           → Execute Task             │ │   │
│  │  │           → OnTaskEnd()              │ │   │
│  │  │        → If retry:                   │ │   │
│  │  │           → OnTaskRetry()            │ │   │
│  │  │     → OnPlayEnd()                    │ │   │
│  │  │                                      │ │   │
│  │  │  → OnPlaybookEnd()                   │ │   │
│  │  └──────────────────────────────────────┘ │   │
│  └──────────────┬───────────────────────────┘   │
│                 │                               │
│         ┌───────▼────────┐                      │
│         │ Callback       │                      │
│         │ Manager        │                      │
│         │ Dispatcher     │                      │
│         └───────┬────────┘                      │
│                 │                               │
│    ┌────────────┼────────────┐                  │
│    │            │            │                  │
│ ┌──▼──┐    ┌──▼──┐      ┌──▼──┐               │
│ │Plugin│    │Plugin│      │Plugin │            │
│ │  A   │    │  B   │      │  C   │            │
│ └──────┘    └──────┘      └──────┘            │
└─────────────────────────────────────────────────┘
```

## Callback Events

There are 7 callback events fired during execution:

### Playbook Events

#### 1. `OnPlaybookStart`

Called when a playbook execution begins.

**Signature**:

```go
OnPlaybookStart(ctx context.Context, playbook *types.Playbook) error
```

**Parameters**:

- `ctx`: Context for cancellation and timeouts
- `playbook`: The playbook being executed
  - `Name`: Playbook name
  - `Plays`: List of plays to execute
  - `Vars`: Variables defined in playbook

**Use Cases**:

- Initialize metrics collection
- Log execution start time
- Validate prerequisites
- Send start notification

**Example**:

```yaml
# playbook.yml
---
- name: My Playbook
  hosts: all
  tasks:
    - name: Task 1
      debug:
        msg: "Hello"
```

When this playbook starts, `OnPlaybookStart` fires with:

- `playbook.Name` = "My Playbook"
- `playbook.Plays` = [Play containing Task 1]

#### 2. `OnPlaybookEnd`

Called when a playbook execution completes.

**Signature**:

```go
OnPlaybookEnd(ctx context.Context, playbook *types.Playbook, success bool, duration time.Duration) error
```

**Parameters**:

- `ctx`: Context for cancellation
- `playbook`: The executed playbook
- `success`: True if all tasks succeeded
- `duration`: Total execution time

**Use Cases**:

- Calculate and report metrics
- Save execution results
- Send completion notification
- Cleanup resources

**Example**:

```go
func (c *MyCallback) OnPlaybookEnd(ctx context.Context, playbook *types.Playbook,
    success bool, duration time.Duration) error {

    status := "SUCCESS"
    if !success {
        status = "FAILED"
    }

    fmt.Printf("Playbook %s: %s (Duration: %s)\n",
        playbook.Name, status, duration)

    return nil
}
```

### Play Events

#### 3. `OnPlayStart`

Called when a play execution begins.

**Signature**:

```go
OnPlayStart(ctx context.Context, play *types.Play) error
```

**Parameters**:

- `ctx`: Context for cancellation
- `play`: The play being executed
  - `Name`: Play name
  - `Hosts`: Target hosts pattern
  - `Tasks`: List of tasks in play

**Use Cases**:

- Track play boundaries
- Log which play is running
- Setup play-specific context
- Validate play configuration

**Example**:

```go
func (c *MyCallback) OnPlayStart(ctx context.Context, play *types.Play) error {
    fmt.Printf("\n--- Starting play: %s ---\n", play.Name)
    fmt.Printf("    Hosts: %s\n", play.Hosts)
    fmt.Printf("    Tasks: %d\n", len(play.Tasks))
    return nil
}
```

#### 4. `OnPlayEnd`

Called when a play execution completes.

**Signature**:

```go
OnPlayEnd(ctx context.Context, play *types.Play, success bool, duration time.Duration) error
```

**Parameters**:

- `ctx`: Context for cancellation
- `play`: The executed play
- `success`: True if all tasks in play succeeded
- `duration`: Play execution time

**Use Cases**:

- Report play results
- Record play-level metrics
- Aggregate task results
- Prepare for next play

**Example**:

```go
func (c *MyCallback) OnPlayEnd(ctx context.Context, play *types.Play,
    success bool, duration time.Duration) error {

    status := "PASSED" if success else "FAILED"
    fmt.Printf("--- Play Completed: %s (%s) ---\n", status, duration)

    return nil
}
```

### Task Events

#### 5. `OnTaskStart`

Called when a task execution begins on a host.

**Signature**:

```go
OnTaskStart(ctx context.Context, task *types.Task, host types.Host) error
```

**Parameters**:

- `ctx`: Context for cancellation
- `task`: The task being executed
  - `Name`: Task name
  - `Module`: Module to execute
  - `Args`: Task arguments
- `host`: Target host
  - `Name`: Host name
  - `Address`: Host address
  - `Groups`: Host groups

**Use Cases**:

- Start timing task execution
- Log task start with host context
- Pre-task validation
- Initialize per-task data

**Example**:

```go
func (c *MyCallback) OnTaskStart(ctx context.Context, task *types.Task,
    host types.Host) error {

    fmt.Printf("  > [%s] %s\n", host.Name, task.Name)

    // Store start time for duration calculation
    taskKey := fmt.Sprintf("%s@%s", task.Name, host.Name)
    c.taskStartTimes[taskKey] = time.Now()

    return nil
}
```

#### 6. `OnTaskEnd`

Called when a task execution completes on a host.

**Signature**:

```go
OnTaskEnd(ctx context.Context, task *types.Task, host types.Host,
    result types.TaskResult) error
```

**Parameters**:

- `ctx`: Context for cancellation
- `task`: The executed task
- `host`: Target host
- `result`: Task execution result
  - `Success`: Whether task succeeded
  - `Changed`: Whether task made changes
  - `Output`: Task output/results
  - `Error`: Error message (if any)
  - `Duration`: Task execution time

**Use Cases**:

- Record task results
- Calculate task duration
- Process task output
- Send result notification
- Update status indicators

**Example**:

```go
func (c *MyCallback) OnTaskEnd(ctx context.Context, task *types.Task,
    host types.Host, result types.TaskResult) error {

    status := "ok"
    if !result.Success {
        status = "FAILED"
    } else if result.Changed {
        status = "CHANGED"
    }

    fmt.Printf("    [%s] %s: %s\n", host.Name, task.Name, status)

    if !result.Success {
        fmt.Printf("    Error: %s\n", result.Error)
    }

    return nil
}
```

#### 7. `OnTaskRetry`

Called when a task is retried after failure.

**Signature**:

```go
OnTaskRetry(ctx context.Context, task *types.Task, host types.Host,
    attempt int, err error) error
```

**Parameters**:

- `ctx`: Context for cancellation
- `task`: The task being retried
- `host`: Target host
- `attempt`: Current attempt number (starts at 1)
- `err`: The error that triggered the retry

**Use Cases**:

- Log retry attempts
- Track retry patterns
- Adjust behavior on retry
- Send retry notifications

**Example**:

```go
func (c *MyCallback) OnTaskRetry(ctx context.Context, task *types.Task,
    host types.Host, attempt int, err error) error {

    fmt.Printf("  [RETRY] %s on %s (Attempt %d): %v\n",
        task.Name, host.Name, attempt, err)

    return nil
}
```

## Using Callbacks

### Registering Callbacks

Callbacks are registered with the callback manager:

```go
import (
    "context"
    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
)

// Create callback manager
callbackManager := plugins.NewCallbackManager()

// Create callback plugin
myCallback := MyCallbackPlugin{}

// Add to manager
callbackManager.AddPlugin(myCallback)
```

### Callback Configuration

Configure callbacks in `plugins.yml`:

```yaml
plugins:
  - name: my_callback
    type: callback
    enabled: true
    path: /path/to/callback.so
    config:
      verbose: true
      log_file: /var/log/onigirazu.log
      send_notifications: true
```

### Multiple Callbacks

Multiple callbacks can be registered and will all fire:

```go
callbackManager.AddPlugin(metricsCallback)
callbackManager.AddPlugin(loggingCallback)
callbackManager.AddPlugin(notificationCallback)

// All three callbacks fire for each event
callbackManager.OnTaskStart(ctx, task, host)
```

**Execution Order**: Callbacks fire in registration order.

## Creating Callback Plugins

### Basic Callback Plugin

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type SimpleCallback struct {
    *plugins.BaseCallbackPlugin
}

// NewPlugin is entry point
func NewPlugin() plugins.Plugin {
    return &SimpleCallback{
        BaseCallbackPlugin: plugins.NewBaseCallbackPlugin(
            "simple_logger",
            "1.0.0",
            "Simple logging callback",
        ),
    }
}

// OnPlaybookStart - Playbook starts
func (c *SimpleCallback) OnPlaybookStart(ctx context.Context,
    playbook *types.Playbook) error {
    fmt.Printf("\n=== Playbook: %s ===\n", playbook.Name)
    return nil
}

// OnPlaybookEnd - Playbook completes
func (c *SimpleCallback) OnPlaybookEnd(ctx context.Context,
    playbook *types.Playbook, success bool, duration time.Duration) error {
    status := "SUCCESS"
    if !success {
        status = "FAILED"
    }
    fmt.Printf("Playbook %s: %s (%s)\n\n", playbook.Name, status, duration)
    return nil
}

// OnPlayStart - Play starts
func (c *SimpleCallback) OnPlayStart(ctx context.Context,
    play *types.Play) error {
    fmt.Printf("\n--- Play: %s ---\n", play.Name)
    return nil
}

// OnPlayEnd - Play completes
func (c *SimpleCallback) OnPlayEnd(ctx context.Context, play *types.Play,
    success bool, duration time.Duration) error {
    fmt.Printf("--- Play End: %s ---\n", play.Name)
    return nil
}

// OnTaskStart - Task starts
func (c *SimpleCallback) OnTaskStart(ctx context.Context, task *types.Task,
    host types.Host) error {
    fmt.Printf("  [%s] %s\n", host.Name, task.Name)
    return nil
}

// OnTaskEnd - Task completes
func (c *SimpleCallback) OnTaskEnd(ctx context.Context, task *types.Task,
    host types.Host, result types.TaskResult) error {
    status := "ok"
    if !result.Success {
        status = "FAILED"
    } else if result.Changed {
        status = "CHANGED"
    }
    fmt.Printf("    %s\n", status)
    return nil
}

// OnTaskRetry - Task is retried
func (c *SimpleCallback) OnTaskRetry(ctx context.Context, task *types.Task,
    host types.Host, attempt int, err error) error {
    fmt.Printf("  [RETRY] Attempt %d: %v\n", attempt, err)
    return nil
}
```

### Callback Plugin with Metrics Collection

See [callback_metrics.go](../examples/plugins/callback_metrics.go) for a complete example:

```go
type MetricsCallback struct {
    *plugins.BaseCallbackPlugin
    mu             sync.RWMutex
    tasksStarted   int
    tasksCompleted int
    tasksSucceeded int
    tasksFailed    int
    totalDuration  time.Duration
    taskDurations  map[string]time.Duration
}

func (c *MetricsCallback) OnTaskStart(ctx context.Context, task *types.Task,
    host types.Host) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.tasksStarted++
    taskKey := fmt.Sprintf("%s@%s", task.Name, host.Name)
    c.taskStartTimes[taskKey] = time.Now()

    return nil
}

func (c *MetricsCallback) OnTaskEnd(ctx context.Context, task *types.Task,
    host types.Host, result types.TaskResult) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.tasksCompleted++
    if result.Success {
        c.tasksSucceeded++
    } else {
        c.tasksFailed++
    }

    // Track duration
    taskKey := fmt.Sprintf("%s@%s", task.Name, host.Name)
    if startTime, exists := c.taskStartTimes[taskKey]; exists {
        duration := time.Since(startTime)
        c.taskDurations[taskKey] = duration
        c.totalDuration += duration
    }

    return nil
}

// Custom method to get metrics
func (c *MetricsCallback) GetMetrics() map[string]interface{} {
    c.mu.RLock()
    defer c.mu.RUnlock()

    return map[string]interface{}{
        "tasks_started":   c.tasksStarted,
        "tasks_completed": c.tasksCompleted,
        "tasks_succeeded": c.tasksSucceeded,
        "tasks_failed":    c.tasksFailed,
        "total_duration":  c.totalDuration.String(),
    }
}
```

## Real-World Examples

### Example 1: Metrics Collection Callback

```go
package main

import (
    "context"
    "sync"
    "time"

    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type MetricsCollector struct {
    *plugins.BaseCallbackPlugin
    mu      sync.RWMutex
    metrics map[string]interface{}
}

func NewMetricsCollector() *MetricsCollector {
    return &MetricsCollector{
        BaseCallbackPlugin: plugins.NewBaseCallbackPlugin(
            "metrics",
            "1.0.0",
            "Collects execution metrics",
        ),
        metrics: make(map[string]interface{}),
    }
}

func (mc *MetricsCollector) OnPlaybookStart(ctx context.Context,
    playbook *types.Playbook) error {
    mc.mu.Lock()
    defer mc.mu.Unlock()

    mc.metrics["playbook_start"] = time.Now()
    mc.metrics["playbook_name"] = playbook.Name

    return nil
}

func (mc *MetricsCollector) OnPlaybookEnd(ctx context.Context,
    playbook *types.Playbook, success bool, duration time.Duration) error {
    mc.mu.Lock()
    defer mc.mu.Unlock()

    mc.metrics["playbook_end"] = time.Now()
    mc.metrics["playbook_success"] = success
    mc.metrics["playbook_duration"] = duration.String()

    return nil
}

// ... other methods
```

### Example 2: Notification Callback

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type NotificationCallback struct {
    *plugins.BaseCallbackPlugin
    webhookURL string
}

func (nc *NotificationCallback) Initialize(ctx context.Context,
    config map[string]interface{}) error {
    if url, ok := config["webhook_url"].(string); ok {
        nc.webhookURL = url
    }
    return nil
}

func (nc *NotificationCallback) sendNotification(title, message string) error {
    if nc.webhookURL == "" {
        return nil
    }

    payload := map[string]interface{}{
        "title":   title,
        "message": message,
        "time":    time.Now(),
    }

    // Send webhook
    _, err := http.Post(nc.webhookURL, "application/json",
        // marshal payload
        nil)

    return err
}

func (nc *NotificationCallback) OnPlaybookStart(ctx context.Context,
    playbook *types.Playbook) error {
    return nc.sendNotification(
        "Playbook Started",
        fmt.Sprintf("Playbook '%s' has started", playbook.Name),
    )
}

func (nc *NotificationCallback) OnPlaybookEnd(ctx context.Context,
    playbook *types.Playbook, success bool, duration time.Duration) error {
    status := "completed successfully"
    if !success {
        status = "failed"
    }

    return nc.sendNotification(
        "Playbook Finished",
        fmt.Sprintf("Playbook '%s' %s in %s",
            playbook.Name, status, duration),
    )
}

// ... other methods
```

## Advanced Patterns

### Callback with State Management

```go
type StatefulCallback struct {
    *plugins.BaseCallbackPlugin
    state map[string]interface{}
    mu    sync.RWMutex
}

func (sc *StatefulCallback) OnPlayStart(ctx context.Context,
    play *types.Play) error {
    sc.mu.Lock()
    defer sc.mu.Unlock()

    playID := play.Name
    sc.state[playID] = map[string]interface{}{
        "start_time": time.Now(),
        "tasks":      make(map[string]interface{}),
    }

    return nil
}

func (sc *StatefulCallback) GetPlayState(playName string) interface{} {
    sc.mu.RLock()
    defer sc.mu.RUnlock()

    return sc.state[playName]
}
```

### Callback with Error Recovery

```go
func (c *MyCallback) OnTaskRetry(ctx context.Context, task *types.Task,
    host types.Host, attempt int, err error) error {

    // Implement exponential backoff logic
    if attempt > 3 {
        return fmt.Errorf("max retries exceeded")
    }

    // Can influence retry behavior
    fmt.Printf("Retry attempt %d for %s\n", attempt, task.Name)

    return nil
}
```

### Callback Event Aggregation

```go
type AggregatingCallback struct {
    *plugins.BaseCallbackPlugin
    events []ExecutionEvent
    mu     sync.Mutex
}

type ExecutionEvent struct {
    Timestamp time.Time
    Type      string
    Data      interface{}
}

func (ac *AggregatingCallback) OnTaskStart(ctx context.Context,
    task *types.Task, host types.Host) error {
    ac.mu.Lock()
    defer ac.mu.Unlock()

    ac.events = append(ac.events, ExecutionEvent{
        Timestamp: time.Now(),
        Type:      "task_start",
        Data: map[string]string{
            "task": task.Name,
            "host": host.Name,
        },
    })

    return nil
}

func (ac *AggregatingCallback) GetEvents() []ExecutionEvent {
    ac.mu.Lock()
    defer ac.mu.Unlock()

    result := make([]ExecutionEvent, len(ac.events))
    copy(result, ac.events)

    return result
}
```

## Best Practices

### 1. **Thread Safety**

Always use proper synchronization:

```go
type SafeCallback struct {
    *plugins.BaseCallbackPlugin
    mu    sync.RWMutex
    data  map[string]interface{}
}

func (sc *SafeCallback) OnTaskEnd(ctx context.Context, task *types.Task,
    host types.Host, result types.TaskResult) error {
    sc.mu.Lock()
    defer sc.mu.Unlock()

    sc.data[task.Name] = result
    return nil
}
```

### 2. **Error Handling**

Handle errors gracefully:

```go
func (c *MyCallback) OnPlaybookEnd(ctx context.Context,
    playbook *types.Playbook, success bool, duration time.Duration) error {

    // Don't crash on errors - callbacks should be resilient
    if err := c.saveMetrics(playbook); err != nil {
        // Log but don't return error
        fmt.Printf("Warning: failed to save metrics: %v\n", err)
    }

    return nil
}
```

### 3. **Resource Cleanup**

Implement cleanup properly:

```go
func (c *MyCallback) Cleanup(ctx context.Context) error {
    // Close connections
    if c.conn != nil {
        c.conn.Close()
    }

    // Flush buffers
    if c.buffer != nil {
        c.buffer.Flush()
    }

    return nil
}
```

### 4. **Context Awareness**

Respect context cancellation:

```go
func (c *MyCallback) OnTaskStart(ctx context.Context, task *types.Task,
    host types.Host) error {

    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Continue normal processing
    }

    // Do work
    return nil
}
```

### 5. **Minimal Overhead**

Keep callbacks fast:

```go
// Good: minimal work in callback
func (c *FastCallback) OnTaskEnd(ctx context.Context, task *types.Task,
    host types.Host, result types.TaskResult) error {

    // Queue work for async processing
    go c.processResult(task, host, result)

    return nil
}

// Bad: heavy work in callback
func (c *SlowCallback) OnTaskEnd(ctx context.Context, task *types.Task,
    host types.Host, result types.TaskResult) error {

    // This blocks execution
    c.analyzeResult(task, host, result)
    c.generateReport()
    c.sendNotifications()

    return nil
}
```

## Troubleshooting

### Callback Not Firing

**Problem**: Callback methods not called.

**Solutions**:

1. Verify callback is registered: `callbackManager.AddPlugin(callback)`
2. Check callback is enabled in config
3. Ensure plugin initialization succeeded
4. Verify callback manager is set on core engine

### Thread Safety Issues

**Problem**: Race conditions in callback data.

**Solution**: Use proper synchronization:

```go
type SafeCallback struct {
    mu   sync.RWMutex
    data map[string]interface{}
}

// Always lock before accessing shared data
func (c *SafeCallback) getData() interface{} {
    c.mu.RLock()
    defer c.mu.RUnlock()

    return c.data["key"]
}
```

### Callbacks Blocking Execution

**Problem**: Tasks are running slowly.

**Solution**: Move heavy work outside callback:

```go
func (c *MyCallback) OnTaskEnd(ctx context.Context, task *types.Task,
    host types.Host, result types.TaskResult) error {

    // Quick: just record the result
    c.recordResult(result)

    // Slow work in background
    go c.analyzeAndReport(result)

    return nil
}
```

### Out of Memory Issues

**Problem**: Callback storing too much data.

**Solution**: Implement data cleanup:

```go
type WiseCallback struct {
    *plugins.BaseCallbackPlugin
    maxEvents int
    events    []Event
    mu        sync.Mutex
}

func (wc *WiseCallback) recordEvent(event Event) {
    wc.mu.Lock()
    defer wc.mu.Unlock()

    wc.events = append(wc.events, event)

    // Keep only recent events
    if len(wc.events) > wc.maxEvents {
        wc.events = wc.events[1:]
    }
}
```

## API Reference

### CallbackPlugin Interface

```go
type CallbackPlugin interface {
    Plugin  // Inherits GetName, GetVersion, GetDescription, Initialize, Cleanup

    OnPlaybookStart(ctx context.Context, playbook *types.Playbook) error
    OnPlaybookEnd(ctx context.Context, playbook *types.Playbook,
        success bool, duration time.Duration) error

    OnPlayStart(ctx context.Context, play *types.Play) error
    OnPlayEnd(ctx context.Context, play *types.Play,
        success bool, duration time.Duration) error

    OnTaskStart(ctx context.Context, task *types.Task, host types.Host) error
    OnTaskEnd(ctx context.Context, task *types.Task, host types.Host,
        result types.TaskResult) error
    OnTaskRetry(ctx context.Context, task *types.Task, host types.Host,
        attempt int, err error) error
}
```

### BaseCallbackPlugin

```go
type BaseCallbackPlugin struct {
    // Unexported fields
}

// Methods
func NewBaseCallbackPlugin(name, version, description string) *BaseCallbackPlugin
func (p *BaseCallbackPlugin) GetName() string
func (p *BaseCallbackPlugin) GetType() PluginType
func (p *BaseCallbackPlugin) GetVersion() string
func (p *BaseCallbackPlugin) GetDescription() string
func (p *BaseCallbackPlugin) Initialize(ctx context.Context,
    config map[string]interface{}) error
func (p *BaseCallbackPlugin) Cleanup(ctx context.Context) error

// Event methods (default no-op implementations)
func (p *BaseCallbackPlugin) OnPlaybookStart(ctx context.Context,
    playbook *types.Playbook) error
func (p *BaseCallbackPlugin) OnPlaybookEnd(ctx context.Context,
    playbook *types.Playbook, success bool, duration time.Duration) error
func (p *BaseCallbackPlugin) OnPlayStart(ctx context.Context, play *types.Play) error
func (p *BaseCallbackPlugin) OnPlayEnd(ctx context.Context, play *types.Play,
    success bool, duration time.Duration) error
func (p *BaseCallbackPlugin) OnTaskStart(ctx context.Context, task *types.Task,
    host types.Host) error
func (p *BaseCallbackPlugin) OnTaskEnd(ctx context.Context, task *types.Task,
    host types.Host, result types.TaskResult) error
func (p *BaseCallbackPlugin) OnTaskRetry(ctx context.Context, task *types.Task,
    host types.Host, attempt int, err error) error
```

### CallbackManager

```go
type CallbackManager struct {
    // Unexported fields
}

// Methods
func NewCallbackManager() *CallbackManager
func (m *CallbackManager) AddPlugin(plugin CallbackPlugin)
func (m *CallbackManager) RemovePlugin(name string)

// Event dispatch methods
func (m *CallbackManager) OnPlaybookStart(ctx context.Context,
    playbook *types.Playbook) error
func (m *CallbackManager) OnPlaybookEnd(ctx context.Context,
    playbook *types.Playbook, success bool, duration time.Duration) error
func (m *CallbackManager) OnPlayStart(ctx context.Context, play *types.Play) error
func (m *CallbackManager) OnPlayEnd(ctx context.Context, play *types.Play,
    success bool, duration time.Duration) error
func (m *CallbackManager) OnTaskStart(ctx context.Context, task *types.Task,
    host types.Host) error
func (m *CallbackManager) OnTaskEnd(ctx context.Context, task *types.Task,
    host types.Host, result types.TaskResult) error
func (m *CallbackManager) OnTaskRetry(ctx context.Context, task *types.Task,
    host types.Host, attempt int, err error) error
```

## See Also

- [PLUGIN_INTEGRATION.md](PLUGIN_INTEGRATION.md) - General plugin integration guide
- [FILTERS_GUIDE.md](FILTERS_GUIDE.md) - Filters guide
- [examples/plugins/callback_metrics.go](../examples/plugins/callback_metrics.go) - Example callback
