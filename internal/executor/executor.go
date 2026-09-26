package executor

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CommandExecutor handles command execution on local or remote hosts
type CommandExecutor struct {
	host         types.Host
	sshClient    *sshpkg.Client
	usePool      bool
	poolReleased bool
	become       bool
	becomeUser   string
	becomeMethod string
	env          string // "env 'K=V' ..." for the task's environment, or empty
}

// NewCommandExecutor creates a new command executor for the given host
// Uses connection pooling for remote hosts by default
func NewCommandExecutor(host types.Host) (*CommandExecutor, error) {
	executor := &CommandExecutor{
		host:    host,
		usePool: true, // Enable pooling by default
	}

	isLocal := sshpkg.IsLocal(host)

	// If it's not a local host, get SSH client from pool
	if !isLocal {
		pool := sshpkg.GetGlobalPool()
		client, err := pool.GetConnection(host)
		if err != nil {
			return nil, err
		}
		executor.sshClient = client
	}
	if host.Become {
		executor.SetBecome(true, host.BecomeUser, host.BecomeMethod)
	}
	executor.setEnvironment(host.Environment)

	return executor, nil
}

// NewCommandExecutorWithoutPool creates a new command executor without using connection pool
// Useful for testing or when connection pooling is not desired
func NewCommandExecutorWithoutPool(host types.Host) (*CommandExecutor, error) {
	executor := &CommandExecutor{
		host:    host,
		usePool: false,
	}

	isLocal := sshpkg.IsLocal(host)

	// If it's not a local host, create SSH client directly
	if !isLocal {
		client, err := sshpkg.NewClient(host)
		if err != nil {
			return nil, err
		}
		executor.sshClient = client
	}
	if host.Become {
		executor.SetBecome(true, host.BecomeUser, host.BecomeMethod)
	}
	executor.setEnvironment(host.Environment)

	return executor, nil
}

// SetBecome enables privilege escalation for command execution
func (e *CommandExecutor) SetBecome(become bool, becomeUser, becomeMethod string) {
	e.become = become
	e.becomeUser = becomeUser
	if becomeMethod == "" {
		e.becomeMethod = "sudo" // Default to sudo
	} else {
		e.becomeMethod = becomeMethod
	}
	if e.becomeUser == "" {
		e.becomeUser = "root" // Default to root
	}
}

// wrapWithBecome runs the whole command line through a root (or become user)
// shell, so redirections, pipes and && chains are escalated too - not only the
// first word, which is all "sudo cmd > file" would cover.
// setEnvironment keeps the task's environment as "env 'K=V' ..." (sorted)
func (e *CommandExecutor) setEnvironment(env map[string]string) {
	if len(env) == 0 {
		e.env = ""
		return
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := []string{"env"}
	for _, k := range keys {
		parts = append(parts, shellQuote(k+"="+env[k]))
	}
	e.env = strings.Join(parts, " ")
}

// withEnvironment runs command under the task's environment. The command
// goes through its own shell, so $VAR in it sees the new value.
func (e *CommandExecutor) withEnvironment(command string) string {
	if e.env == "" {
		return command
	}
	return e.env + " sh -c " + shellQuote(command)
}

func (e *CommandExecutor) wrapWithBecome(command string) string {
	if !e.become {
		return command
	}

	shell := "sh -c " + shellQuote(command)
	switch e.becomeMethod {
	case "su":
		// su -c already hands the whole line to the target user's shell
		if e.becomeUser == "root" {
			return "su -c " + shellQuote(command)
		}
		return fmt.Sprintf("su %s -c %s", e.becomeUser, shellQuote(command))
	case "doas":
		if e.becomeUser == "root" {
			return "doas " + shell
		}
		return fmt.Sprintf("doas -u %s %s", shellQuote(e.becomeUser), shell)
	default: // sudo
		if e.becomeUser == "root" {
			return "sudo -n " + shell
		}
		return fmt.Sprintf("sudo -n -u %s %s", shellQuote(e.becomeUser), shell)
	}
}

// withOutput appends the last lines of a failed command's output to its
// error: "exit status 1" alone says nothing about what went wrong
func withOutput(err error, output string) error {
	if err == nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return err
	}
	if len(lines) > 5 {
		lines = lines[len(lines)-5:]
	}
	return fmt.Errorf("%w: %s", err, strings.Join(lines, " | "))
}

// commandLine builds a shell command line: command is used as written (it may
// contain shell syntax), each separate argument is quoted as one word
func commandLine(command string, args []string) string {
	if len(args) == 0 {
		return command
	}
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = shellQuote(a)
	}
	return command + " " + strings.Join(quoted, " ")
}

// shellQuote quotes s as one POSIX shell word
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Execute runs a command on the appropriate host (local or remote)
func (e *CommandExecutor) Execute(command string, args ...string) (string, error) {
	out, err := e.execute(command, args...)
	return out, withOutput(err, out)
}

func (e *CommandExecutor) execute(command string, args ...string) (string, error) {
	fullCommand := e.withEnvironment(commandLine(command, args))

	// Wrap with become if enabled
	fullCommand = e.wrapWithBecome(fullCommand)

	if e.sshClient != nil {
		// Execute on remote host via SSH
		return e.sshClient.ExecuteCommand(fullCommand)
	} else if e.become || e.env != "" {
		// A single string with spaces goes through sh -c
		return e.executeLocal(fullCommand)
	} else {
		// Execute locally
		return e.executeLocal(command, args...)
	}
}

// ExecuteWithContext runs a command with context on the appropriate host
func (e *CommandExecutor) ExecuteWithContext(ctx context.Context, command string, args ...string) (string, error) {
	out, err := e.executeWithContext(ctx, command, args...)
	return out, withOutput(err, out)
}

func (e *CommandExecutor) executeWithContext(ctx context.Context, command string, args ...string) (string, error) {
	fullCommand := e.withEnvironment(commandLine(command, args))

	// Wrap with become if enabled
	fullCommand = e.wrapWithBecome(fullCommand)

	if e.sshClient != nil {
		// Execute on remote host via SSH with context support
		return e.executeSSHWithContext(ctx, fullCommand)
	} else if e.become || e.env != "" {
		// #nosec G204 -- privilege escalation wraps the module's own command
		cmd := exec.CommandContext(ctx, "sh", "-c", fullCommand)
		output, err := cmd.CombinedOutput()
		return string(output), err
	} else {
		// Execute locally with context
		cmd := exec.CommandContext(ctx, command, args...)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}
}

// ExecuteContext is an alias for ExecuteWithContext for compatibility with ModuleExecutor interface
func (e *CommandExecutor) ExecuteContext(ctx context.Context, command string, args ...string) (string, error) {
	return e.ExecuteWithContext(ctx, command, args...)
}

// ExecuteWithTimeout runs a command with timeout
func (e *CommandExecutor) ExecuteWithTimeout(command string, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return e.ExecuteWithContext(ctx, command, args...)
}

// executeSSHWithContext executes SSH command with context cancellation support
func (e *CommandExecutor) executeSSHWithContext(ctx context.Context, command string) (string, error) {
	// Get the underlying SSH client
	client := e.sshClient.GetClient()
	if client == nil {
		return "", fmt.Errorf("SSH client not available")
	}

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Структура для результата
	type result struct {
		output string
		err    error
	}

	resultChan := make(chan result, 1)

	// Запускаем команду в горутине
	go func() {
		defer close(resultChan)

		output, err := session.CombinedOutput(command)
		select {
		case resultChan <- result{string(output), err}:
		case <-ctx.Done():
			// Контекст отменен, пытаемся завершить сессию
			// Ignore signal error as session may already be closed
			_ = session.Signal(ssh.SIGTERM)
		}
	}()

	// Ждем результат или отмену контекста
	select {
	case res := <-resultChan:
		return res.output, res.err
	case <-ctx.Done():
		// Пытаемся корректно завершить сессию
		// Ignore signal error as we're already in error state
		_ = session.Signal(ssh.SIGTERM)
		return "", fmt.Errorf("command execution canceled: %w", ctx.Err())
	}
}

// executeLocal executes a command locally
func (e *CommandExecutor) executeLocal(command string, args ...string) (string, error) {
	// Separate arguments are passed as argv, exactly as the remote path quotes them
	if len(args) > 0 {
		// #nosec G204 -- modules run the commands they manage
		output, err := exec.Command(command, args...).CombinedOutput()
		return string(output), err
	}
	// A single command line may use shell syntax (pipes, redirects, ...)
	if strings.ContainsAny(command, "|&;<>()$`\\\"' \t\n*?[]{}") {
		// #nosec G204 - This is intentional: we need shell execution for complex commands with pipes, redirects, etc.
		cmd := exec.Command("sh", "-c", command)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}

	// Simple command without args - execute directly
	cmd := exec.Command(command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Close releases the connection back to the pool or closes it if not using pool
func (e *CommandExecutor) Close() error {
	if e.sshClient != nil && !e.poolReleased {
		e.poolReleased = true
		if e.usePool {
			// Return connection to pool for reuse
			pool := sshpkg.GetGlobalPool()
			pool.ReleaseConnection(e.host)
			return nil
		} else {
			// Close connection directly if not using pool
			return e.sshClient.Close()
		}
	}
	return nil
}

// IsRemote returns true if this executor is for a remote host
func (e *CommandExecutor) IsRemote() bool {
	return e.sshClient != nil
}
