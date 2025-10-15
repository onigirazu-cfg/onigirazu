package modules

import (
	"testing"
)

func TestSystemdModule_GetName(t *testing.T) {
	module := NewSystemdModule()
	expected := "systemd"
	if got := module.GetName(); got != expected {
		t.Errorf("GetName() = %v, want %v", got, expected)
	}
}

func TestSystemdModule_GetDescription(t *testing.T) {
	module := NewSystemdModule()
	expected := "Manage systemd services, units, and timers"
	if got := module.GetDescription(); got != expected {
		t.Errorf("GetDescription() = %v, want %v", got, expected)
	}
}

func TestNewSystemdModule(t *testing.T) {
	module := NewSystemdModule()
	if module == nil {
		t.Fatal("NewSystemdModule() returned nil")
	}
	if module.GetName() != "systemd" {
		t.Errorf("NewSystemdModule() name = %v, want systemd", module.GetName())
	}
}

func TestSystemdModule_Validate(t *testing.T) {
	module := NewSystemdModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid service operation",
			args: map[string]interface{}{
				"operation": "service",
				"name":      "nginx",
				"state":     "started",
			},
			wantErr: false,
		},
		{
			name: "service operation missing name",
			args: map[string]interface{}{
				"operation": "service",
				"state":     "started",
			},
			wantErr: true,
			errMsg:  "name parameter is required",
		},
		{
			name: "valid timer operation",
			args: map[string]interface{}{
				"operation": "timer",
				"name":      "backup.timer",
				"state":     "started",
			},
			wantErr: false,
		},
		{
			name: "timer operation missing name",
			args: map[string]interface{}{
				"operation": "timer",
				"state":     "started",
			},
			wantErr: true,
			errMsg:  "name parameter is required",
		},
		{
			name: "valid status operation",
			args: map[string]interface{}{
				"operation": "status",
				"name":      "nginx",
			},
			wantErr: false,
		},
		{
			name: "status operation missing name",
			args: map[string]interface{}{
				"operation": "status",
			},
			wantErr: true,
			errMsg:  "name parameter is required",
		},
		{
			name: "valid unit operation with content",
			args: map[string]interface{}{
				"operation": "unit",
				"name":      "myservice.service",
				"content":   "[Unit]\nDescription=My Service",
				"state":     "present",
			},
			wantErr: false,
		},
		{
			name: "valid unit operation with path",
			args: map[string]interface{}{
				"operation": "unit",
				"name":      "myservice.service",
				"path":      "/etc/systemd/system/myservice.service",
				"state":     "present",
			},
			wantErr: false,
		},
		{
			name: "valid unit operation with absent state",
			args: map[string]interface{}{
				"operation": "unit",
				"name":      "myservice.service",
				"state":     "absent",
			},
			wantErr: false,
		},
		{
			name: "unit operation missing name",
			args: map[string]interface{}{
				"operation": "unit",
				"content":   "[Unit]\nDescription=My Service",
				"state":     "present",
			},
			wantErr: true,
			errMsg:  "name parameter is required",
		},
		{
			name: "unit operation present state missing content and path",
			args: map[string]interface{}{
				"operation": "unit",
				"name":      "myservice.service",
				"state":     "present",
			},
			wantErr: true,
			errMsg:  "either content or path parameter is required",
		},
		{
			name: "valid daemon-reload operation",
			args: map[string]interface{}{
				"operation": "daemon-reload",
			},
			wantErr: false,
		},
		{
			name: "invalid operation",
			args: map[string]interface{}{
				"operation": "invalid",
			},
			wantErr: true,
			errMsg:  "invalid operation: invalid",
		},
		{
			name: "default operation (service) with valid args",
			args: map[string]interface{}{
				"name":  "nginx",
				"state": "started",
			},
			wantErr: false,
		},
		{
			name: "default operation (service) missing name",
			args: map[string]interface{}{
				"state": "started",
			},
			wantErr: true,
			errMsg:  "name parameter is required",
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
