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
		TaskName:  taskName(args),
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

// handlePresent sets a kernel parameter now and, with persist (default), in
// sysctl_file. Each part changes only when it differs.
func (m *SysctlModule) handlePresent(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, name, value, sysctlFile string, reload bool) (types.TaskResult, error) {
	value = normalizeSysctl(value)
	currentValue, err := m.getCurrentValue(exec, name)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to get current sysctl value: %v", err))
	}
	result.Output["sysctl_key"] = name
	result.Output["current_value"] = currentValue
	result.Output["desired_value"] = value

	persist := getBoolArg(args, "persist", true)
	var newFile []byte
	if persist {
		data, _, err := readHostFile(ctx, host, args, sysctlFile)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to read %s: %v", sysctlFile, err))
		}
		if updated, changed := setSysctlLine(string(data), name, value); changed {
			newFile = []byte(updated)
		}
	}
	runtimeWrong := normalizeSysctl(currentValue) != value

	if !runtimeWrong && newFile == nil {
		result.Output["msg"] = fmt.Sprintf("Kernel parameter %s is already set to %s", name, value)
		result.Duration = time.Since(result.Timestamp)
		return result, nil
	}
	result.Changed = true
	if inCheckMode(args) {
		result.Output["msg"] = fmt.Sprintf("Kernel parameter %s would be set to %s", name, value)
		result.Duration = time.Since(result.Timestamp)
		return result, nil
	}

	if runtimeWrong {
		output, err := runOnHost(ctx, host, args, "sysctl", "-w", name+"="+value)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to set kernel parameter: %v", err))
		}
		result.Output["sysctl_output"] = strings.TrimSpace(output)
	}
	if newFile != nil {
		if err := writeHostFile(ctx, host, args, sysctlFile, newFile, 0o644); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to persist sysctl parameter: %v", err))
		}
		result.Output["persisted_to_file"] = sysctlFile
		if reload {
			if _, err := runOnHost(ctx, host, args, "sysctl", "-p", sysctlFile); err != nil {
				result.Output["reload_error"] = err.Error()
			}
		}
	}

	result.Output["msg"] = fmt.Sprintf("Kernel parameter %s set to %s", name, value)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleAbsent removes a parameter from sysctl_file. The kernel keeps its
// current value until reboot; there is nothing to "unset" at runtime.
func (m *SysctlModule) handleAbsent(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, name, sysctlFile string, reload bool) (types.TaskResult, error) {
	result.Output["sysctl_key"] = name
	data, exists, err := readHostFile(ctx, host, args, sysctlFile)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to read %s: %v", sysctlFile, err))
	}
	updated, changed := removeSysctlLine(string(data), name)
	if !exists || !changed {
		result.Output["msg"] = fmt.Sprintf("Kernel parameter %s is not in %s", name, sysctlFile)
		result.Duration = time.Since(result.Timestamp)
		return result, nil
	}
	result.Changed = true
	if inCheckMode(args) {
		result.Output["msg"] = fmt.Sprintf("Kernel parameter %s would be removed from %s", name, sysctlFile)
		result.Duration = time.Since(result.Timestamp)
		return result, nil
	}
	if err := writeHostFile(ctx, host, args, sysctlFile, []byte(updated), 0o644); err != nil {
		return m.failResult(result, fmt.Sprintf("failed to update %s: %v", sysctlFile, err))
	}
	result.Output["removed_from_file"] = sysctlFile
	result.Output["msg"] = fmt.Sprintf("Kernel parameter %s removed from %s", name, sysctlFile)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// normalizeSysctl compares multi-value parameters (e.g. port ranges) by
// their fields: sysctl prints them tab-separated
func normalizeSysctl(v string) string {
	return strings.Join(strings.Fields(v), " ")
}

// setSysctlLine sets "name = value" in a sysctl.d file, keeping other lines
func setSysctlLine(content, name, value string) (string, bool) {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if content == "" {
		lines = nil
	}
	want := name + " = " + value
	found := false
	for i, line := range lines {
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != name || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if found { // a duplicate: drop it
			lines[i] = "\x00"
			continue
		}
		found = true
		if normalizeSysctl(val) == value {
			continue
		}
		lines[i] = want
	}
	if !found {
		lines = append(lines, want)
	}
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if l != "\x00" {
			out = append(out, l)
		}
	}
	updated := strings.Join(out, "\n") + "\n"
	return updated, updated != content
}

// removeSysctlLine drops every "name = ..." line
func removeSysctlLine(content, name string) (string, bool) {
	var out []string
	changed := false
	for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		key, _, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(key) == name && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			changed = true
			continue
		}
		if line != "" || len(out) > 0 {
			out = append(out, line)
		}
	}
	if len(out) == 0 {
		return "", changed
	}
	return strings.Join(out, "\n") + "\n", changed
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
