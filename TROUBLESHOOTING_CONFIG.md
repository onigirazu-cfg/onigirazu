# Configuration Troubleshooting Guide - v1.55.1

**Common Configuration Issues and Solutions**

**Release:** v1.55.1
**Date:** January 29, 2025
**Status:** ✅ Complete

---

## 📋 Table of Contents

1. [Installation Issues](#installation-issues)
2. [Configuration File Not Found](#configuration-file-not-found)
3. [Configuration Errors](#configuration-errors)
4. [Permission Problems](#permission-problems)
5. [Connection Issues](#connection-issues)
6. [Logging Problems](#logging-problems)
7. [Security Issues](#security-issues)
8. [Performance Issues](#performance-issues)
9. [Docker Issues](#docker-issues)
10. [Getting Help](#getting-help)

---

## Installation Issues

### Problem: "command not found: onigirazu"

**Symptoms:**

```bash
$ onigirazu --version
-bash: onigirazu: command not found
```

**Solutions:**

**1. Check if installed correctly:**

```bash
# Linux/macOS
which onigirazu
ls -la /usr/local/bin/onigirazu

# Windows PowerShell
Get-Command onigirazu.exe
```

**2. Add to PATH:**

```bash
# Linux/macOS
export PATH="/usr/local/bin:$PATH"
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Windows PowerShell
$env:Path += ";C:\Program Files\Onigirazu"
[Environment]::SetEnvironmentVariable("Path", $env:Path, "User")
```

**3. Reinstall if needed:**

```bash
# Linux/macOS
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Windows
$ProgressPreference = 'SilentlyContinue'
Invoke-WebRequest -Uri https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Windows_x86_64.zip -OutFile onigirazu.zip
Expand-Archive onigirazu.zip -DestinationPath .
Move-Item onigirazu.exe "$env:ProgramFiles\Onigirazu\" -Force
```

---

### Problem: Installation fails with permission error

**Symptoms:**

```
Permission denied
sudo: onigirazu: command not found
```

**Solutions:**

**1. Use sudo for system-wide install:**

```bash
sudo mv onigirazu /usr/local/bin/
sudo chmod 755 /usr/local/bin/onigirazu
```

**2. Use user-local install (no sudo):**

```bash
mkdir -p ~/.local/bin
mv onigirazu ~/.local/bin/
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

**3. Fix permissions:**

```bash
sudo chown root:root /usr/local/bin/onigirazu
sudo chmod 755 /usr/local/bin/onigirazu
```

---

### Problem: Package installation fails

**Symptoms:**

```
E: Unable to locate package onigirazu
Error 404: Package not found
```

**Solutions:**

**1. Update package lists:**

```bash
# Linux (Debian/Ubuntu)
sudo apt update
sudo apt install onigirazu

# Linux (Red Hat/Fedora)
sudo dnf update
sudo dnf install onigirazu

# macOS
brew update
brew install onigirazu
```

**2. Check if package is available in your region:**

```bash
# Try alternative: direct download
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
```

---

## Configuration File Not Found

### Problem: "Config file not found" error

**Symptoms:**

```
Error: Configuration file not found at /etc/onigirazu/onigirazu.yml
or
Default configuration will be used
```

**Solutions:**

**1. Check current directory:**

```bash
pwd
ls -la onigirazu.yml

# If not in current dir, either:
# - Create it: cp /usr/share/onigirazu/onigirazu.default.yml ./onigirazu.yml
# - Or specify path: onigirazu -c /path/to/config.yml playbook.yml
```

**2. Check home directory:**

```bash
# Linux/macOS
ls -la ~/.onigirazu/onigirazu.yml

# If missing:
mkdir -p ~/.onigirazu
cat > ~/.onigirazu/onigirazu.yml << 'EOF'
max_concurrency: 10
default_timeout: 30s
log_level: info
retry_attempts: 3
retry_delay: 5s
EOF
```

**3. Check system directory:**

```bash
# Linux/macOS
sudo ls -la /etc/onigirazu/onigirazu.yml

# Windows
dir "%ProgramData%\Onigirazu\onigirazu.yml"
```

**4. Use explicit path:**

```bash
# Always works - specify the exact config file
onigirazu -c /usr/share/onigirazu/onigirazu.default.yml playbook.yml

# Or if you know where it is
onigirazu -c ~/my-configs/production.yml playbook.yml
```

---

### Problem: Wrong configuration is being used

**Symptoms:**

```
Expected setting X, but got default value Y
Configuration changes don't take effect
```

**Solutions:**

**1. Check which config is being used:**

```bash
# Add verbose output to see which config is loaded
onigirazu --debug playbook.yml 2>&1 | grep -i config

# Or directly check with explicit path
onigirazu -c /etc/onigirazu/onigirazu.yml --version
```

**2. Verify configuration priority:**

Onigirazu searches in this order:

```
1. CLI flag:        onigirazu -c /explicit/path/config.yml
2. Working dir:     ./onigirazu.yml
3. User home:       ~/.onigirazu/onigirazu.yml
4. System:          /etc/onigirazu/onigirazu.yml
5. Built-in:        Default (compiled in binary)
```

**3. Remove ambiguous configs:**

```bash
# If you have configs in multiple places, remove the ones you don't want:
rm ~/.onigirazu/onigirazu.yml        # User config
rm /etc/onigirazu/onigirazu.yml      # System config

# Or use explicit -c flag to be clear
onigirazu -c /the/exact/config/path/config.yml playbook.yml
```

**4. Verify the config actually exists:**

```bash
# Check file exists and is readable
test -r /etc/onigirazu/onigirazu.yml && echo "File exists" || echo "File not found"

# Show the actual content
cat /etc/onigirazu/onigirazu.yml
```

---

## Configuration Errors

### Problem: "Invalid configuration" or "YAML parse error"

**Symptoms:**

```
Error: YAML parsing failed: unexpected character at line X
Error: Invalid configuration: unknown field 'some_option'
```

**Solutions:**

**1. Check YAML syntax:**

```bash
# Validate YAML file
# Using 'yamllint' if available
yamllint /etc/onigirazu/onigirazu.yml

# Or check manually
cat /etc/onigirazu/onigirazu.yml
```

**2. Common YAML mistakes:**

```yaml
# ❌ WRONG - missing space after colon
max_concurrency:10

# ✅ CORRECT
max_concurrency: 10

# ❌ WRONG - inconsistent indentation
max_concurrency: 10
  default_timeout: 30s  # Extra indent!

# ✅ CORRECT
max_concurrency: 10
default_timeout: 30s

# ❌ WRONG - quotes inside quotes
log_level: "info"  # This is OK, but watch the nesting

# ✅ CORRECT
log_level: info
```

**3. Fix by comparing with template:**

```bash
# Copy a known-good template
cp /usr/share/onigirazu/onigirazu.default.yml /tmp/good-config.yml

# Compare your broken config
diff -u /tmp/good-config.yml /etc/onigirazu/onigirazu.yml

# Or start fresh
cp /usr/share/onigirazu/examples/onigirazu.minimal.yml /etc/onigirazu/onigirazu.yml
```

---

### Problem: "Unknown configuration option"

**Symptoms:**

```
Error: Unknown option 'xyz_setting' at line 10
```

**Solutions:**

**1. Check spelling:**

```yaml
# ❌ WRONG - typo
max_concrrency: 10  # Missing 'u'

# ✅ CORRECT
max_concurrency: 10
```

**2. Verify option is valid:**

```bash
# List all valid options
cat /usr/share/onigirazu/onigirazu.default.yml | grep "^[a-z]" | cut -d: -f1

# Or check documentation
grep "^### " /docs/CONFIGURATION_REFERENCE.md
```

**3. Remove unsupported options:**

```bash
# If you have old config from previous version
# Remove unknown options, or upgrade documentation
nano /etc/onigirazu/onigirazu.yml
```

---

### Problem: "Invalid value for option"

**Symptoms:**

```
Error: max_concurrency must be integer between 1 and 1000, got: "abc"
Error: log_level must be one of: error, warn, info, debug, trace
```

**Solutions:**

**1. Check option value type:**

```bash
# Use proper types in YAML
max_concurrency: 10              # Integer
default_timeout: 30s             # Duration string
log_level: info                  # String enum
enable_caching: true             # Boolean
retry_delay: 5s                  # Duration
cache_ttl: 5m                    # Duration
metrics_port: 9090               # Integer
```

**2. Valid values for enums:**

```yaml
# log_level: must be one of
log_level: error    # ✅
log_level: warn     # ✅
log_level: info     # ✅
log_level: debug    # ✅
log_level: trace    # ✅
log_level: verbose  # ❌ Invalid

# log_format: must be one of
log_format: text    # ✅
log_format: json    # ✅
log_format: xml     # ❌ Invalid
```

**3. Fix values:**

```bash
# Before
max_concurrency: "10"    # String instead of int
default_timeout: 30      # Missing 's' for seconds

# After
max_concurrency: 10      # Correct type
default_timeout: 30s     # Correct format
```

---

## Permission Problems

### Problem: Permission denied when reading config

**Symptoms:**

```
Error: Permission denied: /etc/onigirazu/onigirazu.yml
Error: open /etc/onigirazu/onigirazu.yml: permission denied
```

**Solutions:**

**1. Check file permissions:**

```bash
# Linux/macOS
ls -la /etc/onigirazu/onigirazu.yml

# Should look like:
# -rw-r--r-- 1 root root /etc/onigirazu/onigirazu.yml

# Windows
icacls "%ProgramData%\Onigirazu\onigirazu.yml"
```

**2. Fix permissions:**

```bash
# Linux/macOS - make readable by all
sudo chmod 644 /etc/onigirazu/onigirazu.yml
sudo chmod 755 /etc/onigirazu/

# Windows - grant read permission
icacls "%ProgramData%\Onigirazu" /grant:r "%USERNAME%:F" /T
```

**3. Check ownership:**

```bash
# Should be owned by root or current user
sudo chown root:root /etc/onigirazu/onigirazu.yml

# Or user-owned
sudo chown $USER:$USER ~/.onigirazu/onigirazu.yml
```

**4. Or use readable location:**

```bash
# Create config in user home (no sudo needed)
mkdir -p ~/.onigirazu
cat > ~/.onigirazu/onigirazu.yml << 'EOF'
max_concurrency: 10
default_timeout: 30s
log_level: info
retry_attempts: 3
retry_delay: 5s
EOF

chmod 644 ~/.onigirazu/onigirazu.yml
```

---

### Problem: Permission denied when creating config

**Symptoms:**

```
Cannot create /etc/onigirazu/onigirazu.yml: Permission denied
```

**Solutions:**

**1. Create directory with sudo:**

```bash
sudo mkdir -p /etc/onigirazu
sudo chmod 755 /etc/onigirazu

# Now create config
sudo cat > /etc/onigirazu/onigirazu.yml << 'EOF'
max_concurrency: 10
default_timeout: 30s
EOF

sudo chmod 644 /etc/onigirazu/onigirazu.yml
```

**2. Or create in user directory (no sudo):**

```bash
mkdir -p ~/.onigirazu
cat > ~/.onigirazu/onigirazu.yml << 'EOF'
max_concurrency: 10
default_timeout: 30s
EOF
```

---

## Connection Issues

### Problem: SSH connection times out

**Symptoms:**

```
Error: SSH connection timeout
Error: failed to connect to host
```

**Solutions:**

**1. Increase timeout in config:**

```yaml
# Increase SSH timeout
ssh_timeout: 60s   # Default is 30s

# Increase task timeout
default_timeout: 5m  # Longer for slow networks
```

**2. Check SSH connectivity manually:**

```bash
# Test SSH connection
ssh -v ubuntu@192.168.1.100

# Check if SSH port is open
telnet 192.168.1.100 22
nc -zv 192.168.1.100 22

# Ping to check network
ping 192.168.1.100
```

**3. Check SSH configuration:**

```yaml
# In onigirazu.yml
ssh_port: 22
ssh_user: ubuntu
ssh_key_file: ~/.ssh/id_rsa
ssh_strict_host_key_check: true

# Verify these match your setup
```

---

### Problem: SSH key not found or permission denied

**Symptoms:**

```
Error: SSH key not found
Error: Permission denied (publickey)
```

**Solutions:**

**1. Check SSH key exists:**

```bash
# List available keys
ls -la ~/.ssh/

# Check specific key
test -f ~/.ssh/id_rsa && echo "Key exists" || echo "Key missing"
```

**2. Fix SSH key permissions:**

```bash
# Private key should be readable by user only
chmod 600 ~/.ssh/id_rsa

# Public key should be readable by all
chmod 644 ~/.ssh/id_rsa.pub

# SSH directory permissions
chmod 700 ~/.ssh
```

**3. Configure key in Onigirazu:**

```yaml
# In onigirazu.yml
ssh_key_file: ~/.ssh/id_rsa
ssh_key_pass: ""  # Leave empty or provide password

# Or use SSH agent (recommended)
# Set up SSH agent, no need to specify key_pass
```

**4. Generate new key if missing:**

```bash
# Generate new SSH key
ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_rsa

# Add to remote authorized_keys
ssh-copy-id -i ~/.ssh/id_rsa.pub ubuntu@192.168.1.100
```

---

## Logging Problems

### Problem: Logs not being generated

**Symptoms:**

```
No log files in /var/log/onigirazu/
Logs not appearing anywhere
```

**Solutions:**

**1. Check log configuration:**

```yaml
# Verify these are set:
log_level: info
log_format: text    # or json

# Check if enable_audit is needed for audit logs
enable_audit: true
```

**2. Check log directory permissions:**

```bash
# Directory should exist and be writable
sudo mkdir -p /var/log/onigirazu
sudo chmod 755 /var/log/onigirazu

# Or user-writable directory
mkdir -p ~/.onigirazu/logs
```

**3. Check if logs are in different location:**

```bash
# Search for recent log files
find ~/ -name "*.log" -mmin -5 2>/dev/null

# Check if using JSON logs
ls -la /var/log/onigirazu/
```

**4. Test logging:**

```bash
# Run with verbose output
onigirazu -c onigirazu.yml --debug playbook.yml 2>&1 | head -20
```

---

### Problem: JSON logs not formatted correctly

**Symptoms:**

```
Log lines are plain text, not JSON
Elasticsearch won't parse logs
```

**Solutions:**

**1. Verify JSON logging enabled:**

```yaml
# Must be set to 'json'
log_format: json    # ✅

# NOT
log_format: text    # ❌
```

**2. Check logs are actually JSON:**

```bash
# Good JSON log
tail /var/log/onigirazu/onigirazu.log | head -1 | jq .

# If jq fails, logs aren't JSON
```

**3. Reconfigure and restart:**

```bash
# Update config
sudo cat > /etc/onigirazu/onigirazu.yml << 'EOF'
log_format: json
log_level: info
enable_audit: true
EOF

# Run again
onigirazu playbook.yml

# Verify
tail /var/log/onigirazu/*.log | jq .
```

---

## Security Issues

### Problem: "Security validation failed"

**Symptoms:**

```
Error: security validation failed: File path validation failed
Error: security policy blocked this operation
```

**Solutions:**

**1. Check security policy:**

```bash
# View current policy
cat /etc/onigirazu/security-policy.json | jq .

# Or use minimal one
sudo cp /usr/share/onigirazu/examples/onigirazu.minimal.yml /etc/onigirazu/
```

**2. Understand the restriction:**

```json
{
  "blocked_directories": [
    "/etc/passwd",    // Never allow
    "/boot",
    "/sys",
    "/proc"
  ],
  "allowed_directories": [
    "/opt",
    "/srv",
    "/var/lib"
  ]
}
```

**3. Solution options:**

**A) Update security policy (if admin):**

```bash
sudo cat > /etc/onigirazu/security-policy.json << 'EOF'
{
  "allow_shell_commands": false,
  "allowed_directories": [
    "/opt",
    "/srv",
    "/var/lib",
    "/home",
    "/tmp"  # Add if needed
  ],
  "blocked_directories": [
    "/etc/passwd",
    "/etc/shadow",
    "/boot"
  ]
}
EOF
```

**B) Use allowed module operations:**
Instead of shell commands that get blocked, use modules:

```yaml
# ❌ Blocked shell command
- shell: rm /file.txt

# ✅ Use file module instead
- file:
    path: /file.txt
    state: absent
```

---

### Problem: "Command blocked by security policy"

**Symptoms:**

```
Error: Command 'rm -rf' is blocked by security policy
```

**Solutions:**

**1. Check blocked commands:**

```bash
cat /etc/onigirazu/security-policy.json | jq '.command_blocklist'
```

**2. Use modules instead of shell:**

```yaml
# ❌ Shell command (blocked)
- shell: rm -rf /tmp/old

# ✅ Use file module
- file:
    path: /tmp/old
    state: absent
    force: yes
```

**3. If truly needed, update policy:**

```bash
# Edit security policy
sudo nano /etc/onigirazu/security-policy.json

# Remove command from blocklist if appropriate
```

---

## Performance Issues

### Problem: Playbook runs slowly

**Symptoms:**

```
Playbook taking much longer than expected
High CPU or memory usage
```

**Solutions:**

**1. Increase parallelism:**

```yaml
# Increase concurrent execution
max_concurrency: 50    # From default 10

# Make sure not limited by task timeout
default_timeout: 5m
```

**2. Enable caching:**

```yaml
# Cache results to avoid re-fetching
enable_caching: true
cache_ttl: 10m    # Cache for 10 minutes
```

**3. Check for bottlenecks:**

```bash
# Run with debug output
onigirazu --debug playbook.yml 2>&1 | tail -50

# Look for slow tasks:
# - Are some tasks taking much longer?
# - Is network the bottleneck?
# - Is CPU maxed?
```

**4. Reduce logging verbosity:**

```yaml
# High verbosity slows execution
log_level: warn    # Reduce from 'debug' or 'trace'
```

---

### Problem: Out of memory errors

**Symptoms:**

```
Error: Cannot allocate memory
Process killed by OOM killer
```

**Solutions:**

**1. Reduce parallelism:**

```yaml
# Each concurrent task uses memory
max_concurrency: 5    # Reduce from 50

# Also limit SSH connections
ssh_pool_size: 10     # Limit connection pool
```

**2. Disable caching:**

```yaml
# Caching uses memory
enable_caching: false

# Or use smaller TTL
cache_ttl: 1m    # Shorter cache lifetime
```

**3. Monitor memory:**

```bash
# Watch memory during execution
while true; do free -h; sleep 1; done

# Or use top
top -b -n 1 | head -n 15
```

---

## Docker Issues

### Problem: "Config file not found" in Docker

**Symptoms:**

```
Error: Configuration file not found at /etc/onigirazu/onigirazu.yml
```

**Solutions:**

**1. Mount config volume:**

```bash
docker run \
  -v $(pwd)/onigirazu.yml:/etc/onigirazu/onigirazu.yml \
  -v $(pwd)/playbook.yml:/playbooks/playbook.yml \
  onigirazu/onigirazu \
  -c /etc/onigirazu/onigirazu.yml \
  /playbooks/playbook.yml
```

**2. Or use ConfigMap in Kubernetes:**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: onigirazu-config
data:
  onigirazu.yml: |
    max_concurrency: 10
    default_timeout: 30s
---
apiVersion: batch/v1
kind: Job
metadata:
  name: onigirazu-job
spec:
  template:
    spec:
      containers:
      - name: onigirazu
        image: onigirazu/onigirazu:latest
        volumeMounts:
        - name: config
          mountPath: /etc/onigirazu
      volumes:
      - name: config
        configMap:
          name: onigirazu-config
```

---

### Problem: "Permission denied" in Docker

**Symptoms:**

```
Error: Permission denied: /playbooks/playbook.yml
```

**Solutions:**

**1. Mount with proper permissions:**

```bash
# Make sure files are readable
chmod 644 playbook.yml
chmod 755 .

# Then mount
docker run -v $(pwd):/playbooks onigirazu/onigirazu playbook.yml
```

**2. Or run with explicit user:**

```bash
docker run \
  --user 1000:1000 \
  -v $(pwd):/playbooks \
  onigirazu/onigirazu playbook.yml
```

---

## Getting Help

### Information to Gather

Before asking for help, gather this information:

```bash
# 1. Version
onigirazu --version

# 2. Config file location
cat /etc/onigirazu/onigirazu.yml

# 3. Error output
onigirazu playbook.yml 2>&1 | tail -50

# 4. Debug output
onigirazu --debug playbook.yml 2>&1 | tail -100

# 5. System info
uname -a
os_version  # varies by OS

# 6. Network info (for connection issues)
ssh -v ubuntu@remote-host

# 7. File permissions (for permission issues)
ls -la /etc/onigirazu/
```

### Support Resources

- 📖 **Documentation**: <https://github.com/onigirazu-cfg/onigirazu/docs/>
- 🐛 **Issues**: <https://github.com/onigirazu-cfg/onigirazu/issues>
- 💬 **Discussions**: <https://github.com/onigirazu-cfg/onigirazu/discussions>
- 🔗 **Reference**: See [CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md)
- 📋 **Setup Guide**: See [CONFIGURATION_SETUP_GUIDE.md](CONFIGURATION_SETUP_GUIDE.md)

---

## Quick Diagnostic Script

```bash
#!/bin/bash
# Run this to diagnose Onigirazu configuration issues

echo "=== Onigirazu Diagnostic Report ==="
echo ""

echo "1. Installation Check:"
which onigirazu && echo "✓ Installed" || echo "✗ Not found"
onigirazu --version

echo ""
echo "2. Configuration Files:"
test -f /etc/onigirazu/onigirazu.yml && echo "✓ System config exists" || echo "✗ Missing"
test -f ~/.onigirazu/onigirazu.yml && echo "✓ User config exists" || echo "✗ Missing"
test -f ./onigirazu.yml && echo "✓ Project config exists" || echo "✗ Missing"

echo ""
echo "3. Directory Permissions:"
ls -ld /etc/onigirazu/ 2>/dev/null
ls -ld ~/.onigirazu/ 2>/dev/null

echo ""
echo "4. Config File Validation:"
if [ -f /etc/onigirazu/onigirazu.yml ]; then
    head -5 /etc/onigirazu/onigirazu.yml
fi

echo ""
echo "5. SSH Connectivity (optional - provide host):"
[ -n "$SSH_HOST" ] && ssh -v $SSH_HOST "echo OK" || echo "Skipped"

echo ""
echo "=== End Diagnostic Report ==="
```

Save as `diagnose.sh` and run: `bash diagnose.sh`
