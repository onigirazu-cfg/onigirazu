# State Backend Configuration Guide

## Quick Start

### Default Configuration (File Backend)

The default configuration uses the file backend, which stores state in JSON files:

```yaml
# ~/.onigirazu/config.yml (auto-created if missing)
state:
  backend: "file"
  file:
    directory: "~/.onigirazu/states"
    compression: false
    backup_count: 5
```

No action needed - just use Onigirazu:

```bash
onigirazu apply playbook.yml
```

### Switch to SQLite Backend

To use SQLite for better history and querying:

#### 1. Update Configuration

Create or edit `~/.onigirazu/config.yml`:

```yaml
state:
  backend: "sqlite"
  sqlite:
    database: "~/.onigirazu/state.db"
    auto_vacuum: true
    journal_mode: "wal"
    busy_timeout: 5000
    max_connections: 5
    retention_days: 90
```

#### 2. First Run

```bash
onigirazu apply playbook.yml
```

The SQLite database will be automatically created and initialized.

#### 3. Verify Setup

```bash
ls -la ~/.onigirazu/state.db
```

You should see the database file.

## Configuration File Locations

Onigirazu looks for configuration in this order:

1. `$ONIGIRAZU_CONFIG_PATH` (environment variable)
2. `.onigirazu/config.yml` (current directory)
3. `~/.onigirazu/config.yml` (home directory)
4. `/etc/onigirazu/config.yml` (system-wide)

## Configuration Reference

### State Section

```yaml
state:
  backend: "file"  # "file", "sqlite", or "remote"

  file:
    # Directory where state files are stored
    directory: "~/.onigirazu/states"

    # Enable gzip compression for state files
    compression: false

    # Number of backup files to keep (0 = no backups)
    backup_count: 5

    # Rotate files after this size in bytes (0 = disabled)
    rotation_size: 0

  sqlite:
    # Path to SQLite database file
    database: "~/.onigirazu/state.db"

    # Enable automatic VACUUM (cleanup)
    auto_vacuum: true

    # Journal mode: "wal" (recommended) or "delete"
    journal_mode: "wal"

    # Timeout for busy database in milliseconds
    busy_timeout: 5000

    # Maximum number of concurrent connections
    max_connections: 5

    # Delete states older than this many days (0 = disabled)
    retention_days: 90

  remote:
    # API endpoint for remote state storage (not yet implemented)
    api_url: "https://api.company.com"

    # Authentication token (can use environment variables)
    auth_token: "${ONIGIRAZU_API_TOKEN}"

    # How often to sync with remote server
    sync_interval: "5m"

    # Cache state locally for offline operation
    cache_local: true
```

## Environment Variables

Control state backend via environment variables:

```bash
# Set backend type
export ONIGIRAZU_STATE_BACKEND=sqlite

# Set state file path
export ONIGIRAZU_STATE_FILE=~/.onigirazu/state.db

# Set SQLite database path
export ONIGIRAZU_SQLITE_DB=~/.onigirazu/state.db

# Set file backend directory
export ONIGIRAZU_STATE_DIR=~/.onigirazu/states

# Set retention (days)
export ONIGIRAZU_STATE_RETENTION=90
```

## Choosing a Backend

### Use File Backend If

✅ Single user or small team
✅ Simple deployments
✅ Want minimal dependencies
✅ Quick startup time
✅ Don't need state history

### Use SQLite Backend If

✅ Multiple users or team
✅ Need state history/queries
✅ Enterprise deployments
✅ Want automatic cleanup
✅ Need better concurrency
✅ Audit requirements

### Use Remote Backend If (Future)

✅ Distributed teams
✅ Centralized state management
✅ Multi-region deployments
✅ Third-party integration

## Common Configurations

### Small Personal Project

```yaml
state:
  backend: "file"
  file:
    directory: "~/.onigirazu/states"
    compression: false
    backup_count: 3  # Keep fewer backups
```

### Team Environment

```yaml
state:
  backend: "sqlite"
  sqlite:
    database: "~/.onigirazu/state.db"
    auto_vacuum: true
    journal_mode: "wal"
    retention_days: 90  # Automatic cleanup
```

### Production with Backups

```yaml
state:
  backend: "sqlite"
  file:
    directory: "/var/lib/onigirazu/states"
    backup_count: 10  # Keep more backups
  sqlite:
    database: "/var/lib/onigirazu/state.db"
    auto_vacuum: true
    journal_mode: "wal"
    max_connections: 10
    retention_days: 365  # Keep 1 year of history
```

### CI/CD Pipeline

```yaml
state:
  backend: "sqlite"
  sqlite:
    database: "/tmp/onigirazu/state.db"  # Ephemeral
    retention_days: 0  # Don't keep history
    busy_timeout: 10000  # Longer timeout for concurrency
```

## Troubleshooting

### SQLite: "database is locked"

**Cause:** Multiple Onigirazu processes accessing the same database

**Solutions:**

1. Increase `busy_timeout`:

   ```yaml
   sqlite:
     busy_timeout: 10000  # 10 seconds
   ```

2. Use file backend for CI/CD:

   ```bash
   export ONIGIRAZU_STATE_BACKEND=file
   ```

3. Use separate databases:

   ```bash
   export ONIGIRAZU_SQLITE_DB=/tmp/onigirazu-${JOB_ID}.db
   ```

### SQLite: "no such table: states"

**Cause:** Database not initialized

**Solution:** Run any command to trigger migration:

```bash
onigirazu state info  # Or any other command
```

### File Backend: "permission denied"

**Cause:** Directory not writable

**Solutions:**

```bash
# Fix permissions
mkdir -p ~/.onigirazu/states
chmod 750 ~/.onigirazu/states
chmod 600 ~/.onigirazu/states/*

# Or use different directory
export ONIGIRAZU_STATE_DIR=/var/tmp/onigirazu
```

### State File Corruption

**File Backend Solution:**

```bash
# Restore from backup
ls ~/.onigirazu/states/backups/
cp ~/.onigirazu/states/backups/.onigirazu-state.20250117-225610 \
   ~/.onigirazu/states/.onigirazu-state
```

**SQLite Backend Solution:**

```bash
# Restore from backup or delete and recreate
rm ~/.onigirazu/state.db
# Next run will recreate
onigirazu apply playbook.yml
```

## Migration Between Backends

### File → SQLite

```bash
# 1. Update config to use SQLite
# 2. First run creates database
onigirazu apply playbook.yml
# 3. Previous file backend data is still there
# 4. New data goes to SQLite
```

### SQLite → File

```bash
# 1. Export state from SQLite
onigirazu state export --output=state-backup.json

# 2. Update config to use file backend
# 3. Next run uses file backend
onigirazu apply playbook.yml
```

## Performance Tuning

### For High Concurrency (Multiple Users)

**SQLite:**

```yaml
sqlite:
  max_connections: 20
  busy_timeout: 15000
  journal_mode: "wal"
```

### For Large State Files

**File Backend:**

```yaml
file:
  compression: true
  rotation_size: 10485760  # 10MB
```

**SQLite:**

```yaml
sqlite:
  auto_vacuum: true
  retention_days: 30  # More aggressive cleanup
```

### For CI/CD Pipelines

**File Backend (Recommended):**

```yaml
file:
  directory: "/tmp/onigirazu/${BUILD_ID}"
  backup_count: 1
```

**SQLite (if needed):**

```yaml
sqlite:
  database: "/tmp/state-${BUILD_ID}.db"
  retention_days: 0
  busy_timeout: 20000
```

## Monitoring & Maintenance

### Check Current Backend

```bash
onigirazu state info
```

### View State History (SQLite only)

```bash
onigirazu state history --limit=20
```

### Cleanup Old States

```bash
# Automatic (if retention_days set)
# Manual cleanup:
onigirazu state cleanup --older-than=90d
```

### Export State

```bash
onigirazu state export --output=backup.json
```

### Verify Backend Health

```bash
# File backend
ls -lah ~/.onigirazu/states/

# SQLite backend
sqlite3 ~/.onigirazu/state.db "SELECT COUNT(*) as records FROM states;"
```

## Best Practices

1. **Regular Backups:**
   - File backend: Enable `backup_count`
   - SQLite backend: Export regularly

2. **Retention Policy:**
   - Set appropriate `retention_days`
   - Archive old states if needed

3. **Monitoring:**
   - Monitor database size (SQLite)
   - Check backup directory size (File)

4. **Access Control:**
   - Protect state files: `chmod 600`
   - Protect directories: `chmod 750`
   - Use separate databases for environments

5. **Documentation:**
   - Document your backend choice
   - Keep config.yml in version control (without secrets)
   - Update team on changes
