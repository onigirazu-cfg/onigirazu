package drift

import (
	"time"
)

// DriftType represents the type of drift detected
type DriftType string

const (
	DriftTypeFile     DriftType = "file"
	DriftTypePackage  DriftType = "package"
	DriftTypeService  DriftType = "service"
	DriftTypeUser     DriftType = "user"
	DriftTypeGroup    DriftType = "group"
	DriftTypeConfig   DriftType = "config"
	DriftTypeFirewall DriftType = "firewall"
	DriftTypeCron     DriftType = "cron"
	DriftTypeGit      DriftType = "git"
)

// DriftSeverity represents the severity of drift
type DriftSeverity string

const (
	SeverityCritical DriftSeverity = "critical"
	SeverityHigh     DriftSeverity = "high"
	SeverityMedium   DriftSeverity = "medium"
	SeverityLow      DriftSeverity = "low"
	SeverityInfo     DriftSeverity = "info"
)

// DriftStatus represents the status of drift
type DriftStatus string

const (
	StatusDetected DriftStatus = "detected"
	StatusFixed    DriftStatus = "fixed"
	StatusIgnored  DriftStatus = "ignored"
	StatusFailed   DriftStatus = "failed"
)

// DriftItem represents a single drift detection
type DriftItem struct {
	ID           string                 `json:"id"`
	Type         DriftType              `json:"type"`
	Resource     string                 `json:"resource"`
	Host         string                 `json:"host"`
	Severity     DriftSeverity          `json:"severity"`
	Status       DriftStatus            `json:"status"`
	DetectedAt   time.Time              `json:"detected_at"`
	FixedAt      *time.Time             `json:"fixed_at,omitempty"`
	Expected     map[string]interface{} `json:"expected"`
	Actual       map[string]interface{} `json:"actual"`
	Diff         string                 `json:"diff"`
	Message      string                 `json:"message"`
	CanAutoFix   bool                   `json:"can_auto_fix"`
	FixOperation *FixOperation          `json:"fix_operation,omitempty"`
}

// FixOperation represents an operation to fix drift
type FixOperation struct {
	Module string                 `json:"module"`
	Args   map[string]interface{} `json:"args"`
	Order  int                    `json:"order"`
}

// DriftReport represents a complete drift detection report
type DriftReport struct {
	ID               string                 `json:"id"`
	Timestamp        time.Time              `json:"timestamp"`
	PlaybookID       string                 `json:"playbook_id,omitempty"`
	SnapshotID       string                 `json:"snapshot_id,omitempty"`
	TotalDrifts      int                    `json:"total_drifts"`
	CriticalDrifts   int                    `json:"critical_drifts"`
	HighDrifts       int                    `json:"high_drifts"`
	MediumDrifts     int                    `json:"medium_drifts"`
	LowDrifts        int                    `json:"low_drifts"`
	FixedDrifts      int                    `json:"fixed_drifts"`
	FailedFixes      int                    `json:"failed_fixes"`
	Items            []DriftItem            `json:"items"`
	DriftsByType     map[DriftType]int      `json:"drifts_by_type"`
	DriftsByHost     map[string]int         `json:"drifts_by_host"`
	DriftsBySeverity map[DriftSeverity]int  `json:"drifts_by_severity"`
	Duration         time.Duration          `json:"duration"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// DriftConfig represents drift detection configuration
type DriftConfig struct {
	// Detection settings
	Enabled         bool          `yaml:"enabled" json:"enabled"`
	CheckInterval   time.Duration `yaml:"check_interval" json:"check_interval"`
	Resources       []DriftType   `yaml:"resources" json:"resources"`
	IgnoreResources []string      `yaml:"ignore_resources" json:"ignore_resources"`

	// Auto-fix settings
	AutoFix         bool            `yaml:"auto_fix" json:"auto_fix"`
	AutoFixSeverity []DriftSeverity `yaml:"auto_fix_severity" json:"auto_fix_severity"`
	DryRun          bool            `yaml:"dry_run" json:"dry_run"`
	MaxConcurrency  int             `yaml:"max_concurrency" json:"max_concurrency"`

	// Notification settings
	Notifications NotificationConfig `yaml:"notifications" json:"notifications"`

	// Report settings
	ReportFormat string `yaml:"report_format" json:"report_format"`
	ReportPath   string `yaml:"report_path" json:"report_path"`
	KeepReports  int    `yaml:"keep_reports" json:"keep_reports"`

	// Scheduling
	Schedule   string `yaml:"schedule" json:"schedule"` // Cron expression
	DaemonMode bool   `yaml:"daemon_mode" json:"daemon_mode"`
}

// NotificationConfig represents notification configuration
type NotificationConfig struct {
	Enabled     bool                  `yaml:"enabled" json:"enabled"`
	OnDetect    bool                  `yaml:"on_detect" json:"on_detect"`
	OnFix       bool                  `yaml:"on_fix" json:"on_fix"`
	OnFail      bool                  `yaml:"on_fail" json:"on_fail"`
	Channels    []NotificationChannel `yaml:"channels" json:"channels"`
	MinSeverity DriftSeverity         `yaml:"min_severity" json:"min_severity"`
}

// NotificationChannel represents a notification channel
type NotificationChannel struct {
	Type    string                 `yaml:"type" json:"type"` // email, slack, webhook
	Enabled bool                   `yaml:"enabled" json:"enabled"`
	Config  map[string]interface{} `yaml:"config" json:"config"`
}

// ResourceState represents the expected state of a resource
type ResourceState struct {
	Type       DriftType              `json:"type"`
	Identifier string                 `json:"identifier"`
	Host       string                 `json:"host"`
	State      map[string]interface{} `json:"state"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// DriftCheckResult represents the result of a drift check
type DriftCheckResult struct {
	HasDrift bool                   `json:"has_drift"`
	Expected map[string]interface{} `json:"expected"`
	Actual   map[string]interface{} `json:"actual"`
	Diff     map[string]DiffValue   `json:"diff"`
	Message  string                 `json:"message"`
}

// DiffValue represents a difference between expected and actual values
type DiffValue struct {
	Expected interface{} `json:"expected"`
	Actual   interface{} `json:"actual"`
	Changed  bool        `json:"changed"`
}

// ScheduleEntry represents a scheduled drift check
type ScheduleEntry struct {
	ID       string       `json:"id"`
	Schedule string       `json:"schedule"` // Cron expression
	NextRun  time.Time    `json:"next_run"`
	LastRun  *time.Time   `json:"last_run,omitempty"`
	Enabled  bool         `json:"enabled"`
	Config   *DriftConfig `json:"config"`
}
