package parser

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// testLogger is a simple logger for testing that does nothing
type testLogger struct{}

func (m *testLogger) Debug(format string, args ...interface{}) {}
func (m *testLogger) Info(format string, args ...interface{})  {}
func (m *testLogger) Warn(format string, args ...interface{})  {}

func TestIsInlineInventory(t *testing.T) {
	detector := NewInlineInventoryDetector(&testLogger{})

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Inline inventory cases (should return true)
		{"Single IP", "192.168.1.1", true},
		{"Single hostname", "host.example.com", true},
		{"Single IP with trailing comma", "192.168.1.1,", true},
		{"Multiple IPs", "192.168.1.1,192.168.1.2", true},
		{"Multiple hosts", "host1.example.com,host2.example.com", true},
		{"IP with port", "192.168.1.1:2222", true},
		{"Host with port", "host.example.com:2222", true},
		{"User at host", "ubuntu@192.168.1.1", true},
		{"User at host with port", "ubuntu@192.168.1.1:2222", true},
		{"Mixed specs", "ubuntu@host1:2222,root@192.168.1.1,host3", true},
		{"Ansible-style trailing comma", "host1,host2,host3,", true},

		// File path cases (should return false)
		{"Absolute path", "/etc/inventory", false},
		{"Relative path", "./inventory.yml", false},
		{"Parent directory", "../inventory.yml", false},
		{"Home directory", "~/inventory.yml", false},
		{"YAML file", "inventory.yml", false},
		{"JSON file", "inventory.json", false},
		{"TOML file", "hosts.toml", false},
		{"INI file", "hosts.ini", false},
		{"Text file", "hosts.txt", false},
		{"Path with slash", "configs/hosts", false},
		{"Absolute with extension", "/etc/inventory.yml", false},

		// Edge cases
		{"Empty string", "", false},
		{"Whitespace only", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.IsInlineInventory(tt.input)
			if result != tt.expected {
				t.Errorf("IsInlineInventory(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseInlineInventory(t *testing.T) {
	detector := NewInlineInventoryDetector(&testLogger{})

	tests := []struct {
		name           string
		input          string
		expectedCount  int
		expectedErrors bool
		checkFunc      func(*types.Inventory) bool
	}{
		{
			name:           "Single IP",
			input:          "192.168.1.1",
			expectedCount:  1,
			expectedErrors: false,
			checkFunc: func(inv *types.Inventory) bool {
				return inv.Hosts[0].Address == "192.168.1.1" && inv.Hosts[0].Port == 22
			},
		},
		{
			name:           "Single hostname",
			input:          "host.example.com",
			expectedCount:  1,
			expectedErrors: false,
			checkFunc: func(inv *types.Inventory) bool {
				return inv.Hosts[0].Address == "host.example.com" && inv.Hosts[0].User == "root"
			},
		},
		{
			name:           "IP with custom port",
			input:          "192.168.1.1:2222",
			expectedCount:  1,
			expectedErrors: false,
			checkFunc: func(inv *types.Inventory) bool {
				return inv.Hosts[0].Address == "192.168.1.1" && inv.Hosts[0].Port == 2222
			},
		},
		{
			name:           "User at host",
			input:          "ubuntu@192.168.1.1",
			expectedCount:  1,
			expectedErrors: false,
			checkFunc: func(inv *types.Inventory) bool {
				return inv.Hosts[0].User == "ubuntu" && inv.Hosts[0].Address == "192.168.1.1"
			},
		},
		{
			name:           "User at host with port",
			input:          "ubuntu@192.168.1.1:2222",
			expectedCount:  1,
			expectedErrors: false,
			checkFunc: func(inv *types.Inventory) bool {
				return inv.Hosts[0].User == "ubuntu" &&
					inv.Hosts[0].Address == "192.168.1.1" &&
					inv.Hosts[0].Port == 2222
			},
		},
		{
			name:           "Multiple hosts",
			input:          "host1,host2,host3",
			expectedCount:  3,
			expectedErrors: false,
			checkFunc: func(inv *types.Inventory) bool {
				addresses := []string{inv.Hosts[0].Address, inv.Hosts[1].Address, inv.Hosts[2].Address}
				return contains(addresses, "host1") && contains(addresses, "host2") && contains(addresses, "host3")
			},
		},
		{
			name:           "Mixed specifications",
			input:          "ubuntu@host1:2222,root@192.168.1.1,host3",
			expectedCount:  3,
			expectedErrors: false,
			checkFunc: func(inv *types.Inventory) bool {
				return inv.Hosts[0].User == "ubuntu" &&
					inv.Hosts[0].Port == 2222 &&
					inv.Hosts[1].User == "root" &&
					inv.Hosts[2].User == "root"
			},
		},
		{
			name:           "Ansible-style trailing comma",
			input:          "host1,host2,",
			expectedCount:  2,
			expectedErrors: false,
			checkFunc: func(inv *types.Inventory) bool {
				return inv.Hosts[0].Address == "host1" && inv.Hosts[1].Address == "host2"
			},
		},
		{
			name:           "All group created",
			input:          "host1,host2",
			expectedCount:  2,
			expectedErrors: false,
			checkFunc: func(inv *types.Inventory) bool {
				allGroup, ok := inv.Groups["all"]
				return ok && len(allGroup.Hosts) == 2
			},
		},
		{
			name:           "Empty string",
			input:          "",
			expectedCount:  0,
			expectedErrors: true,
			checkFunc:      nil,
		},
		{
			name:           "Whitespace only",
			input:          "   ",
			expectedCount:  0,
			expectedErrors: true,
			checkFunc:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv, err := detector.ParseInlineInventory(tt.input)

			if tt.expectedErrors && err == nil {
				t.Errorf("Expected error but got nil for input: %q", tt.input)
				return
			}

			if !tt.expectedErrors && err != nil {
				t.Errorf("Unexpected error for input %q: %v", tt.input, err)
				return
			}

			if !tt.expectedErrors {
				if inv == nil {
					t.Errorf("Expected inventory but got nil for input: %q", tt.input)
					return
				}

				if len(inv.Hosts) != tt.expectedCount {
					t.Errorf("Expected %d hosts but got %d for input: %q", tt.expectedCount, len(inv.Hosts), tt.input)
					return
				}

				if tt.checkFunc != nil && !tt.checkFunc(inv) {
					t.Errorf("Check function failed for input: %q", tt.input)
				}
			}
		})
	}
}

func TestIsValidHostSpecification(t *testing.T) {
	detector := NewInlineInventoryDetector(&testLogger{})

	tests := []struct {
		name     string
		hostSpec string
		expected bool
	}{
		{"IP address", "192.168.1.1", true},
		{"Hostname", "example.com", true},
		{"Hostname with subdomain", "host.example.com", true},
		{"IP with port", "192.168.1.1:22", true},
		{"Hostname with port", "example.com:2222", true},
		{"User@IP", "ubuntu@192.168.1.1", true},
		{"User@Hostname", "ubuntu@example.com", true},
		{"User@IP:Port", "ubuntu@192.168.1.1:2222", true},
		{"User@Hostname:Port", "ubuntu@example.com:2222", true},
		{"Single character hostname", "h", true},
		{"Hostname with hyphen", "my-host.example.com", true},
		{"Hostname with numbers", "host123.example.com", true},

		{"Invalid IP", "999.999.999.999", false},
		{"Empty user", "@192.168.1.1", false},
		{"Invalid port", "192.168.1.1:99999", true}, // Port format is valid, range check happens in parsePort
		{"URL format", "http://example.com", false},
		{"Empty string", "", false},
		{"Invalid hostname format", "host..example.com", false},
		{"Hostname starting with hyphen", "-host.example.com", false},
		{"Hostname ending with hyphen", "host-.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.isValidHostSpecification(tt.hostSpec)
			if result != tt.expected {
				t.Errorf("isValidHostSpecification(%q) = %v, expected %v", tt.hostSpec, result, tt.expected)
			}
		})
	}
}

func TestIsFilePath(t *testing.T) {
	detector := NewInlineInventoryDetector(&testLogger{})

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Absolute path", "/etc/inventory", true},
		{"Relative path", "./inventory", true},
		{"Parent directory", "../hosts", true},
		{"Home directory", "~/inventory", true},
		{"YAML extension", "inventory.yml", true},
		{"YAML extension uppercase", "inventory.YML", true},
		{"JSON extension", "hosts.json", true},
		{"TOML extension", "config.toml", true},
		{"INI extension", "hosts.ini", true},
		{"TXT extension", "hosts.txt", true},
		{"Path with slash", "configs/inventory.yml", true},
		{"Backslash path", "configs\\inventory", true},

		{"Hostname only", "example.com", false},
		{"IP only", "192.168.1.1", false},
		{"Hostname with port", "example.com:2222", false},
		{"IP with port", "192.168.1.1:2222", false},
		{"User@host", "ubuntu@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.isFilePath(tt.input)
			if result != tt.expected {
				t.Errorf("isFilePath(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParsePort(t *testing.T) {
	detector := NewInlineInventoryDetector(&testLogger{})

	tests := []struct {
		name        string
		portStr     string
		expected    int
		expectError bool
	}{
		{"Standard SSH port", "22", 22, false},
		{"Common alternate port", "2222", 2222, false},
		{"Port 1", "1", 1, false},
		{"Port 65535", "65535", 65535, false},
		{"Whitespace around port", "  2222  ", 2222, false},

		{"Non-numeric", "abc", 0, true},
		{"Port 0", "0", 0, true},
		{"Port too high", "65536", 0, true},
		{"Negative port", "-22", 0, true},
		{"Empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := detector.parsePort(tt.portStr)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil for port: %q", tt.portStr)
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for port %q: %v", tt.portStr, err)
				return
			}

			if !tt.expectError && result != tt.expected {
				t.Errorf("parsePort(%q) = %d, expected %d", tt.portStr, result, tt.expected)
			}
		})
	}
}

// Helper function to check if a string is in a slice
func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
