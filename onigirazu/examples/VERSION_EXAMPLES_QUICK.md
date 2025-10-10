# Quick Reference: Package Versions

## 🚀 5 Ways to Specify Versions

### 1️⃣ Single Package + Version

```yaml
- name: "Install git 2.43"
  module:
    type: "package"
    name: "git"
    version: "1:2.43.0-1ubuntu7.3"
    state: "present"
```

### 2️⃣ Multiple Packages + Same Version

```yaml
- name: "Install tools (latest)"
  module:
    type: "package"
    name: ["git", "curl", "wget"]
    version: ""  # Empty = latest
    state: "present"
```

### 3️⃣ Multiple Packages + Individual Versions

```yaml
- name: "Install with specific versions"
  module:
    type: "package"
    name:
      - name: "git"
        version: "1:2.43.0-1ubuntu7.3"
      - name: "curl"
        version: "8.5.0-2ubuntu10.6"
      - name: "wget"
        # No version = latest
    state: "present"
```

### 4️⃣ Mixed Format

```yaml
- name: "Mixed packages"
  module:
    type: "package"
    name:
      - name: "git"
        version: "1:2.43.0-1ubuntu7.3"
      - "curl"  # String = latest
      - name: "wget"  # Object without version = latest
    state: "present"
```

### 5️⃣ Global Version + Override

```yaml
- name: "Global with override"
  module:
    type: "package"
    name:
      - name: "git"
        version: "1:2.43.0-1ubuntu7.3"  # Override
      - "curl"  # Uses global
    version: "latest"  # Global default
    state: "present"
```

---

## 🔍 Find Package Versions

### Ubuntu/Debian

```bash
apt-cache policy git
apt-cache madison git
dpkg -l | grep git
```

### RHEL/CentOS

```bash
yum list available git --showduplicates
rpm -qa | grep git
```

### macOS

```bash
brew info git
brew list --versions git
```

---

## 💡 Common Patterns

### Production (Pinned Versions)

```yaml
name:
  - name: "nginx"
    version: "1.24.0-2ubuntu7"
  - name: "postgresql"
    version: "16+262.pgdg24.04+1"
state: "present"
```

### Development (Latest)

```yaml
name: ["git", "vim", "htop"]
state: "latest"
```

### Security Updates

```yaml
name: "openssl"
state: "latest"  # Always update
```

---

## ⚙️ Version Behavior

| State | Version | Result |
|-------|---------|--------|
| `present` | Empty | Install latest |
| `present` | Specified | Install/upgrade to version |
| `absent` | Any | Remove (ignores version) |
| `latest` | Any | Update to latest (ignores version) |

---

## 📊 Output Example

```json
{
  "packages": {
    "git": {
      "requested_version": "1:2.43.0-1ubuntu7.3",
      "current_version": "1:2.43.0-1ubuntu7.3",
      "action": "already_installed",
      "changed": false
    }
  }
}
```

---

## 🎯 Best Practices

✅ **Pin versions in production**
✅ **Use latest in development**
✅ **Document version choices**
✅ **Test in staging first**

---

## 📚 Full Documentation

See [PACKAGE_VERSIONS_GUIDE.md](PACKAGE_VERSIONS_GUIDE.md) for complete guide.
