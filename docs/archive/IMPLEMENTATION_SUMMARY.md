# Implementation Summary: Command-Line Variables & Environment Variables

## Overview

Implemented comprehensive variable management system for Onigirazu with proper priority hierarchy, allowing users to pass variables via command-line flags and environment variables.

## Features Implemented

### 1. Command-Line Extra Variables (`--extra-vars` / `-e`)

- Added `--extra-vars` flag to the `apply` command
- Supports multiple key=value pairs: `-e key1=value1 -e key2=value2`
- Variables have the highest priority and override all other sources

### 2. Environment Variables (`ONIGIRAZU_VAR_*`)

- Automatically loads environment variables with `ONIGIRAZU_VAR_` prefix
- Variable names are converted to lowercase for consistency
- Example: `ONIGIRAZU_VAR_version=1.2.3` becomes `version=1.2.3` in playbooks

### 3. Variable Priority System

Variables are applied in the following order (lowest to highest priority):

1. **Playbook variables** - defined in `vars:` section of playbook
2. **Environment variables** - `ONIGIRAZU_VAR_*` prefix
3. **Command-line extra variables** - `--extra-vars` flag (highest priority)

## Files Modified

### Core Implementation

1. **`onigirazu/internal/engine/execution_engine.go`**
   - Added `extraVars` field to store command-line variables separately
   - Implemented `SetExtraVars()` method to set command-line variables
   - Implemented `LoadEnvironmentVariables()` method to read environment variables
   - Implemented `applyExtraVars()` internal method to apply extra vars with highest priority
   - Modified `ExecutePlaybook()` to load variables in correct priority order
   - Added `GetVariables()` method for testing purposes

2. **`onigirazu/internal/cli/apply.go`**
   - Added `extraVars` flag variable (map[string]string)
   - Added `--extra-vars` / `-e` flag to command-line interface
   - Implemented logic to convert and apply extra variables before playbook execution
   - Updated command documentation with variable priority examples

3. **`onigirazu/internal/interfaces/interfaces.go`**
   - Added SSH-related methods to Config interface:
     - `GetSSHDefaultUser()`
     - `GetSSHDefaultPort()`
     - `GetSSHDefaultKeyFile()`

### Tests

4. **`onigirazu/internal/engine/execution_engine_test.go`**
   - Added `TestSetExtraVars()` - tests setting extra variables
   - Added `TestSetExtraVars_Nil()` - tests nil handling
   - Added `TestSetExtraVars_Overwrite()` - tests variable overwriting
   - Added `TestLoadEnvironmentVariables()` - tests environment variable loading
   - Added `TestLoadEnvironmentVariables_NoVars()` - tests with no env vars
   - Added `TestVariablePriority()` - tests complete priority chain
   - Added `TestVariableConcurrency()` - tests thread safety
   - Added mock methods for SSH config interface

### Documentation & Examples

5. **`examples/variables/playbook.yml`**
   - Example playbook demonstrating variable usage

6. **`examples/variables/inventory.yml`**
   - Simple inventory for testing

7. **`examples/variables/README.md`**
   - Comprehensive guide with usage examples
   - Demonstrates all variable priority scenarios

## Usage Examples

### Basic Usage

```bash
# Using command-line extra variables
onigirazu apply playbook.yml -i inventory.yml -e version=1.2.3 -e env=production

# Using environment variables
export ONIGIRAZU_VAR_version=1.2.3
export ONIGIRAZU_VAR_env=production
onigirazu apply playbook.yml -i inventory.yml

# Combining both (command-line takes precedence)
export ONIGIRAZU_VAR_version=1.0.0
onigirazu apply playbook.yml -i inventory.yml -e version=2.0.0  # version will be 2.0.0
```

## Technical Details

### Variable Storage

- **`variables`** - Main variable map containing all merged variables
- **`extraVars`** - Separate storage for command-line variables to maintain priority

### Priority Implementation

1. Playbook variables are loaded first via `setVariables()`
2. Environment variables are loaded via `LoadEnvironmentVariables()`
3. Extra variables are applied last via `applyExtraVars()` to ensure highest priority

### Thread Safety

- All variable operations are protected by `sync.RWMutex`
- Concurrent access is safe and tested

### Environment Variable Processing

- Only variables with `ONIGIRAZU_VAR_` prefix are loaded
- Prefix is stripped and name is converted to lowercase
- Example: `ONIGIRAZU_VAR_MY_VAR` → `my_var`

## Testing

All tests pass successfully:

- ✅ Unit tests for SetExtraVars
- ✅ Unit tests for LoadEnvironmentVariables
- ✅ Integration test for variable priority
- ✅ Concurrency test for thread safety
- ✅ All existing tests continue to pass

## Compatibility

- Backward compatible - existing playbooks work without changes
- No breaking changes to existing APIs
- Follows Ansible-like variable precedence model for familiarity

## Future Enhancements

Potential improvements for future iterations:

1. Support for JSON/YAML file input: `--extra-vars @vars.json`
2. Support for inline JSON: `--extra-vars '{"key": "value"}'`
3. Variable validation and type checking
4. Variable documentation in playbooks
5. Variable inheritance from parent playbooks

## Complete Variable Priority Chain

The full priority chain (from previous work + this implementation):

1. Built-in defaults (system user, auto-detected SSH key)
2. Global config file (~/.onigirazu/config.yml or /etc/onigirazu/config.yml)
3. Config environment variables (ONIGIRAZU_SSH_USER, etc.)
4. Inventory "all" group variables
5. Inventory specific group variables
6. Inventory host-specific variables
7. **Playbook variables** (vars: section in playbook) ← NEW
8. **Playbook environment variables** (ONIGIRAZU_VAR_*) ← NEW
9. **Command-line extra variables** (--extra-vars, highest priority) ← NEW
