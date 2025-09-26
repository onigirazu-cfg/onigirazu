# Plugin Development Guide

This guide covers the complete process of developing plugins and extending Onigirazu functionality.

## 📋 Table of Contents

- [Plugin System Overview](#plugin-system-overview)
- [Plugin Types](#plugin-types)
- [Development Environment](#development-environment)
- [Creating Your First Plugin](#creating-your-first-plugin)
- [Module Development](#module-development)
- [Advanced Plugin Features](#advanced-plugin-features)
- [Testing Plugins](#testing-plugins)
- [Plugin Distribution](#plugin-distribution)
- [Best Practices](#best-practices)

## 🔌 Plugin System Overview

Onigirazu's plugin system allows you to extend functionality through:

- **Modules**: Task execution plugins (like Ansible modules)
- **Filters**: Data transformation plugins
- **Lookups**: Data retrieval plugins
- **Callbacks**: Event handling plugins
- **Connections**: Transport layer plugins
- **Inventory**: Dynamic inventory plugins

### Plugin Architecture

```go
// Core plugin interface
type Plugin interface {
    Name() string
    Version() string
    Description() string
    Initialize(config map[string]interface{}) error
    Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
    Cleanup() error
    ValidateArgs(args map[string]interface{}) error
    GetSchema() map[string]interface{}
}

// Module-specific interface
type Module interface {
    Plugin
    Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
    Validate(args map[string]interface{}) error
    GetName() string
    GetDescription() string
}
```

## 🎯 Plugin Types

### 1. Modules

Execute tasks on target hosts:

```go
type DatabaseModule struct {
    name        string
    description string
}

func (m *DatabaseModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Implementation
}
```

### 2. Filters

Transform data in templates:

```go
type UppercaseFilter struct{}

func (f *UppercaseFilter) Apply(input interface{}, args ...interface{}) (interface{}, error) {
    if str, ok := input.(string); ok {
        return strings.ToUpper(str), nil
    }
    return input, nil
}
```

### 3. Lookups

Retrieve external data:

```go
type EnvLookup struct{}

func (l *EnvLookup) Lookup(key string, args map[string]interface{}) (interface{}, error) {
    return os.Getenv(key), nil
}
```

### 4. Callbacks

Handle execution events:

```go
type LoggingCallback struct {
    logger *log.Logger
}

func (c *LoggingCallback) OnTaskStart(task types.Task, host types.Host) {
    c.logger.Printf("Starting task %s on %s", task.Name, host.Name)
}
```

## 🛠️ Development Environment

### Prerequisites

```bash
# Go 1.19 or later
go version

# Clone Onigirazu repository
git clone https://github.com/your-org/onigirazu.git
cd onigirazu

# Install dependencies
go mod download
```

### Project Structure

```
my-plugin/
├── go.mod
├── go.sum
├── main.go              # Plugin entry point
├── module.go            # Module implementation
├── schema.json          # Argument schema
├── README.md            # Plugin documentation
├── examples/            # Usage examples
└── tests/              # Test files
```

## 🚀 Creating Your First Plugin

Let's create a simple "hello" module:

### 1. Initialize Plugin Project

```bash
mkdir onigirazu-hello-plugin
cd onigirazu-hello-plugin
go mod init github.com/your-username/onigirazu-hello-plugin
```

### 2. Create Module Implementation

```go
// module.go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/your-org/onigirazu/pkg/types"
)

type HelloModule struct {
    name        string
    description string
}

func NewHelloModule() *HelloModule {
    return &HelloModule{
        name:        "hello",
        description: "A simple greeting module",
    }
}

func (m *HelloModule) Name() string {
    return m.name
}

func (m *HelloModule) Version() string {
    return "1.0.0"
}

func (m *HelloModule) Description() string {
    return m.description
}

func (m *HelloModule) GetName() string {
    return m.name
}

func (m *HelloModule) GetDescription() string {
    return m.description
}

func (m *HelloModule) Initialize(config map[string]interface{}) error {
    // Initialize module with configuration
    return nil
}

func (m *HelloModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    result := types.TaskResult{
        TaskName:  "hello",
        Host:      host.Name,
        Module:    m.name,
        Success:   true,
        Changed:   false,
        Timestamp: time.Now(),
        Output:    make(map[string]interface{}),
    }

    // Get message from arguments
    message, ok := args["message"].(string)
    if !ok {
        message = "Hello, World!"
    }

    // Get target name
    target, ok := args["target"].(string)
    if !ok {
        target = "World"
    }

    greeting := fmt.Sprintf("%s, %s!", message, target)

    result.Output["greeting"] = greeting
    result.Output["message"] = message
    result.Output["target"] = target

    return result, nil
}

func (m *HelloModule) Validate(args map[string]interface{}) error {
    // Validate message if provided
    if message, exists := args["message"]; exists {
        if _, ok := message.(string); !ok {
            return fmt.Errorf("message must be a string")
        }
    }

    // Validate target if provided
    if target, exists := args["target"]; exists {
        if _, ok := target.(string); !ok {
            return fmt.Errorf("target must be a string")
        }
    }

    return nil
}

func (m *HelloModule) ValidateArgs(args map[string]interface{}) error {
    return m.Validate(args)
}

func (m *HelloModule) GetSchema() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "message": map[string]interface{}{
                "type":        "string",
                "description": "The greeting message",
                "default":     "Hello",
            },
            "target": map[string]interface{}{
                "type":        "string",
                "description": "The target of the greeting",
                "default":     "World",
            },
        },
        "additionalProperties": false,
    }
}

func (m *HelloModule) Cleanup() error {
    // Cleanup resources if needed
    return nil
}
```

### 3. Create Plugin Entry Point

```go
// main.go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"

    "github.com/your-org/onigirazu/pkg/types"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Fprintf(os.Stderr, "Usage: %s <command>\n", os.Args[0])
        os.Exit(1)
    }

    module := NewHelloModule()
    command := os.Args[1]

    switch command {
    case "info":
        info := map[string]interface{}{
            "name":        module.Name(),
            "version":     module.Version(),
            "description": module.Description(),
            "schema":      module.GetSchema(),
        }
        json.NewEncoder(os.Stdout).Encode(info)

    case "execute":
        if len(os.Args) < 4 {
            fmt.Fprintf(os.Stderr, "Usage: %s execute <host_json> <args_json>\n", os.Args[0])
            os.Exit(1)
        }

        // Parse host
        var host types.Host
        if err := json.Unmarshal([]byte(os.Args[2]), &host); err != nil {
            fmt.Fprintf(os.Stderr, "Error parsing host: %v\n", err)
            os.Exit(1)
        }

        // Parse arguments
        var args map[string]interface{}
        if err := json.Unmarshal([]byte(os.Args[3]), &args); err != nil {
            fmt.Fprintf(os.Stderr, "Error parsing args: %v\n", err)
            os.Exit(1)
        }

        // Validate arguments
        if err := module.Validate(args); err != nil {
            fmt.Fprintf(os.Stderr, "Validation error: %v\n", err)
            os.Exit(1)
        }

        // Execute module
        result, err := module.Execute(context.Background(), host, args)
        if err != nil {
            result.Success = false
            result.Failed = true
            result.Error = err.Error()
        }

        // Output result
        json.NewEncoder(os.Stdout).Encode(result)

    default:
        fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
        os.Exit(1)
    }
}
```

### 4. Build Plugin

```bash
go build -o onigirazu-hello-plugin
```

### 5. Test Plugin

```bash
# Test plugin info
./onigirazu-hello-plugin info

# Test plugin execution
./onigirazu-hello-plugin execute '{"name":"localhost","address":"127.0.0.1"}' '{"message":"Hi","target":"Onigirazu"}'
```

## 🔧 Module Development

### Advanced Module Example

```go
// file_module.go
package main

import (
    "context"
    "crypto/md5"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"

    "github.com/your-org/onigirazu/pkg/types"
)

type FileModule struct {
    name        string
    description string
}

func NewFileModule() *FileModule {
    return &FileModule{
        name:        "file",
        description: "Manage files and directories",
    }
}

func (m *FileModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    result := types.TaskResult{
        TaskName:  "file",
        Host:      host.Name,
        Module:    m.name,
        Success:   true,
        Changed:   false,
        Timestamp: time.Now(),
        Output:    make(map[string]interface{}),
    }

    path, ok := args["path"].(string)
    if !ok {
        return result, fmt.Errorf("path is required")
    }

    state, ok := args["state"].(string)
    if !ok {
        state = "file"
    }

    switch state {
    case "file":
        return m.ensureFile(path, args, result)
    case "directory":
        return m.ensureDirectory(path, args, result)
    case "absent":
        return m.ensureAbsent(path, result)
    case "touch":
        return m.touchFile(path, result)
    default:
        return result, fmt.Errorf("invalid state: %s", state)
    }
}

func (m *FileModule) ensureFile(path string, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
    // Check if file exists
    info, err := os.Stat(path)
    if os.IsNotExist(err) {
        // Create file
        if content, ok := args["content"].(string); ok {
            if err := os.WriteFile(path, []byte(content), 0644); err != nil {
                result.Success = false
                result.Failed = true
                result.Error = err.Error()
                return result, err
            }
            result.Changed = true
            result.Output["created"] = true
        } else {
            // Create empty file
            file, err := os.Create(path)
            if err != nil {
                result.Success = false
                result.Failed = true
                result.Error = err.Error()
                return result, err
            }
            file.Close()
            result.Changed = true
            result.Output["created"] = true
        }
    } else if err != nil {
        result.Success = false
        result.Failed = true
        result.Error = err.Error()
        return result, err
    } else {
        // File exists, check if content needs updating
        if content, ok := args["content"].(string); ok {
            currentContent, err := os.ReadFile(path)
            if err != nil {
                result.Success = false
                result.Failed = true
                result.Error = err.Error()
                return result, err
            }

            if string(currentContent) != content {
                if err := os.WriteFile(path, []byte(content), info.Mode()); err != nil {
                    result.Success = false
                    result.Failed = true
                    result.Error = err.Error()
                    return result, err
                }
                result.Changed = true
                result.Output["updated"] = true
            }
        }
    }

    // Set file permissions if specified
    if mode, ok := args["mode"]; ok {
        if modeStr, ok := mode.(string); ok {
            if err := m.setFileMode(path, modeStr); err != nil {
                result.Success = false
                result.Failed = true
                result.Error = err.Error()
                return result, err
            }
            result.Changed = true
        }
    }

    // Get file info for output
    info, _ = os.Stat(path)
    result.Output["path"] = path
    result.Output["size"] = info.Size()
    result.Output["mode"] = info.Mode().String()
    result.Output["modified"] = info.ModTime()

    return result, nil
}

func (m *FileModule) ensureDirectory(path string, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
    // Check if directory exists
    info, err := os.Stat(path)
    if os.IsNotExist(err) {
        // Create directory
        mode := os.FileMode(0755)
        if modeArg, ok := args["mode"].(string); ok {
            if parsedMode, err := m.parseFileMode(modeArg); err == nil {
                mode = parsedMode
            }
        }

        if err := os.MkdirAll(path, mode); err != nil {
            result.Success = false
            result.Failed = true
            result.Error = err.Error()
            return result, err
        }
        result.Changed = true
        result.Output["created"] = true
    } else if err != nil {
        result.Success = false
        result.Failed = true
        result.Error = err.Error()
        return result, err
    } else if !info.IsDir() {
        result.Success = false
        result.Failed = true
        result.Error = fmt.Sprintf("%s exists but is not a directory", path)
        return result, fmt.Errorf("%s exists but is not a directory", path)
    }

    result.Output["path"] = path
    result.Output["state"] = "directory"

    return result, nil
}

func (m *FileModule) ensureAbsent(path string, result types.TaskResult) (types.TaskResult, error) {
    // Check if path exists
    _, err := os.Stat(path)
    if os.IsNotExist(err) {
        // Already absent
        result.Output["path"] = path
        result.Output["state"] = "absent"
        return result, nil
    } else if err != nil {
        result.Success = false
        result.Failed = true
        result.Error = err.Error()
        return result, err
    }

    // Remove path
    if err := os.RemoveAll(path); err != nil {
        result.Success = false
        result.Failed = true
        result.Error = err.Error()
        return result, err
    }

    result.Changed = true
    result.Output["path"] = path
    result.Output["state"] = "absent"
    result.Output["removed"] = true

    return result, nil
}

func (m *FileModule) touchFile(path string, result types.TaskResult) (types.TaskResult, error) {
    // Create file if it doesn't exist
    if _, err := os.Stat(path); os.IsNotExist(err) {
        file, err := os.Create(path)
        if err != nil {
            result.Success = false
            result.Failed = true
            result.Error = err.Error()
            return result, err
        }
        file.Close()
        result.Changed = true
        result.Output["created"] = true
    }

    // Update access and modification times
    now := time.Now()
    if err := os.Chtimes(path, now, now); err != nil {
        result.Success = false
        result.Failed = true
        result.Error = err.Error()
        return result, err
    }

    result.Output["path"] = path
    result.Output["touched"] = true

    return result, nil
}

func (m *FileModule) setFileMode(path, modeStr string) error {
    mode, err := m.parseFileMode(modeStr)
    if err != nil {
        return err
    }
    return os.Chmod(path, mode)
}

func (m *FileModule) parseFileMode(modeStr string) (os.FileMode, error) {
    // Simple octal mode parsing
    var mode os.FileMode
    if _, err := fmt.Sscanf(modeStr, "%o", &mode); err != nil {
        return 0, fmt.Errorf("invalid mode: %s", modeStr)
    }
    return mode, nil
}

func (m *FileModule) Validate(args map[string]interface{}) error {
    // Validate required path
    if _, ok := args["path"]; !ok {
        return fmt.Errorf("path is required")
    }

    // Validate state
    if state, ok := args["state"]; ok {
        stateStr, ok := state.(string)
        if !ok {
            return fmt.Errorf("state must be a string")
        }
        validStates := []string{"file", "directory", "absent", "touch"}
        valid := false
        for _, validState := range validStates {
            if stateStr == validState {
                valid = true
                break
            }
        }
        if !valid {
            return fmt.Errorf("invalid state: %s", stateStr)
        }
    }

    return nil
}

func (m *FileModule) GetSchema() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "path": map[string]interface{}{
                "type":        "string",
                "description": "Path to the file or directory",
                "required":    true,
            },
            "state": map[string]interface{}{
                "type":        "string",
                "description": "Desired state of the file",
                "enum":        []string{"file", "directory", "absent", "touch"},
                "default":     "file",
            },
            "content": map[string]interface{}{
                "type":        "string",
                "description": "Content of the file (for state=file)",
            },
            "mode": map[string]interface{}{
                "type":        "string",
                "description": "File permissions in octal format",
                "pattern":     "^[0-7]{3,4}$",
            },
        },
        "required": ["path"],
        "additionalProperties": false,
    }
}

// Implement other required methods...
func (m *FileModule) Name() string { return m.name }
func (m *FileModule) Version() string { return "1.0.0" }
func (m *FileModule) Description() string { return m.description }
func (m *FileModule) GetName() string { return m.name }
func (m *FileModule) GetDescription() string { return m.description }
func (m *FileModule) Initialize(config map[string]interface{}) error { return nil }
func (m *FileModule) ValidateArgs(args map[string]interface{}) error { return m.Validate(args) }
func (m *FileModule) Cleanup() error { return nil }
```

## 🧪 Testing Plugins

### Unit Tests

```go
// module_test.go
package main

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/your-org/onigirazu/pkg/types"
)

func TestHelloModule(t *testing.T) {
    module := NewHelloModule()

    host := types.Host{
        Name:    "testhost",
        Address: "127.0.0.1",
    }

    tests := []struct {
        name     string
        args     map[string]interface{}
        expected string
    }{
        {
            name:     "default greeting",
            args:     map[string]interface{}{},
            expected: "Hello, World!",
        },
        {
            name: "custom message",
            args: map[string]interface{}{
                "message": "Hi",
                "target":  "Onigirazu",
            },
            expected: "Hi, Onigirazu!",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := module.Execute(context.Background(), host, tt.args)
            if err != nil {
                t.Fatalf("Execute failed: %v", err)
            }

            if !result.Success {
                t.Fatalf("Task failed: %s", result.Error)
            }

            greeting, ok := result.Output["greeting"].(string)
            if !ok {
                t.Fatal("No greeting in output")
            }

            if greeting != tt.expected {
                t.Errorf("Expected %q, got %q", tt.expected, greeting)
            }
        })
    }
}

func TestFileModule(t *testing.T) {
    module := NewFileModule()
    tempDir := t.TempDir()

    host := types.Host{
        Name:    "testhost",
        Address: "127.0.0.1",
    }

    t.Run("create file", func(t *testing.T) {
        filePath := filepath.Join(tempDir, "test.txt")
        args := map[string]interface{}{
            "path":    filePath,
            "state":   "file",
            "content": "Hello, World!",
        }

        result, err := module.Execute(context.Background(), host, args)
        if err != nil {
            t.Fatalf("Execute failed: %v", err)
        }

        if !result.Success {
            t.Fatalf("Task failed: %s", result.Error)
        }

        if !result.Changed {
            t.Fatal("Expected task to be changed")
        }

        // Verify file exists and has correct content
        content, err := os.ReadFile(filePath)
        if err != nil {
            t.Fatalf("Failed to read file: %v", err)
        }

        if string(content) != "Hello, World!" {
            t.Errorf("Expected content %q, got %q", "Hello, World!", string(content))
        }
    })

    t.Run("create directory", func(t *testing.T) {
        dirPath := filepath.Join(tempDir, "testdir")
        args := map[string]interface{}{
            "path":  dirPath,
            "state": "directory",
        }

        result, err := module.Execute(context.Background(), host, args)
        if err != nil {
            t.Fatalf("Execute failed: %v", err)
        }

        if !result.Success {
            t.Fatalf("Task failed: %s", result.Error)
        }

        if !result.Changed {
            t.Fatal("Expected task to be changed")
        }

        // Verify directory exists
        info, err := os.Stat(dirPath)
        if err != nil {
            t.Fatalf("Directory not created: %v", err)
        }

        if !info.IsDir() {
            t.Fatal("Path is not a directory")
        }
    })
}
```

### Integration Tests

```go
// integration_test.go
package main

import (
    "context"
    "encoding/json"
    "os/exec"
    "testing"

    "github.com/your-org/onigirazu/pkg/types"
)

func TestPluginIntegration(t *testing.T) {
    // Build plugin
    cmd := exec.Command("go", "build", "-o", "test-plugin")
    if err := cmd.Run(); err != nil {
        t.Fatalf("Failed to build plugin: %v", err)
    }

    // Test plugin info
    cmd = exec.Command("./test-plugin", "info")
    output, err := cmd.Output()
    if err != nil {
        t.Fatalf("Failed to get plugin info: %v", err)
    }

    var info map[string]interface{}
    if err := json.Unmarshal(output, &info); err != nil {
        t.Fatalf("Failed to parse plugin info: %v", err)
    }

    if info["name"] != "hello" {
        t.Errorf("Expected name 'hello', got %v", info["name"])
    }

    // Test plugin execution
    host := types.Host{Name: "testhost", Address: "127.0.0.1"}
    args := map[string]interface{}{"message": "Hi", "target": "Test"}

    hostJSON, _ := json.Marshal(host)
    argsJSON, _ := json.Marshal(args)

    cmd = exec.Command("./test-plugin", "execute", string(hostJSON), string(argsJSON))
    output, err = cmd.Output()
    if err != nil {
        t.Fatalf("Failed to execute plugin: %v", err)
    }

    var result types.TaskResult
    if err := json.Unmarshal(output, &result); err != nil {
        t.Fatalf("Failed to parse result: %v", err)
    }

    if !result.Success {
        t.Fatalf("Task failed: %s", result.Error)
    }

    if result.Output["greeting"] != "Hi, Test!" {
        t.Errorf("Expected greeting 'Hi, Test!', got %v", result.Output["greeting"])
    }
}
```

## 📦 Plugin Distribution

### Plugin Manifest

```json
{
  "name": "onigirazu-hello-plugin",
  "version": "1.0.0",
  "description": "A simple greeting plugin for Onigirazu",
  "author": "Your Name <your.email@example.com>",
  "license": "MIT",
  "homepage": "https://github.com/your-username/onigirazu-hello-plugin",
  "repository": {
    "type": "git",
    "url": "https://github.com/your-username/onigirazu-hello-plugin.git"
  },
  "keywords": ["onigirazu", "plugin", "hello", "greeting"],
  "modules": [
    {
      "name": "hello",
      "description": "Simple greeting module",
      "binary": "onigirazu-hello-plugin"
    }
  ],
  "requirements": {
    "onigirazu": ">=1.0.0",
    "go": ">=1.19"
  },
  "platforms": ["linux", "darwin", "windows"],
  "architectures": ["amd64", "arm64"]
}
```

### Build Script

```bash
#!/bin/bash
# build.sh

set -e

PLUGIN_NAME="onigirazu-hello-plugin"
VERSION="1.0.0"

# Build for multiple platforms
PLATFORMS=("linux/amd64" "darwin/amd64" "darwin/arm64" "windows/amd64")

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r -a array <<< "$platform"
    GOOS="${array[0]}"
    GOARCH="${array[1]}"

    output_name="${PLUGIN_NAME}-${VERSION}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo "Building for $GOOS/$GOARCH..."
    GOOS="$GOOS" GOARCH="$GOARCH" go build -o "dist/$output_name" .
done

echo "Build complete!"
```

## 🎯 Best Practices

### 1. Error Handling

```go
func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    result := types.TaskResult{
        TaskName:  "my_module",
        Host:      host.Name,
        Module:    m.name,
        Success:   true,
        Timestamp: time.Now(),
        Output:    make(map[string]interface{}),
    }

    // Always handle context cancellation
    select {
    case <-ctx.Done():
        result.Success = false
        result.Failed = true
        result.Error = "Task cancelled"
        return result, ctx.Err()
    default:
    }

    // Validate arguments early
    if err := m.Validate(args); err != nil {
        result.Success = false
        result.Failed = true
        result.Error = fmt.Sprintf("Validation failed: %v", err)
        return result, err
    }

    // Handle operations with proper error handling
    if err := m.performOperation(args); err != nil {
        result.Success = false
        result.Failed = true
        result.Error = err.Error()
        return result, err
    }

    return result, nil
}
```

### 2. Logging

```go
import "log/slog"

type MyModule struct {
    logger *slog.Logger
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    m.logger.Info("Starting task execution",
        "module", m.name,
        "host", host.Name,
        "args", args)

    // ... implementation

    m.logger.Info("Task completed successfully",
        "module", m.name,
        "host", host.Name,
        "changed", result.Changed)

    return result, nil
}
```

### 3. Configuration

```go
type ModuleConfig struct {
    Timeout     time.Duration `json:"timeout"`
    RetryCount  int          `json:"retry_count"`
    RetryDelay  time.Duration `json:"retry_delay"`
    Debug       bool         `json:"debug"`
}

func (m *MyModule) Initialize(config map[string]interface{}) error {
    var moduleConfig ModuleConfig

    // Set defaults
    moduleConfig.Timeout = 30 * time.Second
    moduleConfig.RetryCount = 3
    moduleConfig.RetryDelay = 1 * time.Second

    // Parse configuration
    if configData, err := json.Marshal(config); err == nil {
        json.Unmarshal(configData, &moduleConfig)
    }

    m.config = moduleConfig
    return nil
}
```

### 4. Testing

```go
func TestModuleWithMocks(t *testing.T) {
    // Use dependency injection for testability
    module := &MyModule{
        client: &mockClient{},
        logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
    }

    // Test various scenarios
    testCases := []struct {
        name        string
        args        map[string]interface{}
        expectError bool
        expectChanged bool
    }{
        // ... test cases
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // ... test implementation
        })
    }
}
```

### 5. Documentation

```go
// MyModule implements advanced file operations with backup and rollback capabilities.
//
// Supported operations:
//   - create: Create new files with specified content
//   - update: Update existing files with change detection
//   - backup: Create timestamped backups before modifications
//   - rollback: Restore from previous backups
//
// Example usage:
//   - name: "Create configuration file"
//     module: "my_module"
//     args:
//       path: "/etc/myapp/config.yml"
//       content: "{{ config_template }}"
//       backup: true
//       mode: "0644"
type MyModule struct {
    // ... implementation
}
```

This comprehensive guide provides everything needed to develop, test, and distribute plugins for Onigirazu. The modular architecture makes it easy to extend functionality while maintaining security and performance standards.
