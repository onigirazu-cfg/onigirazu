# Systemd, Cron, and Firewall Modules Documentation

This document provides comprehensive documentation for the three new system management modules: **systemd**, **cron**, and **firewall**.

## Table of Contents

- [Systemd Module](#systemd-module)
- [Cron Module](#cron-module)
- [Firewall Module](#firewall-module)

---

## Systemd Module

The **systemd** module provides comprehensive management of systemd services, unit files, and timers.

### Operations

#### 1. Service Management (`operation: service`)

Manage systemd services (start, stop, restart, enable, disable, mask).

**Parameters:**

- `name` (required): Service name
- `state`: Service state (`started`, `stopped`, `restarted`, `reloaded`)
- `enabled`: Enable/disable service on boot (boolean)
- `masked`: Mask/unmask service (boolean)

**Examples:**

```yaml
# Start and enable a service
- name: Start nginx
  systemd:
    operation: service
    name: nginx
    state: started
    enabled: true

# Stop and disable a service
- name: Stop apache2
  systemd:
    operation: service
    name: apache2
    state: stopped
    enabled: false

# Restart a service
- name: Restart mysql
  systemd:
    operation: service
    name: mysql
    state: restarted

# Mask a service (prevent it from starting)
- name: Mask snapd
  systemd:
    operation: service
    name: snapd
    masked: true
```

#### 2. Unit File Management (`operation: unit`)

Create, modify, or remove systemd unit files.

**Parameters:**

- `name` (required): Unit file name (e.g., `myapp.service`)
- `state`: `present` or `absent`
- `content`: Unit file content (required when state is present)
- `path`: Custom path for unit file (optional, defaults to `/etc/systemd/system/<name>`)

**Examples:**

```yaml
# Create a custom service unit
- name: Create myapp service
  systemd:
    operation: unit
    name: myapp.service
    state: present
    content: |
      [Unit]
      Description=My Application
      After=network.target

      [Service]
      Type=simple
      User=www-data
      WorkingDirectory=/opt/myapp
      ExecStart=/opt/myapp/start.sh
      Restart=always
      RestartSec=10

      [Install]
      WantedBy=multi-user.target

# Remove a unit file
- name: Remove old service
  systemd:
    operation: unit
    name: oldapp.service
    state: absent
```

#### 3. Timer Management (`operation: timer`)

Manage systemd timers (systemd's alternative to cron).

**Parameters:**

- `name` (required): Timer name (automatically adds `.timer` suffix if missing)
- `state`: Timer state (`started`, `stopped`)
- `enabled`: Enable/disable timer on boot (boolean)

**Examples:**

```yaml
# Create and enable a timer
- name: Create backup timer unit
  systemd:
    operation: unit
    name: backup.timer
    state: present
    content: |
      [Unit]
      Description=Daily Backup Timer

      [Timer]
      OnCalendar=daily
      Persistent=true

      [Install]
      WantedBy=timers.target

- name: Create backup service unit
  systemd:
    operation: unit
    name: backup.service
    state: present
    content: |
      [Unit]
      Description=Backup Service

      [Service]
      Type=oneshot
      ExecStart=/usr/local/bin/backup.sh

- name: Enable and start timer
  systemd:
    operation: timer
    name: backup
    state: started
    enabled: true
```

#### 4. Daemon Reload (`operation: daemon-reload`)

Reload systemd daemon configuration (required after modifying unit files).

**Examples:**

```yaml
- name: Reload systemd daemon
  systemd:
    operation: daemon-reload
```

#### 5. Status Check (`operation: status`)

Get detailed status information about a service.

**Parameters:**

- `name` (required): Service name

**Examples:**

```yaml
- name: Get nginx status
  systemd:
    operation: status
    name: nginx
  register: nginx_status

- name: Display status
  debug:
    var: nginx_status
```

---

## Cron Module

The **cron** module provides comprehensive management of cron jobs, crontab files, and system cron directories.

### Operations

#### 1. Job Management (`operation: job`)

Manage individual cron jobs in user crontab.

**Parameters:**

- `name` (required): Job identifier (used as comment in crontab)
- `job` (required when state is present): Command to execute
- `minute`: Minute (0-59, default: `*`)
- `hour`: Hour (0-23, default: `*`)
- `day`: Day of month (1-31, default: `*`)
- `month`: Month (1-12, default: `*`)
- `weekday`: Day of week (0-7, default: `*`)
- `special_time`: Special time string (`reboot`, `yearly`, `annually`, `monthly`, `weekly`, `daily`, `hourly`)
- `user`: User whose crontab to modify (default: `root`)
- `state`: `present` or `absent`

**Examples:**

```yaml
# Daily backup at 2 AM
- name: Add daily backup job
  cron:
    operation: job
    name: daily_backup
    job: /usr/local/bin/backup.sh
    minute: "0"
    hour: "2"
    user: root

# Hourly cleanup
- name: Add hourly cleanup
  cron:
    operation: job
    name: hourly_cleanup
    job: /usr/local/bin/cleanup.sh
    minute: "0"
    user: www-data

# Weekly report on Monday at 8 AM
- name: Add weekly report
  cron:
    operation: job
    name: weekly_report
    job: /usr/local/bin/report.sh
    minute: "0"
    hour: "8"
    weekday: "1"

# Run at reboot
- name: Add reboot task
  cron:
    operation: job
    name: reboot_task
    job: /usr/local/bin/startup.sh
    special_time: reboot

# Remove a job
- name: Remove old job
  cron:
    operation: job
    name: old_job
    state: absent
```

#### 2. Crontab File Management (`operation: file`)

Manage entire crontab files for users.

**Parameters:**

- `user`: User whose crontab to manage (default: `root`)
- `content` (required when state is present): Complete crontab content
- `backup`: Create backup before modifying (default: `true`)
- `state`: `present` or `absent`

**Examples:**

```yaml
# Set complete crontab
- name: Set user crontab
  cron:
    operation: file
    user: backup
    backup: true
    content: |
      # Managed by Onigirazu

      # Daily backup at 2 AM
      0 2 * * * /usr/local/bin/backup.sh

      # Weekly cleanup on Sunday
      0 3 * * 0 /usr/local/bin/cleanup.sh

# Remove user's crontab
- name: Remove crontab
  cron:
    operation: file
    user: olduser
    state: absent
```

#### 3. System Cron Management (`operation: system`)

Manage system cron files in `/etc/cron.d`, `/etc/cron.daily`, etc.

**Parameters:**

- `name` (required): File name
- `cron_type`: Type of cron directory (`d`, `daily`, `hourly`, `weekly`, `monthly`)
- `content` (required when state is present): File content
- `state`: `present` or `absent`

**Examples:**

```yaml
# Create cron.d file
- name: Create application cron
  cron:
    operation: system
    name: myapp
    cron_type: d
    content: |
      # MyApp scheduled tasks
      */5 * * * * www-data /opt/myapp/check.sh
      0 */6 * * * www-data /opt/myapp/sync.sh

# Create daily script
- name: Create daily maintenance
  cron:
    operation: system
    name: daily-maintenance
    cron_type: daily
    content: |
      #!/bin/bash
      # Daily maintenance script

      find /tmp -type f -mtime +7 -delete
      /usr/sbin/logrotate /etc/logrotate.conf

# Create hourly script
- name: Create hourly monitoring
  cron:
    operation: system
    name: monitor
    cron_type: hourly
    content: |
      #!/bin/bash
      /usr/local/bin/check-services.sh

# Remove system cron file
- name: Remove old cron
  cron:
    operation: system
    name: old-task
    cron_type: d
    state: absent
```

#### 4. List Jobs (`operation: list`)

List all cron jobs for a user.

**Parameters:**

- `user`: User whose crontab to list (default: `root`)

**Examples:**

```yaml
- name: List root cron jobs
  cron:
    operation: list
    user: root
  register: root_crons

- name: Display cron jobs
  debug:
    var: root_crons
```

---

## Firewall Module

The **firewall** module provides unified management of different firewall systems with automatic detection. Supports **UFW** (Ubuntu/Debian), **firewalld** (RHEL/CentOS/Fedora), and **iptables**.

### Automatic Detection

The module automatically detects which firewall system is available on the target host and uses the appropriate backend.

### Operations

#### 1. Enable Firewall (`operation: enable`)

Enable and start the firewall.

**Examples:**

```yaml
- name: Enable firewall
  firewall:
    operation: enable
```

#### 2. Disable Firewall (`operation: disable`)

Disable and stop the firewall.

**Examples:**

```yaml
- name: Disable firewall
  firewall:
    operation: disable
```

#### 3. Port Rules (`operation: rule`)

Manage firewall rules for specific ports.

**Parameters:**

- `port` (required): Port number
- `protocol`: Protocol (`tcp` or `udp`, default: `tcp`)
- `action`: Action to take (`allow` or `deny`, default: `allow`)
- `state`: `present` or `absent`

**Examples:**

```yaml
# Allow SSH
- name: Allow SSH port
  firewall:
    operation: rule
    port: "22"
    protocol: tcp
    action: allow

# Allow HTTP and HTTPS
- name: Allow HTTP
  firewall:
    operation: rule
    port: "80"
    protocol: tcp
    action: allow

- name: Allow HTTPS
  firewall:
    operation: rule
    port: "443"
    protocol: tcp
    action: allow

# Allow custom port
- name: Allow application port
  firewall:
    operation: rule
    port: "8080"
    protocol: tcp
    action: allow

# Deny specific port
- name: Deny telnet
  firewall:
    operation: rule
    port: "23"
    protocol: tcp
    action: deny

# Remove rule
- name: Remove port rule
  firewall:
    operation: rule
    port: "8080"
    protocol: tcp
    state: absent
```

#### 4. Service Rules (`operation: service`)

Manage firewall rules for named services (UFW and firewalld only).

**Parameters:**

- `service` (required): Service name (e.g., `ssh`, `http`, `https`, `mysql`)
- `action`: Action to take (`allow` or `deny`, default: `allow`)
- `state`: `present` or `absent`

**Examples:**

```yaml
# Allow services
- name: Allow SSH service
  firewall:
    operation: service
    service: ssh
    action: allow

- name: Allow HTTP service
  firewall:
    operation: service
    service: http
    action: allow

# Deny service
- name: Deny FTP
  firewall:
    operation: service
    service: ftp
    action: deny

# Remove service rule
- name: Remove MySQL rule
  firewall:
    operation: service
    service: mysql
    state: absent
```

#### 5. Source-based Rules (`operation: source`)

Manage firewall rules based on source IP addresses or subnets.

**Parameters:**

- `source` (required): IP address or subnet (CIDR notation)
- `action`: Action to take (`allow` or `deny`, default: `allow`)
- `state`: `present` or `absent`

**Examples:**

```yaml
# Allow from specific IP
- name: Allow from admin IP
  firewall:
    operation: source
    source: "192.168.1.100"
    action: allow

# Allow from subnet
- name: Allow from office network
  firewall:
    operation: source
    source: "10.0.0.0/8"
    action: allow

# Deny from IP
- name: Deny from suspicious IP
  firewall:
    operation: source
    source: "203.0.113.0"
    action: deny

# Remove source rule
- name: Remove source rule
  firewall:
    operation: source
    source: "192.168.1.100"
    state: absent
```

#### 6. List Rules (`operation: list`)

List all firewall rules.

**Examples:**

```yaml
- name: List firewall rules
  firewall:
    operation: list
  register: firewall_rules

- name: Display rules
  debug:
    var: firewall_rules
```

#### 7. Reload Firewall (`operation: reload`)

Reload firewall configuration.

**Examples:**

```yaml
- name: Reload firewall
  firewall:
    operation: reload
```

### Complete Examples

#### Web Server Setup

```yaml
- name: Configure firewall for web server
  block:
    - name: Enable firewall
      firewall:
        operation: enable

    - name: Allow SSH
      firewall:
        operation: rule
        port: "22"
        protocol: tcp
        action: allow

    - name: Allow HTTP
      firewall:
        operation: rule
        port: "80"
        protocol: tcp
        action: allow

    - name: Allow HTTPS
      firewall:
        operation: rule
        port: "443"
        protocol: tcp
        action: allow

    - name: Reload firewall
      firewall:
        operation: reload
```

#### Database Server Setup

```yaml
- name: Configure firewall for database server
  block:
    - name: Enable firewall
      firewall:
        operation: enable

    - name: Allow SSH
      firewall:
        operation: rule
        port: "22"
        protocol: tcp
        action: allow

    - name: Allow PostgreSQL from app servers
      firewall:
        operation: source
        source: "10.0.1.0/24"
        action: allow

    - name: Reload firewall
      firewall:
        operation: reload
```

---

## Supported Systems

### Systemd Module

- **Linux**: All distributions with systemd (Ubuntu 16.04+, Debian 8+, CentOS 7+, Fedora, Arch, etc.)

### Cron Module

- **Linux**: All distributions with cron/crontab
- **macOS**: Full support
- **BSD**: Full support

### Firewall Module

- **UFW**: Ubuntu, Debian, Linux Mint
- **firewalld**: RHEL, CentOS, Fedora, Rocky Linux, AlmaLinux
- **iptables**: All Linux distributions (fallback)

---

## Best Practices

### Systemd

1. Always run `daemon-reload` after creating or modifying unit files
2. Use `enabled: true` to ensure services start on boot
3. Use masking for services you never want to start
4. Prefer systemd timers over cron for new projects

### Cron

1. Use descriptive job names for easy identification
2. Enable backup when modifying crontab files
3. Use system cron directories for application-specific tasks
4. Test cron jobs manually before scheduling

### Firewall

1. Always allow SSH before enabling firewall to avoid lockout
2. Test rules on non-production systems first
3. Use service names when available (more readable)
4. Document complex firewall configurations
5. Reload firewall after making changes

---

## Troubleshooting

### Systemd

- **Service fails to start**: Check logs with `journalctl -u <service>`
- **Unit file not found**: Run `daemon-reload` operation
- **Permission denied**: Ensure proper user permissions in unit file

### Cron

- **Job not running**: Check cron logs (`/var/log/cron` or `/var/log/syslog`)
- **Permission denied**: Verify user has permission to run the command
- **Path issues**: Use absolute paths in cron jobs

### Firewall

- **Locked out**: Ensure SSH is allowed before enabling firewall
- **Rules not applying**: Run reload operation
- **Service not found**: Use port numbers instead of service names for iptables

---

## See Also

- [Examples Directory](../examples/)
  - `10-systemd-management.yml`
  - `11-cron-management.yml`
  - `12-firewall-management.yml`
- [Module Development Guide](CONTRIBUTING.md)
- [Playbook Syntax](README.md)
