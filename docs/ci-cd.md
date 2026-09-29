# CI/CD

All automation is GitHub Actions in `.github/workflows/`. Dependency updates come from Dependabot (`.github/dependabot.yml`: Go modules, Actions, Docker, weekly).

## Workflows

| Workflow | Triggers | Jobs |
|----------|----------|------|
| CI (`ci.yml`) | push and PR to `main`, manual | `test`: vet, staticcheck (non-blocking), `go test -race`; `build`: linux/darwin amd64+arm64, windows amd64; `lint`: golangci-lint v2.13.2; `integration-tests`: smoke playbook against localhost, applied twice; `s3-state`: managed state in S3 (`scripts/garage-s3-test.sh`); `docker-build`: image build without push (push events only); `quality-gate` |
| Code Quality (`code-quality.yml`) | push and PR to `main`, Sundays 02:00 UTC, manual | gofmt -s, goimports, vet, staticcheck, misspell, ineffassign, `go mod tidy` check (blocking); gocyclo, golines, TODO count (report only); coverage >= 45% with PR comment; `go doc` export; benchmarks |
| Coverage Gate (`coverage-gate.yml`) | push and PR to `main`, manual | total coverage >= 45%, Codecov upload, PR comment |
| Security Scan (`security.yml`) | push and PR to `main`, Mondays 06:00 UTC, manual | govulncheck (blocking); gosec, Trivy, Nancy (report only, SARIF to code scanning); CodeQL; dependency review on PRs (fails on moderate or higher) |
| License Check (`license-check.yml`) | push and PR to `main` | `go-licenses` check and report (non-blocking) |
| Documentation (`docs.yml`) | push and PR to `main` | renders `README.md` to HTML and deploys `docs/` to GitHub Pages (on `main` only) |
| E2E (`e2e.yml`) | manual, daily 05:00 UTC; janitor hourly | playbooks from `e2e/cases/` on disposable vSphere VMs, 4 shards, self-hosted runners labelled `vsphere-e2e`; see [e2e/README.md](../e2e/README.md) |
| Auto Release (`auto-release.yml`) | push to `main` (not docs/`*.md`/`.github`/examples-only), manual | version bump from commit types, tag, GitHub Release, starts Release Gate |
| Release Gate (`release-gate.yml`) | `v*` tag push, manual | security, code quality, tests (coverage >= 15%), builds, lint; starts Release |
| Release (`release.yml`) | manual (started by Release Gate) | GoReleaser, multi-arch image to `ghcr.io/onigirazu-cfg/onigirazu` |


E2E does not run for pull requests; start it on a branch with `gh workflow run e2e.yml --ref <branch>`.

The release chain is described in [Release Process](RELEASE_PROCESS.md).

## Makefile

| Target | Action |
|--------|--------|
| `build` | `bin/onigirazu` |
| `build-all` | `dist/` binaries for linux, darwin (amd64, arm64), windows amd64 |
| `install` | build and copy to `/usr/local/bin` |
| `test`, `test-race`, `test-coverage`, `bench` | tests |
| `fmt`, `vet`, `lint`, `security`, `vuln-check` | go fmt + goimports, go vet, golangci-lint, gosec, govulncheck |
| `quality` | fmt, vet, lint, security, test |
| `release-test` | GoReleaser snapshot, no publish |
| `release` | prompts for a version, pushes the tag |
| `docker-build` | local image `onigirazu:latest` |
