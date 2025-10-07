# Changelog

## [Unreleased]

## [1.9.0] - 2025-01-16

### ✨ Added

- **Simplified YAML Syntax**: New Ansible-like syntax for task definitions
  - Module name is now used directly as the field name (e.g., `package:`, `service:`, `file:`)
  - Removed redundant `module:` and `type:` wrappers
  - Cleaner, more intuitive syntax with one less nesting level
  - Example: `package: { name: git, state: present }` instead of `module: { type: package, name: git, state: present }`
  - Full backward compatibility maintained - old syntax still works

### 🔧 Changed

- Updated task parser to support both old and new syntax formats
- Modified `MarshalYAML` to output tasks using the new simplified syntax
- Updated all example playbooks (25 files) to use the new syntax
- Updated documentation (README.md, docs/IMPROVED_SYNTAX.md) with new syntax examples

### 📖 Documentation

- Completely revised `docs/IMPROVED_SYNTAX.md` with new syntax examples
- Updated README.md Quick Start guide with simplified syntax
- All 25 example YAML files migrated to new syntax

### 🔄 Backward Compatibility

- **100% backward compatible**: All existing playbooks with old syntax continue to work
- Parser automatically detects and handles both syntax formats
- No breaking changes introduced

## [1.8.0] - 2025-01-15

### ✨ Added

- **New Systemd Module**: Comprehensive systemd management
  - Service management (start, stop, restart, reload, enable, disable, mask)
  - Unit file creation and management
  - Systemd timer management
  - Daemon reload functionality
  - Service status checking
- **New Cron Module**: Complete cron job management
  - Individual cron job management in user crontabs
  - Full crontab file management with backup support
  - System cron management (cron.d, cron.daily, cron.hourly, cron.weekly, cron.monthly)
  - Cron job listing and inspection
  - Support for special time strings (@reboot, @daily, etc.)
- **New Firewall Module**: Unified firewall management with automatic detection
  - Support for UFW (Ubuntu/Debian)
  - Support for firewalld (RHEL/CentOS/Fedora)
  - Support for iptables (fallback)
  - Port-based rules (allow/deny)
  - Service-based rules (UFW and firewalld)
  - Source-based rules (IP addresses and subnets)
  - Firewall enable/disable
  - Rule listing and reload

### 📖 Documentation

- Created comprehensive module documentation: `docs/MODULES_SYSTEMD_CRON_FIREWALL.md`
- Added example playbooks:
  - `examples/10-systemd-management.yml` - Systemd module examples
  - `examples/11-cron-management.yml` - Cron module examples
  - `examples/12-firewall-management.yml` - Firewall module examples

## [1.7.2] - 2025-01-14

### 🐛 Fixed

- Docker image publishing workflow for releases

## [1.7.1] - 2025-10-07

### ✨ Added

- **Multi-Format Inventory Support**: Support for three inventory file formats
  - **YAML format**: Traditional Ansible-style inventory (existing)
  - **TOML format**: Modern structured configuration format with full feature parity
  - **Simple list format**: Plain text files with one host per line (IP, user@host, user@host:port)
- **Automatic Inventory Detection**: Auto-discovery of inventory files in playbook directory
  - Searches for common inventory file names: `inventory.yml`, `inventory.yaml`, `inventory.toml`, `hosts`, etc.
  - Intelligent format detection based on file extension and content analysis
  - Graceful fallback to localhost-only when no inventory found
- **Smart Format Detection**: Automatic format recognition with fallback chain
  - Extension-based detection (`.yml`, `.yaml`, `.toml`, `.txt`)
  - Content-based heuristics for ambiguous cases
  - Priority: YAML → TOML → Simple list
- **Simple List Parser**: Flexible host address parsing
  - Plain IP addresses: `192.168.1.10`
  - IP with custom port: `192.168.1.10:2222`
  - With username: `user@192.168.1.10`
  - Full format: `user@192.168.1.10:2222`
  - Automatic defaults (port 22, user "root")

### 🔧 Changed

- Enhanced `EnhancedParser` with integrated inventory parsing capabilities
- Updated `main.go` to implement auto-detection when `-inventory` flag is not provided
- Added `github.com/pelletier/go-toml/v2` dependency for TOML support

### 🐛 Fixed

- Fixed golangci-lint configuration compatibility (removed unsupported `version` field)
- Fixed Docker multi-architecture build support (simplified to use pre-built binaries)
- Fixed `go.mod` tidiness (moved go-toml to direct dependencies)
- Fixed release process (disabled GPG signing)

### 📖 Documentation

- Created comprehensive `docs/inventory-formats.md` guide
  - Format specifications and examples
  - Migration guide from YAML to TOML
  - Best practices and recommendations
  - Auto-detection behavior explanation
- Added example files:
  - `inventory.example.toml` - TOML format example
  - `inventory.example.txt` - Simple list format example

### 🔄 Backward Compatibility

- **100% backward compatible**: All existing YAML inventory files work unchanged
- Optional feature: Auto-detection only activates when `-inventory` flag is omitted
- Existing `-inventory` flag behavior preserved

## [1.7.0] - 2025-10-07

**Note**: This release was superseded by v1.7.1 due to build configuration issues.

### ✨ Added

- **Multi-Format Inventory Support**: Support for three inventory file formats
  - **YAML format**: Traditional Ansible-style inventory (existing)
  - **TOML format**: Modern structured configuration format with full feature parity
  - **Simple list format**: Plain text files with one host per line (IP, user@host, user@host:port)
- **Automatic Inventory Detection**: Auto-discovery of inventory files in playbook directory
  - Searches for common inventory file names: `inventory.yml`, `inventory.yaml`, `inventory.toml`, `hosts`, etc.
  - Intelligent format detection based on file extension and content analysis
  - Graceful fallback to localhost-only when no inventory found
- **Smart Format Detection**: Automatic format recognition with fallback chain
  - Extension-based detection (`.yml`, `.yaml`, `.toml`, `.txt`)
  - Content-based heuristics for ambiguous cases
  - Priority: YAML → TOML → Simple list
- **Simple List Parser**: Flexible host address parsing
  - Plain IP addresses: `192.168.1.10`
  - IP with custom port: `192.168.1.10:2222`
  - With username: `user@192.168.1.10`
  - Full format: `user@192.168.1.10:2222`
  - Automatic defaults (port 22, user "root")

### 🔧 Changed

- Enhanced `EnhancedParser` with integrated inventory parsing capabilities
- Updated `main.go` to implement auto-detection when `-inventory` flag is not provided
- Added `github.com/pelletier/go-toml/v2` dependency for TOML support

### 📖 Documentation

- Created comprehensive `docs/inventory-formats.md` guide
  - Format specifications and examples
  - Migration guide from YAML to TOML
  - Best practices and recommendations
  - Auto-detection behavior explanation
- Added example files:
  - `inventory.example.toml` - TOML format example
  - `inventory.example.txt` - Simple list format example

### 🔄 Backward Compatibility

- **100% backward compatible**: All existing YAML inventory files work unchanged
- Optional feature: Auto-detection only activates when `-inventory` flag is omitted
- Existing `-inventory` flag behavior preserved

### Example Usage

**Auto-detection:**

```bash
onigirazu playbook.yml  # Automatically finds inventory.yml in same directory
```

**TOML inventory:**

```toml
[hosts.web1]
address = "192.168.1.10"
port = 22
user = "deploy"

[groups.webservers]
hosts = ["web1", "web2"]
```

**Simple list inventory:**

```
192.168.1.10
deploy@192.168.1.11:2222
user@192.168.1.12
```

### Benefits

- **Flexibility**: Choose the format that best fits your workflow
- **Simplicity**: Simple lists for basic inventories, TOML/YAML for complex ones
- **Convenience**: No need to specify inventory path for standard layouts
- **Modern**: TOML support aligns with modern configuration management trends

## [1.6.0] - Previous Release

### ✨ Added

- **Debug Module**: New `debug` module for printing messages and variable values during playbook execution
  - Support for simple text messages
  - Support for multi-line messages
  - Variable interpolation in messages
  - Automatic output formatting
- **Streamlined YAML Syntax**: Eliminated verbose `args:` blocks from task definitions
- **Unquoted String Support**: Simple string values can now be written without quotes
- **Custom YAML Unmarshaler**: Intelligent parsing that distinguishes between task control fields and module arguments
- **Mixed Syntax Support**: Ability to use both old and new syntax in the same playbook
- **Reserved Fields Management**: Comprehensive list of 19 reserved task control fields

### 🔧 Changed

- Task YAML parsing now supports direct module arguments at task level
- Updated all documentation examples to use new compact syntax
- Enhanced README with syntax comparison and migration guide

### 📖 Documentation

- Added comprehensive syntax guide in `docs/IMPROVED_SYNTAX.md`
- Created example playbooks demonstrating new syntax capabilities
- Updated README with before/after syntax examples

### 🧪 Testing

- Added comprehensive test suite for new YAML parsing logic
- Verified backward compatibility with existing playbooks
- Tested complex data types and edge cases

### 🔄 Backward Compatibility

- **100% backward compatible**: All existing playbooks continue to work unchanged
- Old `args:` syntax remains fully supported
- Gradual migration path available

### Example Transformation

**Before:**

```yaml
- name: "Install package"
  module: "package"
  args:
    name: "nginx"
    state: "present"
```

**After:**

```yaml
- name: "Install package"
  module: "package"
  name: nginx
  state: present
```

### Benefits

- **Reduced verbosity**: Less typing required for task definitions
- **Improved readability**: Cleaner, more intuitive YAML structure
- **Better maintainability**: Easier to write and modify playbooks
- **Standards compliance**: Follows YAML best practices for unquoted strings
