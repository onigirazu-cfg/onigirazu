package progress

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProgressRenderer_RenderBatchProgress(t *testing.T) {
	renderer := NewProgressRenderer(true) // No color for testing

	tasks := []HostTaskInfo{
		{
			Host:      "host1.example.com",
			Task:      "Install nginx",
			StartTime: time.Now().Add(-5 * time.Second),
			Status:    "running",
		},
		{
			Host:      "host2.example.com",
			Task:      "Configure web server",
			StartTime: time.Now().Add(-3 * time.Second),
			Status:    "running",
		},
	}

	output := renderer.RenderBatchProgress(15, 20, tasks, 30*time.Second)

	// Check progress bar is present
	assert.Contains(t, output, "15/20")
	assert.Contains(t, output, "75.0%")
	assert.Contains(t, output, "elapsed")

	// Check current tasks are displayed
	assert.Contains(t, output, "Currently Running")
	assert.Contains(t, output, "host1")
	assert.Contains(t, output, "host2")
	assert.Contains(t, output, "Install nginx")
	assert.Contains(t, output, "Configure web server")
}

func TestProgressRenderer_RenderSummaryLine(t *testing.T) {
	renderer := NewProgressRenderer(true)

	output := renderer.RenderSummaryLine(20, 18, 1, 1, 2*time.Minute)

	assert.Contains(t, output, "20 total")
	assert.Contains(t, output, "18 success")
	assert.Contains(t, output, "1 failed")
	assert.Contains(t, output, "1 changed")
	assert.Contains(t, output, "2m")
}

func TestProgressRenderer_ProgressBar_Empty(t *testing.T) {
	renderer := NewProgressRenderer(true)

	output := renderer.RenderBatchProgress(0, 10, []HostTaskInfo{}, 0)

	// Should show empty progress bar
	assert.Contains(t, output, "0/10")
	assert.Contains(t, output, "0.0%")
	assert.Contains(t, output, "░") // Empty bar characters
}

func TestProgressRenderer_ProgressBar_Full(t *testing.T) {
	renderer := NewProgressRenderer(true)

	output := renderer.RenderBatchProgress(10, 10, []HostTaskInfo{}, 0)

	// Should show full progress bar
	assert.Contains(t, output, "10/10")
	assert.Contains(t, output, "100.0%")
	assert.Contains(t, output, "█") // Filled bar characters
}

func TestProgressRenderer_Long_Hostname_Truncation(t *testing.T) {
	renderer := NewProgressRenderer(true)

	tasks := []HostTaskInfo{
		{
			Host:      "very-long-hostname-that-exceeds-limit.example.com",
			Task:      "Some task",
			StartTime: time.Now(),
			Status:    "running",
		},
	}

	output := renderer.RenderBatchProgress(1, 2, tasks, 5*time.Second)

	// Hostname should be truncated (max 20 chars)
	assert.Contains(t, output, "Currently Running")
	// Should not contain the full long hostname
	assert.NotContains(t, output, "very-long-hostname-that-exceeds-limit.example.com")
}

func TestProgressRenderer_NoTasks_Display(t *testing.T) {
	renderer := NewProgressRenderer(true)

	output := renderer.RenderBatchProgress(5, 10, []HostTaskInfo{}, 5*time.Second)

	// Should not show "Currently Running" if there are no tasks
	assert.NotContains(t, output, "Currently Running")
	// But should show the progress bar
	assert.Contains(t, output, "5/10")
}

func TestProgressRenderer_Percentage_Calculation(t *testing.T) {
	renderer := NewProgressRenderer(true)

	tests := []struct {
		completed int
		total     int
		expected  string
	}{
		{0, 10, "0.0%"},
		{5, 10, "50.0%"},
		{1, 3, "33.3%"},
		{10, 10, "100.0%"},
	}

	for _, test := range tests {
		output := renderer.RenderBatchProgress(test.completed, test.total, []HostTaskInfo{}, 0)
		assert.Contains(t, output, test.expected)
	}
}

func TestProgressRenderer_Duration_Display(t *testing.T) {
	renderer := NewProgressRenderer(true)

	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{1 * time.Minute, "1m"},
		{90 * time.Second, "1m"},
	}

	for _, test := range tests {
		output := renderer.RenderBatchProgress(5, 10, []HostTaskInfo{}, test.duration)
		assert.Contains(t, output, "elapsed")
		// Should contain rounded duration
		assert.True(t, strings.Contains(output, "s") || strings.Contains(output, "m"))
	}
}

func TestProgressRenderer_Multiple_Tasks(t *testing.T) {
	renderer := NewProgressRenderer(true)

	tasks := make([]HostTaskInfo, 5)
	for i := 0; i < 5; i++ {
		tasks[i] = HostTaskInfo{
			Host:      "host" + string(rune(49+i)) + ".example.com",
			Task:      "Task " + string(rune(49+i)),
			StartTime: time.Now().Add(-time.Duration(i*1000) * time.Millisecond),
			Status:    "running",
		}
	}

	output := renderer.RenderBatchProgress(10, 20, tasks, 10*time.Second)

	assert.Contains(t, output, "host1")
	assert.Contains(t, output, "host5")
	assert.Contains(t, output, "Task 1")
	assert.Contains(t, output, "Task 5")
}

func TestProgressRenderer_Color_Output(t *testing.T) {
	renderer := NewProgressRenderer(false) // With color

	output := renderer.RenderSummaryLine(10, 8, 1, 1, 1*time.Minute)

	// Should contain the output data regardless of color
	assert.Contains(t, output, "10 total")
	assert.Contains(t, output, "8 success")
	assert.Contains(t, output, "1 failed")
}
