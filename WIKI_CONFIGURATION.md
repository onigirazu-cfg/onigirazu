# ⚙️ Configuration

Onigirazu supports flexible configuration through multiple methods. This guide covers all configuration options and best practices.

## 📋 Configuration Methods

### 1. Configuration File
- **Global config** - `~/.onigirazu/config.yml`
- **Project config** - `./onigirazu.yml`
- **Custom config** - `--config path/to/config.yml`

### 2. Environment Variables
- **ONIGIRAZU_*** - Prefixed environment variables
- **System variables** - Standard system variables

### 3. Command Line Flags
- **Global flags** - Apply to all commands
- **Command flags** - Specific to commands

### 4. Inventory Variables
- **Host variables** - Per-host configuration
- **Group variables** - Per-group configuration

---

## 🔧 Configuration File

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
  check: false
  diff: false

logging:
  level: info
  format: text
  file: /var/log/onigirazu.log
  max_size: 100MB
  max_backups: 3
  max_age: 28

ssh:
  timeout: 30s
  retries: 3
  host_key_checking: true
  known_hosts_file: ~/.ssh/known_hosts
  user: ubuntu
  port: 22

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

modules:
  package:
    manager: auto
    update_cache: true
  service:
    systemctl: true
  file:
    backup: true
    mode: 0644
```

### Project Configuration

Create `./onigirazu.yml`:

```yaml
# Project-specific configuration
defaults:
  inventory: ./inventory.yml
  timeout: 60s
  parallel: 10

logging:
  level: debug
  format: json

ssh:
  user: deploy
  port: 2222

modules:
  package:
    manager: apt
    update_cache: true
  service:
    systemctl: true
```

---

## 🌍 Environment Variables

### Global Variables

```bash
# Core settings
export ONIGIRAZU_INVENTORY=inventory.yml
export ONIGIRAZU_TIMEOUT=60s
export ONIGIRAZU_PARALLEL=10
export ONIGIRAZU_OUTPUT=json
export ONIGIRAZU_VERBOSE=true

# SSH settings
export ONIGIRAZU_SSH_TIMEOUT=30s
export ONIGIRAZU_SSH_RETRIES=3
export ONIGIRAZU_SSH_HOST_KEY_CHECKING=true

# Logging settings
export ONIGIRAZU_LOG_LEVEL=info
export ONIGIRAZU_LOG_FORMAT=text
export ONIGIRAZU_LOG_FILE=/var/log/onigirazu.log

# Cache settings
export ONIGIRAZU_CACHE_ENABLED=true
export ONIGIRAZU_CACHE_TTL=5m
export ONIGIRAZU_CACHE_MAX_SIZE=100MB

# State settings
export ONIGIRAZU_STATE_FILE=.onigirazu-state
export ONIGIRAZU_STATE_BACKUP=true
export ONIGIRAZU_STATE_AUTO_SAVE=true
```

### Module Variables

```bash
# Package module
export ONIGIRAZU_PACKAGE_MANAGER=apt
export ONIGIRAZU_PACKAGE_UPDATE_CACHE=true

# Service module
export ONIGIRAZU_SERVICE_SYSTEMCTL=true

# File module
export ONIGIRAZU_FILE_BACKUP=true
export ONIGIRAZU_FILE_MODE=0644
```

---

## 🎯 Command Line Flags

### Global Flags

```bash
# Core flags
onigirazu --config config.yml
onigirazu --inventory inventory.yml
onigirazu --timeout 60s
onigirazu --parallel 10
onigirazu --output json
onigirazu --verbose
onigirazu --no-color

# SSH flags
onigirazu --ssh-timeout 30s
onigirazu --ssh-retries 3
onigirazu --ssh-host-key-checking
onigirazu --ssh-known-hosts ~/.ssh/known_hosts

# Logging flags
onigirazu --log-level info
onigirazu --log-format text
onigirazu --log-file /var/log/onigirazu.log

# Cache flags
onigirazu --cache
onigirazu --cache-ttl 5m
onigirazu --cache-max-size 100MB

# State flags
onigirazu --state-file .onigirazu-state
onigirazu --state-backup
onigirazu --state-auto-save
```

### Command-Specific Flags

```bash
# Run command flags
onigirazu run all -m ping --check
onigirazu run all -m ping --diff
onigirazu run all -m ping --timeout 30s
onigirazu run all -m ping --parallel 5

# Apply command flags
onigirazu apply playbook.yml --check
onigirazu apply playbook.yml --diff
onigirazu apply playbook.yml --timeout 60s
onigirazu apply playbook.yml --parallel 10
```

---

## 📊 Configuration Hierarchy

### Priority Order

1. **Command line flags** - Highest priority
2. **Environment variables** - Second priority
3. **Project config** - Third priority
4. **Global config** - Fourth priority
5. **Default values** - Lowest priority

### Example

```bash
# Command line (highest priority)
onigirazu run all -m ping --timeout 30s

# Environment variable (second priority)
export ONIGIRAZU_TIMEOUT=60s

# Project config (third priority)
# ./onigirazu.yml
defaults:
  timeout: 90s

# Global config (fourth priority)
# ~/.onigirazu/config.yml
defaults:
  timeout: 120s

# Default value (lowest priority)
# timeout: 30s (default)
```

---

## 🔧 SSH Configuration

### SSH Client Configuration

```bash
# ~/.ssh/config
Host webserver1
    HostName 192.168.1.10
    User ubuntu
    Port 22
    IdentityFile ~/.ssh/id_rsa
    StrictHostKeyChecking yes
    UserKnownHostsFile ~/.ssh/known_hosts

Host webserver2
    HostName 192.168.1.11
    User ubuntu
    Port 22
    IdentityFile ~/.ssh/id_rsa
    StrictHostKeyChecking yes
    UserKnownHostsFile ~/.ssh/known_hosts
```

### SSH Key Management

```bash
# Generate SSH key
ssh-keygen -t rsa -b 4096 -C "your_email@example.com"

# Copy key to host
ssh-copy-id user@host

# Test SSH connection
ssh user@host

# Add to SSH agent
ssh-add ~/.ssh/id_rsa
```

### SSH Host Key Verification

```bash
# Enable host key checking (default)
onigirazu run all -m ping -i inventory.yml

# Disable host key checking (not recommended)
onigirazu run all -m ping -i inventory.yml --skip-host-key-check

# Use custom known hosts file
onigirazu run all -m ping -i inventory.yml --ssh-known-hosts ~/.ssh/known_hosts
```

---

## 📝 Logging Configuration

### Log Levels

```yaml
logging:
  level: debug  # debug, info, warn, error
  format: text  # text, json
  file: /var/log/onigirazu.log
  max_size: 100MB
  max_backups: 3
  max_age: 28
```

### Log Formats

```yaml
# Text format (human-readable)
logging:
  format: text
  level: info

# JSON format (machine-readable)
logging:
  format: json
  level: debug
```

### Log Rotation

```yaml
logging:
  file: /var/log/onigirazu.log
  max_size: 100MB
  max_backups: 3
  max_age: 28
```

---

## 💾 Cache Configuration

### Cache Settings

```yaml
cache:
  enabled: true
  ttl: 5m
  max_size: 100MB
  cleanup_interval: 1h
  facts:
    enabled: true
    ttl: 10m
  templates:
    enabled: true
    ttl: 5m
  packages:
    enabled: true
    ttl: 15m
```

### Cache Types

```yaml
cache:
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

---

## 🗃️ State Configuration

### State Settings

```yaml
state:
  file: .onigirazu-state
  backup: true
  backup_count: 5
  auto_save: true
  compression: true
  encryption: false
```

### State Backup

```yaml
state:
  backup: true
  backup_count: 5
  backup_dir: .onigirazu-backups
  backup_interval: 1h
```

---

## 📦 Module Configuration

### Package Module

```yaml
modules:
  package:
    manager: auto  # auto, apt, yum, dnf, pacman, zypper
    update_cache: true
    force: false
    timeout: 30s
```

### Service Module

```yaml
modules:
  service:
    systemctl: true
    timeout: 30s
    retries: 3
```

### File Module

```yaml
modules:
  file:
    backup: true
    mode: 0644
    owner: root
    group: root
    follow: true
```

---

## 🎯 Best Practices

### Configuration Organization

```yaml
# Global config (~/.onigirazu/config.yml)
defaults:
  inventory: inventory.yml
  timeout: 30s
  parallel: 5

# Project config (./onigirazu.yml)
defaults:
  timeout: 60s
  parallel: 10

# Environment variables
export ONIGIRAZU_TIMEOUT=90s
export ONIGIRAZU_PARALLEL=15
```

### Security Configuration

```yaml
# Secure configuration
ssh:
  host_key_checking: true
  known_hosts_file: ~/.ssh/known_hosts
  timeout: 30s
  retries: 3

logging:
  level: info
  format: text
  file: /var/log/onigirazu.log

state:
  encryption: true
  backup: true
  backup_count: 5
```

### Performance Configuration

```yaml
# Performance optimization
defaults:
  parallel: 10
  timeout: 60s

cache:
  enabled: true
  ttl: 5m
  max_size: 100MB

ssh:
  connection_pool: true
  max_connections: 10
```

---

## 🔧 Advanced Configuration

### Custom Modules

```yaml
modules:
  custom:
    enabled: true
    path: ./modules
    timeout: 30s
    retries: 3
```

### Plugin Configuration

```yaml
plugins:
  enabled: true
  path: ./plugins
  auto_load: true
  timeout: 30s
```

### Workflow Configuration

```yaml
workflows:
  enabled: true
  path: ./workflows
  timeout: 300s
  retries: 3
```

---

## 🚨 Troubleshooting

### Configuration Issues

```bash
# Check configuration
onigirazu run all -m ping -i inventory.yml --config-check

# Validate configuration
onigirazu run all -m ping -i inventory.yml --validate

# Show configuration
onigirazu run all -m ping -i inventory.yml --show-config
```

### Debug Configuration

```bash
# Debug mode
onigirazu run all -m ping -i inventory.yml --debug

# Verbose output
onigirazu run all -m ping -i inventory.yml --verbose

# Trace output
onigirazu run all -m ping -i inventory.yml --trace
```

---

## 📚 Related Documentation

- [Installation](Installation) - Installation guide
- [Quick Start](Quick-Start) - Getting started
- [Troubleshooting](Troubleshooting) - Common issues
- [Architecture](Architecture) - System architecture

---

## 🎯 Summary

### Configuration Checklist

1. **✅ Choose configuration method** - File, environment, or flags
2. **✅ Set global defaults** - Global configuration file
3. **✅ Configure project settings** - Project-specific config
4. **✅ Set up SSH** - SSH keys and configuration
5. **✅ Configure logging** - Log levels and formats
6. **✅ Set up caching** - Performance optimization
7. **✅ Configure state** - State management settings

### Configuration Methods

- **📁 Configuration files** - YAML configuration
- **🌍 Environment variables** - System variables
- **🎯 Command line flags** - Runtime configuration
- **📊 Inventory variables** - Host and group variables

---

**⚙️ Onigirazu is now configured and ready to use!**
