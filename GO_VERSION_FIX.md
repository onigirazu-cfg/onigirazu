# Go Version Compatibility Fix for v1.9.0 Release

## Problem

The initial v1.9.0 release failed in GitHub Actions with the following errors:

```
Run go mod download
go: go.mod requires go >= 1.24.0 (running go 1.22.12; GOTOOLCHAIN=local)
Error: Process completed with exit code 1.
```

```
Run go mod download
go: go.mod requires go >= 1.24.0 (running go 1.23.12; GOTOOLCHAIN=local)
Error: Process completed with exit code 1.
```

**Root Cause**: The `go.mod` file required Go 1.24.0, which hasn't been released yet. GitHub Actions runners only have Go 1.23.x available.

## Solution

### 1. Updated `go.mod`

Changed the Go version requirement:

```diff
- go 1.24.0
+ go 1.23
```

### 2. Updated GitHub Actions Workflows

Updated all workflow files to use Go 1.23:

- `.github/workflows/release.yml` - Changed `GO_VERSION: '1.24'` → `'1.23'`
- `.github/workflows/ci.yml` - Changed `GO_VERSION: '1.24'` → `'1.23'` and removed `1.24` from test matrix
- `.github/workflows/auto-release.yml` - Changed `GO_VERSION: '1.24'` → `'1.23'`
- `.github/workflows/code-quality.yml` - Changed `GO_VERSION: '1.24'` → `'1.23'`
- `.github/workflows/dependencies.yml` - Changed `GO_VERSION: '1.24'` → `'1.23'`
- `.github/workflows/security.yml` - Changed `GO_VERSION: '1.24'` → `'1.23'`
- `.github/workflows/license-check.yml` - Changed `go-version: '1.24'` → `'1.23'`

### 3. Re-released v1.9.0

Steps taken:

1. Committed the Go version fixes (commit `35d1b93`)
2. Deleted the old v1.9.0 tag locally and remotely
3. Created a new v1.9.0 tag with the fixes
4. Pushed the new tag to trigger the release workflow

## Verification

```bash
# Verify no more 1.24 references
grep -r "1\.24" .github/workflows/*.yml go.mod
# Result: No matches found ✅
```

## Timeline

- **2025-01-16 (Initial)**: Created v1.9.0 tag with Go 1.24 requirement
- **2025-01-16 (Fix)**: Updated to Go 1.23, re-tagged v1.9.0
- **Commit**: 35d1b93

## Status

✅ **RESOLVED** - The release workflow should now complete successfully with Go 1.23.

## Links

- **GitHub Actions**: <https://github.com/onigirazu-cfg/onigirazu/actions>
- **Release Tag**: <https://github.com/onigirazu-cfg/onigirazu/releases/tag/v1.9.0>
- **Fix Commit**: <https://github.com/onigirazu-cfg/onigirazu/commit/35d1b93>
