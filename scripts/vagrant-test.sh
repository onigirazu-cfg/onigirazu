#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
INVENTORY="$PROJECT_DIR/vagrant/inventory.ini"
BINARY="$PROJECT_DIR/bin/onigirazu"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

check_vagrant_status() {
    log_info "Checking Vagrant VM status..."
    vagrant status | grep -E "running|poweroff|not created" || true
    echo ""
}

test_group() {
    local group=$1
    local description=$2
    
    log_info "Testing $description..."
    
    if $BINARY run -i "$INVENTORY" -m ping --limit "$group" 2>&1; then
        log_success "$description test passed"
        return 0
    else
        log_error "$description test failed"
        return 1
    fi
}

test_adhoc_command() {
    local group=$1
    local module=$2
    local args=$3
    local description=$4
    
    log_info "Testing $description on $group..."
    
    if $BINARY run -i "$INVENTORY" -m "$module" -a "$args" --limit "$group" 2>&1; then
        log_success "$description test passed"
        return 0
    else
        log_error "$description test failed"
        return 1
    fi
}

main() {
    log_info "Starting comprehensive Onigirazu tests on Vagrant VMs"
    echo ""
    
    if [ ! -f "$BINARY" ]; then
        log_error "Binary not found at $BINARY. Run 'make build' first."
        exit 1
    fi
    
    if [ ! -f "$INVENTORY" ]; then
        log_error "Inventory not found at $INVENTORY"
        exit 1
    fi
    
    check_vagrant_status
    
    local failed_tests=0
    local passed_tests=0
    
    log_info "=== Phase 1: Connectivity Tests ==="
    echo ""
    
    for group in ubuntu debian redhat suse bsd; do
        if test_group "$group" "$group hosts"; then
            ((passed_tests++))
        else
            ((failed_tests++))
        fi
        echo ""
    done
    
    log_info "=== Phase 2: Ad-hoc Command Tests ==="
    echo ""
    
    if test_adhoc_command "linux" "command" "uname -a" "uname command"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi
    echo ""
    
    if test_adhoc_command "linux" "shell" "echo 'Hello from Onigirazu'" "shell command"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi
    echo ""
    
    if test_adhoc_command "linux" "setup" "" "gather facts"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi
    echo ""
    
    log_info "=== Phase 3: File Operations Tests ==="
    echo ""
    
    if test_adhoc_command "linux" "file" "path=/tmp/onigirazu-test state=directory mode=0755" "create directory"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi
    echo ""
    
    if test_adhoc_command "linux" "copy" "content='Test from Onigirazu' dest=/tmp/onigirazu-test/test.txt mode=0644" "copy file"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi
    echo ""
    
    if test_adhoc_command "linux" "file" "path=/tmp/onigirazu-test state=absent" "cleanup test files"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi
    echo ""
    
    log_info "=== Phase 4: User Authentication Tests ==="
    echo ""
    
    log_info "Testing with testuser (password authentication)..."
    if $BINARY run -i "$INVENTORY" -m ping --limit "testuser" -u testuser 2>&1; then
        log_success "Password authentication test passed"
        ((passed_tests++))
    else
        log_warning "Password authentication test failed (may not be supported yet)"
        ((failed_tests++))
    fi
    echo ""
    
    log_info "=== Test Summary ==="
    echo ""
    log_info "Total tests: $((passed_tests + failed_tests))"
    log_success "Passed: $passed_tests"
    
    if [ $failed_tests -gt 0 ]; then
        log_error "Failed: $failed_tests"
        exit 1
    else
        log_success "All tests passed!"
        exit 0
    fi
}

main "$@"
