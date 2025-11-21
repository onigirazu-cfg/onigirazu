package state

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestEnhancedManager_Shutdown(t *testing.T) {
	// Create a temporary state file
	tmpFile := t.TempDir() + "/test.state"

	manager := NewEnhanced(tmpFile, true, 5)

	// Simulate loading state
	_, _ = manager.LoadState(context.Background())

	// Create some goroutines by calling SetTaskState multiple times
	taskState := &types.TaskState{
		Host:     "localhost",
		Module:   "test",
		LastRun:  time.Now(),
		Checksum: "abc123",
	}

	for i := 0; i < 5; i++ {
		manager.SetTaskState("task_"+string(rune(48+i)), taskState)
	}

	// Shutdown should wait for goroutines
	err := manager.Shutdown(time.Second * 10)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

func TestEnhancedManager_ShutdownTimeout(t *testing.T) {
	tmpFile := t.TempDir() + "/test.state"
	manager := NewEnhanced(tmpFile, false, 5) // autoSave=false, so no goroutines

	// Shutdown should succeed quickly with no goroutines
	start := time.Now()
	err := manager.Shutdown(time.Second)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	if elapsed > time.Second*2 {
		t.Errorf("Shutdown took too long: %v", elapsed)
	}
}

func TestEnhancedManager_ChecksumCaching(t *testing.T) {
	tmpFile := t.TempDir() + "/test.state"
	manager := NewEnhanced(tmpFile, false, 5)

	task := types.Task{
		Name:   "test_task",
		Module: "command",
		Args:   map[string]interface{}{"cmd": "echo test"},
	}

	host := types.Host{
		Name: "localhost",
	}

	// First call calculates and caches
	checksum1 := manager.calculateTaskChecksum(task, host)

	// Second call should use cache (even with slightly different input)
	taskID := manager.generateTaskID(task, host)
	checksum2 := manager.getOrCalcTaskChecksum(taskID, task, host)

	if checksum1 != checksum2 {
		t.Errorf("Checksums should match: %s vs %s", checksum1, checksum2)
	}

	// Verify cache was populated
	if cached, ok := manager.checksumCache.Load(taskID); ok {
		if cached.(string) != checksum1 {
			t.Errorf("Cached checksum doesn't match calculated: %v vs %s", cached, checksum1)
		}
	} else {
		t.Errorf("Expected checksum to be cached for task ID %s", taskID)
	}
}

func TestEnhancedManager_IsTaskUpToDate_WithCache(t *testing.T) {
	tmpFile := t.TempDir() + "/test.state"
	manager := NewEnhanced(tmpFile, false, 5)

	task := types.Task{
		Name:   "test_task",
		Module: "command",
		Args:   map[string]interface{}{"cmd": "echo test"},
	}

	host := types.Host{
		Name: "localhost",
	}

	// Task is not up to date initially (no state)
	if manager.IsTaskUpToDate(task, host) {
		t.Errorf("Task should not be up to date with no state")
	}

	// Set task state with checksum
	checksum := manager.calculateTaskChecksum(task, host)
	taskID := manager.generateTaskID(task, host)

	taskState := &types.TaskState{
		Host:     host.Name,
		Module:   task.Module,
		LastRun:  time.Now(),
		Checksum: checksum,
	}

	manager.taskStates[taskID] = taskState

	// Now task should be up to date
	if !manager.IsTaskUpToDate(task, host) {
		t.Errorf("Task should be up to date with matching checksum")
	}

	// Verify cache is being used
	if cached, ok := manager.checksumCache.Load(taskID); !ok {
		t.Errorf("Expected checksum to be cached")
	} else if cached.(string) != checksum {
		t.Errorf("Cached checksum doesn't match")
	}
}

func TestEnhancedManager_ExecutionIsolation(t *testing.T) {
	tmpFile := t.TempDir() + "/test.state"
	manager := NewEnhanced(tmpFile, false, 5)

	// Create isolated executions
	exec1ID := "exec-1"
	exec2ID := "exec-2"

	err := manager.BeginExecution(exec1ID)
	if err != nil {
		t.Errorf("BeginExecution for exec1 failed: %v", err)
	}

	err = manager.BeginExecution(exec2ID)
	if err != nil {
		t.Errorf("BeginExecution for exec2 failed: %v", err)
	}

	// Set task state for exec1
	manager.mutex.Lock()
	manager.currentExecID = exec1ID
	manager.mutex.Unlock()

	taskState1 := &types.TaskState{
		Host:    "host1",
		Module:  "module1",
		LastRun: time.Now(),
	}

	manager.SetTaskState("task1", taskState1)

	// Switch execution context
	manager.mutex.Lock()
	manager.currentExecID = exec2ID
	manager.mutex.Unlock()

	// Set different task state for exec2
	taskState2 := &types.TaskState{
		Host:    "host2",
		Module:  "module2",
		LastRun: time.Now(),
	}

	manager.SetTaskState("task2", taskState2)

	// Verify exec1 and exec2 have different task states
	task1FromExec1, found := manager.GetExecutionTaskState(exec1ID, "task1")
	if !found {
		t.Errorf("Expected task1 in exec1")
	}
	if task1FromExec1.Host != "host1" {
		t.Errorf("Expected task1 host to be host1, got %s", task1FromExec1.Host)
	}

	task2FromExec2, found := manager.GetExecutionTaskState(exec2ID, "task2")
	if !found {
		t.Errorf("Expected task2 in exec2")
	}
	if task2FromExec2.Host != "host2" {
		t.Errorf("Expected task2 host to be host2, got %s", task2FromExec2.Host)
	}
}

func TestEnhancedManager_ListExecutions(t *testing.T) {
	tmpFile := t.TempDir() + "/test.state"
	manager := NewEnhanced(tmpFile, false, 5)

	// Create multiple executions
	execIDs := []string{"exec-1", "exec-2", "exec-3"}

	for _, id := range execIDs {
		err := manager.BeginExecution(id)
		if err != nil {
			t.Errorf("BeginExecution for %s failed: %v", id, err)
		}
	}

	// List executions
	active := manager.ListExecutions()

	if len(active) != len(execIDs) {
		t.Errorf("Expected %d active executions, got %d", len(execIDs), len(active))
	}

	// Verify all IDs are present
	found := make(map[string]bool)
	for _, id := range active {
		found[id] = true
	}

	for _, id := range execIDs {
		if !found[id] {
			t.Errorf("Expected execution %s in active list", id)
		}
	}
}

func TestEnhancedManager_EndExecution(t *testing.T) {
	tmpFile := t.TempDir() + "/test.state"
	manager := NewEnhanced(tmpFile, false, 5)

	execID := "exec-1"
	err := manager.BeginExecution(execID)
	if err != nil {
		t.Errorf("BeginExecution failed: %v", err)
	}

	// Set current execution
	manager.mutex.Lock()
	manager.currentExecID = execID
	manager.mutex.Unlock()

	// End execution without cleanup
	err = manager.EndExecution(execID, false)
	if err != nil {
		t.Errorf("EndExecution failed: %v", err)
	}

	// Execution should still exist
	manager.mutex.RLock()
	_, exists := manager.executions[execID]
	manager.mutex.RUnlock()

	if !exists {
		t.Errorf("Execution should still exist after EndExecution with cleanup=false")
	}

	// End with cleanup
	err = manager.EndExecution(execID, true)
	if err != nil {
		t.Errorf("EndExecution with cleanup failed: %v", err)
	}

	// Execution should be removed
	manager.mutex.RLock()
	_, exists = manager.executions[execID]
	manager.mutex.RUnlock()

	if exists {
		t.Errorf("Execution should be removed after EndExecution with cleanup=true")
	}
}

func TestEnhancedManager_GetExecutionStats(t *testing.T) {
	tmpFile := t.TempDir() + "/test.state"
	manager := NewEnhanced(tmpFile, false, 5)

	execID := "exec-1"
	err := manager.BeginExecution(execID)
	if err != nil {
		t.Errorf("BeginExecution failed: %v", err)
	}

	// Switch to this execution and add tasks
	manager.mutex.Lock()
	manager.currentExecID = execID
	manager.mutex.Unlock()

	for i := 0; i < 3; i++ {
		taskState := &types.TaskState{
			Host:    "localhost",
			Module:  "test",
			LastRun: time.Now(),
		}
		manager.SetTaskState("task_"+string(rune(48+i)), taskState) // 0, 1, 2
	}

	// Get stats
	taskCount, createdAt, updatedAt, err := manager.GetExecutionStats(execID)
	if err != nil {
		t.Errorf("GetExecutionStats failed: %v", err)
	}

	if taskCount != 3 {
		t.Errorf("Expected 3 tasks, got %d", taskCount)
	}

	if createdAt.IsZero() {
		t.Errorf("Expected non-zero createdAt")
	}

	if updatedAt.IsZero() {
		t.Errorf("Expected non-zero updatedAt")
	}
}

func TestEnhancedManager_Clear(t *testing.T) {
	tmpFile := t.TempDir() + "/test.state"
	manager := NewEnhanced(tmpFile, false, 5)

	// Add some data
	_, _ = manager.LoadState(context.Background())
	manager.taskStates["task1"] = &types.TaskState{Host: "localhost", Module: "test"}

	manager.mutex.Lock()
	manager.executions["exec1"] = &ExecutionState{
		TaskStates: make(map[string]*types.TaskState),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	manager.currentExecID = "exec1"
	manager.mutex.Unlock()

	// Clear
	err := manager.Clear()
	if err != nil {
		t.Errorf("Clear failed: %v", err)
	}

	// Verify cleared
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	if len(manager.taskStates) != 0 {
		t.Errorf("Expected empty taskStates after Clear")
	}

	if len(manager.executions) != 0 {
		t.Errorf("Expected empty executions after Clear")
	}

	if manager.currentExecID != "" {
		t.Errorf("Expected empty currentExecID after Clear")
	}
}

func TestEnhancedManager_SaveStateWithAutoSave(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/test.state"

	manager := NewEnhanced(tmpFile, true, 5) // Auto-save enabled

	_, _ = manager.LoadState(context.Background())

	// Set task state should trigger auto-save
	taskState := &types.TaskState{
		Host:   "localhost",
		Module: "test",
		Result: types.TaskResult{
			Success: true,
			Changed: true,
		},
	}

	manager.SetTaskState("task1", taskState)

	// Give background save time to complete
	time.Sleep(time.Millisecond * 100)

	// Shutdown should wait for pending saves
	err := manager.Shutdown(time.Second * 10)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Check that state file was created/updated
	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Errorf("State file not created: %v", err)
	}

	if info.Size() == 0 {
		t.Errorf("State file is empty")
	}
}
