package modules

import (
	"testing"
)

// TestFirewallModule_GetName tests the GetName method
func TestFirewallModule_GetName(t *testing.T) {
	module := NewFirewallModule()

	name := module.GetName()
	if name != "firewall" {
		t.Errorf("Expected name 'firewall', got '%s'", name)
	}
}

// TestFirewallModule_GetDescription tests the GetDescription method
func TestFirewallModule_GetDescription(t *testing.T) {
	module := NewFirewallModule()

	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("Expected non-empty description")
	}

	expectedDesc := "Manage firewall rules (UFW, firewalld, iptables)"
	if desc != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, desc)
	}
}

// TestNewFirewallModule tests module creation
func TestNewFirewallModule(t *testing.T) {
	module := NewFirewallModule()

	if module == nil {
		t.Fatalf("Expected non-nil module")
	}

	if module.GetName() != "firewall" {
		t.Errorf("Expected module name 'firewall', got '%s'", module.GetName())
	}
}

// TestUFWManager_GetType tests UFW manager type identification
func TestUFWManager_GetType(t *testing.T) {
	manager := &UFWManager{}

	typ := manager.GetType()
	if typ != "ufw" {
		t.Errorf("Expected type 'ufw', got '%s'", typ)
	}
}

// TestFirewalldManager_GetType tests firewalld manager type identification
func TestFirewalldManager_GetType(t *testing.T) {
	manager := &FirewalldManager{}

	typ := manager.GetType()
	if typ != "firewalld" {
		t.Errorf("Expected type 'firewalld', got '%s'", typ)
	}
}

// TestIptablesManager_GetType tests iptables manager type identification
func TestIptablesManager_GetType(t *testing.T) {
	manager := &IptablesManager{}

	typ := manager.GetType()
	if typ != "iptables" {
		t.Errorf("Expected type 'iptables', got '%s'", typ)
	}
}

// TestFirewallModule_Validate tests the Validate method
func TestFirewallModule_Validate(t *testing.T) {
	module := NewFirewallModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid enable operation",
			args: map[string]interface{}{
				"operation": "enable",
			},
			wantErr: false,
		},
		{
			name: "valid disable operation",
			args: map[string]interface{}{
				"operation": "disable",
			},
			wantErr: false,
		},
		{
			name: "valid reload operation",
			args: map[string]interface{}{
				"operation": "reload",
			},
			wantErr: false,
		},
		{
			name: "valid list operation",
			args: map[string]interface{}{
				"operation": "list",
			},
			wantErr: false,
		},
		{
			name: "valid rule operation with port",
			args: map[string]interface{}{
				"operation": "rule",
				"port":      "80",
			},
			wantErr: false,
		},
		{
			name: "valid service operation with service",
			args: map[string]interface{}{
				"operation": "service",
				"service":   "ssh",
			},
			wantErr: false,
		},
		{
			name: "valid source operation with source",
			args: map[string]interface{}{
				"operation": "source",
				"source":    "192.168.1.0/24",
			},
			wantErr: false,
		},
		{
			name: "default operation (rule) with port",
			args: map[string]interface{}{
				"port": "443",
			},
			wantErr: false,
		},
		{
			name: "rule operation missing port",
			args: map[string]interface{}{
				"operation": "rule",
			},
			wantErr: true,
			errMsg:  "port parameter is required for rule operation",
		},
		{
			name: "default operation (rule) missing port",
			args: map[string]interface{}{
				"name": "test-task",
			},
			wantErr: true,
			errMsg:  "port parameter is required for rule operation",
		},
		{
			name: "service operation missing service",
			args: map[string]interface{}{
				"operation": "service",
			},
			wantErr: true,
			errMsg:  "service parameter is required for service operation",
		},
		{
			name: "source operation missing source",
			args: map[string]interface{}{
				"operation": "source",
			},
			wantErr: true,
			errMsg:  "source parameter is required for source operation",
		},
		{
			name: "invalid operation",
			args: map[string]interface{}{
				"operation": "invalid",
			},
			wantErr: true,
			errMsg:  "invalid operation: invalid",
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
