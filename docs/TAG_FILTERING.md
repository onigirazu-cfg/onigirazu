# Tag Filtering Guide

Onigirazu supports comprehensive tag-based filtering for tasks, allowing you to selectively run, skip, or filter tasks during playbook execution. This guide covers all tag filtering modes and use cases.

## Overview

Tags are metadata labels attached to tasks that enable flexible execution control. The `--tags` and `--skip-tags` flags allow you to:

- Run only specific tasks
- Skip specific tasks
- Filter by tagged/untagged status
- Combine filters for complex scenarios

## Tag Definition in Playbooks

Define tags on tasks in your playbook YAML:

```yaml
---
name: Configure Web Server
plays:
  - name: Web Server Setup
    hosts: all
    tasks:
      - name: Install Apache
        module: package
        args:
          name: apache2
        tags:
          - setup
          - packages

      - name: Configure SSL
        module: file
        args:
          path: /etc/apache2/ssl
        tags:
          - setup
          - security

      - name: Start Service
        module: service
        args:
          name: apache2
        tags:
          - always  # This task always runs

      - name: Debug Info
        module: debug
        args:
          msg: "Service started"
        tags:
          - never  # This task never runs
```

## CLI Flags

### `--tags` Flag

Run only tasks with specified tags. Multiple tags are comma-separated and operate with **OR** logic (any match runs the task).

#### Syntax

```bash
onigirazu apply playbook.yml --tags TAG1,TAG2,TAG3
```

#### Special Tag Values

- **`--tags all`** - Run all tasks except those with `never` tag (default behavior)
- **`--tags tagged`** - Run only tasks with at least one tag
- **`--tags untagged`** - Run only tasks with no tags (plus `always` tasks)
- **`--tags TAG1,TAG2`** - Run tasks with either TAG1 or TAG2 (plus `always` tasks)

### `--skip-tags` Flag

Skip tasks with specified tags. Multiple tags are comma-separated and operate with **OR** logic (any match skips the task).

#### Syntax

```bash
onigirazu apply playbook.yml --skip-tags DEBUG,TEST
```

## Special Tags

### `always` Tag

Tasks with the `always` tag run regardless of other tag filters:

```bash
# These tasks with 'always' tag will run:
onigirazu apply playbook.yml --tags setup
onigirazu apply playbook.yml --skip-tags debug
onigirazu apply playbook.yml --tags untagged
```

**Exception**: If you explicitly skip the `always` tag, it won't run.

```bash
# This will skip all tasks including 'always' tasks
onigirazu apply playbook.yml --skip-tags always
```

### `never` Tag

Tasks with the `never` tag never run regardless of other tag filters:

```bash
# These tasks with 'never' tag will NOT run:
onigirazu apply playbook.yml --tags setup
onigirazu apply playbook.yml --tags tagged
onigirazu apply playbook.yml               # default
```

## Usage Examples

### Example 1: Run Only Setup Tasks

```bash
onigirazu apply production.yml --tags setup
```

This runs only tasks tagged with `setup`, plus any tasks with `always` tag.

### Example 2: Run Multiple Tag Groups

```bash
onigirazu apply production.yml --tags setup,deployment
```

Runs tasks tagged with either `setup` OR `deployment`, plus `always` tasks.

### Example 3: Skip Debug Tasks

```bash
onigirazu apply production.yml --skip-tags debug,test
```

Runs all tasks EXCEPT those tagged with `debug` or `test`.

### Example 4: Combined Filters

```bash
onigirazu apply production.yml --tags deployment --skip-tags experimental
```

Runs tasks tagged with `deployment`, but skips those also tagged with `experimental`.

### Example 5: Run Only Tagged Tasks

```bash
onigirazu apply production.yml --tags tagged
```

Runs only tasks that have at least one tag (but still respects the `always` and `never` tags).

### Example 6: Run Only Untagged Tasks

```bash
onigirazu apply production.yml --tags untagged
```

Runs only tasks with no tags, plus tasks with the `always` tag.

### Example 7: Dry Run with Tags

```bash
onigirazu apply production.yml --check --tags setup
```

Preview what would happen if you ran only setup tasks.

### Example 8: Production Deployment

```bash
onigirazu apply playbook.yml \
  --tags production,deployment \
  --skip-tags experimental,debug
```

Run production and deployment tasks, but skip anything experimental or for debugging.

## Tag Filtering Priority

The tag filtering logic follows this priority:

1. **`never` tag** - Tasks with this tag NEVER run (highest priority)
2. **`always` tag** - Tasks with this tag ALWAYS run (unless explicitly skipped)
3. **`--skip-tags`** - Explicitly excluded tags
4. **`--tags` filter** - Tags to include (if specified)
5. **Default** - All tasks run (if no filters specified)

## Real-World Scenarios

### CI/CD Pipeline

```bash
# Development environment - run everything except production steps
onigirazu apply deploy.yml --skip-tags production

# Staging environment - run staging and always tasks
onigirazu apply deploy.yml --tags staging,always

# Production environment - only production and critical tasks
onigirazu apply deploy.yml --tags production,critical --skip-tags experimental
```

### Multi-Environment Deployment

```yaml
tasks:
  - name: Update System
    module: apt
    tags: [always, update]

  - name: Configure Development
    module: file
    tags: [dev, configuration]

  - name: Configure Production
    module: file
    tags: [prod, configuration]

  - name: Performance Test
    module: shell
    tags: [performance, never]  # Skip by default
```

Run for specific environment:

```bash
# Development
onigirazu apply deploy.yml --tags dev --skip-tags prod

# Production
onigirazu apply deploy.yml --tags prod --skip-tags dev

# Both (all configurations)
onigirazu apply deploy.yml --tags dev,prod
```

### Testing and Validation

```yaml
tasks:
  - name: Install Application
    module: package
    tags: [install]

  - name: Run Unit Tests
    module: shell
    tags: [test, validation]

  - name: Run Integration Tests
    module: shell
    tags: [test, integration]

  - name: Performance Benchmarks
    module: shell
    tags: [benchmark, never]  # Manual only
```

Run commands:

```bash
# Install only
onigirazu apply test.yml --tags install

# Install and run all tests
onigirazu apply test.yml --tags install,test

# Install and run only unit tests
onigirazu apply test.yml --tags install,validation

# Skip tests
onigirazu apply test.yml --skip-tags test

# Manual benchmark run (normally skipped)
onigirazu apply test.yml --tags benchmark
```

## Discovering Tags and Tasks

### List Available Tags

Before creating tag filters, discover what tags are available in your playbook:

```bash
# Show all tags in a playbook
onigirazu apply playbook.yml --list-tags

# Output:
# Available tags in playbook.yml:
#   - setup (used in 3 tasks)
#   - security (used in 2 tasks)
#   - deployment (used in 4 tasks)
#   - debug (used in 1 task)
#   - always (used in 1 task)
#   - never (used in 1 task)
```

### List Tasks Before Execution

Preview which tasks would run with your current tag filters:

```bash
# Show all tasks that would execute
onigirazu apply playbook.yml --list-tasks

# Show tasks with specific filters
onigirazu apply playbook.yml --list-tasks --tags setup
onigirazu apply playbook.yml --list-tasks --tags setup --skip-tags debug

# Verbose output with more details
onigirazu apply playbook.yml --list-tasks --tags setup --verbose
```

📚 **For detailed discovery features, see [List Tags and Tasks Guide](LIST_TAGS_TASKS_GUIDE.md)**

## Commands Supporting Tag Filtering

### apply

Execute playbook with tag filtering:

```bash
onigirazu apply playbook.yml --tags setup,config --skip-tags debug
```

### plan

Preview execution plan with tag filtering:

```bash
onigirazu plan playbook.yml --tags setup --skip-tags experimental
```

### graph

Generate graph with filtered tasks:

```bash
onigirazu graph playbook.yml --tags setup --format=mermaid
```

## Case-Insensitive Tag Matching

Tag matching is case-insensitive for convenience:

```bash
# These are all equivalent:
onigirazu apply playbook.yml --tags Setup
onigirazu apply playbook.yml --tags SETUP
onigirazu apply playbook.yml --tags setup
```

## Troubleshooting

### Task Not Running

If a task isn't running, check:

1. Does it have a `never` tag? → Remove the tag or use `--tags all`
2. Does it have tags but you're using `--tags untagged`? → Task has tags, won't run
3. Is its tag excluded by `--skip-tags`? → Remove from skip-tags
4. Do all its tags match your `--tags` filter? → At least one tag must match

### Task Running When It Shouldn't

If a task is running unexpectedly:

1. Does it have `always` tag? → Explicitly skip with `--skip-tags always`
2. Is it untagged? → Use `--tags tagged` to skip untagged tasks
3. Does it match a tag in your filter? → Modify your `--tags` filter

## Best Practices

1. **Use descriptive tag names**: `setup`, `security`, `performance` instead of `a`, `b`, `c`

2. **Group related tags**: Group tags that should run together

   ```yaml
   tags: [security, ssl, production]  # All security-related
   ```

3. **Use `always` for critical tasks**: Mark tasks that must always run

   ```yaml
   - name: Health Check
     tags: [always]
   ```

4. **Use `never` for dangerous/debug tasks**: Mark tasks that should only run manually

   ```yaml
   - name: Database Wipe
     tags: [never, dangerous]
   ```

5. **Document your tags**: Keep a README with tag meanings

6. **Combine with `--check`**: Always preview with `--check` first

   ```bash
   onigirazu apply --check --tags production
   onigirazu apply --tags production
   ```

## Implementation Details

- Tag filtering is applied **before** task condition evaluation
- Tag filtering is applied **before** task execution
- Skipped tasks are logged with the reason
- Tag filtering works with all task types (serial, parallel, loops)
- Tag filtering respects all special tags (`always`, `never`)

## See Also

- [Playbook Guide](PLAYBOOK.md)
- [Task Configuration](TASKS.md)
- [Advanced Execution](ADVANCED_EXECUTION.md)
