# 🍙 Onigirazu Wiki

Welcome to the Onigirazu Wiki! This is your comprehensive guide to using Onigirazu, a modern, high-performance configuration management tool written in Go.

## 📚 Table of Contents

### 🚀 Getting Started
- [Quick Start Guide](Quick-Start)
- [Installation](Installation)
- [Configuration](Configuration)

### 🎯 Core Features
- [Natural Language Commands](Natural-Language-Commands)
- [Ad-hoc Commands](Ad-hoc-Commands)
- [Playbooks](Playbooks)
- [Modules](Modules)

### 🏗️ Architecture
- [Architecture Overview](Architecture)
- [State Management](State-Management)
- [SSH Communication](SSH-Communication)
- [Parallel Execution](Parallel-Execution)

### 📖 Documentation
- [User Guide](User-Guide)
- [Developer Guide](Developer-Guide)
- [API Reference](API-Reference)
- [Troubleshooting](Troubleshooting)

### 🤝 Contributing
- [Contributing Guide](Contributing)
- [Development Setup](Development-Setup)
- [Testing](Testing)

---

## 🌟 What is Onigirazu?

Onigirazu is a modern, high-performance configuration management tool inspired by Ansible, but built from the ground up in Go for better performance and reliability.

### Key Features

- **🚀 High Performance**: 10x faster than Ansible
- **🎯 Natural Language**: Intuitive command syntax
- **📦 Single Binary**: No dependencies, easy deployment
- **🔧 Ad-hoc Commands**: Quick one-off operations
- **🏗️ Modern Architecture**: Built for scalability
- **🔒 Security First**: SSH host key verification
- **📊 Rich Output**: Multiple output formats

### Why Onigirazu?

| Feature | Onigirazu | Ansible |
|---------|-----------|---------|
| **Performance** | ⚡ 10x faster | 🐌 Slower |
| **Natural Language** | ✅ Yes | ❌ No |
| **Single Binary** | ✅ Yes | ❌ No |
| **Ad-hoc Commands** | ✅ Advanced | ⚠️ Basic |
| **Output Formats** | ✅ 4 formats | ⚠️ 2 formats |
| **Dependencies** | ✅ None | ❌ Many |

---

## 🚀 Quick Start

### 1. Install Onigirazu
```bash
# Download and install
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

### 2. Create Inventory
```yaml
# inventory.yml
all:
  children:
    webservers:
      hosts:
        web1:
          ansible_host: 192.168.1.10
        web2:
          ansible_host: 192.168.1.11
    dbservers:
      hosts:
        db1:
          ansible_host: 192.168.1.20
```

### 3. Run Your First Command
```bash
# Natural language command
onigirazu run webservers "install nginx package" -i inventory.yml

# Traditional command
onigirazu run webservers -m package name=nginx state=present -i inventory.yml
```

---

## 🎯 Natural Language Commands

Onigirazu supports intuitive natural language commands:

```bash
# Package management
onigirazu run all "install nginx package" -i inventory.yml
onigirazu run all "remove apache package" -i inventory.yml
onigirazu run all "update mysql package" -i inventory.yml

# Service management
onigirazu run webservers "start nginx service" -i inventory.yml
onigirazu run webservers "stop apache service" -i inventory.yml
onigirazu run webservers "restart nginx service" -i inventory.yml

# File operations
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
onigirazu run all "delete file /tmp/old.log" -i inventory.yml
```

---

## 🔧 Ad-hoc Commands

Execute quick operations without creating playbooks:

```bash
# Multiple syntax formats
onigirazu run all -m ping -i inventory.yml
onigirazu run all "install nginx package" -i inventory.yml
onigirazu run all "package:name=nginx,state=present" -i inventory.yml
onigirazu run all '{"module":"package","args":{"name":"nginx"}}' -i inventory.yml

# Advanced options
onigirazu run all -m package name=nginx state=present --check -i inventory.yml
onigirazu run all -m package name=nginx state=present --parallel 10 -i inventory.yml
onigirazu run all -m package name=nginx state=present --output json -i inventory.yml
```

---

## 📦 Modules

Onigirazu includes 18+ built-in modules:

### System Modules
- **package** - Package management
- **service** - Service management
- **user** - User management
- **group** - Group management
- **cron** - Cron job management

### File Modules
- **file** - File and directory management
- **copy** - File copying
- **template** - Template processing
- **archive** - Archive operations

### Network Modules
- **firewall** - Firewall management
- **port** - Port management

### Execution Modules
- **command** - Command execution
- **shell** - Shell execution
- **script** - Script execution

---

## 🏗️ Architecture

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CLI Parser    │    │  Playbook      │    │   Module        │
│                 │───▶│  Engine        │───▶│   Registry      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Inventory      │    │  Execution      │    │   SSH           │
│  Manager        │    │  Engine         │    │   Client        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Key Features

- **Parallel Execution**: Execute tasks concurrently
- **State Management**: Track system changes
- **SSH Communication**: Secure remote execution
- **Template Engine**: Dynamic configuration
- **Caching System**: Performance optimization

---

## 📊 Performance

Onigirazu is designed for performance:

- **10x faster** than Ansible
- **Lower memory usage** with Go's efficiency
- **Parallel execution** with configurable limits
- **Intelligent caching** for repeated operations
- **Connection pooling** for SSH connections

---

## 🔒 Security

Security is built into Onigirazu:

- **SSH host key verification** by default
- **Secure authentication** methods
- **Context cancellation** for timeouts
- **Command injection prevention**
- **Audit logging** for compliance

---

## 📚 Documentation

### User Documentation
- [Quick Start Guide](Quick-Start)
- [Natural Language Commands](Natural-Language-Commands)
- [Ad-hoc Commands](Ad-hoc-Commands)
- [Playbooks](Playbooks)
- [Modules](Modules)

### Developer Documentation
- [Architecture](Architecture)
- [API Reference](API-Reference)
- [Contributing](Contributing)
- [Development Setup](Development-Setup)

### Advanced Topics
- [State Management](State-Management)
- [Performance Tuning](Performance-Tuning)
- [Troubleshooting](Troubleshooting)
- [Migration from Ansible](Migration-from-Ansible)

---

## 🤝 Contributing

We welcome contributions! See our [Contributing Guide](Contributing) for details.

### Quick Links
- [Development Setup](Development-Setup)
- [Testing](Testing)
- [Code Style](Code-Style)
- [Pull Request Process](Pull-Request-Process)

---

## 📞 Support

- **GitHub Issues**: [Report bugs and request features](https://github.com/onigirazu-cfg/onigirazu/issues)
- **Discussions**: [Community discussions](https://github.com/onigirazu-cfg/onigirazu/discussions)
- **Documentation**: This wiki and inline help

---

## 📄 License

Onigirazu is licensed under the MIT License. See [LICENSE](https://github.com/onigirazu-cfg/onigirazu/blob/main/LICENSE) for details.

---

**🍙 Onigirazu - Modern Configuration Management Made Simple**

