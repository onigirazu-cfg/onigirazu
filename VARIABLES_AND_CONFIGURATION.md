# Onigirazu Variables and Configuration Reference

This document provides a comprehensive reference for all configuration parameters and variables available in Onigirazu.

## Table of Contents

- [Configuration Parameters](#configuration-parameters)
- [System Variables (Facts)](#system-variables-facts)
- [Inventory Variables](#inventory-variables)
- [Date and Time Variables](#date-and-time-variables)
- [Environment Variables](#environment-variables)
- [Network Variables](#network-variables)

---

## Configuration Parameters

Onigirazu can be configured using a YAML configuration file. Below are all available configuration parameters:

### Execution Settings

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `max_concurrency` | int | 10 | Maximum number of concurrent tasks |
| `default_timeout` | duration | 30m | Default timeout for task execution |
| `retry_attempts` | int | 3 | Number of retry attempts for failed tasks |
| `retry_delay` | duration | 5s | Delay between retry attempts |

### File Paths

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `state_file` | string | `.onigirazu-state` | Path to state file for saving execution state |
| `config_file` | string | - | Path to configuration file |

### Logging

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `log_level` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `log_format` | string | `text` | Log format: `text`, `json` |

### Security

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `allow_shell_commands` | bool | true | Allow execution of shell commands |
| `blocked_commands` | []string | [] | List of blocked commands |

### Performance

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enable_caching` | bool | true | Enable template and facts caching |
| `cache_ttl` | duration | 10m | Cache time-to-live |
| `enable_checksum` | bool | true | Enable file checksum verification |
| `enable_parallel` | bool | true | Enable parallel task execution |
| `parallel_strategy` | string | `linear` | Parallel strategy: `linear`, `free` |

### Execution Modes

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `dry_run` | bool | false | Dry run mode (no changes) |
| `check_mode` | bool | false | Check mode (validate without executing) |
| `verbose` | bool | false | Verbose output |
| `show_diff` | bool | false | Show differences when changing files |

### UI/UX

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `color_output` | bool | true | Enable colored output |
| `progress_bar` | bool | true | Show progress bar |
| `interactive_mode` | bool | false | Interactive mode |
| `output_format` | string | `text` | Output format: `text`, `json`, `yaml` |

### Monitoring

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enable_metrics` | bool | false | Enable Prometheus metrics |
| `metrics_port` | int | 9090 | Metrics server port |
| `metrics_path` | string | `/metrics` | Metrics endpoint path |
| `enable_profiling` | bool | false | Enable Go profiling |

### SSH/Connection

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `ssh_timeout` | duration | 30s | SSH connection timeout |
| `ssh_keepalive` | duration | 60s | SSH keepalive interval |
| `ssh_max_sessions` | int | 10 | Maximum SSH sessions per host |
| `connection_reuse` | bool | true | Reuse SSH connections |

### Vault Integration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `vault_enabled` | bool | false | Enable HashiCorp Vault integration |
| `vault_address` | string | - | Vault server address |
| `vault_token` | string | - | Vault authentication token |

### Example Configuration File

```yaml
# onigirazu.yml
max_concurrency: 20
default_timeout: 1h
retry_attempts: 3
retry_delay: 10s

log_level: debug
log_format: json

enable_caching: true
cache_ttl: 15m

ssh_timeout: 60s
ssh_keepalive: 30s
connection_reuse: true

color_output: true
progress_bar: true
verbose: true
```

---

## System Variables (Facts)

System variables (facts) are automatically gathered when `gather_facts: true` is set in a play. These variables provide information about the target host.

### Basic Host Information

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `onigirazu_hostname` | string | Hostname from inventory | `webserver01` |
| `onigirazu_host` | string | IP address or hostname | `192.168.1.10` |
| `onigirazu_port` | int | SSH port | `22` |
| `onigirazu_user` | string | SSH username | `admin` |
| `onigirazu_fqdn` | string | Fully qualified domain name | `webserver01.example.com` |

### Operating System

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `onigirazu_os_family` | string | OS family | `Debian`, `RedHat`, `Darwin` |
| `onigirazu_distribution` | string | Distribution name | `ubuntu`, `centos`, `fedora` |
| `onigirazu_distribution_version` | string | Distribution version | `24.04`, `8.5` |
| `onigirazu_architecture` | string | System architecture | `x86_64`, `aarch64`, `arm64` |
| `onigirazu_kernel` | string | Kernel name | `Linux`, `Darwin` |
| `onigirazu_kernel_version` | string | Kernel version | `6.8.0-85-generic` |

### Hardware

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `onigirazu_processor_cores` | int | Number of CPU cores | `16` |
| `onigirazu_memtotal_mb` | string | Total memory | `15Gi`, `8192Mi` |

### User and Environment

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `onigirazu_user_id` | string | Current username on target | `usx` |
| `onigirazu_env.HOME` | string | Home directory | `/home/usx` |
| `onigirazu_env.PATH` | string | PATH environment variable | `/usr/local/bin:/usr/bin` |

---

## Inventory Variables

Inventory variables are defined in the inventory file and can be used in playbooks and templates.

### Connection Parameters

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `onigirazu_host` | string | Target host address | `192.168.1.10` |
| `onigirazu_port` | int | SSH port | `22` |
| `onigirazu_user` | string | SSH username | `deploy` |
| `onigirazu_ssh_private_key_file` | string | Path to SSH private key | `~/.ssh/id_rsa` |
| `onigirazu_ssh_pass` | string | SSH password (not recommended) | `password123` |
| `onigirazu_ssh_common_args` | string | Additional SSH arguments | `-o StrictHostKeyChecking=no` |

### Privilege Escalation

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `onigirazu_become` | bool | Enable privilege escalation | `true` |
| `onigirazu_become_user` | string | User to become | `root` |
| `onigirazu_become_method` | string | Escalation method | `sudo`, `su` |
| `onigirazu_become_pass` | string | Sudo password | `sudopass` |

### Example Inventory

```yaml
# inventory.yml
groups:
  webservers:
    hosts:
      web01:
        onigirazu_host: 192.168.1.10
        onigirazu_user: deploy
        onigirazu_ssh_private_key_file: ~/.ssh/id_rsa
        onigirazu_port: 22
        onigirazu_become: true
        onigirazu_become_user: root
    vars:
      http_port: 80
      domain: example.com
```

---

## Date and Time Variables

Date and time variables are automatically populated when facts are gathered.

### Available in `onigirazu_date_time` map

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `onigirazu_date_time.iso8601` | string | ISO 8601 timestamp | `2025-10-08T18:26:30+02:00` |
| `onigirazu_date_time.date` | string | Date in YYYY-MM-DD format | `2025-10-08` |
| `onigirazu_date_time.time` | string | Time in HH:MM:SS format | `18:26:30` |
| `onigirazu_date_time.year` | int | Year | `2025` |
| `onigirazu_date_time.month` | int | Month (1-12) | `10` |
| `onigirazu_date_time.day` | int | Day of month | `8` |
| `onigirazu_date_time.hour` | int | Hour (0-23) | `18` |
| `onigirazu_date_time.minute` | int | Minute (0-59) | `26` |
| `onigirazu_date_time.second` | int | Second (0-59) | `30` |
| `onigirazu_date_time.epoch` | int | Unix timestamp | `1759940790` |
| `onigirazu_date_time.weekday` | string | Day of week | `Wednesday` |
| `onigirazu_date_time.weekday_number` | int | Day of week (0-6) | `3` |

### Usage Examples

```yaml
tasks:
  - name: Create timestamped backup
    command:
      cmd: "cp /etc/config /backup/config-{{ onigirazu_date_time.date }}.bak"

  - name: Display current time
    debug:
      msg: "Current time: {{ onigirazu_date_time.iso8601 }}"

  - name: Create log file with timestamp
    file:
      path: "/var/log/deploy-{{ onigirazu_date_time.epoch }}.log"
      state: touch
```

---

## Environment Variables

Environment variables from the target host are available in the `onigirazu_env` map.

### Common Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `onigirazu_env.HOME` | User's home directory | `/home/deploy` |
| `onigirazu_env.PATH` | Executable search path | `/usr/local/bin:/usr/bin` |
| `onigirazu_env.USER` | Current username | `deploy` |
| `onigirazu_env.SHELL` | User's shell | `/bin/bash` |
| `onigirazu_env.LANG` | System language | `en_US.UTF-8` |

### Usage Examples

```yaml
tasks:
  - name: Create directory in home
    file:
      path: "{{ onigirazu_env.HOME }}/myapp"
      state: directory

  - name: Display PATH
    debug:
      msg: "PATH: {{ onigirazu_env.PATH }}"
```

---

## Network Variables

Network information is available in the `onigirazu_default_ipv4` map.

### Available Variables

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `onigirazu_default_ipv4.address` | string | Default IPv4 address | `192.168.1.10` |

### Usage Examples

```yaml
tasks:
  - name: Display IP address
    debug:
      msg: "Server IP: {{ onigirazu_default_ipv4.address }}"

  - name: Configure firewall
    command:
      cmd: "ufw allow from {{ onigirazu_default_ipv4.address }}"
```

---

## Using Variables in Playbooks

### Variable Precedence

Variables are resolved in the following order (highest to lowest priority):

1. Task variables (`vars` in task)
2. Play variables (`vars` in play)
3. Host variables (from inventory)
4. Group variables (from inventory)
5. System facts (gathered facts)
6. Default values

### Template Syntax

Variables can be used in templates using Jinja2-like syntax:

```yaml
tasks:
  - name: Display system info
    debug:
      msg: |
        Hostname: {{ onigirazu_hostname }}
        OS: {{ onigirazu_distribution }} {{ onigirazu_distribution_version }}
        Architecture: {{ onigirazu_architecture }}
        CPU Cores: {{ onigirazu_processor_cores }}
        Memory: {{ onigirazu_memtotal_mb }}
```

### Conditional Execution

Use variables in conditionals:

```yaml
tasks:
  - name: Install package (Debian)
    apt:
      name: nginx
      state: present
    when: onigirazu_os_family == "Debian"

  - name: Install package (RedHat)
    yum:
      name: nginx
      state: present
    when: onigirazu_os_family == "RedHat"
```

### Play-Level Variables with Templates

Play variables can reference facts:

```yaml
plays:
  - name: Deploy Application
    hosts: all
    gather_facts: true
    vars:
      app_dir: "{{ onigirazu_env.HOME }}/myapp"
      backup_dir: "/backup/{{ onigirazu_hostname }}"
      log_file: "/var/log/deploy-{{ onigirazu_date_time.epoch }}.log"
    tasks:
      - name: Create app directory
        file:
          path: "{{ app_dir }}"
          state: directory
```

---

## Best Practices

### 1. Always Gather Facts When Needed

```yaml
plays:
  - name: Configure Servers
    hosts: all
    gather_facts: true  # Enable fact gathering
    tasks:
      # Your tasks here
```

### 2. Use Descriptive Variable Names

```yaml
vars:
  nginx_config_dir: "/etc/nginx"
  app_version: "v1.2.3"
  deployment_timestamp: "{{ onigirazu_date_time.epoch }}"
```

### 3. Validate Variables

```yaml
tasks:
  - name: Ensure required variables are set
    assert:
      that:
        - onigirazu_os_family is defined
        - app_version is defined
      fail_msg: "Required variables are not set"
```

### 4. Use Default Values

```yaml
tasks:
  - name: Set default port
    set_fact:
      http_port: "{{ http_port | default(80) }}"
```

### 5. Document Custom Variables

```yaml
# vars/production.yml
# Application configuration for production environment
app_version: "v1.2.3"        # Application version to deploy
db_host: "prod-db.local"     # Database host
db_port: 5432                # Database port
max_connections: 100         # Maximum database connections
```

---

## Troubleshooting

### Variables Not Available

If variables are not available:

1. **Check if facts are gathered**: Ensure `gather_facts: true` is set in the play
2. **Check variable scope**: Variables defined in tasks are not available in other tasks
3. **Check spelling**: Variable names are case-sensitive
4. **Check inventory**: Ensure inventory variables are properly defined

### Template Rendering Errors

If template rendering fails:

1. **Check syntax**: Ensure proper `{{ }}` syntax
2. **Check variable existence**: Use `| default('value')` for optional variables
3. **Check data types**: Ensure variables are of expected type
4. **Enable debug logging**: Use `--log-level debug` to see detailed errors

### Example Debug Task

```yaml
tasks:
  - name: Debug all variables
    debug:
      var: hostvars[inventory_hostname]

  - name: Debug specific variable
    debug:
      msg: "OS Family: {{ onigirazu_os_family | default('not set') }}"
```

---

## See Also

- [Quick Start Guide](quick-start.md)
- [Inventory Formats](inventory-formats.md)
- [Modules Reference](modules/README.md)
- [API Documentation](api/README.md)

---

*Last updated: 2025-10-08*
