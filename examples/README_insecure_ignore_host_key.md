# Using insecure_ignore_host_key in Onigirazu

## Overview

The `insecure_ignore_host_key` parameter allows you to disable SSH host key verification. This is useful for development and testing environments, but **should never be used in production**.

## ⚠️ Security Warning

**DO NOT USE IN PRODUCTION!**

When `insecure_ignore_host_key: true` is set:

- SSH will accept **any** host key without verification
- You are vulnerable to **man-in-the-middle attacks**
- There is **no guarantee** you're connecting to the intended server

## How It Works

### 1. Configuration in Inventory

The `insecure_ignore_host_key` parameter is set in your inventory file, **not** in task arguments.

#### Method 1: Per-Host Configuration

```yaml
hosts:
  dev-server:
    address: 192.168.1.100
    port: 22
    user: deploy
    insecure_ignore_host_key: true  # Only this host ignores host keys
```

#### Method 2: Per-Group Configuration

```yaml
groups:
  dev-servers:
    hosts:
      - dev-server-01
      - dev-server-02
    vars:
      insecure_ignore_host_key: true  # All hosts in this group ignore host keys
```

#### Method 3: Global Configuration

```yaml
groups:
  all:
    vars:
      insecure_ignore_host_key: true  # ALL hosts ignore host keys
```

### 2. Priority Order

Settings are applied in this order (highest to lowest priority):

1. **Host-level** setting (in `hosts` section)
2. **Group-level** setting (in `groups.<group_name>.vars`)
3. **All-group** setting (in `groups.all.vars`)

Example:

```yaml
hosts:
  server-01:
    address: 192.168.1.100
    insecure_ignore_host_key: false  # Host-level: secure

groups:
  dev-servers:
    hosts:
      - server-01
    vars:
      insecure_ignore_host_key: true  # Group-level: insecure
```

Result: `server-01` will use **secure** mode (host-level takes precedence).

### 3. Inventory Format Requirements

⚠️ **IMPORTANT:** The `insecure_ignore_host_key` parameter is **ONLY supported** in structured inventory formats:

**✅ Supported Formats:**

- YAML (`.yml`, `.yaml`)
- TOML (`.toml`)
- JSON (`.json`)

**❌ NOT Supported:**

- Text format (`.txt`, `.ini`)

**Why?** The text format only supports basic host information (`user@host:port`). It cannot store additional parameters like `insecure_ignore_host_key`.

**Solution:** If you're using text format and need `insecure_ignore_host_key`, convert your inventory to YAML:

```text
# ❌ inventory.txt - Cannot use insecure_ignore_host_key
192.168.1.10
192.168.1.11:2222
```

```yaml
# ✅ inventory.yml - Can use insecure_ignore_host_key
hosts:
  server-01:
    address: 192.168.1.10
    insecure_ignore_host_key: true
  server-02:
    address: 192.168.1.11
    port: 2222
    insecure_ignore_host_key: true
```

See [Text Format Limitations](./inventory_text_format_limitations.md) for detailed migration guide.

### 4. Technical Implementation

When you create a module using `BaseExecutorModule`, the setting is automatically handled:

```go
type MyModule struct {
    *modules.BaseExecutorModule
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // The host parameter contains InsecureIgnoreHostKey from inventory
    // You can check it: host.InsecureIgnoreHostKey

    // Pattern 1: Automatic handling
    output, err := m.WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error) {
        return exec.Execute("hostname")
    })
    // The executor automatically uses host.InsecureIgnoreHostKey

    // Pattern 2: Manual executor creation
    exec, err := m.CreateExecutor(host)
    if err != nil {
        return result, err
    }
    defer exec.Close()
    // The executor automatically uses host.InsecureIgnoreHostKey

    return result, nil
}
```

### 4. Internal Flow

```
Inventory File (YAML/JSON/TOML)
    ↓
Parser reads insecure_ignore_host_key
    ↓
types.Host struct (Host.InsecureIgnoreHostKey = true/false)
    ↓
Module.Execute(ctx, host, args) receives Host object
    ↓
BaseExecutorModule.CreateExecutor(host) or WithExecutor(host, func)
    ↓
executor.NewCommandExecutor(host) creates executor
    ↓
ssh.NewClient(host) creates SSH client
    ↓
ssh.NewHostKeyManagerWithInsecure("", false, host.InsecureIgnoreHostKey)
    ↓
HostKeyManager.VerifyHostKey() called during SSH handshake
    ↓
if insecure == true: return nil (skip verification)
if insecure == false: verify against known_hosts file
```

## Use Cases

### ✅ Acceptable Use Cases

1. **Development environments** with ephemeral VMs
2. **CI/CD pipelines** with dynamically created containers
3. **Testing** with frequently recreated infrastructure
4. **Docker containers** that get new host keys on each restart
5. **Vagrant VMs** in local development

### ❌ Never Use For

1. **Production servers**
2. **Staging environments** that mirror production
3. **Any server with sensitive data**
4. **Public-facing infrastructure**
5. **Compliance-regulated systems**

## Examples

### Example 1: Development Inventory

```yaml
# inventory_dev.yml
hosts:
  local-vm:
    address: 192.168.56.10
    user: vagrant
    insecure_ignore_host_key: true  # OK for local Vagrant VM

groups:
  dev:
    hosts:
      - local-vm
    vars:
      environment: development
```

### Example 2: Mixed Environment

```yaml
# inventory_mixed.yml
hosts:
  dev-server:
    address: 192.168.1.100
    user: deploy
    insecure_ignore_host_key: true  # Development server

  prod-server:
    address: 10.0.1.100
    user: deploy
    key_file: ~/.ssh/prod_key
    # insecure_ignore_host_key NOT set = secure by default

groups:
  dev:
    hosts:
      - dev-server
    vars:
      debug: true

  prod:
    hosts:
      - prod-server
    vars:
      backup_enabled: true
```

### Example 3: Playbook Usage

```yaml
# playbook.yml
---
- name: Deploy to development
  hosts: dev-server  # Uses insecure_ignore_host_key: true
  tasks:
    - name: Update code
      git:
        repo: https://github.com/example/app.git
        dest: /opt/app

- name: Deploy to production
  hosts: prod-server  # Uses secure host key verification
  tasks:
    - name: Update code
      git:
        repo: https://github.com/example/app.git
        dest: /opt/app
```

## Best Practices

### 1. Use Separate Inventory Files

```bash
# Development
onigirazu-playbook -i inventory_dev.yml playbook.yml

# Production (secure)
onigirazu-playbook -i inventory_prod.yml playbook.yml
```

### 2. Document Why It's Used

```yaml
hosts:
  ci-runner:
    address: 172.16.0.10
    user: ci
    # insecure_ignore_host_key: true
    # Reason: CI containers are recreated on each run with new host keys
    # Security: OK - isolated CI network, no sensitive data
    insecure_ignore_host_key: true
```

### 3. Use Environment Variables

```yaml
hosts:
  test-server:
    address: "{{ lookup('env', 'TEST_SERVER_IP') }}"
    user: test
    insecure_ignore_host_key: "{{ lookup('env', 'INSECURE_MODE') | default(false) }}"
```

### 4. Prefer SSH Key Management

Instead of disabling host key verification, consider:

```bash
# Add host key to known_hosts
ssh-keyscan -H 192.168.1.100 >> ~/.ssh/known_hosts

# Or connect once manually
ssh deploy@192.168.1.100
```

## Troubleshooting

### Problem: "Host key verification failed"

**Solution 1**: Add host key to known_hosts

```bash
ssh-keyscan -H <hostname> >> ~/.ssh/known_hosts
```

**Solution 2**: Remove old host key

```bash
ssh-keygen -R <hostname>
ssh-keyscan -H <hostname> >> ~/.ssh/known_hosts
```

**Solution 3**: Use insecure_ignore_host_key (only for dev/test)

```yaml
hosts:
  myhost:
    insecure_ignore_host_key: true
```

### Problem: Setting not working

Check priority order:

1. Host-level setting overrides group-level
2. Group-level setting overrides all-level
3. Default is `false` (secure)

### Problem: Need different settings per environment

Use separate inventory files:

```bash
inventory/
  ├── dev.yml          # insecure_ignore_host_key: true
  ├── staging.yml      # insecure_ignore_host_key: false
  └── production.yml   # insecure_ignore_host_key: false
```

## Related Documentation

- [SSH Client Implementation](../../internal/ssh/client.go)
- [Host Key Manager](../../internal/ssh/hostkey.go)
- [BaseExecutorModule](../../internal/modules/base_executor_module.go)
- [Example Module](./example_module_with_base_executor.go)
- [Inventory Examples](./inventory_with_insecure_host_key.yml)

## Security Checklist

Before using `insecure_ignore_host_key: true`, verify:

- [ ] This is NOT a production environment
- [ ] This is NOT a staging environment
- [ ] No sensitive data is involved
- [ ] The network is trusted/isolated
- [ ] You understand the security implications
- [ ] You have documented why it's needed
- [ ] You have a plan to use proper host key verification in production
- [ ] Your security team (if any) has approved this usage

## FAQ

### Q: Do I need to modify module code to use this setting?

**A:** **NO!** If you use `BaseExecutorModule`, the setting is automatically applied. The executor handles it internally.

```go
type MyModule struct {
    *modules.BaseExecutorModule
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // host.InsecureIgnoreHostKey is already set from inventory

    // All these methods automatically use host.InsecureIgnoreHostKey:
    output, err := m.WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error) {
        return exec.Execute("hostname")
    })

    // Nothing extra needed!
    return result, nil
}
```

### Q: Can I use `insecure_ignore_host_key` with text-based inventory?

**A:** **NO**, text format (`.txt`, `.ini`) **does not support** this parameter. Only structured formats (YAML, TOML, JSON) support it.

**Doesn't work:**

```text
# inventory.txt
192.168.1.10
192.168.1.11:2222
```

**Use YAML instead:**

```yaml
# inventory.yml
hosts:
  server-01:
    address: 192.168.1.10
    insecure_ignore_host_key: true
```

See [Text Format Limitations](./inventory_text_format_limitations.md) for migration guide.

### Q: What are the default settings?

**A:** By default, `insecure_ignore_host_key` is `false` (secure mode).

This means SSH verifies host keys against `~/.ssh/known_hosts`.

### Q: How can I fix "Host key verification failed"?

**A:** There are three approaches:

**1. For development (quick, but risky):**

```yaml
hosts:
  myhost:
    insecure_ignore_host_key: true
```

**2. For production (correct approach):**

```bash
# Add host key to known_hosts
ssh-keyscan -H hostname >> ~/.ssh/known_hosts
```

**3. If the key changed:**

```bash
# Remove old key
ssh-keygen -R hostname
# Add new key
ssh-keyscan -H hostname >> ~/.ssh/known_hosts
```

### Q: Can I use different settings for different environments?

**A:** Yes! Use separate inventory files:

```bash
inventory/
  ├── dev.yml          # insecure_ignore_host_key: true
  ├── staging.yml      # insecure_ignore_host_key: false
  └── production.yml   # insecure_ignore_host_key: false (or not set)
```

**Usage:**

```bash
# Development
onigirazu-cli run -i inventory/dev.yml playbook.yml

# Staging
onigirazu-cli run -i inventory/staging.yml playbook.yml

# Production
onigirazu-cli run -i inventory/production.yml playbook.yml
```

### Q: How do I verify this setting is working?

**A:** Enable debug logging:

```bash
onigirazu-cli run -i inventory.yml playbook.yml --log-level debug
```

Look for in logs:

```
DEBUG: Host 'myhost' InsecureIgnoreHostKey: true
DEBUG: Skipping host key verification for host 'myhost'
```

## Summary

| Aspect | Details |
|--------|---------|
| **Configuration** | Set in inventory file (YAML/JSON/TOML) |
| **Scope** | Per-host, per-group, or global |
| **Default** | `false` (secure) |
| **Module Code** | No changes needed - automatic |
| **Use Case** | Development/testing only |
| **Production** | ❌ Never use |
| **Security Risk** | High - vulnerable to MITM attacks |

Remember: **Security first!** Only use `insecure_ignore_host_key: true` when absolutely necessary and never in production.
