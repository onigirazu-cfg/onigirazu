# Container Image (GHCR)

The image is published only to `ghcr.io/onigirazu-cfg/onigirazu`, by the `docker` job of the Release workflow (`.github/workflows/release.yml`) after GoReleaser succeeds. GoReleaser itself builds no images.

- Platforms: `linux/amd64`, `linux/arm64` (Buildx with QEMU)
- Tags: `X.Y.Z`, `X.Y`, `X`, `latest` (no `v` prefix)
- Image: `Dockerfile`, a static binary on `scratch`, runs as `appuser`, entrypoint `/onigirazu`

## Setup

No secrets are needed: the job logs in with `GITHUB_TOKEN` (workflow permission `packages: write`) and then tries to make the package public through the API. If that call is refused, set the visibility once by hand: organization → Packages → `onigirazu` → Package settings → Change visibility → Public.

## Usage

```bash
docker pull ghcr.io/onigirazu-cfg/onigirazu:latest
docker run --rm ghcr.io/onigirazu-cfg/onigirazu:latest version
docker buildx imagetools inspect ghcr.io/onigirazu-cfg/onigirazu:latest
```

## Troubleshooting

- `denied: permission_denied` on push: in the package settings under "Manage Actions access", give the `onigirazu` repository write access.
- The `docker` job runs only when the GoReleaser job succeeded; a failed release publishes no image.
