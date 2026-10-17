# Fleet server

`onigirazu serve` is a small server in the same binary: jobs run on a schedule or on demand
(drift checks, applies, `verify`, compliance profiles), their results are kept, and a web page
and a REST API show them per job and per host together with the run history of the audit
store. One process, one YAML file, no database.

```yaml
# serve.yml
listen: ":8086"
title: RED fleet
url: https://fleet.example.com
notify:
  - url: https://mattermost.example.com/hooks/${FLEET_HOOK}
    on: [pending, failure]
data_dir: /var/lib/onigirazu/serve     # results and job state; default ~/.onigirazu/serve
approval_timeout: 1h                   # default wait for approval
concurrency: 2                         # max jobs executing at once; default 1
retain: 50                             # results kept per job
auth:
  token: ${SERVE_TOKEN}                # API bearer token: an operator
  tokens:
    - name: ci
      token: ${CI_TOKEN}
      groups: [tier2]
  user_header: X-Authentik-Username    # a reverse proxy's identity: a viewer ...
  groups_header: X-Authentik-Groups    # ... an operator when one of the groups is listed
  operators: [admins, tier0-admins]
metrics:
  push: http://victoria:8428/api/v1/import/prometheus   # every job's metrics, optional
  labels: {env: prod}
jobs:
  - name: drift-site
    kind: drift                        # drift (default), apply, verify, comply
    playbook: site.yml
    inventory: hosts.yml
    every: 30m                         # 0 / absent: on demand only
    notify:
      - url: https://mattermost.example.com/hooks/${DRIFT_HOOK}
        on: [pending, drift, failure, cancelled]
      - url: https://events.example.com/${EVENTS_HOOK}
        format: json
  - name: ssh-compliance
    kind: comply
    profile: ssh                       # bundled or a file
    inventory: hosts.yml
    every: 6h
    fail_on: high
  - name: patch-wave
    kind: apply
    playbook: patch.yml
    adhoc: {limit: true, extra_vars: [reboot]}
    operators: [tier0-admins, denys]
    approval: required
    approval_timeout: 2h
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

Up to `concurrency` jobs run at a time (default 1), picked up in FIFO order as they
are due or requested. A job that is pending, queued or running is not requested again,
so a job never overlaps itself. Pause stops new runs from starting while executing runs
finish; resume drains the queue. Server shutdown cancels executing runs and records them
as `error` with summary `interrupted: serve stopping`.

Paths in the file are relative to it. Every job's state file lives in
`data_dir/state/<job>.state`. A job's inventory and state file are passed to the run as
its own arguments, not through the process's global `-i`/`--state`, so they never leak into
another job.

## Guarded runs

A job with `approval: required` creates a pending run when requested, including by the
schedule. An operator approves it to queue execution, or rejects it with an optional comment.
A pending run expires after the job's `approval_timeout`, or the server default (1h).
The flow is requested → pending → approved/rejected/expired; approved or unguarded runs go
through queued → running → result, with cancellation possible while queued or running.
Each job has at most one pending run; the schedule skips it while one waits. The record keeps `approval.by`, `approval.at`
and `approval.comment`. A restart keeps pending runs and marks queued or executing runs as
`error` with `interrupted: serve restarted`; `cancelling` runs become `cancelled` with the same summary.

Limited runs: `POST /api/jobs/{name}/run` accepts an optional JSON body
`{"limit":"web1","extra_vars":{"reboot":"true"}}`. The job must allow `limit` with
`adhoc.limit: true` and list each caller-set variable in `adhoc.extra_vars`. Unfilled
parameters keep the job's values; caller variables override its defaults. The limit is
resolved against the job's inventory (or the global inventory); no matching host or any
host outside the job's own `limit` returns `400`, as does a pattern with characters a host
pattern cannot have (letters, digits and `. _ - * : , ! & ~ [ ] @` are allowed). Unknown
fields and invalid bodies also return `400`. The record keeps `params.limit`, `params.extra_vars` and sorted
`params.hosts`; the page and messages show the parameters and resolved hosts. The run
executes on those host names, not on the pattern, and checks them against the job's limit
again when it starts: an inventory that changed while the run waited for approval cannot
widen it (the run ends as `error` instead). Variables reach the playbook as one JSON `-e`
value, so a value with spaces stays one variable.
For an approval-required patch job this requests a safe reboot of web1:

```bash
curl -X POST -H 'Authorization: Bearer ...' -H 'Content-Type: application/json' \
  localhost:8086/api/jobs/patch-wave/run -d '{"limit":"web1","extra_vars":{"reboot":"true"}}'
```

Notifications: server-wide `notify:` targets and a job's targets add up. Targets accept a
plain URL or a mapping with `url`, `format` (`mattermost`, the default, or `json`) and `on`.
Webhook URLs expand `${VAR}` from the environment. Events are `pending`, `start`, `success`,
`drift`, `failure` (failed or error), `rejected`, `expired` and `cancelled`. Without `on`, targets receive
`pending`, `drift` and `failure`; `notify_always: true` on the job adds `success` and `cancelled` to targets
without `on`. An explicit `on: []` sends nothing.

Mattermost messages include a headline, a colored attachment with the job kind, target,
hosts, trigger, duration and, for pending runs, expiry. For example:
`:white_check_mark: **patch-wave** ok — 12 in sync, 0 changed, 0 failed`; cancellation reads
`:no_entry: **patch-wave** cancelled by alice — cancelled by alice: maintenance`.
Pending/start messages show resolved ad-hoc hosts (up to 10, then `… (+N)`), else
the job's limit (or "all hosts"); caller variables appear as `vars: k=v`; final messages group host
statuses, for example `3 ok, 1 drift, 1 failed`. Generic JSON has the shape
`{"event":"success","job":{"name":"patch-wave","kind":"apply","target":"patch.yml","limit":"web"},"run":{...},"url":"https://fleet.example.com/#run/<id>","text":"patch-wave ok — ..."}`;
`run` is the job result without `report`, and `text` is the headline without formatting.
Set the server's `url` to include a link to `#run/<id>`; without it, JSON `url` is empty
and Mattermost has no link. Delivery is asynchronous with a 10s timeout. Failures are logged
with the webhook URL redacted and never fail the run.

Pause: operators pause execution with a reason from the page or `POST /api/pause`, and
resume with `POST /api/resume`. No new run starts while paused; an executing run finishes.
The state is persisted in `data_dir/pause.json` (written whole, then renamed). When the
file cannot be written the new state still holds in this process and the API answers `500`
with the error: fix `data_dir` before trusting a restart. A `pause.json` that is there but
unreadable pauses the server at start (fail closed) until an operator resumes it. The
schedule and approvals keep working, runs queue up in order, and pending runs still expire.
`onigirazu_serve_paused` is 1 while paused, 0 otherwise.

Cancel: an operator of the job uses Cancel on the page or `POST /api/runs/{id}/cancel`
with an optional `{comment: "..."}`. A queued run becomes `cancelled` without starting and
frees the job immediately; a running run becomes `cancelling` until its context stops, then
`cancelled`. The record keeps `cancel.by`, `cancel.at`, optional `cancel.comment`, elapsed duration
and any partial hosts/report; partial hosts do not count as host verdicts. The schedule can
request the job again. Pending runs must be rejected instead.

## Access

Without `auth` everyone is an operator — put the server behind a reverse proxy or a firewall.
With `token`, `Authorization: Bearer <token>` is an operator (automation, curl). With
`user_header`, the user the proxy sets (Authentik's outpost, oauth2-proxy, nginx auth_request) is
a viewer; when one of the groups in `groups_header` (comma, pipe or space separated) is in
`operators`, an operator. Viewers read everything; operators also start jobs, approve or reject pending runs and cancel queued or running ones.

Named bearer tokens in `auth.tokens` have a `name`, a `token` value (expands `${VAR}`),
and optional `groups`. Each is a server-level operator. Names must be unique and values
must be nonempty. Tokens are checked before proxy headers.

A job's optional `operators` list narrows run, approve, reject and cancel access: the caller must
be a server-level operator and match a listed user name or group, case-insensitively.
Without the list, every server-level operator is allowed. The legacy `auth.token` has the
user name `token`; include `token` in a job's list to admit it. Pause and resume require
only a server-level operator.

Every run, approve, reject, cancel, pause and resume action and refused attempt is recorded in
`data_dir/audit.log` as JSON lines, with identity, job, run ID, result, comment or reason,
and client address. Unauthenticated probes are omitted. Viewers can read the entries,
newest first, through `GET /api/audit` or the Audit tab. The file is append-only and
reopened for every entry, so logrotate can move it aside (no `copytruncate` needed).

## API

| Method and path | Returns |
|---|---|
| `GET /api/audit?limit=N` | action log, newest first; default 100 entries, max 1000 (viewers too) |
| `GET /api/me` | the caller: `user`, `operator`, `title`, `paused` (bool), `running` and `queued` counts, `concurrency` |
| `GET /api/pause` | pause state: `paused`, optional `reason`, `by`, `at` (viewers too) |
| `POST /api/pause` | pauses execution (operators); optional JSON `{reason: "..."}`; `200` with the state; pausing again updates reason and identity; `500` with the state and `error` when `pause.json` could not be written (the pause still holds in this process) |
| `POST /api/resume` | resumes execution (operators); `200` with the state; already resumed is a no-op; `500` as for pause |
| `GET /api/jobs` | the jobs with schedule, next run, `running` flag (queued, running or cancelling), optional `active` record (without report), `queued` flag (waiting for the worker), last executed result, optional `pending` record (without reports), and `approval: "required"` for guarded jobs; optional `operators` list and caller-specific `allowed` bool; `adhoc: {limit, extra_vars}` when enabled |
| `POST /api/jobs/{name}/run` | requests the job (operators); optional JSON `{limit: "host pattern", extra_vars: {k: "v"}}` as allowed by `adhoc`; `400` for invalid parameters or a limit with no hosts or hosts outside the job's limit; `202` with `{queued: name, run: id}` or `{pending: id, run: id}`; `409` when running, queued or pending; `503` when the queue is full; `403` with `{"error":"not an operator of this job"}` when the job disallows the operator |
| `GET /api/jobs/{name}/results` | the kept results, newest first |
| `GET /api/jobs/{name}/results/{id}` | one result with its full report (the drift, verify or comply report as JSON) |
| `GET /api/results/{id}` | one job result with its full report and optional `params` (`limit`, `extra_vars`, sorted `hosts`); `404` if absent |
| `POST /api/runs/{id}/approve` | queues a pending run (operators); optional JSON `{comment: "..."}`; `200` with the record without report, `404` if absent, `409` when the run expired meanwhile (it is marked expired), `503` if the queue is full; `403` with `{"error":"not an operator of this job"}` when the job disallows the operator |
| `POST /api/runs/{id}/reject` | rejects a pending run (operators); optional JSON `{comment: "..."}`; `200` with the record without report, `404` if absent; `403` with `{"error":"not an operator of this job"}` when the job disallows the operator |
| `POST /api/runs/{id}/cancel` | cancels a queued or running run (job operators); optional JSON `{comment: "..."}`; `200` with the cancelled record for queued runs, `202` with the cancelling record for running runs (without report); `404` if absent, `409` if not cancellable (reject a pending run instead), `403` if disallowed |
| `GET /api/hosts` | every host seen by a job's last executed result (cancelled runs are skipped), with its worst status (`ok`, `changed`, `drift`, `failed`, `unreachable`) and the status per job |
| `GET /api/runs[?host=&playbook=]` | the audit store's last 100 runs (every `apply`, not only the server's) |
| `GET /api/runs/{id}` | one audit record |
| `GET /metrics` | `onigirazu_serve_job_last_status{job,kind}` (0 ok, 1 drift/changed, 2 failed, 3 error, 4 cancelled), `onigirazu_serve_job_last_run_timestamp_seconds{job}`, `onigirazu_serve_host_status{job,host}`, `onigirazu_serve_job_pending{job}` (guarded jobs only: 1 pending, 0 otherwise); `onigirazu_serve_paused` (1 paused, 0 otherwise); `onigirazu_serve_running` (executing or cancelling runs), `onigirazu_serve_concurrency` (configured limit); all include configured metric labels |
| `GET /healthz` | `ok` |

Run, approve, reject and cancel return `403` with `{"error":"not an operator of this job"}` when
a server-level operator does not match the job's list. Viewers receive `403` with
`{"error":"operators only"}`; missing identity receives `401`. An unknown pending run
returns `404` before the per-job check.

The page at `/` uses the same API: Jobs (with Run/Approve/Reject/Cancel buttons when `allowed`), Hosts, Runs and Audit tabs,
each result and audit record openable as JSON.

A job result: `{id, job, kind, requested, started, duration_seconds, status, summary, trigger,
hosts, report, approval, cancel, params}`. `requested` is the request time; `started` becomes the execution
start time when the worker begins. `trigger` is `schedule` or `api:<user>`. Status is `pending`,
`queued`, `running`, `cancelling`, `cancelled`, `rejected`, `expired`, `ok`, `drift`, `changed`, `failed` or `error`.
The optional approval object has `required`, `expires`, and, after a decision, `decision`
(`approved`, `rejected`, `expired`), `by` (operator identity; absent for expiry), `at` and an
optional `comment`. Job results open at `#job/<name>` and individual runs at `#run/<id>`.

`metrics.push` sends each drift job's drift metrics and each comply job's compliance metrics
as `drift --metrics-push` and `comply --metrics-push` do.
