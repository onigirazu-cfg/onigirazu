# Executor Safety Architecture

## 🎯 Problem Solved

This architecture prevents the **executor caching bug** - a critical issue where all hosts execute commands on the first host's SSH connection, causing silent data corruption.

## 🏗️ Architecture Overview

### BaseExecutorModule

A new base class that provides safe executor management patterns:

```go
type BaseExecutorModule struct {
    *BaseModule
}
```

**Key Features:**

- ✅ Prevents executor caching bugs by design
- ✅ Automatic executor lifecycle management
- ✅ Three flexible patterns for different use cases
- ✅ Connection pooling for performance
- ✅ Clean, idiomatic Go code

## 🚀 Quick Start

### Creating a New Module

```go
package modules

import (
    "context"
    "github.com/onigirazu-cfg/onigirazu/internal/executor"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type MyModule struct {
    *BaseExecutorModule  // ✅ Use this instead of BaseModule
}

func NewMyModule() *MyModule {
    return &MyModule{
        BaseExecutorModule: NewBaseExecutorModule("mymodule"),
    }
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Pattern 1: Simple command execution
    output, err := m.WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error) {
        return exec.Execute("hostname")
    })

    // Each host gets its own executor - no caching bugs!
    return result, err
}
```

## 📚 Three Usage Patterns

### Pattern 1: WithExecutorResult (Simplest)

Best for: Single command execution

```go
output, err := m.WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error) {
    return exec.Execute("hostname")
})
```

### Pattern 2: WithExecutor (Most Flexible)

Best for: Multiple operations, complex logic

```go
err := m.WithExecutor(host, func(exec *executor.CommandExecutor) error {
    exists, err := m.checkExists(exec, "/path")
    if err != nil {
        return err
    }

    if !exists {
        return m.createFile(exec, "/path")
    }

    return nil
})
```

### Pattern 3: CreateExecutor (Maximum Control)

Best for: When you need explicit control

```go
exec, err := m.CreateExecutor(host)
if err != nil {
    return result, err
}
defer exec.Close()  // Must call Close()

output, err := exec.Execute("hostname")
```

## 🔍 How It Works

### The Bug (Before)

```go
type BadModule struct {
    executor *executor.CommandExecutor  // ❌ Cached!
}

// All hosts use the same connection → all execute on host1
```

### The Fix (After)

```go
type GoodModule struct {
    *BaseExecutorModule  // ✅ No cached executor
}

// Each execution creates fresh executor → correct host
```

### Connection Pooling

Don't worry about performance! The architecture uses connection pooling:

```
┌─────────────┐
│   Module    │
└──────┬──────┘
       │ CreateExecutor(host2)
       ▼
┌─────────────┐
│  Executor   │ ← Fresh executor instance
└──────┬──────┘
       │ GetConnection(host2)
       ▼
┌─────────────┐
│ SSH Pool    │ ← Reuses existing connections
└─────────────┘
  │   │   │
  ▼   ▼   ▼
host1 host2 host3
```

**Result:** Fresh executor per execution + connection reuse = Safe + Fast

## 📖 Documentation

- **[Module Development Guide](./MODULE_DEVELOPMENT_GUIDE.md)** - Complete guide with examples
- **[Example Module](./examples/example_module_with_base_executor.go)** - Working code example

## ✅ Testing

Test your module with multiple hosts:

```go
func TestMyModule_MultipleHosts(t *testing.T) {
    module := NewMyModule()

    hosts := []types.Host{
        {Name: "host1", Address: "192.168.1.1"},
        {Name: "host2", Address: "192.168.1.2"},
    }

    for _, host := range hosts {
        result, err := module.Execute(context.Background(), host, args)
        // Verify each host returns its own result
    }
}
```

## 🎓 Migration Guide

### Step 1: Change Base Class

```diff
type MyModule struct {
-   *BaseModule
+   *BaseExecutorModule
}
```

### Step 2: Update Constructor

```diff
func NewMyModule() *MyModule {
    return &MyModule{
-       BaseModule: NewBaseModule("mymodule"),
+       BaseExecutorModule: NewBaseExecutorModule("mymodule"),
    }
}
```

### Step 3: Remove Cached Executor

```diff
type MyModule struct {
    *BaseExecutorModule
-   executor *executor.CommandExecutor  // ❌ Remove this
}
```

### Step 4: Use Safe Pattern

```diff
func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
-   output, err := m.executor.Execute("hostname")
+   output, err := m.WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error) {
+       return exec.Execute("hostname")
+   })
}
```

## 🛡️ Safety Guarantees

When using `BaseExecutorModule`:

1. ✅ **No executor caching** - Each execution gets fresh executor
2. ✅ **Correct host targeting** - Commands execute on intended host
3. ✅ **Automatic cleanup** - Connections returned to pool
4. ✅ **Connection reuse** - Pool manages SSH connections efficiently
5. ✅ **Type safety** - Compile-time guarantees

## 📊 Performance

| Metric | Before (Cached) | After (BaseExecutorModule) |
|--------|----------------|---------------------------|
| Correctness | ❌ Wrong host | ✅ Correct host |
| Connection overhead | Low | Low (pooled) |
| Memory usage | Low | Low |
| Code complexity | High (error-prone) | Low (safe by design) |

## 🎯 Best Practices

1. **Always use `BaseExecutorModule`** for new modules
2. **Never cache executor** as struct field
3. **Prefer `WithExecutor`** patterns over manual management
4. **Test with multiple hosts** to verify correctness
5. **Pass executor to helpers** as parameter, not from struct

## 🐛 Debugging

If you see the same result from all hosts:

```bash
# All hosts return "host1" - BUG!
host1: hostname=host1
host2: hostname=host1  ← Should be host2!
host3: hostname=host1  ← Should be host3!
```

**Cause:** Module is caching executor

**Fix:** Use `BaseExecutorModule` patterns

## 📝 Summary

The `BaseExecutorModule` architecture solves the executor caching bug by:

1. **Preventing caching** - No executor fields in structs
2. **Enforcing patterns** - Safe APIs that create fresh executors
3. **Maintaining performance** - Connection pooling under the hood
4. **Improving code quality** - Cleaner, more maintainable code

**Result:** Correct multi-host execution with zero performance penalty.

---

For detailed examples and patterns, see [MODULE_DEVELOPMENT_GUIDE.md](./MODULE_DEVELOPMENT_GUIDE.md)
