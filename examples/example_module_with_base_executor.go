package examples

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ExampleModule demonstrates the correct way to create a module
// using BaseExecutorModule for safe executor management
type ExampleModule struct {
	*modules.BaseExecutorModule
}

func NewExampleModule() *ExampleModule {
	return &ExampleModule{
		BaseExecutorModule: modules.NewBaseExecutorModule("example"),
	}
}

func (m *ExampleModule) GetDescription() string {
	return "Example module demonstrating safe executor usage"
}

// Execute demonstrates three patterns for using executors safely
func (m *ExampleModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.GetName(),
		Timestamp: startTime,
	}

	// Validate arguments
	if err := m.Validate(args); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Note: The host parameter contains InsecureIgnoreHostKey setting from inventory
	// It's automatically used by the executor when creating SSH connections
	// You can check it if needed: host.InsecureIgnoreHostKey

	command := args["command"].(string)

	// Pattern 1: WithExecutorResult - for simple cases
	if command == "hostname" {
		return m.executeWithPattern1(host, result, startTime)
	}

	// Pattern 2: WithExecutor - for complex logic
	if command == "check_and_create" {
		return m.executeWithPattern2(host, args, result, startTime)
	}

	// Pattern 3: Manual executor management - for maximum control
	return m.executeWithPattern3(host, command, result, startTime)
}

// Pattern 1: WithExecutorResult - Simple and clean
func (m *ExampleModule) executeWithPattern1(host types.Host, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// ✅ WithExecutorResult creates executor, executes function, and cleans up
	output, err := m.WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error) {
		return exec.Execute("hostname")
	})

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.Success = true
	result.Changed = false
	result.Output = map[string]interface{}{
		"hostname": strings.TrimSpace(output),
	}
	result.Duration = time.Since(startTime)

	return result, nil
}

// Pattern 2: WithExecutor - For complex logic with multiple operations
func (m *ExampleModule) executeWithPattern2(host types.Host, args map[string]interface{}, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	path := args["path"].(string)
	changed := false

	// ✅ WithExecutor provides executor to callback function
	err := m.WithExecutor(host, func(exec *executor.CommandExecutor) error {
		// Check if file exists using helper method
		exists, err := m.fileExists(exec, path)
		if err != nil {
			return err
		}

		if !exists {
			// Create file using helper method
			if err := m.createFile(exec, path); err != nil {
				return err
			}
			changed = true
		}

		return nil
	})

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.Success = true
	result.Changed = changed
	result.Output = map[string]interface{}{
		"path":    path,
		"created": changed,
	}
	result.Duration = time.Since(startTime)

	return result, nil
}

// Pattern 3: Manual executor management - Maximum control
func (m *ExampleModule) executeWithPattern3(host types.Host, command string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// ✅ Create fresh executor for this execution
	exec, err := m.CreateExecutor(host)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	defer exec.Close() // ✅ Always close when done

	// Execute command
	output, err := exec.Execute(command)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.Success = true
	result.Changed = false
	result.Output = map[string]interface{}{
		"output": strings.TrimSpace(output),
	}
	result.Duration = time.Since(startTime)

	return result, nil
}

// Helper method: fileExists checks if a file exists
// ✅ Receives executor as parameter, doesn't use cached executor
func (m *ExampleModule) fileExists(exec *executor.CommandExecutor, path string) (bool, error) {
	output, err := exec.Execute("test", "-f", path, "&&", "echo", "exists", "||", "echo", "not_exists")
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(output) == "exists", nil
}

// Helper method: createFile creates a file
// ✅ Receives executor as parameter, doesn't use cached executor
func (m *ExampleModule) createFile(exec *executor.CommandExecutor, path string) error {
	_, err := exec.Execute("touch", path)
	return err
}

func (m *ExampleModule) Validate(args map[string]interface{}) error {
	if _, exists := args["name"]; !exists {
		return fmt.Errorf("argument 'name' is required")
	}

	if _, exists := args["command"]; !exists {
		return fmt.Errorf("argument 'command' is required")
	}

	return nil
}

// Example usage in a playbook:
//
// tasks:
//   - name: Get hostname
//     example:
//       command: hostname
//
//   - name: Check and create file
//     example:
//       command: check_and_create
//       path: /tmp/test.txt
//
//   - name: Run custom command
//     example:
//       command: "ls -la /tmp"
//
// Example inventory configuration with insecure_ignore_host_key:
//
// hosts:
//   test-server:
//     address: 192.168.1.100
//     port: 22
//     user: deploy
//     insecure_ignore_host_key: true  # Disable SSH host key verification (not recommended for production)
//     vars:
//       environment: dev
//
// groups:
//   dev-servers:
//     hosts:
//       - test-server
//     vars:
//       insecure_ignore_host_key: true  # Apply to all hosts in this group
//
// Note: The insecure_ignore_host_key parameter is set in the inventory file,
// not in the task arguments. It's automatically used by the executor when
// establishing SSH connections. The Host object passed to Execute() already
// contains this setting.
