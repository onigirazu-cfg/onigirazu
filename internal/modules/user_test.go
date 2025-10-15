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

// TestUserModule_BuildUserAddCommand tests the buildUserAddCommand method
func TestUserModule_BuildUserAddCommand(t *testing.T) {
	module := NewUserModuleFixed()

	tests := []struct {
		name     string
		username string
		args     map[string]interface{}
		want     []string
	}{
		{
			name:     "minimal_command",
			username: "testuser",
			args:     map[string]interface{}{},
			want:     []string{"useradd", "-m", "testuser"},
		},
		{
			name:     "with_home_directory",
			username: "testuser",
			args: map[string]interface{}{
				"home": "/custom/home",
			},
			want: []string{"useradd", "-d", "/custom/home", "-m", "testuser"},
		},
		{
			name:     "with_shell",
			username: "testuser",
			args: map[string]interface{}{
				"shell": "/bin/zsh",
			},
			want: []string{"useradd", "-s", "/bin/zsh", "-m", "testuser"},
		},
		{
			name:     "with_primary_group",
			username: "testuser",
			args: map[string]interface{}{
				"group": "developers",
			},
			want: []string{"useradd", "-g", "developers", "-m", "testuser"},
		},
		{
			name:     "with_supplementary_groups",
			username: "testuser",
			args: map[string]interface{}{
				"groups": "wheel,docker",
			},
			want: []string{"useradd", "-G", "wheel,docker", "-m", "testuser"},
		},
		{
			name:     "with_uid_int",
			username: "testuser",
			args: map[string]interface{}{
				"uid": 1500,
			},
			want: []string{"useradd", "-u", "1500", "-m", "testuser"},
		},
		{
			name:     "with_uid_int64",
			username: "testuser",
			args: map[string]interface{}{
				"uid": int64(2000),
			},
			want: []string{"useradd", "-u", "2000", "-m", "testuser"},
		},
		{
			name:     "with_uid_float64",
			username: "testuser",
			args: map[string]interface{}{
				"uid": float64(2500),
			},
			want: []string{"useradd", "-u", "2500", "-m", "testuser"},
		},
		{
			name:     "with_uid_string",
			username: "testuser",
			args: map[string]interface{}{
				"uid": "3000",
			},
			want: []string{"useradd", "-u", "3000", "-m", "testuser"},
		},
		{
			name:     "with_gid_int",
			username: "testuser",
			args: map[string]interface{}{
				"gid": 1500,
			},
			want: []string{"useradd", "-g", "1500", "-m", "testuser"},
		},
		{
			name:     "with_gid_string",
			username: "testuser",
			args: map[string]interface{}{
				"gid": "2000",
			},
			want: []string{"useradd", "-g", "2000", "-m", "testuser"},
		},
		{
			name:     "with_all_options",
			username: "testuser",
			args: map[string]interface{}{
				"home":   "/home/custom",
				"shell":  "/bin/bash",
				"group":  "users",
				"groups": "wheel,docker",
				"uid":    1000,
			},
			want: []string{"useradd", "-d", "/home/custom", "-s", "/bin/bash", "-g", "users", "-G", "wheel,docker", "-u", "1000", "-m", "testuser"},
		},
		{
			name:     "with_empty_home",
			username: "testuser",
			args: map[string]interface{}{
				"home": "",
			},
			want: []string{"useradd", "-m", "testuser"},
		},
		{
			name:     "with_empty_shell",
			username: "testuser",
			args: map[string]interface{}{
				"shell": "",
			},
			want: []string{"useradd", "-m", "testuser"},
		},
		{
			name:     "with_invalid_home_type",
			username: "testuser",
			args: map[string]interface{}{
				"home": 123,
			},
			want: []string{"useradd", "-m", "testuser"},
		},
		{
			name:     "with_invalid_shell_type",
			username: "testuser",
			args: map[string]interface{}{
				"shell": 123,
			},
			want: []string{"useradd", "-m", "testuser"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := module.buildUserAddCommand(tt.username, tt.args)

			if len(got) != len(tt.want) {
				t.Errorf("buildUserAddCommand() length = %v, want %v\nGot: %v\nWant: %v", len(got), len(tt.want), got, tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("buildUserAddCommand()[%d] = %v, want %v\nGot: %v\nWant: %v", i, got[i], tt.want[i], got, tt.want)
				}
			}
		})
	}
}

// TestUserModule_IsIdempotent tests the IsIdempotent method
func TestUserModule_IsIdempotent(t *testing.T) {
	module := NewUserModuleFixed()

	if !module.IsIdempotent() {
		t.Error("IsIdempotent() should return true for user module")
	}
}
