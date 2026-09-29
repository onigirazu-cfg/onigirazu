# Executors in Modules

A module runs commands on a host through an `executor.CommandExecutor`
(`internal/executor`). A module instance is shared by all hosts, so it must never keep an
executor in a struct field: every host would then run commands over the first host's
connection. Create an executor per call instead, from the `host` passed to `Execute`.

## BaseExecutorModule

`internal/modules/base.go` provides `BaseExecutorModule` (embeds `BaseModule`) with three
helpers that each create a fresh executor for the given host:

| Method | Use |
|--------|-----|
| `WithExecutor(host, func(*executor.CommandExecutor) error) error` | several commands; closes the executor afterwards |
| `WithExecutorResult(host, func(*executor.CommandExecutor) (string, error)) (string, error)` | one command returning output |
| `CreateExecutor(host) (*executor.CommandExecutor, error)` | manual control; the caller must `Close()` it |

```go
type MyModule struct {
    *BaseExecutorModule
}

func NewMyModule() *MyModule {
    return &MyModule{BaseExecutorModule: NewBaseExecutorModule("mymodule")}
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    out, err := m.WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error) {
        return exec.Execute("hostname")
    })
    // build the TaskResult from out / err
}
```

Pass the executor to helper methods as a parameter. A working example:
[examples/example_module_with_base_executor.go](examples/example_module_with_base_executor.go);
the full module guide: [MODULE_DEVELOPMENT_GUIDE.md](MODULE_DEVELOPMENT_GUIDE.md).

## CommandExecutor

- `Execute(command, args...)` / `ExecuteContext(ctx, …)` / `ExecuteWithTimeout(…)`: stdout
  and stderr combined; a non-zero exit is an error.
- `Run(ctx, commandLine)`: returns `RunResult{Stdout, Stderr, RC}`; a non-zero exit is not
  an error.
- Separate `args` are shell-quoted; a single command string may use shell syntax.
- The host's `become` settings and task environment are applied to every command
  (`sudo -S` with the become password on stdin, `sudo -n` without one).
- Local hosts (`ansible_connection=local`, `localhost`) run commands with `sh -c` on the
  control machine.

## Connections

`NewCommandExecutor` takes the host's connection from the global SSH pool
(`internal/ssh/pool.go`), keyed by `user@address:port`: one SSH connection per host,
reused by all executors. `Close()` releases it back to the pool; it does not close it.

Commands run through a shell kept open on that connection (`internal/ssh/shell.go`):
one round trip per command instead of a new SSH session each time. Overlapping commands
on the same host get another shell; if the host cannot run the shell, each command gets
its own SSH session. A command that never reached a host whose connection died is sent
again on a new connection.

The shell keeps each command's output in `~/.onigirazu/tmp/sh.XXXXXX` of the connecting
user and removes that directory when the connection closes. A task that cleans `/tmp`
does not touch it; if something removes it, it is created again for the next command.

## Testing

`internal/modules/executor_safety_test.go` checks that modules do not cache executors.
For unit tests of module logic, `internal/modules/mock_executor.go` implements the
`ModuleExecutor` interface (`executor_interface.go`).
