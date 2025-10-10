package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Tracker provides progress tracking and reporting
type Tracker struct {
	total       int
	completed   int
	failed      int
	skipped     int
	currentTask string
	currentHost string
	startTime   time.Time
	mutex       sync.RWMutex
	output      io.Writer
	showBar     bool
	width       int
	lastUpdate  time.Time
	updateRate  time.Duration
}

// NewTracker creates a new progress tracker with default settings
func NewTracker() *Tracker {
	return NewTrackerWithOptions(0, os.Stdout, true)
}

// NewTrackerWithOptions creates a new progress tracker with custom options
func NewTrackerWithOptions(total int, output io.Writer, showBar bool) *Tracker {
	if output == nil {
		output = os.Stdout
	}

	return &Tracker{
		total:      total,
		output:     output,
		showBar:    showBar,
		width:      50,
		startTime:  time.Now(),
		updateRate: 100 * time.Millisecond, // Limit updates to avoid flickering
	}
}

// Start initializes the progress tracker
func (t *Tracker) Start(total int) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.total = total
	t.completed = 0
	t.failed = 0
	t.skipped = 0
	t.startTime = time.Now()
	t.lastUpdate = time.Time{}

	if t.showBar {
		t.render()
	}
}

// Update updates the progress with completion count and current task info
func (t *Tracker) Update(completed int, message string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.completed = completed

	// Parse message to extract task and host info
	if strings.Contains(message, " on ") {
		parts := strings.Split(message, " on ")
		if len(parts) >= 2 {
			t.currentTask = strings.TrimSpace(parts[0])
			t.currentHost = strings.TrimSpace(parts[1])
		}
	} else {
		t.currentTask = message
	}

	// Rate limit updates
	now := time.Now()
	if now.Sub(t.lastUpdate) < t.updateRate {
		return
	}
	t.lastUpdate = now

	if t.showBar {
		t.render()
	}
}

// TaskCompleted marks a task as completed
func (t *Tracker) TaskCompleted(success bool, skipped bool) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if skipped {
		t.skipped++
	} else if success {
		t.completed++
	} else {
		t.failed++
	}

	if t.showBar {
		t.render()
	}
}

// SetError sets an error state
func (t *Tracker) SetError(err error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.failed++

	if t.showBar {
		fmt.Fprintf(t.output, "\nError: %v\n", err)
		t.render()
	}
}

// Finish completes the progress tracking
func (t *Tracker) Finish() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.showBar {
		t.render()
		fmt.Fprintf(t.output, "\n")
	}

	// Print summary
	duration := time.Since(t.startTime)
	fmt.Fprintf(t.output, "\nExecution Summary:\n")
	fmt.Fprintf(t.output, "  Total tasks: %d\n", t.total)
	fmt.Fprintf(t.output, "  Completed:   %d\n", t.completed)
	fmt.Fprintf(t.output, "  Failed:      %d\n", t.failed)
	fmt.Fprintf(t.output, "  Skipped:     %d\n", t.skipped)
	fmt.Fprintf(t.output, "  Duration:    %v\n", duration.Round(time.Millisecond))

	if t.total > 0 {
		successRate := float64(t.completed) / float64(t.total) * 100
		fmt.Fprintf(t.output, "  Success rate: %.1f%%\n", successRate)
	}
}

// render draws the progress bar
func (t *Tracker) render() {
	if t.total == 0 {
		return
	}

	// Calculate progress
	totalProcessed := t.completed + t.failed + t.skipped
	percentage := float64(totalProcessed) / float64(t.total) * 100

	// Create progress bar
	filled := int(float64(t.width) * float64(totalProcessed) / float64(t.total))
	if filled > t.width {
		filled = t.width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", t.width-filled)

	// Calculate ETA
	duration := time.Since(t.startTime)
	var eta string
	if totalProcessed > 0 && totalProcessed < t.total {
		avgTimePerTask := duration / time.Duration(totalProcessed)
		remaining := time.Duration(t.total-totalProcessed) * avgTimePerTask
		eta = fmt.Sprintf(" ETA: %v", remaining.Round(time.Second))
	}

	// Format current task info
	taskInfo := ""
	if t.currentTask != "" {
		taskInfo = fmt.Sprintf(" | %s", t.currentTask)
		if t.currentHost != "" {
			taskInfo += fmt.Sprintf(" on %s", t.currentHost)
		}
	}

	// Clear line and print progress
	fmt.Fprintf(t.output, "\r\033[K[%s] %d/%d (%.1f%%)%s%s",
		bar, totalProcessed, t.total, percentage, eta, taskInfo)
}

// GetProgress returns current progress information (interface method)
func (t *Tracker) GetProgress() (int, int) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.completed + t.failed + t.skipped, t.total
}

// GetProgressInfo returns detailed progress information
func (t *Tracker) GetProgressInfo() types.ProgressInfo {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return types.ProgressInfo{
		Total:       t.total,
		Completed:   t.completed,
		Failed:      t.failed,
		Skipped:     t.skipped,
		CurrentTask: t.currentTask,
		CurrentHost: t.currentHost,
		StartTime:   t.startTime,
		Duration:    time.Since(t.startTime),
	}
}

// SetWidth sets the progress bar width
func (t *Tracker) SetWidth(width int) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if width > 0 {
		t.width = width
	}
}

// SetUpdateRate sets the minimum time between updates
func (t *Tracker) SetUpdateRate(rate time.Duration) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.updateRate = rate
}

// IsComplete returns true if all tasks are processed
func (t *Tracker) IsComplete() bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return (t.completed + t.failed + t.skipped) >= t.total
}

// GetSuccessRate returns the success rate as a percentage
func (t *Tracker) GetSuccessRate() float64 {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	if t.total == 0 {
		return 0
	}

	return float64(t.completed) / float64(t.total) * 100
}

// StartTracking starts the progress tracker without parameters (interface method)
func (t *Tracker) StartTracking() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.completed = 0
	t.failed = 0
	t.skipped = 0
	t.startTime = time.Now()
	t.lastUpdate = time.Time{}

	if t.showBar {
		t.render()
	}
}

// Stop stops the progress tracker (interface method)
func (t *Tracker) Stop() {
	t.Finish()
}

// UpdateTask updates task progress (interface method)
func (t *Tracker) UpdateTask(hostName, taskName string, success bool) {
	t.TaskCompleted(success, false)
}

// UpdateProgress updates progress (interface method)
func (t *Tracker) UpdateProgress(completed, total int) {
	t.Update(completed, "")
}

// GetStats returns progress statistics (interface method)
func (t *Tracker) GetStats() map[string]interface{} {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return map[string]interface{}{
		"total":        t.total,
		"completed":    t.completed,
		"failed":       t.failed,
		"skipped":      t.skipped,
		"current_task": t.currentTask,
		"current_host": t.currentHost,
		"start_time":   t.startTime,
		"duration":     time.Since(t.startTime),
		"success_rate": t.GetSuccessRate(),
	}
}

// MultiTracker manages multiple progress trackers for different plays/hosts
type MultiTracker struct {
	trackers map[string]*Tracker
	mutex    sync.RWMutex
	output   io.Writer
}

// NewMultiTracker creates a new multi-tracker
func NewMultiTracker(output io.Writer) *MultiTracker {
	if output == nil {
		output = os.Stdout
	}

	return &MultiTracker{
		trackers: make(map[string]*Tracker),
		output:   output,
	}
}

// AddTracker adds a new tracker for a specific context
func (mt *MultiTracker) AddTracker(name string, total int, showBar bool) *Tracker {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()

	tracker := NewTrackerWithOptions(total, mt.output, showBar)
	mt.trackers[name] = tracker
	return tracker
}

// GetTracker retrieves a tracker by name
func (mt *MultiTracker) GetTracker(name string) (*Tracker, bool) {
	mt.mutex.RLock()
	defer mt.mutex.RUnlock()

	tracker, exists := mt.trackers[name]
	return tracker, exists
}

// RemoveTracker removes a tracker
func (mt *MultiTracker) RemoveTracker(name string) {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()

	delete(mt.trackers, name)
}

// GetOverallProgress returns combined progress from all trackers
func (mt *MultiTracker) GetOverallProgress() types.ProgressInfo {
	mt.mutex.RLock()
	defer mt.mutex.RUnlock()

	var total, completed, failed, skipped int
	var earliestStart time.Time

	for _, tracker := range mt.trackers {
		progress := tracker.GetProgressInfo()
		total += progress.Total
		completed += progress.Completed
		failed += progress.Failed
		skipped += progress.Skipped

		if earliestStart.IsZero() || progress.StartTime.Before(earliestStart) {
			earliestStart = progress.StartTime
		}
	}

	return types.ProgressInfo{
		Total:     total,
		Completed: completed,
		Failed:    failed,
		Skipped:   skipped,
		StartTime: earliestStart,
		Duration:  time.Since(earliestStart),
	}
}
