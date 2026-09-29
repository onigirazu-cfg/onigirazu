# Systemd, Cron, and Firewall Modules

Reference for the **systemd**, **cron** and **firewall** modules. `systemd` and `cron` support check mode; `firewall` is skipped in check mode.

## Table of Contents

- [Systemd Module](#systemd-module)
- [Cron Module](#cron-module)
- [Firewall Module](#firewall-module)

---

## Systemd Module

The **systemd** module manages systemd services, unit files, and timers. `operation` defaults to `service`.

`daemon_reload: true` (as in Ansible) reloads systemd before the operation; with `operation: service` and no `name` the reload is all the task does.

### Operations

#### 1. Service Management (`operation: service`)

Manage systemd services (start, stop, restart, enable, disable, mask).

**Parameters:**

- `name` (required): Service name
- `state`: Service state (`started`, `stopped`, `restarted`, `reloaded`); `restarted` and `reloaded` always report `changed`
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
- `state`: `present` (default) or `absent`
- `content`: Unit file content
- `path`: Unit file path (default `/etc/systemd/system/<name>`)

With `state: present`, `content` or `path` is required; with `path` and no `content`, the file must already exist.

When the file changes, the module runs `daemon-reload` itself. `state: absent` stops and disables the unit, removes the file and reloads.

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
- `state`: Timer state (`started`, `stopped`; default `started`)
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

Reload the systemd configuration, e.g. after unit files were changed by other modules (`copy`, `template`). Always reports `changed`.

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

The **cron** module manages cron jobs, crontab files, and system cron directories. `operation` defaults to `job`.

### Operations

#### 1. Job Management (`operation: job`)

Manage named jobs in a user's crontab, as Ansible does: each job is a `#Ansible: <name>` comment followed by the job line (the `# Onigirazu: <name>` comments older versions wrote are read too). Only the job's own two lines change; every other line of the crontab stays as it is.

**Parameters:**

- `name` (required): Job identifier (used as comment in crontab)
- `job` (required when state is present): Command to execute
- `minute`: Minute (0-59, default: `*`)
- `hour`: Hour (0-23, default: `*`)
- `day`: Day of month (1-31, default: `*`)
- `month`: Month (1-12, default: `*`)
- `weekday`: Day of week (0-7, default: `*`)
- `special_time`: Special time string (`reboot`, `yearly`, `annually`, `monthly`, `weekly`, `daily`, `hourly`); written as `@<value>` instead of the time fields
- `user`: User whose crontab to modify (default: `root`)
- `state`: `present` (default) or `absent`

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
- `backup`: Save the current crontab to `/root/crontab.<user>.<timestamp>.backup` before replacing it (default: `true`)
- `state`: `present` (default) or `absent` (runs `crontab -r`)

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

- `name` (required): File name, without `/`
- `cron_type`: Cron directory: `d` (default, `/etc/cron.d`, mode `0644`), `daily`, `hourly`, `weekly`, `monthly` (`/etc/cron.<type>`, mode `0755`)
- `content` (required when state is present): File content; a final newline is added if missing
- `state`: `present` (default) or `absent`

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

List the named jobs of a user's crontab. The result has `jobs` (name to job line), `jobs_count` and `raw_crontab`.

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

The **firewall** module manages UFW, firewalld or iptables, whichever the host has. `operation` defaults to `rule`.

The backend is detected on the host in this order: `ufw`, `firewall-cmd`, `iptables`. Rule, service and source operations report `changed` only when the rule set differs afterwards. The module is skipped in check mode.

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

Disable and stop the firewall. With iptables this flushes all rules (`iptables -F`).

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
- `state`: `present` (default) or `absent`

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

## Requirements

- `systemd`: a host with systemd (`systemctl`)
- `cron`: `crontab` on the host; `operation: job`, `file` and `list` run `crontab -u <user>`, which usually needs root (`become: true`)
- `firewall`: `ufw`, `firewalld` or `iptables` on the host; `operation: service` does not work with iptables (use ports)

## Notes

- Allow SSH before `operation: enable`, or the connection may be cut.
- Unit files written with `copy` or `template` need `operation: daemon-reload` (or `daemon_reload: true`); `operation: unit` reloads by itself.
- Use absolute paths in cron jobs.

---

## See Also

- [Examples Directory](../examples/)
  - `10-systemd-management.yml`
  - `11-cron-management.yml`
  - `12-firewall-management.yml`
- [Module Development Guide](MODULE_DEVELOPMENT_GUIDE.md)
- [Playbook Syntax](README.md)
