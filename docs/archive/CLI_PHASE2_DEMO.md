# Phase 2: CLI Commands Demo

This document demonstrates the new CLI commands added in Phase 2.

## Overview

Phase 2 adds four new commands to Onigirazu:

- `validate` - Validate playbook syntax and structure
- `plan` - Show execution plan without making changes
- `state list` - List all resources in state
- `state show` - Show detailed information about a resource

## Commands

### 1. Validate Command

Validates playbook syntax and structure without executing it.

```bash
# Basic validation
./onigirazu validate test-cli-demo.yml

# Verbose validation with playbook structure
./onigirazu validate test-cli-demo.yml --verbose

# Strict validation mode
./onigirazu validate test-cli-demo.yml --strict

# Skip module checking
./onigirazu validate test-cli-demo.yml --check-modules=false
```

**Example Output:**

```
✅ Playbook validation successful!

Playbook: CLI Demo Playbook
Plays:    2
Tasks:    7
Duration: 3ms
```

**Verbose Output:**

```
🔍 Validating playbook: test-cli-demo.yml
📦 Checking module availability...
✅ Playbook validation successful!

Playbook: CLI Demo Playbook
Plays:    2
Tasks:    7
Duration: 0s

📋 Playbook structure:
  Play 1: Configure Web Servers
    Hosts: webservers
    Tasks: 4
      1. Install nginx (module: package)
      2. Start nginx service (module: service)
      3. Copy nginx config (module: copy)
      4. Create web directory (module: file)
  Play 2: Configure Database Servers
    Hosts: dbservers
    Tasks: 3
      1. Install PostgreSQL (module: package)
      2. Start PostgreSQL service (module: service)
      3. Create database user (module: command)
```

### 2. Plan Command

Shows what changes would be made without executing them (dry-run with detailed output).

```bash
# Basic plan
./onigirazu plan test-cli-demo.yml

# Detailed plan with task arguments
./onigirazu plan test-cli-demo.yml --detailed

# Plan with inventory
./onigirazu plan test-cli-demo.yml --inventory hosts.yml

# Plan for specific hosts
./onigirazu plan test-cli-demo.yml --limit "web*"
```

**Example Output:**

```
📋 Generating execution plan for: test-cli-demo.yml

═══════════════════════════════════════════════════════════
Playbook: CLI Demo Playbook
Plays:    2
═══════════════════════════════════════════════════════════

Play 1: Configure Web Servers
  Target hosts: webservers
  Tasks: 4

  Task 1: Install nginx
    Module: package
    Action: 📦 Install/update package

  Task 2: Start nginx service
    Module: service
    Action: 🔧 Manage service

  Task 3: Copy nginx config
    Module: copy
    Action: 📋 Copy file

  Task 4: Create web directory
    Module: file
    Action: 📄 Manage file

───────────────────────────────────────────────────────────

═══════════════════════════════════════════════════════════
Plan Summary:
  Total tasks: 7
  Total plays: 2

⚠️  This is a plan only. No changes will be made.
    Run 'onigirazu apply' to execute the playbook.
═══════════════════════════════════════════════════════════
```

**Detailed Output** (with `--detailed` flag):
Shows full task arguments for each task.

### 3. State List Command

Lists all resources tracked in the state file.

```bash
# List all resources
./onigirazu state list --state test-state.json

# List with verbose output (shows host details)
./onigirazu state list --state test-state.json --verbose

# List in JSON format
./onigirazu state list --state test-state.json --output json

# List in YAML format
./onigirazu state list --state test-state.json --output yaml

# Filter by play name (wildcard support)
./onigirazu state list --state test-state.json --filter "Configure*"

# Filter by substring
./onigirazu state list --state test-state.json --filter "database"
```

**Example Output:**

```
═══════════════════════════════════════════════════════════
State File Resources
Last Run: 2024-01-15T10:34:00Z
Playbook: test-cli-demo.yml
═══════════════════════════════════════════════════════════

✅ Play: Configure Web Servers
   Duration: 2m15s
   Hosts: 2

✅ Play: Configure Database Servers
   Duration: 1m45s
   Hosts: 1

───────────────────────────────────────────────────────────
Total: 2 plays, 3 hosts, 11 tasks
═══════════════════════════════════════════════════════════
```

**Verbose Output:**

```
✅ Play: Configure Web Servers
   Duration: 2m15s
   Hosts: 2
   ✅ Host: web1.example.com
      Tasks: 4
      Changed: 3, Failed: 0, Skipped: 0
   ✅ Host: web2.example.com
      Tasks: 4
      Changed: 3, Failed: 0, Skipped: 0
```

### 4. State Show Command

Shows detailed information about a specific resource (play or host).

```bash
# Show play details
./onigirazu state show "Configure Web Servers" --state test-state.json

# Show host details
./onigirazu state show "web1.example.com" --state test-state.json

# Show in JSON format
./onigirazu state show "web1.example.com" --state test-state.json --output json

# Show in YAML format
./onigirazu state show "web1.example.com" --state test-state.json --output yaml
```

**Example Output (Play):**

```
═══════════════════════════════════════════════════════════
Play: Configure Web Servers
Status: Success
Start: 2024-01-15T10:30:00Z
End: 2024-01-15T10:32:15Z
Duration: 2m15s
Hosts: 2
═══════════════════════════════════════════════════════════

✅ Host: web1.example.com
   Tasks: 4
   Changed: 3, Failed: 0, Skipped: 0

✅ Host: web2.example.com
   Tasks: 4
   Changed: 3, Failed: 0, Skipped: 0
```

**Example Output (Host):**

```
═══════════════════════════════════════════════════════════
Host: web1.example.com
Status: Success
Tasks: 4
═══════════════════════════════════════════════════════════

🔄 Task 1: Install nginx
   Module: package
   Duration: 45s
   Changed: Yes
   Output: Package nginx installed successfully

🔄 Task 2: Start nginx service
   Module: service
   Duration: 15s
   Changed: Yes
   Output: Service nginx started

✅ Task 3: Copy nginx config
   Module: copy
   Duration: 5s
   Output: File already exists with same content

🔄 Task 4: Create web directory
   Module: file
   Duration: 5s
   Changed: Yes
   Output: Directory created
```

## Status Icons

The commands use Unicode icons for visual clarity:

- ✅ Success
- ❌ Failed
- 🔄 Changed
- ⏭️ Skipped
- 📦 Package module
- 🔧 Service module
- 📋 Copy module
- 📄 File module
- ⚡ Command/Shell module
- 👤 User module
- 👥 Group module
- 🔐 Security module

## Output Formats

All state commands support multiple output formats:

- `table` (default) - Human-readable table format
- `json` - JSON format for programmatic processing
- `yaml` - YAML format (currently outputs JSON)

## Global Flags

All commands support these global flags:

- `-c, --config` - Path to configuration file
- `-i, --inventory` - Path to inventory file
- `-s, --state` - Path to state file (default: `.onigirazu-state`)
- `-v, --verbose` - Verbose output
- `--no-color` - Disable colored output

## Files Created for Testing

- `test-cli-demo.yml` - Demo playbook with 2 plays and 7 tasks
- `test-state.json` - Demo state file with execution results

## Implementation Details

### Files Created

- `internal/cli/validate.go` - Validate command implementation
- `internal/cli/plan.go` - Plan command implementation
- `internal/cli/state.go` - State command with list/show subcommands

### Files Modified

- `internal/cli/root.go` - Added command registrations

### Key Features

1. **Comprehensive validation** - Checks YAML syntax, playbook structure, and module names
2. **Detailed planning** - Shows execution plan with task-by-task breakdown
3. **State inspection** - List and show resources with filtering support
4. **Multiple output formats** - Table, JSON, and YAML support
5. **Rich output** - Unicode icons and formatted tables for better UX
6. **Error handling** - Clear error messages with helpful guidance

## Next Steps

Potential enhancements for Phase 3:

- Add `state rm` command to remove resources from state
- Add `state mv` command to rename resources
- Add `state pull` command to refresh state from remote
- Add `state push` command to save state to remote
- Add `diff` command to compare playbook with current state
- Add `graph` command to visualize playbook dependencies
- Add `init` command to initialize new project
- Add `fmt` command to format playbooks
- Add `lint` command for advanced playbook linting
