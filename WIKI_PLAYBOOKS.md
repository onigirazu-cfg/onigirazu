# 📚 Playbooks

Playbooks are the foundation of Onigirazu automation. They define the tasks, hosts, and configuration for your infrastructure management.

## 📋 Overview

Playbooks are YAML files that describe the desired state of your infrastructure. They provide a declarative way to manage systems and applications.

### Key Concepts

- **Plays** - High-level tasks that run on specific hosts
- **Tasks** - Individual operations to perform
- **Modules** - The tools that perform the actual work
- **Variables** - Dynamic values used in playbooks
- **Handlers** - Tasks that run when notified

---

## 🚀 Basic Playbook Structure

### Simple Playbook

```yaml
# simple-playbook.yml
---
- name: Install and configure nginx
  hosts: webservers
  become: true
  tasks:
    - name: Install nginx
      package:
        name: nginx
        state: present
    
    - name: Start nginx service
      service:
        name: nginx
        state: started
        enabled: true
    
    - name: Create nginx configuration
      template:
        src: nginx.conf.j2
        dest: /etc/nginx/nginx.conf
        backup: true
      notify: restart nginx

  handlers:
    - name: restart nginx
      service:
        name: nginx
        state: restarted
```

### Advanced Playbook

```yaml
# advanced-playbook.yml
---
- name: Web server setup
  hosts: webservers
  become: true
  vars:
    nginx_port: 80
    nginx_user: www-data
  tasks:
    - name: Update package cache
      package:
        update_cache: true
      when: ansible_os_family == "Debian"
    
    - name: Install nginx
      package:
        name: nginx
        state: present
    
    - name: Configure nginx
      template:
        src: nginx.conf.j2
        dest: /etc/nginx/nginx.conf
        backup: true
      notify: restart nginx
    
    - name: Start nginx
      service:
        name: nginx
        state: started
        enabled: true

  handlers:
    - name: restart nginx
      service:
        name: nginx
        state: restarted

- name: Database server setup
  hosts: dbservers
  become: true
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
```

---

## 🎯 Playbook Components

### Plays

```yaml
# Play structure
- name: Play description
  hosts: target_hosts
  become: true
  vars:
    variable: value
  tasks:
    - task1
    - task2
  handlers:
    - handler1
```

### Tasks

```yaml
# Task structure
- name: Task description
  module:
    parameter: value
  when: condition
  notify: handler_name
  register: variable_name
```

### Handlers

```yaml
# Handler structure
- name: handler_name
  module:
    parameter: value
  when: condition
```

---

## 📦 Common Playbook Examples

### Web Server Setup

```yaml
# webserver-setup.yml
---
- name: Configure web server
  hosts: webservers
  become: true
  vars:
    web_root: /var/www/html
    nginx_user: www-data
  tasks:
    - name: Install nginx
      package:
        name: nginx
        state: present
    
    - name: Create web directory
      file:
        path: "{{ web_root }}"
        state: directory
        owner: "{{ nginx_user }}"
        group: "{{ nginx_user }}"
        mode: '0755'
    
    - name: Configure nginx
      template:
        src: nginx.conf.j2
        dest: /etc/nginx/nginx.conf
        backup: true
      notify: restart nginx
    
    - name: Start nginx
      service:
        name: nginx
        state: started
        enabled: true

  handlers:
    - name: restart nginx
      service:
        name: nginx
        state: restarted
```

### Database Setup

```yaml
# database-setup.yml
---
- name: Configure database server
  hosts: dbservers
  become: true
  vars:
    mysql_root_password: "{{ vault_mysql_root_password }}"
    mysql_database: myapp
    mysql_user: myapp_user
    mysql_password: "{{ vault_mysql_password }}"
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
    
    - name: Set mysql root password
      mysql_user:
        name: root
        password: "{{ mysql_root_password }}"
        login_password: ""
        login_unix_socket: /var/run/mysqld/mysqld.sock
    
    - name: Create database
      mysql_db:
        name: "{{ mysql_database }}"
        state: present
        login_user: root
        login_password: "{{ mysql_root_password }}"
    
    - name: Create database user
      mysql_user:
        name: "{{ mysql_user }}"
        password: "{{ mysql_password }}"
        priv: "{{ mysql_database }}.*:ALL"
        state: present
        login_user: root
        login_password: "{{ mysql_root_password }}"
```

### Application Deployment

```yaml
# app-deployment.yml
---
- name: Deploy application
  hosts: appservers
  become: true
  vars:
    app_name: myapp
    app_version: "1.0.0"
    app_user: myapp
    app_dir: /opt/myapp
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
    
    - name: Download application
      get_url:
        url: "https://github.com/myorg/myapp/releases/download/v{{ app_version }}/myapp-{{ app_version }}.tar.gz"
        dest: "/tmp/myapp-{{ app_version }}.tar.gz"
        mode: '0644'
    
    - name: Extract application
      unarchive:
        src: "/tmp/myapp-{{ app_version }}.tar.gz"
        dest: "{{ app_dir }}"
        owner: "{{ app_user }}"
        group: "{{ app_user }}"
        remote_src: true
    
    - name: Configure application
      template:
        src: app.conf.j2
        dest: "{{ app_dir }}/config/app.conf"
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

## 🔧 Advanced Playbook Features

### Variables

```yaml
# Variables in playbooks
---
- name: Configure with variables
  hosts: all
  vars:
    # Simple variables
    app_name: myapp
    app_version: "1.0.0"
    
    # Complex variables
    database:
      host: localhost
      port: 3306
      name: myapp
      user: myapp_user
    
    # Lists
    packages:
      - nginx
      - mysql-server
      - php-fpm
    
    # Dictionaries
    users:
      admin:
        name: admin
        password: "{{ vault_admin_password }}"
      user1:
        name: user1
        password: "{{ vault_user1_password }}"
  
  tasks:
    - name: Install packages
      package:
        name: "{{ item }}"
        state: present
      loop: "{{ packages }}"
    
    - name: Create users
      user:
        name: "{{ item.value.name }}"
        password: "{{ item.value.password }}"
      loop: "{{ users | dict2items }}"
```

### Conditionals

```yaml
# Conditional execution
---
- name: Conditional tasks
  hosts: all
  tasks:
    - name: Install nginx on Debian
      package:
        name: nginx
        state: present
      when: ansible_os_family == "Debian"
    
    - name: Install nginx on RedHat
      package:
        name: nginx
        state: present
      when: ansible_os_family == "RedHat"
    
    - name: Start nginx if installed
      service:
        name: nginx
        state: started
      when: nginx_installed is defined
```

### Loops

```yaml
# Loop through items
---
- name: Loop example
  hosts: all
  vars:
    packages:
      - nginx
      - apache2
      - mysql-server
    users:
      - name: user1
        shell: /bin/bash
      - name: user2
        shell: /bin/false
  tasks:
    - name: Install packages
      package:
        name: "{{ item }}"
        state: present
      loop: "{{ packages }}"
    
    - name: Create users
      user:
        name: "{{ item.name }}"
        shell: "{{ item.shell }}"
      loop: "{{ users }}"
```

### Handlers

```yaml
# Handlers for notifications
---
- name: Handler example
  hosts: all
  tasks:
    - name: Configure nginx
      template:
        src: nginx.conf.j2
        dest: /etc/nginx/nginx.conf
      notify: restart nginx
    
    - name: Configure apache
      template:
        src: apache.conf.j2
        dest: /etc/apache2/apache2.conf
      notify: restart apache

  handlers:
    - name: restart nginx
      service:
        name: nginx
        state: restarted
    
    - name: restart apache
      service:
        name: apache2
        state: restarted
```

---

## 🎯 Best Practices

### Playbook Organization

```yaml
# Well-organized playbook
---
- name: Infrastructure setup
  hosts: all
  become: true
  vars:
    # Variables at the top
    app_name: myapp
    app_version: "1.0.0"
  tasks:
    # Tasks in logical order
    - name: Update system
      package:
        update_cache: true
    
    - name: Install base packages
      package:
        name: "{{ item }}"
        state: present
      loop:
        - curl
        - wget
        - git
    
    - name: Configure application
      template:
        src: app.conf.j2
        dest: /etc/myapp.conf
      notify: restart application

  handlers:
    - name: restart application
      service:
        name: "{{ app_name }}"
        state: restarted
```

### Error Handling

```yaml
# Error handling
---
- name: Error handling example
  hosts: all
  tasks:
    - name: Install package
      package:
        name: nginx
        state: present
      ignore_errors: true
      register: install_result
    
    - name: Check installation
      debug:
        msg: "Package installed successfully"
      when: install_result is succeeded
    
    - name: Handle installation failure
      debug:
        msg: "Package installation failed"
      when: install_result is failed
```

### Performance Optimization

```yaml
# Performance optimization
---
- name: Optimized playbook
  hosts: all
  become: true
  tasks:
    - name: Install packages in parallel
      package:
        name: "{{ item }}"
        state: present
      loop: "{{ packages }}"
      async: 30
      poll: 0
    
    - name: Wait for package installation
      async_status:
        jid: "{{ ansible_job_id }}"
      register: job_result
      until: job_result.finished
      retries: 30
```

---

## 🔧 Advanced Features

### Tags

```yaml
# Tagged tasks
---
- name: Tagged playbook
  hosts: all
  tasks:
    - name: Install nginx
      package:
        name: nginx
        state: present
      tags: [nginx, packages]
    
    - name: Configure nginx
      template:
        src: nginx.conf.j2
        dest: /etc/nginx/nginx.conf
      tags: [nginx, config]
    
    - name: Start nginx
      service:
        name: nginx
        state: started
      tags: [nginx, services]
```

### Blocks

```yaml
# Block structure
---
- name: Block example
  hosts: all
  tasks:
    - name: Install and configure nginx
      block:
        - name: Install nginx
          package:
            name: nginx
            state: present
        
        - name: Configure nginx
          template:
            src: nginx.conf.j2
            dest: /etc/nginx/nginx.conf
        
        - name: Start nginx
          service:
            name: nginx
            state: started
      rescue:
        - name: Handle nginx installation failure
          debug:
            msg: "Nginx installation failed"
      always:
        - name: Cleanup
          file:
            path: /tmp/nginx-setup
            state: absent
```

### Include and Import

```yaml
# Include other playbooks
---
- name: Main playbook
  hosts: all
  tasks:
    - name: Include common tasks
      include: common-tasks.yml
    
    - name: Include web server tasks
      include: webserver-tasks.yml
      when: inventory_hostname in groups['webservers']
    
    - name: Include database tasks
      include: database-tasks.yml
      when: inventory_hostname in groups['dbservers']
```

---

## 📊 Playbook Execution

### Running Playbooks

```bash
# Basic execution
onigirazu apply playbook.yml -i inventory.yml

# With options
onigirazu apply playbook.yml -i inventory.yml --check --diff

# Specific hosts
onigirazu apply playbook.yml -i inventory.yml --limit webservers

# Specific tags
onigirazu apply playbook.yml -i inventory.yml --tags nginx

# Parallel execution
onigirazu apply playbook.yml -i inventory.yml --parallel 10
```

### Playbook Validation

```bash
# Syntax check
onigirazu apply playbook.yml -i inventory.yml --syntax-check

# Dry run
onigirazu apply playbook.yml -i inventory.yml --check

# Validate inventory
onigirazu apply playbook.yml -i inventory.yml --validate
```

---

## 📚 Related Documentation

- [Quick Start](Quick-Start) - Getting started
- [Modules](Modules) - Module reference
- [Variables](Variables) - Variable usage
- [Troubleshooting](Troubleshooting) - Common issues

---

## 🎯 Summary

### Playbook Checklist

1. **✅ Define hosts** - Specify target hosts
2. **✅ Set variables** - Define dynamic values
3. **✅ Write tasks** - Define operations
4. **✅ Add handlers** - Define notifications
5. **✅ Test playbook** - Validate syntax
6. **✅ Execute playbook** - Run automation

### Key Features

- **📚 Declarative** - Describe desired state
- **🔄 Idempotent** - Safe to run multiple times
- **🎯 Targeted** - Run on specific hosts
- **🔧 Flexible** - Support variables and conditionals
- **📊 Observable** - Clear output and logging

---

**📚 Playbooks make infrastructure management declarative and repeatable!**

