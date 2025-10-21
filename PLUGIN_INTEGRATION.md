# Plugin System Integration Guide

This document describes how the plugin system is integrated with Onigirazu's core engine and how to use it.

## Overview

The plugin system is fully integrated with:

- **Template Engine** - Filter plugins for custom template filters (see [FILTERS_GUIDE.md](FILTERS_GUIDE.md))
- **Core Engine** - Callback plugins for execution event hooks (see [CALLBACKS_GUIDE.md](CALLBACKS_GUIDE.md))
- **Module Registry** - Module plugins for custom task modules
- **Inventory Manager** - Inventory plugins for dynamic inventory sources

**Quick Links**:

- 📚 [Complete Filters Guide](FILTERS_GUIDE.md) - Built-in filters, custom filters, best practices
- 📚 [Complete Callbacks Guide](CALLBACKS_GUIDE.md) - Callback events, plugin development, examples
- 🔧 [MODULE_DEVELOPMENT_GUIDE.md](MODULE_DEVELOPMENT_GUIDE.md) - Module plugin development

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Core Engine                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Template   │  │   Executor   │  │  Inventory   │      │
│  │    Engine    │  │              │  │   Manager    │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                 │                  │               │
│         └─────────────────┼──────────────────┘               │
│                           │                                  │
│                    ┌──────▼───────┐                         │
│                    │    Plugin    │                         │
│                    │   Manager    │                         │
│                    └──────┬───────┘                         │
│                           │                                  │
│         ┌─────────────────┼─────────────────┐               │
│         │                 │                 │               │
│    ┌────▼────┐      ┌────▼────┐      ┌────▼────┐          │
│    │ Filter  │      │Callback │      │ Module  │          │
│    │ Plugins │      │ Plugins │      │ Plugins │          │
│    └─────────┘      └─────────┘      └─────────┘          │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. Plugin Manager

The `plugins.Manager` is the central component that manages all plugins:

```go
import "github.com/onigirazu-cfg/onigirazu/internal/plugins"

// Create plugin manager
loader := plugins.NewInMemoryLoader()
manager := plugins.NewManager(loader)

// Register plugins
ctx := context.Background()
manager.Register(ctx, myPlugin)
```

### 2. Template Engine Integration

Filter plugins are automatically loaded into the template engine:

```go
import (
    "github.com/onigirazu-cfg/onigirazu/internal/template"
    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
)

// Create template engine with plugin support
pluginManager := plugins.NewManager(loader)
engine := template.NewEngineWithPlugins(pluginManager)

// Or add plugins to existing engine
engine.SetPluginManager(pluginManager)
```

**Built-in Filters:**

The plugin system includes 9 built-in filters:

**String Manipulation**:

- `upper` - Convert to uppercase: `{{ "hello" | upper }}` → `"HELLO"`
- `lower` - Convert to lowercase: `{{ "HELLO" | lower }}` → `"hello"`
- `title` - Convert to title case: `{{ "hello world" | title }}` → `"Hello World"`
- `trim` - Remove whitespace: `{{ "  hello  " | trim }}` → `"hello"`
- `replace(old, new)` - Replace substring: `{{ "hello" | replace("l", "L") }}` → `"heLLo"`

**Collection Operations**:

- `length` - Get length: `{{ [1,2,3] | length }}` → `3`
- `join(sep)` - Join array: `{{ ["a","b"] | join(",") }}` → `"a,b"`
- `split(sep)` - Split string: `{{ "a,b" | split(",") }}` → `["a", "b"]`

**Conditional**:

- `default(value)` - Provide default: `{{ empty_var | default("fallback") }}` → `"fallback"`

For detailed documentation on each filter, including advanced examples and patterns, see [FILTERS_GUIDE.md](FILTERS_GUIDE.md).

**Usage in Templates:**

```yaml
vars:
  app_name: "myapp"
  users: ["alice", "bob"]
  env: "production"

tasks:
  - name: String filters
    debug:
      msg: "App: {{ app_name | upper }}, Env: {{ env | title }}"

  - name: Collection filters
    debug:
      msg: "{{ users | length }} users: {{ users | join(', ') }}"

  - name: Chained filters
    debug:
      msg: "{{ 'a,b,c' | split(',') | join(' -> ') | upper }}"
      # Output: A -> B -> C

  - name: Conditional filter
    debug:
      msg: "Role: {{ user_role | default('guest') }}"
```

### 3. Core Engine Integration

Callback plugins are integrated with the core engine to hook into execution events:

```go
import (
    "github.com/onigirazu-cfg/onigirazu/internal/core"
    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
)

// Create core engine with plugin support
pluginManager := plugins.NewManager(loader)
engine := core.NewCoreEngineWithPlugins(logger, pluginManager)

// Or add plugins to existing engine
engine.SetPluginManager(pluginManager)
```

**Callback Events:**

There are 7 callback events in the execution lifecycle:

**Playbook Level**:

- `OnPlaybookStart(ctx, playbook)` - When playbook execution starts
  - Use for: initialization, logging start time
- `OnPlaybookEnd(ctx, playbook, success, duration)` - When playbook execution ends
  - Use for: finalizing metrics, logging results, cleanup

**Play Level**:

- `OnPlayStart(ctx, play)` - When play execution starts
  - Use for: track play boundaries, setup play context
- `OnPlayEnd(ctx, play, success, duration)` - When play execution ends
  - Use for: record play results, aggregate task metrics

**Task Level**:

- `OnTaskStart(ctx, task, host)` - When task execution starts on a host
  - Use for: start timing, log task start
- `OnTaskEnd(ctx, task, host, result)` - When task execution ends on a host
  - Use for: record results, calculate duration, process output
- `OnTaskRetry(ctx, task, host, attempt, error)` - When task is retried
  - Use for: log retries, track retry patterns

For detailed documentation on callback development, events, and examples, see [CALLBACKS_GUIDE.md](CALLBACKS_GUIDE.md).

### 4. Module Registry Integration

Module plugins can be registered with the module registry:

```go
import (
    "github.com/onigirazu-cfg/onigirazu/internal/modules"
    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
)

// Get module plugin from manager
modulePlugin, err := pluginManager.GetModule("hello")
if err != nil {
    return err
}

// Use module plugin in tasks
result, err := modulePlugin.Execute(ctx, host, args)
```

## Configuration

### Plugin Configuration File

Create a `plugins.yml` file to configure plugins:

```yaml
# plugins.yml
plugins_dir: "./plugins"

plugins:
  # Built-in filter plugin
  - name: builtin_filters
    type: filter
    enabled: true

  # Custom module plugin
  - name: hello
    type: module
    enabled: true
    path: module_hello.so
    config:
      default_greeting: "Hello"

  # Metrics callback plugin
  - name: metrics
    type: callback
    enabled: true
    path: callback_metrics.so
    config:
      output_file: "/tmp/metrics.json"
```

### Loading Configuration

```go
import "github.com/onigirazu-cfg/onigirazu/internal/plugins"

// Load configuration
config, err := plugins.LoadConfig("plugins.yml")
if err != nil {
    return err
}

// Create manager and load plugins
loader := plugins.NewInMemoryLoader()
manager := plugins.NewManager(loader)

ctx := context.Background()
if err := plugins.LoadPluginsFromConfig(ctx, manager, config); err != nil {
    return err
}
```

## Usage Examples

### Example 1: Using Filter Plugins in Playbooks

```yaml
---
- name: Filter Plugin Demo
  hosts: localhost
  vars:
    app_name: "myapp"
    users: ["alice", "bob", "charlie"]

  tasks:
    - name: Display uppercase app name
      debug:
        msg: "{{ app_name | upper }}"

    - name: Display user count
      debug:
        msg: "Users ({{ users | length }}): {{ users | join(', ') }}"
```

### Example 2: Creating a Custom Filter Plugin

For comprehensive filter plugin development, see [FILTERS_GUIDE.md](FILTERS_GUIDE.md).

Quick example of a custom filter:

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
)

type ReverseFilterPlugin struct {
    *plugins.BaseFilterPlugin
}

func NewReverseFilterPlugin() *ReverseFilterPlugin {
    plugin := &ReverseFilterPlugin{
        BaseFilterPlugin: plugins.NewBaseFilterPlugin(
            "reverse_filter",
            "1.0.0",
            "Reverse string filter",
        ),
    }

    // Register filter function
    plugin.AddFilter("reverse", reverseFilter)

    return plugin
}

func reverseFilter(input interface{}, args ...interface{}) (interface{}, error) {
    str, ok := input.(string)
    if !ok {
        return nil, fmt.Errorf("reverse filter expects string input, got %T", input)
    }

    runes := []rune(str)
    for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
        runes[i], runes[j] = runes[j], runes[i]
    }

    return string(runes), nil
}

func NewPlugin() plugins.Plugin {
    return NewReverseFilterPlugin()
}
```

Usage in templates:

```yaml
tasks:
  - name: Reverse strings
    debug:
      msg: "{{ 'hello' | reverse }}"
      # Output: olleh
```

### Example 3: Creating a Callback Plugin for Metrics

For comprehensive callback plugin development, examples, and advanced patterns, see [CALLBACKS_GUIDE.md](CALLBACKS_GUIDE.md).

See `examples/plugins/callback_metrics.go` for a complete working example of a metrics collection callback plugin.

Quick callback example:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type LoggingCallback struct {
    *plugins.BaseCallbackPlugin
}

func NewPlugin() plugins.Plugin {
    return &LoggingCallback{
        BaseCallbackPlugin: plugins.NewBaseCallbackPlugin(
            "logger",
            "1.0.0",
            "Logs execution events",
        ),
    }
}

func (l *LoggingCallback) OnPlaybookStart(ctx context.Context,
    playbook *types.Playbook) error {
    fmt.Printf("\n=== Playbook Started: %s ===\n", playbook.Name)
    return nil
}

func (l *LoggingCallback) OnPlaybookEnd(ctx context.Context,
    playbook *types.Playbook, success bool, duration time.Duration) error {
    status := "SUCCESS"
    if !success {
        status = "FAILED"
    }
    fmt.Printf("=== Playbook %s: %s (%s) ===\n\n", playbook.Name, status, duration)
    return nil
}

func (l *LoggingCallback) OnTaskStart(ctx context.Context, task *types.Task,
    host types.Host) error {
    fmt.Printf("  [%s] %s\n", host.Name, task.Name)
    return nil
}

func (l *LoggingCallback) OnTaskEnd(ctx context.Context, task *types.Task,
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

// ... implement other callback methods
```

### Example 4: Creating a Module Plugin

See `examples/plugins/module_hello.go` for a complete example.

## Building Plugin Binaries

To build a plugin as a Go plugin (.so file):

```bash
# Build module plugin
go build -buildmode=plugin -o module_hello.so examples/plugins/module_hello.go

# Build callback plugin
go build -buildmode=plugin -o callback_metrics.so examples/plugins/callback_metrics.go
```

**Note:** Go plugins have limitations:

- Only work on Linux and macOS
- Must be built with the same Go version as the main binary
- Cannot be unloaded once loaded

For production use, consider using the in-memory loader with compiled-in plugins.

## Best Practices

1. **Use In-Memory Loader for Built-in Plugins**
   - Compile plugins directly into the binary
   - No runtime loading overhead
   - Better portability

2. **Use Go Plugin Loader for External Plugins**
   - Allow users to add custom plugins
   - Keep core binary small
   - Enable plugin marketplace

3. **Always Initialize Plugins**
   - Call `Initialize()` with configuration
   - Handle initialization errors
   - Clean up with `Cleanup()` when done

4. **Thread Safety**
   - All plugin operations are thread-safe
   - Use proper locking in custom plugins
   - Avoid shared mutable state

5. **Error Handling**
   - Return descriptive errors
   - Log warnings for non-critical failures
   - Don't crash on plugin errors

6. **Testing**
   - Test plugins in isolation
   - Use mock hosts and contexts
   - Benchmark performance-critical plugins

## Performance Considerations

- **Filter Plugins**: 50-100 ns/op for simple filters
- **Callback Plugins**: ~10 ns/op for event dispatch
- **Module Plugins**: Depends on implementation
- **Plugin Registration**: ~150 ns/op
- **Plugin Lookup**: ~10 ns/op (zero allocations)

## Troubleshooting

### Plugin Not Loading

1. Check plugin is enabled in configuration
2. Verify plugin path is correct
3. Check Go version compatibility (for .so plugins)
4. Review initialization errors in logs

### Filter Not Available in Templates

1. Ensure plugin manager is set on template engine
2. Verify filter plugin is registered
3. Check filter name matches usage
4. Review plugin initialization logs

### Callback Not Firing

1. Verify callback plugin is registered with callback manager
2. Check plugin is enabled in configuration
3. Review callback registration errors
4. Ensure core engine has plugin manager set

## Documentation & References

### Comprehensive Guides

- **[FILTERS_GUIDE.md](FILTERS_GUIDE.md)** - Complete guide to Onigirazu filters
  - Built-in filters with detailed examples
  - Creating custom filter plugins
  - Advanced patterns and best practices
  - Troubleshooting filter issues

- **[CALLBACKS_GUIDE.md](CALLBACKS_GUIDE.md)** - Complete guide to Onigirazu callbacks
  - All 7 callback events explained
  - Callback plugin development
  - Real-world examples (metrics, logging, notifications)
  - Advanced patterns and best practices
  - Troubleshooting callback issues

### API Reference

Core plugin interfaces and implementations:

- `internal/plugins/interface.go` - Core Plugin interfaces
  - `Plugin` - Base interface for all plugins
  - `CallbackPlugin` - Callback plugin interface (7 event hooks)
  - `FilterPlugin` - Filter plugin interface
  - `ModulePlugin` - Module plugin interface
  - `InventoryPlugin` - Inventory plugin interface
  - `FilterFunc` - Filter function signature

- `internal/plugins/callback.go` - Callback plugin implementations
  - `BaseCallbackPlugin` - Base implementation with no-op methods
  - `CallbackManager` - Event dispatcher for callbacks

- `internal/plugins/filter.go` - Filter plugin implementations
  - `BaseFilterPlugin` - Base implementation for filter plugins
  - `BuiltinFiltersPlugin` - 9 built-in filter implementations

- `internal/plugins/manager.go` - Plugin manager
- `internal/plugins/config.go` - Configuration loading

### Examples

Complete working examples are available in:

- `examples/plugins/callback_metrics.go` - Full metrics collection callback plugin
  - Thread-safe implementation with sync.RWMutex
  - Collects execution metrics and performance data
  - Demonstrates all 7 callback event handlers

- `examples/plugins/module_hello.go` - Basic module plugin example

- `examples/10-plugins-demo.yml` - Example playbook using plugins

- `examples/plugins/plugins.yml` - Plugin configuration example

### Related Documentation

- [MODULE_DEVELOPMENT_GUIDE.md](MODULE_DEVELOPMENT_GUIDE.md) - Module plugin development
- [PLUGIN_INTEGRATION.md](PLUGIN_INTEGRATION.md) - This file
- [docs/plugins/README.md](docs/plugins/README.md) - Plugin development overview
