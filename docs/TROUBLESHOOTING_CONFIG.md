# Troubleshooting

## `Error: unknown command "site.yml" for "onigirazu"`

The playbook must follow a subcommand:

```bash
onigirazu apply site.yml -i inventory.yml
```

## My onigirazu.yml is not used

- Lookup order: `-c FILE`, `onigirazu.yml` next to the playbook, `/etc/onigirazu/onigirazu.yml`.
  `~/.onigirazu/onigirazu.yml` is not read.
- A `-c` path that does not exist is an error.
- Unknown or misspelled keys, and keys that have no effect, are logged as warnings at start
  (list in [CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md)).
- A key in the file wins over the matching `ONIGIRAZU_*` environment variable.

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

A single host on the command line needs a trailing comma, as in Ansible:
`-i 192.168.1.10,`. Without a comma the value is a file name.

## `no valid hosts found in Ansible inventory`

An Ansible-format YAML file needs its hosts inside the `all:` tree (under
`all.hosts` or in any group below `all.children`). Top-level keys other than `all`
are ignored.

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
