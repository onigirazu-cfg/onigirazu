package state

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestExecutionIsolation verifies that concurrent executions maintain isolated state
func TestExecutionIsolation(t *testing.T) {
	// Setup
	stateFile := "./test_state_isolation.json"
	defer os.Remove(stateFile)

	manager := NewEnhanced(stateFile, false, 5)

	// Begin two concurrent executions
	execID1 := "exec-001"
	execID2 := "exec-002"

	if err := manager.BeginExecution(execID1); err != nil {
		t.Fatalf("Failed to begin execution 1: %v", err)
	}

	if err := manager.BeginExecution(execID2); err != nil {
		t.Fatalf("Failed to begin execution 2: %v", err)
	}

	// Set current execution to 1 and store state
	manager.currentExecID = execID1
	taskState1 := &types.TaskState{
		Checksum: "checksum-1",
		TaskName: "test-task-1",
	}
	manager.SetTaskState("task-1", taskState1)

	// Set current execution to 2 and store state
	manager.currentExecID = execID2
	taskState2 := &types.TaskState{
		Checksum: "checksum-2",
		TaskName: "test-task-2",
	}
	manager.SetTaskState("task-1", taskState2)

	// Verify isolation: exec1's task state should be unchanged
	manager.currentExecID = execID1
	retrieved1, exists1 := manager.GetTaskState("task-1")
	if !exists1 {
		t.Fatal("Task state not found in execution 1")
	}
	if retrieved1.Checksum != "checksum-1" {
		t.Errorf("Expected checksum-1, got %s", retrieved1.Checksum)
	}

	// Verify isolation: exec2's task state should be unchanged
	manager.currentExecID = execID2
	retrieved2, exists2 := manager.GetTaskState("task-1")
	if !exists2 {
		t.Fatal("Task state not found in execution 2")
	}
	if retrieved2.Checksum != "checksum-2" {
		t.Errorf("Expected checksum-2, got %s", retrieved2.Checksum)
	}
}

// TestConcurrentExecutions verifies that multiple concurrent goroutines work correctly
func TestConcurrentExecutions(t *testing.T) {
	stateFile := "./test_state_concurrent.json"
	defer os.Remove(stateFile)

	manager := NewEnhanced(stateFile, false, 5)
	numExecutions := 5
	tasksPerExecution := 10

	// Create multiple executions
	for i := 1; i <= numExecutions; i++ {
		execID := fmt.Sprintf("exec-%03d", i)
		if err := manager.BeginExecution(execID); err != nil {
			t.Fatalf("Failed to begin execution %s: %v", execID, err)
		}
	}

	var wg sync.WaitGroup

	// Launch concurrent goroutines, each modifying their own execution's state
	// Using SetTaskStateForExecution to directly set per-execution state
	for i := 1; i <= numExecutions; i++ {
		wg.Add(1)
		go func(execNum int) {
			defer wg.Done()

			execID := fmt.Sprintf("exec-%03d", execNum)

			// Get execution directly and store tasks
			manager.mutex.Lock()
			execState, exists := manager.executions[execID]
			manager.mutex.Unlock()

			if !exists {
				t.Errorf("Execution %s not found", execID)
				return
			}

			// Store multiple tasks directly in this execution
			for j := 1; j <= tasksPerExecution; j++ {
				taskID := fmt.Sprintf("task-%d-%d", execNum, j)
				taskState := &types.TaskState{
					Checksum: fmt.Sprintf("checksum-%d-%d", execNum, j),
					TaskName: fmt.Sprintf("task-name-%d-%d", execNum, j),
				}

				// Store directly in execution state with proper locking
				manager.mutex.Lock()
				execState.TaskStates[taskID] = taskState
				execState.UpdatedAt = time.Now()
				manager.mutex.Unlock()

				// Simulate some work
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify each execution has its own isolated state
	for i := 1; i <= numExecutions; i++ {
		execID := fmt.Sprintf("exec-%03d", i)
		count, created, _, err := manager.GetExecutionStats(execID)
		if err != nil {
			t.Fatalf("Failed to get stats for %s: %v", execID, err)
		}

		if count != tasksPerExecution {
			t.Errorf("Execution %s: expected %d tasks, got %d", execID, tasksPerExecution, count)
		}

		if created.IsZero() {
			t.Errorf("Execution %s: creation time is zero", execID)
		}
	}

	// Verify no cross-contamination
	for i := 1; i <= numExecutions; i++ {
		for j := 1; j <= tasksPerExecution; j++ {
			taskID := fmt.Sprintf("task-%d-%d", i, j)
			for k := 1; k <= numExecutions; k++ {
				if k == i {
					continue // Skip same execution
				}

				execID := fmt.Sprintf("exec-%03d", k)
				_, exists := manager.GetExecutionTaskState(execID, taskID)
				if exists {
					t.Errorf("Task %s found in execution %s (should not exist)", taskID, execID)
				}
			}
		}
	}
}

// TestExecutionLifecycle tests BeginExecution, EndExecution, and cleanup
func TestExecutionLifecycle(t *testing.T) {
	stateFile := "./test_state_lifecycle.json"
	defer os.Remove(stateFile)

	manager := NewEnhanced(stateFile, false, 5)

	execID := "exec-lifecycle-test"

	// Test BeginExecution
	if err := manager.BeginExecution(execID); err != nil {
		t.Fatalf("BeginExecution failed: %v", err)
	}

	// Verify execution exists
	execs := manager.ListExecutions()
	if len(execs) != 1 || execs[0] != execID {
		t.Errorf("Expected execution %s in list, got %v", execID, execs)
	}

	// Add task state
	manager.currentExecID = execID
	taskState := &types.TaskState{
		Checksum: "test-checksum",
		TaskName: "test-task-name",
	}
	manager.SetTaskState("test-task", taskState)

	// Verify task exists
	retrieved, exists := manager.GetExecutionTaskState(execID, "test-task")
	if !exists || retrieved.Checksum != "test-checksum" {
		t.Error("Task state not properly stored")
	}

	// Test EndExecution without cleanup
	if err := manager.EndExecution(execID, false); err != nil {
		t.Fatalf("EndExecution failed: %v", err)
	}

	// Execution should still exist
	_, exists = manager.GetExecutionTaskState(execID, "test-task")
	if !exists {
		t.Error("Execution was removed when it shouldn't have been")
	}

	// Test EndExecution with cleanup
	if err := manager.EndExecution(execID, true); err != nil {
		t.Fatalf("EndExecution with cleanup failed: %v", err)
	}

	// Execution should now be gone
	execs = manager.ListExecutions()
	if len(execs) != 0 {
		t.Errorf("Expected 0 executions after cleanup, got %d", len(execs))
	}
}

// TestGetExecutionTaskState verifies retrieval from specific execution
func TestGetExecutionTaskState(t *testing.T) {
	stateFile := "./test_state_get_exec.json"
	defer os.Remove(stateFile)

	manager := NewEnhanced(stateFile, false, 5)

	// Create execution and add task state
	execID := "exec-get-test"
	_ = manager.BeginExecution(execID) //nolint:errcheck // Test
	manager.currentExecID = execID

	taskID := "task-123"
	taskState := &types.TaskState{
		Checksum: "abc123",
		TaskName: "completed-task",
	}
	manager.SetTaskState(taskID, taskState)

	// Retrieve via GetExecutionTaskState
	retrieved, exists := manager.GetExecutionTaskState(execID, taskID)
	if !exists {
		t.Fatal("GetExecutionTaskState: task not found")
	}

	if retrieved.Checksum != "abc123" {
		t.Errorf("Expected checksum 'abc123', got %s", retrieved.Checksum)
	}

	if retrieved.TaskName != "completed-task" {
		t.Errorf("Expected task name 'completed-task', got %s", retrieved.TaskName)
	}

	// Test non-existent task
	_, exists = manager.GetExecutionTaskState(execID, "non-existent")
	if exists {
		t.Error("GetExecutionTaskState: should return false for non-existent task")
	}

	// Test non-existent execution
	_, exists = manager.GetExecutionTaskState("non-existent-exec", taskID)
	if exists {
		t.Error("GetExecutionTaskState: should return false for non-existent execution")
	}
}

// TestBackwardsCompatibility tests that legacy taskStates still work
func TestBackwardsCompatibility(t *testing.T) {
	stateFile := "./test_state_compat.json"
	defer os.Remove(stateFile)

	manager := NewEnhanced(stateFile, false, 5)

	// Clear any execution context
	manager.currentExecID = ""

	// Set task state (should go to legacy storage)
	taskState := &types.TaskState{
		Checksum: "legacy-checksum",
		TaskName: "legacy-task-name",
	}
	manager.SetTaskState("legacy-task", taskState)

	// Retrieve without execution context (should work)
	retrieved, exists := manager.GetTaskState("legacy-task")
	if !exists {
		t.Fatal("Legacy task state not found")
	}

	if retrieved.Checksum != "legacy-checksum" {
		t.Errorf("Expected legacy-checksum, got %s", retrieved.Checksum)
	}
}

// TestExecutionErrorHandling tests error conditions
func TestExecutionErrorHandling(t *testing.T) {
	stateFile := "./test_state_errors.json"
	defer os.Remove(stateFile)

	manager := NewEnhanced(stateFile, false, 5)

	// Test empty execution ID
	err := manager.BeginExecution("")
	if err == nil {
		t.Error("Expected error for empty execution ID")
	}

	// Test duplicate execution
	execID := "exec-duplicate"
	_ = manager.BeginExecution(execID) //nolint:errcheck // Test
	err = manager.BeginExecution(execID)
	if err == nil {
		t.Error("Expected error for duplicate execution ID")
	}

	// Test EndExecution with invalid execution
	err = manager.EndExecution("non-existent", false)
	if err == nil {
		t.Error("Expected error for non-existent execution")
	}

	// Test EndExecution with empty ID
	err = manager.EndExecution("", false)
	if err == nil {
		t.Error("Expected error for empty execution ID in EndExecution")
	}
}

// TestHighConcurrency stress tests with many concurrent operations
func TestHighConcurrency(t *testing.T) {
	stateFile := "./test_state_high_concurrency.json"
	defer os.Remove(stateFile)

	manager := NewEnhanced(stateFile, false, 5)

	numGoroutines := 20
	operationsPerGoroutine := 50

	// Create all executions first
	for i := 0; i < numGoroutines; i++ {
		execID := fmt.Sprintf("exec-stress-%d", i)
		if err := manager.BeginExecution(execID); err != nil {
			t.Errorf("Failed to begin execution: %v", err)
			return
		}
	}

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			execID := fmt.Sprintf("exec-stress-%d", idx)

			// Get execution directly
			manager.mutex.Lock()
			execState, exists := manager.executions[execID]
			manager.mutex.Unlock()

			if !exists {
				t.Errorf("Execution %s not found", execID)
				return
			}

			// Perform many operations directly on the execution state
			for j := 0; j < operationsPerGoroutine; j++ {
				taskID := fmt.Sprintf("task-%d-%d", idx, j)
				taskState := &types.TaskState{
					Checksum: fmt.Sprintf("checksum-%d-%d", idx, j),
					TaskName: fmt.Sprintf("task-stress-%d-%d", idx, j),
				}

				// Store directly with proper locking
				manager.mutex.Lock()
				execState.TaskStates[taskID] = taskState
				execState.UpdatedAt = time.Now()
				manager.mutex.Unlock()
			}

			_ = manager.EndExecution(execID, false) //nolint:errcheck // Test
		}(i)
	}

	wg.Wait()

	// Verify all executions exist with correct task counts
	execs := manager.ListExecutions()
	if len(execs) != numGoroutines {
		t.Errorf("Expected %d executions, got %d", numGoroutines, len(execs))
	}

	for i := 0; i < numGoroutines; i++ {
		execID := fmt.Sprintf("exec-stress-%d", i)
		count, _, _, err := manager.GetExecutionStats(execID)
		if err != nil {
			t.Errorf("Failed to get stats for %s: %v", execID, err)
			continue
		}

		if count != operationsPerGoroutine {
			t.Errorf("Expected %d tasks in %s, got %d", operationsPerGoroutine, execID, count)
		}
	}
}

// TestContextCancellation tests behavior with canceled context
func TestContextCancellation(t *testing.T) {
	stateFile := "./test_state_context.json"
	defer os.Remove(stateFile)

	manager := NewEnhanced(stateFile, false, 5)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Try to load state with canceled context
	_, err := manager.LoadState(ctx)
	if err == nil {
		t.Error("Expected error for canceled context")
	}
}

// TestListExecutions verifies execution list functionality
func TestListExecutions(t *testing.T) {
	stateFile := "./test_state_list.json"
	defer os.Remove(stateFile)

	manager := NewEnhanced(stateFile, false, 5)

	// Initially empty
	execs := manager.ListExecutions()
	if len(execs) != 0 {
		t.Errorf("Expected empty list, got %d executions", len(execs))
	}

	// Add some executions
	for i := 1; i <= 5; i++ {
		execID := fmt.Sprintf("exec-%d", i)
		_ = manager.BeginExecution(execID) //nolint:errcheck // Test
	}

	execs = manager.ListExecutions()
	if len(execs) != 5 {
		t.Errorf("Expected 5 executions, got %d", len(execs))
	}

	// Remove one with cleanup
	_ = manager.EndExecution("exec-3", true) //nolint:errcheck // Test cleanup

	execs = manager.ListExecutions()
	if len(execs) != 4 {
		t.Errorf("Expected 4 executions after cleanup, got %d", len(execs))
	}
}
