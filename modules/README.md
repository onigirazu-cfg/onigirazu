# Core Modules Documentation

This document provides comprehensive documentation for all built-in modules in Onigirazu.

## 📋 Table of Contents

- [Module Overview](#module-overview)
- [System Modules](#-system-modules)
- [File System Modules](#-file-system-modules)
- [Configuration Modules](#-configuration-modules)
- [Service Modules](#-service-modules)
- [System Control Modules](#-system-control-modules)
- [Package Modules](#package-modules)
- [Network Modules](#network-modules)
- [System Connectivity](#-system-connectivity)
- [Version Control](#-version-control)
- [Scheduled Jobs](#-scheduled-jobs)
- [Security & Firewall](#-security--firewall)
- [Container Management](#-container-management)
- [Database Management](#-database-management)
- [Utility Modules](#-utility-modules)

## 🔧 Module Overview

Onigirazu modules are the building blocks for automation tasks. Each module performs specific operations on target hosts and returns structured results.

### Module Structure

```yaml
- name: "Task Name"
  module_name:
    parameter1: value1
    parameter2: value2
  when: "condition"
  register: "variable_name"
```

### Common Parameters

All modules support these common parameters:

| Parameter | Type | Description |
|-----------|------|-------------|
| `when` | string | Conditional execution |
| `register` | string | Store result in variable |
| `ignore_errors` | boolean | Continue on failure |
| `timeout` | duration | Task timeout |
| `retries` | integer | Retry attempts |
| `delay` | duration | Delay between retries |

## 🖥️ System Modules

### facts

Gather system information and facts about target hosts.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `filter` | string | `*` | Filter facts by pattern |
| `gather_subset` | list | `all` | Subset of facts to gather |
| `timeout` | duration | `30s` | Gathering timeout |

#### Subsets

- `all`: All available facts
- `hardware`: Hardware information
- `network`: Network configuration
- `virtual`: Virtualization facts
- `ohai`: Ohai-style facts
- `facter`: Facter-style facts

#### Example

```yaml
- name: "Gather system facts"
  facts:
    gather_subset:
      - "hardware"
      - "network"
    filter: "onigirazu_*"
```

#### Return Values

```json
{
  "onigirazu_facts": {
    "onigirazu_hostname": "webserver01",
    "onigirazu_os_family": "RedHat",
    "onigirazu_distribution": "CentOS",
    "onigirazu_distribution_version": "8.4",
    "onigirazu_architecture": "x86_64",
    "onigirazu_processor_count": 4,
    "onigirazu_memtotal_mb": 8192,
    "onigirazu_interfaces": ["eth0", "lo"],
    "onigirazu_default_ipv4": {
      "address": "192.168.1.100",
      "gateway": "192.168.1.1",
      "interface": "eth0"
    }
  }
}
```

### command

Execute commands on target hosts.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cmd` | string | - | Command to execute (required) |
| `chdir` | string | - | Change directory before execution |
| `creates` | string | - | Skip if file exists |
| `removes` | string | - | Skip if file doesn't exist |
| `warn` | boolean | `true` | Show warnings for dangerous commands |

#### Example

```yaml
- name: "Check disk usage"
  command:
    cmd: "df -h"
  register: "disk_usage"

- name: "Create backup directory"
  command:
    cmd: "mkdir -p /backup/{{ onigirazu_date_time.date }}"
    creates: "/backup/{{ onigirazu_date_time.date }}"
```

#### Return Values

```json
{
  "cmd": "df -h",
  "stdout": "/dev/sda1  20G  5.5G   14G  30% /\n...",
  "stderr": "",
  "rc": 0,
  "start": "2023-10-01 10:30:00",
  "end": "2023-10-01 10:30:01",
  "delta": "0:00:01.234567"
}
```

### shell

Execute shell commands with full shell features.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cmd` | string | - | Shell command (required) |
| `chdir` | string | - | Working directory |
| `executable` | string | `/bin/sh` | Shell executable |
| `stdin` | string | - | Input to command |

#### Example

```yaml
- name: "Process log files"
  shell:
    cmd: |
      for log in /var/log/*.log; do
        if [ -f "$log" ]; then
          echo "Processing $log"
          gzip "$log"
        fi
      done
    chdir: "/var/log"
```

### script

Execute local scripts on remote hosts.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | Path to local script (required) |
| `args` | list | - | Script arguments |
| `interpreter` | string | - | Script interpreter |
| `creates` | string | - | Skip if file exists |
| `removes` | string | - | Skip if file doesn't exist |

#### Example

```yaml
- name: "Run deployment script"
  script:
    path: "./scripts/deploy.sh"
    args:
      - "production"
      - "v1.2.3"
    interpreter: "/bin/bash"
```

## 📁 File System Modules

### file

Manage files and directories.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | File/directory path (required) |
| `state` | string | `file` | Desired state |
| `mode` | string | - | File permissions |
| `owner` | string | - | File owner |
| `group` | string | - | File group |
| `recurse` | boolean | `false` | Apply recursively |

#### States

- `file`: Ensure file exists
- `directory`: Ensure directory exists
- `absent`: Remove file/directory
- `touch`: Touch file (update timestamps)
- `hard`: Create hard link
- `link`: Create symbolic link

#### Example

```yaml
- name: "Create application directory"
  file:
    path: "/opt/myapp"
    state: "directory"
    mode: "0755"
    owner: "appuser"
    group: "appgroup"

- name: "Create symbolic link"
  file:
    src: "/opt/myapp/current"
    dest: "/opt/myapp/releases/v1.2.3"
    state: "link"
```

### copy

Copy files to target hosts.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `src` | string | - | Source file path |
| `dest` | string | - | Destination path (required) |
| `content` | string | - | File content (alternative to src) |
| `backup` | boolean | `false` | Create backup |
| `mode` | string | - | File permissions |
| `owner` | string | - | File owner |
| `group` | string | - | File group |
| `force` | boolean | `true` | Overwrite existing files |

#### Example

```yaml
- name: "Copy configuration file"
  copy:
    src: "./config/app.conf"
    dest: "/etc/myapp/app.conf"
    backup: true
    mode: "0644"
    owner: "root"
    group: "root"

- name: "Create file with content"
  copy:
    content: |
      server {
        listen 80;
        server_name {{ inventory_hostname }};
        root /var/www/html;
      }
    dest: "/etc/nginx/sites-available/default"
    mode: "0644"
```

### find

Discover files on target hosts using glob patterns and file type filtering. Returns structured results for use in loops and conditionals.

**Available since v1.51.0** ✨ - Native file discovery module with glob pattern support

#### Parameters

| Parameter | Type | Default | Required | Description |
|-----------|------|---------|----------|-------------|
| `path` | string | - | **YES** | Directory path to search (must be non-empty) |
| `pattern` | string | `*` | NO | Glob pattern for file names (e.g., `*.log`, `temp_*`) |
| `type` | string | `file` | NO | Filter by file type: `file`, `directory`, `link`, `socket`, `pipe`, `block`, `char` |
| `limit` | integer | `0` | NO | Maximum files to return (0 = unlimited, capped at 999999) |

#### File Types

| Type Value | Description | Use Case |
|-----------|-------------|----------|
| `file` | Regular files | Most common - for log files, configs, scripts |
| `directory` | Directories only | Finding subdirectories |
| `link` | Symbolic links | Managing symlinks |
| `socket` | Socket files | System file discovery |
| `pipe` | Named pipes (FIFOs) | Advanced IPC mechanisms |
| `block` | Block devices | Device file discovery |
| `char` | Character devices | Device file discovery |

**Important**: If `type` is not specified or is empty, the module defaults to `file` type (regular files only), **not** all file types.

#### Examples

**Find and Process Log Files:**

```yaml
- name: "Find all log files in /var/log"
  find:
    path: "/var/log"
    pattern: "*.log"
    type: "file"
    limit: 100
  register: "log_files"

- name: "Display found log files"
  debug:
    msg: "Found {{ log_files.file_count }} log files"

- name: "Process each log file"
  copy:
    src: "{{ item.path }}"
    dest: "./logs/{{ item.name }}"
  loop: "{{ log_files.files }}"
  ignore_errors: true
```

**Find All Directories:**

```yaml
- name: "Find all directories in /etc"
  find:
    path: "/etc"
    type: "directory"
    limit: 50
  register: "directories"

- name: "List found directories"
  debug:
    msg: "Directory: {{ item.path }}"
  loop: "{{ directories.files }}"
  when: item.isdir
```

**Delete Temporary Files with Size Check:**

```yaml
- name: "Find and remove temporary files"
  find:
    path: "/tmp"
    pattern: "*.tmp"
    type: "file"
  register: "tmp_files"

- name: "Remove large temporary files (>10MB)"
  file:
    path: "{{ item.path }}"
    state: absent
  loop: "{{ tmp_files.files }}"
  when: "item.size | int > 10485760"  # Note: size is string, convert with | int
  ignore_errors: true
```

#### Return Values

```json
{
  "files": [
    {
      "path": "/var/log/syslog",
      "name": "syslog",
      "type": "file",
      "isfile": true,
      "isdir": false,
      "islink": false,
      "size": "1024576",
      "mode": "0644",
      "mtime": "1696086600"
    }
  ],
  "file_count": 1
}
```

**Field Definitions:**

| Field | Type | Description |
|-------|------|-------------|
| **path** | string | Full absolute path to the file |
| **name** | string | Filename only (basename) |
| **type** | string | File type: `file`, `directory`, `link`, `socket`, `pipe`, `block`, `char`, `other` |
| **isfile** | boolean | True if regular file (camelCase - not `is_file`) |
| **isdir** | boolean | True if directory (camelCase - not `is_dir`) |
| **islink** | boolean | True if symbolic link (camelCase - not `is_link`) |
| **size** | string | File size in bytes (returned as **string**, convert with `\| int` for math) |
| **mode** | string | File permissions in octal (e.g., `0644`) |
| **mtime** | string | Modification time as Unix timestamp in seconds (e.g., `1696086600`) |
| **file_count** | integer | Total number of files in results array |

**Important Field Notes:**

- **size**: Returned as STRING, not number. Use `{{ item.size \| int }}` for comparisons
- **mtime**: Unix timestamp (seconds since epoch), not ISO8601 format
- **Field names**: Boolean shortcuts use camelCase: `isfile`, `isdir`, `islink` (not snake_case)
- **type**: For socket/pipe/block/char types, use: `{{ item.type == "socket" }}`
- **Default type**: If not specified, only regular `file` type is returned, not all types

#### Use Cases

- **Log Management**: Find and process log files by pattern or size
- **Backup Operations**: Locate files for backup with size filtering and pattern matching
- **Cleanup Tasks**: Find and remove temporary or old files with conditional logic
- **Deployment**: Discover configuration files across directories for verification
- **Monitoring**: Locate specific file types for analysis and reporting

#### Common Patterns

```yaml
# Find all Python files
- find:
    path: "/opt/app"
    pattern: "*.py"
    type: "file"

# Find configuration directories
- find:
    path: "/etc"
    type: "directory"
    limit: 20

# Find files and process with loop
- find:
    path: "/tmp"
    pattern: "*.tmp"
  register: "tmp_files"

- file:
    path: "{{ item.path }}"
    state: absent
  loop: "{{ tmp_files.files }}"

# Find files by type and size
- find:
    path: "/home"
    type: "file"
  register: "files"

- debug:
    msg: "Large file: {{ item.name }} ({{ item.size | int / 1024 | int }}KB)"
  loop: "{{ files.files }}"
  when: "item.size | int > 102400"  # 100KB
```

#### Troubleshooting

**No files returned but expect results?**

- Does the path exist? Module returns empty array for non-existent paths (no error)
- Is the pattern correct? Use `*` to match everything
- Check the type filter - default is `file` only, not all types

**Field names not working (is_file vs isfile)?**

- Use camelCase: `item.isfile` not `item.is_file`
- Boolean shortcuts available for: `isfile`, `isdir`, `islink` only

**Can't compare file sizes?**

- Size is returned as STRING: `{{ item.size \| int > 1000000 }}`

**Checking for socket/pipe/block/char types?**

- Use the type field: `{{ item.type == "socket" }}`

### template

Process Jinja2 templates and copy to target hosts.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `src` | string | - | Template file path (required) |
| `dest` | string | - | Destination path (required) |
| `backup` | boolean | `false` | Create backup |
| `mode` | string | - | File permissions |
| `owner` | string | - | File owner |
| `group` | string | - | File group |
| `trim_blocks` | boolean | `true` | Trim template blocks |
| `lstrip_blocks` | boolean | `false` | Strip leading whitespace |

#### Example

```yaml
- name: "Deploy application configuration"
  template:
    src: "./templates/app.conf.j2"
    dest: "/etc/myapp/app.conf"
    backup: true
    mode: "0644"
  vars:
    database_host: "{{ groups['database'][0] }}"
    database_port: 5432
```

### fetch

Fetch files from target hosts to local machine.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `src` | string | - | Source file path (required) |
| `dest` | string | - | Local destination (required) |
| `flat` | boolean | `false` | Store without host directory |
| `fail_on_missing` | boolean | `true` | Fail if source missing |
| `validate_checksum` | boolean | `true` | Validate file integrity |

#### Example

```yaml
- name: "Fetch log files"
  fetch:
    src: "/var/log/myapp.log"
    dest: "./logs/"
    flat: false

- name: "Backup configuration"
  fetch:
    src: "/etc/myapp/app.conf"
    dest: "./backups/{{ inventory_hostname }}-app.conf"
    flat: true
```

## ⚙️ Configuration Modules

### config

Manage configuration files in various formats.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | Configuration file path (required) |
| `format` | string | `auto` | Configuration format |
| `action` | string | `set` | Action to perform |
| `key` | string | - | Configuration key |
| `value` | any | - | Configuration value |
| `backup` | boolean | `false` | Create backup |
| `create` | boolean | `false` | Create file if missing |

#### Formats

- `json`: JSON format
- `yaml`: YAML format
- `ini`: INI format
- `toml`: TOML format
- `xml`: XML format
- `auto`: Auto-detect format

#### Actions

- `set`: Set configuration value
- `get`: Get configuration value
- `delete`: Delete configuration key
- `merge`: Merge configuration
- `backup`: Create backup
- `restore`: Restore from backup
- `validate`: Validate configuration

#### Example

```yaml
- name: "Update database configuration"
  config:
    path: "/etc/myapp/config.yml"
    format: "yaml"
    action: "set"
    key: "database.host"
    value: "{{ database_host }}"
    backup: true

- name: "Merge configuration"
  config:
    path: "/etc/myapp/config.json"
    format: "json"
    action: "merge"
    value:
      logging:
        level: "info"
        file: "/var/log/myapp.log"
      cache:
        enabled: true
        ttl: 3600
```

### lineinfile

Manage single lines in text files.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | File path (required) |
| `line` | string | - | Line content |
| `regexp` | string | - | Regular expression pattern |
| `state` | string | `present` | Line state |
| `insertafter` | string | - | Insert after pattern |
| `insertbefore` | string | - | Insert before pattern |
| `backup` | boolean | `false` | Create backup |
| `create` | boolean | `false` | Create file if missing |

#### Example

```yaml
- name: "Add user to sudoers"
  lineinfile:
    path: "/etc/sudoers"
    line: "myuser ALL=(ALL) NOPASSWD: ALL"
    regexp: "^myuser"
    backup: true

- name: "Configure SSH"
  lineinfile:
    path: "/etc/ssh/sshd_config"
    regexp: "^#?PasswordAuthentication"
    line: "PasswordAuthentication no"
    backup: true
```

### blockinfile

Manage blocks of text in files.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | File path (required) |
| `block` | string | - | Block content |
| `marker` | string | `# {mark} ANSIBLE MANAGED BLOCK` | Block markers |
| `insertafter` | string | - | Insert after pattern |
| `insertbefore` | string | - | Insert before pattern |
| `state` | string | `present` | Block state |
| `backup` | boolean | `false` | Create backup |
| `create` | boolean | `false` | Create file if missing |

#### Example

```yaml
- name: "Configure application block"
  blockinfile:
    path: "/etc/hosts"
    block: |
      # Application servers
      192.168.1.10 app1.example.com
      192.168.1.11 app2.example.com
      192.168.1.12 app3.example.com
    marker: "# {mark} APPLICATION SERVERS"
    backup: true
```

## 🔧 Service Modules

### service

Manage system services.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Service name (required) |
| `state` | string | - | Service state |
| `enabled` | boolean | - | Enable at boot |
| `daemon_reload` | boolean | `false` | Reload systemd daemon |
| `scope` | string | `system` | Service scope |

#### States

- `started`: Start service
- `stopped`: Stop service
- `restarted`: Restart service
- `reloaded`: Reload service configuration

#### Example

```yaml
- name: "Start and enable nginx"
  service:
    name: "nginx"
    state: "started"
    enabled: true

- name: "Restart application service"
  service:
    name: "myapp"
    state: "restarted"
    daemon_reload: true
```

### systemd

Advanced systemd service management.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Service name (required) |
| `state` | string | - | Service state |
| `enabled` | boolean | - | Enable at boot |
| `masked` | boolean | - | Mask service |
| `daemon_reload` | boolean | `false` | Reload daemon |
| `scope` | string | `system` | Service scope |
| `user` | string | - | User for user services |

#### Example

```yaml
- name: "Configure systemd service"
  systemd:
    name: "myapp.service"
    state: "started"
    enabled: true
    daemon_reload: true
    scope: "system"

- name: "Mask unwanted service"
  systemd:
    name: "unwanted.service"
    masked: true
```

## 🎛️ System Control Modules

### sysctl

Manage kernel parameters via sysctl. Configure kernel tuning for performance and system behavior.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Sysctl key (required) |
| `value` | string | - | Sysctl value (required) |
| `state` | string | `present` | Parameter state (`present` or `absent`) |
| `sysctl_file` | string | `/etc/sysctl.d/99-onigirazu.conf` | Configuration file for persistence |
| `persist` | boolean | `true` | Persist parameter to sysctl file |
| `reload` | boolean | `true` | Reload sysctl settings after change |

#### Example

```yaml
- name: "Enable IP forwarding"
  sysctl:
    name: "net.ipv4.ip_forward"
    value: "1"
    state: "present"
    persist: true

- name: "Configure TCP parameters"
  sysctl:
    name: "net.ipv4.tcp_max_syn_backlog"
    value: "2048"
    sysctl_file: "/etc/sysctl.d/network.conf"

- name: "Remove custom kernel parameter"
  sysctl:
    name: "kernel.custom_param"
    state: "absent"
```

#### Return Values

```json
{
  "sysctl_key": "net.ipv4.ip_forward",
  "current_value": "0",
  "desired_value": "1",
  "changed": true,
  "msg": "Kernel parameter net.ipv4.ip_forward set to 1",
  "persisted_to_file": "/etc/sysctl.d/99-onigirazu.conf"
}
```

#### Common Use Cases

```yaml
# Enable IP forwarding for router
- sysctl:
    name: "net.ipv4.ip_forward"
    value: "1"

# Increase max connections for web server
- sysctl:
    name: "net.core.somaxconn"
    value: "4096"

# Tune TCP parameters
- sysctl:
    name: "net.ipv4.tcp_max_syn_backlog"
    value: "2048"

# Configure memory management
- sysctl:
    name: "vm.swappiness"
    value: "10"
```

### reboot

Reboot the system with optional pre-reboot checks and delays.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `pre_reboot_delay` | integer | `0` | Delay in seconds before reboot |
| `msg` | string | "System will reboot in a few seconds" | Reboot message |
| `test_boot` | boolean | `false` | Test boot without rebooting |
| `reboot_command` | string | - | Custom reboot command |

#### Example

```yaml
- name: "Reboot system immediately"
  reboot:

- name: "Reboot with delay and notification"
  reboot:
    pre_reboot_delay: 60
    msg: "System maintenance - rebooting in 1 minute"

- name: "Test boot check"
  reboot:
    test_boot: true

- name: "Custom reboot procedure"
  reboot:
    reboot_command: "shutdown -r now"
    pre_reboot_delay: 30
```

#### Return Values

```json
{
  "host": "server01",
  "reboot_initiated": true,
  "msg": "System reboot scheduled to start in 1 minute",
  "changed": true
}
```

#### Important Notes

- Reboot is scheduled with `shutdown -r +1` (1 minute delay) to allow playbook to complete
- Pre-reboot notifications are sent via `wall` command if delay is set
- The module execution returns before actual reboot occurs
- Use in playbooks with proper error handling

### mount

Control active and persistent filesystem mounts. Manage mount points in /etc/fstab and current mount status.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | Mount point path (required) |
| `src` | string | - | Device/source for state=present |
| `state` | string | `present` | Mount state |
| `fstype` | string | `defaults` | Filesystem type |
| `opts` | string | `defaults` | Mount options |
| `backup` | boolean | `true` | Backup /etc/fstab before changes |

#### States

- `present`: Add to fstab and mount
- `absent`: Remove from fstab and unmount
- `mounted`: Ensure filesystem is mounted
- `unmounted`: Ensure filesystem is unmounted

#### Example

```yaml
- name: "Mount NFS share"
  mount:
    path: "/mnt/nfs"
    src: "192.168.1.100:/export/data"
    state: "present"
    fstype: "nfs"
    opts: "defaults,nfsvers=4.0,hard,intr"

- name: "Mount USB drive"
  mount:
    path: "/mnt/usb"
    src: "/dev/sdb1"
    state: "present"
    fstype: "ext4"
    opts: "noatime,defaults"

- name: "Unmount temporary mount"
  mount:
    path: "/mnt/tmp"
    state: "absent"

- name: "Ensure data partition is mounted"
  mount:
    path: "/data"
    state: "mounted"
```

#### Return Values

```json
{
  "path": "/mnt/nfs",
  "src": "192.168.1.100:/export/data",
  "fstype": "nfs",
  "opts": "defaults,nfsvers=4.0",
  "changed": true,
  "msg": "Mount point /mnt/nfs configured and mounted",
  "mounted": true,
  "added_to_fstab": true
}
```

#### Common Use Cases

```yaml
# Production NFS mount with HA options
- mount:
    path: "/data"
    src: "nfs-server:/export/prod"
    fstype: "nfs"
    opts: "defaults,hard,intr,bg,nfsvers=4"
    state: "present"

# Data drive with optimizations
- mount:
    path: "/var/lib/mysql"
    src: "/dev/sdb1"
    fstype: "ext4"
    opts: "noatime,nodiratime,defaults"
    state: "present"

# Loop device or ISO mount
- mount:
    path: "/mnt/iso"
    src: "/path/to/image.iso"
    fstype: "iso9660"
    opts: "ro,loop"
    state: "present"
```

#### Troubleshooting

- **Mount fails after adding to fstab**: Check filesystem type and options are correct
- **Permission denied**: Ensure running with appropriate privileges (sudo/become)
- **Device not found**: Verify source path/device exists and is accessible

## 📦 Package Modules

### package

Universal package management.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string/list | - | Package name(s) (required) |
| `state` | string | `present` | Package state |
| `version` | string | - | Specific version |
| `update_cache` | boolean | `false` | Update package cache |
| `cache_valid_time` | integer | `0` | Cache validity time |

#### States

- `present`: Install package
- `absent`: Remove package
- `latest`: Install latest version

#### Example

```yaml
- name: "Install web server packages"
  package:
    name:
      - "nginx"
      - "php-fpm"
      - "mysql-client"
    state: "present"
    update_cache: true

- name: "Install specific version"
  package:
    name: "docker-ce"
    version: "20.10.17"
    state: "present"
```

### apt

Debian/Ubuntu package management.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string/list | - | Package name(s) (optional if only updating cache) |
| `state` | string | `present` | Package state (`present`, `latest`, or `absent`) |
| `update_cache` | boolean | `false` | Update apt cache before operation |
| `autoremove` | boolean | `false` | Remove unused packages |
| `autoclean` | boolean | `false` | Clean package cache |

#### Example

```yaml
- name: "Update package cache"
  apt:
    update_cache: true
    cache_valid_time: 3600

- name: "Upgrade all packages"
  apt:
    upgrade: "dist"
    update_cache: true
    autoremove: true
```

### yum

RedHat/CentOS package management.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string/list | - | Package name(s) (optional if only updating cache) |
| `state` | string | `present` | Package state (`present`, `latest`, or `absent`) |
| `enablerepo` | string | - | Enable specific repository |
| `disablerepo` | string | - | Disable specific repository |
| `security` | boolean | `false` | Install only security updates |
| `update_cache` | boolean | `false` | Update yum cache |

#### Example

```yaml
- name: "Install development tools"
  yum:
    name: "@Development Tools"
    state: "present"

- name: "Install from specific repo"
  yum:
    name: "docker-ce"
    state: "present"
    enablerepo: "docker-ce-stable"
```

## 🌐 Network Modules

### uri

Interact with HTTP/HTTPS services.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `url` | string | - | Request URL (required) |
| `method` | string | `GET` | HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD) |
| `body` | string/dict | - | Request body (string or dict for JSON) |
| `body_format` | string | `raw` | Body format (`raw` or `json`) |
| `headers` | dict | - | Custom HTTP headers |
| `user` | string | - | Username for basic authentication |
| `password` | string | - | Password for basic authentication |
| `timeout` | integer | `30` | Request timeout in seconds |
| `status_code` | list | - | Acceptable HTTP status codes |

#### Example

```yaml
- name: "Check API health"
  uri:
    url: "https://api.example.com/health"
    method: "GET"
    timeout: 10
  register: "health_check"

- name: "Send webhook notification"
  uri:
    url: "https://hooks.slack.com/services/..."
    method: "POST"
    body_format: "json"
    body:
      text: "Deployment completed successfully"
    headers:
      Content-Type: "application/json"
```

### get_url

Download files from HTTP/HTTPS/FTP.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `url` | string | - | Download URL (required) |
| `dest` | string | - | Destination path (required) |
| `mode` | string | - | File permissions |
| `owner` | string | - | File owner |
| `group` | string | - | File group |
| `backup` | boolean | `false` | Create backup |
| `force` | boolean | `false` | Force download |
| `timeout` | integer | `30` | Download timeout |
| `validate_certs` | boolean | `true` | Validate SSL certificates |

#### Example

```yaml
- name: "Download application binary"
  get_url:
    url: "https://releases.example.com/myapp/v1.2.3/myapp-linux-amd64"
    dest: "/usr/local/bin/myapp"
    mode: "0755"
    owner: "root"
    group: "root"
    backup: true
```

## 🔒 Security Modules

### user

Manage user accounts.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Username (required) |
| `state` | string | `present` | User state |
| `uid` | integer | - | User ID |
| `gid` | integer | - | Primary group ID |
| `groups` | list | - | Additional groups |
| `home` | string | - | Home directory |
| `shell` | string | - | Login shell |
| `password` | string | - | Encrypted password |
| `create_home` | boolean | `true` | Create home directory |
| `system` | boolean | `false` | System user |

#### Example

```yaml
- name: "Create application user"
  user:
    name: "appuser"
    uid: 1001
    group: "appgroup"
    home: "/opt/myapp"
    shell: "/bin/bash"
    create_home: true

- name: "Add user to groups"
  user:
    name: "myuser"
    groups:
      - "docker"
      - "sudo"
    append: true
```

### group

Manage user groups.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Group name (required) |
| `state` | string | `present` | Group state |
| `gid` | integer | - | Group ID |
| `system` | boolean | `false` | System group |

#### Example

```yaml
- name: "Create application group"
  group:
    name: "appgroup"
    gid: 1001
    system: false
```

### authorized_key

Manage SSH authorized keys.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `user` | string | - | Username (required) |
| `key` | string | - | SSH public key (required) |
| `state` | string | `present` | Key state |
| `path` | string | - | Custom authorized_keys path |
| `manage_dir` | boolean | `true` | Manage .ssh directory |
| `exclusive` | boolean | `false` | Remove other keys |
| `comment` | string | - | Key comment |

#### Example

```yaml
- name: "Add SSH key for user"
  authorized_key:
    user: "myuser"
    key: "{{ lookup('file', '~/.ssh/id_rsa.pub') }}"
    state: "present"
    comment: "Deployment key"

- name: "Set exclusive SSH keys"
  authorized_key:
    user: "appuser"
    key: |
      ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ... user1@host1
      ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ... user2@host2
    exclusive: true
```

## 🖥️ System Connectivity

### ping

Tests connectivity to target hosts.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `data` | string | `pong` | Custom response message |

#### Example

```yaml
- name: "Test connectivity to all hosts"
  ping:

- name: "Test with custom data"
  ping:
    data: "custom_response"
```

#### Return Values

```json
{
  "ping": "pong",
  "connection": "ssh",
  "host": "webserver01",
  "address": "192.168.1.100",
  "user": "ubuntu",
  "port": 22
}
```

### stat

Retrieve file or directory status information.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | File or directory path (required) |

#### Example

```yaml
- name: "Get file status"
  stat:
    path: "/etc/nginx/nginx.conf"
  register: "nginx_config"

- name: "Check if directory exists and show permissions"
  stat:
    path: "/opt/myapp"
  register: "app_dir"
```

#### Return Values

```json
{
  "stat": {
    "exists": true,
    "type": "file",
    "size": "4096",
    "mode": "0644",
    "mtime": "1696086600"
  },
  "exists": true
}
```

## 🔄 Version Control

### git

Manages Git repositories on target hosts.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `repo` | string | - | Git repository URL (required) |
| `dest` | string | - | Destination path (required) |
| `version` | string | `HEAD` | Branch, tag, or commit to checkout |
| `force` | boolean | `false` | Force overwrite if dest exists |
| `update` | boolean | `true` | Update existing repository |

#### Example

```yaml
- name: "Clone application repository"
  git:
    repo: "https://github.com/myorg/myapp.git"
    dest: "/opt/myapp"
    version: "main"

- name: "Checkout specific tag"
  git:
    repo: "git@github.com:myorg/myapp.git"
    dest: "/opt/myapp"
    version: "v1.2.3"
    update: true
```

## ⏰ Scheduled Jobs

### cron

Manage cron jobs and system crontabs.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `operation` | string | `job` | `job`, `file`, `system`, or `list` |
| `name` | string | - | Job name/comment |
| `job` | string | - | Job command to execute |
| `minute` | string | `*` | Minute (0-59) |
| `hour` | string | `*` | Hour (0-23) |
| `day` | string | `*` | Day of month (1-31) |
| `month` | string | `*` | Month (1-12) |
| `weekday` | string | `*` | Day of week (0-6) |
| `user` | string | `root` | Cron user |
| `state` | string | `present` | `present` or `absent` |
| `special_time` | string | - | Special time string (@reboot, @hourly, etc.) |

#### Examples

```yaml
- name: "Create daily backup job"
  cron:
    operation: "job"
    name: "Daily database backup"
    job: "/usr/local/bin/backup-db.sh"
    minute: "0"
    hour: "2"
    day: "*"
    state: "present"
    user: "backupuser"

- name: "Schedule task to run at reboot"
  cron:
    operation: "job"
    name: "Start application"
    job: "/opt/myapp/start.sh"
    special_time: "@reboot"
    user: "appuser"

- name: "List all cron jobs"
  cron:
    operation: "list"
  register: "cron_jobs"
```

## 🔥 Security & Firewall

### firewall

Manage firewall rules and services.

#### Supported Firewalls

- UFW (Ubuntu/Debian)
- firewalld (RHEL/CentOS)
- iptables (Generic Linux)

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `operation` | string | `rule` | `enable`, `disable`, `rule`, `service`, `source`, `list`, `reload` |
| `state` | string | `present` | `present` or `absent` |
| `port` | string | - | Port number or port range |
| `protocol` | string | `tcp` | `tcp`, `udp`, or both |
| `service` | string | - | Service name (http, ssh, etc.) |
| `source` | string | - | Source IP or network |
| `rule` | string | - | Custom firewall rule |

#### Examples

```yaml
- name: "Enable firewall"
  firewall:
    operation: "enable"

- name: "Allow SSH port"
  firewall:
    operation: "rule"
    port: "22"
    protocol: "tcp"
    state: "present"

- name: "Allow HTTP/HTTPS services"
  firewall:
    operation: "service"
    service: "http"
    state: "present"

- name: "Allow traffic from specific IP"
  firewall:
    operation: "source"
    source: "192.168.1.0/24"
    state: "present"

- name: "List all firewall rules"
  firewall:
    operation: "list"
  register: "fw_rules"
```

## 🐳 Container Management

### docker_container

Manage Docker containers.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Container name (required) |
| `image` | string | - | Docker image name |
| `state` | string | `started` | `present`, `started`, `stopped`, `absent` |
| `ports` | list | - | Port mappings (e.g., ["8080:80"]) |
| `volumes` | list | - | Volume mounts |
| `env` | dict | - | Environment variables |
| `restart_policy` | string | - | Restart policy |

#### Examples

```yaml
- name: "Create and run web container"
  docker_container:
    name: "myapp"
    image: "nginx:latest"
    state: "started"
    ports:
      - "8080:80"
    volumes:
      - "/var/www/html:/usr/share/nginx/html"
    env:
      ENVIRONMENT: "production"

- name: "Stop container"
  docker_container:
    name: "myapp"
    state: "stopped"
```

### docker_image

Manage Docker images.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Image name (required) |
| `tag` | string | `latest` | Image tag |
| `state` | string | `present` | `present` or `absent` |
| `force` | boolean | `false` | Force pull/removal |

#### Examples

```yaml
- name: "Pull latest nginx image"
  docker_image:
    name: "nginx"
    tag: "latest"
    state: "present"

- name: "Remove old image"
  docker_image:
    name: "myapp"
    tag: "v1.0.0"
    state: "absent"
```

### docker_compose

Manage Docker Compose applications.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `project_name` | string | - | Project name |
| `compose_file` | string | `docker-compose.yml` | Path to compose file |
| `state` | string | `present` | `present`, `started`, `stopped`, `absent` |

#### Examples

```yaml
- name: "Start docker-compose services"
  docker_compose:
    project_name: "mystack"
    compose_file: "/opt/mystack/docker-compose.yml"
    state: "started"

- name: "Stop services"
  docker_compose:
    project_name: "mystack"
    state: "stopped"
```

### podman

Manage Podman containers (Docker-compatible).

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Container name (required) |
| `image` | string | - | Container image |
| `state` | string | `started` | `started`, `stopped`, `absent` |
| `ports` | list | - | Port mappings |
| `volumes` | list | - | Volume mounts |

#### Examples

```yaml
- name: "Run podman container"
  podman:
    name: "myapp"
    image: "myapp:latest"
    state: "started"
    ports:
      - "8080:8080"
```

## 🗄️ Database Management

### mysql_db

Manage MySQL databases.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Database name (required) |
| `state` | string | `present` | `present` or `absent` |
| `collation` | string | `utf8mb4_general_ci` | Database collation |
| `encoding` | string | `utf8mb4` | Database encoding |

#### Examples

```yaml
- name: "Create application database"
  mysql_db:
    name: "myapp_db"
    state: "present"
    encoding: "utf8mb4"
    collation: "utf8mb4_unicode_ci"

- name: "Remove database"
  mysql_db:
    name: "temp_db"
    state: "absent"
```

### mysql_user

Manage MySQL users and permissions.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Username (required) |
| `host` | string | `%` | Host pattern |
| `password` | string | - | User password |
| `state` | string | `present` | `present` or `absent` |
| `priv` | string | - | Privileges (e.g., "mydb.*:ALL") |

#### Examples

```yaml
- name: "Create database user"
  mysql_user:
    name: "appuser"
    host: "192.168.%"
    password: "securepassword"
    priv: "myapp_db.*:ALL"
    state: "present"

- name: "Remove user"
  mysql_user:
    name: "tempuser"
    state: "absent"
```

### postgresql_db

Manage PostgreSQL databases.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Database name (required) |
| `state` | string | `present` | `present` or `absent` |
| `owner` | string | - | Database owner role |
| `encoding` | string | `UTF8` | Database encoding |

#### Examples

```yaml
- name: "Create PostgreSQL database"
  postgresql_db:
    name: "myapp_db"
    owner: "appuser"
    encoding: "UTF8"
    state: "present"
```

### postgresql_user

Manage PostgreSQL users and roles.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Username (required) |
| `password` | string | - | User password |
| `state` | string | `present` | `present` or `absent` |
| `priv` | string | - | Privileges |

#### Examples

```yaml
- name: "Create PostgreSQL user"
  postgresql_user:
    name: "appuser"
    password: "securepassword"
    priv: "myapp_db:ALL"
    state: "present"
```

### mongodb

Manage MongoDB databases and users.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Database or user name (required) |
| `operation` | string | `database` | `database` or `user` |
| `state` | string | `present` | `present` or `absent` |

#### Examples

```yaml
- name: "Create MongoDB database"
  mongodb:
    name: "myapp_db"
    operation: "database"
    state: "present"

- name: "Create MongoDB user"
  mongodb:
    name: "appuser"
    operation: "user"
    state: "present"
```

## 🛠️ Utility Modules

### debug

Print debug information.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `msg` | string | - | Debug message |
| `var` | string | - | Variable to display |
| `verbosity` | integer | `0` | Minimum verbosity level |

#### Example

```yaml
- name: "Debug variable"
  debug:
    var: "onigirazu_facts"

- name: "Debug message"
  debug:
    msg: "Current user is {{ onigirazu_user_id }}"
```

### set_fact

Set variables for use in subsequent tasks.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `key_value` | dict | - | Variables to set |
| `cacheable` | boolean | `false` | Cache across plays |

#### Example

```yaml
- name: "Set deployment facts"
  set_fact:
    deployment_time: "{{ onigirazu_date_time.iso8601 }}"
    app_version: "v1.2.3"
    environment: "production"
    cacheable: true
```

### wait_for

Wait for conditions to be met.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `port` | integer | - | Port number to wait for (check if listening) |
| `host` | string | `localhost` | Hostname/IP address to check |
| `path` | string | - | File path to wait for (check if exists) |
| `search_regex` | string | - | Regex pattern to search in file content |
| `state` | string | `started` | Condition state (`started` = expect condition met, `stopped` = expect condition failed) |
| `timeout` | integer | `300` | Maximum wait time in seconds |
| `delay` | integer | `0` | Initial delay before checking in seconds |

#### Example

```yaml
- name: "Wait for service to start"
  wait_for:
    port: 8080
    host: "{{ inventory_hostname }}"
    timeout: 60

- name: "Wait for log message"
  wait_for:
    path: "/var/log/myapp.log"
    search_regex: "Server started successfully"
    timeout: 120
```

### pause

Pause execution for user input or time.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `seconds` | integer | - | Pause duration in seconds |
| `minutes` | integer | - | Pause duration in minutes |
| `prompt` | string | - | User prompt message (waits for user input if provided) |

#### Example

```yaml
- name: "Pause for confirmation"
  pause:
    prompt: "Press enter to continue with deployment"

- name: "Wait before next step"
  pause:
    seconds: 30
```

### fail

Fail execution with custom message.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `msg` | string | `Failed as requested` | Failure message |

#### Example

```yaml
- name: "Fail if conditions not met"
  fail:
    msg: "Database connection failed"
  when: "database_check.rc != 0"
```

## 📚 Complete Module List

All available modules in Onigirazu (40 total - v1.55.0+):

**System & Connectivity**: command, shell, script, ping, facts, debug, set_fact, wait_for, pause, fail
**File Management**: file, copy, fetch, find, template, lineinfile, blockinfile, stat
**Package Management**: package, apt, yum
**Service Management**: service, systemd, cron
**Security & Firewall**: firewall, authorized_key
**Version Control**: git
**Configuration**: config
**Containers**: docker_container, docker_image, docker_compose, podman
**Databases**: mysql_db, mysql_user, postgresql_db, postgresql_user, mongodb
**Network**: get_url, uri
**User Management**: user, group

### New in v1.55.0

The following 9 modules were added to complete the module coverage:

- **fail**: Fail execution with custom message (Control Flow)
- **pause**: Pause execution for user input or time duration (Control Flow)
- **wait_for**: Wait for conditions like ports, files, or regex patterns (System Utility)
- **script**: Execute local scripts on remote hosts (Execution)
- **authorized_key**: Manage SSH public keys (Security)
- **blockinfile**: Insert/update/remove multi-line text blocks (File Management)
- **apt**: Debian/Ubuntu package management (Package Management)
- **yum**: RedHat/CentOS package management (Package Management)
- **uri**: HTTP/HTTPS API requests with full method support (Network)

---

This comprehensive module documentation provides detailed information about all built-in modules, their parameters, usage examples, and return values. Each module is designed to be idempotent and provide consistent behavior across different platforms.
