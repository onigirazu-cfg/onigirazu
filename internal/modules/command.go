package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CommandModuleFixed executes shell commands using remote executor
type CommandModuleFixed struct {
	*BaseModule
	executor *executor.CommandExecutor
}

func NewCommandModuleFixed() *CommandModuleFixed {
	return &CommandModuleFixed{
		BaseModule: NewBaseModule("command"),
	}
}

// NewCommandModule creates a new command module (compatibility wrapper)
func NewCommandModule() *CommandModuleFixed {
	return NewCommandModuleFixed()
}

func (m *CommandModuleFixed) GetDescription() string {
	return "Executes commands on remote hosts"
}

func (m *CommandModuleFixed) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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

	command := args["command"].(string)
	shell := false
	if shellVal, exists := args["shell"]; exists {
		if shellBool, ok := shellVal.(bool); ok {
			shell = shellBool
		}
	}

	if shell {
		return m.executeShellCommand(command, result, startTime)
	} else {
		return m.executeCommand(command, result, startTime)
	}
}

func (m *CommandModuleFixed) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	// Support both 'command' and 'cmd' (Ansible compatibility)
	command, hasCommand := args["command"]
	cmd, hasCmd := args["cmd"]

	if !hasCommand && !hasCmd {
		return fmt.Errorf("argument 'command' or 'cmd' is required")
	}

	// Use cmd if command is not provided
	if !hasCommand && hasCmd {
		args["command"] = cmd
		command = cmd
	}

	if _, ok := command.(string); !ok {
		return fmt.Errorf("argument 'command' must be a string")
	}

	// Check if command is not empty
	if strings.TrimSpace(command.(string)) == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Validate shell parameter if provided
	if shell, exists := args["shell"]; exists {
		if _, ok := shell.(bool); !ok {
			return fmt.Errorf("argument 'shell' must be a boolean")
		}
	}

	return nil
}

func (m *CommandModuleFixed) executeCommand(command string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Split command into parts for execution
	parts := strings.Fields(command)
	if len(parts) == 0 {
		result.Success = false
		result.Failed = true
		result.Error = "command is empty"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("command is empty")
	}

	// Execute using remote executor
	output, err := m.executor.Execute(parts[0], parts[1:]...)

	if err != nil {
		result.Success = false
		result.Failed = true
		result.Error = fmt.Sprintf("command failed: %v", err)
		result.Output = map[string]interface{}{
			"message": "Command execution failed",
			"error":   err.Error(),
			"stdout":  output,
			"command": command,
		}
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("command failed: %v", err)
	}

	result.Success = true
	result.Changed = true
	result.Output = map[string]interface{}{
		"message": "Command executed successfully",
		"stdout":  output,
		"command": command,
	}
	result.Duration = time.Since(startTime)

	return result, nil
}

func (m *CommandModuleFixed) executeShellCommand(command string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Execute command through shell using remote executor
	output, err := m.executor.Execute("sh", "-c", command)

	if err != nil {
		result.Success = false
		result.Failed = true
		result.Error = fmt.Sprintf("shell command failed: %v", err)
		result.Output = map[string]interface{}{
			"message": "Shell command execution failed",
			"error":   err.Error(),
			"stdout":  output,
			"command": command,
		}
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("shell command failed: %v", err)
	}

	result.Success = true
	result.Changed = true
	result.Output = map[string]interface{}{
		"message": "Shell command executed successfully",
		"stdout":  output,
		"command": command,
	}
	result.Duration = time.Since(startTime)

	return result, nil
}

// ShellModuleFixed executes shell commands with advanced features using remote executor
type ShellModuleFixed struct {
	BaseModule
	executor *executor.CommandExecutor
}

func NewShellModuleFixed() *ShellModuleFixed {
	return &ShellModuleFixed{
		BaseModule: BaseModule{
			name:        "shell",
			description: "Executes shell commands with shell interpretation",
		},
	}
}

// NewShellModule creates a new shell module (compatibility wrapper)
func NewShellModule() *ShellModuleFixed {
	return NewShellModuleFixed()
}

func (m *ShellModuleFixed) GetDescription() string {
	return "Executes shell commands with shell interpretation"
}

func (m *ShellModuleFixed) Validate(args map[string]interface{}) error {
	// Support both 'command' and 'cmd' (Ansible compatibility)
	command, hasCommand := args["command"]
	cmd, hasCmd := args["cmd"]

	if !hasCommand && !hasCmd {
		return fmt.Errorf("command or cmd is required")
	}

	// Use cmd if command is not provided
	if !hasCommand && hasCmd {
		args["command"] = cmd
		command = cmd
	}

	if _, ok := command.(string); !ok {
		return fmt.Errorf("command must be a string")
	}

	// Validate chdir if provided
	if chdir, ok := args["chdir"]; ok {
		if _, ok := chdir.(string); !ok {
			return fmt.Errorf("chdir must be a string")
		}
	}

	// Validate environment if provided
	if env, ok := args["environment"]; ok {
		switch envVal := env.(type) {
		case map[string]interface{}:
			// Validate that all values are strings
			for key, value := range envVal {
				if _, ok := value.(string); !ok {
					return fmt.Errorf("environment variable %s must be a string", key)
				}
			}
		default:
			return fmt.Errorf("environment must be a map of string to string")
		}
	}

	return nil
}

func (m *ShellModuleFixed) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		Success: false,
		Changed: false,
		Output:  make(map[string]interface{}),
	}

	// Initialize executor if not already done
	if m.executor == nil {
		exec, err := executor.NewCommandExecutor(host)
		if err != nil {
			result.Error = fmt.Sprintf("failed to create executor: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("failed to create executor: %v", err)
		}
		m.executor = exec
	}

	// Support both 'command' and 'cmd' (Ansible compatibility)
	command, hasCommand := args["command"].(string)
	if !hasCommand {
		if cmd, hasCmd := args["cmd"].(string); hasCmd {
			command = cmd
			args["command"] = cmd
		}
	}

	// Build the command with environment and working directory
	fullCommand := command

	// Handle working directory change
	if chdir, ok := args["chdir"].(string); ok {
		fullCommand = fmt.Sprintf("cd %s && %s", chdir, command)
	}

	// Handle environment variables
	if env, ok := args["environment"].(map[string]interface{}); ok {
		envVars := make([]string, 0, len(env))
		for key, value := range env {
			if strValue, ok := value.(string); ok {
				envVars = append(envVars, fmt.Sprintf("%s=%s", key, strValue))
			}
		}
		if len(envVars) > 0 {
			envString := strings.Join(envVars, " ")
			fullCommand = fmt.Sprintf("env %s %s", envString, fullCommand)
		}
	}

	// Execute the command using remote executor
	output, err := m.executor.Execute("sh", "-c", fullCommand)

	if err != nil {
		result.Output = map[string]interface{}{
			"message": "Shell command failed",
			"error":   err.Error(),
			"stdout":  output,
			"command": command,
		}
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("command failed: %v", err)
	}

	result.Success = true
	result.Changed = true
	result.Output = map[string]interface{}{
		"message": "Shell command executed successfully",
		"stdout":  output,
		"command": command,
	}
	result.Duration = time.Since(startTime)

	return result, nil
}

func (m *ShellModuleFixed) IsIdempotent() bool {
	// Shell commands are generally not idempotent
	return false
}
