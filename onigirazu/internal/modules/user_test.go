package modules

import (
	"context"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestUserModule_Validate tests user module argument validation
func TestUserModule_Validate(t *testing.T) {
	module := NewUserModuleFixed()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid_present",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
			},
			wantErr: false,
		},
		{
			name: "valid_absent",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "absent",
			},
			wantErr: false,
		},
		{
			name: "valid_with_shell",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
				"shell": "/bin/bash",
			},
			wantErr: false,
		},
		{
			name: "valid_with_home",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
				"home":  "/home/testuser",
			},
			wantErr: false,
		},
		{
			name: "valid_with_group",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
				"group": "users",
			},
			wantErr: false,
		},
		{
			name: "valid_with_groups",
			args: map[string]interface{}{
				"name":   "testuser",
				"state":  "present",
				"groups": "wheel,docker",
			},
			wantErr: false,
		},
		{
			name: "valid_with_uid_int",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
				"uid":   1000,
			},
			wantErr: false,
		},
		{
			name: "valid_with_uid_string",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
				"uid":   "1000",
			},
			wantErr: false,
		},
		{
			name: "valid_with_gid_int",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
				"gid":   1000,
			},
			wantErr: false,
		},
		{
			name: "valid_with_gid_string",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
				"gid":   "1000",
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
				"name": "testuser",
			},
			wantErr: true,
		},
		{
			name: "invalid_state",
			args: map[string]interface{}{
				"name":  "testuser",
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
				"name":  "testuser",
				"state": 123,
			},
			wantErr: true,
		},
		{
			name: "invalid_shell_type",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
				"shell": 123,
			},
			wantErr: true,
		},
		{
			name: "invalid_home_type",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
				"home":  123,
			},
			wantErr: true,
		},
		{
			name: "invalid_group_type",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
				"group": 123,
			},
			wantErr: true,
		},
		{
			name: "invalid_groups_type",
			args: map[string]interface{}{
				"name":   "testuser",
				"state":  "present",
				"groups": 123,
			},
			wantErr: true,
		},
		{
			name: "invalid_uid_type",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
				"uid":   []string{"invalid"},
			},
			wantErr: true,
		},
		{
			name: "invalid_gid_type",
			args: map[string]interface{}{
				"name":  "testuser",
				"state": "present",
				"gid":   []string{"invalid"},
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

// TestUserModule_GetName tests module name retrieval
func TestUserModule_GetName(t *testing.T) {
	module := NewUserModuleFixed()

	if module.GetName() != "user" {
		t.Errorf("Expected name 'user', got '%s'", module.GetName())
	}
}

// TestUserModule_GetDescription tests module description retrieval
func TestUserModule_GetDescription(t *testing.T) {
	module := NewUserModuleFixed()

	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("Expected non-empty description")
	}

	if desc != "Manages system users" {
		t.Errorf("Expected description 'Manages system users', got '%s'", desc)
	}
}

// TestNewUserModule tests module creation
func TestNewUserModule(t *testing.T) {
	module := NewUserModule()

	if module == nil {
		t.Fatal("Expected non-nil module")
	}

	if module.GetName() != "user" {
		t.Errorf("Expected name 'user', got '%s'", module.GetName())
	}
}

// TestUserModule_Execute_MissingName tests validation when name is missing
func TestUserModule_Execute_MissingName(t *testing.T) {
	module := NewUserModuleFixed()

	// Test validation directly since Execute requires name for TaskName
	args := map[string]interface{}{
		"state": "present",
	}

	err := module.Validate(args)
	if err == nil {
		t.Errorf("Expected validation error for missing name, got nil")
	}
}

// TestUserModule_Execute_MissingState tests error when state is missing
func TestUserModule_Execute_MissingState(t *testing.T) {
	module := NewUserModuleFixed()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name": "testuser",
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

// TestUserModule_Execute_InvalidState tests error when state is invalid
func TestUserModule_Execute_InvalidState(t *testing.T) {
	module := NewUserModuleFixed()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "testuser",
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

// TestUserModule_Execute_WithTimeout tests execution with context timeout
func TestUserModule_Execute_WithTimeout(t *testing.T) {
	module := NewUserModuleFixed()

	host := types.Host{
		Name:    "test-host",
		Address: "localhost",
		Port:    22,
	}

	args := map[string]interface{}{
		"name":  "testuser",
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

// BenchmarkUserModule_Validate benchmarks argument validation
func BenchmarkUserModule_Validate(b *testing.B) {
	module := NewUserModuleFixed()

	args := map[string]interface{}{
		"name":   "testuser",
		"state":  "present",
		"shell":  "/bin/bash",
		"home":   "/home/testuser",
		"group":  "users",
		"groups": "wheel,docker",
		"uid":    1000,
		"gid":    1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.Validate(args)
	}
}

// BenchmarkUserModule_Validate_Minimal benchmarks validation with minimal args
func BenchmarkUserModule_Validate_Minimal(b *testing.B) {
	module := NewUserModuleFixed()

	args := map[string]interface{}{
		"name":  "testuser",
		"state": "present",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.Validate(args)
	}
}
