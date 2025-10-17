package state

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNewSQLiteBackend(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, err := NewSQLiteBackend(cfg)
	if err != nil {
		t.Fatalf("failed to create sqlite backend: %v", err)
	}

	if backend == nil {
		t.Fatal("expected non-nil backend")
	}

	defer backend.DeleteState(context.Background())
}

func TestNewSQLiteBackend_NilConfig(t *testing.T) {
	backend, err := NewSQLiteBackend(nil)
	if err != nil {
		t.Fatalf("failed to create sqlite backend with nil config: %v", err)
	}

	if backend == nil {
		t.Fatal("expected non-nil backend")
	}

	defer backend.DeleteState(context.Background())
}

func TestNewSQLiteBackend_CreateDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, err := NewSQLiteBackend(cfg)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("database file not created: %v", err)
	}

	defer backend.DeleteState(context.Background())
}

func TestSQLiteBackendLoadState_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	state, err := backend.LoadState(context.Background())
	if err != nil {
		t.Errorf("failed to load empty state: %v", err)
	}

	if state == nil {
		t.Fatal("expected non-nil state")
	}

	if len(state.Variables) != 0 {
		t.Error("expected empty variables")
	}
}

func TestSQLiteBackendSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	// Create test state
	testState := &types.State{
		Variables: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
		Checksums: map[string]string{
			"file1": "abc123",
		},
	}

	// Save state
	err := backend.SaveState(context.Background(), testState)
	if err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Load state
	loadedState, err := backend.LoadState(context.Background())
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	if loadedState == nil {
		t.Fatal("expected non-nil state")
	}

	if val, ok := loadedState.Variables["key1"]; !ok || val != "value1" {
		t.Errorf("expected key1='value1', got %v", val)
	}

	if val, ok := loadedState.Variables["key2"]; !ok || val != float64(42) {
		t.Errorf("expected key2=42, got %v", val)
	}
}

func TestSQLiteBackendDeleteState(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)

	// Save a state
	testState := &types.State{
		Variables: map[string]interface{}{"key": "value"},
		Checksums: map[string]string{},
	}
	backend.SaveState(context.Background(), testState)

	// Verify state exists
	loaded, _ := backend.LoadState(context.Background())
	if loaded == nil || len(loaded.Variables) == 0 {
		t.Fatal("state was not saved")
	}

	// Delete
	err := backend.DeleteState(context.Background())
	if err != nil {
		t.Errorf("failed to delete state: %v", err)
	}
}

func TestSQLiteBackendGetPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	path := backend.GetPath()
	if path != dbPath {
		t.Errorf("expected path %s, got %s", dbPath, path)
	}
}

func TestSQLiteBackendGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	// Save some state to have data
	testState := &types.State{
		Variables: map[string]interface{}{"key": "value"},
		Checksums: map[string]string{},
	}
	backend.SaveState(context.Background(), testState)

	stats := backend.GetStats()

	if stats == nil {
		t.Fatal("expected non-nil stats")
	}

	if _, ok := stats["database_size"]; !ok {
		t.Error("expected 'database_size' in stats")
	}

	if _, ok := stats["record_count"]; !ok {
		t.Error("expected 'record_count' in stats")
	}
}

func TestSQLiteBackendMigrate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	// Migration happens in NewSQLiteBackend, but we can test it runs without error
	err := backend.Migrate(context.Background())
	if err != nil {
		t.Errorf("migrate failed: %v", err)
	}
}

func TestSQLiteBackendMultipleSaves(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	// Save multiple states
	for i := 1; i <= 3; i++ {
		state := &types.State{
			Variables: map[string]interface{}{"version": i},
			Checksums: map[string]string{},
		}
		err := backend.SaveState(context.Background(), state)
		if err != nil {
			t.Errorf("failed to save state %d: %v", i, err)
		}
	}

	// Load latest
	loaded, err := backend.LoadState(context.Background())
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	if val, ok := loaded.Variables["version"]; !ok || val != float64(3) {
		t.Errorf("expected latest version to be 3, got %v", val)
	}
}

func TestSQLiteBackendContextCancellation_LoadState(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := backend.LoadState(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestSQLiteBackendContextCancellation_SaveState(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	testState := &types.State{
		Variables: map[string]interface{}{},
		Checksums: map[string]string{},
	}

	err := backend.SaveState(ctx, testState)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestSQLiteBackendContextCancellation_Migrate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := backend.Migrate(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestSQLiteBackendContextCancellation_DeleteState(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := backend.DeleteState(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestSQLiteBackendNilMaps(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	// Save state with nil maps
	testState := &types.State{
		Variables: nil,
		Checksums: nil,
	}

	err := backend.SaveState(context.Background(), testState)
	if err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Load and verify maps are initialized
	loaded, err := backend.LoadState(context.Background())
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	if loaded.Variables == nil {
		t.Error("expected variables map to be initialized")
	}

	if loaded.Checksums == nil {
		t.Error("expected checksums map to be initialized")
	}
}

func TestSQLiteBackendDatabaseSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	// Open connection to verify schema
	db, err := sql.Open("sqlite3", "file:"+dbPath+"?cache=shared&mode=rwc")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Query tables
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
	}

	// Check states table exists
	found := false
	for _, table := range tables {
		if table == "states" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'states' table to exist")
	}
}

func TestSQLiteBackendTimestampHandling(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")

	cfg := &SQLiteConfig{
		Database:       dbPath,
		AutoVacuum:     true,
		JournalMode:    "wal",
		BusyTimeout:    5000,
		MaxConnections: 5,
	}

	backend, _ := NewSQLiteBackend(cfg)
	defer backend.DeleteState(context.Background())

	testState := &types.State{
		Variables: map[string]interface{}{"test": true},
		Checksums: map[string]string{},
	}

	beforeSave := time.Now()
	backend.SaveState(context.Background(), testState)
	afterSave := time.Now()

	loaded, _ := backend.LoadState(context.Background())

	// LastRun should be close to when we saved
	if loaded.LastRun.Before(beforeSave) || loaded.LastRun.After(afterSave.Add(100*time.Millisecond)) {
		t.Errorf("unexpected LastRun timestamp: %v", loaded.LastRun)
	}
}
