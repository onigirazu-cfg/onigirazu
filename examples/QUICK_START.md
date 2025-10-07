# Quick Start Guide - Package Module with Lists

## 🚀 Quick Examples

### Install Multiple Packages

```yaml
---
plays:
  - name: "Setup server"
    hosts: all
    tasks:
      - name: "Install essential tools"
        module:
          type: "package"
          name:
            - "git"
            - "curl"
            - "wget"
            - "vim"
          state: "present"
```

### Run the Playbook

```bash
onigirazu -playbook examples/01-package-management-improved.yml \
          -inventory examples/inventory-correct.yml
```

## 📋 Syntax Comparison

### ❌ Old Way (Multiple Tasks)

```yaml
- name: "Install git"
  module: { type: "package", name: "git", state: "present" }
- name: "Install curl"
  module: { type: "package", name: "curl", state: "present" }
- name: "Install wget"
  module: { type: "package", name: "wget", state: "present" }
```

### ✅ New Way (Single Task with List)

```yaml
- name: "Install tools"
  module:
    type: "package"
    name: ["git", "curl", "wget"]
    state: "present"
```

## 🎯 Common Use Cases

### 1. Development Environment Setup

```yaml
- name: "Install dev tools"
  module:
    type: "package"
    name:
      - "git"
      - "vim"
      - "tmux"
      - "htop"
      - "tree"
    state: "present"
```

### 2. Web Server Setup

```yaml
- name: "Install web server packages"
  module:
    type: "package"
    name:
      - "nginx"
      - "certbot"
      - "python3-certbot-nginx"
    state: "present"
```

### 3. Database Server Setup

```yaml
- name: "Install database packages"
  module:
    type: "package"
    name:
      - "postgresql"
      - "postgresql-contrib"
      - "redis-server"
    state: "present"
```

### 4. Cleanup Unnecessary Packages

```yaml
- name: "Remove bloat"
  module:
    type: "package"
    name:
      - "nano"
      - "emacs"
      - "vim-tiny"
    state: "absent"
```

## 📊 Parameters

| Parameter | Type | Required | Default | Example |
|-----------|------|----------|---------|---------|
| `name` | string or list | ✅ Yes | - | `"git"` or `["git", "curl"]` |
| `state` | string | No | `present` | `present`, `absent`, `latest` |
| `version` | string | No | - | `"1.2.3-1"` |
| `update_cache` | boolean | No | `false` | `true` |

## ✅ States

- **`present`** - Install if not already installed (default)
- **`absent`** - Remove if installed
- **`latest`** - Install or update to latest version

## 🔍 Checking Results

### View Task Output

```bash
onigirazu -playbook playbook.yml -inventory hosts.yml -verbose
```

### Verify on Remote Host

```bash
ssh user@host "dpkg -l | grep -E '^ii  (git|curl|wget)'"
```

### Test Idempotency

```bash
# Run twice - second run should show changed=false
onigirazu -playbook playbook.yml -inventory hosts.yml
onigirazu -playbook playbook.yml -inventory hosts.yml
```

## 💡 Tips

1. **Group Related Packages** - Keep related packages in the same task
2. **Use Descriptive Names** - Make task names clear and meaningful
3. **Test Idempotency** - Always run playbooks twice to verify
4. **Check Errors** - Use `-verbose` flag to see detailed output

## 📚 More Information

- [PACKAGE_MODULE_EXAMPLES.md](PACKAGE_MODULE_EXAMPLES.md) - Complete documentation
- [01-package-management-improved.yml](01-package-management-improved.yml) - Working example
- [02-package-advanced.yml](02-package-advanced.yml) - Advanced examples

## 🎉 That's It

You're ready to use the package module with list support. Happy automating! 🚀
