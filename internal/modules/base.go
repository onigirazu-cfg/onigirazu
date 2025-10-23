package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// BaseModule represents base module structure
type BaseModule struct {
	name        string
	description string
}

// BaseExecutorModule extends BaseModule with safe executor management
// This prevents the common bug of caching executors which causes all hosts
// to execute commands on the first host's connection.
type BaseExecutorModule struct {
	*BaseModule
}

func NewBaseModule(name string) *BaseModule {
	return &BaseModule{name: name}
}

func (m *BaseModule) GetName() string {
	return m.name
}

func (m *BaseModule) GetDescription() string {
	return m.description
}

// Execute performs basic module logic
func (m *BaseModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
	}

	// Basic argument validation
	if err := m.Validate(args); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Here should be module-specific logic
	// For now, return successful result
	result.Success = true
	result.Changed = false
	result.Output = map[string]interface{}{
		"message": fmt.Sprintf("Module %s executed on host %s", m.name, host.Name),
	}
	result.Duration = time.Since(startTime)

	return result, nil
}

// Validate validates argument correctness
func (m *BaseModule) Validate(args map[string]interface{}) error {
	if _, exists := args["name"]; !exists {
		return fmt.Errorf("argument 'name' is required")
	}
	return nil
}

// Helper functions for argument parsing
func getStringArg(args map[string]interface{}, key string, defaultValue string) string {
	if val, exists := args[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getBoolArg(args map[string]interface{}, key string, defaultValue bool) bool {
	if val, exists := args[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
		// Handle string values that might come from YAML
		if s, ok := val.(string); ok {
			return s == "true" || s == "True" || s == "TRUE" || s == "yes" || s == "Yes" || s == "YES" || s == "1"
		}
	}
	return defaultValue
}

func getIntArg(args map[string]interface{}, key string, defaultValue int) int {
	if val, exists := args[key]; exists {
		if i, ok := val.(int); ok {
			return i
		}
		if f, ok := val.(float64); ok {
			return int(f)
		}
	}
	return defaultValue
}

func getMapArg(args map[string]interface{}, key string, defaultValue map[string]interface{}) map[string]interface{} {
	if val, exists := args[key]; exists {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	return defaultValue
}

// failResult creates a failed task result with error message
func (m *BaseModule) failResult(result types.TaskResult, errorMsg string) (types.TaskResult, error) {
	result.Success = false
	result.Changed = false
	result.Error = errorMsg
	result.Duration = time.Since(result.Timestamp)
	return result, fmt.Errorf("%s", errorMsg)
}

// NewBaseExecutorModule creates a new base executor module
func NewBaseExecutorModule(name string) *BaseExecutorModule {
	return &BaseExecutorModule{
		BaseModule: NewBaseModule(name),
	}
}

// WithExecutor executes a function with a fresh executor instance
// This ensures that each execution gets its own executor connected to the correct host.
// The executor is automatically closed after the function completes.
//
// Example usage:
//
//	err := m.WithExecutor(host, func(exec *executor.CommandExecutor) error {
//	    output, err := exec.Execute("git", "status")
//	    if err != nil {
//	        return err
//	    }
//	    // ... process output
//	    return nil
//	})
func (m *BaseExecutorModule) WithExecutor(host types.Host, fn func(*executor.CommandExecutor) error) error {
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}
	defer exec.Close()

	return fn(exec)
}

// WithExecutorResult executes a function with a fresh executor and returns both result and error
// This is useful when you need to return a value from the executor function.
//
// Example usage:
//
//	output, err := m.WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error) {
//	    return exec.Execute("git", "status")
//	})
func (m *BaseExecutorModule) WithExecutorResult(host types.Host, fn func(*executor.CommandExecutor) (string, error)) (string, error) {
	var result string
	err := m.WithExecutor(host, func(exec *executor.CommandExecutor) error {
		var execErr error
		result, execErr = fn(exec)
		return execErr
	})
	return result, err
}

// CreateExecutor creates a new executor for the given host
// The caller is responsible for calling Close() on the returned executor.
// Consider using WithExecutor() instead for automatic cleanup.
func (m *BaseExecutorModule) CreateExecutor(host types.Host) (*executor.CommandExecutor, error) {
	return executor.NewCommandExecutor(host)
}
