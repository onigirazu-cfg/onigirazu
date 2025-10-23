# Architecture Diagrams - v1.52.0

**Latest Release:** v1.52.0 - Documentation Complete
**Last Updated:** January 29, 2025

## Problem: Executor Caching Bug (Fixed in Early Versions)

### Before (Broken Architecture)

```
┌─────────────────────────────────────────────────────────────┐
│                        MyModule                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │  executor *executor.CommandExecutor                │    │
│  │      ↓                                              │    │
│  │  [SSH Connection to host1]  ← CACHED!              │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                          ↓
        Execute on host1, host2, host3...
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  host1.Execute("hostname") → "host1"  ✅ Correct            │
│  host2.Execute("hostname") → "host1"  ❌ WRONG!             │
│  host3.Execute("hostname") → "host1"  ❌ WRONG!             │
└─────────────────────────────────────────────────────────────┘

Problem: All hosts use the same cached executor → all execute on host1
```

### After (Fixed Architecture)

```
┌─────────────────────────────────────────────────────────────┐
│                   MyModule                                   │
│  ┌────────────────────────────────────────────────────┐    │
│  │  *BaseExecutorModule                               │    │
│  │      ↓                                              │    │
│  │  WithExecutor(host, fn)                            │    │
│  │      ↓                                              │    │
│  │  Creates fresh executor per call                   │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                          ↓
        Execute on host1, host2, host3...
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  host1.Execute("hostname") → "host1"  ✅ Correct            │
│  host2.Execute("hostname") → "host2"  ✅ Correct            │
│  host3.Execute("hostname") → "host3"  ✅ Correct            │
└─────────────────────────────────────────────────────────────┘

Solution: Fresh executor per execution → each host gets correct connection
```

---

## BaseExecutorModule Architecture

### Component Structure

```
┌──────────────────────────────────────────────────────────────┐
│                      BaseModule                               │
│  - name: string                                               │
│  - description: string                                        │
│  + GetName() string                                           │
│  + GetDescription() string                                    │
│  + Validate(args) error                                       │
└──────────────────────────────────────────────────────────────┘
                          ▲
                          │ embeds
                          │
┌──────────────────────────────────────────────────────────────┐
│                  BaseExecutorModule                           │
│  *BaseModule                                                  │
│                                                               │
│  + WithExecutor(host, fn) error                              │
│  + WithExecutorResult(host, fn) (string, error)              │
│  + CreateExecutor(host) (*CommandExecutor, error)            │
└──────────────────────────────────────────────────────────────┘
                          ▲
                          │ embeds
                          │
┌──────────────────────────────────────────────────────────────┐
│                      MyModule                                 │
│  *BaseExecutorModule                                          │
│                                                               │
│  + Execute(ctx, host, args) (TaskResult, error)              │
│  - helperMethod(exec, ...) error                             │
└──────────────────────────────────────────────────────────────┘
```

### Execution Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. Module.Execute(ctx, host, args)                              │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│ 2. m.WithExecutor(host, func(exec) { ... })                     │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│ 3. exec, err := executor.NewCommandExecutor(host)               │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│ 4. pool.GetConnection(host)                                     │
│    ├─ Connection exists? → Reuse (fast)                         │
│    └─ New connection? → Create (first time only)                │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│ 5. fn(exec) - Execute user function                             │
└────────────────────┬────────────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────────────┐
│ 6. defer exec.Close() - Return connection to pool               │
└─────────────────────────────────────────────────────────────────┘
```

---

## Three Usage Patterns

### Pattern 1: WithExecutorResult (Simple)

```
┌──────────────────────────────────────────────────────┐
│  Module.Execute(ctx, host, args)                     │
│      ↓                                                │
│  output, err := m.WithExecutorResult(host,           │
│      func(exec) (string, error) {                    │
│          return exec.Execute("hostname")             │
│      })                                               │
│      ↓                                                │
│  [Fresh executor created]                            │
│      ↓                                                │
│  [Command executed]                                  │
│      ↓                                                │
│  [Executor closed automatically]                     │
│      ↓                                                │
│  return result                                        │
└──────────────────────────────────────────────────────┘

Use case: Single command execution
Pros: Simplest, most concise
```

### Pattern 2: WithExecutor (Complex)

```
┌──────────────────────────────────────────────────────┐
│  Module.Execute(ctx, host, args)                     │
│      ↓                                                │
│  err := m.WithExecutor(host,                         │
│      func(exec) error {                              │
│          exists, err := checkExists(exec, path)      │
│          if err != nil { return err }                │
│                                                       │
│          if !exists {                                 │
│              return createFile(exec, path)           │
│          }                                            │
│          return nil                                   │
│      })                                               │
│      ↓                                                │
│  [Fresh executor created]                            │
│      ↓                                                │
│  [Multiple operations]                               │
│      ↓                                                │
│  [Executor closed automatically]                     │
│      ↓                                                │
│  return result                                        │
└──────────────────────────────────────────────────────┘

Use case: Multiple operations, complex logic
Pros: Flexible, automatic cleanup
```

### Pattern 3: CreateExecutor (Manual)

```
┌──────────────────────────────────────────────────────┐
│  Module.Execute(ctx, host, args)                     │
│      ↓                                                │
│  exec, err := m.CreateExecutor(host)                 │
│  if err != nil { return err }                        │
│  defer exec.Close()  ← MUST CALL                     │
│      ↓                                                │
│  [Fresh executor created]                            │
│      ↓                                                │
│  output1, err := exec.Execute("cmd1")                │
│  // ... complex logic ...                            │
│  output2, err := exec.Execute("cmd2")                │
│      ↓                                                │
│  [Executor closed by defer]                          │
│      ↓                                                │
│  return result                                        │
└──────────────────────────────────────────────────────┘

Use case: Maximum control, complex flow
Pros: Full control
Cons: Must remember defer exec.Close()
```

---

## Connection Pooling

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                    SSH Connection Pool                       │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│  │ host1      │  │ host2      │  │ host3      │           │
│  │ [SSH Conn] │  │ [SSH Conn] │  │ [SSH Conn] │           │
│  │ In Use: 0  │  │ In Use: 1  │  │ In Use: 0  │           │
│  └────────────┘  └────────────┘  └────────────┘           │
└─────────────────────────────────────────────────────────────┘
         ▲                ▲                ▲
         │                │                │
    GetConnection    GetConnection    GetConnection
         │                │                │
┌────────┴────────┬───────┴────────┬───────┴────────┐
│  Executor 1     │  Executor 2    │  Executor 3    │
│  (host1)        │  (host2)       │  (host3)       │
└─────────────────┴────────────────┴────────────────┘
         ▲                ▲                ▲
         │                │                │
┌────────┴────────┬───────┴────────┬───────┴────────┐
│  Module.Execute │  Module.Execute│  Module.Execute│
│  (host1)        │  (host2)       │  (host3)       │
└─────────────────┴────────────────┴────────────────┘
```

### Performance Impact

```
Traditional (without pool):
┌──────────────────────────────────────────────────────┐
│ Execute on host1                                      │
│   ├─ Create SSH connection (500ms)                   │
│   ├─ Execute command (50ms)                          │
│   └─ Close connection (10ms)                         │
│ Total: 560ms                                          │
└──────────────────────────────────────────────────────┘

With Connection Pool:
┌──────────────────────────────────────────────────────┐
│ Execute on host1 (first time)                        │
│   ├─ Create SSH connection (500ms)                   │
│   ├─ Execute command (50ms)                          │
│   └─ Return to pool (1ms)                            │
│ Total: 551ms                                          │
│                                                       │
│ Execute on host1 (subsequent)                        │
│   ├─ Get from pool (1ms)                             │
│   ├─ Execute command (50ms)                          │
│   └─ Return to pool (1ms)                            │
│ Total: 52ms  ← 10x faster!                           │
└──────────────────────────────────────────────────────┘
```

---

## Verification Flow

### Lint Checker

```
┌─────────────────────────────────────────────────────┐
│  ./scripts/check_executor_caching.sh                │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│  Scan all *.go files in internal/modules/           │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│  Search for pattern:                                 │
│  "executor *executor.CommandExecutor"                │
└────────────────┬────────────────────────────────────┘
                 ↓
         ┌───────┴────────┐
         │                │
    Found?           Not Found?
         │                │
         ↓                ↓
┌────────────────┐  ┌──────────────┐
│ Report issues  │  │ ✅ All good  │
│ with line #s   │  │              │
│ Exit code: 1   │  │ Exit code: 0 │
└────────────────┘  └──────────────┘
```

### CI/CD Integration

```
┌─────────────────────────────────────────────────────┐
│  Developer pushes code                               │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│  GitHub Actions / CI Pipeline                        │
│  ├─ Run tests                                        │
│  ├─ Run linters                                      │
│  ├─ Run check_executor_caching.sh  ← NEW            │
│  └─ Build                                            │
└────────────────┬────────────────────────────────────┘
                 ↓
         ┌───────┴────────┐
         │                │
    All pass?        Failed?
         │                │
         ↓                ↓
┌────────────────┐  ┌──────────────────────┐
│ ✅ Merge OK    │  │ ❌ Block merge       │
│                │  │ Show error message   │
└────────────────┘  └──────────────────────┘
```

---

## Migration Path

### Step-by-Step

```
┌─────────────────────────────────────────────────────┐
│ 1. Identify module with cached executor             │
│    ./scripts/check_executor_caching.sh              │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│ 2. Change base class                                 │
│    *BaseModule → *BaseExecutorModule                │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│ 3. Remove executor field                             │
│    Delete: executor *executor.CommandExecutor       │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│ 4. Update Execute() method                           │
│    Use WithExecutor or WithExecutorResult            │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│ 5. Update helper methods                             │
│    Add exec parameter to all helpers                │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│ 6. Test                                              │
│    go build && go test                               │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│ 7. Verify                                            │
│    ./scripts/check_executor_caching.sh              │
└─────────────────────────────────────────────────────┘
```

---

## Summary

```
┌──────────────────────────────────────────────────────────────┐
│                    Architecture Benefits                      │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ✅ Correctness                                              │
│     Each host executes on correct connection                 │
│                                                               │
│  ✅ Performance                                              │
│     Connection pooling maintains speed                       │
│                                                               │
│  ✅ Safety                                                   │
│     Impossible to cache executor by design                   │
│                                                               │
│  ✅ Maintainability                                          │
│     Cleaner code, less boilerplate                           │
│                                                               │
│  ✅ Developer Experience                                     │
│     Clear patterns, good documentation                       │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```
