# Examples and Use Cases

This directory contains comprehensive examples demonstrating various Onigirazu features and real-world use cases.

## 📋 Table of Contents

- [Basic Examples](#basic-examples)
- [Advanced Workflows](#advanced-workflows)
- [Infrastructure Management](#infrastructure-management)
- [Application Deployment](#application-deployment)
- [Security and Compliance](#security-and-compliance)
- [Monitoring and Alerting](#monitoring-and-alerting)
- [Custom Modules](#custom-modules)
- [Integration Examples](#integration-examples)

## 🚀 Basic Examples

### Simple Task Execution

```yaml
# basic-task.yml
name: "Basic Task Example"
plays:
  - name: "System Information"
    hosts: "all"
    tasks:
      - name: "Gather system facts"
        facts:
        register: "system_info"

      - name: "Display hostname"
        debug:
          msg: "Hostname: {{ onigirazu_hostname }}"

      - name: "Check disk space"
        command:
          cmd: "df -h /"
        register: "disk_space"

      - name: "Display disk usage"
        debug:
          var: "disk_space.stdout"
```

### File Management

```yaml
# file-management.yml
name: "File Management Example"
plays:
  - name: "Manage Configuration Files"
    hosts: "webservers"
    vars:
      app_config_dir: "/etc/myapp"
      backup_dir: "/backup/configs"

    tasks:
      - name: "Create application directory"
        file:
          path: "{{ app_config_dir }}"
          state: "directory"
          mode: "0755"
          owner: "appuser"
          group: "appgroup"

      - name: "Create backup directory"
        file:
          path: "{{ backup_dir }}"
          state: "directory"
          mode: "0755"

      - name: "Deploy configuration file"
        template:
          src: "./templates/app.conf.j2"
          dest: "{{ app_config_dir }}/app.conf"
          backup: true
          mode: "0644"
          owner: "appuser"
          group: "appgroup"
        notify: "restart application"

      - name: "Set configuration values"
        config:
          path: "{{ app_config_dir }}/settings.yml"
          format: "yaml"
          action: "set"
          key: "{{ item.key }}"
          value: "{{ item.value }}"
          backup: true
        loop:
          - { key: "database.host", value: "{{ db_host }}" }
          - { key: "database.port", value: "{{ db_port }}" }
          - { key: "cache.enabled", value: true }
          - { key: "logging.level", value: "info" }

    handlers:
      - name: "restart application"
        service:
          name: "myapp"
          state: "restarted"
```

### Package Installation

```yaml
# package-installation.yml
name: "Package Installation Example"
plays:
  - name: "Install Software Packages"
    hosts: "all"
    become: true

    tasks:
      - name: "Update package cache (Debian/Ubuntu)"
        apt:
          update_cache: true
          cache_valid_time: 3600
        when: "onigirazu_os_family == 'Debian'"

      - name: "Install essential packages"
        package:
          name:
            - "curl"
            - "wget"
            - "git"
            - "vim"
            - "htop"
            - "unzip"
          state: "present"

      - name: "Install web server (Nginx)"
        package:
          name: "nginx"
          state: "present"
        notify: "start nginx"

      - name: "Install database client"
        package:
          name: "{{ db_client_package }}"
          state: "present"
        vars:
          db_client_package: >-
            {%- if onigirazu_os_family == 'Debian' -%}
            mysql-client
            {%- elif onigirazu_os_family == 'RedHat' -%}
            mysql
            {%- else -%}
            mysql-client
            {%- endif -%}

    handlers:
      - name: "start nginx"
        service:
          name: "nginx"
          state: "started"
          enabled: true
```

## 🔄 Advanced Workflows

### Multi-Stage Deployment

```yaml
# multi-stage-deployment.yml
name: "Multi-Stage Deployment Workflow"
plays:
  - name: "Pre-deployment Checks"
    hosts: "all"
    serial: 1

    tasks:
      - name: "Check system resources"
        command:
          cmd: "free -m | awk 'NR==2{printf \"%.2f%%\", $3*100/$2}'"
        register: "memory_usage"

      - name: "Fail if memory usage too high"
        fail:
          msg: "Memory usage too high: {{ memory_usage.stdout }}%"
        when: "memory_usage.stdout | float > 80"

      - name: "Check disk space"
        command:
          cmd: "df / | awk 'NR==2 {print $5}' | sed 's/%//'"
        register: "disk_usage"

      - name: "Fail if disk usage too high"
        fail:
          msg: "Disk usage too high: {{ disk_usage.stdout }}%"
        when: "disk_usage.stdout | int > 85"

  - name: "Database Backup"
    hosts: "database"
    serial: 1

    tasks:
      - name: "Create backup directory"
        file:
          path: "/backup/{{ onigirazu_date_time.date }}"
          state: "directory"
          mode: "0755"

      - name: "Backup database"
        command:
          cmd: >
            mysqldump -u {{ db_user }} -p{{ db_password }}
            --single-transaction --routines --triggers
            {{ db_name }} > /backup/{{ onigirazu_date_time.date }}/{{ db_name }}.sql
        no_log: true

      - name: "Compress backup"
        command:
          cmd: "gzip /backup/{{ onigirazu_date_time.date }}/{{ db_name }}.sql"

  - name: "Application Deployment"
    hosts: "webservers"
    serial: "30%"

    tasks:
      - name: "Stop application service"
        service:
          name: "myapp"
          state: "stopped"

      - name: "Create release directory"
        file:
          path: "/opt/myapp/releases/{{ app_version }}"
          state: "directory"
          owner: "appuser"
          group: "appgroup"

      - name: "Download application archive"
        get_url:
          url: "{{ app_download_url }}/{{ app_version }}/myapp-{{ app_version }}.tar.gz"
          dest: "/tmp/myapp-{{ app_version }}.tar.gz"
          mode: "0644"

      - name: "Extract application"
        command:
          cmd: "tar -xzf /tmp/myapp-{{ app_version }}.tar.gz -C /opt/myapp/releases/{{ app_version }} --strip-components=1"
          creates: "/opt/myapp/releases/{{ app_version }}/app.py"

      - name: "Update symlink"
        file:
          src: "/opt/myapp/releases/{{ app_version }}"
          dest: "/opt/myapp/current"
          state: "link"
          owner: "appuser"
          group: "appgroup"

      - name: "Install dependencies"
        command:
          cmd: "pip install -r requirements.txt"
          chdir: "/opt/myapp/current"
        become_user: "appuser"

      - name: "Run database migrations"
        command:
          cmd: "python manage.py migrate"
          chdir: "/opt/myapp/current"
        become_user: "appuser"
        run_once: true

      - name: "Start application service"
        service:
          name: "myapp"
          state: "started"

      - name: "Wait for application to start"
        wait_for:
          port: 8080
          host: "{{ inventory_hostname }}"
          timeout: 60

  - name: "Post-deployment Verification"
    hosts: "webservers"

    tasks:
      - name: "Health check"
        uri:
          url: "http://{{ inventory_hostname }}:8080/health"
          method: "GET"
          status_code: 200
          timeout: 10
        register: "health_check"
        retries: 5
        delay: 10

      - name: "Verify application version"
        uri:
          url: "http://{{ inventory_hostname }}:8080/version"
          method: "GET"
          return_content: true
        register: "version_check"

      - name: "Validate version"
        fail:
          msg: "Version mismatch: expected {{ app_version }}, got {{ version_check.json.version }}"
        when: "version_check.json.version != app_version"

      - name: "Send deployment notification"
        uri:
          url: "{{ slack_webhook_url }}"
          method: "POST"
          body_format: "json"
          body:
            text: "✅ Deployment successful: {{ app_version }} on {{ inventory_hostname }}"
        delegate_to: "localhost"
        run_once: true
```

### Rolling Update with Rollback

```yaml
# rolling-update.yml
name: "Rolling Update with Rollback"
plays:
  - name: "Rolling Update"
    hosts: "webservers"
    serial: 1
    max_fail_percentage: 0
    any_errors_fatal: true

    vars:
      health_check_retries: 5
      health_check_delay: 10
      rollback_on_failure: true

    tasks:
      - name: "Get current version"
        command:
          cmd: "readlink /opt/myapp/current"
        register: "current_version"
        changed_when: false

      - name: "Set rollback version"
        set_fact:
          rollback_version: "{{ current_version.stdout | basename }}"

      - name: "Remove from load balancer"
        uri:
          url: "{{ lb_api_url }}/servers/{{ inventory_hostname }}/disable"
          method: "POST"
          headers:
            Authorization: "Bearer {{ lb_api_token }}"
        delegate_to: "localhost"

      - name: "Wait for connections to drain"
        pause:
          seconds: 30

      - name: "Deploy new version"
        include_tasks: "deploy-tasks.yml"

      - name: "Health check"
        uri:
          url: "http://{{ inventory_hostname }}:8080/health"
          method: "GET"
          status_code: 200
        register: "health_check"
        retries: "{{ health_check_retries }}"
        delay: "{{ health_check_delay }}"
        failed_when: false

      - name: "Rollback on health check failure"
        block:
          - name: "Stop new version"
            service:
              name: "myapp"
              state: "stopped"

          - name: "Restore previous version"
            file:
              src: "/opt/myapp/releases/{{ rollback_version }}"
              dest: "/opt/myapp/current"
              state: "link"

          - name: "Start previous version"
            service:
              name: "myapp"
              state: "started"

          - name: "Fail deployment"
            fail:
              msg: "Health check failed, rolled back to {{ rollback_version }}"

        when: "health_check.status != 200 and rollback_on_failure"

      - name: "Add back to load balancer"
        uri:
          url: "{{ lb_api_url }}/servers/{{ inventory_hostname }}/enable"
          method: "POST"
          headers:
            Authorization: "Bearer {{ lb_api_token }}"
        delegate_to: "localhost"

      - name: "Verify load balancer status"
        uri:
          url: "{{ lb_api_url }}/servers/{{ inventory_hostname }}/status"
          method: "GET"
          headers:
            Authorization: "Bearer {{ lb_api_token }}"
        register: "lb_status"
        delegate_to: "localhost"
        retries: 3
        delay: 5

      - name: "Cleanup old releases"
        shell:
          cmd: |
            cd /opt/myapp/releases
            ls -t | tail -n +6 | xargs rm -rf
        when: "health_check.status == 200"
```

## 🏗️ Infrastructure Management

### Server Provisioning

```yaml
# server-provisioning.yml
name: "Server Provisioning"
plays:
  - name: "Initial Server Setup"
    hosts: "new_servers"
    become: true

    vars:
      admin_users:
        - { name: "admin1", key: "{{ lookup('file', 'keys/admin1.pub') }}" }
        - { name: "admin2", key: "{{ lookup('file', 'keys/admin2.pub') }}" }

      security_packages:
        - "fail2ban"
        - "ufw"
        - "unattended-upgrades"

    tasks:
      - name: "Update system packages"
        package:
          name: "*"
          state: "latest"
        when: "onigirazu_os_family == 'RedHat'"

      - name: "Update system packages (Debian)"
        apt:
          upgrade: "dist"
          update_cache: true
          autoremove: true
        when: "onigirazu_os_family == 'Debian'"

      - name: "Install security packages"
        package:
          name: "{{ security_packages }}"
          state: "present"

      - name: "Create admin users"
        user:
          name: "{{ item.name }}"
          groups: "sudo"
          shell: "/bin/bash"
          create_home: true
        loop: "{{ admin_users }}"

      - name: "Set up SSH keys for admin users"
        authorized_key:
          user: "{{ item.name }}"
          key: "{{ item.key }}"
          state: "present"
        loop: "{{ admin_users }}"

      - name: "Configure SSH security"
        lineinfile:
          path: "/etc/ssh/sshd_config"
          regexp: "{{ item.regexp }}"
          line: "{{ item.line }}"
          backup: true
        loop:
          - { regexp: "^#?PasswordAuthentication", line: "PasswordAuthentication no" }
          - { regexp: "^#?PermitRootLogin", line: "PermitRootLogin no" }
          - { regexp: "^#?Port", line: "Port {{ ssh_port | default(22) }}" }
          - { regexp: "^#?MaxAuthTries", line: "MaxAuthTries 3" }
        notify: "restart ssh"

      - name: "Configure firewall"
        ufw:
          rule: "{{ item.rule }}"
          port: "{{ item.port }}"
          proto: "{{ item.proto | default('tcp') }}"
        loop:
          - { rule: "allow", port: "{{ ssh_port | default(22) }}" }
          - { rule: "allow", port: "80" }
          - { rule: "allow", port: "443" }
        notify: "enable firewall"

      - name: "Configure automatic updates"
        template:
          src: "./templates/50unattended-upgrades.j2"
          dest: "/etc/apt/apt.conf.d/50unattended-upgrades"
          mode: "0644"
        when: "onigirazu_os_family == 'Debian'"

      - name: "Configure fail2ban"
        template:
          src: "./templates/jail.local.j2"
          dest: "/etc/fail2ban/jail.local"
          mode: "0644"
        notify: "restart fail2ban"

      - name: "Set timezone"
        timezone:
          name: "{{ server_timezone | default('UTC') }}"

      - name: "Configure NTP"
        package:
          name: "chrony"
          state: "present"
        notify: "start chrony"

    handlers:
      - name: "restart ssh"
        service:
          name: "sshd"
          state: "restarted"

      - name: "enable firewall"
        ufw:
          state: "enabled"
          policy: "deny"

      - name: "restart fail2ban"
        service:
          name: "fail2ban"
          state: "restarted"
          enabled: true

      - name: "start chrony"
        service:
          name: "chrony"
          state: "started"
          enabled: true
```

### Load Balancer Configuration

```yaml
# load-balancer.yml
name: "Load Balancer Configuration"
plays:
  - name: "Configure HAProxy Load Balancer"
    hosts: "load_balancers"
    become: true

    vars:
      haproxy_stats_user: "admin"
      haproxy_stats_password: "{{ vault_haproxy_stats_password }}"
      backend_servers:
        - { name: "web1", address: "192.168.1.10", port: 8080, check: "check" }
        - { name: "web2", address: "192.168.1.11", port: 8080, check: "check" }
        - { name: "web3", address: "192.168.1.12", port: 8080, check: "check" }

    tasks:
      - name: "Install HAProxy"
        package:
          name: "haproxy"
          state: "present"

      - name: "Configure HAProxy"
        template:
          src: "./templates/haproxy.cfg.j2"
          dest: "/etc/haproxy/haproxy.cfg"
          backup: true
          validate: "haproxy -f %s -c"
        notify: "restart haproxy"

      - name: "Enable HAProxy service"
        service:
          name: "haproxy"
          enabled: true
          state: "started"

      - name: "Configure rsyslog for HAProxy"
        lineinfile:
          path: "/etc/rsyslog.conf"
          line: "$UDPServerRun 514"
          regexp: "^#?\\$UDPServerRun"
        notify: "restart rsyslog"

      - name: "Create HAProxy log configuration"
        copy:
          content: |
            $UDPServerAddress 127.0.0.1
            local0.*    /var/log/haproxy.log
            & stop
          dest: "/etc/rsyslog.d/49-haproxy.conf"
        notify: "restart rsyslog"

      - name: "Configure log rotation"
        copy:
          content: |
            /var/log/haproxy.log {
                daily
                rotate 52
                missingok
                notifempty
                compress
                delaycompress
                postrotate
                    /bin/kill -HUP `cat /var/run/rsyslogd.pid 2> /dev/null` 2> /dev/null || true
                endscript
            }
          dest: "/etc/logrotate.d/haproxy"

    handlers:
      - name: "restart haproxy"
        service:
          name: "haproxy"
          state: "restarted"

      - name: "restart rsyslog"
        service:
          name: "rsyslog"
          state: "restarted"
```

## 🔒 Security and Compliance

### Security Hardening

```yaml
# security-hardening.yml
name: "Security Hardening"
plays:
  - name: "System Security Hardening"
    hosts: "all"
    become: true

    vars:
      allowed_users:
        - "root"
        - "admin"
        - "appuser"

      sysctl_settings:
        - { name: "net.ipv4.ip_forward", value: "0" }
        - { name: "net.ipv4.conf.all.send_redirects", value: "0" }
        - { name: "net.ipv4.conf.default.send_redirects", value: "0" }
        - { name: "net.ipv4.conf.all.accept_source_route", value: "0" }
        - { name: "net.ipv4.conf.default.accept_source_route", value: "0" }
        - { name: "net.ipv4.conf.all.accept_redirects", value: "0" }
        - { name: "net.ipv4.conf.default.accept_redirects", value: "0" }
        - { name: "net.ipv4.conf.all.secure_redirects", value: "0" }
        - { name: "net.ipv4.conf.default.secure_redirects", value: "0" }
        - { name: "net.ipv4.conf.all.log_martians", value: "1" }
        - { name: "net.ipv4.conf.default.log_martians", value: "1" }
        - { name: "net.ipv4.icmp_echo_ignore_broadcasts", value: "1" }
        - { name: "net.ipv4.icmp_ignore_bogus_error_responses", value: "1" }
        - { name: "net.ipv4.tcp_syncookies", value: "1" }
        - { name: "kernel.dmesg_restrict", value: "1" }
        - { name: "kernel.kptr_restrict", value: "2" }

    tasks:
      - name: "Remove unused packages"
        package:
          name:
            - "telnet"
            - "rsh-server"
            - "rsh"
            - "ypbind"
            - "ypserv"
            - "tftp"
            - "tftp-server"
            - "talk"
            - "talk-server"
          state: "absent"

      - name: "Set password policy"
        lineinfile:
          path: "/etc/login.defs"
          regexp: "{{ item.regexp }}"
          line: "{{ item.line }}"
          backup: true
        loop:
          - { regexp: "^PASS_MAX_DAYS", line: "PASS_MAX_DAYS 90" }
          - { regexp: "^PASS_MIN_DAYS", line: "PASS_MIN_DAYS 7" }
          - { regexp: "^PASS_WARN_AGE", line: "PASS_WARN_AGE 14" }
          - { regexp: "^PASS_MIN_LEN", line: "PASS_MIN_LEN 8" }

      - name: "Configure PAM password requirements"
        lineinfile:
          path: "/etc/pam.d/common-password"
          regexp: "pam_pwquality.so"
          line: "password requisite pam_pwquality.so retry=3 minlen=8 difok=3 ucredit=-1 lcredit=-1 dcredit=-1 ocredit=-1"
          backup: true
        when: "onigirazu_os_family == 'Debian'"

      - name: "Set kernel parameters"
        sysctl:
          name: "{{ item.name }}"
          value: "{{ item.value }}"
          state: "present"
          reload: true
        loop: "{{ sysctl_settings }}"

      - name: "Configure audit daemon"
        package:
          name: "auditd"
          state: "present"

      - name: "Configure audit rules"
        copy:
          content: |
            # Monitor authentication events
            -w /var/log/auth.log -p wa -k authentication
            -w /var/log/secure -p wa -k authentication

            # Monitor system configuration changes
            -w /etc/passwd -p wa -k identity
            -w /etc/group -p wa -k identity
            -w /etc/shadow -p wa -k identity
            -w /etc/sudoers -p wa -k identity

            # Monitor network configuration
            -w /etc/network/ -p wa -k network
            -w /etc/hosts -p wa -k network
            -w /etc/hostname -p wa -k network

            # Monitor critical files
            -w /etc/ssh/sshd_config -p wa -k sshd
            -w /etc/crontab -p wa -k cron
            -w /etc/cron.allow -p wa -k cron
            -w /etc/cron.deny -p wa -k cron

            # Monitor privilege escalation
            -a always,exit -F arch=b64 -S setuid -S setgid -S setreuid -S setregid -k privilege_esc
            -a always,exit -F arch=b32 -S setuid -S setgid -S setreuid -S setregid -k privilege_esc
          dest: "/etc/audit/rules.d/hardening.rules"
        notify: "restart auditd"

      - name: "Remove unauthorized users"
        user:
          name: "{{ item }}"
          state: "absent"
          remove: true
        loop: "{{ unauthorized_users | default([]) }}"
        when: "unauthorized_users is defined"

      - name: "Set file permissions on sensitive files"
        file:
          path: "{{ item.path }}"
          mode: "{{ item.mode }}"
          owner: "{{ item.owner | default('root') }}"
          group: "{{ item.group | default('root') }}"
        loop:
          - { path: "/etc/passwd", mode: "0644" }
          - { path: "/etc/shadow", mode: "0600" }
          - { path: "/etc/group", mode: "0644" }
          - { path: "/etc/gshadow", mode: "0600" }
          - { path: "/etc/ssh/sshd_config", mode: "0600" }
          - { path: "/etc/crontab", mode: "0600" }

      - name: "Configure automatic security updates"
        template:
          src: "./templates/auto-upgrades.j2"
          dest: "/etc/apt/apt.conf.d/20auto-upgrades"
          mode: "0644"
        when: "onigirazu_os_family == 'Debian'"

      - name: "Install and configure AIDE"
        block:
          - name: "Install AIDE"
            package:
              name: "aide"
              state: "present"

          - name: "Initialize AIDE database"
            command:
              cmd: "aideinit"
              creates: "/var/lib/aide/aide.db.new"

          - name: "Move AIDE database"
            command:
              cmd: "mv /var/lib/aide/aide.db.new /var/lib/aide/aide.db"
              creates: "/var/lib/aide/aide.db"

          - name: "Schedule AIDE checks"
            cron:
              name: "AIDE integrity check"
              minute: "0"
              hour: "2"
              job: "/usr/bin/aide --check"
              user: "root"

    handlers:
      - name: "restart auditd"
        service:
          name: "auditd"
          state: "restarted"
          enabled: true
```

### Compliance Scanning

```yaml
# compliance-scan.yml
name: "Security Compliance Scanning"
plays:
  - name: "CIS Benchmark Compliance Check"
    hosts: "all"
    become: true

    vars:
      compliance_report_dir: "/tmp/compliance"
      cis_benchmark_version: "1.0"

    tasks:
      - name: "Create compliance report directory"
        file:
          path: "{{ compliance_report_dir }}"
          state: "directory"
          mode: "0755"

      - name: "Check password policy compliance"
        shell:
          cmd: |
            grep "^PASS_MAX_DAYS" /etc/login.defs | awk '{print $2}'
        register: "pass_max_days"
        changed_when: false

      - name: "Check SSH configuration compliance"
        shell:
          cmd: |
            sshd -T | grep -E "(passwordauthentication|permitrootlogin|protocol)"
        register: "ssh_config"
        changed_when: false

      - name: "Check firewall status"
        command:
          cmd: "ufw status"
        register: "firewall_status"
        changed_when: false
        failed_when: false

      - name: "Check for unauthorized SUID files"
        shell:
          cmd: |
            find / -perm -4000 -type f 2>/dev/null | grep -v -E "(sudo|su|passwd|ping)"
        register: "suid_files"
        changed_when: false
        failed_when: false

      - name: "Check for world-writable files"
        shell:
          cmd: |
            find / -perm -002 -type f 2>/dev/null | head -20
        register: "world_writable"
        changed_when: false
        failed_when: false

      - name: "Generate compliance report"
        template:
          src: "./templates/compliance-report.j2"
          dest: "{{ compliance_report_dir }}/compliance-{{ inventory_hostname }}.html"
          mode: "0644"

      - name: "Fetch compliance report"
        fetch:
          src: "{{ compliance_report_dir }}/compliance-{{ inventory_hostname }}.html"
          dest: "./reports/"
          flat: true
```

## 📊 Monitoring and Alerting

### Monitoring Setup

```yaml
# monitoring-setup.yml
name: "Monitoring Infrastructure Setup"
plays:
  - name: "Install Prometheus"
    hosts: "monitoring"
    become: true

    vars:
      prometheus_version: "2.40.0"
      prometheus_user: "prometheus"
      prometheus_config_dir: "/etc/prometheus"
      prometheus_data_dir: "/var/lib/prometheus"

    tasks:
      - name: "Create prometheus user"
        user:
          name: "{{ prometheus_user }}"
          system: true
          shell: "/bin/false"
          home: "{{ prometheus_data_dir }}"
          create_home: false

      - name: "Create prometheus directories"
        file:
          path: "{{ item }}"
          state: "directory"
          owner: "{{ prometheus_user }}"
          group: "{{ prometheus_user }}"
          mode: "0755"
        loop:
          - "{{ prometheus_config_dir }}"
          - "{{ prometheus_data_dir }}"
          - "/var/log/prometheus"

      - name: "Download Prometheus"
        get_url:
          url: "https://github.com/prometheus/prometheus/releases/download/v{{ prometheus_version }}/prometheus-{{ prometheus_version }}.linux-amd64.tar.gz"
          dest: "/tmp/prometheus-{{ prometheus_version }}.tar.gz"
          mode: "0644"

      - name: "Extract Prometheus"
        unarchive:
          src: "/tmp/prometheus-{{ prometheus_version }}.tar.gz"
          dest: "/tmp"
          remote_src: true
          creates: "/tmp/prometheus-{{ prometheus_version }}.linux-amd64"

      - name: "Install Prometheus binaries"
        copy:
          src: "/tmp/prometheus-{{ prometheus_version }}.linux-amd64/{{ item }}"
          dest: "/usr/local/bin/{{ item }}"
          mode: "0755"
          owner: "root"
          group: "root"
          remote_src: true
        loop:
          - "prometheus"
          - "promtool"

      - name: "Configure Prometheus"
        template:
          src: "./templates/prometheus.yml.j2"
          dest: "{{ prometheus_config_dir }}/prometheus.yml"
          owner: "{{ prometheus_user }}"
          group: "{{ prometheus_user }}"
          mode: "0644"
        notify: "restart prometheus"

      - name: "Create Prometheus systemd service"
        template:
          src: "./templates/prometheus.service.j2"
          dest: "/etc/systemd/system/prometheus.service"
          mode: "0644"
        notify:
          - "reload systemd"
          - "restart prometheus"

      - name: "Start and enable Prometheus"
        service:
          name: "prometheus"
          state: "started"
          enabled: true

  - name: "Install Node Exporter"
    hosts: "all"
    become: true

    vars:
      node_exporter_version: "1.4.0"
      node_exporter_user: "node_exporter"

    tasks:
      - name: "Create node_exporter user"
        user:
          name: "{{ node_exporter_user }}"
          system: true
          shell: "/bin/false"
          create_home: false

      - name: "Download Node Exporter"
        get_url:
          url: "https://github.com/prometheus/node_exporter/releases/download/v{{ node_exporter_version }}/node_exporter-{{ node_exporter_version }}.linux-amd64.tar.gz"
          dest: "/tmp/node_exporter-{{ node_exporter_version }}.tar.gz"
          mode: "0644"

      - name: "Extract Node Exporter"
        unarchive:
          src: "/tmp/node_exporter-{{ node_exporter_version }}.tar.gz"
          dest: "/tmp"
          remote_src: true
          creates: "/tmp/node_exporter-{{ node_exporter_version }}.linux-amd64"

      - name: "Install Node Exporter binary"
        copy:
          src: "/tmp/node_exporter-{{ node_exporter_version }}.linux-amd64/node_exporter"
          dest: "/usr/local/bin/node_exporter"
          mode: "0755"
          owner: "root"
          group: "root"
          remote_src: true

      - name: "Create Node Exporter systemd service"
        copy:
          content: |
            [Unit]
            Description=Node Exporter
            Wants=network-online.target
            After=network-online.target

            [Service]
            User={{ node_exporter_user }}
            Group={{ node_exporter_user }}
            Type=simple
            ExecStart=/usr/local/bin/node_exporter

            [Install]
            WantedBy=multi-user.target
          dest: "/etc/systemd/system/node_exporter.service"
          mode: "0644"
        notify:
          - "reload systemd"
          - "restart node_exporter"

      - name: "Start and enable Node Exporter"
        service:
          name: "node_exporter"
          state: "started"
          enabled: true

  - name: "Install Grafana"
    hosts: "monitoring"
    become: true

    tasks:
      - name: "Add Grafana APT key"
        apt_key:
          url: "https://packages.grafana.com/gpg.key"
          state: "present"
        when: "onigirazu_os_family == 'Debian'"

      - name: "Add Grafana repository"
        apt_repository:
          repo: "deb https://packages.grafana.com/oss/deb stable main"
          state: "present"
        when: "onigirazu_os_family == 'Debian'"

      - name: "Install Grafana"
        package:
          name: "grafana"
          state: "present"
          update_cache: true

      - name: "Configure Grafana"
        template:
          src: "./templates/grafana.ini.j2"
          dest: "/etc/grafana/grafana.ini"
          backup: true
        notify: "restart grafana"

      - name: "Start and enable Grafana"
        service:
          name: "grafana-server"
          state: "started"
          enabled: true

    handlers:
      - name: "reload systemd"
        systemd:
          daemon_reload: true

      - name: "restart prometheus"
        service:
          name: "prometheus"
          state: "restarted"

      - name: "restart node_exporter"
        service:
          name: "node_exporter"
          state: "restarted"

      - name: "restart grafana"
        service:
          name: "grafana-server"
          state: "restarted"
```

### Alerting Configuration

```yaml
# alerting-setup.yml
name: "Alerting Configuration"
plays:
  - name: "Configure Alertmanager"
    hosts: "monitoring"
    become: true

    vars:
      alertmanager_version: "0.25.0"
      alertmanager_user: "alertmanager"
      alertmanager_config_dir: "/etc/alertmanager"
      alertmanager_data_dir: "/var/lib/alertmanager"

      slack_webhook_url: "{{ vault_slack_webhook_url }}"
      email_smtp_server: "smtp.example.com"
      email_from: "alerts@example.com"
      email_to: "ops@example.com"

    tasks:
      - name: "Create alertmanager user"
        user:
          name: "{{ alertmanager_user }}"
          system: true
          shell: "/bin/false"
          home: "{{ alertmanager_data_dir }}"
          create_home: false

      - name: "Create alertmanager directories"
        file:
          path: "{{ item }}"
          state: "directory"
          owner: "{{ alertmanager_user }}"
          group: "{{ alertmanager_user }}"
          mode: "0755"
        loop:
          - "{{ alertmanager_config_dir }}"
          - "{{ alertmanager_data_dir }}"

      - name: "Download Alertmanager"
        get_url:
          url: "https://github.com/prometheus/alertmanager/releases/download/v{{ alertmanager_version }}/alertmanager-{{ alertmanager_version }}.linux-amd64.tar.gz"
          dest: "/tmp/alertmanager-{{ alertmanager_version }}.tar.gz"
          mode: "0644"

      - name: "Extract Alertmanager"
        unarchive:
          src: "/tmp/alertmanager-{{ alertmanager_version }}.tar.gz"
          dest: "/tmp"
          remote_src: true
          creates: "/tmp/alertmanager-{{ alertmanager_version }}.linux-amd64"

      - name: "Install Alertmanager binaries"
        copy:
          src: "/tmp/alertmanager-{{ alertmanager_version }}.linux-amd64/{{ item }}"
          dest: "/usr/local/bin/{{ item }}"
          mode: "0755"
          owner: "root"
          group: "root"
          remote_src: true
        loop:
          - "alertmanager"
          - "amtool"

      - name: "Configure Alertmanager"
        template:
          src: "./templates/alertmanager.yml.j2"
          dest: "{{ alertmanager_config_dir }}/alertmanager.yml"
          owner: "{{ alertmanager_user }}"
          group: "{{ alertmanager_user }}"
          mode: "0644"
        notify: "restart alertmanager"

      - name: "Create alert rules"
        template:
          src: "./templates/alert-rules.yml.j2"
          dest: "/etc/prometheus/alert-rules.yml"
          owner: "prometheus"
          group: "prometheus"
          mode: "0644"
        notify: "restart prometheus"

      - name: "Create Alertmanager systemd service"
        template:
          src: "./templates/alertmanager.service.j2"
          dest: "/etc/systemd/system/alertmanager.service"
          mode: "0644"
        notify:
          - "reload systemd"
          - "restart alertmanager"

      - name: "Start and enable Alertmanager"
        service:
          name: "alertmanager"
          state: "started"
          enabled: true

    handlers:
      - name: "reload systemd"
        systemd:
          daemon_reload: true

      - name: "restart alertmanager"
        service:
          name: "alertmanager"
          state: "restarted"

      - name: "restart prometheus"
        service:
          name: "prometheus"
          state: "restarted"
```

This comprehensive examples documentation provides real-world use cases and practical implementations for various Onigirazu features, from basic task execution to complex infrastructure management, security hardening, and monitoring setups. Each example includes detailed explanations and can be adapted for specific environments and requirements.
