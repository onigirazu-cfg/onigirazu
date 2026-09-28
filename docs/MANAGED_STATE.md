# Managed state

apply remembers what each playbook manages on every host: the files, packages, services, users
and groups its tasks keep, which task claims each of them, and what was there before. When a task
leaves the playbook, `plan` shows what becomes of its resources. Ansible has nothing like it: a
deleted task leaves its file or package behind.

Removing and restoring orphaned resources during apply is not implemented yet; for now `plan`,
`drift` and apply list them.

## Where it lives

`.onigirazu/<playbook name>.state.json` next to the playbook, one file per playbook, mode 0600.
It holds the previous content of adopted text files (up to 1 MiB), so keep it out of public
repositories.

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

With `--limit` only the limited hosts are looked at. A host no play targets any more is not looked at
yet: its records stay until `state rm`.

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

## Commands

```
onigirazu state resources site.yml          # table; --json for the records
onigirazu state rm site.yml web1 file /etc/app.conf   # forget one; the host is not touched
```

## prevent_destroy

```yaml
- name: data directory
  file: {path: /srv/data, state: directory}
  prevent_destroy: true
```

Once such a task leaves the playbook, its resources are only forgotten.
