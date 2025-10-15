package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestDockerContainerModule_Execute(t *testing.T) {
	module := NewDockerContainerModule()
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
			name: "missing container name",
			args: map[string]interface{}{
				"image": "nginx:latest",
			},
			expectError: true,
		},
		{
			name: "valid container creation",
			args: map[string]interface{}{
				"name":  "test-nginx",
				"image": "nginx:latest",
				"state": "present",
			},
			expectError: false,
		},
		{
			name: "container with ports",
			args: map[string]interface{}{
				"name":  "test-web",
				"image": "nginx:latest",
				"ports": []interface{}{"8080:80"},
				"state": "started",
			},
			expectError: false,
		},
		{
			name: "container with environment",
			args: map[string]interface{}{
				"name":  "test-app",
				"image": "alpine:latest",
				"env": map[string]interface{}{
					"APP_ENV": "production",
					"DEBUG":   "false",
				},
				"state": "present",
			},
			expectError: false,
		},
		{
			name: "stop container",
			args: map[string]interface{}{
				"name":  "test-nginx",
				"state": "stopped",
			},
			expectError: false,
		},
		{
			name: "remove container",
			args: map[string]interface{}{
				"name":  "test-nginx",
				"state": "absent",
				"force": true,
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
				assert.Equal(t, "docker_container", result.Module)
			}
		})
	}
}

func TestDockerContainerModule_GetName(t *testing.T) {
	module := NewDockerContainerModule()
	assert.Equal(t, "docker_container", module.GetName())
}

func TestDockerContainerModule_GetDescription(t *testing.T) {
	module := NewDockerContainerModule()
	assert.Equal(t, "Manage Docker containers", module.GetDescription())
}
