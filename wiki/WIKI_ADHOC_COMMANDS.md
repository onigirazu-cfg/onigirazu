# 🔧 Ad-hoc Commands

Ad-hoc commands allow you to execute quick operations without creating playbooks. Onigirazu supports multiple syntax formats for maximum flexibility.

## 📋 Overview

Ad-hoc commands are perfect for:
- **Quick testing** of modules
- **One-off operations** without playbooks
- **Debugging** and troubleshooting
- **Interactive** system management

### Supported Syntax Formats

1. **Ansible-like** - `-m module key=value`
2. **Natural Language** - `"install nginx package"`
3. **Module:args** - `"package:name=nginx,state=present"`
4. **JSON** - `'{"module":"package","args":{"name":"nginx"}}'`
5. **YAML** - `'module: package\nargs: {name: nginx}'`

---

## 🚀 Quick Start

### Basic Usage
```bash
# Ansible-like syntax
onigirazu run all -m ping -i inventory.yml
onigirazu run webservers -m package name=nginx state=present -i inventory.yml

# Natural language
onigirazu run all "install nginx package" -i inventory.yml
onigirazu run webservers "start nginx service" -i inventory.yml

# Module:args syntax
onigirazu run all "package:name=nginx,state=present" -i inventory.yml
onigirazu run all "service:name=nginx,state=started" -i inventory.yml
```

### Advanced Options
```bash
# Check mode (dry-run)
onigirazu run all -m package name=nginx state=present --check -i inventory.yml

# Parallel execution
onigirazu run all -m package name=nginx state=present --parallel 10 -i inventory.yml

# JSON output
onigirazu run all -m ping --output json -i inventory.yml

# Verbose mode
onigirazu run all -m package name=nginx state=present -V -i inventory.yml
```

---

## 📦 Package Management

### Installation
```bash
# Ansible-like
onigirazu run all -m package name=nginx state=present -i inventory.yml
onigirazu run webservers -m package name=apache state=present -i inventory.yml

# Natural language
onigirazu run all "install nginx package" -i inventory.yml
onigirazu run webservers "add apache package" -i inventory.yml

# Module:args
onigirazu run all "package:name=nginx,state=present" -i inventory.yml
onigirazu run all "package:name=apache,state=present" -i inventory.yml

# JSON
onigirazu run all '{"module":"package","args":{"name":"nginx","state":"present"}}' -i inventory.yml
```

### Removal
```bash
# Ansible-like
onigirazu run all -m package name=nginx state=absent -i inventory.yml

# Natural language
onigirazu run all "remove nginx package" -i inventory.yml
onigirazu run all "uninstall apache package" -i inventory.yml

# Module:args
onigirazu run all "package:name=nginx,state=absent" -i inventory.yml
```

### Updates
```bash
# Ansible-like
onigirazu run all -m package name=nginx state=latest -i inventory.yml

# Natural language
onigirazu run all "update nginx package" -i inventory.yml

# Module:args
onigirazu run all "package:name=nginx,state=latest" -i inventory.yml
```

---

## 🔧 Service Management

### Starting Services
```bash
# Ansible-like
onigirazu run webservers -m service name=nginx state=started -i inventory.yml

# Natural language
onigirazu run webservers "start nginx service" -i inventory.yml

# Module:args
onigirazu run webservers "service:name=nginx,state=started" -i inventory.yml
```

### Stopping Services
```bash
# Ansible-like
onigirazu run webservers -m service name=nginx state=stopped -i inventory.yml

# Natural language
onigirazu run webservers "stop nginx service" -i inventory.yml

# Module:args
onigirazu run webservers "service:name=nginx,state=stopped" -i inventory.yml
```

### Restarting Services
```bash
# Ansible-like
onigirazu run webservers -m service name=nginx state=restarted -i inventory.yml

# Natural language
onigirazu run webservers "restart nginx service" -i inventory.yml

# Module:args
onigirazu run webservers "service:name=nginx,state=restarted" -i inventory.yml
```

---

## 📁 File Operations

### Creating Files
```bash
# Ansible-like
onigirazu run all -m file path=/tmp/test.txt state=touch -i inventory.yml

# Natural language
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
onigirazu run all "touch file /tmp/empty.txt" -i inventory.yml

# Module:args
onigirazu run all "file:path=/tmp/test.txt,state=touch" -i inventory.yml
```

### Deleting Files
```bash
# Ansible-like
onigirazu run all -m file path=/tmp/old.txt state=absent -i inventory.yml

# Natural language
onigirazu run all "delete file /tmp/old.txt" -i inventory.yml

# Module:args
onigirazu run all "file:path=/tmp/old.txt,state=absent" -i inventory.yml
```

---

## 🎯 Command Execution

### Basic Commands
```bash
# Ansible-like
onigirazu run all -m command "uptime" -i inventory.yml
onigirazu run all -m shell "ps aux | grep nginx" -i inventory.yml

# Direct command
onigirazu run all "uptime" -i inventory.yml
onigirazu run all "ps aux | grep nginx" -i inventory.yml
```

### Advanced Commands
```bash
# With variables
onigirazu run all -m command "echo $HOME" -i inventory.yml

# With environment
onigirazu run all -m shell "export PATH=/usr/local/bin:$PATH && which nginx" -i inventory.yml
```

---

## 🔧 Advanced Options

### Execution Options
```bash
# Check mode (dry-run)
onigirazu run all -m package name=nginx state=present --check -i inventory.yml

# Show differences
onigirazu run all -m file path=/tmp/test.txt content="Hello" --diff -i inventory.yml

# Timeout
onigirazu run all -m command "sleep 10" --timeout 5s -i inventory.yml

# Parallel execution
onigirazu run all -m package name=nginx state=present --parallel 5 -i inventory.yml
```

### Output Options
```bash
# Text output (default)
onigirazu run all -m ping --output text -i inventory.yml

# JSON output
onigirazu run all -m ping --output json -i inventory.yml

# YAML output
onigirazu run all -m ping --output yaml -i inventory.yml

# Table output
onigirazu run all -m ping --output table -i inventory.yml
```

### Verbose Options
```bash
# Verbose output
onigirazu run all -m package name=nginx state=present -V -i inventory.yml

# No color output
onigirazu run all -m package name=nginx state=present --no-color -i inventory.yml
```

---

## 📊 Real Examples

### Package Management
```bash
# Install nginx on all webservers
onigirazu run webservers "install nginx package" -i inventory.yml

# Remove apache from all hosts
onigirazu run all "remove apache package" -i inventory.yml

# Update mysql on database servers
onigirazu run dbservers "update mysql package" -i inventory.yml
```

### Service Management
```bash
# Start nginx on webservers
onigirazu run webservers "start nginx service" -i inventory.yml

# Stop apache on all hosts
onigirazu run all "stop apache service" -i inventory.yml

# Restart nginx on webservers
onigirazu run webservers "restart nginx service" -i inventory.yml
```

### File Operations
```bash
# Create test file
onigirazu run all "create file /tmp/test.txt" -i inventory.yml

# Delete old logs
onigirazu run all "delete file /tmp/old.log" -i inventory.yml

# Touch empty file
onigirazu run all "touch file /tmp/empty.txt" -i inventory.yml
```

---

## 🎯 Best Practices

### Command Organization
```bash
# Group related operations
onigirazu run webservers "install nginx package" -i inventory.yml
onigirazu run webservers "start nginx service" -i inventory.yml

# Use appropriate syntax for task
onigirazu run all "install nginx package" -i inventory.yml  # Natural language
onigirazu run all -m package name=nginx state=present -i inventory.yml  # Ansible-like
```

### Error Handling
```bash
# Check mode for safety
onigirazu run all "install nginx package" --check -i inventory.yml

# Verbose output for debugging
onigirazu run all "install nginx package" -V -i inventory.yml

# JSON output for parsing
onigirazu run all "install nginx package" --output json -i inventory.yml
```

---

## ⚠️ Limitations

### Current Limitations
- **Single operations**: One operation per command
- **Service dependencies**: Service operations depend on service state
- **Package manager**: Package operations depend on system package manager

### Workarounds
```bash
# For complex operations, use playbooks
# For service dependencies, check service state first
# For package manager issues, use appropriate syntax
```

---

## 📚 Related Documentation

- [Natural Language Commands](Natural-Language-Commands)
- [Modules](Modules)
- [Quick Start](Quick-Start)
- [Troubleshooting](Troubleshooting)

---

**🔧 Ad-hoc commands make Onigirazu perfect for quick operations and debugging!**

