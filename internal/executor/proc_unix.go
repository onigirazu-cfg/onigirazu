//go:build !windows

package executor

import (
	"os/exec"
	"syscall"
	"time"
)

// stopWithContext makes a canceled context stop the whole command: a shell
// and what it started share a process group, killed together; a child that
// escapes the group cannot hold the output pipes open past waitDelay
func stopWithContext(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = waitDelay
}

const waitDelay = 5 * time.Second
