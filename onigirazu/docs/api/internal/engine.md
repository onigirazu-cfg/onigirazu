package engine // import "github.com/onigirazu-cfg/onigirazu/internal/engine"


TYPES

type ExecutionEngine struct {
	// Has unexported fields.
}
    ExecutionEngine is the main engine for executing playbooks

func NewExecutionEngine(
	config interfaces.Config,
	logger interfaces.Logger,
	stateManager interfaces.StateManager,
	inventoryMgr interfaces.InventoryManager,
	moduleRegistry interfaces.ModuleRegistry,
	templateEngine interfaces.TemplateEngine,
	progressTracker interfaces.ProgressTracker,
	executionPool interfaces.ExecutionPool,
	cacheManager interfaces.CacheManager,
) *ExecutionEngine
    NewExecutionEngine creates a new execution engine

func (e *ExecutionEngine) ExecutePlaybook(ctx context.Context, playbook *types.Playbook) (*types.PlaybookResult, error)
    ExecutePlaybook executes a complete playbook

func (e *ExecutionEngine) GetExecutionSummary() map[string]interface{}
    GetExecutionSummary returns a comprehensive execution summary

func (e *ExecutionEngine) GetMetrics() *metrics.Metrics
    GetMetrics returns current execution metrics

type ExecutionStats struct {
	StartTime       time.Time
	EndTime         time.Time
	TotalTasks      int
	CompletedTasks  int
	SuccessfulTasks int
	FailedTasks     int
	ChangedTasks    int
	SkippedTasks    int
	HostStats       map[string]*HostStats
	// Has unexported fields.
}
    ExecutionStats holds execution statistics

type HostStats struct {
	TotalTasks      int
	CompletedTasks  int
	SuccessfulTasks int
	FailedTasks     int
	ChangedTasks    int
	SkippedTasks    int
	LastTaskTime    time.Time
}
    HostStats holds per-host statistics

