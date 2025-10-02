# Release Process

This document describes the release process for Onigirazu.

## Automated Release Process

Onigirazu uses [GoReleaser](https://goreleaser.com/) for automated releases. The release process is triggered automatically when a new tag is pushed to the repository.

### Supported Platforms

Each release includes pre-built binaries for the following platforms:

#### Linux

- **x86_64** (amd64) - Most common 64-bit Linux systems
- **ARM64** (aarch64) - 64-bit ARM systems (Raspberry Pi 4, AWS Graviton, etc.)
- **ARMv6** - Older Raspberry Pi models (Pi 1, Zero)
- **ARMv7** - Raspberry Pi 2, 3
- **i386** (32-bit) - Legacy 32-bit systems

#### macOS (Darwin)

- **x86_64** - Intel-based Macs
- **ARM64** - Apple Silicon (M1, M2, M3, etc.)

#### Windows

- **x86_64** - 64-bit Windows
- **i386** - 32-bit Windows

#### BSD Systems

- **FreeBSD**: x86_64, i386
- **OpenBSD**: x86_64, i386
- **NetBSD**: x86_64, i386

### Package Formats

The following package formats are automatically generated:

- **DEB** - Debian, Ubuntu, Linux Mint, etc.
- **RPM** - RHEL, CentOS, Fedora, openSUSE, etc.
- **APK** - Alpine Linux
- **Arch Linux** - Arch, Manjaro, etc.
- **TAR.GZ** - Universal archive format
- **ZIP** - Windows archive format

### Docker Images

Multi-architecture Docker images are built and published to:

- **Docker Hub**: `onigirazu/onigirazu`
- **GitHub Container Registry**: `ghcr.io/onigirazu-cfg/onigirazu`

Supported architectures:

- `linux/amd64`
- `linux/arm64`

## Creating a New Release

### Prerequisites

1. **Configure GitHub secrets** (first time only):
   - See [Quick Setup Guide (RU)](./GHCR_QUICK_SETUP_RU.md) for step-by-step instructions
   - See [Detailed Docker/GHCR Setup](./DOCKER_GHCR_SETUP.md) for comprehensive documentation
   - Required: `GH_TOKEN` with `repo` and `write:packages` permissions
   - Optional: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN` for Docker Hub publishing
   - Optional: `COSIGN_PRIVATE_KEY`, `COSIGN_PASSWORD` for artifact signing

2. Ensure all tests pass:

   ```bash
   go test ./...
   ```

3. Update version in relevant files if needed

4. Update CHANGELOG.md with release notes

### Release Steps

1. **Create and push a new tag:**

   ```bash
   # For a new version (e.g., v1.2.3)
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

2. **GitHub Actions will automatically:**
   - Run full test suite
   - Run security scans
   - Build binaries for all platforms
   - Create packages (DEB, RPM, APK, etc.)
   - Build and push Docker images
   - Create GitHub release with all artifacts
   - Generate checksums and SBOMs
   - Sign artifacts (if configured)

3. **Monitor the release:**
   - Go to [GitHub Actions](https://github.com/onigirazu-cfg/onigirazu/actions)
   - Check the "Release" workflow
   - Verify all jobs complete successfully

4. **Verify the release:**
   - Check [GitHub Releases](https://github.com/onigirazu-cfg/onigirazu/releases)
   - Verify all artifacts are present
   - Test download and installation on different platforms

## Release Types

### Stable Release

```bash
git tag -a v1.2.3 -m "Release v1.2.3"
```

### Pre-release (Beta, RC, Alpha)

```bash
git tag -a v1.2.3-beta.1 -m "Release v1.2.3-beta.1"
git tag -a v1.2.3-rc.1 -m "Release v1.2.3-rc.1"
git tag -a v1.2.3-alpha.1 -m "Release v1.2.3-alpha.1"
```

Pre-releases are automatically marked as "pre-release" on GitHub.

## Manual Release (Emergency)

If automated release fails, you can trigger it manually:

1. **Via GitHub UI:**
   - Go to Actions → Release workflow
   - Click "Run workflow"
   - Enter the tag name
   - Click "Run workflow"

2. **Via GoReleaser locally:**

   ```bash
   # Install GoReleaser
   brew install goreleaser

   # Create a snapshot release (no push)
   goreleaser release --snapshot --clean

   # Create a real release (requires proper setup)
   export GITHUB_TOKEN="your-token"
   goreleaser release --clean
   ```

## Testing Releases

### Test Snapshot Build Locally

```bash
# Build snapshot without releasing
goreleaser release --snapshot --clean

# Check dist/ directory for artifacts
ls -lh dist/
```

### Test Installation

#### Linux (DEB)

```bash
wget https://github.com/onigirazu-cfg/onigirazu/releases/download/v1.2.3/onigirazu_1.2.3_linux_amd64.deb
sudo dpkg -i onigirazu_1.2.3_linux_amd64.deb
onigirazu --version
```

#### macOS (Homebrew)

```bash
brew install onigirazu-cfg/tap/onigirazu
onigirazu --version
```

#### Docker

```bash
docker pull onigirazu/onigirazu:v1.2.3
docker run --rm onigirazu/onigirazu:v1.2.3 --version
```

## Release Checklist

Before creating a release:

- [ ] All tests pass (`go test ./...`)
- [ ] Security scan passes (`gosec ./...`)
- [ ] CHANGELOG.md is updated
- [ ] Version numbers are correct
- [ ] Documentation is up to date
- [ ] Examples are tested
- [ ] Breaking changes are documented

After release:

- [ ] Verify all artifacts are present on GitHub
- [ ] Test installation on at least 2 platforms
- [ ] Verify Docker images are available
- [ ] Update documentation if needed
- [ ] Announce release (if major version)

## Troubleshooting

### Release workflow fails

1. Check GitHub Actions logs
2. Verify secrets are configured:
   - `GH_TOKEN` - GitHub token with repo access
   - `DOCKERHUB_USERNAME` - Docker Hub username (optional)
   - `DOCKERHUB_TOKEN` - Docker Hub token (optional)
   - `COSIGN_PRIVATE_KEY` - Signing key (optional)
   - `COSIGN_PASSWORD` - Signing password (optional)
   - `FURY_TOKEN` - Fury.io token (optional)

### Binary doesn't work on target platform

1. Check if platform is in the supported list
2. Verify CGO is disabled (`CGO_ENABLED=0`)
3. Check if binary is statically linked: `ldd onigirazu`

### Docker image fails to build

1. Check Dockerfile syntax
2. Verify base image is available
3. Check build arguments are correct

## Version Numbering

Onigirazu follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version for incompatible API changes
- **MINOR** version for new functionality (backwards compatible)
- **PATCH** version for bug fixes (backwards compatible)

Examples:

- `v1.0.0` - Initial stable release
- `v1.1.0` - New features added
- `v1.1.1` - Bug fixes
- `v2.0.0` - Breaking changes

## Release Artifacts

Each release includes:

1. **Source code** (zip, tar.gz)
2. **Binaries** for all supported platforms
3. **Packages** (DEB, RPM, APK, Arch)
4. **Docker images** (multi-arch)
5. **Checksums** (SHA256)
6. **SBOMs** (Software Bill of Materials)
7. **Signatures** (if configured)

## Support

For questions about releases:

- Open an issue: <https://github.com/onigirazu-cfg/onigirazu/issues>
- Check documentation: <https://github.com/onigirazu-cfg/onigirazu/tree/main/docs>
