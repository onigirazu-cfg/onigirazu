# Loops in Onigirazu - Complete Guide

## Table of Contents

1. [Introduction](#introduction)
2. [Loop Basics](#loop-basics)
3. [Loop Types](#loop-types)
   - [Items Loop](#items-loop)
   - [Range Loop](#range-loop)
   - [Character Range Loop](#character-range-loop)
4. [Loop Variables](#loop-variables)
5. [Playbook Examples](#playbook-examples)
6. [Advanced Features](#advanced-features)
7. [Best Practices](#best-practices)
8. [Troubleshooting](#troubleshooting)
9. [API Reference](#api-reference)

---

## Introduction

Loops in Onigirazu allow you to execute the same task multiple times with different values, similar to Ansible's loop functionality. This is essential for:

- Installing multiple packages
- Creating multiple users or files
- Processing collections of items
- Repeating operations with different parameters

Onigirazu provides a powerful, flexible loop system with support for:

- **Item arrays** - Loop over a list of items
- **Numeric ranges** - Loop through numeric sequences with optional steps
- **Character ranges** - Loop through alphabetic characters (a-z, A-Z)

---

## Loop Basics

### Structure

Loops are defined in the task using the `loop` field:

```yaml
- name: "Task name"
  module: "module_name"
  args:
    param1: "{{ item }}"
  loop:
    items: [value1, value2, value3]
    # OR
    range: "1-10"
    # OR for custom variable name
    var: "custom_item"
```

### Loop Fields

```go
type Loop struct {
    Items    []interface{} `yaml:"items,omitempty"`  // Array of items to loop over
    Variable string        `yaml:"var,omitempty"`     // Custom variable name (default: "item")
    Index    string        `yaml:"index,omitempty"`   // Index variable name
    Range    string        `yaml:"range,omitempty"`   // Range specification
}
```

### Available Variables in Loop

- **`item`** (or custom name via `var:`) - The current item value
- **`loop.index`** - Current iteration index (1-based)
- **`loop.index0`** - Current iteration index (0-based)
- **`loop.first`** - Boolean, true if first iteration
- **`loop.last`** - Boolean, true if last iteration
- **`loop.length`** - Total number of items

---

## Loop Types

### Items Loop

The most common loop type - iterate over a list of items.

#### Simple List

```yaml
- name: "Install multiple packages"
  module: "package"
  args:
    name: "{{ item }}"
    state: "present"
  loop:
    items:
      - nginx
      - curl
      - vim
      - git
```

**Result**: Task executes 4 times, once for each package.

#### List of Objects

```yaml
- name: "Create users"
  module: "user"
  args:
    name: "{{ item.username }}"
    shell: "{{ item.shell }}"
    groups: "{{ item.groups }}"
  loop:
    items:
      - username: "alice"
        shell: "/bin/bash"
        groups: "sudo,developers"
      - username: "bob"
        shell: "/bin/bash"
        groups: "developers"
      - username: "charlie"
        shell: "/bin/sh"
        groups: "ops"
```

**Result**: Task executes 3 times, creating 3 users with different configurations.

#### List of Dictionaries

```yaml
- name: "Configure sysctl parameters"
  module: "sysctl"
  args:
    name: "{{ item.key }}"
    value: "{{ item.value }}"
    state: "present"
  loop:
    items:
      - key: "net.core.somaxconn"
        value: "65536"
      - key: "net.ipv4.tcp_max_syn_backlog"
        value: "65536"
      - key: "fs.file-max"
        value: "2097152"
```

---

### Range Loop

Loop through a numeric range with optional step.

#### Basic Numeric Range

```yaml
- name: "Create numbered directories"
  module: "file"
  args:
    path: "/tmp/dir-{{ item }}"
    state: "directory"
  loop:
    range: "1-10"
```

**Result**: Creates directories `/tmp/dir-1` through `/tmp/dir-10`.

**Generated items**: `[1, 2, 3, 4, 5, 6, 7, 8, 9, 10]`

#### Range with Step

```yaml
- name: "Process every other number"
  module: "command"
  args:
    cmd: "echo Number {{ item }}"
  loop:
    range: "1-20:2"
```

**Generated items**: `[1, 3, 5, 7, 9, 11, 13, 15, 17, 19]`

#### Zero-Based Range

```yaml
- name: "Create with zero-based numbering"
  module: "file"
  args:
    path: "/tmp/item-{{ item }}"
    state: "touch"
  loop:
    range: "0-5"
```

**Generated items**: `[0, 1, 2, 3, 4, 5]`

#### Reverse Range

```yaml
- name: "Countdown from 10 to 1"
  module: "command"
  args:
    cmd: "echo Count {{ item }}"
  loop:
    range: "10-1"
```

**Generated items**: `[10, 9, 8, 7, 6, 5, 4, 3, 2, 1]`

#### Reverse Range with Step

```yaml
- name: "Countdown by 2"
  module: "command"
  args:
    cmd: "echo {{ item }}"
  loop:
    range: "20-0:2"
```

**Generated items**: `[20, 18, 16, 14, 12, 10, 8, 6, 4, 2, 0]`

---

### Character Range Loop

Loop through alphabetic character ranges.

#### Lowercase Letters

```yaml
- name: "Create files for each letter"
  module: "file"
  args:
    path: "/tmp/file-{{ item }}"
    state: "touch"
  loop:
    range: "a-z"
```

**Generated items**: `['a', 'b', 'c', ..., 'x', 'y', 'z']` (26 items)

#### Uppercase Letters

```yaml
- name: "Create uppercase files"
  module: "file"
  args:
    path: "/tmp/FILE-{{ item }}"
    state: "touch"
  loop:
    range: "A-Z"
```

**Generated items**: `['A', 'B', 'C', ..., 'X', 'Y', 'Z']` (26 items)

#### Character Range with Step

```yaml
- name: "Create files for every 3rd letter"
  module: "file"
  args:
    path: "/tmp/letter-{{ item }}"
    state: "touch"
  loop:
    range: "a-z:3"
```

**Generated items**: `['a', 'd', 'g', 'j', 'm', 'p', 's', 'v', 'y']`

#### Reverse Character Range

```yaml
- name: "Process letters in reverse"
  module: "command"
  args:
    cmd: "echo {{ item }}"
  loop:
    range: "z-a"
```

**Generated items**: `['z', 'y', 'x', ..., 'b', 'a']`

---

## Loop Variables

### Built-in Loop Variables

Within a loop, you have access to special variables:

#### `loop.index`

**Type**: Integer (1-based)
**Description**: Current iteration number starting from 1

```yaml
- name: "Print with 1-based index"
  module: "command"
  args:
    cmd: "echo Item #{{ loop.index }}: {{ item }}"
  loop:
    items: ["apple", "banana", "cherry"]
```

Output:

```
Item #1: apple
Item #2: banana
Item #3: cherry
```

#### `loop.index0`

**Type**: Integer (0-based)
**Description**: Current iteration number starting from 0

```yaml
- name: "Print with 0-based index"
  module: "command"
  args:
    cmd: "echo Position {{ loop.index0 }}: {{ item }}"
  loop:
    items: ["first", "second", "third"]
```

Output:

```
Position 0: first
Position 1: second
Position 2: third
```

#### `loop.first`

**Type**: Boolean
**Description**: True if this is the first iteration

```yaml
- name: "Special handling for first item"
  module: "command"
  args:
    cmd: "{% if loop.first %}echo 'Processing first item: {{ item }}'{% else %}echo '{{ item }}'{% endif %}"
  loop:
    items: ["primary", "secondary", "tertiary"]
```

#### `loop.last`

**Type**: Boolean
**Description**: True if this is the last iteration

```yaml
- name: "Special handling for last item"
  module: "command"
  args:
    cmd: "{% if loop.last %}echo 'Last: {{ item }}'{% else %}echo '{{ item }},'{% endif %}"
  loop:
    items: ["a", "b", "c", "d"]
```

#### `loop.length`

**Type**: Integer
**Description**: Total number of items in the loop

```yaml
- name: "Show progress"
  module: "command"
  args:
    cmd: "echo Processing {{ loop.index }}/{{ loop.length }}: {{ item }}"
  loop:
    items: ["task1", "task2", "task3", "task4", "task5"]
```

### Custom Loop Variable Name

Use the `var:` field to create a custom variable name instead of `item`:

```yaml
- name: "Install packages with custom variable"
  module: "package"
  args:
    name: "{{ package_name }}"
    state: "present"
  loop:
    items: ["nginx", "postgresql", "redis"]
    var: "package_name"
```

```yaml
- name: "Create users with custom variable"
  module: "user"
  args:
    name: "{{ user.username }}"
    shell: "{{ user.shell }}"
  loop:
    items:
      - username: "alice"
        shell: "/bin/bash"
      - username: "bob"
        shell: "/bin/sh"
    var: "user"
```

---

## Playbook Examples

### Example 1: Multi-Package Installation

```yaml
name: "Install Web Server Stack"
plays:
  - name: "Setup web servers"
    hosts: "web_servers"
    tasks:
      - name: "Install web server packages"
        module: "package"
        args:
          name: "{{ item }}"
          state: "present"
        loop:
          items:
            - nginx
            - ssl-cert
            - curl
```

### Example 2: User Management

```yaml
name: "Create system users"
plays:
  - name: "Add users"
    hosts: "all"
    tasks:
      - name: "Create users with different shells"
        module: "user"
        args:
          name: "{{ item.name }}"
          shell: "{{ item.shell }}"
          createhome: true
        loop:
          items:
            - name: "developer1"
              shell: "/bin/bash"
            - name: "developer2"
              shell: "/bin/bash"
            - name: "sysadmin1"
              shell: "/bin/zsh"
            - name: "monitor"
              shell: "/bin/false"
```

### Example 3: Directory Structure

```yaml
name: "Create directory hierarchy"
plays:
  - name: "Setup app directories"
    hosts: "app_servers"
    tasks:
      - name: "Create application directories"
        module: "file"
        args:
          path: "/opt/app/{{ item }}"
          state: "directory"
          owner: "appuser"
          group: "appgroup"
          mode: "0755"
        loop:
          items:
            - "bin"
            - "lib"
            - "conf"
            - "data"
            - "logs"
            - "tmp"
```

### Example 4: Configuration with Ranges

```yaml
name: "Setup port forwarding"
plays:
  - name: "Configure ports"
    hosts: "routers"
    tasks:
      - name: "Configure NAT rules for port range"
        module: "iptables"
        args:
          table: "nat"
          chain: "PREROUTING"
          in_interface: "eth0"
          destination_port: "{{ item }}"
          jump: "DNAT"
          to_destination: "192.168.1.100:{{ item }}"
          comment: "Port forwarding for {{ item }}"
        loop:
          range: "8000-8010"
```

### Example 5: Letter-Based Tasks

```yaml
name: "Create volume groups"
plays:
  - name: "LVM setup"
    hosts: "storage"
    tasks:
      - name: "Create mount points from a-e"
        module: "file"
        args:
          path: "/mnt/vol{{ item }}"
          state: "directory"
          owner: "root"
          group: "root"
        loop:
          range: "a-e"
```

---

## Advanced Features

### Combining Loops with Conditions

Use `when:` clause with loop variables:

```yaml
- name: "Install packages selectively"
  module: "package"
  args:
    name: "{{ item.package }}"
    state: "present"
  loop:
    items:
      - package: "nginx"
        install: true
      - package: "apache2"
        install: false
      - package: "haproxy"
        install: true
  when: item.install
```

**Result**: Only nginx and haproxy are installed.

### Looping with Handlers

Trigger handlers multiple times:

```yaml
- name: "Restart services"
  module: "systemd"
  args:
    name: "{{ item }}"
    state: "restarted"
  loop:
    items:
      - "nginx"
      - "postgresql"
      - "redis-server"
  notify: "Log restart"

handlers:
  - name: "Log restart"
    module: "command"
    args:
      cmd: "echo Service restarted at $(date)"
```

### Nested Variables

Access nested object properties:

```yaml
- name: "Create firewall rules"
  module: "firewall_rule"
  args:
    port: "{{ item.port }}"
    protocol: "{{ item.protocol }}"
    state: "{{ item.state }}"
    comment: "{{ item.description }}"
  loop:
    items:
      - port: 22
        protocol: "tcp"
        state: "present"
        description: "SSH access"
      - port: 443
        protocol: "tcp"
        state: "present"
        description: "HTTPS access"
      - port: 3306
        protocol: "tcp"
        state: "absent"
        description: "MySQL blocked"
```

### Looping with Registered Variables and Find Module

**NEW in v1.51.0** - Loop over results from the `find:` module for advanced file operations:

```yaml
- name: "Find and process log files"
  hosts: "servers"
  tasks:
    - name: "Discover all log files"
      find:
        path: "/var/log"
        pattern: "*.log"
        type: "file"
        limit: 100
      register: "log_files"

    - name: "Archive each log file"
      command:
        cmd: "gzip -9 {{ item.path }}"
      loop: "{{ log_files.files }}"
      ignore_errors: true

    - name: "Find temporary files for cleanup"
      find:
        path: "/tmp"
        pattern: "*.tmp"
        type: "file"
      register: "tmp_files"

    - name: "Remove temporary files"
      file:
        path: "{{ item.path }}"
        state: absent
      loop: "{{ tmp_files.files }}"
      when: "tmp_files.matched > 0"
```

**Key Points**:

- Use `register:` to capture `find:` module results
- Access the file list via `{{ variable_name.files }}`
- Each file object contains: `path`, `name`, `size`, `mode`, `mtime`, type flags
- Combine with `when:` to check if files were found before processing
- Use `ignore_errors: true` to continue if individual file operations fail

### Combining Indices and Values

Use both `loop.index` and `item`:

```yaml
- name: "Create tiered storage"
  module: "file"
  args:
    path: "/storage/tier{{ loop.index }}/{{ item }}"
    state: "directory"
  loop:
    items:
      - "cache"
      - "working"
      - "archive"
```

**Creates**:

```
/storage/tier1/cache
/storage/tier1/working
/storage/tier1/archive
/storage/tier2/cache
/storage/tier2/working
/storage/tier2/archive
/storage/tier3/cache
/storage/tier3/working
/storage/tier3/archive
```

---

## Best Practices

### 1. Use Meaningful Variable Names

Instead of:

```yaml
loop:
  items: ["nginx", "apache", "haproxy"]
```

Consider using a custom variable name:

```yaml
loop:
  items: ["nginx", "apache", "haproxy"]
  var: "web_server"
args:
  name: "{{ web_server }}"
```

### 2. Document Complex Loops

Add clear comments for complex loop structures:

```yaml
# Loop through users with multiple properties
# Creates users with specific shells and home directories
- name: "Create system users"
  module: "user"
  args:
    name: "{{ item.username }}"
    shell: "{{ item.shell }}"
  loop:
    items:
      - username: "app"
        shell: "/bin/false"
      - username: "backup"
        shell: "/bin/bash"
```

### 3. Use Range for Simple Sequences

For simple numeric or character sequences, use `range:`:

```yaml
# Good
loop:
  range: "1-100"

# Avoid (unnecessary verbosity)
loop:
  items: [1, 2, 3, 4, 5, ..., 100]
```

### 4. Validate Item Structure

For complex items, include validation:

```yaml
- name: "Process configuration items"
  module: "template"
  args:
    src: "config.j2"
    dest: "/etc/{{ item.service }}/config"
  loop:
    items:
      - service: "nginx"
        enabled: true
      - service: "postgresql"
        enabled: true
  # Implicit validation through template processing
```

### 5. Error Handling

Combine loops with error handling:

```yaml
- name: "Start services with error tolerance"
  module: "systemd"
  args:
    name: "{{ item }}"
    state: "started"
  loop:
    items:
      - "nginx"
      - "redis"
      - "postgresql"
  ignore_errors: true
```

---

## Troubleshooting

### Issue: "loop must specify either items or range"

**Cause**: Loop defined without `items:` or `range:`

**Solution**:

```yaml
# Wrong
loop:

# Correct
loop:
  items: ["value1", "value2"]
# OR
loop:
  range: "1-10"
```

### Issue: Template Variable Undefined

**Cause**: Using wrong variable name in template

**Solution**:

```yaml
# If var: is specified, use that name
loop:
  items: ["item1", "item2"]
  var: "custom_name"
args:
  cmd: "echo {{ custom_name }}"  # Correct

# If var: is not specified, use "item"
args:
  cmd: "echo {{ item }}"  # Correct
```

### Issue: Invalid Range Format

**Cause**: Range string doesn't match supported format

**Correct Formats**:

- Numeric: `"1-10"`, `"0-5:2"`, `"10-1"`
- Character: `"a-z"`, `"A-Z:3"`, `"z-a"`

**Wrong Formats** (will error):

- `"1..10"` - Use single dash
- `"1-10-20"` - Multiple dashes
- `"a-5"` - Mix of characters and numbers

### Issue: Step Value Errors

**Cause**: Invalid step specification

**Solution**:

```yaml
# Wrong - zero step
range: "1-10:0"

# Wrong - negative step (use reverse range instead)
range: "1-10:-2"

# Correct - positive step
range: "1-10:2"

# Correct - use reverse range
range: "10-1:2"  # Results in [10, 8, 6, 4, 2]
```

---

## API Reference

### Loop Type Structure

```go
type Loop struct {
    // Items is the list of items to iterate over
    Items []interface{} `yaml:"items,omitempty" json:"items,omitempty"`

    // Variable specifies the custom variable name (default: "item")
    Variable string `yaml:"var,omitempty" json:"var,omitempty"`

    // Index specifies the index variable name
    Index string `yaml:"index,omitempty" json:"index,omitempty"`

    // Range specifies a range to iterate over (e.g., "1-10" or "a-z")
    Range string `yaml:"range,omitempty" json:"range,omitempty"`
}
```

### Engine Methods

#### `getLoopItems(loop *Loop, variables map[string]interface{}) ([]interface{}, error)`

Retrieves all items for a loop.

**Parameters**:

- `loop`: Loop configuration
- `variables`: Available template variables

**Returns**:

- Slice of items to iterate over
- Error if loop is invalid

#### `parseRange(rangeStr string) ([]interface{}, error)`

Parses range string into items.

**Supported Formats**:

- Numeric: `"1-10"`, `"1-10:2"`
- Character: `"a-z"`, `"A-Z:3"`
- Reverse: `"10-1"`, `"z-a:2"`

**Returns**:

- Slice of items (integers for numeric, strings for character)
- Error if format is invalid

### Range Parsing Rules

| Format | Example | Result |
|--------|---------|--------|
| Numeric range | `"1-5"` | `[1, 2, 3, 4, 5]` |
| With step | `"1-10:2"` | `[1, 3, 5, 7, 9]` |
| Reverse numeric | `"10-1"` | `[10, 9, 8, 7, 6, 5, 4, 3, 2, 1]` |
| Reverse with step | `"10-1:2"` | `[10, 8, 6, 4, 2]` |
| Character range | `"a-z"` | `['a', 'b', ..., 'z']` |
| Character step | `"a-z:3"` | `['a', 'd', 'g', 'j', 'm', 'p', 's', 'v', 'y']` |
| Reverse character | `"z-a"` | `['z', 'y', ..., 'a']` |

---

## Related Documentation

- [Task Execution](PLAYBOOK_QUICK_REFERENCE.md)
- [Variables and Templates](VARIABLES_AND_CONFIGURATION.md)
- [Playbook Syntax](PLAYBOOK_FORMATS.md)
- [Handlers Guide](HANDLERS_GUIDE.md)
