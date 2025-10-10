# Ad-hoc Commands

The ad-hoc command system allows you to execute one-off tasks on remote hosts without creating a full playbook.

## Features

- **Multiple syntax formats**: Ansible-like, natural language, JSON, YAML, and module:args
- **Parallel execution**: Control concurrency with `--parallel` flag
- **Multiple output formats**: text, JSON, YAML, table
- **Check mode**: Dry-run without making changes
- **Pattern matching**: Target specific hosts or groups
- **Verbose mode**: Detailed execution information

## Usage

### Basic Syntax

```bash
onigirazu run [host-pattern] [options]
```

### Ansible-like Syntax (Recommended)

Execute modules with explicit module name and arguments:

```bash
# Ping all hosts
onigirazu run all -m ping -i inventory.yml

# Execute command
onigirazu run webservers -m command 'command="uptime"' -i inventory.yml

# Install package
onigirazu run all -m package name=nginx state=present -i inventory.yml

# Multiple arguments
onigirazu run all -m shell 'cmd="echo Hello"' chdir=/tmp -i inventory.yml
```

### Natural Language (Auto-detected)

Use natural language for common operations:

```bash
# Package management
onigirazu run all "install nginx package" -i inventory.yml
onigirazu run all "remove vim package" -i inventory.yml
onigirazu run all "update nginx package" -i inventory.yml

# Service management
onigirazu run webservers "start nginx service" -i inventory.yml
onigirazu run webservers "stop apache service" -i inventory.yml
onigirazu run webservers "restart nginx service" -i inventory.yml

# File operations
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
onigirazu run all "delete file /tmp/test.txt" -i inventory.yml
```

### Module:Args Syntax

Compact syntax for quick commands:

```bash
onigirazu run all "package:name=nginx,state=present" -i inventory.yml
onigirazu run all "service:name=nginx,state=started" -i inventory.yml
onigirazu run all "command:command=uptime" -i inventory.yml
```

### JSON Syntax

For programmatic usage:

```bash
onigirazu run all '{"module":"package","args":{"name":"nginx","state":"present"}}' -i inventory.yml
```

### YAML Syntax

For complex arguments:

```bash
onigirazu run all 'module: package
args:
  name: nginx
  state: present' -i inventory.yml
```

## Options

### Execution Options

- `--check`: Check mode (dry-run, don't make changes)
- `--diff`: Show differences when changing files
- `--timeout duration`: Execution timeout per host (default: 30s)
- `-f, --parallel int`: Number of parallel executions (default: 5)

### Output Options

- `-o, --output string`: Output format (text, json, yaml, table) (default: "text")
- `-V, --verbose-mode`: Verbose output (show detailed results)

### Variable Options

- `-e, --extra-vars key=value`: Extra variables to pass to modules

## Examples

### Basic Commands

```bash
# Ping all hosts
onigirazu run all -m ping -i inventory.yml

# Check disk space
onigirazu run all -m shell 'cmd="df -h"' -i inventory.yml

# Get system info
onigirazu run all -m command 'command="uname -a"' -i inventory.yml
```

### Package Management

```bash
# Install package
onigirazu run webservers -m package name=nginx state=present -i inventory.yml

# Remove package
onigirazu run webservers -m package name=apache2 state=absent -i inventory.yml

# Update package
onigirazu run all -m package name=vim state=latest -i inventory.yml
```

### Service Management

```bash
# Start service
onigirazu run webservers -m service name=nginx state=started -i inventory.yml

# Stop service
onigirazu run webservers -m service name=nginx state=stopped -i inventory.yml

# Restart service
onigirazu run webservers -m service name=nginx state=restarted -i inventory.yml
```

### File Operations

```bash
# Create file
onigirazu run all -m file path=/tmp/test.txt state=touch -i inventory.yml

# Delete file
onigirazu run all -m file path=/tmp/test.txt state=absent -i inventory.yml

# Create directory
onigirazu run all -m file path=/tmp/mydir state=directory -i inventory.yml
```

### Parallel Execution

```bash
# Execute on 10 hosts in parallel
onigirazu run all -m command 'command="uptime"' --parallel 10 -i inventory.yml

# Sequential execution (one at a time)
onigirazu run all -m command 'command="uptime"' --parallel 1 -i inventory.yml
```

### Output Formats

```bash
# JSON output
onigirazu run all -m ping --output json -i inventory.yml

# YAML output
onigirazu run all -m ping --output yaml -i inventory.yml

# Table output
onigirazu run all -m ping --output table -i inventory.yml
```

### Check Mode (Dry-run)

```bash
# Test without making changes
onigirazu run all -m package name=nginx state=present --check -i inventory.yml

# Show what would change
onigirazu run all -m file path=/tmp/test.txt state=absent --check -i inventory.yml
```

### Verbose Mode

```bash
# Show detailed execution information
onigirazu run all -m command 'command="uptime"' -V -i inventory.yml

# Combine with global verbose flag
onigirazu run all -m command 'command="uptime"' -v -V -i inventory.yml
```

## Host Patterns

Target specific hosts or groups:

```bash
# All hosts
onigirazu run all -m ping -i inventory.yml

# Specific group
onigirazu run webservers -m ping -i inventory.yml

# Specific host
onigirazu run server1 -m ping -i inventory.yml

# Multiple groups (comma-separated)
onigirazu run webservers,databases -m ping -i inventory.yml

# Pattern matching (if supported by inventory)
onigirazu run 'web*' -m ping -i inventory.yml
```

## Available Modules

Common modules for ad-hoc commands:

- **command**: Execute commands (no shell processing)
- **shell**: Execute shell commands (with shell processing)
- **ping**: Test connectivity
- **debug**: Print debug messages
- **package**: Manage packages
- **service**: Manage services
- **file**: Manage files and directories
- **copy**: Copy files to remote hosts
- **template**: Template files to remote hosts
- **set_fact**: Set host variables
- **user**: Manage users
- **group**: Manage groups
- **apt**: Manage apt packages (Debian/Ubuntu)
- **yum**: Manage yum packages (RHEL/CentOS)
- **systemd**: Manage systemd services
- **get_url**: Download files from URLs

## Architecture

The ad-hoc command system consists of:

1. **Parser** (`parser.go`): Parses multiple command formats
2. **Executor** (`executor.go`): Executes commands on hosts in parallel
3. **Formatter** (`formatter.go`): Formats output in various formats
4. **Types** (`types.go`): Core data structures
5. **CLI** (`cli/run.go`): Command-line interface

### Parser

The parser supports multiple formats and automatically detects the format:

1. Ansible-like: `-m module key=value`
2. Natural language: `"install nginx package"`
3. Module:args: `"module:key=value,key=value"`
4. JSON: `'{"module":"name","args":{...}}'`
5. YAML: `'module: name\nargs: {...}'`

### Executor

The executor:

- Resolves host patterns to actual hosts
- Executes modules in parallel (configurable concurrency)
- Collects results from all hosts
- Generates execution summary

### Formatter

The formatter supports:

- **text**: Human-readable output (default)
- **json**: JSON format for programmatic usage
- **yaml**: YAML format
- **table**: Tabular format (future)

## Error Handling

The system handles various error scenarios:

- **Module not found**: Returns error if module doesn't exist
- **Host not found**: Returns error if host pattern matches no hosts
- **Execution timeout**: Respects timeout per host
- **Connection errors**: Reports SSH/connection failures
- **Module errors**: Reports module execution failures

## Performance

- **Parallel execution**: Default 5 hosts in parallel
- **Configurable concurrency**: Use `--parallel` to adjust
- **Timeout control**: Use `--timeout` to set per-host timeout
- **Efficient**: Uses goroutines and semaphores for concurrency

## Comparison with Ansible

| Feature | Onigirazu | Ansible |
|---------|-----------|---------|
| Syntax | Multiple formats | Ansible-like only |
| Natural language | ✅ Yes | ❌ No |
| JSON/YAML input | ✅ Yes | ❌ No |
| Output formats | text, JSON, YAML, table | text, JSON |
| Parallel execution | ✅ Configurable | ✅ Configurable |
| Check mode | ✅ Yes | ✅ Yes |
| Module support | 18+ modules | 3000+ modules |

## Future Enhancements

- [ ] Pattern matching with wildcards and regex
- [ ] Limit execution to subset of hosts
- [ ] Fact gathering before execution
- [ ] Variable interpolation in commands
- [ ] Command history and replay
- [ ] Interactive mode
- [ ] Batch execution from file
- [ ] Result caching
- [ ] Retry on failure
- [ ] Conditional execution
