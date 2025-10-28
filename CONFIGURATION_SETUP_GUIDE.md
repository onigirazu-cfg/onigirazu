# Configuration Setup Guide - v1.55.1

**Step-by-Step Setup for Different Scenarios**

**Release:** v1.55.1
**Date:** January 29, 2025
**Status:** ✅ Complete

---

## 📋 Table of Contents

1. [Quick Setup (5 minutes)](#quick-setup-5-minutes)
2. [Development Environment](#development-environment)
3. [Production Environment](#production-environment)
4. [CI/CD Pipeline](#cicd-pipeline)
5. [Embedded Systems](#embedded-systems)
6. [Team/Enterprise Setup](#teamenterise-setup)
7. [Docker & Kubernetes](#docker--kubernetes)
8. [Security Hardening](#security-hardening)
9. [Performance Tuning](#performance-tuning)
10. [Monitoring & Audit](#monitoring--audit)

---

## Quick Setup (5 minutes)

### For Immediate Testing

```bash
# 1. Install
sudo apt install onigirazu  # or brew install, scoop install, etc.

# 2. Done! Use default config
onigirazu --version

# 3. Run a test
onigirazu playbook.yml
```

**That's it!** Onigirazu works with zero configuration. The rest of this guide covers customization.

---

## Development Environment

### Scenario

You're developing locally, testing playbooks, and want quick feedback without restrictions.

### Setup (5 minutes)

**Step 1: Create project config**

```bash
# In your project directory
mkdir -p ~/projects/myapp
cd ~/projects/myapp

# Create minimal config
cat > onigirazu.yml << 'EOF'
# Development configuration
max_concurrency: 1           # Single-threaded for debugging
default_timeout: 60s         # Longer timeout for testing
log_level: debug             # Verbose output for debugging
log_format: text             # Human-readable
retry_attempts: 1            # Quick fail for debugging
color_output: true           # Colors for readability
progress_bar: true           # Visual feedback
show_diff: true              # See what changed
verbose: true                # Maximum detail
enable_caching: false        # Fresh data every run
EOF
```

**Step 2: Create playbooks**

```bash
# Create test playbook
cat > test.yml << 'EOF'
---
- hosts: localhost
  gather_facts: yes
  tasks:
    - name: Hello World
      debug:
        msg: "Hello from development!"
EOF

# Test it
onigirazu test.yml
```

**Step 3: Set up inventory**

```bash
# Create inventory
cat > inventory.yml << 'EOF'
---
all:
  hosts:
    localhost:
      ansible_connection: local
    testvm:
      ansible_host: 192.168.1.100
      ansible_user: ubuntu
      ansible_ssh_key_file: ~/.ssh/id_rsa
EOF

# Run with inventory
onigirazu -i inventory.yml test.yml
```

### Development Features

✅ **Enabled by default:**

- Debug logging
- Colored output
- Progress bars
- Diff display
- Single-threaded execution
- Verbose mode

❌ **Disabled for development:**

- Caching (want fresh data)
- SSH host key checking (for quick local testing)

### Development Workflow

```bash
# 1. Write playbook
nano playbook.yml

# 2. Run with local config
onigirazu playbook.yml

# 3. See full debug info
# Fix issues
# Repeat

# When ready to test on remote
# Switch to production config
onigirazu -c /etc/onigirazu/production.yml playbook.yml
```

---

## Production Environment

### Scenario

Deploying to production servers with security, auditing, and monitoring.

### Setup (10 minutes)

**Step 1: System-wide configuration**

```bash
# 1. Create production config
sudo cat > /etc/onigirazu/onigirazu.yml << 'EOF'
# Production configuration - v1.55.1

# EXECUTION
max_concurrency: 50          # Scale to handle load
default_timeout: 5m          # Longer timeout for prod
retry_attempts: 3            # Retry failed tasks
retry_delay: 10s             # Wait between retries

# LOGGING (for log aggregation)
log_level: info              # Info level for production
log_format: json             # JSON for ELK/Splunk
enable_audit: true           # Track all operations

# SECURITY
allow_shell_commands: false  # Only use modules
ssh_strict_host_key_check: true  # Verify host keys
vault_enabled: true          # Use Vault for secrets
vault_addr: "https://vault.corp.internal:8200"

# PERFORMANCE
enable_caching: true         # Cache for speed
cache_ttl: 5m
enable_checksum: true        # Verify integrity
enable_metrics: true         # Prometheus metrics
metrics_port: 9090

# UI (disable for non-interactive)
color_output: false          # No colors
progress_bar: false          # No progress
verbose: false               # No noise
show_diff: true              # Important for audit
EOF

# 2. Set permissions
sudo chmod 644 /etc/onigirazu/onigirazu.yml
```

**Step 2: Security policy**

```bash
# Create security policy
sudo cat > /etc/onigirazu/security-policy.json << 'EOF'
{
  "allow_shell_commands": false,
  "command_blocklist": [
    "rm -rf",
    "dd if=",
    "mkfs",
    "format",
    "shutdown",
    "reboot",
    "halt",
    ":(){:|:&};:"
  ],
  "allowed_directories": [
    "/opt",
    "/srv",
    "/var/lib",
    "/home"
  ],
  "blocked_directories": [
    "/etc/passwd",
    "/etc/shadow",
    "/boot",
    "/sys",
    "/proc",
    "/dev"
  ],
  "require_encryption": true,
  "audit_enabled": true,
  "log_level": "info"
}
EOF

# Set permissions
sudo chmod 644 /etc/onigirazu/security-policy.json
```

**Step 3: Vault integration**

```bash
# Get Vault token (from your Vault admin)
# Store in secure location
echo "s.xxxxxxxxxxxxxxxx" > ~/.vault-token
chmod 600 ~/.vault-token

# Update Onigirazu config with your Vault
sudo tee -a /etc/onigirazu/onigirazu.yml << 'EOF'
vault_token_path: "~/.vault-token"
vault_namespace: "onigirazu"
EOF
```

**Step 4: Log aggregation setup**

```bash
# For ELK Stack
# 1. JSON logs are automatically sent to Elasticsearch
# 2. Configure Filebeat to read from /var/log/onigirazu/

# For Datadog
# Add to Datadog Agent config
cat >> /etc/datadog-agent/conf.d/onigirazu.d/conf.yaml << 'EOF'
logs:
  - type: file
    path: /var/log/onigirazu/*.log
    service: onigirazu
    source: onigirazu
EOF
```

**Step 5: Monitoring setup**

```bash
# Prometheus scrape config
sudo cat >> /etc/prometheus/prometheus.yml << 'EOF'
  - job_name: 'onigirazu'
    static_configs:
      - targets: ['localhost:9090']
EOF

# Restart Prometheus
sudo systemctl restart prometheus
```

### Production Features

✅ **Enabled:**

- JSON logging for aggregation
- Vault integration
- Audit logging
- Metrics/Prometheus
- Security policies
- SSH host key verification
- Retry logic
- Caching

❌ **Disabled:**

- Colors (not needed)
- Progress bars (server environment)
- Verbose output (reduces noise)
- Shell commands (security)

### Production Workflow

```bash
# 1. Write and test locally
onigirazu -c development.yml playbook.yml

# 2. Deploy to staging
onigirazu -c staging.yml playbook.yml

# 3. Deploy to production (monitored)
onigirazu -c /etc/onigirazu/onigirazu.yml playbook.yml

# 4. Check audit logs
tail -f /var/log/onigirazu/audit.json | jq .

# 5. View metrics
curl http://localhost:9090/metrics | grep onigirazu
```

---

## CI/CD Pipeline

### Scenario

Running Onigirazu in GitHub Actions, GitLab CI, Jenkins, etc.

### Setup (5 minutes)

**GitHub Actions Example**

```yaml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Install Onigirazu
        run: |
          curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
          tar -xzf onigirazu_Linux_x86_64.tar.gz
          sudo mv onigirazu /usr/local/bin/

      - name: Run Playbook
        run: |
          onigirazu \
            -c examples/onigirazu.minimal.yml \
            -i ${{ secrets.INVENTORY }} \
            deploy.yml

      - name: Show Results
        if: always()
        run: |
          echo "Deployment completed"
```

**CI/CD Configuration**

```yaml
# ci-cd.yml - minimal for fast execution
max_concurrency: 5           # Don't overwhelm CI environment
default_timeout: 5m          # Generous timeout
log_level: info              # Just the important stuff
log_format: json             # Easy to parse
retry_attempts: 2            # Quick retry
retry_delay: 5s              # Fast retry
color_output: false          # CI doesn't need colors
progress_bar: false          # No progress
enable_caching: false        # Fresh runs
enable_metrics: false        # No metrics in CI
enable_audit: false          # No audit needed
```

**GitLab CI Example**

```yaml
deploy:
  stage: deploy
  image: ubuntu:latest
  script:
    - apt-get update && apt-get install -y curl
    - curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
    - tar -xzf onigirazu_Linux_x86_64.tar.gz
    - ./onigirazu -c ci-cd.yml playbook.yml
  only:
    - main
```

### CI/CD Features

✅ **Enabled:**

- Minimal config for speed
- JSON output
- Quick retries
- Timeout protection

❌ **Disabled:**

- Caching (fresh runs)
- Metrics
- Interactive features
- Colors

---

## Embedded Systems

### Scenario

Running on IoT devices, Raspberry Pi, constrained hardware.

### Setup (3 minutes)

```bash
# Use the minimal template - smallest footprint

# 1. Copy minimal config
sudo cp /usr/share/onigirazu/examples/onigirazu.minimal.yml /etc/onigirazu/onigirazu.yml

# 2. Verify it's tiny
wc -c /etc/onigirazu/onigirazu.yml
# Output: 356 onigirazu.yml

# 3. Test
onigirazu playbook.yml
```

**Why minimal for embedded?**

| Metric | Default | Minimal | Savings |
|--------|---------|---------|---------|
| **Size** | 7.9 KB | 356 B | 95% smaller |
| **Load time** | 50ms | 5ms | 10x faster |
| **Memory** | 2 MB | 0.5 MB | 4x less |
| **Options** | 30 | 5 | Simpler |

### Embedded Optimization

```yaml
# Minimal optimized for Pi/IoT
max_concurrency: 1          # Single core
default_timeout: 2m         # Slow network
log_level: warn             # No info spam
retry_attempts: 1           # Retry once
retry_delay: 2s             # Shorter retry
enable_caching: true        # Cache to save CPU
cache_ttl: 10m              # Longer cache
enable_parallel: false      # Single core
```

---

## Team/Enterprise Setup

### Scenario

Multiple teams, shared infrastructure, centralized management.

### Setup (15 minutes)

**Step 1: Centralized configuration repository**

```bash
# Create config repo
git clone ssh://git@github.com/company/onigirazu-config.git
cd onigirazu-config

# Directory structure
.
├── base/                     # Base configs
│   ├── onigirazu.yml
│   └── security-policy.json
├── teams/
│   ├── platform/
│   │   ├── onigirazu.yml   # Platform team config
│   │   └── security-policy.json
│   ├── backend/
│   │   ├── onigirazu.yml   # Backend team config
│   │   └── inventory.yml
│   └── devops/
│       └── onigirazu.yml
├── environments/
│   ├── dev/onigirazu.yml
│   ├── staging/onigirazu.yml
│   └── prod/onigirazu.yml
└── README.md

git add .
git commit -m "Initial Onigirazu config"
git push origin main
```

**Step 2: Shared configuration**

```bash
# /etc/onigirazu/onigirazu.yml - base/shared settings
cat > /etc/onigirazu/onigirazu.yml << 'EOF'
# Company-wide settings
max_concurrency: 10
default_timeout: 5m
log_level: info
log_format: json

# Shared Vault
vault_enabled: true
vault_addr: "https://vault.company.internal:8200"

# Company audit
enable_audit: true
audit_enabled: true
EOF

sudo chmod 644 /etc/onigirazu/onigirazu.yml
```

**Step 3: Team-specific overrides**

```bash
# Platform team config at: /var/lib/onigirazu/teams/platform/onigirazu.yml
sudo mkdir -p /var/lib/onigirazu/teams/platform

sudo cat > /var/lib/onigirazu/teams/platform/onigirazu.yml << 'EOF'
# Inherit base config, override for platform team
max_concurrency: 50        # Platform handles more
enable_metrics: true       # Platform monitors everything
metrics_port: 9090
EOF

sudo chmod 644 /var/lib/onigirazu/teams/platform/onigirazu.yml
```

**Step 4: Usage by teams**

```bash
# Platform team uses their config
onigirazu -c /var/lib/onigirazu/teams/platform/onigirazu.yml \
  platform-deploy.yml

# Backend team uses their config
onigirazu -c /var/lib/onigirazu/teams/backend/onigirazu.yml \
  backend-deploy.yml

# Default (company-wide) for others
onigirazu deploy.yml
```

### Enterprise Features

✅ **Enabled:**

- Centralized management
- Team overrides
- Shared Vault
- Company audit logs
- Environment-specific configs
- Version control

---

## Docker & Kubernetes

### Docker Setup

```bash
# 1. Create config
cat > docker-config.yml << 'EOF'
# Docker-specific configuration
max_concurrency: 5
default_timeout: 30s
log_level: info
log_format: json
color_output: false
progress_bar: false
enable_interactive: false
EOF

# 2. Run container
docker run \
  -v $(pwd)/docker-config.yml:/etc/onigirazu/onigirazu.yml \
  -v $(pwd)/playbook.yml:/playbooks/playbook.yml \
  onigirazu/onigirazu \
  -c /etc/onigirazu/onigirazu.yml \
  /playbooks/playbook.yml
```

### Kubernetes Setup

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: onigirazu-config
data:
  onigirazu.yml: |
    max_concurrency: 10
    default_timeout: 30s
    log_level: info
    log_format: json
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: onigirazu-sync
spec:
  schedule: "0 */4 * * *"  # Every 4 hours
  jobTemplate:
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
          restartPolicy: OnFailure
```

---

## Security Hardening

### Scenario

Maximum security with all protections enabled.

### Setup (10 minutes)

```bash
# 1. Create hardened config
sudo cat > /etc/onigirazu/onigirazu.yml << 'EOF'
# SECURITY HARDENED CONFIGURATION

# Execution
max_concurrency: 5           # Lower for control
default_timeout: 30s         # Short timeout
retry_attempts: 1            # Limited retries

# SECURITY - Everything restrictive
allow_shell_commands: false  # No shell
ssh_strict_host_key_check: true  # Verify hosts
ssh_key_pass: ""             # No hardcoded passwords

# Vault (for all secrets)
vault_enabled: true
vault_addr: "https://vault.company.internal:8200"

# Logging and Audit
log_level: debug             # Maximum detail
log_format: json             # For aggregation
enable_audit: true           # Track all operations

# Metrics for security monitoring
enable_metrics: true
metrics_port: 9090
EOF
```

```bash
# 2. Create restrictive security policy
sudo cat > /etc/onigirazu/security-policy.json << 'EOF'
{
  "allow_shell_commands": false,
  "command_blocklist": [
    "sudo",
    "su",
    "chmod",
    "chown",
    "rm -rf",
    "dd",
    "mkfs",
    "shutdown",
    "reboot",
    "halt"
  ],
  "allowed_directories": [
    "/opt",
    "/srv"
  ],
  "blocked_directories": [
    "/etc/passwd",
    "/etc/shadow",
    "/boot",
    "/sys",
    "/proc",
    "/dev",
    "/root"
  ],
  "allowed_modules": [
    "package",
    "service",
    "file",
    "debug"
  ],
  "require_encryption": true,
  "audit_enabled": true,
  "max_file_size": 10485760,
  "log_level": "debug"
}
EOF

sudo chmod 600 /etc/onigirazu/security-policy.json
```

```bash
# 3. Set up SSH key-only access
ssh-keygen -t ed25519 -N "" -f /etc/onigirazu/onigirazu-key
sudo chmod 600 /etc/onigirazu/onigirazu-key

# Add to config
sudo tee -a /etc/onigirazu/onigirazu.yml << 'EOF'
ssh_key_file: /etc/onigirazu/onigirazu-key
ssh_port: 22
EOF
```

### Security Best Practices

✅ **Do:**

- Use Vault for all secrets
- Enable audit logging
- Restrict allowed modules
- Use security policies
- SSH key-only authentication
- Regular security reviews

❌ **Don't:**

- Hardcode passwords
- Disable host key checks
- Allow arbitrary commands
- Skip audit logging
- Use default passwords

---

## Performance Tuning

### Scenario

High-performance deployments with many hosts.

### Setup

```bash
# For 100+ hosts
cat > /etc/onigirazu/high-performance.yml << 'EOF'
# HIGH PERFORMANCE CONFIGURATION

# Execution (maximize parallelism)
max_concurrency: 100         # Aggressive parallelism
default_timeout: 10m         # Generous for scale
retry_attempts: 2            # Quick fail/retry

# Performance
enable_caching: true         # Cache aggressive
cache_ttl: 30m               # Long cache
enable_checksum: true        # Verify data
enable_parallel: true        # Full parallelism

# Connection
ssh_timeout: 60s             # Long timeout for scale
ssh_port: 22

# Logging (reduce I/O)
log_level: warn              # Minimal logging
log_format: json             # Efficient format
enable_metrics: false        # No metrics overhead

# UI (disable for speed)
color_output: false
progress_bar: false
verbose: false
EOF
```

**Tuning for specific workloads:**

```yaml
# I/O Heavy (many file operations)
max_concurrency: 50          # CPU cores × 2
cache_ttl: 10m               # Aggressive cache

# CPU Heavy (computations)
max_concurrency: 16          # Number of cores
cache_ttl: 5m                # Standard cache

# Network Heavy (many SSH calls)
max_concurrency: 100         # Many parallel
ssh_timeout: 120s            # Long timeout
```

---

## Monitoring & Audit

### Setup

```bash
# 1. Enable all monitoring
cat > /etc/onigirazu/monitoring.yml << 'EOF'
# MONITORING AND AUDIT CONFIGURATION

# Metrics
enable_metrics: true
metrics_port: 9090

# Audit logging
enable_audit: true
audit_level: verbose

# Logging
log_level: info
log_format: json
enable_audit: true

# Health checks
enable_healthcheck: true
healthcheck_port: 8080
EOF

# 2. Set up Prometheus scrape
cat > /etc/prometheus/prometheus.yml << 'EOF'
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'onigirazu'
    static_configs:
      - targets: ['localhost:9090']
EOF

# 3. Set up ELK logging
cat > /etc/filebeat/filebeat.yml << 'EOF'
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/onigirazu/*.log

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
EOF

# 4. Restart services
sudo systemctl restart prometheus
sudo systemctl restart filebeat
```

**Monitoring checks:**

```bash
# Check metrics
curl http://localhost:9090/metrics | grep onigirazu

# Check audit logs
tail -f /var/log/onigirazu/audit.json | jq .

# Check health
curl http://localhost:8080/health
```

---

## Summary Matrix

| Use Case | Template | Concurrency | Timeout | Caching | Audit |
|----------|----------|------------|---------|---------|-------|
| Development | Default | 1 | 60s | No | No |
| Production | Production | 50 | 5m | Yes | Yes |
| CI/CD | Minimal | 5 | 5m | No | No |
| Embedded | Minimal | 1 | 2m | Yes | No |
| High Performance | Custom | 100 | 10m | Yes | No |
| Security | Production | 5 | 30s | Yes | Yes |
| Kubernetes | Docker | 10 | 30s | No | No |

---

## Next Steps

1. Choose your scenario
2. Follow the setup instructions
3. Test with a simple playbook
4. Read [TROUBLESHOOTING_CONFIG.md](TROUBLESHOOTING_CONFIG.md) if issues arise
5. See [CONFIGURATION_REFERENCE.md](../CONFIGURATION_REFERENCE.md) for all options

Need help? Check [INSTALLATION_CONFIG.md](INSTALLATION_CONFIG.md) for platform-specific details.
