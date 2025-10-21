# File Operations & Docker Test - Quick Reference Card

## 📍 File Location

```
/onigirazu/examples/file-operations-docker-test.yml
```

## ⚡ Quick Commands

### Run Complete Playbook

```bash
onigirazu run file-operations-docker-test.yml
```

### Run Specific Operations

```bash
# Create test file
onigirazu run file-operations-docker-test.yml --tags create

# Create backup
onigirazu run file-operations-docker-test.yml --tags backup

# Replace text (localhost → example.com)
onigirazu run file-operations-docker-test.yml --tags replace

# Show file statistics
onigirazu run file-operations-docker-test.yml --tags stats

# Run Docker tests
onigirazu run file-operations-docker-test.yml --tags docker

# Show all available tags
onigirazu run file-operations-docker-test.yml --list-tags

# Show all available tasks
onigirazu run file-operations-docker-test.yml --list-tasks
```

## 📋 What It Does

| Operation | File | Tags |
|-----------|------|------|
| **Create** | `~/test-config.txt` | `create` `files` |
| **Backup** | `~/test-config.txt.backup` | `backup` `files` |
| **Replace** | localhost → example.com | `replace` `files` |
| **Stats** | File metadata | `stats` `files` |
| **Docker Test** | Run in Alpine container | `docker` `test` |

## 🔄 Workflow Examples

### Configuration Migration (localhost → production)

```bash
# 1. Create test file with localhost references
onigirazu run file-operations-docker-test.yml --tags create

# 2. Backup original
onigirazu run file-operations-docker-test.yml --tags backup

# 3. Replace localhost with example.com
onigirazu run file-operations-docker-test.yml --tags replace

# 4. Verify in Docker
onigirazu run file-operations-docker-test.yml --tags docker
```

### Quick Test & Validation

```bash
# Everything in one command
onigirazu run file-operations-docker-test.yml
```

### File Analysis Only

```bash
# Just get stats
onigirazu run file-operations-docker-test.yml --tags stats
```

## 📂 Created Files

After running, files appear in home directory:

```
~/test-config.txt              # Modified file (localhost → example.com)
~/test-config.txt.backup       # Original file (backup)
```

### View Results

```bash
# Show original (backup)
cat ~/test-config.txt.backup

# Show updated
cat ~/test-config.txt

# Compare
diff ~/test-config.txt.backup ~/test-config.txt

# Cleanup
rm ~/test-config.txt ~/test-config.txt.backup
```

## 🐳 Docker Testing Details

**Image:** Alpine Linux (`alpine:latest`)
**Mount:** Read-only file mount
**Tests:**

- ✓ System information
- ✓ File content verification
- ✓ Pattern matching (grep for "example.com")
- ✓ File statistics in container

## 🔧 Customization

Edit these variables in the playbook:

```yaml
vars:
  test_file: "{{ ansible_env.HOME }}/test-config.txt"
  backup_file: "{{ ansible_env.HOME }}/test-config.txt.backup"
  docker_image: "alpine:latest"  # Change to ubuntu:latest, centos:8, etc.
```

## 🐛 Troubleshooting

| Issue | Solution |
|-------|----------|
| Docker not found | `brew install docker` (macOS) or `sudo apt-get install docker.io` (Ubuntu) |
| Permission denied | `onigirazu run file-operations-docker-test.yml --become` |
| File exists error | `rm ~/test-config.txt ~/test-config.txt.backup` |
| Docker image pull fails | Check internet connection: `docker pull alpine:latest` |
| Permission denied (Docker) | Add to group: `sudo usermod -aG docker $USER` |

## 📊 Output Preview

```
✓ File created at /home/user/test-config.txt
✓ Backup created at /home/user/test-config.txt.backup
✓ File content updated:
  - localhost → example.com (3 replacements)
  - development → production
  - Initial deployment → Updated production deployment

=== ORIGINAL FILE STATS ===
File: /home/user/test-config.txt
Size: 162 bytes
Mode: 0644
Owner: user (UID: 1000)

✓ Docker Container Test Results:
  - Alpine Linux system verified
  - File mounted successfully
  - Found 3 example.com entries ✓
```

## 📚 Full Documentation

For detailed information, see:

```
/onigirazu/examples/FILE_OPERATIONS_DOCKER_TEST_GUIDE.md
```

## ✅ Requirements

- ✓ Ansible (any recent version)
- ✓ Docker (installed and running)
- ✓ Write access to home directory
- ✓ Docker daemon access

## 🎯 Task Tags Summary

```
create   → File creation
backup   → Backup operations
replace  → Text replacements
stats    → File statistics
docker   → Docker container tests
test     → All docker tests
files    → All file operations
cleanup  → Cleanup (default: disabled)
always   → Summary (always displayed)
never    → Disabled by default
```

## 💡 Pro Tips

✅ **Preview tasks before running:**

```bash
onigirazu run file-operations-docker-test.yml --list-tasks
```

✅ **Run with verbosity for details:**

```bash
onigirazu run file-operations-docker-test.yml -vv
```

✅ **Combine tags for specific workflow:**

```bash
onigirazu run file-operations-docker-test.yml --tags create,replace,docker
```

✅ **Use block tagging for entire groups:**

- All file operations: `--tags files`
- All tests: `--tags docker test`
- Everything: just run without tags

## 🚀 First Time Setup

```bash
# 1. Verify Docker is running
docker ps

# 2. Pull the image (optional - done automatically)
docker pull alpine:latest

# 3. Run the playbook
cd /Users/denys.rastiegaiev/work/onigirazu_project/onigirazu
onigirazu run examples/file-operations-docker-test.yml

# 4. Check results
cat ~/test-config.txt
cat ~/test-config.txt.backup
```

## 📖 Example Playbook Sections

The playbook is organized into task groups:

1. **File Creation** - Creates test config file
2. **File Backup** - Creates backup copy
3. **File Content Replacement** - Updates text patterns
4. **File Statistics** - Retrieves detailed stats
5. **Docker Testing** - Validates in container
6. **Summary** - Displays results
7. **Cleanup** (optional) - Removes files

## 🔗 Related Playbooks

This example demonstrates:

- File creation and manipulation
- Text replacement patterns
- File statistics and metadata
- Docker integration with Ansible
- Container-based testing
- Backup and recovery workflows

Perfect for learning:

- Configuration management
- Infrastructure testing
- Container validation
- Automation workflows

---

**Last Updated:** 2025-01-29
**Version:** 1.0
**Compatible with:** Onigirazu v1.48.0+
