# Docker Testing - Quick Start Guide

## ⚡ Get Started in 3 Steps (15 minutes)

### Step 1: Build & Setup (5 min)

```bash
# Build the binary
make build

# Setup Docker environment (one-time)
make docker-setup
```

### Step 2: Start Containers (3 min)

```bash
# Start all 9 Docker containers
make docker-up

# Optional: Verify containers are running
docker-compose -f docker-compose.test.yml ps
```

### Step 3: Run Tests (5-15 min)

Choose one:

#### Option A: Quick Test (5 min)

```bash
make docker-test-quick
```

✅ Tests all core functionality quickly

#### Option B: Concurrent Tests (3 min)

```bash
make docker-test-concurrent
```

✅ Tests v1.49.0 state isolation feature

#### Option C: Full Suite (15 min)

```bash
make docker-test-comprehensive
```

✅ Tests everything + generates detailed reports

---

## 📊 View Results

### After Tests Complete

```bash
# View summary report
make docker-test-report

# View JSON report
make docker-test-report-json

# View raw logs
cat /tmp/onigirazu-docker-test-report.txt
cat /tmp/onigirazu-docker-test-report.json
```

---

## 🎯 What Gets Tested

### Master Test Suite (00-master.yml)

- ✅ Connectivity to all 9 containers
- ✅ File operations (create, copy, template, etc.)
- ✅ User/Group management
- ✅ System services (SSH, systemd, cron)
- ✅ Command execution
- ✅ Git operations
- ✅ Facts gathering

### Concurrent Execution Tests (01-concurrent-execution.yml)

- ✅ Parallel task execution
- ✅ State isolation (v1.49.0 feature)
- ✅ Concurrent file operations
- ✅ Stress testing with 10+ simultaneous tasks
- ✅ Parallel loops

---

## 📈 Container Coverage

Tests run on all 9 Linux distributions:

| OS | Versions | Count |
|----|----------|-------|
| Ubuntu | 20.04, 22.04, 24.04 | 3 |
| Debian | 11, 12 | 2 |
| Rocky | 8, 9 | 2 |
| **Total** | - | **9** |

Each test runs on all containers in parallel.

---

## 🔧 Common Commands

```bash
# Setup (do once)
make docker-setup
make docker-up

# Run tests (choose one)
make docker-test-quick                    # Fast
make docker-test-concurrent               # v1.49.0 feature focus
make docker-test-comprehensive            # Everything

# View results
make docker-test-report
make docker-test-report-json

# Cleanup
make docker-down
make docker-logs
```

---

## ⚠️ Troubleshooting

### Problem: "Binary not found"

```bash
make build
```

### Problem: "SSH key not found"

```bash
make docker-setup
```

### Problem: "Containers not running"

```bash
make docker-up
# Wait 30 seconds, then run tests
```

### Problem: "Permission denied"

```bash
sudo usermod -aG docker $USER
newgrp docker
```

---

## 📊 Expected Results

### Successful Test Run

```
╔═══════════════════════════════════════════════════════════╗
║ Test Results Summary
╚═══════════════════════════════════════════════════════════╝
Total Tests:  2
Passed:       2 (100%)
Failed:       0 (0%)
═════════════════════════════════════════════════════════════
```

### Test Report Location

- `/tmp/onigirazu-docker-test-report.txt` - Human readable
- `/tmp/onigirazu-docker-test-report.json` - Machine readable
- `/tmp/onigirazu-test-*.log` - Individual test logs

---

## 🚀 Next Steps

After tests pass:

1. **Review Results**

   ```bash
   make docker-test-report
   ```

2. **Check for Any Failures**
   - Look for `[✗ ERROR]` in report
   - Check individual log files

3. **Stop Containers** (when done)

   ```bash
   make docker-down
   ```

4. **Run Tests Again Anytime**

   ```bash
   make docker-test-quick
   ```

---

## 📚 Learn More

- Full Documentation: `README.md` in this directory
- Test Details: `IMPLEMENTATION_SUMMARY.md`
- Main Project: `../../README.md`

---

## 💡 Pro Tips

1. **First Time?** Start with `docker-test-quick`
2. **Check State Isolation?** Use `docker-test-concurrent`
3. **Before Release?** Use `docker-test-comprehensive`
4. **In CI/CD?** Capture reports: `docker-test-*.log`
5. **Save Time?** Run tests while working on other tasks

---

## ✅ Checklist

- [ ] Built binary: `make build`
- [ ] Setup environment: `make docker-setup`
- [ ] Started containers: `make docker-up`
- [ ] Ran tests: `make docker-test-quick`
- [ ] Checked results: `make docker-test-report`
- [ ] All tests passed? ✅

---

**Ready?** Let's go!

```bash
make build && make docker-setup && make docker-up && make docker-test-quick
```

Then view results:

```bash
make docker-test-report
```

---

**Questions?** Check `README.md` for detailed documentation.
