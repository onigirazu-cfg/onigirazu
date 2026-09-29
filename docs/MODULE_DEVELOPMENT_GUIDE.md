# Module Development Guide

Modules live in `internal/modules`, one file per module plus its `_test.go`.
Short, current examples to copy from: `hostname.go`, `slurp.go`, `timezone.go`.

## Interface

Every module implements `types.Module` (`pkg/types/types.go`):

```go
type Module interface {
	Execute(ctx context.Context, host Host, args map[string]interface{}) (TaskResult, error)
	Validate(args map[string]interface{}) error
	GetName() string
	GetDescription() string
}
```

## Skeleton

```go
package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type ExampleModule struct {
	*BaseModule
}

func NewExampleModule() *ExampleModule {
	return &ExampleModule{BaseModule: NewBaseModule("example")}
}

func (m *ExampleModule) GetDescription() string { return "Short description" }

func (m *ExampleModule) Validate(args map[string]interface{}) error {
	return requireStringArg(args, "path")
}

func (m *ExampleModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name,
		Timestamp: start, Success: true, Output: map[string]interface{}{}}
	if err := m.Validate(args); err != nil {
		result.Success, result.Error = false, err.Error()
		return result, nil
	}
	path := getStringArg(args, "path", "")

	out, err := runOnHost(ctx, host, args, "test", "-e", path)
	// ... compare current and desired state, set result.Changed

	if result.Changed && inCheckMode(args) {
		result.Duration = time.Since(start)
		return result, nil
	}
	// ... apply the change
	_ = out
	if err != nil {
		result.Success, result.Error = false, fmt.Sprintf("example failed: %v", err)
	}
	result.Duration = time.Since(start)
	return result, nil
}
```

A result with `Success: false` and a nil error is treated as a failed task
(the registry sets `Failed` and a default `Error`).

## Helpers

Arguments (`base.go`):

- `getStringArg`, `getBoolArg` (accepts `yes`/`no`, `on`/`off`, `1`/`0`, `true`/`false` strings), `getIntArg` (accepts numeric strings), `getMapArg`
- `requireStringArg` - error unless the argument is a non-empty string
- `taskName(args)` - the task title for `result.TaskName`

Running on the target host (`host_exec.go`); each call creates its own executor and honors `become`:

- `runOnHost(ctx, host, args, argv...)` - run a command, every word quoted
- `runShellOnHost(ctx, host, args, script)` - run a `sh -c` script
- `readHostFile`, `writeHostFile`, `ensureOwnership`, `statRemoteFile`
- `shellQuote`, `shellJoin`

Check mode and diff:

- `inCheckMode(args)` - report what would change, change nothing
- `addDiff(args, &result, header, before, after)` - records a diff when `--diff` is on

## Reserved arguments

The registry adds these to `args` before `Execute`; modules must not treat them as user arguments:

| Key | Meaning |
|-----|---------|
| `_task_name` | task title |
| `_vars` | task variables (play vars, facts, `set_fact`, `register`) |
| `_become`, `_become_user`, `_become_method` | privilege escalation |
| `_check_mode` | check mode is on |
| `_diff` | `--diff` is on |
| `_before` | pre-change state of a file module's target (used for rollback) |

## Registration

1. Add `registry.RegisterModule(NewExampleModule())` to `NewRegistry()` in `internal/modules/registry.go`. The parser's module validator takes its list of known names from the registry.
2. If the module supports check mode, add its name to `checkModeModules` in `registry.go`; otherwise tasks using it are skipped in check mode.
3. Optional: argument aliases in `pkg/types/arg_aliases.go`; then `go generate ./internal/cli` updates the argument names `onigirazu lint` knows (a test fails while they are out of date).

## Executor safety

Never store an executor in a module struct. The registry holds one module instance for all hosts, so a cached executor runs every host's commands on the first host.

```go
type BadModule struct {
	*BaseModule
	exec *executor.CommandExecutor // wrong
}
```

Use `runOnHost`/`runShellOnHost`, or create an executor per `Execute` call and close it:

```go
exec, err := executor.NewCommandExecutor(host)
if err != nil {
	return result, err
}
defer exec.Close()
```

Modules embedding `*BaseExecutorModule` (`NewBaseExecutorModule(name)`) can use:

- `WithExecutor(host, func(exec *executor.CommandExecutor) error)`
- `WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error)) (string, error)`

Pass the executor to helper functions as a parameter. Creating executors is cheap: SSH connections come from a shared pool (`sshpkg.GetGlobalPool()`), and `Close` returns them to it.

## Tests

Unit tests run the module against the local machine:

```go
host := types.Host{Name: "local", Address: "localhost",
	Vars: map[string]interface{}{"onigirazu_connection": "local"}}
```

```bash
go test ./internal/modules -run Example -v
go test -race ./...
```

For behavior on real hosts add an e2e case (`e2e/cases/NN-name/` with `playbook.yml` and `verify.sh`); see [e2e/README.md](../e2e/README.md).

## Documentation

Add the module to [modules/README.md](modules/README.md) and [modules/INDEX.md](modules/INDEX.md).
