package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
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

func (m *UserModuleFixed) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
	}

	// Create a fresh executor for this execution
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	defer exec.Close()

	// Configure become (privilege escalation)
	if become, ok := args["_become"].(bool); ok && become {
		becomeUser, _ := args["_become_user"].(string)
		becomeMethod, _ := args["_become_method"].(string)
		exec.SetBecome(true, becomeUser, becomeMethod)
	}

	// Validate arguments
	if err := m.Validate(args); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, nil
	}

	username := args["name"].(string)
	state := args["state"].(string)

	switch state {
	case "present":
		return m.ensureUserPresent(exec, username, args, result, startTime)
	case "absent":
		return m.ensureUserAbsent(exec, username, result, startTime)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("invalid state: %s", state)
		result.Duration = time.Since(startTime)
		return result, nil
	}
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
		return fmt.Errorf("argument 'state' is required")
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
		if _, ok := groups.(string); !ok {
			return fmt.Errorf("argument 'groups' must be a string")
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

func (m *UserModuleFixed) ensureUserPresent(exec *executor.CommandExecutor, username string, args map[string]interface{}, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	if m.userExists(exec, username) {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("User %s already exists", username),
		}
	} else {
		// Create user using remote executor
		cmdArgs := m.buildUserAddCommand(username, args)
		output, err := exec.Execute(cmdArgs[0], cmdArgs[1:]...)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error creating user: %v", err)
			result.Output = map[string]interface{}{
				"message": "User creation failed",
				"error":   err.Error(),
				"stdout":  output,
			}
			result.Duration = time.Since(startTime)
			return result, nil
		}

		result.Success = true
		result.Changed = true
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("User %s created", username),
			"stdout":  output,
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *UserModuleFixed) ensureUserAbsent(exec *executor.CommandExecutor, username string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	if !m.userExists(exec, username) {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("User %s does not exist", username),
		}
	} else {
		// Remove user using remote executor
		output, err := exec.Execute("userdel", "-r", username)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error removing user: %v", err)
			result.Output = map[string]interface{}{
				"message": "User removal failed",
				"error":   err.Error(),
				"stdout":  output,
			}
			result.Duration = time.Since(startTime)
			return result, nil
		}

		result.Success = true
		result.Changed = true
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("User %s removed", username),
			"stdout":  output,
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *UserModuleFixed) userExists(exec *executor.CommandExecutor, username string) bool {
	_, err := exec.Execute("id", username)
	return err == nil
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

	// Create home directory by default
	cmdArgs = append(cmdArgs, "-m")

	// Add username as the last argument
	cmdArgs = append(cmdArgs, username)

	return cmdArgs
}

func (m *UserModuleFixed) IsIdempotent() bool {
	return true
}
