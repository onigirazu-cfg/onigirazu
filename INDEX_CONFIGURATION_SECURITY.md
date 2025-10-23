# Onigirazu Configuration & Security Documentation Index - v1.52.0

**Complete guide to all configuration and security policy documentation**

**Release:** v1.52.0
**Date:** January 29, 2025
**Last Updated:** January 29, 2025

---

## 📚 Documentation Overview

This index helps you find the right documentation for your needs.

---

## 🚀 Getting Started (5 Minutes)

**For immediate setup:**

👉 **[QUICK_START_CONFIGURATION.md](QUICK_START_CONFIGURATION.md)**

- Step-by-step setup in 5 minutes
- How to fix "path is not in allowed directories"
- Copy-paste ready commands
- Common scenarios
- **Start here if you're new**

---

## 📖 Complete Reference Guides

### Main Configuration Reference

**[CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md)** (850+ lines)

Complete documentation of all 35+ Onigirazu configuration options:

**Sections:**

- Configuration file locations and discovery
- Execution settings (concurrency, timeouts, retries)
- Logging configuration
- Security options
- Performance tuning
- Execution modes (dry-run, check-mode)
- User interface settings
- SSH/Connection configuration
- Monitoring & metrics
- Vault integration
- Syntax preferences
- Environment variable overrides
- Complete example configurations
- Common scenarios (dev, production, CI/CD)

**Use when:**

- You need to understand a specific option
- You want to optimize performance
- You're setting up production environment
- You need to enable monitoring or profiling

---

### Security Policy Reference

**[SECURITY_POLICY_GUIDE.md](SECURITY_POLICY_GUIDE.md)** (600+ lines)

Complete documentation of security policies and restrictions:

**Sections:**

- Overview of security policies
- File directory restrictions (FIXES YOUR ISSUE!)
- Host access control
- Port restrictions
- Module whitelisting
- Command blocking patterns
- File type restrictions
- File size limits
- Encryption requirements
- Audit logging
- Configuration examples (dev, production, web, database)
- Troubleshooting

**Use when:**

- You get security validation errors
- You need to restrict file operations
- You need to whitelist/blacklist modules
- You need to set up audit logging
- You're setting up production security
- You want to block dangerous commands

---

## 🔧 Example Configuration Files

### Main Configuration Example

**File:** `/onigirazu/examples/onigirazu.yml`

Ready-to-use example with all 35+ options:

- Fully commented
- Real-world values
- Grouped by sections
- Copy to `/etc/onigirazu/onigirazu.yml`

---

### Security Policy Example

**File:** `/onigirazu/examples/security-policy.json`

Ready-to-use security policy with all options:

- Fully commented
- All settings explained
- Copy to `/etc/onigirazu/security-policy.json`

---

## 🎯 Find What You Need

### By Problem/Error

**"path is not in allowed directories"**

1. Read: SECURITY_POLICY_GUIDE.md → Troubleshooting
2. Action: Add path to `allowed_directories` in security-policy.json
3. Example: Look at `/onigirazu/examples/security-policy.json`

**"blocked command detected"**

1. Read: SECURITY_POLICY_GUIDE.md → Command Blocking
2. Action: Remove from `blocked_commands` or update playbook
3. Reference: SECURITY_POLICY_GUIDE.md → Troubleshooting

**"module not allowed"**

1. Read: SECURITY_POLICY_GUIDE.md → Module Whitelisting
2. Action: Add to `allowed_modules` in security-policy.json
3. Reference: Examples in SECURITY_POLICY_GUIDE.md

**"host not allowed"**

1. Read: SECURITY_POLICY_GUIDE.md → Host Access Control
2. Action: Add to `allowed_hosts` in security-policy.json

**Task timeout**

1. Read: CONFIGURATION_REFERENCE.md → Execution Settings
2. Action: Increase `default_timeout` or per-task timeout
3. Example: `default_timeout: 5m`

**Performance too slow**

1. Read: CONFIGURATION_REFERENCE.md → Performance Tuning
2. Consider:
   - Increase `max_concurrency`
   - Enable `enable_parallel`
   - Disable `enable_caching` if problematic

**Need to run in production**

1. Read: SECURITY_POLICY_GUIDE.md → Configuration Examples → Production
2. Reference: CONFIGURATION_REFERENCE.md → Common Scenarios → Production Deployment

---

### By Role

**For Operators/Users**

1. Start: QUICK_START_CONFIGURATION.md
2. Reference: CONFIGURATION_REFERENCE.md
3. Troubleshoot: SECURITY_POLICY_GUIDE.md → Troubleshooting

**For DevOps/Administrators**

1. Read: SECURITY_POLICY_GUIDE.md (complete)
2. Read: CONFIGURATION_REFERENCE.md → SSH/Connection
3. Create: /etc/onigirazu/onigirazu.yml
4. Create: /etc/onigirazu/security-policy.json
5. Enable: Audit logging and metrics

**For Security Engineers**

1. Read: SECURITY_POLICY_GUIDE.md (complete)
2. Focus: File restrictions, command blocking, audit logging
3. Create: Restrictive security policy
4. Enable: `audit_enabled` and `require_encryption`
5. Configure: `allowed_directories` and `blocked_commands`

**For Developers**

1. Read: CONFIGURATION_REFERENCE.md → Execution Modes
2. Read: CONFIGURATION_REFERENCE.md → User Interface
3. Use: Environment variables for local development
4. Example: `export ONIGIRAZU_LOG_LEVEL=debug`

---

### By Environment

**Development/Local**

1. Reference: CONFIGURATION_REFERENCE.md → Common Scenarios → Development Environment
2. Set: `ssh_strict_host_key: false` (if needed)
3. Enable: `verbose: true` and `show_diff: true`
4. Example: Look at dev scenario examples

**Staging/Testing**

1. Reference: CONFIGURATION_REFERENCE.md → Common Scenarios → Production Deployment
2. Enable: `enable_metrics: true`
3. Set: `log_level: info`
4. Create: Moderate security policy

**Production/Critical**

1. Reference: CONFIGURATION_REFERENCE.md → Common Scenarios → Production Deployment
2. Read: SECURITY_POLICY_GUIDE.md → Best Practices
3. Set: `ssh_strict_host_key: true`
4. Enable: `audit_enabled: true` and `enable_metrics: true`
5. Create: Restrictive security policy

**CI/CD Pipeline**

1. Reference: CONFIGURATION_REFERENCE.md → Common Scenarios → CI/CD Pipeline
2. Set: `log_format: json` for parsing
3. Set: `color_output: false`
4. Enable: `enable_metrics: true`

---

## 🔍 Quick Reference by Topic

### Concurrency & Performance

- **File:** CONFIGURATION_REFERENCE.md
- **Section:** Performance Tuning
- **Options:**
  - `max_concurrency`
  - `enable_parallel`
  - `parallel_strategy`
  - `enable_caching`
  - `cache_ttl`

### Timeout & Retry Settings

- **File:** CONFIGURATION_REFERENCE.md
- **Section:** Execution Settings
- **Options:**
  - `default_timeout`
  - `retry_attempts`
  - `retry_delay`

### SSH/Connection

- **File:** CONFIGURATION_REFERENCE.md
- **Section:** SSH/Connection Settings
- **Options:** ssh_* settings

### Security & Restrictions

- **File:** SECURITY_POLICY_GUIDE.md
- **Topics:**
  - Directory restrictions
  - Command blocking
  - Host/port control
  - Module whitelisting

### Logging & Debugging

- **File:** CONFIGURATION_REFERENCE.md
- **Section:** Logging Configuration
- **Options:**
  - `log_level`
  - `log_format`
  - `verbose`

### Monitoring & Metrics

- **File:** CONFIGURATION_REFERENCE.md
- **Section:** Monitoring & Metrics
- **Options:**
  - `enable_metrics`
  - `metrics_port`
  - `enable_profiling`

### File Operations

- **File:** SECURITY_POLICY_GUIDE.md
- **Section:** File Directory Restrictions
- **Options:**
  - `allowed_directories`
  - `blocked_directories`
  - `allowed_file_types`
  - `max_file_size`

---

## 📋 Configuration File Locations

### Discovery Order

Onigirazu searches on the **control machine** in this order:

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

### Recommended Locations

**Development:**

```
./onigirazu.yml
./security-policy.json
```

**Server/Production:**

```
/etc/onigirazu/onigirazu.yml
/etc/onigirazu/security-policy.json
```

---

## 🎓 Learning Paths

### Path 1: Just Get It Working (5 minutes)

1. Read: QUICK_START_CONFIGURATION.md
2. Copy: examples/onigirazu.yml → /etc/onigirazu/
3. Copy: examples/security-policy.json → /etc/onigirazu/
4. Run playbooks

### Path 2: Understand Configuration (30 minutes)

1. Read: QUICK_START_CONFIGURATION.md
2. Read: CONFIGURATION_REFERENCE.md → Overview sections
3. Reference specific options as needed

### Path 3: Master All Options (2 hours)

1. Read: CONFIGURATION_REFERENCE.md (complete)
2. Read: SECURITY_POLICY_GUIDE.md (complete)
3. Study: /onigirazu/examples/ files
4. Experiment: Try different settings

### Path 4: Production Setup (1 hour)

1. Read: CONFIGURATION_REFERENCE.md → Common Scenarios → Production
2. Read: SECURITY_POLICY_GUIDE.md → Best Practices
3. Read: SECURITY_POLICY_GUIDE.md → Configuration Examples → Production
4. Create: Custom onigirazu.yml and security-policy.json
5. Test: Run playbooks with production settings

---

## 📞 Troubleshooting Quick Links

All troubleshooting sections can be found in:

- [QUICK_START_CONFIGURATION.md](QUICK_START_CONFIGURATION.md) → Troubleshooting
- [SECURITY_POLICY_GUIDE.md](SECURITY_POLICY_GUIDE.md) → Troubleshooting

---

## ✅ All Options Documented

### Main Configuration Options (35+)

See: [CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md)

- Execution: max_concurrency, default_timeout, retry_attempts, retry_delay
- Logging: log_level, log_format
- Security: allow_shell_commands, blocked_commands
- Performance: enable_caching, cache_ttl, enable_checksum, enable_parallel, parallel_strategy
- Execution Modes: dry_run, check_mode, verbose, show_diff
- UI: color_output, progress_bar, interactive_mode, output_format
- SSH: ssh_timeout, ssh_keepalive, ssh_max_sessions, connection_reuse, ssh_strict_host_key, ssh_known_hosts_file, default_insecure_ignore_host_key
- Monitoring: enable_metrics, metrics_port, metrics_path, enable_profiling
- Vault: vault_enabled, vault_address, vault_token
- Syntax: preferred_module_syntax, enforce_module_syntax

### Security Policy Options (13+)

See: [SECURITY_POLICY_GUIDE.md](SECURITY_POLICY_GUIDE.md)

- allowed_hosts, allowed_ports, allowed_modules, blocked_commands
- allowed_directories, blocked_directories
- allowed_file_types, blocked_file_types, max_file_size
- require_encryption, max_retries, max_timeout
- audit_enabled, log_level

---

## 🔗 Related Documentation

- [LOOPS_GUIDE.md](LOOPS_GUIDE.md) - Loop operations and variables
- [MODULE_DEVELOPMENT_GUIDE.md](MODULE_DEVELOPMENT_GUIDE.md) - Creating custom modules
- [CI-CD.md](ci-cd.md) - CI/CD pipeline configuration
- [ARCHITECTURE_DIAGRAM.md](ARCHITECTURE_DIAGRAM.md) - System architecture

---

## 📝 Document Map

```
onigirazu/docs/
├── INDEX_CONFIGURATION_SECURITY.md      ← You are here
├── CONFIGURATION_REFERENCE.md           ← All config options (850+ lines)
├── SECURITY_POLICY_GUIDE.md            ← Security policies (600+ lines)
├── QUICK_START_CONFIGURATION.md        ← Quick setup (200+ lines)
├── LOOPS_GUIDE.md
├── MODULE_DEVELOPMENT_GUIDE.md
├── ci-cd.md
└── ... other documentation

examples/
├── onigirazu.yml                        ← Config example
└── security-policy.json                 ← Security example
```

---

## 🎯 Next Steps

1. **New User?** → Read [QUICK_START_CONFIGURATION.md](QUICK_START_CONFIGURATION.md)
2. **Need Options?** → Read [CONFIGURATION_REFERENCE.md](CONFIGURATION_REFERENCE.md)
3. **Need Security?** → Read [SECURITY_POLICY_GUIDE.md](SECURITY_POLICY_GUIDE.md)
4. **Getting Error?** → Search "Troubleshooting" in relevant guide
5. **Copy Files?** → Use examples in `/onigirazu/examples/`

---

## ✨ Status

✅ **COMPLETE** - All documentation created and verified
✅ **PRODUCTION READY** - All options documented accurately
✅ **CURRENT** - Reflects latest code implementation
✅ **EXAMPLES PROVIDED** - Ready-to-use configuration files

**Last Updated:** 2025-01-29

---

For questions about specific options, see the appropriate guide:

- Configuration options → CONFIGURATION_REFERENCE.md
- Security & restrictions → SECURITY_POLICY_GUIDE.md
- Quick answers → QUICK_START_CONFIGURATION.md
