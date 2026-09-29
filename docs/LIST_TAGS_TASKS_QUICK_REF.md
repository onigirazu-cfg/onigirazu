# Tag and Task Discovery - Quick Reference

Short form of the [List Tags and Tasks Guide](LIST_TAGS_TASKS_GUIDE.md). Both flags only read the
playbook; nothing runs on the hosts.

## Commands

| Task | Command |
|------|---------|
| List all tags | `onigirazu apply playbook.yml --list-tags` |
| List all tasks | `onigirazu apply playbook.yml --list-tasks` |
| Tasks for one tag | `onigirazu apply playbook.yml --list-tasks --tags setup` |
| Tasks without some tags | `onigirazu apply playbook.yml --list-tasks --skip-tags debug` |
| Both filters | `onigirazu apply playbook.yml --list-tasks --tags setup --skip-tags test` |
| JSON / YAML / CSV | `onigirazu apply playbook.yml --list-tasks --output json` (`yaml`, `csv`) |

## `--list-tasks` marks

| Mark | Meaning |
|------|---------|
| `✓` | would run |
| `✓ … [ALWAYS TAG]` | would run because of the `always` tag |
| `✗ … [SKIPPED: NEVER TAG]` | tagged `never` |
| `✗ … [SKIPPED: SKIP-TAG MATCH]` | one of its tags is in `--skip-tags` |
| `✗ … [SKIPPED: TAG MISMATCH]` | none of its tags is in `--tags` |

## Tag filter syntax

```bash
--tags setup                  # one tag
--tags setup,deployment       # any of these
--tags all                    # everything except `never` (the default)
--tags tagged                 # tasks with at least one tag (and `always`)
--tags untagged               # tasks without tags (and `always`)
--skip-tags debug,test        # exclude
```

## Scripting

`--list-tasks` writes only the listing to stdout in every format; `--list-tags` does so only with
`--output json` or `yaml`.

```bash
# tasks that would run
onigirazu apply playbook.yml --list-tasks --tags setup --output json | jq '.summary.WouldExecute'

# total tasks
onigirazu apply playbook.yml --list-tasks --output json | jq '.summary.TotalTasks'

# tag names with counts
onigirazu apply playbook.yml --list-tags --output json | jq -r '.tags.by_count[] | "\(.name): \(.count)"'
```

## See Also

- [List Tags and Tasks Guide](LIST_TAGS_TASKS_GUIDE.md)
- [Tag Filtering Guide](TAG_FILTERING.md)
