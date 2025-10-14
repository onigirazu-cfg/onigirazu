//go:build plugin
// +build plugin

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/plugins"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// MetricsCallback is a callback plugin that collects execution metrics
type MetricsCallback struct {
	*plugins.BaseCallbackPlugin
	mu             sync.RWMutex
	tasksStarted   int
	tasksCompleted int
	tasksSucceeded int
	tasksFailed    int
	totalDuration  time.Duration
	taskDurations  map[string]time.Duration
	taskStartTimes map[string]time.Time
	playbookStart  time.Time
}

// NewPlugin is the entry point for the plugin
func NewPlugin() plugins.Plugin {
	return &MetricsCallback{
		BaseCallbackPlugin: plugins.NewBaseCallbackPlugin(
			"metrics",
			"1.0.0",
			"Collects and reports execution metrics",
		),
		taskDurations:  make(map[string]time.Duration),
		taskStartTimes: make(map[string]time.Time),
	}
}

// Initialize initializes the plugin
func (c *MetricsCallback) Initialize(ctx context.Context, config map[string]interface{}) error {
	// Reset metrics
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tasksStarted = 0
	c.tasksCompleted = 0
	c.tasksSucceeded = 0
	c.tasksFailed = 0
	c.totalDuration = 0
	c.taskDurations = make(map[string]time.Duration)
	c.taskStartTimes = make(map[string]time.Time)

	return nil
}

// OnPlaybookStart is called when playbook execution starts
func (c *MetricsCallback) OnPlaybookStart(ctx context.Context, playbook *types.Playbook) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.playbookStart = time.Now()
	fmt.Printf("\n=== Playbook Execution Started ===\n")
	fmt.Printf("Playbook: %s\n", playbook.Name)
	fmt.Printf("Plays: %d\n\n", len(playbook.Plays))

	return nil
}

// OnPlaybookEnd is called when playbook execution ends
func (c *MetricsCallback) OnPlaybookEnd(ctx context.Context, playbook *types.Playbook, success bool, duration time.Duration) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fmt.Printf("\n=== Playbook Execution Completed ===\n")
	fmt.Printf("Status: ")
	if success {
		fmt.Printf("SUCCESS\n")
	} else {
		fmt.Printf("FAILED\n")
	}
	fmt.Printf("Duration: %s\n", duration)
	fmt.Printf("\n=== Execution Metrics ===\n")
	fmt.Printf("Tasks Started:    %d\n", c.tasksStarted)
	fmt.Printf("Tasks Completed:  %d\n", c.tasksCompleted)
	fmt.Printf("Tasks Succeeded:  %d\n", c.tasksSucceeded)
	fmt.Printf("Tasks Failed:     %d\n", c.tasksFailed)
	fmt.Printf("Total Duration:   %s\n", c.totalDuration)

	if len(c.taskDurations) > 0 {
		fmt.Printf("\n=== Task Durations ===\n")
		for taskName, duration := range c.taskDurations {
			fmt.Printf("  %s: %s\n", taskName, duration)
		}
	}

	// Calculate average duration
	if c.tasksCompleted > 0 {
		avgDuration := c.totalDuration / time.Duration(c.tasksCompleted)
		fmt.Printf("\nAverage Task Duration: %s\n", avgDuration)
	}

	// Calculate success rate
	if c.tasksCompleted > 0 {
		successRate := float64(c.tasksSucceeded) / float64(c.tasksCompleted) * 100
		fmt.Printf("Success Rate: %.2f%%\n", successRate)
	}

	fmt.Printf("\n")

	return nil
}

// OnPlayStart is called when play execution starts
func (c *MetricsCallback) OnPlayStart(ctx context.Context, play *types.Play) error {
	fmt.Printf("\n--- Play: %s ---\n", play.Name)
	return nil
}

// OnPlayEnd is called when play execution ends
func (c *MetricsCallback) OnPlayEnd(ctx context.Context, play *types.Play, success bool, duration time.Duration) error {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}
	fmt.Printf("--- Play Completed: %s (Duration: %s) ---\n", status, duration)
	return nil
}

// OnTaskStart is called when task execution starts
func (c *MetricsCallback) OnTaskStart(ctx context.Context, task *types.Task, host types.Host) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tasksStarted++
	taskKey := fmt.Sprintf("%s@%s", task.Name, host.Name)
	c.taskStartTimes[taskKey] = time.Now()

	return nil
}

// OnTaskEnd is called when task execution ends
func (c *MetricsCallback) OnTaskEnd(ctx context.Context, task *types.Task, host types.Host, result types.TaskResult) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tasksCompleted++

	if result.Success {
		c.tasksSucceeded++
	} else {
		c.tasksFailed++
	}

	// Calculate task duration
	taskKey := fmt.Sprintf("%s@%s", task.Name, host.Name)
	if startTime, exists := c.taskStartTimes[taskKey]; exists {
		duration := time.Since(startTime)
		c.taskDurations[taskKey] = duration
		c.totalDuration += duration
		delete(c.taskStartTimes, taskKey)
	}

	return nil
}

// OnTaskRetry is called when task is retried
func (c *MetricsCallback) OnTaskRetry(ctx context.Context, task *types.Task, host types.Host, attempt int, err error) error {
	fmt.Printf("  [RETRY] Task: %s on %s (Attempt: %d, Error: %v)\n", task.Name, host.Name, attempt, err)
	return nil
}

// GetMetrics returns current metrics (custom method for this plugin)
func (c *MetricsCallback) GetMetrics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"tasks_started":   c.tasksStarted,
		"tasks_completed": c.tasksCompleted,
		"tasks_succeeded": c.tasksSucceeded,
		"tasks_failed":    c.tasksFailed,
		"total_duration":  c.totalDuration.String(),
		"task_durations":  c.taskDurations,
	}
}
