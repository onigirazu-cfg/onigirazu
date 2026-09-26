# Plugins

Plugins add template filters, modules and execution callbacks. They are Go plugins
(`.so` files) loaded by `onigirazu apply`, and therefore also by `plan` and `drift`.

## Loading plugins

`apply` reads a plugin configuration from `--plugins-config FILE`, or else from
`plugins.yml` in the playbook's directory:

```yaml
# plugins.yml
plugins_dir: ./plugins          # default ./plugins, relative to the current directory
plugins:
  - name: hello
    type: module                # module, callback or filter
    enabled: true
    path: module_hello.so       # relative paths are resolved against plugins_dir
    config:                     # passed to the plugin's Initialize()
      default_greeting: Hello
  - name: metrics
    type: callback
    enabled: true
    path: callback_metrics.so
    config:
      output_file: /tmp/metrics.json
```

- Entries without `path` are skipped. The built-in filters (`upper`, `lower`, `title`,
  `trim`, `replace`, `default`, `length`, `join`, `split`, and others) are always available
  and need no entry.
- If any enabled plugin fails to load, `apply` logs a warning and continues with no
  plugins at all.
- Type `inventory` is accepted but inventory plugins are not used.

## What each type does

| Type | Effect |
|------|--------|
| `filter` | Its filters become template filters: `{{ 'hello' \| reverse }}` |
| `module` | Usable in tasks like a built-in module. A plugin cannot replace a built-in module: on a name clash it is not registered and a warning is logged. |
| `callback` | Receives playbook, play and task events, see [CALLBACKS_GUIDE.md](CALLBACKS_GUIDE.md) |

## Writing a plugin

The plugin interfaces are in `internal/plugins` (`Plugin`, `ModulePlugin`,
`CallbackPlugin`, `FilterPlugin`) with base types `BaseModulePlugin`,
`BaseCallbackPlugin` and `BaseFilterPlugin`. A plugin is a `package main` that exports:

```go
func NewPlugin() plugins.Plugin
```

Example filter plugin:

```go
//go:build plugin

package main

import (
    "fmt"

    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
)

func reverse(input interface{}, args ...interface{}) (interface{}, error) {
    s, ok := input.(string)
    if !ok {
        return nil, fmt.Errorf("reverse expects a string, got %T", input)
    }
    r := []rune(s)
    for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
        r[i], r[j] = r[j], r[i]
    }
    return string(r), nil
}

func NewPlugin() plugins.Plugin {
    p := plugins.NewBaseFilterPlugin("reverse_filter", "1.0.0", "Reverse a string")
    p.AddFilter("reverse", reverse)
    return p
}
```

Complete examples: `examples/plugins/module_hello.go`, `examples/plugins/callback_metrics.go`,
configuration `examples/plugins/plugins.yml`, playbook `examples/10-plugins-demo.yml`.

## Building

Because the interfaces live in an `internal` package, a plugin can only be built inside
this repository (for example under `examples/plugins/`), and Go requires the plugin and
the `onigirazu` binary to be built from the same source with the same Go version, with
cgo enabled. Use Linux; on recent macOS versions `dlopen` rejects Go plugins.

```bash
CGO_ENABLED=1 go build -o onigirazu ./cmd/onigirazu
CGO_ENABLED=1 go build -buildmode=plugin -tags plugin -o plugins/module_hello.so ./examples/plugins/module_hello.go
```

Current limitation: the released binaries and Docker images are built with
`CGO_ENABLED=0` and cannot load `.so` plugins; use a binary built as above.

## Troubleshooting

- `Failed to load plugins: …` in the log: check `path`, and that the binary and the
  plugin were built together as described above. `plugin: not implemented` means the
  binary was built without cgo.
- `Module plugin X not registered`: the name is taken by a built-in module; rename it.
