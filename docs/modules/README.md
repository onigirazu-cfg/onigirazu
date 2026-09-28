# Core Modules Documentation

This document provides comprehensive documentation for all built-in modules in Onigirazu.

> 📑 **Looking for a specific module?** Check the [Alphabetical Index](INDEX.md) for a quick reference of all 50 modules.

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

### Task Keywords

Every task accepts these keywords next to its module:

| Keyword | Description |
|---------|-------------|
| `when` | Run only when the expression is true |
| `register` | Store the result under this name (also for failed or skipped tasks) |
| `loop`, `with_items`, `with_list`, `with_dict`, `with_sequence`, `loop_control` | Repeat the task |
| `until`, `retries`, `delay` | Repeat until the expression is true |
| `changed_when`, `failed_when` | Override the changed/failed status |
| `ignore_errors` | Continue on failure |
| `notify`, `listen` | Handlers |
| `become`, `become_user`, `become_method` | Privilege escalation |
| `delegate_to`, `run_once`, `local_action` | Where and how often the task runs |
| `block`, `rescue`, `always` | Task groups with error handling |
| `check_mode` | `false` runs the task for real in a check run |
| `environment`, `vars`, `tags`, `no_log`, `timeout` | Task environment, variables, tags, hidden output, time limit |

`retries` must be a number and `delay`/`timeout` a number of seconds or a duration such as `10s`. `retries` and `delay` can also be templates (`retries: "{{ n }}"`), rendered per host.

### Short Forms

Ansible's short forms work:

```yaml
- command: echo hi chdir=/tmp       # free form; inline options: chdir, creates, removes, executable
- file: path=/tmp/x state=touch     # key=value pairs; \n, \t, \\ and \" in values are decoded
- ping:                             # no arguments
- command: make
  args:
    chdir: /src
- local_action: command hostname    # runs on the control machine (delegate_to: localhost)
```

### Argument Values

- A YAML integer given as `mode` is octal, as in Ansible: `mode: 0644` and `mode: "0644"` are the same.
- Other numbers are passed to modules as strings, so `minute: 0` works in cron.

### Check Mode

With `--check` only these modules run, reporting what they would change: ping, debug, set_fact, stat, find, fail, wait_for, assert, include_vars, file, copy, template, lineinfile, blockinfile, replace, apt, yum, package, service, user, group, cron, sysctl, get_url, git, systemd, mount, config, timezone, unarchive, apt_repository, apt_key, docker_container, podman, docker_image. Every other module (command, shell, script, uri, firewall, archive, fetch, reboot, authorized_key, docker_compose, database modules, ...) is skipped.

## 🖥️ System Modules

### Facts

There is no facts module. Facts are gathered at the start of every play unless the play sets `gather_facts: false`. See the [Variables Cheat Sheet](../VARIABLES_CHEATSHEET.md) for their names (`ansible_*` and `onigirazu_*`).

### command

Run a command without a shell: pipes, redirections and variables such as `$HOME` are not interpreted. Use `shell` for those.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cmd` | string | - | Command to run (required; free form in the short form) |
| `chdir` | string | - | Change into this directory first |
| `creates` | string | - | Skip when this path exists |
| `removes` | string | - | Skip when this path does not exist |
| `environment` | dict | - | Extra environment variables |

A non-zero exit code fails the task; use `failed_when` or `ignore_errors` to accept it. The task always reports `changed`; use `changed_when: false` for read-only commands.

#### Example

```yaml
- name: "Show disk usage"
  command:
    cmd: "df -h /"
  register: disk
  changed_when: false

- name: "Build"
  command: make install chdir=/src creates=/usr/local/bin/app
```

#### Return Values

```json
{
  "cmd": "df -h /",
  "stdout": "Filesystem  Size  Used Avail Use% Mounted on\n...",
  "stderr": "",
  "rc": 0,
  "stdout_lines": ["Filesystem  Size  Used Avail Use% Mounted on", "..."],
  "stderr_lines": [],
  "start": "2026-09-26 10:30:00.000000",
  "end": "2026-09-26 10:30:00.042000",
  "delta": "42ms"
}
```

### shell

Run a command through a shell (`/bin/sh` unless `executable` is set). Same results and failure rule as `command`.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cmd` | string | - | Shell command (required; free form in the short form) |
| `chdir` | string | - | Working directory |
| `executable` | string | - | Shell to run the command with, e.g. `/bin/bash` |
| `creates` | string | - | Skip when this path exists |
| `removes` | string | - | Skip when this path does not exist |
| `environment` | dict | - | Extra environment variables |

#### Example

```yaml
- name: "Count failed logins"
  shell:
    cmd: "grep -c 'Failed password' /var/log/auth.log || true"
    executable: /bin/bash
  register: failed_logins
  changed_when: false
```

### script

Copy a local script to the host and run it with bash.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `script` | string | - | Path to the local script (required) |
| `args` | string | - | Arguments, separated by spaces |

#### Example

```yaml
- name: "Run deployment script"
  script:
    script: "./scripts/deploy.sh"
    args: "production v1.2.3"

- name: "Short form"
  script: ./scripts/deploy.sh production v1.2.3
```

## 📁 File System Modules

### file

Manage files, directories and links.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | Path (required); `dest` and `name` are aliases |
| `state` | string | see below | Desired state |
| `src` | string | - | Link target (`link`, `hard`) |
| `mode` | string | - | Permissions, e.g. `"0644"` |
| `owner` | string | - | Owner |
| `group` | string | - | Group |
| `recurse` | boolean | `false` | With `state: directory`: apply owner/group/mode to everything below |
| `force` | boolean | `false` | Replace an existing non-link path with the link |
| `modification_time`, `access_time` | string | - | With `state: touch`: `preserve` on both leaves an existing file untouched |

#### States

- `file`: the path must exist; only its attributes are set (fails otherwise)
- `directory`: create the directory (with parents)
- `touch`: create an empty file or update its timestamps (an existing file is reported unchanged)
- `present`: create the file if it is missing; with `content`, write that content when it differs
- `absent`: remove the file or directory
- `link`: symbolic link at `path` pointing to `src`
- `hard`: hard link at `path` pointing to `src`

Without `state` the module creates a link when `src` is given, and otherwise sets attributes of an existing path.

#### Example

```yaml
- name: "Create application directory"
  file:
    path: "/opt/myapp"
    state: "directory"
    mode: "0755"
    owner: "appuser"
    group: "appgroup"

- name: "Point current at a release"
  file:
    src: "/opt/myapp/releases/v1.2.3"
    dest: "/opt/myapp/current"
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
| `force` | boolean | `true` | `false` leaves an existing `dest` alone, whatever it contains |
| `remote_src` | boolean | `false` | `src` is a path on the host, not on the control machine |

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
| `src` | string | - | Template file path (`src` or `content` required) |
| `content` | string | - | Template text instead of a file |
| `dest` | string | - | Destination path (required) |
| `vars` | dict | - | Extra variables for this template |
| `backup` | boolean | `false` | Create backup |
| `mode` | string | `0644` | File permissions |
| `owner` | string | - | File owner |
| `group` | string | - | File group |
| `force` | boolean | `false` | Rewrite the file even when the content is unchanged |

As in Ansible, `trim_blocks: true` (default) removes the newline after a `{% ... %}` tag and
`lstrip_blocks: true` (default false) removes spaces before a tag at the start of a line (`{%+` keeps them);
`{%-`/`-%}` strip whitespace.

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
| `validate` | boolean | `true` | Compare checksums after the transfer |

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
| `state` | string | - | Service state; without it the service is not started or stopped |
| `enabled` | boolean | - | Enable at boot |
| `daemon_reload` | boolean | `false` | Reload systemd daemon |
| `scope` | string | `system` | Service scope |

One of `state` or `enabled` is required.

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

### archive

Create and manage compressed archives. Supports multiple formats including tar, gzip, bzip2, xz, and zip with glob pattern matching and selective exclusions.

**Available since v1.57.0** ✨ - Native archive creation module with glob patterns and exclusions

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string/list | - | Source file(s) or pattern(s) (required) |
| `dest` | string | - | Destination archive file (required) |
| `format` | string | `tar.gz` | Archive format: `tar`, `tar.gz`, `tar.bz2`, `tar.xz`, `zip` |
| `exclude` | list | `[]` | Exclude patterns (glob format) |
| `remove` | boolean | `false` | Remove source files after successful archiving |

#### Supported Formats

| Format | Description | File Extension | Use Case |
|--------|-------------|-----------------|----------|
| `tar` | Uncompressed tar archive | `.tar` | Large files, local transfers |
| `tar.gz` | Gzip compressed tar | `.tar.gz` | Default, good compression ratio |
| `tar.bz2` | Bzip2 compressed tar | `.tar.bz2` | Better compression, slower |
| `tar.xz` | XZ compressed tar | `.tar.xz` | Best compression, very slow |
| `zip` | ZIP archive | `.zip` | Cross-platform, Windows compatible |

#### Example

```yaml
- name: "Create simple tar.gz archive"
  archive:
    path: "/var/log/app"
    dest: "/backups/app-logs-{{ onigirazu_date_time.date }}.tar.gz"
    format: "tar.gz"

- name: "Archive multiple specific files"
  archive:
    path:
      - "/etc/nginx/nginx.conf"
      - "/etc/nginx/conf.d/"
    dest: "/backups/nginx-config.tar.gz"

- name: "Archive with exclusions"
  archive:
    path: "/opt/application"
    dest: "/backups/app-full.tar.gz"
    format: "tar.gz"
    exclude:
      - "*.tmp"
      - "cache/*"
      - "logs/*"

- name: "Archive and remove source files"
  archive:
    path: "/tmp/working_files"
    dest: "/archive/project-v1.2.0.zip"
    format: "zip"
    remove: true

- name: "Archive with glob patterns"
  archive:
    path:
      - "/var/log/*.log"
      - "/var/log/app/*"
    dest: "/backups/logs-archive.tar.bz2"
    format: "tar.bz2"
    exclude:
      - "debug.log"
```

#### Return Values

```json
{
  "archived": true,
  "archive_path": "/backups/app-logs-2025-10-28.tar.gz",
  "format": "tar.gz",
  "file_count": 42,
  "size_bytes": 5242880,
  "size_human": "5.0 MB",
  "files_archived": [
    "/var/log/app/access.log",
    "/var/log/app/error.log",
    "/var/log/app/debug.log"
  ],
  "changed": true,
  "msg": "Successfully created archive with 42 files"
}
```

#### Common Use Cases

```yaml
# Database backup
- archive:
    path: "/var/backups/database"
    dest: "/secure/backup-{{ onigirazu_date_time.epoch }}.tar.xz"
    format: "tar.xz"

# Log rotation archive
- archive:
    path: "/var/log"
    dest: "/archive/logs-{{ onigirazu_date_time.year }}-{{ onigirazu_date_time.month }}.tar.gz"
    exclude:
      - "current/*"
      - "*.sock"
    remove: false

# Application deployment archive
- archive:
    path:
      - "/opt/app/src"
      - "/opt/app/config"
      - "/opt/app/README.md"
    dest: "/releases/app-{{ app_version }}.zip"
    format: "zip"
    exclude:
      - "*.pyc"
      - "__pycache__"
      - ".git"
      - "node_modules"

# Cross-platform compatible backup
- archive:
    path: "/home/user/documents"
    dest: "/backup/docs-{{ onigirazu_date_time.date }}.zip"
    format: "zip"
    remove: false
```

#### Important Notes

- **Glob Patterns**: The `path` parameter supports glob patterns (e.g., `*.log`, `app_*/`)
- **Source Files**: Must exist and be readable by the executing user
- **Destination**: Parent directory must exist and be writable
- **Remove Safety**: Files are only removed after successful archive creation
- **Exclusions**: Exclude patterns use glob matching (e.g., `*.tmp`, `dir/*`)
- **Format Selection**: Choose `tar.xz` for maximum compression, `zip` for Windows compatibility

### unarchive

Extracts a tar (any compression tar understands) or zip archive into an existing directory.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `src` | string | - | Archive on the control machine (in a role: `files/`), or on the host with `remote_src` |
| `dest` | string | - | Directory to extract into; it must exist |
| `remote_src` | boolean | `false` | `src` is on the host |
| `creates` | string | - | Skip when this path exists |
| `owner`, `group` | string | - | Owner of the extracted files (recursive) |
| `extra_opts` | list | - | Extra options for `tar` / `unzip` |
| `list_files` | boolean | `false` | Return the archive's members in `files` |

The task is `ok` when every member of the archive already exists in `dest`; otherwise
the archive is extracted over it. Zip archives need `unzip` on the host.

```yaml
- name: Install the app
  unarchive:
    src: app-1.4.tar.gz
    dest: /opt/app
    owner: app
```

### timezone

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | IANA time zone, e.g. `Europe/Madrid`, `UTC` |

Uses `timedatectl` when systemd runs, else links `/etc/localtime` (and writes
`/etc/timezone` where it exists). Also `community.general.timezone`.

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
| `cache_valid_time` | int | - | Skip the cache update if it is younger than this many seconds |
| `upgrade` | string | `no` | `yes`/`safe` (apt-get upgrade), `full`/`dist` (dist-upgrade); predicted in check mode |
| `autoremove` | boolean | `false` | Remove unused packages |
| `autoclean` | boolean | `false` | Clean package cache |

#### Example

```yaml
- name: "Update package cache"
  apt:
    update_cache: true

- name: "Upgrade nginx to the latest version"
  apt:
    name: nginx
    state: latest
    update_cache: true
    autoremove: true

- name: "Upgrade all packages"
  apt:
    update_cache: true
    cache_valid_time: 3600
    upgrade: dist
  become: true
```

### apt_repository

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `repo` | string | - | A `deb ...` / `deb-src ...` line, or `ppa:owner/name` |
| `state` | string | `present` | `present` or `absent` |
| `filename` | string | from the URL | File in `/etc/apt/sources.list.d/` (without `.list`) |
| `update_cache` | boolean | `true` | Run `apt-get update` after a change |

A line is looked for in `/etc/apt/sources.list` and `sources.list.d/*.list`; a repository already
described in a deb822 `.sources` file (such as Ubuntu's own) also counts as present. `.sources` files are not edited. A new line goes to its own file, named as Ansible names it
(`download_docker_com_linux_ubuntu.list`); removing the last line of a `.list` file deletes it.
PPAs go through `add-apt-repository` (package `software-properties-common`).

### apt_key

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `url` / `data` / `file` / `keyserver` | string | - | Where the key comes from (one of them); `url` is fetched by the host with curl or wget, `file` is on the host, `keyserver` needs `id` and gpg |
| `keyring` | string | - | Absolute path to write the key to, e.g. `/etc/apt/keyrings/docker.asc` |
| `id` | string | - | Key id; names the file in `/etc/apt/trusted.gpg.d/` when there is no `keyring` |
| `state` | string | `present` | `present`, or `absent` (removes `keyring`, or the file named by `id`) |

`apt-key` is not used. An ASCII-armored key is written as is (use a `.asc` keyring); for a `.gpg`
keyring it is converted with `gpg --dearmor` on the host. With `keyserver` (and `id`) the host fetches the
key with gpg (`keyserver: hkps://keyserver.ubuntu.com`).

```yaml
- apt_key:
    url: https://download.docker.com/linux/ubuntu/gpg
    keyring: /etc/apt/keyrings/docker.asc
- apt_repository:
    repo: "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu {{ ansible_distribution_release }} stable"
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

Manage user accounts. An existing account is brought to the given settings.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Username (required) |
| `state` | string | `present` | `present` or `absent` |
| `uid` | integer | - | User ID |
| `group` | string | - | Primary group (name or GID) |
| `groups` | list/string | - | Supplementary groups |
| `append` | boolean | `false` | Add to `groups` instead of replacing the list |
| `home` | string | - | Home directory |
| `move_home` | boolean | `false` | Move the old home when `home` changes |
| `shell` | string | - | Login shell |
| `comment` | string | - | GECOS field |
| `password` | string | - | Encrypted password (see the `password_hash` filter) |
| `create_home` | boolean | `true` | Create the home directory (new accounts) |
| `system` | boolean | `false` | System account (new accounts) |
| `remove` | boolean | `false` | With `state: absent`: also remove the home directory |

`gid` is accepted only when the account is created; use `group` to change the primary group of an existing account.

#### Example

```yaml
- name: "Create application user"
  user:
    name: "appuser"
    uid: 1001
    group: "appgroup"
    home: "/opt/myapp"
    shell: "/bin/bash"

- name: "Add user to groups"
  user:
    name: "myuser"
    groups:
      - "docker"
      - "sudo"
    append: true

- name: "Remove a user with their home"
  user:
    name: "olduser"
    state: absent
    remove: true
```

### group

Manage groups.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Group name (required) |
| `state` | string | `present` | `present` or `absent` |
| `gid` | integer | - | Group ID |
| `system` | boolean | `false` | System group |

#### Example

```yaml
- name: "Create application group"
  group:
    name: "appgroup"
    gid: 1001
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

Read the status of a path. Never changes anything.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | - | Path (required) |
| `get_checksum` | boolean | `true` | Compute a checksum of regular files |
| `checksum_algorithm` | string | `sha1` | `md5`, `sha1`, `sha224`, `sha256`, `sha384` or `sha512` |

A link is reported as a link, not followed.

#### Example

```yaml
- name: "Get file status"
  stat:
    path: "/etc/nginx/nginx.conf"
  register: nginx_config

- name: "Show owner and mode"
  debug:
    msg: "{{ nginx_config.stat.pw_name }} {{ nginx_config.stat.mode }}"
  when: nginx_config.stat.exists
```

#### Return Values

```json
{
  "stat": {
    "exists": true,
    "path": "/etc/nginx/nginx.conf",
    "isreg": true,
    "isdir": false,
    "islnk": false,
    "size": 1482,
    "mode": "0644",
    "uid": 0,
    "gid": 0,
    "pw_name": "root",
    "gr_name": "root",
    "mtime": 1696086600,
    "atime": 1696086600,
    "ctime": 1696086600,
    "inode": 131090,
    "nlink": 1,
    "readable": true,
    "writable": true,
    "executable": false,
    "checksum": "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"
  }
}
```

Links also carry `lnk_source` (resolved path) and `lnk_target` (link text). For a missing path only `exists: false` and `path` are returned. The same fields are also available at the top level of the result.

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

Print a message or a variable.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `msg` | string | - | Message (templated) |
| `var` | string | - | Variable or expression to print: `result.stdout`, `result['stdout']`, `items \| length` |

`var` does not evaluate expressions or brackets (`hostvars[inventory_hostname]` prints "VARIABLE IS NOT DEFINED!"); use `msg: "{{ ... }}"` for those.

#### Example

```yaml
- name: "Debug variable"
  debug:
    var: ansible_facts

- name: "Debug message"
  debug:
    msg: "Current user is {{ ansible_user_id }}"
```

### set_fact

Set variables for the current host for the rest of the run. Every argument becomes a variable; types are kept.

#### Example

```yaml
- name: "Set deployment facts"
  set_fact:
    deployment_time: "{{ ansible_date_time.iso8601 }}"
    app_version: "v1.2.3"
    replicas: 3
```

### assert

Fail unless every expression holds.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `that` | string/list | - | Expression or list of expressions (required) |
| `fail_msg` | string | `Assertion failed` | Message on failure (`msg` is an alias) |
| `success_msg` | string | `All assertions passed` | Message on success |

#### Example

```yaml
- name: "Check prerequisites"
  assert:
    that:
      - ansible_os_family == "Debian"
      - ansible_processor_vcpus >= 2
    fail_msg: "Unsupported host"
```

### include_vars

Load variables from YAML files on the control machine into the host's variables.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `file` | string | - | File, relative to the playbook directory (free form in the short form) |
| `dir` | string | - | Load every `.yml`, `.yaml` and `.json` file of this directory, in name order |
| `name` | string | - | Put the variables under this one key |

#### Example

```yaml
- include_vars: vars/{{ ansible_os_family }}.yml

- name: "Load all settings under one key"
  include_vars:
    dir: settings
    name: settings
```

### include_role / import_role

Run a role's tasks at this point. Both are resolved when the playbook is loaded.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | string | - | Role name (required) |
| `tasks_from` | string | `main` | Task file of the role to run |

Tags of the task are added to the role's tasks.

#### Example

```yaml
- name: "Configure nginx"
  include_role:
    name: nginx
    tasks_from: install
```

### meta

Engine actions.

| Action | Effect |
|--------|--------|
| `flush_handlers` | Run notified handlers now |
| `end_host` | Stop the play for this host |
| `end_play` | Stop the play for all hosts |
| `noop`, `clear_host_errors`, `refresh_inventory`, `reset_connection` | Accepted, no effect |

```yaml
- meta: flush_handlers
```

### wait_for

Wait for conditions to be met.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `port` | integer | - | Port number to wait for (check if listening) |
| `host` | string | `127.0.0.1` | Hostname/IP address to check |
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

### slurp

Read a file from the host; `content` is base64 (`{{ r.content | b64decode }}`).
Parameters: `src` (required).

### hostname

Set the host name (`hostnamectl`, else /etc/hostname). Parameters: `name` (required).

### ini_file

One option of an INI file. Also `community.general.ini_file`.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `path` | - | File (created unless `create: false`) |
| `section` | - | Section; none: before the first section |
| `option`, `value` | - | Option and value; without option `present` makes sure the section exists, `absent` removes it |
| `state` | `present` | `present` or `absent` |
| `no_extra_spaces` | `false` | `key=value` instead of `key = value` |
| `backup`, `mode` | - | Backup copy; file mode |

Other lines of the option in the section are removed (Ansible's `exclusive`).

### pip

Python packages. Parameters: `name` (list; `pkg==1.2` pins), `version`, `state`
(`present`, `absent`, `latest`, `forcereinstall`), `requirements`, `virtualenv` (created with
`virtualenv_command`, default `python3 -m venv`), `executable`, `extra_args`.

### ufw

The Uncomplicated Firewall (`community.general.ufw`): `state` (`enabled`, `disabled`, `reloaded`,
`reset`), `policy` with `direction`, `logging`, and rules: `rule` (`allow`, `deny`, `limit`,
`reject`) with `port`, `proto`, `src`/`from_ip`, `dest`/`to_ip`, `from_port`, `interface`,
`direction`, `route`, `delete`, `insert`, `comment`. A rule that exists is not added again.

### docker_host_info

Docker host information (`community.docker.docker_host_info`); with `containers: true` the
containers as the Docker API lists them (`Id`, `Names` with the leading `/`, `Image`, `State`),
filtered by `containers_filters` (`name: [a, b]`), `containers_all` for stopped ones too.

`community.docker.docker_compose_v2` runs `docker_compose`: `project_src`, `files`, and `build`/
`pull` policies (`always`, `missing`, `policy`, `never`) are understood.

## 📚 Complete Module List

All 56 modules:

**Execution**: command, shell, script
**Files on the host**: slurp, ini_file
**Connectivity and Utilities**: ping, debug, set_fact, assert, fail, wait_for, pause
**Playbook Control**: include_vars, include_role, import_role, meta
**File Management**: file, copy, fetch, find, template, lineinfile, blockinfile, replace, stat, archive
**Package Management**: package, apt, yum, pip
**Service Management**: service, systemd, cron, reboot
**System Control**: sysctl, mount, hostname, timezone
**Security & Firewall**: firewall, ufw, authorized_key
**Version Control**: git
**Configuration**: config
**Containers**: docker_container, docker_image, docker_compose, docker_host_info, podman
**Databases**: mysql_db, mysql_user, postgresql_db, postgresql_user, mongodb
**Network**: get_url, uri
**User Management**: user, group

