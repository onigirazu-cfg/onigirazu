# Docker Hub Setup Guide

⚠️ **LEGACY DOCUMENTATION** - Docker Hub is no longer the primary Docker image registry.

**Recommended**: See [DOCKER_GHCR_SETUP.md](./DOCKER_GHCR_SETUP.md) for the current GitHub Container Registry (GHCR) setup, which is integrated with the GitHub release workflow.

This guide is kept for reference and may be deprecated in future versions.

---

This guide explains how to set up Docker Hub publishing for Onigirazu releases.

## Prerequisites

1. Docker Hub account (create at <https://hub.docker.com/>)
2. Docker Hub repository created: `onigirazu/onigirazu`
3. Access to GitHub repository secrets

## Step 1: Create Docker Hub Access Token

1. Go to <https://hub.docker.com/settings/security>
2. Click **"New Access Token"**
3. Name: `github-actions-onigirazu`
4. Permissions: **Read, Write, Delete**
5. Click **"Generate"**
6. **Copy the token** (you won't see it again!)

## Step 2: Add Secrets to GitHub

1. Go to repository settings:

   ```
   https://github.com/onigirazu-cfg/onigirazu/settings/secrets/actions
   ```

2. Add two secrets:

   **Secret 1: DOCKERHUB_USERNAME**
   - Name: `DOCKERHUB_USERNAME`
   - Value: Your Docker Hub username (e.g., `onigirazu`)

   **Secret 2: DOCKERHUB_TOKEN**
   - Name: `DOCKERHUB_TOKEN`
   - Value: The access token you generated in Step 1

## Step 3: Create Docker Hub Repository

1. Go to <https://hub.docker.com/repositories>
2. Click **"Create Repository"**
3. Settings:
   - **Name:** `onigirazu`
   - **Visibility:** Public
   - **Description:** Modern configuration management tool inspired by Ansible
4. Click **"Create"**

## Step 4: Test the Setup

After adding the secrets, the next release will automatically push to Docker Hub.

To test manually:

```bash
# Trigger a release
git tag -a v1.7.2 -m "Test Docker Hub publishing"
git push origin v1.7.2
```

## Verification

After the release workflow completes, verify the images:

```bash
# Pull from Docker Hub
docker pull onigirazu/onigirazu:latest
docker pull onigirazu/onigirazu:v1.7.2

# Pull from GitHub Container Registry (if org settings allow public packages)
docker pull ghcr.io/onigirazu-cfg/onigirazu:latest
docker pull ghcr.io/onigirazu-cfg/onigirazu:v1.7.2
```

## Troubleshooting

### Error: "unauthorized: authentication required"

**Cause:** Secrets not configured or incorrect

**Solution:**

1. Verify secrets are added correctly in GitHub
2. Check Docker Hub token is valid
3. Ensure token has Read, Write, Delete permissions

### Error: "denied: requested access to the resource is denied"

**Cause:** Repository doesn't exist or username is wrong

**Solution:**

1. Create repository on Docker Hub: `onigirazu/onigirazu`
2. Verify DOCKERHUB_USERNAME matches your Docker Hub username

### Images not appearing on Docker Hub

**Cause:** Workflow might have failed or secrets missing

**Solution:**

1. Check GitHub Actions logs: <https://github.com/onigirazu-cfg/onigirazu/actions>
2. Look for the "docker" job
3. Check if login step succeeded

## GitHub Container Registry (GHCR) Alternative

If Docker Hub is not available, you can use GHCR exclusively:

### Making GHCR Packages Public

**Option 1: Organization Settings (Recommended)**

1. Go to: <https://github.com/organizations/onigirazu-cfg/settings/packages>
2. Enable: "Members can change package visibility to public"
3. Go to package: <https://github.com/orgs/onigirazu-cfg/packages>
4. Click on `onigirazu` → Package settings → Change visibility → Public

**Option 2: Use Personal Account**
If organization settings can't be changed, consider publishing under a personal account:

- Change `ghcr.io/onigirazu-cfg/onigirazu` to `ghcr.io/YOUR_USERNAME/onigirazu`
- Personal accounts have more flexibility with package visibility

## Best Practices

1. **Use both registries** for redundancy:
   - Docker Hub: Primary, most accessible
   - GHCR: Backup, integrated with GitHub

2. **Keep tokens secure:**
   - Never commit tokens to git
   - Use GitHub Secrets
   - Rotate tokens periodically

3. **Monitor usage:**
   - Docker Hub free tier: 200 pulls/6 hours
   - Consider Docker Hub Pro if needed

4. **Tag strategy:**
   - `latest`: Always points to newest stable release
   - `v1.7.2`: Specific version
   - `v1.7`: Minor version (auto-updated)
   - `v1`: Major version (auto-updated)

## Support

If you encounter issues:

1. Check GitHub Actions logs
2. Verify Docker Hub repository exists and is public
3. Ensure secrets are correctly configured
4. Open an issue: <https://github.com/onigirazu-cfg/onigirazu/issues>
