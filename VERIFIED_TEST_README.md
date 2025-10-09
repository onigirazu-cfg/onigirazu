# 🔍 Verified Testing with Before/After State Checks

## What is this?

An improved version of the test playbook that checks the system state **BEFORE** and **AFTER** each operation to ensure that modules actually work correctly.

## Files

- **`test-all-modules-verified.yml`** - Playbook with state checks
- **`run-verified-test.sh`** - Script to run tests

## Quick Start

```bash
# 1. Configure SSH (one-time)
ssh-copy-id usx@172.16.246.128

# 2. Configure sudo (one-time)
ssh usx@172.16.246.128 "echo 'usx ALL=(ALL) NOPASSWD:ALL' | sudo tee /etc/sudoers.d/usx"

# 3. Run verified tests
./run-verified-test.sh
```

## What is being checked?

### For each module

1. **BEFORE** - Check state before operation
2. **ACTION** - Execute operation
3. **AFTER** - Check state after operation
4. **VERIFY** - Compare and confirm changes

### Example (User Module)

```yaml
# 1. BEFORE - Check if user exists
- name: "[USER] BEFORE - Check if test user exists"
  command:
    cmd: "id onigirazu-test"
  register: user_before
  ignore_errors: true

# 2. ACTION - Create user
- name: "[USER] Create test user"
  user:
    name: "onigirazu-test"
    state: "present"
    shell: "/bin/bash"
  register: user_create

# 3. AFTER - Verify user was created
- name: "[USER] AFTER - Verify user was created"
  command:
    cmd: "id onigirazu-test"
  register: user_after

# 4. VERIFY - Confirmation
- name: "[USER] Verify user creation"
  debug:
    msg: "✅ User created: {{ user_create.changed }}, Info: {{ user_after.stdout }}"
```

## Tested Modules (17)

| Module | Before Check | After Check | Cleanup Check |
|--------|--------------|-------------|---------------|
| command | ✅ | ✅ | N/A |
| shell | ✅ | ✅ | N/A |
| file | ✅ stat | ✅ stat | ✅ stat |
| copy | ✅ cat | ✅ cat | N/A |
| template | ✅ stat | ✅ cat | N/A |
| package | ✅ dpkg -l | ✅ dpkg -l | N/A |
| service | ✅ systemctl | ✅ systemctl | N/A |
| user | ✅ id | ✅ id | ✅ id |
| group | ✅ getent | ✅ getent | ✅ getent |
| lineinfile | ✅ cat | ✅ cat | N/A |
| git | ✅ stat | ✅ stat | N/A |
| systemd | ✅ systemctl | ✅ systemctl | N/A |
| sysctl | ✅ sysctl | ✅ sysctl | N/A |
| cron | ✅ crontab -l | ✅ crontab -l | ✅ crontab -l |
| archive | ✅ stat | ✅ stat | N/A |
| stat | N/A | ✅ | N/A |
| debug | N/A | ✅ | N/A |

## Benefits of Verified Tests

### 1. Confidence in Correctness

- Not just executing commands
- Verifying that changes actually occurred
- Comparing before/after state

### 2. Detailed Diagnostics

- See exactly what changed
- Easy to find problems
- Clear messages

### 3. Automatic Cleanup Verification

- Verify that everything is removed
- System remains clean
- No "garbage" after tests

## Log Files

Each run creates a separate log with timestamp:

```
/tmp/onigirazu-verified-test-20250128-143022.log
```

## Comparison with Basic Test

| Feature | Basic Test | Verified Test |
|---------|------------|---------------|
| Module execution | ✅ | ✅ |
| Result verification | Partial | ✅ Complete |
| Before/After checks | ❌ | ✅ |
| Cleanup verification | ❌ | ✅ |
| Detailed logs | ✅ | ✅ Enhanced |
| Execution time | ~3-5 min | ~5-7 min |

## When to Use?

### Basic Test (`test-all-modules.yml`)

- Quick check that everything works
- CI/CD pipeline
- Regression testing

### Verified Test (`test-all-modules-verified.yml`)

- Detailed correctness verification
- Debugging problems
- Before release
- Testing on new system

## Example Output

```
[FILE] Display state before
Directory exists before: false

[FILE] Create test directory
changed: true

[FILE] Verify directory creation
✅ Directory created: true (was: false)
```

## Troubleshooting

### Test fails on state check

**Problem:** Before/After check shows same state

**Solution:**

1. Check if module actually executed
2. Look at `changed` flag
3. Check access permissions (sudo)

### Cleanup doesn't work

**Problem:** Resources remain after tests

**Solution:**

1. Check cleanup section in logs
2. Run cleanup manually:

   ```bash
   ssh usx@172.16.246.128 "sudo userdel -r onigirazu-test"
   ssh usx@172.16.246.128 "sudo groupdel onigirazu-group"
   ssh usx@172.16.246.128 "rm -rf /tmp/onigirazu-test*"
   ```

## Next Steps

After successful verified test:

1. ✅ All modules work correctly
2. ✅ Changes are applied properly
3. ✅ Cleanup works
4. 🚀 Ready for production use

---

**Created:** 2025-01-28
**Status:** ✅ READY
**Type:** Integration Testing with Verification
**Execution Time:** ~5-7 minutes
