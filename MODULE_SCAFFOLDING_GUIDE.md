# Module Scaffolding

`scripts/module_scaffold` is a generator for module boilerplate. **Its output does not compile**: the templates are malformed and target an old API (`interfaces.Executor`, `types.TaskDefinition`, `types.ModuleResult`) that no longer exists. Start a new module from an existing one instead, following the [Module Development Guide](MODULE_DEVELOPMENT_GUIDE.md).

## Usage

```bash
go run ./scripts/module_scaffold -name my_module -output internal/modules \
  -desc "Description" -params "path,state"
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-name` | required | module name, lowercase with underscores |
| `-desc` | `""` | description |
| `-params` | `""` | comma-separated parameter names |
| `-output` | an absolute path on the author's machine | output directory; always pass `-output internal/modules` |
| `-idempotent` | `true` | also write `<name>_idempotency_test.go` |

It writes `<name>.go`, `<name>_test.go` and, unless `-idempotent=false`, `<name>_idempotency_test.go`. The module is not registered; see [Registration](MODULE_DEVELOPMENT_GUIDE.md#registration).
