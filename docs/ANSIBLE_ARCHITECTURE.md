# Inventory Parsing Internals

User-facing formats and variables: [INVENTORY_FORMATS.md](INVENTORY_FORMATS.md). This
page describes the code path.

## Loading

```
-i sources ──> inventory.MultiSourceLoader.LoadFromMultipleSources   (internal/inventory/loader.go)
                 ├─ host list with a comma, not a file ──> parser.ParseInventoryOrInline (inline)
                 ├─ directory  ──> every .yml/.yaml/.json/.ini/.toml, then executables
                 ├─ executable ──> run "SCRIPT --list" (30 s timeout, result cached), parse JSON
                 └─ file       ──> parser.InventoryParser.ParseInventoryFile
               group_vars/ and host_vars/ next to each source are applied,
               later sources override earlier ones
          ──> inventory.Manager (patterns, groups, connection variables)
```

## Format dispatch (`internal/parser/inventory_parser.go`)

`ParseInventoryFile` chooses by extension: `.yml`/`.yaml` → `parseYamlInventory`,
`.json` → `parseJsonInventory`, `.toml` → `parseTomlInventory`, `.ini` →
`parseIniInventory`. Any other file goes to `autoDetectAndParse`: executable script,
plain host list, then JSON, YAML, TOML, INI, and a plain list as the last resort.

`parseYamlInventory` uses `isAnsibleYaml`: a top-level `all:` key, or `ansible_*` keys in
top-level `hosts`, selects the Ansible format; otherwise the YAML is read as Onigirazu's own
format (`hosts:` list, `groups:` map). JSON output of an Ansible inventory script (with
`_meta.hostvars` or group objects) is converted to the same tree as Ansible YAML.

## Ansible YAML

`parseAnsibleTree` walks `all` recursively (`ansibleWalk.group`): hosts may appear in any
group, `children` may be a map of groups or a list of names, and a host listed in several
groups gets its settings merged and is parsed once. `all` is kept as a group only when it
has `vars`.

`parseAnsibleHost` maps host keys to `types.Host` fields:

| Key | Field |
|-----|-------|
| `ansible_host` | `Address` (default: host name) |
| `ansible_port` | `Port` |
| `ansible_user` | `User` |
| `ansible_ssh_private_key_file` | `KeyFile` |
| `ansible_password` | `Password` |
| `ansible_become_password`, `ansible_become_pass`, `ansible_sudo_pass`, `onigirazu_become_password` | `BecomePassword` |
| `ansible_ssh_host_key_checking: false` | `InsecureIgnoreHostKey` |

Any other key goes to `Vars` with a leading `ansible_` removed (`ansible_foo` becomes
`foo`). Group `vars` stay on the group; `inventory.Manager` applies connection variables
from groups to hosts that do not set their own.

The parser fails with `no valid hosts found in Ansible inventory` when no host is
defined. An inventory whose hosts are all directly under `all` and that has no `all: vars`
ends up with no group and is rejected by validation (`inventory must contain at least one
group`).

## Tests

`internal/parser/inventory_parser_test.go` (`TestInventoryParser_AnsibleYAML_*`),
`internal/inventory/loader_test.go`, `internal/inventory/sources_test.go`.
