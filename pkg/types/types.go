package types

import (
	"context"
	"time"
)

// Host represents a target host
type Host struct {
	Name     string            `yaml:"name"`
	Address  string            `yaml:"address"`
	Port     int               `yaml:"port,omitempty"`
	User     string            `yaml:"user,omitempty"`
	Password string            `yaml:"password,omitempty"`
	KeyFile  string            `yaml:"key_file,omitempty"`
	Vars     map[string]interface{} `yaml:"vars,omitempty"`
}

// Inventory represents a host inventory
type Inventory struct {
	Hosts  []Host                `yaml:"hosts"`
	Groups map[string]*Group     `yaml:"groups,omitempty"`
}

// Task represents a single task
type Task struct {
	Name         string                 `yaml:"name"`
	Module       string                 `yaml:"module"`
	Args         map[string]interface{} `yaml:"args"`
	When         string                 `yaml:"when,omitempty"`
	Loop         *Loop                  `yaml:"loop,omitempty"`
	Register     string                 `yaml:"register,omitempty"`
	IgnoreErrors bool                   `yaml:"ignore_errors,omitempty"`
	Tags         []string               `yaml:"tags,omitempty"`
	Notify       []string               `yaml:"notify,omitempty"`
	Timeout      time.Duration          `yaml:"timeout,omitempty"`
	Retries      int                    `yaml:"retries,omitempty"`
	Delay        time.Duration          `yaml:"delay,omitempty"`
	Until        string                 `yaml:"until,omitempty"`
	ChangedWhen  string                 `yaml:"changed_when,omitempty"`
	FailedWhen   string                 `yaml:"failed_when,omitempty"`
	Include      string                 `yaml:"include,omitempty"`
	Serial       bool                   `yaml:"serial,omitempty"`
	RetryDelay   time.Duration          `yaml:"retry_delay,omitempty"`
}

// Play represents a set of tasks to execute
type Play struct {
	Name         string                 `yaml:"name"`
	Hosts        string                 `yaml:"hosts"`
	Vars         map[string]interface{} `yaml:"vars,omitempty"`
	Tasks        []Task                 `yaml:"tasks"`
	PreTasks     []Task                 `yaml:"pre_tasks,omitempty"`
	PostTasks    []Task                 `yaml:"post_tasks,omitempty"`
	Handlers     []Task                 `yaml:"handlers,omitempty"`
	Become       bool                   `yaml:"become,omitempty"`
	BecomeUser   string                 `yaml:"become_user,omitempty"`
	BecomeMethod string                 `yaml:"become_method,omitempty"`
	Tags         []string               `yaml:"tags,omitempty"`
	When         string                 `yaml:"when,omitempty"`
	Serial       interface{}            `yaml:"serial,omitempty"`
	MaxFailPercentage int               `yaml:"max_fail_percentage,omitempty"`
	AnyErrorsFatal bool                 `yaml:"any_errors_fatal,omitempty"`
	IgnoreErrors   bool                 `yaml:"ignore_errors,omitempty"`
	GatherFacts    bool                 `yaml:"gather_facts,omitempty"`
}

// Playbook represents a complete playbook
type Playbook struct {
	Name     string                 `yaml:"name"`
	Plays    []Play                 `yaml:"plays"`
	Vars     map[string]interface{} `yaml:"vars,omitempty"`
	FilePath string                 `yaml:"-" json:"-"`
}

// TaskResult represents the result of task execution
type TaskResult struct {
	TaskName   string                 `json:"task_name"`
	Host       string                 `json:"host"`
	Module     string                 `json:"module"`
	Success    bool                   `json:"success"`
	Failed     bool                   `json:"failed"`
	Changed    bool                   `json:"changed"`
	Skipped    bool                   `json:"skipped"`
	Output     map[string]interface{} `json:"output"`
	Error      string                 `json:"error,omitempty"`
	Duration   time.Duration          `json:"duration"`
	Timestamp  time.Time              `json:"timestamp"`
}

// PlayResult represents the result of play execution
type PlayResult struct {
	Name      string        `json:"name"`
	PlayName  string        `json:"play_name"`
	Host      string        `json:"host"`
	Hosts     []HostResult  `json:"hosts"`
	Tasks     []TaskResult  `json:"tasks"`
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
}

// HostResult represents the result for a specific host
type HostResult struct {
	Host    string       `json:"host"`
	Tasks   []TaskResult `json:"tasks"`
	Success bool         `json:"success"`
	Failed  bool         `json:"failed"`
}

// State represents saved state
type State struct {
	LastRun     time.Time              `json:"last_run"`
	Playbook    string                 `json:"playbook"`
	Results     []PlayResult           `json:"results"`
	Variables   map[string]interface{} `json:"variables"`
	Checksums   map[string]string      `json:"checksums"`
}

// Module represents a module interface
type Module interface {
	Execute(ctx context.Context, host Host, args map[string]interface{}) (TaskResult, error)
	Validate(args map[string]interface{}) error
	GetName() string
	GetDescription() string
}

// TaskState represents the state of a single task
type TaskState struct {
	TaskID      string                 `json:"task_id"`
	TaskName    string                 `json:"task_name"`
	Host        string                 `json:"host"`
	Module      string                 `json:"module"`
	Args        map[string]interface{} `json:"args"`
	Result      TaskResult             `json:"result"`
	LastRun     time.Time              `json:"last_run"`
	Checksum    string                 `json:"checksum"`
	Variables   map[string]interface{} `json:"variables"`
}

// ExecutionContext holds context for task execution
type ExecutionContext struct {
	Variables      map[string]interface{} `json:"variables"`
	Facts          map[string]interface{} `json:"facts"`
	Hostvars       map[string]interface{} `json:"hostvars"`
	GroupVars      map[string]interface{} `json:"group_vars"`
	PlayVars       map[string]interface{} `json:"play_vars"`
	TaskVars       map[string]interface{} `json:"task_vars"`
	RegisteredVars map[string]interface{} `json:"registered_vars"`
	Host           *Host                  `json:"host,omitempty"`
	Task           *Task                  `json:"task,omitempty"`
}

// ProgressInfo represents progress information
type ProgressInfo struct {
	Total       int           `json:"total"`
	Completed   int           `json:"completed"`
	Failed      int           `json:"failed"`
	Skipped     int           `json:"skipped"`
	CurrentTask string        `json:"current_task"`
	CurrentHost string        `json:"current_host"`
	StartTime   time.Time     `json:"start_time"`
	Duration    time.Duration `json:"duration"`
}

// RetryInfo holds retry configuration
type RetryInfo struct {
	Attempts    int           `json:"attempts"`
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	LastError   string        `json:"last_error,omitempty"`
}

// ConditionResult represents the result of condition evaluation
type ConditionResult struct {
	Condition string `json:"condition"`
	Result    bool   `json:"result"`
	Error     string `json:"error,omitempty"`
}

// TemplateResult represents the result of template rendering
type TemplateResult struct {
	Original string `json:"original"`
	Rendered string `json:"rendered"`
	Error    string `json:"error,omitempty"`
}

// CacheEntry represents a cache entry
type CacheEntry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	CreatedAt time.Time   `json:"created_at"`
	ExpiresAt time.Time   `json:"expires_at"`
	TTL       time.Duration `json:"ttl"`
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"` // task_start, task_end, playbook_start, playbook_end
	User      string                 `json:"user"`
	Host      string                 `json:"host"`
	Task      string                 `json:"task,omitempty"`
	Playbook  string                 `json:"playbook,omitempty"`
	Result    *TaskResult            `json:"result,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PlaybookResult represents the result of playbook execution
type PlaybookResult struct {
	Name         string                 `json:"name"`
	Success      bool                   `json:"success"`
	Failed       bool                   `json:"failed"`
	PlayResults  []PlayResult           `json:"play_results"`
	Plays        []PlayResult           `json:"plays"`
	TotalTasks   int                    `json:"total_tasks"`
	SuccessTasks int                    `json:"success_tasks"`
	FailedTasks  int                    `json:"failed_tasks"`
	SkippedTasks int                    `json:"skipped_tasks"`
	ChangedTasks int                    `json:"changed_tasks"`
	Duration     time.Duration          `json:"duration"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
	Stats        map[string]interface{} `json:"stats,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// Loop represents loop configuration for tasks
type Loop struct {
	Items    []interface{} `yaml:"items,omitempty" json:"items,omitempty"`
	Variable string        `yaml:"var,omitempty" json:"var,omitempty"`
	Index    string        `yaml:"index,omitempty" json:"index,omitempty"`
	Range    string        `yaml:"range,omitempty" json:"range,omitempty"`
}

// Group represents a host group
type Group struct {
	Name      string                    `yaml:"name" json:"name"`
	Hosts     map[string]*Host          `yaml:"hosts,omitempty" json:"hosts,omitempty"`
	Children  []string                  `yaml:"children,omitempty" json:"children,omitempty"`
	Variables map[string]interface{}    `yaml:"vars,omitempty" json:"vars,omitempty"`
	Vars      map[string]interface{}    `yaml:"-" json:"-"` // Alias for Variables
}
