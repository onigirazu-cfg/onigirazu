# Debug Scripts

This directory contains debug and test scripts used during development.

## ⚠️ Important

These scripts are **NOT** part of the main build and should not be executed as part of the regular test suite.

## Contents

- `test_*.go` - Standalone debug programs with `main()` functions
- `test_*.yml` - Test playbooks for manual testing
- `test_*.sh` - Shell scripts for testing

## Usage

To run a debug script:

```bash
cd debug
go run test_dpkg_debug.go
```

## Note

These files were moved from the project root to prevent conflicts with `go test ./...` and CI/CD pipelines.
