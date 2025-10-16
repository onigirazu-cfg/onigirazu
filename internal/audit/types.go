package audit

import (
	"time"
)

// ExecutionStatus represents the status of a playbook execution
type ExecutionStatus string

const (
	StatusSuccess ExecutionStatus = "success"
	StatusFailure ExecutionStatus = "failure"
	StatusRunning ExecutionStatus = "running"
	StatusSkipped ExecutionStatus = "skipped"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusOk          TaskStatus = "ok"
	TaskStatusChanged     TaskStatus = "changed"
	TaskStatusFailed      TaskStatus = "failed"
	TaskStatusSkipped     TaskStatus = "skipped"
	TaskStatusUnreachable TaskStatus = "unreachable"
)

// TaskResult represents the result of a single task execution
type TaskResult struct {
	Name       string                 `json:"name"`
	Module     string                 `json:"module"`
	Status     TaskStatus             `json:"status"`
	Host       string                 `json:"host"`
	Duration   float64                `json:"duration"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Output     string                 `json:"output,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Changed    bool                   `json:"changed"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	ReturnCode int                    `json:"return_code,omitempty"`
	Stdout     string                 `json:"stdout,omitempty"`
	Stderr     string                 `json:"stderr,omitempty"`
}

// PlayExecution represents the execution of a single play
type PlayExecution struct {
	Name      string          `json:"name"`
	Index     int             `json:"index"`
	Hosts     []string        `json:"hosts"`
	Tasks     []TaskResult    `json:"tasks"`
	StartTime time.Time       `json:"start_time"`
	EndTime   time.Time       `json:"end_time"`
	Duration  float64         `json:"duration"`
	Status    ExecutionStatus `json:"status"`
}

// ExecutionRecord represents a complete playbook execution record
type ExecutionRecord struct {
	ID               string                 `json:"id"`
	PlaybookPath     string                 `json:"playbook_path"`
	InventoryPath    string                 `json:"inventory_path"`
	User             string                 `json:"user"`
	StartTime        time.Time              `json:"start_time"`
	EndTime          time.Time              `json:"end_time,omitempty"`
	Duration         float64                `json:"duration,omitempty"`
	Status           ExecutionStatus        `json:"status"`
	Plays            []PlayExecution        `json:"plays"`
	TotalTasks       int                    `json:"total_tasks"`
	SuccessfulTasks  int                    `json:"successful_tasks"`
	FailedTasks      int                    `json:"failed_tasks"`
	SkippedTasks     int                    `json:"skipped_tasks"`
	UnreachableHosts []string               `json:"unreachable_hosts,omitempty"`
	AffectedHosts    []string               `json:"affected_hosts"`
	Tags             []string               `json:"tags,omitempty"`
	Variables        map[string]interface{} `json:"variables,omitempty"`
	ExitCode         int                    `json:"exit_code"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Environment      map[string]string      `json:"environment,omitempty"`
	CheckMode        bool                   `json:"check_mode"`
	DebugMode        bool                   `json:"debug_mode"`
}

// AuditConfig contains audit configuration
type AuditConfig struct {
	Enabled          bool
	StoragePath      string
	MaxRecords       int
	RetentionDays    int
	EncryptRecords   bool
	IncludeSensitive bool
}

// AuditStatistics contains audit statistics
type AuditStatistics struct {
	TotalExecutions  int       `json:"total_executions"`
	SuccessfulRuns   int       `json:"successful_runs"`
	FailedRuns       int       `json:"failed_runs"`
	TotalTasks       int       `json:"total_tasks"`
	TotalFailedTasks int       `json:"total_failed_tasks"`
	AvgDuration      float64   `json:"avg_duration"`
	FirstExecution   time.Time `json:"first_execution,omitempty"`
	LastExecution    time.Time `json:"last_execution,omitempty"`
	CommonErrors     []string  `json:"common_errors,omitempty"`
	MostUsedModules  []string  `json:"most_used_modules,omitempty"`
}

// HostAuditStatistics contains statistics for a specific host
type HostAuditStatistics struct {
	Host             string    `json:"host"`
	TotalExecutions  int       `json:"total_executions"`
	SuccessfulRuns   int       `json:"successful_runs"`
	FailedRuns       int       `json:"failed_runs"`
	TotalTasks       int       `json:"total_tasks"`
	TotalFailedTasks int       `json:"total_failed_tasks"`
	LastExecution    time.Time `json:"last_execution,omitempty"`
	AvgTaskDuration  float64   `json:"avg_task_duration"`
	MostCommonErrors []string  `json:"most_common_errors,omitempty"`
}

// FilterOptions for querying audit records
type FilterOptions struct {
	PlaybookPath string
	Status       ExecutionStatus
	HostFilter   string
	DateFrom     time.Time
	DateTo       time.Time
	Limit        int
	Offset       int
	SortBy       string // "time", "duration", "status"
	SortOrder    string // "asc", "desc"
}
