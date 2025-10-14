package security

import (
	"fmt"
	"sync"
	"testing"
	"time"

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

func TestSecurityValidator_ValidateHost(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	tests := []struct {
		name        string
		host        types.Host
		wantErr     bool
		wantWarning bool
	}{
		{
			name: "valid host with IP",
			host: types.Host{
				Name:     "test-host",
				Address:  "192.168.1.100",
				Port:     22,
				User:     "testuser",
				KeyFile:  "",
				Password: "",
			},
			wantErr:     false,
			wantWarning: false,
		},
		{
			name: "valid host with hostname",
			host: types.Host{
				Name:     "test-host",
				Address:  "example.com",
				Port:     22,
				User:     "testuser",
				KeyFile:  "",
				Password: "",
			},
			wantErr:     false,
			wantWarning: false,
		},
		{
			name: "invalid hostname format",
			host: types.Host{
				Name:     "test-host",
				Address:  "invalid..hostname",
				Port:     22,
				User:     "testuser",
				KeyFile:  "",
				Password: "",
			},
			wantErr:     true,
			wantWarning: false,
		},
		{
			name: "password authentication warning",
			host: types.Host{
				Name:     "test-host",
				Address:  "192.168.1.100",
				Port:     22,
				User:     "testuser",
				KeyFile:  "",
				Password: "password123",
			},
			wantErr:     false,
			wantWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateHost(tt.host)
			if tt.wantErr {
				assert.False(t, result.Valid)
				assert.NotEmpty(t, result.Violations)
			} else {
				assert.True(t, result.Valid)
			}
			if tt.wantWarning {
				assert.NotEmpty(t, result.Warnings)
			}
			// Check that timestamp and duration are set
			assert.False(t, result.Timestamp.IsZero())
			assert.Greater(t, result.Duration, time.Duration(0))
		})
	}
}

func TestSecurityValidator_ValidateFile(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	tests := []struct {
		name      string
		path      string
		operation string
		wantErr   bool
	}{
		{
			name:      "safe file in /tmp",
			path:      "/tmp/test.txt",
			operation: "write",
			wantErr:   false,
		},
		{
			name:      "blocked path /etc/passwd",
			path:      "/etc/passwd",
			operation: "read",
			wantErr:   true,
		},
		{
			name:      "directory traversal",
			path:      "/tmp/../etc/shadow",
			operation: "read",
			wantErr:   true,
		},
		{
			name:      "safe yaml file",
			path:      "/tmp/config.yaml",
			operation: "write",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateFile(tt.path, tt.operation)
			if tt.wantErr {
				assert.False(t, result.Valid)
				assert.NotEmpty(t, result.Violations)
			} else {
				assert.True(t, result.Valid)
			}
		})
	}
}

func TestSecurityValidator_RemoveRule(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	// Add a custom rule
	rule := ValidationRule{
		Name:     "test_rule",
		Type:     RuleTypeCustom,
		Message:  "Test rule",
		Severity: SeverityLow,
		Enabled:  true,
	}
	validator.AddRule(rule)

	// Verify rule was added
	rules := validator.GetRules()
	initialCount := len(rules)
	assert.Greater(t, initialCount, 0)

	// Remove the rule
	validator.RemoveRule("test_rule")

	// Verify rule was removed
	rules = validator.GetRules()
	assert.Equal(t, initialCount-1, len(rules))

	// Try to remove non-existent rule (should not panic)
	validator.RemoveRule("non_existent_rule")
	assert.Equal(t, initialCount-1, len(validator.GetRules()))
}

func TestSecurityValidator_GetRules(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	rules := validator.GetRules()
	assert.NotNil(t, rules)
	// Should have default rules
	assert.Greater(t, len(rules), 0)

	// Add a custom rule
	customRule := ValidationRule{
		Name:     "custom_rule",
		Type:     RuleTypeCustom,
		Message:  "Custom test rule",
		Severity: SeverityMedium,
		Enabled:  true,
	}
	validator.AddRule(customRule)

	rules = validator.GetRules()
	found := false
	for _, rule := range rules {
		if rule.Name == "custom_rule" {
			found = true
			assert.Equal(t, RuleTypeCustom, rule.Type)
			assert.Equal(t, SeverityMedium, rule.Severity)
			break
		}
	}
	assert.True(t, found, "Custom rule should be in the rules list")
}

func TestValidationResult_Error(t *testing.T) {
	tests := []struct {
		name     string
		result   ValidationResult
		expected string
	}{
		{
			name: "valid result",
			result: ValidationResult{
				Valid:      true,
				Violations: []SecurityViolation{},
			},
			expected: "",
		},
		{
			name: "invalid with violation",
			result: ValidationResult{
				Valid: false,
				Violations: []SecurityViolation{
					{
						Rule:     "test_rule",
						Message:  "Test violation message",
						Severity: SeverityHigh,
					},
				},
			},
			expected: "Test violation message",
		},
		{
			name: "invalid without violations",
			result: ValidationResult{
				Valid:      false,
				Violations: []SecurityViolation{},
			},
			expected: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Error()
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestSecurityValidator_HostValidation_Helpers(t *testing.T) {
	config := DefaultSecurityConfig()
	// Add specific allowed hosts for testing
	config.AllowedHosts = []string{"192.168.1.100", "example.com", "*.test.com"}
	validator := NewSecurityValidator(config)

	t.Run("allowed host validation", func(t *testing.T) {
		// Test with exact IP match
		host := types.Host{
			Name:    "test",
			Address: "192.168.1.100",
			Port:    22,
			User:    "test",
		}
		result := validator.ValidateHost(host)
		assert.True(t, result.Valid)

		// Test with allowed hostname
		host.Address = "example.com"
		result = validator.ValidateHost(host)
		assert.True(t, result.Valid)

		// Test with wildcard hostname
		host.Address = "server.test.com"
		result = validator.ValidateHost(host)
		assert.True(t, result.Valid)
	})

	t.Run("disallowed host validation", func(t *testing.T) {
		// Test with disallowed IP
		host := types.Host{
			Name:    "test",
			Address: "10.0.0.1",
			Port:    22,
			User:    "test",
		}
		result := validator.ValidateHost(host)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Violations)

		// Test with disallowed hostname
		host.Address = "badhost.com"
		result = validator.ValidateHost(host)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Violations)
	})
}

func TestSecurityValidator_ConcurrentValidation(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	// Test concurrent task validation
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			task := types.Task{
				Name:   fmt.Sprintf("task_%d", index),
				Module: "command",
				Args: map[string]interface{}{
					"command": "echo test",
				},
			}
			result := validator.ValidateTask(task)
			assert.True(t, result.Valid)
		}(i)
	}
	wg.Wait()
}

func TestSecurityValidator_ValidationScore(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	t.Run("perfect score", func(t *testing.T) {
		task := types.Task{
			Name:   "safe task",
			Module: "command",
			Args: map[string]interface{}{
				"command": "echo hello",
			},
		}
		result := validator.ValidateTask(task)
		assert.True(t, result.Valid)
		assert.Equal(t, result.MaxScore, result.Score)
	})

	t.Run("reduced score with violations", func(t *testing.T) {
		task := types.Task{
			Name:   "dangerous task",
			Module: "command",
			Args: map[string]interface{}{
				"command": "rm -rf /",
			},
		}
		result := validator.ValidateTask(task)
		assert.False(t, result.Valid)
		assert.Less(t, result.Score, result.MaxScore)
	})
}

func TestSecurityValidator_ValidateKeyFile(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)

	t.Run("non-existent key file", func(t *testing.T) {
		host := types.Host{
			Name:    "test",
			Address: "192.168.1.100",
			Port:    22,
			User:    "test",
			KeyFile: "/nonexistent/key/file.pem",
		}
		result := validator.ValidateHost(host)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Violations)
		// Should have violation about key file
		found := false
		for _, v := range result.Violations {
			if v.Rule == "insecure_key_file" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should have insecure_key_file violation")
	})
}

func TestSecurityAuditor(t *testing.T) {
	config := DefaultSecurityConfig()
	validator := NewSecurityValidator(config)
	auditor := NewSecurityAuditor(validator)
	assert.NotNil(t, auditor)

	t.Run("audit host validation", func(t *testing.T) {
		host := types.Host{
			Name:    "test-host",
			Address: "192.168.1.100",
			Port:    22,
			User:    "testuser",
		}

		entry := auditor.AuditHost(host, "admin")
		assert.NotEmpty(t, entry.ID)
		assert.Equal(t, "host_access", entry.Type)
		assert.Equal(t, "validate", entry.Action)
		assert.Equal(t, host.Address, entry.Resource)
		assert.Equal(t, "admin", entry.User)
		assert.False(t, entry.Timestamp.IsZero())

		// Get audit log
		logs := auditor.GetAuditLog()
		assert.NotEmpty(t, logs)
		assert.Equal(t, 1, len(logs))
		assert.Equal(t, entry.ID, logs[0].ID)
		assert.Equal(t, "host_access", logs[0].Type)
		assert.Equal(t, host.Address, logs[0].Resource)
	})

	t.Run("audit task validation", func(t *testing.T) {
		task := types.Task{
			Name:   "test-task",
			Module: "command",
			Args: map[string]interface{}{
				"command": "echo test",
			},
		}

		entry := auditor.AuditTask(task, "developer")
		assert.NotEmpty(t, entry.ID)
		assert.Equal(t, "task_execution", entry.Type)
		assert.Equal(t, "validate", entry.Action)
		assert.Equal(t, task.Name, entry.Resource)
		assert.Equal(t, "developer", entry.User)

		// Get audit log
		logs := auditor.GetAuditLog()
		assert.Equal(t, 2, len(logs)) // Previous host audit + this task audit

		// Find the task audit entry
		var taskAudit *AuditEntry
		for i := range logs {
			if logs[i].Type == "task_execution" {
				taskAudit = &logs[i]
				break
			}
		}
		assert.NotNil(t, taskAudit)
		assert.Equal(t, entry.ID, taskAudit.ID)
		assert.Equal(t, task.Name, taskAudit.Resource)
	})

	t.Run("audit with violations", func(t *testing.T) {
		task := types.Task{
			Name:   "dangerous-task",
			Module: "command",
			Args: map[string]interface{}{
				"command": "rm -rf /",
			},
		}

		entry := auditor.AuditTask(task, "hacker")
		assert.NotEmpty(t, entry.ID)

		logs := auditor.GetAuditLog()

		// Find the dangerous task audit entry
		var dangerousAudit *AuditEntry
		for i := range logs {
			if logs[i].ID == entry.ID {
				dangerousAudit = &logs[i]
				break
			}
		}
		assert.NotNil(t, dangerousAudit)
		assert.False(t, dangerousAudit.Result.Valid)
		assert.NotEmpty(t, dangerousAudit.Result.Violations)
	})
}
