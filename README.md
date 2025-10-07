# Onigirazu

Onigirazu is a modern, high-performance configuration management tool written in Go, inspired by Ansible. It provides a simple yet powerful way to automate infrastructure configuration, application deployment, and system administration tasks.

## Features

### Core Features

- **YAML-based Playbooks**: Human-readable configuration files with improved syntax
- **Agentless Architecture**: No need to install agents on target hosts
- **SSH-based Communication**: Secure remote execution
- **Parallel Execution**: Concurrent task execution with configurable limits
- **Idempotent Operations**: Safe to run multiple times
- **State Management**: Track changes and maintain system state
- **Template Engine**: Jinja2-like templating for dynamic configurations
- **Flexible Syntax**: Multiple syntax options including nested module syntax for better organization

### Advanced Features

- **Enhanced Logging**: Structured logging with multiple output formats
- **Progress Tracking**: Real-time execution progress with visual indicators
- **Caching System**: Intelligent caching for improved performance
- **Retry Logic**: Configurable retry mechanisms with exponential backoff
- **Conditional Execution**: Skip tasks based on conditions
- **Loop Support**: Iterate over lists and ranges
- **Module System**: Extensible architecture with built-in modules
- **Inventory Management**: Flexible host and group management
- **Variable Interpolation**: Dynamic variable substitution
- **Error Handling**: Comprehensive error handling and reporting

## Installation

### 📦 Pre-built Binaries (Recommended)

Download the latest release for your platform:

```bash
# Linux x86_64
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# macOS (Intel)
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_x86_64.tar.gz
tar -xzf onigirazu_Darwin_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# macOS (Apple Silicon)
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_arm64.tar.gz
tar -xzf onigirazu_Darwin_arm64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

**Available for**: Linux (amd64, arm64, arm), macOS (amd64, arm64), Windows (amd64), FreeBSD (amd64)

### 🍺 Homebrew

```bash
brew tap onigirazu-cfg/tap
brew install onigirazu
```

### 🐳 Docker

```bash
# Docker Hub
docker run --rm onigirazu/onigirazu:latest --version

# GitHub Container Registry
docker run --rm ghcr.io/onigirazu-cfg/onigirazu:latest --version
```

### 📋 Package Managers

- **Debian/Ubuntu**: Download `.deb` from [releases](https://github.com/onigirazu-cfg/onigirazu/releases)
- **Red Hat/CentOS/Fedora**: Download `.rpm` from [releases](https://github.com/onigirazu-cfg/onigirazu/releases)
- **Alpine Linux**: Download `.apk` from [releases](https://github.com/onigirazu-cfg/onigirazu/releases)
- **Arch Linux**: Download `.pkg.tar.xz` from [releases](https://github.com/onigirazu-cfg/onigirazu/releases)

### 🔧 From Source

```bash
# Clone the repository
git clone https://github.com/onigirazu-cfg/onigirazu.git
cd onigirazu

# Build the binary
make build

# Install (optional)
make install
```

### Using Go

```bash
go install github.com/onigirazu-cfg/onigirazu/cmd/onigirazu@latest
```

📚 **For detailed installation instructions, see [INSTALLATION.md](INSTALLATION.md)**

## Quick Start

### 1. Create an Inventory File

Create `inventory.yaml`:

```yaml
groups:
  webservers:
    hosts:
      web1:
        address: "192.168.1.10"
        user: "ubuntu"
      web2:
        address: "192.168.1.11"
        user: "ubuntu"
    vars:
      http_port: 80
```

### 2. Create a Playbook

Create `playbook.yaml`:

```yaml
vars:
  app_name: "myapp"

plays:
  - name: "Install and configure web servers"
    hosts:
      - "webservers"

    tasks:
      - name: "Install Nginx"
        package:
          name: nginx
          state: present

      - name: "Start Nginx service"
        service:
          name: nginx
          state: started
          enabled: true
```

### 3. Run the Playbook

```bash
onigirazu -playbook playbook.yaml -inventory inventory.yaml
```

## ✨ Improved YAML Syntax

Onigirazu features a streamlined YAML syntax that eliminates verbose `args:` blocks and supports unquoted strings:

### Before (Old Syntax)

```yaml
- name: "List files in current directory"
  module:
    type: "command"
    cmd: "ls -la"
    shell: true
```

### After (New Syntax)

```yaml
- name: "List files in current directory"
  command:
    cmd: ls -la
    shell: true
```

### Key Benefits

- **Less typing**: No need for `args:` wrapper
- **Cleaner look**: Direct module arguments at task level
- **Unquoted strings**: Simple values don't need quotes
- **Backward compatible**: Old syntax still works
- **Mixed syntax**: Use both approaches in same playbook

📖 **For complete syntax guide and examples, see [docs/IMPROVED_SYNTAX.md](docs/IMPROVED_SYNTAX.md)**

## Configuration

Onigirazu can be configured using a YAML configuration file:

```yaml
# onigirazu.yaml
max_concurrency: 10
default_timeout: 300s
log_level: "info"
log_format: "text"
enable_caching: true
cache_ttl: 300s
```

Use the configuration file:

```bash
onigirazu -config onigirazu.yaml -playbook playbook.yaml -inventory inventory.yaml
```

## Command Line Options

```bash
Usage: onigirazu -playbook <path> [options]

Options:
  -playbook string
        Path to playbook file
  -inventory string
        Path to inventory file
  -config string
        Path to configuration file
  -verbose
        Verbose output (sets log level to debug)
  -check
        Check mode (dry-run)
  -state string
        State file for saving state (default ".onigirazu-state")
  -log-level string
        Log level (debug, info, warn, error) (default "info")
  -log-format string
        Log format (text, json) (default "text")
  -max-workers int
        Maximum number of worker threads (default 10)
  -timeout duration
        Execution timeout (default 30m0s)
```

## Modules

Onigirazu includes several built-in modules:

### Core Modules

- **file**: File and directory operations
- **package**: Package management
- **service**: Service management
- **command**: Execute commands
- **shell**: Execute shell commands
- **user**: User management
- **group**: Group management

### Extended Modules

- **template**: Template file processing
- **git**: Git repository operations

### System Management Modules

- **systemd**: Systemd service and unit management (service control, unit files, timers, daemon reload)
- **cron**: Cron job management (user crontabs, system cron, job scheduling)
- **firewall**: Unified firewall management with automatic detection (UFW, firewalld, iptables)

### Module Examples

#### File Module

```yaml
- name: "Create directory"
  file:
    path: /opt/myapp
    state: directory
    owner: myuser
    group: mygroup
    mode: "0755"
```

#### Package Module

```yaml
- name: "Install packages"
  package:
    name: nginx
    state: present
```

#### Service Module

```yaml
- name: "Start service"
  service:
    name: nginx
    state: started
    enabled: true
```

#### Template Module

```yaml
- name: "Generate configuration"
  template:
    src: templates/nginx.conf.j2
    dest: /etc/nginx/nginx.conf
    owner: root
    group: root
    mode: "0644"
    backup: true
```

#### Git Module

```yaml
- name: "Clone repository"
  git:
    repo: https://github.com/example/myapp.git
    dest: /opt/myapp
    version: main
    force: true
```

#### Systemd Module

```yaml
- name: "Manage systemd service"
  systemd:
    operation: service
    name: nginx
    state: started
    enabled: true
```

#### Cron Module

```yaml
- name: "Schedule backup job"
  cron:
    operation: job
    name: "Daily backup"
    job: "/usr/local/bin/backup.sh"
    hour: "2"
    minute: "0"
```

#### Firewall Module

```yaml
- name: "Allow HTTP traffic"
  firewall:
    operation: rule
    port: 80
    protocol: tcp
    action: allow
```

For detailed documentation on systemd, cron, and firewall modules, see [docs/MODULES_SYSTEMD_CRON_FIREWALL.md](docs/MODULES_SYSTEMD_CRON_FIREWALL.md).

## Advanced Features

### Loops

Execute tasks multiple times with different values:

```yaml
- name: "Install multiple packages"
  package:
    name: "{{ item }}"
    state: present
  loop:
    items:
      - curl
      - wget
      - git
```

### Conditionals

Skip tasks based on conditions:

```yaml
- name: "Install Docker"
  package:
    name: docker.io
    state: present
  when: "{{ ansible_facts.os_family == 'Debian' }}"
```

### Variables

Use variables for dynamic configurations:

```yaml
vars:
  app_name: "myapp"
  app_version: "1.0.0"

tasks:
  - name: "Create app directory"
    file:
      path: "/opt/{{ app_name }}"
      state: directory
```

### Templates

Use Jinja2-like templates for configuration files:

```yaml
# In playbook
- name: "Generate config"
  template:
    src: app.conf.j2
    dest: /etc/myapp/app.conf
```

```jinja2
# In templates/app.conf.j2
server_name = {{ app_name }}
version = {{ app_version }}
port = {{ http_port | default(8080) }}

{% if ssl_enabled %}
ssl_cert = /etc/ssl/certs/{{ app_name }}.crt
ssl_key = /etc/ssl/private/{{ app_name }}.key
{% endif %}
```

### Error Handling

Control error handling behavior:

```yaml
- name: "Optional task"
  command:
    cmd: some-command
  ignore_errors: true

- name: "Task with retries"
  command:
    cmd: flaky-command
  retries: 3
  retry_delay: 5
```

## Architecture

Onigirazu follows a modular, interface-driven architecture:

### Core Components

- **Execution Engine**: Orchestrates playbook execution
- **Module Registry**: Manages available modules
- **Inventory Manager**: Handles host and group management
- **Template Engine**: Processes templates and variables
- **State Manager**: Tracks system state and changes
- **Progress Tracker**: Monitors execution progress
- **Cache Manager**: Provides intelligent caching
- **Logger**: Structured logging with multiple formats

### Interfaces

All components implement well-defined interfaces, making the system:

- **Testable**: Easy to mock and unit test
- **Extensible**: Simple to add new modules and features
- **Maintainable**: Clear separation of concerns
- **Configurable**: Flexible dependency injection

## Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/onigirazu-cfg/onigirazu.git
cd onigirazu

# Install dependencies
go mod download

# Run tests
make test

# Build binary
make build

# Run with example
./bin/onigirazu -playbook examples/playbook.yaml -inventory examples/inventory.yaml
```

### Project Structure

```
onigirazu/
├── cmd/onigirazu/           # Main application
├── pkg/types/               # Core types and structures
├── internal/
│   ├── config/             # Configuration management
│   ├── logger/             # Enhanced logging
│   ├── cache/              # Caching system
│   ├── template/           # Template engine
│   ├── state/              # State management
│   ├── execution/          # Parallel execution
│   ├── progress/           # Progress tracking
│   ├── modules/            # Built-in modules
│   ├── parser/             # Playbook parsing
│   ├── inventory/          # Inventory management
│   ├── engine/             # Execution engine
│   └── interfaces/         # Interface definitions
├── examples/               # Example configurations
├── templates/              # Template examples
└── docs/                   # Documentation
```

### Adding New Modules

1. Implement the `Module` interface:

```go
type MyModule struct{}

func (m *MyModule) GetName() string {
    return "mymodule"
}

func (m *MyModule) Execute(host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Module implementation
    return types.TaskResult{
        Changed: true,
        Output:  "Task completed successfully",
    }, nil
}
```

2. Register the module:

```go
registry.Register(NewMyModule())
```

## 📚 Documentation

Onigirazu provides comprehensive documentation to help you get started and master the tool.

### API Documentation

- **[HTML Documentation](docs/api/index.html)** - Beautiful, interactive API documentation
- **[API Overview](docs/api/README.md)** - Documentation structure and guidelines
- **[Package Documentation](docs/api/)** - Detailed package-level documentation

### Quick Documentation Commands

```bash
# Generate API documentation
make docs-generate

# Open HTML documentation in browser
make docs-open

# Start interactive documentation server
make docs-serve

# View documentation info
make docs
```

### Available Documentation

| Package | Description |
|---------|-------------|
| `pkg/types` | Core types and interfaces |
| `pkg/utils` | Utility functions and helpers |
| `internal/config` | Configuration management |
| `internal/core` | Core execution engine |
| `internal/modules` | Built-in modules (command, file, template, etc.) |
| `internal/parser` | YAML parsing and validation |
| `internal/workflow` | Workflow orchestration |

### Documentation Features

- 🎨 **Beautiful HTML** - Modern, responsive design
- 🔍 **Searchable** - Easy navigation and package index
- 📱 **Mobile-friendly** - Works on all devices
- 🚀 **Auto-generated** - Always up-to-date with code
- 🌐 **Interactive server** - Browse with `pkgsite`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run tests and linting
6. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by Ansible's simplicity and power
- Built with Go's performance and concurrency in mind
- Designed for modern infrastructure automation needs

## Support

- **📖 Documentation**: [API Documentation](docs/api/index.html) | [Documentation Guide](docs/README.md)
- **🐛 Issues**: [GitHub Issues](https://github.com/onigirazu-cfg/onigirazu/issues)
- **💬 Discussions**: [GitHub Discussions](https://github.com/onigirazu-cfg/onigirazu/discussions)
- **📋 Contributing**: [Contributing Guide](CONTRIBUTING.md)

---

**Onigirazu** - Modern configuration management made simple.
