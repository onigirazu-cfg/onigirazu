package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestDockerComposeModule_Execute(t *testing.T) {
	module := NewDockerComposeModule()
	ctx := context.Background()
	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{
			name: "missing project_dir",
			args: map[string]interface{}{
				"state": "present",
			},
			expectError: true,
		},
		{
			name: "compose up",
			args: map[string]interface{}{
				"project_dir": "/opt/myapp",
				"state":       "present",
			},
			expectError: false,
		},
		{
			name: "compose up with custom file",
			args: map[string]interface{}{
				"project_dir": "/opt/myapp",
				"file":        "docker-compose.prod.yml",
				"state":       "present",
			},
			expectError: false,
		},
		{
			name: "compose up with project name",
			args: map[string]interface{}{
				"project_dir":  "/opt/myapp",
				"project_name": "myapp-prod",
				"state":        "present",
			},
			expectError: false,
		},
		{
			name: "compose up with build",
			args: map[string]interface{}{
				"project_dir": "/opt/myapp",
				"state":       "present",
				"build":       true,
			},
			expectError: false,
		},
		{
			name: "compose up specific services",
			args: map[string]interface{}{
				"project_dir": "/opt/myapp",
				"state":       "present",
				"services":    []interface{}{"web", "db"},
			},
			expectError: false,
		},
		{
			name: "compose down",
			args: map[string]interface{}{
				"project_dir": "/opt/myapp",
				"state":       "absent",
			},
			expectError: false,
		},
		{
			name: "compose down with volumes",
			args: map[string]interface{}{
				"project_dir":    "/opt/myapp",
				"state":          "absent",
				"remove_volumes": true,
			},
			expectError: false,
		},
		{
			name: "compose restart",
			args: map[string]interface{}{
				"project_dir": "/opt/myapp",
				"state":       "restarted",
			},
			expectError: false,
		},
		{
			name: "compose pull",
			args: map[string]interface{}{
				"project_dir": "/opt/myapp",
				"state":       "pull",
			},
			expectError: false,
		},
		{
			name: "compose build",
			args: map[string]interface{}{
				"project_dir": "/opt/myapp",
				"state":       "build",
				"nocache":     true,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := module.Execute(ctx, host, tt.args)

			if tt.expectError {
				assert.Error(t, err)
				assert.False(t, result.Success)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, "docker_compose", result.Module)
			}
		})
	}
}

func TestDockerComposeModule_GetName(t *testing.T) {
	module := NewDockerComposeModule()
	assert.Equal(t, "docker_compose", module.GetName())
}

func TestDockerComposeModule_GetDescription(t *testing.T) {
	module := NewDockerComposeModule()
	assert.Equal(t, "Manage Docker Compose applications", module.GetDescription())
}
