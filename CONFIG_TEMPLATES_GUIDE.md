# Configuration Templates Guide - v1.55.1

**Understanding and Using Onigirazu Configuration Templates**

**Release:** v1.55.1
**Date:** January 29, 2025
**Status:** ✅ Complete

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Template Selection Guide](#template-selection-guide)
3. [Default Template](#default-template)
4. [Minimal Template](#minimal-template)
5. [Production Template](#production-template)
6. [Docker Template](#docker-template)
7. [Location Reference](#location-reference)
8. [How to Use Templates](#how-to-use-templates)
9. [Customizing Templates](#customizing-templates)

---

## Overview

Onigirazu provides **4 pre-built configuration templates** for different use cases. Each template is optimized for specific scenarios and can be found in the `examples/` directory after installation.

| Template | Size | Use Case | Priority |
|----------|------|----------|----------|
| **default** | 7.9 KB | Learning, reference, complete documentation | Development |
| **minimal** | 356 B | Embedded systems, CI/CD, quick testing | Embedded/CI |
| **production** | 4.3 KB | Enterprise, regulated environments, security-hardened | Production |
| **docker** | 3.7 KB | Containers, Kubernetes, orchestration | Containers |

---

## Template Selection Guide

### 🎓 Choose **Default** if you

- ✅ Are new to Onigirazu and learning configuration
- ✅ Want to understand all available options
- ✅ Need comprehensive documentation inline
- ✅ Are setting up a reference installation
- ✅ Want to copy-paste sections for customization

**Start here:** `/usr/share/onigirazu/onigirazu.default.yml`

### 🏗️ Choose **Minimal** if you

- ✅ Are deploying to embedded systems or IoT devices
- ✅ Need to minimize configuration file size
- ✅ Are using Onigirazu in CI/CD pipelines
- ✅ Want quick test configurations
- ✅ Have space constraints (low-disk environments)

**Perfect for:**

```bash
# Quick CI/CD setup
onigirazu -c /usr/share/onigirazu/examples/onigirazu.minimal.yml playbook.yml

# Embedded systems with storage limits
sudo cp /usr/share/onigirazu/examples/onigirazu.minimal.yml /etc/onigirazu/onigirazu.yml
```

### 🔒 Choose **Production** if you

- ✅ Are deploying to production environments
- ✅ Need security hardening enabled by default
- ✅ Work in regulated industries (healthcare, finance, government)
- ✅ Require audit logging and security policies
- ✅ Have compliance requirements (SOC 2, ISO 27001, PCI-DSS)
- ✅ Need enterprise secret management (Vault integration)

**Perfect for:**

```bash
# Enterprise production deployment
sudo cp /usr/share/onigirazu/examples/onigirazu.production.yml /etc/onigirazu/onigirazu.yml

# Regulated environment with logging aggregation
onigirazu -c /etc/onigirazu/onigirazu.yml playbook.yml
```

### 🐳 Choose **Docker** if you

- ✅ Are deploying in Docker containers
- ✅ Are using Kubernetes orchestration
- ✅ Need JSON logging for container log drivers
- ✅ Don't have TTY available (no interactive features needed)
- ✅ Are using log aggregation systems (ELK, Datadog, Splunk)

**Perfect for:**

```dockerfile
# In your Dockerfile
COPY examples/onigirazu.docker.yml /etc/onigirazu/onigirazu.yml

# In your container entrypoint
onigirazu -c /etc/onigirazu/onigirazu.yml playbook.yml
```

---

## Default Template

### Overview

The **Default** template contains **all ~30 configuration options** with detailed comments explaining:

- Purpose of each setting
- Valid values and ranges
- Recommended settings
- Use cases and examples

### What's Included

```yaml
# EXECUTION SETTINGS
max_concurrency: 10
default_timeout: 30s
retry_attempts: 3
retry_delay: 5s

# LOGGING CONFIGURATION
log_level: info
log_format: text

# SECURITY CONFIGURATION
allow_shell_commands: false
command_blocklist: [...]
allow_module_plugins: true

# PERFORMANCE TUNING
enable_caching: true
cache_ttl: 5m
enable_checksum: true

# SSH/CONNECTION SETTINGS
ssh_timeout: 30s
ssh_port: 22
ssh_user: root
ssh_key_pass: ""
ssh_strict_host_key_check: true

# MONITORING & METRICS
enable_metrics: false
metrics_port: 9090
enable_audit: false

# VAULT INTEGRATION
vault_enabled: false
vault_addr: "http://localhost:8200"

# And more...
```

### When to Use

```bash
# Learning configuration
cat /usr/share/onigirazu/onigirazu.default.yml | less

# Reference implementation
onigirazu -c /usr/share/onigirazu/onigirazu.default.yml playbook.yml

# Copy as base for customization
cp /usr/share/onigirazu/onigirazu.default.yml ~/my-config.yml
nano ~/my-config.yml
```

---

## Minimal Template

### Overview

The **Minimal** template contains **only 5 essential settings** in just 356 bytes:

- `max_concurrency`
- `default_timeout`
- `log_level`
- `retry_attempts`
- `retry_delay`

### Content

```yaml
# Onigirazu Minimal Configuration (v1.55.1)
# Perfect for: CI/CD, embedded systems, quick tests
max_concurrency: 10
default_timeout: 30s
log_level: info
retry_attempts: 3
retry_delay: 5s
```

### Advantages

| Feature | Benefit |
|---------|---------|
| **Tiny size** | 356 bytes = instant load |
| **Fast loading** | Minimal parsing overhead |
| **Easy to understand** | Only essential options |
| **Version compatible** | Works across versions |
| **Storage efficient** | For low-disk systems |

### When to Use

```bash
# CI/CD Pipeline
onigirazu -c examples/onigirazu.minimal.yml deploy.yml

# Quick testing
onigirazu -c examples/onigirazu.minimal.yml test.yml

# Embedded system
ssh root@iot-device cp examples/onigirazu.minimal.yml /etc/onigirazu/

# Container base image
FROM alpine:latest
COPY examples/onigirazu.minimal.yml /etc/onigirazu/onigirazu.yml
```

### Size Comparison

```
Default template:  7.9  KB (full documentation)
Production template: 4.3  KB (security hardened)
Docker template:   3.7  KB (container optimized)
Minimal template:    356 B (bare essentials) ← Most compact
```

---

## Production Template

### Overview

The **Production** template is **security-hardened and enterprise-ready**, optimized for:

- Regulated environments
- High-security deployments
- Audit and compliance requirements
- Log aggregation systems

### Security Features Enabled

```yaml
# 🔒 SECURITY: Everything restrictive by default

# Disable shell commands (use modules instead)
allow_shell_commands: false

# Comprehensive command blocklist
command_blocklist:
  - "rm -rf"
  - "dd if="
  - "mkfs"
  - "format"
  - "shutdown"
  - "reboot"

# Strict SSH verification
ssh_strict_host_key_check: true
ssh_key_pass: ""  # No hardcoded passwords

# Extensive blocked directories
blocked_directories:
  - "/etc/passwd"
  - "/etc/shadow"
  - "/boot"
  - "/sys"
  - "/proc"
  - "/dev"

# Audit enabled
enable_audit: true
```

### Production-Specific Settings

```yaml
# Enterprise logging for aggregation systems
log_format: json  # For ELK, Splunk, Datadog
log_level: info   # Balance of details and noise

# Performance tuning for scale
max_concurrency: 50  # Higher for large deployments

# Vault integration for secrets
vault_enabled: true
vault_addr: "https://vault.example.com:8200"

# Metrics for monitoring
enable_metrics: true
metrics_port: 9090

# Disable progress/colors in production
color_output: false
progress_bar: false
```

### When to Use

```bash
# Production deployment
sudo cp /usr/share/onigirazu/examples/onigirazu.production.yml \
  /etc/onigirazu/onigirazu.yml

# Run audited
onigirazu -c /etc/onigirazu/onigirazu.yml playbook.yml

# In Kubernetes
kubectl create configmap onigirazu-config \
  --from-file=/usr/share/onigirazu/examples/onigirazu.production.yml
```

---

## Docker Template

### Overview

The **Docker** template is optimized for **containerized deployments**:

- JSON logging for Docker/Kubernetes
- Disabled colors and progress (no TTY)
- Disabled interactive mode
- Efficient for container logging drivers

### Container-Specific Settings

```yaml
# 🐳 DOCKER OPTIMIZATIONS

# JSON logging for container log drivers
log_format: json
log_level: info

# No colors (unnecessary in containers)
color_output: false

# No progress bars (no TTY in containers)
progress_bar: false

# No interactive mode (no terminal)
enable_interactive: false

# Disable ANSI (not needed in containers)
disable_ansi: true
```

### Use Cases

```bash
# Standard Docker
docker run -v /etc/onigirazu:/etc/onigirazu \
  onigirazu/onigirazu -c /etc/onigirazu/onigirazu.docker.yml playbook.yml

# Kubernetes ConfigMap
kubectl create configmap onigirazu-config \
  --from-file=/usr/share/onigirazu/examples/onigirazu.docker.yml

# Docker Compose
version: '3'
services:
  automation:
    image: onigirazu:latest
    volumes:
      - ./examples/onigirazu.docker.yml:/etc/onigirazu/onigirazu.yml
    command: -c /etc/onigirazu/onigirazu.yml playbook.yml
```

### Log Output Compatibility

```bash
# Docker JSON logging driver
docker logs --follow container_id | jq .

# Kubernetes pod logs
kubectl logs -f pod-name | jq .

# ELK Stack ingestion
filebeat reads logs → Elasticsearch → Kibana

# Datadog Agent compatibility
Datadog Agent reads JSON format directly
```

---

## Location Reference

### After Installation

| Type | Linux | macOS | Windows |
|------|-------|-------|---------|
| **Installed** | `/usr/bin/onigirazu` | `/usr/local/bin/onigirazu` | `C:\Program Files\Onigirazu\` |
| **Default Config** | `/usr/share/onigirazu/onigirazu.default.yml` | N/A | `%ProgramData%\Onigirazu\` |
| **Templates** | `/usr/share/onigirazu/examples/` | N/A | `%ProgramData%\Onigirazu\examples\` |
| **System Config** | `/etc/onigirazu/` | `/etc/onigirazu/` | `%ProgramData%\Onigirazu\` |
| **User Config** | `~/.onigirazu/` | `~/.onigirazu/` | `%APPDATA%\Onigirazu\` |

### Package Manager Locations

**Linux (Debian/Ubuntu):**

```bash
sudo dpkg -L onigirazu | grep config
# Output:
# /usr/share/onigirazu/onigirazu.default.yml
# /usr/share/onigirazu/examples/onigirazu.minimal.yml
# /usr/share/onigirazu/examples/onigirazu.production.yml
# /usr/share/onigirazu/examples/onigirazu.docker.yml
```

**Docker:**

```bash
docker run onigirazu cat /usr/share/onigirazu/onigirazu.default.yml
```

---

## How to Use Templates

### 1. Inspect a Template

```bash
# View default template
cat /usr/share/onigirazu/onigirazu.default.yml

# View minimal template
cat /usr/share/onigirazu/examples/onigirazu.minimal.yml

# Check what options are in production
grep "^[a-z]" /usr/share/onigirazu/examples/onigirazu.production.yml
```

### 2. Copy to Active Config

```bash
# Use default as your config
sudo cp /usr/share/onigirazu/onigirazu.default.yml \
  /etc/onigirazu/onigirazu.yml

# Use production as your config
sudo cp /usr/share/onigirazu/examples/onigirazu.production.yml \
  /etc/onigirazu/onigirazu.yml

# Use minimal for quick test
sudo cp /usr/share/onigirazu/examples/onigirazu.minimal.yml \
  /etc/onigirazu/test-config.yml
```

### 3. Use Template for Single Run

```bash
# Run with default template
onigirazu -c /usr/share/onigirazu/onigirazu.default.yml playbook.yml

# Run with production template (security audit mode)
onigirazu -c /usr/share/onigirazu/examples/onigirazu.production.yml playbook.yml

# Run with minimal for quick test
onigirazu -c /usr/share/onigirazu/examples/onigirazu.minimal.yml playbook.yml

# Run with docker template
docker run -v $(pwd):/playbooks \
  onigirazu -c /usr/share/onigirazu/examples/onigirazu.docker.yml playbook.yml
```

### 4. Customize a Template

```bash
# Start with default, customize for your needs
cp /usr/share/onigirazu/onigirazu.default.yml ~/my-config.yml

# Edit and customize
nano ~/my-config.yml

# Use your custom config
onigirazu -c ~/my-config.yml playbook.yml

# Save as project config
cp ~/my-config.yml /path/to/project/onigirazu.yml
```

---

## Customizing Templates

### Extending a Template

Start with a template and add your customizations:

```bash
# 1. Copy a template
cp /usr/share/onigirazu/examples/onigirazu.production.yml onigirazu.yml

# 2. Keep the good parts, customize the rest
nano onigirazu.yml

# 3. Make these changes:
max_concurrency: 100        # Increase for large deployments
vault_addr: "https://vault.mycompany.com:8200"  # Your vault
vault_token_path: "~/.vault-token"  # Your token location

# 4. Use it
onigirazu -c onigirazu.yml playbook.yml
```

### Common Customizations

**For AI/ML Workloads:**

```yaml
# Start with: Default or Production
# Customize:
max_concurrency: 50      # Scale based on available cores
enable_caching: false    # Disable cache for consistency
cache_ttl: 1m            # Shorter cache for updates
retry_attempts: 5        # More retries for flaky operations
```

**For CI/CD Pipelines:**

```yaml
# Start with: Minimal
# Customize:
max_concurrency: 10      # Keep reasonable
log_format: json         # For log aggregation
log_level: info          # Balance noise and info
enable_audit: false      # Disable audit in CI
```

**For Enterprise:**

```yaml
# Start with: Production
# Customize:
vault_enabled: true
vault_addr: "https://vault.corp.internal:8200"
enable_metrics: true
metrics_port: 9090
log_format: json
enable_audit: true
command_blocklist:
  # Add your company's restrictions
  - "sudo"
  - "su -"
```

---

## Summary: Quick Reference

| Need | Template | Size | Key Feature |
|------|----------|------|------------|
| Learn Onigirazu | **default** | 7.9 KB | 📖 Full documentation |
| Small footprint | **minimal** | 356 B | ⚡ Fastest load time |
| Security hardened | **production** | 4.3 KB | 🔒 Audit ready |
| Containerized | **docker** | 3.7 KB | 🐳 JSON logging |

**Next Steps:**

- Choose a template based on your use case
- Copy to `/etc/onigirazu/onigirazu.yml`
- Customize for your environment
- See [CONFIGURATION_SETUP_GUIDE.md](CONFIGURATION_SETUP_GUIDE.md) for specific scenarios
- See [TROUBLESHOOTING_CONFIG.md](TROUBLESHOOTING_CONFIG.md) if you hit issues
