# Security Policy

A security policy restricts what `onigirazu apply` may do. It lives on the control
machine (where you run `onigirazu`). **Without a policy file nothing is restricted.**

## Where the policy comes from

The first match wins:

1. `--security-policy <file>`
2. `$ONIGIRAZU_SECURITY_POLICY`
3. `./security-policy.json`
4. `~/.onigirazu/security-policy.json`
5. `/etc/onigirazu/security-policy.json`

An explicitly given file (1, 2) must exist. `apply` logs which file was loaded.

## Format

JSON. Every key is optional; a missing key means "no restriction". Keys starting
with `_` are comments. Unknown keys are an error, so typos do not silently disable
a restriction.

| Key | Effect |
|-----|--------|
| `allowed_hosts` | Host address or inventory name must match one entry (`*` wildcards) |
| `allowed_ports` | SSH port of the host must be listed |
| `allowed_modules` | Only these modules may run |
| `allowed_directories` | `dest` of `copy`/`template` and `path` of `file` must be under one of them |
| `blocked_directories` | Those paths are rejected; wins over `allowed_directories` |
| `allowed_file_types` | Extension of `copy`/`template` `dest` must be listed (e.g. `".conf"`) |
| `blocked_commands` | `shell`/`command` commands containing any of these substrings are rejected |
| `max_file_size` | Maximum inline `content` size of the `file` module, bytes |
| `max_retries` | Maximum `retries` of a task |
| `max_timeout` | Maximum task timeout: `"30m"` or nanoseconds |
| `strict` | `true` adds heuristic checks, see below |

`strict: true` also rejects:

- `shell`/`command` with `;`, `$(...)`, backticks, piping into a shell, or mixed/long `&&`/`||`/`|` chains
- `rm -rf`, `dd if=`, `mkfs`, `fdisk` anywhere in a task (name or arguments)
- changes to system users/groups (`root`, `nobody`, `sudo`, …) and UID/GID 0
- `..` in destination paths

## Example

See [examples/security-policy.json](../examples/security-policy.json).

```bash
onigirazu apply site.yml -i inventory.yml --security-policy examples/security-policy.json
```

A task that violates the policy fails with `security validation failed: …`.
