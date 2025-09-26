.PHONY: build test clean run-example install lint fmt vet security coverage release-test ci-setup ci-status ci-validate ci-pipeline ci-pre-check ci-release-prepare ci-release-create test-race test-coverage

# Variables
BINARY_NAME=onigirazu
VERSION?=$(shell git describe --tags --always --dirty)
COMMIT?=$(shell git rev-parse HEAD)
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Build project
build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/onigirazu/main.go

# Build for all platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 cmd/onigirazu/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 cmd/onigirazu/main.go
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 cmd/onigirazu/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 cmd/onigirazu/main.go
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe cmd/onigirazu/main.go

# Install to system
install: build
	sudo cp bin/$(BINARY_NAME) /usr/local/bin/

# Run tests
test:
	go test -timeout=10m ./...

# Run tests with race detection
test-race:
	go test -race -timeout=10m ./...

# Run tests with coverage
test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Legacy coverage target
coverage: test-coverage

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Format code
fmt:
	go fmt ./...
	goimports -w . 2>/dev/null || true

# Lint code
lint:
	golangci-lint run --timeout=10m

# Vet code
vet:
	go vet ./...

# Security scan
security:
	gosec ./...

# Run all quality checks
quality: fmt vet lint security test

# Enhanced security scan with multiple tools
security-full:
	@echo "Running comprehensive security scan..."
	gosec ./... || true
	govulncheck ./... || true
	staticcheck ./... || true

# Clean artifacts
clean:
	rm -rf bin/ dist/
	rm -f .onigirazu-state coverage.out coverage.html

# Run example
run-example: build
	./bin/onigirazu -playbook examples/simple-playbook.yml -inventory examples/simple-inventory.yml -verbose

# Run in check mode
check-example: build
	./bin/onigirazu -playbook examples/simple-playbook.yml -inventory examples/simple-inventory.yml -verbose -check

# Run command example
run-command: build
	./bin/onigirazu -playbook examples/command-playbook.yml -inventory examples/simple-inventory.yml -verbose

# Run advanced command example
run-advanced: build
	./bin/onigirazu -playbook examples/advanced-command-playbook.yml -inventory examples/simple-inventory.yml -verbose

# Run user/group management example
run-users: build
	./bin/onigirazu -playbook examples/user-group-playbook.yml -inventory examples/simple-inventory.yml -verbose

# Run macOS user/group example
run-macos-users: build
	./bin/onigirazu -playbook examples/macos-user-group-playbook.yml -inventory examples/simple-inventory.yml -verbose

# Test release process locally
release-test:
	@echo "Testing release process..."
	goreleaser release --snapshot --clean --skip=publish

# Release (for maintainers)
release:
	@echo "Creating release..."
	@read -p "Enter version (e.g., v1.0.0): " version; \
	git tag -a $$version -m "Release $$version"; \
	git push origin $$version

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/github-action-gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/goreleaser/goreleaser@latest

# CI/CD Management Commands
ci-setup:
	@echo "Setting up CI/CD system..."
	./scripts/ci-manager.sh setup

ci-status:
	@echo "Checking CI/CD system status..."
	./scripts/ci-manager.sh status

ci-validate:
	@echo "Validating CI/CD configuration..."
	./scripts/ci-manager.sh validate

ci-pipeline:
	@echo "Running full CI pipeline locally..."
	./scripts/ci-manager.sh pipeline

ci-pre-check:
	@echo "Running pre-release checks..."
	./scripts/ci-manager.sh pre-check

ci-release-prepare:
	@echo "Preparing release..."
	@read -p "Release type (major/minor/patch): " type; \
	./scripts/ci-manager.sh release prepare $$type

ci-release-create:
	@echo "Creating release..."
	@read -p "Version (e.g., v1.2.3): " version; \
	./scripts/ci-manager.sh release create $$version

ci-logs:
	@echo "Showing workflow logs..."
	./scripts/ci-manager.sh logs

ci-trigger:
	@echo "Available workflows:"
	@ls .github/workflows/*.yml | xargs -I {} basename {} .yml
	@read -p "Workflow to trigger: " workflow; \
	./scripts/ci-manager.sh trigger $$workflow

# Docker build
docker-build:
	docker build -t onigirazu:latest .

# Docker run
docker-run: docker-build
	docker run --rm -v $(PWD)/examples:/examples onigirazu:latest -playbook /examples/simple-playbook.yml -inventory /examples/simple-inventory.yml

# Show help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Build commands:"
	@echo "  build         - Build project"
	@echo "  build-all     - Build for all platforms"
	@echo "  install       - Install to system"
	@echo ""
	@echo "Testing commands:"
	@echo "  test          - Run tests"
	@echo "  test-race     - Run tests with race detection"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  coverage      - Alias for test-coverage"
	@echo "  bench         - Run benchmarks"
	@echo ""
	@echo "Quality commands:"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  vet           - Vet code"
	@echo "  security      - Security scan"
	@echo "  security-full - Comprehensive security scan"
	@echo "  quality       - Run all quality checks"
	@echo ""
	@echo "CI/CD commands:"
	@echo "  ci-setup      - Setup CI/CD system"
	@echo "  ci-status     - Show CI/CD system status"
	@echo "  ci-validate   - Validate CI/CD configuration"
	@echo "  ci-pipeline   - Run full CI pipeline locally"
	@echo "  ci-pre-check  - Run pre-release checks"
	@echo "  ci-release-prepare - Prepare release"
	@echo "  ci-release-create  - Create release"
	@echo "  ci-logs       - Show workflow logs"
	@echo "  ci-trigger    - Trigger workflow"
	@echo ""
	@echo "Example commands:"
	@echo "  run-example   - Run simple example"
	@echo "  check-example - Run example in check mode"
	@echo "  run-command   - Run command module example"
	@echo "  run-advanced  - Run advanced command example"
	@echo "  run-users     - Run user/group management example"
	@echo "  run-macos-users - Run macOS user/group example"
	@echo ""
	@echo "Release commands:"
	@echo "  release-test  - Test release process locally"
	@echo "  release       - Create and push release tag"
	@echo ""
	@echo "Docker commands:"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo ""
	@echo "Development commands:"
	@echo "  dev-setup     - Setup development environment"
	@echo "  clean         - Clean build artifacts"
	@echo "  docs          - Show documentation info"
	@echo "  help          - Show this help"

# Show documentation info
docs:
	@echo "Documentation:"
	@echo "  README.md     - Main project documentation"
	@echo "  docs/logo.md  - Logo design guidelines"
	@echo "  docs/README.md - Documentation overview"
	@echo ""
	@echo "Logo files:"
	@echo "  docs/logo.png - Main logo (200px width)"
	@echo "  docs/logo.svg - Vector version"
