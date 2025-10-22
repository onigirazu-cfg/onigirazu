# Onigirazu Docker Test Playbooks

Comprehensive test suite for testing all Onigirazu modules and functionality in Docker containers across multiple Linux distributions.

## 📋 Overview

This directory contains automated test playbooks for validating Onigirazu functionality across 9 different Linux distributions running in Docker containers:

- **Ubuntu**: 20.04, 22.04, 24.04
- **Debian**: 11, 12
- **Rocky**: 8, 9
- **Linux Base**: All systems combined

## 📁 Test Playbooks

### `00-master.yml` - Complete Test Suite

**Execution Time**: ~5-10 minutes
**Status**: Master playbook that tests all core functionality

**Coverage**:

- ✅ Phase 1: Connectivity verification (ping all hosts)
- ✅ Phase 2: File operations (create, copy, template, stat, lineinfile)
- ✅ Phase 3: Facts and debug operations
- ✅ Phase 4: User and group management
- ✅ Phase 5: System operations (services, systemd, cron)
- ✅ Phase 6: Command and shell execution
- ✅ Phase 7: Git operations (clone, pull)
- ✅ Phase 8: Cleanup

**Modules Tested** (16+):

- `ping`, `file`, `copy`, `template`, `stat`, `lineinfile`
- `facts`, `debug`, `set_fact`
- `user`, `group`
- `service`, `systemd`, `cron`
- `command`, `shell`
- `git`

### `01-concurrent-execution.yml` - Concurrent Execution & State Isolation Tests

**Execution Time**: ~3-5 minutes
**Status**: Tests v1.49.0 state isolation feature

**Coverage**:

- ✅ Concurrent file operations (multiple tasks in parallel)
- ✅ State isolation verification (facts don't interfere)
- ✅ Parallel copy operations
- ✅ High concurrency stress testing (10+ tasks per host)
- ✅ Parallel loops execution
- ✅ Concurrent command execution

**Key Features**:

- Tests max_parallel=9 (all containers simultaneously)
- Verifies each task maintains isolated state
- Validates facts aren't shared between concurrent tasks
- Stress test with 5+ concurrent file operations
- Loop-based concurrent operations

## 🚀 Quick Start

### Option 1: Using Make Commands (Recommended)

```bash
# Build project
make build

# Setup Docker environment
make docker-setup

# Start containers
make docker-up

# Run quick test (master playbook)
make docker-test-quick

# Run concurrent execution tests
make docker-test-concurrent

# Run full comprehensive suite
make docker-test-comprehensive

# View test results
make docker-test-report
make docker-test-report-json
```

### Option 2: Using Direct Commands

```bash
# Setup
./docker/setup.sh
docker-compose -f docker-compose.test.yml up -d

# Run master playbook
./bin/onigirazu playbook docker/test-playbooks/00-master.yml \
  -i docker/inventory.ini \
  -u root \
  -k docker/ssh/id_rsa

# Run concurrent tests
./bin/onigirazu playbook docker/test-playbooks/01-concurrent-execution.yml \
  -i docker/inventory.ini \
  -u root \
  -k docker/ssh/id_rsa

# View results
cat /tmp/onigirazu-docker-test-report.txt
cat /tmp/onigirazu-docker-test-report.json
```

## 📊 Test Execution Workflow

```
┌─────────────────────────────────────┐
│ 1. Build Binary                     │
│    make build                       │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 2. Setup SSH Keys & Docker Env      │
│    make docker-setup                │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 3. Start Containers (9 types)       │
│    make docker-up                   │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 4. Run Test Playbooks               │
│    - 00-master.yml                  │
│    - 01-concurrent-execution.yml    │
└────────────┬────────────────────────┘
             │
┌────────────▼────────────────────────┐
│ 5. Collect Results & Reports        │
│    make docker-test-report          │
└─────────────────────────────────────┘
```

## 📈 Test Coverage Matrix

| Module | Master | Concurrent | Container | User/Root | Notes |
|--------|--------|-----------|-----------|-----------|-------|
| ping | ✅ | ✅ | All | Both | Basic connectivity |
| file | ✅ | ✅ | All | Root | Create, edit, delete |
| copy | ✅ | ✅ | All | Root | File copying |
| template | ✅ | - | All | Root | Jinja2 rendering |
| lineinfile | ✅ | - | All | Root | Line editing |
| stat | ✅ | ✅ | All | Both | File info |
| facts | ✅ | - | All | Both | System facts |
| debug | ✅ | ✅ | All | Both | Output messages |
| set_fact | ✅ | ✅ | All | Both | Custom variables |
| user | ✅ | - | All | Root | User management |
| group | ✅ | - | All | Root | Group management |
| service | ✅ | - | All | Root | Service control |
| systemd | ✅ | - | All | Root | Systemd control |
| cron | ✅ | - | All | Root | Cron jobs |
| command | ✅ | ✅ | All | Both | Shell commands |
| shell | ✅ | - | All | Both | Shell scripts |
| git | ✅ | - | All | Both | Git ops |

## 🎯 Testing Scenarios

### Scenario 1: Verify Core Functionality (5 min)

```bash
make docker-test-quick
```

- Runs `00-master.yml` on all containers
- Validates all major modules work
- Good for CI/CD pipelines

### Scenario 2: Test State Isolation (3 min)

```bash
make docker-test-concurrent
```

- Focuses on v1.49.0 state isolation
- Tests concurrent execution
- Verifies no cross-contamination between tasks

### Scenario 3: Full Validation (10 min)

```bash
make docker-test-comprehensive
```

- Runs all playbooks
- Generates detailed reports
- Collects performance metrics
- Best for release validation

### Scenario 4: Single Container Testing

```bash
# Test just Ubuntu 24.04
./bin/onigirazu playbook docker/test-playbooks/00-master.yml \
  -i docker/inventory.ini \
  -u root \
  -k docker/ssh/id_rsa \
  --hosts ubuntu2404
```

## 📊 Test Reports

### Text Report

Located: `/tmp/onigirazu-docker-test-report.txt`

```
╔═══════════════════════════════════════════════════════════╗
║ Onigirazu Docker Test Suite - [timestamp]                ║
╚═══════════════════════════════════════════════════════════╝

[INFO] Project Directory: /path/to/onigirazu
[✓ SUCCESS] Binary found
[✓ SUCCESS] Inventory found
...
═════════════════════════════════════════════════════════════
Test Results Summary
═════════════════════════════════════════════════════════════
Total Tests:  2
Passed:       2 (100%)
Failed:       0 (0%)
═════════════════════════════════════════════════════════════
```

### JSON Report

Located: `/tmp/onigirazu-docker-test-report.json`

```json
{
  "timestamp": "2025-01-29T10:30:45Z",
  "summary": {
    "total_tests": 2,
    "passed_tests": 2,
    "failed_tests": 0,
    "skipped_tests": 0
  },
  "environment": {
    "binary": "/path/to/bin/onigirazu",
    "inventory": "/path/to/docker/inventory.ini",
    "test_playbooks": "/path/to/docker/test-playbooks",
    "ssh_user": "root"
  }
}
```

## 🔧 Configuration

### Inventory File

Location: `../inventory.ini`

Contains 9 Docker containers grouped by OS:

- `[ubuntu]` - Ubuntu 20.04, 22.04, 24.04
- `[debian]` - Debian 11, 12
- `[redhat]` - Rocky 8, 9
- `[linux]` - All systems

### SSH Keys

Location: `../ssh/`

- `id_rsa` - Private key
- `id_rsa.pub` - Public key
- `authorized_keys` - Pre-configured in containers

### Docker Compose

Location: `../../docker-compose.test.yml`

- Defines all 9 containers
- Pre-configured SSH
- Network isolation

## ⚙️ Advanced Usage

### Run Specific Container Group

```bash
# Only Ubuntu
./bin/onigirazu playbook docker/test-playbooks/00-master.yml \
  -i docker/inventory.ini \
  -u root \
  -k docker/ssh/id_rsa \
  --hosts ubuntu

# Only Debian
./bin/onigirazu playbook docker/test-playbooks/00-master.yml \
  -i docker/inventory.ini \
  -u root \
  -k docker/ssh/id_rsa \
  --hosts debian

# Only Rocky
./bin/onigirazu playbook docker/test-playbooks/00-master.yml \
  -i docker/inventory.ini \
  -u root \
  -k docker/ssh/id_rsa \
  --hosts redhat
```

### Run in Verbose Mode

```bash
make docker-test-quick -- --verbose
```

### Run with Debug Output

```bash
./bin/onigirazu playbook docker/test-playbooks/00-master.yml \
  -i docker/inventory.ini \
  -u root \
  -k docker/ssh/id_rsa \
  --debug
```

### Parallel Execution

All playbooks support `max_parallel` setting:

```yaml
plays:
  - name: "Test"
    hosts: "all"
    max_parallel: 9  # Run on all 9 containers simultaneously
    tasks: []
```

## 🐛 Troubleshooting

### Issue: SSH Connection Failed

```bash
# Verify SSH keys exist
ls -la docker/ssh/

# Regenerate SSH keys
make docker-setup

# Check container status
docker-compose -f docker-compose.test.yml ps
```

### Issue: Binary Not Found

```bash
# Build the binary
make build

# Verify binary exists
ls -la bin/onigirazu
```

### Issue: Container Not Running

```bash
# Start containers
make docker-up

# Check container logs
make docker-logs

# Verify all containers are healthy
docker-compose -f docker-compose.test.yml ps
```

### Issue: Permission Denied

```bash
# Make sure you have Docker permissions
sudo usermod -aG docker $USER
newgrp docker
```

## 📚 Modules Tested in Detail

### File Operations (`00-master.yml` - Phase 2)

- Create directories with permissions
- Create files with content
- Copy files
- Render Jinja2 templates
- Edit lines in files
- Get file statistics

### User Management (`00-master.yml` - Phase 4)

- Create test user with home directory
- Create test group
- Verify user exists via `id` command
- Cleanup (remove user and group)

### System Operations (`00-master.yml` - Phase 5)

- Check SSH service status
- Manage systemd services
- Create and manage cron jobs
- List active cron jobs

### Concurrent Execution (`01-concurrent-execution.yml`)

- Run 3+ tasks in parallel
- Set different facts in parallel (no interference)
- Verify each task has isolated state
- 10+ concurrent tasks stress test
- Loop-based concurrent operations

## 📝 Test Playbook Structure

All playbooks follow this structure:

```yaml
name: "Descriptive test name"
description: "What is being tested"

plays:
  - name: "Phase N: Description"
    hosts: "all"  # or specific group
    max_parallel: 9  # optional

    tasks:
      - name: "Task description"
        module: "module_name"
        args:
          param1: value1
          param2: value2
```

## 🔄 CI/CD Integration

### GitHub Actions Example

```yaml
- name: Run Docker Tests
  run: make docker-test-comprehensive

- name: Upload Report
  uses: actions/upload-artifact@v2
  with:
    name: docker-test-report
    path: /tmp/onigirazu-docker-test-report.*
```

## 📊 Test Metrics

### Execution Time

- **Quick Test** (`00-master.yml`): 5-10 minutes
- **Concurrent Test** (`01-concurrent-execution.yml`): 3-5 minutes
- **Full Suite** (both playbooks): 10-15 minutes

### Coverage

- **Modules**: 16+ core modules
- **Containers**: 9 different Linux distributions
- **Scenarios**: 30+ test scenarios
- **Concurrent Tasks**: 10+ simultaneous operations

### Success Rate

- Target: 100% pass rate
- Current: Baseline tests (no failures expected)
- Expected Coverage: 85-95% of core functionality

## 🎓 Learning Resources

- Main README: `../../README.md`
- Examples: `../../examples/`
- Documentation: `../../docs/`
- Module Reference: `../../docs/MODULES.md`
- Release Notes: `../../RELEASE_NOTES_v1.49.0.md`

## 📞 Support

For issues with tests:

1. Check container logs: `make docker-logs`
2. Run in verbose mode: `--verbose` flag
3. Check SSH connectivity: `ssh -i docker/ssh/id_rsa root@<container_ip>`
4. Review test playbook YAML syntax
5. Check Onigirazu version: `./bin/onigirazu --version`

## 🔗 Related Commands

```bash
# Container Management
make docker-up                      # Start containers
make docker-down                    # Stop containers
make docker-setup                   # Setup environment
make docker-logs                    # View logs

# Testing
make docker-test-quick              # Quick test
make docker-test-concurrent         # Concurrent tests
make docker-test-comprehensive      # Full suite

# Reports
make docker-test-report             # Show text report
make docker-test-report-json        # Show JSON report
```

---

**Last Updated**: v1.49.0
**Test Suite Version**: 1.0
**Compatibility**: Onigirazu v1.49.0+
