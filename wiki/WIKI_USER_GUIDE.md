# 📖 User Guide

This comprehensive user guide covers everything you need to know to use Onigirazu effectively.

## 📋 Table of Contents

### Getting Started
- [Installation](Installation)
- [Quick Start](Quick-Start)
- [Configuration](Configuration)

### Core Features
- [Natural Language Commands](Natural-Language-Commands)
- [Ad-hoc Commands](Ad-hoc-Commands)
- [Playbooks](Playbooks)
- [Modules](Modules)

### Advanced Features
- [State Management](State-Management)
- [Performance Tuning](Performance-Tuning)
- [Security](Security)

### Reference
- [API Reference](API-Reference)
- [Troubleshooting](Troubleshooting)
- [Migration from Ansible](Migration-from-Ansible)

---

## 🚀 Getting Started

### Installation

Onigirazu can be installed in several ways:

#### Pre-built Binaries (Recommended)
```bash
# Linux x86_64
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# macOS (Intel)
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_x86_64.tar.gz
tar -xzf onigirazu_Darwin_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# macOS (Apple Silicon)
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_arm64.tar.gz
tar -xzf onigirazu_Darwin_arm64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

#### Package Managers
```bash
# Homebrew (macOS)
brew install onigirazu

# APT (Ubuntu/Debian)
sudo apt install onigirazu

# YUM (RHEL/CentOS)
sudo yum install onigirazu
```

#### Build from Source
```bash
# Clone repository
git clone https://github.com/onigirazu-cfg/onigirazu.git
cd onigirazu

# Build
go build -o onigirazu cmd/onigirazu/main.go

# Install
sudo mv onigirazu /usr/local/bin/
```

### Verify Installation

```bash
# Check version
onigirazu --version

# Check help
onigirazu --help

# List modules
onigirazu --list-modules
```

---

## 🎯 Quick Start

### 1. Create Inventory

```yaml
# inventory.yml
all:
  children:
    webservers:
      hosts:
        web1:
          ansible_host: 192.168.1.10
          ansible_user: ubuntu
        web2:
          ansible_host: 192.168.1.11
          ansible_user: ubuntu
    dbservers:
      hosts:
        db1:
          ansible_host: 192.168.1.20
          ansible_user: ubuntu
```

### 2. Test Connection

```bash
# Ping all hosts
onigirazu run all -m ping -i inventory.yml

# Ping specific group
onigirazu run webservers -m ping -i inventory.yml
```

### 3. Run Your First Commands

```bash
# Natural language commands
onigirazu run webservers "install nginx package" -i inventory.yml
onigirazu run webservers "start nginx service" -i inventory.yml

# Traditional commands
onigirazu run webservers -m package name=nginx state=present -i inventory.yml
onigirazu run webservers -m service name=nginx state=started -i inventory.yml
```

---

## 🎯 Natural Language Commands

Onigirazu supports intuitive natural language commands that make configuration management more accessible.

### Package Operations

```bash
# Install packages
onigirazu run all "install nginx package" -i inventory.yml
onigirazu run webservers "add apache package" -i inventory.yml

# Remove packages
onigirazu run all "remove nginx package" -i inventory.yml
onigirazu run all "uninstall apache package" -i inventory.yml

# Update packages
onigirazu run all "update nginx package" -i inventory.yml
```

### Service Operations

```bash
# Start services
onigirazu run webservers "start nginx service" -i inventory.yml
onigirazu run all "start apache service" -i inventory.yml

# Stop services
onigirazu run webservers "stop nginx service" -i inventory.yml
onigirazu run all "stop apache service" -i inventory.yml

# Restart services
onigirazu run webservers "restart nginx service" -i inventory.yml
onigirazu run webservers "reload nginx service" -i inventory.yml
```

### File Operations

```bash
# Create files
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
onigirazu run webservers "create file /var/www/index.html" -i inventory.yml

# Delete files
onigirazu run all "delete file /tmp/old.txt" -i inventory.yml
onigirazu run all "delete file /tmp/temp.log" -i inventory.yml
```

---

## 🔧 Ad-hoc Commands

Ad-hoc commands allow you to execute quick operations without creating playbooks.

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
onigirazu run all "install nginx package" --check -i inventory.yml

# Parallel execution
onigirazu run all "install nginx package" --parallel 10 -i inventory.yml

# JSON output
onigirazu run all "install nginx package" --output json -i inventory.yml

# Verbose mode
onigirazu run all "install nginx package" -V -i inventory.yml
```

---

## 📚 Playbooks

Playbooks are YAML files that describe the desired state of your infrastructure.

### Basic Playbook

```yaml
# webserver-setup.yml
---
- name: Install and configure nginx
  hosts: webservers
  become: true
  tasks:
    - name: Install nginx
      package:
        name: nginx
        state: present
    
    - name: Start nginx
      service:
        name: nginx
        state: started
        enabled: true
    
    - name: Configure nginx
      template:
        src: nginx.conf.j2
        dest: /etc/nginx/nginx.conf
      notify: restart nginx

  handlers:
    - name: restart nginx
      service:
        name: nginx
        state: restarted
```

### Advanced Playbook

```yaml
# advanced-setup.yml
---
- name: Web server setup
  hosts: webservers
  become: true
  vars:
    nginx_port: 80
    nginx_user: www-data
  tasks:
    - name: Update package cache
      package:
        update_cache: true
      when: ansible_os_family == "Debian"
    
    - name: Install nginx
      package:
        name: nginx
        state: present
    
    - name: Configure nginx
      template:
        src: nginx.conf.j2
        dest: /etc/nginx/nginx.conf
        backup: true
      notify: restart nginx
    
    - name: Start nginx
      service:
        name: nginx
        state: started
        enabled: true

  handlers:
    - name: restart nginx
      service:
        name: nginx
        state: restarted

- name: Database server setup
  hosts: dbservers
  become: true
  tasks:
    - name: Install mysql
      package:
        name: mysql-server
        state: present
    
    - name: Start mysql
      service:
        name: mysql
        state: started
        enabled: true
```

### Running Playbooks

```bash
# Basic execution
onigirazu apply playbook.yml -i inventory.yml

# With options
onigirazu apply playbook.yml -i inventory.yml --check --diff

# Specific hosts
onigirazu apply playbook.yml -i inventory.yml --limit webservers

# Specific tags
onigirazu apply playbook.yml -i inventory.yml --tags nginx
```

---

## 📦 Modules

Onigirazu includes 18+ built-in modules for comprehensive system management.

### System Modules

#### Package Module
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

### File Modules

#### File Module
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

### Execution Modules

#### Command Module
```yaml
# Execute command
- name: Check uptime
  command: uptime

# Execute with arguments
- name: List files
  command: ls -la /tmp
```

#### Shell Module
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

---

## ⚙️ Configuration

### Global Configuration

Create `~/.onigirazu/config.yml`:

```yaml
# Global Onigirazu configuration
defaults:
  inventory: inventory.yml
  timeout: 30s
  parallel: 5
  output: text
  verbose: false

logging:
  level: info
  format: text
  file: /var/log/onigirazu.log

ssh:
  timeout: 30s
  retries: 3
  host_key_checking: true
  known_hosts_file: ~/.ssh/known_hosts

cache:
  enabled: true
  ttl: 5m
  max_size: 100MB
  cleanup_interval: 1h

state:
  file: .onigirazu-state
  backup: true
  backup_count: 5
  auto_save: true
```

### Environment Variables

```bash
# Set environment variables
export ONIGIRAZU_INVENTORY=inventory.yml
export ONIGIRAZU_TIMEOUT=60s
export ONIGIRAZU_PARALLEL=10
export ONIGIRAZU_OUTPUT=json
export ONIGIRAZU_VERBOSE=true
```

---

## 🗃️ State Management

### State Operations

```bash
# Show current state
onigirazu state show

# Show host state
onigirazu state show --host web1

# Show execution history
onigirazu state history

# Create snapshot
onigirazu state snapshot create --name "backup"

# Rollback to snapshot
onigirazu state rollback --snapshot "backup"
```

### State Configuration

```yaml
# State configuration
state:
  file: .onigirazu-state
  backup: true
  backup_count: 10
  auto_save: true
  compression: true
  encryption: false
  
  # Execution history
  history:
    enabled: true
    max_entries: 1000
    retention_days: 30
  
  # Host state tracking
  host_state:
    enabled: true
    track_packages: true
    track_services: true
    track_files: true
    track_users: true
  
  # Rollback support
  rollback:
    enabled: true
    max_snapshots: 10
    snapshot_interval: 1h
```

---

## ⚡ Performance Tuning

### Parallel Execution

```yaml
# Configuration for parallel execution
defaults:
  parallel: 10  # Number of parallel workers
  timeout: 30s  # Per-host timeout
  retries: 3    # Retry failed operations
```

```bash
# Command line parallel execution
onigirazu run all "install nginx package" --parallel 20 -i inventory.yml
```

### Caching

```yaml
# Cache optimization
cache:
  enabled: true
  ttl: 5m
  max_size: 100MB
  cleanup_interval: 1h
  
  # Cache types
  facts:
    enabled: true
    ttl: 10m
    max_size: 50MB
  
  templates:
    enabled: true
    ttl: 5m
    max_size: 25MB
  
  packages:
    enabled: true
    ttl: 15m
    max_size: 25MB
```

### Connection Pooling

```yaml
# SSH connection pooling
ssh:
  connection_pool: true
  max_connections: 20
  idle_timeout: 5m
  max_lifetime: 1h
```

---

## 🔒 Security

### SSH Security

```bash
# Generate SSH keys
ssh-keygen -t ed25519 -b 4096 -C "your_email@example.com"

# Copy key to host
ssh-copy-id user@host

# Test SSH connection
ssh user@host
```

### Security Configuration

```yaml
# Security configuration
security:
  # Authentication
  authentication:
    methods:
      - ssh_key
      - password
      - mfa
    password_policy:
      min_length: 12
      require_uppercase: true
      require_lowercase: true
      require_numbers: true
      require_symbols: true
      max_age: 90d
  
  # Authorization
  authorization:
    rbac_enabled: true
    default_role: "viewer"
    admin_users: ["admin"]
  
  # Network security
  network:
    host_key_checking: true
    known_hosts_file: "~/.ssh/known_hosts"
    encryption: true
    compression: true
  
  # Data protection
  data_protection:
    encryption: true
    key_rotation: true
    backup_encryption: true
  
  # Audit
  audit:
    enabled: true
    log_level: "info"
    retention: 90d
```

---

## 🚨 Troubleshooting

### Common Issues

#### Connection Issues
```bash
# Check SSH connectivity
ssh user@host

# Check inventory file
onigirazu run all -m ping -i inventory.yml

# Check verbose output
onigirazu run all -m ping -V -i inventory.yml
```

#### Permission Issues
```bash
# Check sudo access
onigirazu run all -m command "sudo whoami" -i inventory.yml

# Use become
onigirazu run all -m package name=nginx state=present --become -i inventory.yml
```

#### Module Issues
```bash
# List available modules
onigirazu --list-modules

# Check module syntax
onigirazu run all -m package name=nginx state=present --check -i inventory.yml
```

### Debug Mode

```bash
# Enable debug output
onigirazu run all "install nginx package" --debug -i inventory.yml

# Check verbose output
onigirazu run all "install nginx package" -V -i inventory.yml

# Check system information
onigirazu run all -m command "uname -a" -i inventory.yml
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

```bash
# JSON output
onigirazu run all "install nginx package" --output json -i inventory.yml
```

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
    }
  ]
}
```

### YAML Output

```bash
# YAML output
onigirazu run all "install nginx package" --output yaml -i inventory.yml
```

### Table Output

```bash
# Table output
onigirazu run all "install nginx package" --output table -i inventory.yml
```

---

## 🎯 Best Practices

### Command Organization

```bash
# Group related operations
onigirazu run webservers "install nginx package" -i inventory.yml
onigirazu run webservers "start nginx service" -i inventory.yml

# Use appropriate syntax
onigirazu run all "install nginx package" -i inventory.yml  # Natural language
onigirazu run all -m package name=nginx state=present -i inventory.yml  # Traditional
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

### Performance

```bash
# Use parallel execution
onigirazu run all "install nginx package" --parallel 10 -i inventory.yml

# Use appropriate timeouts
onigirazu run all "install nginx package" --timeout 60s -i inventory.yml

# Use check mode for testing
onigirazu run all "install nginx package" --check -i inventory.yml
```

---

## 📚 Related Documentation

- [Quick Start](Quick-Start) - Getting started guide
- [Natural Language Commands](Natural-Language-Commands) - Command syntax
- [Ad-hoc Commands](Ad-hoc-Commands) - Quick operations
- [Playbooks](Playbooks) - Playbook reference
- [Modules](Modules) - Module reference
- [API Reference](API-Reference) - API documentation
- [Troubleshooting](Troubleshooting) - Common issues

---

## 🎯 Summary

### Key Features

- **🚀 High Performance** - 10x faster than Ansible
- **🎯 Natural Language** - Intuitive command syntax
- **📦 Single Binary** - No dependencies
- **🔧 Ad-hoc Commands** - Quick operations
- **📚 Playbooks** - Declarative automation
- **📦 Modules** - Comprehensive functionality
- **🗃️ State Management** - Track changes
- **⚡ Performance Tuning** - Optimize execution
- **🔒 Security** - Built-in security features

### Getting Help

- **GitHub Issues** - Report bugs and request features
- **Discussions** - Community discussions
- **Documentation** - This wiki and inline help
- **Examples** - Code examples and use cases

---

**📖 Onigirazu makes infrastructure management simple, fast, and secure!**

