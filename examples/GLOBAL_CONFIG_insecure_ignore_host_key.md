# Global Configuration for insecure_ignore_host_key

## Overview

Starting from this version, Onigirazu supports **global default configuration** for `insecure_ignore_host_key` parameter. This allows you to set a default value that applies to all hosts unless explicitly overridden.

## Priority System

The `insecure_ignore_host_key` parameter follows this priority (from highest to lowest):

1. **Host-level** (in inventory file) - highest priority
2. **Group-level** (in inventory file)
3. **Global default** (in config file or environment variable) - lowest priority

## Configuration Methods

### Method 1: Configuration File

Create or edit `onigirazu.yml` in your project directory:

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true  # Apply to all hosts by default
```

Then run:

```bash
onigirazu-cli apply -c onigirazu.yml playbook.yml
```

### Method 2: Environment Variable

Set the environment variable:

```bash
export ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY=true
```

Then run normally:

```bash
onigirazu-cli apply playbook.yml
```

### Method 3: Per-Session

```bash
ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY=true onigirazu-cli apply playbook.yml
```

## Complete Example

### Scenario: Dev Environment with Global Insecure Mode

**onigirazu.yml:**

```yaml
# Global configuration
default_insecure_ignore_host_key: true  # All hosts ignore host key verification

# Other settings
log_level: debug
max_concurrency: 5
```

**inventory.yml:**

```yaml
hosts:
  dev-server-1:
    address: 192.168.1.10
    # Will use global default: insecure_ignore_host_key = true

  dev-server-2:
    address: 192.168.1.11
    # Will use global default: insecure_ignore_host_key = true

  prod-server:
    address: 10.0.1.100
    insecure_ignore_host_key: false  # Override: use secure mode for this host
```

**Result:**

- `dev-server-1`: insecure mode (from global default)
- `dev-server-2`: insecure mode (from global default)
- `prod-server`: **secure mode** (host-level override)

## Use Cases

### 1. Development Environment

**Problem:** You have many dev servers with frequently changing SSH keys.

**Solution:**

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true
```

All dev servers will ignore host key verification by default.

### 2. Mixed Environment (Dev + Prod)

**Problem:** You want insecure mode for dev, but secure for prod.

**Solution:**

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true  # Default for dev
```

```yaml
# inventory.yml
groups:
  production:
    hosts:
      - prod-1
      - prod-2
    vars:
      insecure_ignore_host_key: false  # Override for prod group

hosts:
  dev-1:
    address: 192.168.1.10
    # Uses global default (true)

  dev-2:
    address: 192.168.1.11
    # Uses global default (true)

  prod-1:
    address: 10.0.1.100
    # Uses group override (false)

  prod-2:
    address: 10.0.1.101
    # Uses group override (false)
```

### 3. CI/CD Pipeline

**Problem:** CI/CD creates ephemeral hosts with dynamic SSH keys.

**Solution:**

```bash
# .gitlab-ci.yml or .github/workflows/deploy.yml
script:
  - export ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY=true
  - onigirazu-cli apply playbook.yml
```

## Comparison: Before vs After

### Before (without global config)

You had to set `insecure_ignore_host_key: true` for **every host**:

```yaml
# inventory.yml
hosts:
  server-1:
    address: 192.168.1.10
    insecure_ignore_host_key: true  # Repetitive!

  server-2:
    address: 192.168.1.11
    insecure_ignore_host_key: true  # Repetitive!

  server-3:
    address: 192.168.1.12
    insecure_ignore_host_key: true  # Repetitive!

  # ... 50 more servers with the same setting
```

### After (with global config)

Set once globally:

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true
```

```yaml
# inventory.yml
hosts:
  server-1:
    address: 192.168.1.10
    # Automatically uses global default

  server-2:
    address: 192.168.1.11
    # Automatically uses global default

  server-3:
    address: 192.168.1.12
    # Automatically uses global default

  # ... 50 more servers - all use global default
```

## Priority Examples

### Example 1: Host Override

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true
```

```yaml
# inventory.yml
hosts:
  myhost:
    address: 192.168.1.10
    insecure_ignore_host_key: false  # Host-level wins!
```

**Result:** `myhost` uses **secure mode** (host-level has highest priority)

### Example 2: Group Override

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true
```

```yaml
# inventory.yml
groups:
  mygroup:
    hosts:
      - myhost
    vars:
      insecure_ignore_host_key: false  # Group-level wins over global

hosts:
  myhost:
    address: 192.168.1.10
    # No explicit setting
```

**Result:** `myhost` uses **secure mode** (group-level overrides global)

### Example 3: Global Default

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true
```

```yaml
# inventory.yml
hosts:
  myhost:
    address: 192.168.1.10
    # No explicit setting, no group vars
```

**Result:** `myhost` uses **insecure mode** (global default applies)

## Environment Variables Reference

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY` | boolean | `false` | Global default for all hosts |
| `ONIGIRAZU_SSH_STRICT_HOST_KEY` | boolean | `false` | Legacy setting (deprecated) |

## Configuration File Reference

```yaml
# onigirazu.yml

# SSH/Connection settings
default_insecure_ignore_host_key: false  # Global default (false = secure)
ssh_timeout: 30s
ssh_keepalive: 60s
ssh_max_sessions: 10
connection_reuse: true
ssh_known_hosts_file: ""  # Empty = use ~/.ssh/known_hosts

# Other settings
max_concurrency: 10
log_level: info
enable_caching: true
```

## Security Warnings

⚠️ **IMPORTANT:**

- **Default is SECURE** (`false`) - you must explicitly enable insecure mode
- **Never use in production** - only for dev/test/CI environments
- **Vulnerable to MITM attacks** when enabled
- **Use proper SSH key management** for production servers

## Troubleshooting

### Issue: Global setting not working

**Check:**

1. Config file location: `onigirazu.yml` in current directory
2. Environment variable: `echo $ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY`
3. Host/group overrides in inventory

**Debug:**

```bash
onigirazu-cli apply playbook.yml --log-level debug
```

Look for:

```
DEBUG: Set default insecure_ignore_host_key to: true
DEBUG: Host 'myhost' InsecureIgnoreHostKey: true
```

### Issue: Some hosts still fail with "host key verification failed"

**Reason:** Host or group has explicit `insecure_ignore_host_key: false`

**Solution:** Check inventory for overrides:

```bash
grep -r "insecure_ignore_host_key" inventory.yml
```

## Migration Guide

### From: Inventory-only configuration

**Before:**

```yaml
# inventory.yml
groups:
  all:
    vars:
      insecure_ignore_host_key: true
```

**After:**

```yaml
# onigirazu.yml
default_insecure_ignore_host_key: true
```

```yaml
# inventory.yml (simplified)
hosts:
  server-1:
    address: 192.168.1.10
```

### From: Environment variable per-run

**Before:**

```bash
# Every time you run
INSECURE=true onigirazu-cli apply playbook.yml
```

**After:**

```bash
# Set once
export ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY=true

# Run normally
onigirazu-cli apply playbook.yml
```

## Best Practices

### ✅ DO

1. **Use global config for dev environments**

   ```yaml
   # dev/onigirazu.yml
   default_insecure_ignore_host_key: true
   ```

2. **Override for sensitive hosts**

   ```yaml
   hosts:
     prod-db:
       insecure_ignore_host_key: false  # Always secure
   ```

3. **Use environment-specific configs**

   ```
   config/
     ├── dev.yml          # insecure: true
     ├── staging.yml      # insecure: false
     └── production.yml   # insecure: false
   ```

### ❌ DON'T

1. **Don't use global insecure mode in production**

   ```yaml
   # ❌ BAD for production
   default_insecure_ignore_host_key: true
   ```

2. **Don't commit insecure configs to production repos**

   ```bash
   # .gitignore
   dev-onigirazu.yml  # Contains insecure settings
   ```

3. **Don't mix secure and insecure without clear separation**

   ```yaml
   # ❌ Confusing
   default_insecure_ignore_host_key: true
   hosts:
     prod-1: ...  # Wait, is this secure or not?
   ```

## Related Documentation

- [Main README](./README_insecure_ignore_host_key.md) - Complete guide
- [Quick Reference](./QUICK_REFERENCE_insecure_ignore_host_key.md) - Quick lookup
- [FAQ](./FAQ_insecure_ignore_host_key.md) - Common questions
- [Ukrainian Quick Start](./ШВИДКИЙ_СТАРТ.md) - Швидкий старт українською

## Summary

**Global configuration** provides:

- ✅ Less repetition in inventory files
- ✅ Easier management of dev environments
- ✅ Flexible override system
- ✅ Environment-specific settings
- ✅ Backward compatible (default is secure)

**Remember:** Global default is the **lowest priority** - you can always override it at group or host level!
