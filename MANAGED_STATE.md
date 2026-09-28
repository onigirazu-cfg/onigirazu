# Managed state

apply remembers what each playbook manages on every host: the files, packages, services, users
and groups its tasks keep, which task claims each of them, and what was there before. When a task
leaves the playbook, `plan` shows what becomes of its resources and the next apply does it, like
`terraform apply`: what onigirazu created is removed, what it took over is put back as it was.
Ansible has nothing like it: a deleted task leaves its file or package behind.

## Where it lives

By default `.onigirazu/<playbook name>.state.json` next to the playbook, one file per playbook,
mode 0600. It holds the previous content of adopted text files (up to 1 MiB), so keep it out of
public repositories.

A team shares it through an S3 bucket (AWS, Garage, MinIO, ...), set in `onigirazu.yml`:

```yaml
managed_state:
  backend: s3
  bucket: onigirazu-state
  prefix: prod/              # object: prod/<playbook name>.state.json
  endpoint: s3.example.org:3900   # empty: AWS
  region: garage
  path_style: true           # most self-hosted servers
  # insecure: true           # plain HTTP
```

Credentials come from `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` (or `AWS_PROFILE` and
`~/.aws/credentials`, or the instance role), never from the file. Playbooks with the same name
need different prefixes.

## What is recorded

| Module | Resource |
|---|---|
| copy, template, file, lineinfile, blockinfile, replace | `file` (the path) |
| apt, yum, dnf, package | `package` (one per name) |
| service, systemd | `service` |
| user, group | `user`, `group` |

Every task that runs claims its resources, changed or not. Origin is set on first sight:

- `created`: it was not there; onigirazu made it
- `adopted`: it was there already
- `unknown`: nothing was captured (check mode, `no_log`)

A task with `state: absent` claims nothing and drops the record.

## Orphans

A resource no task claims any more is an orphan. A claim goes when its task was removed from the
playbook, or when the task ran and manages something else now (another path, fewer loop items or
package names). A claim stays when:

- the task is still in the playbook but did not run on the host (`when`, a loop with no items,
  a handler nobody notified);
- its role or `include_role` was skipped;
- a task failed or was rolled back on the host;
- the run was partial (`--tags`, `--skip-tags`, `--start-at-task`, stopped or canceled).

With `--limit` only the limited hosts are looked at. A host no play matches any more (its group
changed, `hosts:` narrowed) has all its resources orphaned once a complete run without `--limit`
reaches every play; apply connects to it and cleans up. A host that is gone from the inventory too
keeps its records (apply says so; `state rm` forgets them).

```
$ onigirazu plan site.yml -i hosts.yml
No longer in the playbook: 3 resource(s)
  - file /etc/app/extra.conf on web1: remove (onigirazu created it) [was: extra config]
  < file /etc/ssh/sshd_config on web1: put back as it was [was: harden ssh]
  ? user deploy on web1: forget (left as it is) [was: deploy user]
```

| Mark | What happens | When |
|---|---|---|
| `-` | remove | created by onigirazu (a file, a package it installed, a user or group it added) |
| `<` | put back as it was | adopted text file or directory, adopted service |
| `?` | forget, leave it on the host | adopted package, user or group; binary or large files; unknown origin; `prevent_destroy: true` |

`drift` counts orphans as drift (exit code 2).

## apply

After the plays, apply removes or puts back the orphans of the hosts it ran on, newest first
(files inside a directory before the directory), then forgets the `?` ones.

- In a terminal it asks once: `Remove or put back N resource(s) that left the playbook? [y/N]`.
  `--auto-approve` skips the question; without a terminal (CI, cron) apply goes ahead.
- `--no-destroy` keeps them; plan keeps listing them.
- Nothing is removed on a host where a task failed or was rolled back, nor in a partial run
  (`--tags`, `--skip-tags`, `--start-at-task`, stopped or canceled).
- A directory onigirazu created is removed only when empty; otherwise it is left in place and
  forgotten.
- A resource that could not be removed or put back stays an orphan and apply exits non-zero.

```
Managed state: resources that left the playbook
  put back file /etc/ssh/sshd_config on web1
  removed file /etc/app/extra.conf on web1
  kept file /srv/data on web1: directory /srv/data is not empty
```

## Locking

apply locks the playbook's managed state for the run (`<state>.lock`, with who, pid and a lock
ID); a second apply of the same playbook fails at once, or waits with `--lock-timeout 5m`.
In S3 the lock is an object per run under `<state>.lock/`: a run holds it when a listing right
after its write shows no other. It needs no conditional writes (Garage ignores `If-None-Match`),
only read-after-write consistency, which AWS, Garage and MinIO have.
`scripts/garage-s3-test.sh` checks the S3 store and its lock against a throwaway Garage (CI runs it).
`plan` and `drift` do not lock. A lock left by a run that died:

```
onigirazu state unlock site.yml 3f2a9c1d0b7e4a55   # the ID from the error
```

`--lock=false` runs without it.

## Adopting existing hosts

```
onigirazu apply site.yml -i hosts.yml --check --adopt
```

A check run that records what the playbook manages and already exists as `adopted` (with its
current state as "before"), and changes nothing on the hosts; what does not exist yet is recorded
as `created` by the first real apply. Like `terraform import` for a playbook written for hosts
that were set up by hand (`onigirazu import --adopt` does it for the playbook it writes).

## Commands

```
onigirazu state resources site.yml          # table; --json for the records
onigirazu state rm site.yml web1 file /etc/app.conf   # forget one; the host is not touched
onigirazu state unlock site.yml LOCK_ID               # remove a dead run's lock
```

## prevent_destroy

```yaml
- name: data directory
  file: {path: /srv/data, state: directory}
  prevent_destroy: true
```

Once such a task leaves the playbook, its resources are only forgotten, never removed or put back.
