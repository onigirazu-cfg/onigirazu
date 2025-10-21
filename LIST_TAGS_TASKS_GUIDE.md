# Tag and Task Discovery Guide

## Overview

Onigirazu provides powerful discovery features to help you understand and explore your playbooks before execution. The `--list-tags` and `--list-tasks` flags allow you to:

- **`--list-tags`**: Discover all available tags in a playbook
- **`--list-tasks`**: Preview which tasks would execute with current filters

These features help you plan your deployments, understand playbook structure, and verify tag configurations without running any tasks.

## Quick Examples

### List All Available Tags

```bash
# Show all tags in a playbook
onigirazu apply playbook.yml --list-tags

# Output:
# Available tags in playbook.yml:
#   - setup
#   - security
#   - packages
#   - deployment
#   - always
```

### List Tasks with Filters

```bash
# Show which tasks would run with a specific tag filter
onigirazu apply playbook.yml --list-tasks --tags setup

# Output:
# Tasks that would execute with --tags setup:
#
# Play: Web Server Setup (webservers)
#   - [setup, packages] Install Apache
#   - [setup, security] Configure SSL
#   - [always] Start Service
```

### Combine with Skip Tags

```bash
# Show tasks with multiple filters applied
onigirazu apply playbook.yml --list-tasks --tags deployment --skip-tags experimental

# Output:
# Tasks that would execute with --tags deployment --skip-tags experimental:
#
# Play: Deploy Application (production)
#   - [deployment, critical] Create backup
#   - [deployment, config] Update configuration
#   - [always] Run health check
#
# Skipped tasks:
#   - [deployment, experimental] Test new feature
```

## `--list-tags` Command

### Purpose

Discover all unique tags available in your playbook. Useful for:

- Understanding what tags a playbook supports
- Planning tag-based deployments
- Documentation and reference
- Automation script planning

### Syntax

```bash
onigirazu apply <playbook> --list-tags [--inventory <inventory>]
```

### Example Playbook

```yaml
---
name: Complete Application Deployment
plays:
  - name: Infrastructure Setup
    hosts: all
    tasks:
      - name: Install Base Packages
        package:
          name: "{{ item }}"
          state: present
        loop:
          - curl
          - git
          - vim
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

      - name: Run Tests (Dev Only)
        shell: pytest tests/
        tags: [testing, never]  # Runs only when explicitly requested

      - name: Health Check
        shell: curl -f http://localhost/health
        tags: [always]  # Always runs
```

### Running `--list-tags`

```bash
onigirazu apply deployment.yml --list-tags
```

### Output

```
Available tags in deployment.yml:

  Tag Name          Count  Details
  ─────────────────────────────────────────────────────────
  packages          2      Used in: Install Base Packages, Install Dependencies
  setup             3      Used in: Install Base Packages, Configure Firewall, Create Application User
  base              1      Used in: Install Base Packages
  security          1      Used in: Configure Firewall
  users             1      Used in: Create Application User
  deployment        2      Used in: Deploy Latest Code, Install Dependencies
  critical          1      Used in: Deploy Latest Code
  testing           1      Used in: Run Tests (Dev Only)
  always            1      Special tag: Always runs
  never             1      Special tag: Never runs by default

Summary:
  - 10 unique tags
  - 11 tasks total
  - 1 always task
  - 1 never task
  - 8 regular tagged tasks
  - 0 untagged tasks
```

## `--list-tasks` Command

### Purpose

Preview which specific tasks would execute with your current filters. Useful for:

- Verifying tag filters before running
- Understanding task execution order
- Planning partial deployments
- Debugging tag configurations

### Syntax

```bash
onigirazu apply <playbook> --list-tasks [options]

Options:
  --tags TAG1,TAG2,...      Include only tasks with these tags
  --skip-tags TAG1,TAG2,... Exclude tasks with these tags
  --check                   Run in check mode (dry-run)
  --verbose                 Show additional task details
```

### Example 1: List All Tasks

```bash
onigirazu apply deployment.yml --list-tasks
```

### Output

```
Tasks in deployment.yml:

Play 1: Infrastructure Setup (Hosts: all)
  ✓ [packages, setup, base] Install Base Packages
  ✓ [security, setup] Configure Firewall
  ✓ [setup, users] Create Application User

Play 2: Application Deployment (Hosts: webservers)
  ✓ [deployment, critical] Deploy Latest Code
  ✓ [deployment, packages] Install Dependencies
  ⊗ [testing, never] Run Tests (Dev Only) [NEVER TAG]
  ✓ [always] Health Check [ALWAYS TAG]

Summary:
  - 7 tasks total
  - 6 would execute
  - 1 skipped (never tag)
```

### Example 2: List Tasks with Tag Filter

```bash
onigirazu apply deployment.yml --list-tasks --tags setup
```

### Output

```
Tasks that would execute with --tags setup (include: setup):

Play 1: Infrastructure Setup (Hosts: all)
  ✓ [packages, setup, base] Install Base Packages
  ✓ [security, setup] Configure Firewall
  ✓ [setup, users] Create Application User

Play 2: Application Deployment (Hosts: webservers)
  ✓ [always] Health Check [ALWAYS TAG]

Summary:
  - 4 would execute
  - 3 skipped (tag not included)
  - 1 skipped (never tag)
```

### Example 3: List Tasks with Skip Tags

```bash
onigirazu apply deployment.yml --list-tasks --skip-tags never,testing
```

### Output

```
Tasks that would execute with --skip-tags never,testing (exclude: never,testing):

Play 1: Infrastructure Setup (Hosts: all)
  ✓ [packages, setup, base] Install Base Packages
  ✓ [security, setup] Configure Firewall
  ✓ [setup, users] Create Application User

Play 2: Application Deployment (Hosts: webservers)
  ✓ [deployment, critical] Deploy Latest Code
  ✓ [deployment, packages] Install Dependencies
  ✓ [always] Health Check [ALWAYS TAG]

Summary:
  - 6 would execute
  - 1 skipped (explicitly excluded)
```

### Example 4: Combined Filters

```bash
onigirazu apply deployment.yml --list-tasks --tags deployment --skip-tags never
```

### Output

```
Tasks that would execute with --tags deployment --skip-tags never:

Play 1: Infrastructure Setup (Hosts: all)
  (no tasks match filters)

Play 2: Application Deployment (Hosts: webservers)
  ✓ [deployment, critical] Deploy Latest Code
  ✓ [deployment, packages] Install Dependencies
  ✓ [always] Health Check [ALWAYS TAG]

Summary:
  - 3 would execute
  - 4 skipped (tag filter does not include)
```

## Verbose Mode

### `--list-tasks --verbose`

For more detailed information about each task:

```bash
onigirazu apply deployment.yml --list-tasks --tags setup --verbose
```

### Verbose Output

```
Tasks that would execute with --tags setup (include: setup):

Play 1: Infrastructure Setup (Hosts: all)
  ✓ [packages, setup, base] Install Base Packages
    Module: package
    Module Args: name="{{ item }}", state=present
    Loop: 3 items (curl, git, vim)
    Hosts: all
    When: (no conditions)

  ✓ [security, setup] Configure Firewall
    Module: service
    Module Args: name=ufw, state=started, enabled=true
    Hosts: all
    When: (no conditions)

  ✓ [setup, users] Create Application User
    Module: user
    Module Args: name=appuser, shell=/bin/bash
    Hosts: all
    When: (no conditions)

Play 2: Application Deployment (Hosts: webservers)
  ✓ [always] Health Check [ALWAYS TAG]
    Module: shell
    Module Args: curl -f http://localhost/health
    Hosts: webservers
    When: (no conditions)

Summary:
  - 4 would execute
  - 3 skipped (tag not included)
  - 1 skipped (never tag)
```

## Output Formats

Both `--list-tags` and `--list-tasks` support multiple output formats:

### Text Format (Default)

```bash
onigirazu apply playbook.yml --list-tags
```

### JSON Format

```bash
onigirazu apply playbook.yml --list-tags --output json
```

### YAML Format

```bash
onigirazu apply playbook.yml --list-tasks --output yaml
```

### CSV Format

```bash
onigirazu apply playbook.yml --list-tags --output csv
```

## Use Cases

### Use Case 1: Understanding a New Playbook

```bash
# First, see what tags are available
$ onigirazu apply production.yml --list-tags

# Then, see what tasks would run
$ onigirazu apply production.yml --list-tasks

# Explore specific tag groups
$ onigirazu apply production.yml --list-tasks --tags setup
$ onigirazu apply production.yml --list-tasks --tags deployment
```

### Use Case 2: Planning a Phased Deployment

```bash
# Phase 1: Infrastructure setup (dry-run first)
$ onigirazu apply deploy.yml --list-tasks --tags setup
$ onigirazu apply deploy.yml --check --tags setup

# Phase 2: Application deployment
$ onigirazu apply deploy.yml --list-tasks --tags deployment
$ onigirazu apply deploy.yml --check --tags deployment

# Phase 3: Verification
$ onigirazu apply deploy.yml --list-tasks --tags verification
$ onigirazu apply deploy.yml --check --tags verification
```

### Use Case 3: CI/CD Pipeline Debugging

```bash
# Check what would run in your CI pipeline
$ onigirazu apply ci-deploy.yml --list-tasks --tags ci,test --skip-tags experimental

# Then run it with confidence
$ onigirazu apply ci-deploy.yml --tags ci,test --skip-tags experimental
```

### Use Case 4: Documentation Generation

```bash
# Export task list for documentation
$ onigirazu apply deployment.yml --list-tasks --output json > tasks.json
$ onigirazu apply deployment.yml --list-tags --output csv > tags.csv

# Use jq to format as markdown
$ onigirazu apply deployment.yml --list-tasks --output json | jq .
```

## Integration with Other Commands

### With `--check` (Dry-Run)

```bash
# Verify tasks would run AND preview changes (no side effects)
onigirazu apply playbook.yml --check --list-tasks --tags setup
```

### With `--diff`

```bash
# Show task list with expected changes
onigirazu apply playbook.yml --diff --list-tasks --tags setup
```

### With `--verbose`

```bash
# Detailed task information
onigirazu apply playbook.yml --list-tasks --verbose --tags setup
```

## Scripting Examples

### Bash: Run Setup Only If There Are Tasks

```bash
#!/bin/bash

# Check if setup tasks would run
task_count=$(onigirazu apply deploy.yml --list-tasks --tags setup --output json | jq '.tasks | length')

if [ "$task_count" -gt 0 ]; then
    echo "Running $task_count setup tasks..."
    onigirazu apply deploy.yml --tags setup
else
    echo "No setup tasks to run"
fi
```

### Bash: Count Tasks by Tag

```bash
#!/bin/bash

# Export tag usage
onigirazu apply deploy.yml --list-tags --output json | \
  jq '.tags | to_entries | map("\(.key): \(.value)")' | \
  sort -rn

# Output:
# "deployment: 5"
# "setup: 4"
# "always: 2"
# etc.
```

### Python: Parse Task List

```python
#!/usr/bin/env python3
import json
import subprocess

# Get task list as JSON
result = subprocess.run(
    ['onigirazu', 'apply', 'deploy.yml', '--list-tasks', '--output', 'json'],
    capture_output=True,
    text=True
)

tasks = json.loads(result.stdout)

# Analyze
for play in tasks['plays']:
    print(f"Play: {play['name']}")
    for task in play['tasks']:
        print(f"  - {task['name']} (tags: {', '.join(task['tags'])})")
```

## Best Practices

1. **Always preview first**

   ```bash
   # Preview before running
   onigirazu apply playbook.yml --list-tasks --tags production
   onigirazu apply playbook.yml --check --tags production
   onigirazu apply playbook.yml --tags production
   ```

2. **Use with CI/CD**

   ```yaml
   # In your CI/CD pipeline
   - name: Preview Tasks
     run: onigirazu apply deploy.yml --list-tasks --tags $DEPLOYMENT_TAGS

   - name: Run Deployment
     run: onigirazu apply deploy.yml --tags $DEPLOYMENT_TAGS
   ```

3. **Document your tags**

   ```bash
   # Generate tag documentation
   onigirazu apply playbook.yml --list-tags --output markdown > TAGS.md
   ```

4. **Validate playbooks in advance**

   ```bash
   # Check playbook structure and tags
   onigirazu validate playbook.yml
   onigirazu apply playbook.yml --list-tags
   onigirazu apply playbook.yml --list-tasks
   ```

## See Also

- [Tag Filtering Guide](TAG_FILTERING.md) - Detailed tag filtering documentation
- [CLI Reference](../README.md#cli-commands) - All CLI commands
- [Playbook Format](PLAYBOOK.md) - How to write playbooks
- [Advanced Execution](ADVANCED_EXECUTION.md) - Advanced execution options
