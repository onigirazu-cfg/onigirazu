package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// GroupModuleFixed manages system groups using remote executor
type GroupModuleFixed struct {
	*BaseModule
	executor *executor.CommandExecutor
}

func NewGroupModuleFixed() *GroupModuleFixed {
	return &GroupModuleFixed{
		BaseModule: NewBaseModule("group"),
	}
}

// NewGroupModule creates a new group module (compatibility wrapper)
func NewGroupModule() *GroupModuleFixed {
	return NewGroupModuleFixed()
}

func (m *GroupModuleFixed) GetDescription() string {
	return "Manages system groups"
}

func (m *GroupModuleFixed) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
	}

	// Initialize executor if not already done
	if m.executor == nil {
		exec, err := executor.NewCommandExecutor(host)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create executor: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		m.executor = exec
	}

	// Validate arguments
	if err := m.Validate(args); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, nil
	}

	groupname := args["name"].(string)
	state := args["state"].(string)

	switch state {
	case "present":
		return m.ensureGroupPresent(groupname, args, result, startTime)
	case "absent":
		return m.ensureGroupAbsent(groupname, result, startTime)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("invalid state: %s", state)
		result.Duration = time.Since(startTime)
		return result, nil
	}
}

func (m *GroupModuleFixed) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	name, exists := args["name"]
	if !exists {
		return fmt.Errorf("argument 'name' is required")
	}

	if _, ok := name.(string); !ok {
		return fmt.Errorf("argument 'name' must be a string")
	}

	state, exists := args["state"]
	if !exists {
		return fmt.Errorf("argument 'state' is required")
	}

	if stateStr, ok := state.(string); !ok {
		return fmt.Errorf("argument 'state' must be a string")
	} else if stateStr != "present" && stateStr != "absent" {
		return fmt.Errorf("argument 'state' must be 'present' or 'absent'")
	}

	// Validate optional arguments
	if gid, exists := args["gid"]; exists {
		switch gid.(type) {
		case int, int64, float64:
			// Valid numeric types
		case string:
			// Allow string representation of numbers
		default:
			return fmt.Errorf("argument 'gid' must be a number or string")
		}
	}

	return nil
}

func (m *GroupModuleFixed) ensureGroupPresent(groupname string, args map[string]interface{}, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	if m.groupExists(groupname) {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("Group %s already exists", groupname),
		}
	} else {
		// Create group using remote executor
		cmdArgs := m.buildGroupAddCommand(groupname, args)
		output, err := m.executor.Execute(cmdArgs[0], cmdArgs[1:]...)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error creating group: %v", err)
			result.Output = map[string]interface{}{
				"message": "Group creation failed",
				"error":   err.Error(),
				"stdout":  output,
			}
			result.Duration = time.Since(startTime)
			return result, nil
		}

		result.Success = true
		result.Changed = true
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("Group %s created", groupname),
			"stdout":  output,
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *GroupModuleFixed) ensureGroupAbsent(groupname string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	if !m.groupExists(groupname) {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("Group %s does not exist", groupname),
		}
	} else {
		// Remove group using remote executor
		output, err := m.executor.Execute("groupdel", groupname)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error removing group: %v", err)
			result.Output = map[string]interface{}{
				"message": "Group removal failed",
				"error":   err.Error(),
				"stdout":  output,
			}
			result.Duration = time.Since(startTime)
			return result, nil
		}

		result.Success = true
		result.Changed = true
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("Group %s removed", groupname),
			"stdout":  output,
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *GroupModuleFixed) groupExists(groupname string) bool {
	_, err := m.executor.Execute("getent", "group", groupname)
	return err == nil
}

func (m *GroupModuleFixed) buildGroupAddCommand(groupname string, args map[string]interface{}) []string {
	cmdArgs := []string{"groupadd"}

	// Add GID if specified
	if gid, exists := args["gid"]; exists {
		var gidStr string
		switch v := gid.(type) {
		case int:
			gidStr = fmt.Sprintf("%d", v)
		case int64:
			gidStr = fmt.Sprintf("%d", v)
		case float64:
			gidStr = fmt.Sprintf("%.0f", v)
		case string:
			gidStr = v
		}
		if gidStr != "" {
			cmdArgs = append(cmdArgs, "-g", gidStr)
		}
	}

	// Add groupname as the last argument
	cmdArgs = append(cmdArgs, groupname)

	return cmdArgs
}

func (m *GroupModuleFixed) IsIdempotent() bool {
	return true
}
