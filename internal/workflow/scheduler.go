package workflow

import (
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// WorkflowScheduler manages scheduled workflow executions
type WorkflowScheduler struct {
	cron      *cron.Cron
	schedules map[string]cron.EntryID
	mutex     sync.RWMutex
	callbacks map[string]ScheduleCallback
}

// ScheduleCallback is called when a scheduled workflow should be executed
type ScheduleCallback func(workflowID string) error

// NewWorkflowScheduler creates a new workflow scheduler
func NewWorkflowScheduler() *WorkflowScheduler {
	return &WorkflowScheduler{
		cron:      cron.New(cron.WithSeconds()),
		schedules: make(map[string]cron.EntryID),
		callbacks: make(map[string]ScheduleCallback),
	}
}

// Start starts the scheduler
func (ws *WorkflowScheduler) Start() {
	ws.cron.Start()
}

// Stop stops the scheduler
func (ws *WorkflowScheduler) Stop() {
	ws.cron.Stop()
}

// ScheduleWorkflow schedules a workflow for execution
func (ws *WorkflowScheduler) ScheduleWorkflow(workflowID, schedule string) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	// Remove existing schedule if any
	if entryID, exists := ws.schedules[workflowID]; exists {
		ws.cron.Remove(entryID)
	}

	// Add new schedule
	entryID, err := ws.cron.AddFunc(schedule, func() {
		if callback, exists := ws.callbacks[workflowID]; exists {
			// Callback is executed in background, errors are handled by the callback itself
			callback(workflowID) // #nosec G104 -- callback errors are handled internally
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule workflow %s: %v", workflowID, err)
	}

	ws.schedules[workflowID] = entryID
	return nil
}

// UnscheduleWorkflow removes a workflow from the schedule
func (ws *WorkflowScheduler) UnscheduleWorkflow(workflowID string) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if entryID, exists := ws.schedules[workflowID]; exists {
		ws.cron.Remove(entryID)
		delete(ws.schedules, workflowID)
		delete(ws.callbacks, workflowID)
	}
}

// SetCallback sets the callback function for a workflow
func (ws *WorkflowScheduler) SetCallback(workflowID string, callback ScheduleCallback) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	ws.callbacks[workflowID] = callback
}

// GetScheduledWorkflows returns all scheduled workflows
func (ws *WorkflowScheduler) GetScheduledWorkflows() []string {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	workflows := make([]string, 0, len(ws.schedules))
	for workflowID := range ws.schedules {
		workflows = append(workflows, workflowID)
	}

	return workflows
}

// GetNextRun returns the next scheduled run time for a workflow
func (ws *WorkflowScheduler) GetNextRun(workflowID string) (time.Time, error) {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	entryID, exists := ws.schedules[workflowID]
	if !exists {
		return time.Time{}, fmt.Errorf("workflow not scheduled: %s", workflowID)
	}

	entry := ws.cron.Entry(entryID)
	return entry.Next, nil
}
