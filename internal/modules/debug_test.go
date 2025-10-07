package modules

import (
	"context"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestDebugModule_Execute_WithMsg tests debug module with msg parameter
func TestDebugModule_Execute_WithMsg(t *testing.T) {
	module := NewDebugModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name": "test_debug",
		"msg":  "Hello, World!",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Changed {
		t.Errorf("Expected changed=false, got true")
	}

	if msg, ok := result.Output["msg"].(string); !ok || msg != "Hello, World!" {
		t.Errorf("Expected msg='Hello, World!', got %v", result.Output["msg"])
	}
}

// TestDebugModule_Execute_WithVar tests debug module with var parameter
func TestDebugModule_Execute_WithVar(t *testing.T) {
	module := NewDebugModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name": "test_debug",
		"var":  "test_variable",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Changed {
		t.Errorf("Expected changed=false, got true")
	}

	if msg, ok := result.Output["msg"].(string); !ok {
		t.Errorf("Expected msg to be string, got %T", result.Output["msg"])
	} else if msg == "" {
		t.Errorf("Expected non-empty msg")
	}
}

// TestDebugModule_Execute_WithComplexVar tests debug module with complex variable
func TestDebugModule_Execute_WithComplexVar(t *testing.T) {
	module := NewDebugModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	complexVar := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": []string{"a", "b", "c"},
	}

	args := map[string]interface{}{
		"name": "test_debug",
		"var":  complexVar,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if msg, ok := result.Output["msg"].(string); !ok {
		t.Errorf("Expected msg to be string, got %T", result.Output["msg"])
	} else if msg == "" {
		t.Errorf("Expected non-empty msg")
	}
}

// TestDebugModule_Execute_WithNonStringMsg tests debug module with non-string msg
func TestDebugModule_Execute_WithNonStringMsg(t *testing.T) {
	module := NewDebugModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name": "test_debug",
		"msg":  123,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if msg, ok := result.Output["msg"].(string); !ok {
		t.Errorf("Expected msg to be string, got %T", result.Output["msg"])
	} else if msg != "123" {
		t.Errorf("Expected msg='123', got %s", msg)
	}
}

// TestDebugModule_Execute_MissingParameters tests error when both msg and var are missing
func TestDebugModule_Execute_MissingParameters(t *testing.T) {
	module := NewDebugModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name": "test_debug",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Errorf("Expected failure, got success")
	}

	if result.Error == "" {
		t.Errorf("Expected error message, got empty string")
	}
}

// TestDebugModule_Execute_WithTimeout tests execution with context timeout
func TestDebugModule_Execute_WithTimeout(t *testing.T) {
	module := NewDebugModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name": "test_debug",
		"msg":  "Test message",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}
}

// TestDebugModule_Validate tests argument validation
func TestDebugModule_Validate(t *testing.T) {
	module := NewDebugModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid_with_msg",
			args: map[string]interface{}{
				"name": "test",
				"msg":  "Hello",
			},
			wantErr: false,
		},
		{
			name: "valid_with_var",
			args: map[string]interface{}{
				"name": "test",
				"var":  "variable",
			},
			wantErr: false,
		},
		{
			name: "valid_with_both",
			args: map[string]interface{}{
				"name": "test",
				"msg":  "Hello",
				"var":  "variable",
			},
			wantErr: false,
		},
		{
			name: "missing_msg_and_var",
			args: map[string]interface{}{
				"name": "test",
			},
			wantErr: true,
		},
		{
			name: "missing_name",
			args: map[string]interface{}{
				"msg": "Hello",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDebugModule_GetName tests module name retrieval
func TestDebugModule_GetName(t *testing.T) {
	module := NewDebugModule()

	if module.GetName() != "debug" {
		t.Errorf("Expected name 'debug', got '%s'", module.GetName())
	}
}

// TestDebugModule_GetDescription tests module description retrieval
func TestDebugModule_GetDescription(t *testing.T) {
	module := NewDebugModule()

	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("Expected non-empty description")
	}

	if desc != "Prints debug messages" {
		t.Errorf("Expected description 'Prints debug messages', got '%s'", desc)
	}
}

// TestNewDebugModule tests module creation
func TestNewDebugModule(t *testing.T) {
	module := NewDebugModule()

	if module == nil {
		t.Fatal("Expected non-nil module")
	}

	if module.GetName() != "debug" {
		t.Errorf("Expected name 'debug', got '%s'", module.GetName())
	}
}

// TestDebugModule_Execute_EmptyMsg tests debug module with empty message
func TestDebugModule_Execute_EmptyMsg(t *testing.T) {
	module := NewDebugModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name": "test_debug",
		"msg":  "",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if msg, ok := result.Output["msg"].(string); !ok || msg != "" {
		t.Errorf("Expected msg='', got %v", result.Output["msg"])
	}
}

// BenchmarkDebugModule_Execute benchmarks debug module execution
func BenchmarkDebugModule_Execute(b *testing.B) {
	module := NewDebugModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name": "test_debug",
		"msg":  "Benchmark message",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = module.Execute(ctx, host, args)
	}
}

// BenchmarkDebugModule_Validate benchmarks argument validation
func BenchmarkDebugModule_Validate(b *testing.B) {
	module := NewDebugModule()

	args := map[string]interface{}{
		"name": "test",
		"msg":  "Hello",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.Validate(args)
	}
}
