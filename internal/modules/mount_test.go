package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestMountModule(t *testing.T) {
	module := NewMountModule()

	// Test basic properties
	if module.GetName() != "mount" {
		t.Errorf("Expected name 'mount', got '%s'", module.GetName())
	}

	if module.GetDescription() == "" {
		t.Error("Expected non-empty description")
	}
}

func TestMountModuleValidate(t *testing.T) {
	module := NewMountModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "missing path",
			args: map[string]interface{}{
				"src": "/dev/sdb1",
			},
			wantErr: true,
		},
		{
			name: "present state missing src",
			args: map[string]interface{}{
				"path":  "/mnt/data",
				"state": "present",
			},
			wantErr: true,
		},
		{
			name: "valid present state",
			args: map[string]interface{}{
				"path":  "/mnt/data",
				"src":   "/dev/sdb1",
				"state": "present",
			},
			wantErr: false,
		},
		{
			name: "valid absent state",
			args: map[string]interface{}{
				"path":  "/mnt/data",
				"state": "absent",
			},
			wantErr: false,
		},
		{
			name: "valid mounted state",
			args: map[string]interface{}{
				"path":  "/mnt/data",
				"state": "mounted",
			},
			wantErr: false,
		},
		{
			name: "valid unmounted state",
			args: map[string]interface{}{
				"path":  "/mnt/data",
				"state": "unmounted",
			},
			wantErr: false,
		},
		{
			name: "invalid state",
			args: map[string]interface{}{
				"path":  "/mnt/data",
				"state": "invalid",
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

func TestMountModuleExecute(t *testing.T) {
	module := NewMountModule()
	ctx := context.Background()

	testHost := types.Host{
		Name: "test-host",
	}

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "missing required parameter",
			args: map[string]interface{}{
				"name": "mount",
			},
			wantErr: true,
		},
		{
			name: "present state without executor",
			args: map[string]interface{}{
				"name":  "mount",
				"path":  "/mnt/data",
				"src":   "/dev/sdb1",
				"state": "present",
			},
			wantErr: true, // Will fail without real host
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := module.Execute(ctx, testHost, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if result.Module != "mount" {
				t.Errorf("Expected module 'mount', got '%s'", result.Module)
			}
		})
	}
}
