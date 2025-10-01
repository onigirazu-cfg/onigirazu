# Changelog

## [Unreleased]

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
