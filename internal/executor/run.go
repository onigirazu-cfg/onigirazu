package executor

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
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

	out, errOut, rc, err := e.exec(ctx, full, false)
	if err != nil {
		return RunResult{}, err
	}
	return RunResult{string(out), string(errOut), rc}, nil
}
