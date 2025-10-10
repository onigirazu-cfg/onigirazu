package workflow // import "github.com/onigirazu-cfg/onigirazu/internal/workflow"


TYPES

type ActionType string
    ActionType represents the type of action

const (
	ActionTypeExecuteTask     ActionType = "execute_task"
	ActionTypeExecutePlaybook ActionType = "execute_playbook"
	ActionTypeWait            ActionType = "wait"
	ActionTypeNotify          ActionType = "notify"
	ActionTypeSetVariable     ActionType = "set_variable"
	ActionTypeCallAPI         ActionType = "call_api"
	ActionTypeRunScript       ActionType = "run_script"
)
type BackoffType string
    BackoffType represents the type of backoff strategy

const (
	BackoffTypeFixed       BackoffType = "fixed"
	BackoffTypeLinear      BackoffType = "linear"
	BackoffTypeExponential BackoffType = "exponential"
)
type ConditionType string
    ConditionType represents the type of condition

const (
	ConditionTypeVariable   ConditionType = "variable"
	ConditionTypeExpression ConditionType = "expression"
	ConditionTypeTime       ConditionType = "time"
	ConditionTypeEvent      ConditionType = "event"
)
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata"`
}
    Event represents an event in the system

type EventBus struct {
	// Has unexported fields.
}
    EventBus manages event publishing and subscription

func NewEventBus() *EventBus
    NewEventBus creates a new event bus

func (eb *EventBus) GetEventLog() []Event
    GetEventLog returns the event log

func (eb *EventBus) Publish(eventType string, data interface{})
    Publish publishes an event

func (eb *EventBus) Subscribe(eventType string, handler EventHandler)
    Subscribe subscribes to events of a specific type

func (eb *EventBus) Unsubscribe(eventType string)
    Unsubscribe removes all handlers for an event type

type EventHandler func(event Event)
    EventHandler handles events

type ExecutionStatus string
    ExecutionStatus represents the status of execution

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "canceled"
	StatusSkipped   ExecutionStatus = "skipped"
)
type OrchestratorConfig struct {
	MaxConcurrentWorkflows int           `json:"max_concurrent_workflows"`
	DefaultTimeout         time.Duration `json:"default_timeout"`
	RetryPolicy            RetryPolicy   `json:"retry_policy"`
	EnableMetrics          bool          `json:"enable_metrics"`
	EnableAudit            bool          `json:"enable_audit"`
}
    OrchestratorConfig holds orchestrator configuration

type RetryPolicy struct {
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	BackoffType BackoffType   `json:"backoff_type"`
	MaxDelay    time.Duration `json:"max_delay"`
}
    RetryPolicy defines retry behavior

type ScheduleCallback func(workflowID string) error
    ScheduleCallback is called when a scheduled workflow should be executed

type StepAction struct {
	Type       ActionType             `json:"type"`
	Target     string                 `json:"target"`
	Parameters map[string]interface{} `json:"parameters"`
	Timeout    time.Duration          `json:"timeout"`
}
    StepAction represents an action to be performed

type StepCondition struct {
	Type       ConditionType `json:"type"`
	Variable   string        `json:"variable"`
	Operator   string        `json:"operator"`
	Value      interface{}   `json:"value"`
	Expression string        `json:"expression"`
}
    StepCondition represents a condition for step execution

type StepExecution struct {
	ID         string                 `json:"id"`
	StepID     string                 `json:"step_id"`
	Status     ExecutionStatus        `json:"status"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	RetryCount int                    `json:"retry_count"`
	Output     map[string]interface{} `json:"output"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
}
    StepExecution represents a step execution instance

type StepType string
    StepType represents the type of workflow step

const (
	StepTypeTask         StepType = "task"
	StepTypePlaybook     StepType = "playbook"
	StepTypeCondition    StepType = "condition"
	StepTypeLoop         StepType = "loop"
	StepTypeParallel     StepType = "parallel"
	StepTypeWait         StepType = "wait"
	StepTypeNotification StepType = "notification"
	StepTypeCustom       StepType = "custom"
)
type TriggerCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}
    TriggerCondition represents a trigger condition

type TriggerType string
    TriggerType represents the type of trigger

const (
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeEvent    TriggerType = "event"
	TriggerTypeManual   TriggerType = "manual"
	TriggerTypeWebhook  TriggerType = "webhook"
)
type Workflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Steps       []WorkflowStep         `json:"steps"`
	Triggers    []WorkflowTrigger      `json:"triggers"`
	Variables   map[string]interface{} `json:"variables"`
	Timeout     time.Duration          `json:"timeout"`
	RetryPolicy RetryPolicy            `json:"retry_policy"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Enabled     bool                   `json:"enabled"`
}
    Workflow represents a complex workflow

type WorkflowExecution struct {
	ID         string                    `json:"id"`
	WorkflowID string                    `json:"workflow_id"`
	Status     ExecutionStatus           `json:"status"`
	StartTime  time.Time                 `json:"start_time"`
	EndTime    time.Time                 `json:"end_time"`
	Duration   time.Duration             `json:"duration"`
	Steps      map[string]*StepExecution `json:"steps"`
	Variables  map[string]interface{}    `json:"variables"`
	Trigger    *WorkflowTrigger          `json:"trigger"`
	Error      string                    `json:"error,omitempty"`
	Metadata   map[string]interface{}    `json:"metadata"`
	Context    context.Context           `json:"-"`
	CancelFunc context.CancelFunc        `json:"-"`
}
    WorkflowExecution represents a workflow execution instance

type WorkflowOrchestrator struct {
	// Has unexported fields.
}
    WorkflowOrchestrator manages complex workflow execution

func NewWorkflowOrchestrator(config OrchestratorConfig) *WorkflowOrchestrator
    NewWorkflowOrchestrator creates a new workflow orchestrator

func (wo *WorkflowOrchestrator) CancelExecution(id string) error
    CancelExecution cancels a running execution

func (wo *WorkflowOrchestrator) ExecuteWorkflow(workflowID string, trigger *WorkflowTrigger, variables map[string]interface{}) (*WorkflowExecution, error)
    ExecuteWorkflow executes a workflow

func (wo *WorkflowOrchestrator) GetExecution(id string) (*WorkflowExecution, error)
    GetExecution returns an execution by ID

func (wo *WorkflowOrchestrator) GetWorkflow(id string) (*Workflow, error)
    GetWorkflow returns a workflow by ID

func (wo *WorkflowOrchestrator) ListExecutions() []*WorkflowExecution
    ListExecutions returns all executions

func (wo *WorkflowOrchestrator) ListWorkflows() []*Workflow
    ListWorkflows returns all registered workflows

func (wo *WorkflowOrchestrator) RegisterWorkflow(workflow *Workflow) error
    RegisterWorkflow registers a new workflow

type WorkflowScheduler struct {
	// Has unexported fields.
}
    WorkflowScheduler manages scheduled workflow executions

func NewWorkflowScheduler() *WorkflowScheduler
    NewWorkflowScheduler creates a new workflow scheduler

func (ws *WorkflowScheduler) GetNextRun(workflowID string) (time.Time, error)
    GetNextRun returns the next scheduled run time for a workflow

func (ws *WorkflowScheduler) GetScheduledWorkflows() []string
    GetScheduledWorkflows returns all scheduled workflows

func (ws *WorkflowScheduler) ScheduleWorkflow(workflowID, schedule string) error
    ScheduleWorkflow schedules a workflow for execution

func (ws *WorkflowScheduler) SetCallback(workflowID string, callback ScheduleCallback)
    SetCallback sets the callback function for a workflow

func (ws *WorkflowScheduler) Start()
    Start starts the scheduler

func (ws *WorkflowScheduler) Stop()
    Stop stops the scheduler

func (ws *WorkflowScheduler) UnscheduleWorkflow(workflowID string)
    UnscheduleWorkflow removes a workflow from the schedule

type WorkflowStep struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         StepType               `json:"type"`
	Action       StepAction             `json:"action"`
	Dependencies []string               `json:"dependencies"`
	Conditions   []StepCondition        `json:"conditions"`
	Timeout      time.Duration          `json:"timeout"`
	RetryPolicy  RetryPolicy            `json:"retry_policy"`
	OnSuccess    []StepAction           `json:"on_success"`
	OnFailure    []StepAction           `json:"on_failure"`
	Variables    map[string]interface{} `json:"variables"`
	Metadata     map[string]interface{} `json:"metadata"`
}
    WorkflowStep represents a single step in a workflow

type WorkflowTrigger struct {
	ID         string                 `json:"id"`
	Type       TriggerType            `json:"type"`
	Schedule   string                 `json:"schedule,omitempty"`
	Event      string                 `json:"event,omitempty"`
	Conditions []TriggerCondition     `json:"conditions"`
	Parameters map[string]interface{} `json:"parameters"`
	Enabled    bool                   `json:"enabled"`
}
    WorkflowTrigger represents a workflow trigger

