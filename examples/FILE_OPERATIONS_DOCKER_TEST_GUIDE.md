# File Operations & Docker Testing Playbook Guide

## Overview

This playbook demonstrates comprehensive file operations combined with Docker container testing. It includes:

✅ Creating files with test content
✅ Making backups
✅ Replacing text patterns
✅ Retrieving file statistics
✅ Running Docker container tests

**File:** `file-operations-docker-test.yml`

---

## Quick Start

### Run the complete playbook

```bash
onigirazu run file-operations-docker-test.yml
```

### Run specific task groups by tags

```bash
# Create and verify file creation
onigirazu run file-operations-docker-test.yml --tags create

# Create backups
onigirazu run file-operations-docker-test.yml --tags backup

# Replace text in files
onigirazu run file-operations-docker-test.yml --tags replace

# Retrieve file statistics
onigirazu run file-operations-docker-test.yml --tags stats

# Run Docker tests
onigirazu run file-operations-docker-test.yml --tags docker

# List all available tags
onigirazu run file-operations-docker-test.yml --list-tags

# Preview all tasks
onigirazu run file-operations-docker-test.yml --list-tasks
```

---

## What Each Task Group Does

### 1. File Creation (`create` tag)

**Tasks:**

- Creates a test configuration file in home directory (`~/test-config.txt`)
- Populates it with sample configuration content
- Contains references to `localhost` that will be replaced

**Output:**

```
test-config.txt:
  Server: localhost
  Port: 8080
  Environment: development
  Database: localhost:5432
  Cache: localhost:6379
  Message: "Initial deployment"
```

### 2. File Backup (`backup` tag)

**Tasks:**

- Creates a backup copy (`~/test-config.txt.backup`)
- Verifies the backup was created successfully
- Displays backup file statistics

**Use case:** Preserve original configuration before making changes

### 3. Text Replacement (`replace` tag)

**Tasks:**

- Replaces all `localhost` references with `example.com`
- Updates environment from `development` to `production`
- Updates deployment message to reflect changes
- Displays the updated file content

**Before:**

```
Server: localhost
Environment: development
Database: localhost:5432
```

**After:**

```
Server: example.com
Environment: production
Database: example.com:5432
```

### 4. File Statistics (`stats` tag)

**Tasks:**

- Retrieves detailed file statistics for both original and backup files
- Displays:
  - File size
  - Permissions
  - Owner and group
  - Creation, modification, and access times

**Example output:**

```
=== ORIGINAL FILE STATS ===
File: /home/user/test-config.txt
Size: 156 bytes
Mode: 0644
Owner: user (UID: 1000)
Group: group (GID: 1000)
Created: 2025-01-29 14:30:45
Modified: 2025-01-29 14:30:50
```

### 5. Docker Testing (`docker` tag)

**Tasks:**

- Pulls Alpine Linux Docker image
- Runs Docker container to verify system information
- Mounts the test file into the container (read-only)
- Validates file content inside Docker
- Verifies `example.com` replacements were successful
- Displays file statistics from within Docker container

**What gets tested:**

```
✓ Docker image availability
✓ Container execution
✓ File mounting capability
✓ Content verification (grep for "example.com")
✓ File statistics validation
```

---

## Requirements

### System Prerequisites

1. **Ansible**: Must be available in PATH
2. **Docker**: Must be installed and running

   ```bash
   docker --version
   docker run --rm hello-world
   ```

3. **Permissions**:
   - Write access to home directory
   - Docker daemon access (may require sudo or group membership)

### Docker Prerequisites

The playbook uses Alpine Linux (`alpine:latest`). On first run:

```bash
docker pull alpine:latest
```

---

## Example Output

### Successful Run

```
PLAY [File Operations and Docker Testing] **************************

TASK [Create test file in home directory] ***************************
ok: [localhost]

TASK [Confirm file creation] *****************************************
ok: [localhost] => {
    "msg": "Test file created at /home/user/test-config.txt"
}

TASK [Backup test file] **********************************************
ok: [localhost]

TASK [Verify backup was created] *************************************
ok: [localhost]

TASK [Replace localhost with example.com (Server)] *******************
changed: [localhost]

TASK [Display updated file content] **********************************
ok: [localhost] => {
    "msg": "✓ File content updated:\n# Application Configuration\nServer: example.com\nPort: 8080\n..."
}

TASK [Get original file stats] ***************************************
ok: [localhost]

TASK [Display comprehensive file statistics] *************************
ok: [localhost] => {
    "msg": "=== ORIGINAL FILE STATS ===\nFile: /home/user/test-config.txt\nSize: 162 bytes\n..."
}

TASK [Run Docker container - System info] ****************************
ok: [localhost]

TASK [Display Docker test results] ***********************************
ok: [localhost] => {
    "msg": "✓ Docker Container Test Results:\n=== System Information ===\nLinux 5.15.0 #1 SMP x86_64\n..."
}

PLAY RECAP ***************************************************************
localhost : ok=15 changed=3 unreachable=0 failed=0
```

---

## Variable Customization

Edit these variables in the playbook to customize behavior:

```yaml
vars:
  test_file: "{{ ansible_env.HOME }}/test-config.txt"      # Output file location
  backup_file: "{{ ansible_env.HOME }}/test-config.txt.backup"  # Backup location
  docker_image: "alpine:latest"                            # Docker image to use
  test_message: "Initial server configuration at localhost"  # Initial test content
```

### Examples

**Use custom filename:**

```yaml
vars:
  test_file: "{{ ansible_env.HOME }}/my-config.yml"
  backup_file: "{{ ansible_env.HOME }}/my-config.yml.backup"
```

**Use different Docker image:**

```yaml
vars:
  docker_image: "ubuntu:latest"
  # or
  docker_image: "centos:8"
```

---

## Common Use Cases

### 1. Configuration Management Test

```bash
# Create test config and validate it works in Docker
onigirazu run file-operations-docker-test.yml --tags create,docker
```

### 2. Backup and Update Workflow

```bash
# Backup existing config, then update it
onigirazu run file-operations-docker-test.yml --tags backup,replace
```

### 3. File Analysis

```bash
# Get detailed file statistics
onigirazu run file-operations-docker-test.yml --tags stats
```

### 4. Environment Migration

```bash
# Create file with localhost refs, replace with example.com for production
onigirazu run file-operations-docker-test.yml --tags create,backup,replace
```

---

## Troubleshooting

### Docker Not Found

**Error:** `docker: command not found`

**Solution:**

```bash
# Install Docker
# On macOS: brew install docker
# On Ubuntu: sudo apt-get install docker.io
# Then ensure Docker daemon is running
sudo systemctl start docker
```

### Permission Denied (Docker)

**Error:** `permission denied while trying to connect to Docker daemon`

**Solution:**

```bash
# Option 1: Run with sudo
onigirazu run file-operations-docker-test.yml --become

# Option 2: Add user to docker group (requires logout/login)
sudo usermod -aG docker $USER
```

### File Already Exists

**Error:** `File exists: /home/user/test-config.txt`

**Solution:**

```bash
# Remove existing files
rm ~/test-config.txt ~/test-config.txt.backup

# Or modify playbook to overwrite
# Change 'copy' module to include force: yes
```

### Alpine Image Pull Fails

**Error:** `Failed to pull alpine:latest`

**Solution:**

```bash
# Check Docker connectivity
docker pull alpine:latest

# Or use a local image
# Modify docker_image variable to use locally available image
```

---

## File Locations After Execution

| File | Location | Purpose |
|------|----------|---------|
| Original | `~/test-config.txt` | Updated configuration with replacements |
| Backup | `~/test-config.txt.backup` | Original configuration (before replacements) |

### View Results

```bash
# View current file
cat ~/test-config.txt

# View backup
cat ~/test-config.txt.backup

# Compare files
diff ~/test-config.txt.backup ~/test-config.txt

# Remove when done
rm ~/test-config.txt ~/test-config.txt.backup
```

---

## Advanced Features

### Conditional Cleanup

The playbook includes an optional cleanup task (disabled by default):

```yaml
- name: "Optional cleanup (commented out)"
  tags: [cleanup, never]
  block:
    - name: "Remove test file"
      file:
        path: "{{ test_file }}"
        state: absent
      when: false  # Change to true to enable
```

**To enable cleanup:**

```bash
# Modify the playbook:
# 1. Change 'when: false' to 'when: true' in cleanup tasks
# OR use specific tags:
onigirazu run file-operations-docker-test.yml --tags cleanup
```

### Custom Text Replacements

To add more replacements, add additional tasks:

```yaml
- name: "Custom replacement"
  replace:
    path: "{{ test_file }}"
    regexp: 'pattern-to-find'
    replace: 'replacement-text'
```

---

## Task Tags Reference

| Tag | Tasks Included | Use Case |
|-----|---|---|
| `create` | File creation | Initialize test environment |
| `backup` | Backup operations | Preserve original state |
| `replace` | Text replacements | Update configuration |
| `stats` | File statistics | Analyze file properties |
| `docker` | Docker testing | Validate in container |
| `test` | All docker tests | Full validation suite |
| `files` | All file operations | File management workflow |
| `cleanup` | Cleanup operations | Remove test files |
| `always` | Summary display | Always shown |
| `never` | Disabled by default | Requires explicit --tags |

---

## Integration with Other Playbooks

This playbook can be imported into larger automation workflows:

```yaml
---
name: Complex Configuration Management
hosts: all

tasks:
  - name: "Include file operations"
    include: file-operations-docker-test.yml
    tags: [config, docker-validate]
```

---

## Module Reference

**Modules Used:**

1. **`copy`** - Create files with content
2. **`replace`** - Find and replace text patterns
3. **`stat`** - Retrieve file statistics
4. **`command`** - Execute Docker commands
5. **`block`** - Group related tasks
6. **`debug`** - Display information
7. **`file`** - File operations (cleanup)

---

## Performance Notes

- **First run**: May take longer due to Docker image pull (~30-60 seconds)
- **Subsequent runs**: Faster as image is cached (~5-10 seconds)
- **Docker container execution**: Minimal overhead (~1-2 seconds per container)

---

## Best Practices

✅ **Always review file content before replacing text**

```bash
cat ~/test-config.txt
```

✅ **Keep backups before major changes**

```bash
onigirazu run file-operations-docker-test.yml --tags backup
```

✅ **Use specific tags for targeted operations**

```bash
onigirazu run file-operations-docker-test.yml --tags replace --tags docker
```

✅ **Verify Docker is running before testing**

```bash
docker ps
```

---

## Version Information

- **Playbook Version**: 1.0
- **Compatible with**: Onigirazu v1.48.0+
- **Tested on**: Alpine Linux, Ubuntu 20.04+, macOS 11+

---

## Support and Feedback

For issues or improvements:

1. Check the troubleshooting section above
2. Verify Docker is properly installed
3. Ensure file permissions are correct
4. Review Ansible logs with `-v` flag

```bash
# Run with verbose output
onigirazu run file-operations-docker-test.yml -v
onigirazu run file-operations-docker-test.yml -vv  # More verbose
onigirazu run file-operations-docker-test.yml -vvv # Maximum verbosity
```
