# Contributing to Onigirazu

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).

## Setup

Requirements: Go 1.26.6 or later (see `go.mod`), Git, Make, [golangci-lint](https://golangci-lint.run/) v2 (CI uses v2.13.2).

```bash
git clone https://github.com/onigirazu-cfg/onigirazu.git
cd onigirazu
go mod download
make build        # bin/onigirazu
make test
```

## Workflow

1. Branch from `main` (`feat/...`, `fix/...`, `docs/...`, `ci/...`, `chore/...`).
2. Make the change with tests.
3. Run the checks CI runs:

   ```bash
   gofmt -s -l .          # must print nothing
   goimports -l .         # must print nothing
   go vet ./...
   make lint              # golangci-lint
   go test -race ./...
   go mod tidy            # must not change go.mod/go.sum
   ```

4. Open a pull request against `main`.

## Checks on a pull request

| Workflow | What must pass |
|----------|----------------|
| CI (`ci.yml`) | vet, race tests, cross-builds, golangci-lint, localhost smoke test, S3 state test |
| Code Quality (`code-quality.yml`) | gofmt, goimports, staticcheck, misspell, ineffassign, `go mod tidy`, coverage >= 45% |
| Coverage Gate (`coverage-gate.yml`) | total coverage >= 45% |
| Security Scan (`security.yml`) | govulncheck, CodeQL, dependency review |

E2E (`e2e.yml`) does not run for pull requests. Start it on the branch and wait for it to pass:

```bash
gh workflow run e2e.yml --ref <branch>
gh workflow run e2e.yml --ref <branch> -f cases="01-file 02-copy"   # subset
```

Cases live in `e2e/cases/`; see [e2e/README.md](e2e/README.md).

## Commits and merging

Commit messages and PR titles follow [Conventional Commits](https://www.conventionalcommits.org/): `feat`, `fix`, `docs`, `refactor`, `perf`, `test`, `ci`, `chore`, with an optional scope, lowercase.

Pull requests are squash-merged once CI, Code Quality, Coverage Gate, Security Scan (including CodeQL) and E2E are green; `main` requires no review:

```bash
gh pr merge <N> --squash --subject "feat(scope): summary (#<N>)"
```

A squash commit whose subject starts with `feat` or `fix` produces a release; see [Release Process](docs/RELEASE_PROCESS.md).

## Modules

See the [Module Development Guide](docs/MODULE_DEVELOPMENT_GUIDE.md).

## Issues

Include the onigirazu version (`onigirazu version`), OS, the playbook or a minimal reproduction, and the full output.
