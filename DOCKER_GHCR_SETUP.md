# Docker Image Publication to GitHub Container Registry (GHCR)

## ✅ What's Already Configured

The project is fully configured to publish Docker images to GHCR. The configuration includes:

1. **`.goreleaser.yml`** - configured to build Docker images for GHCR
2. **`.github/workflows/release.yml`** - workflow configured for publication
3. **`Dockerfile`** - multi-stage build with minimal image size

## 🔑 Required GitHub Secrets

To publish to GHCR, you need to configure the following secrets in the repository:

### 1. GH_TOKEN (required)

This is a Personal Access Token (PAT) with package publication rights.

**How to create:**

1. Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Give the token a name: `ONIGIRAZU_RELEASE_TOKEN`
4. Set expiration: `No expiration` (or as you prefer)
5. Select the following scopes:
   - ✅ `repo` (Full control of private repositories)
   - ✅ `write:packages` (Upload packages to GitHub Package Registry)
   - ✅ `read:packages` (Download packages from GitHub Package Registry)
   - ✅ `delete:packages` (Delete packages from GitHub Package Registry)
6. Click "Generate token"
7. **IMPORTANT:** Copy the token immediately, it won't be shown again!

**How to add to the repository:**

1. Go to repository → Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Name: `GH_TOKEN`
4. Secret: paste the copied token
5. Click "Add secret"

### 2. DOCKERHUB_USERNAME and DOCKERHUB_TOKEN (optional)

These secrets are only needed if you want to publish images to Docker Hub as well.

**How to create Docker Hub token:**

1. Log in to Docker Hub
2. Go to Account Settings → Security → Access Tokens
3. Click "New Access Token"
4. Name: `onigirazu-release`
5. Select permissions: `Read, Write, Delete`
6. Click "Generate"
7. Copy the token

**How to add to the repository:**

1. Go to repository → Settings → Secrets and variables → Actions
2. Add two secrets:
   - `DOCKERHUB_USERNAME` - your Docker Hub username
   - `DOCKERHUB_TOKEN` - the copied token

### 3. COSIGN_PRIVATE_KEY and COSIGN_PASSWORD (optional)

These secrets are needed for signing artifacts with cosign.

**How to create:**

```bash
# Install cosign
brew install cosign  # macOS
# or
go install github.com/sigstore/cosign/v2/cmd/cosign@latest

# Create key pair
cosign generate-key-pair

# This will create two files:
# - cosign.key (private key)
# - cosign.pub (public key)
```

**How to add to the repository:**

1. Go to repository → Settings → Secrets and variables → Actions
2. Add two secrets:
   - `COSIGN_PRIVATE_KEY` - contents of the `cosign.key` file
   - `COSIGN_PASSWORD` - the password you specified when creating the keys

### 4. FURY_TOKEN (optional)

This secret is needed to publish packages to Fury.io (alternative package repository).

## 📋 Verifying Configuration

### 1. Check workflow permissions

The file `.github/workflows/release.yml` should have the following permissions:

```yaml
permissions:
  contents: write      # For creating releases
  packages: write      # For publishing to GHCR
  id-token: write      # For signing artifacts
```

✅ **Already configured in the project**

### 2. Check repository settings

1. Go to Settings → Actions → General
2. In the "Workflow permissions" section, select:
   - ✅ "Read and write permissions"
   - ✅ "Allow GitHub Actions to create and approve pull requests"

### 3. Check package visibility

After the first publication:

1. Go to GitHub profile → Packages
2. Find the `onigirazu` package
3. Click on it → Package settings
4. In the "Danger Zone" section → "Change package visibility"
5. Select "Public" (if you want the images to be public)

## 🚀 How to Publish a Release

### Automatic Publishing (recommended)

Simply create and push a tag:

```bash
# Create tag
git tag -a v1.0.0 -m "Release v1.0.0"

# Push tag
git push origin v1.0.0
```

This automatically runs the workflow, which:

1. Runs tests
2. Builds binaries for all platforms
3. Creates a release on GitHub
4. Builds and publishes Docker images to GHCR (and Docker Hub, if configured)

### Manual Publishing

You can also run the workflow manually:

1. Go to repository → Actions → Release
2. Click "Run workflow"
3. Enter the tag (e.g., `v1.0.0`)
4. Click "Run workflow"

## 🐳 Using Published Images

After publication, images will be available at:

### GitHub Container Registry (GHCR)

```bash
# Latest version
docker pull ghcr.io/onigirazu-cfg/onigirazu:latest

# Specific version
docker pull ghcr.io/onigirazu-cfg/onigirazu:v1.0.0

# Specific version and architecture
docker pull ghcr.io/onigirazu-cfg/onigirazu:v1.0.0-amd64
docker pull ghcr.io/onigirazu-cfg/onigirazu:v1.0.0-arm64v8
```

### Docker Hub (if configured)

```bash
# Latest version
docker pull onigirazu/onigirazu:latest

# Specific version
docker pull onigirazu/onigirazu:v1.0.0
```

## 🔍 Verifying Publication

### 1. Check GitHub Actions

1. Go to repository → Actions
2. Find the "Release" workflow
3. Check that all steps completed successfully:
   - ✅ validate
   - ✅ test
   - ✅ release
   - ✅ docker
   - ✅ notify

### 2. Check Release

1. Go to repository → Releases
2. Find the created release
3. Verify all artifacts are present:
   - Binaries for all platforms
   - Packages (DEB, RPM, APK, Arch)
   - Checksums
   - SBOM files

### 3. Check Docker Images

```bash
# Verify the image is available
docker pull ghcr.io/onigirazu-cfg/onigirazu:latest

# Check version
docker run --rm ghcr.io/onigirazu-cfg/onigirazu:latest --version

# Check supported architectures
docker manifest inspect ghcr.io/onigirazu-cfg/onigirazu:latest
```

### 4. Check GHCR Packages

1. Go to GitHub profile → Packages
2. Find the `onigirazu` package
3. Check the list of versions
4. Check download statistics

## 🐛 Troubleshooting

### Error: "Resource not accessible by integration"

**Cause:** Workflow doesn't have sufficient permissions

**Solution:**

1. Check permissions in `.github/workflows/release.yml`
2. Check repository settings (Settings → Actions → General)
3. Make sure "Read and write permissions" is selected

### Error: "authentication required"

**Cause:** Incorrect or missing token

**Solution:**

1. Verify that the `GH_TOKEN` secret is added to the repository
2. Verify that the token has `write:packages` permission
3. Create a new token if the old one has expired

### Error: "denied: permission_denied"

**Cause:** Package exists but token doesn't have write permission

**Solution:**

1. Go to GitHub profile → Packages → onigirazu
2. Package settings → Manage Actions access
3. Add the repository with "Write" permission

### Docker images are not being built

**Cause:** Issues with Docker Buildx or platforms

**Solution:**

1. Check workflow logs
2. Make sure the runner supports multi-arch builds
3. Verify that Docker Buildx is installed and configured

### Images are published only to GHCR, not to Docker Hub

**Cause:** Docker Hub secrets are not configured

**Solution:**

1. This is normal if you don't want to publish to Docker Hub
2. If you do - add `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` secrets

## 📚 Additional Resources

- [GitHub Container Registry Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [GoReleaser Docker Documentation](https://goreleaser.com/customization/docker/)
- [Docker Buildx Documentation](https://docs.docker.com/buildx/working-with-buildx/)
- [Cosign Documentation](https://docs.sigstore.dev/cosign/overview/)

## ✅ Pre-Release Checklist

- [ ] Created Personal Access Token with `repo` and `write:packages` permissions
- [ ] Token added to repository secrets as `GH_TOKEN`
- [ ] Verified workflow permissions in `.github/workflows/release.yml`
- [ ] Verified repository settings (Actions → General → Workflow permissions)
- [ ] (Optional) Configured Docker Hub secrets
- [ ] (Optional) Configured Cosign keys for signing
- [ ] Verified Dockerfile
- [ ] Verified `.goreleaser.yml` configuration
- [ ] Ran local test: `./scripts/test-release.sh`
- [ ] Created and pushed test tag (e.g., `v0.1.0-beta.1`)
- [ ] Verified successful test release publication
- [ ] Verified Docker images are accessible

After completing all items, you're ready to create a stable release! 🎉
