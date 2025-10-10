#!/bin/bash

# Pre-release Check Script for Onigirazu
# This script performs comprehensive checks before creating a release

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
CHECKS_PASSED=0
CHECKS_FAILED=0
CHECKS_WARNING=0

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
    CHECKS_PASSED=$((CHECKS_PASSED + 1))
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
    CHECKS_WARNING=$((CHECKS_WARNING + 1))
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
    CHECKS_FAILED=$((CHECKS_FAILED + 1))
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to run check with error handling
run_check() {
    local check_name="$1"
    local check_command="$2"
    local is_critical="${3:-true}"

    print_status "Running $check_name..."

    if eval "$check_command" >/dev/null 2>&1; then
        print_success "$check_name passed"
        return 0
    else
        if [ "$is_critical" = "true" ]; then
            print_error "$check_name failed"
            return 1
        else
            print_warning "$check_name failed (non-critical)"
            return 0
        fi
    fi
}

# Check Go version
check_go_version() {
    if command_exists go; then
        local go_version=$(go version | awk '{print $3}' | sed 's/go//')
        local required_version="1.22"

        if [ "$(printf '%s\n' "$required_version" "$go_version" | sort -V | head -n1)" = "$required_version" ]; then
            print_success "Go version $go_version is compatible"
        else
            print_error "Go version $go_version is too old. Required: $required_version or higher"
            return 1
        fi
    else
        print_error "Go is not installed"
        return 1
    fi
}

# Check working tree
check_working_tree() {
    if git diff-index --quiet HEAD --; then
        print_success "Working tree is clean"
    else
        print_error "Working tree has uncommitted changes"
        return 1
    fi
}

# Check branch
check_branch() {
    local current_branch=$(git branch --show-current)
    if [ "$current_branch" = "main" ] || [ "$current_branch" = "master" ]; then
        print_success "On main branch ($current_branch)"
    else
        print_warning "Not on main branch (current: $current_branch)"
    fi
}

# Check for required files
check_required_files() {
    local required_files=(
        "go.mod"
        "go.sum"
        ".goreleaser.yml"
        "Makefile"
        "README.md"
        "LICENSE"
    )

    local missing_files=()

    for file in "${required_files[@]}"; do
        if [ -f "$file" ]; then
            print_success "Required file exists: $file"
        else
            print_error "Missing required file: $file"
            missing_files+=("$file")
        fi
    done

    if [ ${#missing_files[@]} -gt 0 ]; then
        return 1
    fi
}

# Check Go modules
check_go_modules() {
    print_status "Checking Go modules..."

    # Check if go.mod is valid
    if go mod verify >/dev/null 2>&1; then
        print_success "Go modules verified"
    else
        print_error "Go modules verification failed"
        return 1
    fi

    # Check for tidy modules
    go mod tidy
    if git diff --quiet go.mod go.sum; then
        print_success "Go modules are tidy"
    else
        print_warning "Go modules need tidying (auto-fixed)"
    fi
}

# Run tests
check_tests() {
    print_status "Running tests..."

    # Unit tests
    if go test ./... -v; then
        print_success "Unit tests passed"
    else
        print_error "Unit tests failed"
        return 1
    fi

    # Race condition tests
    if go test -race ./...; then
        print_success "Race condition tests passed"
    else
        print_error "Race condition tests failed"
        return 1
    fi

    # Test coverage
    go test -coverprofile=coverage.out ./...
    local coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    local threshold=70

    if (( $(echo "$coverage >= $threshold" | bc -l) )); then
        print_success "Test coverage: ${coverage}% (>= ${threshold}%)"
    else
        print_error "Test coverage: ${coverage}% (< ${threshold}%)"
        return 1
    fi
}

# Run linting
check_linting() {
    print_status "Running linting..."

    # Go fmt
    if [ -z "$(gofmt -l .)" ]; then
        print_success "Code is properly formatted"
    else
        print_error "Code formatting issues found"
        return 1
    fi

    # Go vet
    if go vet ./...; then
        print_success "Go vet passed"
    else
        print_error "Go vet failed"
        return 1
    fi

    # golangci-lint
    if command_exists golangci-lint; then
        if golangci-lint run; then
            print_success "golangci-lint passed"
        else
            print_error "golangci-lint failed"
            return 1
        fi
    else
        print_warning "golangci-lint not installed"
    fi

    # staticcheck
    if command_exists staticcheck; then
        if staticcheck ./...; then
            print_success "staticcheck passed"
        else
            print_error "staticcheck failed"
            return 1
        fi
    else
        print_warning "staticcheck not installed"
    fi
}

# Check security
check_security() {
    print_status "Running security checks..."

    # gosec
    if command_exists gosec; then
        if gosec ./...; then
            print_success "gosec security scan passed"
        else
            print_error "gosec security scan failed"
            return 1
        fi
    else
        print_warning "gosec not installed"
    fi

    # govulncheck
    if command_exists govulncheck; then
        if govulncheck ./...; then
            print_success "Vulnerability check passed"
        else
            print_error "Vulnerability check failed"
            return 1
        fi
    else
        print_warning "govulncheck not installed"
    fi
}

# Check build
check_build() {
    print_status "Testing build..."

    # Clean build
    if make clean && make build; then
        print_success "Build successful"

        # Test binary
        if [ -f "bin/onigirazu" ]; then
            if ./bin/onigirazu --version >/dev/null 2>&1; then
                print_success "Binary works correctly"
            else
                print_error "Binary has issues"
                return 1
            fi
        else
            print_error "Binary not found after build"
            return 1
        fi
    else
        print_error "Build failed"
        return 1
    fi
}

# Check GoReleaser configuration
check_goreleaser() {
    if [ -f ".goreleaser.yml" ]; then
        if command_exists goreleaser; then
            if goreleaser check; then
                print_success "GoReleaser configuration is valid"
            else
                print_error "GoReleaser configuration has issues"
                return 1
            fi
        else
            print_warning "GoReleaser not installed"
        fi
    else
        print_error "GoReleaser configuration not found"
        return 1
    fi
}

# Check documentation
check_documentation() {
    print_status "Checking documentation..."

    # README.md
    if [ -f "README.md" ] && [ -s "README.md" ]; then
        print_success "README.md exists and is not empty"
    else
        print_error "README.md is missing or empty"
        return 1
    fi

    # LICENSE
    if [ -f "LICENSE" ] && [ -s "LICENSE" ]; then
        print_success "LICENSE file exists"
    else
        print_error "LICENSE file is missing or empty"
        return 1
    fi

    # Go doc coverage (basic check)
    local undocumented=$(find . -name "*.go" -not -path "./vendor/*" -exec grep -l "^func [A-Z]" {} \; | \
        xargs -I {} sh -c 'grep -B1 "^func [A-Z]" {} | grep -v "^//"' | \
        grep -c "^func [A-Z]" || echo "0")

    if [ "$undocumented" -eq 0 ]; then
        print_success "All exported functions are documented"
    else
        print_warning "$undocumented exported functions may lack documentation"
    fi
}

# Check dependencies
check_dependencies() {
    print_status "Checking dependencies..."

    # Check for outdated dependencies
    local outdated=$(go list -u -m all | grep -c "\[" || echo "0")

    if [ "$outdated" -eq 0 ]; then
        print_success "All dependencies are up to date"
    else
        print_warning "$outdated dependencies have updates available"
    fi
}

# Check Git tags
check_git_tags() {
    local current_version=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    local commits_since_tag=$(git rev-list ${current_version}..HEAD --count)

    print_status "Current version: $current_version"
    print_status "Commits since last tag: $commits_since_tag"

    if [ "$commits_since_tag" -gt 0 ]; then
        print_success "Ready for new release"
    else
        print_warning "No commits since last release"
    fi
}

# Generate summary
generate_summary() {
    echo
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}         Pre-release Check Summary      ${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo

    echo -e "✅ Checks passed: ${GREEN}$CHECKS_PASSED${NC}"
    echo -e "⚠️  Warnings: ${YELLOW}$CHECKS_WARNING${NC}"
    echo -e "❌ Checks failed: ${RED}$CHECKS_FAILED${NC}"
    echo

    if [ "$CHECKS_FAILED" -eq 0 ]; then
        echo -e "${GREEN}🎉 All critical checks passed! Ready for release.${NC}"
        return 0
    else
        echo -e "${RED}💥 Some critical checks failed. Please fix before releasing.${NC}"
        return 1
    fi
}

# Main function
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}    Onigirazu Pre-release Checker      ${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo

    # Check if we're in the right directory
    if [ ! -f "go.mod" ] || [ ! -d "cmd/onigirazu" ]; then
        print_error "This script must be run from the project root directory"
        exit 1
    fi

    # Run all checks
    check_go_version || true
    check_working_tree || true
    check_branch || true
    check_required_files || true
    check_go_modules || true
    check_tests || true
    check_linting || true
    check_security || true
    check_build || true
    check_goreleaser || true
    check_documentation || true
    check_dependencies || true
    check_git_tags || true

    # Generate summary and exit with appropriate code
    if generate_summary; then
        exit 0
    else
        exit 1
    fi
}

# Run main function
main "$@"
