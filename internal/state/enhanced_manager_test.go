package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNewEnhanced(t *testing.T) {
	stateFile := "/tmp/test_enhanced_state.json"
	manager := NewEnhanced(stateFile, true, 5)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}
	if manager.stateFile != stateFile {
		t.Errorf("Expected stateFile '%s', got '%s'", stateFile, manager.stateFile)
	}
	if !manager.autoSave {
		t.Error("Expected autoSave to be true")
	}
	if manager.backupCount != 5 {
		t.Errorf("Expected backupCount=5, got %d", manager.backupCount)
	}
	if manager.taskStates == nil {
		t.Error("Expected taskStates map to be initialized")
	}
}

func TestNewEnhancedManager(t *testing.T) {
	stateFile := "/tmp/test_enhanced_state.json"
	manager := NewEnhancedManager(stateFile, nil)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}
	if !manager.autoSave {
		t.Error("Expected autoSave to be true by default")
	}
	if manager.backupCount != 5 {
		t.Errorf("Expected backupCount=5 by default, got %d", manager.backupCount)
	}
}

func TestEnhancedLoadState_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "nonexistent.json")
	manager := NewEnhanced(stateFile, false, 0)

	ctx := context.Background()
	state, err := manager.LoadState(ctx)
	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got: %v", err)
	}
	if state == nil {
		t.Fatal("Expected state to be initialized")
	}
	if state.Variables == nil {
		t.Error("Expected Variables map to be initialized")
	}
	if state.Checksums == nil {
		t.Error("Expected Checksums map to be initialized")
	}
}

func TestEnhancedLoadState_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create a valid state file
	testState := types.State{
		Variables: map[string]interface{}{
			"key1": "value1",
		},
		Checksums: map[string]string{
			"file1.txt": "abc123",
		},
		LastRun: time.Now(),
	}

	data, err := json.MarshalIndent(testState, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test state: %v", err)
	}

	if err := os.WriteFile(stateFile, data, 0600); err != nil {
		t.Fatalf("Failed to write test state file: %v", err)
	}

	manager := NewEnhanced(stateFile, false, 0)
	ctx := context.Background()
	state, err := manager.LoadState(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if state.Variables["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got '%v'", state.Variables["key1"])
	}
}

func TestEnhancedLoadState_CanceledContext(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	manager := NewEnhanced(stateFile, false, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := manager.LoadState(ctx)
	if err == nil {
		t.Error("Expected error for canceled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestEnhancedLoadState_NilMaps(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create state with nil maps
	testState := types.State{
		Variables: nil,
		Checksums: nil,
		LastRun:   time.Now(),
	}

	data, err := json.MarshalIndent(testState, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test state: %v", err)
	}

	if err := os.WriteFile(stateFile, data, 0600); err != nil {
		t.Fatalf("Failed to write test state file: %v", err)
	}

	manager := NewEnhanced(stateFile, false, 0)
	ctx := context.Background()
	state, err := manager.LoadState(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Maps should be initialized
	if state.Variables == nil {
		t.Error("Expected Variables map to be initialized")
	}
	if state.Checksums == nil {
		t.Error("Expected Checksums map to be initialized")
	}
}

func TestEnhancedSaveState(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "subdir", "state.json")
	manager := NewEnhanced(stateFile, false, 0)

	testState := &types.State{
		Variables: map[string]interface{}{
			"test_key": "test_value",
		},
		Checksums: map[string]string{
			"test_file": "checksum123",
		},
	}

	ctx := context.Background()
	err := manager.SaveState(ctx, testState)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("Expected state file to be created")
	}

	// Verify content
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read saved state file: %v", err)
	}

	var loadedState types.State
	if err := json.Unmarshal(data, &loadedState); err != nil {
		t.Fatalf("Failed to unmarshal saved state: %v", err)
	}

	if loadedState.Variables["test_key"] != "test_value" {
		t.Errorf("Expected test_key='test_value', got '%v'", loadedState.Variables["test_key"])
	}
}

func TestEnhancedSaveState_CanceledContext(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	manager := NewEnhanced(stateFile, false, 0)

	testState := &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := manager.SaveState(ctx, testState)
	if err == nil {
		t.Error("Expected error for canceled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestEnhancedSaveState_WithBackup(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	manager := NewEnhanced(stateFile, false, 3)

	// Create initial state
	initialState := &types.State{
		Variables: map[string]interface{}{"version": 1},
		Checksums: make(map[string]string),
	}

	ctx := context.Background()
	if err := manager.SaveState(ctx, initialState); err != nil {
		t.Fatalf("Failed to save initial state: %v", err)
	}

	// Save again to create backup
	updatedState := &types.State{
		Variables: map[string]interface{}{"version": 2},
		Checksums: make(map[string]string),
	}

	if err := manager.SaveState(ctx, updatedState); err != nil {
		t.Fatalf("Failed to save updated state: %v", err)
	}

	// Check for backup file
	pattern := filepath.Join(tmpDir, "state.json.backup.*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to glob backup files: %v", err)
	}

	if len(matches) == 0 {
		t.Error("Expected backup file to be created")
	}
}

func TestSaveCurrentState(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	manager := NewEnhanced(stateFile, false, 0)

	// Save without loading first (state is nil)
	err := manager.SaveCurrentState()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("Expected state file to be created")
	}
}

func TestHasChanged_NoState(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewEnhanced(filepath.Join(tmpDir, "state.json"), false, 0)

	// No state loaded, should return true
	changed, err := manager.HasChanged(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !changed {
		t.Error("Expected file to be marked as changed when no state is loaded")
	}
}

func TestHasChanged_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	stateFile := filepath.Join(tmpDir, "state.json")
	manager := NewEnhanced(stateFile, false, 0)

	// Load empty state
	ctx := context.Background()
	if _, err := manager.LoadState(ctx); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// New file should be marked as changed
	changed, err := manager.HasChanged(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !changed {
		t.Error("Expected new file to be marked as changed")
	}
}

func TestGetTaskState(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewEnhanced(filepath.Join(tmpDir, "state.json"), false, 0)

	taskState := &types.TaskState{
		TaskID:   "test-task",
		Checksum: "abc123",
		LastRun:  time.Now(),
	}

	manager.SetTaskState("test-task", taskState)

	retrieved, exists := manager.GetTaskState("test-task")
	if !exists {
		t.Error("Expected task state to exist")
	}
	if retrieved.TaskID != "test-task" {
		t.Errorf("Expected TaskID='test-task', got '%s'", retrieved.TaskID)
	}
	if retrieved.Checksum != "abc123" {
		t.Errorf("Expected Checksum='abc123', got '%s'", retrieved.Checksum)
	}
}

func TestGetTaskState_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewEnhanced(filepath.Join(tmpDir, "state.json"), false, 0)

	_, exists := manager.GetTaskState("nonexistent")
	if exists {
		t.Error("Expected task state to not exist")
	}
}

func TestSetTaskState(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewEnhanced(filepath.Join(tmpDir, "state.json"), false, 0)

	taskState := &types.TaskState{
		TaskID:   "test-task",
		Checksum: "abc123",
	}

	manager.SetTaskState("test-task", taskState)

	retrieved, exists := manager.GetTaskState("test-task")
	if !exists {
		t.Error("Expected task state to exist after setting")
	}
	if retrieved.TaskID != "test-task" {
		t.Errorf("Expected TaskID='test-task', got '%s'", retrieved.TaskID)
	}
}

func TestClear(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	manager := NewEnhanced(stateFile, false, 0)

	// Load state with data
	ctx := context.Background()
	state, err := manager.LoadState(ctx)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	state.Variables["key"] = "value"
	state.Checksums["file"] = "checksum"

	// Add task state
	manager.SetTaskState("task1", &types.TaskState{TaskID: "task1"})

	// Clear
	if err := manager.Clear(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify state is cleared
	stats := manager.GetStats()
	if stats.Variables != 0 {
		t.Errorf("Expected 0 variables after clear, got %d", stats.Variables)
	}
	if stats.Checksums != 0 {
		t.Errorf("Expected 0 checksums after clear, got %d", stats.Checksums)
	}
	if stats.TaskStates != 0 {
		t.Errorf("Expected 0 task states after clear, got %d", stats.TaskStates)
	}
}

func TestUpdateChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	stateFile := filepath.Join(tmpDir, "state.json")
	manager := NewEnhanced(stateFile, false, 0)

	// Load state
	ctx := context.Background()
	if _, err := manager.LoadState(ctx); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Update checksum
	if err := manager.UpdateChecksum(testFile); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify checksum was stored
	stats := manager.GetStats()
	if stats.Checksums != 1 {
		t.Errorf("Expected 1 checksum, got %d", stats.Checksums)
	}
}

func TestUpdateChecksum_NoState(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewEnhanced(filepath.Join(tmpDir, "state.json"), false, 0)

	// Try to update checksum without loading state
	err := manager.UpdateChecksum(testFile)
	if err == nil {
		t.Error("Expected error when state not loaded, got nil")
	}
}

func TestGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	manager := NewEnhanced(stateFile, false, 0)

	// Load state
	ctx := context.Background()
	state, err := manager.LoadState(ctx)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Add some data
	state.Variables["key1"] = "value1"
	state.Variables["key2"] = "value2"
	state.Checksums["file1"] = "checksum1"
	manager.SetTaskState("task1", &types.TaskState{TaskID: "task1"})

	// Save to create file
	if err := manager.SaveState(ctx, state); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Get stats
	stats := manager.GetStats()
	if stats.Variables != 2 {
		t.Errorf("Expected 2 variables, got %d", stats.Variables)
	}
	if stats.Checksums != 1 {
		t.Errorf("Expected 1 checksum, got %d", stats.Checksums)
	}
	if stats.TaskStates != 1 {
		t.Errorf("Expected 1 task state, got %d", stats.TaskStates)
	}
	if stats.StateFileSize == 0 {
		t.Error("Expected non-zero state file size")
	}
}

func TestIsTaskUpToDate(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewEnhanced(filepath.Join(tmpDir, "state.json"), false, 0)

	task := types.Task{
		Name:   "test_task",
		Module: "shell",
		Args:   map[string]interface{}{"cmd": "echo hello"},
	}

	host := types.Host{
		Name: "localhost",
	}

	// Task not in state, should be out of date
	if manager.IsTaskUpToDate(task, host) {
		t.Error("Expected task to be out of date when not in state")
	}

	// Add task state with correct checksum
	taskID := manager.generateTaskID(task, host)
	checksum := manager.calculateTaskChecksum(task, host)
	manager.SetTaskState(taskID, &types.TaskState{
		TaskID:   taskID,
		Checksum: checksum,
	})

	// Now task should be up to date
	if !manager.IsTaskUpToDate(task, host) {
		t.Error("Expected task to be up to date")
	}

	// Change task args
	task.Args = map[string]interface{}{"cmd": "echo goodbye"}

	// Task should be out of date
	if manager.IsTaskUpToDate(task, host) {
		t.Error("Expected task to be out of date after args change")
	}
}

func TestGenerateTaskID(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewEnhanced(filepath.Join(tmpDir, "state.json"), false, 0)

	task := types.Task{
		Name:   "test_task",
		Module: "shell",
	}

	host := types.Host{
		Name: "localhost",
	}

	taskID := manager.generateTaskID(task, host)
	expected := "localhost-shell-test_task"
	if taskID != expected {
		t.Errorf("Expected taskID='%s', got '%s'", expected, taskID)
	}
}

func TestCalculateTaskChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewEnhanced(filepath.Join(tmpDir, "state.json"), false, 0)

	task := types.Task{
		Name:   "test_task",
		Module: "shell",
		Args:   map[string]interface{}{"cmd": "echo hello"},
	}

	host := types.Host{
		Name: "localhost",
	}

	checksum1 := manager.calculateTaskChecksum(task, host)
	if checksum1 == "" {
		t.Error("Expected non-empty checksum")
	}

	// Same task should produce same checksum
	checksum2 := manager.calculateTaskChecksum(task, host)
	if checksum1 != checksum2 {
		t.Error("Expected consistent checksum for same task")
	}

	// Different args should produce different checksum
	task.Args = map[string]interface{}{"cmd": "echo goodbye"}
	checksum3 := manager.calculateTaskChecksum(task, host)
	if checksum1 == checksum3 {
		t.Error("Expected different checksum for different args")
	}
}

func TestRestore(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	manager := NewEnhanced(stateFile, false, 0)

	// Create initial state
	initialState := &types.State{
		Variables: map[string]interface{}{"version": 1},
		Checksums: make(map[string]string),
	}

	ctx := context.Background()
	if err := manager.SaveState(ctx, initialState); err != nil {
		t.Fatalf("Failed to save initial state: %v", err)
	}

	// Create backup manually
	backupFile := filepath.Join(tmpDir, "state.json.backup.test")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}
	if err := os.WriteFile(backupFile, data, 0600); err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify state
	updatedState := &types.State{
		Variables: map[string]interface{}{"version": 2},
		Checksums: make(map[string]string),
	}
	if err := manager.SaveState(ctx, updatedState); err != nil {
		t.Fatalf("Failed to save updated state: %v", err)
	}

	// Restore from backup
	if err := manager.Restore(backupFile); err != nil {
		t.Fatalf("Failed to restore: %v", err)
	}

	// Verify state was restored
	state, err := manager.LoadState(ctx)
	if err != nil {
		t.Fatalf("Failed to load restored state: %v", err)
	}

	version, ok := state.Variables["version"].(float64) // JSON unmarshals numbers as float64
	if !ok {
		t.Fatalf("Expected version to be a number, got %T", state.Variables["version"])
	}
	if int(version) != 1 {
		t.Errorf("Expected version=1 after restore, got %v", version)
	}
}

func TestRestore_NonExistentBackup(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewEnhanced(filepath.Join(tmpDir, "state.json"), false, 0)

	err := manager.Restore(filepath.Join(tmpDir, "nonexistent.backup"))
	if err == nil {
		t.Error("Expected error for non-existent backup, got nil")
	}
}

func TestCleanupOldBackups(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	manager := NewEnhanced(stateFile, false, 2) // Keep only 2 backups

	// Create initial state
	initialState := &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
	}

	ctx := context.Background()
	if err := manager.SaveState(ctx, initialState); err != nil {
		t.Fatalf("Failed to save initial state: %v", err)
	}

	// Create multiple backups by saving multiple times
	for i := 0; i < 4; i++ {
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		state := &types.State{
			Variables: map[string]interface{}{"version": i},
			Checksums: make(map[string]string),
		}
		if err := manager.SaveState(ctx, state); err != nil {
			t.Fatalf("Failed to save state iteration %d: %v", i, err)
		}
	}

	// Check number of backups
	pattern := filepath.Join(tmpDir, "state.json.backup.*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to glob backup files: %v", err)
	}

	if len(matches) > 2 {
		t.Errorf("Expected at most 2 backups, got %d", len(matches))
	}
}

func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	manager := NewEnhanced(stateFile, false, 0)

	// Load initial state
	ctx := context.Background()
	if _, err := manager.LoadState(ctx); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Concurrent reads and writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			taskState := &types.TaskState{
				TaskID:   fmt.Sprintf("task-%d", id),
				Checksum: fmt.Sprintf("checksum-%d", id),
			}
			manager.SetTaskState(taskState.TaskID, taskState)
			_, _ = manager.GetTaskState(taskState.TaskID)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all task states were stored
	stats := manager.GetStats()
	if stats.TaskStates != 10 {
		t.Errorf("Expected 10 task states, got %d", stats.TaskStates)
	}
}
