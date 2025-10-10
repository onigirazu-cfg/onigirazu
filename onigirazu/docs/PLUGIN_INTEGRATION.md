# Plugin System Integration Guide

This document describes how the plugin system is integrated with Onigirazu's core engine and how to use it.

## Overview

The plugin system is fully integrated with:

- **Template Engine** - Filter plugins for custom template filters
- **Core Engine** - Callback plugins for execution event hooks
- **Module Registry** - Module plugins for custom task modules
- **Inventory Manager** - Inventory plugins for dynamic inventory sources

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

- `upper` - Convert to uppercase
- `lower` - Convert to lowercase
- `title` - Convert to title case
- `trim` - Remove whitespace
- `replace` - Replace substring
- `default` - Default value if empty
- `length` - Get length
- `join` - Join array to string
- `split` - Split string to array

**Usage in Templates:**

```yaml
vars:
  app_name: "myapp"
  users: ["alice", "bob"]

tasks:
  - name: Use filters
    debug:
      msg: "{{ app_name | upper }} has {{ users | length }} users: {{ users | join(', ') }}"
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

- `OnPlaybookStart` - When playbook execution starts
- `OnPlaybookEnd` - When playbook execution ends
- `OnPlayStart` - When play execution starts
- `OnPlayEnd` - When play execution ends
- `OnTaskStart` - When task execution starts
- `OnTaskEnd` - When task execution ends
- `OnTaskRetry` - When task is retried

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

```go
package main

import (
    "context"
    "strings"

    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
)

type ReverseFilterPlugin struct {
    plugins.BaseFilterPlugin
}

func NewReverseFilterPlugin() *ReverseFilterPlugin {
    plugin := &ReverseFilterPlugin{}
    plugin.BaseFilterPlugin = *plugins.NewBaseFilterPlugin(
        "reverse_filter",
        "1.0.0",
        "Reverse string filter",
    )

    // Register filter
    plugin.RegisterFilter("reverse", plugin.reverseFilter)

    return plugin
}

func (p *ReverseFilterPlugin) reverseFilter(ctx context.Context, args ...interface{}) (interface{}, error) {
    if len(args) == 0 {
        return "", nil
    }

    str, ok := args[0].(string)
    if !ok {
        return args[0], nil
    }

    runes := []rune(str)
    for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
        runes[i], runes[j] = runes[j], runes[i]
    }

    return string(runes), nil
}
```

### Example 3: Creating a Callback Plugin for Metrics

See `examples/plugins/callback_metrics.go` for a complete example.

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

## API Reference

See the following files for detailed API documentation:

- `internal/plugins/interface.go` - Plugin interfaces
- `internal/plugins/manager.go` - Plugin manager
- `internal/plugins/filter.go` - Filter plugin base
- `internal/plugins/callback.go` - Callback plugin base
- `internal/plugins/module.go` - Module plugin base
- `internal/plugins/config.go` - Configuration loading

## Examples

Complete examples are available in:

- `examples/plugins/README.md` - Plugin development guide
- `examples/plugins/module_hello.go` - Module plugin example
- `examples/plugins/callback_metrics.go` - Callback plugin example
- `examples/10-plugins-demo.yml` - Playbook using plugins
- `examples/plugins/plugins.yml` - Plugin configuration example
