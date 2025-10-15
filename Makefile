.PHONY: build test clean run-example install lint fmt vet security coverage release-test ci-setup ci-status ci-validate ci-pipeline ci-pre-check ci-release-prepare ci-release-create test-race test-coverage docs docs-generate docs-serve docs-open docs-clean vagrant-up vagrant-up-all vagrant-halt vagrant-halt-all vagrant-destroy vagrant-destroy-all vagrant-status vagrant-ssh vagrant-test vagrant-test-all vagrant-provision

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

# Clean documentation
docs-clean:
	@echo "Cleaning generated documentation..."
	rm -rf docs/api/pkg docs/api/internal
	rm -f docs/api/*.md docs/api/index.html
	@echo "✅ Documentation cleaned"

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
	@echo ""
	@echo "Documentation commands:"
	@echo "  docs          - Show documentation info"
	@echo "  docs-generate - Generate API documentation"
	@echo "  docs-serve    - Start documentation server"
	@echo "  docs-open     - Open documentation in browser"
	@echo "  docs-clean    - Clean generated documentation"
	@echo ""
	@echo "Vagrant commands:"
	@echo "  vagrant-up         - Start a specific VM"
	@echo "  vagrant-up-all     - Start all VMs"
	@echo "  vagrant-halt       - Stop a specific VM"
	@echo "  vagrant-halt-all   - Stop all VMs"
	@echo "  vagrant-destroy    - Destroy a specific VM"
	@echo "  vagrant-destroy-all - Destroy all VMs"
	@echo "  vagrant-status     - Show status of all VMs"
	@echo "  vagrant-ssh        - SSH into a specific VM"
	@echo "  vagrant-provision  - Re-provision a specific VM"
	@echo "  vagrant-test       - Test Onigirazu against VM group"
	@echo "  vagrant-test-all   - Run comprehensive tests on all VMs"
	@echo ""
	@echo "Other commands:"
	@echo "  help          - Show this help"

# Generate API documentation
docs-generate:
	@echo "Generating API documentation..."
	./scripts/generate-docs.sh
	go run scripts/docgen/main.go
	@echo "✅ Documentation generated:"
	@echo "  📄 Markdown: docs/api/"
	@echo "  🌐 HTML: docs/api/index.html"
	@echo "  🚀 View: open docs/api/index.html"

# Start documentation server
docs-serve:
	@echo "Starting documentation server..."
	@echo "📖 Documentation will be available at: http://localhost:8080"
	@echo "🔗 Direct link: http://localhost:8080/github.com/onigirazu-cfg/onigirazu"
	@echo ""
	@echo "Press Ctrl+C to stop the server"
	@echo ""
	~/go/bin/pkgsite -http=:8080

# Open documentation in browser
docs-open:
	@echo "Opening documentation..."
	open docs/api/index.html

# Show documentation info
docs:
	@echo "Documentation commands:"
	@echo "  docs-generate - Generate API documentation (markdown + HTML)"
	@echo "  docs-serve    - Start local documentation server"
	@echo "  docs-open     - Open HTML documentation in browser"
	@echo ""
	@echo "Documentation files:"
	@echo "  README.md     - Main project documentation"
	@echo "  docs/api/     - Auto-generated API documentation"
	@echo "  docs/logo.md  - Logo design guidelines"
	@echo ""
	@echo "Logo files:"
	@echo "  docs/logo.png - Main logo (200px width)"
	@echo "  docs/logo.svg - Vector version"

# Vagrant commands for local testing
vagrant-up:
	@read -p "Enter VM name (ubuntu2004, ubuntu2204, debian11, centos7, rocky8, opensuse15, freebsd13, etc.): " vm; \
	vagrant up $$vm

vagrant-up-all:
	@echo "Starting all VMs (this may take a while)..."
	vagrant up ubuntu2004 ubuntu2204 ubuntu2404 debian11 debian12 centos7 rocky8 rocky9 opensuse15 freebsd13 freebsd14

vagrant-halt:
	@read -p "Enter VM name to halt: " vm; \
	vagrant halt $$vm

vagrant-halt-all:
	vagrant halt

vagrant-destroy:
	@read -p "Enter VM name to destroy: " vm; \
	vagrant destroy -f $$vm

vagrant-destroy-all:
	vagrant destroy -f

vagrant-status:
	vagrant status

vagrant-ssh:
	@read -p "Enter VM name to SSH into: " vm; \
	vagrant ssh $$vm

vagrant-provision:
	@read -p "Enter VM name to provision: " vm; \
	vagrant provision $$vm

vagrant-test: build
	@echo "Testing Onigirazu against Vagrant VMs..."
	@read -p "Enter VM group (ubuntu, debian, redhat, suse, bsd, linux, all): " group; \
	./bin/onigirazu run -i vagrant/inventory.ini -m ping --limit $$group

vagrant-test-all: build
	@echo "Running comprehensive tests on all VMs..."
	./scripts/vagrant-test.sh
