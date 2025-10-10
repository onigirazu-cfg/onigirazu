# YAML Syntax for Onigirazu

Onigirazu uses a simplified YAML syntax for defining tasks with module parameters.

## 🎯 Syntax Overview

### **Standard Syntax**

```yaml
- name: "List files in current directory"
  command:
    cmd: "ls -la"
```

### **Alternative: Inline Module Type**

```yaml
- name: "List files in current directory"
  command: { cmd: "ls -la" }
```

## 📋 **Features**

### 1. **Structured Module Definition**

Module parameters are specified directly under the module name:

```yaml
# File operations
- name: "Create configuration file"
  file:
    path: "/tmp/config.conf"
    state: "touch"
    mode: "0644"
    owner: "root"
    group: "root"

# Copy with content
- name: "Deploy script"
  copy:
    dest: "/usr/local/bin/deploy.sh"
    content: |
      #!/bin/bash
      echo "Deployment script"
    mode: "0755"
    backup: true

# Service management
- name: "Start nginx service"
  service:
    name: "nginx"
    state: "started"
    enabled: true
```

### 2. **String Values Without Quotes**

Simple string values can be written without quotes (YAML standard):

```yaml
- name: "Package installation"
  package:
    name: git
    state: present
```

### 3. **Complex Data Types**

Lists and objects work naturally:

```yaml
- name: "User management"
  user:
    name: "appuser"
    state: "present"
    groups:
      - "users"
      - "wheel"
      - "docker"
    shell: "/bin/bash"
    create_home: true
```

### 4. **Inline Syntax**

For simple modules, you can use inline YAML object syntax:

```yaml
- name: "Quick package install"
  package: { name: "git", state: "present" }

- name: "Quick command"
  command: { cmd: "echo 'Hello World'" }
```

## 🔧 **Reserved Fields**

The following fields are reserved for task control and will not be passed as module arguments:

- `name` - Task name
- `module` - Module to execute
- `args` - Explicit arguments block
- `when` - Conditional execution
- `loop` - Loop control
- `register` - Variable registration
- `ignore_errors` - Error handling
- `tags` - Task tags
- `notify` - Notification handlers
- `timeout` - Task timeout
- `retries` - Retry count
- `delay` - Retry delay
- `until` - Retry condition
- `changed_when` - Change detection
- `failed_when` - Failure detection
- `include` - Task inclusion
- `serial` - Serial execution
- `retry_delay` - Retry delay

## 📖 **Complete Examples**

### Command Execution

```yaml
# Expanded syntax
- name: "Check disk space"
  command:
    cmd: "df -h"

# Inline syntax
- name: "Check disk space"
  command: { cmd: "df -h" }
```

### File Management

```yaml
# Expanded syntax
- name: "Create application directory"
  file:
    path: "/opt/myapp"
    state: "directory"
    mode: "0755"
    owner: "app"
    group: "app"

# Inline syntax
- name: "Create application directory"
  file: { path: "/opt/myapp", state: "directory", mode: "0755" }
```

### Template Deployment

```yaml
- name: "Deploy nginx configuration"
  template:
    src: "nginx.conf.j2"
    dest: "/etc/nginx/nginx.conf"
    backup: true
    mode: "0644"
    owner: "root"
    group: "root"
  notify:
    - "restart nginx"
```

### Package Management

```yaml
# Multiple packages
- name: "Install development tools"
  package:
    name:
      - "git"
      - "curl"
      - "vim"
      - "htop"
    state: "present"

# Single package (inline)
- name: "Install git"
  package: { name: "git", state: "present" }
```

## 🚀 **Benefits**

1. **Clear Structure**: Module name and parameters are clearly organized
2. **Better Readability**: Simplified YAML structure is easy to understand
3. **Flexible**: Choose between expanded or inline syntax
4. **Less Verbose**: No need for `module:` and `type:` wrappers
5. **Standard YAML**: Follows YAML best practices
6. **Consistent**: Same pattern across all modules
7. **Ansible-like**: Familiar syntax for Ansible users

## 🔄 **Syntax Styles**

You can choose the style that best fits your needs:

### Expanded Style (Recommended for Complex Tasks)

```yaml
- name: "Setup application"
  file:
    path: "/opt/app"
    state: "directory"
    mode: "0755"
    owner: "app"
    group: "app"
```

### Inline Style (Good for Simple Tasks)

```yaml
- name: "Setup application"
  file: { path: "/opt/app", state: "directory", mode: "0755" }
```

## 📚 **More Examples**

### Shell Commands

```yaml
- name: "System information"
  shell:
    cmd: |
      echo "Hostname: $(hostname)"
      echo "OS: $(uname -s)"
      echo "Kernel: $(uname -r)"
```

### Service Management

```yaml
- name: "Start and enable nginx"
  service:
    name: "nginx"
    state: "started"
    enabled: true
```

### Copy Files

```yaml
- name: "Deploy configuration"
  copy:
    src: "app.conf"
    dest: "/etc/app/app.conf"
    mode: "0644"
    backup: true
```

## ✅ **Best Practices**

1. **Use expanded syntax** for tasks with many parameters
2. **Use inline syntax** for simple, one-line tasks
3. **Use module name directly** as the field name (e.g., `package:`, `service:`, `file:`)
4. **Use quotes** for strings with special characters
5. **Indent consistently** (2 spaces recommended)
6. **Group related tasks** together in your playbooks

Start using this simplified syntax for cleaner, more maintainable playbooks!
