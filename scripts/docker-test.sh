#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
INVENTORY="$PROJECT_DIR/docker/inventory.ini"
BINARY="$PROJECT_DIR/bin/onigirazu"
SSH_USER="root"
SSH_KEY="$PROJECT_DIR/docker/ssh/id_rsa"

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

check_containers() {
    log_info "Checking Docker container status..."
    docker-compose -f "$PROJECT_DIR/docker-compose.test.yml" ps
    echo ""
}

test_group() {
    local group=$1
    local description=$2
    
    log_info "Testing $description..."
    
    if $BINARY run "$group" -i "$INVENTORY" -m ping -u "$SSH_USER" -k "$SSH_KEY" 2>&1; then
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
    shift 2
    local description="${!#}"
    local args=("${@:1:$#-1}")
    
    log_info "Testing $description on $group..."
    
    if $BINARY run "$group" -i "$INVENTORY" -m "$module" "${args[@]}" -u "$SSH_USER" -k "$SSH_KEY" 2>&1; then
        log_success "$description test passed"
        return 0
    else
        log_error "$description test failed"
        return 1
    fi
}

main() {
    log_info "Starting comprehensive Onigirazu tests on Docker containers"
    echo ""
    
    if [ ! -f "$BINARY" ]; then
        log_error "Binary not found at $BINARY. Run 'make build' first."
        exit 1
    fi
    
    if [ ! -f "$INVENTORY" ]; then
        log_error "Inventory not found at $INVENTORY"
        exit 1
    fi
    
    check_containers
    
    local failed_tests=0
    local passed_tests=0
    
    log_info "=== Phase 1: Connectivity Tests ==="
    echo ""
    
    for group in ubuntu debian redhat; do
        if test_group "$group" "$group hosts"; then
            ((passed_tests++))
        else
            ((failed_tests++))
        fi
        echo ""
    done
    
    log_info "=== Phase 2: Ad-hoc Command Tests ==="
    echo ""
    
    if test_adhoc_command "linux" "command" -a "cmd=uname -a" "uname command"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi
    echo ""
    
    if test_adhoc_command "linux" "shell" -a "cmd=echo 'Hello from Onigirazu'" "shell command"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi
    echo ""
    
    log_info "=== Phase 3: File Operations Tests ==="
    echo ""
    
    if test_adhoc_command "linux" "file" -a "path=/tmp/onigirazu-test" -a "state=directory" -a "mode=0755" "create directory"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi
    echo ""
    
    if test_adhoc_command "linux" "copy" -a "content=Test from Onigirazu" -a "dest=/tmp/onigirazu-test/test.txt" -a "mode=0644" "copy file"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi
    echo ""
    
    if test_adhoc_command "linux" "file" -a "path=/tmp/onigirazu-test" -a "state=absent" "cleanup test files"; then
        ((passed_tests++))
    else
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
