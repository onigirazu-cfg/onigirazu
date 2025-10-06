package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WorkflowOrchestrator manages complex workflow execution
type WorkflowOrchestrator struct {
	workflows  map[string]*Workflow
	executions map[string]*WorkflowExecution
	scheduler  *WorkflowScheduler
	eventBus   *EventBus
	mutex      sync.RWMutex
	config     OrchestratorConfig
}

// OrchestratorConfig holds orchestrator configuration
type OrchestratorConfig struct {
	MaxConcurrentWorkflows int           `json:"max_concurrent_workflows"`
	DefaultTimeout         time.Duration `json:"default_timeout"`
	RetryPolicy            RetryPolicy   `json:"retry_policy"`
	EnableMetrics          bool          `json:"enable_metrics"`
	EnableAudit            bool          `json:"enable_audit"`
}

// Workflow represents a complex workflow
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

// WorkflowStep represents a single step in a workflow
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

// StepType represents the type of workflow step
type StepType string

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

// StepAction represents an action to be performed
type StepAction struct {
	Type       ActionType             `json:"type"`
	Target     string                 `json:"target"`
	Parameters map[string]interface{} `json:"parameters"`
	Timeout    time.Duration          `json:"timeout"`
}

// ActionType represents the type of action
type ActionType string

const (
	ActionTypeExecuteTask     ActionType = "execute_task"
	ActionTypeExecutePlaybook ActionType = "execute_playbook"
	ActionTypeWait            ActionType = "wait"
	ActionTypeNotify          ActionType = "notify"
	ActionTypeSetVariable     ActionType = "set_variable"
	ActionTypeCallAPI         ActionType = "call_api"
	ActionTypeRunScript       ActionType = "run_script"
)

// StepCondition represents a condition for step execution
type StepCondition struct {
	Type       ConditionType `json:"type"`
	Variable   string        `json:"variable"`
	Operator   string        `json:"operator"`
	Value      interface{}   `json:"value"`
	Expression string        `json:"expression"`
}

// ConditionType represents the type of condition
type ConditionType string

const (
	ConditionTypeVariable   ConditionType = "variable"
	ConditionTypeExpression ConditionType = "expression"
	ConditionTypeTime       ConditionType = "time"
	ConditionTypeEvent      ConditionType = "event"
)

// WorkflowTrigger represents a workflow trigger
type WorkflowTrigger struct {
	ID         string                 `json:"id"`
	Type       TriggerType            `json:"type"`
	Schedule   string                 `json:"schedule,omitempty"`
	Event      string                 `json:"event,omitempty"`
	Conditions []TriggerCondition     `json:"conditions"`
	Parameters map[string]interface{} `json:"parameters"`
	Enabled    bool                   `json:"enabled"`
}

// TriggerType represents the type of trigger
type TriggerType string

const (
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeEvent    TriggerType = "event"
	TriggerTypeManual   TriggerType = "manual"
	TriggerTypeWebhook  TriggerType = "webhook"
)

// TriggerCondition represents a trigger condition
type TriggerCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// WorkflowExecution represents a workflow execution instance
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

// StepExecution represents a step execution instance
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

// ExecutionStatus represents the status of execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "canceled"
	StatusSkipped   ExecutionStatus = "skipped"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	BackoffType BackoffType   `json:"backoff_type"`
	MaxDelay    time.Duration `json:"max_delay"`
}

// BackoffType represents the type of backoff strategy
type BackoffType string

const (
	BackoffTypeFixed       BackoffType = "fixed"
	BackoffTypeLinear      BackoffType = "linear"
	BackoffTypeExponential BackoffType = "exponential"
)

// NewWorkflowOrchestrator creates a new workflow orchestrator
func NewWorkflowOrchestrator(config OrchestratorConfig) *WorkflowOrchestrator {
	return &WorkflowOrchestrator{
		workflows:  make(map[string]*Workflow),
		executions: make(map[string]*WorkflowExecution),
		scheduler:  NewWorkflowScheduler(),
		eventBus:   NewEventBus(),
		config:     config,
	}
}

// RegisterWorkflow registers a new workflow
func (wo *WorkflowOrchestrator) RegisterWorkflow(workflow *Workflow) error {
	wo.mutex.Lock()
	defer wo.mutex.Unlock()

	if workflow.ID == "" {
		workflow.ID = generateWorkflowID()
	}

	workflow.CreatedAt = time.Now()
	workflow.UpdatedAt = time.Now()

	wo.workflows[workflow.ID] = workflow

	// Register triggers with scheduler
	for _, trigger := range workflow.Triggers {
		if trigger.Type == TriggerTypeSchedule && trigger.Enabled {
			// Ignore scheduling errors during registration, they will be logged by scheduler
			_ = wo.scheduler.ScheduleWorkflow(workflow.ID, trigger.Schedule)
		}
	}

	return nil
}

// ExecuteWorkflow executes a workflow
func (wo *WorkflowOrchestrator) ExecuteWorkflow(workflowID string, trigger *WorkflowTrigger, variables map[string]interface{}) (*WorkflowExecution, error) {
	wo.mutex.RLock()
	workflow, exists := wo.workflows[workflowID]
	wo.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	if !workflow.Enabled {
		return nil, fmt.Errorf("workflow is disabled: %s", workflowID)
	}

	// Check concurrent execution limit
	if wo.countRunningExecutions() >= wo.config.MaxConcurrentWorkflows {
		return nil, fmt.Errorf("maximum concurrent workflows reached")
	}

	// Create execution context
	ctx, cancel := context.WithTimeout(context.Background(), workflow.Timeout)
	if workflow.Timeout == 0 {
		ctx, cancel = context.WithTimeout(context.Background(), wo.config.DefaultTimeout)
	}

	execution := &WorkflowExecution{
		ID:         generateExecutionID(),
		WorkflowID: workflowID,
		Status:     StatusPending,
		StartTime:  time.Now(),
		Steps:      make(map[string]*StepExecution),
		Variables:  mergeVariables(workflow.Variables, variables),
		Trigger:    trigger,
		Metadata:   make(map[string]interface{}),
		Context:    ctx,
		CancelFunc: cancel,
	}

	wo.mutex.Lock()
	wo.executions[execution.ID] = execution
	wo.mutex.Unlock()

	// Start execution in goroutine
	go wo.executeWorkflowAsync(execution, workflow)

	return execution, nil
}

// executeWorkflowAsync executes workflow asynchronously
func (wo *WorkflowOrchestrator) executeWorkflowAsync(execution *WorkflowExecution, workflow *Workflow) {
	defer execution.CancelFunc()

	execution.Status = StatusRunning
	wo.eventBus.Publish("workflow.started", execution)

	// Execute workflow steps
	err := wo.executeWorkflowSteps(execution, workflow)

	// Update execution status
	execution.EndTime = time.Now()
	execution.Duration = execution.EndTime.Sub(execution.StartTime)

	if err != nil {
		execution.Status = StatusFailed
		execution.Error = err.Error()
		wo.eventBus.Publish("workflow.failed", execution)
	} else {
		execution.Status = StatusCompleted
		wo.eventBus.Publish("workflow.completed", execution)
	}
}

// executeWorkflowSteps executes all workflow steps
func (wo *WorkflowOrchestrator) executeWorkflowSteps(execution *WorkflowExecution, workflow *Workflow) error {
	// Build dependency graph
	dependencyGraph := wo.buildDependencyGraph(workflow.Steps)

	// Execute steps based on dependencies
	return wo.executeStepsWithDependencies(execution, workflow, dependencyGraph)
}

// buildDependencyGraph builds a dependency graph for workflow steps
func (wo *WorkflowOrchestrator) buildDependencyGraph(steps []WorkflowStep) map[string][]string {
	graph := make(map[string][]string)

	for _, step := range steps {
		graph[step.ID] = step.Dependencies
	}

	return graph
}

// executeStepsWithDependencies executes steps respecting dependencies
func (wo *WorkflowOrchestrator) executeStepsWithDependencies(execution *WorkflowExecution, workflow *Workflow, dependencyGraph map[string][]string) error {
	completed := make(map[string]bool)
	failed := make(map[string]bool)

	for len(completed)+len(failed) < len(workflow.Steps) {
		// Find steps ready to execute
		readySteps := wo.findReadySteps(workflow.Steps, dependencyGraph, completed, failed)

		if len(readySteps) == 0 {
			// No more steps can be executed
			break
		}

		// Execute ready steps in parallel
		var wg sync.WaitGroup
		stepResults := make(chan stepResult, len(readySteps))

		for _, step := range readySteps {
			wg.Add(1)
			go func(s WorkflowStep) {
				defer wg.Done()
				err := wo.executeStep(execution, s)
				stepResults <- stepResult{stepID: s.ID, err: err}
			}(step)
		}

		wg.Wait()
		close(stepResults)

		// Process results
		for result := range stepResults {
			if result.err != nil {
				failed[result.stepID] = true
				// Check if this is a critical failure
				if wo.isCriticalStep(workflow, result.stepID) {
					return fmt.Errorf("critical step failed: %s - %v", result.stepID, result.err)
				}
			} else {
				completed[result.stepID] = true
			}
		}

		// Check for context cancellation
		select {
		case <-execution.Context.Done():
			return execution.Context.Err()
		default:
		}
	}

	return nil
}

// stepResult represents the result of step execution
type stepResult struct {
	stepID string
	err    error
}

// findReadySteps finds steps that are ready to execute
func (wo *WorkflowOrchestrator) findReadySteps(steps []WorkflowStep, dependencyGraph map[string][]string, completed, failed map[string]bool) []WorkflowStep {
	var readySteps []WorkflowStep

	for _, step := range steps {
		// Skip if already completed or failed
		if completed[step.ID] || failed[step.ID] {
			continue
		}

		// Check if all dependencies are completed
		ready := true
		for _, dep := range dependencyGraph[step.ID] {
			if !completed[dep] {
				ready = false
				break
			}
		}

		if ready && wo.evaluateStepConditions(step) {
			readySteps = append(readySteps, step)
		}
	}

	return readySteps
}

// executeStep executes a single workflow step
func (wo *WorkflowOrchestrator) executeStep(execution *WorkflowExecution, step WorkflowStep) error {
	stepExecution := &StepExecution{
		ID:        generateStepExecutionID(),
		StepID:    step.ID,
		Status:    StatusRunning,
		StartTime: time.Now(),
		Output:    make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
	}

	execution.Steps[step.ID] = stepExecution

	// Execute step with retry policy
	var err error
	for attempt := 0; attempt <= step.RetryPolicy.MaxAttempts; attempt++ {
		stepExecution.RetryCount = attempt

		err = wo.executeStepAction(execution, step, stepExecution)
		if err == nil {
			break
		}

		// Wait before retry
		if attempt < step.RetryPolicy.MaxAttempts {
			delay := wo.calculateRetryDelay(step.RetryPolicy, attempt)
			time.Sleep(delay)
		}
	}

	// Update step execution status
	stepExecution.EndTime = time.Now()
	stepExecution.Duration = stepExecution.EndTime.Sub(stepExecution.StartTime)

	if err != nil {
		stepExecution.Status = StatusFailed
		stepExecution.Error = err.Error()

		// Execute on_failure actions
		wo.executeStepActions(execution, step.OnFailure)
	} else {
		stepExecution.Status = StatusCompleted

		// Execute on_success actions
		wo.executeStepActions(execution, step.OnSuccess)
	}

	return err
}

// executeStepAction executes the main action of a step
func (wo *WorkflowOrchestrator) executeStepAction(execution *WorkflowExecution, step WorkflowStep, stepExecution *StepExecution) error {
	switch step.Type {
	case StepTypeTask:
		return wo.executeTaskStep(execution, step, stepExecution)
	case StepTypePlaybook:
		return wo.executePlaybookStep(execution, step, stepExecution)
	case StepTypeWait:
		return wo.executeWaitStep(execution, step, stepExecution)
	case StepTypeCondition:
		return wo.executeConditionStep(execution, step, stepExecution)
	case StepTypeLoop:
		return wo.executeLoopStep(execution, step, stepExecution)
	case StepTypeParallel:
		return wo.executeParallelStep(execution, step, stepExecution)
	case StepTypeNotification:
		return wo.executeNotificationStep(execution, step, stepExecution)
	default:
		return fmt.Errorf("unsupported step type: %s", step.Type)
	}
}

// executeTaskStep executes a task step
func (wo *WorkflowOrchestrator) executeTaskStep(execution *WorkflowExecution, step WorkflowStep, stepExecution *StepExecution) error {
	// This would integrate with the task execution engine
	// For now, simulate task execution
	time.Sleep(100 * time.Millisecond)
	stepExecution.Output["result"] = "task completed"
	return nil
}

// executePlaybookStep executes a playbook step
func (wo *WorkflowOrchestrator) executePlaybookStep(execution *WorkflowExecution, step WorkflowStep, stepExecution *StepExecution) error {
	// This would integrate with the playbook execution engine
	// For now, simulate playbook execution
	time.Sleep(500 * time.Millisecond)
	stepExecution.Output["result"] = "playbook completed"
	return nil
}

// executeWaitStep executes a wait step
func (wo *WorkflowOrchestrator) executeWaitStep(execution *WorkflowExecution, step WorkflowStep, stepExecution *StepExecution) error {
	duration := step.Action.Timeout
	if duration == 0 {
		duration = 1 * time.Second
	}

	select {
	case <-time.After(duration):
		stepExecution.Output["waited"] = duration.String()
		return nil
	case <-execution.Context.Done():
		return execution.Context.Err()
	}
}

// executeConditionStep executes a condition step
func (wo *WorkflowOrchestrator) executeConditionStep(execution *WorkflowExecution, step WorkflowStep, stepExecution *StepExecution) error {
	// Evaluate conditions and set result
	result := wo.evaluateStepConditions(step)
	stepExecution.Output["condition_result"] = result

	if !result {
		stepExecution.Status = StatusSkipped
	}

	return nil
}

// executeLoopStep executes a loop step
func (wo *WorkflowOrchestrator) executeLoopStep(execution *WorkflowExecution, step WorkflowStep, stepExecution *StepExecution) error {
	// This would implement loop logic
	// For now, simulate loop execution
	iterations := 3
	if iter, exists := step.Action.Parameters["iterations"]; exists {
		if iterInt, ok := iter.(int); ok {
			iterations = iterInt
		}
	}

	results := make([]interface{}, iterations)
	for i := 0; i < iterations; i++ {
		// Simulate loop iteration
		time.Sleep(50 * time.Millisecond)
		results[i] = fmt.Sprintf("iteration_%d_result", i)
	}

	stepExecution.Output["loop_results"] = results
	return nil
}

// executeParallelStep executes a parallel step
func (wo *WorkflowOrchestrator) executeParallelStep(execution *WorkflowExecution, step WorkflowStep, stepExecution *StepExecution) error {
	// This would implement parallel execution logic
	// For now, simulate parallel execution
	var wg sync.WaitGroup
	results := make([]interface{}, 3)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			time.Sleep(100 * time.Millisecond)
			results[index] = fmt.Sprintf("parallel_result_%d", index)
		}(i)
	}

	wg.Wait()
	stepExecution.Output["parallel_results"] = results
	return nil
}

// executeNotificationStep executes a notification step
func (wo *WorkflowOrchestrator) executeNotificationStep(execution *WorkflowExecution, step WorkflowStep, stepExecution *StepExecution) error {
	// This would implement notification logic
	// For now, simulate notification
	message := "Workflow notification"
	if msg, exists := step.Action.Parameters["message"]; exists {
		if msgStr, ok := msg.(string); ok {
			message = msgStr
		}
	}

	stepExecution.Output["notification_sent"] = true
	stepExecution.Output["message"] = message
	return nil
}

// executeStepActions executes a list of step actions
func (wo *WorkflowOrchestrator) executeStepActions(execution *WorkflowExecution, actions []StepAction) {
	for _, action := range actions {
		// Actions are best-effort, errors are logged but don't stop execution
		_ = wo.executeAction(execution, action)
	}
}

// executeAction executes a single action
func (wo *WorkflowOrchestrator) executeAction(execution *WorkflowExecution, action StepAction) error {
	switch action.Type {
	case ActionTypeSetVariable:
		if name, exists := action.Parameters["name"]; exists {
			if value, exists := action.Parameters["value"]; exists {
				execution.Variables[name.(string)] = value
			}
		}
	case ActionTypeNotify:
		// Send notification
		wo.eventBus.Publish("workflow.notification", map[string]interface{}{
			"execution_id": execution.ID,
			"message":      action.Parameters["message"],
		})
	}

	return nil
}

// Helper methods

func (wo *WorkflowOrchestrator) evaluateStepConditions(step WorkflowStep) bool {
	// Simplified condition evaluation
	// In a real implementation, this would be more sophisticated
	return true
}

func (wo *WorkflowOrchestrator) isCriticalStep(workflow *Workflow, stepID string) bool {
	// Check if step is marked as critical
	for _, step := range workflow.Steps {
		if step.ID == stepID {
			if critical, exists := step.Metadata["critical"]; exists {
				if criticalBool, ok := critical.(bool); ok {
					return criticalBool
				}
			}
		}
	}
	return false
}

func (wo *WorkflowOrchestrator) calculateRetryDelay(policy RetryPolicy, attempt int) time.Duration {
	switch policy.BackoffType {
	case BackoffTypeFixed:
		return policy.Delay
	case BackoffTypeLinear:
		return policy.Delay * time.Duration(attempt+1)
	case BackoffTypeExponential:
		// Prevent integer overflow by capping the attempt value
		// Max safe value for bit shift is 62 (since 1<<63 would overflow)
		safeAttempt := attempt
		if safeAttempt < 0 {
			safeAttempt = 0
		}
		if safeAttempt > 62 {
			safeAttempt = 62
		}

		// Use uint conversion only after validation to avoid gosec warning
		// safeAttempt is guaranteed to be in range [0, 62]
		var multiplier uint64 = 1 << uint(safeAttempt) // #nosec G115 -- safeAttempt is validated to be in [0,62]

		// Check if multiplier would overflow when converted to int64/time.Duration
		// time.Duration is int64, so max safe value is 1<<63-1
		const maxInt64 = uint64(1<<63 - 1)
		if multiplier > maxInt64 {
			return policy.MaxDelay
		}

		delay := policy.Delay * time.Duration(multiplier) // #nosec G115 -- multiplier is validated to fit in int64
		if delay > policy.MaxDelay || delay < 0 {
			return policy.MaxDelay
		}
		return delay
	default:
		return policy.Delay
	}
}

func (wo *WorkflowOrchestrator) countRunningExecutions() int {
	wo.mutex.RLock()
	defer wo.mutex.RUnlock()

	count := 0
	for _, execution := range wo.executions {
		if execution.Status == StatusRunning {
			count++
		}
	}
	return count
}

// GetWorkflow returns a workflow by ID
func (wo *WorkflowOrchestrator) GetWorkflow(id string) (*Workflow, error) {
	wo.mutex.RLock()
	defer wo.mutex.RUnlock()

	workflow, exists := wo.workflows[id]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}

	return workflow, nil
}

// GetExecution returns an execution by ID
func (wo *WorkflowOrchestrator) GetExecution(id string) (*WorkflowExecution, error) {
	wo.mutex.RLock()
	defer wo.mutex.RUnlock()

	execution, exists := wo.executions[id]
	if !exists {
		return nil, fmt.Errorf("execution not found: %s", id)
	}

	return execution, nil
}

// CancelExecution cancels a running execution
func (wo *WorkflowOrchestrator) CancelExecution(id string) error {
	wo.mutex.RLock()
	execution, exists := wo.executions[id]
	wo.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("execution not found: %s", id)
	}

	if execution.Status == StatusRunning {
		execution.CancelFunc()
		execution.Status = StatusCancelled
		wo.eventBus.Publish("workflow.cancelled", execution)
	}

	return nil
}

// ListWorkflows returns all registered workflows
func (wo *WorkflowOrchestrator) ListWorkflows() []*Workflow {
	wo.mutex.RLock()
	defer wo.mutex.RUnlock()

	workflows := make([]*Workflow, 0, len(wo.workflows))
	for _, workflow := range wo.workflows {
		workflows = append(workflows, workflow)
	}

	return workflows
}

// ListExecutions returns all executions
func (wo *WorkflowOrchestrator) ListExecutions() []*WorkflowExecution {
	wo.mutex.RLock()
	defer wo.mutex.RUnlock()

	executions := make([]*WorkflowExecution, 0, len(wo.executions))
	for _, execution := range wo.executions {
		executions = append(executions, execution)
	}

	return executions
}

// Helper functions for ID generation
func generateWorkflowID() string {
	return fmt.Sprintf("workflow_%d", time.Now().UnixNano())
}

func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

func generateStepExecutionID() string {
	return fmt.Sprintf("step_%d", time.Now().UnixNano())
}

// mergeVariables merges two variable maps
func mergeVariables(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy base variables
	for k, v := range base {
		result[k] = v
	}

	// Override with new variables
	for k, v := range override {
		result[k] = v
	}

	return result
}
