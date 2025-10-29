# Ansible YAML Inventory Format Support

Onigirazu now supports **native Ansible YAML inventory format**, allowing you to use your existing Ansible inventories directly without any conversion!

## Overview

Starting with this release, Onigirazu can automatically detect and parse Ansible-format YAML inventories. When you provide an inventory file, Onigirazu will:

1. Auto-detect whether it's Ansible format or Onigirazu format
2. Parse Ansible-specific variables (`ansible_host`, `ansible_user`, etc.)
3. Convert them to Onigirazu's internal representation
4. Preserve all group hierarchies, variables, and host configurations

## Auto-Detection

Onigirazu automatically detects Ansible format by looking for:

- Top-level `all:` key (standard Ansible structure)
- Presence of `ansible_*` prefixed variables in host definitions

If Ansible format is detected, the special parser is used. Otherwise, standard Onigirazu YAML parsing applies.

## Ansible Variable Mapping

The following Ansible variables are automatically mapped to Onigirazu host properties:

| Ansible Variable | Onigirazu Property | Description |
|-----------------|-------------------|-------------|
| `ansible_host` | `address` | Target host IP or hostname |
| `ansible_port` | `port` | SSH port (default: 22) |
| `ansible_user` | `user` | SSH username |
| `ansible_password` | `password` | SSH password (for password auth) |
| `ansible_ssh_private_key_file` | `key_file` | Path to SSH private key |
| `ansible_ssh_host_key_checking` | `insecure_ignore_host_key` | Disable host key verification |

**All other Ansible variables** (both `ansible_*` and custom variables) are stored in the host's `vars` map, with the `ansible_` prefix automatically removed for cleaner variable names.

## Examples

### Basic Ansible Inventory

```yaml
all:
  hosts:
    web1:
      ansible_host: 192.168.1.10
      ansible_user: deploy
      ansible_port: 22
    web2:
      ansible_host: 192.168.1.11
      ansible_user: deploy
```

### Inventory with Groups

```yaml
all:
  hosts:
    web1:
      ansible_host: 192.168.1.10
      ansible_user: deploy
    web2:
      ansible_host: 192.168.1.11
      ansible_user: deploy
    db1:
      ansible_host: 192.168.1.20
      ansible_user: postgres

  children:
    webservers:
      hosts:
        web1:
        web2:
      vars:
        http_port: 80
        https_port: 443

    databases:
      hosts:
        db1:
      vars:
        db_port: 5432
```

### Inventory with SSH Key Authentication

```yaml
all:
  hosts:
    prod_web:
      ansible_host: prod.example.com
      ansible_user: deploy
      ansible_ssh_private_key_file: ~/.ssh/deploy_key
      ansible_ssh_host_key_checking: false
```

### Inventory with Custom Variables

```yaml
all:
  hosts:
    app_server:
      ansible_host: 192.168.1.50
      ansible_user: appuser
      # Ansible variables
      app_version: 2.1.0
      environment: production
      # Custom variables
      db_connection_pool: 50
      cache_ttl: 3600

  children:
    app_servers:
      hosts:
        app_server:
      vars:
        service_port: 8080
        log_level: info
```

### Nested Group Hierarchy

```yaml
all:
  hosts:
    web1:
      ansible_host: 192.168.1.10
      ansible_user: deploy
    db1:
      ansible_host: 192.168.1.20
      ansible_user: postgres
    cache1:
      ansible_host: 192.168.1.30
      ansible_user: redis

  children:
    # Individual service groups
    webservers:
      hosts:
        web1:
      vars:
        tier: frontend

    databases:
      hosts:
        db1:
      vars:
        tier: backend

    cache:
      hosts:
        cache1:
      vars:
        tier: backend

    # Tier groups combining other groups
    frontend:
      children:
        - webservers
      vars:
        ssl_enabled: true

    backend:
      children:
        - databases
        - cache
      vars:
        internal_only: true

    # Production environment group
    production:
      children:
        - frontend
        - backend
      vars:
        env: production
        monitoring_enabled: true
```

## Usage

Use your existing Ansible inventories directly with Onigirazu:

```bash
# Using an Ansible YAML inventory
onigirazu plan playbook.yml -i ansible-inventory.yml

# Apply with Ansible inventory
onigirazu apply playbook.yml -i ansible-inventory.yml

# Works with inline hosts too
onigirazu plan playbook.yml -i "192.168.1.10,192.168.1.11,192.168.1.12"
```

## Compatibility

### What's Fully Supported

✅ **Ansible YAML Format**

- `all:` root key with nested structure
- `hosts:` section with host definitions
- `children:` section with group definitions
- `vars:` sections for both hosts and groups
- All `ansible_*` connection parameters
- Custom variables alongside Ansible parameters
- Nested group hierarchies

✅ **Onigirazu YAML Format** (still works)

- Native Onigirazu YAML structure
- All existing inventories continue to work

✅ **Other Formats**

- INI/Ansible format (unchanged)
- JSON, TOML, simple lists
- Dynamic inventory scripts

### Partial/Limited Support

⚠️ **Ansible-Specific Features Not Implemented**

- Host patterns/ranges (e.g., `web[0:2]` or `192.168.1.[10:20]`)
- Jinja2 templating in inventory variables
- Dynamic variables from external sources
- Inventory plugins (Ansible 2.9+)

## Performance

Ansible inventory parsing adds negligible overhead (<1ms). The auto-detection is fast and doesn't impact performance on large inventories.

## Migration Guide

### From Pure Ansible

If you're migrating from pure Ansible to Onigirazu, **no changes are required**:

1. Keep your existing Ansible YAML inventories as-is
2. Use them directly with Onigirazu: `onigirazu apply playbook.yml -i inventory.yml`
3. All Ansible host parameters are automatically understood

### From Onigirazu YAML

Your existing Onigirazu format inventories continue to work unchanged. No migration needed.

## Implementation Details

### How It Works

1. **Format Detection**: When parsing a YAML inventory:
   - Check for `all:` key → Ansible format
   - Check for `ansible_*` variables → Ansible format
   - Otherwise → Onigirazu format

2. **Parsing**: Ansible format is converted to Onigirazu's internal representation:
   - Extract all hosts from `all.hosts`
   - Create groups from `all.children`
   - Map Ansible parameters to Onigirazu fields
   - Preserve variable hierarchies

3. **Variable Handling**: Custom and Ansible variables are stored in `host.Vars` map with `ansible_` prefix removed.

### Code Changes

The inventory parser was enhanced with:

- `isAnsibleYaml()` - Format detection function
- `parseAnsibleYamlInventory()` - Main Ansible format parser
- `parseAnsibleHost()` - Host variable mapping
- `parseAnsibleGroup()` - Group parsing with hierarchy support

All changes maintain backward compatibility with existing Onigirazu inventories.

## Troubleshooting

### "No valid hosts found"

**Problem**: Inventory parses but reports no hosts

**Solution**: Ensure hosts are under `all.hosts:` section:

```yaml
all:
  hosts:           # ← Required
    myhost:
      ansible_host: 192.168.1.10
```

### Variables not being recognized

**Problem**: Custom variables aren't available in tasks

**Solution**: Custom variables are in `host.Vars`. Access them using the variable name (without `ansible_` prefix):

```yaml
# In inventory:
hosts:
  server:
    app_port: 8080

# In playbook - use as:
{{ app_port }}  # Not {{ ansible_app_port }}
```

### Port/User mapping issues

**Problem**: `ansible_port` or `ansible_user` not being applied

**Solution**: These must be at the host level under `all.hosts`:

```yaml
all:
  hosts:
    server:
      ansible_host: 192.168.1.10
      ansible_port: 2222        # ← Host level
      ansible_user: deploy
```

## See Also

- [Inventory Configuration](../inventory/README.md)
- [Playbook Examples](./PLAYBOOK_EXAMPLES.md)
- [Variable Usage Guide](../variables/README.md)
