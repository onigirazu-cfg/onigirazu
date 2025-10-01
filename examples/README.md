# Onigirazu Examples

This directory contains comprehensive examples demonstrating all available modules in Onigirazu using the modern nested syntax.

## Available Examples

### Basic Configuration

- `inventory.yml` - Basic inventory configuration
- `onigirazu.yml` - Configuration file with nested syntax enabled

### Module Examples

- `01-package-management.yml` - Package installation and management
- `02-file-operations.yml` - File creation, copying, and templating
- `03-service-management.yml` - Service control and management
- `04-user-group-management.yml` - User and group operations
- `05-command-execution.yml` - Command and shell execution
- `06-git-operations.yml` - Git repository management

### Complete Workflows

- `complete-server-setup.yml` - Full server configuration example
- `development-environment.yml` - Development environment setup
- `web-server-deployment.yml` - Web server deployment example

## Syntax

All examples use the modern nested module syntax:

```yaml
- name: "Task description"
  module:
    type: "module_name"
    parameter1: "value1"
    parameter2: "value2"
```

## Running Examples

### Local Testing

```bash
onigirazu --inventory inventory.yml --config onigirazu.yml --playbook quick-test.yml
```

### Remote Server (cs.rastiegaiev.com)

```bash
# Basic package management
onigirazu --inventory inventory-remote.yml --config onigirazu.yml --playbook 01-package-management.yml

# CS server specific setup
onigirazu --inventory inventory-remote.yml --config onigirazu.yml --playbook cs-server-setup.yml

# Complete server setup
onigirazu --inventory inventory-remote.yml --config onigirazu.yml --playbook complete-server-setup.yml
```

### Setup Steps

1. Navigate to examples directory: `cd /Users/denys.rastiegaiev/work/go_teransible/examples`
2. Configure SSH access to your server
3. Modify `inventory-remote.yml` for your hosts
4. Run the desired playbook

### File Locations

All files are in `/Users/denys.rastiegaiev/work/go_teransible/examples/`:

- `onigirazu.yml` - Configuration file with nested syntax enabled
- `inventory.yml` - Local inventory
- `inventory-remote.yml` - Remote server inventory
- `01-06-*.yml` - Module examples
- `complete-*.yml` - Full workflow examples

See `RUNNING_EXAMPLES.md` for detailed command examples and troubleshooting.

## Available Modules

- **package** - Package management (install, remove, update)
- **enhanced_package** - Advanced package management with better error handling
- **file** - File operations (create, modify, permissions)
- **copy** - Copy files between locations
- **template** - Template processing with variables
- **service** - Service management (start, stop, enable, disable)
- **command** - Execute commands
- **shell** - Execute shell commands with full shell features
- **user** - User management (create, modify, delete)
- **group** - Group management (create, modify, delete)
- **git** - Git repository operations (clone, pull, checkout)
