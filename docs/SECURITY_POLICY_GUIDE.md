# Onigirazu Security Policy Guide

**Complete reference for security policies and file operation restrictions**

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Security Policies Configuration](#security-policies-configuration)
3. [File Directory Restrictions](#file-directory-restrictions)
4. [Host Access Control](#host-access-control)
5. [Port Restrictions](#port-restrictions)
6. [Module Whitelisting](#module-whitelisting)
7. [Command Blocking](#command-blocking)
8. [File Type Restrictions](#file-type-restrictions)
9. [Audit Logging](#audit-logging)
10. [Security Policy File Locations](#security-policy-file-locations)
11. [Configuration Examples](#configuration-examples)
12. [Troubleshooting](#troubleshooting)

---

## Overview

**IMPORTANT:** Security policies are configured on the **CONTROL MACHINE** (where you run `onigirazu` command), not on remote hosts.

Security policies control what file operations, commands, and connections are allowed when executing playbooks.

### Default Security Settings

Onigirazu comes with sensible defaults:

```
Allowed Directories:     ["/tmp", "/var/tmp", "/home", "/opt"]
Blocked Directories:     ["/etc/passwd", "/etc/shadow", "/boot", "/sys", "/proc"]
Blocked Commands:        ["rm -rf", "format", "mkfs", "dd if=", ":(){ :|:& };:"]
Shell Commands:          Allowed (can be disabled)
Audit Logging:           Enabled
```

---

## Security Policies Configuration

Security policies are stored in a JSON configuration file read by Onigirazu on the **control machine**.

### Configuration File Locations (on control machine)

**Priority 1: Explicitly specified**

```bash
onigirazu --security-policy /etc/onigirazu/security-policy.json playbook.yml
```

**Priority 2: Default location**

```
/etc/onigirazu/security-policy.json
```

**Priority 3: Project directory**

```
./security-policy.json
```

**Priority 4: Built-in defaults**
If no file found, hardcoded defaults are used.

### Configuration Format

Security policies are stored in **JSON format**:

```json
{
  "allowed_hosts": ["192.168.1.*", "10.0.0.*"],
  "allowed_ports": [22, 80, 443, 8080],
  "allowed_modules": ["copy", "file", "package", "service"],
  "blocked_commands": ["rm -rf", "format"],
  "max_file_size": 10485760,
  "allowed_file_types": [".yml", ".yaml", ".json", ".txt"],
  "allowed_directories": ["/tmp", "/home", "/opt"],
  "blocked_directories": ["/etc/passwd", "/boot", "/sys"],
  "require_encryption": false,
  "max_retries": 3,
  "max_timeout": 1800000000000,
  "audit_enabled": true,
  "log_level": "info"
}
```

---

## File Directory Restrictions

### Allowed Directories

Only file operations in these directories are permitted. The error you encountered means your path is not in this list.

```json
{
  "allowed_directories": [
    "/tmp",           # Temporary files (primary use)
    "/var/tmp",       # System temporary files
    "/home",          # User home directories
    "/opt"            # Optional software and data
  ]
}
```

### Adding More Allowed Directories

To allow operations in additional directories, edit your security policy file:

```json
{
  "allowed_directories": [
    "/tmp",
    "/var/tmp",
    "/home",
    "/opt",
    "/srv/app",       # Add application directory
    "/var/www",       # Add web server directory
    "/data/backup"    # Add backup directory
  ]
}
```

### Blocked Directories

These directories are always protected from file operations:

```json
{
  "blocked_directories": [
    "/etc/passwd",    # System user database
    "/etc/shadow",    # Password hashes
    "/boot",          # System bootloader
    "/sys",           # System kernel interface
    "/proc",          # Process information
    "/dev",           # Device files
    "/root"           # Root home directory
  ]
}
```

**Note:** Blocked directories take precedence. Even if a path matches allowed directories, if it's in blocked directories it will be rejected.

---

## Host Access Control

### Allowed Hosts

Control which hosts can be accessed by Onigirazu.

```json
{
  "allowed_hosts": [
    "192.168.1.0/24",        # CIDR notation
    "10.0.0.*",              # Wildcard notation
    "server1.example.com",   # Fully qualified domain names
    "app-*",                 # Wildcard hostnames
    "localhost",             # Localhost
    "127.0.0.1"              # Loopback IP
  ]
}
```

### Empty List (Allow All)

If `allowed_hosts` is empty, all hosts are allowed:

```json
{
  "allowed_hosts": []  # Allow all hosts
}
```

---

## Port Restrictions

### Allowed Ports

Control which network ports can be used for connections.

```json
{
  "allowed_ports": [
    22,     # SSH (default)
    80,     # HTTP
    443,    # HTTPS
    8080,   # Common web server
    9090,   # Metrics/Admin
    3306    # MySQL
  ]
}
```

### Blocked Ports

Explicitly block certain ports:

```json
{
  "blocked_ports": [
    23,     # Telnet (insecure)
    21,     # FTP (insecure)
    25,     # SMTP (unencrypted)
    69      # TFTP (insecure)
  ]
}
```

---

## Module Whitelisting

Control which Onigirazu modules are allowed to run.

### Allowed Modules

```json
{
  "allowed_modules": [
    "copy",
    "file",
    "package",
    "service",
    "command",
    "shell",
    "find",
    "stat",
    "debug",
    "set_fact",
    "group_by",
    "lineinfile",
    "template"
  ]
}
```

### Empty List (Allow All)

```json
{
  "allowed_modules": []  # Allow all modules
}
```

### Deny Dangerous Modules

```json
{
  "allowed_modules": [
    // whitelist safe modules
  ],
  // Now only these modules work, others are blocked
}
```

---

## Command Blocking

### Blocked Commands

Block specific command patterns to prevent dangerous operations.

```json
{
  "blocked_commands": [
    "rm -rf",                           # Recursive force delete
    "rm -rf /",                         # Delete entire filesystem
    "format",                           # Disk formatting
    "mkfs",                             # Make filesystem
    "dd if=",                           # Direct disk write
    ":(){ :|:& };:",                    # Fork bomb
    "shutdown",                         # System shutdown
    "reboot",                           # System reboot
    "halt",                             # System halt
    "poweroff",                         # Power off
    "init 0",                           # Shutdown via init
    "init 6"                            # Reboot via init
  ]
}
```

### Pattern Matching

Command patterns support wildcards and regex:

```json
{
  "blocked_commands": [
    "*rm -rf*",                 # Wildcard: anything with "rm -rf"
    "^rm ",                     # Regex: starts with "rm "
    ".* > /dev/null.*",         # Regex: redirects to /dev/null
    ".* | sh$",                 # Regex: pipes to sh
    ".* && .*",                 # Regex: command chaining with &&
    ".* || .*",                 # Regex: command chaining with ||
    "sudo rm"                   # Block rm with sudo
  ]
}
```

---

## File Type Restrictions

### Allowed File Types

Restrict file operations to specific file extensions.

```json
{
  "allowed_file_types": [
    ".conf",      # Configuration files
    ".yml",       # YAML
    ".yaml",
    ".json",      # JSON
    ".log",       # Log files
    ".txt",       # Text files
    ".sh",        # Shell scripts
    ".py",        # Python scripts
    ".go",        # Go source
    ".c",         # C source
    ".cfg",       # Configuration
    ".ini",       # INI format
    ".env"        # Environment files
  ]
}
```

### Blocked File Types

```json
{
  "blocked_file_types": [
    ".exe",       # Executables (Windows)
    ".bat",       # Batch files
    ".cmd",       # Command files
    ".com",       # COM files
    ".scr",       # Screen saver (executable)
    ".dll"        # Dynamic libraries (Windows)
  ]
}
```

### Empty List

```json
{
  "allowed_file_types": []  // Allow all file types
}
```

---

## File Size Restrictions

### Max File Size

Limit the maximum size of files that can be transferred.

```json
{
  "max_file_size": 10485760    // 10 MB
}
```

### Calculating File Sizes

```
1 KB  = 1024 bytes
1 MB  = 1048576 bytes
10 MB = 10485760 bytes
100 MB = 104857600 bytes
1 GB = 1073741824 bytes
```

### Example Sizes

```json
{
  "max_file_size": 1048576        // 1 MB
}

{
  "max_file_size": 104857600      // 100 MB
}

{
  "max_file_size": 1073741824     // 1 GB
}
```

---

## Encryption Requirements

### Require Encryption

Require that all file transfers are encrypted.

```json
{
  "require_encryption": true
}
```

When enabled:

- All file transfers must use encryption
- Unencrypted transfers are blocked
- SSH (default) provides encryption

---

## Audit Logging

### Audit Configuration

```json
{
  "audit_enabled": true,        // Enable audit logging
  "log_level": "info"           // Audit log level
}
```

### Audit Log Levels

```json
{
  "log_level": "error"          // Only errors
}

{
  "log_level": "warn"           // Warnings and errors
}

{
  "log_level": "info"           // Normal operations
}

{
  "log_level": "debug"          // Detailed information
}
```

### Audit Log Contents

When audit logging is enabled, Onigirazu logs:

- File operations and paths accessed
- Commands executed
- Security policy violations
- Host connections
- Module usage

---

## Security Policy File Locations

### Location on Control Machine

Create this file on the machine where you run `onigirazu` command:

**System-wide (recommended for servers):**

```
/etc/onigirazu/security-policy.json
```

**Project-specific:**

```
./security-policy.json
```

**User-specific:**

```
~/.onigirazu/security-policy.json
```

### How to Specify

```bash
# Use default location (automatic discovery)
onigirazu playbook.yml

# Specify explicit path
onigirazu --security-policy /etc/onigirazu/security-policy.json playbook.yml

# Use environment variable
export ONIGIRAZU_SECURITY_POLICY=/etc/onigirazu/security-policy.json
onigirazu playbook.yml
```

---

## Configuration Examples

### Example 1: Development Environment (Permissive)

```json
{
  "allowed_directories": [
    "/tmp",
    "/var/tmp",
    "/home",
    "/opt",
    "/srv",
    "/data"
  ],
  "allowed_modules": [],
  "blocked_commands": [
    "rm -rf /",
    "mkfs",
    "format"
  ],
  "max_file_size": 104857600,
  "allowed_file_types": [],
  "audit_enabled": true,
  "log_level": "debug"
}
```

### Example 2: Production Environment (Restrictive)

```json
{
  "allowed_hosts": [
    "10.0.0.0/8",
    "192.168.1.0/24"
  ],
  "allowed_ports": [22, 443],
  "allowed_directories": [
    "/opt/app",
    "/var/log",
    "/var/backup"
  ],
  "blocked_directories": [
    "/etc/passwd",
    "/etc/shadow",
    "/boot",
    "/sys",
    "/proc",
    "/root"
  ],
  "allowed_modules": [
    "copy",
    "file",
    "package",
    "service",
    "find",
    "stat"
  ],
  "blocked_commands": [
    "rm -rf",
    "format",
    "mkfs",
    "dd if=",
    ":(){ :|:& };:",
    "shutdown",
    "reboot",
    ".* | sh$",
    ".* && rm"
  ],
  "max_file_size": 10485760,
  "allowed_file_types": [
    ".conf",
    ".yml",
    ".yaml",
    ".json",
    ".log",
    ".txt",
    ".sh"
  ],
  "require_encryption": true,
  "audit_enabled": true,
  "log_level": "warn"
}
```

### Example 3: Web Server Deployment

```json
{
  "allowed_directories": [
    "/var/www",
    "/etc/nginx",
    "/var/log/nginx",
    "/tmp"
  ],
  "allowed_modules": [
    "copy",
    "file",
    "template",
    "service",
    "lineinfile"
  ],
  "blocked_commands": [
    "rm -rf /",
    "mkfs",
    "reboot",
    "shutdown"
  ],
  "max_file_size": 52428800,
  "allowed_file_types": [
    ".conf",
    ".html",
    ".css",
    ".js",
    ".json",
    ".txt",
    ".log"
  ],
  "audit_enabled": true,
  "log_level": "info"
}
```

### Example 4: Database Server

```json
{
  "allowed_directories": [
    "/var/lib/mysql",
    "/var/lib/postgresql",
    "/var/backup",
    "/tmp"
  ],
  "allowed_modules": [
    "file",
    "copy",
    "service",
    "command"
  ],
  "blocked_commands": [
    "rm -rf /",
    "format",
    "mkfs",
    "dd if=",
    "reboot"
  ],
  "max_file_size": 104857600,
  "allowed_file_types": [
    ".sql",
    ".conf",
    ".log",
    ".txt",
    ".gz",
    ".tar"
  ],
  "require_encryption": true,
  "audit_enabled": true,
  "log_level": "info"
}
```

---

## Troubleshooting

### Error: "path is not in allowed directories"

**Cause:** Your playbook is trying to access a file outside of allowed directories.

**Solution:**

1. Check which directory you're accessing in your playbook
2. Add it to `allowed_directories` in security policy:

```json
{
  "allowed_directories": [
    "/tmp",
    "/home",
    "/opt",
    "/srv/myapp"    // Add your directory here
  ]
}
```

3. Restart Onigirazu

### Error: "blocked command detected"

**Cause:** Your playbook contains a blocked command.

**Solution:**

1. Check the error message for the blocked command
2. Either:
   - Remove the command from the playbook
   - Remove it from `blocked_commands` if it's safe:

```json
{
  "blocked_commands": [
    // Remove the line with your command
  ]
}
```

### Error: "module not allowed"

**Cause:** You're trying to use a module that's not whitelisted.

**Solution:**

1. Identify the module name from error message
2. Add it to `allowed_modules`:

```json
{
  "allowed_modules": [
    "existing_module",
    "your_new_module"    // Add here
  ]
}
```

### Error: "host not allowed"

**Cause:** Trying to connect to a host not in allowed list.

**Solution:**

Add the host to `allowed_hosts`:

```json
{
  "allowed_hosts": [
    "192.168.1.*",
    "new-server.example.com"    // Add here
  ]
}
```

### Error: "port not allowed"

**Cause:** Trying to connect to a port not in allowed list.

**Solution:**

```json
{
  "allowed_ports": [22, 80, 443, 8080],  // Add your port number
}
```

### Error: "file type not allowed"

**Cause:** Trying to work with a file extension not in allowed list.

**Solution:**

```json
{
  "allowed_file_types": [
    ".yml",
    ".py",
    ".myext"    // Add your extension
  ]
}
```

---

## Security Best Practices

### 1. Principle of Least Privilege

Only allow what's necessary:

```json
{
  "allowed_directories": ["/opt/app"],      // NOT /tmp
  "allowed_modules": ["copy", "file"],      // NOT all modules
  "allowed_ports": [22],                    // NOT all ports
}
```

### 2. Explicit Blocking

Block dangerous operations explicitly:

```json
{
  "blocked_commands": [
    "rm -rf /",
    "mkfs",
    "dd if=",
    "reboot",
    "shutdown"
  ]
}
```

### 3. Audit Everything

Always enable audit logging in production:

```json
{
  "audit_enabled": true,
  "log_level": "info"
}
```

### 4. File Size Limits

Prevent large file transfers that could affect performance:

```json
{
  "max_file_size": 10485760    // 10 MB limit
}
```

### 5. Require Encryption

In production, always require encryption:

```json
{
  "require_encryption": true
}
```

### 6. File Type Validation

Only allow necessary file types:

```json
{
  "allowed_file_types": [
    ".yml",
    ".conf",
    ".log"
  ]
}
```

---

## Integration with Main Config

Security policies are typically in separate files, but can be referenced in main config:

**onigirazu.yml:**

```yaml
# Main configuration file
log_level: info

# Note: security-policy.json is loaded separately
# from /etc/onigirazu/security-policy.json
```

**security-policy.json:**

```json
{
  "allowed_directories": ["/opt/app"],
  "audit_enabled": true
}
```

---

## See Also

- [CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md) - Main configuration guide
- [LOOPS_GUIDE.md](LOOPS_GUIDE.md) - Working with loops and file operations
