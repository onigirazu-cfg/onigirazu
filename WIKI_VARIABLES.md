# 📊 Variables

Variables in Onigirazu provide dynamic configuration and flexibility for your automation tasks.

## 📋 Variable Types

### Global Variables
- **Configuration variables** - Set in configuration files
- **Environment variables** - Set via environment
- **Command line variables** - Set via command line flags

### Playbook Variables
- **Play variables** - Set at play level
- **Task variables** - Set at task level
- **Host variables** - Set per host
- **Group variables** - Set per group

### System Variables
- **Facts** - System information
- **Built-in variables** - Onigirazu internal variables
- **Runtime variables** - Set during execution

---

## 🔧 Variable Definition

### Global Variables

#### Configuration File
```yaml
# ~/.onigirazu/config.yml
defaults:
  inventory: inventory.yml
  timeout: 30s
  parallel: 5
  output: text
  verbose: false

# Global variables
variables:
  app_name: "myapp"
  app_version: "1.0.0"
  app_port: 8080
  app_user: "myapp"
  app_dir: "/opt/myapp"
```

#### Environment Variables
```bash
# Set environment variables
export ONIGIRAZU_APP_NAME="myapp"
export ONIGIRAZU_APP_VERSION="1.0.0"
export ONIGIRAZU_APP_PORT="8080"
export ONIGIRAZU_APP_USER="myapp"
export ONIGIRAZU_APP_DIR="/opt/myapp"
```

#### Command Line Variables
```bash
# Set variables via command line
onigirazu run all -m package name=nginx state=present -e app_name=myapp -e app_version=1.0.0 -i inventory.yml
onigirazu apply playbook.yml -e app_name=myapp -e app_version=1.0.0 -i inventory.yml
```

### Playbook Variables

#### Play Variables
```yaml
# playbook.yml
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    app_name: "myapp"
    app_version: "1.0.0"
    app_port: 8080
    app_user: "myapp"
    app_dir: "/opt/myapp"
  tasks:
    - name: Install application
      package:
        name: "{{ app_name }}"
        state: present
```

#### Task Variables
```yaml
# playbook.yml
---
- name: Deploy application
  hosts: appservers
  become: true
  tasks:
    - name: Install application
      package:
        name: "{{ app_name }}"
        state: present
      vars:
        app_name: "myapp"
        app_version: "1.0.0"
```

### Inventory Variables

#### Host Variables
```yaml
# inventory.yml
all:
  children:
    webservers:
      hosts:
        web1:
          ansible_host: 192.168.1.10
          ansible_user: ubuntu
          app_name: "webapp"
          app_port: 80
        web2:
          ansible_host: 192.168.1.11
          ansible_user: ubuntu
          app_name: "webapp"
          app_port: 80
    dbservers:
      hosts:
        db1:
          ansible_host: 192.168.1.20
          ansible_user: ubuntu
          db_name: "mydb"
          db_user: "dbuser"
          db_password: "{{ vault_db_password }}"
```

#### Group Variables
```yaml
# inventory.yml
all:
  children:
    webservers:
      hosts:
        web1:
          ansible_host: 192.168.1.10
        web2:
          ansible_host: 192.168.1.11
      vars:
        app_name: "webapp"
        app_port: 80
        nginx_user: "www-data"
    dbservers:
      hosts:
        db1:
          ansible_host: 192.168.1.20
      vars:
        db_name: "mydb"
        db_user: "dbuser"
        db_password: "{{ vault_db_password }}"
```

---

## 📊 Variable Usage

### Basic Variable Usage

```yaml
# Basic variable usage
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    app_name: "myapp"
    app_version: "1.0.0"
    app_port: 8080
  tasks:
    - name: Install application
      package:
        name: "{{ app_name }}"
        state: present
    
    - name: Configure application
      template:
        src: app.conf.j2
        dest: "/etc/{{ app_name }}/app.conf"
        mode: '0644'
    
    - name: Start application
      service:
        name: "{{ app_name }}"
        state: started
        enabled: true
```

### Advanced Variable Usage

```yaml
# Advanced variable usage
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    app_name: "myapp"
    app_version: "1.0.0"
    app_port: 8080
    app_user: "myapp"
    app_dir: "/opt/{{ app_name }}"
    app_config: "/etc/{{ app_name }}/config.yml"
    app_logs: "/var/log/{{ app_name }}"
  tasks:
    - name: Create application user
      user:
        name: "{{ app_user }}"
        system: true
        shell: /bin/false
        home: "{{ app_dir }}"
    
    - name: Create application directory
      file:
        path: "{{ app_dir }}"
        state: directory
        owner: "{{ app_user }}"
        group: "{{ app_user }}"
        mode: '0755'
    
    - name: Create application config
      template:
        src: app-config.yml.j2
        dest: "{{ app_config }}"
        owner: "{{ app_user }}"
        group: "{{ app_user }}"
        mode: '0644'
      notify: restart application
    
    - name: Start application
      service:
        name: "{{ app_name }}"
        state: started
        enabled: true

  handlers:
    - name: restart application
      service:
        name: "{{ app_name }}"
        state: restarted
```

---

## 🔧 Variable Interpolation

### String Interpolation

```yaml
# String interpolation
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    app_name: "myapp"
    app_version: "1.0.0"
    app_port: 8080
  tasks:
    - name: Create application directory
      file:
        path: "/opt/{{ app_name }}-{{ app_version }}"
        state: directory
    
    - name: Create application config
      template:
        src: app.conf.j2
        dest: "/etc/{{ app_name }}/app.conf"
        mode: '0644'
    
    - name: Create application log directory
      file:
        path: "/var/log/{{ app_name }}"
        state: directory
        owner: "{{ app_user }}"
        group: "{{ app_user }}"
        mode: '0755'
```

### Complex Variable Usage

```yaml
# Complex variable usage
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    app_name: "myapp"
    app_version: "1.0.0"
    app_port: 8080
    app_user: "myapp"
    app_dir: "/opt/{{ app_name }}"
    app_config: "/etc/{{ app_name }}/config.yml"
    app_logs: "/var/log/{{ app_name }}"
    app_pid: "/var/run/{{ app_name }}.pid"
    app_socket: "/var/run/{{ app_name }}.sock"
  tasks:
    - name: Create application directories
      file:
        path: "{{ item }}"
        state: directory
        owner: "{{ app_user }}"
        group: "{{ app_user }}"
        mode: '0755'
      loop:
        - "{{ app_dir }}"
        - "{{ app_logs }}"
        - "/var/run/{{ app_name }}"
    
    - name: Create application config
      template:
        src: app-config.yml.j2
        dest: "{{ app_config }}"
        owner: "{{ app_user }}"
        group: "{{ app_user }}"
        mode: '0644'
      notify: restart application
    
    - name: Start application
      service:
        name: "{{ app_name }}"
        state: started
        enabled: true
```

---

## 📊 System Variables

### Facts

```yaml
# System facts
---
- name: Deploy application
  hosts: appservers
  become: true
  tasks:
    - name: Install application
      package:
        name: "{{ app_name }}"
        state: present
      when: ansible_os_family == "Debian"
    
    - name: Configure application
      template:
        src: app.conf.j2
        dest: "/etc/{{ app_name }}/app.conf"
        mode: '0644'
      when: ansible_os_family == "Debian"
    
    - name: Start application
      service:
        name: "{{ app_name }}"
        state: started
        enabled: true
      when: ansible_os_family == "Debian"
```

### Built-in Variables

```yaml
# Built-in variables
---
- name: Deploy application
  hosts: appservers
  become: true
  tasks:
    - name: Install application
      package:
        name: "{{ app_name }}"
        state: present
    
    - name: Configure application
      template:
        src: app.conf.j2
        dest: "/etc/{{ app_name }}/app.conf"
        mode: '0644'
      notify: restart application
    
    - name: Start application
      service:
        name: "{{ app_name }}"
        state: started
        enabled: true
      when: ansible_os_family == "Debian"
```

---

## 🔧 Variable Precedence

### Precedence Order

1. **Command line variables** - Highest priority
2. **Play variables** - Second priority
3. **Task variables** - Third priority
4. **Host variables** - Fourth priority
5. **Group variables** - Fifth priority
6. **Global variables** - Sixth priority
7. **Default values** - Lowest priority

### Example

```yaml
# Variable precedence example
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    app_name: "myapp"  # Play variable
    app_version: "1.0.0"
    app_port: 8080
  tasks:
    - name: Install application
      package:
        name: "{{ app_name }}"
        state: present
      vars:
        app_name: "webapp"  # Task variable (overrides play variable)
    
    - name: Configure application
      template:
        src: app.conf.j2
        dest: "/etc/{{ app_name }}/app.conf"
        mode: '0644'
      # Uses play variable (app_name: "myapp")
```

---

## 🔧 Variable Validation

### Required Variables

```yaml
# Required variables
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    app_name: "{{ app_name | default('myapp') }}"
    app_version: "{{ app_version | default('1.0.0') }}"
    app_port: "{{ app_port | default(8080) }}"
  tasks:
    - name: Validate required variables
      assert:
        that:
          - app_name is defined
          - app_version is defined
          - app_port is defined
        fail_msg: "Required variables are missing"
    
    - name: Install application
      package:
        name: "{{ app_name }}"
        state: present
```

### Variable Types

```yaml
# Variable type validation
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    app_name: "{{ app_name | default('myapp') }}"
    app_version: "{{ app_version | default('1.0.0') }}"
    app_port: "{{ app_port | default(8080) | int }}"
  tasks:
    - name: Validate variable types
      assert:
        that:
          - app_name is string
          - app_version is string
          - app_port is number
        fail_msg: "Invalid variable types"
    
    - name: Install application
      package:
        name: "{{ app_name }}"
        state: present
```

---

## 🔧 Variable Filters

### String Filters

```yaml
# String filters
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    app_name: "myapp"
    app_version: "1.0.0"
    app_port: 8080
  tasks:
    - name: Create application directory
      file:
        path: "/opt/{{ app_name | upper }}"
        state: directory
    
    - name: Create application config
      template:
        src: app.conf.j2
        dest: "/etc/{{ app_name | lower }}/app.conf"
        mode: '0644'
    
    - name: Create application log directory
      file:
        path: "/var/log/{{ app_name | title }}"
        state: directory
```

### Number Filters

```yaml
# Number filters
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    app_name: "myapp"
    app_version: "1.0.0"
    app_port: 8080
  tasks:
    - name: Create application directory
      file:
        path: "/opt/{{ app_name }}-{{ app_version | int }}"
        state: directory
    
    - name: Create application config
      template:
        src: app.conf.j2
        dest: "/etc/{{ app_name }}/app.conf"
        mode: '0644'
    
    - name: Create application log directory
      file:
        path: "/var/log/{{ app_name }}"
        state: directory
```

---

## 🔧 Variable Examples

### Application Deployment

```yaml
# Application deployment with variables
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    app_name: "myapp"
    app_version: "1.0.0"
    app_port: 8080
    app_user: "myapp"
    app_dir: "/opt/{{ app_name }}"
    app_config: "/etc/{{ app_name }}/config.yml"
    app_logs: "/var/log/{{ app_name }}"
    app_pid: "/var/run/{{ app_name }}.pid"
    app_socket: "/var/run/{{ app_name }}.sock"
  tasks:
    - name: Create application user
      user:
        name: "{{ app_user }}"
        system: true
        shell: /bin/false
        home: "{{ app_dir }}"
    
    - name: Create application directories
      file:
        path: "{{ item }}"
        state: directory
        owner: "{{ app_user }}"
        group: "{{ app_user }}"
        mode: '0755'
      loop:
        - "{{ app_dir }}"
        - "{{ app_logs }}"
        - "/var/run/{{ app_name }}"
    
    - name: Create application config
      template:
        src: app-config.yml.j2
        dest: "{{ app_config }}"
        owner: "{{ app_user }}"
        group: "{{ app_user }}"
        mode: '0644'
      notify: restart application
    
    - name: Start application
      service:
        name: "{{ app_name }}"
        state: started
        enabled: true

  handlers:
    - name: restart application
      service:
        name: "{{ app_name }}"
        state: restarted
```

### Database Configuration

```yaml
# Database configuration with variables
---
- name: Configure database
  hosts: dbservers
  become: true
  vars:
    db_name: "mydb"
    db_user: "dbuser"
    db_password: "{{ vault_db_password }}"
    db_host: "localhost"
    db_port: 3306
    db_charset: "utf8mb4"
    db_collation: "utf8mb4_unicode_ci"
  tasks:
    - name: Install mysql
      package:
        name: mysql-server
        state: present
    
    - name: Start mysql
      service:
        name: mysql
        state: started
        enabled: true
    
    - name: Create database
      mysql_db:
        name: "{{ db_name }}"
        state: present
        login_user: root
        login_password: "{{ vault_mysql_root_password }}"
    
    - name: Create database user
      mysql_user:
        name: "{{ db_user }}"
        password: "{{ db_password }}"
        priv: "{{ db_name }}.*:ALL"
        state: present
        login_user: root
        login_password: "{{ vault_mysql_root_password }}"
```

---

## 🔧 Best Practices

### Variable Organization

```yaml
# Well-organized variables
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    # Application settings
    app_name: "myapp"
    app_version: "1.0.0"
    app_port: 8080
    
    # User settings
    app_user: "myapp"
    app_group: "myapp"
    
    # Directory settings
    app_dir: "/opt/{{ app_name }}"
    app_config: "/etc/{{ app_name }}/config.yml"
    app_logs: "/var/log/{{ app_name }}"
    
    # Service settings
    app_service: "{{ app_name }}"
    app_pid: "/var/run/{{ app_name }}.pid"
    app_socket: "/var/run/{{ app_name }}.sock"
  tasks:
    # Tasks here
```

### Variable Documentation

```yaml
# Documented variables
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    # Application name
    app_name: "myapp"
    
    # Application version
    app_version: "1.0.0"
    
    # Application port
    app_port: 8080
    
    # Application user
    app_user: "myapp"
    
    # Application directory
    app_dir: "/opt/{{ app_name }}"
    
    # Application configuration file
    app_config: "/etc/{{ app_name }}/config.yml"
    
    # Application log directory
    app_logs: "/var/log/{{ app_name }}"
  tasks:
    # Tasks here
```

---

## 📚 Related Documentation

- [Playbooks](Playbooks) - Playbook reference
- [Configuration](Configuration) - Configuration options
- [Troubleshooting](Troubleshooting) - Common issues
- [API Reference](API-Reference) - API documentation

---

## 🎯 Summary

### Variable Features

- **📊 Multiple types** - Global, play, task, host, group
- **🔧 Flexible usage** - String interpolation, filters
- **📈 Precedence** - Clear priority order
- **✅ Validation** - Type and value validation
- **📚 Documentation** - Well-documented variables

### Variable Benefits

- **🔧 Flexibility** - Dynamic configuration
- **📊 Reusability** - Reusable playbooks
- **🎯 Maintainability** - Easy to maintain
- **📈 Scalability** - Handle large deployments
- **🔒 Security** - Secure variable handling

---

**📊 Variables make Onigirazu playbooks flexible and reusable!**
