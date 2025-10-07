# Quick Start Guide

Get up and running with Onigirazu in minutes! This guide will walk you through installation, basic configuration, and your first automation tasks.

## 🚀 Installation

### Prerequisites

- Go 1.19 or later
- Git
- SSH access to target hosts

### Install from Source

```bash
# Clone the repository
git clone https://github.com/your-org/onigirazu.git
cd onigirazu

# Build the binary
go build -o onigirazu ./cmd/onigirazu

# Install to system PATH (optional)
sudo mv onigirazu /usr/local/bin/
```

### Install using Go

```bash
go install github.com/your-org/onigirazu/cmd/onigirazu@latest
```

### Verify Installation

```bash
onigirazu --version
```

## 📝 Basic Configuration

### 1. Create Inventory File

Create an inventory file to define your target hosts:

```yaml
# inventory.yml
hosts:
  - name: "webserver1"
    address: "192.168.1.10"
    user: "admin"
    key_file: "~/.ssh/id_rsa"
    vars:
      role: "webserver"
      environment: "production"

  - name: "webserver2"
    address: "192.168.1.11"
    user: "admin"
    key_file: "~/.ssh/id_rsa"
    vars:
      role: "webserver"
      environment: "production"

groups:
  webservers:
    hosts:
      webserver1: {}
      webserver2: {}
    vars:
      http_port: 80
      max_clients: 200

  production:
    children:
      - "webservers"
    vars:
      environment: "production"
      backup_enabled: true
```

### 2. Test Connectivity

Verify that Onigirazu can connect to your hosts:

```bash
onigirazu ping -i inventory.yml all
```

Expected output:

```
webserver1 | SUCCESS => {
    "ping": "pong"
}
webserver2 | SUCCESS => {
    "ping": "pong"
}
```

## 🎯 Your First Playbook

### 1. Create a Simple Playbook

Create your first playbook to gather system information:

```yaml
# first-playbook.yml
name: "My First Playbook"
plays:
  - name: "Gather System Information"
    hosts: "all"
    tasks:
      - name: "Gather system facts"
        facts:
        register: "system_info"

      - name: "Display hostname"
        debug:
          msg: "Hello from {{ ansible_hostname }}!"

      - name: "Show system information"
        debug:
          msg: |
            System: {{ ansible_distribution }} {{ ansible_distribution_version }}
            Architecture: {{ ansible_architecture }}
            Memory: {{ ansible_memtotal_mb }}MB
            CPU Cores: {{ ansible_processor_count }}
```

### 2. Run the Playbook

Execute your first playbook:

```bash
onigirazu-playbook -i inventory.yml first-playbook.yml
```

Expected output:

```
PLAY [Gather System Information] **********************************************

TASK [Gather system facts] ***************************************************
ok: [webserver1]
ok: [webserver2]

TASK [Display hostname] ******************************************************
ok: [webserver1] => {
    "msg": "Hello from webserver1!"
}
ok: [webserver2] => {
    "msg": "Hello from webserver2!"
}

TASK [Show system information] ***********************************************
ok: [webserver1] => {
    "msg": "System: Ubuntu 20.04\nArchitecture: x86_64\nMemory: 2048MB\nCPU Cores: 2"
}
ok: [webserver2] => {
    "msg": "System: Ubuntu 20.04\nArchitecture: x86_64\nMemory: 4096MB\nCPU Cores: 4"
}

PLAY RECAP ********************************************************************
webserver1                 : ok=3    changed=0    unreachable=0    failed=0
webserver2                 : ok=3    changed=0    unreachable=0    failed=0
```

## 🔧 Common Tasks

### Package Management

Install and manage packages across your infrastructure:

```yaml
# package-management.yml
name: "Package Management"
plays:
  - name: "Install Essential Packages"
    hosts: "all"
    become: true
    tasks:
      - name: "Update package cache"
        package:
          name: "*"
          state: "latest"
          update_cache: true

      - name: "Install essential tools"
        package:
          name:
            - "curl"
            - "wget"
            - "git"
            - "vim"
            - "htop"
          state: "present"

      - name: "Install web server"
        package:
          name: "nginx"
          state: "present"
        notify: "start nginx"

    handlers:
      - name: "start nginx"
        service:
          name: "nginx"
          state: "started"
          enabled: true
```

Run the playbook:

```bash
onigirazu-playbook -i inventory.yml package-management.yml --become
```

### File Management

Manage configuration files and directories:

```yaml
# file-management.yml
name: "File Management"
plays:
  - name: "Configure Web Server"
    hosts: "webservers"
    become: true
    vars:
      nginx_config: |
        server {
            listen 80;
            server_name {{ inventory_hostname }};
            root /var/www/html;
            index index.html;

            location / {
                try_files $uri $uri/ =404;
            }
        }

    tasks:
      - name: "Create web directory"
        file:
          path: "/var/www/html"
          state: "directory"
          owner: "www-data"
          group: "www-data"
          mode: "0755"

      - name: "Create index page"
        copy:
          content: |
            <!DOCTYPE html>
            <html>
            <head>
                <title>Welcome to {{ inventory_hostname }}</title>
            </head>
            <body>
                <h1>Hello from {{ inventory_hostname }}!</h1>
                <p>This server is managed by Onigirazu.</p>
                <p>Environment: {{ environment }}</p>
            </body>
            </html>
          dest: "/var/www/html/index.html"
          owner: "www-data"
          group: "www-data"
          mode: "0644"

      - name: "Configure nginx site"
        copy:
          content: "{{ nginx_config }}"
          dest: "/etc/nginx/sites-available/default"
          backup: true
        notify: "reload nginx"

    handlers:
      - name: "reload nginx"
        service:
          name: "nginx"
          state: "reloaded"
```

### Service Management

Control system services:

```yaml
# service-management.yml
name: "Service Management"
plays:
  - name: "Manage Services"
    hosts: "all"
    become: true
    tasks:
      - name: "Ensure nginx is running"
        service:
          name: "nginx"
          state: "started"
          enabled: true

      - name: "Check service status"
        command:
          cmd: "systemctl is-active nginx"
        register: "nginx_status"
        changed_when: false

      - name: "Display service status"
        debug:
          msg: "Nginx status: {{ nginx_status.stdout }}"
```

## 🔍 Advanced Features

### Using Variables

Create a variables file for environment-specific configuration:

```yaml
# vars/production.yml
environment: "production"
db_host: "prod-db.example.com"
db_port: 5432
app_version: "v1.2.3"
backup_enabled: true
monitoring_enabled: true

# Security settings
ssh_port: 2222
firewall_enabled: true
fail2ban_enabled: true
```

Use variables in your playbook:

```yaml
# advanced-playbook.yml
name: "Advanced Configuration"
plays:
  - name: "Configure Application"
    hosts: "webservers"
    vars_files:
      - "vars/{{ environment }}.yml"

    tasks:
      - name: "Display environment"
        debug:
          msg: "Configuring {{ environment }} environment"

      - name: "Configure application"
        template:
          src: "templates/app.conf.j2"
          dest: "/etc/myapp/app.conf"
          backup: true
        vars:
          database_url: "postgresql://user:pass@{{ db_host }}:{{ db_port }}/myapp"
```

### Conditional Execution

Use conditions to control task execution:

```yaml
# conditional-tasks.yml
name: "Conditional Tasks"
plays:
  - name: "OS-Specific Tasks"
    hosts: "all"
    tasks:
      - name: "Install package (Ubuntu/Debian)"
        apt:
          name: "apache2"
          state: "present"
        when: "ansible_os_family == 'Debian'"

      - name: "Install package (CentOS/RHEL)"
        yum:
          name: "httpd"
          state: "present"
        when: "ansible_os_family == 'RedHat'"

      - name: "Configure firewall (if enabled)"
        ufw:
          rule: "allow"
          port: "80"
        when: "firewall_enabled | default(false)"
```

### Loops and Iteration

Process multiple items efficiently:

```yaml
# loops-example.yml
name: "Loops and Iteration"
plays:
  - name: "Create Multiple Users"
    hosts: "all"
    become: true
    vars:
      users:
        - { name: "alice", uid: 1001, groups: ["sudo", "docker"] }
        - { name: "bob", uid: 1002, groups: ["docker"] }
        - { name: "charlie", uid: 1003, groups: ["sudo"] }

    tasks:
      - name: "Create user accounts"
        user:
          name: "{{ item.name }}"
          uid: "{{ item.uid }}"
          groups: "{{ item.groups }}"
          shell: "/bin/bash"
          create_home: true
        loop: "{{ users }}"

      - name: "Install packages"
        package:
          name: "{{ item }}"
          state: "present"
        loop:
          - "git"
          - "curl"
          - "vim"
          - "htop"
```

## 🛡️ Security Best Practices

### Secure Inventory

Use encrypted variables for sensitive data:

```bash
# Create encrypted variable file
onigirazu-vault create vars/secrets.yml
```

```yaml
# vars/secrets.yml (encrypted)
$ANSIBLE_VAULT;1.1;AES256
66386439653...
```

### SSH Key Authentication

Configure SSH key-based authentication:

```yaml
# ssh-setup.yml
name: "SSH Security Setup"
plays:
  - name: "Configure SSH Security"
    hosts: "all"
    become: true
    tasks:
      - name: "Disable password authentication"
        lineinfile:
          path: "/etc/ssh/sshd_config"
          regexp: "^#?PasswordAuthentication"
          line: "PasswordAuthentication no"
          backup: true
        notify: "restart ssh"

      - name: "Disable root login"
        lineinfile:
          path: "/etc/ssh/sshd_config"
          regexp: "^#?PermitRootLogin"
          line: "PermitRootLogin no"
          backup: true
        notify: "restart ssh"

    handlers:
      - name: "restart ssh"
        service:
          name: "sshd"
          state: "restarted"
```

## 📊 Monitoring and Logging

### Enable Verbose Output

Get detailed execution information:

```bash
# Verbose output
onigirazu-playbook -i inventory.yml playbook.yml -v

# Very verbose output
onigirazu-playbook -i inventory.yml playbook.yml -vv

# Debug output
onigirazu-playbook -i inventory.yml playbook.yml -vvv
```

### Check Mode (Dry Run)

Test your playbooks without making changes:

```bash
onigirazu-playbook -i inventory.yml playbook.yml --check
```

### Diff Mode

See what changes would be made:

```bash
onigirazu-playbook -i inventory.yml playbook.yml --diff
```

## 🔧 Command Line Options

### Common Options

```bash
# Basic execution
onigirazu-playbook -i inventory.yml playbook.yml

# Limit to specific hosts
onigirazu-playbook -i inventory.yml playbook.yml --limit webservers

# Run with elevated privileges
onigirazu-playbook -i inventory.yml playbook.yml --become

# Use specific user
onigirazu-playbook -i inventory.yml playbook.yml --become-user root

# Set extra variables
onigirazu-playbook -i inventory.yml playbook.yml --extra-vars "version=1.2.3"

# Skip specific tags
onigirazu-playbook -i inventory.yml playbook.yml --skip-tags "database"

# Run only specific tags
onigirazu-playbook -i inventory.yml playbook.yml --tags "webserver,nginx"
```

### Ad-hoc Commands

Run single tasks without a playbook:

```bash
# Ping all hosts
onigirazu -i inventory.yml all -m ping

# Run a command
onigirazu -i inventory.yml webservers -m command -a "uptime"

# Copy a file
onigirazu -i inventory.yml all -m copy -a "src=/etc/hosts dest=/tmp/hosts"

# Install a package
onigirazu -i inventory.yml all -m package -a "name=htop state=present" --become
```

## 🎯 Next Steps

Now that you've completed the quick start guide, explore these advanced topics:

1. **[Architecture Overview](./architecture.md)** - Understand Onigirazu's internal structure
2. **[Core Modules](./modules/README.md)** - Learn about all available modules
3. **[Plugin Development](./plugins/README.md)** - Create custom modules and plugins
4. **[Security Guide](./security.md)** - Implement security best practices
5. **[Monitoring & Metrics](./monitoring.md)** - Set up comprehensive monitoring
6. **[Examples](./examples/README.md)** - Explore real-world use cases

## 🆘 Getting Help

- **Documentation**: Browse the complete documentation in the `docs/` directory
- **Examples**: Check the `examples/` directory for more complex scenarios
- **Issues**: Report bugs and request features on GitHub
- **Community**: Join our community discussions

## 📚 Additional Resources

### Configuration Files

Create a configuration file for default settings:

```yaml
# ~/.onigirazu/config.yml
defaults:
  inventory: "./inventory.yml"
  remote_user: "admin"
  private_key_file: "~/.ssh/id_rsa"
  host_key_checking: false
  timeout: 30

logging:
  level: "info"
  file: "~/.onigirazu/logs/onigirazu.log"

security:
  vault_password_file: "~/.onigirazu/vault_pass"

performance:
  forks: 5
  poll_interval: 15
```

### Environment Variables

Set common options via environment variables:

```bash
export ONIGIRAZU_INVENTORY=./inventory.yml
export ONIGIRAZU_REMOTE_USER=admin
export ONIGIRAZU_PRIVATE_KEY_FILE=~/.ssh/id_rsa
export ONIGIRAZU_HOST_KEY_CHECKING=False
```

Congratulations! You've successfully completed the Onigirazu quick start guide. You now have the foundation to automate your infrastructure with confidence.
