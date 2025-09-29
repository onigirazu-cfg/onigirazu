package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// BaseModule represents base module structure
type BaseModule struct {
	name        string
	description string
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
