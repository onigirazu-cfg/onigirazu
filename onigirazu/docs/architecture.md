# Architecture Overview

Onigirazu is built with a modular, extensible architecture that supports complex automation scenarios while maintaining security and performance.

## 🏗️ System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Onigirazu Platform                       │
├─────────────────────────────────────────────────────────────┤
│  CLI Interface  │  REST API  │  Web UI  │  Workflow Engine │
├─────────────────────────────────────────────────────────────┤
│                    Core Engine                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │  Playbook   │ │   Task      │ │  Inventory  │           │
│  │  Processor  │ │  Executor   │ │  Manager    │           │
│  └─────────────┘ └─────────────┘ └─────────────┘           │
├─────────────────────────────────────────────────────────────┤
│                   Module System                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │   Built-in  │ │   Custom    │ │   Plugin    │           │
│  │   Modules   │ │   Modules   │ │   Loader    │           │
│  └─────────────┘ └─────────────┘ └─────────────┘           │
├─────────────────────────────────────────────────────────────┤
│                 Infrastructure Layer                        │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │  Security   │ │  Monitoring │ │   Cache     │           │
│  │  Framework  │ │   System    │ │   System    │           │
│  └─────────────┘ └─────────────┘ └─────────────┘           │
├─────────────────────────────────────────────────────────────┤
│                   Transport Layer                           │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │     SSH     │ │    WinRM    │ │    Local    │           │
│  │  Transport  │ │  Transport  │ │  Transport  │           │
│  └─────────────┘ └─────────────┘ └─────────────┘           │
└─────────────────────────────────────────────────────────────┘
```

## 🔧 Core Components

### 1. Core Engine

The heart of Onigirazu, responsible for:

- **Playbook Processing**: YAML parsing and validation
- **Task Execution**: Sequential and parallel task execution
- **Inventory Management**: Host and group management
- **Variable Resolution**: Template rendering and variable substitution
- **Condition Evaluation**: When/until condition processing

### 2. Module System

Extensible module architecture supporting:

- **Built-in Modules**: Core functionality (file, service, package, etc.)
- **Custom Modules**: User-defined modules
- **Plugin Loading**: Dynamic module discovery and loading
- **Module Validation**: Input validation and error handling

### 3. Security Framework

Multi-layer security system:

- **Input Validation**: Sanitization of all user inputs
- **Access Control**: Host and operation restrictions
- **Audit Logging**: Comprehensive security event tracking
- **Threat Detection**: Pattern-based security analysis

### 4. Monitoring System

Real-time performance monitoring:

- **Metrics Collection**: System and application metrics
- **Performance Tracking**: Task execution monitoring
- **Resource Monitoring**: Memory, CPU, and I/O tracking
- **Report Generation**: Automated performance reports

### 5. Workflow Engine

Advanced automation orchestration:

- **Dependency Management**: Task and workflow dependencies
- **Parallel Execution**: Concurrent task processing
- **Event System**: Event-driven workflow triggers
- **Scheduling**: Cron-based workflow scheduling

## 📦 Package Structure

```
onigirazu/
├── cmd/                    # Command-line interfaces
│   ├── onigirazu/         # Main CLI application
│   └── onigirazu-server/  # Server mode
├── pkg/                   # Public packages
│   ├── types/            # Core type definitions
│   ├── utils/            # Utility functions
│   └── client/           # Client libraries
├── internal/             # Internal packages
│   ├── core/            # Core engine
│   ├── modules/         # Built-in modules
│   ├── security/        # Security framework
│   ├── monitoring/      # Monitoring system
│   ├── workflow/        # Workflow engine
│   ├── transport/       # Transport layer
│   └── plugins/         # Plugin system
├── plugins/             # External plugins
├── docs/               # Documentation
├── examples/           # Example configurations
└── tests/             # Test suites
```

## 🔄 Execution Flow

### 1. Initialization Phase

```go
// Load configuration
config := LoadConfig()

// Initialize security validator
validator := security.NewValidator(config.Security)

// Initialize monitoring
metrics := monitoring.NewCollector()

// Initialize workflow engine
orchestrator := workflow.NewOrchestrator()
```

### 2. Playbook Processing

```go
// Parse playbook
playbook, err := ParsePlaybook("playbook.yml")

// Validate security
result := validator.ValidatePlaybook(playbook)

// Load inventory
inventory := LoadInventory("inventory.yml")
```

### 3. Task Execution

```go
// For each play in playbook
for _, play := range playbook.Plays {
    // Resolve hosts
    hosts := inventory.ResolveHosts(play.Hosts)

    // Execute tasks
    for _, task := range play.Tasks {
        // Validate task
        validator.ValidateTask(task)

        // Execute on each host
        for _, host := range hosts {
            result := executor.ExecuteTask(task, host)
            metrics.RecordTaskExecution(result)
        }
    }
}
```

## 🔌 Plugin Architecture

### Plugin Interface

```go
type Plugin interface {
    // Plugin metadata
    Name() string
    Version() string
    Description() string

    // Plugin lifecycle
    Initialize(config map[string]interface{}) error
    Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
    Cleanup() error

    // Plugin validation
    ValidateArgs(args map[string]interface{}) error
    GetSchema() map[string]interface{}
}
```

### Module Interface

```go
type Module interface {
    Execute(ctx context.Context, host Host, args map[string]interface{}) (TaskResult, error)
    Validate(args map[string]interface{}) error
    GetName() string
    GetDescription() string
}
```

## 🔒 Security Architecture

### Security Layers

1. **Input Validation**: All inputs are validated and sanitized
2. **Access Control**: Host and operation restrictions
3. **Audit Trail**: Complete operation logging
4. **Threat Detection**: Pattern-based security analysis
5. **Encryption**: Secure communication channels

### Security Flow

```go
// Validate host access
hostResult := validator.ValidateHost(host)

// Validate task execution
taskResult := validator.ValidateTask(task)

// Audit operation
auditor.AuditTask(task, user)

// Execute with security context
result := executor.ExecuteSecure(task, host, securityContext)
```

## 📊 Monitoring Architecture

### Metrics Collection

- **System Metrics**: CPU, memory, disk, network
- **Application Metrics**: Task execution, performance
- **Custom Metrics**: User-defined metrics
- **Real-time Monitoring**: Live metric streaming

### Metric Types

- **Counters**: Incrementing values
- **Gauges**: Current values
- **Histograms**: Value distributions
- **Timers**: Duration measurements

## 🌊 Workflow Architecture

### Workflow Components

- **Steps**: Individual workflow operations
- **Dependencies**: Step execution order
- **Conditions**: Conditional execution
- **Events**: Workflow triggers
- **Scheduling**: Time-based execution

### Workflow Types

- **Sequential**: Step-by-step execution
- **Parallel**: Concurrent execution
- **Conditional**: Condition-based execution
- **Event-driven**: Event-triggered execution

## 🚀 Performance Considerations

### Optimization Strategies

- **Parallel Execution**: Concurrent task processing
- **Caching**: Result and metadata caching
- **Connection Pooling**: Efficient connection management
- **Resource Management**: Memory and CPU optimization

### Scalability Features

- **Horizontal Scaling**: Multi-node execution
- **Load Balancing**: Task distribution
- **Resource Limits**: Configurable resource constraints
- **Batch Processing**: Efficient bulk operations

This architecture provides a solid foundation for building complex automation solutions while maintaining security, performance, and extensibility.
