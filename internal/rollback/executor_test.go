package rollback

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Info(format string, args ...interface{})                        {}
func (m *mockLogger) Error(format string, args ...interface{})                       {}
func (m *mockLogger) Warn(format string, args ...interface{})                        {}
func (m *mockLogger) Debug(format string, args ...interface{})                       {}
func (m *mockLogger) Fatal(format string, args ...interface{})                       {}
func (m *mockLogger) SetLevel(level string)                                          {}
func (m *mockLogger) TaskStart(name, host string)                                    {}
func (m *mockLogger) TaskEnd(name, host string, changed, success bool)               {}
func (m *mockLogger) PlayStart(name string, num, total int)                          {}
func (m *mockLogger) PlayEnd(name, host string, success bool, dur time.Duration)     {}
func (m *mockLogger) Progress(completed, total int, currentTask, currentHost string) {}
func (m *mockLogger) Retry(taskName, hostName string, attempt, maxAttempts int, delay time.Duration, err error) {
}

// Mock module registry for testing
type mockModuleRegistry struct {
	modules map[string]*mockModule
}

func newMockModuleRegistry() *mockModuleRegistry {
	return &mockModuleRegistry{
		modules: make(map[string]*mockModule),
	}
}

func (m *mockModuleRegistry) RegisterModule(name string, module *mockModule) {
	m.modules[name] = module
}

func (m *mockModuleRegistry) Register(name string, module interfaces.ModuleExecutor) error {
	if mod, ok := module.(*mockModule); ok {
		m.modules[name] = mod
	}
	return nil
}

func (m *mockModuleRegistry) Get(name string) (interfaces.ModuleExecutor, error) {
	if mod, ok := m.modules[name]; ok {
		return mod, nil
	}
	return nil, fmt.Errorf("module not found: %s", name)
}

func (m *mockModuleRegistry) ExecuteTask(ctx context.Context, task *types.Task, host types.Host, variables map[string]interface{}) (types.TaskResult, error) {
	return types.TaskResult{Success: true}, nil
}

func (m *mockModuleRegistry) List() []string {
	names := make([]string, 0, len(m.modules))
	for name := range m.modules {
		names = append(names, name)
	}
	return names
}

func (m *mockModuleRegistry) Unregister(name string) error {
	delete(m.modules, name)
	return nil
}

// Mock module for testing
type mockModule struct {
	name         string
	description  string
	validateFunc func(args map[string]interface{}) error
	executeFunc  func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
}

func (m *mockModule) GetName() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

func (m *mockModule) GetDescription() string {
	if m.description != "" {
		return m.description
	}
	return "Mock module for testing"
}

func (m *mockModule) Validate(args map[string]interface{}) error {
	if m.validateFunc != nil {
		return m.validateFunc(args)
	}
	return nil
}

func (m *mockModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, host, args)
	}
	return types.TaskResult{Success: true}, nil
}

func TestRollbackExecutor_DryRunRollback(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSnapshotManager(tmpDir)
	registry := newMockModuleRegistry()
	logger := &mockLogger{}

	executor := NewRollbackExecutor(sm, registry, logger)

	// Create a snapshot with resources
	snapshot, err := sm.CreateSnapshot("playbook-123", "Test snapshot")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Add resources
	resources := []ResourceSnapshot{
		{
			Type:       "file",
			Identifier: "/etc/test.conf",
			Host:       "server1",
			Module:     "file",
			Reversible: true,
			RollbackOp: &RollbackOperation{
				Module: "file",
				Args:   map[string]interface{}{"path": "/etc/test.conf", "state": "absent"},
				Order:  80,
			},
		},
		{
			Type:       "package",
			Identifier: "nginx",
			Host:       "server1",
			Module:     "package",
			Reversible: true,
			RollbackOp: &RollbackOperation{
				Module: "package",
				Args:   map[string]interface{}{"name": "nginx", "state": "absent"},
				Order:  60,
			},
		},
	}

	for _, r := range resources {
		sm.AddResourceSnapshot(snapshot, r)
	}

	if err := sm.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Perform dry run
	plan, err := executor.DryRunRollback(context.Background(), snapshot.ID)
	if err != nil {
		t.Fatalf("Failed to perform dry run: %v", err)
	}

	if plan.SnapshotID != snapshot.ID {
		t.Errorf("Expected snapshot ID '%s', got '%s'", snapshot.ID, plan.SnapshotID)
	}

	if plan.TotalOperations != 2 {
		t.Errorf("Expected 2 operations, got %d", plan.TotalOperations)
	}

	if plan.ReversibleOperations != 2 {
		t.Errorf("Expected 2 reversible operations, got %d", plan.ReversibleOperations)
	}

	// Verify operations are sorted by order (file should come before package)
	if len(plan.Operations) >= 2 {
		if plan.Operations[0].Module != "file" {
			t.Errorf("Expected first operation to be 'file', got '%s'", plan.Operations[0].Module)
		}
		if plan.Operations[1].Module != "package" {
			t.Errorf("Expected second operation to be 'package', got '%s'", plan.Operations[1].Module)
		}
	}
}

func TestRollbackExecutor_GetSnapshotInfo(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSnapshotManager(tmpDir)
	registry := newMockModuleRegistry()
	logger := &mockLogger{}

	executor := NewRollbackExecutor(sm, registry, logger)

	// Create a snapshot with resources
	snapshot, err := sm.CreateSnapshot("playbook-123", "Test snapshot")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Add resources
	resources := []ResourceSnapshot{
		{
			Type:       "file",
			Identifier: "/etc/test1.conf",
			Host:       "server1",
			Reversible: true,
			RollbackOp: &RollbackOperation{
				Module: "file",
				Args:   map[string]interface{}{"path": "/etc/test1.conf", "state": "absent"},
			},
		},
		{
			Type:       "file",
			Identifier: "/etc/test2.conf",
			Host:       "server2",
			Reversible: true,
			RollbackOp: &RollbackOperation{
				Module: "file",
				Args:   map[string]interface{}{"path": "/etc/test2.conf", "state": "absent"},
			},
		},
		{
			Type:       "package",
			Identifier: "nginx",
			Host:       "server1",
			Reversible: true,
			RollbackOp: &RollbackOperation{
				Module: "package",
				Args:   map[string]interface{}{"name": "nginx", "state": "absent"},
			},
		},
		{
			Type:       "command",
			Identifier: "echo test",
			Host:       "server1",
			Reversible: false,
		},
	}

	for _, r := range resources {
		sm.AddResourceSnapshot(snapshot, r)
	}

	if err := sm.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Get snapshot info
	info, err := executor.GetSnapshotInfo(snapshot.ID)
	if err != nil {
		t.Fatalf("Failed to get snapshot info: %v", err)
	}

	if info.ID != snapshot.ID {
		t.Errorf("Expected ID '%s', got '%s'", snapshot.ID, info.ID)
	}

	if info.TotalResources != 4 {
		t.Errorf("Expected 4 total resources, got %d", info.TotalResources)
	}

	if info.ReversibleCount != 3 {
		t.Errorf("Expected 3 reversible resources, got %d", info.ReversibleCount)
	}

	if info.ResourcesByType["file"] != 2 {
		t.Errorf("Expected 2 file resources, got %d", info.ResourcesByType["file"])
	}

	if info.ResourcesByType["package"] != 1 {
		t.Errorf("Expected 1 package resource, got %d", info.ResourcesByType["package"])
	}

	if info.ResourcesByHost["server1"] != 3 {
		t.Errorf("Expected 3 resources on server1, got %d", info.ResourcesByHost["server1"])
	}

	if info.ResourcesByHost["server2"] != 1 {
		t.Errorf("Expected 1 resource on server2, got %d", info.ResourcesByHost["server2"])
	}
}

func TestRollbackExecutor_ListSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSnapshotManager(tmpDir)
	registry := newMockModuleRegistry()
	logger := &mockLogger{}

	executor := NewRollbackExecutor(sm, registry, logger)

	// Create multiple snapshots
	for i := 0; i < 3; i++ {
		snapshot, err := sm.CreateSnapshot("playbook-123", "Test snapshot")
		if err != nil {
			t.Fatalf("Failed to create snapshot: %v", err)
		}

		// Add a resource
		resource := ResourceSnapshot{
			Type:       "file",
			Identifier: "/etc/test.conf",
			Host:       "server1",
			Reversible: true,
		}
		sm.AddResourceSnapshot(snapshot, resource)

		if err := sm.SaveSnapshot(snapshot); err != nil {
			t.Fatalf("Failed to save snapshot: %v", err)
		}

		time.Sleep(1 * time.Millisecond) // Ensure unique IDs
	}

	// List snapshots
	infos, err := executor.ListSnapshots()
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(infos) != 3 {
		t.Errorf("Expected 3 snapshots, got %d", len(infos))
	}

	for _, info := range infos {
		if info.TotalResources != 1 {
			t.Errorf("Expected 1 resource per snapshot, got %d", info.TotalResources)
		}
	}
}

func TestCountReversible(t *testing.T) {
	resources := []ResourceSnapshot{
		{Reversible: true, RollbackOp: &RollbackOperation{}},
		{Reversible: true, RollbackOp: &RollbackOperation{}},
		{Reversible: false, RollbackOp: nil},
		{Reversible: true, RollbackOp: nil}, // Reversible but no rollback op
	}

	count := countReversible(resources)
	if count != 2 {
		t.Errorf("Expected 2 reversible resources, got %d", count)
	}
}

func TestRollbackExecutor_Rollback_SnapshotNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSnapshotManager(tmpDir)
	registry := newMockModuleRegistry()
	logger := &mockLogger{}

	executor := NewRollbackExecutor(sm, registry, logger)

	// Try to rollback non-existent snapshot
	result, err := executor.Rollback(context.Background(), "non-existent")
	if err == nil {
		t.Error("Expected error when rolling back non-existent snapshot")
	}

	if result.Success {
		t.Error("Expected rollback to fail")
	}
}

func TestRollbackExecutor_Rollback_EmptySnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSnapshotManager(tmpDir)
	registry := newMockModuleRegistry()
	logger := &mockLogger{}

	executor := NewRollbackExecutor(sm, registry, logger)

	// Create empty snapshot
	snapshot, err := sm.CreateSnapshot("playbook-123", "Empty snapshot")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	if err := sm.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Rollback empty snapshot
	result, err := executor.Rollback(context.Background(), snapshot.ID)
	if err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	if !result.Success {
		t.Error("Expected rollback to succeed for empty snapshot")
	}

	if result.ResourcesRolled != 0 {
		t.Errorf("Expected 0 resources rolled back, got %d", result.ResourcesRolled)
	}
}

func TestRollbackExecutor_Rollback_NonReversibleResources(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSnapshotManager(tmpDir)
	registry := newMockModuleRegistry()
	logger := &mockLogger{}

	executor := NewRollbackExecutor(sm, registry, logger)

	// Create snapshot with non-reversible resources
	snapshot, err := sm.CreateSnapshot("playbook-123", "Test snapshot")
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	resource := ResourceSnapshot{
		Type:       "command",
		Identifier: "echo test",
		Host:       "server1",
		Reversible: false,
	}
	sm.AddResourceSnapshot(snapshot, resource)

	if err := sm.SaveSnapshot(snapshot); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Rollback
	result, err := executor.Rollback(context.Background(), snapshot.ID)
	if err != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	if !result.Success {
		t.Error("Expected rollback to succeed (skipping non-reversible resources)")
	}

	if result.ResourcesRolled != 0 {
		t.Errorf("Expected 0 resources rolled back, got %d", result.ResourcesRolled)
	}
}
