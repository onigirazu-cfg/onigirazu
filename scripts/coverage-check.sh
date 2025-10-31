#!/bin/bash

# Coverage Check Script
# This script validates test coverage against defined thresholds
# Usage: ./scripts/coverage-check.sh [verbose]

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
MIN_COVERAGE_OVERALL=60
MIN_COVERAGE_CRITICAL=70

# Declare critical packages and their minimum coverage thresholds
declare -A CRITICAL_PACKAGES=(
    ["internal/execution"]=70
    ["internal/ssh"]=70
    ["internal/cli"]=70
    ["internal/modules"]=70
    ["internal/facts"]=70
    ["internal/output"]=70
    ["internal/checksum"]=70
    ["internal/adhoc"]=70
    ["pkg/errors"]=90
    ["pkg/types"]=70
    ["internal/config"]=75
    ["internal/engine"]=75
    ["internal/executor"]=70
    ["internal/audit"]=70
    ["internal/parser"]=70
    ["internal/validator"]=75
    ["internal/workflow"]=75
)

# Declare packages with 0% coverage expectations (no tests needed)
declare -A NO_COVERAGE_PACKAGES=(
    ["internal/interfaces"]=0
    ["cmd/onigirazu"]=0
    ["cmd/onigirazu-gen"]=0
)

VERBOSE="${1:-false}"
COVERAGE_FILE="coverage.out"
COVERAGE_TXT="coverage.txt"

# Print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

print_header() {
    echo ""
    print_status "$BLUE" "=== $1 ==="
    echo ""
}

# Check if coverage.out exists
if [ ! -f "$COVERAGE_FILE" ]; then
    print_status "$YELLOW" "Coverage file not found. Running tests with coverage..."
    go test -race -coverprofile="$COVERAGE_FILE" -covermode=atomic -timeout=10m ./... || true
fi

# Generate coverage text report
go tool cover -func="$COVERAGE_FILE" > "$COVERAGE_TXT"

# Extract overall coverage
TOTAL_COVERAGE=$(grep total "$COVERAGE_TXT" | awk '{print $NF}' | sed 's/%//')

print_header "Coverage Report"
print_status "$BLUE" "Overall Coverage: $TOTAL_COVERAGE%"

# Check overall coverage threshold
if (( $(echo "$TOTAL_COVERAGE < $MIN_COVERAGE_OVERALL" | bc -l) )); then
    print_status "$RED" "❌ Overall coverage $TOTAL_COVERAGE% is below threshold $MIN_COVERAGE_OVERALL%"
    OVERALL_PASSED=false
else
    print_status "$GREEN" "✅ Overall coverage $TOTAL_COVERAGE% meets threshold $MIN_COVERAGE_OVERALL%"
    OVERALL_PASSED=true
fi

# Check critical packages
print_header "Critical Packages Coverage"

CRITICAL_PASSED=true
FAILED_PACKAGES=()

for package in "${!CRITICAL_PACKAGES[@]}"; do
    threshold=${CRITICAL_PACKAGES[$package]}

    # Extract coverage for this package
    package_coverage=$(go tool cover -func="$COVERAGE_FILE" | grep "$package" | awk '{print $NF}' | sed 's/%//' | head -1)

    if [ -z "$package_coverage" ]; then
        package_coverage=0
    fi

    if (( $(echo "$package_coverage < $threshold" | bc -l) )); then
        print_status "$RED" "❌ $package: $package_coverage% (required: $threshold%)"
        FAILED_PACKAGES+=("$package")
        CRITICAL_PASSED=false
    else
        print_status "$GREEN" "✅ $package: $package_coverage% (required: $threshold%)"
    fi
done

# Check packages with no coverage (optional)
if [ "$VERBOSE" == "true" ]; then
    print_header "Packages with No Coverage (Expected)"
    for package in "${!NO_COVERAGE_PACKAGES[@]}"; do
        print_status "$BLUE" "ℹ️  $package (no tests expected)"
    done
fi

# List all packages with their coverage
if [ "$VERBOSE" == "true" ]; then
    print_header "All Packages Coverage"
    go tool cover -func="$COVERAGE_FILE" | grep -v "total" | sort -t: -k2 -rn | head -50
fi

# Summary
print_header "Summary"

if [ "$OVERALL_PASSED" = true ] && [ "$CRITICAL_PASSED" = true ]; then
    print_status "$GREEN" "✅ All coverage checks PASSED"
    exit 0
else
    print_status "$RED" "❌ Coverage checks FAILED"

    if [ ${#FAILED_PACKAGES[@]} -gt 0 ]; then
        echo ""
        print_status "$RED" "Failed packages:"
        for package in "${FAILED_PACKAGES[@]}"; do
            echo "  - $package"
        done
    fi

    echo ""
    print_status "$YELLOW" "Recommendations:"
    echo "  1. Add tests to improve coverage for failed packages"
    echo "  2. Use 'go test -coverprofile=coverage.out ./...' to generate coverage"
    echo "  3. Use 'go tool cover -html=coverage.out' to view visual coverage report"
    echo "  4. Run '$0 true' for verbose output"

    exit 1
fi
