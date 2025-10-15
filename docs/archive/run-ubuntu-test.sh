#!/bin/bash

# Onigirazu Ubuntu Integration Test Runner
# Tests all modules on real Ubuntu machine

set -e

echo "========================================="
echo "Onigirazu Ubuntu Integration Test"
echo "========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
UBUNTU_HOST="172.16.246.128"
UBUNTU_USER="usx"
INVENTORY="test-inventory-ubuntu.yml"
PLAYBOOK="test-all-modules.yml"
BINARY="./bin/onigirazu"

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

# Run the test playbook
echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Running Comprehensive Module Tests${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# Run with verbose output
$BINARY \
    -playbook "$PLAYBOOK" \
    -inventory "$INVENTORY" \
    -verbose 2>&1 | tee /tmp/onigirazu-test-output.log

EXIT_CODE=${PIPESTATUS[0]}

echo ""
echo -e "${BLUE}=========================================${NC}"

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ ALL TESTS PASSED!${NC}"
    echo ""
    echo -e "${GREEN}Test Summary:${NC}"
    echo "  • All modules tested successfully"
    echo "  • No errors detected"
    echo "  • System cleaned up"
    echo ""
    echo -e "${BLUE}Tested Modules:${NC}"
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
    echo -e "${BLUE}Log saved to:${NC} /tmp/onigirazu-test-output.log"
else
    echo -e "${RED}❌ TESTS FAILED!${NC}"
    echo ""
    echo -e "${RED}Exit code: $EXIT_CODE${NC}"
    echo -e "${YELLOW}Check the log for details:${NC} /tmp/onigirazu-test-output.log"
    echo ""
    echo -e "${YELLOW}Common issues:${NC}"
    echo "  • SSH connection problems"
    echo "  • Missing sudo privileges"
    echo "  • Package manager locked"
    echo "  • Insufficient permissions"
fi

echo -e "${BLUE}=========================================${NC}"
echo ""

exit $EXIT_CODE
