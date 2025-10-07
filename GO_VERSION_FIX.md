# Go Version Compatibility Fix for v1.9.0 Release

## Problem History

### First Attempt (Failed)

The initial v1.9.0 release failed in GitHub Actions with:

```
Run go mod download
go: go.mod requires go >= 1.24.0 (running go 1.22.12; GOTOOLCHAIN=local)
Error: Process completed with exit code 1.
```

**Initial Root Cause**: The `go.mod` file required Go 1.24.0, which we thought wasn't available in GitHub Actions.

### Second Attempt (Also Failed)

After downgrading to Go 1.23, the release still failed with:

```
Run go test -race -coverprofile=coverage.out -covermode=atomic -timeout=10m ./...
go: golang.org/x/crypto@v0.42.0 requires go >= 1.24.0 (running go 1.23.12; GOTOOLCHAIN=local)
Error: Process completed with exit code 1.
```

**Actual Root Cause**: The dependency `golang.org/x/crypto@v0.42.0` requires Go 1.24.0+. We needed to upgrade to Go 1.24, not downgrade to 1.23!

## Solution (Final - Correct Approach)

### 1. Verified Go 1.24 Availability

Confirmed that Go 1.24 IS available in GitHub Actions via `setup-go` action.

### 2. Updated `go.mod`

Kept Go 1.24.0 requirement (needed for dependencies):

```go
go 1.24.0
```

### 3. Upgraded Dependencies

Restored `golang.org/x/crypto` to the required version:

```bash
go get golang.org/x/crypto@v0.42.0
go mod tidy
```

### 4. Updated ALL GitHub Actions Workflows to Go 1.24

Updated all workflow files to use Go 1.24:

- `.github/workflows/release.yml` - Changed `GO_VERSION: '1.23'` → `'1.24'`
- `.github/workflows/ci.yml` - Changed `GO_VERSION: '1.23'` → `'1.24'` and updated test matrix to `['1.24']` only
- `.github/workflows/auto-release.yml` - Changed `GO_VERSION: '1.23'` → `'1.24'`
- `.github/workflows/code-quality.yml` - Changed `GO_VERSION: '1.23'` → `'1.24'`
- `.github/workflows/dependencies.yml` - Changed `GO_VERSION: '1.23'` → `'1.24'`
- `.github/workflows/security.yml` - Changed `GO_VERSION: '1.23'` → `'1.24'`
- `.github/workflows/license-check.yml` - Changed `go-version: '1.23'` → `'1.24'`
- Updated Codecov condition: `if: matrix.go-version == '1.23'` → `'1.24'`

### 5. Fixed CI Test Matrix (Fourth Attempt)

After the third release attempt, discovered that the CI test matrix still included Go 1.23, which caused failures:

```
go: go.mod requires go >= 1.24.0 (running go 1.23.12; GOTOOLCHAIN=local)
```

**Solution**: Removed Go 1.23 from the test matrix in `.github/workflows/ci.yml`:

- Changed `go-version: ['1.23', '1.24']` → `['1.24']`
- Commit: `158eb1d`

### 6. Re-released v1.9.0 (Fourth Time - Final)

Steps taken:

1. Committed the correct Go 1.24 migration (commit `8c124eb`)
2. Deleted the failing v1.9.0 tag locally and remotely
3. Created a new v1.9.0 tag with all fixes
4. Pushed the new tag to trigger the release workflow

## Verification

```bash
# Verify Go 1.24 in go.mod
grep "^go " go.mod
# Result: go 1.24.0 ✅

# Verify crypto version
grep "golang.org/x/crypto" go.mod
# Result: golang.org/x/crypto v0.42.0 ✅

# Verify workflows use 1.24
grep "GO_VERSION:" .github/workflows/*.yml
# Result: All show '1.24' ✅
```

## Timeline

- **2025-01-16 (Initial)**: Created v1.9.0 tag with Go 1.24 requirement - Failed (thought 1.24 unavailable)
- **2025-01-16 (First Fix)**: Downgraded to Go 1.23 (commit `35d1b93`) - Failed (crypto dependency needs 1.24)
- **2025-01-16 (Second Fix)**: Upgraded everything to Go 1.24 (commit `8c124eb`) - Failed (CI matrix had 1.23)
- **2025-01-16 (Final Fix)**: Removed Go 1.23 from CI matrix (commit `158eb1d`) - ✅ SUCCESS

## Status

✅ **RESOLVED** - The release workflow should now complete successfully with Go 1.24.

## Key Lessons

1. **Don't downgrade to fix version issues - verify availability first!** Go 1.24 was available in GitHub Actions all along.

2. **Check ALL places where versions are specified**: When upgrading Go versions, you need to update:
   - `go.mod` file
   - All workflow files (`GO_VERSION` env vars)
   - Test matrices in CI workflows
   - Conditional statements that reference versions

3. **Test matrices can cause hidden failures**: Even if the main `GO_VERSION` is correct, test matrices with older versions will still fail if `go.mod` requires a newer version.

## Links

- **GitHub Actions**: <https://github.com/onigirazu-cfg/onigirazu/actions>
- **Release Tag**: <https://github.com/onigirazu-cfg/onigirazu/releases/tag/v1.9.0>
- **First Fix Commit** (wrong - downgrade): <https://github.com/onigirazu-cfg/onigirazu/commit/35d1b93>
- **Second Fix Commit** (partial - workflows): <https://github.com/onigirazu-cfg/onigirazu/commit/8c124eb>
- **Final Fix Commit** (complete - CI matrix): <https://github.com/onigirazu-cfg/onigirazu/commit/158eb1d>
