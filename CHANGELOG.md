# Changelog

## [Unreleased]

## [1.7.0] - 2025-10-07

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
