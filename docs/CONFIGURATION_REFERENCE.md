# Configuration Reference

Settings for the control machine (where you run `onigirazu`) live in `onigirazu.yml`.
Restrictions on what a playbook may do live in a separate security policy, see
[SECURITY_POLICY_GUIDE.md](SECURITY_POLICY_GUIDE.md).

## Where the file comes from

`apply` (and `plan`/`drift`, which run through it) uses the first match:

1. `-c/--config <file>`
2. `onigirazu.yml` in the playbook's directory
3. `/etc/onigirazu/onigirazu.yml`
4. built-in defaults

`run` uses 1, 3 and 4 (it has no playbook directory).

Notes:

- A `-c` file that does not exist is an error.
- Unknown keys and keys that have no effect are reported as warnings at start.
- No file is read from `~/.onigirazu/`.

## Order of precedence

```
command-line flags > onigirazu.yml > ONIGIRAZU_* environment variables > defaults
```

Environment variables only change the defaults; a key in `onigirazu.yml` wins over them.

## Settings that take effect

| Key | Env variable | Default | Effect |
|-----|--------------|---------|--------|
| `max_concurrency` | `ONIGIRAZU_MAX_CONCURRENCY` | `10` | Hosts worked on in parallel. `apply -f N` overrides it (when N is not 10). |
| `log_level` | `ONIGIRAZU_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`. `apply -l` and `-v` override it. |
| `log_format` | `ONIGIRAZU_LOG_FORMAT` | `text` | `text` or `json` ([MACHINE_OUTPUT.md](MACHINE_OUTPUT.md)). `apply --log-format` overrides it. |
| `state_file` | `ONIGIRAZU_STATE_FILE` | `.onigirazu-state` | State file. `-s` overrides it. |
| `roles_path` | `ANSIBLE_ROLES_PATH` | - | Directories searched for roles after `roles/` next to the playbook (list; relative to the config file). ansible.cfg `roles_path` is read too. |
| `collections_path` | `ANSIBLE_COLLECTIONS_PATH` | `~/.ansible/collections` | Where `namespace.collection.role` roles are found (`ansible_collections/<ns>/<coll>/roles/<role>`). ansible.cfg `collections_path` is read too. |
| `secrets` | - | - | Secret providers for templates: `cache_ttl` (default `5m`), `vault` with `address` (default `VAULT_ADDR`), `namespace`, `mount` (default `secret`). Credentials come from the environment. See [BITWARDEN_INTEGRATION.md](BITWARDEN_INTEGRATION.md). |
| `ansible_bridge` | - | - | Modules that run through ansible-core: `modules` (names or patterns such as `community.general.*`, `win_*`), `ansible_playbook` (default from `PATH`). See [ANSIBLE_BRIDGE.md](ANSIBLE_BRIDGE.md). |
| `managed_state` | - | file next to the playbook | Where the managed state lives (`ONIGIRAZU_MANAGED_STATE_DIR` moves the files of `backend: file`): `backend: file` or `s3` with `bucket`, `prefix`, `endpoint`, `region`, `insecure`, `path_style`. See [MANAGED_STATE.md](MANAGED_STATE.md). |
| `dry_run` | `ONIGIRAZU_DRY_RUN` | `false` | Tasks run in check mode. Prefer `apply --check`. |
| `color_output` | `ONIGIRAZU_COLOR_OUTPUT` | `true` | Colored output. `--no-color` overrides it. |
| `ssh_strict_host_key` | `ONIGIRAZU_SSH_STRICT_HOST_KEY` | `false` | Reject hosts whose key is not in the known_hosts file. |
| `ssh_known_hosts_file` | `ONIGIRAZU_SSH_KNOWN_HOSTS_FILE` | `~/.ssh/known_hosts` | known_hosts file for host key checks. |
| `remote_server` | `ONIGIRAZU_REMOTE_SERVER` | `auto` | Command server on SSH hosts: `auto`, `python` or `sh` (see below). |
| `enable_metrics` | `ONIGIRAZU_ENABLE_METRICS` | `false` | Start a Prometheus endpoint during `apply` (`/metrics`, `/health`, `/summary`). |
| `metrics_listen_address` | `ONIGIRAZU_METRICS_LISTEN_ADDRESS` | `127.0.0.1` | Listen address of the metrics server. |
| `metrics_port` | `ONIGIRAZU_METRICS_PORT` | `9090` | Port of the metrics server. |
| `metrics_auth_token` | `ONIGIRAZU_METRICS_AUTH_TOKEN` | empty | If set, requests need `Authorization: Bearer <token>`. |
| `metrics_ip_whitelist` | `ONIGIRAZU_METRICS_IP_WHITELIST` | empty | If set, only these client IPs may connect (env: comma-separated). |
| `ssh_timeout` | `ONIGIRAZU_SSH_TIMEOUT` | `30s` | Time to connect to a host. |
| `show_diff` | — | `false` | As `apply --diff`. |
| `default_timeout` | — | none | As `apply --timeout` (the whole run; state, audit and snapshot are still saved). |
| `verbose` | — | `false` | Debug logging, as `apply -v`. |
| `output_format` | — | `text` | As `apply -o` (`text`, `json`, `yaml`). |
| `interactive_mode` | — | `false` | As `apply --interactive` (TUI). |

`show_diff`, `default_timeout`, `verbose`, `output_format` and `interactive_mode` apply only when written in the file, and the
command-line flag wins.

Durations are written as `30s`, `5m`, `1h`.

`check_mode: true` and `dry_run: true` (or their `ONIGIRAZU_*` variables) are the
same as `apply --check`: no task changes anything.

### Command server on SSH hosts (`remote_server`)

onigirazu runs commands on an SSH host through one long-lived command server per connection (and one started with
`sudo` for become). The agent and the Python server also read and write the files of file tasks themselves; the
`sh` server runs commands for that. `remote_server` picks it:

| Value | Server |
|-------|--------|
| `auto` (default) | `onigirazu-agent`, else Python, else `sh` |
| `python` | Python (needs `python3`), else `sh` |
| `sh` | POSIX `sh` (needs `sh`, `base64`, `stat`) |

`onigirazu-agent` is a small Go binary. On the first connection to a host it is uploaded to
`~/.onigirazu/bin/onigirazu-agent-<hash>` and reused until the version changes. Release builds carry it for Linux
amd64, arm64, arm and 386; for other platforms put `onigirazu-agent-<os>-<arch>` next to `onigirazu` or in
`ONIGIRAZU_AGENT_DIR`. Each fallback is automatic: no binary for the platform, a `noexec` home, or a become user
without access to the login user's home falls to Python, a host without `python3` to `sh`.
`ONIGIRAZU_NO_PYTHON=1` is the same as `remote_server: sh`.

Windows hosts over OpenSSH (`ansible_shell_type: powershell` or `cmd`) get the Windows build of the agent
(amd64, arm64) at `%USERPROFILE%\.onigirazu\bin\onigirazu-agent-<hash>.exe`; it runs each command as a PowerShell
script and writes files natively, so a play needs no session per task. Without it (no binary for the platform)
every command is its own SSH session.

## Keys that are accepted but have no effect

These keys are parsed but nothing reads them; a file that sets one gets a warning:

`retry_attempts`, `retry_delay`, `config_file`,
`allow_shell_commands`, `blocked_commands` (use the security policy instead),
`enable_caching`, `cache_ttl`, `enable_checksum`, `enable_parallel`,
`parallel_strategy`, `progress_bar`, `metrics_path` (the path is always `/metrics`), `enable_profiling`,
`ssh_keepalive`, `ssh_max_sessions`, `connection_reuse`,
`default_insecure_ignore_host_key`, `vault_enabled`, `vault_address`, `vault_token` (use `secrets.vault`),
`preferred_module_syntax`, `enforce_module_syntax`.

Use command-line flags instead: `--profile`; task-level `retries`/`delay` for retries.

## Example

```yaml
# onigirazu.yml
max_concurrency: 20
log_level: info
log_format: json
ssh_strict_host_key: true
enable_metrics: true
metrics_port: 9090
```

## Checking the effective settings

With `-v`, `apply` logs the loaded values (`Configuration loaded: max_concurrency=…, log_level=…`).
Pass the file explicitly with `-c` when in doubt:

```bash
onigirazu apply site.yml -i hosts.yml -c ./onigirazu.yml --check -v
```
