# Onigirazu

**Current Release:** v1.57.0 — [System Control & Archive Module Release](https://github.com/onigirazu-cfg/onigirazu/releases/tag/v1.57.0)

Onigirazu is a modern, high-performance configuration management tool written in Go, inspired by Ansible. It provides a simple yet powerful way to automate infrastructure configuration, application deployment, and system administration tasks.

## ✨ Key Highlights

- **🚀 Fast**: Written in Go for exceptional performance
- **📝 Simple YAML Syntax**: Human-readable, clean playbook format
- **🤖 Agentless**: SSH-based communication, no agent installation required
- **⚡ Parallel Execution**: Concurrent task execution with configurable limits
- **🔄 Idempotent**: Safe to run multiple times
- **📊 State Management**: Track changes and system state with rollback support
- **🎯 5 Ad-hoc Input Formats**: Including unique natural language support
- **📚 44 Built-in Modules**: Complete automation capabilities with system control, archive, and more (v1.57.0+)
- **📖 Complete Documentation**: 2,500+ lines of guides for configuration, security, and testing
- **🖥️ Interactive Mode (v1.54.0+)**: Real-time TUI dashboard with live logs, mode switching, and graceful control
- **⚙️ Module Scaffolding Tool**: Rapidly generate new modules with boilerplate, tests, and best practices in seconds

## Features

### Core Features

- **YAML-based Playbooks**: Clean, human-readable configuration format
- **Agentless Architecture**: No agents needed on target hosts
- **SSH-based Communication**: Secure remote execution with host key validation
- **Parallel Execution**: Concurrent task execution with configurable concurrency
- **Idempotent Operations**: Safe to run multiple times without side effects
- **State Management**: Track changes and maintain system state
- **Template Engine**: Jinja2-like templating for dynamic configurations
- **Inline Host Inventory**: Specify hosts directly via `-i` flag without inventory files

### Advanced Features

- **Ad-hoc Commands**: Execute commands without playbooks (5 input formats including natural language)
- **Rollback Support**: Automatic snapshots and one-command rollback
- **Drift Detection**: Detect and automatically fix configuration drift
- **Enhanced Logging**: Structured logging with multiple output formats
- **Progress Tracking**: Real-time execution progress with visual indicators
- **Interactive Mode**: Beautiful TUI dashboard with live log streaming, multi-mode display switching, and graceful shutdown
- **Caching System**: Intelligent caching for improved performance (facts, templates, packages)
- **Retry Logic**: Configurable retry mechanisms with exponential backoff
- **Conditional Execution**: Skip tasks based on conditions
- **Loop Support**: Iterate over lists and ranges
- **Module System**: Extensible architecture with 44 built-in modules (sysctl, reboot, mount, archive, and 40 others)
- **Plugin System**: Extensible plugin architecture (modules, callbacks, filters, inventory)
- **Secrets Management**: Bitwarden integration for secure credential management
- **Security Policies**: File operation restrictions and access control on control machine
- **Flexible Inventory**: Multiple inventory formats (YAML, TOML, JSON, INI, text)
- **Variable Interpolation**: Dynamic variable substitution
- **Error Handling**: Comprehensive error handling and reporting
- **Audit & Analytics**: Track execution history, performance metrics, and audit data with `audit` command
- **Complete Configuration Reference**: 35+ configuration options fully documented
- **Security Policy Guide**: 13+ security options fully documented
- **Module Scaffolding Tool**: Generate new modules with complete boilerplate, unit tests, idempotency tests, and benchmarks in seconds

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

**Available for**: Linux (amd64, arm64, arm), macOS (amd64, arm64), Windows (amd64), FreeBSD, OpenBSD, NetBSD

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
git clone https://github.com/onigirazu-cfg/onigirazu.git
cd onigirazu
make build
make install  # optional
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
```

Or use inline inventory directly (no file needed):

```bash
onigirazu apply playbook.yml -i "ubuntu@web1,ubuntu@web2"
```

### 2. Create a Playbook

Create `playbook.yaml`:

```yaml
name: "Web Server Setup"
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

      - name: "Create web directory"
        file:
          path: /var/www/html
          state: directory
          mode: "0755"
```

### 3. Run the Playbook

```bash
onigirazu apply playbook.yaml -i inventory.yaml
```

📖 **For complete playbook format reference, see:**

- **[Playbook Format Guide](docs/examples/README.md)** - Real playbook examples and patterns
- **[Playbook Types Documentation](docs/api/pkg/types.md)** - Technical type definitions

## 🖥️ Interactive Mode (v1.54.0+)

Interactive Mode provides a beautiful, responsive terminal UI dashboard for real-time execution monitoring. Perfect for long-running playbooks, debugging, and interactive control.

### Enable Interactive Mode

Simply add the `--interactive` flag to any playbook execution:

```bash
onigirazu apply playbook.yaml -i inventory.yaml --interactive
```

### Features

- **📊 Live Log Dashboard**: Real-time log streaming with automatic scrolling
- **🔄 Multi-Mode Display**: Switch between NORMAL, VERBOSE, and DEBUG modes
- **📈 Execution Statistics**: Track progress with task counts and timing
- **⌨️ Keyboard Control**: Responsive interactive controls
- **🎯 Graceful Shutdown**: Stop execution cleanly with proper cleanup

### Keyboard Controls

| Key | Action |
|-----|--------|
| **V** | Toggle VERBOSE mode (show more details) |
| **D** | Toggle DEBUG mode (show debug information) |
| **N** | Switch to NORMAL mode (less verbose) |
| **S** | Show execution statistics |
| **H** | Display help overlay |
| **↑/↓** | Scroll logs up/down |
| **Page Up/Down** | Page scroll logs |
| **G** | Graceful stop (clean shutdown) |
| **Q** | Quit TUI (without stopping execution) |
| **Ctrl+C** | Force quit (emergency exit) |

### Example Workflow

```bash
# 1. Start your playbook with interactive mode
onigirazu apply deployment.yaml -i "ubuntu@server1" --interactive

# 2. Watch the live dashboard
# 3. Press V to see verbose logs if you want more detail
# 4. Press S to check progress statistics
# 5. Press G to gracefully stop if needed
# 6. Press Q to exit TUI when done
```

### Display Modes

| Mode | What You See |
|------|--------------|
| **NORMAL** | Standard task output and errors only |
| **VERBOSE** | Detailed output with variable values and timestamps |
| **DEBUG** | Complete debug information for troubleshooting |

All modes show real-time updates as tasks execute. The dashboard automatically updates at 30 FPS for smooth performance.

📖 **For detailed interactive mode guide, see [docs/INTERACTIVE_MODE.md](docs/INTERACTIVE_MODE.md)**

## Playbook Format

Onigirazu uses a clean YAML format for playbooks. Here's the structure:

```yaml
# Optional global variables
vars:
  app_name: "myapp"
  app_version: "1.0.0"

# List of plays (each targets specific hosts)
plays:
  - name: "Play description"
    hosts:
      - "group_name"
      - "single_host"

    # Play-level variables (override global vars)
    vars:
      local_var: "value"

    # Tasks to execute
    tasks:
      - name: "Task name"
        module_name:
          arg1: value1
          arg2: value2

    # Handlers (triggered by notify)
    handlers:
      - name: "Handler name"
        service:
          name: nginx
          state: restarted
```

**Key features:**

- ✅ Multiple plays targeting different host groups
- ✅ Hierarchical variable scoping
- ✅ Conditional execution with `when`
- ✅ Loops with `loop`
- ✅ Task notifications and handlers
- ✅ Error handling with `ignore_errors` and `retries`

📚 **For detailed examples, see [docs/examples/README.md](docs/examples/README.md)**

## CLI Commands

### All Available Commands

**Quick Reference:**

| Command | Purpose | Use Case |
|---------|---------|----------|
| `apply` | Execute playbooks | Run your automation playbooks |
| `run` | Ad-hoc commands | Quick one-off tasks without playbooks |
| `plan` | Preview changes | Dry-run before executing |
| `validate` | Check playbook syntax | Validate playbook structure |
| `lint` | Check best practices | Find errors and style issues |
| `fmt` | Format playbook files | Auto-format YAML |
| `graph` | Visualize playbook | Show playbook structure |
| `audit` | Audit & analytics | Track execution history |
| `diff` | Show differences | Compare playbook vs current state |
| `drift` | Detect & fix drift | Find and fix configuration drift |
| `healthcheck` | Check host health | Verify inventory hosts are reachable |
| `inventory` | Manage inventory | Query and manage hosts |
| `state` | Manage state | View, clear, or export state |
| `rollback` | Rollback changes | Revert to previous snapshots |
| `completion` | Shell completion | Generate autocompletion script |

### Core Commands

#### `apply` - Execute playbooks

```bash
# Run a playbook
onigirazu apply playbook.yml -i inventory.yml

# Check mode (dry-run)
onigirazu apply playbook.yml --check

# With tags (run only specific tasks)
onigirazu apply playbook.yml --tags=setup,config

# Discover available tags
onigirazu apply playbook.yml --list-tags

# Preview which tasks would run
onigirazu apply playbook.yml --list-tasks --tags=setup

# Verbose output
onigirazu apply playbook.yml -v
```

**Tag Discovery Options:**

- `--list-tags` - List all available tags in the playbook
- `--list-tasks` - Show which tasks would execute with current filters
- `--list-tasks --tags TAG1,TAG2` - Preview specific task tags
- `--list-tasks --skip-tags TAG1` - Preview with tag exclusion
- `--list-tasks --verbose` - Detailed task information

📚 See [Tag and Task Discovery Guide](docs/LIST_TAGS_TASKS_GUIDE.md) for detailed examples

#### `run` - Execute ad-hoc commands

```bash
# Simple ping
onigirazu run all -m ping -i inventory.yml

# Install package
onigirazu run webservers -m package name=nginx state=present

# Natural language command
onigirazu run all "install nginx" -i inventory.yml

# Run shell command
onigirazu run all -m shell "ps aux | grep nginx"
```

📚 See [Ad-hoc Commands Guide](docs/ADHOC_GUIDE.md) for all 5 input formats

#### `plan` - Preview changes

```bash
# Show what would change
onigirazu plan playbook.yml -i inventory.yml

# Verbose output
onigirazu plan playbook.yml --verbose
```

#### `validate` - Validate playbook syntax

```bash
# Validate a playbook
onigirazu validate playbook.yml

# Validate with inventory
onigirazu validate playbook.yml -i inventory.yml
```

#### `diff` - Show differences

```bash
# Show differences between playbook and current state
onigirazu diff playbook.yml -i inventory.yml

# Verbose output
onigirazu diff playbook.yml --verbose
```

#### `inventory` - Manage inventory

```bash
# List all hosts
onigirazu inventory list -i inventory.yml

# List specific group
onigirazu inventory list webservers -i inventory.yml

# Show host details
onigirazu inventory host web1 -i inventory.yml
```

#### `state` - Manage state

```bash
# Show current state
onigirazu state show

# Clear state
onigirazu state clear

# Export state
onigirazu state export > state-backup.json

# Import state
onigirazu state import < state-backup.json
```

### Development & Analysis Tools

#### `fmt` - Format playbook files

```bash
# Format a playbook
onigirazu fmt playbook.yml

# Check without modifying
onigirazu fmt --check playbook.yml

# Format directory recursively
onigirazu fmt --recursive playbooks/
```

#### `lint` - Check for errors and best practices

```bash
# Lint a playbook
onigirazu lint playbook.yml

# Strict mode (warnings as errors)
onigirazu lint --strict playbook.yml

# Lint directory
onigirazu lint --recursive playbooks/
```

#### `graph` - Visualize playbook structure

```bash
# Generate ASCII graph
onigirazu graph playbook.yml

# Show variables and handlers
onigirazu graph --show-vars --show-handlers playbook.yml

# Generate Mermaid diagram
onigirazu graph --format=mermaid playbook.yml
```

#### `audit` - Audit playbook executions

```bash
# View execution history
onigirazu audit list

# Show specific execution report
onigirazu audit show <execution_id>

# Export audit data
onigirazu audit export > audit-report.json
```

📚 See [Audit & Analytics Guide](../onigirazu_docs/features/AUDIT_AND_ANALYTICS_SYSTEM.md)

### Advanced Features

#### `drift` - Detect and fix configuration drift

```bash
# Detect drift
onigirazu drift detect playbook.yml -i inventory.yml

# Fix drift automatically
onigirazu drift fix playbook.yml -i inventory.yml
```

#### `rollback` - Rollback changes

```bash
# List available snapshots
onigirazu rollback list

# Rollback to specific snapshot
onigirazu rollback restore <snapshot_id>

# Rollback to previous execution
onigirazu rollback restore --previous
```

#### `healthcheck` - Check host health

```bash
# Check all hosts
onigirazu healthcheck all -i inventory.yml

# Check specific group
onigirazu healthcheck webservers -i inventory.yml
```

### Utility Commands

#### `completion` - Generate shell completion

```bash
# Generate bash completion
onigirazu completion bash > ~/.bash_completion.d/onigirazu

# Generate zsh completion
onigirazu completion zsh > ~/.zsh/completions/_onigirazu

# Generate fish completion
onigirazu completion fish > ~/.config/fish/completions/onigirazu.fish
```

#### `version` - Show version information

```bash
# Display version
onigirazu version
```

#### `help` - Get command help

```bash
# Show general help
onigirazu help

# Show help for specific command
onigirazu apply --help
onigirazu run --help
```

### Global Flags & Options

These flags work with all commands:

```bash
# Configuration and inventory
-c, --config FILE        Path to configuration file
-i, --inventory FILE     Path to inventory file or inline hosts (e.g., "ubuntu@host1,ubuntu@host2")
-s, --state FILE         Path to state file (default: ".onigirazu-state")

# Output control
-v, --verbose            Verbose output (show more details)
--show-debug            Show debug and info messages
--no-color              Disable colored output

# Help and version
-h, --help              Show help for any command
--version               Show version information
```

**Example usage:**

```bash
# Run with custom config and verbose output
onigirazu apply playbook.yml -c /etc/onigirazu/config.yaml -v

# Run with inline inventory and debug output
onigirazu run all -m ping -i "ubuntu@web1,ubuntu@web2" --show-debug

# Run without colors (useful for logs/CI)
onigirazu apply playbook.yml -i inventory.yml --no-color

# Use custom state file location
onigirazu apply playbook.yml -s /var/lib/onigirazu/state.json
```

## 🚀 Ad-hoc Commands

Execute commands without creating playbooks using 5 different input formats:

### Quick Examples

```bash
# Simple ping
onigirazu run all -m ping -i inventory.yml

# Install package (Ansible-like syntax)
onigirazu run all -m package name=nginx state=present -i inventory.yml

# Natural language (unique to Onigirazu!)
onigirazu run all "install nginx package" -i inventory.yml

# Inline host specification
onigirazu run all -m ping -i "ubuntu@web1,ubuntu@web2"
```

### 5 Input Formats

#### 1. **Ansible-like Syntax** (Familiar)

```bash
onigirazu run all -m package name=nginx state=present -i inventory.yml
onigirazu run webservers -m service name=nginx state=started -i inventory.yml
```

#### 2. **Natural Language** (Unique! 🌟)

```bash
onigirazu run all "install nginx package" -i inventory.yml
onigirazu run webservers "start nginx service" -i inventory.yml
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
```

#### 3. **Module:Args Format** (Compact)

```bash
onigirazu run all "package:name=nginx,state=present" -i inventory.yml
```

#### 4. **JSON Format** (Structured)

```bash
onigirazu run all '{"module":"package","args":{"name":"nginx","state":"present"}}' -i inventory.yml
```

#### 5. **YAML Format** (Readable)

```bash
onigirazu run all 'module: package
args:
  name: nginx
  state: present' -i inventory.yml
```

### Output Formats

```bash
# Text (default, colored, human-readable)
onigirazu run all -m ping -i inventory.yml

# JSON (for scripting)
onigirazu run all -m ping -i inventory.yml -o json

# YAML (structured)
onigirazu run all -m ping -i inventory.yml -o yaml

# Table (compact view)
onigirazu run all -m ping -i inventory.yml -o table
```

📚 **For detailed ad-hoc documentation, see [docs/ADHOC_GUIDE.md](docs/ADHOC_GUIDE.md)**

## Modules

Onigirazu includes 22+ built-in modules for common automation tasks:

### System Modules

- **file**: File and directory operations
- **package**: Package management
- **service**: Service management
- **user**: User management
- **group**: Group management
- **command**: Execute shell commands
- **shell**: Execute shell commands with shell features

### Configuration Modules

- **template**: Template file processing with Jinja2
- **git**: Git repository operations
- **systemd**: Systemd service and unit management
- **cron**: Cron job scheduling

### Network & Firewall

- **firewall**: Unified firewall management (UFW, firewalld, iptables)

### Container Modules

- **docker_container**: Docker container management
- **docker_image**: Docker image management
- **docker_compose**: Docker Compose application management
- **podman**: Podman container management

### Database Modules

- **mysql_db**: MySQL database management
- **mysql_user**: MySQL user management
- **postgresql_db**: PostgreSQL database management
- **postgresql_user**: PostgreSQL user management
- **mongodb**: MongoDB management

### Utility Modules

- **debug**: Display debug messages
- **set_fact**: Set variables

### Module Usage Example

```yaml
tasks:
  - name: "Install package"
    package:
      name: nginx
      state: present

  - name: "Start service"
    service:
      name: nginx
      state: started
      enabled: true

  - name: "Create directory"
    file:
      path: /opt/myapp
      state: directory
      mode: "0755"

  - name: "Generate config from template"
    template:
      src: app.conf.j2
      dest: /etc/myapp/app.conf
      backup: true
```

📚 **For complete modules reference, see [docs/modules/README.md](docs/modules/README.md)**

## Advanced Features

### Loops

Execute tasks multiple times:

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
- name: "Install Docker on Debian"
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
- name: "Generate configuration"
  template:
    src: app.conf.j2
    dest: /etc/myapp/app.conf
```

### Error Handling

Control error behavior:

```yaml
- name: "Task that might fail"
  command:
    cmd: some-command
  ignore_errors: true

- name: "Task with retries"
  command:
    cmd: flaky-command
  retries: 3
  retry_delay: 5
```

## 🔄 State Management & Rollback

Automatic snapshots and one-command rollback for safe infrastructure changes:

```bash
# Show current state
onigirazu state show

# List available snapshots
onigirazu rollback --list

# Preview rollback
onigirazu rollback --dry-run --snapshot <id>

# Perform rollback
onigirazu rollback --snapshot <id>

# Cleanup old snapshots
onigirazu rollback --cleanup --max-age 30d
```

## 🎯 Drift Detection

Detect and automatically fix configuration drift:

```bash
# Detect drift
onigirazu drift detect -p playbook.yml -i inventory.yml

# Show drift report
onigirazu drift report

# Fix detected drift
onigirazu drift fix -p playbook.yml -i inventory.yml
```

## Architecture

Onigirazu follows a clean, modular architecture:

- **Execution Engine**: Orchestrates playbook execution
- **Module Registry**: Manages available modules
- **Inventory Manager**: Handles host and group management
- **Template Engine**: Processes templates and variables
- **State Manager**: Tracks system state and changes
- **Cache Manager**: Provides intelligent caching
- **Logger**: Structured logging with multiple formats

All components implement well-defined interfaces for:

- ✅ Testability
- ✅ Extensibility
- ✅ Maintainability
- ✅ Configurability

## 📚 Documentation

### Configuration & Security (⭐ NEW in v1.52.0)

These comprehensive guides cover all configuration options and security policies:

- **[Complete Configuration Reference](docs/CONFIGURATION_REFERENCE.md)** - All 35+ configuration options with examples
- **[Security Policy Guide](docs/SECURITY_POLICY_GUIDE.md)** - All 13+ security policy options with real-world examples
- **[Quick Start Configuration](docs/QUICK_START_CONFIGURATION.md)** - Get configured in 5 minutes
- **[Configuration & Security Index](docs/INDEX_CONFIGURATION_SECURITY.md)** - Find answers by problem or role

### User Documentation

- **[Quick Start Guide](docs/QUICK_START_CONFIGURATION.md)** - Get started in 5 minutes
- **[Installation Guide](INSTALLATION.md)** - Detailed installation instructions
- **[Playbook Format & Examples](docs/examples/README.md)** - Real playbook examples and patterns
- **[Inventory Formats](docs/INVENTORY_FORMATS.md)** - Supported inventory formats
- **[Variables Reference](docs/VARIABLES_CHEATSHEET.md)** - Complete variables guide
- **[Modules Reference](docs/modules/README.md)** - All built-in modules
- **[Ad-hoc Commands Guide](docs/ADHOC_GUIDE.md)** - Ad-hoc command documentation
- **[Loop Guide](docs/LOOPS_GUIDE.md)** - Loop syntax and examples
- **[Handlers Guide](docs/HANDLERS_GUIDE.md)** - Task notifications and handlers
- **[Audit & Analytics Guide](../onigirazu_docs/features/AUDIT_AND_ANALYTICS_SYSTEM.md)** - Track execution history and analyze performance

### Developer Documentation

- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute
- **[Module Development Guide](docs/MODULE_DEVELOPMENT_GUIDE.md)** - Create custom modules
- **[Architecture Guide](docs/ARCHITECTURE_DIAGRAM.md)** - System architecture

## 🧪 Testing & Quality

Onigirazu maintains high code quality through comprehensive testing:

- **Test Coverage**: 65%+ overall, >80% for critical packages
- **Race Detection**: Automated race condition detection
- **CI/CD**: Continuous integration on every commit
- **Code Quality**: golangci-lint, staticcheck, and gofmt

### Running Tests

```bash
# Run all tests
make test

# Run tests with race detector
make test-race

# Run tests with coverage
make test-coverage

# Generate HTML coverage report
make coverage-html
```

## Development

### Building from Source

```bash
git clone https://github.com/onigirazu-cfg/onigirazu.git
cd onigirazu

# Install dependencies
go mod download

# Run tests
make test

# Build binary
make build

# Run with example
./bin/onigirazu apply examples/playbook.yaml
```

### Project Structure

```
onigirazu/
├── cmd/onigirazu/           # Main application
├── pkg/types/               # Core types and structures
├── internal/
│   ├── config/              # Configuration management
│   ├── logger/              # Enhanced logging
│   ├── cache/               # Caching system
│   ├── template/            # Template engine
│   ├── state/               # State management
│   ├── execution/           # Parallel execution
│   ├── progress/            # Progress tracking
│   ├── modules/             # Built-in modules
│   ├── parser/              # Playbook parsing (including inline inventory)
│   ├── inventory/           # Inventory management
│   ├── engine/              # Execution engine
│   └── interfaces/          # Interface definitions
├── examples/                # Example configurations
├── templates/               # Template examples
└── docs/                    # Documentation
```

## Contributing

We welcome contributions! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes and add tests
4. Run tests: `make test`
5. Run linting: `make lint`
6. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- 📖 **Documentation**: [Guide](docs/README.md) | [API Docs](docs/api/index.html)
- 🐛 **Issues**: [GitHub Issues](https://github.com/onigirazu-cfg/onigirazu/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/onigirazu-cfg/onigirazu/discussions)
- 📝 **Contributing**: [Contributing Guide](CONTRIBUTING.md)

---

**Onigirazu** - Modern configuration management made simple. Built with Go for performance. Inspired by Ansible for simplicity.
