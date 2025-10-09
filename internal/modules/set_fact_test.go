package modules

import (
	"context"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestSetFactModule_Execute_SingleFact tests setting a single fact
func TestSetFactModule_Execute_SingleFact(t *testing.T) {
	module := NewSetFactModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":    "test_set_fact",
		"my_fact": "my_value",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	if val, ok := result.Output["my_fact"].(string); !ok || val != "my_value" {
		t.Errorf("Expected my_fact='my_value', got %v", result.Output["my_fact"])
	}

	if facts, ok := result.Output["onigirazu_facts"].(map[string]interface{}); !ok {
		t.Errorf("Expected onigirazu_facts to be map, got %T", result.Output["onigirazu_facts"])
	} else if val, ok := facts["my_fact"].(string); !ok || val != "my_value" {
		t.Errorf("Expected onigirazu_facts[my_fact]='my_value', got %v", facts["my_fact"])
	}
}

// TestSetFactModule_Execute_MultipleFacts tests setting multiple facts
func TestSetFactModule_Execute_MultipleFacts(t *testing.T) {
	module := NewSetFactModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "test_set_fact",
		"fact1": "value1",
		"fact2": 123,
		"fact3": true,
		"fact4": []string{"a", "b", "c"},
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Errorf("Expected changed=true, got false")
	}

	// Check individual facts
	if val, ok := result.Output["fact1"].(string); !ok || val != "value1" {
		t.Errorf("Expected fact1='value1', got %v", result.Output["fact1"])
	}

	if val, ok := result.Output["fact2"].(int); !ok || val != 123 {
		t.Errorf("Expected fact2=123, got %v", result.Output["fact2"])
	}

	if val, ok := result.Output["fact3"].(bool); !ok || !val {
		t.Errorf("Expected fact3=true, got %v", result.Output["fact3"])
	}

	if val, ok := result.Output["fact4"].([]string); !ok || len(val) != 3 {
		t.Errorf("Expected fact4 to be []string with 3 elements, got %v", result.Output["fact4"])
	}

	// Check onigirazu_facts
	if facts, ok := result.Output["onigirazu_facts"].(map[string]interface{}); !ok {
		t.Errorf("Expected onigirazu_facts to be map, got %T", result.Output["onigirazu_facts"])
	} else {
		if len(facts) != 4 {
			t.Errorf("Expected 4 facts in onigirazu_facts, got %d", len(facts))
		}
	}
}

// TestSetFactModule_Execute_ComplexFacts tests setting complex facts
func TestSetFactModule_Execute_ComplexFacts(t *testing.T) {
	module := NewSetFactModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	complexValue := map[string]interface{}{
		"nested1": "value1",
		"nested2": 123,
		"nested3": []string{"a", "b"},
	}

	args := map[string]interface{}{
		"name":         "test_set_fact",
		"complex_fact": complexValue,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if val, ok := result.Output["complex_fact"].(map[string]interface{}); !ok {
		t.Errorf("Expected complex_fact to be map, got %T", result.Output["complex_fact"])
	} else {
		if nested, ok := val["nested1"].(string); !ok || nested != "value1" {
			t.Errorf("Expected nested1='value1', got %v", val["nested1"])
		}
	}
}

// TestSetFactModule_Execute_NoFacts tests error when no facts are provided
func TestSetFactModule_Execute_NoFacts(t *testing.T) {
	module := NewSetFactModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name": "test_set_fact",
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

// TestSetFactModule_Execute_WithTimeout tests execution with context timeout
func TestSetFactModule_Execute_WithTimeout(t *testing.T) {
	module := NewSetFactModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":    "test_set_fact",
		"my_fact": "my_value",
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

// TestSetFactModule_Validate tests argument validation
func TestSetFactModule_Validate(t *testing.T) {
	module := NewSetFactModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid_single_fact",
			args: map[string]interface{}{
				"name":    "test",
				"my_fact": "value",
			},
			wantErr: false,
		},
		{
			name: "valid_multiple_facts",
			args: map[string]interface{}{
				"name":  "test",
				"fact1": "value1",
				"fact2": 123,
				"fact3": true,
			},
			wantErr: false,
		},
		{
			name: "no_facts",
			args: map[string]interface{}{
				"name": "test",
			},
			wantErr: true,
		},
		{
			name: "missing_name",
			args: map[string]interface{}{
				"my_fact": "value",
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

// TestSetFactModule_GetName tests module name retrieval
func TestSetFactModule_GetName(t *testing.T) {
	module := NewSetFactModule()

	if module.GetName() != "set_fact" {
		t.Errorf("Expected name 'set_fact', got '%s'", module.GetName())
	}
}

// TestSetFactModule_GetDescription tests module description retrieval
func TestSetFactModule_GetDescription(t *testing.T) {
	module := NewSetFactModule()

	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("Expected non-empty description")
	}

	if desc != "Sets facts (variables) for the current host" {
		t.Errorf("Expected description 'Sets facts (variables) for the current host', got '%s'", desc)
	}
}

// TestNewSetFactModule tests module creation
func TestNewSetFactModule(t *testing.T) {
	module := NewSetFactModule()

	if module == nil {
		t.Fatal("Expected non-nil module")
	}

	if module.GetName() != "set_fact" {
		t.Errorf("Expected name 'set_fact', got '%s'", module.GetName())
	}
}

// TestSetFactModule_Execute_NilValues tests setting facts with nil values
func TestSetFactModule_Execute_NilValues(t *testing.T) {
	module := NewSetFactModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":     "test_set_fact",
		"nil_fact": nil,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if val, exists := result.Output["nil_fact"]; !exists {
		t.Errorf("Expected nil_fact to exist in output")
	} else if val != nil {
		t.Errorf("Expected nil_fact=nil, got %v", val)
	}
}

// TestSetFactModule_Execute_EmptyStringFact tests setting facts with empty string
func TestSetFactModule_Execute_EmptyStringFact(t *testing.T) {
	module := NewSetFactModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":       "test_set_fact",
		"empty_fact": "",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if val, ok := result.Output["empty_fact"].(string); !ok || val != "" {
		t.Errorf("Expected empty_fact='', got %v", result.Output["empty_fact"])
	}
}

// TestSetFactModule_Execute_ZeroValues tests setting facts with zero values
func TestSetFactModule_Execute_ZeroValues(t *testing.T) {
	module := NewSetFactModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":       "test_set_fact",
		"zero_int":   0,
		"zero_float": 0.0,
		"false_bool": false,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if val, ok := result.Output["zero_int"].(int); !ok || val != 0 {
		t.Errorf("Expected zero_int=0, got %v", result.Output["zero_int"])
	}

	if val, ok := result.Output["zero_float"].(float64); !ok || val != 0.0 {
		t.Errorf("Expected zero_float=0.0, got %v", result.Output["zero_float"])
	}

	if val, ok := result.Output["false_bool"].(bool); !ok || val {
		t.Errorf("Expected false_bool=false, got %v", result.Output["false_bool"])
	}
}

// BenchmarkSetFactModule_Execute benchmarks set_fact module execution
func BenchmarkSetFactModule_Execute(b *testing.B) {
	module := NewSetFactModule()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "test_set_fact",
		"fact1": "value1",
		"fact2": 123,
		"fact3": true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = module.Execute(ctx, host, args)
	}
}

// BenchmarkSetFactModule_Validate benchmarks argument validation
func BenchmarkSetFactModule_Validate(b *testing.B) {
	module := NewSetFactModule()

	args := map[string]interface{}{
		"name":  "test",
		"fact1": "value1",
		"fact2": 123,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.Validate(args)
	}
}
