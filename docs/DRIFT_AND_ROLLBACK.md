# Plan, drift, diff and rollback

Four ways to see and control what a playbook does to hosts. All of them use the
same inventory flags as `apply` (`-i`, `--limit`, `-e`, `-b`, `--become-user`,
`-u`, `--private-key`, `--tags`, `--skip-tags`).

| Command | What it does | Changes hosts |
|---------|--------------|---------------|
| `onigirazu plan site.yml -i hosts.yml` | shows what `apply` would change, per host | no |
| `onigirazu drift site.yml -i hosts.yml` | reports hosts that no longer match the playbook | no (`--fix`: yes) |
| `onigirazu apply site.yml --diff` | applies and shows before/after of changed files | yes |
| `onigirazu rollback --last -i hosts.yml` | puts back what the last run changed | yes |

## plan

`plan` runs the playbook in check mode against the hosts. Every task that would
change something is listed under its host, with a unified diff for file changes.

```
$ onigirazu plan site.yml -i hosts.yml
Plan: 2 task(s) would change 1 of 3 host(s)

web1
  ~ App config (template)
      --- before: /etc/app.conf
      +++ after: /etc/app.conf
      @@ -1,2 +1,2 @@
      -port = 8081
      +port = 8080
       workers = 4
  ~ App user (user): user app would be created
```

Exit code 0 (1 when a task could not be checked). `--format json|html` and
`--output FILE` as for drift.

Modules without check mode support are skipped in plan and drift and do not
show up; see [Check mode](../README.md#check-mode) for the list.

## drift

`drift` is the same check, meant to run on a schedule: are the hosts still what
the playbook says?

```bash
onigirazu drift site.yml -i hosts.yml                          # text report
onigirazu drift site.yml -i hosts.yml --format json            # for scripts
onigirazu drift site.yml -i hosts.yml --format html --output drift.html
onigirazu drift site.yml -i hosts.yml --fix                    # apply when drift is found
onigirazu drift site.yml --history                             # past checks
```

| Exit code | Meaning |
|-----------|---------|
| 0 | all hosts match the playbook |
| 2 | drift: at least one task would change a host |
| 1 | at least one task could not be checked (unreachable host, failed task) |

Every check is stored in `~/.onigirazu/drift-history` (the newest 500). A
drifting task shows since when it has drifted in the checks in a row (`[since
3h]`; `since` in JSON). A check in which the host was in sync ends the run; a
check that did not look at the host (another `--limit`) does not.

### Notifications

`--notify URL` (repeatable) posts to a Slack or Mattermost style webhook when
drift or errors are found; `--notify-always` also when all hosts are in sync.
The body is `{"text": "<the text report>", "report": <the JSON report>}`. A
failed post is reported on stderr and does not change the exit code.

### On a schedule

```ini
# /etc/systemd/system/onigirazu-drift.service
[Service]
Type=oneshot
WorkingDirectory=/srv/infra
EnvironmentFile=/etc/onigirazu/drift.env
ExecStart=/usr/local/bin/onigirazu drift site.yml -i hosts.yml --notify ${WEBHOOK}
SuccessExitStatus=2

# /etc/systemd/system/onigirazu-drift.timer
[Timer]
OnCalendar=hourly

[Install]
WantedBy=timers.target
```

In CI, run `drift` in a scheduled job and keep `--format html --output
drift.html` as an artifact; exit code 2 fails the job when hosts have drifted.

## --diff

`apply --diff` (with or without `--check`) prints, after the run, a unified diff
for every changed file: `copy`, `template`, `lineinfile`, `blockinfile`,
`replace`, and mode changes of `file`, `copy` and `template`. Binary files and
files over 256 KiB are summarized instead of shown.

## rollback

Every `apply` (also a failed one) keeps a snapshot in `~/.onigirazu/snapshots`
of what it changed and how it was before:

| Module | Kept before the change | Rollback |
|--------|------------------------|----------|
| copy, template, lineinfile, blockinfile, replace | content (text up to 1 MiB), mode, owner, group — or that the file did not exist | restores the file, or removes it |
| file | mode, owner, group of a directory — or that the path did not exist | restores attributes, or removes the path |
| apt, yum, dnf, package | which of the task's packages were installed | removes what the run installed, reinstalls what it removed |
| service, systemd | running and enabled state | restores both |
| user, group | whether the account existed | deletes an account the run created (a home directory stays) |

Not reversible, and listed as such: upgrades of installed packages, changes to
existing accounts, binary or larger files, other modules.

```bash
onigirazu rollback --list                            # snapshots, newest first
onigirazu rollback --last --dry-run                  # what would be restored
onigirazu rollback --snapshot <id> -i hosts.yml      # restore
onigirazu rollback --cleanup --max-age 30d
```

```
ORDER  HOST  RESOURCE        DETAILS
1      web1  /etc/app.d      restore directory mode=0755 owner=root group=root
2      web1  /etc/app.conf   restore content (812 bytes) mode=0644 owner=root group=root
3      web1  [nginx]         remove packages [nginx]
4      web1  /etc/app.new    remove (did not exist before)
```

Changes are undone newest first, so a file changed twice in a run ends as it was
before the run. Rollback reaches the hosts through the inventory (`-i`) and uses
the task's `become`. The apply log names the snapshot of each run:
`Snapshot created: <id>`.

Snapshots hold file contents; they are written with mode 0600 in your home
directory. Tasks with `no_log: true` keep nothing.
