# Text Format Inventory Limitations

## ⚠️ Important: Text Format Does NOT Support `insecure_ignore_host_key`

The simple text format (`.txt`, `.ini`) for inventory files **does not support** the `insecure_ignore_host_key` parameter or any other advanced host configuration options.

## What Text Format Supports

The text format only supports basic host connection information:

```text
# Format: [user@]host[:port]

# Examples:
192.168.1.10
192.168.1.11:2222
deploy@192.168.1.20
deploy@192.168.1.21:2222
root@server.example.com:22
```

### Supported in Text Format

- ✅ Host address (IP or hostname)
- ✅ Port number
- ✅ SSH user
- ✅ Comments (lines starting with `#`)

### NOT Supported in Text Format

- ❌ `insecure_ignore_host_key`
- ❌ `key_file`
- ❌ `password`
- ❌ Custom variables
- ❌ Groups
- ❌ Group variables
- ❌ Host-specific variables

## Solutions

### Solution 1: Use YAML Format (Recommended)

Convert your text inventory to YAML format to use `insecure_ignore_host_key`:

**Before (inventory.txt):**

```text
192.168.1.10
192.168.1.11:2222
deploy@192.168.1.20
```

**After (inventory.yml):**

```yaml
hosts:
  server-01:
    address: 192.168.1.10
    user: root
    port: 22
    insecure_ignore_host_key: true  # ✅ Now supported!

  server-02:
    address: 192.168.1.11
    user: root
    port: 2222
    insecure_ignore_host_key: true

  server-03:
    address: 192.168.1.20
    user: deploy
    port: 22
    insecure_ignore_host_key: true
```

### Solution 2: Use YAML with Groups

For multiple hosts with the same settings:

```yaml
groups:
  dev-servers:
    hosts:
      - server-01
      - server-02
      - server-03
    vars:
      insecure_ignore_host_key: true  # ✅ Applied to all hosts in group

hosts:
  server-01:
    address: 192.168.1.10
  server-02:
    address: 192.168.1.11
    port: 2222
  server-03:
    address: 192.168.1.20
    user: deploy
```

### Solution 3: Use TOML Format

TOML is another structured format that supports advanced options:

```toml
[hosts.server-01]
address = "192.168.1.10"
user = "root"
port = 22
insecure_ignore_host_key = true  # ✅ Supported!

[hosts.server-02]
address = "192.168.1.11"
user = "root"
port = 2222
insecure_ignore_host_key = true

[hosts.server-03]
address = "192.168.1.20"
user = "deploy"
port = 22
insecure_ignore_host_key = true
```

### Solution 4: Use JSON Format

JSON also supports all advanced options:

```json
{
  "hosts": {
    "server-01": {
      "address": "192.168.1.10",
      "user": "root",
      "port": 22,
      "insecure_ignore_host_key": true
    },
    "server-02": {
      "address": "192.168.1.11",
      "user": "root",
      "port": 2222,
      "insecure_ignore_host_key": true
    },
    "server-03": {
      "address": "192.168.1.20",
      "user": "deploy",
      "port": 22,
      "insecure_ignore_host_key": true
    }
  }
}
```

## Migration Guide

### Step 1: Identify Your Current Text Inventory

```text
# inventory.txt
192.168.1.10
192.168.1.11:2222
deploy@192.168.1.20
root@server.example.com:22
```

### Step 2: Convert to YAML

```yaml
# inventory.yml
hosts:
  host-1:
    address: 192.168.1.10
    user: root
    port: 22

  host-2:
    address: 192.168.1.11
    user: root
    port: 2222

  host-3:
    address: 192.168.1.20
    user: deploy
    port: 22

  host-4:
    address: server.example.com
    user: root
    port: 22
```

### Step 3: Add insecure_ignore_host_key

```yaml
# inventory.yml
hosts:
  host-1:
    address: 192.168.1.10
    user: root
    port: 22
    insecure_ignore_host_key: true  # ← Add this

  host-2:
    address: 192.168.1.11
    user: root
    port: 2222
    insecure_ignore_host_key: true  # ← Add this

  host-3:
    address: 192.168.1.20
    user: deploy
    port: 22
    insecure_ignore_host_key: true  # ← Add this

  host-4:
    address: server.example.com
    user: root
    port: 22
    insecure_ignore_host_key: true  # ← Add this
```

### Step 4: Update Your Commands

```bash
# Before:
onigirazu-cli run -i inventory.txt playbook.yml

# After:
onigirazu-cli run -i inventory.yml playbook.yml
```

## Comparison Table

| Feature | Text Format | YAML | TOML | JSON |
|---------|-------------|------|------|------|
| Basic host info | ✅ | ✅ | ✅ | ✅ |
| `insecure_ignore_host_key` | ❌ | ✅ | ✅ | ✅ |
| `key_file` | ❌ | ✅ | ✅ | ✅ |
| Custom variables | ❌ | ✅ | ✅ | ✅ |
| Groups | ❌ | ✅ | ✅ | ✅ |
| Group variables | ❌ | ✅ | ✅ | ✅ |
| Comments | ✅ | ✅ | ✅ | ❌ |
| Human-readable | ✅ | ✅ | ✅ | ⚠️ |
| Simple syntax | ✅ | ✅ | ⚠️ | ⚠️ |

## Why This Limitation Exists

The text format is designed for **simplicity** and **quick testing**. It's meant for scenarios where you just need to list a few hosts without any advanced configuration.

For production use or when you need advanced features like `insecure_ignore_host_key`, you should use a structured format (YAML, TOML, or JSON).

## Technical Details

The text format parser (`parseSimpleList` in `inventory_parser.go`) only extracts:

- User (from `user@host` syntax)
- Host address
- Port (from `host:port` syntax)

It does not support key-value pairs or additional parameters.

## Recommendations

### For Development/Testing

Use YAML with `insecure_ignore_host_key: true`:

```yaml
groups:
  all:
    vars:
      insecure_ignore_host_key: true  # Apply to all hosts
```

### For Production

Use YAML with proper SSH key configuration:

```yaml
hosts:
  prod-server:
    address: 10.0.1.100
    user: deploy
    key_file: ~/.ssh/prod_key
    # insecure_ignore_host_key: false (default - secure)
```

### For Quick Testing

Text format is fine if you don't need advanced features:

```text
# test-hosts.txt
192.168.1.10
192.168.1.11
```

## See Also

- [Complete insecure_ignore_host_key Guide](./README_insecure_ignore_host_key.md)
- [Quick Reference](./QUICK_REFERENCE_insecure_ignore_host_key.md)
- [YAML Inventory Example](./inventory_with_insecure_host_key.yml)
- [Inventory Examples](../../onigirazu/docs/examples/)

## Summary

**Question:** How to set `insecure_ignore_host_key: true` in text format inventory?

**Answer:** You **cannot**. Text format does not support this parameter. Use YAML, TOML, or JSON format instead.

**Quick Migration:**

```bash
# 1. Rename your file
mv inventory.txt inventory.yml

# 2. Convert format (manual or use a tool)
# 3. Add insecure_ignore_host_key where needed
# 4. Use the new file
onigirazu-cli run -i inventory.yml playbook.yml
```

---

**Remember:**

- 📝 Text format = Simple, limited features
- 🎯 YAML/TOML/JSON = Full features, including `insecure_ignore_host_key`
- 🔒 Always use secure mode in production (don't set `insecure_ignore_host_key: true`)
