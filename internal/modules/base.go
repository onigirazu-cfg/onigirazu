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

// --- Idempotency Support (Phase 1 Enhancement) ---

// PreCheckResult represents the result of a pre-check state comparison
type PreCheckResult struct {
	// IsStateCorrect indicates whether the current system state matches the desired state
	IsStateCorrect bool

	// Reason describes why the state is or isn't correct
	Reason string

	// CurrentState holds the current system state (for diagnostics)
	CurrentState map[string]interface{}

	// Differences holds a map of fields that don't match desired state (for diagnostics)
	// Key: field name, Value: "current_value vs desired_value"
	Differences map[string]string

	// ShouldExecute indicates whether the module should proceed with execution
	// Usually: !IsStateCorrect, but some modules (like command) always execute
	ShouldExecute bool

	// PreCheckDuration is how long the pre-check took
	Duration time.Duration
}

// PreCheckState performs a pre-execution check to determine if the module should execute
// This is the foundation of idempotency - check state before making changes
//
// For idempotent modules, implement this method to:
// 1. Get current system state
// 2. Compare with desired state from args
// 3. Return whether execution is needed
//
// For non-idempotent modules (command, shell, script), always return ShouldExecute=true
//
// Default implementation: assumes module should always execute (non-idempotent behavior)
// Override in specific module implementations
func (m *BaseModule) PreCheckState(ctx context.Context, host types.Host, args map[string]interface{}) (*PreCheckResult, error) {
	return &PreCheckResult{
		IsStateCorrect: false,
		Reason:         "PreCheckState not implemented for this module",
		CurrentState:   make(map[string]interface{}),
		Differences:    make(map[string]string),
		ShouldExecute:  true, // Default: always execute
		Duration:       0,
	}, nil
}

// ApplyIdempotencyCheck wraps module execution with automatic pre-check
// If pre-check indicates state is already correct, skips execution and returns appropriate result
//
// Usage in Execute methods:
//
//	preCheck, err := m.ApplyIdempotencyCheck(ctx, host, args)
//	if err != nil {
//	    return result, err
//	}
//	if !preCheck.ShouldExecute {
//	    // State is already correct, no changes needed
//	    result.Changed = false
//	    result.Output = preCheck.CurrentState
//	    return result, nil
//	}
//	// ... continue with actual execution
func (m *BaseModule) ApplyIdempotencyCheck(ctx context.Context, host types.Host, args map[string]interface{}) (*PreCheckResult, error) {
	startTime := time.Now()
	preCheck, err := m.PreCheckState(ctx, host, args)
	preCheck.Duration = time.Since(startTime)

	if err != nil {
		// Pre-check failed - should we proceed or abort?
		// Default: proceed (non-idempotent fallback)
		return &PreCheckResult{
			ShouldExecute: true,
			Reason:        fmt.Sprintf("PreCheck error: %v", err),
		}, nil
	}

	return preCheck, nil
}

// CompareStateFields is a utility method to compare two state objects
// Returns: (equal bool, differences map)
// Useful for implementing PreCheckState in specific modules
func CompareStateFields(desired map[string]interface{}, current map[string]interface{}, criticalFields []string) (bool, map[string]string) {
	differences := make(map[string]string)

	// Check all critical fields
	for _, field := range criticalFields {
		desiredVal, desiredOk := desired[field]
		currentVal, currentOk := current[field]

		if !desiredOk || !currentOk {
			if desiredOk != currentOk {
				differences[field] = fmt.Sprintf("%v vs %v", currentVal, desiredVal)
			}
			continue
		}

		// Compare values
		if fmt.Sprintf("%v", desiredVal) != fmt.Sprintf("%v", currentVal) {
			differences[field] = fmt.Sprintf("%v vs %v", currentVal, desiredVal)
		}
	}

	return len(differences) == 0, differences
}

// BuildStateOutput is a utility method to format state for output
// Helps modules return consistent state information in results
func BuildStateOutput(state map[string]interface{}, changed bool, changeDetails string) map[string]interface{} {
	output := make(map[string]interface{})

	// Copy state fields to output
	for k, v := range state {
		output[k] = v
	}

	// Add metadata
	output["changed"] = changed
	if changeDetails != "" {
		output["change_details"] = changeDetails
	}

	return output
}
