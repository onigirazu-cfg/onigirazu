# 🚀 Onigirazu Ad-hoc Commands - Complete Guide

## Table of Contents

- [Introduction](#introduction)
- [Quick Start](#quick-start)
- [5 Input Formats](#5-input-formats)
- [Output Formats](#output-formats)
- [Advanced Features](#advanced-features)
- [Real-World Scenarios](#real-world-scenarios)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)
- [Migration from Ansible](#migration-from-ansible)

---

## Introduction

Onigirazu's ad-hoc command system is **unique in the configuration management world** - it supports **5 different input formats**, including natural language, making it the most flexible and user-friendly tool available.

### Why Ad-hoc Commands?

Ad-hoc commands are perfect for:

- 🔍 **Quick system checks** - Check disk space, memory, uptime
- 🚀 **Rapid deployments** - Install packages, restart services
- 🐛 **Troubleshooting** - Check logs, test connectivity
- 📊 **Information gathering** - Collect system information
- ⚡ **One-off tasks** - Tasks you don't want to write a playbook for

### What Makes Onigirazu Special?

| Feature | Description | Unique? |
|---------|-------------|---------|
| **5 Input Formats** | Ansible-like, Natural Language, Module:Args, JSON, YAML | ✅ Yes |
| **Natural Language** | "install nginx package", "start nginx service" | ✅ Yes |
| **4 Output Formats** | Text (colored), JSON, YAML, Table | ⚠️ Partial |
| **Formatted stdout** | Multi-line command output with proper indentation | ✅ Yes |
| **Parallel Execution** | Configurable concurrency (default: 5) | ❌ No |
| **Check Mode** | Dry-run without making changes | ❌ No |

---

## Quick Start

### Basic Syntax

```bash
onigirazu run <pattern> [options] <command>
```

- `<pattern>` - Host or group name (all, webservers, localhost, etc.)
- `[options]` - Flags like `-m`, `-i`, `--parallel`, etc.
- `<command>` - The command in any of the 5 supported formats

### Your First Commands

```bash
# 1. Ping all hosts
onigirazu run all -m ping -i inventory.yml

# 2. Check uptime
onigirazu run all -m shell 'cmd="uptime"' -i inventory.yml

# 3. Install nginx (natural language!)
onigirazu run webservers "install nginx package" -i inventory.yml
```

---

## 5 Input Formats

### Format 1: Ansible-like Syntax (Familiar)

**Best for:** Users migrating from Ansible, traditional automation

**Syntax:**

```bash
onigirazu run <pattern> -m <module> [key=value key=value ...] -i inventory.yml
```

**Examples:**

```bash
# Package management
onigirazu run all -m package name=nginx state=present -i inventory.yml
onigirazu run all -m package name=apache2 state=absent -i inventory.yml
onigirazu run all -m package name=mysql-server state=latest -i inventory.yml

# Service management
onigirazu run webservers -m service name=nginx state=started -i inventory.yml
onigirazu run webservers -m service name=nginx state=stopped -i inventory.yml
onigirazu run webservers -m service name=nginx state=restarted enabled=true -i inventory.yml

# Command execution
onigirazu run all -m command 'command="uptime"' -i inventory.yml
onigirazu run all -m shell 'cmd="df -h | grep /dev"' -i inventory.yml

# File operations
onigirazu run all -m file path=/tmp/test.txt state=touch -i inventory.yml
onigirazu run all -m file path=/tmp/old.log state=absent -i inventory.yml

# User management
onigirazu run all -m user name=john state=present -i inventory.yml
onigirazu run all -m user name=olduser state=absent -i inventory.yml
```

**Pros:**

- ✅ Familiar to Ansible users
- ✅ Explicit and clear
- ✅ Good for scripting

**Cons:**

- ❌ More typing required
- ❌ Less intuitive for beginners

---

### Format 2: Natural Language (Unique! 🌟)

**Best for:** Beginners, quick tasks, intuitive operations

**Syntax:**

```bash
onigirazu run <pattern> "<action> <target> <type>" -i inventory.yml
```

**Supported Patterns:**

#### Package Operations

```bash
# Install packages
onigirazu run all "install nginx package" -i inventory.yml
onigirazu run all "add apache package" -i inventory.yml
onigirazu run all "install the mysql package" -i inventory.yml

# Remove packages
onigirazu run all "remove nginx package" -i inventory.yml
onigirazu run all "uninstall apache package" -i inventory.yml
onigirazu run all "delete old-package package" -i inventory.yml

# Update packages
onigirazu run all "update nginx package" -i inventory.yml
onigirazu run all "upgrade mysql package" -i inventory.yml
onigirazu run all "update all package" -i inventory.yml
```

#### Service Operations

```bash
# Start services
onigirazu run webservers "start nginx service" -i inventory.yml
onigirazu run all "start apache service" -i inventory.yml
onigirazu run dbservers "start mysql service" -i inventory.yml

# Stop services
onigirazu run webservers "stop nginx service" -i inventory.yml
onigirazu run all "stop apache service" -i inventory.yml

# Restart services
onigirazu run webservers "restart nginx service" -i inventory.yml
onigirazu run all "restart apache service" -i inventory.yml

# Reload services
onigirazu run webservers "reload nginx service" -i inventory.yml
```

#### File Operations

```bash
# Create files
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
onigirazu run all "touch file /tmp/empty.txt" -i inventory.yml
onigirazu run webservers "create file /var/www/index.html" -i inventory.yml

# Delete files
onigirazu run all "delete file /tmp/old.txt" -i inventory.yml
onigirazu run all "remove file /tmp/temp.log" -i inventory.yml
```

**Pattern Recognition:**

The parser recognizes these patterns:

| Pattern | Module | Args |
|---------|--------|------|
| `install <name> package` | package | name=\<name>, state=present |
| `remove <name> package` | package | name=\<name>, state=absent |
| `update <name> package` | package | name=\<name>, state=latest |
| `start <name> service` | service | name=\<name>, state=started |
| `stop <name> service` | service | name=\<name>, state=stopped |
| `restart <name> service` | service | name=\<name>, state=restarted |
| `reload <name> service` | service | name=\<name>, state=reloaded |
| `create file <path>` | file | path=\<path>, state=touch |
| `delete file <path>` | file | path=\<path>, state=absent |

**Pros:**

- ✅ Extremely intuitive
- ✅ Perfect for beginners
- ✅ Fast for common operations
- ✅ Self-documenting

**Cons:**

- ❌ Limited to supported patterns
- ❌ Can't specify all module arguments

---

### Format 3: Module:Args Syntax (Compact)

**Best for:** Quick one-liners, compact commands

**Syntax:**

```bash
onigirazu run <pattern> "<module>:<key>=<value>,<key>=<value>" -i inventory.yml
```

**Examples:**

```bash
# Package management
onigirazu run all "package:name=nginx,state=present" -i inventory.yml
onigirazu run all "package:name=apache2,state=absent" -i inventory.yml

# Service management
onigirazu run webservers "service:name=nginx,state=started,enabled=true" -i inventory.yml
onigirazu run webservers "service:name=nginx,state=restarted" -i inventory.yml

# Command execution
onigirazu run all "shell:cmd=uptime" -i inventory.yml
onigirazu run all "command:command=df -h" -i inventory.yml

# File operations
onigirazu run all "file:path=/tmp/test.txt,state=touch" -i inventory.yml
onigirazu run all "file:path=/tmp/old.log,state=absent" -i inventory.yml
```

**Pros:**

- ✅ Compact and concise
- ✅ All arguments in one string
- ✅ Good for copy-paste

**Cons:**

- ❌ Less readable for complex arguments
- ❌ Comma-separated values can be confusing

---

### Format 4: JSON Format (Structured)

**Best for:** Programmatic usage, API integration, complex arguments

**Syntax:**

```bash
onigirazu run <pattern> '{"module":"<name>","args":{...}}' -i inventory.yml
```

**Examples:**

```bash
# Package management
onigirazu run all '{"module":"package","args":{"name":"nginx","state":"present"}}' -i inventory.yml

# Service management
onigirazu run webservers '{"module":"service","args":{"name":"nginx","state":"started","enabled":true}}' -i inventory.yml

# Command execution
onigirazu run all '{"module":"shell","args":{"cmd":"df -h"}}' -i inventory.yml

# File operations with multiple args
onigirazu run all '{"module":"file","args":{"path":"/tmp/test.txt","state":"touch","mode":"0644","owner":"root"}}' -i inventory.yml

# Complex template operation
onigirazu run webservers '{"module":"template","args":{"src":"nginx.conf.j2","dest":"/etc/nginx/nginx.conf","mode":"0644","validate":"nginx -t -c %s"}}' -i inventory.yml
```

**Pros:**

- ✅ Structured and parseable
- ✅ Perfect for API/programmatic usage
- ✅ Supports complex nested arguments
- ✅ Easy to generate from code

**Cons:**

- ❌ Verbose
- ❌ Requires proper JSON escaping
- ❌ Not human-friendly

---

### Format 5: YAML Format (Readable)

**Best for:** Complex arguments, multi-line values, readability

**Syntax:**

```bash
onigirazu run <pattern> 'module: <name>
args:
  key: value
  key: value' -i inventory.yml
```

**Examples:**

```bash
# Package management
onigirazu run all 'module: package
args:
  name: nginx
  state: present' -i inventory.yml

# Service management
onigirazu run webservers 'module: service
args:
  name: nginx
  state: started
  enabled: true' -i inventory.yml

# File with multiple arguments
onigirazu run all 'module: file
args:
  path: /tmp/test.txt
  state: touch
  mode: "0644"
  owner: root
  group: root' -i inventory.yml

# Template with validation
onigirazu run webservers 'module: template
args:
  src: nginx.conf.j2
  dest: /etc/nginx/nginx.conf
  mode: "0644"
  validate: "nginx -t -c %s"
  backup: true' -i inventory.yml
```

**Pros:**

- ✅ Most readable format
- ✅ Supports multi-line values
- ✅ Natural for complex arguments
- ✅ Familiar to YAML users

**Cons:**

- ❌ Requires proper YAML syntax
- ❌ Multi-line in shell can be tricky
- ❌ Indentation matters

---

## Output Formats

Onigirazu supports **4 output formats** to suit different needs:

### 1. Text Format (Default)

**Best for:** Human reading, terminal usage

```bash
onigirazu run all -m shell 'cmd="df -h | head -5"' -i inventory.yml
```

**Output:**

```
localhost | CHANGED =>
    Filesystem      Size  Used Avail Use% Mounted on
    /dev/sda1       100G   45G   50G  48% /
    tmpfs           7.8G     0  7.8G   0% /dev/shm
    /dev/sdb1       500G  200G  275G  42% /data

=== Summary: Total: 1 | Success: 1 | Failed: 0 | Changed: 1 | Duration: 15ms ===
```

**Features:**

- ✅ Color-coded status (SUCCESS=green, CHANGED=yellow, FAILED=red)
- ✅ Formatted multi-line output with indentation
- ✅ Summary line with statistics
- ✅ Human-readable

**Disable colors:**

```bash
onigirazu run all -m ping --no-color -i inventory.yml
```

---

### 2. JSON Format

**Best for:** Scripting, parsing, API integration

```bash
onigirazu run all -m shell 'cmd="whoami"' -i inventory.yml -o json
```

**Output:**

```json
{
  "total": 1,
  "success": 1,
  "failed": 0,
  "changed": 1,
  "skipped": 0,
  "duration": "10.5ms",
  "results": [
    {
      "host": "localhost",
      "status": "success",
      "changed": true,
      "message": "root\n",
      "duration": "10.2ms"
    }
  ]
}
```

**Features:**

- ✅ Structured data
- ✅ Easy to parse with `jq`
- ✅ Perfect for automation
- ✅ Includes all metadata

**Parse with jq:**

```bash
# Get only successful hosts
onigirazu run all -m ping -i inventory.yml -o json | jq '.results[] | select(.status=="success") | .host'

# Get total execution time
onigirazu run all -m shell 'cmd="uptime"' -i inventory.yml -o json | jq '.duration'

# Extract stdout from all hosts
onigirazu run all -m shell 'cmd="hostname"' -i inventory.yml -o json | jq '.results[].message'
```

---

### 3. YAML Format

**Best for:** Configuration files, human-readable structured data

```bash
onigirazu run all -m shell 'cmd="whoami"' -i inventory.yml -o yaml
```

**Output:**

```yaml
changed: 1
duration: 10.5ms
failed: 0
results:
    - changed: true
      duration: 10.2ms
      host: localhost
      message: |
        root
      status: success
skipped: 0
success: 1
total: 1
```

**Features:**

- ✅ Human-readable structure
- ✅ Multi-line strings properly formatted
- ✅ Easy to read and understand
- ✅ Can be used as input for other tools

**Parse with yq:**

```bash
# Get successful hosts
onigirazu run all -m ping -i inventory.yml -o yaml | yq '.results[] | select(.status=="success") | .host'
```

---

### 4. Table Format

**Best for:** Quick overview, compact display

```bash
onigirazu run all -m ping -i inventory.yml -o table
```

**Output:**

```
+------------------+----------+---------+------------------+
| Host             | Status   | Changed | Duration         |
+------------------+----------+---------+------------------+
| localhost        | SUCCESS  | No      | 5ms              |
| web1             | SUCCESS  | No      | 12ms             |
| web2             | SUCCESS  | No      | 15ms             |
| db1              | SUCCESS  | No      | 20ms             |
+------------------+----------+---------+------------------+

Total: 4 | Success: 4 | Failed: 0 | Changed: 0 | Duration: 52ms
```

**Features:**

- ✅ Compact overview
- ✅ Easy to scan
- ✅ Good for many hosts
- ✅ ASCII table format

---

## Advanced Features

### Parallel Execution

Control how many hosts to execute on simultaneously:

```bash
# Default: 5 hosts in parallel
onigirazu run all -m shell 'cmd="uptime"' -i inventory.yml

# Sequential execution (one at a time)
onigirazu run all -m shell 'cmd="uptime"' --parallel 1 -i inventory.yml

# High parallelism (20 hosts at once)
onigirazu run all -m shell 'cmd="uptime"' --parallel 20 -i inventory.yml

# Unlimited parallelism (all hosts at once)
onigirazu run all -m shell 'cmd="uptime"' --parallel 0 -i inventory.yml
```

**When to use:**

- `--parallel 1` - For operations that must be sequential (database migrations, rolling updates)
- `--parallel 5` (default) - Good balance for most operations
- `--parallel 20+` - For read-only operations on many hosts (gathering info)
- `--parallel 0` - For maximum speed when order doesn't matter

---

### Check Mode (Dry-run)

Preview what would change without actually making changes:

```bash
# Check what would happen
onigirazu run all -m package name=nginx state=present --check -i inventory.yml

# Check with verbose output
onigirazu run all -m package name=nginx state=present --check -V -i inventory.yml
```

**Supported by modules:**

- ✅ package
- ✅ service
- ✅ file
- ✅ user
- ✅ group
- ❌ command (can't predict output)
- ❌ shell (can't predict output)

---

### Verbose Mode

Get detailed execution information:

```bash
# Verbose output
onigirazu run all -m shell 'cmd="uptime"' -V -i inventory.yml

# Very verbose (debug level)
onigirazu run all -m shell 'cmd="uptime"' -VV -i inventory.yml
```

**Shows:**

- Connection details
- Module arguments
- Execution steps
- Timing information
- Debug messages

---

### Timeout Control

Set maximum execution time:

```bash
# Default timeout (30 seconds)
onigirazu run all -m shell 'cmd="uptime"' -i inventory.yml

# Custom timeout (2 minutes)
onigirazu run all -m shell 'cmd="long-task"' --timeout 120s -i inventory.yml

# Short timeout (5 seconds)
onigirazu run all -m ping --timeout 5s -i inventory.yml
```

**Timeout formats:**

- `30s` - 30 seconds
- `5m` - 5 minutes
- `1h` - 1 hour
- `500ms` - 500 milliseconds

---

## Real-World Scenarios

### Scenario 1: System Health Check

```bash
# Check all systems are reachable
onigirazu run all -m ping -i inventory.yml

# Check disk space
onigirazu run all -m shell 'cmd="df -h"' -i inventory.yml -o table

# Check memory usage
onigirazu run all -m shell 'cmd="free -h"' -i inventory.yml

# Check system load
onigirazu run all -m shell 'cmd="uptime"' -i inventory.yml

# Check running services
onigirazu run all -m shell 'cmd="systemctl list-units --type=service --state=running"' -i inventory.yml
```

---

### Scenario 2: Quick Nginx Deployment

```bash
# 1. Install nginx on all web servers
onigirazu run webservers "install nginx package" -i inventory.yml

# 2. Start nginx service
onigirazu run webservers "start nginx service" -i inventory.yml

# 3. Verify nginx is running
onigirazu run webservers -m shell 'cmd="systemctl status nginx"' -i inventory.yml

# 4. Check nginx is listening on port 80
onigirazu run webservers -m shell 'cmd="netstat -tlnp | grep :80"' -i inventory.yml
```

---

### Scenario 3: Troubleshooting

```bash
# Check nginx error logs
onigirazu run webservers -m shell 'cmd="tail -n 50 /var/log/nginx/error.log"' -i inventory.yml

# Check system logs for errors
onigirazu run all -m shell 'cmd="journalctl -p err -n 20"' -i inventory.yml

# Check disk I/O
onigirazu run all -m shell 'cmd="iostat -x 1 3"' -i inventory.yml

# Check network connectivity
onigirazu run all -m shell 'cmd="ping -c 3 8.8.8.8"' -i inventory.yml

# Check DNS resolution
onigirazu run all -m shell 'cmd="nslookup google.com"' -i inventory.yml
```

---

### Scenario 4: Security Audit

```bash
# Check for security updates
onigirazu run all -m shell 'cmd="apt list --upgradable 2>/dev/null | grep security"' -i inventory.yml

# Check open ports
onigirazu run all -m shell 'cmd="netstat -tlnp"' -i inventory.yml

# Check failed login attempts
onigirazu run all -m shell 'cmd="grep \"Failed password\" /var/log/auth.log | tail -20"' -i inventory.yml

# Check sudo usage
onigirazu run all -m shell 'cmd="grep sudo /var/log/auth.log | tail -20"' -i inventory.yml

# List users with shell access
onigirazu run all -m shell 'cmd="grep -v nologin /etc/passwd"' -i inventory.yml
```

---

### Scenario 5: Batch Updates

```bash
# Update all packages (check mode first!)
onigirazu run all "update all package" --check -i inventory.yml

# Actually update (with high parallelism)
onigirazu run all "update all package" --parallel 20 -i inventory.yml

# Restart services after update
onigirazu run webservers "restart nginx service" -i inventory.yml
onigirazu run dbservers "restart mysql service" -i inventory.yml

# Verify services are running
onigirazu run all -m shell 'cmd="systemctl is-active nginx mysql"' -i inventory.yml
```

---

## Best Practices

### 1. Always Use Check Mode First

```bash
# ❌ Don't do this directly
onigirazu run all -m package name=nginx state=absent -i inventory.yml

# ✅ Do this first
onigirazu run all -m package name=nginx state=absent --check -i inventory.yml
# Then if OK, run without --check
```

### 2. Use Appropriate Parallelism

```bash
# ❌ Don't use high parallelism for writes
onigirazu run all -m package name=nginx state=present --parallel 50 -i inventory.yml

# ✅ Use moderate parallelism
onigirazu run all -m package name=nginx state=present --parallel 5 -i inventory.yml

# ✅ High parallelism is OK for reads
onigirazu run all -m shell 'cmd="uptime"' --parallel 50 -i inventory.yml
```

### 3. Choose the Right Format

```bash
# ✅ Natural language for simple operations
onigirazu run all "install nginx package" -i inventory.yml

# ✅ Ansible-like for complex arguments
onigirazu run all -m file path=/tmp/test mode=0644 owner=root group=root -i inventory.yml

# ✅ JSON for programmatic usage
COMMAND='{"module":"package","args":{"name":"nginx","state":"present"}}'
onigirazu run all "$COMMAND" -i inventory.yml
```

### 4. Use Appropriate Output Format

```bash
# ✅ Text for terminal
onigirazu run all -m ping -i inventory.yml

# ✅ JSON for scripting
onigirazu run all -m shell 'cmd="hostname"' -i inventory.yml -o json | jq '.results[].message'

# ✅ Table for overview
onigirazu run all -m ping -i inventory.yml -o table
```

### 5. Set Reasonable Timeouts

```bash
# ✅ Short timeout for quick operations
onigirazu run all -m ping --timeout 5s -i inventory.yml

# ✅ Long timeout for slow operations
onigirazu run all -m package name=nginx state=present --timeout 300s -i inventory.yml
```

---

## Troubleshooting

### Problem: Command not found

```bash
# Error: onigirazu: command not found

# Solution: Check installation
which onigirazu
# If not found, reinstall or add to PATH
```

### Problem: Inventory file not found

```bash
# Error: failed to load inventory: file not found

# Solution: Use absolute path or check file exists
ls -la inventory.yml
onigirazu run all -m ping -i /full/path/to/inventory.yml
```

### Problem: No hosts matched

```bash
# Error: no hosts matched pattern

# Solution: Check pattern and inventory
onigirazu run all -m ping -i inventory.yml  # Use 'all' to test
# Check your inventory file has the group/host you're targeting
```

### Problem: SSH connection failed

```bash
# Error: failed to connect to host

# Solution: Test SSH manually
ssh user@host
# Check SSH keys, permissions, firewall
```

### Problem: Module not found

```bash
# Error: module 'xyz' not found

# Solution: Check available modules
onigirazu modules list
# Use correct module name
```

### Problem: Timeout

```bash
# Error: execution timeout

# Solution: Increase timeout
onigirazu run all -m shell 'cmd="slow-command"' --timeout 300s -i inventory.yml
```

### Problem: Permission denied

```bash
# Error: permission denied

# Solution: Use sudo or check user permissions
# Add become: true in inventory or use appropriate user
```

---

## Migration from Ansible

### Ansible vs Onigirazu Syntax

| Ansible | Onigirazu (Ansible-like) | Onigirazu (Natural Language) |
|---------|--------------------------|------------------------------|
| `ansible all -m ping` | `onigirazu run all -m ping -i inventory.yml` | `onigirazu run all -m ping -i inventory.yml` |
| `ansible all -m shell -a "uptime"` | `onigirazu run all -m shell 'cmd="uptime"' -i inventory.yml` | N/A |
| `ansible all -m package -a "name=nginx state=present"` | `onigirazu run all -m package name=nginx state=present -i inventory.yml` | `onigirazu run all "install nginx package" -i inventory.yml` |
| `ansible all -m service -a "name=nginx state=started"` | `onigirazu run all -m service name=nginx state=started -i inventory.yml` | `onigirazu run all "start nginx service" -i inventory.yml` |

### Key Differences

1. **Inventory flag is required** in Onigirazu: `-i inventory.yml`
2. **Shell module** uses `cmd` instead of direct argument
3. **Natural language** is available as an alternative
4. **Output formats** are more flexible (4 formats vs 2)

### Migration Tips

1. **Start with Ansible-like syntax** - it's familiar
2. **Gradually adopt natural language** for simple operations
3. **Use JSON output** for existing scripts that parse Ansible output
4. **Test with check mode** before running on production

---

## Summary

Onigirazu's ad-hoc command system is **the most flexible and user-friendly** in the configuration management world:

### ✅ Unique Features

- **5 input formats** - Choose what works best for you
- **Natural language** - "install nginx package" just works
- **4 output formats** - Text, JSON, YAML, Table
- **Formatted stdout** - Multi-line output properly displayed
- **Flexible and powerful** - From beginners to experts

### 🚀 Quick Reference

```bash
# Ping
onigirazu run all -m ping -i inventory.yml

# Command
onigirazu run all -m shell 'cmd="uptime"' -i inventory.yml

# Natural language
onigirazu run all "install nginx package" -i inventory.yml

# JSON output
onigirazu run all -m ping -i inventory.yml -o json

# Parallel execution
onigirazu run all -m shell 'cmd="uptime"' --parallel 10 -i inventory.yml

# Check mode
onigirazu run all -m package name=nginx state=present --check -i inventory.yml
```

### 📚 Further Reading

- [Module Documentation](modules/README.md)
- [Inventory Guide](INVENTORY.md)
- [Playbook Guide](PLAYBOOK.md)
- [Best Practices](BEST_PRACTICES.md)

---

**Made with ❤️ by the Onigirazu team**
