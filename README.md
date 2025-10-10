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

## CLI Commands

Onigirazu provides a modern CLI with subcommands for different operations:

### Core Commands

#### `apply` - Execute playbooks

```bash
# Run a playbook
onigirazu apply playbook.yml -i inventory.yml

# Check mode (dry-run)
onigirazu apply playbook.yml --check

# With tags
onigirazu apply playbook.yml --tags=setup,config
```

#### `validate` - Validate playbook syntax

```bash
# Validate a single playbook
onigirazu validate playbook.yml

# Validate with inventory
onigirazu validate playbook.yml -i inventory.yml
```

#### `plan` - Preview changes without execution

```bash
# Show what would change
onigirazu plan playbook.yml -i inventory.yml

# Detailed output
onigirazu plan playbook.yml --verbose
```

### Development Tools

#### `diff` - Compare playbook versions

```bash
# Compare two playbooks
onigirazu diff old-playbook.yml new-playbook.yml

# Show only changes
onigirazu diff --changes-only playbook-v1.yml playbook-v2.yml

# Unified diff format
onigirazu diff --format=unified old.yml new.yml
```

#### `fmt` - Format playbook files

```bash
# Format a playbook (in-place)
onigirazu fmt playbook.yml

# Check formatting without changes
onigirazu fmt --check playbook.yml

# Show diff of changes
onigirazu fmt --diff playbook.yml

# Format all YAML files recursively
onigirazu fmt --recursive playbooks/

# Custom indentation
onigirazu fmt --indent 4 playbook.yml
```

#### `lint` - Check for errors and best practices

```bash
# Lint a playbook
onigirazu lint playbook.yml

# Strict mode (warnings as errors)
onigirazu lint --strict playbook.yml

# Only show errors
onigirazu lint --no-warnings playbook.yml

# Run specific rules
onigirazu lint --rules=syntax,security playbook.yml

# Skip specific rules
onigirazu lint --skip-rules=task-name playbook.yml

# Lint all files in directory
onigirazu lint --recursive playbooks/
```

**Linting Rules:**

- `syntax` - YAML syntax validation
- `required-fields` - Check for required fields
- `task-name` - Ensure tasks have names
- `module-args` - Validate module arguments
- `deprecated` - Detect deprecated features
- `security` - Security best practices
- `best-practices` - General best practices

#### `graph` - Visualize playbook structure

```bash
# Generate ASCII graph
onigirazu graph playbook.yml

# Show variables and handlers
onigirazu graph --show-vars --show-handlers playbook.yml

# Compact view
onigirazu graph --compact playbook.yml

# Generate GraphViz DOT format
onigirazu graph --format=dot playbook.yml > graph.dot
dot -Tpng graph.dot -o graph.png

# Generate Mermaid diagram
onigirazu graph --format=mermaid playbook.yml
```

**Output Formats:**

- `ascii` - Simple ASCII art (default)
- `dot` - GraphViz DOT format
- `mermaid` - Mermaid diagram format

### State Management

#### `state` - Manage execution state

```bash
# Show current state
onigirazu state show

# Clear state
onigirazu state clear

# Export state
onigirazu state export > state-backup.json
```

### Global Flags

```bash
  -c, --config string      Path to configuration file
  -i, --inventory string   Path to inventory file
  -s, --state string       Path to state file (default ".onigirazu-state")
  -v, --verbose            Verbose output
      --no-color           Disable colored output
```

### Legacy Mode

For backward compatibility, the old syntax is still supported:

```bash
onigirazu -playbook playbook.yml -inventory inventory.yml
```

**Note:** Legacy mode will show a deprecation warning. Use `onigirazu apply` instead.

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
  when: "{{ onigirazu_os_family == 'Debian' }}"
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

### User Documentation

- **[Quick Start Guide](docs/quick-start.md)** - Get started with Onigirazu in minutes
- **[Variables Cheat Sheet](docs/VARIABLES_CHEATSHEET.md)** - Quick reference for common variables ⚡
- **[Variables and Configuration Reference](docs/VARIABLES_AND_CONFIGURATION.md)** - Complete guide to all configuration parameters and system variables
- **[Inventory Formats](docs/inventory-formats.md)** - Supported inventory file formats (YAML, TOML, simple list)
- **[Modules Reference](docs/modules/README.md)** - Built-in modules documentation

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

## 🧪 Testing & Quality

Onigirazu maintains high code quality through comprehensive testing and continuous integration.

### Test Coverage

- **Overall Coverage:** 36.5%
- **Critical Packages:** >80% coverage
- **Race Conditions:** Zero detected
- **CI/CD:** Automated testing on every commit

#### Coverage by Category

| Category | Packages | Coverage |
|----------|----------|----------|
| ✅ Excellent (>80%) | 5 packages | 94.4% - 85.3% |
| ✅ Good (60-80%) | 4 packages | 77.0% - 64.3% |
| ⚠️ Medium (40-60%) | 3 packages | 59.0% - 42.1% |
| ⚠️ Low (<40%) | 6 packages | 33.3% - 10.9% |
| ❌ No Tests | 11 packages | 0.0% |

**Critical packages with excellent coverage:**

- `internal/workflow` - 89.8%
- `internal/execution` - 87.8%
- `internal/inventory` - 85.3%
- `internal/cache` - 94.2%
- `internal/bufferpool` - 94.4%

📊 **For detailed coverage report, see [TEST_COVERAGE_REPORT.md](TEST_COVERAGE_REPORT.md)**

### Running Tests

```bash
# Run all tests
make test

# Run tests with race detector
make test-race

# Run tests with coverage
make test-coverage

# Generate HTML coverage report
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out -o coverage.html
```

### Quality Checks

```bash
# Run all quality checks
make lint

# Run specific checks
make vet          # go vet
make staticcheck  # staticcheck
make golangci     # golangci-lint
```

### CI/CD Pipeline

Every commit is automatically tested with:

- ✅ Unit tests with race detector
- ✅ Integration tests
- ✅ Code linting (golangci-lint)
- ✅ Static analysis (staticcheck)
- ✅ Code formatting (gofmt)
- ✅ Coverage reporting

## Documentation

### User Documentation

- **[Quick Start Guide](docs/quick-start.md)** - Get started in 5 minutes
- **[Installation Guide](INSTALLATION.md)** - Detailed installation instructions
- **[Examples](docs/examples/README.md)** - Real-world usage examples
- **[API Documentation](docs/api/index.html)** - Complete API reference

### Development Documentation

All development-related documentation (optimization reports, release notes, implementation guides, etc.) is located in the **[onigirazu_docs](../onigirazu_docs/)** directory:

- **[Development Documentation Index](../onigirazu_docs/README.md)** - Complete index of all development docs
- **[Project Status](../onigirazu_docs/PROJECT_STATUS.md)** - Current project status
- **[Optimization Analysis](../onigirazu_docs/OPTIMIZATION_AND_FEATURES_ANALYSIS.md)** - Performance optimization details
- **[Release Guide](../onigirazu_docs/RELEASE_GUIDE.md)** - How to create releases

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run tests and linting
6. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

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
