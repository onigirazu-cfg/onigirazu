# SSH Host Key Checking

## Default behaviour

- Host keys are checked against `~/.ssh/known_hosts` (`ssh_known_hosts_file` in
  `onigirazu.yml` names another file; `~` is the home directory). The file is read with
  OpenSSH's rules: plain and hashed names (`|1|…`), `[host]:port` for other ports, several
  names per line, `@revoked`. Lines that hold no key are skipped.
- A host without an entry is accepted and one OpenSSH line is appended
  (`web1 ssh-ed25519 AAAA…`, `[10.0.0.5]:2222 …` for another port). With
  `ssh_strict_host_key: true` an unknown host fails instead.
- A host whose key differs from its entry fails with `the host key changed`. Remove the
  stale entry (`ssh-keygen -R <host>` or `ssh-keygen -R '[host]:port'`) and connect again.
- Earlier versions appended lines with the key type written twice
  (`host ssh-ed25519 ssh-ed25519 AAAA…`), one per run. They are skipped; to remove them:
  `grep -vE '^\S+ (ssh-[a-z0-9]+|ecdsa-sha2-nistp[0-9]+) (ssh-[a-z0-9]+|ecdsa-sha2-nistp[0-9]+) ' ~/.ssh/known_hosts > kh.tmp && mv kh.tmp ~/.ssh/known_hosts`.

## Disabling the check for a host

Two inventory settings mark a host as "ignore host key":

- `insecure_ignore_host_key: true`
- `ansible_ssh_common_args` (or `ansible_ssh_extra_args`) containing
  `-o StrictHostKeyChecking=no`; `ansible_ssh_host_key_checking: false` in Ansible YAML

The mark applies to every connection to the host: fact gathering, tasks, `run` and
`healthcheck`.

Where each setting is recognised:

| Inventory | Recognised | Not recognised |
|-----------|------------|----------------|
| Onigirazu YAML (`groups:`) | host key `insecure_ignore_host_key: true`, host key `ansible_ssh_common_args` | the same keys under group `vars` |
| TOML | `insecure_ignore_host_key = true` in `[hosts.X]` or in group `vars` | |
| JSON | `"insecure_ignore_host_key": true` in the host object inside the group's `hosts` | |
| Ansible YAML (`all:`) | host or group variable `insecure_ignore_host_key: true`, host `ansible_ssh_host_key_checking: false`, group variable `ansible_ssh_common_args` | host variable `ansible_ssh_common_args` |
| INI | `[group:vars]` line `ansible_ssh_common_args=-o StrictHostKeyChecking=no`; on a host line only without spaces: `ansible_ssh_common_args=-oStrictHostKeyChecking=no` | `insecure_ignore_host_key=true` (read as a string), a quoted value with spaces on a host line |
| Plain host list | | nothing |

A group can switch the mark on but a host cannot switch it off again:
`insecure_ignore_host_key: false` on a host does not override `true` from its group.

`default_insecure_ignore_host_key` in `onigirazu.yml` and the environment variable
`ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY` have no effect.

## Examples

```yaml
# inventory.yml (Onigirazu YAML)
groups:
  dev:
    hosts:
      dev1:
        onigirazu_host: 192.168.56.10
        onigirazu_user: vagrant
        insecure_ignore_host_key: true
```

```ini
# inventory.ini
[dev]
dev1 ansible_host=192.168.56.10

[dev:vars]
ansible_ssh_common_args=-o StrictHostKeyChecking=no
```

The setting belongs in the inventory; as a module argument in a task it is not a
connection option.
