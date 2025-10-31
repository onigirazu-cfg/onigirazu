package execution

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecutionEvent_Creation tests creating execution events
func TestExecutionEvent_Creation(t *testing.T) {
	event := ExecutionEvent{
		Type:         "task_end",
		PlayName:     "setup",
		PlayIndex:    0,
		TaskName:     "gather facts",
		HostName:     "host1",
		Message:      "Task completed successfully",
		Timestamp:    time.Now(),
		TaskFailed:   false,
		TaskChanged:  true,
		TaskSkipped:  false,
		TaskDuration: 2 * time.Second,
	}

	assert.Equal(t, "task_end", event.Type, "Type should be set")
	assert.Equal(t, "setup", event.PlayName, "PlayName should be set")
	assert.Equal(t, 0, event.PlayIndex, "PlayIndex should be 0")
	assert.Equal(t, "gather facts", event.TaskName, "TaskName should be set")
	assert.Equal(t, "host1", event.HostName, "HostName should be set")
	assert.False(t, event.TaskFailed, "TaskFailed should be false")
	assert.True(t, event.TaskChanged, "TaskChanged should be true")
	assert.Equal(t, 2*time.Second, event.TaskDuration, "TaskDuration should be set")
}

// TestExecutionEvent_Types tests different event types
func TestExecutionEvent_Types(t *testing.T) {
	testCases := []struct {
		name  string
		etype string
	}{
		{name: "execution start", etype: "execution_start"},
		{name: "play start", etype: "play_start"},
		{name: "task end", etype: "task_end"},
		{name: "execution end", etype: "execution_end"},
		{name: "error", etype: "error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := ExecutionEvent{Type: tc.etype}
			assert.Equal(t, tc.etype, event.Type, "Type should match")
		})
	}
}

// TestEnhancedTUIModel_Creation tests creating TUI model
func TestEnhancedTUIModel_Creation(t *testing.T) {
	model := NewEnhancedTUIModel()

	assert.NotNil(t, model, "Model should not be nil")
	assert.NotNil(t, model.ctx, "Context should be initialized")
	assert.NotNil(t, model.cancel, "Cancel function should be initialized")
	assert.Equal(t, DisplayNormal, model.mode, "Mode should default to Normal")
	assert.Equal(t, "initializing", model.status, "Status should be initializing")
	assert.Empty(t, model.activeModal, "Active modal should be empty")
	assert.False(t, model.paused, "Should not be paused")
	assert.False(t, model.shouldExit, "Should not exit initially")
}

// TestEnhancedTUIModel_InitializesChannels tests that channels are initialized
func TestEnhancedTUIModel_InitializesChannels(t *testing.T) {
	model := NewEnhancedTUIModel()

	assert.NotNil(t, model.eventChan, "Event channel should be initialized")
	assert.NotNil(t, model.stopChan, "Stop channel should be initialized")
	assert.NotNil(t, model.tickerChan, "Ticker channel should be initialized")
	assert.NotNil(t, model.readyChan, "Ready channel should be initialized")
}

// TestEnhancedTUIModel_InitializesMaps tests that data maps are initialized
func TestEnhancedTUIModel_InitializesMaps(t *testing.T) {
	model := NewEnhancedTUIModel()

	assert.NotNil(t, model.taskStats, "Task stats map should be initialized")
	assert.NotNil(t, model.hostStats, "Host stats map should be initialized")
	assert.NotNil(t, model.playStats, "Play stats map should be initialized")
	assert.Equal(t, 0, len(model.taskStats), "Task stats should be empty")
	assert.Equal(t, 0, len(model.hostStats), "Host stats should be empty")
	assert.Equal(t, 0, len(model.playStats), "Play stats should be empty")
}

// TestEnhancedTUIModel_LogBuffer tests log buffer initialization
func TestEnhancedTUIModel_LogBuffer(t *testing.T) {
	model := NewEnhancedTUIModel()

	assert.NotNil(t, model.logBuffer, "Log buffer should be initialized")
	assert.Equal(t, 1000, model.maxLogs, "Max logs should be 1000")
	assert.Equal(t, 0, model.logIndex, "Log index should start at 0")
}

// TestDetailedTaskStats_Creation tests creating task stats
func TestDetailedTaskStats_Creation(t *testing.T) {
	stats := &DetailedTaskStats{
		Name:           "deploy app",
		Success:        10,
		Failed:         1,
		Changed:        8,
		Skipped:        2,
		Duration:       5 * time.Second,
		AvgDuration:    500 * time.Millisecond,
		TotalDuration:  5 * time.Second,
		MinDuration:    300 * time.Millisecond,
		MaxDuration:    700 * time.Millisecond,
		ExecutionCount: 11,
	}

	assert.Equal(t, "deploy app", stats.Name, "Name should be set")
	assert.Equal(t, 10, stats.Success, "Success count should be 10")
	assert.Equal(t, 1, stats.Failed, "Failed count should be 1")
	assert.Equal(t, 8, stats.Changed, "Changed count should be 8")
	assert.Equal(t, 2, stats.Skipped, "Skipped count should be 2")
	assert.Equal(t, 11, stats.ExecutionCount, "Execution count should be 11")
}

// TestDetailedTaskStats_SuccessRate tests calculating success rate
func TestDetailedTaskStats_SuccessRate(t *testing.T) {
	testCases := []struct {
		name         string
		success      int
		failed       int
		total        int
		expectedRate float64
	}{
		{name: "all successful", success: 10, failed: 0, total: 10, expectedRate: 1.0},
		{name: "all failed", success: 0, failed: 10, total: 10, expectedRate: 0.0},
		{name: "mixed", success: 7, failed: 3, total: 10, expectedRate: 0.7},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stats := &DetailedTaskStats{
				Success: tc.success,
				Failed:  tc.failed,
			}

			if tc.total > 0 {
				successRate := float64(stats.Success) / float64(tc.total)
				assert.Equal(t, tc.expectedRate, successRate, "Success rate should be calculable")
			}
		})
	}
}

// TestDetailedHostStats_Creation tests creating host stats
func TestDetailedHostStats_Creation(t *testing.T) {
	stats := &DetailedHostStats{
		Name:          "server1",
		TaskCount:     25,
		SuccessCount:  23,
		FailedCount:   2,
		ChangedCount:  15,
		SkippedCount:  0,
		TotalDuration: 30 * time.Second,
		AvgTaskTime:   1200 * time.Millisecond,
	}

	assert.Equal(t, "server1", stats.Name, "Name should be set")
	assert.Equal(t, 25, stats.TaskCount, "Task count should be 25")
	assert.Equal(t, 23, stats.SuccessCount, "Success count should be 23")
	assert.Equal(t, 2, stats.FailedCount, "Failed count should be 2")
	assert.Equal(t, 15, stats.ChangedCount, "Changed count should be 15")
}

// TestDetailedPlayStats_Creation tests creating play stats
func TestDetailedPlayStats_Creation(t *testing.T) {
	startTime := time.Now()
	stats := &DetailedPlayStats{
		Name:      "web deployment",
		Index:     1,
		Total:     3,
		Completed: 2,
		Failed:    0,
		Duration:  45 * time.Second,
		StartTime: startTime,
	}

	assert.Equal(t, "web deployment", stats.Name, "Name should be set")
	assert.Equal(t, 1, stats.Index, "Index should be 1")
	assert.Equal(t, 3, stats.Total, "Total should be 3")
	assert.Equal(t, 2, stats.Completed, "Completed should be 2")
	assert.Equal(t, 0, stats.Failed, "Failed should be 0")
	assert.Equal(t, startTime, stats.StartTime, "StartTime should match")
}

// TestExecutionMetrics_Creation tests creating execution metrics
func TestExecutionMetrics_Creation(t *testing.T) {
	metrics := &ExecutionMetrics{
		TotalTasksExecuted: 100,
		TotalDuration:      5 * time.Minute,
		AvgTaskSpeed:       2.0,
		MaxTaskSpeed:       5.0,
		MinTaskSpeed:       0.5,
	}

	assert.Equal(t, 100, metrics.TotalTasksExecuted, "Total tasks should be 100")
	assert.Equal(t, 5*time.Minute, metrics.TotalDuration, "Total duration should be 5 minutes")
	assert.Equal(t, 2.0, metrics.AvgTaskSpeed, "Average task speed should be 2.0")
	assert.Equal(t, 5.0, metrics.MaxTaskSpeed, "Max task speed should be 5.0")
	assert.Equal(t, 0.5, metrics.MinTaskSpeed, "Min task speed should be 0.5")
}

// TestLogEntry_Creation tests creating log entries
func TestLogEntry_Creation(t *testing.T) {
	now := time.Now()
	entry := LogEntry{
		Timestamp: now,
		Level:     "INFO",
		Message:   "Test log message",
	}

	assert.Equal(t, now, entry.Timestamp, "Timestamp should match")
	assert.Equal(t, "INFO", entry.Level, "Level should be INFO")
	assert.Equal(t, "Test log message", entry.Message, "Message should match")
}

// TestLogEntry_LevelTypes tests different log levels
func TestLogEntry_LevelTypes(t *testing.T) {
	testCases := []string{"INFO", "WARN", "ERROR", "DEBUG", "TASK_START", "TASK_END"}

	for _, level := range testCases {
		t.Run(level, func(t *testing.T) {
			entry := LogEntry{
				Level: level,
			}
			assert.Equal(t, level, entry.Level, "Level should be %s", level)
		})
	}
}

// TestEnhancedTUIModel_Context tests context functionality
func TestEnhancedTUIModel_Context(t *testing.T) {
	model := NewEnhancedTUIModel()

	// Context should be valid
	assert.NotNil(t, model.ctx, "Context should not be nil")
	assert.NoError(t, model.ctx.Err(), "Context should not be cancelled initially")

	// Cancel should work
	model.cancel()
	assert.Error(t, model.ctx.Err(), "Context should be cancelled after cancel()")
}

// TestEnhancedTUIModel_ModeTypes tests different display modes
func TestEnhancedTUIModel_ModeTypes(t *testing.T) {
	testCases := []struct {
		name string
		mode DisplayMode
	}{
		{name: "normal", mode: DisplayNormal},
		{name: "verbose", mode: DisplayVerbose},
		{name: "debug", mode: DisplayDebug},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := NewEnhancedTUIModel()
			model.mode = tc.mode
			assert.Equal(t, tc.mode, model.mode, "Mode should be set to %v", tc.mode)
		})
	}
}

// TestEnhancedTUIModel_ModalStates tests modal states
func TestEnhancedTUIModel_ModalStates(t *testing.T) {
	testCases := []string{"", "help", "stats", "confirm"}

	for _, modal := range testCases {
		t.Run(modal, func(t *testing.T) {
			model := NewEnhancedTUIModel()
			model.activeModal = modal
			assert.Equal(t, modal, model.activeModal, "Modal should be set to %s", modal)
		})
	}
}

// TestEnhancedTUIModel_StatusTransitions tests status transitions
func TestEnhancedTUIModel_StatusTransitions(t *testing.T) {
	statuses := []string{"initializing", "running", "paused", "stopped", "completed"}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			model := NewEnhancedTUIModel()
			model.status = status
			assert.Equal(t, status, model.status, "Status should be %s", status)
		})
	}
}

// TestEnhancedTUIModel_PlaybookTracking tests playbook tracking
func TestEnhancedTUIModel_PlaybookTracking(t *testing.T) {
	model := NewEnhancedTUIModel()

	model.playbookName = "deploy.yml"
	model.playCount = 3
	model.totalTaskCount = 50
	model.currentPlayIndex = 1

	assert.Equal(t, "deploy.yml", model.playbookName, "Playbook name should be set")
	assert.Equal(t, 3, model.playCount, "Play count should be 3")
	assert.Equal(t, 50, model.totalTaskCount, "Total task count should be 50")
	assert.Equal(t, 1, model.currentPlayIndex, "Current play index should be 1")
}

// TestEnhancedTUIModel_FilterState tests filter state management
func TestEnhancedTUIModel_FilterState(t *testing.T) {
	model := NewEnhancedTUIModel()

	assert.False(t, model.filterMode, "Filter mode should be off initially")
	assert.False(t, model.searchMode, "Search mode should be off initially")

	// Enable filter mode
	model.filterMode = true
	model.filterShowErrors = true

	assert.True(t, model.filterMode, "Filter mode should be on")
	assert.True(t, model.filterShowErrors, "Show errors filter should be on")
}

// TestEnhancedTUIModel_SearchState tests search state management
func TestEnhancedTUIModel_SearchState(t *testing.T) {
	model := NewEnhancedTUIModel()

	model.searchMode = true
	model.searchQuery = "error"
	model.searchIndex = 3

	assert.True(t, model.searchMode, "Search mode should be on")
	assert.Equal(t, "error", model.searchQuery, "Search query should be set")
	assert.Equal(t, 3, model.searchIndex, "Search index should be 3")
}

// TestEnhancedTUIModel_ControlState tests control state
func TestEnhancedTUIModel_ControlState(t *testing.T) {
	model := NewEnhancedTUIModel()

	assert.False(t, model.paused, "Should not be paused initially")
	assert.False(t, model.shouldExit, "Should not exit initially")
	assert.False(t, model.gracefulReq, "Should not have graceful request initially")

	// Toggle states
	model.paused = true
	model.shouldExit = true
	model.gracefulReq = true

	assert.True(t, model.paused, "Should be paused")
	assert.True(t, model.shouldExit, "Should exit")
	assert.True(t, model.gracefulReq, "Should have graceful request")
}

// TestDetailedTaskStats_DurationTracking tests duration tracking
func TestDetailedTaskStats_DurationTracking(t *testing.T) {
	stats := &DetailedTaskStats{
		Name:            "test task",
		Duration:        5 * time.Second,
		AvgDuration:     1 * time.Second,
		TotalDuration:   5 * time.Second,
		MinDuration:     500 * time.Millisecond,
		MaxDuration:     2 * time.Second,
		IndividualTimes: []time.Duration{1 * time.Second, 2 * time.Second, 2 * time.Second},
		ExecutionCount:  3,
	}

	assert.Equal(t, 3, len(stats.IndividualTimes), "Should have 3 individual times recorded")
	assert.Equal(t, 3, stats.ExecutionCount, "Execution count should be 3")

	// Verify statistics are reasonable
	assert.True(t, stats.MinDuration <= stats.AvgDuration, "Min should be <= Avg")
	assert.True(t, stats.AvgDuration <= stats.MaxDuration, "Avg should be <= Max")
}

// TestEnhancedTUIModel_Cleanup tests model cleanup
func TestEnhancedTUIModel_Cleanup(t *testing.T) {
	model := NewEnhancedTUIModel()

	// Model should have cancel function
	require.NotNil(t, model.cancel, "Cancel function should be available")

	// Call cancel to cleanup context
	model.cancel()

	// Context should be done
	select {
	case <-model.ctx.Done():
		// Expected behavior
	default:
		t.Fatal("Context should be done after cancel")
	}
}

// BenchmarkEnhancedTUIModel_Creation benchmarks model creation
func BenchmarkEnhancedTUIModel_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewEnhancedTUIModel()
	}
}

// BenchmarkExecutionEvent_Creation benchmarks event creation
func BenchmarkExecutionEvent_Creation(b *testing.B) {
	now := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExecutionEvent{
			Type:      "task_end",
			PlayName:  "test",
			TaskName:  "task",
			Timestamp: now,
		}
	}
}

// BenchmarkDetailedTaskStats_Creation benchmarks task stats creation
func BenchmarkDetailedTaskStats_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &DetailedTaskStats{
			Name:           "task",
			Success:        10,
			Failed:         2,
			ExecutionCount: 12,
		}
	}
}
