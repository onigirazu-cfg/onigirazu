# Machine-readable results

## `apply --log-format json`

One JSON record per line on stdout; the progress bar may share the line, so split at `{"timestamp"`.
Each finished task on a host is a record with `fields.type == "task_end"`:

| Field | Meaning |
|-------|---------|
| `host`, `task`, `module` | An unnamed task is called after its module, as in Ansible |
| `success` | `false` when the task failed (also when the failure was ignored) |
| `changed`, `skipped` | Task status |
| `ignored` | Failed, handled by `ignore_errors` or a `rescue` section |
| `msg` | Error of a failed task; message (or `var=` output) of a debug task |

```bash
onigirazu apply site.yml -i hosts.yml --log-format json --no-color |
  grep -o '{"timestamp.*' | jq -c 'select(.fields.type == "task_end") | .fields'
```

## `apply -o json`

The run as one document: `tasks` in run order, one entry per task (tasks with the same name stay
separate), `host_results.<host>` with `status` (`success`, `changed`, `skipped`, `failed`,
`ignored`), `error` and `output` (the task's message, as `msg` above), plus totals.
