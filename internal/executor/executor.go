package executor

import (
	"context"
	"fmt"
	"os/exec"
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

	return executor, nil
}

// Execute runs a command on the appropriate host (local or remote)
func (e *CommandExecutor) Execute(command string, args ...string) (string, error) {
	fullCommand := command
	if len(args) > 0 {
		fullCommand = command + " " + strings.Join(args, " ")
	}

	if e.sshClient != nil {
		// Execute on remote host via SSH
		return e.sshClient.ExecuteCommand(fullCommand)
	} else {
		// Execute locally
		return e.executeLocal(command, args...)
	}
}

// ExecuteWithContext runs a command with context on the appropriate host
func (e *CommandExecutor) ExecuteWithContext(ctx context.Context, command string, args ...string) (string, error) {
	fullCommand := command
	if len(args) > 0 {
		fullCommand = command + " " + strings.Join(args, " ")
	}

	if e.sshClient != nil {
		// Execute on remote host via SSH with context support
		return e.executeSSHWithContext(ctx, fullCommand)
	} else {
		// Execute locally with context
		cmd := exec.CommandContext(ctx, command, args...)
		output, err := cmd.CombinedOutput()
		return string(output), err
	}
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
			session.Signal(ssh.SIGTERM)
		}
	}()

	// Ждем результат или отмену контекста
	select {
	case res := <-resultChan:
		return res.output, res.err
	case <-ctx.Done():
		// Пытаемся корректно завершить сессию
		session.Signal(ssh.SIGTERM)
		return "", fmt.Errorf("command execution cancelled: %w", ctx.Err())
	}
}

// executeLocal executes a command locally
func (e *CommandExecutor) executeLocal(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
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
