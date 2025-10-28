package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// SysctlModule implements kernel parameter management
type SysctlModule struct {
	*BaseExecutorModule
}

// NewSysctlModule creates a new sysctl module
func NewSysctlModule() *SysctlModule {
	return &SysctlModule{
		BaseExecutorModule: NewBaseExecutorModule("sysctl"),
	}
}

// GetDescription returns the module description
func (m *SysctlModule) GetDescription() string {
	return "Manage kernel parameters via sysctl"
}

// Execute manages sysctl kernel parameters
func (m *SysctlModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  getStringArg(args, "name", "sysctl"),
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Validate required parameters
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return m.failResult(result, "parameter 'name' (sysctl key) is required")
	}

	value, ok := args["value"].(string)
	if !ok {
		// Try to convert from interface{} to string
		valueInterface, exists := args["value"]
		if !exists {
			return m.failResult(result, "parameter 'value' is required")
		}
		value = fmt.Sprintf("%v", valueInterface)
	}

	state := getStringArg(args, "state", "present")
	sysctlFile := getStringArg(args, "sysctl_file", "/etc/sysctl.d/99-onigirazu.conf")
	reload := getBoolArg(args, "reload", true)

	var execResult types.TaskResult = result

	execErr := m.WithExecutor(host, func(exec *executor.CommandExecutor) error {
		var err error
		switch state {
		case "present":
			execResult, err = m.handlePresent(ctx, exec, host, args, result, name, value, sysctlFile, reload)
		case "absent":
			execResult, err = m.handleAbsent(ctx, exec, host, args, result, name, sysctlFile, reload)
		default:
			execResult, err = m.failResult(result, fmt.Sprintf("invalid state: %s", state))
		}
		return err
	})

	if execErr != nil {
		return result, execErr
	}

	return execResult, nil
}

// handlePresent ensures a kernel parameter is set
func (m *SysctlModule) handlePresent(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, name, value, sysctlFile string, reload bool) (types.TaskResult, error) {
	// Get current value
	currentValue, err := m.getCurrentValue(exec, name)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to get current sysctl value: %v", err))
	}

	result.Output["sysctl_key"] = name
	result.Output["current_value"] = currentValue
	result.Output["desired_value"] = value

	// Check if value is already set correctly
	if currentValue == value {
		result.Changed = false
		result.Output["msg"] = fmt.Sprintf("Kernel parameter %s is already set to %s", name, value)
		result.Duration = time.Since(result.Timestamp)
		return result, nil
	}

	// Set the value immediately with sysctl
	cmd := fmt.Sprintf("sysctl -w '%s=%s' 2>&1", name, strings.TrimSpace(value))
	output, err := exec.Execute("sh", "-c", cmd)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to set kernel parameter: %v", err))
	}

	result.Output["sysctl_output"] = output
	result.Changed = true

	// Persist to sysctl configuration file if requested
	if getBoolArg(args, "persist", true) {
		// Read current file content
		catCmd := fmt.Sprintf("cat '%s' 2>/dev/null || echo ''", sysctlFile)
		fileContent, err := exec.Execute("sh", "-c", catCmd)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to read sysctl file: %v", err))
		}

		// Check if parameter is already in file
		lines := strings.Split(fileContent, "\n")
		paramPattern := name + "="
		found := false
		newLines := []string{}

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, paramPattern) {
				// Update existing parameter
				newLines = append(newLines, fmt.Sprintf("%s=%s", name, value))
				found = true
			} else if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				newLines = append(newLines, line)
			} else if trimmed != "" {
				newLines = append(newLines, line)
			}
		}

		if !found {
			// Add new parameter
			newLines = append(newLines, fmt.Sprintf("%s=%s", name, value))
		}

		// Write updated content back
		newContent := strings.Join(newLines, "\n")
		writeCmd := fmt.Sprintf("echo '%s' | tee '%s' > /dev/null", strings.ReplaceAll(newContent, "'", "'\\''"), sysctlFile)
		_, err = exec.Execute("sh", "-c", writeCmd)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to persist sysctl parameter: %v", err))
		}

		result.Output["persisted_to_file"] = sysctlFile
	}

	// Reload sysctl settings if requested
	if reload {
		_, err := exec.Execute("sysctl", "-p")
		if err != nil {
			// Don't fail if reload fails - the setting is still in effect
			result.Output["reload_error"] = err.Error()
		}
	}

	result.Output["msg"] = fmt.Sprintf("Kernel parameter %s set to %s", name, value)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleAbsent removes a kernel parameter
func (m *SysctlModule) handleAbsent(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, name, sysctlFile string, reload bool) (types.TaskResult, error) {
	// Get current value
	currentValue, err := m.getCurrentValue(exec, name)
	if err != nil || currentValue == "" {
		result.Changed = false
		result.Output["msg"] = fmt.Sprintf("Kernel parameter %s is not set", name)
		result.Duration = time.Since(result.Timestamp)
		return result, nil
	}

	result.Output["sysctl_key"] = name
	result.Output["removed_value"] = currentValue

	// Remove from sysctl file if requested
	if getBoolArg(args, "persist", true) {
		// Read current file content
		catCmd := fmt.Sprintf("cat '%s' 2>/dev/null || echo ''", sysctlFile)
		fileContent, err := exec.Execute("sh", "-c", catCmd)
		if err == nil && fileContent != "" {
			// Remove the parameter line
			lines := strings.Split(fileContent, "\n")
			paramPattern := name + "="
			newLines := []string{}

			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if !strings.HasPrefix(trimmed, paramPattern) && trimmed != "" {
					newLines = append(newLines, line)
				}
			}

			newContent := strings.Join(newLines, "\n")
			if newContent != "" {
				writeCmd := fmt.Sprintf("echo '%s' | tee '%s' > /dev/null", strings.ReplaceAll(newContent, "'", "'\\''"), sysctlFile)
				exec.Execute("sh", "-c", writeCmd)
			} else {
				// Remove empty file
				exec.Execute("rm", "-f", sysctlFile)
			}

			result.Output["removed_from_file"] = sysctlFile
		}
	}

	// Reload sysctl settings if requested
	if reload {
		exec.Execute("sysctl", "-p")
	}

	result.Changed = true
	result.Output["msg"] = fmt.Sprintf("Kernel parameter %s removed", name)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// getCurrentValue gets the current value of a sysctl parameter
func (m *SysctlModule) getCurrentValue(exec *executor.CommandExecutor, name string) (string, error) {
	output, err := exec.Execute("sysctl", "-n", name)
	if err != nil {
		// Parameter doesn't exist or error
		return "", nil
	}
	return strings.TrimSpace(output), nil
}

// Validate validates argument correctness
func (m *SysctlModule) Validate(args map[string]interface{}) error {
	if _, exists := args["name"]; !exists {
		return fmt.Errorf("'name' parameter is required")
	}
	if _, exists := args["value"]; !exists {
		return fmt.Errorf("'value' parameter is required")
	}
	return nil
}
