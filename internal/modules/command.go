package modules

import (
	"context"
	"fmt"
	"sort"
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

	return runCommand(ctx, exec, args, getBoolArg(args, "shell", false), result, startTime)
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

// runCommand runs a command or shell task: the command line gets chdir,
// the module's environment argument and executable (shell only), and the
// result carries stdout, stderr and rc as in Ansible. A non-zero exit code
// fails the task; failed_when can still accept it.
func runCommand(ctx context.Context, exec *executor.CommandExecutor, args map[string]interface{}, shell bool, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	command := getStringArg(args, "command", getStringArg(args, "cmd", ""))
	line := command
	if !shell {
		// split like a shell would (quotes, backslashes), then quote each word:
		// no shell syntax is interpreted
		parts, err := splitCommandLine(command)
		if err == nil && len(parts) == 0 {
			err = fmt.Errorf("command is empty")
		}
		if err != nil {
			result.Success, result.Failed, result.Error = false, true, err.Error()
			result.Duration = time.Since(startTime)
			return result, nil
		}
		line = shellJoin(parts...)
	} else if executable := getStringArg(args, "executable", ""); executable != "" {
		line = shellJoin(executable, "-c", command)
	}
	if env, ok := args["environment"].(map[string]interface{}); ok && len(env) > 0 {
		keys := make([]string, 0, len(env))
		for k := range env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		words := []string{"env"}
		for _, k := range keys {
			words = append(words, k+"="+fmt.Sprint(env[k]))
		}
		// its own shell, so $VAR in the command sees the new value
		line = shellJoin(append(words, "sh", "-c", line)...)
	}
	if chdir := getStringArg(args, "chdir", ""); chdir != "" {
		line = "cd " + shellQuote(chdir) + " && " + line
	}

	run, err := exec.Run(ctx, line)
	end := time.Now()
	stdout := strings.TrimRight(run.Stdout, "\r\n")
	stderr := strings.TrimRight(run.Stderr, "\r\n")
	result.Output = map[string]interface{}{
		"cmd":    command,
		"stdout": stdout,
		"stderr": stderr,
		"rc":     run.RC,
		"start":  startTime.Format("2006-01-02 15:04:05.000000"),
		"end":    end.Format("2006-01-02 15:04:05.000000"),
		"delta":  end.Sub(startTime).String(),
	}
	result.Duration = time.Since(startTime)
	if err != nil {
		result.Success, result.Failed = false, true
		result.Error = fmt.Sprintf("command could not run: %v", err)
		return result, nil
	}
	result.Changed = true
	if run.RC != 0 {
		result.Success, result.Failed = false, true
		result.Error = fmt.Sprintf("non-zero return code %d", run.RC)
		if stderr != "" {
			result.Error += ": " + lastLines(stderr, 5)
		}
		return result, nil
	}
	result.Success = true
	return result, nil
}

// lastLines is the tail of a text, for error messages
func lastLines(text string, n int) string {
	lines := strings.Split(text, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " | ")
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

	if skip, msg := skipByCreatesRemoves(exec, args); skip {
		result.Success = true
		result.Output = map[string]interface{}{"msg": msg}
		result.Duration = time.Since(startTime)
		return result, nil
	}
	return runCommand(ctx, exec, args, true, result, startTime)
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
