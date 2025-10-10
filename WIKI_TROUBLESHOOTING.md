# 🚨 Troubleshooting

This guide helps you diagnose and resolve common issues with Onigirazu.

## 📋 Common Issues

### Connection Issues

#### SSH Connection Refused
**Symptoms:**
```
Error: failed to connect to host: connection refused
```

**Solutions:**
```bash
# 1. Check SSH service
onigirazu run all -m command "systemctl status ssh" -i inventory.yml

# 2. Test SSH manually
ssh user@host

# 3. Check firewall
onigirazu run all -m command "ufw status" -i inventory.yml

# 4. Verify inventory
onigirazu run all -m ping -i inventory.yml
```

#### SSH Authentication Failed
**Symptoms:**
```
Error: authentication failed
```

**Solutions:**
```bash
# 1. Check SSH keys
ssh-keygen -t rsa -b 4096 -C "your_email@example.com"
ssh-copy-id user@host

# 2. Test SSH key
ssh -i ~/.ssh/id_rsa user@host

# 3. Check SSH agent
ssh-add -l

# 4. Use password authentication
onigirazu run all -m ping -i inventory.yml --ask-pass
```

#### Host Key Verification Failed
**Symptoms:**
```
Error: host key verification failed
```

**Solutions:**
```bash
# 1. Add host to known_hosts
ssh-keyscan host >> ~/.ssh/known_hosts

# 2. Disable host key checking (not recommended)
onigirazu run all -m ping -i inventory.yml --skip-host-key-check

# 3. Check known_hosts file
cat ~/.ssh/known_hosts
```

---

### Module Issues

#### Module Not Found
**Symptoms:**
```
Error: module not found: package
```

**Solutions:**
```bash
# 1. List available modules
onigirazu --list-modules

# 2. Check module name
onigirazu run all -m package name=nginx state=present --check -i inventory.yml

# 3. Verify module syntax
onigirazu run all -m package name=nginx state=present --syntax-check -i inventory.yml
```

#### Module Execution Failed
**Symptoms:**
```
Error: module execution failed
```

**Solutions:**
```bash
# 1. Check module arguments
onigirazu run all -m package name=nginx state=present --check -i inventory.yml

# 2. Use verbose output
onigirazu run all -m package name=nginx state=present -V -i inventory.yml

# 3. Check module documentation
onigirazu run all -m package --help
```

#### Permission Denied
**Symptoms:**
```
Error: permission denied
```

**Solutions:**
```bash
# 1. Use sudo
onigirazu run all -m package name=nginx state=present --become -i inventory.yml

# 2. Check sudo access
onigirazu run all -m command "sudo whoami" -i inventory.yml

# 3. Use specific user
onigirazu run all -m package name=nginx state=present --become-user root -i inventory.yml
```

---

### Performance Issues

#### Slow Execution
**Symptoms:**
```
Tasks taking too long to complete
```

**Solutions:**
```bash
# 1. Increase parallel execution
onigirazu run all -m package name=nginx state=present --parallel 10 -i inventory.yml

# 2. Use connection pooling
onigirazu run all -m package name=nginx state=present --connection-pool -i inventory.yml

# 3. Enable caching
onigirazu run all -m package name=nginx state=present --cache -i inventory.yml

# 4. Check network latency
onigirazu run all -m command "ping -c 3 8.8.8.8" -i inventory.yml
```

#### High Memory Usage
**Symptoms:**
```
High memory consumption
```

**Solutions:**
```bash
# 1. Reduce parallel execution
onigirazu run all -m package name=nginx state=present --parallel 5 -i inventory.yml

# 2. Disable caching
onigirazu run all -m package name=nginx state=present --no-cache -i inventory.yml

# 3. Use streaming output
onigirazu run all -m package name=nginx state=present --stream -i inventory.yml
```

---

### Output Issues

#### No Output
**Symptoms:**
```
No output from commands
```

**Solutions:**
```bash
# 1. Use verbose output
onigirazu run all -m package name=nginx state=present -V -i inventory.yml

# 2. Check output format
onigirazu run all -m package name=nginx state=present --output text -i inventory.yml

# 3. Enable debug mode
onigirazu run all -m package name=nginx state=present --debug -i inventory.yml
```

#### Incorrect Output Format
**Symptoms:**
```
Output not in expected format
```

**Solutions:**
```bash
# 1. Specify output format
onigirazu run all -m package name=nginx state=present --output json -i inventory.yml

# 2. Use table format
onigirazu run all -m package name=nginx state=present --output table -i inventory.yml

# 3. Check output options
onigirazu run all -m package name=nginx state=present --help
```

---

## 🔧 Debug Mode

### Enable Debug Output

```bash
# Verbose output
onigirazu run all -m package name=nginx state=present -V -i inventory.yml

# Debug output
onigirazu run all -m package name=nginx state=present --debug -i inventory.yml

# Trace output
onigirazu run all -m package name=nginx state=present --trace -i inventory.yml
```

### Debug Information

```bash
# System information
onigirazu run all -m command "uname -a" -i inventory.yml

# Environment variables
onigirazu run all -m command "env" -i inventory.yml

# Process information
onigirazu run all -m command "ps aux" -i inventory.yml

# Network information
onigirazu run all -m command "netstat -tuln" -i inventory.yml
```

---

## 🔍 Diagnostic Commands

### System Diagnostics

```bash
# Check system status
onigirazu run all -m command "systemctl status" -i inventory.yml

# Check disk usage
onigirazu run all -m command "df -h" -i inventory.yml

# Check memory usage
onigirazu run all -m command "free -h" -i inventory.yml

# Check network connectivity
onigirazu run all -m command "ping -c 3 8.8.8.8" -i inventory.yml
```

### Onigirazu Diagnostics

```bash
# Check version
onigirazu --version

# Check modules
onigirazu --list-modules

# Check configuration
onigirazu run all -m ping -i inventory.yml

# Check inventory
onigirazu run all -m command "hostname" -i inventory.yml
```

---

## 🚨 Error Codes

### Common Error Codes

| Code | Description | Solution |
|------|-------------|----------|
| **1** | General error | Check command syntax |
| **2** | Connection failed | Check SSH connectivity |
| **3** | Authentication failed | Check SSH keys |
| **4** | Permission denied | Use sudo or check permissions |
| **5** | Module not found | Check module name |
| **6** | Timeout | Increase timeout value |
| **7** | Invalid arguments | Check module arguments |
| **8** | Host unreachable | Check network connectivity |

### Error Handling

```bash
# Continue on error
onigirazu run all -m package name=nginx state=present --continue-on-error -i inventory.yml

# Retry on failure
onigirazu run all -m package name=nginx state=present --retry 3 -i inventory.yml

# Ignore errors
onigirazu run all -m package name=nginx state=present --ignore-errors -i inventory.yml
```

---

## 🔧 Configuration Issues

### Configuration File Problems

```bash
# Check configuration
onigirazu run all -m ping -i inventory.yml --config-check

# Validate configuration
onigirazu run all -m ping -i inventory.yml --validate

# Show configuration
onigirazu run all -m ping -i inventory.yml --show-config
```

### Inventory Issues

```bash
# Check inventory syntax
onigirazu run all -m ping -i inventory.yml --syntax-check

# Validate inventory
onigirazu run all -m ping -i inventory.yml --validate

# List hosts
onigirazu run all -m ping -i inventory.yml --list-hosts
```

---

## 🎯 Best Practices

### Error Prevention

```bash
# Use check mode
onigirazu run all -m package name=nginx state=present --check -i inventory.yml

# Test connectivity first
onigirazu run all -m ping -i inventory.yml

# Use verbose output
onigirazu run all -m package name=nginx state=present -V -i inventory.yml
```

### Performance Optimization

```bash
# Use parallel execution
onigirazu run all -m package name=nginx state=present --parallel 10 -i inventory.yml

# Enable caching
onigirazu run all -m package name=nginx state=present --cache -i inventory.yml

# Use connection pooling
onigirazu run all -m package name=nginx state=present --connection-pool -i inventory.yml
```

### Security Best Practices

```bash
# Use SSH keys
ssh-keygen -t rsa -b 4096 -C "your_email@example.com"

# Verify host keys
onigirazu run all -m ping -i inventory.yml --verify-host-keys

# Use sudo when needed
onigirazu run all -m package name=nginx state=present --become -i inventory.yml
```

---

## 📚 Getting Help

### Documentation

- [Quick Start](Quick-Start) - Getting started guide
- [Natural Language Commands](Natural-Language-Commands) - Command syntax
- [Ad-hoc Commands](Ad-hoc-Commands) - Quick operations
- [Modules](Modules) - Module reference
- [Architecture](Architecture) - System architecture

### Community Support

- **GitHub Issues**: [Report bugs and request features](https://github.com/onigirazu-cfg/onigirazu/issues)
- **Discussions**: [Community discussions](https://github.com/onigirazu-cfg/onigirazu/discussions)
- **Documentation**: This wiki and inline help

### Professional Support

- **Enterprise Support**: Available for enterprise customers
- **Training**: Onigirazu training and certification
- **Consulting**: Professional consulting services

---

## 🎯 Summary

### Quick Troubleshooting Checklist

1. **✅ Check connectivity** - `onigirazu run all -m ping -i inventory.yml`
2. **✅ Verify authentication** - `ssh user@host`
3. **✅ Test module** - `onigirazu run all -m package name=nginx state=present --check -i inventory.yml`
4. **✅ Use verbose output** - `onigirazu run all -m package name=nginx state=present -V -i inventory.yml`
5. **✅ Check permissions** - `onigirazu run all -m command "sudo whoami" -i inventory.yml`

### Common Solutions

- **Connection issues** → Check SSH and firewall
- **Authentication issues** → Check SSH keys and passwords
- **Permission issues** → Use sudo or check user permissions
- **Module issues** → Check module name and arguments
- **Performance issues** → Adjust parallel execution and caching

---

**🚨 Troubleshooting helps you resolve issues quickly and efficiently!**
