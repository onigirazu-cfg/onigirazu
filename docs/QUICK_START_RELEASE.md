# Quick Start: Creating a Release

## TL;DR

```bash
# 1. Run tests
go test ./...

# 2. Create and push tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 3. Wait for GitHub Actions to complete
# 4. Check https://github.com/onigirazu-cfg/onigirazu/releases
```

## What Gets Built

Every release automatically builds:

### 🖥️ Binaries (19 platforms)

- Linux: x86_64, ARM64, ARMv7, ARMv6, i386
- macOS: x86_64, ARM64
- Windows: x86_64, i386
- FreeBSD: x86_64, i386
- OpenBSD: x86_64, i386
- NetBSD: x86_64, i386

### 📦 Packages

- DEB (Debian/Ubuntu)
- RPM (RHEL/Fedora)
- APK (Alpine)
- Arch Linux
- TAR.GZ (Universal)
- ZIP (Windows)

### 🐳 Docker Images

- `onigirazu/onigirazu:v1.0.0`
- `ghcr.io/onigirazu-cfg/onigirazu:v1.0.0`
- Multi-arch: linux/amd64, linux/arm64

## Testing Before Release

```bash
# Test build locally (no publish)
./scripts/test-release.sh
```

## Release Types

### Stable Release

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
```

### Pre-release

```bash
git tag -a v1.0.0-beta.1 -m "Beta release"
git tag -a v1.0.0-rc.1 -m "Release candidate"
```

## Monitoring Release

1. Go to [GitHub Actions](https://github.com/onigirazu-cfg/onigirazu/actions)
2. Watch "Release" workflow
3. Check [Releases page](https://github.com/onigirazu-cfg/onigirazu/releases)

## Verification

```bash
# Download and test
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.0.0/onigirazu_Linux_x86_64.tar.gz
tar -xzf onigirazu_Linux_x86_64.tar.gz
./onigirazu --version

# Verify checksum
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.0.0/checksums.txt
sha256sum -c checksums.txt --ignore-missing
```

## Troubleshooting

### Release fails

- Check GitHub Actions logs
- Verify all tests pass locally
- Check secrets are configured

### Missing artifacts

- Wait for workflow to complete (can take 10-15 minutes)
- Check if all jobs succeeded

## Full Documentation

- [Complete Release Guide](./RELEASE.md)
- [Platform Support](./PLATFORMS.md)
- [GitHub Actions Workflow](../.github/workflows/release.yml)
- [GoReleaser Config](../.goreleaser.yml)
