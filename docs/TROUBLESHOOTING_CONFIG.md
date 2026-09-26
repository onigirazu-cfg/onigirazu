# Troubleshooting

## `Error: unknown command "site.yml" for "onigirazu"`

The playbook must follow a subcommand:

```bash
onigirazu apply site.yml -i inventory.yml
```

## My onigirazu.yml is not used

- Lookup order: `-c FILE`, `onigirazu.yml` next to the playbook, `/etc/onigirazu/onigirazu.yml`.
  `~/.onigirazu/onigirazu.yml` is not read.
- A `-c` path that does not exist is silently ignored; the defaults are used.
- Unknown or misspelled keys are silently ignored.
- Most keys have no effect; see the list in [CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md).
- A key in the file wins over the matching `ONIGIRAZU_*` environment variable.

## `check_mode: true` in onigirazu.yml still changes hosts

Current limitation: the config key is not honoured by task execution. Use
`onigirazu apply site.yml --check` or `onigirazu plan site.yml`.

## `invalid security policy …: unknown or unsupported keys: …`

The policy is strict JSON. Remove the listed keys. `require_encryption`,
`required_permissions`, `audit_enabled` and `log_level` are rejected because they are
not enforced. Valid keys: [SECURITY_POLICY_GUIDE.md](SECURITY_POLICY_GUIDE.md).

## `security validation failed: …`

A policy file was found and the task violates it. `apply` logs
`Security policy loaded from <file>`; the search order is `--security-policy`,
`$ONIGIRAZU_SECURITY_POLICY`, `./security-policy.json`,
`~/.onigirazu/security-policy.json`, `/etc/onigirazu/security-policy.json`.
Change the task, or the policy (for example add the path to `allowed_directories`).

## `inventory source not found: …/192.168.1.10`

Current limitation: inline inventories (`-i "host1,host2"`, `-i 192.168.1.10,`) are not
supported by `apply`, `plan`, `drift` and `run`. Put the hosts in a file:

```bash
printf '192.168.1.10\n192.168.1.11\n' > hosts.txt
onigirazu run all -m ping -i hosts.txt
```

## `inventory must contain at least one group`

Current limitation: a plain host list with `user@host` or `host:port` lines is not
parsed. Use an INI file instead:

```ini
[web]
web1 ansible_host=192.168.1.10 ansible_user=deploy ansible_port=2222
```

## `no valid hosts found in Ansible inventory`

Current limitation: in Ansible-format YAML only hosts listed under `all.hosts` are
loaded. List every host under `all.hosts` (groups may then refer to them by name),
or use an INI inventory.

## Host key verification fails

With `ssh_strict_host_key: true` every host must be in `~/.ssh/known_hosts` (or
`ssh_known_hosts_file`). Add it with `ssh-keyscan -H host >> ~/.ssh/known_hosts`.

## More output

`-v` (debug log) or `--show-debug`. For `run`, `-V` shows detailed results.

## Docker

The image is `ghcr.io/onigirazu-cfg/onigirazu`; the binary is the entrypoint, so pass
the subcommand:

```bash
docker run --rm -v "$PWD:/work" -w /work ghcr.io/onigirazu-cfg/onigirazu \
  apply site.yml -i inventory.yml
```
