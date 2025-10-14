# 📦 Installation

This guide covers different ways to install Onigirazu on various platforms.

## 📋 Prerequisites

- **Linux/macOS/Windows** - Onigirazu supports all major platforms
- **SSH access** - For remote host management
- **Basic command line** - Familiarity with terminal commands

---

## 🚀 Quick Installation

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

### Option 2: Package Managers

```bash
# Homebrew (macOS)
brew install onigirazu

# APT (Ubuntu/Debian)
sudo apt update
sudo apt install onigirazu

# YUM (RHEL/CentOS)
sudo yum install onigirazu

# DNF (Fedora)
sudo dnf install onigirazu

# Pacman (Arch Linux)
sudo pacman -S onigirazu
```

---

## 🏗️ Build from Source

### Prerequisites

- **Go 1.24.0+** - Latest Go version
- **Git** - Version control
- **Make** - Build automation (optional)

### Build Steps

```bash
# Clone repository
git clone https://github.com/onigirazu-cfg/onigirazu.git
cd onigirazu

# Install dependencies
go mod download

# Build
go build -o onigirazu cmd/onigirazu/main.go

# Install
sudo mv onigirazu /usr/local/bin/

# Verify installation
onigirazu --version
```

### Build with Make

```bash
# Build
make build

# Install
make install

# Test
make test

# Clean
make clean
```

---

## 🐧 Linux Installation

### Ubuntu/Debian

```bash
# Download and install
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Verify installation
onigirazu --version
```

### RHEL/CentOS

```bash
# Download and install
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Verify installation
onigirazu --version
```

### Arch Linux

```bash
# Using AUR
yay -S onigirazu

# Or manual installation
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

### Alpine Linux

```bash
# Download and install
wget https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

---

## 🍎 macOS Installation

### Intel Macs

```bash
# Download and install
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_x86_64.tar.gz
tar -xzf onigirazu_Darwin_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Verify installation
onigirazu --version
```

### Apple Silicon Macs

```bash
# Download and install
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_arm64.tar.gz
tar -xzf onigirazu_Darwin_arm64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Verify installation
onigirazu --version
```

### Using Homebrew

```bash
# Install via Homebrew
brew install onigirazu

# Verify installation
onigirazu --version
```

---

## 🪟 Windows Installation

### PowerShell

```powershell
# Download and install
Invoke-WebRequest -Uri "https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Windows_x86_64.zip" -OutFile "onigirazu.zip"
Expand-Archive -Path "onigirazu.zip" -DestinationPath "C:\Program Files\Onigirazu"
$env:PATH += ";C:\Program Files\Onigirazu"

# Verify installation
onigirazu --version
```

### Command Prompt

```cmd
# Download and install
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Windows_x86_64.zip
unzip onigirazu_Windows_x86_64.zip
move onigirazu.exe C:\Program Files\Onigirazu\
set PATH=%PATH%;C:\Program Files\Onigirazu

# Verify installation
onigirazu --version
```

### Using Chocolatey

```powershell
# Install via Chocolatey
choco install onigirazu

# Verify installation
onigirazu --version
```

---

## 🐳 Docker Installation

### Using Docker Hub

```bash
# Pull image
docker pull onigirazu/onigirazu:latest

# Run container
docker run --rm -it onigirazu/onigirazu:latest --version
```

### Using GitHub Container Registry

```bash
# Pull image
docker pull ghcr.io/onigirazu-cfg/onigirazu:latest

# Run container
docker run --rm -it ghcr.io/onigirazu-cfg/onigirazu:latest --version
```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  onigirazu:
    image: onigirazu/onigirazu:latest
    volumes:
      - ./inventory.yml:/app/inventory.yml
      - ./playbooks:/app/playbooks
    working_dir: /app
    command: ["--help"]
```

---

## ☁️ Cloud Installation

### AWS EC2

```bash
# Download and install
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Verify installation
onigirazu --version
```

### Google Cloud Platform

```bash
# Download and install
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Verify installation
onigirazu --version
```

### Azure

```bash
# Download and install
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Verify installation
onigirazu --version
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

### SSH Configuration

```bash
# Configure SSH
ssh-keygen -t rsa -b 4096 -C "your_email@example.com"
ssh-copy-id user@host

# Test SSH connection
ssh user@host
```

---

## ✅ Verification

### Check Installation

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

### Test Installation

```bash
# Create test inventory
cat > test-inventory.yml << EOF
all:
  hosts:
    localhost:
      ansible_connection: local
EOF

# Test ping
onigirazu run localhost -m ping -i test-inventory.yml

# Test package module
onigirazu run localhost -m package name=curl state=present -i test-inventory.yml
```

---

## 🚨 Troubleshooting

### Common Issues

#### Permission Denied
```bash
# Check permissions
ls -la /usr/local/bin/onigirazu

# Fix permissions
sudo chmod +x /usr/local/bin/onigirazu
```

#### Command Not Found
```bash
# Check PATH
echo $PATH

# Add to PATH
export PATH=$PATH:/usr/local/bin

# Or create symlink
sudo ln -s /usr/local/bin/onigirazu /usr/bin/onigirazu
```

#### Version Mismatch
```bash
# Check Go version
go version

# Update Go
# Follow Go installation guide for your platform
```

### Debug Installation

```bash
# Check installation
which onigirazu

# Check version
onigirazu --version

# Check modules
onigirazu --list-modules

# Test connectivity
onigirazu run localhost -m ping -i test-inventory.yml
```

---

## 🔄 Updates

### Update Onigirazu

```bash
# Download latest version
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Verify update
onigirazu --version
```

### Using Package Managers

```bash
# Update via package manager
sudo apt update && sudo apt upgrade onigirazu
sudo yum update onigirazu
brew upgrade onigirazu
```

---

## 🗑️ Uninstallation

### Remove Binary

```bash
# Remove binary
sudo rm /usr/local/bin/onigirazu

# Or if installed via package manager
sudo apt remove onigirazu
sudo yum remove onigirazu
brew uninstall onigirazu
```

### Clean Configuration

```bash
# Remove configuration
rm -rf ~/.onigirazu

# Remove state files
rm -f .onigirazu-state
```

---

## 📚 Next Steps

### After Installation

1. **Create inventory** - Define your hosts
2. **Test connectivity** - Verify SSH access
3. **Run first command** - Execute your first task
4. **Explore modules** - Learn available modules

### Learn More

- [Quick Start](Quick-Start) - Getting started guide
- [Natural Language Commands](Natural-Language-Commands) - Command syntax
- [Ad-hoc Commands](Ad-hoc-Commands) - Quick operations
- [Modules](Modules) - Module reference

---

## 🎯 Summary

### Installation Checklist

1. **✅ Choose installation method** - Binary, package manager, or source
2. **✅ Download and install** - Follow platform-specific steps
3. **✅ Verify installation** - Check version and modules
4. **✅ Configure SSH** - Set up remote access
5. **✅ Test installation** - Run first commands

### Supported Platforms

- **Linux** - Ubuntu, Debian, RHEL, CentOS, Arch, Alpine
- **macOS** - Intel and Apple Silicon
- **Windows** - PowerShell and Command Prompt
- **Docker** - Containerized deployment
- **Cloud** - AWS, GCP, Azure

---

**📦 Onigirazu is now ready to use!**

