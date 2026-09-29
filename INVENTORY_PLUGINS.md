# Inventory Plugins

Inventory plugins do not supply hosts today. For hosts from a cloud API or a CMDB use a
[dynamic inventory script](INVENTORY_FORMATS.md#dynamic-inventory): any executable that
prints Ansible inventory JSON for `--list`, including existing Ansible inventory scripts.

## What exists

- The Go interface `InventoryPlugin` in `internal/plugins/interface.go` (`GetHosts`,
  `GetGroups`, `Refresh`, `GetCacheTTL`) and `BaseInventoryPlugin` in
  `internal/plugins/inventory.go`.
- `apply --plugins-config FILE` (or a `plugins.yml` next to the playbook) loads Go plugins
  (`.so`, built with `go build -buildmode=plugin`, exporting `NewPlugin`). Inventory
  plugins are loaded and registered, but nothing asks them for hosts or groups; the
  inventory comes only from `-i`. `run`, `plan` and `inventory` do not load plugins.
- `examples/plugins/aws_ec2`, `azure_vm` and `gcp_compute` return fixed mock hosts; they do
  not call any cloud API.

Plugin configuration file format (as `examples/plugins/inventory_plugins.yml`):

```yaml
plugins_dir: ./plugins        # relative paths below are resolved against it
plugins:
  - name: aws_ec2
    type: inventory           # module, callback, inventory or filter
    path: inventory_aws_ec2.so
    enabled: true
    config:                   # passed to the plugin's Initialize
      region: us-east-1
```

A plugin that fails to load is logged as a warning and `apply` continues without plugins.
