#!/bin/bash

# ============================================================================
# Onigirazu Docker Testing - Quick Start Script
# ============================================================================
# This script shows you exactly how to get started with Docker testing
# ============================================================================

cat << 'EOF'

╔═══════════════════════════════════════════════════════════════════════════╗
║                                                                           ║
║   🐳 ONIGIRAZU DOCKER TESTING - QUICK START                             ║
║                                                                           ║
║   Comprehensive test suite for all modules across 9 Linux distributions   ║
║                                                                           ║
╚═══════════════════════════════════════════════════════════════════════════╝

📋 WHAT YOU HAVE:
  ✅ Master test playbook (16+ modules, 8 phases)
  ✅ Concurrent execution tests (state isolation for v1.49.0)
  ✅ Automated test runner with reporting
  ✅ Easy Make commands for everything
  ✅ Comprehensive documentation

═══════════════════════════════════════════════════════════════════════════

🚀 GET STARTED IN 3 STEPS:

Step 1: Build Binary
───────────────────────────────────────────────────────────────────────────
$ make build

Step 2: Setup Docker Environment
───────────────────────────────────────────────────────────────────────────
$ make docker-setup
$ make docker-up

Step 3: Run Tests
───────────────────────────────────────────────────────────────────────────
Choose ONE:

  Option A - Quick Test (5 min):
  $ make docker-test-quick

  Option B - Concurrent Tests (3 min):
  $ make docker-test-concurrent

  Option C - Full Suite (15 min):
  $ make docker-test-comprehensive

Then view results:
  $ make docker-test-report

═══════════════════════════════════════════════════════════════════════════

⚡ SINGLE COMMAND START:

$ make build && make docker-setup && make docker-up && make docker-test-quick

Then:

$ make docker-test-report

═══════════════════════════════════════════════════════════════════════════

📊 TEST COVERAGE:

Container Platforms (9 total):
  • Ubuntu: 20.04, 22.04, 24.04
  • Debian: 11, 12
  • Rocky: 8, 9

Modules Tested (16+):
  • ping, file, copy, template, lineinfile, stat
  • facts, debug, set_fact
  • user, group
  • service, systemd, cron
  • command, shell
  • git

Features:
  • 30+ test scenarios
  • Concurrent execution tests
  • State isolation verification (v1.49.0)
  • Automated reporting (text + JSON)
  • Performance metrics

═══════════════════════════════════════════════════════════════════════════

📍 IMPORTANT LOCATIONS:

Test Files:
  📁 docker/test-playbooks/
     ├── 00-master.yml (all core functionality)
     ├── 01-concurrent-execution.yml (state isolation)
     ├── README.md (full documentation)
     ├── QUICK_START.md (quick reference)
     └── IMPLEMENTATION_SUMMARY.md (technical details)

Test Results:
  📄 /tmp/onigirazu-docker-test-report.txt (human readable)
  📄 /tmp/onigirazu-docker-test-report.json (machine readable)
  📄 /tmp/onigirazu-test-*.log (individual test logs)

═══════════════════════════════════════════════════════════════════════════

🎯 TEST MODES:

1. QUICK TEST (5 minutes)
   Tests all core modules on all 9 containers
   $ make docker-test-quick

2. CONCURRENT TEST (3 minutes)
   Tests v1.49.0 state isolation with stress testing
   $ make docker-test-concurrent

3. FULL SUITE (15 minutes)
   Runs all tests and generates comprehensive reports
   $ make docker-test-comprehensive

═══════════════════════════════════════════════════════════════════════════

📚 DOCUMENTATION:

  Full Guide:      docker/test-playbooks/README.md
  Quick Start:     docker/test-playbooks/QUICK_START.md
  Technical:       docker/test-playbooks/IMPLEMENTATION_SUMMARY.md
  This Summary:    DOCKER_TESTING_TIER1_COMPLETE.md

═══════════════════════════════════════════════════════════════════════════

⚙️ ALL MAKE COMMANDS:

Setup & Start:
  make docker-setup              # Setup SSH environment
  make docker-up                 # Start containers
  make docker-down               # Stop containers

Run Tests:
  make docker-test-quick         # 5 min - core tests
  make docker-test-concurrent    # 3 min - state isolation
  make docker-test-comprehensive # 15 min - everything

View Results:
  make docker-test-report        # Show text report
  make docker-test-report-json   # Show JSON report
  make docker-logs               # Show container logs

Cleanup:
  make clean                     # Clean build artifacts

═══════════════════════════════════════════════════════════════════════════

🔍 EXPECTED OUTPUT (after tests):

  ╔═══════════════════════════════════════════════════════════╗
  ║ Test Results Summary
  ╚═══════════════════════════════════════════════════════════╝
  Total Tests:  2
  ✓ Passed:     2 (100%)
  ✗ Failed:     0 (0%)

═══════════════════════════════════════════════════════════════════════════

❓ TROUBLESHOOTING:

Q: Binary not found?
A: $ make build

Q: SSH keys not found?
A: $ make docker-setup

Q: Containers not running?
A: $ make docker-up
   (wait 30 seconds)

Q: Permission denied?
A: $ sudo usermod -aG docker $USER
   $ newgrp docker

═══════════════════════════════════════════════════════════════════════════

✅ READY? START NOW:

FAST START (copy & paste):
─────────────────────────────────────────────────────────────────────────

make build && \
make docker-setup && \
make docker-up && \
make docker-test-quick && \
make docker-test-report

═══════════════════════════════════════════════════════════════════════════

📞 NEED HELP?

  1. Check quick reference:    docker/test-playbooks/QUICK_START.md
  2. Read full guide:          docker/test-playbooks/README.md
  3. View test playbooks:      docker/test-playbooks/*.yml
  4. Check logs:               make docker-logs

═══════════════════════════════════════════════════════════════════════════

🎊 YOU'RE ALL SET!

All test files are in: docker/test-playbooks/
No files in repository root - everything organized! ✅

Start testing now:
  $ make docker-test-quick

View results:
  $ make docker-test-report

═══════════════════════════════════════════════════════════════════════════

Onigirazu v1.49.0 - Docker Testing Suite (Tier 1)
Ready to test all modules across 9 Linux distributions!

EOF

echo ""
echo "═══════════════════════════════════════════════════════════════════════════"
echo ""
echo "🚀 NEXT STEP: Run 'make docker-test-quick' to start testing!"
echo ""
