package types

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestTaskUnmarshalYAML_NewSyntax(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		expected Task
	}{
		{
			name: "Simple command with new syntax",
			yamlData: `
name: "List files"
module: "command"
command: "ls -la"
shell: true`,
			expected: Task{
				Name:   "List files",
				Module: "command",
				Args: map[string]interface{}{
					"command": "ls -la",
					"shell":   true,
				},
			},
		},
		{
			name: "Old syntax with args",
			yamlData: `
name: "List files old way"
module: "command"
args:
  command: "ls -la"
  shell: true`,
			expected: Task{
				Name:   "List files old way",
				Module: "command",
				Args: map[string]interface{}{
					"command": "ls -la",
					"shell":   true,
				},
			},
		},
		{
			name: "Complex task with mixed types",
			yamlData: `
name: "File operations"
module: "file"
path: "/tmp/test"
state: "directory"
mode: "0755"
recurse: true
owner: "root"
group: "root"
when: "ansible_os_family == 'RedHat'"
tags:
  - "filesystem"
  - "setup"`,
			expected: Task{
				Name:   "File operations",
				Module: "file",
				When:   "ansible_os_family == 'RedHat'",
				Tags:   []string{"filesystem", "setup"},
				Args: map[string]interface{}{
					"path":    "/tmp/test",
					"state":   "directory",
					"mode":    "0755",
					"recurse": true,
					"owner":   "root",
					"group":   "root",
				},
			},
		},
		{
			name: "Task with timeout and retries",
			yamlData: `
name: "Network operation"
module: "uri"
url: "https://api.example.com/status"
method: "GET"
timeout: "30s"
retries: 3
delay: "5s"`,
			expected: Task{
				Name:    "Network operation",
				Module:  "uri",
				Timeout: 30 * time.Second,
				Retries: 3,
				Delay:   5 * time.Second,
				Args: map[string]interface{}{
					"url":    "https://api.example.com/status",
					"method": "GET",
				},
			},
		},
		{
			name: "Task with loop",
			yamlData: `
name: "Create multiple files"
module: "file"
path: "/tmp/file-{{ item }}"
state: "touch"
loop:
  items:
    - "one"
    - "two"
    - "three"
  variable: "item"`,
			expected: Task{
				Name:   "Create multiple files",
				Module: "file",
				Loop: &Loop{
					Items:    []interface{}{"one", "two", "three"},
					Variable: "item",
				},
				Args: map[string]interface{}{
					"path":  "/tmp/file-{{ item }}",
					"state": "touch",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var task Task
			err := yaml.Unmarshal([]byte(tt.yamlData), &task)
			if err != nil {
				t.Fatalf("Failed to unmarshal YAML: %v", err)
			}

			// Check basic fields
			if task.Name != tt.expected.Name {
				t.Errorf("Name: expected %q, got %q", tt.expected.Name, task.Name)
			}
			if task.Module != tt.expected.Module {
				t.Errorf("Module: expected %q, got %q", tt.expected.Module, task.Module)
			}
			if task.When != tt.expected.When {
				t.Errorf("When: expected %q, got %q", tt.expected.When, task.When)
			}
			if task.Timeout != tt.expected.Timeout {
				t.Errorf("Timeout: expected %v, got %v", tt.expected.Timeout, task.Timeout)
			}
			if task.Retries != tt.expected.Retries {
				t.Errorf("Retries: expected %d, got %d", tt.expected.Retries, task.Retries)
			}
			if task.Delay != tt.expected.Delay {
				t.Errorf("Delay: expected %v, got %v", tt.expected.Delay, task.Delay)
			}

			// Check tags
			if len(task.Tags) != len(tt.expected.Tags) {
				t.Errorf("Tags length: expected %d, got %d", len(tt.expected.Tags), len(task.Tags))
			} else {
				for i, tag := range tt.expected.Tags {
					if task.Tags[i] != tag {
						t.Errorf("Tags[%d]: expected %q, got %q", i, tag, task.Tags[i])
					}
				}
			}

			// Check loop
			if tt.expected.Loop != nil {
				if task.Loop == nil {
					t.Error("Expected loop to be set, but it's nil")
				} else {
					if task.Loop.Variable != tt.expected.Loop.Variable {
						t.Errorf("Loop.Variable: expected %q, got %q", tt.expected.Loop.Variable, task.Loop.Variable)
					}
					if len(task.Loop.Items) != len(tt.expected.Loop.Items) {
						t.Errorf("Loop.Items length: expected %d, got %d", len(tt.expected.Loop.Items), len(task.Loop.Items))
					}
				}
			}

			// Check args
			if len(task.Args) != len(tt.expected.Args) {
				t.Errorf("Args length: expected %d, got %d", len(tt.expected.Args), len(task.Args))
			}
			for key, expectedValue := range tt.expected.Args {
				if actualValue, exists := task.Args[key]; !exists {
					t.Errorf("Args[%q]: expected to exist", key)
				} else if actualValue != expectedValue {
					t.Errorf("Args[%q]: expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestTaskUnmarshalYAML_BackwardCompatibility(t *testing.T) {
	// Test that old syntax still works
	yamlData := `
name: "Backward compatibility test"
module: "command"
args:
  command: "echo hello"
  shell: true
when: "true"
tags:
  - "test"`

	var task Task
	err := yaml.Unmarshal([]byte(yamlData), &task)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if task.Name != "Backward compatibility test" {
		t.Errorf("Name: expected %q, got %q", "Backward compatibility test", task.Name)
	}
	if task.Module != "command" {
		t.Errorf("Module: expected %q, got %q", "command", task.Module)
	}
	if task.When != "true" {
		t.Errorf("When: expected %q, got %q", "true", task.When)
	}
	if len(task.Tags) != 1 || task.Tags[0] != "test" {
		t.Errorf("Tags: expected [\"test\"], got %v", task.Tags)
	}

	expectedArgs := map[string]interface{}{
		"command": "echo hello",
		"shell":   true,
	}
	if len(task.Args) != len(expectedArgs) {
		t.Errorf("Args length: expected %d, got %d", len(expectedArgs), len(task.Args))
	}
	for key, expectedValue := range expectedArgs {
		if actualValue, exists := task.Args[key]; !exists {
			t.Errorf("Args[%q]: expected to exist", key)
		} else if actualValue != expectedValue {
			t.Errorf("Args[%q]: expected %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestTaskUnmarshalYAML_ReservedFieldsNotInArgs(t *testing.T) {
	// Test that reserved fields don't end up in Args
	yamlData := `
name: "Reserved fields test"
module: "command"
command: "echo test"
when: "true"
register: "result"
ignore_errors: true
timeout: "30s"`

	var task Task
	err := yaml.Unmarshal([]byte(yamlData), &task)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Check that reserved fields are properly set
	if task.Name != "Reserved fields test" {
		t.Errorf("Name not set correctly")
	}
	if task.Module != "command" {
		t.Errorf("Module not set correctly")
	}
	if task.When != "true" {
		t.Errorf("When not set correctly")
	}
	if task.Register != "result" {
		t.Errorf("Register not set correctly")
	}
	if !task.IgnoreErrors {
		t.Errorf("IgnoreErrors not set correctly")
	}
	if task.Timeout != 30*time.Second {
		t.Errorf("Timeout not set correctly")
	}

	// Check that only non-reserved fields are in Args
	expectedArgs := map[string]interface{}{
		"command": "echo test",
	}
	if len(task.Args) != len(expectedArgs) {
		t.Errorf("Args should only contain non-reserved fields. Expected %d, got %d", len(expectedArgs), len(task.Args))
	}
	if task.Args["command"] != "echo test" {
		t.Errorf("Args should contain command")
	}

	// Ensure reserved fields are NOT in Args
	reservedInArgs := []string{"name", "module", "when", "register", "ignore_errors", "timeout"}
	for _, field := range reservedInArgs {
		if _, exists := task.Args[field]; exists {
			t.Errorf("Reserved field %q should not be in Args", field)
		}
	}
}

func TestTaskUnmarshalYAML_NewSimplifiedSyntax(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		expected Task
	}{
		{
			name: "Package module with new syntax",
			yamlData: `
name: "Install git"
package:
  name: git
  state: present`,
			expected: Task{
				Name:   "Install git",
				Module: "package",
				Args: map[string]interface{}{
					"name":  "git",
					"state": "present",
				},
			},
		},
		{
			name: "User module with new syntax",
			yamlData: `
name: "Create user"
user:
  name: "appuser"
  state: "present"
  groups:
    - "users"
    - "wheel"
  shell: "/bin/bash"
  create_home: true`,
			expected: Task{
				Name:   "Create user",
				Module: "user",
				Args: map[string]interface{}{
					"name":        "appuser",
					"state":       "present",
					"groups":      []interface{}{"users", "wheel"},
					"shell":       "/bin/bash",
					"create_home": true,
				},
			},
		},
		{
			name: "Template module with new syntax and notify",
			yamlData: `
name: "Deploy nginx configuration"
template:
  src: "nginx.conf.j2"
  dest: "/etc/nginx/nginx.conf"
  backup: true
  mode: "0644"
  owner: "root"
  group: "root"
notify:
  - "restart nginx"`,
			expected: Task{
				Name:   "Deploy nginx configuration",
				Module: "template",
				Args: map[string]interface{}{
					"src":    "nginx.conf.j2",
					"dest":   "/etc/nginx/nginx.conf",
					"backup": true,
					"mode":   "0644",
					"owner":  "root",
					"group":  "root",
				},
				Notify: []string{"restart nginx"},
			},
		},
		{
			name: "File module with new syntax and when condition",
			yamlData: `
name: "Create directory"
file:
  path: "/opt/app"
  state: "directory"
  mode: "0755"
  owner: "root"
when: "ansible_os_family == 'Debian'"`,
			expected: Task{
				Name:   "Create directory",
				Module: "file",
				Args: map[string]interface{}{
					"path":  "/opt/app",
					"state": "directory",
					"mode":  "0755",
					"owner": "root",
				},
				When: "ansible_os_family == 'Debian'",
			},
		},
		{
			name: "Command module with new syntax",
			yamlData: `
name: "Run command"
command:
  cmd: "ls -la"
  chdir: "/tmp"
register: "output"`,
			expected: Task{
				Name:   "Run command",
				Module: "command",
				Args: map[string]interface{}{
					"cmd":   "ls -la",
					"chdir": "/tmp",
				},
				Register: "output",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var task Task
			err := yaml.Unmarshal([]byte(tt.yamlData), &task)
			if err != nil {
				t.Fatalf("Failed to unmarshal YAML: %v", err)
			}

			// Check basic fields
			if task.Name != tt.expected.Name {
				t.Errorf("Name: expected %q, got %q", tt.expected.Name, task.Name)
			}
			if task.Module != tt.expected.Module {
				t.Errorf("Module: expected %q, got %q", tt.expected.Module, task.Module)
			}
			if task.When != tt.expected.When {
				t.Errorf("When: expected %q, got %q", tt.expected.When, task.When)
			}
			if task.Register != tt.expected.Register {
				t.Errorf("Register: expected %q, got %q", tt.expected.Register, task.Register)
			}

			// Check notify
			if len(task.Notify) != len(tt.expected.Notify) {
				t.Errorf("Notify length: expected %d, got %d", len(tt.expected.Notify), len(task.Notify))
			} else {
				for i, n := range tt.expected.Notify {
					if task.Notify[i] != n {
						t.Errorf("Notify[%d]: expected %q, got %q", i, n, task.Notify[i])
					}
				}
			}

			// Check args
			if len(task.Args) != len(tt.expected.Args) {
				t.Errorf("Args length: expected %d, got %d", len(tt.expected.Args), len(task.Args))
			}
			for key, expectedValue := range tt.expected.Args {
				actualValue, exists := task.Args[key]
				if !exists {
					t.Errorf("Args[%q]: expected to exist", key)
					continue
				}

				// Handle slice comparison
				if expectedSlice, ok := expectedValue.([]interface{}); ok {
					actualSlice, ok := actualValue.([]interface{})
					if !ok {
						t.Errorf("Args[%q]: expected slice, got %T", key, actualValue)
						continue
					}
					if len(actualSlice) != len(expectedSlice) {
						t.Errorf("Args[%q]: slice length expected %d, got %d", key, len(expectedSlice), len(actualSlice))
						continue
					}
					for i, ev := range expectedSlice {
						if actualSlice[i] != ev {
							t.Errorf("Args[%q][%d]: expected %v, got %v", key, i, ev, actualSlice[i])
						}
					}
				} else if actualValue != expectedValue {
					t.Errorf("Args[%q]: expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestTaskMarshalYAML_NewSyntax(t *testing.T) {
	// Test that tasks are marshaled using the new simplified syntax
	task := Task{
		Name:   "Install package",
		Module: "package",
		Args: map[string]interface{}{
			"name":  "git",
			"state": "present",
		},
	}

	data, err := yaml.Marshal(&task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	yamlStr := string(data)

	// Check that it uses new syntax (package: instead of module: { type: package })
	if !contains(yamlStr, "package:") {
		t.Errorf("Expected 'package:' in marshaled YAML, got:\n%s", yamlStr)
	}
	if contains(yamlStr, "module:") {
		t.Errorf("Should not contain 'module:' in new syntax, got:\n%s", yamlStr)
	}
	if contains(yamlStr, "type:") {
		t.Errorf("Should not contain 'type:' in new syntax, got:\n%s", yamlStr)
	}

	// Unmarshal it back and verify
	var unmarshaled Task
	err = yaml.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Name != task.Name {
		t.Errorf("Name: expected %q, got %q", task.Name, unmarshaled.Name)
	}
	if unmarshaled.Module != task.Module {
		t.Errorf("Module: expected %q, got %q", task.Module, unmarshaled.Module)
	}
	if len(unmarshaled.Args) != len(task.Args) {
		t.Errorf("Args length: expected %d, got %d", len(task.Args), len(unmarshaled.Args))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
