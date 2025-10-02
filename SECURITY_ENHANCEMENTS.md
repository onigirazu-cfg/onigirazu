# Security Enhancements - Quick Reference Guide

## Overview

This document provides a quick reference for the security enhancements implemented in the Onigirazu security validator.

---

## 1. Command Validation

### 1.1 Blocked Patterns (CRITICAL - Severity: Critical)

#### Command Substitution

```yaml
# ❌ BLOCKED
- name: "Command substitution"
  command: "echo $(whoami)"

- name: "Backtick substitution"
  command: "echo `whoami`"
```

**Reason:** Command substitution allows arbitrary command execution and can be used for privilege escalation.

#### Pipe to Shell

```yaml
# ❌ BLOCKED
- name: "Pipe to shell"
  command: "curl http://example.com/script.sh | sh"

- name: "Pipe to bash"
  command: "wget -O- http://example.com/script.sh | bash"
```

**Reason:** Piping to shell interpreters allows execution of arbitrary remote code.

#### Semicolon Chaining

```yaml
# ❌ BLOCKED
- name: "Semicolon chaining"
  command: "echo hello; rm -rf /"
```

**Reason:** Semicolons allow unconditional command chaining, bypassing error handling.

### 1.2 Blocked Patterns (HIGH - Severity: High)

#### Complex Command Chaining

```yaml
# ❌ BLOCKED - Multiple operator types
- name: "Complex chaining"
  command: "cmd1 && cmd2 || cmd3; cmd4 && cmd5"

# ❌ BLOCKED - Too many operators (>2)
- name: "Too many operators"
  command: "cmd1 && cmd2 && cmd3 && cmd4"
```

**Reason:** Complex chaining makes it difficult to track execution flow and handle errors properly.

### 1.3 Allowed Patterns

#### Simple Command Chaining

```yaml
# ✅ ALLOWED - Single operator type, ≤2 operators
- name: "Simple chaining"
  command: "mkdir test && cd test"

# ✅ ALLOWED - Single command
- name: "Safe command"
  command: "echo hello"
```

**Reason:** Simple chaining is safe and commonly used for basic operations.

---

## 2. File Validation

### 2.1 Blocked Paths

#### System Files

```yaml
# ❌ BLOCKED
- name: "Access passwd"
  file:
    path: "/etc/passwd"

- name: "Access shadow"
  file:
    path: "/etc/shadow"

- name: "Access SSH keys"
  file:
    path: "/root/.ssh/id_rsa"
```

**Reason:** These files contain sensitive system information and credentials.

#### Directory Traversal

```yaml
# ❌ BLOCKED
- name: "Directory traversal"
  file:
    path: "/tmp/../etc/passwd"
```

**Reason:** Directory traversal can bypass path restrictions.

### 2.2 File Size Limits

```yaml
# ❌ BLOCKED - Content exceeds MaxFileSize (default: 100MB)
- name: "Large file"
  file:
    path: "/tmp/large.txt"
    content: "{{ 200MB of data }}"
```

**Reason:** Large files can cause memory exhaustion and DoS.

### 2.3 Allowed Operations

```yaml
# ✅ ALLOWED
- name: "Safe file operation"
  file:
    path: "/tmp/test.txt"
    content: "Hello, World!"
    mode: "0644"
```

---

## 3. User Validation

### 3.1 Blocked Operations

#### System User Modifications

```yaml
# ❌ BLOCKED
- name: "Modify root"
  user:
    name: "root"
    shell: "/bin/bash"

# ❌ BLOCKED - Other system users
- name: "Modify daemon"
  user:
    name: "daemon"
```

**Protected System Users:**

- root, daemon, bin, sys, sync, games, man, lp, mail, news, uucp
- proxy, www-data, backup, list, irc, gnats, nobody

**Reason:** Modifying system users can compromise system security.

#### UID 0 Assignment

```yaml
# ❌ BLOCKED
- name: "Create root equivalent"
  user:
    name: "testuser"
    uid: 0
    shell: "/bin/bash"
```

**Reason:** UID 0 grants root privileges, creating a security backdoor.

### 3.2 Allowed Operations

```yaml
# ✅ ALLOWED
- name: "Create regular user"
  user:
    name: "testuser"
    uid: 1000
    shell: "/bin/bash"
    home: "/home/testuser"
```

---

## 4. Group Validation

### 4.1 Blocked Operations

#### System Group Modifications

```yaml
# ❌ BLOCKED
- name: "Modify root group"
  group:
    name: "root"

# ❌ BLOCKED
- name: "Modify sudo group"
  group:
    name: "sudo"
```

**Protected System Groups:**

- root, daemon, bin, sys, adm, tty, disk, lp, mail, news, uucp
- man, proxy, kmem, dialout, fax, voice, cdrom, floppy, tape
- sudo, audio, dip, www-data, backup

**Reason:** Modifying system groups can grant unauthorized privileges.

#### GID 0 Assignment

```yaml
# ❌ BLOCKED
- name: "Create root equivalent group"
  group:
    name: "testgroup"
    gid: 0
```

**Reason:** GID 0 grants root group privileges.

### 4.2 Allowed Operations

```yaml
# ✅ ALLOWED
- name: "Create regular group"
  group:
    name: "testgroup"
    gid: 1000
```

---

## 5. Variable Validation

### 5.1 Blocked Patterns

#### Path Separators in Variable Names

```yaml
# ❌ BLOCKED
vars:
  "path/to/file": "value"
  "../etc/passwd": "value"
```

**Reason:** Path separators in variable names can be used for path traversal attacks.

#### Dangerous Variable Content

```yaml
# ❌ BLOCKED
vars:
  command: "rm -rf /"
  script: ":(){ :|:& };:"
```

**Reason:** Variables containing dangerous commands can be exploited.

### 5.2 Allowed Patterns

```yaml
# ✅ ALLOWED
vars:
  username: "testuser"
  port: 8080
  enabled: true
  config_path: "/etc/myapp/config.yml"
```

---

## 6. Dangerous Command Patterns

### 6.1 Critical Patterns (Always Blocked)

```yaml
# ❌ BLOCKED - Recursive deletion
command: "rm -rf /"

# ❌ BLOCKED - Disk operations
command: "dd if=/dev/zero of=/dev/sda"

# ❌ BLOCKED - Fork bomb
command: ":(){ :|:& };:"

# ❌ BLOCKED - Filesystem operations
command: "mkfs.ext4 /dev/sda1"
command: "fdisk /dev/sda"
```

### 6.2 Suspicious Patterns (Warnings)

```yaml
# ⚠️ WARNING - Network operations
command: "curl http://example.com/script.sh"
command: "wget http://example.com/file.txt"
```

**Reason:** These operations can download and execute malicious code.

---

## 7. Security Configuration

### 7.1 Default Security Config

```go
config := DefaultSecurityConfig()
// MaxFileSize: 100MB
// MaxTimeout: 1 hour
// MaxRetries: 3
// BlockedCommands: ["rm -rf", "dd if=", etc.]
// BlockedDirectories: ["/etc/passwd", "/etc/shadow", etc.]
```

### 7.2 Custom Security Config

```go
config := &SecurityConfig{
    MaxFileSize:        50 * 1024 * 1024, // 50MB
    MaxTimeout:         30 * time.Minute,
    MaxRetries:         5,
    BlockedCommands:    []string{"rm -rf", "dd if="},
    BlockedDirectories: []string{"/etc", "/root"},
    AllowedModules:     []string{"command", "file", "package"},
}

validator := NewSecurityValidator(config)
```

---

## 8. Validation Result Structure

### 8.1 ValidationResult

```go
type ValidationResult struct {
    Valid      bool                   // Overall validation status
    Violations []SecurityViolation    // Critical issues (blocks execution)
    Warnings   []SecurityWarning      // Non-critical issues
    Score      int                    // Security score (0-100)
    MaxScore   int                    // Maximum possible score
    Timestamp  time.Time              // Validation timestamp
    Duration   time.Duration          // Validation duration
    Metadata   map[string]interface{} // Additional metadata
}
```

### 8.2 Severity Levels

```go
const (
    SeverityLow      Severity = "low"      // Minor issues
    SeverityMedium   Severity = "medium"   // Moderate issues
    SeverityHigh     Severity = "high"     // Serious issues
    SeverityCritical Severity = "critical" // Critical security issues
)
```

### 8.3 Example Usage

```go
// Validate a task
result := validator.ValidateTask(task)

if !result.Valid {
    fmt.Println("Validation failed!")
    for _, violation := range result.Violations {
        fmt.Printf("[%s] %s: %s\n",
            violation.Severity,
            violation.Rule,
            violation.Message)
    }
}

// Check warnings
for _, warning := range result.Warnings {
    fmt.Printf("[WARNING] %s: %s\n",
        warning.Rule,
        warning.Message)
}

// Check security score
fmt.Printf("Security Score: %d/%d\n",
    result.Score,
    result.MaxScore)
```

---

## 9. Best Practices

### 9.1 Command Execution

✅ **DO:**

- Use separate tasks for separate operations
- Use built-in modules instead of shell commands
- Validate all user inputs
- Use absolute paths

❌ **DON'T:**

- Chain multiple commands with `;`, `&&`, `||`
- Use command substitution `$()` or backticks
- Pipe to shell interpreters
- Use relative paths with `..`

### 9.2 File Operations

✅ **DO:**

- Use absolute paths
- Validate file sizes
- Check file permissions
- Use appropriate file modes

❌ **DON'T:**

- Access system files (`/etc/passwd`, `/etc/shadow`)
- Use directory traversal (`../`)
- Create files larger than MaxFileSize
- Modify SSH keys or credentials

### 9.3 User/Group Management

✅ **DO:**

- Create users with UID ≥ 1000
- Create groups with GID ≥ 1000
- Use descriptive user/group names
- Set appropriate shells and home directories

❌ **DON'T:**

- Modify system users/groups
- Create users with UID 0
- Create groups with GID 0
- Use privileged user names

### 9.4 Variable Usage

✅ **DO:**

- Use descriptive variable names
- Validate variable content
- Use appropriate data types
- Document variable purposes

❌ **DON'T:**

- Use path separators in variable names
- Store dangerous commands in variables
- Use variables for code execution
- Store credentials in plain text

---

## 10. Testing Security Validation

### 10.1 Test Examples

```go
func TestSecurityValidation(t *testing.T) {
    config := DefaultSecurityConfig()
    validator := NewSecurityValidator(config)

    // Test dangerous command
    task := &types.Task{
        Name:   "dangerous command",
        Module: "command",
        Args: map[string]interface{}{
            "command": "rm -rf /",
        },
    }

    result := validator.ValidateTask(*task)
    assert.False(t, result.Valid)
    assert.NotEmpty(t, result.Violations)
}
```

### 10.2 Integration Testing

```go
func TestPlaybookValidation(t *testing.T) {
    validator := NewSecurityValidator(DefaultSecurityConfig())

    playbook := loadPlaybook("test.yml")
    result := validator.ValidatePlaybook(playbook)

    if !result.Valid {
        t.Logf("Validation failed with %d violations",
            len(result.Violations))
        for _, v := range result.Violations {
            t.Logf("  - %s: %s", v.Rule, v.Message)
        }
    }
}
```

---

## 11. Migration Guide

### 11.1 Updating Existing Playbooks

If you have existing playbooks that fail validation, here's how to fix them:

#### Before (Blocked)

```yaml
- name: "Setup environment"
  command: "mkdir /app && cd /app && git clone repo.git"
```

#### After (Allowed)

```yaml
- name: "Create directory"
  file:
    path: "/app"
    state: directory

- name: "Clone repository"
  git:
    repo: "repo.git"
    dest: "/app"
```

### 11.2 Command Chaining Migration

#### Before (Blocked)

```yaml
- name: "Complex setup"
  command: "cmd1 && cmd2 || cmd3; cmd4"
```

#### After (Allowed)

```yaml
- name: "Step 1"
  command: "cmd1"

- name: "Step 2"
  command: "cmd2"
  when: step1_succeeded

- name: "Step 3"
  command: "cmd3"
  when: step2_failed

- name: "Step 4"
  command: "cmd4"
```

---

## 12. Troubleshooting

### 12.1 Common Validation Errors

#### Error: "Command contains command substitution pattern"

**Solution:** Remove `$()` or backticks, use variables instead.

#### Error: "Path traversal detected"

**Solution:** Use absolute paths without `..`

#### Error: "Attempting to modify system user"

**Solution:** Use a non-system username (not root, daemon, etc.)

#### Error: "Content size exceeds maximum allowed size"

**Solution:** Reduce file size or increase MaxFileSize in config.

### 12.2 Getting Help

If you encounter validation issues:

1. Check the violation message for specific guidance
2. Review this security guide
3. Check the suggestion field in the violation
4. Consult the main documentation

---

## 13. Security Score Calculation

The security score is calculated based on violations:

- **Critical violation:** -25 points
- **High violation:** -15 points
- **Medium violation:** -10 points
- **Low violation:** -5 points

**Maximum score:** 100 points

**Example:**

- 1 Critical violation: 100 - 25 = 75 points
- 2 High violations: 100 - 30 = 70 points
- 1 Critical + 1 High: 100 - 40 = 60 points

---

## 14. Additional Resources

- **Main Documentation:** `/docs/security.md`
- **API Reference:** `/docs/api.md`
- **Test Examples:** `/internal/security/validator_test.go`
- **Configuration Guide:** `/docs/configuration.md`

---

**Last Updated:** 2025
**Version:** 1.0
**Status:** ✅ Production Ready
