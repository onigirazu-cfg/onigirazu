# 🚀 Quick Start Guide

Get up and running with Onigirazu in minutes! This guide will help you install Onigirazu and run your first commands.

## 📋 Prerequisites

- **Linux/macOS/Windows** - Onigirazu supports all major platforms
- **SSH access** - For remote host management
- **Basic command line** - Familiarity with terminal commands

---

## 🏗️ Installation

### Option 1: Pre-built Binaries (Recommended)

Download the latest release for your platform:

```bash
# Linux x86_64
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# macOS (Intel)
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_x86_64.tar.gz
tar -xzf onigirazu_Darwin_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# macOS (Apple Silicon)
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_arm64.tar.gz
tar -xzf onigirazu_Darwin_arm64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Windows
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Windows_x86_64.zip
unzip onigirazu_Windows_x86_64.zip
# Move onigirazu.exe to your PATH
```

### Option 2: Build from Source

```bash
# Clone repository
git clone https://github.com/onigirazu-cfg/onigirazu.git
cd onigirazu

# Build
go build -o onigirazu cmd/onigirazu/main.go

# Install
sudo mv onigirazu /usr/local/bin/
```

### Option 3: Package Managers

```bash
# Homebrew (macOS)
brew install onigirazu

# APT (Ubuntu/Debian)
sudo apt install onigirazu

# YUM (RHEL/CentOS)
sudo yum install onigirazu
```

---

## ✅ Verify Installation

```bash
# Check version
onigirazu --version

# Check help
onigirazu --help

# List modules
onigirazu --list-modules
```

Expected output:
```
Onigirazu v1.26.1
Built with Go 1.24.0
```

---

## 📝 Create Your First Inventory

Create an inventory file to define your hosts:

```yaml
# inventory.yml
all:
  children:
    webservers:
      hosts:
        web1:
          ansible_host: 192.168.1.10
          ansible_user: ubuntu
        web2:
          ansible_host: 192.168.1.11
          ansible_user: ubuntu
    dbservers:
      hosts:
        db1:
          ansible_host: 192.168.1.20
          ansible_user: ubuntu
```

### Local Testing

For local testing, create a simple inventory:

```yaml
# local-inventory.yml
all:
  hosts:
    localhost:
      ansible_connection: local
```

---

## 🎯 Your First Commands

### 1. Test Connection

```bash
# Ping all hosts
onigirazu run all -m ping -i inventory.yml

# Ping specific group
onigirazu run webservers -m ping -i inventory.yml
```

### 2. Natural Language Commands

```bash
# Install package (natural language)
onigirazu run webservers "install nginx package" -i inventory.yml

# Start service (natural language)
onigirazu run webservers "start nginx service" -i inventory.yml

# Create file (natural language)
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
```

### 3. Traditional Commands

```bash
# Install package (traditional)
onigirazu run webservers -m package name=nginx state=present -i inventory.yml

# Start service (traditional)
onigirazu run webservers -m service name=nginx state=started -i inventory.yml

# Create file (traditional)
onigirazu run all -m file path=/tmp/test.txt state=touch -i inventory.yml
```

---

## 🔧 Advanced Examples

### Package Management

```bash
# Install multiple packages
onigirazu run webservers "install nginx package" -i inventory.yml
onigirazu run webservers "install apache package" -i inventory.yml

# Remove packages
onigirazu run all "remove old-package package" -i inventory.yml

# Update packages
onigirazu run all "update nginx package" -i inventory.yml
```

### Service Management

```bash
# Start services
onigirazu run webservers "start nginx service" -i inventory.yml
onigirazu run dbservers "start mysql service" -i inventory.yml

# Stop services
onigirazu run all "stop apache service" -i inventory.yml

# Restart services
onigirazu run webservers "restart nginx service" -i inventory.yml
```

### File Operations

```bash
# Create files
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
onigirazu run webservers "create file /var/www/index.html" -i inventory.yml

# Delete files
onigirazu run all "delete file /tmp/old.log" -i inventory.yml
```

---

## 🎯 Command Options

### Execution Options

```bash
# Check mode (dry-run)
onigirazu run all "install nginx package" --check -i inventory.yml

# Parallel execution
onigirazu run all "install nginx package" --parallel 10 -i inventory.yml

# Timeout
onigirazu run all "install nginx package" --timeout 60s -i inventory.yml
```

### Output Options

```bash
# JSON output
onigirazu run all "install nginx package" --output json -i inventory.yml

# YAML output
onigirazu run all "install nginx package" --output yaml -i inventory.yml

# Table output
onigirazu run all "install nginx package" --output table -i inventory.yml
```

### Verbose Options

```bash
# Verbose output
onigirazu run all "install nginx package" -V -i inventory.yml

# No color output
onigirazu run all "install nginx package" --no-color -i inventory.yml
```

---

## 📊 Real Examples

### Web Server Setup

```bash
# 1. Install nginx
onigirazu run webservers "install nginx package" -i inventory.yml

# 2. Start nginx
onigirazu run webservers "start nginx service" -i inventory.yml

# 3. Create index page
onigirazu run webservers "create file /var/www/html/index.html" -i inventory.yml

# 4. Check status
onigirazu run webservers -m command "systemctl status nginx" -i inventory.yml
```

### Database Setup

```bash
# 1. Install mysql
onigirazu run dbservers "install mysql package" -i inventory.yml

# 2. Start mysql
onigirazu run dbservers "start mysql service" -i inventory.yml

# 3. Check status
onigirazu run dbservers -m command "systemctl status mysql" -i inventory.yml
```

### System Maintenance

```bash
# 1. Update all packages
onigirazu run all "update nginx package" -i inventory.yml

# 2. Restart services
onigirazu run webservers "restart nginx service" -i inventory.yml

# 3. Clean up
onigirazu run all "delete file /tmp/old.log" -i inventory.yml
```

---

## 🔧 Configuration

### Global Configuration

Create a configuration file:

```yaml
# ~/.onigirazu/config.yml
defaults:
  inventory: inventory.yml
  timeout: 30s
  parallel: 5
  output: text
  verbose: false

logging:
  level: info
  format: text
  file: /var/log/onigirazu.log

ssh:
  timeout: 30s
  retries: 3
  host_key_checking: true
```

### Environment Variables

```bash
# Set environment variables
export ONIGIRAZU_INVENTORY=inventory.yml
export ONIGIRAZU_TIMEOUT=60s
export ONIGIRAZU_PARALLEL=10
export ONIGIRAZU_OUTPUT=json
```

---

## 🎯 Best Practices

### Command Organization

```bash
# Group related operations
onigirazu run webservers "install nginx package" -i inventory.yml
onigirazu run webservers "start nginx service" -i inventory.yml

# Use appropriate syntax
onigirazu run all "install nginx package" -i inventory.yml  # Natural language
onigirazu run all -m package name=nginx state=present -i inventory.yml  # Traditional
```

### Error Handling

```bash
# Check mode for safety
onigirazu run all "install nginx package" --check -i inventory.yml

# Verbose output for debugging
onigirazu run all "install nginx package" -V -i inventory.yml

# JSON output for parsing
onigirazu run all "install nginx package" --output json -i inventory.yml
```

### Performance

```bash
# Use parallel execution
onigirazu run all "install nginx package" --parallel 10 -i inventory.yml

# Use appropriate timeouts
onigirazu run all "install nginx package" --timeout 60s -i inventory.yml

# Use check mode for testing
onigirazu run all "install nginx package" --check -i inventory.yml
```

---

## 🚨 Troubleshooting

### Common Issues

**Connection refused:**
```bash
# Check SSH connectivity
ssh user@host

# Check inventory file
onigirazu run all -m ping -i inventory.yml
```

**Permission denied:**
```bash
# Check sudo access
onigirazu run all -m command "sudo whoami" -i inventory.yml

# Use become
onigirazu run all -m package name=nginx state=present --become -i inventory.yml
```

**Module not found:**
```bash
# List available modules
onigirazu --list-modules

# Check module syntax
onigirazu run all -m package name=nginx state=present --check -i inventory.yml
```

### Debug Mode

```bash
# Enable debug output
onigirazu run all "install nginx package" -V -i inventory.yml

# Check verbose output
onigirazu run all "install nginx package" --verbose -i inventory.yml
```

---

## 📚 Next Steps

### Learn More

- [Natural Language Commands](Natural-Language-Commands) - Intuitive command syntax
- [Ad-hoc Commands](Ad-hoc-Commands) - Quick operations without playbooks
- [Modules](Modules) - Comprehensive module reference
- [Architecture](Architecture) - Understanding Onigirazu internals

### Advanced Topics

- [Playbooks](Playbooks) - Complex automation workflows
- [State Management](State-Management) - Tracking system changes
- [Performance Tuning](Performance-Tuning) - Optimizing execution
- [Troubleshooting](Troubleshooting) - Solving common problems

### Contributing

- [Contributing](Contributing) - How to contribute to Onigirazu
- [Development Setup](Development-Setup) - Setting up development environment
- [Testing](Testing) - Running and writing tests

---

## 🎯 Summary

You've successfully:
- ✅ **Installed Onigirazu**
- ✅ **Created an inventory**
- ✅ **Run your first commands**
- ✅ **Used natural language syntax**
- ✅ **Explored advanced options**

**🍙 Welcome to Onigirazu - Modern Configuration Management Made Simple!**

---

**Next:** [Natural Language Commands](Natural-Language-Commands) | [Ad-hoc Commands](Ad-hoc-Commands) | [Modules](Modules)
