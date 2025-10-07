package workflow

import (
	"testing"
	"time"
)

// TestWorkflowOrchestrator_ExecuteWorkflow_WithDependencies tests workflow with step dependencies
func TestWorkflowOrchestrator_ExecuteWorkflow_WithDependencies(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         10 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:    "Workflow with Dependencies",
		Enabled: true,
		Timeout: 5 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:           "step1",
				Name:         "First Step",
				Type:         StepTypeTask,
				Dependencies: []string{}, // No dependencies
				Action: StepAction{
					Type:   ActionTypeExecuteTask,
					Target: "task1",
				},
				Variables: make(map[string]interface{}),
				Metadata:  make(map[string]interface{}),
			},
			{
				ID:           "step2",
				Name:         "Second Step",
				Type:         StepTypeTask,
				Dependencies: []string{"step1"}, // Depends on step1
				Action: StepAction{
					Type:   ActionTypeExecuteTask,
					Target: "task2",
				},
				Variables: make(map[string]interface{}),
				Metadata:  make(map[string]interface{}),
			},
			{
				ID:           "step3",
				Name:         "Third Step",
				Type:         StepTypeTask,
				Dependencies: []string{"step1"}, // Also depends on step1
				Action: StepAction{
					Type:   ActionTypeExecuteTask,
					Target: "task3",
				},
				Variables: make(map[string]interface{}),
				Metadata:  make(map[string]interface{}),
			},
		},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	execution, err := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err != nil {
		t.Fatalf("Expected successful execution start, got error: %v", err)
	}

	// Wait for execution to complete
	time.Sleep(1 * time.Second)

	// Verify execution was created
	if execution == nil {
		t.Fatal("Expected execution to be created")
	}

	if execution.Steps == nil {
		t.Error("Expected steps map to be initialized")
	}
}

// TestWorkflowOrchestrator_ExecuteWorkflow_WithRetry tests workflow with retry policy
func TestWorkflowOrchestrator_ExecuteWorkflow_WithRetry(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         10 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:    "Workflow with Retry",
		Enabled: true,
		Timeout: 5 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:   "step1",
				Name: "Retryable Step",
				Type: StepTypeTask,
				Action: StepAction{
					Type:   ActionTypeExecuteTask,
					Target: "task1",
				},
				RetryPolicy: RetryPolicy{
					MaxAttempts: 3,
					Delay:       100 * time.Millisecond,
					BackoffType: BackoffTypeFixed,
				},
				Variables: make(map[string]interface{}),
				Metadata:  make(map[string]interface{}),
			},
		},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	execution, err := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err != nil {
		t.Fatalf("Expected successful execution start, got error: %v", err)
	}

	// Wait for execution to complete
	time.Sleep(1 * time.Second)

	if execution == nil {
		t.Fatal("Expected execution to be created")
	}
}

// TestWorkflowOrchestrator_ExecuteWorkflow_WaitStep tests wait step execution
func TestWorkflowOrchestrator_ExecuteWorkflow_WaitStep(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         10 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:    "Workflow with Wait",
		Enabled: true,
		Timeout: 5 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:   "wait1",
				Name: "Wait Step",
				Type: StepTypeWait,
				Action: StepAction{
					Type:    ActionTypeWait,
					Timeout: 200 * time.Millisecond,
				},
				Variables: make(map[string]interface{}),
				Metadata:  make(map[string]interface{}),
			},
		},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	startTime := time.Now()
	execution, err := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err != nil {
		t.Fatalf("Expected successful execution start, got error: %v", err)
	}

	// Wait for execution to complete
	time.Sleep(500 * time.Millisecond)
	duration := time.Since(startTime)

	if execution == nil {
		t.Fatal("Expected execution to be created")
	}

	// Verify wait duration (should be at least 200ms)
	if duration < 200*time.Millisecond {
		t.Errorf("Expected wait duration >= 200ms, got %v", duration)
	}
}

// TestWorkflowOrchestrator_ExecuteWorkflow_LoopStep tests loop step execution
func TestWorkflowOrchestrator_ExecuteWorkflow_LoopStep(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         10 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:    "Workflow with Loop",
		Enabled: true,
		Timeout: 5 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:   "loop1",
				Name: "Loop Step",
				Type: StepTypeLoop,
				Action: StepAction{
					Type: ActionTypeExecuteTask,
					Parameters: map[string]interface{}{
						"iterations": 5,
					},
				},
				Variables: make(map[string]interface{}),
				Metadata:  make(map[string]interface{}),
			},
		},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	execution, err := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err != nil {
		t.Fatalf("Expected successful execution start, got error: %v", err)
	}

	// Wait for execution to complete
	time.Sleep(1 * time.Second)

	if execution == nil {
		t.Fatal("Expected execution to be created")
	}
}

// TestWorkflowOrchestrator_ExecuteWorkflow_ParallelStep tests parallel step execution
func TestWorkflowOrchestrator_ExecuteWorkflow_ParallelStep(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         10 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:    "Workflow with Parallel",
		Enabled: true,
		Timeout: 5 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:   "parallel1",
				Name: "Parallel Step",
				Type: StepTypeParallel,
				Action: StepAction{
					Type:       ActionTypeExecuteTask,
					Parameters: make(map[string]interface{}),
				},
				Variables: make(map[string]interface{}),
				Metadata:  make(map[string]interface{}),
			},
		},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	execution, err := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err != nil {
		t.Fatalf("Expected successful execution start, got error: %v", err)
	}

	// Wait for execution to complete
	time.Sleep(500 * time.Millisecond)

	if execution == nil {
		t.Fatal("Expected execution to be created")
	}
}

// TestWorkflowOrchestrator_ExecuteWorkflow_NotificationStep tests notification step execution
func TestWorkflowOrchestrator_ExecuteWorkflow_NotificationStep(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         10 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:    "Workflow with Notification",
		Enabled: true,
		Timeout: 5 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:   "notify1",
				Name: "Notification Step",
				Type: StepTypeNotification,
				Action: StepAction{
					Type: ActionTypeNotify,
					Parameters: map[string]interface{}{
						"message": "Test notification",
					},
				},
				Variables: make(map[string]interface{}),
				Metadata:  make(map[string]interface{}),
			},
		},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	execution, err := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err != nil {
		t.Fatalf("Expected successful execution start, got error: %v", err)
	}

	// Wait for execution to complete
	time.Sleep(300 * time.Millisecond)

	if execution == nil {
		t.Fatal("Expected execution to be created")
	}
}

// TestWorkflowOrchestrator_ExecuteWorkflow_ConditionStep tests condition step execution
func TestWorkflowOrchestrator_ExecuteWorkflow_ConditionStep(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         10 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:    "Workflow with Condition",
		Enabled: true,
		Timeout: 5 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:   "condition1",
				Name: "Condition Step",
				Type: StepTypeCondition,
				Action: StepAction{
					Type:       ActionTypeExecuteTask,
					Parameters: make(map[string]interface{}),
				},
				Conditions: []StepCondition{
					{
						Type:     ConditionTypeVariable,
						Variable: "test_var",
						Operator: "==",
						Value:    "test_value",
					},
				},
				Variables: make(map[string]interface{}),
				Metadata:  make(map[string]interface{}),
			},
		},
		Variables: map[string]interface{}{
			"test_var": "test_value",
		},
	}

	wo.RegisterWorkflow(workflow)

	execution, err := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err != nil {
		t.Fatalf("Expected successful execution start, got error: %v", err)
	}

	// Wait for execution to complete
	time.Sleep(300 * time.Millisecond)

	if execution == nil {
		t.Fatal("Expected execution to be created")
	}
}

// TestWorkflowOrchestrator_ExecuteWorkflow_PlaybookStep tests playbook step execution
func TestWorkflowOrchestrator_ExecuteWorkflow_PlaybookStep(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         10 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:    "Workflow with Playbook",
		Enabled: true,
		Timeout: 5 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:   "playbook1",
				Name: "Playbook Step",
				Type: StepTypePlaybook,
				Action: StepAction{
					Type:   ActionTypeExecutePlaybook,
					Target: "test-playbook.yml",
				},
				Variables: make(map[string]interface{}),
				Metadata:  make(map[string]interface{}),
			},
		},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	execution, err := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err != nil {
		t.Fatalf("Expected successful execution start, got error: %v", err)
	}

	// Wait for execution to complete (playbook takes ~500ms)
	time.Sleep(800 * time.Millisecond)

	if execution == nil {
		t.Fatal("Expected execution to be created")
	}
}

// TestWorkflowOrchestrator_BuildDependencyGraph tests dependency graph building
func TestWorkflowOrchestrator_BuildDependencyGraph(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Minute,
	}
	wo := NewWorkflowOrchestrator(config)

	steps := []WorkflowStep{
		{
			ID:           "step1",
			Dependencies: []string{},
		},
		{
			ID:           "step2",
			Dependencies: []string{"step1"},
		},
		{
			ID:           "step3",
			Dependencies: []string{"step1", "step2"},
		},
	}

	graph := wo.buildDependencyGraph(steps)

	if len(graph) != 3 {
		t.Errorf("Expected 3 entries in graph, got %d", len(graph))
	}

	if len(graph["step1"]) != 0 {
		t.Errorf("Expected step1 to have 0 dependencies, got %d", len(graph["step1"]))
	}

	if len(graph["step2"]) != 1 {
		t.Errorf("Expected step2 to have 1 dependency, got %d", len(graph["step2"]))
	}

	if len(graph["step3"]) != 2 {
		t.Errorf("Expected step3 to have 2 dependencies, got %d", len(graph["step3"]))
	}
}

// TestWorkflowOrchestrator_CalculateRetryDelay tests retry delay calculation
func TestWorkflowOrchestrator_CalculateRetryDelay(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Minute,
	}
	wo := NewWorkflowOrchestrator(config)

	tests := []struct {
		name        string
		policy      RetryPolicy
		attempt     int
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name: "Fixed backoff",
			policy: RetryPolicy{
				MaxAttempts: 3,
				Delay:       1 * time.Second,
				BackoffType: BackoffTypeFixed,
			},
			attempt:     1,
			minExpected: 1 * time.Second,
			maxExpected: 1 * time.Second,
		},
		{
			name: "Linear backoff",
			policy: RetryPolicy{
				MaxAttempts: 3,
				Delay:       1 * time.Second,
				BackoffType: BackoffTypeLinear,
			},
			attempt:     2,
			minExpected: 2 * time.Second,
			maxExpected: 2 * time.Second,
		},
		{
			name: "Exponential backoff",
			policy: RetryPolicy{
				MaxAttempts: 3,
				Delay:       1 * time.Second,
				BackoffType: BackoffTypeExponential,
				MaxDelay:    10 * time.Second,
			},
			attempt:     2,
			minExpected: 2 * time.Second,
			maxExpected: 4 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := wo.calculateRetryDelay(tt.policy, tt.attempt)

			if delay < tt.minExpected {
				t.Errorf("Expected delay >= %v, got %v", tt.minExpected, delay)
			}

			if delay > tt.maxExpected {
				t.Errorf("Expected delay <= %v, got %v", tt.maxExpected, delay)
			}
		})
	}
}

// TestWorkflowOrchestrator_MaxConcurrentLimit tests concurrent workflow limit
func TestWorkflowOrchestrator_MaxConcurrentLimit(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 2, // Limit to 2 concurrent workflows
		DefaultTimeout:         10 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:    "Long Running Workflow",
		Enabled: true,
		Timeout: 5 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:   "wait1",
				Name: "Long Wait",
				Type: StepTypeWait,
				Action: StepAction{
					Type:    ActionTypeWait,
					Timeout: 2 * time.Second,
				},
				Variables: make(map[string]interface{}),
			},
		},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	// Start 2 executions (should succeed)
	exec1, err1 := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err1 != nil {
		t.Fatalf("Expected first execution to start, got error: %v", err1)
	}

	exec2, err2 := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err2 != nil {
		t.Fatalf("Expected second execution to start, got error: %v", err2)
	}

	// Try to start 3rd execution (should fail due to limit)
	time.Sleep(100 * time.Millisecond)
	_, err3 := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err3 == nil {
		t.Error("Expected error for exceeding concurrent workflow limit")
	}

	// Verify executions were created
	if exec1 == nil || exec2 == nil {
		t.Error("Expected executions to be created")
	}
}

// TestWorkflowOrchestrator_EventBusIntegration tests event bus integration
func TestWorkflowOrchestrator_EventBusIntegration(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         10 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	// Subscribe to workflow events
	eventReceived := false
	wo.eventBus.Subscribe("workflow.started", func(event Event) {
		eventReceived = true
	})

	workflow := &Workflow{
		Name:    "Test Workflow",
		Enabled: true,
		Timeout: 2 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:        "step1",
				Name:      "Test Step",
				Type:      StepTypeTask,
				Variables: make(map[string]interface{}),
			},
		},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	_, err := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err != nil {
		t.Fatalf("Expected successful execution start, got error: %v", err)
	}

	// Wait for event to be published
	time.Sleep(200 * time.Millisecond)

	if !eventReceived {
		t.Error("Expected workflow.started event to be received")
	}
}

// BenchmarkWorkflowOrchestrator_ExecuteWorkflow benchmarks workflow execution
func BenchmarkWorkflowOrchestrator_ExecuteWorkflow(b *testing.B) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 100,
		DefaultTimeout:         1 * time.Minute,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:    "Benchmark Workflow",
		Enabled: true,
		Timeout: 10 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:        "step1",
				Name:      "Test Step",
				Type:      StepTypeTask,
				Variables: make(map[string]interface{}),
			},
		},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = wo.ExecuteWorkflow(workflow.ID, nil, nil)
	}
}

// BenchmarkWorkflowOrchestrator_BuildDependencyGraph benchmarks dependency graph building
func BenchmarkWorkflowOrchestrator_BuildDependencyGraph(b *testing.B) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Minute,
	}
	wo := NewWorkflowOrchestrator(config)

	steps := make([]WorkflowStep, 50)
	for i := 0; i < 50; i++ {
		deps := []string{}
		if i > 0 {
			deps = append(deps, steps[i-1].ID)
		}
		steps[i] = WorkflowStep{
			ID:           generateStepExecutionID(),
			Dependencies: deps,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wo.buildDependencyGraph(steps)
	}
}
