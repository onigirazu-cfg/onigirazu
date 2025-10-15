#!/bin/bash

# Onigirazu Ubuntu Integration Test Runner with State Verification
# Tests all modules with before/after state checks

set -e

echo "========================================="
echo "Onigirazu Verified Integration Test"
echo "With Before/After State Verification"
echo "========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
UBUNTU_HOST="172.16.246.128"
UBUNTU_USER="usx"
INVENTORY="test-inventory-ubuntu.yml"
PLAYBOOK="test-all-modules-verified.yml"
BINARY="./bin/onigirazu"
LOG_FILE="/tmp/onigirazu-verified-test-$(date +%Y%m%d-%H%M%S).log"

# Check if binary exists
if [ ! -f "$BINARY" ]; then
    echo -e "${YELLOW}Binary not found. Building...${NC}"
    make build
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ Build failed!${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Build successful${NC}"
    echo ""
fi

# Display binary version
echo -e "${BLUE}Binary Information:${NC}"
$BINARY --version
echo ""

# Check SSH connectivity
echo -e "${BLUE}Checking SSH connectivity to ${UBUNTU_HOST}...${NC}"
if ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no ${UBUNTU_USER}@${UBUNTU_HOST} "echo 'SSH connection successful'" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ SSH connection successful${NC}"
else
    echo -e "${RED}❌ Cannot connect to ${UBUNTU_HOST}${NC}"
    echo -e "${YELLOW}Please ensure:${NC}"
    echo "  1. Host is reachable: ping ${UBUNTU_HOST}"
    echo "  2. SSH is running on the host"
    echo "  3. SSH key is configured: ssh-copy-id ${UBUNTU_USER}@${UBUNTU_HOST}"
    echo "  4. User has sudo privileges"
    exit 1
fi
echo ""

# Display target system info
echo -e "${BLUE}Target System Information:${NC}"
ssh -o StrictHostKeyChecking=no ${UBUNTU_USER}@${UBUNTU_HOST} "uname -a && lsb_release -a 2>/dev/null || cat /etc/os-release" 2>/dev/null
echo ""

# Check sudo access
echo -e "${BLUE}Checking sudo access...${NC}"
if ssh -o StrictHostKeyChecking=no ${UBUNTU_USER}@${UBUNTU_HOST} "sudo -n true" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Passwordless sudo is configured${NC}"
else
    echo -e "${YELLOW}⚠️  Passwordless sudo is NOT configured${NC}"
    echo -e "${YELLOW}Some tests may require password input${NC}"
    echo ""
    echo -e "${YELLOW}To enable passwordless sudo, run on the Ubuntu machine:${NC}"
    echo "  echo '${UBUNTU_USER} ALL=(ALL) NOPASSWD:ALL' | sudo tee /etc/sudoers.d/${UBUNTU_USER}"
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi
echo ""

# Display test information
echo -e "${CYAN}=========================================${NC}"
echo -e "${CYAN}Test Configuration${NC}"
echo -e "${CYAN}=========================================${NC}"
echo -e "${BLUE}Playbook:${NC} $PLAYBOOK"
echo -e "${BLUE}Inventory:${NC} $INVENTORY"
echo -e "${BLUE}Log file:${NC} $LOG_FILE"
echo -e "${BLUE}Features:${NC}"
echo "  • Before/After state verification"
echo "  • 17 modules tested"
echo "  • Automatic cleanup"
echo "  • Detailed logging"
echo ""

# Run the test playbook
echo -e "${CYAN}=========================================${NC}"
echo -e "${CYAN}Running Verified Module Tests${NC}"
echo -e "${CYAN}=========================================${NC}"
echo ""

# Run with verbose output
$BINARY \
    -playbook "$PLAYBOOK" \
    -inventory "$INVENTORY" \
    -verbose 2>&1 | tee "$LOG_FILE"

EXIT_CODE=${PIPESTATUS[0]}

echo ""
echo -e "${CYAN}=========================================${NC}"

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ ALL VERIFIED TESTS PASSED!${NC}"
    echo ""
    echo -e "${GREEN}Test Summary:${NC}"
    echo "  • All modules tested with state verification"
    echo "  • Before/After checks completed"
    echo "  • No errors detected"
    echo "  • System cleaned up"
    echo ""
    echo -e "${BLUE}Tested Modules (17):${NC}"
    echo "  ✅ command      - Command execution"
    echo "  ✅ shell        - Shell commands with pipes"
    echo "  ✅ file         - File and directory management"
    echo "  ✅ copy         - File copying"
    echo "  ✅ template     - Template rendering"
    echo "  ✅ package      - Package management (APT)"
    echo "  ✅ service      - Service management"
    echo "  ✅ user         - User management"
    echo "  ✅ group        - Group management"
    echo "  ✅ lineinfile   - Line-in-file editing"
    echo "  ✅ git          - Git operations"
    echo "  ✅ systemd      - Systemd service control"
    echo "  ✅ sysctl       - Kernel parameters"
    echo "  ✅ cron         - Cron job management"
    echo "  ✅ archive      - Archive creation"
    echo "  ✅ stat         - File statistics"
    echo "  ✅ debug        - Debug output"
    echo ""
    echo -e "${BLUE}Verification Features:${NC}"
    echo "  ✅ State checked before each operation"
    echo "  ✅ State verified after each operation"
    echo "  ✅ Changes confirmed with actual system state"
    echo "  ✅ Cleanup verified"
    echo ""
    echo -e "${BLUE}Log saved to:${NC} $LOG_FILE"
    echo ""
    echo -e "${GREEN}=========================================${NC}"
    echo -e "${GREEN}🎉 VERIFICATION SUCCESSFUL!${NC}"
    echo -e "${GREEN}All modules work correctly!${NC}"
    echo -e "${GREEN}=========================================${NC}"
else
    echo -e "${RED}❌ TESTS FAILED!${NC}"
    echo ""
    echo -e "${RED}Exit code: $EXIT_CODE${NC}"
    echo -e "${YELLOW}Check the log for details:${NC} $LOG_FILE"
    echo ""
    echo -e "${YELLOW}Common issues:${NC}"
    echo "  • SSH connection problems"
    echo "  • Missing sudo privileges"
    echo "  • Package manager locked"
    echo "  • Insufficient permissions"
    echo "  • Network connectivity issues"
    echo ""
    echo -e "${YELLOW}Troubleshooting:${NC}"
    echo "  1. Check SSH: ssh ${UBUNTU_USER}@${UBUNTU_HOST}"
    echo "  2. Check sudo: ssh ${UBUNTU_USER}@${UBUNTU_HOST} 'sudo -n true'"
    echo "  3. Check logs: cat $LOG_FILE"
    echo "  4. See guide: UBUNTU_TESTING_GUIDE.md"
fi

echo -e "${CYAN}=========================================${NC}"
echo ""

exit $EXIT_CODE
