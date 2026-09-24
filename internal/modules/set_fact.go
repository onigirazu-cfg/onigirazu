package modules

import (
	"context"
	"fmt"
	"strings"
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
		TaskName:  taskName(args),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   true,
		Changed:   true, // Setting facts is always considered a change
		Output:    make(map[string]interface{}),
	}

	facts := factArgs(args)

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

	hasFactToSet := len(factArgs(args)) > 0

	if !hasFactToSet {
		return fmt.Errorf("set_fact module requires at least one fact to set")
	}

	return nil
}

// factArgs returns the facts to set: every argument except internal "_" keys
func factArgs(args map[string]interface{}) map[string]interface{} {
	facts := make(map[string]interface{})
	for key, value := range args {
		if !strings.HasPrefix(key, "_") {
			facts[key] = value
		}
	}
	return facts
}
