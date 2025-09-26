package modules

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// UserModule manages system users
type UserModule struct {
	*BaseModule
}

func NewUserModule() *UserModule {
	return &UserModule{
		BaseModule: NewBaseModule("user"),
	}
}

func (m *UserModule) GetDescription() string {
	return "Manages system users"
}

func (m *UserModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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

	username := args["name"].(string)
	state := args["state"].(string)

	switch state {
	case "present":
		return m.ensureUserPresent(username, args, result, startTime)
	case "absent":
		return m.ensureUserAbsent(username, result, startTime)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("unsupported state: %s", state)
		result.Duration = time.Since(startTime)
		return result, nil
	}
}

func (m *UserModule) Validate(args map[string]interface{}) error {
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

func (m *UserModule) ensureUserPresent(username string, args map[string]interface{}, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Check if user exists
	if m.userExists(username) {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("User %s already exists", username),
		}
	} else {
		// Create user
		cmd := m.buildUserAddCommand(username, args)
		if err := cmd.Run(); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error creating user: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		result.Success = true
		result.Changed = true
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("User %s created", username),
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *UserModule) ensureUserAbsent(username string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	if !m.userExists(username) {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("User %s does not exist", username),
		}
	} else {
		// Remove user
		cmd := exec.Command("userdel", "-r", username)
		if err := cmd.Run(); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error removing user: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		result.Success = true
		result.Changed = true
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("User %s removed", username),
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *UserModule) userExists(username string) bool {
	cmd := exec.Command("id", username)
	return cmd.Run() == nil
}

func (m *UserModule) buildUserAddCommand(username string, args map[string]interface{}) *exec.Cmd {
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

	// Add groups if specified
	if groups, exists := args["groups"]; exists {
		if groupsStr, ok := groups.(string); ok && groupsStr != "" {
			cmdArgs = append(cmdArgs, "-G", groupsStr)
		}
	}

	// Add system user flag if specified
	if system, exists := args["system"]; exists {
		if systemBool, ok := system.(bool); ok && systemBool {
			cmdArgs = append(cmdArgs, "-r")
		}
	}

	// Add comment if specified
	if comment, exists := args["comment"]; exists {
		if commentStr, ok := comment.(string); ok && commentStr != "" {
			cmdArgs = append(cmdArgs, "-c", commentStr)
		}
	}

	cmdArgs = append(cmdArgs, username)
	return exec.Command(cmdArgs[0], cmdArgs[1:]...)
}
