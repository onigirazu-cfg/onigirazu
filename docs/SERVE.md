# Fleet server

`onigirazu serve` is a small server in the same binary: jobs run on a schedule or on demand
(drift checks, applies, `verify`, compliance profiles), up to `concurrency` of them at a time,
their results are kept, and a web page and a REST API show them per job and per host together
with the run history of the audit store. Guarded jobs wait for an operator's approval, ad-hoc
runs take a host limit and variables (secret ones never touch the disk). Runs can be requested
for later, apply jobs can stay within maintenance windows, and scheduled, queued or running
runs can be cancelled. Execution can be paused. One process, one YAML file, no database.

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
run_timeout: 0s                        # default run limit; 0: no limit, negative: invalid
shutdown_timeout: 10m                  # how long a stop waits for executing runs; 0: cancel at once
concurrency: 2                         # max jobs executing at once; default 1
# missed: skip                        # optional default for all jobs; job setting wins
retain: 50                             # results kept per job
retain_age: 168h                        # optional maximum result age (Go duration)
auth:
  tokens:
    - name: ci
      token: sha256:3f96ac8166d16e1cc7f6cbe569d04a399edbbf39ea622448c685d633733a4cba
      role: operator
      jobs: [drift-site]
      groups: [tier2]
      expires: 2027-01-21              # generated with serve token new
  user_header: X-Authentik-Username    # a reverse proxy's identity: a viewer ...
  groups_header: X-Authentik-Groups    # ... an operator when one of the groups is listed
  operators: [admins, tier0-admins]
  trusted_proxies: [192.0.2.5, 192.0.2.0/24]   # proxy headers only from these (address or network)
  sign_out_url: /outpost.goauthentik.io/sign_out   # optional "Sign out" in the page's user menu
ui:
  pages_dir: pages                     # optional static pages; relative to serve.yml
  links:
    - {title: Map, href: /pages/map.html} # extra sidebar entry
metrics:
  push: https://metrics.example.com/api/v1/import/prometheus   # every job's metrics, optional
  labels: {env: prod}
jobs:
  - name: drift-site
    kind: drift                        # drift (default), apply, verify, comply
    playbook: site.yml
    inventory: hosts.yml
    every: 30m                         # absent with no at: on demand only
    # at: ["mon,thu 03:30", "sat 12:00"] # use instead of every
    # at_tz: Europe/Madrid              # default UTC; only with at
    # missed: run_once                  # run_once or skip; overrides server default
    timeout: 10m                       # run limit; 0 / absent: server run_timeout; negative: invalid
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
    adhoc:
      limit: true
      extra_vars:
        - {name: reboot, type: bool, default: false, description: Reboot after patching}
        - {name: batch_size, type: int, min: 1, max: 20, default: 5}
        - {name: wave, type: choice, choices: [blue, green], required: true}
        - {name: ticket, pattern: 'OPS-[0-9]+', required: true}
        - {name: token, secret: true}
    operators: [tier0-admins, denys]
    approval: required
    check_approval: required           # optional: also guard check runs
    approval_timeout: 2h
    windows: ["mon-fri 23:00-02:00", "sat,sun 01:00-04:00"]
    windows_tz: Europe/Madrid          # default UTC
  - name: deploy-web
    kind: apply
    playbook: deploy.yml
    inventory: hosts.yml
    limit: web
    become: true
```

Variable declarations in `adhoc.extra_vars` accept a plain name (an optional string)
or a mapping with `name` and optional `secret`, `type`, `choices`, `default`, `pattern`,
`min`, `max`, `description` and `required`. `type` defaults to `string`; supported types
are `string`, `int`, `bool` and `choice`. `choices` must be nonempty for `choice` and
cannot be used with other types. Integer bounds `min` and `max` are inclusive and
require `min <= max`. `pattern` is an RE2 expression that must match the whole value.
Secret variables support only `string` and `int`. Invalid types, patterns, bounds,
choices and defaults reject the configuration at startup.

The server validates every supplied value: integers must parse as signed 64-bit integers
within bounds, booleans must be exactly `true` or `false`, choices must be declared,
and patterns must match. API values remain strings, including integers and booleans.
A missing required variable returns `400` unless it has a declared default. Defaults
are applied server-side only for omitted variables, including requests without a body,
and recorded in `params.extra_vars`. Validation errors identify the variable and rule
without exposing secret values. The page shows selects, checkboxes, number inputs,
text or password fields, with defaults, descriptions and required markers; a pattern is shown
as a hint and checked only by the server (RE2), so the browser's regex dialect never blocks a value. Secret
defaults are never sent to the page; leaving a secret field empty uses its server default.

```bash
onigirazu serve --config-file serve.yml
```

Up to `concurrency` jobs run at a time (default 1), picked up in FIFO order as they
are due or requested. A job that is pending, queued or running is not requested again,
so a job never overlaps itself. Pause stops new runs from starting while executing runs
finish; resume drains the queue.

On SIGTERM or SIGINT serve stops gracefully. It refuses new runs and approvals with `503`,
starts nothing from the queue or the schedule, and waits up to `shutdown_timeout`
(default 10m) for executing runs to finish. The page and API keep answering meanwhile, and
`/api/me` carries `stopping: {since, deadline}`, which the page shows as a banner. Runs still
going at the deadline, or when a second signal arrives, are cancelled and recorded as `error`
with summary `interrupted: serve stopping`. `shutdown_timeout: 0` cancels them at once.
Queued, scheduled and pending runs stay on disk and continue after the next start.

Run time limits start when execution starts, excluding time scheduled, queued or pending approval.
An unfinished run that reaches its limit becomes `timed_out` with summary
`timed out after <duration>` and keeps any partial hosts and report (drift, apply, verify and compliance
jobs); partial hosts do not count as host verdicts. Operator cancellation
and shutdown retain their usual outcomes; a completed run keeps its result.

Paths in the file are relative to it. Every job's state file lives in
`data_dir/state/<job>.state`. A job's inventory and state file are passed to the run as
its own arguments, not through the process's global `-i`/`--state`, so they never leak into
another job.

Results are written to `data_dir/results/<id>.json.tmp`, then renamed to `<id>.json`,
so readers see a complete record. A failed write keeps the record in memory; run and
approval requests return `500` with the record and `error`. Workers log the failure.
`GET /api/me` exposes `storage_error` (empty when fine), the page shows it in red below
the header, and `onigirazu_serve_storage_error` is 1 while set, 0 otherwise. A later
successful result write clears the error.

Retention applies at startup and when a result is stored. `retain` keeps the newest
50 results per job by default: with `every: 30m`, that is about a day. Rejected and expired
requests consume slots too. Optional `retain_age` also removes finished results older
than the duration, measured from their request time. Go durations use hours, such as
`168h` for seven days. Pending, scheduled, queued and running results are never pruned by age;
active results are also exempt from the count limit.

## Install as a service

`serve install` writes `/etc/systemd/system/onigirazu-serve.service`, reloads systemd,
and enables and starts the service. The unit retains the global `-c/--config` file
for in-process playbook runs and uses absolute paths. Review it first with `--print`;
this validates serve.yml and prints the unit without creating directories, writing
files or calling systemd.

```bash
sudo onigirazu -c /srv/app/onigirazu.yml serve install -f /srv/app/serve.yml \
  --user app --environment-file /etc/onigirazu/serve.env \
  --env PATH=/srv/app/.venv/bin:/usr/local/bin:/usr/bin:/bin \
  --addr 127.0.0.1:8086 --print
# Remove --print to install and start the service.
```

`--user NAME` selects the service account; its primary group is used unless
`--group NAME` is given. Without `--user`, the service runs as root.
`--working-directory DIR` defaults to the directory containing serve.yml.
`--environment-file FILE` loads service variables, and repeatable `--env KEY=VALUE`
adds variables; matching keys in the environment file take precedence, as in
systemd. The installer applies both when validating `${VAR}` in serve.yml.
Environment files follow systemd's rules with one assignment per line (no
continuation lines or multi-line values): `KEY=VALUE`, single-quoted values taken
literally, double-quoted values with `\"`, `\\`, `` \` `` and `\$` escapes, blank lines
and comments starting with `#` or `;`; omit shell `export` prefixes.
The working directory and environment file must already exist.

The unit waits for the network and restarts on failure. It sets
`NoNewPrivileges=yes`, `ProtectSystem=strict` and `PrivateTmp=yes`.
For the graceful stop it sets `KillMode=mixed` (SIGTERM reaches serve only, so commands a run
started keep working) and `TimeoutStopSec` to `shutdown_timeout` plus one minute; re-run
install after changing `shutdown_timeout`. A hand-written unit needs the same two lines.
`systemctl stop` or `restart` therefore waits until the running runs finish. To cancel them
now, send the second signal with `systemctl kill --kill-whom=main -s SIGTERM onigirazu-serve`.
`ReadWritePaths` allows writes to `data_dir`, the working directory and the service
user's home when it exists or install creates it for `data_dir` (for the audit store
and facts cache). Add repeatable
`--read-write-path DIR` for any other writable location. When serve.yml omits
`data_dir`, its default is the service user's `~/.onigirazu/serve`; installation
creates the data directory and assigns it to the selected user and group.

## Scheduling

`every: 30m` requests a job at a fixed interval. `at` anchors requests to local days
and times instead: `at: "tue 18:00"`, `at: ["mon,thu 03:30", "sat 12:00"]` or
`at: "* 04:00"`. Each entry is `"<days> HH:MM"`; days are `mon` through `sun`,
comma-separated lists, inclusive ranges (including wrapping ranges such as `fri-mon`),
or `*` for every day. `every` and `at` are exclusive. `at_tz` is an IANA time zone,
defaulting to `UTC`; it requires `at`, and unknown zones reject the configuration.
For daylight-saving transitions, candidates use Go's `time.Date`: nonexistent local
times are normalized (for example, 02:30 becomes 03:30 at the spring transition in
Europe/Madrid), and repeated local times use the first occurrence.

The server keeps each job's schedule signature, last planned slot, next slot, planning
reason, missed slot (when known) and last request outcome in `data_dir/schedule.json`.
Writes use a temporary file and atomic rename with mode `0600`. A restart restores the
next run from this record instead of requesting every scheduled job at once. Interval
schedules retain their phase rather than moving with each scheduler tick. Changing a
schedule recomputes its next slot, using the last planned slot when available.

`missed: skip` skips slots that passed while the server was down or the clock jumped
forward. `missed: run_once` requests one catch-up run, however many slots were missed,
then resumes the normal schedule. A job's setting overrides top-level `missed:`. With
neither set, `apply` defaults to `skip` to avoid unexpected changes after an upgrade or
restart; read-only `drift`, `verify` and `comply` jobs default to `run_once`. Other
values reject the configuration. A policy on an on-demand job has no effect.

On first start, a new `run_once` job requests its first run after a short stagger
(10 seconds plus 5 seconds per job); a new `skip` job waits for its first future slot.
When upgrading from a version without `schedule.json`, the most recent scheduled
request in retained results supplies the last slot. A weekly apply last requested two
days ago therefore waits for the remaining five days. If neither state nor scheduled
history exists, the job follows the first-start rule.

A forward clock jump never causes a burst of requests: missed slots follow the policy
above, with at most one request per job per tick. A backward jump caps the wait at one
interval, or the next anchored occurrence. An unreadable `schedule.json` is logged and
renamed to `schedule.json.corrupt-<UTC timestamp>` for inspection. The server writes a
fresh plan and waits for future slots; nothing runs immediately in this case. Schedule
write failures appear through the existing storage-error reporting.

Catch-up requests still follow approval and maintenance-window rules. A slot is consumed
even when its request is refused (for example, the job already has a pending or queued run);
`last_outcome` then says why. Pausing execution keeps
scheduled requests working as before; workers wait until execution resumes.

`GET /api/jobs` exposes `every`, `at`, `at_tz` and the effective `missed` policy, plus
`next`, `next_reason`, `missed_at`, `last_scheduled` and `last_outcome`. Reasons are
`scheduled`, `missed_skipped`, `missed_run_once`, `state_unreadable` and `new`.
`last_scheduled` is the planned slot rather than the tick's wall time; `last_outcome`
records `requested <run id>` (with pending or scheduled status when applicable) or
`skipped: <error>`. The page shows the next slot and explains skipped or caught-up slots.

## Run later and maintenance windows

Operators can schedule any job kind with `POST /api/jobs/{name}/run` and a body such as
`{"not_before":"2026-10-24T23:00:00+02:00"}`. The RFC 3339 time must be in the future
and at most 30 days ahead; invalid, past or too distant times return `400`. It can be
combined with `mode`, `limit` and `extra_vars`. The record keeps `not_before` and has
status `scheduled`. A guarded run waits as `pending` first; approval schedules it if
its time is still ahead, otherwise it can queue immediately.

Only one future request per job is accepted. A second request with `not_before` returns
`409 {"error":"a run is already scheduled for this job"}`; the periodic scheduler likewise keeps
a single deferral outside a window instead of adding one per tick. Scheduled runs do not hold
the job slot: manual runs and periodic schedules continue. Every five seconds, due runs
queue when the job is free; busy jobs leave them scheduled with summary
`waiting for the job to be free` until a later tick. Cancel works on scheduled runs.
If an immediate run misses its window while another run is already scheduled, both
are preserved as scheduled runs. The same applies to a run approved after its window
closes. In this collision case, `/api/jobs` also returns `scheduled_runs` with all
deferred records; the page shows each one with its own Cancel button.
A restart re-arms deferred runs at their recorded times; deferred runs with secret parameters instead become
`error`, because the secret values were kept only in memory.

Apply jobs can set `windows` to a list of `"<days> HH:MM-HH:MM"` specifications.
Days are `mon` through `sun`, ranges such as `mon-fri`, comma-separated lists such as
`sat,sun`, or `*` for every day. An end earlier than the start crosses midnight, with
the day referring to the start. `windows_tz` selects an IANA time zone, default `UTC`;
invalid specifications or unknown zones fail configuration loading. For example,
`windows: ["mon-fri 23:00-02:00"]` with `windows_tz: Europe/Madrid` permits starts
from 23:00 on weekdays until 02:00 the following day, following local daylight saving time.

Windows gate apply runs, including `every` triggers and approved runs. Check-mode
runs and other job kinds ignore them. Outside a window, the run becomes `scheduled`
with `not_before` set to the next window start. The worker checks again at pickup:
a queued run that missed the window returns to `scheduled`. An executing run can
finish after the window ends. The next start is searched up to 15 days ahead (a weekly
window skipped by a DST change is found the week after); without one the run stays
`scheduled` and is checked again a day later, never started outside its windows. Jobs
expose `windows`, `windows_tz` and, while no window is open, `next_window` (the next start)
in the API, and the page shows their windows.

The page offers Run at… next to Run for every job, and a Run at field in the ad-hoc
form. These use local date/time inputs and send RFC 3339 with the local UTC offset.
The jobs table shows a scheduled run's time and relative wait, links to its record,
and offers Cancel. The header counts scheduled runs alongside running and queued runs.

## Guarded runs

A job with `approval: required` creates a pending run when requested, including by the
schedule. An operator approves it to queue execution, or rejects it with an optional comment.
A pending run expires after the job's `approval_timeout`, or the server default (1h).
The flow is requested → pending → approved/rejected/expired; approved or unguarded runs go
through optional scheduled → queued → running → result, with cancellation possible while
scheduled, queued or running.
Each job has at most one pending run; the schedule skips it while one waits. The record keeps `approval.by`, `approval.at`
and `approval.comment`. A restart keeps pending, scheduled and queued runs (queued ones queue again in request
order) and marks runs that were executing as `error` with `interrupted: serve restarted`; `cancelling` runs become `cancelled` with the same summary.

Apply jobs accept `{"mode":"check"}` at `POST /api/jobs/{name}/run` to preview changes
without applying them. A check run skips the plays' `verify:` checks, which would test the
state before the changes (drift jobs still run them). `mode: "apply"` or an empty mode keeps normal behavior; other
values return `400`. Check mode is only valid for apply jobs (other kinds return
`{"error":"only apply jobs have a check mode"}`). It uses the same job operators and
busy rule as Run and needs no `adhoc` setting. A check can also include allowed
`limit` and `extra_vars`. Checks bypass `approval: required` unless the job sets
`check_approval: required` (the only accepted values are `required` or empty). Each
check is a separate run and never approves or unlocks an apply run. Its record keeps
`mode: "check"` across restarts; normal runs omit `mode`.

The ad-hoc form lists clickable groups and hosts within the job's limit and previews the hosts matched by an editable limit pattern before Run.

Limited runs: `POST /api/jobs/{name}/run` accepts an optional JSON body
`{"limit":"web1","extra_vars":{"reboot":"true"}}`. The job must allow `limit` with
`adhoc.limit: true` and list each caller-set variable in `adhoc.extra_vars`. Unfilled
parameters use declared defaults when present, otherwise keep the job's values; caller variables override its defaults. The limit is
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

Secret parameters: declare an allowed variable as `{name: token, secret: true}` in
`adhoc.extra_vars`; plain names still work. The page renders secret fields as password
inputs. Secret values appear as `***` in `params.extra_vars`, result files and endpoints,
the action log reason, notifications and the page. The record keeps only their names
in optional `params.secret_vars`, so restart detection still works if the configuration
changes. The actual values stay in memory until the run ends, is rejected, expires or is cancelled, and are merged into the job's
variables only for execution. They are not kept across restarts: a pending, scheduled or queued
run with secret parameters becomes `error` with summary
`interrupted: serve restarted (secret parameters are not kept)`.
Masking covers the serve records, reports, action log and notifications. Playbook-generated
task output still uses the execution engine's existing logging, audit and cache handling.

Notifications: server-wide `notify:` targets and a job's targets add up. Targets accept a
plain URL or a mapping with `url`, `format` (`mattermost`, the default, or `json`) and `on`.
Webhook URLs expand `${VAR}` and `$VAR` from the environment. A URL that expands to
empty (after trimming whitespace) disables that target with a startup warning naming
the job (or server-wide target) and each unset or empty variable, without printing values.
This lets `notify:` stay in `serve.yml` before the webhook exists. A literally empty URL
or a partly expanded URL with an unset variable is an error. Format and event validation
still applies to disabled targets. Events are `pending`, `start`, `success`,
`drift`, `failure` (failed, error or timed_out), `rejected`, `expired`, `cancelled` and `scheduled`. Without `on`, targets receive
`pending`, `drift` and `failure`; `notify_always: true` on the job adds `success` and `cancelled` to targets
without `on`. An explicit `on: []` sends nothing.

Mattermost messages include a headline, a colored attachment with the job kind, target,
hosts, trigger, duration, scheduled `not_before` and, for pending runs, expiry. The `scheduled`
event is opt-in with `on: [scheduled]`, like `cancelled`. For example:
`:white_check_mark: **patch-wave** ok — 12 in sync, 0 changed, 0 failed`; cancellation reads
`:no_entry: **patch-wave** cancelled by alice — cancelled by alice: maintenance`.
Timeouts use the `failure` event with headline `patch-wave timed out`.
Pending/start messages show resolved ad-hoc hosts (up to 10, then `… (+N)`), else
the job's limit (or "all hosts"); caller variables appear as `vars: k=v`; final messages group host
statuses, for example `3 ok, 1 drift, 1 failed`. Generic JSON has the shape
`{"event":"success","job":{"name":"patch-wave","kind":"apply","target":"patch.yml","limit":"web"},"run":{...},"url":"https://fleet.example.com/#run/<id>","text":"patch-wave ok — ..."}`;
`run` is the job result without `report`, including `mode: "check"` for checks,
and `text` is the headline without formatting. Check headlines add ` (check)` after
the job name and use the same events.
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

Rerun failed hosts: the run page offers **Rerun failed hosts (N)** to job operators
after an executed run with `failed` or `unreachable` hosts. `POST /api/runs/{id}/rerun-failed`
creates a run on just those hosts, retaining the mode and non-secret caller variables.
The hosts are validated against the current job limit, even when `adhoc.limit` is off, and the
caller variables against the job's current declarations (a value that no longer fits returns `400`).
Approval, maintenance windows and busy rules still apply. The new record has
`parent: "<id>"` and `trigger: "rerun:<user>"`; the page links back to its parent.
An unexecuted run or one without failed hosts returns `400`. A parent with secret
parameters returns `400` with `{"error":"the run had secret parameters; start it again from the form"}`.
The audit action is `rerun_failed`, with the parent ID in its reason.

Cancel: an operator of the job uses Cancel on the page or `POST /api/runs/{id}/cancel`
with an optional `{comment: "..."}`. A scheduled or queued run becomes `cancelled` without starting and
frees the job immediately; a running run becomes `cancelling` until its context stops, then
`cancelled`. The record keeps `cancel.by`, `cancel.at`, optional `cancel.comment`, elapsed duration
and any partial hosts/report (drift, apply, verify and compliance jobs); partial hosts do not count as host verdicts. The schedule can
request the job again. Pending runs must be rejected instead.

## Access

Without `auth` everyone is an operator — put the server behind a reverse proxy or a firewall.
Bearer tokens in `auth.tokens` accept `Authorization: Bearer <token>` (automation, curl).
Their `role` is `operator` by default, or `viewer` for read access. The legacy `auth.token`
is loaded as a token named `token`; that name cannot also appear in `auth.tokens`. With
`user_header`, the user the proxy sets (Authentik's outpost, oauth2-proxy, nginx auth_request) is
a viewer; when one of the groups in `groups_header` (separated by `|` as Authentik sends them, or by
commas; a space belongs to the name, "Fleet Operators") is in `operators`, an operator.

The proxy headers are only as trustworthy as the path to the server: anyone who reaches `serve`
directly can send `X-Authentik-Groups: admins`. Set `trusted_proxies` to the proxy's address or
network and headers from any other peer are ignored (the request is then unauthenticated, `401`
unless it carries a bearer token); without it the server warns at start. Behind Authentik's proxy
outpost: `user_header: X-Authentik-Username`, `groups_header: X-Authentik-Groups`, `operators` =
the Authentik groups that may run jobs, `trusted_proxies` = the outpost's address, and
`sign_out_url: /outpost.goauthentik.io/sign_out` for a Sign out entry in the user menu
(`/api/me` returns it as `sign_out_url`; only `/…` or `http(s)://` links are shown). Viewers read everything; operators also start jobs, approve or reject pending runs and cancel scheduled, queued or running ones.

### Browser sign-in

When `/api/me` returns 401, the main page shows a sign-in screen with a Token password
field and “Remember on this device”. Config and generated tokens both work. The user
menu shows the name, role, authentication kind and token expiry, when set. Token users
sign out there; proxy users keep the configured `sign_out_url` link. Open servers show
neither sign-out action. Old sessionStorage tokens are migrated once and removed.

`POST /api/session` accepts `{"token":"…","remember":false}` and returns the same
identity and state as `GET /api/me`. Unknown or expired tokens return 401. Successful
sign-in sets `onigirazu_session` to the token value with `Path=/`, `HttpOnly` and
`SameSite=Strict`. It adds `Secure` on TLS or when a trusted proxy sends
`X-Forwarded-Proto: https` (the same `auth.trusted_proxies` rules apply as for proxy
identity headers). By default the cookie has no Max-Age or Expires. Remember sets
Max-Age to 30 days, capped at the token's expiry. Revoking or expiring the token ends
access immediately; restarting the server preserves sessions.

`DELETE /api/session` clears the cookie and returns 204. Sign-in successes and failures
are audited as `sign_in`, with the client address and no token value. Resolved sign-outs
are audited as `sign_out`. After 10 failed sign-ins in a minute per audit client address,
further attempts from that client return 429 without checking the token, until failures
age out. The limit is held in memory per client: the peer address, or, when the peer is
listed in `auth.trusted_proxies`, the last `X-Forwarded-For` entry it added.

Authentication checks the Authorization header, then the session cookie, then proxy
headers. Cookie-authenticated requests other than GET, HEAD and OPTIONS must show they
come from the page itself: they return 403 `{"error":"cross-site request"}` when they
carry neither `Origin` nor `Sec-Fetch-Site`, when `Sec-Fetch-Site` is not `same-origin`,
or when `Origin` has a different host from the request. Sign-in and sign-out refuse only
the cross-site cases. Bearer-header requests are exempt from this check.

### Behind Authentik

Authentik signs people in and passes who they are in headers. The token sign-in screen
also works when proxy authentication is configured.

```yaml
listen: "192.0.2.20:8086"           # an internal address only the proxy reaches
auth:
  user_header: X-Authentik-Username
  groups_header: X-Authentik-Groups
  operators: [fleet-operators]      # Authentik groups that may run, approve and cancel
  trusted_proxies: [192.0.2.5]     # the outpost or the reverse proxy (see below)
  sign_out_url: /outpost.goauthentik.io/sign_out
  tokens:
    - name: ci
      token: ${SERVE_CI_TOKEN}      # automation, not through Authentik
```

In Authentik:

1. Create a Proxy Provider. "Proxy" mode: the outpost itself forwards to `http://192.0.2.20:8086`.
   "Forward auth (single application)": nginx or Traefik fronts `serve` and asks the outpost.
2. Create an Application for it; bind the groups that may open the page at all (everyone else is
   stopped by Authentik). Who is an operator is decided by `operators` above.
3. Add the Application to an outpost (the embedded one is enough).

`trusted_proxies` is the address that connects to `serve`: the outpost in "Proxy" mode, the
reverse proxy (nginx, Traefik) in forward-auth mode, which copies the `X-Authentik-*` headers from
the auth response and must overwrite any the client sent (nginx `proxy_set_header` does). The
outpost's own paths, such as `/outpost.goauthentik.io/sign_out`, must reach the outpost.

Automation with a bearer token goes around Authentik: call `serve` on its own address, or give
`/api/` a route on the reverse proxy that skips Authentik (the token is still required there).

### Other identity proxies

Any reverse proxy that signs people in and passes who they are in headers works the same way: point
`user_header` and `groups_header` at its headers, `sign_out_url` at its sign-out path, and
`trusted_proxies` at its address. Groups may come as one header joined by `|` or `,`, or as several
header lines (all are read).

| Proxy | `user_header` | `groups_header` | `sign_out_url` |
|---|---|---|---|
| Authentik outpost | `X-Authentik-Username` | `X-Authentik-Groups` | `/outpost.goauthentik.io/sign_out` |
| oauth2-proxy (`--pass-user-headers`, proxy mode) | `X-Forwarded-Email` or `X-Forwarded-User` | `X-Forwarded-Groups` | `/oauth2/sign_out` |
| oauth2-proxy (`--set-xauthrequest`, nginx `auth_request`) | `X-Auth-Request-Email` | `X-Auth-Request-Groups` | `/oauth2/sign_out` |
| Authelia (forward auth) | `Remote-User` | `Remote-Groups` | the Authelia portal's `/logout` |
| Pomerium (`pass_identity_headers` + `jwt_claims_headers`) | `X-Pomerium-Claim-Email` | `X-Pomerium-Claim-Groups` | `/.pomerium/sign_out` |
| Cloudflare Access | `Cf-Access-Authenticated-User-Email` | none | `/cdn-cgi/access/logout` |
| Tailscale Serve | `Tailscale-User-Login` | none | none |

Without a groups header every signed-in user is a viewer; operators then use bearer tokens, or put a
proxy that passes groups in front. Behind Cloudflare Access only cloudflared should reach `serve`;
Tailscale Serve strips client-sent identity headers and expects the backend on localhost.

**oauth2-proxy** (Keycloak, Entra ID, Okta, GitLab, any OIDC provider; Google below). In proxy mode it
forwards to `serve` itself:

```sh
oauth2-proxy --provider=oidc --oidc-issuer-url=https://sso.example.com/realms/ops \
  --client-id=serve --client-secret-file=/run/secrets/serve-oidc --cookie-secret-file=/run/secrets/cookie \
  --email-domain=example.com --upstream=http://192.0.2.20:8086 --http-address=0.0.0.0:4180 \
  --pass-user-headers --oidc-groups-claim=groups
```

```yaml
auth:
  user_header: X-Forwarded-Email
  groups_header: X-Forwarded-Groups
  operators: [fleet-operators]
  trusted_proxies: [192.0.2.5]           # the oauth2-proxy host
  sign_out_url: /oauth2/sign_out
```

The OIDC provider must put the groups into the `groups` claim (Keycloak: a group membership mapper).
In nginx `auth_request` mode copy `X-Auth-Request-Email` and `X-Auth-Request-Groups` from the auth
response into the request to `serve`, and check the result with `GET /api/me` (user and operator).

**Google** goes through oauth2-proxy with `--provider=google`. The e-mail always arrives
(`X-Forwarded-Email`); groups only for Google Workspace: oauth2-proxy reads them from the Admin SDK with
a service account (`--google-service-account-json`, domain-wide delegation with the
`https://www.googleapis.com/auth/admin.directory.group.member.readonly` scope, `--google-admin-email`) and passes the user's groups,
limited to `--google-group` when set. Plain Gmail accounts have no groups, so they sign in as viewers.
Alternatively add Google as a sign-in source in Authentik and keep the groups there.

**Authelia** with Traefik forward auth: the middleware copies the identity headers to `serve`
(`authResponseHeaders: Remote-User,Remote-Groups,Remote-Email`), and `trusted_proxies` is Traefik:

```yaml
auth:
  user_header: Remote-User
  groups_header: Remote-Groups
  operators: [fleet-operators]
  trusted_proxies: [192.0.2.5]           # Traefik
  sign_out_url: https://auth.example.com/logout
```

### Tokens

Generate and save a token with `onigirazu serve token new -f serve.yml --name ci`.
The value is printed once. Keep it in your secret manager. Without `--name` the name is
made from the role, the job when there is one, and four random hex digits
(`viewer-drift-site-3f2a`); the same applies to an empty name in the API and the Access tab.
`serve token list -f serve.yml` lists tokens. `serve token revoke ci -f serve.yml`
revokes a generated token. Config tokens must be removed from `serve.yml`.
Use `--print` to print a value and hashed YAML entry without reading or writing files.

Generated tokens live in `data_dir/tokens.json`, mode 0600. The file holds hashes,
never values. Writers use `tokens.lock` and replace the file atomically. A running
server picks up changes within seconds. Last use is kept in `token-usage.json`.
The server writes usage at most once a minute and when it stops.
Once `tokens.json` exists, authentication stays on, even after every token in it expired
or was revoked: sign in again with a token from `serve token new` on the server, or delete
the file to open the server. A `tokens.json` that cannot be read stops `serve` from starting.

The Access tab lists config and generated tokens. Operators can create and revoke
tokens there. The value appears once with a Copy button. Creating the first
token enables authentication. The tab asks for confirmation because callers without
a token lose access.

`GET /api/tokens` lists metadata for authenticated callers, without values or hashes.
`POST /api/tokens` accepts `name`, `role`, `jobs` and `expires` and returns the value once.
Only operators can create tokens. A token cannot grant a higher role than its caller.
A scoped caller must supply a nonempty subset of its own jobs. Jobs must exist.
Every job must be one the caller can run (its operators list admits the caller); without
`jobs` the token covers every job the caller can run. The token then passes the operators
list of those jobs: it gets its creator's access to exactly them, recorded as `delegated`
in the store and in `GET /api/tokens`. The pass is fixed when the token is made: a
creator who later leaves the group keeps the token working until it expires or is revoked.
Only jobs that already had an operators list are recorded; an operators list added to a job
later applies to existing tokens too, so tightening a job never leaves old tokens past it.
The Access form lists only the jobs the caller can run.
A new token never expires later than the caller's own token; a later expiry is cut to it.
Groups cannot be set through the API. Names are unique across config and store.
When authentication is off, creation requires `enable_auth: true`.
`DELETE /api/tokens/{name}` revokes a generated token immediately. An operator may
revoke any generated token when unscoped, or a token they created when scoped.
Creation, revocation and refused attempts are audited.

Generated tokens match a job's operators list by name only as `token:<name>`, by their delegated jobs,
or by groups given with `serve token new --groups`. `serve token new --jobs` delegates the listed jobs.
A bare name still matches config tokens for compatibility. CLI generation accepts
`--groups`; the API and Access form do not accept groups.

Named bearer tokens in `auth.tokens` have a `name`, a `token` value (expands `${VAR}`),
and optional `groups`, `role`, `jobs` and `expires`. Names must be unique and values
must be nonempty. `role` is `operator` (default) or `viewer`. `jobs: [drift-site]` limits
run, approve, reject and cancel actions to existing jobs in that list. Reading stays open.
A jobs-scoped token cannot pause or resume the server or refresh the inventory. `expires` accepts a date (start
of that day in UTC) or RFC3339. Expired tokens return `401`.

`token` accepts a plain value or `sha256:<64 hex digits>` for its SHA-256 hash. Both
forms use a constant-time digest comparison. Tokens are checked before proxy headers.
Generate a value and a YAML entry with its hash:

```sh
onigirazu serve token new --print --name ci --jobs drift-site --groups tier2 --expires 90d
```

The command does not read `serve.yml`. Store the printed value in your secret manager
and add the entry to `auth.tokens`. Names must match `[a-z0-9][a-z0-9._-]{0,63}`.
`--role` defaults to `operator`; use `viewer` for read access. `--jobs` and `--groups`
accept comma-separated lists. `--expires` defaults to `90d` and accepts a number of
days, a date, RFC3339, or `never` to omit expiry. Day counts use the UTC date now plus
that many days. Values start with `ozs_` followed by 43 base64url characters.

`GET /api/me` includes `auth` (`token`, `proxy`, `none`) and `role` (`operator`, `viewer`).
It includes `via` (`header` or `cookie` for tokens, empty otherwise), `jobs` for scoped
tokens and `expires` for tokens with an expiry.
`GET /metrics` exposes `onigirazu_serve_token_expiry_timestamp_seconds{name}` for each
token with an expiry, including expired tokens.

A job's optional `operators` list narrows run, approve, reject and cancel access: the caller must
be a server-level operator and match a listed user name, group or `token:<name>`,
case-insensitively. `token:ci` admits only the token named `ci`. Bare token names still
match for compatibility. A token's `jobs` scope also applies when the job has no
`operators` list. The legacy `auth.token` has the user name `token`; include `token:token`
or `token` in a job's list to admit it. Pause, resume and inventory refresh require an unscoped operator.

Every run, approve, reject, cancel, pause and resume action and refused attempt is recorded in
`data_dir/audit.log` as JSON lines, with identity, job, run ID, result, comment or reason,
and client address. Scheduling details include `not_before=<time>`. Unauthenticated probes other than sign-in attempts are omitted. Viewers can read the entries,
newest first, through `GET /api/audit` or the Audit tab. The file is append-only and
reopened for every entry, so logrotate can move it aside (no `copytruncate` needed).

## Extra pages

`ui.pages_dir` serves your own static pages under `/pages/`, on the same origin as
the UI and REST API. Relative paths resolve against the directory containing
`serve.yml`; the directory must exist at startup. `ui.links` adds sidebar entries:
each needs a nonempty `title` and an `href` beginning with `/` (not `//`), `http://`
or `https://`. External links open in a new tab. A link to a page under `/pages/`
opens inside the UI (in the main area, with the sidebar kept) and is loaded with
`?embed=1`, so the page can hide its own header; other `/` links navigate normally.
On wide screens the menu button in the top bar hides or shows the sidebar; the choice is
remembered in the browser. On phones it opens the navigation drawer.
Links can also be configured without `pages_dir`.

```yaml
ui:
  pages_dir: /srv/app/serve/pages
  links:
    - {title: Map, href: /pages/map.html}
```

Pages require the same identity as the API. With tokens, signing in on the main page
is enough: pages and their fetch calls then work without Authorization headers. Proxy
cookies work on the same origin too; pages also work when authentication is unconfigured.
An unauthenticated HTML request gets a sign-in link to `/`.

Create `map.html` inside the pages directory, for example:

```html
<!doctype html>
<title>Fleet map</title>
<pre id="jobs"></pre>
<script>
fetch('/api/jobs').then(r => {
  if (!r.ok) throw new Error('API request failed: ' + r.status);
  return r.json();
}).then(jobs => {
  document.getElementById('jobs').textContent = JSON.stringify(jobs, null, 2);
}).catch(error => {
  document.getElementById('jobs').textContent = error.message;
});
</script>
```

Only GET and HEAD are supported. Directories serve `index.html` when present;
there are no directory listings. Hidden path segments and symlinks escaping the
pages directory are not served. Responses use `Cache-Control: no-cache`,
`X-Content-Type-Options: nosniff`, and `Referrer-Policy: same-origin`.

## Live progress

Pending, scheduled, queued and running run pages open a Progress tab with each host's current task,
result counts and a live task event log. Events carry play and task names, host,
status, timing and a per-run sequence number. Secret parameter values in names
are masked. Output, error text, diffs and variables are never sent. Progress is
memory-only: the latest 1000 events and host totals are kept for each active run
and the last 20 finished runs, and disappear after restart. The page reconnects
with `after` when a stream drops and opens the report when the run ends.
Planned task totals are estimates from the playbook; handlers and health checks join
the plan when they run, conditions and dynamic includes can make the actual number
differ. A loop is one task: the host row shows its item (`item 48 / 50`) and the loop
counts once, with its worst item result (failed, changed, ok, skipped).

## API

| Method and path | Returns |
|---|---|
| `GET /api/audit` | action log array, newest first (viewers too); filters `user`, `action`, `job`, `run`, `ok=true/false`, `from`, `to` (RFC 3339); `limit` defaults to 100, max 1000; pass `X-Next-Cursor` as `before` for older entries; `400` for invalid cursor, `ok` or dates |
| `POST /api/session` | sign in with `{token, remember}`; returns `/api/me` state and sets the session cookie; `401` for unknown/expired tokens, `429` after 10 failures per minute per client address |
| `DELETE /api/session` | clears the session cookie; `204`; cookie CSRF rule applies; audits `sign_out` when the identity resolves |
| `GET /api/me` | the caller: `user`, `operator`, `auth`, `role`, `via`, optional `jobs` and `expires`, `title`, `paused` (bool), `running`, `queued` and `scheduled` counts, `concurrency`, `storage_error` (last result storage error, empty when fine) |
| `GET /api/pause` | pause state: `paused`, optional `reason`, `by`, `at` (viewers too) |
| `POST /api/pause` | pauses execution (operators); optional JSON `{reason: "..."}`; `200` with the state; pausing again updates reason and identity; `500` with the state and `error` when `pause.json` could not be written (the pause still holds in this process) |
| `POST /api/resume` | resumes execution (operators); `200` with the state; already resumed is a no-op; `500` as for pause |
| `GET /api/jobs` | the jobs with schedule (`every` or `at` and `at_tz`), effective `missed` policy, `next`, `next_reason`, optional `missed_at`, `last_scheduled` and `last_outcome`, optional job `timeout` (duration string), `running` flag (queued, running or cancelling), optional `active` record (without report; `active.requested` and `active.started` give queue and execution timing), `queued` flag (waiting for the worker), last executed result, `recent: [{status, mode, started}]` (up to 20 executed runs, newest first, including checks), optional `pending` and `scheduled` records (without reports), `scheduled_runs` when automatic deferrals collide, optional `windows`, `windows_tz` and `next_window` for gated jobs, and `approval: "required"` for guarded jobs; optional `operators` list and caller-specific `allowed` bool; `adhoc: {limit, extra_vars, secret_vars, vars}` (`extra_vars` remains a list of names; `vars` contains declarations with `name`, `type`, `choices`, `default`, `pattern`, `min`, `max`, `description`, `required`, `secret`; secret defaults are omitted) when enabled |
| `GET /api/jobs/{name}/inventory` | operators of jobs with `adhoc.limit: true` only; sorted reachable host names and nonempty groups that lie entirely within the job's limit (recursive, excluding `all`); `?limit=pattern` previews resolved host names; `400` for invalid limits or jobs without ad-hoc limits; `403` for callers not allowed to operate the job; `500` for inventory load errors |
| `POST /api/jobs/{name}/run` | requests the job (operators); optional JSON `{mode: "check", limit: "host pattern", extra_vars: {k: "v"}, not_before: "RFC 3339 time"}` with limit/variables as allowed by `adhoc`; mode is `check` (apply jobs only), `apply` or empty; `400` for invalid parameters or a limit with no hosts or hosts outside the job's limit; `202` with `{queued: name, run: id}` or `{pending: id, run: id}` or `{scheduled: id, run: id}`; `409` when running, queued or pending for an immediate request, or when another run is already scheduled for a request with `not_before`; `503` when the queue is full; `500` with the record and `error` when storage fails (the run remains in memory); `403` with `{"error":"not an operator of this job"}` when the job disallows the operator |
| `GET /api/jobs/{name}/history?limit=N` | light retained history, newest first (viewers allowed); default 50, maximum 500: `[{id, status, mode, trigger, requested, started, duration_seconds, summary, hosts_count, failed_hosts, params, approval}]`; `params` as stored (secret values masked) and `approval` appear when set; reports and host lists are never included |
| `GET /api/jobs/{name}/compare?a=<id>&b=<id>` | compare two different executed drift/apply runs of this job with reports (viewers allowed; `400` when `a` equals `b`): `{a: {id, started, status, playbook_hash}, b: {...}, same_playbook, hosts: [{host, a, b}], appeared: [{host, key, play, task, module}], resolved: [...]}`; only changed host verdicts are listed (empty verdict means absent); items match by host and task key when both have keys, otherwise host, play and task; `404` for unknown runs, `400` for another job, unsupported kind, unexecuted runs or missing reports |
| `GET /api/jobs/{name}/results` | the kept results, newest first |
| `GET /api/jobs/{name}/results/{id}` | one result with its full report (the drift, verify or comply report as JSON) |
| `GET /api/results/{id}` | one job result with its full report, optional `mode: "check"`, optional `not_before`, optional `audit_execution_id`, optional `parent` run ID for reruns, and optional `params` (`limit`, `extra_vars`, sorted `hosts`, optional `secret_vars` names); `404` if absent |
| `POST /api/runs/{id}/rerun-failed` | job operators; reruns only failed/unreachable hosts of an executed run with the same mode and non-secret variables; `202` with `{run: id}`; new record links to `parent`; `400` for unexecuted runs, no failed hosts, invalid host limits or secret parameters; `403` if disallowed, `404` if absent, `409` if busy, `503` if queue full; normal approval/windows apply |
| `POST /api/runs/{id}/approve` | queues or schedules an approved pending run (operators); optional JSON `{comment: "..."}`; `200` with the record without report, `404` if absent, `409` when the run expired meanwhile (it is marked expired), `503` if the queue is full; `500` with the record and `error` on storage failure; `403` with `{"error":"not an operator of this job"}` when the job disallows the operator |
| `POST /api/runs/{id}/reject` | rejects a pending run (operators); optional JSON `{comment: "..."}`; `200` with the record without report, `404` if absent; `500` with the record and `error` on storage failure; `403` with `{"error":"not an operator of this job"}` when the job disallows the operator |
| `POST /api/runs/{id}/cancel` | cancels a scheduled, queued or running run (job operators); optional JSON `{comment: "..."}`; `200` with the cancelled record for scheduled or queued runs, `202` with the cancelling record for running runs (without report); `404` if absent, `409` if not cancellable (reject a pending run instead), `500` with the record and `error` on storage failure; `403` if disallowed |
| `GET /api/hosts/{name}/results` | that host’s retained results across all jobs, newest `started` first, at most 200: `[{id, job, kind, status, host_status, started, summary}]`; an unknown host returns `[]` |
| `GET /api/results/{id}/events?after=N` | live NDJSON progress (viewers allowed): snapshot, replay after sequence N, gap marker when replay was truncated, task events and final end; idle pings every 15 seconds |
| `GET /api/inventory` | inventory sources shared by jobs (viewers too), with sorted groups, direct host membership, recursive host groups and variable names; operators also receive host addresses, host `effective` entries (`name`, `from`: `host` or `group:<name>`, `source`) sorted by name, and group `var_sources` maps from variable name to source path; values are never included; each entry has `source`, `jobs`, `loaded_at`, `groups`, `hosts`, and `error` if loading failed |
| `POST /api/inventory/refresh` | clears inventory snapshots (operators); the next GET reloads; `200 {"ok":true}`; audited as `inventory_refresh` |
| `GET /api/hosts` | every host seen by a job's last executed result (cancelled and check runs are skipped), with its worst status (`ok`, `changed`, `drift`, `failed`, `unreachable`) and the status per job |
| `GET /api/runs` | audit store run array, newest first (viewers too); filters `host`, `playbook` (path substring), `status`, `job` (linked serve runs), `from`, `to` (RFC 3339); `limit` defaults to 100, max 1000, `offset` defaults to 0; `X-Next-Offset` gives the next offset when more exist; `400` for invalid dates or a non-numeric, negative or overflowing offset; optional `serve_run` links to the retained serve result |
| `GET /api/runs/{id}` | one audit record, with optional `serve_run` |
| `GET /metrics` | `onigirazu_serve_job_last_status{job,kind}` (0 ok, 1 drift/changed, 2 failed, 3 error, 4 cancelled, 5 timed_out), `onigirazu_serve_job_last_run_timestamp_seconds{job}`, `onigirazu_serve_host_status{job,host}`, `onigirazu_serve_job_pending{job}` (guarded jobs only: 1 pending, 0 otherwise); `onigirazu_serve_paused` (1 paused, 0 otherwise); `onigirazu_serve_running` (executing or cancelling runs), `onigirazu_serve_concurrency` (configured limit), `onigirazu_serve_scheduled` (scheduled run count); `onigirazu_serve_storage_error` (1 while the last result storage operation failed, 0 otherwise); `onigirazu_serve_token_expiry_timestamp_seconds{name}` (expiry as Unix seconds); all include configured metric labels |
| `GET /healthz` | `ok` |

Run, approve, reject and cancel return `403` with `{"error":"not an operator of this job"}` when
a server-level operator does not match the job's list. Viewers receive `403` with
`{"error":"operators only"}`; missing identity receives `401`. An unknown pending run
returns `404` before the per-job check.

Check runs are excluded from job last-status, last-run timestamp and host-status metrics
and host verdicts. Apply checks do not push drift metrics. The jobs API still shows the
newest executed result in `last`, including checks.

The Runs and Audit tabs have compact filters with shareable hash links and an Older button that appends earlier entries.
On Runs and Audit, From and To use the browser's date and time picker (local time, sent as RFC 3339) with quick ranges (last hour, 24 h, 7 days, 30 days); Job is a list of the jobs, Host suggests the hosts seen in results and in the inventory, Playbook suggests file names (the filter matches part of the path), User suggests the users already listed. Active filters show as removable chips; on phones the filters fold under a Filters button. Apply, Reset and the per-page selector control the loaded history; paging cursors stay out of the form.

Page: the page at `/` uses the same API. A top bar holds the fleet name, a search box (`/`
focuses it; it filters the current list), the activity counts, Pause/Resume for operators, the
theme switch (system, light, dark; kept in the browser) and the user menu with the bearer token.
A sidebar leads to Jobs, Runs, Hosts, Inventory and Audit; it shrinks to icons below 900 px and
becomes a drawer below 600 px. Jobs is a table (cards on phones) with Run and a menu for Run…,
Run at…, Check and Cancel when `allowed`; pending approvals are cards above it. Run… and Run at…
open a side sheet with the limit picker, variables and the start time. A job page lists its runs on
the left and shows the selected run (the newest when the page opens) on the right, with Rerun
failed hosts, Cancel, Approve/Reject and the report tabs; on phones the two stack. Check runs have
a `check` badge in the jobs table, run list and run header.

Inventory shows one card per inventory source with its jobs, groups, hosts and a host filter.
Snapshots load lazily and are reused for five minutes, including failed loads; concurrent
requests for the same source share one load, other sources are not held up by it, and a load
gives up after one minute. Variable names are shown without values or connection credentials.
Operators can expand a host's effective variables to see whether each comes from the host
or a group, and its source file (full path in the tooltip). Sources include inventory files,
dynamic scripts and `group_vars`/`host_vars` files. Only operators see these paths, host
addresses and inline inventories (viewers see `inline` instead). Operators can use Refresh
to clear snapshots for the next request. Conflicting native host definitions report a
provenance error when their effective sources depend on the selected group; the engine's
resolution behavior is unchanged.

Jobs show up to 20 status squares, newest on the right, with hollow squares for check
runs and status/date tooltips. The job page uses light history with a duration sparkline
for up to 50 runs. Select two executed drift/apply runs and choose Compare to see host
verdict changes and appeared/resolved drift tasks; a warning appears when playbook hashes
differ or are unavailable. Individual run pages still load full reports.

Run pages show metadata and tabs by kind: drift/apply
have expandable Hosts, Changes grouped by task, coloured Diff (limited to the first
200 KB), Tasks with collapsible per-host status and duration lists, host duration totals
and the slowest 10 retained tasks, and Orphans when present; verify has expandable Hosts and Checks across hosts;
comply has a control-by-host Matrix with unreachable hosts and scores, plus Remediation
grouped by control. Apply runs include the diff of what they changed. Every kind retains a full JSON tab. Audit records open as JSON.
Host names open `#host/<name>` with retained serve results across jobs and a separate
“runs in the audit store” link. Jobs show a muted “never” badge before the first executed
run and “stale” when an interval job’s last execution is older than twice its interval (anchored `at` jobs have no stale badge);
queued and running jobs show wait and elapsed time. Tables scroll horizontally on narrow
screens; at 700px and below the Jobs Target column is hidden and dates are shortened.

A job result: `{id, job, kind, mode, requested, started, duration_seconds, status, summary, trigger,
hosts, report, approval, cancel, params, audit_execution_id, playbook_hash}`. `requested` is the request time; `started` becomes the execution
start time when the worker begins. `trigger` is `schedule` or `api:<user>`. Status is `pending`, `scheduled`,
`queued`, `running`, `cancelling`, `cancelled`, `timed_out`, `rejected`, `expired`, `ok`, `drift`, `changed`, `failed` or `error`.
An executed run's `summary` counts hosts (`2 in sync, 1 changed, 1 failed`); when a task failed it
ends with ` — <host>: <first line of the first failure>` (hosts sorted), so a playbook that puts a
one-line reason first in its failure message is self-explaining in lists and notifications.
The optional approval object has `required`, `expires`, and, after a decision, `decision`
(`approved`, `rejected`, `expired`), `by` (operator identity; absent for expiry), `at` and an
optional `comment`. Job results open at `#job/<name>` and individual runs at `#run/<id>`.

Drift/apply/verify results include optional `playbook_hash`: the first 16 hexadecimal
characters of the SHA-256 of the playbook file at execution time (that file only: roles,
included files and variables are not part of it). It is absent if the
file could not be read or for comply jobs. Drift report items include optional `key`,
the stable task key; older records may lack it. `same_playbook` is true only when both
compared runs have equal, nonempty hashes. Comparison follows the requested A-to-B
direction; the API does not require chronological order.

Serve results include `audit_execution_id` when drift, apply, verify or compliance starts audit
recording. Audit history rows expose the matching `serve_run` while that serve result
is retained. Both pages link to each other at `#run/<id>`. Compliance runs retain
completed controls and the execution link when canceled or timed out;
hosts without completed controls receive no verdict.

Drift/apply report JSON includes optional `tasks`, keyed by host, with rows in run order:
`{key, play, task, module, status, duration_ms}`. Status is `ok`, `changed`, `failed`,
`skipped` or `ignored`. Rows exclude output and diffs and are capped at 2000 per host;
`tasks_truncated: true` signals omitted rows. Host duration totals and the slowest
list then cover retained rows only. Drift/error items also include optional `duration_ms`.

`metrics.push` sends each drift job's drift metrics and each comply job's compliance metrics
as `drift --metrics-push` and `comply --metrics-push` do. Stopped compliance runs do not push compliance metrics.
