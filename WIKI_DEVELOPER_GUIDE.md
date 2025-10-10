# 👨‍💻 Developer Guide

This guide helps developers understand Onigirazu's architecture, contribute to the project, and extend its functionality.

## 📋 Table of Contents

### Development Setup
- [Development Setup](Development-Setup)
- [Architecture](Architecture)
- [Testing](Testing)

### Contributing
- [Contributing](Contributing)
- [Code Style](Code-Style)
- [Pull Request Process](Pull-Request-Process)

### Advanced Topics
- [Plugin Development](Plugin-Development)
- [Module Development](Module-Development)
- [API Development](API-Development)

---

## 🏗️ Architecture Overview

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    CLI Layer                                │
├─────────────────────────────────────────────────────────────┤
│                 Engine Layer                                │
├─────────────────────────────────────────────────────────────┤
│                Module Layer                                 │
├─────────────────────────────────────────────────────────────┤
│                Execution Layer                              │
├─────────────────────────────────────────────────────────────┤
│                Communication Layer                          │
└─────────────────────────────────────────────────────────────┘
```

### Core Components

- **CLI Layer** - Command-line interface and argument parsing
- **Engine Layer** - Playbook execution and task orchestration
- **Module Layer** - Built-in modules and plugin system
- **Execution Layer** - Task execution and parallel processing
- **Communication Layer** - SSH communication and connection pooling

---

## 🔧 Development Setup

### Prerequisites

- **Go 1.24.0+** - Latest Go version
- **Git** - Version control
- **Make** - Build automation
- **Docker** - Container testing (optional)

### Quick Setup

```bash
# Clone repository
git clone https://github.com/onigirazu-cfg/onigirazu.git
cd onigirazu

# Install dependencies
go mod download

# Install development tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Build project
go build -o onigirazu cmd/onigirazu/main.go

# Run tests
go test ./...
```

### Development Tools

```bash
# Format code
go fmt ./...

# Format imports
goimports -w .

# Run linter
golangci-lint run

# Run security scan
gosec ./...

# Run all checks
make check
```

---

## 🏗️ Code Organization

### Project Structure

```
onigirazu/
├── cmd/                    # CLI commands
│   └── onigirazu/
│       └── main.go        # Main entry point
├── internal/              # Internal packages
│   ├── cli/               # CLI implementation
│   ├── engine/            # Execution engines
│   ├── modules/           # Built-in modules
│   ├── ssh/               # SSH communication
│   ├── state/             # State management
│   └── ...                # Other internal packages
├── pkg/                   # Public packages
│   ├── types/             # Type definitions
│   └── utils/             # Utility functions
├── docs/                  # Documentation
├── tests/                 # Test files
├── examples/              # Example files
├── go.mod                 # Go module file
├── go.sum                 # Go module checksums
├── Makefile               # Build automation
└── README.md              # Project documentation
```

### Package Structure

```go
// Package organization
package modules

import (
    "context"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Module interface
type Module interface {
    GetName() string
    GetDescription() string
    Execute(ctx context.Context, host Host, args map[string]interface{}) (TaskResult, error)
}

// Base module implementation
type BaseModule struct {
    name        string
    description string
}

func (m *BaseModule) GetName() string {
    return m.name
}

func (m *BaseModule) GetDescription() string {
    return m.description
}
```

---

## 🧪 Testing

### Unit Testing

```go
// internal/modules/package_test.go
package modules

import (
    "context"
    "testing"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestPackageModule_Execute(t *testing.T) {
    module := NewPackageModule()
    host := types.Host{
        Name: "test-host",
        Address: "127.0.0.1",
    }
    
    testCases := []struct {
        name     string
        args     map[string]interface{}
        expected types.TaskResult
    }{
        {
            name: "install package",
            args: map[string]interface{}{
                "name": "nginx",
                "state": "present",
            },
            expected: types.TaskResult{
                Changed: true,
                Output: map[string]interface{}{
                    "package": "nginx",
                    "state": "installed",
                },
            },
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := module.Execute(context.Background(), host, tc.args)
            
            require.NoError(t, err)
            assert.Equal(t, tc.expected.Changed, result.Changed)
            assert.Equal(t, tc.expected.Output, result.Output)
        })
    }
}
```

### Integration Testing

```go
// tests/integration/package_integration_test.go
package integration

import (
    "context"
    "testing"
    "github.com/onigirazu-cfg/onigirazu/internal/modules"
    "github.com/onigirazu-cfg/onigirazu/internal/inventory"
)

func TestPackageIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Set up test environment
    module := modules.NewPackageModule()
    inventoryMgr := inventory.NewManager(parser, logger, cache)
    
    // Load test inventory
    err := inventoryMgr.LoadInventory(context.Background(), "test-inventory.yml")
    require.NoError(t, err)
    
    // Get test host
    hosts, err := inventoryMgr.GetHosts("localhost")
    require.NoError(t, err)
    require.Len(t, hosts, 1)
    
    host := hosts[0]
    args := map[string]interface{}{
        "name": "curl",
        "state": "present",
    }
    
    // Execute module
    result, err := module.Execute(context.Background(), host, args)
    
    require.NoError(t, err)
    assert.True(t, result.Changed)
}
```

### Performance Testing

```go
// internal/modules/package_benchmark_test.go
package modules

import (
    "context"
    "testing"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func BenchmarkPackageModule_Execute(b *testing.B) {
    module := NewPackageModule()
    host := types.Host{
        Name: "test-host",
        Address: "127.0.0.1",
    }
    args := map[string]interface{}{
        "name": "nginx",
        "state": "present",
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := module.Execute(context.Background(), host, args)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

---

## 🔧 Module Development

### Creating New Modules

```go
// internal/modules/your_module.go
package modules

import (
    "context"
    "fmt"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type YourModule struct {
    BaseModule
}

func NewYourModule() *YourModule {
    return &YourModule{
        BaseModule: BaseModule{
            name:        "your_module",
            description: "Your module description",
        },
    }
}

func (m *YourModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Validate arguments
    if err := m.validateArgs(args); err != nil {
        return types.TaskResult{}, err
    }
    
    // Execute module logic
    result, err := m.executeLogic(ctx, host, args)
    if err != nil {
        return types.TaskResult{}, err
    }
    
    return types.TaskResult{
        Changed: result.Changed,
        Output:  result.Output,
        Error:   result.Error,
    }, nil
}

func (m *YourModule) validateArgs(args map[string]interface{}) error {
    // Validate required arguments
    if _, exists := args["name"]; !exists {
        return fmt.Errorf("required argument 'name' is missing")
    }
    
    return nil
}

func (m *YourModule) executeLogic(ctx context.Context, host types.Host, args map[string]interface{}) (*ModuleResult, error) {
    // Implement module logic
    // ...
    
    return &ModuleResult{
        Changed: true,
        Output:  map[string]interface{}{"result": "success"},
    }, nil
}
```

### Module Testing

```go
// internal/modules/your_module_test.go
package modules

import (
    "context"
    "testing"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestYourModule_Execute(t *testing.T) {
    module := NewYourModule()
    host := types.Host{
        Name: "test-host",
        Address: "127.0.0.1",
    }
    
    testCases := []struct {
        name     string
        args     map[string]interface{}
        expected types.TaskResult
    }{
        {
            name: "valid arguments",
            args: map[string]interface{}{
                "name": "test",
                "state": "present",
            },
            expected: types.TaskResult{
                Changed: true,
                Output: map[string]interface{}{
                    "result": "success",
                },
            },
        },
        {
            name: "missing required argument",
            args: map[string]interface{}{
                "state": "present",
            },
            expected: types.TaskResult{
                Error: "required argument 'name' is missing",
            },
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := module.Execute(context.Background(), host, tc.args)
            
            if tc.expected.Error != "" {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tc.expected.Error)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tc.expected.Changed, result.Changed)
                assert.Equal(t, tc.expected.Output, result.Output)
            }
        })
    }
}
```

---

## 🔌 Plugin Development

### Creating Plugins

```go
// plugins/your_plugin.go
package plugins

import (
    "context"
    "github.com/onigirazu-cfg/onigirazu/internal/interfaces"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type YourPlugin struct {
    name        string
    description string
    version     string
}

func NewYourPlugin() *YourPlugin {
    return &YourPlugin{
        name:        "your_plugin",
        description: "Your plugin description",
        version:     "1.0.0",
    }
}

func (p *YourPlugin) GetName() string {
    return p.name
}

func (p *YourPlugin) GetDescription() string {
    return p.description
}

func (p *YourPlugin) GetVersion() string {
    return p.version
}

func (p *YourPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
    // Initialize plugin
    return nil
}

func (p *YourPlugin) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Execute plugin logic
    return types.TaskResult{
        Changed: true,
        Output:  map[string]interface{}{"result": "success"},
    }, nil
}

func (p *YourPlugin) Cleanup(ctx context.Context) error {
    // Cleanup plugin resources
    return nil
}
```

### Plugin Registration

```go
// plugins/registry.go
package plugins

import (
    "context"
    "github.com/onigirazu-cfg/onigirazu/internal/interfaces"
)

type PluginRegistry struct {
    plugins map[string]interfaces.Plugin
}

func NewPluginRegistry() *PluginRegistry {
    return &PluginRegistry{
        plugins: make(map[string]interfaces.Plugin),
    }
}

func (r *PluginRegistry) RegisterPlugin(plugin interfaces.Plugin) error {
    r.plugins[plugin.GetName()] = plugin
    return nil
}

func (r *PluginRegistry) GetPlugin(name string) (interfaces.Plugin, error) {
    plugin, exists := r.plugins[name]
    if !exists {
        return nil, fmt.Errorf("plugin not found: %s", name)
    }
    return plugin, nil
}

func (r *PluginRegistry) ListPlugins() []string {
    var names []string
    for name := range r.plugins {
        names = append(names, name)
    }
    return names
}
```

---

## 🔧 CLI Development

### Adding New Commands

```go
// internal/cli/your_command.go
package cli

import (
    "fmt"
    "github.com/spf13/cobra"
)

var yourCmd = &cobra.Command{
    Use:   "your-command",
    Short: "Your command description",
    Long:  "Your command long description with examples",
    Args:  cobra.MinimumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // Command implementation
        return executeYourCommand(args)
    },
}

func init() {
    rootCmd.AddCommand(yourCmd)
    
    // Add flags
    yourCmd.Flags().StringVarP(&flagValue, "flag", "f", "", "Flag description")
    yourCmd.Flags().BoolVar(&boolFlag, "bool-flag", false, "Boolean flag description")
}

func executeYourCommand(args []string) error {
    // Command logic
    fmt.Printf("Executing command with args: %v\n", args)
    return nil
}
```

### Command Testing

```go
// internal/cli/your_command_test.go
package cli

import (
    "testing"
    "github.com/spf13/cobra"
    "github.com/stretchr/testify/assert"
)

func TestYourCommand(t *testing.T) {
    cmd := &cobra.Command{
        Use: "test-command",
        RunE: func(cmd *cobra.Command, args []string) error {
            return nil
        },
    }
    
    err := cmd.Execute()
    assert.NoError(t, err)
}
```

---

## 📊 API Development

### REST API

```go
// internal/api/rest.go
package api

import (
    "context"
    "encoding/json"
    "net/http"
    "github.com/gorilla/mux"
    "github.com/onigirazu-cfg/onigirazu/internal/engine"
)

type RESTAPI struct {
    engine *engine.ExecutionEngine
    router *mux.Router
}

func NewRESTAPI(engine *engine.ExecutionEngine) *RESTAPI {
    api := &RESTAPI{
        engine: engine,
        router: mux.NewRouter(),
    }
    
    api.setupRoutes()
    return api
}

func (api *RESTAPI) setupRoutes() {
    api.router.HandleFunc("/api/v1/playbooks", api.handlePlaybooks).Methods("GET", "POST")
    api.router.HandleFunc("/api/v1/playbooks/{id}", api.handlePlaybook).Methods("GET", "PUT", "DELETE")
    api.router.HandleFunc("/api/v1/executions", api.handleExecutions).Methods("GET", "POST")
    api.router.HandleFunc("/api/v1/executions/{id}", api.handleExecution).Methods("GET")
}

func (api *RESTAPI) handlePlaybooks(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case "GET":
        api.listPlaybooks(w, r)
    case "POST":
        api.createPlaybook(w, r)
    }
}

func (api *RESTAPI) listPlaybooks(w http.ResponseWriter, r *http.Request) {
    playbooks := api.engine.ListPlaybooks()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(playbooks)
}
```

### GraphQL API

```go
// internal/api/graphql.go
package api

import (
    "context"
    "github.com/graphql-go/graphql"
    "github.com/onigirazu-cfg/onigirazu/internal/engine"
)

type GraphQLAPI struct {
    engine *engine.ExecutionEngine
    schema graphql.Schema
}

func NewGraphQLAPI(engine *engine.ExecutionEngine) *GraphQLAPI {
    api := &GraphQLAPI{
        engine: engine,
    }
    
    api.setupSchema()
    return api
}

func (api *GraphQLAPI) setupSchema() {
    playbookType := graphql.NewObject(graphql.ObjectConfig{
        Name: "Playbook",
        Fields: graphql.Fields{
            "id": &graphql.Field{
                Type: graphql.String,
            },
            "name": &graphql.Field{
                Type: graphql.String,
            },
            "description": &graphql.Field{
                Type: graphql.String,
            },
        },
    })
    
    queryType := graphql.NewObject(graphql.ObjectConfig{
        Name: "Query",
        Fields: graphql.Fields{
            "playbooks": &graphql.Field{
                Type: graphql.NewList(playbookType),
                Resolve: func(p graphql.ResolveParams) (interface{}, error) {
                    return api.engine.ListPlaybooks(), nil
                },
            },
        },
    })
    
    api.schema, _ = graphql.NewSchema(graphql.SchemaConfig{
        Query: queryType,
    })
}
```

---

## 🔧 Code Style

### Go Code Style

```go
// Package documentation
// Package your_package provides functionality for...
package your_package

import (
    "context"
    "fmt"
    "time"
    
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// YourFunction does something useful.
// It takes a context and returns an error.
func YourFunction(ctx context.Context) error {
    // Implementation
    return nil
}

// YourStruct represents something.
type YourStruct struct {
    // Field documentation
    Name string `json:"name"`
    Age  int    `json:"age"`
}

// NewYourStruct creates a new YourStruct.
func NewYourStruct(name string, age int) *YourStruct {
    return &YourStruct{
        Name: name,
        Age:  age,
    }
}

// Method documentation
func (s *YourStruct) GetName() string {
    return s.Name
}
```

### Testing Style

```go
// Test function naming
func TestYourFunction(t *testing.T) {
    // Test implementation
}

func TestYourFunction_WithSpecificCase(t *testing.T) {
    // Specific test case
}

func BenchmarkYourFunction(b *testing.B) {
    // Benchmark implementation
}

// Test table structure
func TestYourFunction_Table(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "case1",
            input:    "input1",
            expected: "expected1",
        },
        {
            name:     "case2",
            input:    "input2",
            expected: "expected2",
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result := YourFunction(tc.input)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

---

## 🔧 Best Practices

### Error Handling

```go
// Proper error handling
func YourFunction() error {
    result, err := SomeFunction()
    if err != nil {
        return fmt.Errorf("failed to execute function: %w", err)
    }
    
    // Handle result
    return nil
}

// Context usage
func YourFunctionWithContext(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    result, err := SomeFunction(ctx)
    if err != nil {
        return fmt.Errorf("function failed: %w", err)
    }
    
    return nil
}
```

### Resource Management

```go
// Proper resource management
func YourFunction() error {
    resource, err := AcquireResource()
    if err != nil {
        return err
    }
    defer ReleaseResource(resource)
    
    // Use resource
    return nil
}

// Connection pooling
func YourFunctionWithPool() error {
    conn, err := pool.GetConnection()
    if err != nil {
        return err
    }
    defer pool.ReleaseConnection(conn)
    
    // Use connection
    return nil
}
```

### Performance

```go
// Performance optimization
func YourFunction() {
    // Use sync.Pool for object reuse
    pool := &sync.Pool{
        New: func() interface{} {
            return &Buffer{}
        },
    }
    
    buffer := pool.Get().(*Buffer)
    defer pool.Put(buffer)
    
    // Use buffered channels
    results := make(chan Result, 100)
    
    // Use context for cancellation
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Use goroutines efficiently
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            processItem(ctx)
        }()
    }
    wg.Wait()
}
```

---

## 📚 Documentation

### Code Documentation

```go
// Package documentation
// Package your_package provides functionality for...
package your_package

// YourFunction does something useful.
// It takes a context and returns an error.
func YourFunction(ctx context.Context) error {
    // Implementation
    return nil
}

// YourStruct represents something.
type YourStruct struct {
    // Field documentation
    Name string `json:"name"`
    Age  int    `json:"age"`
}
```

### API Documentation

```go
// API documentation
// @title Onigirazu API
// @description Onigirazu REST API
// @version 1.0
// @host localhost:8080
// @BasePath /api/v1

// @Summary List playbooks
// @Description Get list of all playbooks
// @Tags playbooks
// @Accept json
// @Produce json
// @Success 200 {array} Playbook
// @Router /playbooks [get]
func (api *RESTAPI) listPlaybooks(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

---

## 🚨 Troubleshooting

### Development Issues

#### Build Issues
```bash
# Check Go version
go version

# Clean module cache
go clean -modcache

# Download modules
go mod download

# Verify modules
go mod verify
```

#### Test Issues
```bash
# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestSpecificFunction ./...

# Run tests with race detection
go test -race ./...
```

#### Linting Issues
```bash
# Run linter
golangci-lint run

# Run with specific rules
golangci-lint run --enable=gofmt,goimports,vet

# Fix issues
golangci-lint run --fix
```

### Debug Development

```bash
# Enable debug output
export ONIGIRAZU_DEBUG=true

# Run with debug
go run cmd/onigirazu/main.go --debug

# Use delve debugger
dlv debug cmd/onigirazu/main.go
```

---

## 📚 Related Documentation

- [Development Setup](Development-Setup) - Development environment
- [Architecture](Architecture) - System architecture
- [Testing](Testing) - Testing guide
- [Contributing](Contributing) - Contribution guidelines
- [API Reference](API-Reference) - API documentation

---

## 🎯 Summary

### Development Features

- **🏗️ Modular architecture** - Easy to extend
- **🧪 Comprehensive testing** - Unit, integration, performance
- **🔌 Plugin system** - Extensible functionality
- **📊 Rich APIs** - REST and GraphQL
- **🔧 Development tools** - Linting, formatting, security

### Development Benefits

- **🚀 Fast development** - Go efficiency
- **🔒 Secure by default** - Built-in security
- **📈 Scalable** - Handle large deployments
- **🔧 Maintainable** - Clean code practices
- **📚 Well documented** - Comprehensive guides

---

**👨‍💻 Onigirazu development is designed for productivity and maintainability!**
