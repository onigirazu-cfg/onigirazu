#!/bin/bash

# CI/CD Setup Script for Onigirazu
# This script helps set up the CI/CD environment and checks prerequisites

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check Go version
check_go_version() {
    if command_exists go; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        REQUIRED_VERSION="1.22"

        if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" = "$REQUIRED_VERSION" ]; then
            print_success "Go version $GO_VERSION is compatible"
            return 0
        else
            print_error "Go version $GO_VERSION is too old. Required: $REQUIRED_VERSION or higher"
            return 1
        fi
    else
        print_error "Go is not installed"
        return 1
    fi
}

# Function to install development tools
install_dev_tools() {
    print_status "Installing development tools..."

    # Go tools
    go install honnef.co/go/tools/cmd/staticcheck@latest
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    go install golang.org/x/tools/cmd/goimports@latest
    go install golang.org/x/vuln/cmd/govulncheck@latest
    go install github.com/goreleaser/goreleaser@latest

    print_success "Development tools installed"
}

# Function to check GitHub CLI
check_github_cli() {
    if command_exists gh; then
        print_success "GitHub CLI is available"

        # Check if authenticated
        if gh auth status >/dev/null 2>&1; then
            print_success "GitHub CLI is authenticated"
        else
            print_warning "GitHub CLI is not authenticated. Run 'gh auth login' to authenticate"
        fi
    else
        print_warning "GitHub CLI is not installed. Install it for better GitHub integration"
        print_status "Visit: https://cli.github.com/"
    fi
}

# Function to check Docker
check_docker() {
    if command_exists docker; then
        if docker info >/dev/null 2>&1; then
            print_success "Docker is available and running"
        else
            print_warning "Docker is installed but not running"
        fi
    else
        print_warning "Docker is not installed. Required for container builds"
    fi
}

# Function to validate GitHub Actions workflows
validate_workflows() {
    print_status "Validating GitHub Actions workflows..."

    WORKFLOW_DIR=".github/workflows"
    if [ -d "$WORKFLOW_DIR" ]; then
        WORKFLOW_COUNT=$(find "$WORKFLOW_DIR" -name "*.yml" -o -name "*.yaml" | wc -l)
        print_success "Found $WORKFLOW_COUNT workflow files"

        # List workflows
        for workflow in "$WORKFLOW_DIR"/*.yml "$WORKFLOW_DIR"/*.yaml; do
            if [ -f "$workflow" ]; then
                WORKFLOW_NAME=$(basename "$workflow")
                print_status "  - $WORKFLOW_NAME"
            fi
        done
    else
        print_error "GitHub Actions workflows directory not found"
        return 1
    fi
}

# Function to check GoReleaser configuration
check_goreleaser() {
    if [ -f ".goreleaser.yml" ]; then
        print_success "GoReleaser configuration found"

        if command_exists goreleaser; then
            print_status "Validating GoReleaser configuration..."
            if goreleaser check; then
                print_success "GoReleaser configuration is valid"
            else
                print_error "GoReleaser configuration has issues"
                return 1
            fi
        else
            print_warning "GoReleaser not installed. Install it to validate configuration"
        fi
    else
        print_error "GoReleaser configuration (.goreleaser.yml) not found"
        return 1
    fi
}

# Function to check required secrets
check_secrets() {
    print_status "Checking required GitHub secrets..."

    REQUIRED_SECRETS=(
        "GH_TOKEN"
        "CODECOV_TOKEN"
        "DOCKERHUB_USERNAME"
        "DOCKERHUB_TOKEN"
        "GPG_FINGERPRINT"
        "COSIGN_PRIVATE_KEY"
        "COSIGN_PASSWORD"
        "FURY_TOKEN"
    )

    print_status "Required secrets for full CI/CD functionality:"
    for secret in "${REQUIRED_SECRETS[@]}"; do
        print_status "  - $secret"
    done

    print_warning "Make sure these secrets are configured in your GitHub repository settings"
    print_status "Repository Settings > Secrets and variables > Actions"
}

# Function to run tests
run_tests() {
    print_status "Running tests to verify setup..."

    if go test ./...; then
        print_success "All tests passed"
    else
        print_error "Some tests failed"
        return 1
    fi
}

# Function to test build
test_build() {
    print_status "Testing build process..."

    if make build; then
        print_success "Build successful"

        # Test the binary
        if [ -f "bin/onigirazu" ]; then
            print_status "Testing binary..."
            if ./bin/onigirazu --version >/dev/null 2>&1; then
                print_success "Binary works correctly"
            else
                print_warning "Binary exists but may have issues"
            fi
        fi
    else
        print_error "Build failed"
        return 1
    fi
}

# Function to show CI/CD status
show_status() {
    print_status "CI/CD Setup Status:"
    echo

    # Go version
    if check_go_version >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓${NC} Go version compatible"
    else
        echo -e "  ${RED}✗${NC} Go version incompatible or missing"
    fi

    # Development tools
    if command_exists golangci-lint && command_exists staticcheck; then
        echo -e "  ${GREEN}✓${NC} Development tools installed"
    else
        echo -e "  ${RED}✗${NC} Development tools missing"
    fi

    # GitHub CLI
    if command_exists gh; then
        echo -e "  ${GREEN}✓${NC} GitHub CLI available"
    else
        echo -e "  ${YELLOW}!${NC} GitHub CLI not installed"
    fi

    # Docker
    if command_exists docker && docker info >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓${NC} Docker available"
    else
        echo -e "  ${YELLOW}!${NC} Docker not available"
    fi

    # Workflows
    if [ -d ".github/workflows" ]; then
        echo -e "  ${GREEN}✓${NC} GitHub Actions workflows present"
    else
        echo -e "  ${RED}✗${NC} GitHub Actions workflows missing"
    fi

    # GoReleaser
    if [ -f ".goreleaser.yml" ]; then
        echo -e "  ${GREEN}✓${NC} GoReleaser configuration present"
    else
        echo -e "  ${RED}✗${NC} GoReleaser configuration missing"
    fi

    echo
}

# Main function
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}    Onigirazu CI/CD Setup Script       ${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo

    # Check if we're in the right directory
    if [ ! -f "go.mod" ] || [ ! -d "cmd/onigirazu" ]; then
        print_error "This script must be run from the project root directory"
        exit 1
    fi

    case "${1:-setup}" in
        "setup")
            print_status "Setting up CI/CD environment..."

            check_go_version || exit 1
            install_dev_tools
            check_github_cli
            check_docker
            validate_workflows || exit 1
            check_goreleaser || exit 1
            check_secrets
            run_tests || exit 1
            test_build || exit 1

            print_success "CI/CD setup completed successfully!"
            echo
            print_status "Next steps:"
            print_status "1. Configure required GitHub secrets"
            print_status "2. Push changes to trigger CI/CD workflows"
            print_status "3. Create a release tag to test the release process"
            ;;

        "status")
            show_status
            ;;

        "validate")
            print_status "Validating CI/CD configuration..."
            validate_workflows || exit 1
            check_goreleaser || exit 1
            run_tests || exit 1
            print_success "Validation completed successfully!"
            ;;

        "tools")
            install_dev_tools
            ;;

        "help")
            echo "Usage: $0 [command]"
            echo
            echo "Commands:"
            echo "  setup     - Full CI/CD setup (default)"
            echo "  status    - Show CI/CD status"
            echo "  validate  - Validate configuration"
            echo "  tools     - Install development tools"
            echo "  help      - Show this help"
            ;;

        *)
            print_error "Unknown command: $1"
            print_status "Run '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
