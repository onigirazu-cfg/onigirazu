# Fleet server

`onigirazu serve` is a small server in the same binary: jobs run on a schedule or on demand
(drift checks, applies, `verify`, compliance profiles), their results are kept, and a web page
and a REST API show them per job and per host together with the run history of the audit
store. One process, one YAML file, no database.

```yaml
# serve.yml
listen: ":8086"
title: RED fleet
data_dir: /var/lib/onigirazu/serve     # results and job state; default ~/.onigirazu/serve
retain: 50                             # results kept per job
auth:
  token: ${SERVE_TOKEN}                # API bearer token: an operator
  user_header: X-Authentik-Username    # a reverse proxy's identity: a viewer ...
  groups_header: X-Authentik-Groups    # ... an operator when one of the groups is listed
  operators: [admins]
metrics:
  push: http://victoria:8428/api/v1/import/prometheus   # every job's metrics, optional
  labels: {env: prod}
jobs:
  - name: drift-site
    kind: drift                        # drift (default), apply, verify, comply
    playbook: site.yml
    inventory: hosts.yml
    every: 30m                         # 0 / absent: on demand only
    notify: [https://mattermost.example.com/hooks/xxx]
  - name: ssh-compliance
    kind: comply
    profile: ssh                       # bundled or a file
    inventory: hosts.yml
    every: 6h
    fail_on: high
  - name: deploy-web
    kind: apply
    playbook: deploy.yml
    inventory: hosts.yml
    limit: web
    become: true
```

```bash
onigirazu serve --config-file serve.yml
sudo onigirazu serve install -f /etc/onigirazu/serve.yml      # systemd service
```

Jobs run one at a time, in the order they are due or requested; a job that is running or
queued is not queued again. Paths in the file are relative to it. Every job's state file lives
in `data_dir/state/<job>.state`.

## Access

Without `auth` everyone is an operator — put the server behind a reverse proxy or a firewall.
With `token`, `Authorization: Bearer <token>` is an operator (automation, curl). With
`user_header`, the user the proxy sets (Authentik's outpost, oauth2-proxy, nginx auth_request) is
a viewer; when one of the groups in `groups_header` (comma, pipe or space separated) is in
`operators`, an operator. Viewers read everything; operators also start jobs.

## API

| Method and path | Returns |
|---|---|
| `GET /api/me` | the caller: `user`, `operator`, `title` |
| `GET /api/jobs` | the jobs with schedule, next run, running flag and the last result (without its report) |
| `POST /api/jobs/{name}/run` | queues the job (operators); `409` when it is already running or queued |
| `GET /api/jobs/{name}/results` | the kept results, newest first |
| `GET /api/jobs/{name}/results/{id}` | one result with its full report (the drift, verify or comply report as JSON) |
| `GET /api/hosts` | every host seen by a job's last result with its worst status (`ok`, `changed`, `drift`, `failed`, `unreachable`) and the status per job |
| `GET /api/runs[?host=&playbook=]` | the audit store's last 100 runs (every `apply`, not only the server's) |
| `GET /api/runs/{id}` | one audit record |
| `GET /metrics` | `onigirazu_serve_job_last_status{job,kind}` (0 ok, 1 drift/changed, 2 failed, 3 error), `onigirazu_serve_job_last_run_timestamp_seconds{job}`, `onigirazu_serve_host_status{job,host}` |
| `GET /healthz` | `ok` |

The page at `/` uses the same API: Jobs (with a Run button for operators), Hosts and Runs tabs,
each result and audit record openable as JSON.

A job result: `{id, job, kind, started, duration_seconds, status, summary, trigger, hosts,
report}` where `trigger` is `schedule` or `api:<user>` and `status` is `ok`, `drift`, `failed`
or `error`. `metrics.push` sends each drift job's drift metrics and each comply job's
compliance metrics as `drift --metrics-push` and `comply --metrics-push` do.
