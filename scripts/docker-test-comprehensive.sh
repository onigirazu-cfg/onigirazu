#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY="$PROJECT_DIR/bin/onigirazu"
INVENTORY="$PROJECT_DIR/docker/inventory.ini"
TEST_PLAYBOOKS_DIR="$PROJECT_DIR/docker/test-playbooks"
SSH_USER="root"
SSH_KEY="$PROJECT_DIR/docker/ssh/id_rsa"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Test results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Test report
TEST_REPORT=""
TEST_REPORT_FILE="/tmp/onigirazu-docker-test-report.txt"
TEST_JSON_REPORT="/tmp/onigirazu-docker-test-report.json"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$TEST_REPORT_FILE"
}

log_success() {
    echo -e "${GREEN}[✓ SUCCESS]${NC} $1" | tee -a "$TEST_REPORT_FILE"
}

log_error() {
    echo -e "${RED}[✗ ERROR]${NC} $1" | tee -a "$TEST_REPORT_FILE"
}

log_warning() {
    echo -e "${YELLOW}[⚠ WARNING]${NC} $1" | tee -a "$TEST_REPORT_FILE"
}

log_section() {
    echo "" | tee -a "$TEST_REPORT_FILE"
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════╗${NC}" | tee -a "$TEST_REPORT_FILE"
    echo -e "${CYAN}║ $1${NC}" | tee -a "$TEST_REPORT_FILE"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════╝${NC}" | tee -a "$TEST_REPORT_FILE"
    echo "" | tee -a "$TEST_REPORT_FILE"
}

# Initialize test report
init_report() {
    > "$TEST_REPORT_FILE"
    log_section "Onigirazu Docker Test Suite - $(date)"
    log_info "Project Directory: $PROJECT_DIR"
    log_info "Test Playbooks: $TEST_PLAYBOOKS_DIR"
    log_info "Binary: $BINARY"
    log_info "Inventory: $INVENTORY"
    log_info "SSH Key: $SSH_KEY"
}

# Check prerequisites
check_prerequisites() {
    log_section "Checking Prerequisites"

    if [ ! -f "$BINARY" ]; then
        log_error "Binary not found at $BINARY"
        log_info "Run 'make build' first"
        exit 1
    fi
    log_success "Binary found"

    if [ ! -f "$INVENTORY" ]; then
        log_error "Inventory not found at $INVENTORY"
        exit 1
    fi
    log_success "Inventory found"

    if [ ! -d "$TEST_PLAYBOOKS_DIR" ]; then
        log_error "Test playbooks directory not found at $TEST_PLAYBOOKS_DIR"
        exit 1
    fi
    log_success "Test playbooks directory found"

    if [ ! -f "$SSH_KEY" ]; then
        log_error "SSH key not found at $SSH_KEY"
        exit 1
    fi
    log_success "SSH key found"
}

# Check Docker containers
check_containers() {
    log_section "Checking Docker Containers"

    if ! command -v docker-compose &> /dev/null; then
        log_warning "docker-compose not found, skipping container check"
        return 1
    fi

    log_info "Running container status check..."
    if docker-compose -f "$PROJECT_DIR/docker-compose.test.yml" ps 2>&1 | tee -a "$TEST_REPORT_FILE"; then
        log_success "Container status retrieved"
    else
        log_warning "Could not retrieve container status"
        return 1
    fi
}

# Run a single test playbook
run_test_playbook() {
    local playbook=$1
    local playbook_name=$(basename "$playbook" .yml)

    log_info "Running: $playbook_name"

    ((TOTAL_TESTS++))

    local output_file="/tmp/onigirazu-test-${playbook_name}.log"
    > "$output_file"

    if "$BINARY" apply "$playbook" \
        -i "$INVENTORY" \
        2>&1 | tee -a "$output_file"; then

        log_success "$playbook_name completed successfully"
        ((PASSED_TESTS++))
        return 0
    else
        log_error "$playbook_name failed"
        log_info "Details saved to: $output_file"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Run all tests
run_all_tests() {
    log_section "Running Comprehensive Test Suite"

    # Master test playbook
    if [ -f "$TEST_PLAYBOOKS_DIR/00-master.yml" ]; then
        run_test_playbook "$TEST_PLAYBOOKS_DIR/00-master.yml"
    else
        log_warning "Master test playbook not found, skipping"
    fi

    # Concurrent execution test
    if [ -f "$TEST_PLAYBOOKS_DIR/01-concurrent-execution.yml" ]; then
        run_test_playbook "$TEST_PLAYBOOKS_DIR/01-concurrent-execution.yml"
    else
        log_warning "Concurrent execution test playbook not found, skipping"
    fi
}

# Performance metrics
collect_metrics() {
    log_section "Collecting Performance Metrics"

    local start_time=$(date +%s%N)

    # Run a simple ping test to measure baseline
    if "$BINARY" run "all" -i "$INVENTORY" -m ping -u "$SSH_USER" -k "$SSH_KEY" \
        --quiet 2>&1 >> "$TEST_REPORT_FILE"; then

        local end_time=$(date +%s%N)
        local duration=$(( (end_time - start_time) / 1000000 ))

        log_info "Baseline execution time: ${duration}ms"
    else
        log_warning "Could not collect performance metrics"
    fi
}

# Generate summary report
generate_report() {
    log_section "Test Summary Report"

    local passed_percent=0
    local failed_percent=0

    if [ $TOTAL_TESTS -gt 0 ]; then
        passed_percent=$(( (PASSED_TESTS * 100) / TOTAL_TESTS ))
        failed_percent=$(( (FAILED_TESTS * 100) / TOTAL_TESTS ))
    fi

    log_info "Total Tests: $TOTAL_TESTS"
    log_success "Passed: $PASSED_TESTS ($passed_percent%)"

    if [ $FAILED_TESTS -gt 0 ]; then
        log_error "Failed: $FAILED_TESTS ($failed_percent%)"
    else
        log_success "Failed: 0 (0%)"
    fi

    if [ $SKIPPED_TESTS -gt 0 ]; then
        log_warning "Skipped: $SKIPPED_TESTS"
    fi

    # Print summary to console
    echo ""
    echo -e "${CYAN}═════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}Test Results Summary${NC}"
    echo -e "${CYAN}═════════════════════════════════════════════════════════════${NC}"
    echo -e "Total Tests:  $TOTAL_TESTS"
    echo -e "${GREEN}Passed:       $PASSED_TESTS ($passed_percent%)${NC}"

    if [ $FAILED_TESTS -gt 0 ]; then
        echo -e "${RED}Failed:       $FAILED_TESTS ($failed_percent%)${NC}"
    else
        echo -e "${GREEN}Failed:       0 (0%)${NC}"
    fi

    if [ $SKIPPED_TESTS -gt 0 ]; then
        echo -e "${YELLOW}Skipped:      $SKIPPED_TESTS${NC}"
    fi

    echo -e "${CYAN}═════════════════════════════════════════════════════════════${NC}"
    echo ""

    # Generate JSON report
    generate_json_report
}

# Generate JSON report
generate_json_report() {
    cat > "$TEST_JSON_REPORT" <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "summary": {
    "total_tests": $TOTAL_TESTS,
    "passed_tests": $PASSED_TESTS,
    "failed_tests": $FAILED_TESTS,
    "skipped_tests": $SKIPPED_TESTS
  },
  "environment": {
    "binary": "$BINARY",
    "inventory": "$INVENTORY",
    "test_playbooks": "$TEST_PLAYBOOKS_DIR",
    "ssh_user": "$SSH_USER"
  }
}
EOF
    log_info "JSON report saved to: $TEST_JSON_REPORT"
}

# Main execution
main() {
    init_report
    check_prerequisites
    check_containers
    run_all_tests
    collect_metrics
    generate_report

    # Print report file location
    log_section "Test Report Files"
    log_info "Full report: $TEST_REPORT_FILE"
    log_info "JSON report: $TEST_JSON_REPORT"

    # Exit with appropriate code
    if [ $FAILED_TESTS -gt 0 ]; then
        echo ""
        log_error "Test suite failed with $FAILED_TESTS failures"
        exit 1
    else
        echo ""
        log_success "All tests passed! ✓"
        exit 0
    fi
}

# Show help
show_help() {
    cat <<EOF
Usage: $0 [OPTIONS]

Comprehensive Docker test suite for Onigirazu

OPTIONS:
    -h, --help              Show this help message
    -p, --playbook FILE     Run specific test playbook
    -q, --quick             Run only quick tests (master playbook)
    -c, --concurrent        Run only concurrent execution tests
    -m, --metrics           Collect performance metrics
    -r, --report            Show test report
    -v, --verbose           Enable verbose output

EXAMPLES:
    # Run full test suite
    $0

    # Run only quick tests
    $0 --quick

    # Run concurrent tests
    $0 --concurrent

    # View test report
    $0 --report

EOF
}

# Handle arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -q|--quick)
            QUICK_MODE=true
            shift
            ;;
        -c|--concurrent)
            CONCURRENT_ONLY=true
            shift
            ;;
        -r|--report)
            if [ -f "$TEST_REPORT_FILE" ]; then
                cat "$TEST_REPORT_FILE"
            else
                echo "No test report found. Run tests first."
            fi
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Run main function
main "$@"
