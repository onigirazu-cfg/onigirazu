# Using Ansible Inventories

Onigirazu reads Ansible inventories in INI and YAML format, including `group_vars/` and
`host_vars/` next to the inventory or the playbook.

```bash
onigirazu plan site.yml -i hosts.yml    # what would change (check mode on the hosts)
onigirazu apply site.yml -i hosts.yml
```

## INI

Ansible INI files work as they are: groups, `[group:vars]`, `[group:children]`,
host variables on the host line. See [INVENTORY_FORMATS.md](INVENTORY_FORMATS.md#ini).

## YAML

A YAML file with a top-level `all:` key is read as an Ansible inventory:

```yaml
# hosts.yml
all:
  hosts:
    app1:
      ansible_host: app1.example.com
      ansible_user: appuser
      ansible_ssh_private_key_file: ~/.ssh/app_key
      version: "2.1.0"
    app2:
      ansible_host: app2.example.com
      ansible_user: appuser
    db1:
      ansible_host: db1.example.com
      ansible_user: postgres
  children:
    appservers:
      hosts:
        app1:
        app2:
      vars:
        service_port: 8080
    databases:
      hosts:
        db1:
      vars:
        backup_enabled: true
    production:
      children:
        appservers: {}
        databases: {}
      vars:
        env: production
```

Hosts may be defined under `all.hosts` or directly in any group, at any depth of
`children`. A host listed in several groups gets the settings of all of them.

`children` of a group may be a map (Ansible style) or a list of group names.

## Recognised variables

| Variable | Meaning |
|----------|---------|
| `ansible_host` | address to connect to (default: the host name) |
| `ansible_port` | SSH port (default 22) |
| `ansible_user` | SSH user (default: the local `$USER`) |
| `ansible_password` | SSH password |
| `ansible_ssh_private_key_file` | private key |
| `ansible_become_password` (`ansible_become_pass`, `ansible_sudo_pass`) | sudo password for `become` |
| `ansible_ssh_host_key_checking` | `false` marks the host as "ignore host key", see [host key checking](examples/README_insecure_ignore_host_key.md) |
| anything else | host or group variable, available in templates |

On a host, any other `ansible_*` variable loses its prefix: `ansible_python_interpreter`
becomes `python_interpreter`, and `ansible_connection: local` is not recognised (the host
is reached over SSH). The connection variables above also work as group variables
(`all.vars`, a group's `vars`, `group_vars/`).

Top-level keys outside `all:` are ignored; variables belong on a host, in a group's
`vars`, in `all.vars`, or in `group_vars/`/`host_vars/`.

## Playbooks

Playbooks use the Ansible layout: a list of plays with `hosts`, `vars` and `tasks`.

```yaml
- name: Deploy application
  hosts: appservers
  tasks:
    - name: Show version
      debug:
        msg: "Deploying {{ version }} on port {{ service_port }}"
```

## Limitations

- Host ranges (`web[1:3]`) are not expanded; the name is taken literally.
- A file with hosts only under `all.hosts` and neither `all.vars` nor a child group fails
  with `inventory must contain at least one group`. Add a group or `all.vars`.

## Troubleshooting

- `no valid hosts found in Ansible inventory`: the file has no host under `all` (hosts
  must be inside the `all:` tree).
- More output: `-v` or `--show-debug`.
- `onigirazu inventory --host NAME --json -i hosts.yml` prints the variables a host ends
  up with.

Example inventory: `examples/inventory-ansible-full.yml`.
