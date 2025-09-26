package modules

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CommandModule executes shell commands
type CommandModule struct {
	*BaseModule
}

func NewCommandModule() *CommandModule {
	return &CommandModule{
		BaseModule: NewBaseModule("command"),
	}
}

func (m *CommandModule) GetDescription() string {
	return "Executes commands on remote hosts"
}

func (m *CommandModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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

	command := args["command"].(string)
	shell := args["shell"].(bool)

	if shell {
		return m.executeShellCommand(command, result, startTime)
	} else {
		return m.executeCommand(command, result, startTime)
	}
}

func (m *CommandModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	command, exists := args["command"]
	if !exists {
		return fmt.Errorf("argument 'command' is required")
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

func (m *CommandModule) executeCommand(command string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Split command into parts for exec.Command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		result.Success = false
		result.Failed = true
		result.Error = "command is empty"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("command is empty")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := cmd.Output()

	if err != nil {
		result.Success = false
		result.Failed = true
		result.Error = fmt.Sprintf("command failed: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("command failed: %v", err)
	}

	result.Success = true
	result.Changed = true
	result.Output = map[string]interface{}{
		"message": "Command executed successfully",
		"stdout":  string(output),
		"command": command,
	}
	result.Duration = time.Since(startTime)

	return result, nil
}

func (m *CommandModule) executeShellCommand(command string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Execute command through shell
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.Output()

	if err != nil {
		result.Success = false
		result.Failed = true
		result.Error = fmt.Sprintf("shell command failed: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("shell command failed: %v", err)
	}

	result.Success = true
	result.Changed = true
	result.Output = map[string]interface{}{
		"message": "Shell command executed successfully",
		"stdout":  string(output),
		"command": command,
	}
	result.Duration = time.Since(startTime)

	return result, nil
}

// ShellModule executes shell commands with advanced features
type ShellModule struct {
	BaseModule
}

func NewShellModule() *ShellModule {
	return &ShellModule{
		BaseModule: BaseModule{
			name:        "shell",
			description: "Executes shell commands with shell interpretation",
		},
	}
}

func (m *ShellModule) GetDescription() string {
	return "Executes shell commands with shell interpretation"
}

func (m *ShellModule) Validate(args map[string]interface{}) error {
	command, ok := args["command"]
	if !ok {
		return fmt.Errorf("command is required")
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

func (m *ShellModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		Success: false,
		Changed: false,
		Output:  make(map[string]interface{}),
	}

	command, _ := args["command"].(string)

	// Prepare the command
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)

	// Set working directory if specified
	if chdir, ok := args["chdir"].(string); ok {
		cmd.Dir = chdir
	}

	// Set environment variables if specified
	if env, ok := args["environment"].(map[string]interface{}); ok {
		cmd.Env = os.Environ() // Start with current environment
		for key, value := range env {
			if strValue, ok := value.(string); ok {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, strValue))
			}
		}
	}

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Output = map[string]interface{}{
			"message": "Shell command failed",
			"error":   err.Error(),
			"stdout":  string(output),
			"command": command,
		}
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("command failed: %v", err)
	}

	result.Success = true
	result.Changed = true
	result.Output = map[string]interface{}{
		"message": "Shell command executed successfully",
		"stdout":  string(output),
		"command": command,
	}
	result.Duration = time.Since(startTime)

	return result, nil
}

func (m *ShellModule) IsIdempotent() bool {
	// Shell commands are generally not idempotent
	return false
}
