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
	BaseModule
}

// NewSystemdModule creates a new systemd module
func NewSystemdModule() *SystemdModule {
	return &SystemdModule{
		BaseModule: BaseModule{
			name:        "systemd",
			description: "Manage systemd services, units, and timers",
		},
	}
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

	// Create a fresh executor for this execution
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to create executor: %v", err))
	}
	defer exec.Close()

	// Get operation type
	operation := getStringArg(args, "operation", "service")

	switch operation {
	case "service":
		return m.handleService(ctx, exec, host, args, result)
	case "unit":
		return m.handleUnit(ctx, exec, host, args, result)
	case "timer":
		return m.handleTimer(ctx, exec, host, args, result)
	case "daemon-reload":
		return m.handleDaemonReload(ctx, exec, host, args, result)
	case "status":
		return m.handleStatus(ctx, exec, host, args, result)
	default:
		return m.failResult(result, fmt.Sprintf("unknown operation: %s", operation))
	}
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
				if _, err := exec.Execute("systemctl", "start", name); err != nil {
					return m.failResult(result, fmt.Sprintf("failed to start service: %v", err))
				}
				changed = true
				result.Output["action"] = "started"
			}
		case "stopped":
			if currentState == "active" {
				if _, err := exec.Execute("systemctl", "stop", name); err != nil {
					return m.failResult(result, fmt.Sprintf("failed to stop service: %v", err))
				}
				changed = true
				result.Output["action"] = "stopped"
			}
		case "restarted":
			if _, err := exec.Execute("systemctl", "restart", name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to restart service: %v", err))
			}
			changed = true
			result.Output["action"] = "restarted"
		case "reloaded":
			if _, err := exec.Execute("systemctl", "reload", name); err != nil {
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
			if _, err := exec.Execute("systemctl", "enable", name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to enable service: %v", err))
			}
			changed = true
			result.Output["enabled"] = true
		} else if !enabledBool && isEnabled {
			if _, err := exec.Execute("systemctl", "disable", name); err != nil {
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
			if _, err := exec.Execute("systemctl", "mask", name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to mask service: %v", err))
			}
			changed = true
			result.Output["masked"] = true
		} else if !maskedBool && isMasked {
			if _, err := exec.Execute("systemctl", "unmask", name); err != nil {
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

		// Check if unit file exists
		_, err := exec.Execute("test", "-f", unitPath)
		unitExists := err == nil

		if content != "" {
			// Write unit file content
			tmpFile := fmt.Sprintf("/tmp/%s", name)
			if _, err := exec.Execute("sh", "-c", fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", tmpFile, content)); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to write unit file: %v", err))
			}

			// Move to systemd directory
			if _, err := exec.Execute("mv", tmpFile, unitPath); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to move unit file: %v", err))
			}

			changed = true
			result.Output["action"] = "unit_created"
		}

		// Reload systemd if unit was created or modified
		if changed || !unitExists {
			if _, err := exec.Execute("systemctl", "daemon-reload"); err != nil {
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
			_, _ = exec.Execute("systemctl", "stop", name)
			_, _ = exec.Execute("systemctl", "disable", name)

			// Remove unit file
			if _, err := exec.Execute("rm", "-f", unitPath); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to remove unit file: %v", err))
			}

			// Reload systemd
			if _, err := exec.Execute("systemctl", "daemon-reload"); err != nil {
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
				if _, err := exec.Execute("systemctl", "start", name); err != nil {
					return m.failResult(result, fmt.Sprintf("failed to start timer: %v", err))
				}
				changed = true
				result.Output["action"] = "started"
			}
		case "stopped":
			if currentState == "active" {
				if _, err := exec.Execute("systemctl", "stop", name); err != nil {
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
			if _, err := exec.Execute("systemctl", "enable", name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to enable timer: %v", err))
			}
			changed = true
			result.Output["enabled"] = true
		} else if !enabledBool && isEnabled {
			if _, err := exec.Execute("systemctl", "disable", name); err != nil {
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
	if _, err := exec.Execute("systemctl", "daemon-reload"); err != nil {
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
