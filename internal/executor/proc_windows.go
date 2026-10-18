//go:build windows

package executor

import (
	"os/exec"
	"time"
)

// stopWithContext: a child that outlives the killed command cannot hold
// the output pipes open past waitDelay
func stopWithContext(cmd *exec.Cmd) {
	cmd.WaitDelay = waitDelay
}

const waitDelay = 5 * time.Second
