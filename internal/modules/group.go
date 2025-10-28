package modules

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// GroupModuleFixed manages system groups using remote executor
type GroupModuleFixed struct {
	*BaseModule
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

// PreCheckState checks if group is already in the desired state
func (m *GroupModuleFixed) PreCheckState(ctx context.Context, host types.Host, args map[string]interface{}) (*PreCheckResult, error) {
	// Get groupname
	groupname, ok := args["name"].(string)
	if !ok || groupname == "" {
		return &PreCheckResult{
			ShouldExecute: true,
			Reason:        "No groupname specified",
		}, nil
	}

	state := "present"
	if stateVal, exists := args["state"]; exists {
		if stateStr, ok := stateVal.(string); ok {
			state = stateStr
		}
	}

	// Check current group existence using getent (fast: ~30ms)
	cmd := exec.CommandContext(ctx, "getent", "group", groupname)
	groupExists := cmd.Run() == nil

	currentState := map[string]interface{}{
		"exists": groupExists,
	}

	// Check if desired state matches current state
	allCorrect := false
	if state == "present" && groupExists {
		allCorrect = true
	} else if state == "absent" && !groupExists {
		allCorrect = true
	}

	if allCorrect {
		// State is already correct - skip execution
		return &PreCheckResult{
			IsStateCorrect: true,
			ShouldExecute:  false,
			Reason:         fmt.Sprintf("Group already in state: %s", state),
			CurrentState:   currentState,
		}, nil
	}

	// State needs to change - execute the operation
	return &PreCheckResult{
		IsStateCorrect: false,
		ShouldExecute:  true,
		Reason:         fmt.Sprintf("Group needs to be set to state: %s", state),
		CurrentState:   currentState,
	}, nil
}

func (m *GroupModuleFixed) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
	}

	// Pre-check: if state is already correct, skip execution
	preCheck, err := m.PreCheckState(ctx, host, args)
	if err == nil && preCheck.IsStateCorrect {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message":     preCheck.Reason,
			"pre_checked": true,
			"group_state": preCheck.CurrentState,
		}
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Initialize executor for this execution
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	defer exec.Close()

	// Configure become (privilege escalation) - always reset to ensure correct state
	if become, ok := args["_become"].(bool); ok && become {
		becomeUser, _ := args["_become_user"].(string)
		becomeMethod, _ := args["_become_method"].(string)
		exec.SetBecome(true, becomeUser, becomeMethod)
	} else {
		// Reset become if not requested
		exec.SetBecome(false, "", "")
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
		return m.ensureGroupPresent(exec, groupname, args, result, startTime)
	case "absent":
		return m.ensureGroupAbsent(exec, groupname, result, startTime)
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

	// Validate system argument if provided
	if system, exists := args["system"]; exists {
		if systemBool, ok := system.(bool); ok {
			// Valid boolean
			_ = systemBool
		} else if systemStr, ok := system.(string); ok {
			// Allow string representation of boolean
			if systemStr != "true" && systemStr != "false" && systemStr != "yes" && systemStr != "no" {
				return fmt.Errorf("argument 'system' must be a boolean or 'true'/'false'/'yes'/'no'")
			}
		} else {
			return fmt.Errorf("argument 'system' must be a boolean")
		}
	}

	return nil
}

func (m *GroupModuleFixed) ensureGroupPresent(exec *executor.CommandExecutor, groupname string, args map[string]interface{}, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	if m.groupExists(exec, groupname) {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("Group %s already exists", groupname),
		}
	} else {
		// Create group using remote executor
		cmdArgs := m.buildGroupAddCommand(groupname, args)
		output, err := exec.Execute(cmdArgs[0], cmdArgs[1:]...)
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

func (m *GroupModuleFixed) ensureGroupAbsent(exec *executor.CommandExecutor, groupname string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	if !m.groupExists(exec, groupname) {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("Group %s does not exist", groupname),
		}
	} else {
		// Remove group using remote executor
		output, err := exec.Execute("groupdel", groupname)
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

func (m *GroupModuleFixed) groupExists(exec *executor.CommandExecutor, groupname string) bool {
	_, err := exec.Execute("getent", "group", groupname)
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
