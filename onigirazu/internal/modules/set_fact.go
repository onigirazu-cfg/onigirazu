package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// SetFactModule sets facts (variables) for the current host
type SetFactModule struct {
	*BaseModule
}

// NewSetFactModule creates a new set_fact module
func NewSetFactModule() *SetFactModule {
	return &SetFactModule{
		BaseModule: NewBaseModule("set_fact"),
	}
}

func (m *SetFactModule) GetDescription() string {
	return "Sets facts (variables) for the current host"
}

func (m *SetFactModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  getTaskName(args),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   true,
		Changed:   true, // Setting facts is always considered a change
		Output:    make(map[string]interface{}),
	}

	// Remove reserved fields from args to get only the facts to set
	reservedFields := map[string]bool{
		"name": true,
	}

	facts := make(map[string]interface{})
	for key, value := range args {
		if !reservedFields[key] {
			facts[key] = value
		}
	}

	if len(facts) == 0 {
		result.Success = false
		result.Error = "set_fact module requires at least one fact to set"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Store all facts in the output
	result.Output["onigirazu_facts"] = facts

	// Also store individual facts at the root level for easier access
	for key, value := range facts {
		result.Output[key] = value
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *SetFactModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	// Check that at least one fact is provided (excluding reserved fields)
	reservedFields := map[string]bool{
		"name": true,
	}

	hasFactToSet := false
	for key := range args {
		if !reservedFields[key] {
			hasFactToSet = true
			break
		}
	}

	if !hasFactToSet {
		return fmt.Errorf("set_fact module requires at least one fact to set")
	}

	return nil
}

// Helper function to safely get task name
func getTaskName(args map[string]interface{}) string {
	if name, ok := args["name"].(string); ok {
		return name
	}
	return "unnamed task"
}
