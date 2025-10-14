# 🧪 Testing

Onigirazu includes comprehensive testing to ensure reliability, performance, and security. This guide covers all testing aspects.

## 📋 Testing Overview

### Testing Types

- **Unit Tests** - Individual component testing
- **Integration Tests** - Component interaction testing
- **Performance Tests** - Speed and resource usage
- **Security Tests** - Vulnerability scanning
- **End-to-End Tests** - Complete workflow testing

### Testing Tools

- **Go Testing** - Built-in testing framework
- **Testify** - Testing utilities
- **Ginkgo** - BDD testing framework
- **Gomega** - Matcher library
- **Docker** - Container testing

---

## 🧪 Unit Testing

### Basic Unit Tests

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
        {
            name: "remove package",
            args: map[string]interface{}{
                "name": "apache2",
                "state": "absent",
            },
            expected: types.TaskResult{
                Changed: true,
                Output: map[string]interface{}{
                    "package": "apache2",
                    "state": "removed",
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

### Advanced Unit Tests

```go
// Test with mocks
func TestPackageModule_ExecuteWithMock(t *testing.T) {
    // Create mock executor
    mockExecutor := &MockCommandExecutor{}
    mockExecutor.On("Execute", "apt", "install", "nginx").Return("nginx installed", nil)
    
    module := NewPackageModule()
    module.executor = mockExecutor
    
    host := types.Host{Name: "test-host", Address: "127.0.0.1"}
    args := map[string]interface{}{
        "name": "nginx",
        "state": "present",
    }
    
    result, err := module.Execute(context.Background(), host, args)
    
    require.NoError(t, err)
    assert.True(t, result.Changed)
    mockExecutor.AssertExpectations(t)
}
```

### Test Utilities

```go
// Test utilities
func createTestHost() types.Host {
    return types.Host{
        Name: "test-host",
        Address: "127.0.0.1",
        User: "testuser",
        Port: 22,
    }
}

func createTestArgs() map[string]interface{} {
    return map[string]interface{}{
        "name": "nginx",
        "state": "present",
    }
}

func assertTaskResult(t *testing.T, result types.TaskResult, expected types.TaskResult) {
    assert.Equal(t, expected.Changed, result.Changed)
    assert.Equal(t, expected.Output, result.Output)
    assert.Equal(t, expected.Error, result.Error)
}
```

---

## 🔗 Integration Testing

### Basic Integration Tests

```go
// tests/integration/package_integration_test.go
package integration

import (
    "context"
    "testing"
    "github.com/onigirazu-cfg/onigirazu/internal/modules"
    "github.com/onigirazu-cfg/onigirazu/internal/inventory"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
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

### Docker Integration Tests

```go
// tests/integration/docker_integration_test.go
package integration

import (
    "context"
    "testing"
    "github.com/docker/docker/client"
    "github.com/onigirazu-cfg/onigirazu/internal/modules"
)

func TestDockerIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping Docker integration test")
    }
    
    // Create Docker client
    dockerClient, err := client.NewClientWithOpts(client.FromEnv)
    require.NoError(t, err)
    defer dockerClient.Close()
    
    // Start test container
    containerID := startTestContainer(t, dockerClient)
    defer stopTestContainer(t, dockerClient, containerID)
    
    // Test module
    module := modules.NewPackageModule()
    host := types.Host{
        Name: "test-container",
        Address: "localhost",
        Port: 2222,
    }
    
    args := map[string]interface{}{
        "name": "nginx",
        "state": "present",
    }
    
    result, err := module.Execute(context.Background(), host, args)
    
    require.NoError(t, err)
    assert.True(t, result.Changed)
}
```

---

## ⚡ Performance Testing

### Benchmark Tests

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

### Performance Benchmarks

```go
// tests/performance/benchmark_test.go
package performance

import (
    "testing"
    "github.com/onigirazu-cfg/onigirazu/internal/engine"
    "github.com/onigirazu-cfg/onigirazu/internal/modules"
)

func BenchmarkEngine_ExecutePlaybook(b *testing.B) {
    engine := engine.NewExecutionEngine(moduleRegistry, inventoryMgr, logger)
    playbook := createTestPlaybook()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := engine.ExecutePlaybook(context.Background(), playbook)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkModule_Execute(b *testing.B) {
    module := modules.NewPackageModule()
    host := createTestHost()
    args := createTestArgs()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := module.Execute(context.Background(), host, args)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Memory Profiling

```go
// tests/performance/memory_test.go
package performance

import (
    "testing"
    "runtime"
    "github.com/onigirazu-cfg/onigirazu/internal/engine"
)

func TestMemoryUsage(t *testing.T) {
    var m1, m2 runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&m1)
    
    // Execute operations
    engine := engine.NewExecutionEngine(moduleRegistry, inventoryMgr, logger)
    playbook := createTestPlaybook()
    
    _, err := engine.ExecutePlaybook(context.Background(), playbook)
    require.NoError(t, err)
    
    runtime.GC()
    runtime.ReadMemStats(&m2)
    
    // Check memory usage
    memoryUsed := m2.Alloc - m1.Alloc
    assert.Less(t, memoryUsed, uint64(100*1024*1024)) // 100MB limit
}
```

---

## 🔒 Security Testing

### Security Scanning

```go
// tests/security/security_test.go
package security

import (
    "testing"
    "github.com/securecodewarrior/gosec/v2"
    "github.com/securecodewarrior/gosec/v2/rules"
)

func TestSecurityScan(t *testing.T) {
    config := gosec.NewConfig()
    config.GlobalOptions = gosec.GlobalOptions{
        Include: []string{"G401", "G402", "G403", "G404"},
    }
    
    analyzer := gosec.NewAnalyzer(config)
    err := analyzer.Process("./...")
    require.NoError(t, err)
    
    issues := analyzer.Report()
    assert.Empty(t, issues, "Security issues found: %v", issues)
}
```

### Input Validation Tests

```go
// tests/security/input_validation_test.go
package security

import (
    "testing"
    "github.com/onigirazu-cfg/onigirazu/internal/modules"
    "github.com/stretchr/testify/assert"
)

func TestInputValidation(t *testing.T) {
    module := modules.NewPackageModule()
    host := types.Host{Name: "test-host", Address: "127.0.0.1"}
    
    testCases := []struct {
        name string
        args map[string]interface{}
        shouldFail bool
    }{
        {
            name: "valid input",
            args: map[string]interface{}{
                "name": "nginx",
                "state": "present",
            },
            shouldFail: false,
        },
        {
            name: "invalid package name",
            args: map[string]interface{}{
                "name": "../../etc/passwd",
                "state": "present",
            },
            shouldFail: true,
        },
        {
            name: "command injection",
            args: map[string]interface{}{
                "name": "nginx; rm -rf /",
                "state": "present",
            },
            shouldFail: true,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            _, err := module.Execute(context.Background(), host, tc.args)
            
            if tc.shouldFail {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

---

## 🎯 End-to-End Testing

### Complete Workflow Tests

```go
// tests/e2e/workflow_test.go
package e2e

import (
    "context"
    "testing"
    "github.com/onigirazu-cfg/onigirazu/internal/engine"
    "github.com/onigirazu-cfg/onigirazu/internal/inventory"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestCompleteWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test")
    }
    
    // Set up test environment
    engine := engine.NewExecutionEngine(moduleRegistry, inventoryMgr, logger)
    inventoryMgr := inventory.NewManager(parser, logger, cache)
    
    // Load test inventory
    err := inventoryMgr.LoadInventory(context.Background(), "test-inventory.yml")
    require.NoError(t, err)
    
    // Create test playbook
    playbook := &types.Playbook{
        Name: "test-playbook",
        Plays: []types.Play{
            {
                Name: "Test play",
                Hosts: []string{"localhost"},
                Tasks: []types.Task{
                    {
                        Name: "Install nginx",
                        Module: "package",
                        Args: map[string]interface{}{
                            "name": "nginx",
                            "state": "present",
                        },
                    },
                    {
                        Name: "Start nginx",
                        Module: "service",
                        Args: map[string]interface{}{
                            "name": "nginx",
                            "state": "started",
                        },
                    },
                },
            },
        },
    }
    
    // Execute playbook
    results, err := engine.ExecutePlaybook(context.Background(), playbook)
    
    require.NoError(t, err)
    assert.Len(t, results, 1)
    assert.True(t, results[0].Changed)
}
```

### Real Environment Tests

```go
// tests/e2e/real_environment_test.go
package e2e

import (
    "context"
    "testing"
    "github.com/onigirazu-cfg/onigirazu/internal/engine"
)

func TestRealEnvironment(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping real environment test")
    }
    
    // Test with real hosts
    engine := engine.NewExecutionEngine(moduleRegistry, inventoryMgr, logger)
    
    // Load real inventory
    err := inventoryMgr.LoadInventory(context.Background(), "real-inventory.yml")
    require.NoError(t, err)
    
    // Create real playbook
    playbook := createRealPlaybook()
    
    // Execute playbook
    results, err := engine.ExecutePlaybook(context.Background(), playbook)
    
    require.NoError(t, err)
    assert.NotEmpty(t, results)
}
```

---

## 🔧 Test Configuration

### Test Environment

```yaml
# test-config.yml
testing:
  unit:
    enabled: true
    timeout: 30s
    parallel: 4
  
  integration:
    enabled: true
    timeout: 5m
    parallel: 2
    docker: true
  
  performance:
    enabled: true
    timeout: 10m
    benchmarks: true
  
  security:
    enabled: true
    timeout: 5m
    gosec: true
  
  e2e:
    enabled: true
    timeout: 30m
    real_hosts: false
```

### Test Data

```yaml
# test-data.yml
test_hosts:
  - name: test-host-1
    address: 127.0.0.1
    user: testuser
    port: 22
  
  - name: test-host-2
    address: 127.0.0.1
    user: testuser
    port: 2222

test_playbooks:
  - name: simple-playbook
    plays:
      - name: Test play
        hosts: [test-host-1]
        tasks:
          - name: Test task
            module: command
            args:
              command: "echo 'Hello World'"
```

---

## 🚀 Running Tests

### Local Testing

```bash
# Run all tests
go test ./...

# Run specific test types
go test -tags=unit ./...
go test -tags=integration ./...
go test -tags=performance ./...
go test -tags=security ./...
go test -tags=e2e ./...

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

### CI/CD Testing

```bash
# Run CI tests
make ci

# Run specific CI steps
make test
make lint
make security
make performance
make e2e
```

### Docker Testing

```bash
# Run tests in Docker
docker run --rm -v $(pwd):/app -w /app golang:1.24 go test ./...

# Run integration tests with Docker Compose
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

---

## 📊 Test Reporting

### Coverage Reports

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# View coverage
open coverage.html
```

### Performance Reports

```bash
# Generate performance report
go test -bench=. -benchmem ./... > performance.txt

# View performance
cat performance.txt
```

### Security Reports

```bash
# Generate security report
gosec -fmt json -out security.json ./

# View security
cat security.json
```

---

## 🎯 Best Practices

### Test Organization

```go
// Organize tests by functionality
func TestPackageModule_Install(t *testing.T) {
    // Test installation
}

func TestPackageModule_Remove(t *testing.T) {
    // Test removal
}

func TestPackageModule_Update(t *testing.T) {
    // Test update
}
```

### Test Data Management

```go
// Use test fixtures
func loadTestData(t *testing.T) map[string]interface{} {
    data := make(map[string]interface{})
    // Load test data
    return data
}

// Clean up test data
func cleanupTestData(t *testing.T) {
    // Clean up test data
}
```

### Test Isolation

```go
// Isolate tests
func TestIsolated(t *testing.T) {
    // Set up test environment
    setupTest(t)
    defer cleanupTest(t)
    
    // Run test
    // ...
}
```

---

## 🚨 Troubleshooting

### Common Test Issues

#### Test Failures
```bash
# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestSpecificFunction ./...

# Debug test
go test -v -run TestSpecificFunction ./...
```

#### Performance Issues
```bash
# Run performance tests
go test -bench=. ./...

# Profile performance
go test -cpuprofile=cpu.prof ./...
go test -memprofile=mem.prof ./...
```

#### Integration Test Issues
```bash
# Check Docker
docker ps

# Check test environment
go test -tags=integration -v ./...
```

---

## 📚 Related Documentation

- [Development Setup](Development-Setup) - Development environment
- [Contributing](Contributing) - Contribution guidelines
- [Architecture](Architecture) - System architecture
- [Troubleshooting](Troubleshooting) - Common issues

---

## 🎯 Summary

### Testing Features

- **🧪 Unit Tests** - Component testing
- **🔗 Integration Tests** - Component interaction
- **⚡ Performance Tests** - Speed and resource usage
- **🔒 Security Tests** - Vulnerability scanning
- **🎯 E2E Tests** - Complete workflow testing

### Testing Tools

- **Go Testing** - Built-in framework
- **Testify** - Testing utilities
- **Ginkgo** - BDD framework
- **Docker** - Container testing
- **Gosec** - Security scanning

---

**🧪 Comprehensive testing ensures Onigirazu reliability and performance!**

