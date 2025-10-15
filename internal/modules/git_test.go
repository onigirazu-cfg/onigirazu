package modules

import (
	"testing"
)

// TestGitModule_GetDescription tests the GetDescription method
func TestGitModule_GetDescription(t *testing.T) {
	module := NewGitModule()

	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("Expected non-empty description")
	}

	expectedDesc := "Manages Git repositories"
	if desc != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, desc)
	}
}

// TestGitModule_GetName tests the GetName method
func TestGitModule_GetName(t *testing.T) {
	module := NewGitModule()

	name := module.GetName()
	if name != "git" {
		t.Errorf("Expected name 'git', got '%s'", name)
	}
}

// TestGitModule_IsIdempotent tests the IsIdempotent method
func TestGitModule_IsIdempotent(t *testing.T) {
	module := NewGitModule()

	if !module.IsIdempotent() {
		t.Errorf("Expected IsIdempotent to return true")
	}
}

// TestNewGitModule tests module creation
func TestNewGitModule(t *testing.T) {
	module := NewGitModule()

	if module == nil {
		t.Fatalf("Expected non-nil module")
	}

	if module.GetName() != "git" {
		t.Errorf("Expected module name 'git', got '%s'", module.GetName())
	}
}

// TestNewGitModuleFixed tests module creation with fixed constructor
func TestNewGitModuleFixed(t *testing.T) {
	module := NewGitModuleFixed()

	if module == nil {
		t.Fatalf("Expected non-nil module")
	}

	if module.GetName() != "git" {
		t.Errorf("Expected module name 'git', got '%s'", module.GetName())
	}
}

// TestGitModuleFixed_Validate tests the Validate method
func TestGitModuleFixed_Validate(t *testing.T) {
	module := NewGitModuleFixed()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal args",
			args: map[string]interface{}{
				"name": "test-task",
				"repo": "https://github.com/user/repo.git",
				"dest": "/opt/repo",
			},
			wantErr: false,
		},
		{
			name: "valid with version",
			args: map[string]interface{}{
				"name":    "test-task",
				"repo":    "https://github.com/user/repo.git",
				"dest":    "/opt/repo",
				"version": "v1.0.0",
			},
			wantErr: false,
		},
		{
			name: "valid with force",
			args: map[string]interface{}{
				"name":  "test-task",
				"repo":  "https://github.com/user/repo.git",
				"dest":  "/opt/repo",
				"force": true,
			},
			wantErr: false,
		},
		{
			name: "valid with update",
			args: map[string]interface{}{
				"name":   "test-task",
				"repo":   "https://github.com/user/repo.git",
				"dest":   "/opt/repo",
				"update": true,
			},
			wantErr: false,
		},
		{
			name: "valid with all optional params",
			args: map[string]interface{}{
				"name":    "test-task",
				"repo":    "https://github.com/user/repo.git",
				"dest":    "/opt/repo",
				"version": "main",
				"force":   false,
				"update":  true,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			args: map[string]interface{}{
				"repo": "https://github.com/user/repo.git",
				"dest": "/opt/repo",
			},
			wantErr: true,
			errMsg:  "argument 'name' is required",
		},
		{
			name: "missing repo",
			args: map[string]interface{}{
				"name": "test-task",
				"dest": "/opt/repo",
			},
			wantErr: true,
			errMsg:  "argument 'repo' is required",
		},
		{
			name: "missing dest",
			args: map[string]interface{}{
				"name": "test-task",
				"repo": "https://github.com/user/repo.git",
			},
			wantErr: true,
			errMsg:  "argument 'dest' is required",
		},
		{
			name: "repo not a string",
			args: map[string]interface{}{
				"name": "test-task",
				"repo": 123,
				"dest": "/opt/repo",
			},
			wantErr: true,
			errMsg:  "argument 'repo' must be a string",
		},
		{
			name: "dest not a string",
			args: map[string]interface{}{
				"name": "test-task",
				"repo": "https://github.com/user/repo.git",
				"dest": 123,
			},
			wantErr: true,
			errMsg:  "argument 'dest' must be a string",
		},
		{
			name: "version not a string",
			args: map[string]interface{}{
				"name":    "test-task",
				"repo":    "https://github.com/user/repo.git",
				"dest":    "/opt/repo",
				"version": 123,
			},
			wantErr: true,
			errMsg:  "argument 'version' must be a string",
		},
		{
			name: "force not a boolean",
			args: map[string]interface{}{
				"name":  "test-task",
				"repo":  "https://github.com/user/repo.git",
				"dest":  "/opt/repo",
				"force": "true",
			},
			wantErr: true,
			errMsg:  "argument 'force' must be a boolean",
		},
		{
			name: "update not a boolean",
			args: map[string]interface{}{
				"name":   "test-task",
				"repo":   "https://github.com/user/repo.git",
				"dest":   "/opt/repo",
				"update": "true",
			},
			wantErr: true,
			errMsg:  "argument 'update' must be a boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}
