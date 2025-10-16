package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
)

// Recorder manages audit recording of playbook executions
type Recorder struct {
	config        AuditConfig
	currentRecord *ExecutionRecord
	storage       *Storage
	logger        interfaces.Logger
	mutex         sync.RWMutex
	started       bool
}

// NewRecorder creates a new audit recorder
func NewRecorder(config AuditConfig, logger interfaces.Logger) (*Recorder, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("audit recording is disabled")
	}

	storage, err := NewStorage(config.StoragePath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	return &Recorder{
		config:  config,
		storage: storage,
		logger:  logger,
		started: false,
	}, nil
}

// StartExecution begins a new execution record
func (r *Recorder) StartExecution(playbookPath, inventoryPath string, tags []string) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.started {
		return "", fmt.Errorf("execution already in progress")
	}

	currentUser := getCurrentUser()
	id := uuid.New().String()

	r.currentRecord = &ExecutionRecord{
		ID:            id,
		PlaybookPath:  playbookPath,
		InventoryPath: inventoryPath,
		User:          currentUser,
		StartTime:     time.Now(),
		Status:        StatusRunning,
		Plays:         []PlayExecution{},
		Tags:          tags,
		Variables:     make(map[string]interface{}),
		Metadata:      make(map[string]interface{}),
		Environment:   getEnvironmentMetadata(),
		AffectedHosts: []string{},
	}

	r.started = true
	r.logger.Info("Audit: Started execution recording: %s", id)
	return id, nil
}

// RecordPlay records the execution of a play
func (r *Recorder) RecordPlay(name string, playIndex int, hosts []string) *PlayRecorder {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if !r.started || r.currentRecord == nil {
		return nil
	}

	play := PlayExecution{
		Name:      name,
		Index:     playIndex,
		Hosts:     hosts,
		Tasks:     []TaskResult{},
		StartTime: time.Now(),
		Status:    StatusRunning,
	}

	// Add to current record (note: we'll need to update this reference)
	r.currentRecord.Plays = append(r.currentRecord.Plays, play)

	return &PlayRecorder{
		play:     &r.currentRecord.Plays[len(r.currentRecord.Plays)-1],
		recorder: r,
	}
}

// RecordTaskResult records the result of a task execution
func (r *Recorder) RecordTaskResult(taskResult TaskResult) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.started || r.currentRecord == nil {
		return fmt.Errorf("no execution in progress")
	}

	if len(r.currentRecord.Plays) == 0 {
		return fmt.Errorf("no play in progress")
	}

	// Add to the last play
	lastPlay := &r.currentRecord.Plays[len(r.currentRecord.Plays)-1]
	lastPlay.Tasks = append(lastPlay.Tasks, taskResult)

	// Update statistics
	r.currentRecord.TotalTasks++
	switch taskResult.Status {
	case TaskStatusOk:
		r.currentRecord.SuccessfulTasks++
	case TaskStatusChanged:
		r.currentRecord.SuccessfulTasks++
	case TaskStatusFailed:
		r.currentRecord.FailedTasks++
	case TaskStatusSkipped:
		r.currentRecord.SkippedTasks++
	}

	// Track affected hosts
	if !contains(r.currentRecord.AffectedHosts, taskResult.Host) && taskResult.Status != TaskStatusSkipped {
		r.currentRecord.AffectedHosts = append(r.currentRecord.AffectedHosts, taskResult.Host)
	}

	return nil
}

// RecordUnreachableHost records a host that couldn't be reached
func (r *Recorder) RecordUnreachableHost(host string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.started || r.currentRecord == nil {
		return fmt.Errorf("no execution in progress")
	}

	if !contains(r.currentRecord.UnreachableHosts, host) {
		r.currentRecord.UnreachableHosts = append(r.currentRecord.UnreachableHosts, host)
	}

	return nil
}

// SetVariables sets the variables used in the execution
func (r *Recorder) SetVariables(vars map[string]interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.started || r.currentRecord == nil {
		return fmt.Errorf("no execution in progress")
	}

	if !r.config.IncludeSensitive {
		// Filter out sensitive variables
		vars = filterSensitiveVariables(vars)
	}

	r.currentRecord.Variables = vars
	return nil
}

// SetMetadata sets additional metadata for the execution
func (r *Recorder) SetMetadata(key string, value interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.started || r.currentRecord == nil {
		return fmt.Errorf("no execution in progress")
	}

	r.currentRecord.Metadata[key] = value
	return nil
}

// SetCheckMode sets the check mode flag
func (r *Recorder) SetCheckMode(checkMode bool) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.started || r.currentRecord == nil {
		return fmt.Errorf("no execution in progress")
	}

	r.currentRecord.CheckMode = checkMode
	return nil
}

// SetDebugMode sets the debug mode flag
func (r *Recorder) SetDebugMode(debugMode bool) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.started || r.currentRecord == nil {
		return fmt.Errorf("no execution in progress")
	}

	r.currentRecord.DebugMode = debugMode
	return nil
}

// CompleteExecution finishes the current execution record
func (r *Recorder) CompleteExecution(status ExecutionStatus, exitCode int, errMsg string) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.started || r.currentRecord == nil {
		return "", fmt.Errorf("no execution in progress")
	}

	// Finalize all plays
	now := time.Now()
	for i := range r.currentRecord.Plays {
		if r.currentRecord.Plays[i].EndTime.IsZero() {
			r.currentRecord.Plays[i].EndTime = now
			r.currentRecord.Plays[i].Duration = now.Sub(r.currentRecord.Plays[i].StartTime).Seconds()
			// Determine play status based on tasks
			r.currentRecord.Plays[i].Status = determinePlayStatus(r.currentRecord.Plays[i].Tasks)
		}
	}

	r.currentRecord.EndTime = now
	r.currentRecord.Status = status
	r.currentRecord.Duration = now.Sub(r.currentRecord.StartTime).Seconds()
	r.currentRecord.ExitCode = exitCode
	r.currentRecord.ErrorMessage = errMsg

	// Save to storage
	err := r.storage.SaveRecord(r.currentRecord)
	if err != nil {
		r.logger.Error("Failed to save audit record: %v", err)
		return "", err
	}

	recordID := r.currentRecord.ID
	r.currentRecord = nil
	r.started = false

	r.logger.Info("Audit: Completed execution recording: %s (status: %s)", recordID, status)
	return recordID, nil
}

// GetCurrentRecord returns the current execution record
func (r *Recorder) GetCurrentRecord() *ExecutionRecord {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.currentRecord == nil {
		return nil
	}

	// Return a copy
	recordCopy := *r.currentRecord
	return &recordCopy
}

// Close closes the recorder and ensures all data is saved
func (r *Recorder) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.started && r.currentRecord != nil {
		// Force completion if still running
		r.currentRecord.Status = StatusFailure
		r.currentRecord.ErrorMessage = "execution interrupted"
		r.currentRecord.EndTime = time.Now()
		r.currentRecord.Duration = r.currentRecord.EndTime.Sub(r.currentRecord.StartTime).Seconds()
		_ = r.storage.SaveRecord(r.currentRecord)
		r.started = false
	}

	return r.storage.Close()
}

// PlayRecorder handles recording of a specific play
type PlayRecorder struct {
	play     *PlayExecution
	recorder *Recorder
}

// RecordTask records a task within the play
func (pr *PlayRecorder) RecordTask(task TaskResult) error {
	if pr.play == nil || pr.recorder == nil {
		return fmt.Errorf("invalid play recorder state")
	}

	pr.play.Tasks = append(pr.play.Tasks, task)
	return pr.recorder.RecordTaskResult(task)
}

// Complete marks the play as complete
func (pr *PlayRecorder) Complete(status ExecutionStatus) {
	if pr.play == nil {
		return
	}

	now := time.Now()
	pr.play.EndTime = now
	pr.play.Duration = now.Sub(pr.play.StartTime).Seconds()
	pr.play.Status = status
}

// Helper functions

func getCurrentUser() string {
	currentUser, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return currentUser.Username
}

func getEnvironmentMetadata() map[string]string {
	return map[string]string{
		"HOSTNAME": getHostname(),
		"PWD":      os.Getenv("PWD"),
		"SHELL":    os.Getenv("SHELL"),
		"USER":     os.Getenv("USER"),
		"PATH":     os.Getenv("PATH"),
	}
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func contains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func filterSensitiveVariables(vars map[string]interface{}) map[string]interface{} {
	sensitiveKeys := []string{
		"password", "passwd", "pwd",
		"token", "secret", "api_key", "apikey",
		"private_key", "privatekey",
		"access_key", "accesskey",
		"auth", "credential", "credentials",
	}

	filtered := make(map[string]interface{})
	for k, v := range vars {
		isSensitive := false
		for _, sensitive := range sensitiveKeys {
			if contains([]string{k}, sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			filtered[k] = "***REDACTED***"
		} else {
			filtered[k] = v
		}
	}

	return filtered
}

func determinePlayStatus(tasks []TaskResult) ExecutionStatus {
	if len(tasks) == 0 {
		return StatusSkipped
	}

	for _, task := range tasks {
		if task.Status == TaskStatusFailed {
			return StatusFailure
		}
	}

	return StatusSuccess
}

// MarshalJSON provides custom JSON marshaling for ExecutionRecord
func (r *ExecutionRecord) MarshalJSON() ([]byte, error) {
	type Alias ExecutionRecord
	return json.Marshal(&struct {
		*Alias
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time,omitempty"`
		CreatedAt string `json:"created_at"`
	}{
		Alias:     (*Alias)(r),
		StartTime: r.StartTime.Format(time.RFC3339),
		EndTime:   r.EndTime.Format(time.RFC3339),
		CreatedAt: time.Now().Format(time.RFC3339),
	})
}
