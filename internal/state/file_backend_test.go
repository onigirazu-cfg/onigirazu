package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNewFileBackend(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend, err := NewFileBackend(cfg, "")
	if err != nil {
		t.Fatalf("failed to create file backend: %v", err)
	}

	if backend == nil {
		t.Fatal("expected non-nil backend")
	}

	// Check that directory was created
	if _, err := os.Stat(tmpDir); err != nil {
		t.Errorf("state directory was not created: %v", err)
	}
}

func TestNewFileBackend_NilConfig(t *testing.T) {
	backend, err := NewFileBackend(nil, "")
	if err != nil {
		t.Fatalf("failed to create file backend with nil config: %v", err)
	}

	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
}

func TestNewFileBackend_CustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "custom-state.json")
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend, err := NewFileBackend(cfg, stateFile)
	if err != nil {
		t.Fatalf("failed to create file backend: %v", err)
	}

	if backend.GetPath() != stateFile {
		t.Errorf("expected path %s, got %s", stateFile, backend.GetPath())
	}
}

func TestFileBackendLoadState_NotExist(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend, _ := NewFileBackend(cfg, filepath.Join(tmpDir, "nonexistent.json"))

	state, err := backend.LoadState(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if state == nil {
		t.Fatal("expected non-nil state")
	}

	if len(state.Variables) != 0 {
		t.Error("expected empty variables map")
	}
}

func TestFileBackendSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend, _ := NewFileBackend(cfg, stateFile)

	// Create test state
	testState := &types.State{
		Variables: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
		Checksums: map[string]string{
			"file1": "abc123",
		},
		LastRun: time.Now(),
	}

	// Save state
	err := backend.SaveState(context.Background(), testState)
	if err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(stateFile); err != nil {
		t.Errorf("state file was not created: %v", err)
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

func TestFileBackendDeleteState(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend, _ := NewFileBackend(cfg, stateFile)

	// Save a state
	testState := &types.State{
		Variables: map[string]interface{}{"key": "value"},
		Checksums: map[string]string{},
	}
	backend.SaveState(context.Background(), testState)

	// Verify file exists
	if _, err := os.Stat(stateFile); err != nil {
		t.Fatal("state file was not created")
	}

	// Delete state
	err := backend.DeleteState(context.Background())
	if err != nil {
		t.Errorf("failed to delete state: %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Error("state file was not deleted")
	}
}

func TestFileBackendDeleteState_NotExist(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend, _ := NewFileBackend(cfg, filepath.Join(tmpDir, "nonexistent.json"))

	// Should not error on deleting non-existent file
	err := backend.DeleteState(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestFileBackendGetPath(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend, _ := NewFileBackend(cfg, stateFile)

	if backend.GetPath() != stateFile {
		t.Errorf("expected path %s, got %s", stateFile, backend.GetPath())
	}
}

func TestFileBackendGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	cfg := &FileConfig{
		Directory:    tmpDir,
		BackupCount:  5,
		Compression:  true,
		RotationSize: 1024,
	}

	backend, _ := NewFileBackend(cfg, stateFile)

	// Save a state to get file size
	testState := &types.State{
		Variables: map[string]interface{}{},
		Checksums: map[string]string{},
	}
	backend.SaveState(context.Background(), testState)

	stats := backend.GetStats()

	if stats == nil {
		t.Fatal("expected non-nil stats")
	}

	if _, ok := stats["file_size"]; !ok {
		t.Error("expected file_size in stats")
	}

	if _, ok := stats["last_modified"]; !ok {
		t.Error("expected last_modified in stats")
	}

	if _, ok := stats["backup_dir"]; !ok {
		t.Error("expected backup_dir in stats")
	}

	if comp, ok := stats["compression_enabled"].(bool); !ok || !comp {
		t.Error("expected compression_enabled=true")
	}

	if bc, ok := stats["backup_count"].(int); !ok || bc != 5 {
		t.Error("expected backup_count=5")
	}
}

func TestFileBackendMigrate(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "states")
	cfg := &FileConfig{
		Directory:   stateDir,
		BackupCount: 5,
	}

	backend, _ := NewFileBackend(cfg, filepath.Join(stateDir, "state.json"))

	// Directory should already be created by NewFileBackend
	// Try migrate to ensure it works
	err := backend.Migrate(context.Background())
	if err != nil {
		t.Errorf("failed to migrate: %v", err)
	}

	// Check directories exist
	if _, err := os.Stat(stateDir); err != nil {
		t.Errorf("state directory does not exist: %v", err)
	}
}

func TestFileBackendContextCancellation_LoadState(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend, _ := NewFileBackend(cfg, filepath.Join(tmpDir, "state.json"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := backend.LoadState(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestFileBackendContextCancellation_SaveState(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend, _ := NewFileBackend(cfg, filepath.Join(tmpDir, "state.json"))

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

func TestFileBackendContextCancellation_DeleteState(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend, _ := NewFileBackend(cfg, filepath.Join(tmpDir, "state.json"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := backend.DeleteState(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestFileBackendBackupCreation(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	backupDir := filepath.Join(tmpDir, "backups")
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend, _ := NewFileBackend(cfg, stateFile)

	// Save initial state
	state1 := &types.State{
		Variables: map[string]interface{}{"version": 1},
		Checksums: map[string]string{},
	}
	backend.SaveState(context.Background(), state1)

	// Save another state (should create backup)
	state2 := &types.State{
		Variables: map[string]interface{}{"version": 2},
		Checksums: map[string]string{},
	}
	backend.SaveState(context.Background(), state2)

	// Check if backups were created
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Errorf("failed to read backup dir: %v", err)
	}

	if len(entries) == 0 {
		t.Error("expected backups to be created")
	}
}

func TestFileBackendNilMaps(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend, _ := NewFileBackend(cfg, stateFile)

	// Create state with nil maps
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

func TestFileBackendDefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	// Create backend without specifying state file
	backend, err := NewFileBackend(cfg, "")
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, ".onigirazu-state")
	if backend.GetPath() != expectedPath {
		t.Errorf("expected default path %s, got %s", expectedPath, backend.GetPath())
	}
}
