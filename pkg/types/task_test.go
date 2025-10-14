package types

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestTaskUnmarshalYAML_EdgeCases tests edge cases and less common paths in Task.UnmarshalYAML
func TestTaskUnmarshalYAML_EdgeCases(t *testing.T) {
	t.Run("Task with become fields", func(t *testing.T) {
		yamlData := `
name: "Install package with privilege escalation"
module: "package"
name_arg: "nginx"
state: "present"
become: true
become_user: "root"
become_method: "sudo"
`
		var task Task
		err := yaml.Unmarshal([]byte(yamlData), &task)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if task.Name != "Install package with privilege escalation" {
			t.Errorf("Name: expected %q, got %q", "Install package with privilege escalation", task.Name)
		}
		if !task.Become {
			t.Errorf("Become: expected true, got false")
		}
		if task.BecomeUser != "root" {
			t.Errorf("BecomeUser: expected %q, got %q", "root", task.BecomeUser)
		}
		if task.BecomeMethod != "sudo" {
			t.Errorf("BecomeMethod: expected %q, got %q", "sudo", task.BecomeMethod)
		}
	})

	t.Run("Task with notify handlers", func(t *testing.T) {
		yamlData := `
name: "Update config"
module: "template"
src: "/tmp/config.j2"
dest: "/etc/app/config.yml"
notify:
  - "restart app"
  - "reload nginx"
`
		var task Task
		err := yaml.Unmarshal([]byte(yamlData), &task)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(task.Notify) != 2 {
			t.Fatalf("Notify: expected 2 handlers, got %d", len(task.Notify))
		}
		if task.Notify[0] != "restart app" {
			t.Errorf("Notify[0]: expected %q, got %q", "restart app", task.Notify[0])
		}
		if task.Notify[1] != "reload nginx" {
			t.Errorf("Notify[1]: expected %q, got %q", "reload nginx", task.Notify[1])
		}
	})

	t.Run("Task with retry_delay", func(t *testing.T) {
		yamlData := `
name: "Retry with delay"
module: "uri"
url: "https://api.example.com"
retries: 5
retry_delay: "10s"
`
		var task Task
		err := yaml.Unmarshal([]byte(yamlData), &task)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if task.Retries != 5 {
			t.Errorf("Retries: expected 5, got %d", task.Retries)
		}
		if task.RetryDelay != 10*time.Second {
			t.Errorf("RetryDelay: expected 10s, got %v", task.RetryDelay)
		}
	})

	t.Run("Task with until, changed_when, failed_when", func(t *testing.T) {
		yamlData := `
name: "Wait for service"
module: "command"
cmd: "systemctl is-active myservice"
until: "result.rc == 0"
changed_when: "false"
failed_when: "result.rc > 1"
retries: 10
delay: "5s"
`
		var task Task
		err := yaml.Unmarshal([]byte(yamlData), &task)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if task.Until != "result.rc == 0" {
			t.Errorf("Until: expected %q, got %q", "result.rc == 0", task.Until)
		}
		if task.ChangedWhen != "false" {
			t.Errorf("ChangedWhen: expected %q, got %q", "false", task.ChangedWhen)
		}
		if task.FailedWhen != "result.rc > 1" {
			t.Errorf("FailedWhen: expected %q, got %q", "result.rc > 1", task.FailedWhen)
		}
	})

	t.Run("Task with include and serial", func(t *testing.T) {
		yamlData := `
name: "Include tasks"
include: "common_tasks.yml"
serial: true
`
		var task Task
		err := yaml.Unmarshal([]byte(yamlData), &task)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if task.Include != "common_tasks.yml" {
			t.Errorf("Include: expected %q, got %q", "common_tasks.yml", task.Include)
		}
		if !task.Serial {
			t.Errorf("Serial: expected true, got false")
		}
	})

	t.Run("Task with nested module syntax (old style)", func(t *testing.T) {
		yamlData := `
name: "Old nested syntax"
module:
  type: "package"
  name: "git"
  state: "present"
`
		var task Task
		err := yaml.Unmarshal([]byte(yamlData), &task)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if task.Module != "package" {
			t.Errorf("Module: expected %q, got %q", "package", task.Module)
		}
		if task.Args["name"] != "git" {
			t.Errorf("Args[name]: expected %q, got %v", "git", task.Args["name"])
		}
		if task.Args["state"] != "present" {
			t.Errorf("Args[state]: expected %q, got %v", "present", task.Args["state"])
		}
	})

	t.Run("Task with new simplified syntax (module as field)", func(t *testing.T) {
		yamlData := `
name: "New simplified syntax"
package:
  name: "nginx"
  state: "latest"
when: "onigirazu_os_family == 'Debian'"
`
		var task Task
		err := yaml.Unmarshal([]byte(yamlData), &task)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if task.Module != "package" {
			t.Errorf("Module: expected %q, got %q", "package", task.Module)
		}
		if task.Args["name"] != "nginx" {
			t.Errorf("Args[name]: expected %q, got %v", "nginx", task.Args["name"])
		}
		if task.Args["state"] != "latest" {
			t.Errorf("Args[state]: expected %q, got %v", "latest", task.Args["state"])
		}
	})

	t.Run("Task with args as map[interface{}]interface{}", func(t *testing.T) {
		// This tests the conversion path for map[interface{}]interface{}
		yamlData := `
name: "Test interface map"
module: "command"
args:
  cmd: "echo test"
  creates: "/tmp/marker"
`
		var task Task
		err := yaml.Unmarshal([]byte(yamlData), &task)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if task.Args["cmd"] != "echo test" {
			t.Errorf("Args[cmd]: expected %q, got %v", "echo test", task.Args["cmd"])
		}
	})
}

// TestTaskMarshalYAML_EdgeCases tests edge cases in Task.MarshalYAML
func TestTaskMarshalYAML_EdgeCases(t *testing.T) {
	t.Run("Task with all optional fields", func(t *testing.T) {
		task := Task{
			Name:         "Complex task",
			Module:       "command",
			Args:         map[string]interface{}{"cmd": "echo test"},
			When:         "onigirazu_os == 'Linux'",
			Register:     "result",
			IgnoreErrors: true,
			Tags:         []string{"test", "debug"},
			Notify:       []string{"handler1", "handler2"},
			Timeout:      30 * time.Second,
			Retries:      3,
			Delay:        5 * time.Second,
			Until:        "result.rc == 0",
			ChangedWhen:  "false",
			FailedWhen:   "result.rc > 1",
			Include:      "tasks.yml",
			Serial:       true,
			RetryDelay:   10 * time.Second,
		}

		data, err := yaml.Marshal(&task)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		yamlStr := string(data)

		// Verify all fields are present
		if !contains(yamlStr, "name: Complex task") {
			t.Errorf("Expected 'name: Complex task' in output")
		}
		if !contains(yamlStr, "command:") {
			t.Errorf("Expected 'command:' in output")
		}
		if !contains(yamlStr, "when:") {
			t.Errorf("Expected 'when:' in output")
		}
		if !contains(yamlStr, "register:") {
			t.Errorf("Expected 'register:' in output")
		}
		if !contains(yamlStr, "ignore_errors:") {
			t.Errorf("Expected 'ignore_errors:' in output")
		}
		if !contains(yamlStr, "tags:") {
			t.Errorf("Expected 'tags:' in output")
		}
		if !contains(yamlStr, "notify:") {
			t.Errorf("Expected 'notify:' in output")
		}
		if !contains(yamlStr, "timeout:") {
			t.Errorf("Expected 'timeout:' in output")
		}
		if !contains(yamlStr, "retries:") {
			t.Errorf("Expected 'retries:' in output")
		}
		if !contains(yamlStr, "delay:") {
			t.Errorf("Expected 'delay:' in output")
		}
		if !contains(yamlStr, "until:") {
			t.Errorf("Expected 'until:' in output")
		}
		if !contains(yamlStr, "changed_when:") {
			t.Errorf("Expected 'changed_when:' in output")
		}
		if !contains(yamlStr, "failed_when:") {
			t.Errorf("Expected 'failed_when:' in output")
		}
		if !contains(yamlStr, "include:") {
			t.Errorf("Expected 'include:' in output")
		}
		if !contains(yamlStr, "serial:") {
			t.Errorf("Expected 'serial:' in output")
		}
		if !contains(yamlStr, "retry_delay:") {
			t.Errorf("Expected 'retry_delay:' in output")
		}

		// Unmarshal and verify round-trip
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
		if unmarshaled.When != task.When {
			t.Errorf("When: expected %q, got %q", task.When, unmarshaled.When)
		}
		if unmarshaled.Register != task.Register {
			t.Errorf("Register: expected %q, got %q", task.Register, unmarshaled.Register)
		}
		if unmarshaled.IgnoreErrors != task.IgnoreErrors {
			t.Errorf("IgnoreErrors: expected %v, got %v", task.IgnoreErrors, unmarshaled.IgnoreErrors)
		}
		if unmarshaled.Until != task.Until {
			t.Errorf("Until: expected %q, got %q", task.Until, unmarshaled.Until)
		}
		if unmarshaled.ChangedWhen != task.ChangedWhen {
			t.Errorf("ChangedWhen: expected %q, got %q", task.ChangedWhen, unmarshaled.ChangedWhen)
		}
		if unmarshaled.FailedWhen != task.FailedWhen {
			t.Errorf("FailedWhen: expected %q, got %q", task.FailedWhen, unmarshaled.FailedWhen)
		}
		if unmarshaled.Include != task.Include {
			t.Errorf("Include: expected %q, got %q", task.Include, unmarshaled.Include)
		}
		if unmarshaled.Serial != task.Serial {
			t.Errorf("Serial: expected %v, got %v", task.Serial, unmarshaled.Serial)
		}
	})

	t.Run("Task with loop", func(t *testing.T) {
		task := Task{
			Name:   "Task with loop",
			Module: "file",
			Args: map[string]interface{}{
				"path":  "/tmp/{{ item }}",
				"state": "touch",
			},
			Loop: &Loop{
				Items:    []interface{}{"file1", "file2", "file3"},
				Variable: "item",
			},
		}

		data, err := yaml.Marshal(&task)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		yamlStr := string(data)
		if !contains(yamlStr, "loop:") {
			t.Errorf("Expected 'loop:' in output")
		}

		// Unmarshal and verify
		var unmarshaled Task
		err = yaml.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if unmarshaled.Loop == nil {
			t.Fatal("Loop should not be nil")
		}
		if len(unmarshaled.Loop.Items) != 3 {
			t.Errorf("Loop.Items: expected 3 items, got %d", len(unmarshaled.Loop.Items))
		}
		// Note: Loop.Variable is stored in YAML as 'var', not 'variable'
		// The field name in struct is Variable but YAML tag is 'var'
	})

	t.Run("Minimal task (only name and module)", func(t *testing.T) {
		task := Task{
			Name:   "Minimal",
			Module: "ping",
			Args:   map[string]interface{}{"data": "pong"},
		}

		data, err := yaml.Marshal(&task)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		yamlStr := string(data)
		if !contains(yamlStr, "name: Minimal") {
			t.Errorf("Expected 'name: Minimal' in output")
		}
		if !contains(yamlStr, "ping:") {
			t.Errorf("Expected 'ping:' in output")
		}

		// Should not contain optional fields
		if contains(yamlStr, "when:") {
			t.Errorf("Should not contain 'when:' in minimal task")
		}
		if contains(yamlStr, "register:") {
			t.Errorf("Should not contain 'register:' in minimal task")
		}
	})

	t.Run("Task with empty module name", func(t *testing.T) {
		task := Task{
			Name:   "No module",
			Module: "",
			Args:   map[string]interface{}{},
		}

		data, err := yaml.Marshal(&task)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		yamlStr := string(data)
		// Should only have name field
		if !contains(yamlStr, "name: No module") {
			t.Errorf("Expected 'name: No module' in output")
		}
	})

	t.Run("Task with empty args", func(t *testing.T) {
		task := Task{
			Name:   "Empty args",
			Module: "ping",
			Args:   map[string]interface{}{},
		}

		data, err := yaml.Marshal(&task)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		yamlStr := string(data)
		// Module with empty args should not be included
		if !contains(yamlStr, "name: Empty args") {
			t.Errorf("Expected 'name: Empty args' in output")
		}
	})
}
