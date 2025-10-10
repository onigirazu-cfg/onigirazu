package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// MockModule is a simple mock module for testing
type MockModule struct {
	*BaseModule
	executeFunc func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
}

func NewMockModule(name string) *MockModule {
	return &MockModule{
		BaseModule: NewBaseModule(name),
	}
}

func (m *MockModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, host, args)
	}
	return m.BaseModule.Execute(ctx, host, args)
}

// TestNewRegistry tests registry creation
func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("Expected registry to be created")
	}

	if registry.modules == nil {
		t.Fatal("Expected modules map to be initialized")
	}

	// Check that built-in modules are registered
	modules := registry.ListModules()
	if len(modules) == 0 {
		t.Error("Expected built-in modules to be registered")
	}

	// Check for some known modules
	expectedModules := []string{"file", "copy", "command", "shell", "debug", "package"}
	for _, name := range expectedModules {
		_, err := registry.GetModule(name)
		if err != nil {
			t.Errorf("Expected module '%s' to be registered, got error: %v", name, err)
		}
	}
}

// TestRegistry_RegisterModule tests module registration
func TestRegistry_RegisterModule(t *testing.T) {
	registry := &Registry{
		modules: make(map[string]types.Module),
	}

	mockModule := NewMockModule("test_module")
	registry.RegisterModule(mockModule)

	module, err := registry.GetModule("test_module")
	if err != nil {
		t.Fatalf("Expected module to be registered, got error: %v", err)
	}

	if module.GetName() != "test_module" {
		t.Errorf("Expected module name 'test_module', got '%s'", module.GetName())
	}
}

// TestRegistry_Register tests interface method for registration
func TestRegistry_Register(t *testing.T) {
	registry := &Registry{
		modules: make(map[string]types.Module),
	}

	mockModule := NewMockModule("test_module")

	// Test successful registration
	err := registry.Register("test_module", mockModule)
	if err != nil {
		t.Fatalf("Expected successful registration, got error: %v", err)
	}

	// Test duplicate registration
	err = registry.Register("test_module", mockModule)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
	if err.Error() != "module 'test_module' already registered" {
		t.Errorf("Expected duplicate error message, got: %v", err)
	}
}

// TestRegistry_GetModule tests module retrieval
func TestRegistry_GetModule(t *testing.T) {
	registry := &Registry{
		modules: make(map[string]types.Module),
	}

	mockModule := NewMockModule("test_module")
	registry.RegisterModule(mockModule)

	// Test successful retrieval
	module, err := registry.GetModule("test_module")
	if err != nil {
		t.Fatalf("Expected to get module, got error: %v", err)
	}
	if module.GetName() != "test_module" {
		t.Errorf("Expected module name 'test_module', got '%s'", module.GetName())
	}

	// Test non-existent module
	_, err = registry.GetModule("non_existent")
	if err == nil {
		t.Error("Expected error for non-existent module")
	}
	if err.Error() != "module 'non_existent' not found" {
		t.Errorf("Expected not found error, got: %v", err)
	}
}

// TestRegistry_Get tests interface method for retrieval
func TestRegistry_Get(t *testing.T) {
	registry := &Registry{
		modules: make(map[string]types.Module),
	}

	mockModule := NewMockModule("test_module")
	registry.RegisterModule(mockModule)

	// Test successful retrieval
	executor, err := registry.Get("test_module")
	if err != nil {
		t.Fatalf("Expected to get module, got error: %v", err)
	}
	if executor == nil {
		t.Error("Expected non-nil executor")
	}

	// Test non-existent module
	_, err = registry.Get("non_existent")
	if err == nil {
		t.Error("Expected error for non-existent module")
	}
}

// TestRegistry_ListModules tests listing all modules
func TestRegistry_ListModules(t *testing.T) {
	registry := &Registry{
		modules: make(map[string]types.Module),
	}

	// Empty registry
	modules := registry.ListModules()
	if len(modules) != 0 {
		t.Errorf("Expected empty list, got %d modules", len(modules))
	}

	// Add modules
	registry.RegisterModule(NewMockModule("module1"))
	registry.RegisterModule(NewMockModule("module2"))
	registry.RegisterModule(NewMockModule("module3"))

	modules = registry.ListModules()
	if len(modules) != 3 {
		t.Errorf("Expected 3 modules, got %d", len(modules))
	}

	// Check that all modules are present
	moduleMap := make(map[string]bool)
	for _, name := range modules {
		moduleMap[name] = true
	}

	expectedModules := []string{"module1", "module2", "module3"}
	for _, name := range expectedModules {
		if !moduleMap[name] {
			t.Errorf("Expected module '%s' in list", name)
		}
	}
}

// TestRegistry_List tests interface method for listing
func TestRegistry_List(t *testing.T) {
	registry := &Registry{
		modules: make(map[string]types.Module),
	}

	registry.RegisterModule(NewMockModule("module1"))
	registry.RegisterModule(NewMockModule("module2"))

	modules := registry.List()
	if len(modules) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(modules))
	}
}

// TestRegistry_Unregister tests module unregistration
func TestRegistry_Unregister(t *testing.T) {
	registry := &Registry{
		modules: make(map[string]types.Module),
	}

	mockModule := NewMockModule("test_module")
	registry.RegisterModule(mockModule)

	// Verify module exists
	_, err := registry.GetModule("test_module")
	if err != nil {
		t.Fatalf("Expected module to exist, got error: %v", err)
	}

	// Unregister module
	err = registry.Unregister("test_module")
	if err != nil {
		t.Fatalf("Expected successful unregistration, got error: %v", err)
	}

	// Verify module is gone
	_, err = registry.GetModule("test_module")
	if err == nil {
		t.Error("Expected error after unregistration")
	}

	// Test unregistering non-existent module
	err = registry.Unregister("non_existent")
	if err == nil {
		t.Error("Expected error for unregistering non-existent module")
	}
	if err.Error() != "module 'non_existent' not found" {
		t.Errorf("Expected not found error, got: %v", err)
	}
}

// TestRegistry_ExecuteTask tests task execution through registry
func TestRegistry_ExecuteTask(t *testing.T) {
	registry := &Registry{
		modules: make(map[string]types.Module),
	}

	// Create mock module with custom execute function
	mockModule := NewMockModule("test_module")
	mockModule.executeFunc = func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
		return types.TaskResult{
			TaskName: args["name"].(string),
			Host:     host.Name,
			Module:   "test_module",
			Success:  true,
			Changed:  true,
			Output: map[string]interface{}{
				"custom": "output",
			},
		}, nil
	}
	registry.RegisterModule(mockModule)

	// Create test task
	task := &types.Task{
		Name:   "Test Task",
		Module: "test_module",
		Args: map[string]interface{}{
			"arg1": "value1",
		},
	}

	host := types.Host{
		Name: "test-host",
		Vars: make(map[string]interface{}),
	}

	variables := map[string]interface{}{
		"var1": "varvalue1",
	}

	// Execute task
	result, err := registry.ExecuteTask(context.Background(), task, host, variables)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful result")
	}

	if result.TaskName != "Test Task" {
		t.Errorf("Expected task name 'Test Task', got '%s'", result.TaskName)
	}

	if result.Host != "test-host" {
		t.Errorf("Expected host 'test-host', got '%s'", result.Host)
	}

	if result.Module != "test_module" {
		t.Errorf("Expected module 'test_module', got '%s'", result.Module)
	}

	// Check custom output
	if output, ok := result.Output["custom"]; !ok || output != "output" {
		t.Error("Expected custom output in result")
	}
}

// TestRegistry_ExecuteTask_NonExistentModule tests execution with non-existent module
func TestRegistry_ExecuteTask_NonExistentModule(t *testing.T) {
	registry := &Registry{
		modules: make(map[string]types.Module),
	}

	task := &types.Task{
		Name:   "Test Task",
		Module: "non_existent",
		Args:   make(map[string]interface{}),
	}

	host := types.Host{
		Name: "test-host",
		Vars: make(map[string]interface{}),
	}

	_, err := registry.ExecuteTask(context.Background(), task, host, nil)
	if err == nil {
		t.Error("Expected error for non-existent module")
	}
	if err.Error() != "module 'non_existent' not found" {
		t.Errorf("Expected not found error, got: %v", err)
	}
}

// TestRegistry_ExecuteTask_ArgumentMerging tests argument merging logic
func TestRegistry_ExecuteTask_ArgumentMerging(t *testing.T) {
	registry := &Registry{
		modules: make(map[string]types.Module),
	}

	// Track received arguments
	var receivedArgs map[string]interface{}

	mockModule := NewMockModule("test_module")
	mockModule.executeFunc = func(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
		receivedArgs = args
		return types.TaskResult{
			TaskName: args["name"].(string),
			Host:     host.Name,
			Module:   "test_module",
			Success:  true,
		}, nil
	}
	registry.RegisterModule(mockModule)

	task := &types.Task{
		Name:   "Test Task",
		Module: "test_module",
		Args: map[string]interface{}{
			"task_arg": "task_value",
		},
	}

	host := types.Host{
		Name: "test-host",
		Vars: make(map[string]interface{}),
	}

	variables := map[string]interface{}{
		"var_arg": "var_value",
	}

	_, err := registry.ExecuteTask(context.Background(), task, host, variables)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	// Check that task name was added
	if receivedArgs["name"] != "Test Task" {
		t.Errorf("Expected name 'Test Task', got '%v'", receivedArgs["name"])
	}

	// Check that task args were passed
	if receivedArgs["task_arg"] != "task_value" {
		t.Errorf("Expected task_arg 'task_value', got '%v'", receivedArgs["task_arg"])
	}

	// Check that variables were passed
	if receivedArgs["var_arg"] != "var_value" {
		t.Errorf("Expected var_arg 'var_value', got '%v'", receivedArgs["var_arg"])
	}

	// Test that task args take precedence over variables
	task.Args["conflict"] = "task_wins"
	variables["conflict"] = "var_loses"

	_, err = registry.ExecuteTask(context.Background(), task, host, variables)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	if receivedArgs["conflict"] != "task_wins" {
		t.Errorf("Expected task args to take precedence, got '%v'", receivedArgs["conflict"])
	}
}

// BenchmarkRegistry_GetModule benchmarks module retrieval
func BenchmarkRegistry_GetModule(b *testing.B) {
	registry := NewRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.GetModule("file")
	}
}

// BenchmarkRegistry_ExecuteTask benchmarks task execution
func BenchmarkRegistry_ExecuteTask(b *testing.B) {
	registry := NewRegistry()

	task := &types.Task{
		Name:   "Benchmark Task",
		Module: "debug",
		Args: map[string]interface{}{
			"msg": "test message",
		},
	}

	host := types.Host{
		Name: "test-host",
		Vars: make(map[string]interface{}),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.ExecuteTask(context.Background(), task, host, nil)
	}
}
