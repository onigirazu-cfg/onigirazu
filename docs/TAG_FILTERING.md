# Tag Filtering Guide

Tags select which tasks run. The `--tags` and `--skip-tags` flags let you:

- Run only specific tasks
- Skip specific tasks
- Filter by tagged/untagged status
- Combine both

## Tag Definition in Playbooks

`tags` is a list on a task:

```yaml
---
- name: Web Server Setup
  hosts: all
  tasks:
    - name: Install Apache
      package:
        name: apache2
      tags: [setup, packages]

    - name: Configure SSL directory
      file:
        path: /etc/apache2/ssl
        state: directory
      tags:
        - setup
        - security

    - name: Start Service
      service:
        name: apache2
        state: started
      tags: [always]     # runs whatever the filter

    - name: Debug Info
      debug:
        msg: "Service started"
      tags: [never]      # never runs
```

`tags` is a list or a string; a string is split on commas (`tags: setup, config`).

### Tag Inheritance

Tags are passed down to the tasks inside:

| Where the tags are set | Inherited by |
|------------------------|--------------|
| play | every task of the play, its roles included |
| `roles:` entry (`- { role: web, tags: web }`) | the role's tasks and its dependencies |
| `block` | its `block`, `rescue` and `always` tasks |
| `include_tasks` / `import_tasks` | the included tasks |
| `include_role` / `import_role` | the role's tasks |

Handlers are not filtered by tags; they run when notified.

## CLI Flags

### `--tags`

Run only tasks with at least one of the given tags (comma-separated, OR logic), plus `always` tasks.

```bash
onigirazu apply playbook.yml --tags TAG1,TAG2,TAG3
```

Special values:

- **`--tags all`**: no tag filter (the default)
- **`--tags tagged`**: only tasks with at least one tag (plus `always` tasks)
- **`--tags untagged`**: only tasks with no tags (plus `always` tasks)

### `--skip-tags`

Skip tasks with any of the given tags (comma-separated).

```bash
onigirazu apply playbook.yml --skip-tags debug,test
```

## Special Tags

### `always`

Tasks tagged `always` run with any `--tags` value and any `--skip-tags` value, except `--skip-tags always`:

```bash
# 'always' tasks run:
onigirazu apply playbook.yml --tags setup
onigirazu apply playbook.yml --skip-tags debug
onigirazu apply playbook.yml --tags untagged

# 'always' tasks are skipped:
onigirazu apply playbook.yml --skip-tags always
```

### `never`

Tasks tagged `never` do not run, with any filter, including `--tags never` or `--tags` with one of their other tags. To run such a task, remove the `never` tag.

A task tagged both `always` and `never` runs (`always` is checked first).

## Evaluation Order

For each task:

1. Tagged `always`: runs, unless `--skip-tags` contains `always`
2. Tagged `never`: skipped
3. Any tag listed in `--skip-tags`: skipped
4. `--tags`: `tagged` / `untagged` / a tag list must match
5. No `--tags`: runs

Tag matching is case-insensitive (`--tags Setup` matches `setup`).

## Usage Examples

### Run only setup tasks

```bash
onigirazu apply production.yml --tags setup
```

Runs tasks tagged `setup`, plus `always` tasks.

### Run several tag groups

```bash
onigirazu apply production.yml --tags setup,deployment
```

### Skip debug tasks

```bash
onigirazu apply production.yml --skip-tags debug,test
```

### Combine filters

```bash
onigirazu apply production.yml --tags deployment --skip-tags experimental
```

Runs tasks tagged `deployment`, except those also tagged `experimental`.

### Only tagged / only untagged

```bash
onigirazu apply production.yml --tags tagged
onigirazu apply production.yml --tags untagged
```

### Check mode with tags

```bash
onigirazu apply production.yml --check --tags setup
```

## Real-World Scenarios

### Multi-environment deployment

```yaml
tasks:
  - name: Update apt cache
    apt:
      update_cache: true
    tags: [always]

  - name: Configure development
    template: src=dev.conf.j2 dest=/etc/app/app.conf
    tags: [dev, configuration]

  - name: Configure production
    template: src=prod.conf.j2 dest=/etc/app/app.conf
    tags: [prod, configuration]
```

```bash
onigirazu apply deploy.yml --tags dev
onigirazu apply deploy.yml --tags prod
```

### Testing and validation

```yaml
tasks:
  - name: Install application
    package: name=myapp state=present
    tags: [install]

  - name: Run unit tests
    shell: cd /opt/myapp && make test
    tags: [test, unit]

  - name: Run integration tests
    shell: cd /opt/myapp && make integration
    tags: [test, integration]
```

```bash
onigirazu apply test.yml --tags install          # install only
onigirazu apply test.yml --tags install,test     # install and all tests
onigirazu apply test.yml --tags install,unit     # install and unit tests
onigirazu apply test.yml --skip-tags test        # everything except tests
```

For a task that should run only on request, give it its own tag and skip that tag by default (`--skip-tags benchmark`) instead of using `never`.

## Discovering Tags and Tasks

```bash
# Tags used in the playbook, with counts
onigirazu apply playbook.yml --list-tags

# Tasks and whether they would run with the given filters
onigirazu apply playbook.yml --list-tasks --tags setup --skip-tags debug
```

**Current limitation**: `--list-tags` and `--list-tasks` do not look inside roles or blocks, list handlers as tasks, and `--list-tasks` does not apply all execution rules (see [List Tags and Tasks Guide](LIST_TAGS_TASKS_GUIDE.md)). Use `plan` for an exact preview.

## Commands Supporting Tag Filtering

### apply

```bash
onigirazu apply playbook.yml --tags setup,config --skip-tags debug
```

### plan

Runs the playbook in check mode against the hosts with the same tag filter:

```bash
onigirazu plan playbook.yml --tags setup --skip-tags experimental
```

### drift

```bash
onigirazu drift playbook.yml --tags setup
```

**Current limitation**: `graph` accepts `--tags` and `--skip-tags` but ignores them.

## Troubleshooting

### Task not running

1. Does it have the `never` tag? It never runs; remove the tag.
2. Does `--tags untagged` apply while the task has tags (also inherited from the play or role)?
3. Is one of its tags, or one inherited, in `--skip-tags`?
4. With `--tags`, does at least one of its tags match?

### Task running when it shouldn't

1. Does it have the `always` tag? Skip it with `--skip-tags always`.
2. Is it untagged while no `--tags` is given? Untagged tasks run by default.
3. Does it inherit a matching tag from a block, include or `include_role`?

## Best Practices

1. **Use descriptive tag names**: `setup`, `security`, `deploy`.
2. **Always write tags as a list**, even for one tag: `tags: [setup]`.
3. **Put shared tags on a block** instead of repeating them on each task.
4. **Use `always` for tasks every run needs**, such as gathering data used later.
5. **Preview with `plan`** before applying a filtered run.

## Implementation Details

- The tag filter is applied before `when` is evaluated.
- Tasks filtered out count as skipped in the run statistics.
- A looped task is filtered as a whole.

## See Also

- [List Tags and Tasks Guide](LIST_TAGS_TASKS_GUIDE.md)
- [Handlers Guide](HANDLERS_GUIDE.md)
