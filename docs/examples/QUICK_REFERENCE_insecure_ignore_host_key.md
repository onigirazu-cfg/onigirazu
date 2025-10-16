# Quick Reference: insecure_ignore_host_key

## TL;DR

```yaml
# In inventory file (NOT in playbook tasks!)
hosts:
  myhost:
    address: 192.168.1.100
    insecure_ignore_host_key: true  # ⚠️ Dev/test only!
```

## Three Ways to Set It

### 1. Per Host

```yaml
hosts:
  dev-server:
    address: 192.168.1.100
    insecure_ignore_host_key: true
```

### 2. Per Group

```yaml
groups:
  dev-servers:
    hosts:
      - dev-server-01
      - dev-server-02
    vars:
      insecure_ignore_host_key: true
```

### 3. Global (All Hosts)

```yaml
groups:
  all:
    vars:
      insecure_ignore_host_key: true
```

## Priority

Host-level > Group-level > All-level

## In Module Code

**You don't need to do anything!** It's automatic.

```go
type MyModule struct {
    *modules.BaseExecutorModule
}

func (m *MyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
    // Option 1: Automatic (recommended)
    output, err := m.WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error) {
        return exec.Execute("hostname")
    })

    // Option 2: Also automatic
    exec, err := m.CreateExecutor(host)
    defer exec.Close()

    // Both automatically use host.InsecureIgnoreHostKey
    // No extra code needed!

    return result, nil
}
```

## When to Use

✅ **YES:**

- Local Vagrant VMs
- Docker containers (dev)
- CI/CD with dynamic hosts
- Ephemeral test environments

❌ **NO:**

- Production servers
- Staging environments
- Any server with real data
- Public-facing infrastructure

## Common Mistakes

### ❌ Wrong: Setting in task

```yaml
tasks:
  - name: Run command
    shell:
      command: hostname
      insecure_ignore_host_key: true  # ❌ This doesn't work!
```

### ❌ Wrong: Using text format inventory

```text
# inventory.txt
192.168.1.10
192.168.1.11:2222
# ❌ Text format doesn't support insecure_ignore_host_key!
# Use YAML/TOML/JSON instead
```

### ✅ Correct: Setting in inventory (YAML/TOML/JSON)

```yaml
# inventory.yml
hosts:
  myhost:
    insecure_ignore_host_key: true  # ✅ Correct!

# playbook.yml
tasks:
  - name: Run command
    shell:
      command: hostname  # Uses setting from inventory
```

## Troubleshooting

### "Host key verification failed"

**Quick fix for dev:**

```yaml
hosts:
  myhost:
    insecure_ignore_host_key: true
```

**Proper fix for prod:**

```bash
ssh-keyscan -H hostname >> ~/.ssh/known_hosts
```

### Setting not working?

Check priority:

```yaml
hosts:
  server:
    insecure_ignore_host_key: false  # ← This wins!

groups:
  mygroup:
    vars:
      insecure_ignore_host_key: true  # ← This is ignored
```

## Files

- Full docs: [README_insecure_ignore_host_key.md](./README_insecure_ignore_host_key.md)
- Flow diagram: [FLOW_insecure_ignore_host_key.md](./FLOW_insecure_ignore_host_key.md)
- Example inventory: [inventory_with_insecure_host_key.yml](./inventory_with_insecure_host_key.yml)
- Example playbook: [playbook_with_insecure_hosts.yml](./playbook_with_insecure_hosts.yml)
- Example module: [example_module_with_base_executor.go](./example_module_with_base_executor.go)
- Text format limitations: [inventory_text_format_limitations.md](./inventory_text_format_limitations.md)

## Remember

🔒 **Default is SECURE** (insecure_ignore_host_key: false)
⚠️ **Only use in dev/test**
🚫 **Never use in production**
✅ **No module code changes needed**
