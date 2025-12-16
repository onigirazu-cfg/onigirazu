package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
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
		TaskName:  getStringArg(args, "name", "reboot"),
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Get parameters
	delaySeconds := getIntArg(args, "pre_reboot_delay", 0)
	msgText := getStringArg(args, "msg", "System will reboot in a few seconds")
	testBoot := getBoolArg(args, "test_boot", false)
	rebootCommand := getStringArg(args, "reboot_command", "")

	result.Output["host"] = host.Name

	var execResult types.TaskResult = result

	execErr := m.WithExecutor(host, func(exec *executor.CommandExecutor) error {
		var err error

		// Notify users about reboot if delay is set
		if delaySeconds > 0 {
			notifyCmd := fmt.Sprintf("wall '%s' || echo 'Warning message could not be broadcast'", msgText)
			_, _ = exec.Execute("sh", "-c", notifyCmd)
			result.Output["msg"] = msgText
			result.Output["pre_reboot_delay"] = delaySeconds

			// Wait before reboot
			time.Sleep(time.Duration(delaySeconds) * time.Second)
		}

		execResult, err = m.performReboot(exec, host, args, result, rebootCommand, testBoot)
		return err
	})

	if execErr != nil {
		return result, execErr
	}

	return execResult, nil
}

// performReboot performs the actual reboot
func (m *RebootModule) performReboot(exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, rebootCommand string, testBoot bool) (types.TaskResult, error) {
	// Test boot if requested (check if system can boot without actually rebooting)
	if testBoot {
		// This would be implementation-specific - for now, just check systemctl status
		output, err := exec.Execute("systemctl", "status")
		if err != nil {
			return m.failResult(result, fmt.Sprintf("test boot check failed: %v", err))
		}
		result.Output["test_boot_output"] = output
		result.Output["msg"] = "System passed boot test - reboot canceled (test_boot=true)"
		result.Duration = time.Since(result.Timestamp)
		return result, nil
	}

	// Use custom reboot command if provided
	if rebootCommand != "" {
		_, err := exec.Execute("sh", "-c", rebootCommand)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("reboot command failed: %v", err))
		}
		result.Changed = true
		result.Output["msg"] = fmt.Sprintf("System reboot initiated with custom command: %s", rebootCommand)
		result.Duration = time.Since(result.Timestamp)
		return result, nil
	}

	// Use standard reboot command
	// We use 'shutdown -r' instead of 'reboot' because it's more graceful
	// The '+1' means reboot in 1 minute, giving time for playbook to finish
	_, err := exec.Execute("shutdown", "-r", "+1", "Rebooting via Onigirazu")
	if err != nil {
		return m.failResult(result, fmt.Sprintf("reboot failed: %v", err))
	}

	result.Changed = true
	result.Output["msg"] = "System reboot scheduled to start in 1 minute"
	result.Output["reboot_initiated"] = true
	result.Duration = time.Since(result.Timestamp)
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
