# Inventory Formats

An inventory lists the hosts and groups a playbook or ad-hoc command runs on. Pass it
with `-i`; `-i` can be repeated and can point to a file, a directory or an executable
script.

| Format | How it is recognised |
|--------|----------------------|
| YAML (Onigirazu `groups:` or Ansible `all:`) | `.yml`, `.yaml` |
| JSON | `.json` |
| TOML | `.toml` |
| INI (Ansible style) | `.ini` |
| Plain host list | other files whose lines contain no `:`, `=` or `[` |
| Dynamic inventory | executable file |

A file with another or no extension is detected from its content: executable script,
plain list, then JSON, YAML, TOML and INI are tried in that order.

## YAML

```yaml
groups:
  webservers:
    hosts:
      web1:
        onigirazu_host: 192.168.1.1
        onigirazu_port: 22
        onigirazu_user: admin
        app_port: 8080          # any other key is a host variable
      web2:
        onigirazu_host: 192.168.1.2
    vars:
      http_port: 80
  production:
    children:
      - webservers
    vars:
      environment: production
```

A YAML file with a top-level `all:` key is read as an Ansible inventory, see
[ANSIBLE_INVENTORY_QUICK_START.md](ANSIBLE_INVENTORY_QUICK_START.md).

## JSON

```json
{
  "hosts": [
    {"name": "web1", "address": "192.168.1.1", "port": 22, "user": "admin",
     "key_file": "/home/admin/.ssh/id_rsa", "vars": {"app_port": 8080}},
    {"name": "db1", "address": "192.168.1.10", "user": "postgres"}
  ],
  "groups": {
    "webservers": {"name": "webservers", "hosts": {"web1": {}}, "vars": {"http_port": 80}, "children": []},
    "databases":  {"name": "databases",  "hosts": {"db1": {}},  "vars": {"db_port": 5432}, "children": []}
  }
}
```

## TOML

```toml
[hosts.web1]
address = "192.168.1.1"
user = "admin"
key_file = "/home/admin/.ssh/id_rsa"
vars = { app_port = 8080 }

[hosts.db1]
address = "192.168.1.10"
user = "postgres"

[groups.webservers]
hosts = ["web1"]
vars = { http_port = 80 }

[groups.databases]
hosts = ["db1"]

[groups.production]
children = ["webservers", "databases"]
vars = { environment = "production" }
```

## INI

```ini
# comments start with # or ;
[webservers]
web1 ansible_host=192.168.1.1 ansible_user=admin app_port=8080
web2 ansible_host=192.168.1.2 ansible_port=2222

[databases]
db1 ansible_host=192.168.1.10 ansible_user=postgres

[webservers:vars]
http_port=80

[production:children]
webservers
databases

[production:vars]
environment=production
```

## Plain host list

One address or host name per line; `#` starts a comment. Hosts get port 22 and user
`root` and belong to the group `all`.

```
192.168.1.1
192.168.1.2
web3.example.com
```

Current limitation: `user@host` and `host:port` lines are not parsed (the file fails with
"inventory must contain at least one group"). Use an INI file for per-host users and ports.

## Dynamic inventory

An executable file (`chmod +x`) that prints a JSON inventory (format as above) on stdout.

```bash
#!/bin/bash
cat <<'EOF'
{
  "hosts": [{"name": "web1", "address": "192.168.1.1", "user": "admin"}],
  "groups": {"webservers": {"name": "webservers", "hosts": {"web1": {}}, "vars": {}}}
}
EOF
```

`apply`, `plan`, `drift` and `run` execute the script without arguments (30 s timeout,
output cached for 10 minutes); `onigirazu inventory` runs it with `--list`. Print the
inventory in both cases.

## Host variables

| Variable | Meaning | Default |
|----------|---------|---------|
| `onigirazu_host` / `ansible_host` (`address` in JSON/TOML) | address to connect to | host name |
| `onigirazu_port` / `ansible_port` (`port`) | SSH port | 22 |
| `onigirazu_user` / `ansible_user` (`user`) | SSH user | `root` in plain lists and Ansible YAML, otherwise the local `$USER` |
| `onigirazu_ssh_private_key_file` / `ansible_ssh_private_key_file` (`key_file`) | private key | — |
| `onigirazu_password` / `ansible_password` (`password`) | SSH password | — |
| any other key | host variable for templates | — |

`apply -u USER` and `--private-key FILE` (`run -u`/`-k`) override user and key for every host.

## group_vars and host_vars

As in Ansible, variables are also read from `group_vars/<group>.yml` and
`host_vars/<host>.yml` (`.yaml`, `.json`, no extension, or a directory of such files
merged in name order). They are looked up next to each inventory file, inside an
inventory directory, and, for `apply`/`plan`/`drift`, next to the playbook (read last,
so they win). They override variables written in the inventory itself.

## Several sources and directories

- `-i a.yml -i b.ini`: sources are merged; for the same host or group the later source wins.
- `-i inventory/`: every `.yml`, `.yaml`, `.json`, `.ini` and `.toml` file (alphabetically)
  and then every executable in the directory tree is loaded. Other files, hidden
  directories and `group_vars/`/`host_vars/` are skipped.

## Without -i

`run` requires `-i`. `apply` (and `plan`/`drift`) look in the playbook's directory for
`inventory.yml`, `inventory.yaml`, `inventory.toml`, `inventory.json`, `inventory.ini`,
`hosts`, `hosts.yml`, `hosts.yaml`, `hosts.toml`, `hosts.json`, `hosts.ini`, `inventory`,
then in `/etc/onigirazu/` for `inventory.yml`, `hosts.yml`, `inventory`.

## Limitations

- Inline inventories (`-i "192.168.1.1,192.168.1.2"`, `-i host,`) do not work with
  `apply`, `plan`, `drift` and `run`: `-i` values are split on commas and each part is
  opened as a file. Put the hosts in a file.
- `onigirazu inventory` takes a single `-i`, and a relative path without `/` or extension
  is taken as a host name; write `./hosts`.

## Inspecting an inventory

```bash
onigirazu inventory --list -i inventory.yml       # hosts and groups
onigirazu inventory --graph -i inventory.yml      # group tree
onigirazu inventory --host web1 -i inventory.yml  # groups of one host
```
