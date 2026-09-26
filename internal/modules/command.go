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
		TaskName:  taskName(args),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
	}

	// Create executor for this specific host
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	defer exec.Close()

	// Validate arguments
	if err := m.Validate(args); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, nil
	}

	if skip, msg := skipByCreatesRemoves(exec, args); skip {
		result.Success = true
		result.Output = map[string]interface{}{"msg": msg}
		result.Duration = time.Since(startTime)
		return result, nil
	}

	command, _ := args["command"].(string)
	shell := getBoolArg(args, "shell", false)

	if shell {
		return m.executeShellCommand(exec, ctx, command, result, startTime)
	} else {
		return m.executeCommand(exec, ctx, command, result, startTime)
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

	cmdStr, ok := command.(string)
	if !ok {
		return fmt.Errorf("argument 'command' must be a string")
	}

	// Check if command is not empty
	if strings.TrimSpace(cmdStr) == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Validate shell parameter if provided
	if shell, exists := args["shell"]; exists {
		if _, ok := parseBool(shell); !ok {
			return fmt.Errorf("argument 'shell' must be a boolean")
		}
	}

	return nil
}

func (m *CommandModuleFixed) executeCommand(exec *executor.CommandExecutor, ctx context.Context, command string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Split like a shell would (quotes, backslashes), without running one
	parts, err := splitCommandLine(command)
	if err != nil {
		result.Success = false
		result.Failed = true
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}
	if len(parts) == 0 {
		result.Success = false
		result.Failed = true
		result.Error = "command is empty"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("command is empty")
	}

	// Execute using remote executor with context for graceful shutdown support
	output, err := exec.ExecuteWithContext(ctx, parts[0], parts[1:]...)

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

func (m *CommandModuleFixed) executeShellCommand(exec *executor.CommandExecutor, ctx context.Context, command string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Execute command through shell using remote executor with context for graceful shutdown support
	output, err := exec.ExecuteWithContext(ctx, "sh", "-c", command)

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

	// Create executor for this specific host
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("failed to create executor: %v", err)
	}
	defer exec.Close()

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

	if skip, msg := skipByCreatesRemoves(exec, args); skip {
		result.Success = true
		result.Output = map[string]interface{}{"msg": msg}
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Handle working directory change
	if chdir, ok := args["chdir"].(string); ok {
		fullCommand = fmt.Sprintf("cd %s && %s", shellQuote(chdir), command)
	}

	// Handle environment variables
	if env, ok := args["environment"].(map[string]interface{}); ok {
		envVars := make([]string, 0, len(env))
		for key, value := range env {
			if strValue, ok := value.(string); ok {
				envVars = append(envVars, shellQuote(key+"="+strValue))
			}
		}
		if len(envVars) > 0 {
			envString := strings.Join(envVars, " ")
			// its own shell, so $VAR in the command sees the new value
			fullCommand = fmt.Sprintf("env %s sh -c %s", envString, shellQuote(fullCommand))
		}
	}

	// Execute the command using remote executor
	// Note: executor.Execute will automatically use shell if needed
	output, execErr := exec.Execute(fullCommand)

	if execErr != nil {
		result.Output = map[string]interface{}{
			"message": "Shell command failed",
			"error":   execErr.Error(),
			"stdout":  output,
			"command": command,
		}
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("command failed: %v", execErr)
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

// splitCommandLine splits a command line into words the way a POSIX shell
// does for quoting: single quotes are literal, double quotes allow backslash
// escapes of " \ $ `, and a backslash outside quotes escapes the next char.
// Operators such as > or | stay ordinary words (use shell: true for those).
func splitCommandLine(line string) ([]string, error) {
	var words []string
	var cur strings.Builder
	inWord := false
	const (
		none = iota
		single
		double
	)
	quote := none
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch quote {
		case single:
			if c == '\'' {
				quote = none
			} else {
				cur.WriteRune(c)
			}
		case double:
			switch {
			case c == '"':
				quote = none
			case c == '\\' && i+1 < len(runes) && strings.ContainsRune("\"\\$`", runes[i+1]):
				i++
				cur.WriteRune(runes[i])
			default:
				cur.WriteRune(c)
			}
		default:
			switch {
			case c == ' ' || c == '\t' || c == '\n':
				if inWord {
					words = append(words, cur.String())
					cur.Reset()
					inWord = false
				}
			case c == '\'':
				quote, inWord = single, true
			case c == '"':
				quote, inWord = double, true
			case c == '\\' && i+1 < len(runes):
				i++
				cur.WriteRune(runes[i])
				inWord = true
			default:
				cur.WriteRune(c)
				inWord = true
			}
		}
	}
	if quote != none {
		return nil, fmt.Errorf("unterminated quote in command: %s", line)
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words, nil
}

// skipByCreatesRemoves applies creates/removes: the command is skipped when
// the creates path already exists or the removes path does not
func skipByCreatesRemoves(exec *executor.CommandExecutor, args map[string]interface{}) (bool, string) {
	exists := func(path string) bool {
		_, err := exec.Execute("test -e " + shellQuote(path))
		return err == nil
	}
	if path := getStringArg(args, "creates", ""); path != "" && exists(path) {
		return true, fmt.Sprintf("skipped: %s exists", path)
	}
	if path := getStringArg(args, "removes", ""); path != "" && !exists(path) {
		return true, fmt.Sprintf("skipped: %s does not exist", path)
	}
	return false, ""
}
