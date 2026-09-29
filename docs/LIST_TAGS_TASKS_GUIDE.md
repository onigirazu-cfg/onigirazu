# Tag and Task Discovery Guide

## Overview

Two `apply` flags inspect a playbook without running anything:

- **`--list-tags`**: the tags used in the playbook, with counts
- **`--list-tasks`**: the tasks, marked as would-run or skipped for the given `--tags` / `--skip-tags`

Both read only the playbook; nothing runs on the hosts, so `--check` and `--diff` have no effect on the listing. For an exact per-host preview of what would change, use `onigirazu plan`.

## What is listed

Tasks as they run: the tasks of `roles:` entries (named `role : task`) and the tasks inside `block`s
and of `include_role` / `import_role` are listed, with the tags they inherit from the play, the
`roles:` entry, the include and the block. Handlers are not listed; they run only when notified. The
decision uses the same tag filter as execution (case-insensitive, `tagged`/`untagged`/`all`). Not
listed: tasks of role dependencies; `when` conditions are not evaluated.

## Quick Examples

```bash
# Tags in a playbook
onigirazu apply playbook.yml --list-tags

# Tasks that would run with a filter
onigirazu apply playbook.yml --list-tasks --tags setup

# With skip tags
onigirazu apply playbook.yml --list-tasks --tags deployment --skip-tags experimental
```

## Example Playbook

```yaml
---
- name: Infrastructure Setup
  hosts: all
  tasks:
    - name: Install Base Packages
      package:
        name: "{{ item }}"
        state: present
      loop: [curl, git, vim]
      tags: [packages, setup, base]

    - name: Configure Firewall
      service:
        name: ufw
        state: started
        enabled: true
      tags: [security, setup]

    - name: Create Application User
      user:
        name: appuser
        shell: /bin/bash
      tags: [setup, users]

- name: Application Deployment
  hosts: webservers
  tasks:
    - name: Deploy Latest Code
      shell: git pull origin main
      tags: [deployment, critical]

    - name: Install Dependencies
      package:
        name: python3-pip
        state: present
      tags: [deployment, packages]

    - name: Run Tests
      shell: pytest tests/
      tags: [testing, experimental]

    - name: Health Check
      shell: curl -f http://localhost/health
      tags: [always]
```

## `--list-tags`

```bash
onigirazu apply deployment.yml --list-tags
```

Output (after the startup banner):

```
Available tags in playbook:

  Tag Name          Count
  ─────────────────────────────────────
  setup             3
  deployment        2
  packages          2
  base              1
  critical          1
  experimental      1
  security          1
  testing           1
  users             1

Special tags:
  always            Always runs (1 tasks)

Summary:
  Total unique tags:  10
  Total tasks:        7
  Tagged tasks:       7
  Untagged tasks:     0
  Always tasks:       1
```

Tags are sorted by count, then by name. `always` and `never` are listed separately.

## `--list-tasks`

```bash
onigirazu apply <playbook> --list-tasks [--tags TAG1,TAG2] [--skip-tags TAG3] [--output text|json|yaml|csv]
```

Tasks are listed per play in the order pre_tasks, roles, tasks, post_tasks. Each task gets one of:

| Mark | Meaning |
|------|---------|
| `✓` | would run |
| `✓ … [ALWAYS TAG]` | would run because of the `always` tag |
| `✗ … [SKIPPED: NEVER TAG]` | tagged `never` |
| `✗ … [SKIPPED: SKIP-TAG MATCH]` | one of its tags is in `--skip-tags` |
| `✗ … [SKIPPED: TAG MISMATCH]` | none of its tags is in `--tags` |

`when` conditions are not evaluated.

### Example: tags and skip tags

```bash
onigirazu apply deployment.yml --list-tasks --tags deployment --skip-tags experimental
```

```
Tasks that would execute:

Filters applied: --tags deployment
Skip filters: --skip-tags experimental

Play 1: Infrastructure Setup (Hosts: all)
  ✗ [packages, setup, base] Install Base Packages [SKIPPED: TAG MISMATCH]
  ✗ [security, setup] Configure Firewall [SKIPPED: TAG MISMATCH]
  ✗ [setup, users] Create Application User [SKIPPED: TAG MISMATCH]
  Summary: 0 would execute, 3 skipped

Play 2: Application Deployment (Hosts: webservers)
  ✓ [deployment, critical] Deploy Latest Code
  ✓ [deployment, packages] Install Dependencies
  ✗ [testing, experimental] Run Tests [SKIPPED: SKIP-TAG MATCH]
  ✓ [always] Health Check [ALWAYS TAG]
  Summary: 3 would execute, 1 skipped

Overall Summary:
  Total tasks:       7
  Would execute:     3
  Would skip:        4
  Skip reasons:
    - skip-tag match: 1
    - tag mismatch: 3
```

A task that matches `--skip-tags` is reported as a skip-tag match even when it also misses `--tags`.

## Output Formats

`--output` (`-o`) selects `text` (default), `json`, `yaml` or `csv`.

`--list-tasks` writes only the listing to stdout in every format; the banner and logs go to stderr. `--list-tags` does so only with `json` and `yaml`; with `text` and `csv` the banner precedes the listing on stdout.

### `--list-tasks --output json`

```json
{
  "filters": ["setup"],
  "skip_tags": [],
  "plays": [
    {
      "name": "Infrastructure Setup",
      "hosts": "all",
      "tasks": [
        {"name": "Install Base Packages", "module": "package", "tags": ["packages", "setup", "base"],
         "status": "execute", "skip_reason": ""}
      ],
      "summary": {"Total": 3, "Would": 3, "Skipped": 0, "SkipInfo": {}}
    }
  ],
  "summary": {"TotalTasks": 7, "WouldExecute": 4, "Skipped": 3, "SkipDetails": {"tag mismatch": 3}}
}
```

`status` is one of `execute`, `unconditional` (always tag), `skip_never`, `skip_tags`, `skip_skip_tags`.

### `--list-tags --output json`

```json
{
  "tags": {
    "count": 9,
    "by_count": [{"name": "setup", "count": 3, "tasks": ["Install Base Packages", "Configure Firewall", "Create Application User"]}]
  },
  "special_tags": {
    "count": 1,
    "data": [{"name": "always", "count": 1, "tasks": ["Health Check"]}]
  },
  "summary": {"TotalTags": 9, "TotalTasks": 7, "AlwaysTasks": 1, "NeverTasks": 0,
              "TaggedTasks": 7, "UntaggedTasks": 0, "UniqueTags": 10}
}
```

### CSV columns

- `--list-tags`: `Tag Name, Type, Count, Tasks`
- `--list-tasks`: `Play, Task Name, Tags, Module, Status, Skip Reason` (status is `execute` or `skip`)

## Use Cases

### Understanding a new playbook

```bash
onigirazu apply production.yml --list-tags
onigirazu apply production.yml --list-tasks
onigirazu apply production.yml --list-tasks --tags setup
```

### Phased deployment

```bash
onigirazu apply deploy.yml --list-tasks --tags setup
onigirazu plan deploy.yml --tags setup
onigirazu apply deploy.yml --tags setup
```

## Scripting Examples

### Run setup only if there are setup tasks

```bash
#!/bin/bash
count=$(onigirazu apply deploy.yml --list-tasks --tags setup --output json | jq '.summary.WouldExecute')

if [ "$count" -gt 0 ]; then
    echo "Running $count setup tasks..."
    onigirazu apply deploy.yml --tags setup
else
    echo "No setup tasks to run"
fi
```

### Tag counts

```bash
onigirazu apply deploy.yml --list-tags --output json | jq -r '.tags.by_count[] | "\(.name): \(.count)"'
# setup: 3
# deployment: 2
# ...
```

### Python: parse the task list

```python
#!/usr/bin/env python3
import json
import subprocess

result = subprocess.run(
    ['onigirazu', 'apply', 'deploy.yml', '--list-tasks', '--output', 'json'],
    capture_output=True, text=True, check=True,
)
data = json.loads(result.stdout)

for play in data['plays']:
    print(f"Play: {play['name']}")
    for task in play['tasks']:
        print(f"  - {task['name']} ({task['status']}; tags: {', '.join(task['tags'] or [])})")
```

## Best Practices

1. **List, then plan, then apply**:

   ```bash
   onigirazu apply playbook.yml --list-tasks --tags production
   onigirazu plan playbook.yml --tags production
   onigirazu apply playbook.yml --tags production
   ```

2. **Use `--output json` in CI** for a stable document to parse.

3. **Validate playbooks in advance**:

   ```bash
   onigirazu validate playbook.yml
   onigirazu apply playbook.yml --list-tags
   ```

## See Also

- [Tag Filtering Guide](TAG_FILTERING.md)
- [Handlers Guide](HANDLERS_GUIDE.md)
