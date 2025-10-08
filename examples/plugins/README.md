# Onigirazu Plugin Examples

This directory contains example plugins for Onigirazu.

## Plugin Types

Onigirazu supports four types of plugins:

1. **Module Plugins** - Add new modules for task execution
2. **Callback Plugins** - Hook into execution events
3. **Inventory Plugins** - Provide dynamic inventory sources
4. **Filter Plugins** - Add custom template filters

## Example Plugins

### Module Plugin: Hello World

A simple module plugin that prints a hello message.

**File:** `module_hello/hello.go`

```go
package main

import (
 "context"
 "time"

 "github.com/onigirazu-cfg/onigirazu/internal/plugins"
 "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type HelloModule struct {
 *plugins.BaseModulePlugin
}

func NewPlugin() plugins.Plugin {
 module := &HelloModule{
  BaseModulePlugin: plugins.NewBaseModulePlugin("hello", "1.0.0", "Hello world module"),
 }
 module.AddRequiredArg("message")
 return module
}

func (m *HelloModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
 startTime := time.Now()

 message := plugins.GetStringArg(args, "message", "Hello, World!")

 result := types.TaskResult{
  TaskName:  plugins.GetStringArg(args, "name", "hello"),
  Host:      host.Name,
  Module:    m.GetName(),
  Timestamp: startTime,
  Success:   true,
  Changed:   false,
  Output: map[string]interface{}{
   "message": message,
  },
  Duration: time.Since(startTime),
 }

 return result, nil
}
```

**Usage in playbook:**

```yaml
- name: Say hello
  hello:
    message: "Hello from plugin!"
```

### Callback Plugin: Logger

A callback plugin that logs execution events to a file.

**File:** `callback_logger/logger.go`

```go
package main

import (
 "context"
 "fmt"
 "os"
 "time"

 "github.com/onigirazu-cfg/onigirazu/internal/plugins"
 "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type LoggerCallback struct {
 *plugins.BaseCallbackPlugin
 logFile *os.File
}

func NewPlugin() plugins.Plugin {
 return &LoggerCallback{
  BaseCallbackPlugin: plugins.NewBaseCallbackPlugin("logger", "1.0.0", "File logger callback"),
 }
}

func (c *LoggerCallback) Initialize(ctx context.Context, config map[string]interface{}) error {
 logPath := plugins.GetStringArg(config, "log_path", "/tmp/onigirazu.log")

 file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
 if err != nil {
  return err
 }

 c.logFile = file
 return nil
}

func (c *LoggerCallback) Cleanup(ctx context.Context) error {
 if c.logFile != nil {
  return c.logFile.Close()
 }
 return nil
}

func (c *LoggerCallback) OnTaskStart(ctx context.Context, task *types.Task, host types.Host) error {
 msg := fmt.Sprintf("[%s] Task started: %s on %s\n", time.Now().Format(time.RFC3339), task.Name, host.Name)
 _, err := c.logFile.WriteString(msg)
 return err
}

func (c *LoggerCallback) OnTaskEnd(ctx context.Context, task *types.Task, host types.Host, result types.TaskResult) error {
 status := "SUCCESS"
 if !result.Success {
  status = "FAILED"
 }
 msg := fmt.Sprintf("[%s] Task ended: %s on %s - %s\n", time.Now().Format(time.RFC3339), task.Name, host.Name, status)
 _, err := c.logFile.WriteString(msg)
 return err
}
```

### Filter Plugin: Custom Filters

A filter plugin that adds custom template filters.

**File:** `filter_custom/custom.go`

```go
package main

import (
 "fmt"
 "strings"

 "github.com/onigirazu-cfg/onigirazu/internal/plugins"
)

type CustomFilters struct {
 *plugins.BaseFilterPlugin
}

func NewPlugin() plugins.Plugin {
 plugin := &CustomFilters{
  BaseFilterPlugin: plugins.NewBaseFilterPlugin("custom", "1.0.0", "Custom filter functions"),
 }

 // Register custom filters
 plugin.AddFilter("reverse", reverseFilter)
 plugin.AddFilter("capitalize", capitalizeFilter)

 return plugin
}

func reverseFilter(input interface{}, args ...interface{}) (interface{}, error) {
 str, ok := input.(string)
 if !ok {
  return nil, fmt.Errorf("reverse filter expects string input")
 }

 runes := []rune(str)
 for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
  runes[i], runes[j] = runes[j], runes[i]
 }

 return string(runes), nil
}

func capitalizeFilter(input interface{}, args ...interface{}) (interface{}, error) {
 str, ok := input.(string)
 if !ok {
  return nil, fmt.Errorf("capitalize filter expects string input")
 }

 if len(str) == 0 {
  return str, nil
 }

 return strings.ToUpper(str[:1]) + strings.ToLower(str[1:]), nil
}
```

**Usage in template:**

```yaml
vars:
  name: "john doe"

tasks:
  - name: Use custom filters
    debug:
      msg: "{{ name | capitalize }}"  # Output: "John doe"
```

## Building Plugins

To build a plugin as a shared library:

```bash
go build -buildmode=plugin -o hello.so module_hello/hello.go
```

## Loading Plugins

Plugins can be loaded in several ways:

### 1. Configuration File

```yaml
plugins:
  directory: /path/to/plugins
  modules:
    - hello.so
  callbacks:
    - logger.so
  filters:
    - custom.so
```

### 2. Command Line

```bash
onigirazu --plugin-dir /path/to/plugins playbook.yml
```

### 3. Programmatically

```go
manager := plugins.NewManager(plugins.NewGoPluginLoader())
err := manager.LoadPlugin(ctx, "/path/to/hello.so")
```

## Plugin Development Guidelines

1. **Always implement the Plugin interface** - All plugins must implement the base Plugin interface
2. **Use base implementations** - Extend BaseModulePlugin, BaseCallbackPlugin, etc.
3. **Export NewPlugin function** - Must return a Plugin interface
4. **Handle errors gracefully** - Return meaningful error messages
5. **Clean up resources** - Implement Cleanup() method properly
6. **Document your plugin** - Add clear descriptions and examples
7. **Test thoroughly** - Write unit tests for your plugin

## Plugin API Reference

See the main documentation for detailed API reference:

- [Plugin Interface](../../internal/plugins/interface.go)
- [Module Plugins](../../internal/plugins/module.go)
- [Callback Plugins](../../internal/plugins/callback.go)
- [Inventory Plugins](../../internal/plugins/inventory.go)
- [Filter Plugins](../../internal/plugins/filter.go)
