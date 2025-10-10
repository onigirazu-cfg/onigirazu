# 📦 Modules

Onigirazu includes 18+ built-in modules for comprehensive system management. Each module is designed for specific tasks and follows consistent patterns.

## 📋 Overview

Modules are the building blocks of Onigirazu. They provide specific functionality for different aspects of system management.

### Module Categories

- **System Modules** - Package, service, user management
- **File Modules** - File and directory operations
- **Network Modules** - Firewall, port management
- **Execution Modules** - Command, shell, script execution
- **Template Modules** - Template processing
- **Utility Modules** - Debug, facts, variables

---

## 🏗️ System Modules

### Package Module
Manages system packages using the appropriate package manager.

```yaml
# Install package
- name: Install nginx
  package:
    name: nginx
    state: present

# Remove package
- name: Remove apache
  package:
    name: apache2
    state: absent

# Update package
- name: Update nginx
  package:
    name: nginx
    state: latest
```

**Parameters:**
- `name` - Package name (required)
- `state` - Package state: present, absent, latest
- `update_cache` - Update package cache first

### Service Module
Manages system services.

```yaml
# Start service
- name: Start nginx
  service:
    name: nginx
    state: started

# Stop service
- name: Stop apache
  service:
    name: apache2
    state: stopped

# Restart service
- name: Restart nginx
  service:
    name: nginx
    state: restarted
```

**Parameters:**
- `name` - Service name (required)
- `state` - Service state: started, stopped, restarted, reloaded
- `enabled` - Enable service at boot: true, false

### User Module
Manages system users.

```yaml
# Create user
- name: Create user
  user:
    name: john
    state: present
    shell: /bin/bash
    home: /home/john

# Remove user
- name: Remove user
  user:
    name: olduser
    state: absent
    remove: true
```

**Parameters:**
- `name` - Username (required)
- `state` - User state: present, absent
- `shell` - User shell
- `home` - Home directory
- `groups` - User groups

### Group Module
Manages system groups.

```yaml
# Create group
- name: Create group
  group:
    name: developers
    state: present

# Remove group
- name: Remove group
  group:
    name: oldgroup
    state: absent
```

**Parameters:**
- `name` - Group name (required)
- `state` - Group state: present, absent
- `gid` - Group ID

---

## 📁 File Modules

### File Module
Manages files and directories.

```yaml
# Create file
- name: Create file
  file:
    path: /tmp/test.txt
    state: touch

# Create directory
- name: Create directory
  file:
    path: /var/www
    state: directory
    mode: '0755'

# Delete file
- name: Delete file
  file:
    path: /tmp/old.txt
    state: absent
```

**Parameters:**
- `path` - File/directory path (required)
- `state` - File state: file, directory, link, absent, touch
- `mode` - File permissions
- `owner` - File owner
- `group` - File group

### Copy Module
Copies files to remote hosts.

```yaml
# Copy file
- name: Copy file
  copy:
    src: /local/file.txt
    dest: /remote/file.txt
    mode: '0644'

# Copy with backup
- name: Copy with backup
  copy:
    src: /local/config.conf
    dest: /etc/config.conf
    backup: true
```

**Parameters:**
- `src` - Source file (required)
- `dest` - Destination path (required)
- `mode` - File permissions
- `backup` - Create backup: true, false
- `force` - Force copy: true, false

### Template Module
Processes templates with variables.

```yaml
# Process template
- name: Process template
  template:
    src: nginx.conf.j2
    dest: /etc/nginx/nginx.conf
    mode: '0644'
    backup: true
```

**Parameters:**
- `src` - Template source (required)
- `dest` - Destination path (required)
- `mode` - File permissions
- `backup` - Create backup: true, false

---

## 🌐 Network Modules

### Firewall Module
Manages firewall rules.

```yaml
# Allow port
- name: Allow HTTP
  firewall:
    port: 80
    protocol: tcp
    state: present

# Deny port
- name: Deny SSH
  firewall:
    port: 22
    protocol: tcp
    state: absent
```

**Parameters:**
- `port` - Port number (required)
- `protocol` - Protocol: tcp, udp
- `state` - Rule state: present, absent
- `source` - Source IP/network
- `destination` - Destination IP/network

### Port Module
Manages port configurations.

```yaml
# Open port
- name: Open port
  port:
    port: 8080
    state: open

# Close port
- name: Close port
  port:
    port: 8080
    state: closed
```

**Parameters:**
- `port` - Port number (required)
- `state` - Port state: open, closed
- `protocol` - Protocol: tcp, udp

---

## ⚡ Execution Modules

### Command Module
Executes commands on remote hosts.

```yaml
# Execute command
- name: Check uptime
  command: uptime

# Execute with arguments
- name: List files
  command: ls -la /tmp
```

**Parameters:**
- `command` - Command to execute (required)
- `args` - Command arguments
- `creates` - File that command creates
- `removes` - File that command removes

### Shell Module
Executes shell commands with pipes and redirection.

```yaml
# Shell command
- name: Count processes
  shell: ps aux | grep nginx | wc -l

# Complex shell command
- name: Complex command
  shell: |
    cd /var/www
    tar -czf backup.tar.gz .
    mv backup.tar.gz /tmp/
```

**Parameters:**
- `shell` - Shell command (required)
- `creates` - File that command creates
- `removes` - File that command removes

### Script Module
Executes local scripts on remote hosts.

```yaml
# Execute script
- name: Run script
  script: /local/script.sh

# Execute with arguments
- name: Run script with args
  script: /local/script.sh arg1 arg2
```

**Parameters:**
- `script` - Script path (required)
- `args` - Script arguments

---

## 🔧 Utility Modules

### Debug Module
Outputs debug information.

```yaml
# Debug message
- name: Debug message
  debug:
    msg: "Hello from Onigirazu"

# Debug variable
- name: Debug variable
  debug:
    var: ansible_facts
```

**Parameters:**
- `msg` - Debug message
- `var` - Variable to debug

### Facts Module
Gathers system facts.

```yaml
# Gather facts
- name: Gather facts
  facts:
    gather_subset: all
```

**Parameters:**
- `gather_subset` - Facts to gather: all, hardware, network, virtual

### Set Fact Module
Sets custom facts.

```yaml
# Set fact
- name: Set custom fact
  set_fact:
    custom_var: "custom_value"

# Set multiple facts
- name: Set multiple facts
  set_fact:
    app_name: "myapp"
    app_version: "1.0.0"
```

**Parameters:**
- Any custom variable name and value

---

## 🎯 Module Usage Examples

### Package Management
```bash
# Install package
onigirazu run all -m package name=nginx state=present -i inventory.yml

# Remove package
onigirazu run all -m package name=apache2 state=absent -i inventory.yml

# Update package
onigirazu run all -m package name=nginx state=latest -i inventory.yml
```

### Service Management
```bash
# Start service
onigirazu run webservers -m service name=nginx state=started -i inventory.yml

# Stop service
onigirazu run all -m service name=apache2 state=stopped -i inventory.yml

# Restart service
onigirazu run webservers -m service name=nginx state=restarted -i inventory.yml
```

### File Operations
```bash
# Create file
onigirazu run all -m file path=/tmp/test.txt state=touch -i inventory.yml

# Create directory
onigirazu run all -m file path=/var/www state=directory -i inventory.yml

# Delete file
onigirazu run all -m file path=/tmp/old.txt state=absent -i inventory.yml
```

### Command Execution
```bash
# Execute command
onigirazu run all -m command "uptime" -i inventory.yml

# Shell command
onigirazu run all -m shell "ps aux | grep nginx" -i inventory.yml
```

---

## 🔧 Advanced Module Usage

### With Variables
```yaml
# Using variables in modules
- name: Install package
  package:
    name: "{{ package_name }}"
    state: "{{ package_state }}"
```

### With Conditions
```yaml
# Conditional module execution
- name: Install package
  package:
    name: nginx
    state: present
  when: ansible_os_family == "Debian"
```

### With Loops
```yaml
# Loop through packages
- name: Install packages
  package:
    name: "{{ item }}"
    state: present
  loop:
    - nginx
    - apache2
    - mysql
```

---

## 📊 Module Performance

### Performance Tips
- **Use appropriate modules** for specific tasks
- **Combine operations** when possible
- **Use check mode** for testing
- **Leverage caching** for repeated operations

### Best Practices
- **Idempotent operations** - safe to run multiple times
- **Error handling** - check for failures
- **Resource management** - use appropriate timeouts
- **Logging** - enable verbose output for debugging

---

## 📚 Related Documentation

- [Natural Language Commands](Natural-Language-Commands)
- [Ad-hoc Commands](Ad-hoc-Commands)
- [Quick Start](Quick-Start)
- [Troubleshooting](Troubleshooting)

---

**📦 Modules provide the foundation for all Onigirazu operations!**
