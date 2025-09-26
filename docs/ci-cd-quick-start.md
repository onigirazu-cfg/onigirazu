# CI/CD Quick Start Guide

This guide will help you quickly set up and use the CI/CD system for Onigirazu.

## 🚀 Quick Setup

### 1. Initial Setup

```bash
# Setup CI/CD system
make ci-setup

# Or use the script directly
./scripts/ci-manager.sh setup
```

### 2. Check Status

```bash
# Check CI/CD system status
make ci-status

# Or use the script directly
./scripts/ci-manager.sh status
```

## 🔧 Daily Development Workflow

### Running Tests Locally

```bash
# Run all tests
make test

# Run tests with race detection
make test-race

# Run tests with coverage
make test-coverage

# Run full CI pipeline locally
make ci-pipeline
```

### Code Quality Checks

```bash
# Format code
make fmt

# Run linting
make lint

# Run security scan
make security

# Run all quality checks
make quality
```

### Pre-release Checks

```bash
# Run comprehensive pre-release checks
make ci-pre-check

# Or use the script directly
./scripts/pre-release-check.sh
```

## 📦 Release Management

### Preparing a Release

```bash
# Interactive release preparation
make ci-release-prepare

# Or specify release type directly
./scripts/release-helper.sh prepare minor
```

### Creating a Release

```bash
# Interactive release creation
make ci-release-create

# Or create specific version
./scripts/release-helper.sh create v1.2.3
```

### Hotfix Release

```bash
# Create hotfix release
./scripts/release-helper.sh hotfix
```

## 📊 Monitoring and Logs

### Viewing Workflow Status

```bash
# Show workflow logs
make ci-logs

# Show specific workflow logs
./scripts/ci-manager.sh logs ci
```

### Triggering Workflows

```bash
# Trigger workflow interactively
make ci-trigger

# Trigger specific workflow
./scripts/ci-manager.sh trigger ci
```

## 🛠️ Available Scripts

| Script | Purpose |
|--------|---------|
| `scripts/ci-manager.sh` | Main CI/CD management interface |
| `scripts/setup-ci.sh` | CI/CD system setup and validation |
| `scripts/pre-release-check.sh` | Comprehensive pre-release checks |
| `scripts/release-helper.sh` | Release management utilities |

## 📋 Common Commands

### Development

```bash
make dev-setup          # Setup development environment
make build              # Build project
make test               # Run tests
make lint               # Run linting
make security           # Run security scan
make clean              # Clean build artifacts
```

### CI/CD Management

```bash
make ci-status          # Show CI/CD status
make ci-validate        # Validate configuration
make ci-pipeline        # Run full pipeline locally
make ci-pre-check       # Run pre-release checks
```

### Release Management

```bash
make ci-release-prepare # Prepare release
make ci-release-create  # Create release
make release-test       # Test release process
```

## 🔍 Troubleshooting

### Common Issues

#### 1. Scripts Not Executable

```bash
chmod +x scripts/*.sh
```

#### 2. Missing Development Tools

```bash
make dev-setup
```

#### 3. GitHub CLI Not Authenticated

```bash
gh auth login
```

#### 4. Docker Not Running

```bash
# Start Docker Desktop or Docker daemon
docker info
```

### Getting Help

```bash
# Show available make targets
make help

# Show CI/CD manager help
./scripts/ci-manager.sh help

# Show release helper help
./scripts/release-helper.sh help
```

## 🔐 Required Secrets

For full CI/CD functionality, configure these GitHub secrets:

| Secret | Purpose |
|--------|---------|
| `GITHUB_TOKEN` | GitHub API access (auto-provided) |
| `CODECOV_TOKEN` | Code coverage reporting |
| `DOCKERHUB_USERNAME` | Docker Hub publishing |
| `DOCKERHUB_TOKEN` | Docker Hub authentication |
| `GPG_FINGERPRINT` | Package signing |
| `COSIGN_PRIVATE_KEY` | Binary signing |
| `COSIGN_PASSWORD` | Cosign key password |
| `FURY_TOKEN` | Package repository access |

## 📈 Workflow Overview

### CI Workflow (`.github/workflows/ci.yml`)

- Runs on every push and PR
- Multi-OS testing (Ubuntu, macOS, Windows)
- Multi-Go version testing (1.22, 1.23, 1.24)
- Cross-platform builds
- Integration tests
- Quality gates

### Release Workflow (`.github/workflows/release.yml`)

- Triggered by tags or manual dispatch
- Comprehensive testing
- Multi-architecture builds
- Package publishing
- Binary signing
- Release notes generation

### Security Workflow (`.github/workflows/security.yml`)

- CodeQL analysis
- Dependency review
- Vulnerability scanning
- SARIF reporting

### Auto-release Workflow (`.github/workflows/auto-release.yml`)

- Automatic version calculation
- Conventional commit analysis
- Changelog generation
- Automated releases

### Code Quality Workflow (`.github/workflows/code-quality.yml`)

- Code formatting validation
- Test coverage enforcement
- Documentation checks
- Performance benchmarks

### Dependencies Workflow (`.github/workflows/dependencies.yml`)

- Automated dependency updates
- Security auditing
- PR creation with detailed reports

## 🎯 Best Practices

### Commit Messages

Use conventional commit format:

```
feat: add new feature
fix: resolve bug
docs: update documentation
chore: update dependencies
```

### Branch Protection

- Require PR reviews
- Require status checks
- Require up-to-date branches
- Restrict pushes to main branch

### Release Process

1. Run `make ci-pre-check` before releasing
2. Use semantic versioning (v1.2.3)
3. Create releases from main branch
4. Test releases with `make release-test`

### Security

- Keep dependencies updated
- Run security scans regularly
- Review dependency changes
- Monitor vulnerability reports

## 📚 Additional Resources

- [Full CI/CD Documentation](ci-cd.md)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GoReleaser Documentation](https://goreleaser.com/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
