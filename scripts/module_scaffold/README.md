# module_scaffold

Writes the skeleton of a new built-in module and its test:

```bash
go run ./scripts/module_scaffold -name my_module -desc "Manage X" -params path,state
```

| Flag | Default | |
|------|---------|--|
| `-name` | - | module name (lowercase, digits, `_`) |
| `-desc` | `Manage <name>` | one-line description |
| `-params` | `path` | arguments read by the module; the first is required |
| `-output` | `internal/modules` | directory of the modules package |
| `-force` | false | overwrite existing files |

Then register the module in `NewRegistry` (`internal/modules/registry.go`), run
`go generate ./internal/cli` and document it. See the
[Module Development Guide](../../docs/MODULE_DEVELOPMENT_GUIDE.md).
