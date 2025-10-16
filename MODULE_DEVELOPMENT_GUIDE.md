# Module Development Guide

## Executor Safety: Critical Best Practices

### ⚠️ The Problem: Executor Caching Bug

**NEVER cache `executor.CommandExecutor` as a struct field!**

This is a critical bug that causes all hosts to execute commands on the first host's connection.

#### ❌ WRONG - Don't Do This

```go
type MyModule struct {
    *BaseModule
    executor *executor.CommandExecutor  // ❌ BUG: This will be reused across all hosts!
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // This executor is cached and connected to the first host only
    output, err := m.executor.Execute("hostname")
    // All hosts will return the same hostname!
}
```

**Why is this wrong?**

- The executor maintains an SSH connection to a specific host
- When cached as a struct field, all subsequent executions reuse the same connection
- Commands intended for host2, host3, etc. all execute on host1
- This causes silent data corruption and incorrect results

---

## ✅ Correct Patterns

### Pattern 1: Use BaseExecutorModule (Recommended)

The safest and cleanest approach:

```go
type MyModule struct {
    *BaseExecutorModule  // ✅ Provides safe executor management
}

func NewMyModule() *MyModule {
    return &MyModule{
        BaseExecutorModule: NewBaseExecutorModule("mymodule"),
    }
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    var output string

    // ✅ WithExecutor creates a fresh executor for this host
    err := m.WithExecutor(host, func(exec *executor.CommandExecutor) error {
        var execErr error
        output, execErr = exec.Execute("hostname")
        return execErr
    })

    // Each host gets its own executor and returns its own hostname
    return result, err
}
```

**Benefits:**

- Automatic executor creation and cleanup
- No risk of caching bugs
- Clean, readable code
- Follows the principle of least surprise

### Pattern 2: WithExecutorResult for Simple Cases

When you need to return a value directly:

```go
func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // ✅ WithExecutorResult handles both result and error
    output, err := m.WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error) {
        return exec.Execute("hostname")
    })

    if err != nil {
        return m.failResult(result, err.Error())
    }

    // Process output...
}
```

### Pattern 3: Manual Executor Management (When Needed)

For complex cases where you need more control:

```go
func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // ✅ Create fresh executor for THIS execution
    exec, err := executor.NewCommandExecutor(host)
    if err != nil {
        return result, err
    }
    defer exec.Close()  // ✅ Always close when done

    // Use executor...
    output, err := exec.Execute("hostname")

    // Each host gets correct results
}
```

**Important:**

- Always use `defer exec.Close()` immediately after creation
- Never store the executor in a struct field
- Create a new executor for each `Execute()` call

---

## Helper Methods Pattern

When your module has helper methods that need an executor:

### ✅ Correct: Pass executor as parameter

```go
func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    exec, err := executor.NewCommandExecutor(host)
    if err != nil {
        return result, err
    }
    defer exec.Close()

    // ✅ Pass executor to helper methods
    exists, err := m.checkExists(exec, "/path/to/file")
    if err != nil {
        return result, err
    }

    if !exists {
        err = m.createFile(exec, "/path/to/file")
    }

    return result, err
}

// ✅ Helper methods accept executor as parameter
func (m *MyModule) checkExists(exec *executor.CommandExecutor, path string) (bool, error) {
    output, err := exec.Execute("test", "-f", path)
    // ...
}

func (m *MyModule) createFile(exec *executor.CommandExecutor, path string) error {
    _, err := exec.Execute("touch", path)
    return err
}
```

### ❌ Wrong: Using cached executor in helpers

```go
// ❌ DON'T DO THIS
func (m *MyModule) checkExists(path string) (bool, error) {
    output, err := m.executor.Execute("test", "-f", path)  // ❌ Uses cached executor
    // ...
}
```

---

## Testing Your Module

Always test multi-host execution:

```go
func TestMyModule_MultipleHosts(t *testing.T) {
    module := NewMyModule()

    hosts := []types.Host{
        {Name: "host1", Address: "192.168.1.1"},
        {Name: "host2", Address: "192.168.1.2"},
        {Name: "host3", Address: "192.168.1.3"},
    }

    results := make(map[string]string)

    for _, host := range hosts {
        result, err := module.Execute(context.Background(), host, args)
        if err != nil {
            t.Fatalf("Failed for host %s: %v", host.Name, err)
        }

        results[host.Name] = result.Output["hostname"].(string)
    }

    // Verify each host returned its own hostname
    if results["host1"] == results["host2"] {
        t.Error("host1 and host2 returned the same result - executor caching bug!")
    }
}
```

---

## Code Review Checklist

When reviewing module code, check for:

- [ ] No `executor *executor.CommandExecutor` fields in module structs
- [ ] Each `Execute()` call creates a fresh executor
- [ ] `defer exec.Close()` is called after executor creation
- [ ] Helper methods receive executor as parameter, not from struct field
- [ ] Module uses `BaseExecutorModule` or follows manual pattern correctly
- [ ] Tests verify multi-host execution works correctly

---

## Why Connection Pooling Doesn't Cause Performance Issues

You might think: "Creating a new executor for each execution is expensive!"

**This is not a problem because:**

1. **Connection Pooling**: The executor uses `sshpkg.GetGlobalPool()` internally
2. **Connection Reuse**: Creating a new executor doesn't create a new SSH connection
3. **Automatic Management**: Connections are reused from the pool automatically
4. **Proper Cleanup**: `defer exec.Close()` returns the connection to the pool

```go
// This is efficient - connections are pooled
exec, err := executor.NewCommandExecutor(host)  // Gets connection from pool
defer exec.Close()                               // Returns connection to pool
```

---

## Migration Guide

If you have an existing module with cached executor:

### Before

```go
type MyModule struct {
    *BaseModule
    executor *executor.CommandExecutor  // ❌ Cached
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    output, err := m.executor.Execute("hostname")
    // ...
}
```

### After (Option 1 - BaseExecutorModule)

```go
type MyModule struct {
    *BaseExecutorModule  // ✅ Safe
}

func NewMyModule() *MyModule {
    return &MyModule{
        BaseExecutorModule: NewBaseExecutorModule("mymodule"),
    }
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    var output string
    err := m.WithExecutor(host, func(exec *executor.CommandExecutor) error {
        var execErr error
        output, execErr = exec.Execute("hostname")
        return execErr
    })
    // ...
}
```

### After (Option 2 - Manual)

```go
type MyModule struct {
    *BaseModule
    // ✅ No executor field
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    exec, err := executor.NewCommandExecutor(host)  // ✅ Fresh executor
    if err != nil {
        return result, err
    }
    defer exec.Close()  // ✅ Cleanup

    output, err := exec.Execute("hostname")
    // ...
}
```

---

## Summary

**Golden Rules:**

1. ✅ Use `BaseExecutorModule` for new modules
2. ✅ Create fresh executor in each `Execute()` call
3. ✅ Always `defer exec.Close()`
4. ✅ Pass executor to helper methods as parameter
5. ❌ NEVER cache executor as struct field

Following these patterns ensures your module works correctly across multiple hosts and prevents silent data corruption bugs.
