package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestDockerImageModule_Execute(t *testing.T) {
	module := NewDockerImageModule()
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
			name: "missing image name",
			args: map[string]interface{}{
				"state": "present",
			},
			expectError: true,
		},
		{
			name: "pull image",
			args: map[string]interface{}{
				"name":  "alpine",
				"tag":   "latest",
				"state": "present",
			},
			expectError: false,
		},
		{
			name: "pull image with platform",
			args: map[string]interface{}{
				"name":     "nginx",
				"tag":      "alpine",
				"state":    "present",
				"platform": "linux/amd64",
			},
			expectError: false,
		},
		{
			name: "force pull image",
			args: map[string]interface{}{
				"name":  "alpine",
				"tag":   "latest",
				"state": "present",
				"force": true,
			},
			expectError: false,
		},
		{
			name: "build image",
			args: map[string]interface{}{
				"name":  "myapp",
				"tag":   "v1.0",
				"state": "build",
				"path":  ".",
			},
			expectError: false,
		},
		{
			name: "build with dockerfile",
			args: map[string]interface{}{
				"name":       "myapp",
				"tag":        "v1.0",
				"state":      "build",
				"path":       ".",
				"dockerfile": "Dockerfile.prod",
			},
			expectError: false,
		},
		{
			name: "build with args",
			args: map[string]interface{}{
				"name":  "myapp",
				"tag":   "v1.0",
				"state": "build",
				"path":  ".",
				"build_args": map[string]interface{}{
					"VERSION": "1.0.0",
					"ENV":     "production",
				},
			},
			expectError: false,
		},
		{
			name: "remove image",
			args: map[string]interface{}{
				"name":  "alpine",
				"tag":   "latest",
				"state": "absent",
			},
			expectError: false,
		},
		{
			name: "force remove image",
			args: map[string]interface{}{
				"name":  "myapp",
				"tag":   "v1.0",
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
				assert.Equal(t, "docker_image", result.Module)
			}
		})
	}
}

func TestDockerImageModule_GetName(t *testing.T) {
	module := NewDockerImageModule()
	assert.Equal(t, "docker_image", module.GetName())
}

func TestDockerImageModule_GetDescription(t *testing.T) {
	module := NewDockerImageModule()
	assert.Equal(t, "Manage Docker images", module.GetDescription())
}
