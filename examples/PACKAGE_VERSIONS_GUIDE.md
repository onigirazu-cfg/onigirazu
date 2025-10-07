# Package Module - Version Management Guide

## 📦 Overview

The `package` module supports multiple ways to specify package versions, from simple global versions to per-package version control.

## 🎯 Supported Formats

### Format 1: Single Package with Version

```yaml
- name: "Install specific version"
  module:
    type: "package"
    name: "git"
    version: "1:2.43.0-1ubuntu7.3"
    state: "present"
```

**Use case:** When you need to pin a single package to a specific version.

---

### Format 2: Multiple Packages with Same Version

```yaml
- name: "Install packages with same version"
  module:
    type: "package"
    name: ["curl", "wget", "git"]
    version: "latest"  # or specific version if applicable
    state: "present"
```

**Use case:** When multiple packages should use the same version constraint.

---

### Format 3: Multiple Packages with Individual Versions

```yaml
- name: "Install packages with specific versions"
  module:
    type: "package"
    name:
      - name: "git"
        version: "1:2.43.0-1ubuntu7.3"
      - name: "curl"
        version: "8.5.0-2ubuntu10.6"
      - name: "wget"
        # No version = latest available
    state: "present"
```

**Use case:** When each package needs a different version.

---

### Format 4: Mixed Format (Strings and Objects)

```yaml
- name: "Install mixed packages"
  module:
    type: "package"
    name:
      - name: "git"
        version: "1:2.43.0-1ubuntu7.3"
      - "curl"  # Simple string = latest
      - name: "wget"
        # Object without version = latest
    state: "present"
```

**Use case:** When some packages need specific versions and others can use latest.

---

### Format 5: Global Version with Override

```yaml
- name: "Install with global version and overrides"
  module:
    type: "package"
    name:
      - name: "git"
        version: "1:2.43.0-1ubuntu7.3"  # Override
      - "curl"  # Uses global version
      - name: "wget"
        # No version = uses global version
    version: "latest"  # Global default
    state: "present"
```

**Use case:** Set a default version for most packages, but override for specific ones.

---

## 📋 Version Specification Examples

### Ubuntu/Debian (apt)

```yaml
# Full version with epoch and revision
- name: "git"
  version: "1:2.43.0-1ubuntu7.3"

# Version without epoch
- name: "curl"
  version: "8.5.0-2ubuntu10.6"

# Major.minor version
- name: "python3"
  version: "3.12"

# Empty or omitted = latest
- name: "wget"
  version: ""
```

### RHEL/CentOS (yum)

```yaml
# Full version with release
- name: "git"
  version: "2.43.0-1.el8"

# Version only
- name: "curl"
  version: "8.5.0"
```

### macOS (brew)

```yaml
# Brew typically uses latest
- name: "git"
  version: ""  # Latest from brew

# Specific version (if available)
- name: "python"
  version: "3.12"
```

---

## 🔍 How to Find Package Versions

### Ubuntu/Debian

```bash
# List available versions
apt-cache policy git

# Show installed version
dpkg -l | grep git

# Search for specific version
apt-cache madison git
```

### RHEL/CentOS

```bash
# List available versions
yum list available git --showduplicates

# Show installed version
rpm -qa | grep git
```

### macOS

```bash
# Show available versions
brew info git

# Show installed version
brew list --versions git
```

---

## 💡 Practical Examples

### Example 1: Development Environment with Pinned Versions

```yaml
---
plays:
  - name: "Setup dev environment"
    hosts: all
    tasks:
      - name: "Install development tools with specific versions"
        module:
          type: "package"
          name:
            - name: "git"
              version: "1:2.43.0-1ubuntu7.3"
            - name: "vim"
              version: "2:9.1.0016-1ubuntu7.9"
            - name: "htop"
              # Latest version
          state: "present"
```

### Example 2: Production Server with Version Control

```yaml
---
plays:
  - name: "Production server setup"
    hosts: production
    tasks:
      - name: "Install web server with specific versions"
        module:
          type: "package"
          name:
            - name: "nginx"
              version: "1.24.0-2ubuntu7"
            - name: "postgresql"
              version: "16+262.pgdg24.04+1"
            - name: "redis-server"
              version: "5:7.0.15-1ubuntu0.1"
          state: "present"
```

### Example 3: Update Specific Package to Latest

```yaml
---
plays:
  - name: "Update security packages"
    hosts: all
    tasks:
      - name: "Update git to latest"
        module:
          type: "package"
          name: "git"
          state: "latest"  # Always update to latest
```

### Example 4: Downgrade Package Version

```yaml
---
plays:
  - name: "Downgrade package"
    hosts: all
    tasks:
      - name: "Install older version of git"
        module:
          type: "package"
          name: "git"
          version: "1:2.40.0-1ubuntu1"  # Older version
          state: "present"
```

---

## ⚙️ Version Behavior by State

| State | Version Parameter | Behavior |
|-------|------------------|----------|
| `present` | Not specified | Install latest if not installed |
| `present` | Specified | Install specific version, upgrade/downgrade if different |
| `absent` | Any | Remove package (version ignored) |
| `latest` | Any | Always update to latest (version ignored) |

---

## 🎯 Best Practices

### 1. Pin Versions in Production

```yaml
# Good: Predictable production environment
- name: "Production packages"
  module:
    type: "package"
    name:
      - name: "nginx"
        version: "1.24.0-2ubuntu7"
      - name: "postgresql"
        version: "16+262.pgdg24.04+1"
    state: "present"
```

### 2. Use Latest in Development

```yaml
# Good: Always get latest features in dev
- name: "Dev tools"
  module:
    type: "package"
    name: ["git", "vim", "htop"]
    state: "latest"
```

### 3. Document Version Choices

```yaml
# Good: Explain why specific version is used
- name: "Install git 2.43.0 (required for feature X)"
  module:
    type: "package"
    name: "git"
    version: "1:2.43.0-1ubuntu7.3"
    state: "present"
```

### 4. Test Version Changes

```bash
# Always test version changes in staging first
onigirazu -playbook staging.yml -inventory staging-hosts.yml

# Then apply to production
onigirazu -playbook production.yml -inventory prod-hosts.yml
```

---

## 🔧 Troubleshooting

### Version Not Found

**Error:**

```
Error: failed to install: version '1.2.3' not found
```

**Solution:**

```bash
# Check available versions
apt-cache policy package-name

# Use correct version format
apt-cache madison package-name
```

### Version Conflict

**Error:**

```
Error: version conflict with dependencies
```

**Solution:**

- Check dependency versions
- Use compatible version range
- Update dependencies first

### Downgrade Not Allowed

**Error:**

```
Error: downgrade not permitted
```

**Solution:**

```yaml
# Force reinstall with specific version
- name: "Force specific version"
  module:
    type: "package"
    name: "package-name"
    state: "absent"

- name: "Install specific version"
  module:
    type: "package"
    name: "package-name"
    version: "1.2.3"
    state: "present"
```

---

## 📊 Output Format

When versions are specified, the output includes version information:

```json
{
  "packages": {
    "git": {
      "requested_version": "1:2.43.0-1ubuntu7.3",
      "current_version": "1:2.43.0-1ubuntu7.3",
      "installed": true,
      "action": "already_installed",
      "changed": false
    },
    "curl": {
      "requested_version": "",
      "current_version": "8.5.0-2ubuntu10.6",
      "installed": true,
      "action": "already_installed",
      "changed": false
    }
  },
  "package_count": 2
}
```

---

## 🚀 Advanced Usage

### Conditional Version Installation

```yaml
- name: "Install version based on OS"
  module:
    type: "package"
    name:
      - name: "git"
        version: "{{ git_version_ubuntu }}"  # Variable
    state: "present"
```

### Version Validation

```yaml
- name: "Verify installed version"
  module:
    type: "command"
    command: "git --version"
  register: git_version

- name: "Check version matches"
  module:
    type: "debug"
    msg: "Git version: {{ git_version.stdout }}"
```

---

## 📚 Related Documentation

- [PACKAGE_MODULE_EXAMPLES.md](PACKAGE_MODULE_EXAMPLES.md) - General package module examples
- [QUICK_START.md](QUICK_START.md) - Quick start guide
- [03-package-versions.yml](03-package-versions.yml) - Version examples playbook

---

## 🎉 Summary

The package module supports flexible version management:

✅ **Single package with version** - Pin specific versions
✅ **Global version for all packages** - Apply same version to multiple packages
✅ **Per-package versions** - Different version for each package
✅ **Mixed formats** - Combine strings and objects
✅ **Version override** - Global default with per-package overrides

Choose the format that best fits your use case!
