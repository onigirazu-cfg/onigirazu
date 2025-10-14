# 📚 API Reference

This document provides comprehensive API reference for Onigirazu, including CLI commands, modules, and programmatic interfaces.

## 📋 CLI Commands

### Global Commands

#### `onigirazu --version`
Show version information.

```bash
onigirazu --version
# Output: Onigirazu v1.26.1
```

#### `onigirazu --help`
Show help information.

```bash
onigirazu --help
```

#### `onigirazu --list-modules`
List available modules.

```bash
onigirazu --list-modules
```

#### `onigirazu --list-plugins`
List loaded plugins.

```bash
onigirazu --list-plugins
```

---

### Run Command

#### `onigirazu run [host-pattern] [command] [flags]`
Execute ad-hoc commands on target hosts.

**Syntax:**
```bash
onigirazu run <host-pattern> <command> [flags]
```

**Examples:**
```bash
# Natural language
onigirazu run all "install nginx package" -i inventory.yml

# Ansible-like syntax
onigirazu run webservers -m package name=nginx state=present -i inventory.yml

# Module:args syntax
onigirazu run all "package:name=nginx,state=present" -i inventory.yml
```

**Flags:**
- `-i, --inventory` - Inventory file path
- `-m, --module` - Module name
- `-a, --args` - Module arguments
- `--check` - Check mode (dry-run)
- `--diff` - Show differences
- `--timeout` - Execution timeout
- `--parallel` - Parallel execution count
- `--output` - Output format (text, json, yaml, table)
- `-V, --verbose` - Verbose output
- `--no-color` - Disable colored output

---

### Apply Command

#### `onigirazu apply [playbook] [flags]`
Execute playbooks on target hosts.

**Syntax:**
```bash
onigirazu apply <playbook> [flags]
```

**Examples:**
```bash
# Basic playbook execution
onigirazu apply playbook.yml -i inventory.yml

# With options
onigirazu apply playbook.yml -i inventory.yml --check --diff

# Specific hosts
onigirazu apply playbook.yml -i inventory.yml --limit webservers
```

**Flags:**
- `-i, --inventory` - Inventory file path
- `--check` - Check mode (dry-run)
- `--diff` - Show differences
- `--limit` - Limit to specific hosts
- `--tags` - Run only tagged tasks
- `--skip-tags` - Skip tagged tasks
- `--timeout` - Execution timeout
- `--parallel` - Parallel execution count
- `--output` - Output format
- `-V, --verbose` - Verbose output

---

### State Commands

#### `onigirazu state show [flags]`
Show current state information.

```bash
onigirazu state show
onigirazu state show --host web1
onigirazu state show --format json
```

#### `onigirazu state history [flags]`
Show execution history.

```bash
onigirazu state history
onigirazu state history --since 2024-01-01
onigirazu state history --host web1
```

#### `onigirazu state snapshot [command] [flags]`
Manage state snapshots.

```bash
# Create snapshot
onigirazu state snapshot create --name "backup"

# List snapshots
onigirazu state snapshot list

# Show snapshot
onigirazu state snapshot show --name "backup"
```

#### `onigirazu state rollback [flags]`
Rollback to previous state.

```bash
# Rollback to snapshot
onigirazu state rollback --snapshot "backup"

# Rollback to execution
onigirazu state rollback --execution-id 123
```

---

## 📦 Module Reference

### System Modules

#### Package Module
Manages system packages.

**Parameters:**
- `name` (string, required) - Package name
- `state` (string) - Package state: present, absent, latest
- `update_cache` (boolean) - Update package cache

**Examples:**
```yaml
# Install package
- name: Install nginx
  package:
    name: nginx
    state: present

# Remove package
- name: Remove apache
  package:
    name: apache2
    state: absent

# Update package
- name: Update nginx
  package:
    name: nginx
    state: latest
```

#### Service Module
Manages system services.

**Parameters:**
- `name` (string, required) - Service name
- `state` (string) - Service state: started, stopped, restarted, reloaded
- `enabled` (boolean) - Enable service at boot

**Examples:**
```yaml
# Start service
- name: Start nginx
  service:
    name: nginx
    state: started

# Stop service
- name: Stop apache
  service:
    name: apache2
    state: stopped

# Restart service
- name: Restart nginx
  service:
    name: nginx
    state: restarted
```

#### User Module
Manages system users.

**Parameters:**
- `name` (string, required) - Username
- `state` (string) - User state: present, absent
- `shell` (string) - User shell
- `home` (string) - Home directory
- `groups` (list) - User groups

**Examples:**
```yaml
# Create user
- name: Create user
  user:
    name: john
    state: present
    shell: /bin/bash
    home: /home/john

# Remove user
- name: Remove user
  user:
    name: olduser
    state: absent
    remove: true
```

#### Group Module
Manages system groups.

**Parameters:**
- `name` (string, required) - Group name
- `state` (string) - Group state: present, absent
- `gid` (integer) - Group ID

**Examples:**
```yaml
# Create group
- name: Create group
  group:
    name: developers
    state: present

# Remove group
- name: Remove group
  group:
    name: oldgroup
    state: absent
```

---

### File Modules

#### File Module
Manages files and directories.

**Parameters:**
- `path` (string, required) - File/directory path
- `state` (string) - File state: file, directory, link, absent, touch
- `mode` (string) - File permissions
- `owner` (string) - File owner
- `group` (string) - File group

**Examples:**
```yaml
# Create file
- name: Create file
  file:
    path: /tmp/test.txt
    state: touch

# Create directory
- name: Create directory
  file:
    path: /var/www
    state: directory
    mode: '0755'

# Delete file
- name: Delete file
  file:
    path: /tmp/old.txt
    state: absent
```

#### Copy Module
Copies files to remote hosts.

**Parameters:**
- `src` (string, required) - Source file
- `dest` (string, required) - Destination path
- `mode` (string) - File permissions
- `backup` (boolean) - Create backup
- `force` (boolean) - Force copy

**Examples:**
```yaml
# Copy file
- name: Copy file
  copy:
    src: /local/file.txt
    dest: /remote/file.txt
    mode: '0644'

# Copy with backup
- name: Copy with backup
  copy:
    src: /local/config.conf
    dest: /etc/config.conf
    backup: true
```

#### Template Module
Processes templates with variables.

**Parameters:**
- `src` (string, required) - Template source
- `dest` (string, required) - Destination path
- `mode` (string) - File permissions
- `backup` (boolean) - Create backup

**Examples:**
```yaml
# Process template
- name: Process template
  template:
    src: nginx.conf.j2
    dest: /etc/nginx/nginx.conf
    mode: '0644'
    backup: true
```

---

### Network Modules

#### Firewall Module
Manages firewall rules.

**Parameters:**
- `port` (integer, required) - Port number
- `protocol` (string) - Protocol: tcp, udp
- `state` (string) - Rule state: present, absent
- `source` (string) - Source IP/network
- `destination` (string) - Destination IP/network

**Examples:**
```yaml
# Allow port
- name: Allow HTTP
  firewall:
    port: 80
    protocol: tcp
    state: present

# Deny port
- name: Deny SSH
  firewall:
    port: 22
    protocol: tcp
    state: absent
```

#### Port Module
Manages port configurations.

**Parameters:**
- `port` (integer, required) - Port number
- `state` (string) - Port state: open, closed
- `protocol` (string) - Protocol: tcp, udp

**Examples:**
```yaml
# Open port
- name: Open port
  port:
    port: 8080
    state: open

# Close port
- name: Close port
  port:
    port: 8080
    state: closed
```

---

### Execution Modules

#### Command Module
Executes commands on remote hosts.

**Parameters:**
- `command` (string, required) - Command to execute
- `args` (string) - Command arguments
- `creates` (string) - File that command creates
- `removes` (string) - File that command removes

**Examples:**
```yaml
# Execute command
- name: Check uptime
  command: uptime

# Execute with arguments
- name: List files
  command: ls -la /tmp
```

#### Shell Module
Executes shell commands with pipes and redirection.

**Parameters:**
- `shell` (string, required) - Shell command
- `creates` (string) - File that command creates
- `removes` (string) - File that command removes

**Examples:**
```yaml
# Shell command
- name: Count processes
  shell: ps aux | grep nginx | wc -l

# Complex shell command
- name: Complex command
  shell: |
    cd /var/www
    tar -czf backup.tar.gz .
    mv backup.tar.gz /tmp/
```

#### Script Module
Executes local scripts on remote hosts.

**Parameters:**
- `script` (string, required) - Script path
- `args` (string) - Script arguments

**Examples:**
```yaml
# Execute script
- name: Run script
  script: /local/script.sh

# Execute with arguments
- name: Run script with args
  script: /local/script.sh arg1 arg2
```

---

### Utility Modules

#### Debug Module
Outputs debug information.

**Parameters:**
- `msg` (string) - Debug message
- `var` (string) - Variable to debug

**Examples:**
```yaml
# Debug message
- name: Debug message
  debug:
    msg: "Hello from Onigirazu"

# Debug variable
- name: Debug variable
  debug:
    var: ansible_facts
```

#### Facts Module
Gathers system facts.

**Parameters:**
- `gather_subset` (string) - Facts to gather: all, hardware, network, virtual

**Examples:**
```yaml
# Gather facts
- name: Gather facts
  facts:
    gather_subset: all
```

#### Set Fact Module
Sets custom facts.

**Parameters:**
- Any custom variable name and value

**Examples:**
```yaml
# Set fact
- name: Set custom fact
  set_fact:
    custom_var: "custom_value"

# Set multiple facts
- name: Set multiple facts
  set_fact:
    app_name: "myapp"
    app_version: "1.0.0"
```

---

## 🔧 Programmatic API

### Go API

#### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/onigirazu-cfg/onigirazu/internal/engine"
    "github.com/onigirazu-cfg/onigirazu/internal/inventory"
    "github.com/onigirazu-cfg/onigirazu/internal/modules"
)

func main() {
    // Initialize components
    moduleRegistry := modules.NewRegistry()
    inventoryMgr := inventory.NewManager(parser, logger, cache)
    
    // Load inventory
    ctx := context.Background()
    err := inventoryMgr.LoadInventory(ctx, "inventory.yml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Execute module
    module, err := moduleRegistry.GetModule("package")
    if err != nil {
        log.Fatal(err)
    }
    
    host := types.Host{Name: "web1", Address: "192.168.1.10"}
    args := map[string]interface{}{
        "name": "nginx",
        "state": "present",
    }
    
    result, err := module.Execute(ctx, host, args)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Result: %+v\n", result)
}
```

#### Advanced Usage

```go
// Create execution engine
engine := execution.NewEngine(moduleRegistry, inventoryMgr, logger)

// Execute playbook
playbook := &types.Playbook{
    Name: "webserver-setup",
    Plays: []types.Play{
        {
            Name: "Configure web server",
            Hosts: []string{"webservers"},
            Tasks: []types.Task{
                {
                    Name: "Install nginx",
                    Module: "package",
                    Args: map[string]interface{}{
                        "name": "nginx",
                        "state": "present",
                    },
                },
            },
        },
    },
}

results, err := engine.ExecutePlaybook(ctx, playbook)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Results: %+v\n", results)
```

---

## 📊 Output Formats

### Text Output (Default)

```
web1 | CHANGED => nginx installed
web2 | CHANGED => nginx installed

=== Summary: Total: 2 | Success: 2 | Failed: 0 | Changed: 2 | Duration: 1.2s ===
```

### JSON Output

```json
{
  "total": 2,
  "success": 2,
  "failed": 0,
  "changed": 2,
  "duration": "1.2s",
  "results": [
    {
      "host": "web1",
      "status": "success",
      "changed": true,
      "message": "nginx installed",
      "duration": "0.6s"
    },
    {
      "host": "web2",
      "status": "success",
      "changed": true,
      "message": "nginx installed",
      "duration": "0.6s"
    }
  ]
}
```

### YAML Output

```yaml
total: 2
success: 2
failed: 0
changed: 2
duration: 1.2s
results:
  - host: web1
    status: success
    changed: true
    message: nginx installed
    duration: 0.6s
  - host: web2
    status: success
    changed: true
    message: nginx installed
    duration: 0.6s
```

### Table Output

```
+------------------+----------+---------+------------------+
| Host             | Status   | Changed | Duration         |
+------------------+----------+---------+------------------+
| web1             | SUCCESS  | Yes     | 0.6s             |
| web2             | SUCCESS  | Yes     | 0.6s             |
+------------------+----------+---------+------------------+

Total: 2 | Success: 2 | Failed: 0 | Changed: 2 | Duration: 1.2s
```

---

## 🎯 Best Practices

### Command Usage

```bash
# Use appropriate syntax
onigirazu run all "install nginx package" -i inventory.yml  # Natural language
onigirazu run all -m package name=nginx state=present -i inventory.yml  # Traditional

# Use check mode for safety
onigirazu run all "install nginx package" --check -i inventory.yml

# Use parallel execution for performance
onigirazu run all "install nginx package" --parallel 10 -i inventory.yml
```

### Module Usage

```yaml
# Use idempotent operations
- name: Install nginx
  package:
    name: nginx
    state: present

# Use handlers for notifications
- name: Configure nginx
  template:
    src: nginx.conf.j2
    dest: /etc/nginx/nginx.conf
  notify: restart nginx

# Use conditionals for flexibility
- name: Install nginx on Debian
  package:
    name: nginx
    state: present
  when: ansible_os_family == "Debian"
```

---

## 📚 Related Documentation

- [Quick Start](Quick-Start) - Getting started
- [Modules](Modules) - Module reference
- [Playbooks](Playbooks) - Playbook reference
- [Troubleshooting](Troubleshooting) - Common issues

---

## 🎯 Summary

### API Features

- **📚 Comprehensive CLI** - Complete command reference
- **📦 Rich Modules** - 18+ built-in modules
- **🔧 Programmatic API** - Go API for integration
- **📊 Multiple Formats** - Text, JSON, YAML, Table output
- **🎯 Best Practices** - Usage guidelines

### Key Benefits

- **🚀 Easy to use** - Intuitive commands
- **🔧 Flexible** - Multiple syntax options
- **📊 Observable** - Rich output formats
- **🔒 Secure** - Built-in security features
- **📈 Performant** - Optimized execution

---

**📚 Onigirazu API provides everything you need for infrastructure automation!**

