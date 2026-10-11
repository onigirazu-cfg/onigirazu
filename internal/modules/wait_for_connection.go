package modules

import (
	"context"
	"fmt"
	"time"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// WaitForConnectionModule waits until the host answers over its connection
// (Ansible's wait_for_connection): after a reboot, a fresh VM, a network
// change. connect_timeout per attempt, sleep between attempts, timeout in
// all, delay before the first
type WaitForConnectionModule struct {
	*BaseModule
}

// NewWaitForConnectionModule creates the module
func NewWaitForConnectionModule() *WaitForConnectionModule {
	return &WaitForConnectionModule{BaseModule: NewBaseModule("wait_for_connection")}
}

func (m *WaitForConnectionModule) GetDescription() string {
	return "Wait until the host answers over its connection"
}

func (m *WaitForConnectionModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	timeout := time.Duration(getIntArg(args, "timeout", 600)) * time.Second
	delay := time.Duration(getIntArg(args, "delay", 0)) * time.Second
	pause := time.Duration(getIntArg(args, "sleep", 1)) * time.Second
	connectTimeout := time.Duration(getIntArg(args, "connect_timeout", 5)) * time.Second
	if delay > 0 {
		select {
		case <-ctx.Done():
			result.Success, result.Error = false, "canceled"
			return result, nil
		case <-time.After(delay):
		}
	}
	deadline := start.Add(timeout)
	var lastErr error
	for attempt := 1; ; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, connectTimeout+10*time.Second)
		out, err := runOnHost(attemptCtx, host, args, "echo", "ready")
		cancel()
		if err == nil && out != "" {
			result.Output["elapsed"] = int(time.Since(start).Seconds())
			result.Output["attempts"] = attempt
			result.Output["msg"] = fmt.Sprintf("%s answers after %s", host.Name, time.Since(start).Round(time.Second))
			result.Duration = time.Since(start)
			return result, nil
		}
		lastErr = err
		// a failed attempt leaves a dead pooled connection behind: drop it
		closePooledConnection(host)
		if time.Now().Add(pause).After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			result.Success, result.Error = false, "canceled"
			return result, nil
		case <-time.After(pause):
		}
	}
	result.Success = false
	result.Error = fmt.Sprintf("timeout waiting for %s after %s: %v", host.Name, timeout, lastErr)
	result.Output["elapsed"] = int(time.Since(start).Seconds())
	result.Duration = time.Since(start)
	return result, nil
}

// closePooledConnection drops the host's pooled SSH connection so the next
// attempt dials afresh
func closePooledConnection(host types.Host) {
	_ = sshpkg.GetGlobalPool().CloseConnection(host)
}
