# Supported Platforms

Onigirazu is built for multiple platforms and architectures, ensuring broad compatibility across different systems.

## Binary Distributions

### Linux

Onigirazu provides native binaries for all major Linux architectures:

| Architecture | Description | Use Cases |
|--------------|-------------|-----------|
| **x86_64** (amd64) | 64-bit Intel/AMD | Most common desktop and server systems |
| **ARM64** (aarch64) | 64-bit ARM | Raspberry Pi 4+, AWS Graviton, Apple M-series under Linux |
| **ARMv7** | 32-bit ARM v7 | Raspberry Pi 2, 3 |
| **ARMv6** | 32-bit ARM v6 | Raspberry Pi 1, Zero, Zero W |
| **i386** | 32-bit x86 | Legacy systems |

#### Package Formats

- **DEB** - Debian, Ubuntu, Linux Mint, Pop!_OS, Elementary OS
- **RPM** - RHEL, CentOS, Fedora, openSUSE, Amazon Linux
- **APK** - Alpine Linux
- **Arch** - Arch Linux, Manjaro, EndeavourOS
- **TAR.GZ** - Universal format for all distributions

### macOS (Darwin)

| Architecture | Description | Supported Versions |
|--------------|-------------|-------------------|
| **x86_64** | Intel-based Macs | macOS 10.13+ |
| **ARM64** | Apple Silicon (M1/M2/M3) | macOS 11.0+ |

Universal binaries work on both Intel and Apple Silicon Macs.

### Windows

| Architecture | Description | Supported Versions |
|--------------|-------------|-------------------|
| **x86_64** | 64-bit Windows | Windows 10, 11, Server 2016+ |
| **i386** | 32-bit Windows | Windows 7+, Server 2008+ |

### BSD Systems

| OS | Architectures | Notes |
|----|---------------|-------|
| **FreeBSD** | x86_64, i386 | FreeBSD 12.0+ |
| **OpenBSD** | x86_64, i386 | OpenBSD 6.8+ |
| **NetBSD** | x86_64, i386 | NetBSD 9.0+ |

## Docker Images

Multi-architecture Docker images are available for:

- **linux/amd64** - Standard x86_64 systems
- **linux/arm64** - ARM64 systems (AWS Graviton, Raspberry Pi 4+, etc.)

### Registries

- **Docker Hub**: `onigirazu/onigirazu`
- **GitHub Container Registry**: `ghcr.io/onigirazu-cfg/onigirazu`

## Installation Methods

### Package Managers

#### Homebrew (macOS/Linux)

```bash
brew install onigirazu-cfg/tap/onigirazu
```

#### APT (Debian/Ubuntu)

```bash
wget https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_linux_amd64.deb
sudo dpkg -i onigirazu_linux_amd64.deb
```

#### YUM/DNF (RHEL/CentOS/Fedora)

```bash
wget https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_linux_amd64.rpm
sudo rpm -i onigirazu_linux_amd64.rpm
```

#### APK (Alpine)

```bash
wget https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_linux_amd64.apk
sudo apk add --allow-untrusted onigirazu_linux_amd64.apk
```

#### Pacman (Arch Linux)

```bash
wget https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_linux_amd64.pkg.tar.zst
sudo pacman -U onigirazu_linux_amd64.pkg.tar.zst
```

### Go Install

```bash
go install github.com/onigirazu-cfg/onigirazu/cmd/onigirazu@latest
```

### Docker

```bash
docker pull onigirazu/onigirazu:latest
docker run --rm onigirazu/onigirazu:latest --version
```

## Platform-Specific Notes

### Linux

#### ARM Systems (Raspberry Pi)

- **Raspberry Pi 4, 400, 5**: Use ARM64 binary
- **Raspberry Pi 2, 3**: Use ARMv7 binary
- **Raspberry Pi 1, Zero, Zero W**: Use ARMv6 binary

Example for Raspberry Pi 4:

```bash
wget https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Linux_arm64.tar.gz
tar -xzf onigirazu_Linux_arm64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

#### Cloud Platforms

- **AWS Graviton**: Use ARM64 binary
- **AWS x86**: Use x86_64 binary
- **Google Cloud**: Use x86_64 binary
- **Azure**: Use x86_64 binary

### macOS

#### Apple Silicon (M1/M2/M3)

The ARM64 binary runs natively on Apple Silicon:

```bash
# Download ARM64 version
wget https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_arm64.tar.gz
tar -xzf onigirazu_Darwin_arm64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

#### Intel Macs

```bash
# Download x86_64 version
wget https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Darwin_x86_64.tar.gz
tar -xzf onigirazu_Darwin_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

### Windows

#### PowerShell Installation

```powershell
# Download latest release
Invoke-WebRequest -Uri "https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Windows_x86_64.zip" -OutFile "onigirazu.zip"

# Extract
Expand-Archive -Path "onigirazu.zip" -DestinationPath "C:\Program Files\Onigirazu"

# Add to PATH
$env:Path += ";C:\Program Files\Onigirazu"
```

#### Chocolatey (Coming Soon)

```powershell
choco install onigirazu
```

### BSD Systems

#### FreeBSD

```bash
# Using pkg (if available in ports)
pkg install onigirazu

# Or manual installation
fetch https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/onigirazu_Freebsd_x86_64.tar.gz
tar -xzf onigirazu_Freebsd_x86_64.tar.gz
sudo mv onigirazu /usr/local/bin/
```

## Verification

### Check Binary Architecture

#### Linux/macOS

```bash
file $(which onigirazu)
```

#### Check Version

```bash
onigirazu --version
```

### Verify Checksum

Download checksums file:

```bash
wget https://github.com/onigirazu-cfg/onigirazu/releases/latest/download/checksums.txt
```

Verify your download:

```bash
# Linux/macOS
sha256sum -c checksums.txt --ignore-missing

# macOS alternative
shasum -a 256 -c checksums.txt
```

## Unsupported Platforms

The following platforms are **not** currently supported:

- **Solaris/Illumos** - May work but not tested
- **AIX** - Not supported
- **Plan 9** - Not supported
- **Android** - Not officially supported (may work with Termux)
- **iOS** - Not supported

If you need support for a specific platform, please [open an issue](https://github.com/onigirazu-cfg/onigirazu/issues).

## Building from Source

If your platform is not supported, you can build from source:

```bash
# Clone repository
git clone https://github.com/onigirazu-cfg/onigirazu.git
cd onigirazu

# Build
go build -o onigirazu ./cmd/onigirazu

# Install
sudo mv onigirazu /usr/local/bin/
```

### Cross-Compilation

Build for a different platform:

```bash
# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o onigirazu-linux-arm64 ./cmd/onigirazu

# Windows x86_64
GOOS=windows GOARCH=amd64 go build -o onigirazu.exe ./cmd/onigirazu

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o onigirazu-darwin-arm64 ./cmd/onigirazu
```

## System Requirements

### Minimum Requirements

- **RAM**: 128 MB
- **Disk**: 50 MB for binary + space for logs and state
- **CPU**: Any modern CPU (x86, ARM, etc.)
- **OS**: Linux 2.6.32+, macOS 10.13+, Windows 7+

### Recommended Requirements

- **RAM**: 512 MB+
- **Disk**: 500 MB+ (for logs, cache, and state)
- **CPU**: Multi-core processor
- **Network**: Internet connection for remote hosts

## Performance Notes

### Architecture Performance

- **x86_64**: Best performance on Intel/AMD systems
- **ARM64**: Excellent performance on modern ARM systems
- **ARMv7/ARMv6**: Good performance on Raspberry Pi, may be slower for large playbooks

### Platform-Specific Optimizations

- All binaries are statically linked (no external dependencies)
- Binaries are stripped and optimized for size
- CGO is disabled for maximum portability

## Support Matrix

| Platform | Status | Package Formats | Docker |
|----------|--------|----------------|--------|
| Linux x86_64 | ✅ Fully Supported | DEB, RPM, APK, Arch, TAR.GZ | ✅ |
| Linux ARM64 | ✅ Fully Supported | DEB, RPM, APK, Arch, TAR.GZ | ✅ |
| Linux ARMv7 | ✅ Fully Supported | DEB, RPM, APK, Arch, TAR.GZ | ❌ |
| Linux ARMv6 | ✅ Fully Supported | DEB, RPM, APK, Arch, TAR.GZ | ❌ |
| Linux i386 | ✅ Fully Supported | DEB, RPM, APK, Arch, TAR.GZ | ❌ |
| macOS x86_64 | ✅ Fully Supported | TAR.GZ, Homebrew | ❌ |
| macOS ARM64 | ✅ Fully Supported | TAR.GZ, Homebrew | ❌ |
| Windows x86_64 | ✅ Fully Supported | ZIP | ❌ |
| Windows i386 | ✅ Fully Supported | ZIP | ❌ |
| FreeBSD x86_64 | ⚠️ Community Supported | TAR.GZ | ❌ |
| FreeBSD i386 | ⚠️ Community Supported | TAR.GZ | ❌ |
| OpenBSD x86_64 | ⚠️ Community Supported | TAR.GZ | ❌ |
| OpenBSD i386 | ⚠️ Community Supported | TAR.GZ | ❌ |
| NetBSD x86_64 | ⚠️ Community Supported | TAR.GZ | ❌ |
| NetBSD i386 | ⚠️ Community Supported | TAR.GZ | ❌ |

**Legend:**

- ✅ Fully Supported - Tested and maintained
- ⚠️ Community Supported - Built but not regularly tested
- ❌ Not Supported

## Getting Help

If you encounter issues on your platform:

1. Check the [troubleshooting guide](./TROUBLESHOOTING.md)
2. Search [existing issues](https://github.com/onigirazu-cfg/onigirazu/issues)
3. Open a [new issue](https://github.com/onigirazu-cfg/onigirazu/issues/new) with:
   - Platform and architecture (`uname -a`)
   - Onigirazu version (`onigirazu --version`)
   - Error messages and logs
