# Tag and Task Discovery - Quick Reference

## At a Glance

### `--list-tags`: See Available Tags

```bash
onigirazu apply playbook.yml --list-tags
```

**Output**: All tags used in playbook, with usage count

### `--list-tasks`: See What Would Run

```bash
onigirazu apply playbook.yml --list-tasks
```

**Output**: All tasks that would execute, marked with ✓ or ✗

## Common Commands

| Task | Command |
|------|---------|
| **List all tags** | `onigirazu apply playbook.yml --list-tags` |
| **List all tasks** | `onigirazu apply playbook.yml --list-tasks` |
| **List tasks for one tag** | `onigirazu apply playbook.yml --list-tasks --tags setup` |
| **List tasks excluding tags** | `onigirazu apply playbook.yml --list-tasks --skip-tags debug` |
| **Combine tag filters** | `onigirazu apply playbook.yml --list-tasks --tags setup --skip-tags test` |
| **Detailed task info** | `onigirazu apply playbook.yml --list-tasks --verbose` |
| **Export as JSON** | `onigirazu apply playbook.yml --list-tasks --output json` |
| **Export as CSV** | `onigirazu apply playbook.yml --list-tasks --output csv` |

## Output Symbols

### `--list-tasks` Output

| Symbol | Meaning |
|--------|---------|
| ✓ | Task WILL execute |
| ✗ | Task WILL NOT execute (skipped) |
| `[tag1, tag2]` | Tags on this task |
| `[ALWAYS TAG]` | Task always runs (unless explicitly skipped) |
| `[NEVER TAG]` | Task never runs by default |

### Why Tasks Are Skipped

| Reason | Meaning |
|--------|---------|
| `never` | Task has `never` tag |
| `tag not included` | Task's tags don't match `--tags` filter |
| `tag explicitly excluded` | Task's tag is in `--skip-tags` |
| `untagged` | Task has no tags and `--tags tagged` used |

## Tag Filter Syntax

```bash
# Single tag
--tags setup

# Multiple tags (OR logic - any match runs)
--tags setup,deployment,test

# Special filters
--tags all           # All tasks except "never"
--tags tagged        # Only tagged tasks
--tags untagged      # Only untagged tasks + always tasks

# Exclude tags
--skip-tags debug,test

# Combine filters
--tags production --skip-tags experimental
```

## Output Formats

### Default (Text, Human-Readable)

```bash
onigirazu apply playbook.yml --list-tags
```

### JSON (Machine-Readable, Great for Scripting)

```bash
onigirazu apply playbook.yml --list-tags --output json | jq .
```

### CSV (Spreadsheet Import)

```bash
onigirazu apply playbook.yml --list-tasks --output csv > tasks.csv
```

### YAML (Structured, Readable)

```bash
onigirazu apply playbook.yml --list-tasks --output yaml
```

## Real-World Examples

### Example 1: Explore a New Playbook

```bash
# Step 1: What tags does it have?
$ onigirazu apply deploy.yml --list-tags

# Step 2: What tasks are there?
$ onigirazu apply deploy.yml --list-tasks

# Step 3: Which tasks would run for staging?
$ onigirazu apply deploy.yml --list-tasks --tags staging

# Step 4: Preview the actual execution
$ onigirazu apply deploy.yml --check --tags staging
```

### Example 2: Planning a CI/CD Deployment

```bash
# Step 1: Check what CI tasks exist
$ onigirazu apply ci.yml --list-tasks --tags ci

# Step 2: Preview changes
$ onigirazu apply ci.yml --check --tags ci

# Step 3: Run it
$ onigirazu apply ci.yml --tags ci
```

### Example 3: Documentation Generation

```bash
# Generate tag reference
$ onigirazu apply playbook.yml --list-tags --output json > tags.json

# Generate task reference
$ onigirazu apply playbook.yml --list-tasks --output csv > tasks.csv

# Use in documentation
$ cat tags.json | jq '.tags | keys' > TAGS.txt
```

### Example 4: Debugging Tag Configuration

```bash
# I want to run "security" and "setup" tasks
# But skip "debug" tasks
$ onigirazu apply playbook.yml --list-tasks \
    --tags security,setup \
    --skip-tags debug \
    --verbose

# Then actually run it
$ onigirazu apply playbook.yml \
    --tags security,setup \
    --skip-tags debug
```

## Combining with Other Flags

```bash
# Preview with verbose output
onigirazu apply playbook.yml --list-tasks --verbose

# Preview with specific inventory
onigirazu apply playbook.yml --list-tasks -i inventory.yml

# List tasks for a tag with dry-run to see changes
onigirazu apply playbook.yml --check --list-tasks --tags setup

# Export filtered task list
onigirazu apply playbook.yml --list-tasks --tags production --output json > prod-tasks.json
```

## Tips & Tricks

### Tip 1: Use with `grep` for Quick Filtering

```bash
# Find all tasks with "install" in name
$ onigirazu apply playbook.yml --list-tasks | grep -i install
```

### Tip 2: Count Tasks by Tag

```bash
# Export to JSON and count
$ onigirazu apply playbook.yml --list-tags --output json | \
  jq '.tags | to_entries | map(.value) | length'
```

### Tip 3: Generate Playbook Documentation

```bash
# Create TAGS.md with all tags
$ onigirazu apply playbook.yml --list-tags --output markdown > TAGS.md

# Create TASKS.md with all tasks
$ onigirazu apply playbook.yml --list-tasks --output markdown > TASKS.md
```

### Tip 4: Verify Tag Configuration Before Running

```bash
# Always preview first!
$ onigirazu apply playbook.yml --list-tasks --tags prod --skip-tags experimental
# Then run it
$ onigirazu apply playbook.yml --tags prod --skip-tags experimental
```

### Tip 5: Extract Task Count

```bash
# How many tasks would run?
$ onigirazu apply playbook.yml --list-tasks --tags setup --output json | \
  jq '.summary.executing'
```

## Troubleshooting

### Problem: "No tasks match my filter"

**Solution**: Preview what would run:

```bash
$ onigirazu apply playbook.yml --list-tasks --tags mytag
# If empty, your tag doesn't exist or is misspelled
```

**Check available tags**:

```bash
onigirazu apply playbook.yml --list-tags
```

### Problem: Task runs but shouldn't

**Solution**: Check task tags:

```bash
$ onigirazu apply playbook.yml --list-tasks --verbose
# Look for [always] tag or mismatched tags
```

### Problem: Task doesn't run when it should

**Solution**: Check tag filters:

```bash
$ onigirazu apply playbook.yml --list-tasks \
    --tags mytag \
    --skip-tags excluded \
    --verbose
# Verify the task appears with ✓
```

## Integration with Scripting

### Bash: Count Total Tasks

```bash
#!/bin/bash
count=$(onigirazu apply playbook.yml --list-tasks --output json | \
  jq '.summary.total')
echo "Total tasks: $count"
```

### Python: Parse Task List

```python
import json
import subprocess

result = subprocess.run(
    ['onigirazu', 'apply', 'playbook.yml',
     '--list-tasks', '--output', 'json'],
    capture_output=True,
    text=True
)
tasks = json.loads(result.stdout)
print(f"Would execute: {tasks['summary']['executing']} tasks")
```

### CI/CD: Verify Deployment Plan

```yaml
# GitHub Actions example
- name: Preview deployment
  run: |
    onigirazu apply deploy.yml \
      --list-tasks \
      --tags ${{ env.DEPLOYMENT_TAGS }} \
      --output json > deployment-plan.json

- name: Check plan is not empty
  run: |
    tasks=$(jq '.summary.executing' deployment-plan.json)
    if [ $tasks -eq 0 ]; then
      echo "Error: No tasks would execute!"
      exit 1
    fi
```

## Performance Notes

- **`--list-tags`**: Fast - just parses playbook
- **`--list-tasks`**: Fast - parses playbook + applies filters
- Large playbooks (1000+ tasks): Still milliseconds
- Output to file is faster than piping to terminal

## See Also

- Full guide: [List Tags and Tasks Guide](LIST_TAGS_TASKS_GUIDE.md)
- Tag filtering: [Tag Filtering Guide](TAG_FILTERING.md)
- Implementation: [Implementation Guide](LIST_TAGS_TASKS_IMPLEMENTATION.md)
