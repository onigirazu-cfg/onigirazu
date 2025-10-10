package interfaces

import (
	"context"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ModuleExecutor defines the interface for all modules
type ModuleExecutor interface {
	Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
	Validate(args map[string]interface{}) error
	GetName() string
	GetDescription() string
}

// StateManager defines the interface for state management
type StateManager interface {
	LoadState(ctx context.Context) (*types.State, error)
	SaveState(ctx context.Context, state *types.State) error
	HasChanged(filePath string) (bool, error)
	GetTaskState(taskID string) (*types.TaskState, bool)
	SetTaskState(taskID string, state *types.TaskState)
	Clear() error
}

// InventoryManager defines the interface for inventory management
type InventoryManager interface {
	LoadInventory(ctx context.Context, filePath string) error
	GetHosts(pattern string) ([]types.Host, error)
	GetGroups(pattern string) (map[string]*types.Group, error)
	GetHostByName(name string) (*types.Host, error)
	GetGroupByName(name string) (*types.Group, error)
	ListHosts() []string
	ListGroups() []string
	GetInventoryStats() map[string]interface{}
}

// PlaybookParser defines the interface for playbook parsing
type PlaybookParser interface {
	ParsePlaybook(ctx context.Context, filePath string) (*types.Playbook, error)
	ParseInventory(ctx context.Context, filePath string) (*types.Inventory, error)
	ValidatePlaybook(playbook *types.Playbook) error
	SetVariables(variables map[string]interface{})
	AddVariable(key string, value interface{})
}

// Logger defines the interface for logging
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
	SetLevel(level string)

	// Task and play logging
	TaskStart(taskName, hostName string)
	TaskEnd(taskName, hostName string, changed, success bool)
	PlayStart(playName string, playIndex, totalPlays int)
	PlayEnd(playName, hostName string, success bool, duration time.Duration)
	Progress(completed, total int, currentTask, currentHost string)
	Retry(taskName, hostName string, attempt, maxAttempts int, delay time.Duration, err error)
}

// ModuleRegistry defines the interface for module registration and retrieval
type ModuleRegistry interface {
	Register(name string, module ModuleExecutor) error
	Get(name string) (ModuleExecutor, error)
	ExecuteTask(ctx context.Context, task *types.Task, host types.Host, variables map[string]interface{}) (types.TaskResult, error)
	List() []string
	Unregister(name string) error
}

// CacheManager defines the interface for caching
type CacheManager interface {
	Get(ctx context.Context, key string) (interface{}, bool)
	Set(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Size() int
}

// TemplateEngine defines the interface for template processing
type TemplateEngine interface {
	Render(ctx context.Context, template string, variables map[string]interface{}) (string, error)
	RenderFile(ctx context.Context, filePath string, variables map[string]interface{}) (string, error)
	RenderTaskArgs(ctx context.Context, args map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error)
	ValidateTemplate(template string) error
}

// VaultManager defines the interface for secret management
type VaultManager interface {
	Encrypt(ctx context.Context, data []byte) ([]byte, error)
	Decrypt(ctx context.Context, data []byte) ([]byte, error)
	Store(ctx context.Context, key string, value []byte) error
	Retrieve(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// Config defines the interface for configuration management
type Config interface {
	GetMaxConcurrency() int
	GetDefaultTimeout() time.Duration
	GetRetryAttempts() int
	GetRetryDelay() time.Duration
	GetStateFile() string
	GetLogLevel() string
	GetLogFormat() string
	IsShellCommandsAllowed() bool
	GetBlockedCommands() []string
	IsCachingEnabled() bool
	GetCacheTTL() time.Duration
	IsChecksumEnabled() bool
	GetDryRun() bool
	GetCheckMode() bool
	// SSH connection defaults
	GetSSHDefaultUser() string
	GetSSHDefaultPort() int
	GetSSHDefaultKeyFile() string
}

// ProgressTracker defines the interface for progress tracking
type ProgressTracker interface {
	StartTracking()
	Stop()
	UpdateTask(hostName, taskName string, success bool)
	UpdateProgress(completed, total int)
	GetProgress() (completed, total int)
	GetStats() map[string]interface{}
}

// ExecutionPool defines the interface for parallel execution
type ExecutionPool interface {
	Submit(task func())
	SubmitWithResult(task func() error) <-chan error
	Shutdown(ctx context.Context) error
	GetStats() map[string]interface{}
}

// ExecutionEngine defines the interface for task execution
type ExecutionEngine interface {
	Execute(ctx context.Context, playbook *types.Playbook, inventory *types.Inventory, options *ExecutionOptions) error
	ExecuteTask(ctx context.Context, task *types.Task, host types.Host, variables map[string]interface{}) (types.TaskResult, error)
	ExecuteParallel(ctx context.Context, tasks []*types.Task, hosts []types.Host, variables map[string]interface{}) ([]types.TaskResult, error)
}

// ExecutionOptions holds options for execution
type ExecutionOptions struct {
	CheckMode   bool
	DryRun      bool
	Parallel    bool
	MaxWorkers  int
	Tags        []string
	SkipTags    []string
	StartAtTask string
	StopAtTask  string
	Variables   map[string]interface{}
}

// ProgressReporter defines the interface for progress reporting
type ProgressReporter interface {
	Start(total int)
	Update(completed int, message string)
	Finish()
	SetError(err error)
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	LogTaskStart(ctx context.Context, task *types.Task, host types.Host)
	LogTaskEnd(ctx context.Context, task *types.Task, host types.Host, result types.TaskResult, duration time.Duration)
	LogPlaybookStart(ctx context.Context, playbook *types.Playbook)
	LogPlaybookEnd(ctx context.Context, playbook *types.Playbook, success bool, duration time.Duration)
}

// CommandSanitizer defines the interface for command validation
type CommandSanitizer interface {
	Validate(ctx context.Context, command string) error
	IsAllowed(command string) bool
	SanitizeArgs(args []string) ([]string, error)
}
