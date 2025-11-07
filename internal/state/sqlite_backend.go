package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// SQLiteBackend implements StateBackend interface for SQLite-based storage
type SQLiteBackend struct {
	config *SQLiteConfig
	db     *sql.DB
}

// NewSQLiteBackend creates a new SQLite-based state backend
func NewSQLiteBackend(config *SQLiteConfig) (*SQLiteBackend, error) {
	if config == nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}

		config = &SQLiteConfig{
			Database:       filepath.Join(homeDir, ".onigirazu", "state.db"),
			AutoVacuum:     true,
			JournalMode:    "wal",
			BusyTimeout:    5000,
			MaxConnections: 5,
			RetentionDays:  90,
		}
	}

	// Ensure directory exists
	dbDir := filepath.Dir(config.Database)
	if err := os.MkdirAll(dbDir, 0750); err != nil {
		return nil, fmt.Errorf("cannot create database directory: %w", err)
	}

	// Open database connection
	dsn := fmt.Sprintf("file:%s?cache=shared&mode=rwc&_busy_timeout=%d", config.Database, config.BusyTimeout)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("cannot open sqlite database: %w", err)
	}

	// Configure connection pool
	// Note: For SQLite, max connections should be reasonable but typically 5-10 is sufficient
	db.SetMaxOpenConns(config.MaxConnections)
	// Set idle connections to 50% of max to avoid recreating connections under load
	db.SetMaxIdleConns(config.MaxConnections / 2)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot connect to sqlite database: %w", err)
	}

	backend := &SQLiteBackend{
		config: config,
		db:     db,
	}

	// Run migrations
	if err := backend.Migrate(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return backend, nil
}

// LoadState loads the latest state from database
func (sb *SQLiteBackend) LoadState(ctx context.Context) (*types.State, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	row := sb.db.QueryRowContext(ctx, `
		SELECT state_data, last_updated
		FROM states
		ORDER BY id DESC
		LIMIT 1
	`)

	var stateJSON string
	var lastUpdated time.Time

	err := row.Scan(&stateJSON, &lastUpdated)
	if err == sql.ErrNoRows {
		// No state exists yet
		return &types.State{
			Variables: make(map[string]interface{}),
			Checksums: make(map[string]string),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error loading state: %w", err)
	}

	var state types.State
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, fmt.Errorf("error parsing state: %w", err)
	}

	// Initialize maps if nil
	if state.Variables == nil {
		state.Variables = make(map[string]interface{})
	}
	if state.Checksums == nil {
		state.Checksums = make(map[string]string)
	}

	state.LastRun = lastUpdated

	return &state, nil
}

// SaveState saves state to database
func (sb *SQLiteBackend) SaveState(ctx context.Context, state *types.State) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Marshal state to JSON
	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing state: %w", err)
	}

	now := time.Now()

	// Insert new state record
	_, err = sb.db.ExecContext(ctx, `
		INSERT INTO states (state_data, last_updated, created_at)
		VALUES (?, ?, ?)
	`, string(stateJSON), now, now)

	if err != nil {
		return fmt.Errorf("error saving state: %w", err)
	}

	// Cleanup old records based on retention policy
	if sb.config.RetentionDays > 0 {
		cutoff := now.AddDate(0, 0, -sb.config.RetentionDays)
		_, err := sb.db.ExecContext(ctx, `
			DELETE FROM states
			WHERE created_at < ?
		`, cutoff)

		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to cleanup old states: %v\n", err)
		}
	}

	return nil
}

// DeleteState removes the latest state
func (sb *SQLiteBackend) DeleteState(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	_, err := sb.db.ExecContext(ctx, `
		DELETE FROM states
		WHERE id = (SELECT MAX(id) FROM states)
	`)

	return err
}

// GetPath returns the database path
func (sb *SQLiteBackend) GetPath() string {
	return sb.config.Database
}

// GetStats returns backend statistics
func (sb *SQLiteBackend) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Database file info
	info, err := os.Stat(sb.config.Database)
	if err == nil {
		stats["database_size"] = info.Size()
		stats["last_modified"] = info.ModTime()
	}

	// Record count
	var recordCount int64
	sb.db.QueryRow("SELECT COUNT(*) FROM states").Scan(&recordCount)
	stats["record_count"] = recordCount

	// Configuration
	stats["auto_vacuum"] = sb.config.AutoVacuum
	stats["journal_mode"] = sb.config.JournalMode
	stats["retention_days"] = sb.config.RetentionDays
	stats["max_connections"] = sb.config.MaxConnections

	return stats
}

// Migrate creates necessary tables and indexes
func (sb *SQLiteBackend) Migrate(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Set pragmas
	if sb.config.AutoVacuum {
		if _, err := sb.db.ExecContext(ctx, "PRAGMA auto_vacuum = FULL"); err != nil {
			return fmt.Errorf("error setting auto_vacuum: %w", err)
		}
	}

	if sb.config.JournalMode != "" {
		query := fmt.Sprintf("PRAGMA journal_mode = %s", sb.config.JournalMode)
		if _, err := sb.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("error setting journal_mode: %w", err)
		}
	}

	// Create states table
	createStatesTable := `
		CREATE TABLE IF NOT EXISTS states (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state_data TEXT NOT NULL,
			last_updated DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			metadata JSON
		)
	`

	if _, err := sb.db.ExecContext(ctx, createStatesTable); err != nil {
		return fmt.Errorf("error creating states table: %w", err)
	}

	// Create indexes
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_states_created_at ON states(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_states_last_updated ON states(last_updated)`,
	}

	for _, idx := range indexes {
		if _, err := sb.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("error creating index: %w", err)
		}
	}

	return nil
}

// Close closes the database connection
func (sb *SQLiteBackend) Close() error {
	if sb.db != nil {
		return sb.db.Close()
	}
	return nil
}

// GetStateHistory returns recent state records
func (sb *SQLiteBackend) GetStateHistory(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	rows, err := sb.db.QueryContext(ctx, `
		SELECT id, last_updated, created_at
		FROM states
		ORDER BY id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("error querying state history: %w", err)
	}
	defer rows.Close()

	var history []map[string]interface{}

	for rows.Next() {
		var id int64
		var lastUpdated, createdAt time.Time

		if err := rows.Scan(&id, &lastUpdated, &createdAt); err != nil {
			return nil, err
		}

		history = append(history, map[string]interface{}{
			"id":           id,
			"last_updated": lastUpdated,
			"created_at":   createdAt,
		})
	}

	return history, rows.Err()
}

// CleanupOldStates removes states older than the retention period
func (sb *SQLiteBackend) CleanupOldStates(ctx context.Context, maxAge time.Duration) (int64, error) {
	cutoff := time.Now().Add(-maxAge)

	result, err := sb.db.ExecContext(ctx, `
		DELETE FROM states
		WHERE created_at < ?
	`, cutoff)

	if err != nil {
		return 0, fmt.Errorf("error cleaning up old states: %w", err)
	}

	return result.RowsAffected()
}
