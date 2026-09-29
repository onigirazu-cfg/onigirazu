# Metrics Endpoint

With `enable_metrics: true`, `onigirazu apply` (and `plan`/`drift`, which run through it)
starts an HTTP server for the duration of the run. It stops when the process exits.

The counters cover the run so far: `onigirazu_playbooks_total`, `onigirazu_plays_total`,
`onigirazu_tasks_total{status,module}` (status `success`, `failed`, `skipped`), task
durations per module and host, errors and module usage.

## Configuration

| Key | Environment | Default |
|-----|-------------|---------|
| `enable_metrics` | `ONIGIRAZU_ENABLE_METRICS` | `false` |
| `metrics_listen_address` | `ONIGIRAZU_METRICS_LISTEN_ADDRESS` | `127.0.0.1` |
| `metrics_port` | `ONIGIRAZU_METRICS_PORT` | `9090` |
| `metrics_auth_token` | `ONIGIRAZU_METRICS_AUTH_TOKEN` | empty (no token check) |
| `metrics_ip_whitelist` | `ONIGIRAZU_METRICS_IP_WHITELIST` (comma-separated) | empty (any client) |

`metrics_path` has no effect: the paths are fixed. See
[CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md) for how keys and environment
variables combine.

```yaml
# onigirazu.yml
enable_metrics: true
metrics_port: 9090
metrics_auth_token: a-long-random-token
```

## Endpoints

| Path | Content |
|------|---------|
| `/metrics` | Prometheus text format |
| `/summary` | JSON summary (durations are in nanoseconds) |
| `/health` | `OK` |

All three go through the same checks, in this order:

1. IP whitelist (if set): the address of the connecting peer must equal one of the
   entries exactly, otherwise `403 Forbidden: IP not in whitelist`. CIDR ranges are not
   supported. `X-Forwarded-For` and `X-Real-IP` are ignored: behind a reverse proxy the
   peer is the proxy.
2. Token (if set): the request needs `Authorization: Bearer <token>`, otherwise `401`.

```bash
curl -H "Authorization: Bearer a-long-random-token" http://127.0.0.1:9090/metrics
```

Prometheus series: `onigirazu_tasks_total{status,module}`,
`onigirazu_task_duration_seconds{module,host}`, `onigirazu_playbooks_total`,
`onigirazu_plays_total`, `onigirazu_cache_hit_rate`, `onigirazu_hosts_connected`,
`onigirazu_concurrent_tasks`, `onigirazu_errors_total{type,module}`,
`onigirazu_module_usage_total{module}`.

The server uses read/write timeouts of 15 s and an idle timeout of 60 s. It speaks plain
HTTP only.

## Security notes

- The default listen address `127.0.0.1` keeps the endpoint local. Change it only
  together with `metrics_auth_token`.
- The token is compared as a plain string. Keep it out of files in version control:
  set it through `ONIGIRAZU_METRICS_AUTH_TOKEN`.
- Terminate TLS in a reverse proxy if the endpoint must leave the host.
- Metric labels contain host and module names.

## Troubleshooting

- Connection refused: `enable_metrics` is not set, the run has already finished, or the
  port is taken (the log then shows `Metrics server error: …` and the run continues).
- `403`: the peer address is not in `metrics_ip_whitelist`.
- `401`: the `Authorization` header is missing or is not `Bearer <token>`.
