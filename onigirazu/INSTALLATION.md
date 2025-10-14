# Installation Guide

Onigirazu provides multiple installation methods to suit different environments and preferences.

## 📦 Pre-built Binaries (Recommended)

Download the latest release for your platform from [GitHub Releases](https://github.com/onigirazu-cfg/onigirazu/releases):

### Linux

```bash
# Download for Linux x86_64
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Download for Linux ARM64
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_arm64.tar.gz
tar -xzf onigirazu_Linux_arm64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

### macOS

```bash
# Download for macOS x86_64 (Intel)
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_x86_64.tar.gz
tar -xzf onigirazu_Darwin_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/

# Download for macOS ARM64 (Apple Silicon)
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_arm64.tar.gz
tar -xzf onigirazu_Darwin_arm64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

### Windows

1. Download `onigirazu_Windows_x86_64.zip` from the [releases page](https://github.com/onigirazu-cfg/onigirazu/releases)
2. Extract the ZIP file
3. Add the extracted directory to your PATH environment variable

## 🍺 Homebrew (macOS/Linux)

```bash
# Add the tap
brew tap onigirazu-cfg/tap

# Install Onigirazu
brew install onigirazu

# Update to latest version
brew upgrade onigirazu
```

## 📋 Package Managers

### Debian/Ubuntu (DEB)

```bash
# Download and install DEB package
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_linux_amd64.deb
sudo dpkg -i onigirazu_linux_amd64.deb
```

### Red Hat/CentOS/Fedora (RPM)

```bash
# Download and install RPM package
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_linux_amd64.rpm
sudo rpm -i onigirazu_linux_amd64.rpm
```

### Alpine Linux (APK)

```bash
# Download and install APK package
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_linux_amd64.apk
sudo apk add --allow-untrusted onigirazu_linux_amd64.apk
```

### Arch Linux

```bash
# Download and install Arch package
curl -LO https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_linux_amd64.pkg.tar.xz
sudo pacman -U onigirazu_linux_amd64.pkg.tar.xz
```

## 🐳 Docker

### Docker Hub

```bash
# Run with Docker
docker run --rm -v $(pwd):/workspace onigirazu/onigirazu:latest --version

# Use in your own Dockerfile
FROM onigirazu/onigirazu:latest
```

### GitHub Container Registry

```bash
# Run with Docker
docker run --rm -v $(pwd):/workspace ghcr.io/onigirazu-cfg/onigirazu:latest --version

# Use in your own Dockerfile
FROM ghcr.io/onigirazu-cfg/onigirazu:latest
```

## 🔧 Build from Source

### Prerequisites

- Go 1.21 or later
- Git

### Build Steps

```bash
# Clone the repository
git clone https://github.com/onigirazu-cfg/onigirazu.git
cd onigirazu

# Build the binary
go build -o onigirazu ./cmd/onigirazu

# Install to system PATH
sudo mv onigirazu /usr/local/bin/
```

### Development Build

```bash
# Install directly with Go
go install github.com/onigirazu-cfg/onigirazu/cmd/onigirazu@latest
```

## ✅ Verify Installation

After installation, verify that Onigirazu is working correctly:

```bash
# Check version
onigirazu --version

# List available modules
onigirazu --list-modules

# Show help
onigirazu --help
```

## 🔄 Updating

### Pre-built Binaries

Simply download and replace the binary with the latest version from the releases page.

### Homebrew

```bash
brew upgrade onigirazu
```

### Package Managers

Use your system's package manager update commands after downloading the latest package.

### Docker

```bash
docker pull onigirazu/onigirazu:latest
# or
docker pull ghcr.io/onigirazu-cfg/onigirazu:latest
```

## 🆘 Troubleshooting

### Permission Issues

If you encounter permission issues, ensure the binary is executable:

```bash
chmod +x onigirazu
```

### PATH Issues

Make sure `/usr/local/bin` is in your PATH:

```bash
echo $PATH
export PATH="/usr/local/bin:$PATH"
```

### Dependencies

Onigirazu binaries are statically compiled and have no external dependencies. However, for full functionality, you may want:

- SSH client (for remote connections)
- Git (for git module operations)

## 📚 Next Steps

After installation, check out:

- [Getting Started Guide](docs/getting-started.md)
- [Configuration Reference](docs/configuration.md)
- [Module Documentation](docs/modules.md)
- [Examples](examples/)

For support, please visit our [GitHub Issues](https://github.com/onigirazu-cfg/onigirazu/issues) page.
