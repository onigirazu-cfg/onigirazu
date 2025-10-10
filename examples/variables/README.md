# Variable Priority Example

This example demonstrates how Onigirazu handles variable priority from different sources.

## Variable Priority (from lowest to highest)

1. **Playbook variables** (defined in `vars:` section)
2. **Environment variables** (with `ONIGIRAZU_VAR_` prefix)
3. **Command-line extra variables** (via `--extra-vars` or `-e` flag)

## Usage Examples

### 1. Using only playbook variables (default values)

```bash
onigirazu apply playbook.yml -i inventory.yml
```

Expected output:

- app_name: myapp
- version: 1.0.0
- environment: development
- debug_mode: false

### 2. Override with environment variables

```bash
export ONIGIRAZU_VAR_version=2.0.0
export ONIGIRAZU_VAR_environment=staging
onigirazu apply playbook.yml -i inventory.yml
```

Expected output:

- app_name: myapp
- version: 2.0.0 (from environment)
- environment: staging (from environment)
- debug_mode: false

### 3. Override with command-line extra variables (highest priority)

```bash
onigirazu apply playbook.yml -i inventory.yml -e version=3.0.0 -e environment=production -e debug_mode=true
```

Expected output:

- app_name: myapp
- version: 3.0.0 (from command line)
- environment: production (from command line)
- debug_mode: true (from command line)

### 4. Combining all sources (command-line wins)

```bash
export ONIGIRAZU_VAR_version=2.0.0
export ONIGIRAZU_VAR_environment=staging
onigirazu apply playbook.yml -i inventory.yml -e version=3.0.0 -e debug_mode=true
```

Expected output:

- app_name: myapp (from playbook)
- version: 3.0.0 (from command line, overrides environment)
- environment: staging (from environment, overrides playbook)
- debug_mode: true (from command line, overrides playbook)

## Notes

- Environment variable names are automatically converted to lowercase
- The `ONIGIRAZU_VAR_` prefix is required for environment variables to be recognized
- Command-line extra variables always have the highest priority
- Variables can be passed multiple times: `-e key1=value1 -e key2=value2`
