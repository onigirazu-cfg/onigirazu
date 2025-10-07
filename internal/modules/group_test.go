package modules

import (
	"context"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestGroupModule_Validate tests group module argument validation
func TestGroupModule_Validate(t *testing.T) {
	module := NewGroupModuleFixed()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid_present",
			args: map[string]interface{}{
				"name":  "testgroup",
				"state": "present",
			},
			wantErr: false,
		},
		{
			name: "valid_absent",
			args: map[string]interface{}{
				"name":  "testgroup",
				"state": "absent",
			},
			wantErr: false,
		},
		{
			name: "valid_with_gid_int",
			args: map[string]interface{}{
				"name":  "testgroup",
				"state": "present",
				"gid":   1000,
			},
			wantErr: false,
		},
		{
			name: "valid_with_gid_string",
			args: map[string]interface{}{
				"name":  "testgroup",
				"state": "present",
				"gid":   "1000",
			},
			wantErr: false,
		},
		{
			name: "valid_with_system",
			args: map[string]interface{}{
				"name":   "testgroup",
				"state":  "present",
				"system": true,
			},
			wantErr: false,
		},
		{
			name: "missing_name",
			args: map[string]interface{}{
				"state": "present",
			},
			wantErr: true,
		},
		{
			name: "missing_state",
			args: map[string]interface{}{
				"name": "testgroup",
			},
			wantErr: true,
		},
		{
			name: "invalid_state",
			args: map[string]interface{}{
				"name":  "testgroup",
				"state": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid_name_type",
			args: map[string]interface{}{
				"name":  123,
				"state": "present",
			},
			wantErr: true,
		},
		{
			name: "invalid_state_type",
			args: map[string]interface{}{
				"name":  "testgroup",
				"state": 123,
			},
			wantErr: true,
		},
		{
			name: "invalid_gid_type",
			args: map[string]interface{}{
				"name":  "testgroup",
				"state": "present",
				"gid":   []string{"invalid"},
			},
			wantErr: true,
		},
		{
			name: "invalid_system_type",
			args: map[string]interface{}{
				"name":   "testgroup",
				"state":  "present",
				"system": "invalid",
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

// TestGroupModule_GetName tests module name retrieval
func TestGroupModule_GetName(t *testing.T) {
	module := NewGroupModuleFixed()

	if module.GetName() != "group" {
		t.Errorf("Expected name 'group', got '%s'", module.GetName())
	}
}

// TestGroupModule_GetDescription tests module description retrieval
func TestGroupModule_GetDescription(t *testing.T) {
	module := NewGroupModuleFixed()

	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("Expected non-empty description")
	}

	if desc != "Manages system groups" {
		t.Errorf("Expected description 'Manages system groups', got '%s'", desc)
	}
}

// TestNewGroupModule tests module creation
func TestNewGroupModule(t *testing.T) {
	module := NewGroupModule()

	if module == nil {
		t.Fatal("Expected non-nil module")
	}

	if module.GetName() != "group" {
		t.Errorf("Expected name 'group', got '%s'", module.GetName())
	}
}

// TestGroupModule_Execute_MissingName tests error when name is missing
func TestGroupModule_Execute_MissingName(t *testing.T) {
	module := NewGroupModuleFixed()

	host := types.Host{
		Name: "test-host",
		Host: "localhost",
		Port: 22,
	}

	args := map[string]interface{}{
		"state": "present",
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

// TestGroupModule_Execute_MissingState tests error when state is missing
func TestGroupModule_Execute_MissingState(t *testing.T) {
	module := NewGroupModuleFixed()

	host := types.Host{
		Name: "test-host",
		Host: "localhost",
		Port: 22,
	}

	args := map[string]interface{}{
		"name": "testgroup",
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

// TestGroupModule_Execute_InvalidState tests error when state is invalid
func TestGroupModule_Execute_InvalidState(t *testing.T) {
	module := NewGroupModuleFixed()

	host := types.Host{
		Name: "test-host",
		Host: "localhost",
		Port: 22,
	}

	args := map[string]interface{}{
		"name":  "testgroup",
		"state": "invalid",
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

// TestGroupModule_Execute_WithTimeout tests execution with context timeout
func TestGroupModule_Execute_WithTimeout(t *testing.T) {
	module := NewGroupModuleFixed()

	host := types.Host{
		Name: "test-host",
		Host: "localhost",
		Port: 22,
	}

	args := map[string]interface{}{
		"name":  "testgroup",
		"state": "present",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This will fail because we can't actually connect to localhost:22
	// but we're testing that the context is properly passed
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// We expect failure because we can't connect
	if result.Success {
		t.Logf("Warning: Execute succeeded unexpectedly (might be running on a system with SSH)")
	}
}

// TestGroupModule_Validate_AllFields tests validation with all optional fields
func TestGroupModule_Validate_AllFields(t *testing.T) {
	module := NewGroupModuleFixed()

	args := map[string]interface{}{
		"name":   "testgroup",
		"state":  "present",
		"gid":    1000,
		"system": true,
	}

	err := module.Validate(args)
	if err != nil {
		t.Errorf("Validate() with all fields failed: %v", err)
	}
}

// TestGroupModule_Validate_MinimalFields tests validation with minimal fields
func TestGroupModule_Validate_MinimalFields(t *testing.T) {
	module := NewGroupModuleFixed()

	args := map[string]interface{}{
		"name":  "testgroup",
		"state": "present",
	}

	err := module.Validate(args)
	if err != nil {
		t.Errorf("Validate() with minimal fields failed: %v", err)
	}
}

// BenchmarkGroupModule_Validate benchmarks argument validation
func BenchmarkGroupModule_Validate(b *testing.B) {
	module := NewGroupModuleFixed()

	args := map[string]interface{}{
		"name":   "testgroup",
		"state":  "present",
		"gid":    1000,
		"system": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.Validate(args)
	}
}

// BenchmarkGroupModule_Validate_Minimal benchmarks validation with minimal args
func BenchmarkGroupModule_Validate_Minimal(b *testing.B) {
	module := NewGroupModuleFixed()

	args := map[string]interface{}{
		"name":  "testgroup",
		"state": "present",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.Validate(args)
	}
}

// TestGroupModule_Validate_EdgeCases tests edge cases in validation
func TestGroupModule_Validate_EdgeCases(t *testing.T) {
	module := NewGroupModuleFixed()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "empty_name",
			args: map[string]interface{}{
				"name":  "",
				"state": "present",
			},
			wantErr: false, // Empty string is still a valid string type
		},
		{
			name: "gid_zero",
			args: map[string]interface{}{
				"name":  "testgroup",
				"state": "present",
				"gid":   0,
			},
			wantErr: false, // Zero is a valid GID
		},
		{
			name: "gid_negative",
			args: map[string]interface{}{
				"name":  "testgroup",
				"state": "present",
				"gid":   -1,
			},
			wantErr: false, // Validation doesn't check for negative values
		},
		{
			name: "system_false",
			args: map[string]interface{}{
				"name":   "testgroup",
				"state":  "present",
				"system": false,
			},
			wantErr: false,
		},
		{
			name: "gid_float",
			args: map[string]interface{}{
				"name":  "testgroup",
				"state": "present",
				"gid":   1000.5,
			},
			wantErr: false, // Float is accepted as numeric type
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
