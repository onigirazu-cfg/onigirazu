# Onigirazu Variables Cheat Sheet

Quick reference for the most commonly used Onigirazu variables.

## 🚀 Quick Start

Enable fact gathering in your play:

```yaml
plays:
  - name: My Play
    hosts: all
    gather_facts: true  # ← Enable this!
    tasks:
      # Your tasks here
```

---

## 📋 Most Common Variables

### Basic Host Info

```yaml
{{ onigirazu_hostname }}              # webserver01
{{ onigirazu_host }}                  # 192.168.1.10
{{ onigirazu_port }}                  # 22
{{ onigirazu_user }}                  # deploy
{{ onigirazu_fqdn }}                  # webserver01.example.com
```

### Operating System

```yaml
{{ onigirazu_os_family }}             # Debian, RedHat, Darwin
{{ onigirazu_distribution }}          # ubuntu, centos, fedora
{{ onigirazu_distribution_version }}  # 24.04, 8.5
{{ onigirazu_architecture }}          # x86_64, aarch64
{{ onigirazu_kernel }}                # Linux, Darwin
```

### Hardware

```yaml
{{ onigirazu_processor_cores }}       # 16
{{ onigirazu_memtotal_mb }}           # 15Gi
```

### User & Environment

```yaml
{{ onigirazu_user_id }}               # usx
{{ onigirazu_env.HOME }}              # /home/usx
{{ onigirazu_env.PATH }}              # /usr/local/bin:/usr/bin
```

### Date & Time

```yaml
{{ onigirazu_date_time.iso8601 }}     # 2025-10-08T18:26:30+02:00
{{ onigirazu_date_time.date }}        # 2025-10-08
{{ onigirazu_date_time.time }}        # 18:26:30
{{ onigirazu_date_time.epoch }}       # 1759940790
{{ onigirazu_date_time.weekday }}     # Wednesday
```

### Network

```yaml
{{ onigirazu_default_ipv4.address }}  # 192.168.1.10
```

---

## 💡 Common Use Cases

### 1. OS-Specific Tasks

```yaml
tasks:
  - name: Install package (Debian)
    apt:
      name: nginx
    when: onigirazu_os_family == "Debian"

  - name: Install package (RedHat)
    yum:
      name: nginx
    when: onigirazu_os_family == "RedHat"
```

### 2. Create Timestamped Files

```yaml
tasks:
  - name: Create backup with timestamp
    command:
      cmd: "cp /etc/config /backup/config-{{ onigirazu_date_time.epoch }}.bak"

  - name: Create log file
    file:
      path: "/var/log/deploy-{{ onigirazu_date_time.date }}.log"
      state: touch
```

### 3. Use Home Directory

```yaml
tasks:
  - name: Create app directory in home
    file:
      path: "{{ onigirazu_env.HOME }}/myapp"
      state: directory

  - name: Deploy config to home
    copy:
      src: config.yml
      dest: "{{ onigirazu_env.HOME }}/.config/myapp/config.yml"
```

### 4. Architecture-Specific Downloads

```yaml
tasks:
  - name: Download binary for architecture
    get_url:
      url: "https://example.com/app-{{ onigirazu_architecture }}.tar.gz"
      dest: "/tmp/app.tar.gz"
```

### 5. Generate System Report

```yaml
tasks:
  - name: Create system report
    copy:
      content: |
        System Report
        =============
        Hostname: {{ onigirazu_hostname }}
        FQDN: {{ onigirazu_fqdn }}
        OS: {{ onigirazu_distribution }} {{ onigirazu_distribution_version }}
        Architecture: {{ onigirazu_architecture }}
        Kernel: {{ onigirazu_kernel }} {{ onigirazu_kernel_version }}
        CPU Cores: {{ onigirazu_processor_cores }}
        Memory: {{ onigirazu_memtotal_mb }}
        IP Address: {{ onigirazu_default_ipv4.address }}
        User: {{ onigirazu_user_id }}
        Home: {{ onigirazu_env.HOME }}
        Generated: {{ onigirazu_date_time.iso8601 }}
      dest: "/tmp/system-report.txt"
```

---

## 🔧 Inventory Variables

### Connection Settings

```yaml
# inventory.yml
groups:
  webservers:
    hosts:
      web01:
        onigirazu_host: 192.168.1.10
        onigirazu_port: 22
        onigirazu_user: deploy
        onigirazu_ssh_private_key_file: ~/.ssh/id_rsa
        onigirazu_ssh_common_args: '-o StrictHostKeyChecking=no'
```

### Privilege Escalation

```yaml
onigirazu_become: true
onigirazu_become_user: root
onigirazu_become_method: sudo
```

---

## 🎯 Play Variables with Templates

```yaml
plays:
  - name: Deploy App
    hosts: all
    gather_facts: true
    vars:
      app_dir: "{{ onigirazu_env.HOME }}/myapp"
      backup_dir: "/backup/{{ onigirazu_hostname }}"
      log_file: "/var/log/deploy-{{ onigirazu_date_time.epoch }}.log"
    tasks:
      - name: Create directories
        file:
          path: "{{ item }}"
          state: directory
        loop:
          items:
            - "{{ app_dir }}"
            - "{{ backup_dir }}"

      # For comprehensive loop documentation, see LOOPS_GUIDE.md
      # Examples:
      - name: Loop with numeric range
        file:
          path: "/data/vol{{ item }}"
          state: directory
        loop:
          range: "1-10"

      - name: Loop with character range
        file:
          path: "/mnt/{{ item }}"
          state: directory
        loop:
          range: "a-z"
```

---

## 🐛 Debugging Variables

### Show All Variables

```yaml
tasks:
  - name: Show all host variables
    debug:
      var: hostvars[inventory_hostname]
```

### Show Specific Variable

```yaml
tasks:
  - name: Show OS family
    debug:
      msg: "OS Family: {{ onigirazu_os_family }}"
```

### Show Multiple Variables

```yaml
tasks:
  - name: Show system info
    debug:
      msg: |
        Hostname: {{ onigirazu_hostname }}
        OS: {{ onigirazu_os_family }}
        Arch: {{ onigirazu_architecture }}
        Cores: {{ onigirazu_processor_cores }}
```

---

## ⚠️ Common Pitfalls

### 1. Forgot to Enable Facts

❌ **Wrong:**

```yaml
plays:
  - name: My Play
    hosts: all
    # gather_facts not set!
    tasks:
      - debug:
          msg: "{{ onigirazu_os_family }}"  # Will fail!
```

✅ **Correct:**

```yaml
plays:
  - name: My Play
    hosts: all
    gather_facts: true  # ← Add this!
    tasks:
      - debug:
          msg: "{{ onigirazu_os_family }}"  # Works!
```

### 2. Using Variables Before Facts Are Gathered

❌ **Wrong:**

```yaml
plays:
  - name: My Play
    hosts: all
    vars:
      app_dir: "{{ onigirazu_env.HOME }}/app"  # Facts not available yet!
    gather_facts: true
```

✅ **Correct:**

```yaml
plays:
  - name: My Play
    hosts: all
    gather_facts: true  # Gather facts first!
    vars:
      app_dir: "{{ onigirazu_env.HOME }}/app"  # Now it works!
```

### 3. Missing Default Values

❌ **Wrong:**

```yaml
tasks:
  - name: Use optional variable
    debug:
      msg: "{{ custom_var }}"  # Fails if not defined!
```

✅ **Correct:**

```yaml
tasks:
  - name: Use optional variable
    debug:
      msg: "{{ custom_var | default('default_value') }}"
```

---

## 📚 Full Documentation

For complete documentation, see:

- [Configuration Reference](CONFIGURATION_REFERENCE.md)
- [Quick Start Configuration](QUICK_START_CONFIGURATION.md)
- [Inventory Formats](INVENTORY_FORMATS.md)

---

## 🔗 Quick Links

| Topic | Link |
|-------|------|
| Configuration | [CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md) |
| Quick Start | [QUICK_START_CONFIGURATION.md](QUICK_START_CONFIGURATION.md) |
| All Formats | [INVENTORY_FORMATS.md](INVENTORY_FORMATS.md) |
| Modules | [modules/README.md](modules/README.md) |
| Playbook Examples | [examples/README.md](examples/README.md) |

---

*Quick reference for Onigirazu v1.x*
