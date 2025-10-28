package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestRebootModule(t *testing.T) {
	module := NewRebootModule()

	// Test basic properties
	if module.GetName() != "reboot" {
		t.Errorf("Expected name 'reboot', got '%s'", module.GetName())
	}

	if module.GetDescription() == "" {
		t.Error("Expected non-empty description")
	}
}

func TestRebootModuleValidate(t *testing.T) {
	module := NewRebootModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid default args",
			args: map[string]interface{}{
				"name": "reboot",
			},
			wantErr: false,
		},
		{
			name: "with delay",
			args: map[string]interface{}{
				"name":             "reboot",
				"pre_reboot_delay": 30,
				"msg":              "System will reboot",
			},
			wantErr: false,
		},
		{
			name: "test boot mode",
			args: map[string]interface{}{
				"name":      "reboot",
				"test_boot": true,
			},
			wantErr: false,
		},
		{
			name: "custom reboot command",
			args: map[string]interface{}{
				"name":           "reboot",
				"reboot_command": "shutdown -r now",
			},
			wantErr: false,
		},
		{
			name: "test boot and custom command conflict",
			args: map[string]interface{}{
				"name":           "reboot",
				"test_boot":      true,
				"reboot_command": "shutdown -r now",
			},
			wantErr: true,
		},
		{
			name: "negative delay",
			args: map[string]interface{}{
				"name":             "reboot",
				"pre_reboot_delay": -10,
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

func TestRebootModuleExecute(t *testing.T) {
	module := NewRebootModule()
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
			name: "test boot mode",
			args: map[string]interface{}{
				"name":      "reboot",
				"test_boot": true,
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
			if result.Module != "reboot" {
				t.Errorf("Expected module 'reboot', got '%s'", result.Module)
			}
		})
	}
}
