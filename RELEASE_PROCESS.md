# Release Process

## Overview

The Onigirazu project uses a **quality-gated release process** to ensure that all releases meet high standards for security, code quality, and functionality.

This process was implemented in **Phase 2** and automatically validates all releases before they go public.

## How It Works

### 1. Quality Gate (Automatic)

When you push a version tag (e.g., `v1.56.0`), the **Release Gate** workflow automatically runs:

```bash
git tag -a v1.56.0 -m "Release v1.56.0"
git push origin v1.56.0
```

The Release Gate performs the following checks:

- ✅ **Security Scan**: Runs `gosec` and `govulncheck` to detect security vulnerabilities
- ✅ **Code Quality**: Checks formatting, imports, runs `go vet` and `staticcheck`
- ✅ **Tests**: Runs full test suite with race detection and coverage checks (minimum 15%)
- ✅ **Build**: Verifies that binaries can be built for all target platforms
- ✅ **Lint**: Runs `golangci-lint` to catch code issues

### 2. Release (Automatic)

**Only if all quality checks pass**, the **Release** workflow is automatically triggered:

- 📦 Builds binaries for all platforms using GoReleaser
- 🐳 Builds and pushes Docker images
- 📝 Creates GitHub Release with artifacts
- 🔐 Signs artifacts with cosign
- 📢 Publishes to package managers

### 3. If Quality Gate Fails

If any check fails, the release is **blocked** and you'll see:

```
❌ Quality gate FAILED! Release is blocked.
Please fix the failing checks before releasing.
```

**What to do:**

1. Check the failed workflow logs
2. Fix the issues locally
3. Commit and push the fixes
4. Delete and recreate the tag:

   ```bash
   git tag -d v1.56.0
   git push origin :refs/tags/v1.56.0
   git tag -a v1.56.0 -m "Release v1.56.0"
   git push origin v1.56.0
   ```

### Monitoring the Release

After pushing a tag, monitor the pipelines:

```bash
# Check Release Gate status
gh run list --workflow=release-gate.yml --limit=1

# Watch in real-time
gh run watch <RUN_ID>

# Get detailed release info
gh release view --json tagName,name,publishedAt
```

Expected timeline:

- **Release Gate**: 5-7 minutes (quality checks)
- **Release Workflow**: 8-12 minutes (build & publish)
- **Total**: 15-20 minutes until GitHub Release is created

## Manual Release (Emergency)

In case you need to bypass quality checks (not recommended), you can trigger a manual release:

1. Go to **Actions** → **Release** workflow
2. Click **Run workflow**
3. Enter the tag name (e.g., `v1.56.0`)
4. Optionally check **Skip quality checks** (⚠️ use with caution)
5. Click **Run workflow**

## Release Checklist

Before creating a release:

- [ ] All changes are committed and pushed
- [ ] CHANGELOG.md is updated
- [ ] Version number follows semantic versioning
- [ ] All tests pass locally: `go test ./...`
- [ ] Code is formatted: `gofmt -s -w .`
- [ ] No security issues: `gosec ./...`
- [ ] Documentation is updated

## Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **Major** (v2.0.0): Breaking changes
- **Minor** (v1.56.0): New features, backward compatible (e.g., Phase 2 configuration management)
- **Patch** (v1.56.1): Bug fixes, backward compatible

Pre-release versions:

- **Alpha** (v1.28.0-alpha.1): Early testing
- **Beta** (v1.28.0-beta.1): Feature complete, testing
- **RC** (v1.28.0-rc.1): Release candidate

## Workflow Files

- `.github/workflows/release-gate.yml` - Quality checks before release
- `.github/workflows/release.yml` - Actual release process
- `.github/workflows/ci.yml` - Continuous integration
- `.github/workflows/security.yml` - Security scanning
- `.github/workflows/code-quality.yml` - Code quality checks

## Benefits of Quality-Gated Releases

✅ **Reliability**: Every release is tested and verified
✅ **Security**: No releases with known vulnerabilities
✅ **Quality**: Consistent code quality standards
✅ **Confidence**: Automated checks catch issues early
✅ **Transparency**: Clear visibility into what's being released

## Troubleshooting

### "Quality gate failed" error

Check which specific check failed:

- Security Scan → Fix security issues
- Code Quality → Run `gofmt`, `goimports`, fix linting issues
- Tests → Fix failing tests, improve coverage
- Build → Fix compilation errors
- Lint → Fix linting issues

### Tag already exists

Delete the tag locally and remotely:

```bash
git tag -d v1.56.0
git push origin :refs/tags/v1.56.0
```

### Release workflow not triggered

Ensure:

1. Tag follows format `v*.*.*`
2. Quality gate workflow completed successfully
3. You have proper permissions

## Questions?

If you have questions about the release process, please:

1. Check the workflow logs in GitHub Actions
2. Review this documentation
3. Open an issue with the `release` label
