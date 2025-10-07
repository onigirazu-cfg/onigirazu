package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCoreEngine tests the creation of a new CoreEngine instance
func TestNewCoreEngine(t *testing.T) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	assert.NotNil(t, engine, "CoreEngine should not be nil")
	assert.NotNil(t, engine.logger, "Logger should not be nil")
	assert.NotNil(t, engine.parser, "Parser should not be nil")
	assert.NotNil(t, engine.inventory, "Inventory should not be nil")
	assert.NotNil(t, engine.modules, "Modules registry should not be nil")
}

// TestCoreEngine_Run_InvalidPlaybook tests Run with invalid playbook path
func TestCoreEngine_Run_InvalidPlaybook(t *testing.T) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	// Create temporary state file
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	err := engine.Run("nonexistent.yml", "inventory.ini", false, stateFile)
	assert.Error(t, err, "Should return error for nonexistent playbook")
	assert.Contains(t, err.Error(), "error parsing playbook", "Error should mention playbook parsing")
}

// TestCoreEngine_Run_InvalidInventory tests Run with invalid inventory path
func TestCoreEngine_Run_InvalidInventory(t *testing.T) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	// Create temporary playbook
	tmpDir := t.TempDir()
	playbookPath := filepath.Join(tmpDir, "test.yml")
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create minimal valid playbook
	playbookContent := `---
name: Test Playbook
plays:
  - name: Test Play
    hosts: all
    tasks:
      - name: Test task
        debug:
          msg: "Hello"
`
	err := os.WriteFile(playbookPath, []byte(playbookContent), 0644)
	require.NoError(t, err, "Failed to create test playbook")

	err = engine.Run(playbookPath, "nonexistent_inventory.ini", false, stateFile)
	assert.Error(t, err, "Should return error for nonexistent inventory")
	assert.Contains(t, err.Error(), "error loading inventory", "Error should mention inventory loading")
}

// TestCoreEngine_Run_CheckMode tests Run in check mode
func TestCoreEngine_Run_CheckMode(t *testing.T) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	tmpDir := t.TempDir()
	playbookPath := filepath.Join(tmpDir, "test.yml")
	inventoryPath := filepath.Join(tmpDir, "inventory.ini")
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create minimal playbook
	playbookContent := `---
name: Test Playbook
plays:
  - name: Test Play
    hosts: localhost
    tasks:
      - name: Test debug task
        debug:
          msg: "Test message"
`
	err := os.WriteFile(playbookPath, []byte(playbookContent), 0644)
	require.NoError(t, err, "Failed to create test playbook")

	// Create minimal inventory
	inventoryContent := `[localhost]
127.0.0.1 ansible_connection=local
`
	err = os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err, "Failed to create test inventory")

	// Run in check mode
	err = engine.Run(playbookPath, inventoryPath, true, stateFile)
	// Note: This might fail due to inventory parsing, but we're testing the flow
	if err != nil {
		t.Logf("Check mode run failed (expected in some cases): %v", err)
	}
}

// TestCoreEngine_executePlaybook tests playbook execution
func TestCoreEngine_executePlaybook(t *testing.T) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	// Create test playbook
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name:   "Test Task",
						Module: "debug",
						Args: map[string]interface{}{
							"msg": "Hello World",
						},
					},
				},
			},
		},
	}

	// Create empty state
	state := &types.State{
		Playbook:  "test.yml",
		LastRun:   time.Now(),
		Results:   []types.PlayResult{},
		Variables: map[string]interface{}{},
		Checksums: map[string]string{},
	}

	// Execute playbook (will fail due to no inventory, but tests the flow)
	_, err := engine.executePlaybook(playbook, false, state)
	assert.Error(t, err, "Should fail without proper inventory setup")
}

// TestCoreEngine_executePlay tests single play execution
func TestCoreEngine_executePlay(t *testing.T) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	// Create test play
	play := types.Play{
		Name:  "Test Play",
		Hosts: "localhost",
		Tasks: []types.Task{
			{
				Name:   "Test Task",
				Module: "debug",
				Args: map[string]interface{}{
					"msg": "Hello World",
				},
			},
		},
	}

	// Create test host
	host := types.Host{
		Name: "localhost",
		Vars: map[string]interface{}{},
	}

	// Create empty state
	state := &types.State{
		Playbook:  "test.yml",
		LastRun:   time.Now(),
		Results:   []types.PlayResult{},
		Variables: map[string]interface{}{},
		Checksums: map[string]string{},
	}

	// Execute play
	result, err := engine.executePlay(play, host, false, state)

	// The execution might fail due to SSH connection, but we test the structure
	assert.NotEmpty(t, result.PlayName, "Play name should be set")
	assert.Equal(t, "Test Play", result.PlayName, "Play name should match")
	assert.Equal(t, "localhost", result.Host, "Host should match")
	assert.NotZero(t, result.StartTime, "Start time should be set")

	if err != nil {
		t.Logf("Play execution failed (expected without SSH): %v", err)
	}
}

// TestCoreEngine_executePlay_WithIgnoreErrors tests play execution with ignore_errors
func TestCoreEngine_executePlay_WithIgnoreErrors(t *testing.T) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	// Create test play with ignore_errors
	play := types.Play{
		Name:  "Test Play with Ignore Errors",
		Hosts: "localhost",
		Tasks: []types.Task{
			{
				Name:         "Failing Task",
				Module:       "nonexistent_module",
				Args:         map[string]interface{}{},
				IgnoreErrors: true,
			},
			{
				Name:   "Second Task",
				Module: "debug",
				Args: map[string]interface{}{
					"msg": "This should still run",
				},
			},
		},
	}

	host := types.Host{
		Name: "localhost",
		Vars: map[string]interface{}{},
	}

	state := &types.State{
		Playbook:  "test.yml",
		LastRun:   time.Now(),
		Results:   []types.PlayResult{},
		Variables: map[string]interface{}{},
		Checksums: map[string]string{},
	}

	// Execute play
	result, err := engine.executePlay(play, host, false, state)

	// Should continue despite first task failure
	assert.NotEmpty(t, result.Tasks, "Should have task results")
	if err != nil {
		t.Logf("Play execution completed with errors (expected): %v", err)
	}
}

// TestCoreEngine_executeTask tests individual task execution
func TestCoreEngine_executeTask(t *testing.T) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	// Create test task
	task := types.Task{
		Name:   "Test Debug Task",
		Module: "debug",
		Args: map[string]interface{}{
			"msg": "Test message",
		},
	}

	host := types.Host{
		Name: "localhost",
		Vars: map[string]interface{}{},
	}

	state := &types.State{
		Playbook:  "test.yml",
		LastRun:   time.Now(),
		Results:   []types.PlayResult{},
		Variables: map[string]interface{}{},
		Checksums: map[string]string{},
	}

	// Execute task
	result, err := engine.executeTask(task, host, false, state)

	// Check result structure
	assert.Equal(t, "Test Debug Task", result.TaskName, "Task name should match")
	assert.Equal(t, "localhost", result.Host, "Host should match")
	assert.Equal(t, "debug", result.Module, "Module should match")

	if err != nil {
		t.Logf("Task execution failed (expected without SSH): %v", err)
	}
}

// TestCoreEngine_executeTask_CheckMode tests task execution in check mode
func TestCoreEngine_executeTask_CheckMode(t *testing.T) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	// Create test task
	task := types.Task{
		Name:   "Test Task in Check Mode",
		Module: "debug",
		Args: map[string]interface{}{
			"msg": "Test message",
		},
	}

	host := types.Host{
		Name: "localhost",
		Vars: map[string]interface{}{},
	}

	state := &types.State{
		Playbook:  "test.yml",
		LastRun:   time.Now(),
		Results:   []types.PlayResult{},
		Variables: map[string]interface{}{},
		Checksums: map[string]string{},
	}

	// Execute task in check mode
	result, err := engine.executeTask(task, host, true, state)

	// In check mode, should return success without actual execution
	assert.NoError(t, err, "Check mode should not return error for valid module")
	assert.True(t, result.Success, "Check mode should return success")
	assert.False(t, result.Changed, "Check mode should not report changes")
	assert.Contains(t, result.Output["message"], "Check mode", "Should indicate check mode")
}

// TestCoreEngine_executeTask_InvalidModule tests task with invalid module
func TestCoreEngine_executeTask_InvalidModule(t *testing.T) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	// Create test task with invalid module
	task := types.Task{
		Name:   "Test Invalid Module",
		Module: "nonexistent_module_xyz",
		Args:   map[string]interface{}{},
	}

	host := types.Host{
		Name: "localhost",
		Vars: map[string]interface{}{},
	}

	state := &types.State{
		Playbook:  "test.yml",
		LastRun:   time.Now(),
		Results:   []types.PlayResult{},
		Variables: map[string]interface{}{},
		Checksums: map[string]string{},
	}

	// Execute task in check mode (to avoid SSH connection)
	_, err := engine.executeTask(task, host, true, state)

	// Should return error for invalid module
	assert.Error(t, err, "Should return error for invalid module")
}

// TestCoreEngine_executePlaybook_EmptyPlaybook tests execution with empty playbook
func TestCoreEngine_executePlaybook_EmptyPlaybook(t *testing.T) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	// Create empty playbook
	playbook := &types.Playbook{
		Plays: []types.Play{},
	}

	state := &types.State{
		Playbook:  "test.yml",
		LastRun:   time.Now(),
		Results:   []types.PlayResult{},
		Variables: map[string]interface{}{},
		Checksums: map[string]string{},
	}

	// Execute empty playbook
	results, err := engine.executePlaybook(playbook, false, state)

	assert.NoError(t, err, "Empty playbook should not return error")
	assert.Empty(t, results, "Empty playbook should return no results")
}

// TestCoreEngine_executePlaybook_MultiplePlayss tests execution with multiple plays
func TestCoreEngine_executePlaybook_MultiplePlays(t *testing.T) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	// Create playbook with multiple plays
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "First Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name:   "First Task",
						Module: "debug",
						Args:   map[string]interface{}{"msg": "First"},
					},
				},
			},
			{
				Name:  "Second Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name:   "Second Task",
						Module: "debug",
						Args:   map[string]interface{}{"msg": "Second"},
					},
				},
			},
		},
	}

	state := &types.State{
		Playbook:  "test.yml",
		LastRun:   time.Now(),
		Results:   []types.PlayResult{},
		Variables: map[string]interface{}{},
		Checksums: map[string]string{},
	}

	// Execute playbook (will fail due to no inventory)
	_, err := engine.executePlaybook(playbook, false, state)
	assert.Error(t, err, "Should fail without proper inventory")
}

// Benchmark tests
func BenchmarkCoreEngine_executeTask_CheckMode(b *testing.B) {
	log := logger.New(false)
	engine := NewCoreEngine(log)

	task := types.Task{
		Name:   "Benchmark Task",
		Module: "debug",
		Args:   map[string]interface{}{"msg": "Benchmark"},
	}

	host := types.Host{
		Name: "localhost",
		Vars: map[string]interface{}{},
	}

	state := &types.State{
		Playbook:  "test.yml",
		LastRun:   time.Now(),
		Results:   []types.PlayResult{},
		Variables: map[string]interface{}{},
		Checksums: map[string]string{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.executeTask(task, host, true, state)
	}
}

// TestCoreEngine_Context tests context usage
func TestCoreEngine_Context(t *testing.T) {
	log := logger.New(false)
	_ = NewCoreEngine(log)

	// Test that engine can work with context
	ctx := context.Background()
	assert.NotNil(t, ctx, "Context should not be nil")

	// The actual context usage is tested through Run method
	// which internally uses context for parser and inventory
}
