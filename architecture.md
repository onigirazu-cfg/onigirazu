# Architecture

Onigirazu is a single Go binary (`cmd/onigirazu`) that runs Ansible-style playbooks on
hosts over SSH, or locally. Nothing is installed on the hosts; commands run through
`sh` on each of them.

## Run of `onigirazu apply`

1. **CLI** (`internal/cli/apply.go`): loads the configuration (`internal/config`,
   [CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md)), the security policy
   (`internal/security`), secret providers (`internal/secrets`) and plugins
   (`internal/plugins`, [PLUGIN_INTEGRATION.md](PLUGIN_INTEGRATION.md)).
2. **Inventory**: `inventory.MultiSourceLoader` merges every `-i` source (files,
   directories, scripts, host lists); `internal/parser/inventory_parser.go` reads each
   format ([INVENTORY_FORMATS.md](INVENTORY_FORMATS.md)); `inventory.Manager` resolves host
   patterns, groups and group variables.
3. **Playbook**: `parser.EnhancedParser` reads the playbook, imports, includes, roles
   and `vars_files`; `internal/validator` checks module arguments before anything runs.
4. **Execution** (`engine.ExecutionEngine.ExecutePlaybook`): plays run in order. Within
   a play each task runs on all its hosts in parallel (bounded by `max_concurrency`, `serial`
   and `throttle`), then the next task starts. Per task and host the engine evaluates
   `when` (`internal/expression`), renders arguments (`internal/template`), looks the
   module up in `modules.Registry` and calls `Module.Execute`. Handlers, blocks, loops,
   `run_once`, `delegate_to`, facts (`internal/facts`), tags (`internal/tagfilter`) and
   callback plugins are handled here too.
5. **Modules** (`internal/modules`) run commands through `executor.CommandExecutor`,
   which uses one pooled SSH connection per host (`internal/ssh`) with a long-lived
   remote shell; work files are in `~/.onigirazu/tmp` on the host. See
   [EXECUTOR_SAFETY_README.md](EXECUTOR_SAFETY_README.md).
6. **Results**: the run summary is printed (`internal/output`, `internal/execution`
   for the TUI); the result is cached in `~/.onigirazu/cache/executions`
   (`show-execution`, `list-executions`), recorded in `~/.onigirazu/audit`
   (`internal/audit`), in the state file (`internal/state`) and in the managed state
   (`internal/managed`); `internal/rollback` takes snapshots for `rollback`.

`plan` and `drift` run the same apply code in check/diff mode and report what would
change ([DRIFT_AND_ROLLBACK.md](DRIFT_AND_ROLLBACK.md)).

## Packages

### internal

| Package | Responsibility |
|---------|----------------|
| `cli` | Cobra commands (`apply`, `run`, `plan`, `drift`, `rollback`, `state`, `import`, `galaxy`, `inventory`, `lint`, `validate`, …) |
| `config` | `onigirazu.yml` and `ONIGIRAZU_*` environment variables |
| `parser` | Playbooks, roles, includes, inventory files and inline host lists |
| `inventory` | Merging inventory sources, `group_vars`/`host_vars`, host patterns |
| `engine` | Playbook execution: plays, tasks, handlers, loops, serial, throttle, rollout, callbacks |
| `execution` | Worker pool, signal handling, TUI, execution result cache |
| `expression` | Jinja-style expressions, tests and filters (`when`, `{{ }}`) |
| `template` | Template rendering (Jinja syntax on top of Go templates), template cache ([TEMPLATE_CACHING.md](TEMPLATE_CACHING.md)) |
| `modules` | Built-in modules and the module registry |
| `executor` | Running a command on a host (local or SSH), `become`, environment |
| `ssh` | SSH client, connection pool, persistent remote shell, host key checking |
| `facts` | Fact gathering (`setup`, `gather_facts`) |
| `managed` | Managed state: resources a playbook owns, orphan cleanup ([MANAGED_STATE.md](MANAGED_STATE.md)) |
| `rollback` | Snapshots and restore |
| `drift` | Drift detection and reports |
| `state` | State file backends (file, SQLite) |
| `audit` | Execution audit records and reports |
| `importer` | `import`: a playbook from running hosts ([IMPORT.md](IMPORT.md)) |
| `galaxy` | `galaxy install` of roles and collections |
| `secrets` | Vault and Bitwarden lookups |
| `security` | Security policy ([SECURITY_POLICY_GUIDE.md](SECURITY_POLICY_GUIDE.md)) |
| `validator` | Module argument validation |
| `plugins` | Plugin interfaces, loader, manager, built-in filters |
| `adhoc` | `run` (ad-hoc commands) |
| `healthcheck` | `healthcheck` command |
| `metrics` | Metrics collection and the optional HTTP endpoint ([METRICS_SECURITY_GUIDE.md](METRICS_SECURITY_GUIDE.md)) |
| `cache` | Generic TTL cache, facts, package and template caches |
| `output`, `progress`, `diffview` | Console output, progress, diffs |
| `tagfilter`, `tagdiscovery`, `taskpreview` | `--tags`/`--skip-tags`, `--list-tags`, `--list-tasks` |
| `interfaces` | Shared interfaces (logger, parser, cache, module registry) |
| `logger`, `bufferpool`, `version` | Logging, buffer reuse, version information |

### pkg and cmd

| Path | Content |
|------|---------|
| `pkg/types` | Core types: `Host`, `Inventory`, `Playbook`, `Play`, `Task`, `TaskResult`, `Module` |
| `pkg/errors` | Error types |
| `pkg/formatter` | YAML formatting (`fmt`) |
| `pkg/profiler` | `--profile` support |
| `pkg/utils` | Terminal colours |
| `cmd/onigirazu` | Main binary |
| `cmd/yaml-format` | Standalone YAML formatter |

## Module interface

```go
// pkg/types
type Module interface {
    Execute(ctx context.Context, host Host, args map[string]interface{}) (TaskResult, error)
    Validate(args map[string]interface{}) error
    GetName() string
    GetDescription() string
}
```

Writing a module: [MODULE_DEVELOPMENT_GUIDE.md](MODULE_DEVELOPMENT_GUIDE.md). Plugin
interfaces: [PLUGIN_INTEGRATION.md](PLUGIN_INTEGRATION.md) and
[CALLBACKS_GUIDE.md](CALLBACKS_GUIDE.md).
