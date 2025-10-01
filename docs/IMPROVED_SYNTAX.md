# Improved YAML Syntax for Onigirazu

Onigirazu now supports a more compact and readable YAML syntax for tasks, eliminating the need for the verbose `args:` block in most cases.

## 🎯 Key Improvements

### ✅ **Before (Old Syntax)**

```yaml
- name: "List files in current directory"
  module: "command"
  args:
    command: "ls -la"
    shell: true
```

### ✨ **After (New Syntax)**

```yaml
- name: "List files in current directory"
  module: "command"
  command: "ls -la"
  shell: true
```

## 📋 **Features**

### 1. **Inline Module Arguments**

Module arguments can now be specified directly at the task level, without the `args:` wrapper:

```yaml
# File operations
- name: "Create configuration file"
  module: "file"
  path: "/tmp/config.conf"
  state: "touch"
  mode: "0644"
  owner: "root"
  group: "root"

# Copy with content
- name: "Deploy script"
  module: "copy"
  dest: "/usr/local/bin/deploy.sh"
  content: |
    #!/bin/bash
    echo "Deployment script"
  mode: "0755"
  backup: true

# Service management
- name: "Start nginx service"
  module: "service"
  name: "nginx"
  state: "started"
  enabled: true
```

### 2. **String Values Without Quotes**

Simple string values can be written without quotes (YAML standard):

```yaml
- name: "Package installation"
  module: "package"
  name: git
  state: present
```

### 3. **Complex Data Types**

Lists and objects work naturally:

```yaml
- name: "User management"
  module: "user"
  name: "appuser"
  state: "present"
  groups:
    - "users"
    - "wheel"
    - "docker"
  shell: "/bin/bash"
  create_home: true
```

### 4. **Backward Compatibility**

The old `args:` syntax is still fully supported:

```yaml
# This still works perfectly
- name: "Old syntax compatibility"
  module: "command"
  args:
    command: "echo 'Hello World'"
    shell: true
```

### 5. **Mixed Syntax**

You can even mix both approaches in the same task:

```yaml
- name: "Mixed syntax example"
  module: "copy"
  dest: "/tmp/script.sh"
  mode: "0755"
  args:
    content: |
      #!/bin/bash
      echo "Complex script content"
      # More script logic here
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
# Old way
- name: "Check disk space"
  module: "command"
  args:
    command: "df -h"
    shell: true

# New way
- name: "Check disk space"
  module: "command"
  command: "df -h"
  shell: true
```

### File Management

```yaml
# Old way
- name: "Create application directory"
  module: "file"
  args:
    path: "/opt/myapp"
    state: "directory"
    mode: "0755"
    owner: "app"
    group: "app"

# New way
- name: "Create application directory"
  module: "file"
  path: "/opt/myapp"
  state: "directory"
  mode: "0755"
  owner: "app"
  group: "app"
```

### Template Deployment

```yaml
# Old way
- name: "Deploy nginx configuration"
  module: "template"
  args:
    src: "nginx.conf.j2"
    dest: "/etc/nginx/nginx.conf"
    backup: true
    mode: "0644"
    owner: "root"
    group: "root"
  notify:
    - "restart nginx"

# New way
- name: "Deploy nginx configuration"
  module: "template"
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
# Old way
- name: "Install development tools"
  module: "package"
  args:
    name:
      - "git"
      - "curl"
      - "vim"
      - "htop"
    state: "present"

# New way
- name: "Install development tools"
  module: "package"
  name:
    - "git"
    - "curl"
    - "vim"
    - "htop"
  state: "present"
```

## 🚀 **Benefits**

1. **Reduced Verbosity**: Eliminates unnecessary `args:` wrapper
2. **Better Readability**: More natural YAML structure
3. **Faster Writing**: Less typing required
4. **Backward Compatible**: Existing playbooks continue to work
5. **Flexible**: Mix old and new syntax as needed
6. **Standard YAML**: Follows YAML best practices

## 🔄 **Migration Guide**

To migrate existing playbooks:

1. **Automatic**: No changes required - old syntax still works
2. **Gradual**: Update tasks one by one when convenient
3. **Mixed**: Use both syntaxes in the same playbook
4. **Complete**: Remove all `args:` blocks for maximum compactness

### Migration Example

```yaml
# Before
tasks:
  - name: "Setup application"
    module: "file"
    args:
      path: "/opt/app"
      state: "directory"
      mode: "0755"

# After
tasks:
  - name: "Setup application"
    module: "file"
    path: "/opt/app"
    state: "directory"
    mode: "0755"
```

## ✅ **Testing**

The new syntax has been thoroughly tested with:

- ✅ Unit tests for YAML parsing
- ✅ Integration tests with real modules
- ✅ Backward compatibility verification
- ✅ Mixed syntax scenarios
- ✅ Complex data type handling

Start using the improved syntax today for cleaner, more maintainable playbooks!
