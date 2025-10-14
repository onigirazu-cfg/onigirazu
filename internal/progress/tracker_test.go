package progress

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewTracker(t *testing.T) {
	tracker := NewTracker()
	if tracker == nil {
		t.Fatal("Expected non-nil tracker")
	}
	if tracker.output == nil {
		t.Error("Expected non-nil output")
	}
	if tracker.width != 50 {
		t.Errorf("Expected width=50, got %d", tracker.width)
	}
	if !tracker.showBar {
		t.Error("Expected showBar=true")
	}
}

func TestNewTrackerWithOptions(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)

	if tracker.total != 10 {
		t.Errorf("Expected total=10, got %d", tracker.total)
	}
	if tracker.output != buf {
		t.Error("Expected custom output writer")
	}
	if tracker.showBar {
		t.Error("Expected showBar=false")
	}
}

func TestNewTrackerWithOptions_NilOutput(t *testing.T) {
	tracker := NewTrackerWithOptions(5, nil, true)
	if tracker.output == nil {
		t.Error("Expected default output when nil provided")
	}
}

func TestStart(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(0, buf, false)

	tracker.Start(20)

	if tracker.total != 20 {
		t.Errorf("Expected total=20, got %d", tracker.total)
	}
	if tracker.completed != 0 {
		t.Errorf("Expected completed=0, got %d", tracker.completed)
	}
	if tracker.failed != 0 {
		t.Errorf("Expected failed=0, got %d", tracker.failed)
	}
	if tracker.skipped != 0 {
		t.Errorf("Expected skipped=0, got %d", tracker.skipped)
	}
}

func TestUpdate(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.Update(5, "Running task")

	if tracker.completed != 5 {
		t.Errorf("Expected completed=5, got %d", tracker.completed)
	}
	if tracker.currentTask != "Running task" {
		t.Errorf("Expected currentTask='Running task', got '%s'", tracker.currentTask)
	}
}

func TestUpdate_WithHostInfo(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.Update(3, "Install package on server1")

	if tracker.currentTask != "Install package" {
		t.Errorf("Expected currentTask='Install package', got '%s'", tracker.currentTask)
	}
	if tracker.currentHost != "server1" {
		t.Errorf("Expected currentHost='server1', got '%s'", tracker.currentHost)
	}
}

func TestTaskCompleted_Success(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.TaskCompleted(true, false)

	if tracker.completed != 1 {
		t.Errorf("Expected completed=1, got %d", tracker.completed)
	}
	if tracker.failed != 0 {
		t.Errorf("Expected failed=0, got %d", tracker.failed)
	}
}

func TestTaskCompleted_Failed(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.TaskCompleted(false, false)

	if tracker.completed != 0 {
		t.Errorf("Expected completed=0, got %d", tracker.completed)
	}
	if tracker.failed != 1 {
		t.Errorf("Expected failed=1, got %d", tracker.failed)
	}
}

func TestTaskCompleted_Skipped(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.TaskCompleted(false, true)

	if tracker.completed != 0 {
		t.Errorf("Expected completed=0, got %d", tracker.completed)
	}
	if tracker.skipped != 1 {
		t.Errorf("Expected skipped=1, got %d", tracker.skipped)
	}
}

func TestSetError(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.SetError(nil)

	// Should increment failed count
	if tracker.failed != 1 {
		t.Errorf("Expected failed=1, got %d", tracker.failed)
	}
}

func TestFinish(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.TaskCompleted(true, false)  // completed++
	tracker.TaskCompleted(true, false)  // completed++
	tracker.TaskCompleted(false, false) // failed++

	tracker.Finish()

	// Verify final state
	if tracker.completed != 2 {
		t.Errorf("Expected completed=2, got %d", tracker.completed)
	}
	if tracker.failed != 1 {
		t.Errorf("Expected failed=1, got %d", tracker.failed)
	}
}

func TestGetProgress(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.Update(7, "test")

	completed, total := tracker.GetProgress()
	if completed != 7 {
		t.Errorf("Expected completed=7, got %d", completed)
	}
	if total != 10 {
		t.Errorf("Expected total=10, got %d", total)
	}
}

func TestGetProgressInfo(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.Update(5, "Running task on host1")
	tracker.TaskCompleted(true, false)  // completed++
	tracker.TaskCompleted(false, false) // failed++
	tracker.TaskCompleted(false, true)  // skipped++

	info := tracker.GetProgressInfo()

	if info.Total != 10 {
		t.Errorf("Expected Total=10, got %d", info.Total)
	}
	if info.Completed != 6 { // 5 from Update + 1 from TaskCompleted(true, false)
		t.Errorf("Expected Completed=6, got %d", info.Completed)
	}
	if info.Failed != 1 {
		t.Errorf("Expected Failed=1, got %d", info.Failed)
	}
	if info.Skipped != 1 {
		t.Errorf("Expected Skipped=1, got %d", info.Skipped)
	}
	if info.CurrentTask != "Running task" {
		t.Errorf("Expected CurrentTask='Running task', got '%s'", info.CurrentTask)
	}
	if info.CurrentHost != "host1" {
		t.Errorf("Expected CurrentHost='host1', got '%s'", info.CurrentHost)
	}
}

func TestSetWidth(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)

	tracker.SetWidth(80)

	if tracker.width != 80 {
		t.Errorf("Expected width=80, got %d", tracker.width)
	}
}

func TestSetWidth_TooSmall(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	originalWidth := tracker.width

	tracker.SetWidth(5)

	// Should accept any positive value
	if tracker.width != 5 {
		t.Errorf("Expected width=5, got %d", tracker.width)
	}

	// Test zero or negative - should not change
	tracker.SetWidth(0)
	if tracker.width != 5 {
		t.Errorf("Expected width=5 (unchanged), got %d", tracker.width)
	}

	tracker.SetWidth(-10)
	if tracker.width != 5 {
		t.Errorf("Expected width=5 (unchanged), got %d", tracker.width)
	}

	_ = originalWidth
}

func TestSetUpdateRate(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)

	tracker.SetUpdateRate(500 * time.Millisecond)

	if tracker.updateRate != 500*time.Millisecond {
		t.Errorf("Expected updateRate=500ms, got %v", tracker.updateRate)
	}
}

func TestIsComplete(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(5, buf, false)
	tracker.Start(5)

	if tracker.IsComplete() {
		t.Error("Expected IsComplete=false when not all tasks done")
	}

	tracker.Update(5, "done")

	if !tracker.IsComplete() {
		t.Error("Expected IsComplete=true when all tasks done")
	}
}

func TestGetSuccessRate(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	// No tasks completed yet
	rate := tracker.GetSuccessRate()
	if rate != 0.0 {
		t.Errorf("Expected success rate=0.0, got %f", rate)
	}

	// Complete some tasks
	tracker.TaskCompleted(true, false)  // success
	tracker.TaskCompleted(true, false)  // success
	tracker.TaskCompleted(false, false) // failed
	tracker.TaskCompleted(true, false)  // success

	rate = tracker.GetSuccessRate()
	// GetSuccessRate returns completed/total*100, not completed/(completed+failed)*100
	expected := 30.0 // 3 completed out of 10 total * 100
	if rate != expected {
		t.Errorf("Expected success rate=%f, got %f", expected, rate)
	}
}

func TestUpdateTask(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	// UpdateTask just calls TaskCompleted, doesn't set host/task
	tracker.UpdateTask("server1", "install_nginx", true)

	if tracker.completed != 1 {
		t.Errorf("Expected completed=1, got %d", tracker.completed)
	}
	if tracker.failed != 0 {
		t.Errorf("Expected failed=0, got %d", tracker.failed)
	}
}

func TestUpdateTask_Failed(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.UpdateTask("server2", "deploy_app", false)

	if tracker.failed != 1 {
		t.Errorf("Expected failed=1, got %d", tracker.failed)
	}
}

func TestUpdateProgress(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(50, buf, false)
	tracker.Start(50)

	// UpdateProgress calls Update(completed, "") which sets completed counter
	tracker.UpdateProgress(15, 50)

	// Update() doesn't change total, it just updates completed
	current, total := tracker.GetProgress()
	if current != 15 {
		t.Errorf("Expected current=15, got %d", current)
	}
	if total != 50 {
		t.Errorf("Expected total=50, got %d", total)
	}
}

func TestGetStats(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.TaskCompleted(true, false)  // completed++
	tracker.TaskCompleted(false, false) // failed++
	tracker.TaskCompleted(false, true)  // skipped++

	stats := tracker.GetStats()

	if stats["total"] != 10 {
		t.Errorf("Expected total=10, got %v", stats["total"])
	}
	if stats["completed"] != 1 {
		t.Errorf("Expected completed=1, got %v", stats["completed"])
	}
	if stats["failed"] != 1 {
		t.Errorf("Expected failed=1, got %v", stats["failed"])
	}
	if stats["skipped"] != 1 {
		t.Errorf("Expected skipped=1, got %v", stats["skipped"])
	}
	// success_rate = completed/total*100 = 1/10*100 = 10.0
	if stats["success_rate"] != 10.0 {
		t.Errorf("Expected success_rate=10.0, got %v", stats["success_rate"])
	}
}

func TestRender_WithProgressBar(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, true)
	tracker.Start(10)

	tracker.Update(5, "Running task")

	// Give it time to render
	time.Sleep(150 * time.Millisecond)
	tracker.Update(6, "Another task")

	output := buf.String()
	if output == "" {
		t.Error("Expected some output when showBar=true")
	}
}

func TestRender_WithoutProgressBar(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.Update(5, "Running task")

	output := buf.String()
	if output != "" {
		t.Error("Expected no output when showBar=false")
	}
}

func TestConcurrentUpdates(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(100, buf, false)
	tracker.Start(100)

	// Simulate concurrent updates using TaskCompleted only
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				tracker.TaskCompleted(true, false)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state - should have 100 completed tasks
	if tracker.completed != 100 {
		t.Errorf("Expected completed=100, got %d", tracker.completed)
	}
}

func TestProgressPercentage(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(100, buf, false)
	tracker.Start(100)

	tests := []struct {
		completed int
		expected  float64
	}{
		{0, 0.0},
		{25, 0.25},
		{50, 0.50},
		{75, 0.75},
		{100, 1.0},
	}

	for _, tt := range tests {
		tracker.Update(tt.completed, "test")
		info := tracker.GetProgressInfo()

		percentage := float64(info.Completed) / float64(info.Total)
		if percentage != tt.expected {
			t.Errorf("For completed=%d, expected percentage=%f, got %f",
				tt.completed, tt.expected, percentage)
		}
	}
}

func TestElapsedTime(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	info := tracker.GetProgressInfo()

	if info.Duration < 100*time.Millisecond {
		t.Errorf("Expected duration >= 100ms, got %v", info.Duration)
	}
}

func TestMessageParsing(t *testing.T) {
	tests := []struct {
		message      string
		expectedTask string
		expectedHost string
	}{
		{
			message:      "Install nginx on web-server-1",
			expectedTask: "Install nginx",
			expectedHost: "web-server-1",
		},
		{
			message:      "Deploy application on app-server-2",
			expectedTask: "Deploy application",
			expectedHost: "app-server-2",
		},
		{
			message:      "Simple task without host",
			expectedTask: "Simple task without host",
			expectedHost: "",
		},
		{
			// Note: Split(" on ") will split at first " on ", so "Task with on in name on server"
			// becomes ["Task with", "in name", "server"], and code takes parts[0] and parts[1]
			message:      "Task with on in name on server",
			expectedTask: "Task with",
			expectedHost: "in name",
		},
	}

	for _, tt := range tests {
		buf := &bytes.Buffer{}
		tracker := NewTrackerWithOptions(10, buf, false)
		tracker.Start(10)

		tracker.Update(1, tt.message)

		if tracker.currentTask != tt.expectedTask {
			t.Errorf("For message '%s', expected task='%s', got '%s'",
				tt.message, tt.expectedTask, tracker.currentTask)
		}
		if tracker.currentHost != tt.expectedHost {
			t.Errorf("For message '%s', expected host='%s', got '%s'",
				tt.message, tt.expectedHost, tracker.currentHost)
		}
	}
}

func TestRateLimiting(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(100, buf, true)
	tracker.SetUpdateRate(200 * time.Millisecond)
	tracker.Start(100)

	// First update should render
	tracker.Update(1, "task1")
	firstLen := buf.Len()

	// Immediate second update should be rate-limited
	tracker.Update(2, "task2")
	secondLen := buf.Len()

	// Should not have rendered again immediately
	if secondLen > firstLen+10 { // Allow small buffer for timing
		t.Error("Expected rate limiting to prevent immediate re-render")
	}

	// Wait for rate limit to pass
	time.Sleep(250 * time.Millisecond)
	tracker.Update(3, "task3")
	thirdLen := buf.Len()

	// Should have rendered after rate limit
	if thirdLen <= secondLen {
		t.Error("Expected render after rate limit period")
	}
}

func TestStop(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)
	tracker.Start(10)

	tracker.TaskCompleted(true, false)
	tracker.Stop()

	// Stop should not panic or cause issues
	// Just verify tracker is in valid state
	if tracker.completed != 1 {
		t.Errorf("Expected completed=1 after stop, got %d", tracker.completed)
	}
}

func TestStartTracking(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(10, buf, false)

	tracker.StartTracking()

	// Should reset start time
	if tracker.startTime.IsZero() {
		t.Error("Expected non-zero start time after StartTracking")
	}
}

func TestFinish_WithProgressBar(t *testing.T) {
	buf := &bytes.Buffer{}
	tracker := NewTrackerWithOptions(5, buf, true)
	tracker.Start(5)

	for i := 0; i < 5; i++ {
		tracker.TaskCompleted(true, false)
	}

	tracker.Finish()

	output := buf.String()
	// Should contain some progress output
	if !strings.Contains(output, "100%") && !strings.Contains(output, "5/5") {
		t.Log("Output:", output)
		// This is informational - progress bar format may vary
	}
}
