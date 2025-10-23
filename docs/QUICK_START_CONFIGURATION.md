# Quick Start: Onigirazu Configuration Setup - v1.52.0

**Get Onigirazu running in 5 minutes - with proper security configuration**

**Release:** v1.52.0
**Date:** January 29, 2025
**Setup Time:** 5 minutes ⏱️

---

## 🚀 Setup for filetest.yml (and All Playbooks)

### Problem

```
[WARN] Task failed: security validation failed: File path validation failed:
path is not in allowed directories
```

### Solution

Security policies run on your **control machine** (where you run `onigirazu`). You need to allow `/tmp` for file operations.

---

## Step 1: Create Security Policy File

**On your control machine**, create `/etc/onigirazu/security-policy.json`:

```bash
sudo mkdir -p /etc/onigirazu
sudo cat > /etc/onigirazu/security-policy.json << 'EOF'
{
  "allowed_hosts": [],
  "allowed_ports": [],
  "allowed_modules": [],
  "blocked_commands": [
    "rm -rf",
    "format",
    "mkfs",
    "dd if=",
    ":(){ :|:& };:",
    "shutdown",
    "reboot",
    "halt"
  ],
  "allowed_directories": [
    "/tmp",
    "/var/tmp",
    "/home",
    "/opt"
  ],
  "blocked_directories": [
    "/etc/passwd",
    "/etc/shadow",
    "/boot",
    "/sys",
    "/proc"
  ],
  "allowed_file_types": [],
  "max_file_size": 104857600,
  "require_encryption": false,
  "audit_enabled": true,
  "log_level": "info"
}
EOF
sudo chmod 644 /etc/onigirazu/security-policy.json
```

---

## Step 2: Create Main Configuration File

**On your control machine**, create `/etc/onigirazu/onigirazu.yml`:

```bash
sudo cat > /etc/onigirazu/onigirazu.yml << 'EOF'
# Onigirazu main configuration

# Execution
max_concurrency: 10
default_timeout: 30s
retry_attempts: 3
retry_delay: 5s

# Logging
log_level: info
log_format: text

# Performance
enable_caching: true
cache_ttl: 5m
enable_checksum: true
enable_parallel: false

# Development settings
color_output: true
progress_bar: true
verbose: false
show_diff: true

# SSH
ssh_timeout: 30s
ssh_keepalive: 60s
ssh_max_sessions: 10
connection_reuse: true
ssh_strict_host_key: false
default_insecure_ignore_host_key: false
EOF
sudo chmod 644 /etc/onigirazu/onigirazu.yml
```

---

## Step 3: Run Your Playbook

Now you can run playbooks like `filetest.yml`:

```bash
# Run with automatic config discovery
onigirazu -i inventory_local.txt filetest.yml

# Or explicitly specify the config file
onigirazu -c /etc/onigirazu/onigirazu.yml -i inventory_local.txt filetest.yml

# Run specific phases
onigirazu -i inventory_local.txt filetest.yml --tags create
onigirazu -i inventory_local.txt filetest.yml --tags verify
onigirazu -i inventory_local.txt filetest.yml -e cleanup_enabled=true
```

---

## Configuration Location Priority

Onigirazu automatically searches for configuration in this order (on control machine):

1. **Explicitly specified:**

   ```bash
   onigirazu -c /path/to/onigirazu.yml playbook.yml
   ```

2. **Playbook directory:**

   ```
   ./onigirazu.yml
   ```

3. **System directory:**

   ```
   /etc/onigirazu/onigirazu.yml
   ```

4. **Built-in defaults** (if no file found)

---

## Common Configuration Scenarios

### Development Workstation

**`~/.onigirazu/onigirazu.yml`:**

```yaml
max_concurrency: 5
log_level: debug
ssh_strict_host_key: false
default_insecure_ignore_host_key: true
enable_caching: false
color_output: true
show_diff: true
```

**`security-policy.json`:**

```json
{
  "allowed_hosts": [],
  "allowed_directories": ["/tmp", "/home", "/opt", "/srv"],
  "blocked_commands": [],
  "audit_enabled": false,
  "log_level": "debug"
}
```

### Production Server

**`/etc/onigirazu/onigirazu.yml`:**

```yaml
max_concurrency: 20
log_level: warn
log_format: json
ssh_strict_host_key: true
enable_metrics: true
retry_attempts: 5
enable_caching: true
```

**`/etc/onigirazu/security-policy.json`:**

```json
{
  "allowed_hosts": ["10.0.0.0/8"],
  "allowed_ports": [22, 443],
  "allowed_directories": ["/opt/app", "/var/log", "/var/backup"],
  "blocked_commands": ["rm -rf", "mkfs", "reboot", "shutdown"],
  "blocked_file_types": [".exe", ".bat", ".dll"],
  "require_encryption": true,
  "audit_enabled": true,
  "log_level": "info"
}
```

### CI/CD Pipeline

**`onigirazu.yml` (in repo root):**

```yaml
max_concurrency: 50
log_level: info
log_format: json
color_output: false
progress_bar: false
output_format: json
enable_caching: true
dry_run: false
check_mode: false
```

---

## Environment Variable Configuration

You can override config file settings with environment variables:

```bash
# Set via environment
export ONIGIRAZU_MAX_CONCURRENCY=20
export ONIGIRAZU_LOG_LEVEL=debug
export ONIGIRAZU_DRY_RUN=true

# Run playbook - will use env vars
onigirazu playbook.yml

# Or inline:
ONIGIRAZU_LOG_LEVEL=debug onigirazu playbook.yml
```

All available environment variables:

- `ONIGIRAZU_MAX_CONCURRENCY`
- `ONIGIRAZU_TIMEOUT`
- `ONIGIRAZU_RETRY_ATTEMPTS`
- `ONIGIRAZU_RETRY_DELAY`
- `ONIGIRAZU_LOG_LEVEL`
- `ONIGIRAZU_LOG_FORMAT`
- `ONIGIRAZU_ALLOW_SHELL`
- `ONIGIRAZU_ENABLE_CACHE`
- `ONIGIRAZU_CACHE_TTL`
- `ONIGIRAZU_ENABLE_CHECKSUM`
- `ONIGIRAZU_ENABLE_PARALLEL`
- `ONIGIRAZU_PARALLEL_STRATEGY`
- `ONIGIRAZU_DRY_RUN`
- `ONIGIRAZU_CHECK_MODE`
- `ONIGIRAZU_VERBOSE`
- `ONIGIRAZU_SHOW_DIFF`
- `ONIGIRAZU_COLOR_OUTPUT`
- `ONIGIRAZU_PROGRESS_BAR`
- `ONIGIRAZU_INTERACTIVE`
- `ONIGIRAZU_OUTPUT_FORMAT`
- `ONIGIRAZU_SSH_TIMEOUT`
- `ONIGIRAZU_SSH_KEEPALIVE`
- `ONIGIRAZU_SSH_MAX_SESSIONS`
- `ONIGIRAZU_CONNECTION_REUSE`
- `ONIGIRAZU_SSH_STRICT_HOST_KEY`
- `ONIGIRAZU_SSH_KNOWN_HOSTS_FILE`
- `ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY`
- `ONIGIRAZU_VAULT_ENABLED`
- `ONIGIRAZU_VAULT_ADDRESS`
- `ONIGIRAZU_VAULT_TOKEN`
- `ONIGIRAZU_SECURITY_POLICY`

---

## Troubleshooting

### "path is not in allowed directories"

**Fix:** Add directory to `allowed_directories` in security-policy.json

```json
{
  "allowed_directories": [
    "/tmp",
    "/home",
    "/opt",
    "/your/directory/here"
  ]
}
```

### "command not allowed"

**Fix:** Remove from `blocked_commands` or allow with permission

```json
{
  "blocked_commands": [
    "rm -rf /",
    "mkfs",
    "format"
  ]
}
```

### "module not allowed"

**Fix:** Add to `allowed_modules` (if whitelist mode)

```json
{
  "allowed_modules": [
    "copy",
    "file",
    "your_module_here"
  ]
}
```

### "host not allowed"

**Fix:** Add to `allowed_hosts`

```json
{
  "allowed_hosts": [
    "192.168.1.*",
    "your-host.example.com"
  ]
}
```

### Config not being loaded

**Check discovery order:**

```bash
# 1. Create in playbook directory
./onigirazu.yml

# 2. Or system directory
/etc/onigirazu/onigirazu.yml

# 3. Or specify explicitly
onigirazu -c /path/to/onigirazu.yml playbook.yml
```

---

## File Organization

**Recommended structure:**

```
# Development
my-project/
├── onigirazu.yml              # Project config
├── security-policy.json       # Project security
├── playbooks/
│   ├── deploy.yml
│   ├── maintenance.yml
│   └── filetest.yml
└── inventory/
    ├── production
    └── staging

# Production
/etc/onigirazu/
├── onigirazu.yml              # System config
├── security-policy.json       # System security
└── inventory/                 # System inventory
    ├── production
    ├── staging
    └── development
```

---

## Next Steps

- 📖 Read [CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md) for all options
- 🔒 Read [SECURITY_POLICY_GUIDE.md](SECURITY_POLICY_GUIDE.md) for security details
- 🔄 Read [LOOPS_GUIDE.md](LOOPS_GUIDE.md) for loop operations
- 📝 Check examples in `onigirazu/examples/` directory

---

## Example Playbooks

Now these playbooks work with proper configuration:

```bash
# Stress test with 1000+ files
onigirazu -i inventory_local.txt filetest.yml

# Deploy application
onigirazu -i inventory_production playbooks/deploy.yml

# Run in check mode (no changes)
onigirazu -i inventory playbooks/deploy.yml --check

# Dry run
onigirazu -i inventory playbooks/deploy.yml --dry-run

# Verbose debug output
ONIGIRAZU_LOG_LEVEL=debug onigirazu -i inventory playbooks/deploy.yml
```

---

## Additional Resources

- Main Configuration: [CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md)
- Security Policies: [SECURITY_POLICY_GUIDE.md](SECURITY_POLICY_GUIDE.md)
- Loop Examples: [LOOPS_GUIDE.md](LOOPS_GUIDE.md)
- Module Development: [MODULE_DEVELOPMENT_GUIDE.md](MODULE_DEVELOPMENT_GUIDE.md)
