# Module Scaffolding

`scripts/module_scaffold` writes a module skeleton that compiles: `internal/modules/<name>.go`
(the structure of the [Module Development Guide](MODULE_DEVELOPMENT_GUIDE.md): validation,
reading arguments, a command on the host, check mode) and `<name>_test.go`.

```bash
go run ./scripts/module_scaffold -name my_module -desc "Manage X" -params path,state
```

Flags: `-name` (required), `-desc`, `-params` (the first is required by `Validate`),
`-output` (default `internal/modules`), `-force`.

After generating:

1. `registry.RegisterModule(NewMyModuleModule())` in `NewRegistry` (`internal/modules/registry.go`);
   add the name to `checkModeModules` once check mode works.
2. `go generate ./internal/cli`, so `onigirazu lint` knows the arguments.
3. Fill in the state comparison and the change, write tests, document the module in
   `docs/modules/README.md` and `INDEX.md`, add an e2e case.

A test (`go test ./scripts/module_scaffold`) compiles the generated code inside the modules
package.
