# Pull mode

A host converges itself from a git repository, on a timer, instead of being pushed to. Useful for
hosts the control machine cannot reach (NAT, laptops, air-gapped sites with a mirror) and for a fleet
that should stay converged without a central scheduler.

```bash
onigirazu pull --repo git@git.example.com:ops/site.git --playbook site.yml
```

clones (or updates) the repository into `~/.onigirazu/pull/site` and runs `site.yml` against this
machine over a local connection (an inventory of `localhost` with `ansible_connection: local`). With
`--inventory hosts.yml` (a path inside the repository) and `--limit $(hostname)` the repository's own
inventory is used instead.

| Flag | Meaning |
|---|---|
| `--repo URL` | Git repository (required); any URL git accepts, credentials from the host's git/ssh setup |
| `--branch` | Branch to follow (`main`) |
| `--playbook` | Playbook path inside the repository (`site.yml`) |
| `-i, --inventory` | Inventory path inside the repository; default `localhost`, local connection |
| `--limit` (with `--inventory`), `-e`/`--extra-vars`, `-b`/`--become` | As for `apply` |
| `--dir` | Checkout directory (`~/.onigirazu/pull/<repository name>`) |
| `--interval` | Run again every interval; `0` runs once (`pull install` uses `30m` then) |
| `--only-on-change` | Skip the run when the branch did not move since the last completed run; a skipped run sends no notification and writes no metrics |
| `--drift-only` | Check mode: change nothing, exit 2 when something would change |
| `--notify URL` | Webhook (Slack/Mattermost style) posted to when a task fails or `--drift-only` finds drift; `--notify-always` after every run |
| `--metrics-file`, `--metrics-push`, `--metrics-label` | The run's metrics for node_exporter's textfile collector or pushed to VictoriaMetrics/Pushgateway (see [drift metrics](DRIFT_AND_ROLLBACK.md#metrics)): `onigirazu_pull_last_run_timestamp_seconds`, `onigirazu_pull_ok`, `onigirazu_pull_failed_tasks`, `onigirazu_pull_changed_tasks` (or `_drift_tasks`), `onigirazu_pull_commit_info{commit}` with labels `repo`, `playbook` |

Exit codes: 0 converged, 1 a task failed or the repository could not be fetched (a fetch failure
sends no notification and writes no metrics), 2 drift (with `--drift-only`).

## As a systemd timer

```bash
sudo onigirazu pull install --repo git@git.example.com:ops/site.git --playbook site.yml --interval 30m
```

writes `/etc/systemd/system/onigirazu-pull.service` (oneshot, `onigirazu pull` with these options)
and `onigirazu-pull.timer` (2 minutes after boot, then every interval with up to a minute of
jitter) and starts the timer. `journalctl -u onigirazu-pull` has the runs. The checkout then lives
under root's home (`/root/.onigirazu/pull/<name>`), the repository is read with root's git and ssh
configuration (a read-only deploy key in `/root/.ssh`).

Every run is a normal `apply`: the state file, audit history and rollback snapshots are written in
the checkout directory as for any run.
