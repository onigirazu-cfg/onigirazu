package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestFileBackend_BackupCleanup_SortsOldestFirst(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	os.MkdirAll(backupDir, 0750)

	// Create mock backup files with different modification times
	oldBackup := filepath.Join(backupDir, "state.2024-01-01-100000")
	midBackup := filepath.Join(backupDir, "state.2024-01-02-100000")
	newBackup := filepath.Join(backupDir, "state.2024-01-03-100000")

	// Create files
	createTestFile(oldBackup)
	createTestFile(midBackup)
	createTestFile(newBackup)

	// Set modification times (oldest first)
	baseTime := time.Now().Add(-time.Hour * 72)
	os.Chtimes(oldBackup, baseTime, baseTime)
	os.Chtimes(midBackup, baseTime.Add(time.Hour*24), baseTime.Add(time.Hour*24))
	os.Chtimes(newBackup, baseTime.Add(time.Hour*48), baseTime.Add(time.Hour*48))

	// Create backend with BackupCount=2 (should keep 2, delete 1 oldest)
	config := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 2,
	}

	backend := &FileBackend{
		config:    config,
		stateFile: filepath.Join(tmpDir, ".onigirazu-state"),
		backupDir: backupDir,
	}

	// Run cleanup
	err := backend.cleanupOldBackups()
	if err != nil {
		t.Errorf("cleanupOldBackups failed: %v", err)
	}

	// Verify oldest backup was deleted
	if _, err := os.Stat(oldBackup); !os.IsNotExist(err) {
		t.Errorf("Expected oldest backup to be deleted")
	}

	// Verify newer backups still exist
	if _, err := os.Stat(midBackup); err != nil {
		t.Errorf("Expected mid backup to exist after cleanup: %v", err)
	}

	if _, err := os.Stat(newBackup); err != nil {
		t.Errorf("Expected new backup to exist after cleanup: %v", err)
	}
}

func TestFileBackend_BackupCleanup_DoesNotCleanIfUnderLimit(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	os.MkdirAll(backupDir, 0750)

	// Create 2 backup files
	backup1 := filepath.Join(backupDir, "state.backup.1")
	backup2 := filepath.Join(backupDir, "state.backup.2")

	createTestFile(backup1)
	createTestFile(backup2)

	// Create backend with BackupCount=5 (should not delete anything)
	config := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
	}

	backend := &FileBackend{
		config:    config,
		stateFile: filepath.Join(tmpDir, ".onigirazu-state"),
		backupDir: backupDir,
	}

	// Run cleanup
	err := backend.cleanupOldBackups()
	if err != nil {
		t.Errorf("cleanupOldBackups failed: %v", err)
	}

	// Verify all backups still exist
	if _, err := os.Stat(backup1); err != nil {
		t.Errorf("Expected backup1 to exist: %v", err)
	}

	if _, err := os.Stat(backup2); err != nil {
		t.Errorf("Expected backup2 to exist: %v", err)
	}
}

func TestFileBackend_BackupCleanup_DeletesMultipleWhenNeeded(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	os.MkdirAll(backupDir, 0750)

	// Create 10 backup files
	backups := make([]string, 10)
	for i := 0; i < 10; i++ {
		backups[i] = filepath.Join(backupDir, "state.backup."+string(rune(48+i)))
		createTestFile(backups[i])

		// Set different modification times
		modTime := time.Now().Add(-time.Hour * time.Duration(10-i))
		os.Chtimes(backups[i], modTime, modTime)
	}

	// Create backend with BackupCount=3 (should keep 3, delete 7 oldest)
	config := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 3,
	}

	backend := &FileBackend{
		config:    config,
		stateFile: filepath.Join(tmpDir, ".onigirazu-state"),
		backupDir: backupDir,
	}

	// Run cleanup
	err := backend.cleanupOldBackups()
	if err != nil {
		t.Errorf("cleanupOldBackups failed: %v", err)
	}

	// Count remaining backups
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Errorf("Failed to read backup dir: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 backups remaining, got %d", len(entries))
	}

	// Verify oldest backups were deleted (those with lower indices)
	for i := 0; i < 7; i++ {
		if _, err := os.Stat(backups[i]); !os.IsNotExist(err) {
			t.Errorf("Expected backup %d to be deleted", i)
		}
	}

	// Verify newest backups still exist (those with higher indices)
	for i := 7; i < 10; i++ {
		if _, err := os.Stat(backups[i]); err != nil {
			t.Errorf("Expected backup %d to exist: %v", i, err)
		}
	}
}

func TestFileBackend_BackupCreation(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, ".onigirazu-state")
	backupDir := filepath.Join(tmpDir, "backups")

	// Create initial state file
	testState := []byte(`{"variables": {}, "checksums": {}, "last_run": "2024-01-01T00:00:00Z"}`)
	os.WriteFile(stateFile, testState, 0600)

	os.MkdirAll(backupDir, 0750)

	backend := &FileBackend{
		config: &FileConfig{
			Directory:   tmpDir,
			BackupCount: 5,
		},
		stateFile: stateFile,
		backupDir: backupDir,
	}

	// Create backups
	err := backend.createBackup()
	if err != nil {
		t.Errorf("createBackup failed: %v", err)
	}

	// Verify backup was created
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Errorf("Failed to read backup dir: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 backup, got %d", len(entries))
	}

	// Verify backup content matches original
	backupFile := filepath.Join(backupDir, entries[0].Name())
	backupContent, err := os.ReadFile(backupFile)
	if err != nil {
		t.Errorf("Failed to read backup: %v", err)
	}

	if string(backupContent) != string(testState) {
		t.Errorf("Backup content doesn't match original")
	}
}

func TestFileBackend_SaveState_CreatesBackup(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, ".onigirazu-state")
	backupDir := filepath.Join(tmpDir, "backups")

	os.MkdirAll(backupDir, 0750)

	config := &FileConfig{
		Directory:   tmpDir,
		BackupCount: 5,
		Compression: false,
	}

	backend := &FileBackend{
		config:     config,
		stateFile:  stateFile,
		backupDir:  backupDir,
		compressor: NewCompressionManager(DefaultCompressionConfig()),
	}

	// Create initial state file
	initialState := &types.State{
		Variables: map[string]interface{}{"key": "value1"},
		Checksums: map[string]string{},
		LastRun:   time.Now(),
	}

	err := backend.SaveState(context.Background(), initialState)
	if err != nil {
		t.Errorf("First SaveState failed: %v", err)
	}

	// Verify state file was created
	if _, err := os.Stat(stateFile); err != nil {
		t.Errorf("State file not created: %v", err)
	}

	// Save again with different content
	updatedState := &types.State{
		Variables: map[string]interface{}{"key": "value2"},
		Checksums: map[string]string{},
		LastRun:   time.Now(),
	}

	err = backend.SaveState(context.Background(), updatedState)
	if err != nil {
		t.Errorf("Second SaveState failed: %v", err)
	}

	// Verify backup was created
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Errorf("Failed to read backup dir: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 backup after second save, got %d", len(entries))
	}
}

func TestFileBackend_GetStats_IncludesBackupInfo(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	os.MkdirAll(backupDir, 0750)

	// Create some backups
	for i := 0; i < 3; i++ {
		backupFile := filepath.Join(backupDir, "state.backup."+string(rune(48+i)))
		createTestFile(backupFile)
	}

	backend := &FileBackend{
		config: &FileConfig{
			Directory:   tmpDir,
			BackupCount: 5,
			Compression: false,
		},
		stateFile: filepath.Join(tmpDir, ".onigirazu-state"),
		backupDir: backupDir,
	}

	stats := backend.GetStats()

	if backups, ok := stats["existing_backups"]; ok {
		if backups.(int) != 3 {
			t.Errorf("Expected 3 backups in stats, got %d", backups)
		}
	} else {
		t.Errorf("Expected existing_backups in stats")
	}

	if backupDirStat, ok := stats["backup_dir"]; ok {
		if backupDirStat != backupDir {
			t.Errorf("Expected backup_dir to be %s, got %s", backupDir, backupDirStat)
		}
	} else {
		t.Errorf("Expected backup_dir in stats")
	}
}

// Helper function to create test files
func createTestFile(path string) error {
	return os.WriteFile(path, []byte("test content"), 0600)
}
