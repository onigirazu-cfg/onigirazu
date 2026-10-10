# Release Process

Releases are automatic. Merging a pull request to `main` with a squash subject that starts with `feat` or `fix` produces a new version.

## Pipeline

1. **Auto Release** (`auto-release.yml`) runs on every push to `main`, except pushes that only change `docs/**`, `*.md`, `.github/**` or `examples/**`. It looks at commits since the latest tag:
   - any commit containing `BREAKING CHANGE` - major
   - a subject starting with `feat` - minor
   - a subject starting with `fix` - patch
   - otherwise no release

   It runs `go test -race ./...`, pushes the tag `vX.Y.Z`, creates the GitHub Release with a changelog, and starts the Release Gate.
2. **Release Gate** (`release-gate.yml`) on the tag: gosec (report only), govulncheck, gofmt/goimports/vet/staticcheck, race tests with coverage >= 15%, builds, golangci-lint. If all pass, it starts Release.
3. **Release** (`release.yml`): GoReleaser runs `go generate ./...` (embeds the Linux and Windows agents), publishes archives (with `onigirazu` and `onigirazu-test`), packages and `checksums.txt` to the GitHub Release; then the multi-arch image is pushed to GHCR.

The whole chain takes about 20 minutes.

## Artifacts

- Archives: Linux (x86_64, arm64, armv6, armv7, i386), macOS (x86_64, arm64), Windows (x86_64, i386, zip), FreeBSD, OpenBSD, NetBSD (x86_64, i386)
- Packages: deb, rpm, apk for amd64, 386, arm64, armv6, armv7; Arch (`.pkg.tar.zst`) for all but armv6
- `checksums.txt` (SHA-256)
- Image `ghcr.io/onigirazu-cfg/onigirazu`, `linux/amd64` and `linux/arm64`, tags `X.Y.Z`, `X.Y`, `X`, `latest` (no `v` prefix)

Artifacts are not signed and no SBOM is produced.

## Monitoring

```bash
gh run list --workflow=auto-release.yml --limit=3
gh run list --workflow=release-gate.yml --limit=3
gh run list --workflow=release.yml --limit=3
gh release view
```

## Manual release

Start Auto Release by hand; it releases regardless of commit types:

```bash
gh workflow run auto-release.yml --ref main -f release_type=minor   # patch (default), minor, major, prerelease
```

`prerelease` produces `vX.Y.Z-alpha`. A tag can also be pushed by hand (`make release` prompts for it); a pushed `v*` tag starts the Release Gate.

To run only the publishing step for a tag (runs the tests first unless `-f skip_checks=true`):

```bash
gh workflow run release.yml -f tag=vX.Y.Z
```

## Troubleshooting

- **Auto Release fails with "refusing to allow a GitHub App ... without `workflows` permission"**: the run tried to tag a commit behind `main` after workflow files changed. Run it again from `main`: `gh workflow run auto-release.yml --ref main -f release_type=<minor|patch>`.
- **Release Gate failed**: the tag and an empty GitHub Release exist, nothing is published. For a flaky failure rerun the gate on the tag (`gh workflow run release-gate.yml --ref vX.Y.Z`); otherwise fix `main`, and the next `feat`/`fix` merge releases a new version.
- **No release after a merge**: the squash subject did not start with lowercase `feat` or `fix`, or the merge only touched ignored paths.

## Local test

```bash
make release-test   # goreleaser release --snapshot --clean --skip=publish
```
