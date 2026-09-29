# Ansible YAML Inventory Examples

Recognised variables, limitations and troubleshooting:
[ANSIBLE_INVENTORY_QUICK_START.md](../ANSIBLE_INVENTORY_QUICK_START.md). All formats:
[INVENTORY_FORMATS.md](../INVENTORY_FORMATS.md).

A YAML file with a top-level `all:` key is read as an Ansible inventory; other top-level
keys are ignored.

## Hosts and groups

```yaml
all:
  vars:
    ansible_user: deploy
  hosts:
    web1:
      ansible_host: 192.168.1.10
    web2:
      ansible_host: 192.168.1.11
      ansible_port: 2222
    db1:
      ansible_host: 192.168.1.20
      ansible_user: postgres
      ansible_ssh_private_key_file: ~/.ssh/db_key
  children:
    webservers:
      hosts:
        web1:
        web2:
      vars:
        http_port: 80
    databases:
      hosts:
        db1:
      vars:
        db_port: 5432
```

## Nested groups

`children` may be a map (Ansible style) or a list of group names.

```yaml
all:
  children:
    webservers:
      hosts:
        web1: {ansible_host: 192.168.1.10}
      vars:
        tier: frontend
    databases:
      hosts:
        db1: {ansible_host: 192.168.1.20}
    production:
      children:
        - webservers
        - databases
      vars:
        env: production
```

## Usage

```bash
onigirazu inventory --graph -i inventory.yml
onigirazu plan playbook.yml -i inventory.yml
onigirazu apply playbook.yml -i inventory.yml
```

Variables other than the connection variables keep their names (`http_port` is
`{{ http_port }}`), except that a host's own `ansible_*` variables lose the prefix.
