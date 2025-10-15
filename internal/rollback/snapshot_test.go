package rollback

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestSnapshotManager_CreateSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSnapshotManager(tmpDir)

	snapshot, err := sm.CreateSnapshot("playbook-123", "Test snapshot")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	if snapshot.ID == "" {
		t.Error("Snapshot ID should not be empty")
	}

	if snapshot.PlaybookID != "playbook-123" {
		t.Errorf("Expected playbook ID 'playbook-123', got '%s'", snapshot.PlaybookID)
	}

	if snapshot.Description != "Test snapshot" {
		t.Errorf("Expected description 'Test snapshot', got '%s'", snapshot.Description)
	}

	if len(snapshot.Resources) != 0 {
		t.Errorf("Expected 0 resources, got %d", len(snapshot.Resources))
	}
}

func TestSnapshotManager_SaveAndLoadSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSnapshotManager(tmpDir)

	// Create snapshot
	snapshot, err := sm.CreateSnapshot("playbook-123", "Test snapshot")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Add a resource
	resource := ResourceSnapshot{
		Type:       "file",
		Identifier: "/etc/test.conf",
		Host:       "server1",
		State: map[string]interface{}{
			"exists": true,
			"mode":   "0644",
		},
		Action:     "present",
		Module:     "file",
		TaskName:   "Create test file",
		Reversible: true,
	}
	sm.AddResourceSnapshot(snapshot, resource)

	// Save snapshot
	if err := sm.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Load snapshot
	loaded, err := sm.LoadSnapshot(snapshot.ID)
	if err != nil {
		t.Fatalf("Failed to load snapshot: %v", err)
	}

	if loaded.ID != snapshot.ID {
		t.Errorf("Expected ID '%s', got '%s'", snapshot.ID, loaded.ID)
	}

	if len(loaded.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(loaded.Resources))
	}

	if loaded.Resources[0].Type != "file" {
		t.Errorf("Expected resource type 'file', got '%s'", loaded.Resources[0].Type)
	}
}

func TestSnapshotManager_ListSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSnapshotManager(tmpDir)

	// Create multiple snapshots
	for i := 0; i < 3; i++ {
		snapshot, err := sm.CreateSnapshot("playbook-123", "Test snapshot")
		if err != nil {
			t.Fatalf("Failed to create snapshot: %v", err)
		}
		if err := sm.SaveSnapshot(snapshot); err != nil {
			t.Fatalf("Failed to save snapshot: %v", err)
		}
		time.Sleep(1 * time.Millisecond) // Ensure unique IDs
	}

	// List snapshots
	snapshots, err := sm.ListSnapshots()
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) != 3 {
		t.Errorf("Expected 3 snapshots, got %d", len(snapshots))
	}
}

func TestSnapshotManager_DeleteSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSnapshotManager(tmpDir)

	// Create and save snapshot
	snapshot, err := sm.CreateSnapshot("playbook-123", "Test snapshot")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}
	if err := sm.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Delete snapshot
	if err := sm.DeleteSnapshot(snapshot.ID); err != nil {
		t.Fatalf("Failed to delete snapshot: %v", err)
	}

	// Verify deletion
	_, err = sm.LoadSnapshot(snapshot.ID)
	if err == nil {
		t.Error("Expected error when loading deleted snapshot")
	}
}

func TestSnapshotManager_CleanupOldSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSnapshotManager(tmpDir)

	// Create old snapshot
	oldSnapshot, err := sm.CreateSnapshot("playbook-old", "Old snapshot")
	if err != nil {
		t.Fatalf("Failed to create old snapshot: %v", err)
	}
	oldSnapshot.Timestamp = time.Now().Add(-48 * time.Hour)
	if err := sm.SaveSnapshot(oldSnapshot); err != nil {
		t.Fatalf("Failed to save old snapshot: %v", err)
	}

	// Create recent snapshot
	recentSnapshot, err := sm.CreateSnapshot("playbook-recent", "Recent snapshot")
	if err != nil {
		t.Fatalf("Failed to create recent snapshot: %v", err)
	}
	if err := sm.SaveSnapshot(recentSnapshot); err != nil {
		t.Fatalf("Failed to save recent snapshot: %v", err)
	}

	// Cleanup snapshots older than 24 hours
	if err := sm.CleanupOldSnapshots(24 * time.Hour); err != nil {
		t.Fatalf("Failed to cleanup old snapshots: %v", err)
	}

	// Verify old snapshot is deleted
	_, err = sm.LoadSnapshot(oldSnapshot.ID)
	if err == nil {
		t.Error("Expected old snapshot to be deleted")
	}

	// Verify recent snapshot still exists
	_, err = sm.LoadSnapshot(recentSnapshot.ID)
	if err != nil {
		t.Errorf("Recent snapshot should still exist: %v", err)
	}
}

func TestCreateResourceSnapshot(t *testing.T) {
	ctx := context.Background()

	task := &types.Task{
		Name:   "Create test file",
		Module: "file",
		Args: map[string]interface{}{
			"path":  "/etc/test.conf",
			"state": "present",
			"mode":  "0644",
		},
	}

	host := &types.Host{
		Name: "server1",
	}

	currentState := map[string]interface{}{
		"exists": false,
	}

	resource := CreateResourceSnapshot(ctx, task, host, currentState)

	if resource.Type != "file" {
		t.Errorf("Expected type 'file', got '%s'", resource.Type)
	}

	if resource.Identifier != "/etc/test.conf" {
		t.Errorf("Expected identifier '/etc/test.conf', got '%s'", resource.Identifier)
	}

	if resource.Host != "server1" {
		t.Errorf("Expected host 'server1', got '%s'", resource.Host)
	}

	if !resource.Reversible {
		t.Error("File module should be reversible")
	}

	if resource.RollbackOp == nil {
		t.Error("Rollback operation should be generated")
	}
}

func TestGenerateRollbackOperation_File(t *testing.T) {
	tests := []struct {
		name         string
		task         *types.Task
		currentState map[string]interface{}
		expectedArgs map[string]interface{}
	}{
		{
			name: "Create new file - should remove on rollback",
			task: &types.Task{
				Module: "file",
				Args: map[string]interface{}{
					"path":  "/etc/test.conf",
					"state": "present",
				},
			},
			currentState: map[string]interface{}{
				"exists": false,
			},
			expectedArgs: map[string]interface{}{
				"path":  "/etc/test.conf",
				"state": "absent",
			},
		},
		{
			name: "Modify existing file - should restore on rollback",
			task: &types.Task{
				Module: "file",
				Args: map[string]interface{}{
					"path":  "/etc/test.conf",
					"mode":  "0755",
					"state": "present",
				},
			},
			currentState: map[string]interface{}{
				"exists":  true,
				"mode":    "0644",
				"owner":   "root",
				"group":   "root",
				"content": "original content",
			},
			expectedArgs: map[string]interface{}{
				"dest":    "/etc/test.conf",
				"content": "original content",
				"mode":    "0644",
				"owner":   "root",
				"group":   "root",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := generateRollbackOperation(tt.task, tt.currentState)
			if op == nil {
				t.Fatal("Expected rollback operation, got nil")
			}

			for key, expectedValue := range tt.expectedArgs {
				actualValue, ok := op.Args[key]
				if !ok {
					t.Errorf("Expected arg '%s' not found", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Arg '%s': expected '%v', got '%v'", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestGenerateRollbackOperation_Package(t *testing.T) {
	tests := []struct {
		name         string
		task         *types.Task
		currentState map[string]interface{}
		expectedArgs map[string]interface{}
	}{
		{
			name: "Install package - should remove on rollback",
			task: &types.Task{
				Module: "package",
				Args: map[string]interface{}{
					"name":  "nginx",
					"state": "present",
				},
			},
			currentState: map[string]interface{}{},
			expectedArgs: map[string]interface{}{
				"name":  "nginx",
				"state": "absent",
			},
		},
		{
			name: "Remove package - should reinstall on rollback",
			task: &types.Task{
				Module: "package",
				Args: map[string]interface{}{
					"name":  "nginx",
					"state": "absent",
				},
			},
			currentState: map[string]interface{}{
				"version": "1.18.0",
			},
			expectedArgs: map[string]interface{}{
				"name":    "nginx",
				"state":   "present",
				"version": "1.18.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := generateRollbackOperation(tt.task, tt.currentState)
			if op == nil {
				t.Fatal("Expected rollback operation, got nil")
			}

			for key, expectedValue := range tt.expectedArgs {
				actualValue, ok := op.Args[key]
				if !ok {
					t.Errorf("Expected arg '%s' not found", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("Arg '%s': expected '%v', got '%v'", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestIsReversible(t *testing.T) {
	tests := []struct {
		module     string
		reversible bool
	}{
		{"file", true},
		{"copy", true},
		{"template", true},
		{"package", true},
		{"service", true},
		{"user", true},
		{"group", true},
		{"command", false},
		{"shell", false},
		{"debug", false},
	}

	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			result := isReversible(tt.module)
			if result != tt.reversible {
				t.Errorf("Expected %s to be reversible=%v, got %v", tt.module, tt.reversible, result)
			}
		})
	}
}

func TestCalculateRollbackOrder(t *testing.T) {
	tests := []struct {
		module string
		order  int
	}{
		{"service", 100},
		{"systemd", 100},
		{"cron", 90},
		{"file", 80},
		{"package", 60},
		{"user", 50},
		{"group", 40},
		{"unknown", 50},
	}

	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			order := calculateRollbackOrder(tt.module)
			if order != tt.order {
				t.Errorf("Expected order %d for %s, got %d", tt.order, tt.module, order)
			}
		})
	}
}

func TestSnapshotManager_SaveSnapshot_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotDir := filepath.Join(tmpDir, "snapshots", "nested")
	sm := NewSnapshotManager(snapshotDir)

	snapshot, err := sm.CreateSnapshot("playbook-123", "Test")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	if err := sm.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		t.Error("Snapshot directory was not created")
	}
}
