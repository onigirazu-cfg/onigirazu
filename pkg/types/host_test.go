package types

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestHostUnmarshalYAML_BasicFields(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		expected Host
	}{
		{
			name: "All standard fields",
			yamlData: `
name: "web-server"
address: "192.168.1.100"
port: 22
user: "admin"
password: "secret"
key_file: "/path/to/key"`,
			expected: Host{
				Name:     "web-server",
				Address:  "192.168.1.100",
				Port:     22,
				User:     "admin",
				Password: "secret",
				KeyFile:  "/path/to/key",
				Vars:     map[string]interface{}{},
			},
		},
		{
			name: "Minimal host",
			yamlData: `
name: "minimal"
address: "10.0.0.1"`,
			expected: Host{
				Name:    "minimal",
				Address: "10.0.0.1",
				Vars:    map[string]interface{}{},
			},
		},
		{
			name: "Host with custom port",
			yamlData: `
name: "custom-port"
address: "example.com"
port: 2222
user: "deploy"`,
			expected: Host{
				Name:    "custom-port",
				Address: "example.com",
				Port:    2222,
				User:    "deploy",
				Vars:    map[string]interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var host Host
			err := yaml.Unmarshal([]byte(tt.yamlData), &host)
			if err != nil {
				t.Fatalf("Failed to unmarshal YAML: %v", err)
			}

			if host.Name != tt.expected.Name {
				t.Errorf("Name: expected %q, got %q", tt.expected.Name, host.Name)
			}
			if host.Address != tt.expected.Address {
				t.Errorf("Address: expected %q, got %q", tt.expected.Address, host.Address)
			}
			if host.Port != tt.expected.Port {
				t.Errorf("Port: expected %d, got %d", tt.expected.Port, host.Port)
			}
			if host.User != tt.expected.User {
				t.Errorf("User: expected %q, got %q", tt.expected.User, host.User)
			}
			if host.Password != tt.expected.Password {
				t.Errorf("Password: expected %q, got %q", tt.expected.Password, host.Password)
			}
			if host.KeyFile != tt.expected.KeyFile {
				t.Errorf("KeyFile: expected %q, got %q", tt.expected.KeyFile, host.KeyFile)
			}
		})
	}
}

func TestHostUnmarshalYAML_WithVars(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		expected Host
	}{
		{
			name: "Host with nested vars",
			yamlData: `
name: "app-server"
address: "192.168.1.50"
vars:
  app_name: "myapp"
  app_version: "1.0.0"
  debug: true`,
			expected: Host{
				Name:    "app-server",
				Address: "192.168.1.50",
				Vars: map[string]interface{}{
					"app_name":    "myapp",
					"app_version": "1.0.0",
					"debug":       true,
				},
			},
		},
		{
			name: "Host with onigirazu-style variables",
			yamlData: `
name: "ansible-compat"
address: "10.0.0.5"
onigirazu_host: "custom-host"
onigirazu_user: "custom-user"
onigirazu_port: 2222
custom_var: "value"`,
			expected: Host{
				Name:    "ansible-compat",
				Address: "10.0.0.5",
				Vars: map[string]interface{}{
					"onigirazu_host": "custom-host",
					"onigirazu_user": "custom-user",
					"onigirazu_port": 2222,
					"custom_var":     "value",
				},
			},
		},
		{
			name: "Host with mixed vars and custom fields",
			yamlData: `
name: "mixed"
address: "example.com"
port: 22
user: "admin"
vars:
  env: "production"
  region: "us-east-1"
custom_field: "custom_value"
another_field: 123`,
			expected: Host{
				Name:    "mixed",
				Address: "example.com",
				Port:    22,
				User:    "admin",
				Vars: map[string]interface{}{
					"env":           "production",
					"region":        "us-east-1",
					"custom_field":  "custom_value",
					"another_field": 123,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var host Host
			err := yaml.Unmarshal([]byte(tt.yamlData), &host)
			if err != nil {
				t.Fatalf("Failed to unmarshal YAML: %v", err)
			}

			if host.Name != tt.expected.Name {
				t.Errorf("Name: expected %q, got %q", tt.expected.Name, host.Name)
			}
			if host.Address != tt.expected.Address {
				t.Errorf("Address: expected %q, got %q", tt.expected.Address, host.Address)
			}
			if host.Port != tt.expected.Port {
				t.Errorf("Port: expected %d, got %d", tt.expected.Port, host.Port)
			}
			if host.User != tt.expected.User {
				t.Errorf("User: expected %q, got %q", tt.expected.User, host.User)
			}

			// Check vars
			if len(host.Vars) != len(tt.expected.Vars) {
				t.Errorf("Vars length: expected %d, got %d", len(tt.expected.Vars), len(host.Vars))
			}
			for key, expectedValue := range tt.expected.Vars {
				if actualValue, exists := host.Vars[key]; !exists {
					t.Errorf("Vars[%q]: expected to exist", key)
				} else if actualValue != expectedValue {
					t.Errorf("Vars[%q]: expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestHostUnmarshalYAML_ReservedFieldsNotInVars(t *testing.T) {
	yamlData := `
name: "test-host"
address: "192.168.1.1"
port: 22
user: "testuser"
password: "testpass"
key_file: "/test/key"
vars:
  custom: "value"
extra_field: "extra"`

	var host Host
	err := yaml.Unmarshal([]byte(yamlData), &host)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Check that reserved fields are properly set
	if host.Name != "test-host" {
		t.Errorf("Name not set correctly")
	}
	if host.Address != "192.168.1.1" {
		t.Errorf("Address not set correctly")
	}
	if host.Port != 22 {
		t.Errorf("Port not set correctly")
	}
	if host.User != "testuser" {
		t.Errorf("User not set correctly")
	}
	if host.Password != "testpass" {
		t.Errorf("Password not set correctly")
	}
	if host.KeyFile != "/test/key" {
		t.Errorf("KeyFile not set correctly")
	}

	// Check that only non-reserved fields are in Vars
	expectedVars := map[string]interface{}{
		"custom":      "value",
		"extra_field": "extra",
	}
	if len(host.Vars) != len(expectedVars) {
		t.Errorf("Vars should only contain non-reserved fields. Expected %d, got %d", len(expectedVars), len(host.Vars))
	}

	// Ensure reserved fields are NOT in Vars
	reservedInVars := []string{"name", "address", "port", "user", "password", "key_file", "vars"}
	for _, field := range reservedInVars {
		if _, exists := host.Vars[field]; exists {
			t.Errorf("Reserved field %q should not be in Vars", field)
		}
	}
}

func TestHostUnmarshalYAML_InvalidYAML(t *testing.T) {
	yamlData := `
name: "test"
address: [invalid yaml structure`

	var host Host
	err := yaml.Unmarshal([]byte(yamlData), &host)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
