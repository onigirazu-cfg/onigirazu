package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// RebootModule implements system reboot functionality
type RebootModule struct {
	*BaseExecutorModule
}

// NewRebootModule creates a new reboot module
func NewRebootModule() *RebootModule {
	return &RebootModule{
		BaseExecutorModule: NewBaseExecutorModule("reboot"),
	}
}

// GetDescription returns the module description
func (m *RebootModule) GetDescription() string {
	return "Reboot the system, with optional delay and pre-reboot checks"
}

// Execute manages system reboot
func (m *RebootModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  taskName(args),
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success = false
		result.Error = msg
		result.Duration = time.Since(startTime)
		return result, nil
	}
	sleep := func(d time.Duration) bool {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(d):
			return true
		}
	}

	if sshpkg.IsLocal(host) {
		return fail("refusing to reboot the control machine (host is local)")
	}

	if getBoolArg(args, "test_boot", false) {
		output, err := runOnHost(ctx, host, args, "systemctl", "is-system-running")
		result.Output["test_boot_output"] = strings.TrimSpace(output)
		if err != nil && !strings.Contains(output, "degraded") {
			return fail(fmt.Sprintf("test boot check failed: %v", err))
		}
		result.Output["msg"] = "System passed boot test - reboot canceled (test_boot=true)"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	bootID := func() (string, error) {
		out, err := runOnHost(ctx, host, args, "cat", "/proc/sys/kernel/random/boot_id")
		return strings.TrimSpace(out), err
	}
	before, err := bootID()
	if err != nil || before == "" {
		return fail(fmt.Sprintf("failed to read boot id: %v", err))
	}

	if delay := getIntArg(args, "pre_reboot_delay", 0); delay > 0 {
		msg := getStringArg(args, "msg", "System will reboot in a few seconds")
		_, _ = runShellOnHost(ctx, host, args, "wall "+shellQuote(msg)+" 2>/dev/null || true")
		if !sleep(time.Duration(delay) * time.Second) {
			return fail("canceled")
		}
	}

	// Start the reboot a moment later, so this SSH command can return first
	command := getStringArg(args, "reboot_command", "")
	if command == "" {
		command = "systemd-run --on-active=2 /bin/systemctl reboot >/dev/null 2>&1 || " +
			"(nohup sh -c 'sleep 2; reboot' >/dev/null 2>&1 </dev/null &)"
	}
	if _, err := runShellOnHost(ctx, host, args, command); err != nil {
		return fail(fmt.Sprintf("reboot command failed: %v", err))
	}
	result.Changed = true

	// Wait until the host answers with a new boot id
	pool := sshpkg.GetGlobalPool()
	timeout := time.Duration(getIntArg(args, "reboot_timeout", 600)) * time.Second
	deadline := time.Now().Add(timeout)
	for {
		if !sleep(5 * time.Second) {
			return fail("canceled while waiting for the host")
		}
		_ = pool.CloseConnection(host) // connections from before the reboot are dead
		if after, err := bootID(); err == nil && after != "" && after != before {
			break
		}
		if time.Now().After(deadline) {
			return fail(fmt.Sprintf("host did not come back within %s", timeout))
		}
	}
	if delay := getIntArg(args, "post_reboot_delay", 0); delay > 0 && !sleep(time.Duration(delay)*time.Second) {
		return fail("canceled")
	}

	result.Output["msg"] = "System rebooted"
	result.Output["elapsed"] = time.Since(startTime).Seconds()
	result.Duration = time.Since(startTime)
	return result, nil
}

// Validate validates argument correctness
func (m *RebootModule) Validate(args map[string]interface{}) error {
	// Check for test boot and custom command conflict
	testBoot := getBoolArg(args, "test_boot", false)
	rebootCommand := getStringArg(args, "reboot_command", "")

	if testBoot && rebootCommand != "" {
		return fmt.Errorf("cannot specify both 'test_boot' and 'reboot_command'")
	}

	// Validate pre_reboot_delay if provided
	if delayVal, exists := args["pre_reboot_delay"]; exists {
		switch v := delayVal.(type) {
		case float64:
			if v < 0 {
				return fmt.Errorf("pre_reboot_delay must be >= 0, got %v", v)
			}
		case int:
			if v < 0 {
				return fmt.Errorf("pre_reboot_delay must be >= 0, got %v", v)
			}
		case string:
			// Try to parse
			var delay int
			_, err := fmt.Sscanf(v, "%d", &delay)
			if err != nil || delay < 0 {
				return fmt.Errorf("invalid pre_reboot_delay value: %s", v)
			}
		}
	}

	return nil
}
