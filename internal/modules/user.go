package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// UserModuleFixed manages system users using remote executor
type UserModuleFixed struct {
	*BaseModule
}

func NewUserModuleFixed() *UserModuleFixed {
	return &UserModuleFixed{
		BaseModule: NewBaseModule("user"),
	}
}

// NewUserModule creates a new user module (compatibility wrapper)
func NewUserModule() *UserModuleFixed {
	return NewUserModuleFixed()
}

func (m *UserModuleFixed) GetDescription() string {
	return "Manages system users"
}

// PreCheckState checks if user is already in the desired state
func (m *UserModuleFixed) PreCheckState(ctx context.Context, host types.Host, args map[string]interface{}) (*PreCheckResult, error) {
	// Get username
	username, ok := args["name"].(string)
	if !ok || username == "" {
		return &PreCheckResult{
			ShouldExecute: true,
			Reason:        "No username specified",
		}, nil
	}

	state := "present"
	if stateVal, exists := args["state"]; exists {
		if stateStr, ok := stateVal.(string); ok {
			state = stateStr
		}
	}

	// Check current user existence using getent (fast: ~30ms)
	_, err := runOnHost(ctx, host, args, "getent", "passwd", username)
	userExists := err == nil

	currentState := map[string]interface{}{
		"exists": userExists,
	}

	// Check if desired state matches current state
	allCorrect := false
	if state == "present" && userExists {
		allCorrect = true
	} else if state == "absent" && !userExists {
		allCorrect = true
	}

	if allCorrect {
		// State is already correct - skip execution
		return &PreCheckResult{
			IsStateCorrect: true,
			ShouldExecute:  false,
			Reason:         fmt.Sprintf("User already in state: %s", state),
			CurrentState:   currentState,
		}, nil
	}

	// State needs to change - execute the operation
	return &PreCheckResult{
		IsStateCorrect: false,
		ShouldExecute:  true,
		Reason:         fmt.Sprintf("User needs to be set to state: %s", state),
		CurrentState:   currentState,
	}, nil
}

func (m *UserModuleFixed) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  taskName(args),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
	}
	normalizeUserArgs(args)
	if err := m.Validate(args); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, nil
	}
	return m.convergeUser(ctx, host, args, result, startTime)
}

func (m *UserModuleFixed) Validate(args map[string]interface{}) error {
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
		state = "present"
	}

	if stateStr, ok := state.(string); !ok {
		return fmt.Errorf("argument 'state' must be a string")
	} else if stateStr != "present" && stateStr != "absent" {
		return fmt.Errorf("argument 'state' must be 'present' or 'absent'")
	}

	// Validate optional arguments
	if shell, exists := args["shell"]; exists {
		if _, ok := shell.(string); !ok {
			return fmt.Errorf("argument 'shell' must be a string")
		}
	}

	if home, exists := args["home"]; exists {
		if _, ok := home.(string); !ok {
			return fmt.Errorf("argument 'home' must be a string")
		}
	}

	if group, exists := args["group"]; exists {
		if _, ok := group.(string); !ok {
			return fmt.Errorf("argument 'group' must be a string")
		}
	}

	if groups, exists := args["groups"]; exists {
		switch groups.(type) {
		case string, []interface{}:
		default:
			return fmt.Errorf("argument 'groups' must be a list or a comma separated string")
		}
	}

	if uid, exists := args["uid"]; exists {
		switch uid.(type) {
		case int, int64, float64:
			// Valid numeric types
		case string:
			// Allow string representation of numbers
		default:
			return fmt.Errorf("argument 'uid' must be a number or string")
		}
	}

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

func (m *UserModuleFixed) buildUserAddCommand(username string, args map[string]interface{}) []string {
	cmdArgs := []string{"useradd"}

	// Add home directory if specified
	if homeDir, exists := args["home"]; exists {
		if homeDirStr, ok := homeDir.(string); ok && homeDirStr != "" {
			cmdArgs = append(cmdArgs, "-d", homeDirStr)
		}
	}

	// Add shell if specified
	if shell, exists := args["shell"]; exists {
		if shellStr, ok := shell.(string); ok && shellStr != "" {
			cmdArgs = append(cmdArgs, "-s", shellStr)
		}
	}

	// Add primary group if specified
	if group, exists := args["group"]; exists {
		if groupStr, ok := group.(string); ok && groupStr != "" {
			cmdArgs = append(cmdArgs, "-g", groupStr)
		}
	}

	// Add supplementary groups if specified
	if groups, exists := args["groups"]; exists {
		if groupsStr, ok := groups.(string); ok && groupsStr != "" {
			cmdArgs = append(cmdArgs, "-G", groupsStr)
		}
	}

	// Add UID if specified
	if uid, exists := args["uid"]; exists {
		var uidStr string
		switch v := uid.(type) {
		case int:
			uidStr = fmt.Sprintf("%d", v)
		case int64:
			uidStr = fmt.Sprintf("%d", v)
		case float64:
			uidStr = fmt.Sprintf("%.0f", v)
		case string:
			uidStr = v
		}
		if uidStr != "" {
			cmdArgs = append(cmdArgs, "-u", uidStr)
		}
	}

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

	if comment, ok := args["comment"].(string); ok && comment != "" {
		cmdArgs = append(cmdArgs, "-c", comment)
	}
	if password := getStringArg(args, "password", ""); password != "" {
		cmdArgs = append(cmdArgs, "-p", password)
	}
	if getBoolArg(args, "system", false) {
		cmdArgs = append(cmdArgs, "-r")
	}

	// Create home directory by default
	if getBoolArg(args, "create_home", true) {
		cmdArgs = append(cmdArgs, "-m")
	} else {
		cmdArgs = append(cmdArgs, "-M")
	}

	// Add username as the last argument
	cmdArgs = append(cmdArgs, username)

	return cmdArgs
}

func (m *UserModuleFixed) IsIdempotent() bool {
	return true
}
