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
| `ansible_user` | SSH user (default `root`) |
| `ansible_password` | SSH password |
| `ansible_ssh_private_key_file` | private key |
| `ansible_ssh_host_key_checking` | `false` disables host key verification for this host |
| anything else | host or group variable, available in templates |

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

## Troubleshooting

- `no valid hosts found in Ansible inventory`: the file has no host under `all` (hosts
  must be inside the `all:` tree).
- More output: `-v` or `--show-debug`.

Example inventory: `examples/inventory-ansible-full.yml`.
