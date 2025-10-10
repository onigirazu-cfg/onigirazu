# CI/CD Documentation

This document describes the Continuous Integration and Continuous Deployment (CI/CD) system for the Onigirazu project.

## Overview

Our CI/CD system is built using GitHub Actions and provides comprehensive automation for:

- **Code Quality**: Automated testing, linting, and security scanning
- **Multi-platform Builds**: Cross-platform binary compilation
- **Automated Releases**: Semantic versioning and automated release creation
- **Security**: Vulnerability scanning and dependency auditing
- **Documentation**: Automated documentation generation and updates

## Workflows

### 1. CI Workflow (`.github/workflows/ci.yml`)

**Triggers:**

- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches
- Manual trigger via workflow_dispatch

**Jobs:**

- **Test**: Runs tests across multiple Go versions (1.22, 1.23, 1.24) and OS platforms (Ubuntu, macOS, Windows)
- **Build**: Cross-platform binary compilation for Linux, Windows, and macOS
- **Lint**: Code quality checks using golangci-lint
- **Integration Tests**: End-to-end testing with built binaries
- **Docker Build**: Multi-architecture Docker image building
- **Quality Gate**: Final validation of all checks

**Features:**

- Race condition detection
- Code coverage reporting
- Benchmark execution
- Artifact uploading
- Caching for faster builds

### 2. Release Workflow (`.github/workflows/release.yml`)

**Triggers:**

- Git tags matching `v*` pattern
- Manual trigger with tag input

**Jobs:**

- **Validate**: Tag format validation and prerelease detection
- **Test**: Full test suite execution before release
- **Release**: GoReleaser execution for multi-platform releases
- **Docker**: Multi-architecture Docker image publishing
- **Notify**: Release status notifications

**Features:**

- Semantic version validation
- Prerelease detection
- Binary signing with cosign
- SBOM generation
- Multi-registry Docker publishing
- Package manager publishing (Homebrew, APT, RPM)

### 3. Security Workflow (`.github/workflows/security.yml`)

**Triggers:**

- Push to `main` or `develop` branches
- Pull requests
- Weekly schedule (Mondays at 6 AM)
- Manual trigger

**Jobs:**

- **Security Scan**: Multiple security tools (Gosec, Trivy, govulncheck, Nancy)
- **CodeQL**: GitHub's semantic code analysis
- **Dependency Review**: License and vulnerability checking for PRs

**Features:**

- SARIF report generation
- Security advisory integration
- Vulnerability database scanning
- License compliance checking

### 4. Auto Release Workflow (`.github/workflows/auto-release.yml`)

**Triggers:**

- Push to `main` branch (excluding docs and examples)
- Manual trigger with release type selection

**Jobs:**

- **Check Changes**: Analyzes commits for conventional commit patterns
- **Create Release**: Automatic version bumping and release creation

**Features:**

- Conventional commit analysis
- Semantic version calculation
- Automated changelog generation
- Release type detection (major/minor/patch)

### 5. Code Quality Workflow (`.github/workflows/code-quality.yml`)

**Triggers:**

- Push to `main` or `develop` branches
- Pull requests
- Weekly schedule (Sundays at 2 AM)
- Manual trigger

**Jobs:**

- **Code Quality**: Formatting, imports, complexity analysis
- **Test Coverage**: Coverage threshold enforcement
- **Documentation**: Documentation coverage analysis
- **Performance**: Benchmark execution and reporting

**Features:**

- Code formatting validation
- Import organization checking
- Cyclomatic complexity analysis
- Coverage reporting with PR comments
- Performance regression detection

### 6. Dependencies Workflow (`.github/workflows/dependencies.yml`)

**Triggers:**

- Weekly schedule (Mondays at midnight)
- Manual trigger with update type selection

**Jobs:**

- **Update Dependencies**: Automated dependency updates
- **Security Audit**: Comprehensive security analysis

**Features:**

- Selective update types (patch/minor/major)
- Vulnerability scanning
- Automated PR creation
- Security audit reporting

## Configuration Files

### GoReleaser (`.goreleaser.yml`)

Comprehensive release configuration including:

- Multi-platform builds (Linux, Windows, macOS, FreeBSD)
- Multiple architectures (amd64, arm64, arm)
- Package generation (DEB, RPM, APK, Arch)
- Docker multi-architecture images
- Homebrew formula generation
- Binary signing and SBOM generation

### Makefile

Development and build automation:

- **Build Commands**: `build`, `build-all`, `install`
- **Testing Commands**: `test`, `coverage`, `bench`
- **Quality Commands**: `fmt`, `lint`, `vet`, `security`, `quality`
- **Release Commands**: `release-test`, `release`
- **Docker Commands**: `docker-build`, `docker-run`
- **Development Commands**: `dev-setup`, `clean`

## Security Features

### Code Signing

- Binary signing with cosign
- GPG signature generation
- SBOM (Software Bill of Materials) generation

### Vulnerability Scanning

- **Gosec**: Go security checker
- **Trivy**: Vulnerability scanner for containers and filesystems
- **govulncheck**: Go vulnerability database checker
- **Nancy**: Dependency vulnerability scanner
- **CodeQL**: Semantic code analysis

### Dependency Management

- Automated dependency updates
- License compliance checking
- Security advisory monitoring
- Vulnerability impact assessment

## Release Process

### Automatic Releases

1. Commits are analyzed for conventional commit patterns
2. Version is automatically calculated based on commit types
3. Changelog is generated from commit messages
4. Release is created and tagged
5. Binaries are built and published
6. Docker images are pushed to registries
7. Package managers are updated

### Manual Releases

1. Use `make release` to create a tag locally
2. Push the tag to trigger the release workflow
3. Or use the GitHub Actions UI for manual triggering

### Release Types

- **Patch**: Bug fixes (`fix:` commits)
- **Minor**: New features (`feat:` commits)
- **Major**: Breaking changes (`BREAKING CHANGE:` in commit body)
- **Prerelease**: Alpha, beta, or RC versions

## Monitoring and Notifications

### Quality Gates

- All tests must pass
- Code coverage must meet threshold (70%)
- Security scans must pass
- Linting must pass without errors

### Artifacts

- Binary artifacts for all platforms
- Docker images for multiple architectures
- Package manager packages (DEB, RPM, etc.)
- Coverage reports
- Security scan results
- Documentation

### Notifications

- PR comments for coverage reports
- Release notifications
- Security alert integration
- Dependency update notifications

## Development Workflow

### Setting Up Development Environment

```bash
make dev-setup
```

### Running Quality Checks

```bash
make quality
```

### Testing Release Process

```bash
make release-test
```

### Building for All Platforms

```bash
make build-all
```

## Best Practices

### Commit Messages

Use conventional commit format:

- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `chore:` for maintenance tasks
- `refactor:` for code refactoring
- `test:` for test additions/changes

### Pull Requests

- All checks must pass before merging
- Code coverage should not decrease
- Security scans must pass
- Documentation should be updated for new features

### Releases

- Follow semantic versioning (SemVer)
- Include comprehensive changelog
- Test releases in staging environment
- Monitor post-release metrics

## Troubleshooting

### Common Issues

1. **Build Failures**
   - Check Go version compatibility
   - Verify all dependencies are available
   - Review build logs for specific errors

2. **Test Failures**
   - Run tests locally first
   - Check for race conditions
   - Verify test environment setup

3. **Security Scan Failures**
   - Review security findings
   - Update vulnerable dependencies
   - Apply security patches

4. **Release Failures**
   - Verify tag format (must start with 'v')
   - Check GoReleaser configuration
   - Ensure all secrets are configured

### Getting Help

- Check workflow logs in GitHub Actions
- Review this documentation
- Open an issue for CI/CD problems
- Contact maintainers for access issues

## Future Improvements

- [ ] Add performance regression testing
- [ ] Implement canary deployments
- [ ] Add integration with external monitoring
- [ ] Enhance security scanning with custom rules
- [ ] Add automated dependency license checking
- [ ] Implement blue-green deployment strategy
- [ ] Add chaos engineering tests
- [ ] Enhance release notes automation
