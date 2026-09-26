package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// SystemdModule implements systemd management
type SystemdModule struct {
	*BaseExecutorModule
}

// NewSystemdModule creates a new systemd module
func NewSystemdModule() *SystemdModule {
	return &SystemdModule{
		BaseExecutorModule: NewBaseExecutorModule("systemd"),
	}
}

// GetDescription returns the module description
func (m *SystemdModule) GetDescription() string {
	return "Manage systemd services, units, and timers"
}

// Execute manages systemd operations
func (m *SystemdModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "systemd",
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Use WithExecutor to get fresh executor for this host
	var execResult types.TaskResult = result

	execErr := m.WithExecutor(host, func(exec *executor.CommandExecutor) error {
		// Get operation type
		operation := getStringArg(args, "operation", "service")

		var err error
		switch operation {
		case "service":
			execResult, err = m.handleService(ctx, exec, host, args, result)
		case "unit":
			execResult, err = m.handleUnit(ctx, exec, host, args, result)
		case "timer":
			execResult, err = m.handleTimer(ctx, exec, host, args, result)
		case "daemon-reload":
			execResult, err = m.handleDaemonReload(ctx, exec, host, args, result)
		case "status":
			execResult, err = m.handleStatus(ctx, exec, host, args, result)
		default:
			execResult, err = m.failResult(result, fmt.Sprintf("unknown operation: %s", operation))
		}
		return err
	})

	if execErr != nil {
		return result, execErr
	}

	return execResult, nil
}

// handleService manages systemd services
func (m *SystemdModule) handleService(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	name, ok := args["name"].(string)
	if !ok {
		return m.failResult(result, "name parameter is required")
	}

	state := getStringArg(args, "state", "")
	enabled := args["enabled"]
	masked := args["masked"]

	changed := false

	// Handle state changes
	if state != "" {
		currentState, err := m.getServiceState(exec, name)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to get service state: %v", err))
		}

		switch state {
		case "started":
			if currentState != "active" {
				if _, err := m.systemctl(exec, args, "start", name); err != nil {
					return m.failResult(result, fmt.Sprintf("failed to start service: %v", err))
				}
				changed = true
				result.Output["action"] = "started"
			}
		case "stopped":
			if currentState == "active" {
				if _, err := m.systemctl(exec, args, "stop", name); err != nil {
					return m.failResult(result, fmt.Sprintf("failed to stop service: %v", err))
				}
				changed = true
				result.Output["action"] = "stopped"
			}
		case "restarted":
			if _, err := m.systemctl(exec, args, "restart", name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to restart service: %v", err))
			}
			changed = true
			result.Output["action"] = "restarted"
		case "reloaded":
			if _, err := m.systemctl(exec, args, "reload", name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to reload service: %v", err))
			}
			changed = true
			result.Output["action"] = "reloaded"
		}
	}

	// Handle enabled state
	if enabled != nil {
		enabledBool := getBoolArg(args, "enabled", false)
		isEnabled, err := m.isServiceEnabled(exec, name)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to check enabled state: %v", err))
		}

		if enabledBool && !isEnabled {
			if _, err := m.systemctl(exec, args, "enable", name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to enable service: %v", err))
			}
			changed = true
			result.Output["enabled"] = true
		} else if !enabledBool && isEnabled {
			if _, err := m.systemctl(exec, args, "disable", name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to disable service: %v", err))
			}
			changed = true
			result.Output["enabled"] = false
		}
	}

	// Handle masked state
	if masked != nil {
		maskedBool := getBoolArg(args, "masked", false)
		isMasked, err := m.isServiceMasked(exec, name)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to check masked state: %v", err))
		}

		if maskedBool && !isMasked {
			if _, err := m.systemctl(exec, args, "mask", name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to mask service: %v", err))
			}
			changed = true
			result.Output["masked"] = true
		} else if !maskedBool && isMasked {
			if _, err := m.systemctl(exec, args, "unmask", name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to unmask service: %v", err))
			}
			changed = true
			result.Output["masked"] = false
		}
	}

	// Get final status
	status, err := m.getServiceStatus(exec, name)
	if err == nil {
		result.Output["status"] = status
	}

	result.Changed = changed
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleUnit manages systemd unit files
func (m *SystemdModule) handleUnit(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	name, ok := args["name"].(string)
	if !ok {
		return m.failResult(result, "name parameter is required")
	}
	if getStringArg(args, "path", "") == "" && (strings.ContainsAny(name, "/\\") || name == "." || name == "..") {
		return m.failResult(result, fmt.Sprintf("invalid unit name %q", name))
	}

	content := getStringArg(args, "content", "")
	path := getStringArg(args, "path", "")
	state := getStringArg(args, "state", "present")

	changed := false

	if state == "present" {
		if content == "" && path == "" {
			return m.failResult(result, "either content or path parameter is required")
		}

		// Determine unit file path
		unitPath := path
		if unitPath == "" {
			unitPath = fmt.Sprintf("/etc/systemd/system/%s", name)
		}

		if content != "" {
			current, exists, err := readHostFile(ctx, host, args, unitPath)
			if err != nil {
				return m.failResult(result, err.Error())
			}
			if !exists || string(current) != content {
				if !inCheckMode(args) {
					if err := writeHostFile(ctx, host, args, unitPath, []byte(content), 0644); err != nil {
						return m.failResult(result, fmt.Sprintf("failed to write unit file: %v", err))
					}
				}
				changed = true
				result.Output["action"] = "unit_written"
			}
		} else if _, err := exec.Execute("test", "-f", unitPath); err != nil {
			return m.failResult(result, fmt.Sprintf("unit file %s does not exist", unitPath))
		}

		// Reload systemd only when a unit file changed
		if changed {
			if _, err := m.systemctl(exec, args, "daemon-reload"); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to reload systemd: %v", err))
			}
		}
	} else if state == "absent" {
		unitPath := path
		if unitPath == "" {
			unitPath = fmt.Sprintf("/etc/systemd/system/%s", name)
		}

		// Check if unit file exists
		_, err := exec.Execute("test", "-f", unitPath)
		if err == nil {
			// Stop and disable service first (ignore errors as service might not be running/enabled)
			_, _ = m.systemctl(exec, args, "stop", name)
			_, _ = m.systemctl(exec, args, "disable", name)

			// Remove unit file
			if _, err := m.run(exec, args, "rm", "-f", unitPath); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to remove unit file: %v", err))
			}

			// Reload systemd
			if _, err := m.systemctl(exec, args, "daemon-reload"); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to reload systemd: %v", err))
			}

			changed = true
			result.Output["action"] = "unit_removed"
		}
	}

	result.Changed = changed
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleTimer manages systemd timers
func (m *SystemdModule) handleTimer(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	name, ok := args["name"].(string)
	if !ok {
		return m.failResult(result, "name parameter is required")
	}

	state := getStringArg(args, "state", "started")
	enabled := args["enabled"]

	changed := false

	// Ensure .timer suffix
	if !strings.HasSuffix(name, ".timer") {
		name = name + ".timer"
	}

	// Handle state changes
	if state != "" {
		currentState, err := m.getServiceState(exec, name)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to get timer state: %v", err))
		}

		switch state {
		case "started":
			if currentState != "active" {
				if _, err := m.systemctl(exec, args, "start", name); err != nil {
					return m.failResult(result, fmt.Sprintf("failed to start timer: %v", err))
				}
				changed = true
				result.Output["action"] = "started"
			}
		case "stopped":
			if currentState == "active" {
				if _, err := m.systemctl(exec, args, "stop", name); err != nil {
					return m.failResult(result, fmt.Sprintf("failed to stop timer: %v", err))
				}
				changed = true
				result.Output["action"] = "stopped"
			}
		}
	}

	// Handle enabled state
	if enabled != nil {
		enabledBool := getBoolArg(args, "enabled", false)
		isEnabled, err := m.isServiceEnabled(exec, name)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to check enabled state: %v", err))
		}

		if enabledBool && !isEnabled {
			if _, err := m.systemctl(exec, args, "enable", name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to enable timer: %v", err))
			}
			changed = true
			result.Output["enabled"] = true
		} else if !enabledBool && isEnabled {
			if _, err := m.systemctl(exec, args, "disable", name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to disable timer: %v", err))
			}
			changed = true
			result.Output["enabled"] = false
		}
	}

	// Get timer status
	output, err := exec.Execute("systemctl", "list-timers", "--all", name)
	if err == nil {
		result.Output["timer_info"] = strings.TrimSpace(output)
	}

	result.Changed = changed
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleDaemonReload reloads systemd daemon
func (m *SystemdModule) handleDaemonReload(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	if _, err := m.systemctl(exec, args, "daemon-reload"); err != nil {
		return m.failResult(result, fmt.Sprintf("failed to reload systemd: %v", err))
	}

	result.Changed = true
	result.Output["action"] = "daemon_reloaded"
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleStatus gets systemd status
func (m *SystemdModule) handleStatus(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	name, ok := args["name"].(string)
	if !ok {
		return m.failResult(result, "name parameter is required")
	}

	status, err := m.getServiceStatus(exec, name)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to get status: %v", err))
	}

	result.Output["status"] = status
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// Helper methods
func (m *SystemdModule) getServiceState(exec *executor.CommandExecutor, name string) (string, error) {
	output, err := exec.Execute("systemctl", "is-active", name)
	if err != nil {
		return "inactive", nil
	}
	return strings.TrimSpace(output), nil
}

func (m *SystemdModule) isServiceEnabled(exec *executor.CommandExecutor, name string) (bool, error) {
	output, err := exec.Execute("systemctl", "is-enabled", name)
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(output) == "enabled", nil
}

func (m *SystemdModule) isServiceMasked(exec *executor.CommandExecutor, name string) (bool, error) {
	output, err := exec.Execute("systemctl", "is-enabled", name)
	if err != nil {
		return strings.TrimSpace(output) == "masked", nil
	}
	return false, nil
}

func (m *SystemdModule) getServiceStatus(exec *executor.CommandExecutor, name string) (map[string]string, error) {
	status := make(map[string]string)

	output, err := exec.Execute("systemctl", "show", name, "--property=LoadState,ActiveState,SubState,MainPID,Description,UnitFileState")
	if err != nil {
		return status, err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			status[parts[0]] = parts[1]
		}
	}

	return status, nil
}

// Validate validates systemd module arguments
func (m *SystemdModule) Validate(args map[string]interface{}) error {
	operation := getStringArg(args, "operation", "service")

	switch operation {
	case "service", "timer", "status":
		if _, exists := args["name"]; !exists {
			return fmt.Errorf("name parameter is required")
		}
	case "unit":
		if _, exists := args["name"]; !exists {
			return fmt.Errorf("name parameter is required")
		}
		state := getStringArg(args, "state", "present")
		if state == "present" {
			if _, hasContent := args["content"]; !hasContent {
				if _, hasPath := args["path"]; !hasPath {
					return fmt.Errorf("either content or path parameter is required")
				}
			}
		}
	case "daemon-reload":
		// No additional validation needed
	default:
		return fmt.Errorf("invalid operation: %s", operation)
	}

	return nil
}

// systemctl runs a systemctl command that changes the host; in check mode it
// is not run, and the caller reports the change it would have made
func (m *SystemdModule) systemctl(exec *executor.CommandExecutor, args map[string]interface{}, cmdArgs ...string) (string, error) {
	return m.run(exec, args, "systemctl", cmdArgs...)
}

// run executes a changing command unless the task is in check mode
func (m *SystemdModule) run(exec *executor.CommandExecutor, args map[string]interface{}, command string, cmdArgs ...string) (string, error) {
	if inCheckMode(args) {
		return "", nil
	}
	return exec.Execute(command, cmdArgs...)
}
