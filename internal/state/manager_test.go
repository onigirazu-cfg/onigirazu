package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNew(t *testing.T) {
	stateFile := "/tmp/test_state.json"
	manager := New(stateFile)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}
	if manager.stateFile != stateFile {
		t.Errorf("Expected stateFile '%s', got '%s'", stateFile, manager.stateFile)
	}
}

func TestLoadState_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "nonexistent.json")
	manager := New(stateFile)

	state, err := manager.LoadState()
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

func TestLoadState_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create a valid state file
	testState := types.State{
		Variables: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
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

	manager := New(stateFile)
	state, err := manager.LoadState()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if state.Variables["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got '%v'", state.Variables["key1"])
	}
	if state.Checksums["file1.txt"] != "abc123" {
		t.Errorf("Expected checksum='abc123', got '%s'", state.Checksums["file1.txt"])
	}
}

func TestLoadState_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	if err := os.WriteFile(stateFile, []byte("invalid json {"), 0600); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	manager := New(stateFile)
	_, err := manager.LoadState()
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestSaveState(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "subdir", "state.json")
	manager := New(stateFile)

	testState := &types.State{
		Variables: map[string]interface{}{
			"test_key": "test_value",
		},
		Checksums: map[string]string{
			"test_file": "checksum123",
		},
		LastRun: time.Now(),
	}

	err := manager.SaveState(testState)
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

func TestSaveState_InvalidPath(t *testing.T) {
	// Use an invalid path (e.g., trying to create a file in a non-writable location)
	stateFile := "/invalid/path/that/does/not/exist/state.json"
	manager := New(stateFile)

	testState := &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
	}

	err := manager.SaveState(testState)
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

func TestUpdateState(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	manager := New(stateFile)

	state := &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
	}

	endTime := time.Now()
	results := []types.PlayResult{
		{
			PlayName:  "test_play",
			Success:   true,
			StartTime: endTime.Add(-5 * time.Minute),
			EndTime:   endTime,
		},
	}

	manager.UpdateState(state, results)

	if len(state.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(state.Results))
	}
	if state.Results[0].PlayName != "test_play" {
		t.Errorf("Expected PlayName='test_play', got '%s'", state.Results[0].PlayName)
	}
	if !state.LastRun.Equal(endTime) {
		t.Errorf("Expected LastRun=%v, got %v", endTime, state.LastRun)
	}
}

func TestUpdateState_EmptyResults(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	manager := New(stateFile)

	state := &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
	}

	beforeUpdate := time.Now()
	manager.UpdateState(state, []types.PlayResult{})
	afterUpdate := time.Now()

	if len(state.Results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(state.Results))
	}
	// LastRun should be set to current time
	if state.LastRun.Before(beforeUpdate) || state.LastRun.After(afterUpdate) {
		t.Errorf("Expected LastRun to be between %v and %v, got %v", beforeUpdate, afterUpdate, state.LastRun)
	}
}

func TestCalculateChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "test content for checksum"

	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := New(filepath.Join(tmpDir, "state.json"))
	checksum, err := manager.CalculateChecksum(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if checksum == "" {
		t.Error("Expected non-empty checksum")
	}

	// Verify checksum is consistent
	checksum2, err := manager.CalculateChecksum(testFile)
	if err != nil {
		t.Fatalf("Expected no error on second calculation, got: %v", err)
	}
	if checksum != checksum2 {
		t.Errorf("Expected consistent checksum, got '%s' and '%s'", checksum, checksum2)
	}

	// Verify checksum changes when content changes
	if err := os.WriteFile(testFile, []byte("different content"), 0600); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}
	checksum3, err := manager.CalculateChecksum(testFile)
	if err != nil {
		t.Fatalf("Expected no error after content change, got: %v", err)
	}
	if checksum == checksum3 {
		t.Error("Expected checksum to change when content changes")
	}
}

func TestCalculateChecksum_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	manager := New(filepath.Join(tmpDir, "state.json"))

	_, err := manager.CalculateChecksum(filepath.Join(tmpDir, "nonexistent.txt"))
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestHasFileChanged_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := New(filepath.Join(tmpDir, "state.json"))
	state := &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
	}

	changed, err := manager.HasFileChanged(testFile, state)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !changed {
		t.Error("Expected new file to be marked as changed")
	}
	if state.Checksums[testFile] == "" {
		t.Error("Expected checksum to be stored in state")
	}
}

func TestHasFileChanged_UnchangedFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := New(filepath.Join(tmpDir, "state.json"))
	state := &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
	}

	// First check - should be changed (new file)
	changed, err := manager.HasFileChanged(testFile, state)
	if err != nil {
		t.Fatalf("Expected no error on first check, got: %v", err)
	}
	if !changed {
		t.Error("Expected new file to be marked as changed")
	}

	// Second check - should not be changed
	changed, err = manager.HasFileChanged(testFile, state)
	if err != nil {
		t.Fatalf("Expected no error on second check, got: %v", err)
	}
	if changed {
		t.Error("Expected unchanged file to not be marked as changed")
	}
}

func TestHasFileChanged_ModifiedFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("original content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := New(filepath.Join(tmpDir, "state.json"))
	state := &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
	}

	// First check
	_, err := manager.HasFileChanged(testFile, state)
	if err != nil {
		t.Fatalf("Expected no error on first check, got: %v", err)
	}

	// Modify file
	if err := os.WriteFile(testFile, []byte("modified content"), 0600); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Second check - should be changed
	changed, err := manager.HasFileChanged(testFile, state)
	if err != nil {
		t.Fatalf("Expected no error on second check, got: %v", err)
	}
	if !changed {
		t.Error("Expected modified file to be marked as changed")
	}
}

func TestGetLastResults(t *testing.T) {
	tmpDir := t.TempDir()
	manager := New(filepath.Join(tmpDir, "state.json"))

	results := []types.PlayResult{
		{PlayName: "play1", Success: true},
		{PlayName: "play2", Success: false},
	}

	state := &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
		Results:   results,
	}

	lastResults := manager.GetLastResults(state)
	if len(lastResults) != 2 {
		t.Errorf("Expected 2 results, got %d", len(lastResults))
	}
	if lastResults[0].PlayName != "play1" {
		t.Errorf("Expected first result PlayName='play1', got '%s'", lastResults[0].PlayName)
	}
}

func TestCleanupOldResults(t *testing.T) {
	tmpDir := t.TempDir()
	manager := New(filepath.Join(tmpDir, "state.json"))

	now := time.Now()
	oldTime := now.AddDate(0, 0, -31)    // 31 days ago
	recentTime := now.AddDate(0, 0, -15) // 15 days ago

	state := &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
		Results: []types.PlayResult{
			{PlayName: "old_play", EndTime: oldTime},
			{PlayName: "recent_play", EndTime: recentTime},
			{PlayName: "current_play", EndTime: now},
		},
	}

	manager.CleanupOldResults(state)

	if len(state.Results) != 2 {
		t.Errorf("Expected 2 results after cleanup, got %d", len(state.Results))
	}

	// Verify old result was removed
	for _, result := range state.Results {
		if result.PlayName == "old_play" {
			t.Error("Expected old_play to be removed")
		}
	}

	// Verify recent results remain
	foundRecent := false
	foundCurrent := false
	for _, result := range state.Results {
		if result.PlayName == "recent_play" {
			foundRecent = true
		}
		if result.PlayName == "current_play" {
			foundCurrent = true
		}
	}
	if !foundRecent {
		t.Error("Expected recent_play to remain")
	}
	if !foundCurrent {
		t.Error("Expected current_play to remain")
	}
}

func TestCleanupOldResults_EmptyResults(t *testing.T) {
	tmpDir := t.TempDir()
	manager := New(filepath.Join(tmpDir, "state.json"))

	state := &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
		Results:   []types.PlayResult{},
	}

	// Should not panic
	manager.CleanupOldResults(state)

	if len(state.Results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(state.Results))
	}
}

func TestCleanupOldResults_AllOld(t *testing.T) {
	tmpDir := t.TempDir()
	manager := New(filepath.Join(tmpDir, "state.json"))

	oldTime := time.Now().AddDate(0, 0, -40) // 40 days ago

	state := &types.State{
		Variables: make(map[string]interface{}),
		Checksums: make(map[string]string),
		Results: []types.PlayResult{
			{PlayName: "old_play1", EndTime: oldTime},
			{PlayName: "old_play2", EndTime: oldTime.Add(time.Hour)},
		},
	}

	manager.CleanupOldResults(state)

	if len(state.Results) != 0 {
		t.Errorf("Expected all results to be removed, got %d remaining", len(state.Results))
	}
}

func TestRoundTrip_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	manager := New(stateFile)

	originalState := &types.State{
		Variables: map[string]interface{}{
			"var1": "value1",
			"var2": 123,
			"var3": true,
		},
		Checksums: map[string]string{
			"file1": "checksum1",
			"file2": "checksum2",
		},
		Results: []types.PlayResult{
			{PlayName: "play1", Success: true},
		},
		LastRun: time.Now().Truncate(time.Second), // Truncate to avoid precision issues
	}

	// Save
	if err := manager.SaveState(originalState); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Load
	loadedState, err := manager.LoadState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Compare
	if loadedState.Variables["var1"] != "value1" {
		t.Errorf("Expected var1='value1', got '%v'", loadedState.Variables["var1"])
	}
	if loadedState.Checksums["file1"] != "checksum1" {
		t.Errorf("Expected checksum1, got '%s'", loadedState.Checksums["file1"])
	}
	if len(loadedState.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(loadedState.Results))
	}
	if !loadedState.LastRun.Equal(originalState.LastRun) {
		t.Errorf("Expected LastRun=%v, got %v", originalState.LastRun, loadedState.LastRun)
	}
}
