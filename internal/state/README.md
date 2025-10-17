# Multi-Backend State Management System

This document describes the multi-backend state management system for Onigirazu.

## Overview

Onigirazu now supports multiple storage backends for state files:

- **File Backend** (default) - JSON files stored locally
- **SQLite Backend** - SQLite database for better querying and history
- **Remote Backend** (planned) - Centralized state management via API

## Architecture

### StateBackend Interface

All backends implement the `StateBackend` interface:

```go
type StateBackend interface {
    LoadState(ctx context.Context) (*types.State, error)
    SaveState(ctx context.Context, state *types.State) error
    DeleteState(ctx context.Context) error
    GetPath() string
    GetStats() map[string]interface{}
    Migrate(ctx context.Context) error
}
```

### Backend Factory

The `BackendFactory` creates backend instances based on configuration:

```go
factory := state.NewBackendFactory(config)
backend, err := factory.CreateBackend(stateFile)
```

## Configuration

### File Backend Configuration

```yaml
state:
  backend: "file"
  file:
    directory: "~/.onigirazu/states"
    compression: false
    backup_count: 5
    rotation_size: 0  # 0 = disabled
```

**Features:**

- Local JSON file storage
- Automatic backups with configurable retention
- Optional gzip compression
- Atomic writes with temporary files
- Fast startup

### SQLite Backend Configuration

```yaml
state:
  backend: "sqlite"
  sqlite:
    database: "~/.onigirazu/state.db"
    auto_vacuum: true
    journal_mode: "wal"  # or "delete"
    busy_timeout: 5000   # milliseconds
    max_connections: 5
    retention_days: 90   # 0 = disabled
```

**Features:**

- Persistent database storage
- Full query history support
- Automatic old state cleanup
- WAL mode for better concurrency
- Connection pooling
- Indexes for fast lookups

**Advantages:**

- Better for large-scale deployments
- State history tracking
- Efficient storage
- Better for team environments

### Remote Backend Configuration (Future)

```yaml
state:
  backend: "remote"
  remote:
    api_url: "https://api.company.com"
    auth_token: "${ONIGIRAZU_API_TOKEN}"
    sync_interval: "5m"
    cache_local: true
```

## Usage Examples

### Using the Default File Backend

```bash
# The default backend is file-based
onigirazu apply playbook.yml
```

### Using SQLite Backend

#### Option 1: Set environment variable

```bash
export ONIGIRAZU_STATE_BACKEND=sqlite
onigirazu apply playbook.yml
```

#### Option 2: Configure in code

```go
config := state.NewDefaultConfig()
config.Backend = state.BackendTypeSQLite
factory := state.NewBackendFactory(config)
backend, err := factory.CreateBackend(stateFile)
```

### Querying State History (SQLite only)

```go
backend, _ := factory.CreateSQLiteBackend()
sqliteBackend := backend.(*state.SQLiteBackend)

// Get last 10 state records
history, _ := sqliteBackend.GetStateHistory(ctx, 10)
for _, record := range history {
    fmt.Printf("ID: %v, Created: %v\n", record["id"], record["created_at"])
}

// Cleanup old states
affected, _ := sqliteBackend.CleanupOldStates(ctx, 30*24*time.Hour)
fmt.Printf("Deleted %d old state records\n", affected)
```

## Migration Guide

### From File to SQLite

1. **Update Configuration:**

   ```yaml
   state:
     backend: "sqlite"
     sqlite:
       database: "~/.onigirazu/state.db"
   ```

2. **Run Migration:**

   ```bash
   onigirazu state migrate --from=file --to=sqlite
   ```

3. **Verify:**

   ```bash
   onigirazu state info
   ```

### From SQLite to File

1. **Export Current State:**

   ```bash
   onigirazu state export --output=state-backup.json
   ```

2. **Update Configuration:**

   ```yaml
   state:
     backend: "file"
     file:
       directory: "~/.onigirazu/states"
   ```

3. **Restart:**

   ```bash
   onigirazu apply playbook.yml
   ```

## State Backend Implementation Details

### File Backend

**Storage Structure:**

```
~/.onigirazu/
├── states/
│   ├── .onigirazu-state          # Current state
│   ├── .onigirazu-state.20250117-225610  # Backup
│   ├── .onigirazu-state.20250117-225643  # Backup
│   └── backups/
│       ├── .onigirazu-state.20250117-225610
│       └── .onigirazu-state.20250117-225643
```

**Backup Rotation:**

- Creates backup before each state save
- Keeps only `backup_count` most recent backups
- Automatic cleanup of old backups

**Atomic Writes:**

- Writes to temporary file first
- Atomic rename to final location
- Prevents corruption on interruption

### SQLite Backend

**Schema:**

```sql
CREATE TABLE states (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    state_data TEXT NOT NULL,
    last_updated DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata JSON
);

CREATE INDEX idx_states_created_at ON states(created_at);
CREATE INDEX idx_states_last_updated ON states(last_updated);
```

**Pragmas:**

- `auto_vacuum = FULL` - Automatic cleanup
- `journal_mode = WAL` - Write-Ahead Logging for concurrency
- `busy_timeout` - Automatic retry on lock

**Connection Pool:**

- Configurable max connections
- Automatic connection reuse
- Timeout handling

## Performance Considerations

### File Backend

- **Pros:** Fast, simple, no external dependencies
- **Cons:** Limited query capabilities, no built-in history
- **Best for:** Single-user, simple deployments

### SQLite Backend

- **Pros:** Full history, queryable, efficient storage
- **Cons:** Slightly more overhead, requires sqlite3 driver
- **Best for:** Team environments, audit requirements, large deployments

## Error Handling

All backends implement graceful error handling:

```go
backend, err := factory.CreateBackend(stateFile)
if err != nil {
    log.Error("Failed to create backend: %v", err)
    // Fallback to default
    backend, _ = factory.CreateFileBackend(stateFile)
}
```

## Testing

Each backend includes comprehensive tests:

```bash
# Test file backend
go test ./internal/state/... -run TestFileBackend

# Test SQLite backend
go test ./internal/state/... -run TestSQLiteBackend

# Test factory
go test ./internal/state/... -run TestBackendFactory
```

## Future Enhancements

1. **Remote Backend:**
   - Centralized state management
   - Team collaboration
   - Audit trails
   - API-based access

2. **State Encryption:**
   - Encrypt sensitive state data
   - Per-backend encryption policies

3. **State Versioning:**
   - Automatic state versioning
   - Rollback to previous states
   - Diff generation

4. **Compression:**
   - Automatic compression for file backend
   - Size-based rotation triggers

5. **Replication:**
   - Multi-region state replication
   - Consistency checking
   - Conflict resolution
