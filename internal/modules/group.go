package modules

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// GroupModule manages system groups
type GroupModule struct {
	*BaseModule
}

func NewGroupModule() *GroupModule {
	return &GroupModule{
		BaseModule: NewBaseModule("group"),
	}
}

func (m *GroupModule) GetDescription() string {
	return "Manages system groups"
}

func (m *GroupModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
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
		result.Error = fmt.Sprintf("unsupported state: %s", state)
		result.Duration = time.Since(startTime)
		return result, nil
	}
}

func (m *GroupModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	state, exists := args["state"]
	if !exists {
		return fmt.Errorf("argument 'state' is required")
	}

	if _, ok := state.(string); !ok {
		return fmt.Errorf("argument 'state' must be a string")
	}

	validStates := []string{"present", "absent"}
	stateStr := state.(string)
	for _, validState := range validStates {
		if stateStr == validState {
			return nil
		}
	}

	return fmt.Errorf("unsupported state: %s", stateStr)
}

func (m *GroupModule) ensureGroupPresent(groupname string, args map[string]interface{}, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Check if group exists
	if m.groupExists(groupname) {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("Group %s already exists", groupname),
		}
	} else {
		// Create group
		cmd := m.buildGroupAddCommand(groupname, args)
		if err := cmd.Run(); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error creating group: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		result.Success = true
		result.Changed = true
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("Group %s created", groupname),
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *GroupModule) ensureGroupAbsent(groupname string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	if !m.groupExists(groupname) {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("Group %s does not exist", groupname),
		}
	} else {
		// Remove group
		cmd := exec.Command("groupdel", groupname)
		if err := cmd.Run(); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error removing group: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		result.Success = true
		result.Changed = true
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("Group %s removed", groupname),
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *GroupModule) groupExists(groupname string) bool {
	cmd := exec.Command("getent", "group", groupname)
	return cmd.Run() == nil
}

func (m *GroupModule) buildGroupAddCommand(groupname string, args map[string]interface{}) *exec.Cmd {
	cmdArgs := []string{"groupadd"}

	// Add system group flag if specified
	if system, exists := args["system"]; exists {
		if systemBool, ok := system.(bool); ok && systemBool {
			cmdArgs = append(cmdArgs, "-r")
		}
	}

	// Add GID if specified
	if gid, exists := args["gid"]; exists {
		if gidStr, ok := gid.(string); ok && gidStr != "" {
			cmdArgs = append(cmdArgs, "-g", gidStr)
		}
	}

	cmdArgs = append(cmdArgs, groupname)
	return exec.Command(cmdArgs[0], cmdArgs[1:]...)
}
