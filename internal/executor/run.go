package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"

	"golang.org/x/crypto/ssh"
)

// RunResult is what a command that ran left behind
type RunResult struct {
	Stdout string
	Stderr string
	RC     int
}

// Run runs a shell command line on the host (with the task environment and
// become) and keeps stdout, stderr and the exit code apart. A non-zero exit
// is not an error; the error is for a command that could not be run.
func (e *CommandExecutor) Run(ctx context.Context, commandLine string) (RunResult, error) {
	full := e.wrapWithBecome(e.withEnvironment(commandLine))
	var stdout, stderr bytes.Buffer

	if e.sshClient == nil {
		// #nosec G204 -- modules run the commands they manage
		cmd := exec.CommandContext(ctx, "sh", "-c", full)
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && ctx.Err() == nil {
			return RunResult{stdout.String(), stderr.String(), exitErr.ExitCode()}, nil
		}
		return RunResult{stdout.String(), stderr.String(), 0}, err
	}

	client := e.sshClient.GetClient()
	if client == nil {
		return RunResult{}, fmt.Errorf("SSH client not available")
	}
	session, err := client.NewSession()
	if err != nil {
		return RunResult{}, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()
	session.Stdout, session.Stderr = &stdout, &stderr

	done := make(chan error, 1)
	go func() { done <- session.Run(full) }()
	select {
	case err = <-done:
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		// the session may still be writing to the buffers: leave them
		return RunResult{}, fmt.Errorf("command execution canceled: %w", ctx.Err())
	}
	var exitErr *ssh.ExitError
	if errors.As(err, &exitErr) {
		return RunResult{stdout.String(), stderr.String(), exitErr.ExitStatus()}, nil
	}
	return RunResult{stdout.String(), stderr.String(), 0}, err
}
