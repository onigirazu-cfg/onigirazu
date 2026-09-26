# Quick Start

Run a first playbook, then add configuration and a security policy only if you need them.

## 1. Inventory

```yaml
# inventory.yml
groups:
  web:
    hosts:
      web1:
        onigirazu_host: 192.168.1.10
        onigirazu_user: deploy
```

Ansible-style YAML and INI inventories work too, see [INVENTORY_FORMATS.md](INVENTORY_FORMATS.md).
Variables can also come from `group_vars/` and `host_vars/` next to the inventory or the playbook.

## 2. Playbook

```yaml
# site.yml
- name: Web servers
  hosts: web
  become: true
  tasks:
    - name: Install nginx
      package:
        name: nginx
        state: present
```

## 3. Run

```bash
onigirazu plan site.yml -i inventory.yml          # what would change, nothing is changed
onigirazu apply site.yml -i inventory.yml --check # same run in check mode
onigirazu apply site.yml -i inventory.yml         # apply
onigirazu apply site.yml -i inventory.yml --diff  # apply and show file diffs
```

Frequently used `apply` flags:

| Flag | Meaning |
|------|---------|
| `-e key=value`, `-e @vars.yml`, `-e '{"k":"v"}'` | extra variables, override all others (repeatable) |
| `--limit web1` / `--limit 'web:!web3'` | run only on matching hosts |
| `--tags a,b` / `--skip-tags c` | select tasks by tag |
| `-b`, `--become-user USER` | privilege escalation in every play |
| `-u USER`, `--private-key FILE` | SSH user and key for every host |
| `--start-at-task NAME` | skip tasks until this one |
| `-C/--check`, `-d/--diff` | check mode, file diffs |
| `-o json` / `-o yaml` | machine-readable result on stdout |
| `-f N` | hosts in parallel (default 10) |

The playbook is always given to a subcommand: `onigirazu apply site.yml`, not `onigirazu site.yml`.

To check hosts later against the same playbook, or undo a run, see
[DRIFT_AND_ROLLBACK.md](DRIFT_AND_ROLLBACK.md).

## 4. Optional: onigirazu.yml

Put `onigirazu.yml` next to the playbook or in `/etc/onigirazu/`, or pass `-c FILE`:

```yaml
max_concurrency: 20
log_format: json
ssh_strict_host_key: true
```

Only a few keys take effect; see [CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md).

## 5. Optional: security policy

Without a policy file nothing is restricted. To restrict what playbooks may do, create
`security-policy.json` (current directory, `~/.onigirazu/`, `/etc/onigirazu/`, or
`--security-policy FILE`):

```json
{
  "allowed_directories": ["/tmp", "/opt", "/srv"],
  "blocked_directories": ["/etc/shadow", "/boot"],
  "blocked_commands": ["rm -rf /", "mkfs"]
}
```

Unknown keys make the policy invalid. A task that violates it fails with
`security validation failed: …`; for "path is not in allowed directories", add the
directory to `allowed_directories`. All keys: [SECURITY_POLICY_GUIDE.md](SECURITY_POLICY_GUIDE.md).

## Next

- [ADHOC_GUIDE.md](ADHOC_GUIDE.md) — one-off commands without a playbook
- [VARIABLES_CHEATSHEET.md](VARIABLES_CHEATSHEET.md) — variables and precedence
- [modules/README.md](modules/README.md) — module reference
- [TROUBLESHOOTING_CONFIG.md](TROUBLESHOOTING_CONFIG.md) — common problems
