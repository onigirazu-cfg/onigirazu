# 🗃️ State Management

Onigirazu provides comprehensive state management to track system changes, enable rollbacks, and maintain consistency across your infrastructure.

## 📋 Overview

State management in Onigirazu tracks:
- **Execution history** - What was run and when
- **System changes** - What changed on each host
- **Host state** - Current state of each host
- **Rollback points** - Safe points to revert to

### Key Features

- **📊 Execution tracking** - Complete execution history
- **🔄 Change detection** - Track what changed
- **⏪ Rollback support** - Revert to previous states
- **📈 Performance optimization** - Incremental updates
- **🔒 Security** - Encrypted state storage

---

## 🏗️ State Architecture

### State Components

```
State Management
├── Execution History
│   ├── Playbook runs
│   ├── Task executions
│   └── Host operations
├── Host State
│   ├── Package states
│   ├── Service states
│   └── File states
├── Change Tracking
│   ├── What changed
│   ├── When changed
│   └── Who changed
└── Rollback Points
    ├── Snapshots
    ├── Checkpoints
    └── Recovery points
```

### State Storage

```
State Storage
├── JSON Files (default)
├── SQLite (optional)
├── Redis (optional)
└── PostgreSQL (optional)
```

---

## 🔧 State Configuration

### Basic Configuration

```yaml
# ~/.onigirazu/config.yml
state:
  file: .onigirazu-state
  backup: true
  backup_count: 5
  auto_save: true
  compression: true
  encryption: false
```

### Advanced Configuration

```yaml
# Advanced state configuration
state:
  file: .onigirazu-state
  backup: true
  backup_count: 10
  auto_save: true
  compression: true
  encryption: true
  encryption_key: "{{ vault_state_key }}"
  
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

## 📊 State Structure

### State File Format

```json
{
  "last_run": "2024-01-15T10:30:00Z",
  "playbook": "webserver-setup.yml",
  "results": [
    {
      "play": "Configure web server",
      "hosts": ["web1", "web2"],
      "tasks": [
        {
          "name": "Install nginx",
          "module": "package",
          "args": {"name": "nginx", "state": "present"},
          "result": {
            "changed": true,
            "output": {"package": "nginx", "state": "installed"}
          }
        }
      ]
    }
  ],
  "variables": {
    "nginx_port": 80,
    "nginx_user": "www-data"
  },
  "checksums": {
    "inventory.yml": "sha256:abc123...",
    "playbook.yml": "sha256:def456..."
  }
}
```

### Host State Format

```json
{
  "host": "web1",
  "last_updated": "2024-01-15T10:30:00Z",
  "packages": {
    "nginx": {
      "state": "installed",
      "version": "1.18.0",
      "changed": true,
      "changed_at": "2024-01-15T10:30:00Z"
    }
  },
  "services": {
    "nginx": {
      "state": "running",
      "enabled": true,
      "changed": true,
      "changed_at": "2024-01-15T10:30:00Z"
    }
  },
  "files": {
    "/etc/nginx/nginx.conf": {
      "checksum": "sha256:abc123...",
      "changed": true,
      "changed_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

---

## 🔄 State Operations

### Loading State

```go
// Load state from file
state, err := stateManager.LoadState(ctx)
if err != nil {
    log.Fatalf("Failed to load state: %v", err)
}
```

### Saving State

```go
// Save state to file
err := stateManager.SaveState(ctx, state)
if err != nil {
    log.Fatalf("Failed to save state: %v", err)
}
```

### Updating State

```go
// Update state with new results
stateManager.UpdateState(state, results)

// Save updated state
err := stateManager.SaveState(ctx, state)
```

---

## 📈 Execution History

### History Tracking

```yaml
# Enable execution history
state:
  history:
    enabled: true
    max_entries: 1000
    retention_days: 30
```

### History Queries

```bash
# List execution history
onigirazu state history

# Show specific execution
onigirazu state history --execution-id 123

# Filter by date
onigirazu state history --since 2024-01-01

# Filter by host
onigirazu state history --host web1
```

### History Output

```json
{
  "executions": [
    {
      "id": "123",
      "timestamp": "2024-01-15T10:30:00Z",
      "playbook": "webserver-setup.yml",
      "hosts": ["web1", "web2"],
      "duration": "2m30s",
      "status": "success",
      "changed": 5,
      "failed": 0
    }
  ]
}
```

---

## 🏠 Host State Tracking

### Package State

```json
{
  "packages": {
    "nginx": {
      "state": "installed",
      "version": "1.18.0",
      "changed": true,
      "changed_at": "2024-01-15T10:30:00Z"
    },
    "apache2": {
      "state": "absent",
      "changed": true,
      "changed_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Service State

```json
{
  "services": {
    "nginx": {
      "state": "running",
      "enabled": true,
      "changed": true,
      "changed_at": "2024-01-15T10:30:00Z"
    },
    "apache2": {
      "state": "stopped",
      "enabled": false,
      "changed": true,
      "changed_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### File State

```json
{
  "files": {
    "/etc/nginx/nginx.conf": {
      "checksum": "sha256:abc123...",
      "size": 1024,
      "changed": true,
      "changed_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

---

## ⏪ Rollback Support

### Creating Snapshots

```bash
# Create snapshot
onigirazu state snapshot create --name "before-nginx-update"

# List snapshots
onigirazu state snapshot list

# Show snapshot details
onigirazu state snapshot show --name "before-nginx-update"
```

### Rollback Operations

```bash
# Rollback to snapshot
onigirazu state rollback --snapshot "before-nginx-update"

# Rollback to specific execution
onigirazu state rollback --execution-id 123

# Rollback specific host
onigirazu state rollback --host web1 --snapshot "before-nginx-update"
```

### Snapshot Management

```bash
# Create automatic snapshots
onigirazu state snapshot auto --interval 1h

# Clean old snapshots
onigirazu state snapshot clean --keep 10

# Export snapshot
onigirazu state snapshot export --name "backup" --file backup.json
```

---

## 🔒 Security Features

### Encryption

```yaml
# Enable state encryption
state:
  encryption: true
  encryption_key: "{{ vault_state_key }}"
  encryption_algorithm: "AES-256-GCM"
```

### Access Control

```yaml
# State access control
state:
  access_control:
    enabled: true
    users:
      - name: admin
        permissions: ["read", "write", "rollback"]
      - name: operator
        permissions: ["read"]
```

### Audit Logging

```yaml
# Audit logging
state:
  audit:
    enabled: true
    log_file: /var/log/onigirazu-audit.log
    events: ["read", "write", "rollback", "snapshot"]
```

---

## 📊 Performance Optimization

### Incremental Updates

```go
// Incremental state updates
type StateManager struct {
    // Only update changed hosts
    UpdateHostState(host string, changes map[string]interface{}) error
    
    // Batch updates for performance
    BatchUpdate(changes []HostChange) error
}
```

### Compression

```yaml
# State compression
state:
  compression: true
  compression_algorithm: "gzip"
  compression_level: 6
```

### Caching

```yaml
# State caching
state:
  cache:
    enabled: true
    ttl: 5m
    max_size: 100MB
```

---

## 🔧 State Commands

### State Inspection

```bash
# Show current state
onigirazu state show

# Show host state
onigirazu state show --host web1

# Show execution history
onigirazu state history

# Show snapshots
onigirazu state snapshot list
```

### State Management

```bash
# Create snapshot
onigirazu state snapshot create --name "backup"

# Rollback to snapshot
onigirazu state rollback --snapshot "backup"

# Clean old state
onigirazu state clean --older-than 30d

# Export state
onigirazu state export --file state-backup.json
```

### State Validation

```bash
# Validate state file
onigirazu state validate

# Check state integrity
onigirazu state check

# Repair state file
onigirazu state repair
```

---

## 🎯 Best Practices

### State Organization

```yaml
# Organize state by environment
state:
  file: ".onigirazu-state-{{ environment }}"
  backup_dir: ".onigirazu-backups-{{ environment }}"
```

### Backup Strategy

```yaml
# Backup configuration
state:
  backup: true
  backup_count: 10
  backup_interval: 1h
  backup_retention: 30d
```

### Security Best Practices

```yaml
# Security configuration
state:
  encryption: true
  encryption_key: "{{ vault_state_key }}"
  access_control:
    enabled: true
  audit:
    enabled: true
```

---

## 🚨 Troubleshooting

### Common Issues

#### State File Corruption
```bash
# Check state file
onigirazu state validate

# Repair state file
onigirazu state repair

# Restore from backup
onigirazu state restore --backup backup-2024-01-15.json
```

#### Performance Issues
```bash
# Clean old state
onigirazu state clean --older-than 30d

# Compress state file
onigirazu state compress

# Optimize state
onigirazu state optimize
```

#### Rollback Issues
```bash
# Check snapshots
onigirazu state snapshot list

# Validate snapshot
onigirazu state snapshot validate --name "backup"

# Test rollback
onigirazu state rollback --snapshot "backup" --check
```

---

## 📚 Related Documentation

- [Architecture](Architecture) - System architecture
- [Configuration](Configuration) - Configuration options
- [Troubleshooting](Troubleshooting) - Common issues
- [Security](Security) - Security best practices

---

## 🎯 Summary

### State Management Features

- **📊 Execution tracking** - Complete history
- **🏠 Host state** - Current system state
- **⏪ Rollback support** - Safe reversions
- **🔒 Security** - Encrypted storage
- **📈 Performance** - Optimized operations

### Key Benefits

- **🔄 Consistency** - Maintain system state
- **⏪ Safety** - Rollback capabilities
- **📊 Visibility** - Track all changes
- **🔒 Security** - Encrypted state
- **📈 Performance** - Efficient operations

---

**🗃️ State management ensures your infrastructure remains consistent and recoverable!**
