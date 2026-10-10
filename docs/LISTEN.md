# Listen mode: playbooks on events

`onigirazu listen` serves HTTP and runs playbooks for the events that match its rules: an alert
from Alertmanager or vmalert, a push or release from GitHub, a Mattermost slash command, or any
JSON webhook. Runs are serial (one at a time) and throttled per rule and host.

```yaml
# listen.yml
listen: ":8085"
sources:
  - name: alerts
    type: alertmanager          # webhook | alertmanager | github | mattermost
    path: /alertmanager         # default: /<name>
    token: ${LISTEN_TOKEN}      # Authorization: Bearer, X-Token or ?token=; ${VAR} is expanded
  - name: gh
    type: github
    token: ${GITHUB_WEBHOOK_SECRET}   # the webhook secret: X-Hub-Signature-256 is checked
  - name: chat
    type: mattermost
    token: ${MM_TOKEN}          # the token Mattermost sends with the command
rules:
  - name: disk full
    source: alerts
    when: "event.status == 'firing' and event.labels.alertname == 'DiskFull'"
    playbook: fix-disk.yml
    inventory: hosts.yml
    limit: "{{ event.alerts | map(attribute='labels.instance') | map('regex_replace', ':.*$', '') | join(',') }}"
    extra_vars: {severity: "{{ event.labels.severity }}"}
    become: true
    throttle: 10m               # once per 10 minutes for the same rendered limit
    notify: [https://mattermost.example.com/hooks/xxx]
  - name: deploy main
    source: gh
    when: "event.type == 'push' and event.ref == 'refs/heads/main'"
    playbook: deploy.yml
    extra_vars: {sha: "{{ event.after }}"}
  - name: restart from chat
    source: chat
    when: "event.args[0] == 'restart'"
    playbook: restart.yml
    limit: "{{ event.args[1] }}"
```

```bash
onigirazu listen --config-file listen.yml                 # serve
onigirazu listen --config-file listen.yml --dry-run       # match and log, run nothing
onigirazu listen test -f listen.yml --source alerts --file alert.json   # what an event would run
sudo onigirazu listen install -f /etc/onigirazu/listen.yml              # systemd service
```

## Events

`when` is a Jinja expression over `event`, `headers` (lower-case names) and `source`; `limit` and
`extra_vars` values are templates over the same variables. What `event` holds:

| Source type | `event` |
|---|---|
| `webhook` | the JSON body |
| `alertmanager` | the webhook body (`status`, `alerts[]` with `labels`/`annotations`, `groupLabels`, ...); `event.labels` is the first alert's labels |
| `github` | the payload plus `type` (the `X-GitHub-Event` header) and `delivery` |
| `mattermost` | the form or JSON fields (`text`, `user_name`, `channel_name`, `trigger_word`, ...) plus `args`: the words of `text` after the trigger word; `token` is removed |

A source without `token` accepts anything: put the listener behind a reverse proxy or a firewall
then. Bodies above 1 MB are rejected.

## Runs

A matching rule queues `apply <playbook> [--check] [--limit ...] [--become] [-e k=v ...]`; the
queue is served by one worker, so runs never overlap. `throttle` skips a rule for the same rendered
`limit` within the period (a flapping alert runs the fix once). The response is `202` with the
rules queued and throttled; `422` when a rule's expression fails. `notify` posts the report as
`drift --notify` does when a task fails (or always with `notify_always: true`); `drift_only: true`
runs in check mode.

`/metrics` has `onigirazu_listen_events_total{source}` and `onigirazu_listen_runs_total{rule,result}`
(`ok`, `failed`, `drift`, `error`, `dry-run`); `/healthz` answers `ok`.

Paths in the file (`playbook`, `inventory`) are relative to the file. The state file of a run is
`.onigirazu-state` next to the playbook unless `-s` is given.
