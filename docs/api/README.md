# API Reference

This document provides comprehensive API reference for Onigirazu's programmatic interfaces.

## 📋 Table of Contents

- [Core API](#core-api)
- [Module API](#module-api)
- [Security API](#security-api)
- [Monitoring API](#monitoring-api)
- [Workflow API](#workflow-api)
- [Plugin API](#plugin-api)
- [REST API](#rest-api)
- [Client Libraries](#client-libraries)

## 🔧 Core API

### Executor Interface

The main execution engine for running playbooks and tasks.

```go
package core

import (
    "context"
    "github.com/your-org/onigirazu/pkg/types"
)

// Executor defines the main execution interface
type Executor interface {
    // Execute a single task on a host
    ExecuteTask(ctx context.Context, task types.Task, host types.Host) (types.TaskResult, error)

    // Execute a play on multiple hosts
    ExecutePlay(ctx context.Context, play types.Play, inventory types.Inventory) (types.PlayResult, error)

    // Execute a complete playbook
    ExecutePlaybook(ctx context.Context, playbook types.Playbook, inventory types.Inventory) (types.PlaybookResult, error)

    // Get execution statistics
    GetStats() ExecutionStats

    // Set execution options
    SetOptions(options ExecutionOptions) error
}

// ExecutionOptions configures execution behavior
type ExecutionOptions struct {
    // Parallel execution settings
    Parallelism     int           `json:"parallelism"`
    SerialBatches   int           `json:"serial_batches"`

    // Timeout settings
    TaskTimeout     time.Duration `json:"task_timeout"`
    PlayTimeout     time.Duration `json:"play_timeout"`

    // Retry settings
    DefaultRetries  int           `json:"default_retries"`
    RetryDelay      time.Duration `json:"retry_delay"`

    // Behavior settings
    CheckMode       bool          `json:"check_mode"`
    DiffMode        bool          `json:"diff_mode"`
    VerboseLevel    int           `json:"verbose_level"`

    // Error handling
    IgnoreErrors    bool          `json:"ignore_errors"`
    AnyErrorsFatal  bool          `json:"any_errors_fatal"`
    MaxFailPercent  int           `json:"max_fail_percent"`
}

// ExecutionStats provides execution statistics
type ExecutionStats struct {
    TotalTasks      int           `json:"total_tasks"`
    CompletedTasks  int           `json:"completed_tasks"`
    FailedTasks     int           `json:"failed_tasks"`
    SkippedTasks    int           `json:"skipped_tasks"`
    ChangedTasks    int           `json:"changed_tasks"`

    TotalHosts      int           `json:"total_hosts"`
    UnreachableHosts int          `json:"unreachable_hosts"`

    StartTime       time.Time     `json:"start_time"`
    EndTime         time.Time     `json:"end_time"`
    Duration        time.Duration `json:"duration"`
}
```

### Usage Example

```go
package main

import (
    "context"
    "log"

    "github.com/your-org/onigirazu/internal/core"
    "github.com/your-org/onigirazu/pkg/types"
)

func main() {
    // Create executor
    executor := core.NewExecutor()

    // Configure execution options
    options := core.ExecutionOptions{
        Parallelism:    5,
        TaskTimeout:    time.Minute * 5,
        DefaultRetries: 3,
        VerboseLevel:   1,
    }
    executor.SetOptions(options)

    // Load playbook and inventory
    playbook, err := LoadPlaybook("playbook.yml")
    if err != nil {
        log.Fatal(err)
    }

    inventory, err := LoadInventory("inventory.yml")
    if err != nil {
        log.Fatal(err)
    }

    // Execute playbook
    ctx := context.Background()
    result, err := executor.ExecutePlaybook(ctx, playbook, inventory)
    if err != nil {
        log.Fatal(err)
    }

    // Print results
    log.Printf("Playbook execution completed: %+v", result)

    // Get statistics
    stats := executor.GetStats()
    log.Printf("Execution stats: %+v", stats)
}
```

### Inventory Management

```go
package inventory

// InventoryManager manages host inventories
type InventoryManager interface {
    // Load inventory from file
    LoadFromFile(path string) (types.Inventory, error)

    // Load inventory from string
    LoadFromString(content string) (types.Inventory, error)

    // Add host to inventory
    AddHost(host types.Host) error

    // Remove host from inventory
    RemoveHost(hostname string) error

    // Add group to inventory
    AddGroup(group types.Group) error

    // Remove group from inventory
    RemoveGroup(groupname string) error

    // Resolve hosts by pattern
    ResolveHosts(pattern string) ([]types.Host, error)

    // Get all hosts
    GetAllHosts() []types.Host

    // Get hosts in group
    GetGroupHosts(groupname string) ([]types.Host, error)

    // Get host variables
    GetHostVars(hostname string) (map[string]interface{}, error)

    // Get group variables
    GetGroupVars(groupname string) (map[string]interface{}, error)
}

// Usage example
func ExampleInventoryUsage() {
    manager := inventory.NewManager()

    // Load inventory
    inv, err := manager.LoadFromFile("inventory.yml")
    if err != nil {
        log.Fatal(err)
    }

    // Resolve hosts
    webservers, err := manager.ResolveHosts("webservers")
    if err != nil {
        log.Fatal(err)
    }

    // Get host variables
    for _, host := range webservers {
        vars, err := manager.GetHostVars(host.Name)
        if err != nil {
            log.Printf("Failed to get vars for %s: %v", host.Name, err)
            continue
        }
        log.Printf("Host %s vars: %+v", host.Name, vars)
    }
}
```

## 🔌 Module API

### Module Interface

All modules must implement the Module interface:

```go
package modules

import (
    "context"
    "github.com/your-org/onigirazu/pkg/types"
)

// Module defines the interface for all modules
type Module interface {
    // Execute the module on a host
    Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)

    // Validate module arguments
    Validate(args map[string]interface{}) error

    // Get module name
    GetName() string

    // Get module description
    GetDescription() string

    // Get argument schema
    GetSchema() map[string]interface{}

    // Initialize module with configuration
    Initialize(config map[string]interface{}) error

    // Cleanup module resources
    Cleanup() error
}

// ModuleRegistry manages module registration and discovery
type ModuleRegistry interface {
    // Register a module
    Register(module Module) error

    // Unregister a module
    Unregister(name string) error

    // Get module by name
    Get(name string) (Module, error)

    // List all registered modules
    List() []string

    // Check if module exists
    Exists(name string) bool
}
```

### Creating Custom Modules

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/your-org/onigirazu/pkg/types"
)

// CustomModule implements a custom module
type CustomModule struct {
    name        string
    description string
    config      map[string]interface{}
}

func NewCustomModule() *CustomModule {
    return &CustomModule{
        name:        "custom",
        description: "A custom module example",
        config:      make(map[string]interface{}),
    }
}

func (m *CustomModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    result := types.TaskResult{
        TaskName:  "custom",
        Host:      host.Name,
        Module:    m.name,
        Success:   true,
        Changed:   false,
        Timestamp: time.Now(),
        Output:    make(map[string]interface{}),
    }

    // Check for cancellation
    select {
    case <-ctx.Done():
        result.Success = false
        result.Error = "Task cancelled"
        return result, ctx.Err()
    default:
    }

    // Validate arguments
    if err := m.Validate(args); err != nil {
        result.Success = false
        result.Error = err.Error()
        return result, err
    }

    // Perform module operation
    action, ok := args["action"].(string)
    if !ok {
        action = "default"
    }

    switch action {
    case "create":
        result.Changed = true
        result.Output["created"] = true
        result.Output["message"] = "Resource created successfully"
    case "delete":
        result.Changed = true
        result.Output["deleted"] = true
        result.Output["message"] = "Resource deleted successfully"
    default:
        result.Output["message"] = "No action performed"
    }

    return result, nil
}

func (m *CustomModule) Validate(args map[string]interface{}) error {
    // Validate required arguments
    if action, exists := args["action"]; exists {
        if actionStr, ok := action.(string); ok {
            validActions := []string{"create", "delete", "default"}
            for _, valid := range validActions {
                if actionStr == valid {
                    return nil
                }
            }
            return fmt.Errorf("invalid action: %s", actionStr)
        }
        return fmt.Errorf("action must be a string")
    }
    return nil
}

func (m *CustomModule) GetName() string {
    return m.name
}

func (m *CustomModule) GetDescription() string {
    return m.description
}

func (m *CustomModule) GetSchema() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "action": map[string]interface{}{
                "type":        "string",
                "description": "Action to perform",
                "enum":        []string{"create", "delete", "default"},
                "default":     "default",
            },
            "name": map[string]interface{}{
                "type":        "string",
                "description": "Resource name",
            },
        },
        "additionalProperties": false,
    }
}

func (m *CustomModule) Initialize(config map[string]interface{}) error {
    m.config = config
    return nil
}

func (m *CustomModule) Cleanup() error {
    // Cleanup resources
    return nil
}

// Register the module
func init() {
    registry := modules.GetRegistry()
    registry.Register(NewCustomModule())
}
```

## 🔒 Security API

### Security Validator

```go
package security

import (
    "github.com/your-org/onigirazu/pkg/types"
)

// SecurityValidator validates security aspects of operations
type SecurityValidator interface {
    // Validate host access
    ValidateHost(host types.Host) ValidationResult

    // Validate task execution
    ValidateTask(task types.Task) ValidationResult

    // Validate playbook
    ValidatePlaybook(playbook types.Playbook) ValidationResult

    // Validate file operations
    ValidateFileOperation(operation FileOperation) ValidationResult

    // Get security score
    GetSecurityScore(target interface{}) SecurityScore

    // Configure validation rules
    SetRules(rules SecurityRules) error

    // Get current rules
    GetRules() SecurityRules
}

// ValidationResult contains validation results
type ValidationResult struct {
    Valid       bool                   `json:"valid"`
    Score       int                    `json:"score"`
    Violations  []SecurityViolation    `json:"violations"`
    Warnings    []SecurityWarning      `json:"warnings"`
    Metadata    map[string]interface{} `json:"metadata"`
}

// SecurityViolation represents a security violation
type SecurityViolation struct {
    Type        string                 `json:"type"`
    Severity    string                 `json:"severity"`
    Message     string                 `json:"message"`
    Context     string                 `json:"context"`
    Suggestion  string                 `json:"suggestion"`
    Metadata    map[string]interface{} `json:"metadata"`
}

// SecurityRules defines security validation rules
type SecurityRules struct {
    // Host validation rules
    AllowedHosts        []string          `json:"allowed_hosts"`
    BlockedHosts        []string          `json:"blocked_hosts"`
    AllowedPorts        []int             `json:"allowed_ports"`
    BlockedPorts        []int             `json:"blocked_ports"`
    RequireSSHKeys      bool              `json:"require_ssh_keys"`

    // Module validation rules
    AllowedModules      []string          `json:"allowed_modules"`
    BlockedModules      []string          `json:"blocked_modules"`
    DangerousPatterns   []string          `json:"dangerous_patterns"`

    // File operation rules
    AllowedPaths        []string          `json:"allowed_paths"`
    BlockedPaths        []string          `json:"blocked_paths"`
    RequireBackup       bool              `json:"require_backup"`

    // Command execution rules
    AllowedCommands     []string          `json:"allowed_commands"`
    BlockedCommands     []string          `json:"blocked_commands"`
    RequireSudo         bool              `json:"require_sudo"`

    // General rules
    MaxSecurityScore    int               `json:"max_security_score"`
    StrictMode          bool              `json:"strict_mode"`
    AuditMode           bool              `json:"audit_mode"`
}

// Usage example
func ExampleSecurityValidation() {
    validator := security.NewValidator()

    // Configure security rules
    rules := security.SecurityRules{
        AllowedHosts:      []string{"192.168.1.*", "10.0.0.*"},
        BlockedModules:    []string{"shell", "command"},
        RequireSSHKeys:    true,
        RequireBackup:     true,
        MaxSecurityScore:  80,
        StrictMode:        true,
    }
    validator.SetRules(rules)

    // Validate host
    host := types.Host{
        Name:     "webserver",
        Address:  "192.168.1.100",
        User:     "admin",
        KeyFile:  "/path/to/key",
    }

    result := validator.ValidateHost(host)
    if !result.Valid {
        log.Printf("Host validation failed: %+v", result.Violations)
    }

    // Validate task
    task := types.Task{
        Name:   "Install package",
        Module: "package",
        Args: map[string]interface{}{
            "name":  "nginx",
            "state": "present",
        },
    }

    result = validator.ValidateTask(task)
    if !result.Valid {
        log.Printf("Task validation failed: %+v", result.Violations)
    }
}
```

### Security Auditor

```go
package security

// SecurityAuditor provides security auditing capabilities
type SecurityAuditor interface {
    // Audit host access
    AuditHost(host types.Host, user string) AuditEntry

    // Audit task execution
    AuditTask(task types.Task, user string) AuditEntry

    // Audit playbook execution
    AuditPlaybook(playbook types.Playbook, user string) AuditEntry

    // Get audit log
    GetAuditLog() []AuditEntry

    // Get audit log by criteria
    GetAuditLogByCriteria(criteria AuditCriteria) []AuditEntry

    // Export audit log
    ExportAuditLog(format string) ([]byte, error)

    // Clear audit log
    ClearAuditLog() error
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
    ID          string                 `json:"id"`
    Timestamp   time.Time              `json:"timestamp"`
    Type        string                 `json:"type"`
    User        string                 `json:"user"`
    Host        string                 `json:"host"`
    Action      string                 `json:"action"`
    Resource    string                 `json:"resource"`
    Result      string                 `json:"result"`
    SecurityScore int                  `json:"security_score"`
    Violations  []SecurityViolation    `json:"violations"`
    Metadata    map[string]interface{} `json:"metadata"`
}

// AuditCriteria defines criteria for audit log queries
type AuditCriteria struct {
    StartTime   *time.Time `json:"start_time,omitempty"`
    EndTime     *time.Time `json:"end_time,omitempty"`
    User        string     `json:"user,omitempty"`
    Host        string     `json:"host,omitempty"`
    Type        string     `json:"type,omitempty"`
    Action      string     `json:"action,omitempty"`
    MinScore    int        `json:"min_score,omitempty"`
    MaxScore    int        `json:"max_score,omitempty"`
    HasViolations bool     `json:"has_violations,omitempty"`
}
```

## 📊 Monitoring API

### Metrics Collector

```go
package monitoring

import (
    "time"
)

// MetricsCollector collects and manages metrics
type MetricsCollector interface {
    // Record a metric
    RecordMetric(name string, value interface{}, labels map[string]string) error

    // Increment a counter
    IncrementCounter(name string, labels map[string]string) error

    // Set a gauge value
    SetGauge(name string, value float64, labels map[string]string) error

    // Record a histogram value
    RecordHistogram(name string, value float64, labels map[string]string) error

    // Record a timer
    RecordTimer(name string, duration time.Duration, labels map[string]string) error

    // Get all metrics
    GetMetrics() []Metric

    // Get metrics by name
    GetMetricsByName(name string) []Metric

    // Get system metrics
    GetSystemMetrics() SystemMetrics

    // Start background collection
    StartCollection(interval time.Duration) error

    // Stop background collection
    StopCollection() error

    // Export metrics
    ExportMetrics(format string) ([]byte, error)
}

// Metric represents a single metric
type Metric struct {
    Name      string                 `json:"name"`
    Type      string                 `json:"type"`
    Value     interface{}            `json:"value"`
    Labels    map[string]string      `json:"labels"`
    Timestamp time.Time              `json:"timestamp"`
    Metadata  map[string]interface{} `json:"metadata"`
}

// SystemMetrics contains system-level metrics
type SystemMetrics struct {
    // Memory metrics
    Memory struct {
        Alloc      uint64 `json:"alloc"`
        TotalAlloc uint64 `json:"total_alloc"`
        Sys        uint64 `json:"sys"`
        Lookups    uint64 `json:"lookups"`
        Mallocs    uint64 `json:"mallocs"`
        Frees      uint64 `json:"frees"`
    } `json:"memory"`

    // Goroutine metrics
    Goroutines struct {
        Total   int `json:"total"`
        Running int `json:"running"`
        Waiting int `json:"waiting"`
    } `json:"goroutines"`

    // GC metrics
    GC struct {
        NumGC        uint32        `json:"num_gc"`
        PauseTotal   time.Duration `json:"pause_total"`
        PauseNs      []uint64      `json:"pause_ns"`
        LastGC       time.Time     `json:"last_gc"`
        NextGC       uint64        `json:"next_gc"`
        EnabledGC    bool          `json:"enabled_gc"`
    } `json:"gc"`

    // Runtime metrics
    Runtime struct {
        Version   string        `json:"version"`
        GOOS      string        `json:"goos"`
        GOARCH    string        `json:"goarch"`
        NumCPU    int           `json:"num_cpu"`
        Uptime    time.Duration `json:"uptime"`
    } `json:"runtime"`
}

// Usage example
func ExampleMetricsUsage() {
    collector := monitoring.NewCollector()

    // Start background collection
    collector.StartCollection(time.Second * 30)
    defer collector.StopCollection()

    // Record custom metrics
    labels := map[string]string{
        "host":   "webserver01",
        "module": "package",
    }

    collector.IncrementCounter("tasks_executed", labels)
    collector.SetGauge("active_connections", 42.0, labels)
    collector.RecordTimer("task_duration", time.Second*5, labels)

    // Get metrics
    metrics := collector.GetMetrics()
    for _, metric := range metrics {
        log.Printf("Metric: %s = %v", metric.Name, metric.Value)
    }

    // Export metrics
    prometheusData, err := collector.ExportMetrics("prometheus")
    if err != nil {
        log.Printf("Failed to export metrics: %v", err)
    } else {
        log.Printf("Prometheus metrics: %s", string(prometheusData))
    }
}
```

### Metrics Reporter

```go
package monitoring

// MetricsReporter generates reports from collected metrics
type MetricsReporter interface {
    // Generate a report
    GenerateReport(criteria ReportCriteria) (Report, error)

    // Get all reports
    GetReports() []Report

    // Schedule periodic reports
    ScheduleReport(schedule string, criteria ReportCriteria) error

    // Export report
    ExportReport(report Report, format string) ([]byte, error)

    // Get report templates
    GetTemplates() []ReportTemplate

    // Create custom template
    CreateTemplate(template ReportTemplate) error
}

// Report represents a metrics report
type Report struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    GeneratedAt time.Time              `json:"generated_at"`
    Period      ReportPeriod           `json:"period"`
    Summary     ReportSummary          `json:"summary"`
    Sections    []ReportSection        `json:"sections"`
    Metadata    map[string]interface{} `json:"metadata"`
}

// ReportCriteria defines report generation criteria
type ReportCriteria struct {
    Name        string       `json:"name"`
    StartTime   time.Time    `json:"start_time"`
    EndTime     time.Time    `json:"end_time"`
    Metrics     []string     `json:"metrics"`
    Hosts       []string     `json:"hosts"`
    Modules     []string     `json:"modules"`
    Template    string       `json:"template"`
    Format      string       `json:"format"`
}

// ReportSummary contains report summary information
type ReportSummary struct {
    TotalMetrics    int                    `json:"total_metrics"`
    TotalHosts      int                    `json:"total_hosts"`
    TotalTasks      int                    `json:"total_tasks"`
    SuccessRate     float64                `json:"success_rate"`
    AverageLatency  time.Duration          `json:"average_latency"`
    TopMetrics      []MetricSummary        `json:"top_metrics"`
    Alerts          []Alert                `json:"alerts"`
    Recommendations []string               `json:"recommendations"`
}
```

## 🌊 Workflow API

### Workflow Orchestrator

```go
package workflow

import (
    "context"
    "time"
)

// WorkflowOrchestrator manages workflow execution
type WorkflowOrchestrator interface {
    // Execute a workflow
    ExecuteWorkflow(ctx context.Context, workflow Workflow) (WorkflowResult, error)

    // Schedule a workflow
    ScheduleWorkflow(workflow Workflow, schedule Schedule) error

    // Cancel a workflow
    CancelWorkflow(workflowID string) error

    // Get workflow status
    GetWorkflowStatus(workflowID string) (WorkflowStatus, error)

    // List active workflows
    ListActiveWorkflows() []WorkflowStatus

    // Get workflow history
    GetWorkflowHistory(workflowID string) ([]WorkflowExecution, error)

    // Register workflow event handler
    RegisterEventHandler(eventType string, handler EventHandler) error

    // Unregister workflow event handler
    UnregisterEventHandler(eventType string) error
}

// Workflow defines a workflow structure
type Workflow struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Version     string                 `json:"version"`
    Steps       []WorkflowStep         `json:"steps"`
    Variables   map[string]interface{} `json:"variables"`
    Triggers    []WorkflowTrigger      `json:"triggers"`
    Timeout     time.Duration          `json:"timeout"`
    RetryPolicy RetryPolicy            `json:"retry_policy"`
    Metadata    map[string]interface{} `json:"metadata"`
}

// WorkflowStep represents a single workflow step
type WorkflowStep struct {
    ID           string                 `json:"id"`
    Name         string                 `json:"name"`
    Type         string                 `json:"type"`
    Config       map[string]interface{} `json:"config"`
    Dependencies []string               `json:"dependencies"`
    Conditions   []string               `json:"conditions"`
    Timeout      time.Duration          `json:"timeout"`
    RetryPolicy  RetryPolicy            `json:"retry_policy"`
    OnSuccess    []string               `json:"on_success"`
    OnFailure    []string               `json:"on_failure"`
    Metadata     map[string]interface{} `json:"metadata"`
}

// WorkflowResult contains workflow execution results
type WorkflowResult struct {
    WorkflowID   string                 `json:"workflow_id"`
    Status       string                 `json:"status"`
    StartTime    time.Time              `json:"start_time"`
    EndTime      time.Time              `json:"end_time"`
    Duration     time.Duration          `json:"duration"`
    StepResults  []StepResult           `json:"step_results"`
    Variables    map[string]interface{} `json:"variables"`
    Error        string                 `json:"error,omitempty"`
    Metadata     map[string]interface{} `json:"metadata"`
}

// Usage example
func ExampleWorkflowUsage() {
    orchestrator := workflow.NewOrchestrator()

    // Define workflow
    wf := workflow.Workflow{
        ID:   "deploy-app",
        Name: "Deploy Application",
        Steps: []workflow.WorkflowStep{
            {
                ID:   "backup-db",
                Name: "Backup Database",
                Type: "task",
                Config: map[string]interface{}{
                    "playbook": "backup.yml",
                    "inventory": "production",
                },
            },
            {
                ID:   "deploy-code",
                Name: "Deploy Code",
                Type: "task",
                Dependencies: []string{"backup-db"},
                Config: map[string]interface{}{
                    "playbook": "deploy.yml",
                    "inventory": "production",
                },
            },
            {
                ID:   "health-check",
                Name: "Health Check",
                Type: "condition",
                Dependencies: []string{"deploy-code"},
                Config: map[string]interface{}{
                    "url": "https://app.example.com/health",
                    "expected_status": 200,
                },
            },
        },
        Timeout: time.Hour,
        RetryPolicy: workflow.RetryPolicy{
            MaxAttempts: 3,
            Delay:       time.Minute,
            BackoffType: "exponential",
        },
    }

    // Execute workflow
    ctx := context.Background()
    result, err := orchestrator.ExecuteWorkflow(ctx, wf)
    if err != nil {
        log.Printf("Workflow execution failed: %v", err)
        return
    }

    log.Printf("Workflow completed: %s", result.Status)
    for _, stepResult := range result.StepResults {
        log.Printf("Step %s: %s", stepResult.StepID, stepResult.Status)
    }
}
```

### Workflow Scheduler

```go
package workflow

// WorkflowScheduler manages workflow scheduling
type WorkflowScheduler interface {
    // Schedule a workflow
    Schedule(workflow Workflow, schedule Schedule) error

    // Unschedule a workflow
    Unschedule(workflowID string) error

    // List scheduled workflows
    ListScheduled() []ScheduledWorkflow

    // Get next execution time
    GetNextExecution(workflowID string) (time.Time, error)

    // Start scheduler
    Start() error

    // Stop scheduler
    Stop() error

    // Get scheduler status
    GetStatus() SchedulerStatus
}

// Schedule defines when a workflow should run
type Schedule struct {
    Type        string                 `json:"type"`
    Expression  string                 `json:"expression"`
    Timezone    string                 `json:"timezone"`
    StartTime   *time.Time             `json:"start_time,omitempty"`
    EndTime     *time.Time             `json:"end_time,omitempty"`
    MaxRuns     int                    `json:"max_runs,omitempty"`
    Enabled     bool                   `json:"enabled"`
    Metadata    map[string]interface{} `json:"metadata"`
}

// ScheduledWorkflow represents a scheduled workflow
type ScheduledWorkflow struct {
    WorkflowID    string    `json:"workflow_id"`
    Schedule      Schedule  `json:"schedule"`
    NextExecution time.Time `json:"next_execution"`
    LastExecution time.Time `json:"last_execution"`
    RunCount      int       `json:"run_count"`
    Status        string    `json:"status"`
}
```

## 🔌 Plugin API

### Plugin Manager

```go
package plugins

// PluginManager manages plugin lifecycle
type PluginManager interface {
    // Load plugin from file
    LoadPlugin(path string) (Plugin, error)

    // Unload plugin
    UnloadPlugin(name string) error

    // List loaded plugins
    ListPlugins() []PluginInfo

    // Get plugin by name
    GetPlugin(name string) (Plugin, error)

    // Register plugin
    RegisterPlugin(plugin Plugin) error

    // Unregister plugin
    UnregisterPlugin(name string) error

    // Validate plugin
    ValidatePlugin(plugin Plugin) error

    // Get plugin dependencies
    GetDependencies(name string) ([]string, error)

    // Resolve plugin dependencies
    ResolveDependencies(plugins []string) ([]string, error)
}

// Plugin defines the base plugin interface
type Plugin interface {
    // Plugin metadata
    Name() string
    Version() string
    Description() string
    Author() string
    License() string

    // Plugin lifecycle
    Initialize(config map[string]interface{}) error
    Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
    Cleanup() error

    // Plugin validation
    ValidateArgs(args map[string]interface{}) error
    GetSchema() map[string]interface{}

    // Plugin dependencies
    GetDependencies() []string
    GetRequirements() map[string]string
}

// PluginInfo contains plugin information
type PluginInfo struct {
    Name         string                 `json:"name"`
    Version      string                 `json:"version"`
    Description  string                 `json:"description"`
    Author       string                 `json:"author"`
    License      string                 `json:"license"`
    Type         string                 `json:"type"`
    Status       string                 `json:"status"`
    LoadedAt     time.Time              `json:"loaded_at"`
    Dependencies []string               `json:"dependencies"`
    Requirements map[string]string      `json:"requirements"`
    Metadata     map[string]interface{} `json:"metadata"`
}

// Usage example
func ExamplePluginUsage() {
    manager := plugins.NewManager()

    // Load plugin
    plugin, err := manager.LoadPlugin("./plugins/custom-plugin.so")
    if err != nil {
        log.Printf("Failed to load plugin: %v", err)
        return
    }

    // Initialize plugin
    config := map[string]interface{}{
        "timeout": "30s",
        "retries": 3,
    }

    if err := plugin.Initialize(config); err != nil {
        log.Printf("Failed to initialize plugin: %v", err)
        return
    }

    // Execute plugin
    args := map[string]interface{}{
        "action": "process",
        "data":   "example data",
    }

    ctx := context.Background()
    result, err := plugin.Execute(ctx, args)
    if err != nil {
        log.Printf("Plugin execution failed: %v", err)
        return
    }

    log.Printf("Plugin result: %+v", result)

    // Cleanup
    defer plugin.Cleanup()
}
```

## 🌐 REST API

### HTTP Server

Onigirazu provides a REST API for remote management and integration.

#### Base URL

```
http://localhost:8080/api/v1
```

#### Authentication

```bash
# API Key authentication
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/v1/status

# Bearer token authentication
curl -H "Authorization: Bearer your-token" http://localhost:8080/api/v1/status
```

#### Endpoints

##### System Status

```http
GET /api/v1/status
```

Response:

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "2h30m15s",
  "system": {
    "memory_usage": 45.2,
    "cpu_usage": 12.5,
    "goroutines": 42
  }
}
```

##### Execute Playbook

```http
POST /api/v1/playbooks/execute
Content-Type: application/json

{
  "playbook": "deploy.yml",
  "inventory": "production.yml",
  "variables": {
    "app_version": "v1.2.3",
    "environment": "production"
  },
  "options": {
    "check_mode": false,
    "verbose": 1
  }
}
```

Response:

```json
{
  "execution_id": "exec-123456",
  "status": "running",
  "started_at": "2023-10-01T10:30:00Z",
  "playbook": "deploy.yml",
  "inventory": "production.yml"
}
```

##### Get Execution Status

```http
GET /api/v1/executions/{execution_id}
```

Response:

```json
{
  "execution_id": "exec-123456",
  "status": "completed",
  "started_at": "2023-10-01T10:30:00Z",
  "completed_at": "2023-10-01T10:35:30Z",
  "duration": "5m30s",
  "result": {
    "success": true,
    "total_tasks": 15,
    "changed_tasks": 8,
    "failed_tasks": 0
  }
}
```

##### List Inventories

```http
GET /api/v1/inventories
```

##### Get Inventory

```http
GET /api/v1/inventories/{name}
```

##### List Playbooks

```http
GET /api/v1/playbooks
```

##### Get Metrics

```http
GET /api/v1/metrics
```

##### Get Security Audit Log

```http
GET /api/v1/security/audit
```

## 📚 Client Libraries

### Go Client

```go
package main

import (
    "context"
    "log"

    "github.com/your-org/onigirazu/pkg/client"
)

func main() {
    // Create client
    config := client.Config{
        BaseURL: "http://localhost:8080",
        APIKey:  "your-api-key",
        Timeout: time.Minute * 5,
    }

    client, err := client.New(config)
    if err != nil {
        log.Fatal(err)
    }

    // Execute playbook
    request := client.ExecutePlaybookRequest{
        Playbook:  "deploy.yml",
        Inventory: "production.yml",
        Variables: map[string]interface{}{
            "app_version": "v1.2.3",
        },
        Options: client.ExecutionOptions{
            CheckMode: false,
            Verbose:   1,
        },
    }

    ctx := context.Background()
    execution, err := client.ExecutePlaybook(ctx, request)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Execution started: %s", execution.ID)

    // Wait for completion
    result, err := client.WaitForExecution(ctx, execution.ID)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Execution completed: %+v", result)
}
```

### Python Client

```python
import onigirazu

# Create client
client = onigirazu.Client(
    base_url="http://localhost:8080",
    api_key="your-api-key"
)

# Execute playbook
execution = client.execute_playbook(
    playbook="deploy.yml",
    inventory="production.yml",
    variables={
        "app_version": "v1.2.3"
    },
    options={
        "check_mode": False,
        "verbose": 1
    }
)

print(f"Execution started: {execution.id}")

# Wait for completion
result = client.wait_for_execution(execution.id)
print(f"Execution completed: {result.status}")
```

### JavaScript Client

```javascript
const OnigirasuClient = require('@onigirazu/client');

// Create client
const client = new OnigirasuClient({
    baseURL: 'http://localhost:8080',
    apiKey: 'your-api-key'
});

// Execute playbook
async function deployApp() {
    try {
        const execution = await client.executePlaybook({
            playbook: 'deploy.yml',
            inventory: 'production.yml',
            variables: {
                app_version: 'v1.2.3'
            },
            options: {
                checkMode: false,
                verbose: 1
            }
        });

        console.log(`Execution started: ${execution.id}`);

        // Wait for completion
        const result = await client.waitForExecution(execution.id);
        console.log(`Execution completed: ${result.status}`);

    } catch (error) {
        console.error('Execution failed:', error);
    }
}

deployApp();
```

This comprehensive API reference provides detailed information about all programmatic interfaces available in Onigirazu, including core APIs, module development, security validation, monitoring, workflow orchestration, plugin management, and client libraries for multiple programming languages.
