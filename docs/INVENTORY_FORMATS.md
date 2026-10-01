# Inventory Formats

An inventory lists the hosts and groups a playbook or ad-hoc command runs on. Pass it
with `-i`; `-i` can be repeated and can point to a file, a directory, an executable
script or a list of hosts.

| Format | How it is recognised |
|--------|----------------------|
| YAML (Onigirazu `groups:` or Ansible `all:`) | `.yml`, `.yaml` |
| JSON | `.json` |
| TOML | `.toml` |
| INI (Ansible style) | `.ini` |
| Plain host list | other files whose lines are `[user@]host[:port]` |
| Host list on the command line | `-i host1,user@host2:2222` or `-i host1,` (a comma, not a file) |
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

Hosts run in the order Ansible uses: as they first appear in the file, `all`'s own hosts, then each child group in turn, depth first. `serial` batches and `run_once` follow that order.

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

Hosts before the first section are in the group `ungrouped`, as in Ansible. A group may be
empty (`[pending]`, or `pending: {hosts: }` in YAML).

## Plain host list

One `[user@]host[:port]` per line; `#` starts a comment. Hosts get port 22 and the local user
unless given and belong to the group `all`. A host is named by its address; when one
address appears with several ports, as `address:port`.

```
192.168.1.1
deploy@192.168.1.2:2222
web3.example.com
```

## Host list on the command line

As in Ansible, a `-i` value with a comma that is not a file is a list of hosts in the
same `[user@]host[:port]` form: `-i web1,web2`, `-i deploy@10.0.0.5:2222,` (a single host
needs the trailing comma). `-i a.yml,b.yml` loads both files when both exist.

## Dynamic inventory

An executable file (`chmod +x`), run with `--list` (30 s timeout, output cached for 10
minutes). It prints either an Ansible inventory script's JSON:

```json
{
  "web": {"hosts": ["web1", "web2"], "vars": {"tier": "front"}, "children": []},
  "db": ["db1"],
  "_meta": {"hostvars": {"web1": {"ansible_host": "192.168.1.1", "ansible_user": "admin"}}}
}
```

or Onigirazu's own JSON inventory:

```bash
#!/bin/bash
cat <<'EOF'
{
  "hosts": [{"name": "web1", "address": "192.168.1.1", "user": "admin"}],
  "groups": {"webservers": {"name": "webservers", "hosts": {"web1": {}}, "vars": {}}}
}
EOF
```

Existing Ansible inventory scripts work unchanged. As in Ansible, a host that is only in
`_meta.hostvars` is not added.

## Host variables

| Variable | Meaning | Default |
|----------|---------|---------|
| `onigirazu_host` / `ansible_host` (`address` in JSON/TOML) | address to connect to | host name |
| `onigirazu_port` / `ansible_port` (`port`) | SSH port | 22 |
| `onigirazu_user` / `ansible_user` (`user`) | SSH user | the local `$USER`, as in Ansible |
| `onigirazu_ssh_private_key_file` / `ansible_ssh_private_key_file` (`key_file`) | private key | — |
| `onigirazu_password` / `ansible_password` (`password`) | SSH password | — |
| `onigirazu_become_password` / `ansible_become_password` (`ansible_become_pass`) | sudo password for `become` (passed on stdin, never on a command line) | — (`sudo -n`) |
| `ansible_ssh_common_args`, `ansible_ssh_extra_args` | `-o ConnectTimeout=N`; `-J`/`-o ProxyJump=[user@]host[:port][,...]` (aliases from `~/.ssh/config` give HostName, User, Port, IdentityFile); `-o ProxyCommand=...` (`%h`, `%p`, `%r`); `-o StrictHostKeyChecking=no` skips the host key check; other options are ignored | — |
| `onigirazu_connection` / `ansible_connection` | `ssh`; `local` (this machine); `docker` or `podman` (also `community.docker.docker`, `containers.podman.podman`): commands run with `docker exec` in the container named by `ansible_host`, else the host name, as `ansible_user` if set; no SSH in the container; `winrm`: a Windows host (win_* modules); `ssh` with `ansible_shell_type: powershell` (or `cmd`) is a Windows host over OpenSSH | ssh |
| `ansible_winrm_transport`, `ansible_winrm_scheme`, `ansible_winrm_message_encryption`, `ansible_winrm_server_cert_validation`, `ansible_winrm_ca_trust_path`, `ansible_winrm_read_timeout_sec` | WinRM: `ntlm` (default) or `basic`; `https` unless the port is 5985 (default port 5986); encryption `auto` (NTLM over http), `always`, `never`; `ignore` skips the certificate check | — |
| any other key | host variable for templates | — |

These variables also work as group variables (`all: vars: ansible_user: deploy`, group_vars
files): a host takes them unless it sets its own.

`apply -u USER` and `--private-key FILE` (`run -u`/`-k`) override user and key for every host.
As in Ansible, these variables given with `-e` override the inventory for every host
(`-e ansible_user=ansible`, `-e @bootstrap.yml` with `ansible_password` and
`ansible_become_password`).

## group_vars and host_vars

As in Ansible, variables are also read from `group_vars/<group>.yml` and
`host_vars/<host>.yml` (`.yaml`, `.json`, no extension, or a directory of such files
merged in name order). They are looked up next to each inventory file, inside an
inventory directory, and, for `apply`/`plan`/`drift`, next to the playbook (read last,
so they win). They override variables written in the inventory itself.

## Several sources and directories

- `-i a.yml -i b.ini` (or `-i a.yml,b.ini`): sources are merged; for the same host or
  group the later source wins.
- `-i inventory/`: every `.yml`, `.yaml`, `.json`, `.ini` and `.toml` file (alphabetically)
  and then every executable in the directory tree is loaded. Other files, hidden
  directories and `group_vars/`/`host_vars/` are skipped.

## Without -i

`ANSIBLE_INVENTORY` (a comma separated list of sources) is the default of `-i`. Without it,
`run` requires `-i`. `apply` (and `plan`/`drift`) look in the playbook's directory for
`inventory.yml`, `inventory.yaml`, `inventory.toml`, `inventory.json`, `inventory.ini`,
`hosts`, `hosts.yml`, `hosts.yaml`, `hosts.toml`, `hosts.json`, `hosts.ini`, `inventory`,
then in `/etc/onigirazu/` for `inventory.yml`, `hosts.yml`, `inventory`. When none is found,
the inventory is `localhost` alone, run on the control machine.

## Limitations

- `onigirazu inventory` takes a single `-i`.

## Inspecting an inventory

```bash
onigirazu inventory --list -i inventory.yml       # hosts and groups
onigirazu inventory --graph -i inventory.yml      # group tree
onigirazu inventory --host web1 -i inventory.yml  # groups of one host
onigirazu inventory --list --json -i inventory.yml   # as ansible-inventory --list
onigirazu inventory --host web1 --json -i inventory.yml  # as ansible-inventory --host
```

`--json` prints what `ansible-inventory` prints: groups with their own hosts and children,
`all` with the top groups and `ungrouped`, `_meta.hostvars` with each host's variables (group
variables resolved, templated connection variables rendered). Passwords are left out. Messages
go to stderr, so the output can be piped to a script.
