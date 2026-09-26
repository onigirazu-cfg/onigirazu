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

- A `-c` file that does not exist is silently ignored and the defaults are used.
- Unknown keys are silently ignored, so check spelling.
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
| `log_format` | `ONIGIRAZU_LOG_FORMAT` | `text` | `text` or `json`. `apply --log-format` overrides it. |
| `state_file` | `ONIGIRAZU_STATE_FILE` | `.onigirazu-state` | State file. `-s` overrides it. |
| `dry_run` | `ONIGIRAZU_DRY_RUN` | `false` | Tasks run in check mode. Prefer `apply --check`. |
| `color_output` | `ONIGIRAZU_COLOR_OUTPUT` | `true` | Colored output. `--no-color` overrides it. |
| `ssh_strict_host_key` | `ONIGIRAZU_SSH_STRICT_HOST_KEY` | `false` | Reject hosts whose key is not in the known_hosts file. |
| `ssh_known_hosts_file` | `ONIGIRAZU_SSH_KNOWN_HOSTS_FILE` | `~/.ssh/known_hosts` | known_hosts file for host key checks. |
| `enable_metrics` | `ONIGIRAZU_ENABLE_METRICS` | `false` | Start a Prometheus endpoint during `apply` (`/metrics`, `/health`, `/summary`). |
| `metrics_listen_address` | `ONIGIRAZU_METRICS_LISTEN_ADDRESS` | `127.0.0.1` | Listen address of the metrics server. |
| `metrics_port` | `ONIGIRAZU_METRICS_PORT` | `9090` | Port of the metrics server. |
| `metrics_auth_token` | `ONIGIRAZU_METRICS_AUTH_TOKEN` | empty | If set, requests need `Authorization: Bearer <token>`. |
| `metrics_ip_whitelist` | `ONIGIRAZU_METRICS_IP_WHITELIST` | empty | If set, only these client IPs may connect (env: comma-separated). |

Durations are written as `30s`, `5m`, `1h`.

Current limitation: `check_mode: true` (or `ONIGIRAZU_CHECK_MODE=true`) prints
"Running in check mode" but tasks still make changes. Use `apply --check`.

## Keys that are accepted but have no effect

These keys are parsed but nothing reads them. Leave them out:

`default_timeout`, `retry_attempts`, `retry_delay`, `config_file`,
`allow_shell_commands`, `blocked_commands` (use the security policy instead),
`enable_caching`, `cache_ttl`, `enable_checksum`, `enable_parallel`,
`parallel_strategy`, `verbose`, `show_diff`, `progress_bar`, `interactive_mode`,
`output_format`, `metrics_path` (the path is always `/metrics`), `enable_profiling`,
`ssh_timeout`, `ssh_keepalive`, `ssh_max_sessions`, `connection_reuse`,
`default_insecure_ignore_host_key`, `vault_enabled`, `vault_address`, `vault_token`,
`preferred_module_syntax`, `enforce_module_syntax`.

Use command-line flags instead: `apply --timeout`, `--diff`, `--interactive`,
`-o json|yaml`, `--profile`; task-level `retries`/`delay` for retries.

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
