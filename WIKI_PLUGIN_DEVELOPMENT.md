# 🔌 Plugin Development

This guide covers developing plugins for Onigirazu to extend its functionality.

## 📋 Plugin Overview

### Plugin Types

- **Module Plugins** - Extend module functionality
- **Inventory Plugins** - Custom inventory sources
- **Connection Plugins** - Custom connection methods
- **Callback Plugins** - Event callbacks
- **Filter Plugins** - Data filtering

### Plugin Benefits

- **Extensibility** - Add new functionality
- **Modularity** - Keep features separate
- **Reusability** - Share plugins across projects
- **Maintainability** - Easier to maintain
- **Community** - Community contributions

---

## 🚀 Getting Started

### 1. Plugin Structure

```
your-plugin/
├── main.go              # Plugin entry point
├── plugin.go            # Plugin implementation
├── config.go            # Configuration
├── types.go             # Type definitions
├── go.mod               # Go module file
├── go.sum               # Go module checksums
├── README.md            # Plugin documentation
├── examples/            # Usage examples
├── tests/               # Test files
└── docs/                # Documentation
```

### 2. Basic Plugin

```go
// main.go
package main

import (
    "context"
    "github.com/onigirazu-cfg/onigirazu/internal/interfaces"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type YourPlugin struct {
    name        string
    description string
    version     string
    config      map[string]interface{}
}

func NewYourPlugin() *YourPlugin {
    return &YourPlugin{
        name:        "your_plugin",
        description: "Your plugin description",
        version:     "1.0.0",
        config:      make(map[string]interface{}),
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
    p.config = config
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

func main() {
    // Plugin entry point
    plugin := NewYourPlugin()
    // Register plugin
}
```

---

## 🔧 Module Plugins

### Module Plugin Interface

```go
// Module plugin interface
type ModulePlugin interface {
    interfaces.Plugin
    
    // Module-specific methods
    GetModuleName() string
    GetModuleDescription() string
    ExecuteModule(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
}
```

### Module Plugin Implementation

```go
// your_module_plugin.go
package main

import (
    "context"
    "fmt"
    "github.com/onigirazu-cfg/onigirazu/internal/interfaces"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type YourModulePlugin struct {
    name        string
    description string
    version     string
    config      map[string]interface{}
}

func NewYourModulePlugin() *YourModulePlugin {
    return &YourModulePlugin{
        name:        "your_module",
        description: "Your module plugin",
        version:     "1.0.0",
        config:      make(map[string]interface{}),
    }
}

func (p *YourModulePlugin) GetName() string {
    return p.name
}

func (p *YourModulePlugin) GetDescription() string {
    return p.description
}

func (p *YourModulePlugin) GetVersion() string {
    return p.version
}

func (p *YourModulePlugin) GetModuleName() string {
    return "your_module"
}

func (p *YourModulePlugin) GetModuleDescription() string {
    return "Your module plugin for Onigirazu"
}

func (p *YourModulePlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
    // Initialize plugin
    p.config = config
    return nil
}

func (p *YourModulePlugin) ExecuteModule(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Validate arguments
    if err := p.validateArgs(args); err != nil {
        return types.TaskResult{}, err
    }
    
    // Execute module logic
    result, err := p.executeLogic(ctx, host, args)
    if err != nil {
        return types.TaskResult{}, err
    }
    
    return types.TaskResult{
        Changed: result.Changed,
        Output:  result.Output,
        Error:   result.Error,
    }, nil
}

func (p *YourModulePlugin) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    return p.ExecuteModule(ctx, host, args)
}

func (p *YourModulePlugin) Cleanup(ctx context.Context) error {
    // Cleanup plugin resources
    return nil
}

func (p *YourModulePlugin) validateArgs(args map[string]interface{}) error {
    // Validate required arguments
    if _, exists := args["name"]; !exists {
        return fmt.Errorf("required argument 'name' is missing")
    }
    
    return nil
}

func (p *YourModulePlugin) executeLogic(ctx context.Context, host types.Host, args map[string]interface{}) (*ModuleResult, error) {
    // Implement module logic
    // ...
    
    return &ModuleResult{
        Changed: true,
        Output:  map[string]interface{}{"result": "success"},
    }, nil
}
```

---

## 🔧 Inventory Plugins

### Inventory Plugin Interface

```go
// Inventory plugin interface
type InventoryPlugin interface {
    interfaces.Plugin
    
    // Inventory-specific methods
    GetInventorySource() string
    LoadInventory(ctx context.Context, source string) ([]types.Host, error)
    GetHosts(ctx context.Context, pattern string) ([]types.Host, error)
}
```

### Inventory Plugin Implementation

```go
// your_inventory_plugin.go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "github.com/onigirazu-cfg/onigirazu/internal/interfaces"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type YourInventoryPlugin struct {
    name        string
    description string
    version     string
    config      map[string]interface{}
    client      *http.Client
}

func NewYourInventoryPlugin() *YourInventoryPlugin {
    return &YourInventoryPlugin{
        name:        "your_inventory",
        description: "Your inventory plugin",
        version:     "1.0.0",
        config:      make(map[string]interface{}),
        client:      &http.Client{},
    }
}

func (p *YourInventoryPlugin) GetName() string {
    return p.name
}

func (p *YourInventoryPlugin) GetDescription() string {
    return p.description
}

func (p *YourInventoryPlugin) GetVersion() string {
    return p.version
}

func (p *YourInventoryPlugin) GetInventorySource() string {
    return "your_inventory"
}

func (p *YourInventoryPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
    // Initialize plugin
    p.config = config
    return nil
}

func (p *YourInventoryPlugin) LoadInventory(ctx context.Context, source string) ([]types.Host, error) {
    // Load inventory from source
    hosts, err := p.loadFromSource(ctx, source)
    if err != nil {
        return nil, err
    }
    
    return hosts, nil
}

func (p *YourInventoryPlugin) GetHosts(ctx context.Context, pattern string) ([]types.Host, error) {
    // Get hosts matching pattern
    hosts, err := p.loadFromSource(ctx, pattern)
    if err != nil {
        return nil, err
    }
    
    return hosts, nil
}

func (p *YourInventoryPlugin) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Not applicable for inventory plugins
    return types.TaskResult{}, fmt.Errorf("execute not supported for inventory plugins")
}

func (p *YourInventoryPlugin) Cleanup(ctx context.Context) error {
    // Cleanup plugin resources
    return nil
}

func (p *YourInventoryPlugin) loadFromSource(ctx context.Context, source string) ([]types.Host, error) {
    // Load hosts from source
    // This could be from a database, API, file, etc.
    
    // Example: Load from API
    req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var hosts []types.Host
    if err := json.NewDecoder(resp.Body).Decode(&hosts); err != nil {
        return nil, err
    }
    
    return hosts, nil
}
```

---

## 🔧 Connection Plugins

### Connection Plugin Interface

```go
// Connection plugin interface
type ConnectionPlugin interface {
    interfaces.Plugin
    
    // Connection-specific methods
    GetConnectionType() string
    Connect(ctx context.Context, host types.Host) (Connection, error)
    ExecuteCommand(ctx context.Context, conn Connection, command string) (string, error)
    Close(ctx context.Context, conn Connection) error
}
```

### Connection Plugin Implementation

```go
// your_connection_plugin.go
package main

import (
    "context"
    "fmt"
    "net"
    "time"
    "github.com/onigirazu-cfg/onigirazu/internal/interfaces"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type YourConnectionPlugin struct {
    name        string
    description string
    version     string
    config      map[string]interface{}
}

type YourConnection struct {
    conn net.Conn
    host types.Host
}

func NewYourConnectionPlugin() *YourConnectionPlugin {
    return &YourConnectionPlugin{
        name:        "your_connection",
        description: "Your connection plugin",
        version:     "1.0.0",
        config:      make(map[string]interface{}),
    }
}

func (p *YourConnectionPlugin) GetName() string {
    return p.name
}

func (p *YourConnectionPlugin) GetDescription() string {
    return p.description
}

func (p *YourConnectionPlugin) GetVersion() string {
    return p.version
}

func (p *YourConnectionPlugin) GetConnectionType() string {
    return "your_connection"
}

func (p *YourConnectionPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
    // Initialize plugin
    p.config = config
    return nil
}

func (p *YourConnectionPlugin) Connect(ctx context.Context, host types.Host) (Connection, error) {
    // Connect to host
    conn, err := net.DialTimeout("tcp", host.Address, 30*time.Second)
    if err != nil {
        return nil, err
    }
    
    return &YourConnection{
        conn: conn,
        host: host,
    }, nil
}

func (p *YourConnectionPlugin) ExecuteCommand(ctx context.Context, conn Connection, command string) (string, error) {
    // Execute command on connection
    yourConn := conn.(*YourConnection)
    
    // Send command
    _, err := yourConn.conn.Write([]byte(command + "\n"))
    if err != nil {
        return "", err
    }
    
    // Read response
    buffer := make([]byte, 4096)
    n, err := yourConn.conn.Read(buffer)
    if err != nil {
        return "", err
    }
    
    return string(buffer[:n]), nil
}

func (p *YourConnectionPlugin) Close(ctx context.Context, conn Connection) error {
    // Close connection
    yourConn := conn.(*YourConnection)
    return yourConn.conn.Close()
}

func (p *YourConnectionPlugin) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Not applicable for connection plugins
    return types.TaskResult{}, fmt.Errorf("execute not supported for connection plugins")
}

func (p *YourConnectionPlugin) Cleanup(ctx context.Context) error {
    // Cleanup plugin resources
    return nil
}
```

---

## 🔧 Callback Plugins

### Callback Plugin Interface

```go
// Callback plugin interface
type CallbackPlugin interface {
    interfaces.Plugin
    
    // Callback-specific methods
    GetCallbackTypes() []string
    OnEvent(ctx context.Context, eventType string, data interface{}) error
}
```

### Callback Plugin Implementation

```go
// your_callback_plugin.go
package main

import (
    "context"
    "fmt"
    "log"
    "github.com/onigirazu-cfg/onigirazu/internal/interfaces"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type YourCallbackPlugin struct {
    name        string
    description string
    version     string
    config      map[string]interface{}
}

func NewYourCallbackPlugin() *YourCallbackPlugin {
    return &YourCallbackPlugin{
        name:        "your_callback",
        description: "Your callback plugin",
        version:     "1.0.0",
        config:      make(map[string]interface{}),
    }
}

func (p *YourCallbackPlugin) GetName() string {
    return p.name
}

func (p *YourCallbackPlugin) GetDescription() string {
    return p.description
}

func (p *YourCallbackPlugin) GetVersion() string {
    return p.version
}

func (p *YourCallbackPlugin) GetCallbackTypes() []string {
    return []string{"task_start", "task_end", "play_start", "play_end"}
}

func (p *YourCallbackPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
    // Initialize plugin
    p.config = config
    return nil
}

func (p *YourCallbackPlugin) OnEvent(ctx context.Context, eventType string, data interface{}) error {
    // Handle callback event
    switch eventType {
    case "task_start":
        return p.onTaskStart(ctx, data)
    case "task_end":
        return p.onTaskEnd(ctx, data)
    case "play_start":
        return p.onPlayStart(ctx, data)
    case "play_end":
        return p.onPlayEnd(ctx, data)
    default:
        return fmt.Errorf("unknown event type: %s", eventType)
    }
}

func (p *YourCallbackPlugin) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Not applicable for callback plugins
    return types.TaskResult{}, fmt.Errorf("execute not supported for callback plugins")
}

func (p *YourCallbackPlugin) Cleanup(ctx context.Context) error {
    // Cleanup plugin resources
    return nil
}

func (p *YourCallbackPlugin) onTaskStart(ctx context.Context, data interface{}) error {
    log.Printf("Task started: %v", data)
    return nil
}

func (p *YourCallbackPlugin) onTaskEnd(ctx context.Context, data interface{}) error {
    log.Printf("Task ended: %v", data)
    return nil
}

func (p *YourCallbackPlugin) onPlayStart(ctx context.Context, data interface{}) error {
    log.Printf("Play started: %v", data)
    return nil
}

func (p *YourCallbackPlugin) onPlayEnd(ctx context.Context, data interface{}) error {
    log.Printf("Play ended: %v", data)
    return nil
}
```

---

## 🔧 Filter Plugins

### Filter Plugin Interface

```go
// Filter plugin interface
type FilterPlugin interface {
    interfaces.Plugin
    
    // Filter-specific methods
    GetFilterTypes() []string
    Filter(ctx context.Context, filterType string, data interface{}) (interface{}, error)
}
```

### Filter Plugin Implementation

```go
// your_filter_plugin.go
package main

import (
    "context"
    "fmt"
    "strings"
    "github.com/onigirazu-cfg/onigirazu/internal/interfaces"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type YourFilterPlugin struct {
    name        string
    description string
    version     string
    config      map[string]interface{}
}

func NewYourFilterPlugin() *YourFilterPlugin {
    return &YourFilterPlugin{
        name:        "your_filter",
        description: "Your filter plugin",
        version:     "1.0.0",
        config:      make(map[string]interface{}),
    }
}

func (p *YourFilterPlugin) GetName() string {
    return p.name
}

func (p *YourFilterPlugin) GetDescription() string {
    return p.description
}

func (p *YourFilterPlugin) GetVersion() string {
    return p.version
}

func (p *YourFilterPlugin) GetFilterTypes() []string {
    return []string{"string", "json", "yaml", "xml"}
}

func (p *YourFilterPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
    // Initialize plugin
    p.config = config
    return nil
}

func (p *YourFilterPlugin) Filter(ctx context.Context, filterType string, data interface{}) (interface{}, error) {
    // Apply filter
    switch filterType {
    case "string":
        return p.filterString(ctx, data)
    case "json":
        return p.filterJSON(ctx, data)
    case "yaml":
        return p.filterYAML(ctx, data)
    case "xml":
        return p.filterXML(ctx, data)
    default:
        return nil, fmt.Errorf("unknown filter type: %s", filterType)
    }
}

func (p *YourFilterPlugin) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Not applicable for filter plugins
    return types.TaskResult{}, fmt.Errorf("execute not supported for filter plugins")
}

func (p *YourFilterPlugin) Cleanup(ctx context.Context) error {
    // Cleanup plugin resources
    return nil
}

func (p *YourFilterPlugin) filterString(ctx context.Context, data interface{}) (interface{}, error) {
    // Filter string data
    str, ok := data.(string)
    if !ok {
        return nil, fmt.Errorf("expected string, got %T", data)
    }
    
    // Apply string filter
    filtered := strings.TrimSpace(str)
    filtered = strings.ToLower(filtered)
    
    return filtered, nil
}

func (p *YourFilterPlugin) filterJSON(ctx context.Context, data interface{}) (interface{}, error) {
    // Filter JSON data
    // Implementation
    return data, nil
}

func (p *YourFilterPlugin) filterYAML(ctx context.Context, data interface{}) (interface{}, error) {
    // Filter YAML data
    // Implementation
    return data, nil
}

func (p *YourFilterPlugin) filterXML(ctx context.Context, data interface{}) (interface{}, error) {
    // Filter XML data
    // Implementation
    return data, nil
}
```

---

## 🧪 Plugin Testing

### Unit Testing

```go
// your_plugin_test.go
package main

import (
    "context"
    "testing"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestYourPlugin_Execute(t *testing.T) {
    plugin := NewYourPlugin()
    host := types.Host{
        Name: "test-host",
        Address: "127.0.0.1",
    }
    args := map[string]interface{}{
        "param": "value",
    }
    
    result, err := plugin.Execute(context.Background(), host, args)
    
    require.NoError(t, err)
    assert.True(t, result.Changed)
    assert.Equal(t, "success", result.Output["result"])
}

func TestYourPlugin_Initialize(t *testing.T) {
    plugin := NewYourPlugin()
    config := map[string]interface{}{
        "key": "value",
    }
    
    err := plugin.Initialize(context.Background(), config)
    
    require.NoError(t, err)
    assert.Equal(t, config, plugin.config)
}

func TestYourPlugin_Cleanup(t *testing.T) {
    plugin := NewYourPlugin()
    
    err := plugin.Cleanup(context.Background())
    
    require.NoError(t, err)
}
```

### Integration Testing

```go
// your_plugin_integration_test.go
package main

import (
    "context"
    "testing"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestYourPlugin_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    plugin := NewYourPlugin()
    host := types.Host{
        Name: "test-host",
        Address: "127.0.0.1",
    }
    args := map[string]interface{}{
        "param": "value",
    }
    
    // Initialize plugin
    err := plugin.Initialize(context.Background(), map[string]interface{}{})
    require.NoError(t, err)
    
    // Execute plugin
    result, err := plugin.Execute(context.Background(), host, args)
    require.NoError(t, err)
    assert.True(t, result.Changed)
    
    // Cleanup plugin
    err = plugin.Cleanup(context.Background())
    require.NoError(t, err)
}
```

---

## 📚 Plugin Documentation

### Plugin README

```markdown
# Your Plugin

Your plugin description.

## Features

- Feature 1
- Feature 2
- Feature 3

## Installation

```bash
go install github.com/your-username/your-plugin@latest
```

## Usage

```yaml
# Use in playbook
- name: Use your plugin
  your_module:
    param: value
```

## Configuration

```yaml
# Plugin configuration
your_plugin:
  key: value
  option: true
```

## Examples

See examples/ directory for usage examples.

## Documentation

See docs/ directory for detailed documentation.
```

### API Documentation

```go
// API documentation
// @title Your Plugin API
// @description Your Plugin API for Onigirazu
// @version 1.0
// @host localhost:8080
// @BasePath /api/v1

// @Summary Execute plugin
// @Description Execute your plugin
// @Tags plugin
// @Accept json
// @Produce json
// @Param request body PluginRequest true "Plugin request"
// @Success 200 {object} PluginResponse
// @Router /plugin/execute [post]
func (p *YourPlugin) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Implementation
}
```

---

## 🔧 Plugin Configuration

### Configuration Structure

```go
// Plugin configuration
type PluginConfig struct {
    // Plugin settings
    Name        string                 `yaml:"name"`
    Version     string                 `yaml:"version"`
    Description string                 `yaml:"description"`
    
    // Plugin options
    Options     map[string]interface{} `yaml:"options"`
    
    // Plugin settings
    Settings    PluginSettings        `yaml:"settings"`
}

type PluginSettings struct {
    // Plugin-specific settings
    Timeout     time.Duration `yaml:"timeout"`
    Retries     int          `yaml:"retries"`
    Debug       bool         `yaml:"debug"`
}
```

### Configuration Loading

```go
// Load plugin configuration
func LoadPluginConfig(path string) (*PluginConfig, error) {
    // Read file
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    // Parse YAML
    var config PluginConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }
    
    // Validate config
    if err := config.Validate(); err != nil {
        return nil, err
    }
    
    return &config, nil
}
```

---

## 🚀 Plugin Deployment

### Build Plugin

```bash
# Build plugin
go build -o your-plugin main.go

# Build with version
go build -ldflags "-X main.version=1.0.0" -o your-plugin main.go

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o your-plugin-linux-amd64 main.go
GOOS=windows GOARCH=amd64 go build -o your-plugin-windows-amd64.exe main.go
```

### Deploy Plugin

```bash
# Install plugin
go install github.com/your-username/your-plugin@latest

# Copy plugin binary
cp your-plugin /usr/local/bin/

# Set permissions
chmod +x /usr/local/bin/your-plugin
```

### Plugin Registry

```yaml
# Plugin registry
plugins:
  your_plugin:
    name: "your_plugin"
    version: "1.0.0"
    description: "Your plugin description"
    author: "Your Name"
    license: "MIT"
    repository: "https://github.com/your-username/your-plugin"
    homepage: "https://your-plugin.example.com"
    documentation: "https://docs.your-plugin.example.com"
```

---

## 📚 Related Documentation

- [Development Setup](Development-Setup) - Development environment
- [Testing](Testing) - Testing guide
- [Contributing](Contributing) - Contribution guidelines
- [API Reference](API-Reference) - API documentation

---

## 🎯 Summary

### Plugin Features

- **🔌 Extensible** - Add new functionality
- **📦 Modular** - Keep features separate
- **🔄 Reusable** - Share across projects
- **🔧 Maintainable** - Easy to maintain
- **👥 Community** - Community contributions

### Plugin Benefits

- **🚀 Productivity** - Faster development
- **🔧 Flexibility** - Custom functionality
- **📈 Scalability** - Handle large deployments
- **🔒 Security** - Secure plugins
- **📚 Documentation** - Well-documented plugins

---

**🔌 Plugin development makes Onigirazu highly extensible and customizable!**
