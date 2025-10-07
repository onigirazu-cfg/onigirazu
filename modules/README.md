# Core Modules Documentation

This document provides comprehensive documentation for all built-in modules in Onigirazu.

## 📋 Table of Contents

- [Module Overview](#module-overview)
- [System Modules](#system-modules)
- [Configuration Modules](#configuration-modules)
- [Service Modules](#service-modules)
- [Package Modules](#package-modules)
- [Network Modules](#network-modules)
- [Security Modules](#security-modules)
- [Utility Modules](#utility-modules)

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
    filter: "ansible_*"
```

#### Return Values

```json
{
  "ansible_facts": {
    "ansible_hostname": "webserver01",
    "ansible_os_family": "RedHat",
    "ansible_distribution": "CentOS",
    "ansible_distribution_version": "8.4",
    "ansible_architecture": "x86_64",
    "ansible_processor_count": 4,
    "ansible_memtotal_mb": 8192,
    "ansible_interfaces": ["eth0", "lo"],
    "ansible_default_ipv4": {
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
    cmd: "mkdir -p /backup/{{ ansible_date_time.date }}"
    creates: "/backup/{{ ansible_date_time.date }}"
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
| `name` | string/list | - | Package name(s) |
| `state` | string | `present` | Package state |
| `update_cache` | boolean | `false` | Update apt cache |
| `cache_valid_time` | integer | `0` | Cache validity |
| `upgrade` | string | `no` | Upgrade packages |
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
| `name` | string/list | - | Package name(s) |
| `state` | string | `present` | Package state |
| `enablerepo` | string | - | Enable repository |
| `disablerepo` | string | - | Disable repository |
| `exclude` | string | - | Exclude packages |
| `update_cache` | boolean | `false` | Update cache |

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
| `method` | string | `GET` | HTTP method |
| `body` | string | - | Request body |
| `body_format` | string | `raw` | Body format |
| `headers` | dict | - | HTTP headers |
| `timeout` | integer | `30` | Request timeout |
| `validate_certs` | boolean | `true` | Validate SSL certificates |
| `status_code` | list | `[200]` | Expected status codes |

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
    var: "ansible_facts"

- name: "Debug message"
  debug:
    msg: "Current user is {{ ansible_user_id }}"
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
    deployment_time: "{{ ansible_date_time.iso8601 }}"
    app_version: "v1.2.3"
    environment: "production"
    cacheable: true
```

### wait_for

Wait for conditions to be met.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `port` | integer | - | Port to wait for |
| `host` | string | `127.0.0.1` | Host to check |
| `path` | string | - | File path to wait for |
| `search_regex` | string | - | Regex pattern in file |
| `state` | string | `started` | Condition state |
| `timeout` | integer | `300` | Wait timeout |
| `delay` | integer | `0` | Initial delay |
| `sleep` | integer | `1` | Check interval |

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
| `seconds` | integer | - | Pause duration |
| `minutes` | integer | - | Pause duration in minutes |
| `prompt` | string | - | User prompt message |
| `echo` | boolean | `true` | Echo user input |

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

This comprehensive module documentation provides detailed information about all built-in modules, their parameters, usage examples, and return values. Each module is designed to be idempotent and provide consistent behavior across different platforms.
