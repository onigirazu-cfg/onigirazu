# Installation & Configuration Guide - v1.55.1

**Platform-Specific Installation and Configuration Setup**

**Release:** v1.55.1
**Date:** January 29, 2025
**Status:** ✅ Complete

---

## 📋 Table of Contents

1. [Quick Start](#quick-start)
2. [Linux Installation](#linux-installation)
3. [macOS Installation](#macos-installation)
4. [Windows Installation](#windows-installation)
5. [Docker & Kubernetes](#docker--kubernetes)
6. [FreeBSD & Other UNIX](#freebsd--other-unix)
7. [Configuration Priority](#configuration-priority)
8. [Verification](#verification)
9. [Post-Installation Checks](#post-installation-checks)

---

## Quick Start

Choose your platform:

| Platform | Time | Difficulty |
|----------|------|-----------|
| [Linux (Recommended)](#linux-installation) | 2 min | Easy |
| [macOS](#macos-installation) | 3 min | Easy |
| [Windows](#windows-installation) | 3 min | Easy |
| [Docker](#docker--kubernetes) | 2 min | Easy |
| [FreeBSD](#freebsd--other-unix) | 3 min | Medium |

---

## Linux Installation

### 1️⃣ Install via Package Manager (Recommended)

#### Debian/Ubuntu

```bash
# Add the repository (if needed)
curl -s https://apt.onigirazu-cfg.com/repo.gpg | sudo apt-key add -
echo "deb [signed-by=/usr/share/keyrings/onigirazu.gpg] https://apt.onigirazu-cfg.com focal main" | \
  sudo tee /etc/apt/sources.list.d/onigirazu.list

# Install
sudo apt update
sudo apt install -y onigirazu

# Verify installation
onigirazu --version
```

**What's Installed:**

```
✓ /usr/bin/onigirazu                          (main binary)
✓ /usr/share/onigirazu/onigirazu.default.yml (reference config)
✓ /usr/share/onigirazu/examples/              (4 templates)
✓ /etc/onigirazu/                             (config directory)
✓ /var/log/onigirazu/                         (log directory)
✓ /etc/onigirazu/onigirazu.yml                (auto-generated)
```

**Post-Installation Output:**

```
✓ Onigirazu has been installed successfully!

📋 Available configuration templates:
  - /usr/share/onigirazu/onigirazu.default.yml
  - /usr/share/onigirazu/examples/onigirazu.minimal.yml
  - /usr/share/onigirazu/examples/onigirazu.production.yml
  - /usr/share/onigirazu/examples/onigirazu.docker.yml

✓ Default config created: /etc/onigirazu/onigirazu.yml
✓ Ready to use!
```

#### Red Hat / Fedora / CentOS

```bash
# Add repository
sudo dnf config-manager --add-repo https://rpm.onigirazu-cfg.com/repo.repo

# Install
sudo dnf install -y onigirazu

# Verify
onigirazu --version
```

### 2️⃣ Install from Binary

```bash
# Download latest release
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz

# Extract
tar -xzf onigirazu_Linux_x86_64.tar.gz

# Install
sudo mv onigirazu /usr/local/bin/

# Verify
onigirazu --version
```

### 3️⃣ Configure Manually

```bash
# Create config directory
sudo mkdir -p /etc/onigirazu

# Choose a template
# Option 1: Use minimal
sudo cat > /etc/onigirazu/onigirazu.yml << 'EOF'
max_concurrency: 10
default_timeout: 30s
log_level: info
retry_attempts: 3
retry_delay: 5s
EOF

# Option 2: Download default
curl -LO https://raw.githubusercontent.com/onigirazu-cfg/onigirazu/main/examples/onigirazu.default.yml
sudo mv onigirazu.default.yml /etc/onigirazu/onigirazu.yml

# Set permissions
sudo chmod 644 /etc/onigirazu/onigirazu.yml

# Create log directory
sudo mkdir -p /var/log/onigirazu
sudo chmod 755 /var/log/onigirazu

# Verify
onigirazu -c /etc/onigirazu/onigirazu.yml --version
```

### Linux Configuration Locations

**Priority order** (first found is used):

1. **CLI Flag** (highest priority):

   ```bash
   onigirazu -c /path/to/config.yml playbook.yml
   ```

2. **Working Directory**:

   ```
   ~/my-project/
   ├── onigirazu.yml           ← Found here
   └── playbook.yml
   ```

3. **User Home**:

   ```
   ~/.onigirazu/onigirazu.yml
   ```

4. **System-wide**:

   ```
   /etc/onigirazu/onigirazu.yml
   ```

5. **Defaults** (lowest priority):

   ```
   Built-in defaults used if no file found
   ```

---

## macOS Installation

### 1️⃣ Install via Homebrew (Recommended)

```bash
# Add tap
brew tap onigirazu-cfg/onigirazu

# Install
brew install onigirazu

# Verify
onigirazu --version
```

**What's Installed:**

```
✓ /usr/local/bin/onigirazu
✓ Configuration: ~/.onigirazu/ (user) or /etc/onigirazu/ (system)
```

### 2️⃣ Install from Binary

```bash
# Download latest release (Intel)
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_x86_64.tar.gz

# Download latest release (Apple Silicon)
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_arm64.tar.gz

# Extract
tar -xzf onigirazu_Darwin_*.tar.gz

# Install to /usr/local/bin (may require sudo)
sudo mv onigirazu /usr/local/bin/

# Or install to user bin (no sudo needed)
mkdir -p ~/.local/bin
mv onigirazu ~/.local/bin/
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# Verify
onigirazu --version
```

### 3️⃣ macOS Configuration

```bash
# Create user configuration directory
mkdir -p ~/.onigirazu

# Create config file
cat > ~/.onigirazu/onigirazu.yml << 'EOF'
max_concurrency: 10
default_timeout: 30s
log_level: info
retry_attempts: 3
retry_delay: 5s
EOF

# Or system-wide (requires sudo)
sudo mkdir -p /etc/onigirazu
sudo cat > /etc/onigirazu/onigirazu.yml << 'EOF'
max_concurrency: 10
default_timeout: 30s
log_level: info
retry_attempts: 3
retry_delay: 5s
EOF

# Verify
onigirazu --version
```

### macOS Configuration Locations

1. **CLI Flag**:

   ```bash
   onigirazu -c /path/to/config.yml playbook.yml
   ```

2. **Working Directory**:

   ```
   ~/project/onigirazu.yml
   ```

3. **User Home**:

   ```
   ~/.onigirazu/onigirazu.yml
   ```

4. **System-wide**:

   ```
   /etc/onigirazu/onigirazu.yml
   ```

---

## Windows Installation

### 1️⃣ Install via Scoop (Recommended)

```powershell
# Install Scoop if needed
iwr -useb get.scoop.sh | iex

# Add bucket
scoop bucket add onigirazu https://github.com/onigirazu-cfg/scoop-bucket.git

# Install
scoop install onigirazu

# Verify
onigirazu --version
```

### 2️⃣ Install via Chocolatey

```powershell
# Install Chocolatey if needed
Set-ExecutionPolicy Bypass -Scope Process -Force; `
  [System.Net.ServicePointManager]::SecurityProtocol = `
  [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; `
  iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

# Install Onigirazu
choco install onigirazu -y

# Verify
onigirazu --version
```

### 3️⃣ Install from Binary

```powershell
# Download latest release
$ProgressPreference = 'SilentlyContinue'
Invoke-WebRequest `
  -Uri https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Windows_x86_64.zip `
  -OutFile onigirazu.zip

# Extract
Expand-Archive onigirazu.zip -DestinationPath .

# Move to Program Files
Move-Item onigirazu.exe "$env:ProgramFiles\Onigirazu\" -Force

# Add to PATH (if not already)
$env:Path += ";$env:ProgramFiles\Onigirazu"

# Verify
onigirazu --version
```

### 4️⃣ Windows Configuration

**User Configuration (No Admin Required):**

```powershell
# Create user config directory
New-Item -Type Directory -Path "$env:APPDATA\Onigirazu" -Force | Out-Null

# Create config file
@"
max_concurrency: 10
default_timeout: 30s
log_level: info
retry_attempts: 3
retry_delay: 5s
"@ | Out-File "$env:APPDATA\Onigirazu\onigirazu.yml" -Encoding UTF8

# Verify
onigirazu --version
```

**System Configuration (Admin Required):**

```powershell
# Run PowerShell as Administrator first!

# Create system config directory
New-Item -Type Directory -Path "$env:ProgramData\Onigirazu" -Force | Out-Null

# Create config file
@"
max_concurrency: 10
default_timeout: 30s
log_level: info
retry_attempts: 3
retry_delay: 5s
"@ | Out-File "$env:ProgramData\Onigirazu\onigirazu.yml" -Encoding UTF8

# Set permissions
icacls "$env:ProgramData\Onigirazu" /grant Users:(OI)(CI)F /T
```

### Windows Configuration Locations

1. **CLI Flag**:

   ```powershell
   onigirazu -c C:\Users\YourUser\AppData\Roaming\Onigirazu\onigirazu.yml playbook.yml
   ```

2. **Working Directory**:

   ```
   C:\Users\YourUser\project\onigirazu.yml
   ```

3. **User Config** (highest priority for users):

   ```
   %APPDATA%\Onigirazu\onigirazu.yml
   ```

4. **System Config** (requires admin):

   ```
   %ProgramData%\Onigirazu\onigirazu.yml
   ```

---

## Docker & Kubernetes

### Docker Image

```bash
# Pull image
docker pull onigirazu/onigirazu:latest

# Run with local playbook
docker run \
  -v $(pwd):/playbooks \
  -v $(pwd)/onigirazu.yml:/etc/onigirazu/onigirazu.yml \
  onigirazu/onigirazu \
  -c /etc/onigirazu/onigirazu.yml \
  /playbooks/playbook.yml

# Run with docker template (optimized)
docker run \
  -v $(pwd):/playbooks \
  onigirazu/onigirazu \
  -c /usr/share/onigirazu/examples/onigirazu.docker.yml \
  /playbooks/playbook.yml
```

### Docker Compose

```yaml
version: '3.8'

services:
  automation:
    image: onigirazu/onigirazu:latest
    container_name: onigirazu-runner
    volumes:
      - ./playbooks:/playbooks
      - ./config/onigirazu.yml:/etc/onigirazu/onigirazu.yml
    environment:
      - LOG_LEVEL=info
    command:
      - -c
      - /etc/onigirazu/onigirazu.yml
      - /playbooks/site.yml
```

### Kubernetes Deployment

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: onigirazu-config
  namespace: default
data:
  onigirazu.yml: |
    max_concurrency: 10
    default_timeout: 30s
    log_level: info
    log_format: json
    retry_attempts: 3
    retry_delay: 5s
---
apiVersion: batch/v1
kind: Job
metadata:
  name: onigirazu-deploy
  namespace: default
spec:
  template:
    spec:
      serviceAccountName: onigirazu
      containers:
      - name: onigirazu
        image: onigirazu/onigirazu:latest
        volumeMounts:
        - name: config
          mountPath: /etc/onigirazu
        - name: playbooks
          mountPath: /playbooks
        command:
        - onigirazu
        - -c
        - /etc/onigirazu/onigirazu.yml
        - /playbooks/site.yml
      volumes:
      - name: config
        configMap:
          name: onigirazu-config
      - name: playbooks
        configMap:
          name: playbook-data
      restartPolicy: OnFailure
```

---

## FreeBSD & Other UNIX

### FreeBSD Installation

```bash
# Via pkg (if available)
sudo pkg install onigirazu

# Or build from source
cd /usr/ports/sysutils/onigirazu
sudo make install clean

# Verify
onigirazu --version
```

### Generic UNIX Installation

```bash
# Download latest release
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_FreeBSD_amd64.tar.gz

# Extract
tar -xzf onigirazu_FreeBSD_amd64.tar.gz

# Install
sudo mv onigirazu /usr/local/bin/

# Configure
sudo mkdir -p /etc/onigirazu
sudo cat > /etc/onigirazu/onigirazu.yml << 'EOF'
max_concurrency: 10
default_timeout: 30s
log_level: info
retry_attempts: 3
retry_delay: 5s
EOF

# Verify
onigirazu --version
```

---

## Configuration Priority

Onigirazu searches for configuration in this order:

```
┌─────────────────────────────────────────────────┐
│ 1. CLI Flag (Highest Priority)                  │
│    onigirazu -c /explicit/path/config.yml       │
└────────────────┬────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────┐
│ 2. Working Directory                            │
│    ./onigirazu.yml (if in same dir as playbook) │
└────────────────┬────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────┐
│ 3. User Home Directory                          │
│    ~/.onigirazu/onigirazu.yml (Linux/macOS)     │
│    %APPDATA%\Onigirazu\ (Windows)               │
└────────────────┬────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────┐
│ 4. System Directory                             │
│    /etc/onigirazu/onigirazu.yml (Linux/macOS)   │
│    %ProgramData%\Onigirazu\ (Windows)           │
└────────────────┬────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────┐
│ 5. Built-in Defaults (Lowest Priority)          │
│    Compiled into binary                         │
└─────────────────────────────────────────────────┘
```

### Using Configuration Priority

```bash
# Example: Project-specific config
~/projects/app/
├── onigirazu.yml         ← Found by priority #2
├── playbook.yml
└── roles/

# Run from project directory
cd ~/projects/app
onigirazu playbook.yml    # Uses ./onigirazu.yml

# Override with explicit path
onigirazu -c /etc/onigirazu/production.yml playbook.yml
```

---

## Verification

### Verify Installation

```bash
# Check version
onigirazu --version

# Check configuration path
onigirazu -c /etc/onigirazu/onigirazu.yml --version

# List available modules
onigirazu --list-modules

# Test configuration
onigirazu -c /etc/onigirazu/onigirazu.yml --help
```

### Test Configuration

```bash
# Create test playbook
cat > test.yml << 'EOF'
---
- hosts: localhost
  tasks:
    - name: Test task
      debug:
        msg: "Onigirazu is working!"
EOF

# Run test
onigirazu -c /etc/onigirazu/onigirazu.yml test.yml

# Expected output:
# ✓ Test task completed successfully
```

---

## Post-Installation Checks

### Linux Checklist

```bash
# ✓ Check installation
which onigirazu

# ✓ Verify permissions
ls -la /usr/bin/onigirazu

# ✓ Check config files
ls -la /usr/share/onigirazu/examples/

# ✓ Verify system config
cat /etc/onigirazu/onigirazu.yml

# ✓ Check log directory
ls -la /var/log/onigirazu/

# ✓ Test run
onigirazu --version
```

### macOS Checklist

```bash
# ✓ Check installation
which onigirazu

# ✓ Check user config
ls -la ~/.onigirazu/

# ✓ Test version
onigirazu --version

# ✓ Test with custom config
onigirazu -c ~/.onigirazu/onigirazu.yml --version
```

### Windows Checklist

```powershell
# ✓ Check installation
Get-Command onigirazu.exe

# ✓ Check user config
Test-Path "$env:APPDATA\Onigirazu\onigirazu.yml"

# ✓ Test version
onigirazu --version

# ✓ Test with custom config
onigirazu -c "$env:APPDATA\Onigirazu\onigirazu.yml" --version
```

### Docker Checklist

```bash
# ✓ Check image
docker pull onigirazu/onigirazu

# ✓ List images
docker images | grep onigirazu

# ✓ Test run
docker run onigirazu/onigirazu --version

# ✓ Test with config
docker run -v $(pwd)/config.yml:/etc/onigirazu/onigirazu.yml \
  onigirazu/onigirazu --version
```

---

## Troubleshooting

### "Command not found"

```bash
# Linux/macOS
export PATH="/usr/local/bin:$PATH"
onigirazu --version

# Windows PowerShell
$env:Path += ";$env:ProgramFiles\Onigirazu"
onigirazu --version
```

### "Config file not found"

```bash
# Check current directory
pwd
ls onigirazu.yml

# Check home directory
ls ~/.onigirazu/

# Check system
sudo ls /etc/onigirazu/

# Use explicit path
onigirazu -c /etc/onigirazu/onigirazu.yml playbook.yml
```

### Permission Errors

```bash
# Linux/macOS
sudo chown -R $(whoami) /etc/onigirazu/
sudo chmod 755 /etc/onigirazu/
sudo chmod 644 /etc/onigirazu/onigirazu.yml

# Windows (run as Administrator)
icacls "C:\ProgramData\Onigirazu" /grant:r "%USERNAME%:(OI)(CI)F" /T
```

See [TROUBLESHOOTING_CONFIG.md](TROUBLESHOOTING_CONFIG.md) for more solutions.
