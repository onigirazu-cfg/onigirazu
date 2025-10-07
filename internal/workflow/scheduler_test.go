package workflow

import (
	"sync"
	"testing"
	"time"
)

// TestNewWorkflowScheduler tests scheduler creation
func TestNewWorkflowScheduler(t *testing.T) {
	ws := NewWorkflowScheduler()

	if ws == nil {
		t.Fatal("Expected scheduler to be created")
	}

	if ws.cron == nil {
		t.Error("Expected cron to be initialized")
	}

	if ws.schedules == nil {
		t.Error("Expected schedules map to be initialized")
	}

	if ws.callbacks == nil {
		t.Error("Expected callbacks map to be initialized")
	}
}

// TestWorkflowScheduler_StartStop tests starting and stopping scheduler
func TestWorkflowScheduler_StartStop(t *testing.T) {
	ws := NewWorkflowScheduler()

	// Should not panic
	ws.Start()
	time.Sleep(10 * time.Millisecond)
	ws.Stop()
}

// TestWorkflowScheduler_ScheduleWorkflow tests workflow scheduling
func TestWorkflowScheduler_ScheduleWorkflow(t *testing.T) {
	ws := NewWorkflowScheduler()
	defer ws.Stop()

	workflowID := "test-workflow"
	schedule := "*/5 * * * * *" // Every 5 seconds

	err := ws.ScheduleWorkflow(workflowID, schedule)
	if err != nil {
		t.Fatalf("Expected successful scheduling, got error: %v", err)
	}

	// Verify schedule was added
	ws.mutex.RLock()
	_, exists := ws.schedules[workflowID]
	ws.mutex.RUnlock()

	if !exists {
		t.Error("Expected workflow to be scheduled")
	}
}

// TestWorkflowScheduler_ScheduleWorkflow_InvalidSchedule tests invalid schedule
func TestWorkflowScheduler_ScheduleWorkflow_InvalidSchedule(t *testing.T) {
	ws := NewWorkflowScheduler()
	defer ws.Stop()

	workflowID := "test-workflow"
	invalidSchedule := "invalid cron expression"

	err := ws.ScheduleWorkflow(workflowID, invalidSchedule)
	if err == nil {
		t.Error("Expected error for invalid schedule")
	}
}

// TestWorkflowScheduler_ScheduleWorkflow_Replace tests replacing existing schedule
func TestWorkflowScheduler_ScheduleWorkflow_Replace(t *testing.T) {
	ws := NewWorkflowScheduler()
	defer ws.Stop()

	workflowID := "test-workflow"
	schedule1 := "*/5 * * * * *"
	schedule2 := "*/10 * * * * *"

	// Schedule first time
	err := ws.ScheduleWorkflow(workflowID, schedule1)
	if err != nil {
		t.Fatalf("Expected successful scheduling, got error: %v", err)
	}

	ws.mutex.RLock()
	entryID1 := ws.schedules[workflowID]
	ws.mutex.RUnlock()

	// Schedule again with different schedule
	err = ws.ScheduleWorkflow(workflowID, schedule2)
	if err != nil {
		t.Fatalf("Expected successful rescheduling, got error: %v", err)
	}

	ws.mutex.RLock()
	entryID2 := ws.schedules[workflowID]
	ws.mutex.RUnlock()

	// Entry IDs should be different
	if entryID1 == entryID2 {
		t.Error("Expected different entry IDs after rescheduling")
	}
}

// TestWorkflowScheduler_UnscheduleWorkflow tests workflow unscheduling
func TestWorkflowScheduler_UnscheduleWorkflow(t *testing.T) {
	ws := NewWorkflowScheduler()
	defer ws.Stop()

	workflowID := "test-workflow"
	schedule := "*/5 * * * * *"

	// Schedule workflow
	err := ws.ScheduleWorkflow(workflowID, schedule)
	if err != nil {
		t.Fatalf("Expected successful scheduling, got error: %v", err)
	}

	// Verify it's scheduled
	ws.mutex.RLock()
	_, exists := ws.schedules[workflowID]
	ws.mutex.RUnlock()

	if !exists {
		t.Fatal("Expected workflow to be scheduled")
	}

	// Unschedule
	ws.UnscheduleWorkflow(workflowID)

	// Verify it's removed
	ws.mutex.RLock()
	_, exists = ws.schedules[workflowID]
	ws.mutex.RUnlock()

	if exists {
		t.Error("Expected workflow to be unscheduled")
	}
}

// TestWorkflowScheduler_UnscheduleWorkflow_NonExistent tests unscheduling non-existent workflow
func TestWorkflowScheduler_UnscheduleWorkflow_NonExistent(t *testing.T) {
	ws := NewWorkflowScheduler()
	defer ws.Stop()

	// Should not panic
	ws.UnscheduleWorkflow("non-existent")
}

// TestWorkflowScheduler_SetCallback tests setting callback
func TestWorkflowScheduler_SetCallback(t *testing.T) {
	ws := NewWorkflowScheduler()
	defer ws.Stop()

	workflowID := "test-workflow"
	callbackCalled := false

	callback := func(id string) error {
		callbackCalled = true
		return nil
	}

	ws.SetCallback(workflowID, callback)

	// Verify callback was set
	ws.mutex.RLock()
	_, exists := ws.callbacks[workflowID]
	ws.mutex.RUnlock()

	if !exists {
		t.Error("Expected callback to be set")
	}
}

// TestWorkflowScheduler_CallbackExecution tests callback execution
func TestWorkflowScheduler_CallbackExecution(t *testing.T) {
	ws := NewWorkflowScheduler()
	ws.Start()
	defer ws.Stop()

	workflowID := "test-workflow"
	var callbackCalled bool
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(1)

	callback := func(id string) error {
		mu.Lock()
		callbackCalled = true
		mu.Unlock()
		wg.Done()
		return nil
	}

	ws.SetCallback(workflowID, callback)

	// Schedule to run every second
	schedule := "* * * * * *"
	err := ws.ScheduleWorkflow(workflowID, schedule)
	if err != nil {
		t.Fatalf("Expected successful scheduling, got error: %v", err)
	}

	// Wait for callback to be called (with timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Callback was not called within timeout")
	}

	mu.Lock()
	called := callbackCalled
	mu.Unlock()

	if !called {
		t.Error("Expected callback to be called")
	}
}

// TestWorkflowScheduler_GetScheduledWorkflows tests getting scheduled workflows
func TestWorkflowScheduler_GetScheduledWorkflows(t *testing.T) {
	ws := NewWorkflowScheduler()
	defer ws.Stop()

	// Empty initially
	workflows := ws.GetScheduledWorkflows()
	if len(workflows) != 0 {
		t.Errorf("Expected 0 scheduled workflows, got %d", len(workflows))
	}

	// Schedule some workflows
	schedule := "*/5 * * * * *"
	ws.ScheduleWorkflow("workflow1", schedule)
	ws.ScheduleWorkflow("workflow2", schedule)
	ws.ScheduleWorkflow("workflow3", schedule)

	workflows = ws.GetScheduledWorkflows()
	if len(workflows) != 3 {
		t.Errorf("Expected 3 scheduled workflows, got %d", len(workflows))
	}

	// Verify all workflows are present
	workflowMap := make(map[string]bool)
	for _, id := range workflows {
		workflowMap[id] = true
	}

	expectedWorkflows := []string{"workflow1", "workflow2", "workflow3"}
	for _, id := range expectedWorkflows {
		if !workflowMap[id] {
			t.Errorf("Expected workflow '%s' in scheduled list", id)
		}
	}
}

// TestWorkflowScheduler_GetNextRun tests getting next run time
func TestWorkflowScheduler_GetNextRun(t *testing.T) {
	ws := NewWorkflowScheduler()
	ws.Start()
	defer ws.Stop()

	workflowID := "test-workflow"
	schedule := "*/5 * * * * *" // Every 5 seconds

	err := ws.ScheduleWorkflow(workflowID, schedule)
	if err != nil {
		t.Fatalf("Expected successful scheduling, got error: %v", err)
	}

	// Give scheduler time to process
	time.Sleep(10 * time.Millisecond)

	nextRun, err := ws.GetNextRun(workflowID)
	if err != nil {
		t.Fatalf("Expected to get next run time, got error: %v", err)
	}

	if nextRun.IsZero() {
		t.Error("Expected non-zero next run time")
	}

	if nextRun.Before(time.Now()) {
		t.Error("Expected next run time to be in the future")
	}
}

// TestWorkflowScheduler_GetNextRun_NonExistent tests getting next run for non-existent workflow
func TestWorkflowScheduler_GetNextRun_NonExistent(t *testing.T) {
	ws := NewWorkflowScheduler()
	defer ws.Stop()

	_, err := ws.GetNextRun("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent workflow")
	}
}

// TestWorkflowScheduler_MultipleWorkflows tests scheduling multiple workflows
func TestWorkflowScheduler_MultipleWorkflows(t *testing.T) {
	ws := NewWorkflowScheduler()
	ws.Start()
	defer ws.Stop()

	var mu sync.Mutex
	callCounts := make(map[string]int)
	var wg sync.WaitGroup

	// Schedule 3 workflows
	for i := 1; i <= 3; i++ {
		workflowID := "workflow" + string(rune('0'+i))
		wg.Add(1)

		callback := func(id string) error {
			mu.Lock()
			callCounts[id]++
			if callCounts[id] == 1 {
				wg.Done()
			}
			mu.Unlock()
			return nil
		}

		ws.SetCallback(workflowID, callback)
		schedule := "* * * * * *" // Every second
		err := ws.ScheduleWorkflow(workflowID, schedule)
		if err != nil {
			t.Fatalf("Expected successful scheduling for %s, got error: %v", workflowID, err)
		}
	}

	// Wait for all callbacks to be called at least once
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatal("Not all callbacks were called within timeout")
	}

	mu.Lock()
	defer mu.Unlock()

	for i := 1; i <= 3; i++ {
		workflowID := "workflow" + string(rune('0'+i))
		if callCounts[workflowID] < 1 {
			t.Errorf("Expected workflow %s to be called at least once, got %d", workflowID, callCounts[workflowID])
		}
	}
}

// TestWorkflowScheduler_UnscheduleRemovesCallback tests that unscheduling removes callback
func TestWorkflowScheduler_UnscheduleRemovesCallback(t *testing.T) {
	ws := NewWorkflowScheduler()
	defer ws.Stop()

	workflowID := "test-workflow"
	callback := func(id string) error { return nil }

	ws.SetCallback(workflowID, callback)
	ws.ScheduleWorkflow(workflowID, "*/5 * * * * *")

	// Verify callback exists
	ws.mutex.RLock()
	_, exists := ws.callbacks[workflowID]
	ws.mutex.RUnlock()

	if !exists {
		t.Fatal("Expected callback to exist")
	}

	// Unschedule
	ws.UnscheduleWorkflow(workflowID)

	// Verify callback removed
	ws.mutex.RLock()
	_, exists = ws.callbacks[workflowID]
	ws.mutex.RUnlock()

	if exists {
		t.Error("Expected callback to be removed")
	}
}

// TestWorkflowScheduler_ConcurrentScheduling tests concurrent scheduling
func TestWorkflowScheduler_ConcurrentScheduling(t *testing.T) {
	ws := NewWorkflowScheduler()
	defer ws.Stop()

	var wg sync.WaitGroup
	workflowCount := 20

	for i := 0; i < workflowCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			workflowID := "workflow-" + string(rune('0'+id))
			schedule := "*/5 * * * * *"
			_ = ws.ScheduleWorkflow(workflowID, schedule)
		}(i)
	}

	wg.Wait()

	workflows := ws.GetScheduledWorkflows()
	if len(workflows) != workflowCount {
		t.Errorf("Expected %d scheduled workflows, got %d", workflowCount, len(workflows))
	}
}

// BenchmarkWorkflowScheduler_ScheduleWorkflow benchmarks workflow scheduling
func BenchmarkWorkflowScheduler_ScheduleWorkflow(b *testing.B) {
	ws := NewWorkflowScheduler()
	defer ws.Stop()

	schedule := "*/5 * * * * *"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		workflowID := "workflow-" + string(rune('0'+(i%10)))
		_ = ws.ScheduleWorkflow(workflowID, schedule)
	}
}

// BenchmarkWorkflowScheduler_GetScheduledWorkflows benchmarks getting scheduled workflows
func BenchmarkWorkflowScheduler_GetScheduledWorkflows(b *testing.B) {
	ws := NewWorkflowScheduler()
	defer ws.Stop()

	// Pre-populate with workflows
	schedule := "*/5 * * * * *"
	for i := 0; i < 100; i++ {
		workflowID := "workflow-" + string(rune('0'+(i%10)))
		_ = ws.ScheduleWorkflow(workflowID, schedule)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ws.GetScheduledWorkflows()
	}
}
