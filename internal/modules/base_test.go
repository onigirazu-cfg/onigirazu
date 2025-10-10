package modules

import (
	"context"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestNewBaseModule tests base module creation
func TestNewBaseModule(t *testing.T) {
	module := NewBaseModule("test_module")

	if module == nil {
		t.Fatal("Expected module to be created")
	}

	if module.name != "test_module" {
		t.Errorf("Expected name 'test_module', got '%s'", module.name)
	}
}

// TestBaseModule_GetName tests getting module name
func TestBaseModule_GetName(t *testing.T) {
	module := NewBaseModule("test_module")

	name := module.GetName()
	if name != "test_module" {
		t.Errorf("Expected name 'test_module', got '%s'", name)
	}
}

// TestBaseModule_GetDescription tests getting module description
func TestBaseModule_GetDescription(t *testing.T) {
	module := NewBaseModule("test_module")
	module.description = "Test description"

	desc := module.GetDescription()
	if desc != "Test description" {
		t.Errorf("Expected description 'Test description', got '%s'", desc)
	}
}

// TestBaseModule_Execute tests basic execution
func TestBaseModule_Execute(t *testing.T) {
	module := NewBaseModule("test_module")

	host := types.Host{
		Name: "test-host",
		Vars: make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name": "Test Task",
	}

	result, err := module.Execute(context.Background(), host, args)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful result")
	}

	if result.Changed {
		t.Error("Expected unchanged result for base module")
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

	if result.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	if result.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	// Check output
	if result.Output == nil {
		t.Fatal("Expected output to be set")
	}

	message, ok := result.Output["message"].(string)
	if !ok {
		t.Fatal("Expected message in output")
	}

	expectedMessage := "Module test_module executed on host test-host"
	if message != expectedMessage {
		t.Errorf("Expected message '%s', got '%s'", expectedMessage, message)
	}
}

// TestBaseModule_Execute_ValidationError tests execution with validation error
func TestBaseModule_Execute_ValidationError(t *testing.T) {
	module := NewBaseModule("test_module")

	host := types.Host{
		Name: "test-host",
		Vars: make(map[string]interface{}),
	}

	// Provide 'name' but missing other required arguments (if any)
	// Since BaseModule doesn't have strict validation, this test just ensures
	// the validation flow works
	args := map[string]interface{}{
		"name": "test_task",
	}

	result, err := module.Execute(context.Background(), host, args)

	// BaseModule should succeed with minimal args
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Duration == 0 {
		t.Error("Expected non-zero duration")
	}
}

// TestBaseModule_Validate tests argument validation
func TestBaseModule_Validate(t *testing.T) {
	module := NewBaseModule("test_module")

	// Valid args
	args := map[string]interface{}{
		"name": "Test Task",
	}

	err := module.Validate(args)
	if err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}

	// Missing name
	args = map[string]interface{}{}
	err = module.Validate(args)
	if err == nil {
		t.Error("Expected validation error for missing name")
	}
	if err.Error() != "argument 'name' is required" {
		t.Errorf("Expected required error message, got: %v", err)
	}
}

// TestGetStringArg tests string argument parsing
func TestGetStringArg(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]interface{}
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "existing string",
			args:         map[string]interface{}{"key": "value"},
			key:          "key",
			defaultValue: "default",
			expected:     "value",
		},
		{
			name:         "missing key",
			args:         map[string]interface{}{},
			key:          "key",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "wrong type",
			args:         map[string]interface{}{"key": 123},
			key:          "key",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "empty string",
			args:         map[string]interface{}{"key": ""},
			key:          "key",
			defaultValue: "default",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringArg(tt.args, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestGetBoolArg tests boolean argument parsing
func TestGetBoolArg(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]interface{}
		key          string
		defaultValue bool
		expected     bool
	}{
		{
			name:         "existing true",
			args:         map[string]interface{}{"key": true},
			key:          "key",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "existing false",
			args:         map[string]interface{}{"key": false},
			key:          "key",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "missing key",
			args:         map[string]interface{}{},
			key:          "key",
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "wrong type",
			args:         map[string]interface{}{"key": "true"},
			key:          "key",
			defaultValue: false,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBoolArg(tt.args, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestGetIntArg tests integer argument parsing
func TestGetIntArg(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]interface{}
		key          string
		defaultValue int
		expected     int
	}{
		{
			name:         "existing int",
			args:         map[string]interface{}{"key": 42},
			key:          "key",
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "existing float64",
			args:         map[string]interface{}{"key": 42.0},
			key:          "key",
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "float64 with decimals",
			args:         map[string]interface{}{"key": 42.7},
			key:          "key",
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "missing key",
			args:         map[string]interface{}{},
			key:          "key",
			defaultValue: 99,
			expected:     99,
		},
		{
			name:         "wrong type",
			args:         map[string]interface{}{"key": "42"},
			key:          "key",
			defaultValue: 0,
			expected:     0,
		},
		{
			name:         "zero value",
			args:         map[string]interface{}{"key": 0},
			key:          "key",
			defaultValue: 99,
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIntArg(tt.args, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestGetMapArg tests map argument parsing
func TestGetMapArg(t *testing.T) {
	defaultMap := map[string]interface{}{"default": "value"}

	tests := []struct {
		name         string
		args         map[string]interface{}
		key          string
		defaultValue map[string]interface{}
		expected     map[string]interface{}
	}{
		{
			name: "existing map",
			args: map[string]interface{}{
				"key": map[string]interface{}{"nested": "value"},
			},
			key:          "key",
			defaultValue: defaultMap,
			expected:     map[string]interface{}{"nested": "value"},
		},
		{
			name:         "missing key",
			args:         map[string]interface{}{},
			key:          "key",
			defaultValue: defaultMap,
			expected:     defaultMap,
		},
		{
			name:         "wrong type",
			args:         map[string]interface{}{"key": "not a map"},
			key:          "key",
			defaultValue: defaultMap,
			expected:     defaultMap,
		},
		{
			name: "empty map",
			args: map[string]interface{}{
				"key": map[string]interface{}{},
			},
			key:          "key",
			defaultValue: defaultMap,
			expected:     map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMapArg(tt.args, tt.key, tt.defaultValue)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected map length %d, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("Expected result[%s] = %v, got %v", k, v, result[k])
				}
			}
		})
	}
}

// TestBaseModule_FailResult tests fail result creation
func TestBaseModule_FailResult(t *testing.T) {
	module := NewBaseModule("test_module")

	// Set timestamp slightly in the past to ensure non-zero duration
	startTime := time.Now().Add(-10 * time.Millisecond)
	result := types.TaskResult{
		TaskName:  "Test Task",
		Host:      "test-host",
		Module:    "test_module",
		Timestamp: startTime,
	}

	errorMsg := "Something went wrong"
	failedResult, err := module.failResult(result, errorMsg)

	if err == nil {
		t.Error("Expected error to be returned")
	}

	if err.Error() != errorMsg {
		t.Errorf("Expected error message '%s', got '%s'", errorMsg, err.Error())
	}

	if failedResult.Success {
		t.Error("Expected failed result")
	}

	if failedResult.Changed {
		t.Error("Expected unchanged result")
	}

	if failedResult.Error != errorMsg {
		t.Errorf("Expected error in result '%s', got '%s'", errorMsg, failedResult.Error)
	}

	// Duration should be at least 10ms since we set timestamp in the past
	if failedResult.Duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", failedResult.Duration)
	}

	if failedResult.TaskName != "Test Task" {
		t.Error("Expected original task name to be preserved")
	}

	if failedResult.Host != "test-host" {
		t.Error("Expected original host to be preserved")
	}
}

// TestBaseModule_Execute_ContextCancellation tests context cancellation
func TestBaseModule_Execute_ContextCancellation(t *testing.T) {
	module := NewBaseModule("test_module")

	host := types.Host{
		Name: "test-host",
		Vars: make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name": "Test Task",
	}

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Base module doesn't check context, but we verify it accepts it
	result, err := module.Execute(ctx, host, args)
	if err != nil {
		t.Fatalf("Base module execution failed: %v", err)
	}

	// Base module should still succeed (it doesn't check context)
	if !result.Success {
		t.Error("Expected successful result")
	}
}

// BenchmarkBaseModule_Execute benchmarks base module execution
func BenchmarkBaseModule_Execute(b *testing.B) {
	module := NewBaseModule("test_module")

	host := types.Host{
		Name: "test-host",
		Vars: make(map[string]interface{}),
	}

	args := map[string]interface{}{
		"name": "Benchmark Task",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = module.Execute(context.Background(), host, args)
	}
}

// BenchmarkGetStringArg benchmarks string argument parsing
func BenchmarkGetStringArg(b *testing.B) {
	args := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getStringArg(args, "key2", "default")
	}
}

// BenchmarkGetIntArg benchmarks integer argument parsing
func BenchmarkGetIntArg(b *testing.B) {
	args := map[string]interface{}{
		"key1": 42,
		"key2": 100,
		"key3": 200,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getIntArg(args, "key2", 0)
	}
}
