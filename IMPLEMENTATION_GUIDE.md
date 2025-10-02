# Implementation Guide - Optimization Recommendations

This guide provides concrete code examples for implementing the recommended optimizations.

---

## 1. Test Coverage Implementation

### 1.1 Engine Module Tests (Priority: CRITICAL)

Create `/internal/engine/execution_engine_test.go`:

```go
package engine

import (
    "context"
    "testing"
    "time"

    "github.com/onigirazu-cfg/onigirazu/pkg/types"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock executor for testing
type MockExecutor struct {
    mock.Mock
}

func (m *MockExecutor) Execute(ctx context.Context, task types.Task) error {
    args := m.Called(ctx, task)
    return args.Error(0)
}

func TestExecutionEngine_ExecuteTask(t *testing.T) {
    tests := []struct {
        name    string
        task    types.Task
        wantErr bool
    }{
        {
            name: "successful task execution",
            task: types.Task{
                Name:   "test task",
                Module: "command",
                Args: map[string]interface{}{
                    "command": "echo hello",
                },
            },
            wantErr: false,
        },
        {
            name: "task execution with timeout",
            task: types.Task{
                Name:    "timeout task",
                Module:  "command",
                Timeout: 1 * time.Second,
                Args: map[string]interface{}{
                    "command": "sleep 10",
                },
            },
            wantErr: true,
        },
        {
            name: "task execution with retries",
            task: types.Task{
                Name:    "retry task",
                Module:  "command",
                Retries: 3,
                Args: map[string]interface{}{
                    "command": "exit 1",
                },
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            engine := NewExecutionEngine()
            ctx := context.Background()

            err := engine.ExecuteTask(ctx, tt.task)

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func TestExecutionEngine_ParallelExecution(t *testing.T) {
    engine := NewExecutionEngine()
    ctx := context.Background()

    tasks := []types.Task{
        {Name: "task1", Module: "command", Args: map[string]interface{}{"command": "echo 1"}},
        {Name: "task2", Module: "command", Args: map[string]interface{}{"command": "echo 2"}},
        {Name: "task3", Module: "command", Args: map[string]interface{}{"command": "echo 3"}},
    }

    start := time.Now()
    results := engine.ExecuteParallel(ctx, tasks)
    duration := time.Since(start)

    assert.Len(t, results, 3)
    assert.Less(t, duration, 1*time.Second, "Parallel execution should be faster")
}

func TestExecutionEngine_ErrorHandling(t *testing.T) {
    engine := NewExecutionEngine()
    ctx := context.Background()

    task := types.Task{
        Name:   "failing task",
        Module: "command",
        Args: map[string]interface{}{
            "command": "exit 1",
        },
    }

    err := engine.ExecuteTask(ctx, task)
    assert.Error(t, err)

    // Verify error contains useful information
    assert.Contains(t, err.Error(), "task")
    assert.Contains(t, err.Error(), "failed")
}

func TestExecutionEngine_ContextCancellation(t *testing.T) {
    engine := NewExecutionEngine()
    ctx, cancel := context.WithCancel(context.Background())

    task := types.Task{
        Name:   "long running task",
        Module: "command",
        Args: map[string]interface{}{
            "command": "sleep 60",
        },
    }

    // Cancel context after 100ms
    go func() {
        time.Sleep(100 * time.Millisecond)
        cancel()
    }()

    err := engine.ExecuteTask(ctx, task)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "context")
}
```

### 1.2 Parser Module Tests (Priority: CRITICAL)

Create `/internal/parser/parser_test.go`:

```go
package parser

import (
    "os"
    "testing"

    "github.com/onigirazu-cfg/onigirazu/pkg/types"
    "github.com/stretchr/testify/assert"
)

func TestParser_ParsePlaybook(t *testing.T) {
    tests := []struct {
        name    string
        yaml    string
        wantErr bool
    }{
        {
            name: "valid playbook",
            yaml: `
name: Test Playbook
plays:
  - name: Test Play
    hosts: localhost
    tasks:
      - name: Test Task
        module: command
        args:
          command: echo hello
`,
            wantErr: false,
        },
        {
            name: "invalid yaml",
            yaml: `
name: Test Playbook
plays:
  - name: Test Play
    hosts: localhost
    tasks:
      - name: Test Task
        module: command
        args:
          command: echo hello
        invalid_indent
`,
            wantErr: true,
        },
        {
            name:    "empty playbook",
            yaml:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := NewParser()

            playbook, err := parser.ParseYAML([]byte(tt.yaml))

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, playbook)
            }
        })
    }
}

func TestParser_ParseFile(t *testing.T) {
    // Create temporary test file
    tmpfile, err := os.CreateTemp("", "playbook-*.yml")
    assert.NoError(t, err)
    defer os.Remove(tmpfile.Name())

    content := `
name: Test Playbook
plays:
  - name: Test Play
    hosts: localhost
    tasks:
      - name: Test Task
        module: command
        args:
          command: echo hello
`
    _, err = tmpfile.Write([]byte(content))
    assert.NoError(t, err)
    tmpfile.Close()

    parser := NewParser()
    playbook, err := parser.ParseFile(tmpfile.Name())

    assert.NoError(t, err)
    assert.NotNil(t, playbook)
    assert.Equal(t, "Test Playbook", playbook.Name)
}

func TestParser_LargeFile(t *testing.T) {
    // Test parsing large playbook
    parser := NewParser()

    // Generate large playbook
    largeYAML := generateLargePlaybook(1000) // 1000 tasks

    start := time.Now()
    playbook, err := parser.ParseYAML([]byte(largeYAML))
    duration := time.Since(start)

    assert.NoError(t, err)
    assert.NotNil(t, playbook)
    assert.Less(t, duration, 5*time.Second, "Large file parsing should complete in <5s")
}

func generateLargePlaybook(numTasks int) string {
    yaml := "name: Large Playbook\nplays:\n  - name: Large Play\n    hosts: localhost\n    tasks:\n"
    for i := 0; i < numTasks; i++ {
        yaml += fmt.Sprintf("      - name: Task %d\n        module: command\n        args:\n          command: echo %d\n", i, i)
    }
    return yaml
}
```

### 1.3 Inventory Module Tests (Priority: CRITICAL)

Create `/internal/inventory/inventory_test.go`:

```go
package inventory

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestInventory_AddHost(t *testing.T) {
    inv := NewInventory()

    host := &Host{
        Name:    "server1",
        Address: "192.168.1.10",
        Port:    22,
        User:    "admin",
    }

    err := inv.AddHost(host)
    assert.NoError(t, err)

    // Verify host was added
    retrieved, err := inv.GetHost("server1")
    assert.NoError(t, err)
    assert.Equal(t, host.Address, retrieved.Address)
}

func TestInventory_AddGroup(t *testing.T) {
    inv := NewInventory()

    group := &Group{
        Name: "webservers",
        Hosts: []string{"web1", "web2", "web3"},
        Vars: map[string]interface{}{
            "http_port": 80,
        },
    }

    err := inv.AddGroup(group)
    assert.NoError(t, err)

    // Verify group was added
    retrieved, err := inv.GetGroup("webservers")
    assert.NoError(t, err)
    assert.Len(t, retrieved.Hosts, 3)
}

func TestInventory_HostPattern(t *testing.T) {
    inv := NewInventory()

    // Add test hosts
    hosts := []string{"web1", "web2", "db1", "db2"}
    for _, name := range hosts {
        inv.AddHost(&Host{Name: name})
    }

    tests := []struct {
        pattern string
        want    []string
    }{
        {"web*", []string{"web1", "web2"}},
        {"db*", []string{"db1", "db2"}},
        {"*1", []string{"web1", "db1"}},
        {"all", []string{"web1", "web2", "db1", "db2"}},
    }

    for _, tt := range tests {
        t.Run(tt.pattern, func(t *testing.T) {
            matched := inv.MatchHosts(tt.pattern)
            assert.ElementsMatch(t, tt.want, matched)
        })
    }
}

func TestInventory_VariableInheritance(t *testing.T) {
    inv := NewInventory()

    // Add group with variables
    inv.AddGroup(&Group{
        Name: "webservers",
        Hosts: []string{"web1"},
        Vars: map[string]interface{}{
            "http_port": 80,
            "app_name":  "myapp",
        },
    })

    // Add host with variables
    inv.AddHost(&Host{
        Name: "web1",
        Vars: map[string]interface{}{
            "http_port": 8080, // Override group variable
        },
    })

    // Get merged variables
    vars := inv.GetHostVars("web1")

    assert.Equal(t, 8080, vars["http_port"], "Host variable should override group variable")
    assert.Equal(t, "myapp", vars["app_name"], "Group variable should be inherited")
}
```

---

## 2. Performance Optimization

### 2.1 Mutex Profiling

Add to your test files:

```go
func TestMain(m *testing.M) {
    // Enable mutex profiling
    runtime.SetMutexProfileFraction(1)

    code := m.Run()

    // Write mutex profile
    f, err := os.Create("mutex.prof")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    if err := pprof.Lookup("mutex").WriteTo(f, 0); err != nil {
        log.Fatal(err)
    }

    os.Exit(code)
}
```

Run tests and analyze:

```bash
go test -mutexprofile=mutex.prof ./...
go tool pprof mutex.prof
```

### 2.2 Connection Pool Metrics

Add to `/internal/ssh/pool.go`:

```go
type PoolMetrics struct {
    TotalConnections    int64
    ActiveConnections   int64
    IdleConnections     int64
    ConnectionsCreated  int64
    ConnectionsReused   int64
    ConnectionErrors    int64
    AverageWaitTime     time.Duration
}

func (p *ConnectionPool) GetMetrics() PoolMetrics {
    p.mu.RLock()
    defer p.mu.RUnlock()

    return PoolMetrics{
        TotalConnections:   int64(len(p.connections)),
        ActiveConnections:  p.activeCount,
        IdleConnections:    int64(len(p.connections)) - p.activeCount,
        ConnectionsCreated: p.stats.created,
        ConnectionsReused:  p.stats.reused,
        ConnectionErrors:   p.stats.errors,
        AverageWaitTime:    p.stats.avgWaitTime,
    }
}

// Add Prometheus metrics
func (p *ConnectionPool) RegisterMetrics(registry *prometheus.Registry) {
    poolSize := prometheus.NewGaugeFunc(
        prometheus.GaugeOpts{
            Name: "ssh_pool_size",
            Help: "Current size of SSH connection pool",
        },
        func() float64 {
            metrics := p.GetMetrics()
            return float64(metrics.TotalConnections)
        },
    )

    activeConnections := prometheus.NewGaugeFunc(
        prometheus.GaugeOpts{
            Name: "ssh_active_connections",
            Help: "Number of active SSH connections",
        },
        func() float64 {
            metrics := p.GetMetrics()
            return float64(metrics.ActiveConnections)
        },
    )

    registry.MustRegister(poolSize, activeConnections)
}
```

### 2.3 Cache Optimization with LRU

Create `/internal/cache/lru_cache.go`:

```go
package cache

import (
    "container/list"
    "sync"
    "time"
)

type LRUCache struct {
    capacity int
    items    map[string]*list.Element
    lru      *list.List
    mu       sync.RWMutex
    stats    CacheStats
}

type cacheEntry struct {
    key       string
    value     interface{}
    expiresAt time.Time
}

type CacheStats struct {
    Hits        int64
    Misses      int64
    Evictions   int64
    Expirations int64
}

func NewLRUCache(capacity int) *LRUCache {
    return &LRUCache{
        capacity: capacity,
        items:    make(map[string]*list.Element),
        lru:      list.New(),
    }
}

func (c *LRUCache) Get(key string) (interface{}, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()

    elem, exists := c.items[key]
    if !exists {
        c.stats.Misses++
        return nil, false
    }

    entry := elem.Value.(*cacheEntry)

    // Check expiration
    if time.Now().After(entry.expiresAt) {
        c.removeElement(elem)
        c.stats.Expirations++
        c.stats.Misses++
        return nil, false
    }

    // Move to front (most recently used)
    c.lru.MoveToFront(elem)
    c.stats.Hits++

    return entry.value, true
}

func (c *LRUCache) Set(key string, value interface{}, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()

    expiresAt := time.Now().Add(ttl)

    // Update existing entry
    if elem, exists := c.items[key]; exists {
        c.lru.MoveToFront(elem)
        entry := elem.Value.(*cacheEntry)
        entry.value = value
        entry.expiresAt = expiresAt
        return
    }

    // Add new entry
    entry := &cacheEntry{
        key:       key,
        value:     value,
        expiresAt: expiresAt,
    }

    elem := c.lru.PushFront(entry)
    c.items[key] = elem

    // Evict if over capacity
    if c.lru.Len() > c.capacity {
        c.evictOldest()
    }
}

func (c *LRUCache) evictOldest() {
    elem := c.lru.Back()
    if elem != nil {
        c.removeElement(elem)
        c.stats.Evictions++
    }
}

func (c *LRUCache) removeElement(elem *list.Element) {
    c.lru.Remove(elem)
    entry := elem.Value.(*cacheEntry)
    delete(c.items, entry.key)
}

func (c *LRUCache) GetStats() CacheStats {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.stats
}

func (c *LRUCache) HitRate() float64 {
    c.mu.RLock()
    defer c.mu.RUnlock()

    total := c.stats.Hits + c.stats.Misses
    if total == 0 {
        return 0
    }

    return float64(c.stats.Hits) / float64(total)
}
```

---

## 3. Observability Implementation

### 3.1 Prometheus Metrics

Create `/internal/metrics/prometheus.go`:

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    TasksExecuted = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "onigirazu_tasks_executed_total",
            Help: "Total number of tasks executed",
        },
        []string{"module", "status"},
    )

    TaskDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "onigirazu_task_duration_seconds",
            Help:    "Task execution duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"module"},
    )

    ActiveTasks = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "onigirazu_active_tasks",
            Help: "Number of currently executing tasks",
        },
    )

    SecurityViolations = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "onigirazu_security_violations_total",
            Help: "Total number of security violations detected",
        },
        []string{"rule", "severity"},
    )

    CacheHitRate = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "onigirazu_cache_hit_rate",
            Help: "Cache hit rate (0-1)",
        },
    )
)

// RecordTaskExecution records task execution metrics
func RecordTaskExecution(module, status string, duration time.Duration) {
    TasksExecuted.WithLabelValues(module, status).Inc()
    TaskDuration.WithLabelValues(module).Observe(duration.Seconds())
}

// RecordSecurityViolation records security violation
func RecordSecurityViolation(rule, severity string) {
    SecurityViolations.WithLabelValues(rule, severity).Inc()
}
```

### 3.2 Distributed Tracing with OpenTelemetry

Create `/internal/tracing/tracer.go`:

```go
package tracing

import (
    "context"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("onigirazu")

// StartTaskSpan starts a new span for task execution
func StartTaskSpan(ctx context.Context, taskName, module string) (context.Context, trace.Span) {
    return tracer.Start(ctx, "task.execute",
        trace.WithAttributes(
            attribute.String("task.name", taskName),
            attribute.String("task.module", module),
        ),
    )
}

// StartSSHSpan starts a new span for SSH operations
func StartSSHSpan(ctx context.Context, host, operation string) (context.Context, trace.Span) {
    return tracer.Start(ctx, "ssh."+operation,
        trace.WithAttributes(
            attribute.String("ssh.host", host),
            attribute.String("ssh.operation", operation),
        ),
    )
}

// RecordError records an error in the current span
func RecordError(span trace.Span, err error) {
    span.RecordError(err)
    span.SetAttributes(attribute.Bool("error", true))
}
```

Usage in engine:

```go
func (e *ExecutionEngine) ExecuteTask(ctx context.Context, task types.Task) error {
    // Start tracing span
    ctx, span := tracing.StartTaskSpan(ctx, task.Name, task.Module)
    defer span.End()

    // Record metrics
    start := time.Now()
    metrics.ActiveTasks.Inc()
    defer metrics.ActiveTasks.Dec()

    // Execute task
    err := e.executeTaskInternal(ctx, task)

    // Record results
    duration := time.Since(start)
    status := "success"
    if err != nil {
        status = "failure"
        tracing.RecordError(span, err)
    }

    metrics.RecordTaskExecution(task.Module, status, duration)

    return err
}
```

---

## 4. Benchmarking Implementation

### 4.1 Performance Benchmarks

Create `/internal/engine/benchmark_test.go`:

```go
package engine

import (
    "context"
    "testing"

    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func BenchmarkTaskExecution(b *testing.B) {
    engine := NewExecutionEngine()
    ctx := context.Background()

    task := types.Task{
        Name:   "benchmark task",
        Module: "command",
        Args: map[string]interface{}{
            "command": "echo hello",
        },
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        engine.ExecuteTask(ctx, task)
    }
}

func BenchmarkParallelExecution(b *testing.B) {
    engine := NewExecutionEngine()
    ctx := context.Background()

    tasks := make([]types.Task, 10)
    for i := range tasks {
        tasks[i] = types.Task{
            Name:   fmt.Sprintf("task-%d", i),
            Module: "command",
            Args: map[string]interface{}{
                "command": "echo hello",
            },
        }
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        engine.ExecuteParallel(ctx, tasks)
    }
}

func BenchmarkSSHConnection(b *testing.B) {
    pool := NewConnectionPool(10)

    config := &SSHConfig{
        Host: "localhost",
        Port: 22,
        User: "test",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        conn, _ := pool.Get(config)
        pool.Put(conn)
    }
}

func BenchmarkCacheOperations(b *testing.B) {
    cache := NewLRUCache(1000)

    b.Run("Set", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            cache.Set(fmt.Sprintf("key-%d", i), "value", 1*time.Hour)
        }
    })

    b.Run("Get", func(b *testing.B) {
        // Populate cache
        for i := 0; i < 1000; i++ {
            cache.Set(fmt.Sprintf("key-%d", i), "value", 1*time.Hour)
        }

        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            cache.Get(fmt.Sprintf("key-%d", i%1000))
        }
    })
}
```

Run benchmarks:

```bash
go test -bench=. -benchmem -cpuprofile=cpu.prof ./internal/engine/
go tool pprof cpu.prof
```

---

## 5. Context Propagation

### 5.1 Context-Aware Execution

Update engine to use context properly:

```go
func (e *ExecutionEngine) ExecuteTask(ctx context.Context, task types.Task) error {
    // Create timeout context if task has timeout
    if task.Timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, task.Timeout)
        defer cancel()
    }

    // Check context before execution
    select {
    case <-ctx.Done():
        return fmt.Errorf("task cancelled: %w", ctx.Err())
    default:
    }

    // Execute with context
    return e.executor.Execute(ctx, task)
}

func (e *ExecutionEngine) ExecuteParallel(ctx context.Context, tasks []types.Task) []error {
    results := make([]error, len(tasks))
    var wg sync.WaitGroup

    for i, task := range tasks {
        wg.Add(1)
        go func(idx int, t types.Task) {
            defer wg.Done()

            // Check context before starting
            select {
            case <-ctx.Done():
                results[idx] = ctx.Err()
                return
            default:
            }

            results[idx] = e.ExecuteTask(ctx, t)
        }(i, task)
    }

    wg.Wait()
    return results
}
```

---

## 6. Error Handling Enhancement

### 6.1 Structured Error Types

Create `/pkg/errors/errors.go`:

```go
package errors

import (
    "fmt"
)

type ErrorType string

const (
    ErrorTypeValidation  ErrorType = "validation"
    ErrorTypeExecution   ErrorType = "execution"
    ErrorTypeConnection  ErrorType = "connection"
    ErrorTypeTimeout     ErrorType = "timeout"
    ErrorTypeSecurity    ErrorType = "security"
)

type OnigiraError struct {
    Type    ErrorType
    Message string
    Cause   error
    Context map[string]interface{}
}

func (e *OnigiraError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

func (e *OnigiraError) Unwrap() error {
    return e.Cause
}

func NewValidationError(message string, cause error) *OnigiraError {
    return &OnigiraError{
        Type:    ErrorTypeValidation,
        Message: message,
        Cause:   cause,
        Context: make(map[string]interface{}),
    }
}

func NewExecutionError(message string, cause error) *OnigiraError {
    return &OnigiraError{
        Type:    ErrorTypeExecution,
        Message: message,
        Cause:   cause,
        Context: make(map[string]interface{}),
    }
}

func (e *OnigiraError) WithContext(key string, value interface{}) *OnigiraError {
    e.Context[key] = value
    return e
}
```

Usage:

```go
func (e *ExecutionEngine) ExecuteTask(ctx context.Context, task types.Task) error {
    err := e.executor.Execute(ctx, task)
    if err != nil {
        return errors.NewExecutionError("task execution failed", err).
            WithContext("task_name", task.Name).
            WithContext("module", task.Module)
    }
    return nil
}
```

---

## 7. Health Checks

### 7.1 Health Check Endpoint

Create `/internal/health/health.go`:

```go
package health

import (
    "context"
    "encoding/json"
    "net/http"
    "time"
)

type HealthStatus string

const (
    StatusHealthy   HealthStatus = "healthy"
    StatusDegraded  HealthStatus = "degraded"
    StatusUnhealthy HealthStatus = "unhealthy"
)

type HealthCheck struct {
    Name   string       `json:"name"`
    Status HealthStatus `json:"status"`
    Error  string       `json:"error,omitempty"`
}

type HealthResponse struct {
    Status    HealthStatus  `json:"status"`
    Timestamp time.Time     `json:"timestamp"`
    Checks    []HealthCheck `json:"checks"`
}

type Checker interface {
    Check(ctx context.Context) HealthCheck
}

type HealthService struct {
    checkers []Checker
}

func NewHealthService() *HealthService {
    return &HealthService{
        checkers: make([]Checker, 0),
    }
}

func (hs *HealthService) AddChecker(checker Checker) {
    hs.checkers = append(hs.checkers, checker)
}

func (hs *HealthService) Check(ctx context.Context) HealthResponse {
    checks := make([]HealthCheck, len(hs.checkers))
    overallStatus := StatusHealthy

    for i, checker := range hs.checkers {
        checks[i] = checker.Check(ctx)

        if checks[i].Status == StatusUnhealthy {
            overallStatus = StatusUnhealthy
        } else if checks[i].Status == StatusDegraded && overallStatus == StatusHealthy {
            overallStatus = StatusDegraded
        }
    }

    return HealthResponse{
        Status:    overallStatus,
        Timestamp: time.Now(),
        Checks:    checks,
    }
}

func (hs *HealthService) Handler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()

        response := hs.Check(ctx)

        w.Header().Set("Content-Type", "application/json")

        if response.Status == StatusUnhealthy {
            w.WriteHeader(http.StatusServiceUnavailable)
        } else if response.Status == StatusDegraded {
            w.WriteHeader(http.StatusOK)
        } else {
            w.WriteHeader(http.StatusOK)
        }

        json.NewEncoder(w).Encode(response)
    }
}

// Example checker implementations
type SSHPoolChecker struct {
    pool *ssh.ConnectionPool
}

func (c *SSHPoolChecker) Check(ctx context.Context) HealthCheck {
    metrics := c.pool.GetMetrics()

    if metrics.ConnectionErrors > 10 {
        return HealthCheck{
            Name:   "ssh_pool",
            Status: StatusDegraded,
            Error:  fmt.Sprintf("high error rate: %d errors", metrics.ConnectionErrors),
        }
    }

    return HealthCheck{
        Name:   "ssh_pool",
        Status: StatusHealthy,
    }
}
```

---

## 8. Running the Optimizations

### 8.1 Test Coverage Script

Create `/scripts/test-coverage.sh`:

```bash
#!/bin/bash

echo "Running tests with coverage..."
go test -coverprofile=coverage.out ./...

echo "Generating coverage report..."
go tool cover -html=coverage.out -o coverage.html

echo "Coverage by package:"
go tool cover -func=coverage.out | grep total

echo "Opening coverage report..."
open coverage.html
```

### 8.2 Benchmark Script

Create `/scripts/benchmark.sh`:

```bash
#!/bin/bash

echo "Running benchmarks..."
go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./...

echo "Analyzing CPU profile..."
go tool pprof -top cpu.prof

echo "Analyzing memory profile..."
go tool pprof -top mem.prof
```

### 8.3 Continuous Integration

Update `.github/workflows/ci.yml`:

```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.21

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Check coverage
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: $coverage%"
          if (( $(echo "$coverage < 60" | bc -l) )); then
            echo "Coverage is below 60%"
            exit 1
          fi

      - name: Run benchmarks
        run: go test -bench=. -benchmem ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v2
        with:
          files: ./coverage.out
```

---

**Last Updated:** 2025
**Version:** 1.0
**Status:** Ready for Implementation
