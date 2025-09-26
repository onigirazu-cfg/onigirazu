#!/bin/bash

# CI/CD Manager Script for Onigirazu
# This script provides a unified interface for managing the CI/CD system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_header() {
    echo -e "${PURPLE}$1${NC}"
}

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

print_command() {
    echo -e "${CYAN}$ $1${NC}"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to run command with logging
run_command() {
    local cmd="$1"
    local description="$2"

    if [ -n "$description" ]; then
        print_status "$description"
    fi

    print_command "$cmd"

    if eval "$cmd"; then
        if [ -n "$description" ]; then
            print_success "$description completed"
        fi
        return 0
    else
        if [ -n "$description" ]; then
            print_error "$description failed"
        fi
        return 1
    fi
}

# Function to show CI/CD status
show_status() {
    print_header "========================================="
    print_header "         CI/CD System Status            "
    print_header "========================================="
    echo

    # Project info
    print_status "Project: $(basename $(pwd))"
    print_status "Branch: $(git branch --show-current)"
    print_status "Last commit: $(git log -1 --pretty=format:'%h - %s (%cr)')"
    echo

    # Version info
    local current_version=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    local commits_since_tag=$(git rev-list ${current_version}..HEAD --count)
    print_status "Current version: $current_version"
    print_status "Commits since last release: $commits_since_tag"
    echo

    # Workflow status
    print_status "GitHub Actions Workflows:"
    if [ -d ".github/workflows" ]; then
        for workflow in .github/workflows/*.yml; do
            if [ -f "$workflow" ]; then
                local workflow_name=$(basename "$workflow" .yml)
                print_status "  ✓ $workflow_name"
            fi
        done
    else
        print_warning "  No workflows found"
    fi
    echo

    # Tools status
    print_status "Development Tools:"
    local tools=("go" "git" "make" "docker" "gh" "goreleaser" "golangci-lint" "staticcheck")
    for tool in "${tools[@]}"; do
        if command_exists "$tool"; then
            print_status "  ✓ $tool"
        else
            print_warning "  ✗ $tool (not installed)"
        fi
    done
    echo

    # Configuration files
    print_status "Configuration Files:"
    local configs=(".goreleaser.yml" "Makefile" ".ci-config.yml" ".golangci.yml")
    for config in "${configs[@]}"; do
        if [ -f "$config" ]; then
            print_status "  ✓ $config"
        else
            print_warning "  ✗ $config (missing)"
        fi
    done
}

# Function to setup CI/CD
setup_ci() {
    print_header "========================================="
    print_header "         Setting up CI/CD System        "
    print_header "========================================="
    echo

    run_command "./scripts/setup-ci.sh setup" "Running CI/CD setup"
}

# Function to validate configuration
validate_config() {
    print_header "========================================="
    print_header "       Validating CI/CD Configuration   "
    print_header "========================================="
    echo

    run_command "./scripts/setup-ci.sh validate" "Validating CI/CD configuration"
}

# Function to run pre-release checks
pre_release_check() {
    print_header "========================================="
    print_header "        Running Pre-release Checks      "
    print_header "========================================="
    echo

    run_command "./scripts/pre-release-check.sh" "Running pre-release checks"
}

# Function to prepare release
prepare_release() {
    local release_type="$1"

    print_header "========================================="
    print_header "         Preparing Release              "
    print_header "========================================="
    echo

    if [ -n "$release_type" ]; then
        run_command "./scripts/release-helper.sh prepare $release_type" "Preparing $release_type release"
    else
        run_command "./scripts/release-helper.sh prepare" "Preparing release"
    fi
}

# Function to create release
create_release() {
    local version="$1"

    print_header "========================================="
    print_header "         Creating Release               "
    print_header "========================================="
    echo

    if [ -n "$version" ]; then
        run_command "./scripts/release-helper.sh create $version" "Creating release $version"
    else
        print_error "Version required for release creation"
        print_status "Usage: $0 release create v1.2.3"
        return 1
    fi
}

# Function to run tests
run_tests() {
    print_header "========================================="
    print_header "           Running Tests                "
    print_header "========================================="
    echo

    run_command "make test" "Running unit tests"
    run_command "make test-race" "Running race condition tests"
    run_command "make test-coverage" "Running coverage tests"
}

# Function to run linting
run_linting() {
    print_header "========================================="
    print_header "           Running Linting              "
    print_header "========================================="
    echo

    run_command "make lint" "Running linting checks"
}

# Function to run security checks
run_security() {
    print_header "========================================="
    print_header "         Running Security Checks        "
    print_header "========================================="
    echo

    run_command "make security" "Running security checks"
}

# Function to build project
build_project() {
    print_header "========================================="
    print_header "           Building Project             "
    print_header "========================================="
    echo

    run_command "make clean" "Cleaning previous builds"
    run_command "make build" "Building project"
}

# Function to run full CI pipeline locally
run_ci_pipeline() {
    print_header "========================================="
    print_header "       Running Full CI Pipeline         "
    print_header "========================================="
    echo

    print_status "This will run the complete CI pipeline locally"
    echo

    # Tests
    run_tests || return 1

    # Linting
    run_linting || return 1

    # Security
    run_security || return 1

    # Build
    build_project || return 1

    print_success "Full CI pipeline completed successfully!"
}

# Function to show workflow logs
show_logs() {
    local workflow="$1"

    if ! command_exists gh; then
        print_error "GitHub CLI (gh) is required to view workflow logs"
        return 1
    fi

    if [ -n "$workflow" ]; then
        run_command "gh run list --workflow=$workflow" "Showing logs for $workflow workflow"
    else
        run_command "gh run list" "Showing recent workflow runs"
    fi
}

# Function to trigger workflow
trigger_workflow() {
    local workflow="$1"

    if ! command_exists gh; then
        print_error "GitHub CLI (gh) is required to trigger workflows"
        return 1
    fi

    if [ -z "$workflow" ]; then
        print_error "Workflow name required"
        print_status "Available workflows:"
        ls .github/workflows/*.yml | xargs -I {} basename {} .yml
        return 1
    fi

    run_command "gh workflow run $workflow" "Triggering $workflow workflow"
}

# Function to show help
show_help() {
    echo "CI/CD Manager for Onigirazu"
    echo
    echo "Usage: $0 [command] [options]"
    echo
    echo "Commands:"
    echo "  status                    - Show CI/CD system status"
    echo "  setup                     - Setup CI/CD system"
    echo "  validate                  - Validate CI/CD configuration"
    echo "  test                      - Run tests"
    echo "  lint                      - Run linting"
    echo "  security                  - Run security checks"
    echo "  build                     - Build project"
    echo "  pipeline                  - Run full CI pipeline locally"
    echo "  pre-check                 - Run pre-release checks"
    echo "  release prepare [type]    - Prepare release (major/minor/patch)"
    echo "  release create <version>  - Create specific release"
    echo "  logs [workflow]           - Show workflow logs"
    echo "  trigger <workflow>        - Trigger workflow"
    echo "  help                      - Show this help"
    echo
    echo "Examples:"
    echo "  $0 status                 # Show system status"
    echo "  $0 pipeline               # Run full CI pipeline"
    echo "  $0 release prepare minor  # Prepare minor release"
    echo "  $0 release create v1.2.3  # Create specific release"
    echo "  $0 trigger ci             # Trigger CI workflow"
    echo "  $0 logs release           # Show release workflow logs"
}

# Function to check prerequisites
check_prerequisites() {
    # Check if we're in the right directory
    if [ ! -f "go.mod" ] || [ ! -d "cmd/onigirazu" ]; then
        print_error "This script must be run from the project root directory"
        exit 1
    fi

    # Check if scripts exist
    local scripts=("scripts/setup-ci.sh" "scripts/pre-release-check.sh" "scripts/release-helper.sh")
    for script in "${scripts[@]}"; do
        if [ ! -f "$script" ]; then
            print_error "Required script not found: $script"
            exit 1
        fi

        if [ ! -x "$script" ]; then
            print_warning "Making $script executable"
            chmod +x "$script"
        fi
    done
}

# Main function
main() {
    # Check prerequisites
    check_prerequisites

    case "${1:-status}" in
        "status")
            show_status
            ;;

        "setup")
            setup_ci
            ;;

        "validate")
            validate_config
            ;;

        "test")
            run_tests
            ;;

        "lint")
            run_linting
            ;;

        "security")
            run_security
            ;;

        "build")
            build_project
            ;;

        "pipeline")
            run_ci_pipeline
            ;;

        "pre-check")
            pre_release_check
            ;;

        "release")
            case "$2" in
                "prepare")
                    prepare_release "$3"
                    ;;
                "create")
                    create_release "$3"
                    ;;
                *)
                    print_error "Invalid release command: $2"
                    print_status "Usage: $0 release [prepare|create] [options]"
                    exit 1
                    ;;
            esac
            ;;

        "logs")
            show_logs "$2"
            ;;

        "trigger")
            trigger_workflow "$2"
            ;;

        "help")
            show_help
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
