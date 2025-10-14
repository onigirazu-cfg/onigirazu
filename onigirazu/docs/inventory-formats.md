# Inventory File Formats

Onigirazu supports multiple inventory file formats to provide flexibility in managing your infrastructure.

## Auto-Detection

When you don't specify an inventory file with the `-inventory` flag, Onigirazu automatically searches for inventory files in the playbook directory with the following names (in order):

1. `inventory.yml`
2. `inventory.yaml`
3. `inventory.toml`
4. `hosts`
5. `hosts.yml`
6. `hosts.yaml`
7. `hosts.toml`
8. `inventory` (simple list format)

If no inventory file is found, Onigirazu continues with only `localhost` available.

## Supported Formats

### 1. YAML Format (Traditional)

The standard YAML format with full support for hosts, groups, and variables:

```yaml
hosts:
  - name: web-server-01
    address: 192.168.1.10
    port: 22
    user: deploy
    vars:
      http_port: 80
      domain: example.com

groups:
  webservers:
    hosts:
      web-server-01:
        address: 192.168.1.10
    vars:
      nginx_version: latest
```

See `examples/inventory.yml` for a complete example.

### 2. TOML Format

A more structured format using TOML syntax:

```toml
# Individual hosts
[hosts.web-server-01]
address = "192.168.1.10"
port = 22
user = "deploy"
[hosts.web-server-01.vars]
http_port = 80
domain = "example.com"

# Groups
[groups.webservers]
hosts = ["web-server-01"]
[groups.webservers.vars]
nginx_version = "latest"
```

See `inventory.example.toml` for a complete example.

### 3. Simple List Format

A plain text file with one host per line. Perfect for quick setups or simple infrastructures:

```
# Comments are supported
127.0.0.1
192.168.1.10
deploy@192.168.1.11
deploy@192.168.1.12:2222
user@server.example.com:22
```

Supported address formats:

- **Plain IP**: `192.168.1.10` (uses default port 22 and user "root")
- **IP with port**: `192.168.1.10:2222`
- **Hostname**: `server.example.com`
- **Hostname with port**: `server.example.com:2222`
- **With user**: `user@192.168.1.10`
- **With user and port**: `user@192.168.1.10:2222`

All hosts are automatically added to the `all` group.

See `inventory.example.txt` for a complete example.

## Usage Examples

### Using a specific inventory file

```bash
# YAML format
onigirazu -playbook playbook.yml -inventory inventory.yml

# TOML format
onigirazu -playbook playbook.yml -inventory inventory.toml

# Simple list format
onigirazu -playbook playbook.yml -inventory hosts.txt
```

### Auto-detection (recommended)

```bash
# Place inventory file in the same directory as your playbook
# with one of the standard names (inventory.yml, hosts, etc.)
onigirazu -playbook playbook.yml
```

## Format Selection

Onigirazu automatically detects the format based on:

1. **File extension**: `.toml`, `.yml`, `.yaml`
2. **Content analysis**: If extension is ambiguous, analyzes content to detect format
3. **Fallback order**: YAML → TOML → Simple List

## Best Practices

- **YAML**: Use for complex inventories with multiple groups and variables
- **TOML**: Use when you prefer more structured configuration files
- **Simple List**: Use for quick tests, simple setups, or when you only need basic host lists

## Migration

All formats are fully compatible. You can:

- Convert between formats without losing functionality
- Use different formats for different environments
- Mix formats across different playbooks

The simple list format is a subset of functionality - it only supports hosts without custom variables or complex group structures.
