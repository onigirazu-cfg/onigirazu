# SSH Host Key Checking

## Default behaviour

- Host keys are checked against `~/.ssh/known_hosts`. The config keys
  `ssh_known_hosts_file` and `ssh_strict_host_key` currently have no effect on `run` and
  `apply`.
- An entry matches when its host field is exactly `address:port` (for example
  `10.0.0.5:22`) or the plain IP address of the server. Hashed entries (`|1|…`, as written
  by `ssh-keygen -H` or `ssh-keyscan -H`), `[host]:port` entries and plain host names are
  not matched.
- A host without a matching entry is accepted and its key is appended to
  `~/.ssh/known_hosts`. That line is not read back on the next run, so such a host is
  accepted again every time.
- A host whose key differs from a matching entry fails with
  `host key verification failed for …: key mismatch`. Remove the stale entry
  (`ssh-keygen -R <ip>`) and connect again.

## Disabling the check for a host

Two inventory settings mark a host as "ignore host key":

- `insecure_ignore_host_key: true`
- `ansible_ssh_common_args` (or `ansible_ssh_extra_args`) containing
  `-o StrictHostKeyChecking=no`; `ansible_ssh_host_key_checking: false` in Ansible YAML

Currently only fact gathering honours this mark. Module tasks of `run` and `apply` connect
through the connection pool, which checks the key anyway, so a changed key still fails
them.

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
