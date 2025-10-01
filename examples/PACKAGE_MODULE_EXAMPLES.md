# Package Module - Examples and Usage

## Overview

The `package` module manages system packages across different package managers (apt, yum, brew, etc.). It supports both single package and multiple package operations.

## Features

✅ **Single Package Management** - Install, remove, or update individual packages
✅ **Batch Operations** - Process multiple packages in a single task
✅ **Idempotency** - Safe to run multiple times without side effects
✅ **Cross-Platform** - Automatically detects the correct package manager
✅ **Version Control** - Install specific package versions
✅ **Cache Management** - Update package cache before operations

## Supported Package Managers

- **apt-get** (Debian/Ubuntu)
- **yum** (RHEL/CentOS)
- **brew** (macOS)
- **pacman** (Arch Linux)
- **zypper** (openSUSE)

## Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `name` | string, list, or objects | Yes | - | Package name(s) to manage. Can be:<br>• String: `"git"`<br>• List: `["git", "curl"]`<br>• Objects: `[{name: "git", version: "1.2.3"}]` |
| `state` | string | No | `present` | Desired state: `present`, `absent`, or `latest` |
| `version` | string | No | - | Global version for all packages (can be overridden per-package) |
| `update_cache` | boolean | No | `false` | Update package cache before operation |

### Name Parameter Formats

The `name` parameter supports three formats:

1. **String** - Single package: `"git"`
2. **List of strings** - Multiple packages: `["git", "curl", "wget"]`
3. **List of objects** - Packages with versions:

   ```yaml
   name:
     - name: "git"
       version: "1:2.43.0-1ubuntu7.3"
     - name: "curl"
       # No version = latest
     - "wget"  # Can mix strings and objects
   ```

## Examples

### Example 1: Single Package Installation

```yaml
---
plays:
  - name: "Install single package"
    hosts: all
    tasks:
      - name: "Install git"
        module:
          type: "package"
          name: "git"
          state: "present"
```

### Example 2: Multiple Packages (List Format)

```yaml
---
plays:
  - name: "Install multiple packages"
    hosts: all
    tasks:
      - name: "Install development tools"
        module:
          type: "package"
          name:
            - "git"
            - "curl"
            - "wget"
            - "vim"
            - "htop"
          state: "present"
```

### Example 3: Package Removal

```yaml
---
plays:
  - name: "Remove packages"
    hosts: all
    tasks:
      - name: "Remove unnecessary packages"
        module:
          type: "package"
          name:
            - "nano"
            - "emacs"
          state: "absent"
```

### Example 4: Install Specific Version (Single Package)

```yaml
---
plays:
  - name: "Install specific version"
    hosts: all
    tasks:
      - name: "Install git with specific version"
        module:
          type: "package"
          name: "git"
          state: "present"
          version: "1:2.43.0-1ubuntu7.3"
```

### Example 4b: Install Multiple Packages with Individual Versions

```yaml
---
plays:
  - name: "Install packages with specific versions"
    hosts: all
    tasks:
      - name: "Install tools with version control"
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

### Example 5: Update to Latest Version

```yaml
---
plays:
  - name: "Update packages"
    hosts: all
    tasks:
      - name: "Update git to latest"
        module:
          type: "package"
          name: "git"
          state: "latest"
```

### Example 6: Update Cache Before Installation

```yaml
---
plays:
  - name: "Install with cache update"
    hosts: all
    tasks:
      - name: "Update cache and install packages"
        module:
          type: "package"
          name:
            - "nginx"
            - "postgresql"
          state: "present"
          update_cache: true
```

### Example 7: Complete Setup (Mixed Operations)

```yaml
---
plays:
  - name: "Complete server setup"
    hosts: all
    tasks:
      # Install essential tools
      - name: "Install essential development tools"
        module:
          type: "package"
          name:
            - "git"
            - "curl"
            - "wget"
            - "tree"
            - "jq"
            - "htop"
            - "vim"
          state: "present"
          update_cache: true

      # Remove unnecessary packages
      - name: "Remove unnecessary packages"
        module:
          type: "package"
          name:
            - "nano"
          state: "absent"

      # Ensure specific package is latest
      - name: "Ensure git is latest version"
        module:
          type: "package"
          name: "git"
          state: "latest"
```

## Output Format

When processing multiple packages, the module returns detailed information for each package:

```json
{
  "packages": {
    "git": {
      "installed": true,
      "current_version": "1:2.43.0-1ubuntu7.3",
      "action": "already_installed",
      "changed": false,
      "package_info": {
        "name": "git",
        "version": "1:2.43.0-1ubuntu7.3",
        "description": "fast, scalable, distributed revision control system"
      }
    },
    "tree": {
      "installed": false,
      "current_version": "",
      "action": "installed",
      "changed": true,
      "package_info": {
        "name": "tree",
        "version": "2.1.1-2ubuntu3",
        "description": "displays an indented directory tree, in color"
      }
    }
  },
  "package_count": 2
}
```

## Idempotency

The module is fully idempotent:

- ✅ Running the same playbook multiple times produces the same result
- ✅ `changed=false` when packages are already in the desired state
- ✅ `changed=true` only when actual changes are made

### Example: Idempotency Test

```bash
# First run - installs missing packages
$ ./onigirazu -playbook install.yml -inventory hosts.yml
# Result: changed=true (packages installed)

# Second run - no changes needed
$ ./onigirazu -playbook install.yml -inventory hosts.yml
# Result: changed=false (packages already installed)
```

## Performance Comparison

### Before (Individual Tasks)

```yaml
# 7 separate tasks = ~23 seconds
- name: "Install git"
  module: { type: "package", name: "git", state: "present" }
- name: "Install curl"
  module: { type: "package", name: "curl", state: "present" }
# ... 5 more tasks
```

### After (Batch Operation)

```yaml
# 1 task with list = ~11 seconds (idempotent run)
- name: "Install all tools"
  module:
    type: "package"
    name: ["git", "curl", "wget", "tree", "jq", "htop", "vim"]
    state: "present"
```

**Performance Improvement:** ~50% faster for idempotent runs!

## Error Handling

The module handles errors gracefully:

- If one package fails, others continue to be processed
- Errors are reported in the output for each package
- The task succeeds if at least one package operation succeeds
- Cache update failures don't fail the entire task

### Example Error Output

```json
{
  "packages": {
    "valid-package": {
      "action": "installed",
      "changed": true
    },
    "invalid-package": {
      "error": "failed to install: package not found"
    }
  }
}
```

## Best Practices

1. **Use Lists for Related Packages**

   ```yaml
   # Good: Group related packages
   name: ["git", "curl", "wget"]

   # Avoid: Separate tasks for related packages
   ```

2. **Update Cache for Fresh Installations**

   ```yaml
   # Good: Update cache on new systems
   update_cache: true

   # Note: Not needed for every run (idempotency)
   ```

3. **Use Descriptive Task Names**

   ```yaml
   # Good
   - name: "Install development tools"

   # Avoid
   - name: "Install packages"
   ```

4. **Group by Purpose**

   ```yaml
   # Good: Separate tasks by purpose
   - name: "Install web server packages"
     module: { type: "package", name: ["nginx", "certbot"] }

   - name: "Install database packages"
     module: { type: "package", name: ["postgresql", "redis"] }
   ```

## Testing

### Verify Installation

```bash
# SSH to remote host
ssh user@host "dpkg -l | grep -E '^ii  (git|curl|wget)'"

# Or use command module in playbook
- name: "Verify git installation"
  module:
    type: "command"
    command: "git --version"
```

### Test Idempotency

```bash
# Run playbook twice
./onigirazu -playbook test.yml -inventory hosts.yml
./onigirazu -playbook test.yml -inventory hosts.yml

# Second run should show: changed=false
```

## Troubleshooting

### Package Not Found

```
Error: failed to install: package not found
```

**Solution:** Check package name spelling and availability in repositories

### Permission Denied

```
Error: failed to install: permission denied
```

**Solution:** Ensure SSH user has sudo privileges

### Cache Update Failed

```
Warning: cache_update_warning: failed to update cache
```

**Solution:** This is a warning, not an error. Check network connectivity.

## Related Modules

- `command` - Run arbitrary commands
- `shell` - Run shell commands
- `apt` - Advanced APT-specific operations
- `yum` - Advanced YUM-specific operations

## See Also

- [01-package-management.yml](01-package-management.yml) - Original example with individual tasks
- [01-package-management-improved.yml](01-package-management-improved.yml) - Improved example with lists
- [inventory-correct.yml](inventory-correct.yml) - Example inventory file
