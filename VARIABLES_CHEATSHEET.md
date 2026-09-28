# Onigirazu Variables Cheat Sheet

Quick reference for the most commonly used Onigirazu variables.

## 🚀 Quick Start

Facts are gathered at the start of every play, as in Ansible. Skip it when a
play does not need them:

```yaml
plays:
  - name: My Play
    hosts: all
    gather_facts: false  # no facts, a faster start
    tasks:
      # Your tasks here
```

---

## 📋 Most Common Variables

Every fact exists under two names: `onigirazu_*` and the Ansible name `ansible_*`. The Ansible names are also collected in the `ansible_facts` dictionary without the prefix (`ansible_facts.os_family`).

### Always Defined

```yaml
{{ inventory_hostname }}              # web01 (name in the inventory)
{{ group_names }}                     # ["webservers"] (groups of this host)
{{ groups['webservers'] }}            # ["web01", "web02"]
{{ hostvars['web02'].ansible_host }}  # variables of another host
{{ playbook_dir }}                    # directory of the playbook
{{ ansible_check_mode }}              # true under --check
{{ ansible_diff_mode }}               # true under --diff
{{ ansible_limit }}                   # the --limit pattern (defined only with --limit)
{{ ansible_play_hosts }}              # hosts of the play's current batch still running
{{ ansible_play_batch }}              # same as ansible_play_hosts
{{ ansible_play_hosts_all }}          # every host of the play
```

### Basic Host Info

```yaml
{{ onigirazu_hostname }}              # web01 (inventory name)
{{ ansible_hostname }}                # webserver01 (short hostname of the machine)
{{ ansible_nodename }}                # webserver01.example.com
{{ onigirazu_host }}                  # 192.168.1.10 (connection address)
{{ onigirazu_port }}                  # 22
{{ onigirazu_user }}                  # deploy
{{ ansible_fqdn }}                    # webserver01.example.com (also onigirazu_fqdn)
```

### Operating System

```yaml
{{ ansible_os_family }}                    # Debian, RedHat, Darwin
{{ ansible_pkg_mgr }}                      # apt, dnf, yum, zypper, pacman, apk
{{ ansible_distribution }}                 # Ubuntu, Rocky, RedHat (Ansible spelling)
{{ onigirazu_distribution }}               # ubuntu, rocky, rhel (os-release ID)
{{ ansible_distribution_version }}         # 24.04, 9.4
{{ ansible_distribution_release }}         # noble, Blue Onyx
{{ ansible_distribution_major_version }}   # 24, 9
{{ ansible_architecture }}                 # x86_64, aarch64
{{ ansible_system }}                       # Linux, Darwin (onigirazu_kernel)
{{ ansible_kernel }}                       # 6.8.0-45-generic (onigirazu_kernel_version)
```

### Hardware

```yaml
{{ ansible_processor_vcpus }}         # 16 (also ansible_processor_cores, onigirazu_processor_cores)
{{ onigirazu_memtotal_mb }}           # 15970 (also ansible_memtotal_mb)
{{ ansible_virtualization_type }}     # docker, podman, lxc, kvm, VMware, ... or NA
{{ ansible_virtualization_role }}     # guest, or NA
```

### Local facts

```yaml
{{ ansible_local.myapp.version }}     # /etc/ansible/facts.d/myapp.fact (JSON or INI)
```

Gathered with the other facts; `setup` reads them again after a task wrote one.

### User & Environment

```yaml
{{ ansible_user_id }}                 # usx (also onigirazu_user_id)
{{ ansible_env.HOME }}                # /home/usx (also onigirazu_env.HOME)
{{ ansible_env.PATH }}                # /usr/local/bin:/usr/bin
```

Only `HOME` and `PATH` are gathered.

### Date & Time

```yaml
{{ ansible_date_time.iso8601 }}       # 2025-10-08T18:26:30+02:00
{{ ansible_date_time.date }}          # 2025-10-08
{{ ansible_date_time.time }}          # 18:26:30
{{ ansible_date_time.epoch }}         # 1759940790
{{ ansible_date_time.weekday }}       # Wednesday
```

Also `year`, `month`, `day`, `hour`, `minute`, `second`, `weekday_number`. The time is taken when facts are gathered, on the control machine.

### Network

```yaml
{{ ansible_default_ipv4.address }}    # 192.168.1.10 (only the address is gathered)
```

### Registered Results

```yaml
{{ result.rc }}  {{ result.stdout }}  {{ result.stdout_lines }}  {{ result.stderr_lines }}
{{ result.changed }}  {{ result.failed }}  {{ result.skipped }}  {{ result.msg }}
```

A failed or skipped task is registered too. In a `rescue` section `ansible_failed_task` (name, module) and `ansible_failed_result` describe the failure.

---

## 💡 Common Use Cases

### 1. OS-Specific Tasks

```yaml
tasks:
  - name: Install package (Debian)
    apt:
      name: nginx
    when: ansible_os_family == "Debian"

  - name: Install package (RedHat)
    yum:
      name: nginx
    when: ansible_os_family == "RedHat"
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
        ansible_host: 192.168.1.10
        ansible_port: 22
        ansible_user: deploy
        ansible_ssh_private_key_file: ~/.ssh/id_rsa
```

Recognised connection variables: `ansible_host`, `ansible_port`, `ansible_user`, `ansible_password`, `ansible_ssh_private_key_file`, `ansible_connection` (`local`); the `onigirazu_` spellings of the same names also work. `ansible_ssh_common_args` and `onigirazu_ssh_common_args` are not read.

### Privilege Escalation

Set `become`, `become_user` and `become_method` on the play or the task; `ansible_become*` / `onigirazu_become*` inventory variables are not read.

```yaml
plays:
  - name: Configure
    hosts: all
    become: true
    become_user: root
```

---

## 🎯 Play Variables with Templates

Play `vars` with templates are rendered per host when a task uses them:
`backup_dir: "/backup/{{ ansible_hostname }}"` is each host's own directory. Play vars may
refer to each other. Task `vars` work the same way for a single task, and so do role
defaults and vars, inventory and `vars_files` values, also inside lists and maps
(`ssh_port: "{{ custom_port | default(22) }}"` in a role's defaults). `-e` values and what
`register`/`set_fact` stored are data and are not rendered again.

```yaml
plays:
  - name: Deploy App
    hosts: all
    gather_facts: true
    vars:
      release: "2.4.1"
    tasks:
      - name: Create the application directory
        file:
          path: "{{ app_dir }}"
          state: directory
        vars:                                  # task vars: rendered per host
          app_dir: "{{ ansible_env.HOME }}/myapp-{{ release }}"

      - name: Create the backup directory
        file:
          path: "/backup/{{ ansible_hostname }}"
          state: directory

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
      msg: "{{ hostvars[inventory_hostname] }}"   # debug var takes only dotted paths
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

### 1. Turned Facts Off

With `gather_facts: false` the `onigirazu_*` and `ansible_*` facts are
undefined; remove it (facts are gathered by default) or guard with
`is defined`.

### 2. Play Vars Are Rendered per Host

A play var with a template is rendered for each host when a task uses it, so it
can use that host's facts and `inventory_hostname`:

```yaml
plays:
  - name: My Play
    hosts: all
    vars:
      app_dir: "{{ ansible_env.HOME }}/app"   # each host's own HOME
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
| Filters and Lookups | [FILTERS_GUIDE.md](FILTERS_GUIDE.md) |
| Playbook Examples | [examples/README.md](examples/README.md) |

---

*Quick reference for Onigirazu v1.x*
