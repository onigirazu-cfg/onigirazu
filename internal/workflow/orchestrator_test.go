package workflow

import (
	"testing"
	"time"
)

// TestNewWorkflowOrchestrator tests orchestrator creation
func TestNewWorkflowOrchestrator(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Minute,
		EnableMetrics:          true,
		EnableAudit:            true,
	}

	wo := NewWorkflowOrchestrator(config)

	if wo == nil {
		t.Fatal("Expected orchestrator to be created")
	}

	if wo.workflows == nil {
		t.Error("Expected workflows map to be initialized")
	}

	if wo.executions == nil {
		t.Error("Expected executions map to be initialized")
	}

	if wo.scheduler == nil {
		t.Error("Expected scheduler to be initialized")
	}

	if wo.eventBus == nil {
		t.Error("Expected event bus to be initialized")
	}

	if wo.config.MaxConcurrentWorkflows != 10 {
		t.Errorf("Expected max concurrent workflows 10, got %d", wo.config.MaxConcurrentWorkflows)
	}
}

// TestWorkflowOrchestrator_RegisterWorkflow tests workflow registration
func TestWorkflowOrchestrator_RegisterWorkflow(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Minute,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:        "Test Workflow",
		Description: "A test workflow",
		Version:     "1.0.0",
		Enabled:     true,
		Steps:       []WorkflowStep{},
		Variables:   make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}

	err := wo.RegisterWorkflow(workflow)
	if err != nil {
		t.Fatalf("Expected successful registration, got error: %v", err)
	}

	if workflow.ID == "" {
		t.Error("Expected workflow ID to be generated")
	}

	if workflow.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if workflow.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// Verify workflow was added to map
	wo.mutex.RLock()
	_, exists := wo.workflows[workflow.ID]
	wo.mutex.RUnlock()

	if !exists {
		t.Error("Expected workflow to be in workflows map")
	}
}

// TestWorkflowOrchestrator_RegisterWorkflow_WithID tests registration with existing ID
func TestWorkflowOrchestrator_RegisterWorkflow_WithID(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Minute,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		ID:          "custom-workflow-id",
		Name:        "Test Workflow",
		Description: "A test workflow",
		Enabled:     true,
		Steps:       []WorkflowStep{},
		Variables:   make(map[string]interface{}),
	}

	err := wo.RegisterWorkflow(workflow)
	if err != nil {
		t.Fatalf("Expected successful registration, got error: %v", err)
	}

	if workflow.ID != "custom-workflow-id" {
		t.Errorf("Expected workflow ID 'custom-workflow-id', got '%s'", workflow.ID)
	}
}

// TestWorkflowOrchestrator_GetWorkflow tests workflow retrieval
func TestWorkflowOrchestrator_GetWorkflow(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Minute,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:      "Test Workflow",
		Enabled:   true,
		Steps:     []WorkflowStep{},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	// Get workflow
	retrieved, err := wo.GetWorkflow(workflow.ID)
	if err != nil {
		t.Fatalf("Expected to get workflow, got error: %v", err)
	}

	if retrieved.ID != workflow.ID {
		t.Errorf("Expected workflow ID '%s', got '%s'", workflow.ID, retrieved.ID)
	}

	if retrieved.Name != workflow.Name {
		t.Errorf("Expected workflow name '%s', got '%s'", workflow.Name, retrieved.Name)
	}
}

// TestWorkflowOrchestrator_GetWorkflow_NonExistent tests getting non-existent workflow
func TestWorkflowOrchestrator_GetWorkflow_NonExistent(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Minute,
	}
	wo := NewWorkflowOrchestrator(config)

	_, err := wo.GetWorkflow("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent workflow")
	}
}

// TestWorkflowOrchestrator_ListWorkflows tests listing workflows
func TestWorkflowOrchestrator_ListWorkflows(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Minute,
	}
	wo := NewWorkflowOrchestrator(config)

	// Empty initially
	workflows := wo.ListWorkflows()
	if len(workflows) != 0 {
		t.Errorf("Expected 0 workflows, got %d", len(workflows))
	}

	// Register workflows
	for i := 0; i < 3; i++ {
		workflow := &Workflow{
			Name:      "Test Workflow",
			Enabled:   true,
			Steps:     []WorkflowStep{},
			Variables: make(map[string]interface{}),
		}
		wo.RegisterWorkflow(workflow)
	}

	workflows = wo.ListWorkflows()
	if len(workflows) != 3 {
		t.Errorf("Expected 3 workflows, got %d", len(workflows))
	}
}

// TestWorkflowOrchestrator_ExecuteWorkflow_Disabled tests executing disabled workflow
func TestWorkflowOrchestrator_ExecuteWorkflow_Disabled(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Minute,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:      "Test Workflow",
		Enabled:   false, // Disabled
		Steps:     []WorkflowStep{},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	_, err := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err == nil {
		t.Error("Expected error for disabled workflow")
	}
}

// TestWorkflowOrchestrator_ExecuteWorkflow_NonExistent tests executing non-existent workflow
func TestWorkflowOrchestrator_ExecuteWorkflow_NonExistent(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Minute,
	}
	wo := NewWorkflowOrchestrator(config)

	_, err := wo.ExecuteWorkflow("non-existent", nil, nil)
	if err == nil {
		t.Error("Expected error for non-existent workflow")
	}
}

// TestWorkflowOrchestrator_ExecuteWorkflow_Simple tests simple workflow execution
func TestWorkflowOrchestrator_ExecuteWorkflow_Simple(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:    "Test Workflow",
		Enabled: true,
		Timeout: 2 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:   "step1",
				Name: "Test Step",
				Type: StepTypeTask,
				Action: StepAction{
					Type:   ActionTypeExecuteTask,
					Target: "test-task",
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

	if execution == nil {
		t.Fatal("Expected execution to be created")
	}

	if execution.ID == "" {
		t.Error("Expected execution ID to be generated")
	}

	if execution.WorkflowID != workflow.ID {
		t.Errorf("Expected workflow ID '%s', got '%s'", workflow.ID, execution.WorkflowID)
	}

	if execution.Status != StatusPending {
		t.Errorf("Expected status pending, got '%s'", execution.Status)
	}

	if execution.Context == nil {
		t.Error("Expected context to be set")
	}

	if execution.CancelFunc == nil {
		t.Error("Expected cancel func to be set")
	}

	// Wait a bit for execution to start
	time.Sleep(200 * time.Millisecond)
}

// TestWorkflowOrchestrator_ListExecutions tests listing executions
func TestWorkflowOrchestrator_ListExecutions(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	// Empty initially
	executions := wo.ListExecutions()
	if len(executions) != 0 {
		t.Errorf("Expected 0 executions, got %d", len(executions))
	}

	// Create workflow
	workflow := &Workflow{
		Name:      "Test Workflow",
		Enabled:   true,
		Timeout:   2 * time.Second,
		Steps:     []WorkflowStep{},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	// Execute multiple times
	for i := 0; i < 3; i++ {
		_, err := wo.ExecuteWorkflow(workflow.ID, nil, nil)
		if err != nil {
			t.Fatalf("Expected successful execution, got error: %v", err)
		}
	}

	executions = wo.ListExecutions()
	if len(executions) != 3 {
		t.Errorf("Expected 3 executions, got %d", len(executions))
	}
}

// TestMergeVariables tests variable merging
func TestMergeVariables(t *testing.T) {
	base := map[string]interface{}{
		"var1": "value1",
		"var2": "value2",
		"var3": "value3",
	}

	override := map[string]interface{}{
		"var2": "overridden",
		"var4": "value4",
	}

	result := mergeVariables(base, override)

	// Check base variables
	if result["var1"] != "value1" {
		t.Errorf("Expected var1='value1', got '%v'", result["var1"])
	}

	// Check overridden variable
	if result["var2"] != "overridden" {
		t.Errorf("Expected var2='overridden', got '%v'", result["var2"])
	}

	// Check unchanged variable
	if result["var3"] != "value3" {
		t.Errorf("Expected var3='value3', got '%v'", result["var3"])
	}

	// Check new variable
	if result["var4"] != "value4" {
		t.Errorf("Expected var4='value4', got '%v'", result["var4"])
	}

	// Verify original maps unchanged
	if base["var2"] != "value2" {
		t.Error("Expected base map to be unchanged")
	}
}

// TestMergeVariables_NilMaps tests merging with nil maps
func TestMergeVariables_NilMaps(t *testing.T) {
	base := map[string]interface{}{
		"var1": "value1",
	}

	result := mergeVariables(base, nil)
	if len(result) != 1 {
		t.Errorf("Expected 1 variable, got %d", len(result))
	}

	result = mergeVariables(nil, base)
	if len(result) != 1 {
		t.Errorf("Expected 1 variable, got %d", len(result))
	}

	result = mergeVariables(nil, nil)
	if len(result) != 0 {
		t.Errorf("Expected 0 variables, got %d", len(result))
	}
}

// TestGenerateWorkflowID tests workflow ID generation
func TestGenerateWorkflowID(t *testing.T) {
	id1 := generateWorkflowID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateWorkflowID()

	if id1 == "" {
		t.Error("Expected non-empty workflow ID")
	}

	if id2 == "" {
		t.Error("Expected non-empty workflow ID")
	}

	if id1 == id2 {
		t.Error("Expected unique workflow IDs")
	}
}

// TestGenerateExecutionID tests execution ID generation
func TestGenerateExecutionID(t *testing.T) {
	id1 := generateExecutionID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateExecutionID()

	if id1 == "" {
		t.Error("Expected non-empty execution ID")
	}

	if id2 == "" {
		t.Error("Expected non-empty execution ID")
	}

	if id1 == id2 {
		t.Error("Expected unique execution IDs")
	}
}

// TestGenerateStepExecutionID tests step execution ID generation
func TestGenerateStepExecutionID(t *testing.T) {
	id1 := generateStepExecutionID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateStepExecutionID()

	if id1 == "" {
		t.Error("Expected non-empty step execution ID")
	}

	if id2 == "" {
		t.Error("Expected non-empty step execution ID")
	}

	if id1 == id2 {
		t.Error("Expected unique step execution IDs")
	}
}

// TestWorkflowOrchestrator_CancelExecution tests execution cancellation
func TestWorkflowOrchestrator_CancelExecution(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         10 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	workflow := &Workflow{
		Name:    "Test Workflow",
		Enabled: true,
		Timeout: 10 * time.Second,
		Steps: []WorkflowStep{
			{
				ID:        "step1",
				Name:      "Long Step",
				Type:      StepTypeWait,
				Variables: make(map[string]interface{}),
			},
		},
		Variables: make(map[string]interface{}),
	}

	wo.RegisterWorkflow(workflow)

	execution, err := wo.ExecuteWorkflow(workflow.ID, nil, nil)
	if err != nil {
		t.Fatalf("Expected successful execution start, got error: %v", err)
	}

	// Wait for execution to start
	time.Sleep(100 * time.Millisecond)

	// Cancel execution
	err = wo.CancelExecution(execution.ID)
	if err != nil {
		t.Fatalf("Expected successful cancellation, got error: %v", err)
	}

	// Verify status
	if execution.Status != StatusCancelled {
		t.Errorf("Expected status cancelled, got '%s'", execution.Status)
	}
}

// TestWorkflowOrchestrator_CancelExecution_NonExistent tests cancelling non-existent execution
func TestWorkflowOrchestrator_CancelExecution_NonExistent(t *testing.T) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 10,
		DefaultTimeout:         5 * time.Second,
	}
	wo := NewWorkflowOrchestrator(config)

	err := wo.CancelExecution("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent execution")
	}
}

// BenchmarkWorkflowOrchestrator_RegisterWorkflow benchmarks workflow registration
func BenchmarkWorkflowOrchestrator_RegisterWorkflow(b *testing.B) {
	config := OrchestratorConfig{
		MaxConcurrentWorkflows: 100,
		DefaultTimeout:         5 * time.Minute,
	}
	wo := NewWorkflowOrchestrator(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		workflow := &Workflow{
			Name:      "Benchmark Workflow",
			Enabled:   true,
			Steps:     []WorkflowStep{},
			Variables: make(map[string]interface{}),
		}
		_ = wo.RegisterWorkflow(workflow)
	}
}

// BenchmarkMergeVariables benchmarks variable merging
func BenchmarkMergeVariables(b *testing.B) {
	base := map[string]interface{}{
		"var1": "value1",
		"var2": "value2",
		"var3": "value3",
	}

	override := map[string]interface{}{
		"var2": "overridden",
		"var4": "value4",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mergeVariables(base, override)
	}
}
