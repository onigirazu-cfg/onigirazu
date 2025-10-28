package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestSysctlModule(t *testing.T) {
	module := NewSysctlModule()

	// Test basic properties
	if module.GetName() != "sysctl" {
		t.Errorf("Expected name 'sysctl', got '%s'", module.GetName())
	}

	if module.GetDescription() == "" {
		t.Error("Expected non-empty description")
	}
}

func TestSysctlModuleValidate(t *testing.T) {
	module := NewSysctlModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "missing name",
			args: map[string]interface{}{
				"value": "1",
			},
			wantErr: true,
		},
		{
			name: "missing value",
			args: map[string]interface{}{
				"name": "net.ipv4.ip_forward",
			},
			wantErr: true,
		},
		{
			name: "valid args",
			args: map[string]interface{}{
				"name":  "net.ipv4.ip_forward",
				"value": "1",
			},
			wantErr: false,
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

func TestSysctlModuleExecute(t *testing.T) {
	module := NewSysctlModule()
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
				"name": "sysctl",
			},
			wantErr: true,
		},
		{
			name: "valid but cannot execute without real host",
			args: map[string]interface{}{
				"name":  "net.ipv4.ip_forward",
				"value": "1",
			},
			wantErr: true, // Will fail because test host is not real
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := module.Execute(ctx, testHost, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if result.Module != "sysctl" {
				t.Errorf("Expected module 'sysctl', got '%s'", result.Module)
			}
		})
	}
}
