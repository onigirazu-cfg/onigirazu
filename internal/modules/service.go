package modules

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name        string `json:"name"`
	Active      bool   `json:"active"`
	Enabled     bool   `json:"enabled"`
	Status      string `json:"status"`
	SubState    string `json:"sub_state,omitempty"`
	LoadState   string `json:"load_state,omitempty"`
	Running     bool   `json:"running"`
	ActiveState string `json:"active_state,omitempty"`
	PID         string `json:"pid,omitempty"`
	Description string `json:"description,omitempty"`
}

// ServiceModule implements service management
type ServiceModuleFixed struct {
	BaseModule
	// testServiceManager is used for testing only - if set, it will be used instead of detecting the service manager
	testServiceManager ServiceManagerFixed
}

// ServiceManagerFixed interface for different service management systems
type ServiceManagerFixed interface {
	Start(name string) error
	Stop(name string) error
	Restart(name string) error
	Reload(name string) error
	Enable(name string) error
	Disable(name string) error
	IsRunning(name string) (bool, error)
	IsEnabled(name string) (bool, error)
	GetStatus(name string) (ServiceStatus, error)
	SetExecutor(executor *executor.CommandExecutor)
}

// NewServiceModuleFixed creates a new service module
func NewServiceModuleFixed() *ServiceModuleFixed {
	return &ServiceModuleFixed{
		BaseModule: BaseModule{
			name:        "service",
			description: "Manage system services",
		},
	}
}

// NewServiceModule creates a new service module (compatibility wrapper)
func NewServiceModule() *ServiceModuleFixed {
	return NewServiceModuleFixed()
}

// Execute manages system services
func (m *ServiceModuleFixed) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "service",
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

	// Configure become (privilege escalation) if requested
	if become, ok := args["_become"].(bool); ok && become {
		becomeUser, _ := args["_become_user"].(string)
		becomeMethod, _ := args["_become_method"].(string)
		exec.SetBecome(true, becomeUser, becomeMethod)
	}

	// Use test service manager if set (for testing), otherwise detect it
	var serviceManager ServiceManagerFixed
	if m.testServiceManager != nil {
		serviceManager = m.testServiceManager
		serviceManager.SetExecutor(exec)
	} else {
		// Detect service manager for the remote host
		serviceManager, err = m.detectServiceManager(exec)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to detect service manager: %v", err))
		}
		serviceManager.SetExecutor(exec)
	}

	// Get required parameters
	name, ok := args["name"].(string)
	if !ok {
		return m.failResult(result, "name parameter is required")
	}

	state := getStringArg(args, "state", "started")
	enabled := getBoolArg(args, "enabled", false)

	// Get current status
	currentStatus, err := serviceManager.GetStatus(name)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to get service status: %v", err))
	}

	result.Output["service_status"] = currentStatus

	// Handle state changes
	changed := false
	switch state {
	case "started":
		if !currentStatus.Running {
			if err := serviceManager.Start(name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to start service: %v", err))
			}
			changed = true
			result.Output["action"] = "started"
		}
	case "stopped":
		if currentStatus.Running {
			if err := serviceManager.Stop(name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to stop service: %v", err))
			}
			changed = true
			result.Output["action"] = "stopped"
		}
	case "restarted":
		if err := serviceManager.Restart(name); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to restart service: %v", err))
		}
		changed = true
		result.Output["action"] = "restarted"
	case "reloaded":
		if err := serviceManager.Reload(name); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to reload service: %v", err))
		}
		changed = true
		result.Output["action"] = "reloaded"
	}

	// Handle enabled state
	if args["enabled"] != nil {
		if enabled && !currentStatus.Enabled {
			if err := serviceManager.Enable(name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to enable service: %v", err))
			}
			changed = true
			result.Output["enabled"] = true
		} else if !enabled && currentStatus.Enabled {
			if err := serviceManager.Disable(name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to disable service: %v", err))
			}
			changed = true
			result.Output["enabled"] = false
		}
	}

	// Get updated status
	if changed {
		updatedStatus, err := serviceManager.GetStatus(name)
		if err == nil {
			result.Output["service_status"] = updatedStatus
		}
	}

	result.Changed = changed
	result.Duration = time.Since(startTime)

	return result, nil
}

// Validate validates service module arguments
func (m *ServiceModuleFixed) Validate(args map[string]interface{}) error {
	if _, exists := args["name"]; !exists {
		return fmt.Errorf("name parameter is required")
	}

	if state, exists := args["state"]; exists {
		if stateStr, ok := state.(string); ok {
			validStates := []string{"started", "stopped", "restarted", "reloaded"}
			valid := false
			for _, validState := range validStates {
				if stateStr == validState {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid state: %s", stateStr)
			}
		}
	}

	return nil
}

// SystemdManagerFixed implements service management for systemd
type SystemdManagerFixed struct {
	executor *executor.CommandExecutor
}

func (s *SystemdManagerFixed) Start(name string) error {
	return s.runSystemctl("start", name)
}

func (s *SystemdManagerFixed) Stop(name string) error {
	return s.runSystemctl("stop", name)
}

func (s *SystemdManagerFixed) Restart(name string) error {
	return s.runSystemctl("restart", name)
}

func (s *SystemdManagerFixed) Reload(name string) error {
	return s.runSystemctl("reload", name)
}

func (s *SystemdManagerFixed) Enable(name string) error {
	return s.runSystemctl("enable", name)
}

func (s *SystemdManagerFixed) Disable(name string) error {
	return s.runSystemctl("disable", name)
}

func (s *SystemdManagerFixed) IsRunning(name string) (bool, error) {
	if s.executor == nil {
		return false, fmt.Errorf("executor not initialized")
	}
	output, err := s.executor.Execute("systemctl", "is-active", name)
	if err != nil {
		return false, nil // Service not running
	}
	return strings.TrimSpace(output) == "active", nil
}

func (s *SystemdManagerFixed) IsEnabled(name string) (bool, error) {
	if s.executor == nil {
		return false, fmt.Errorf("executor not initialized")
	}
	output, err := s.executor.Execute("systemctl", "is-enabled", name)
	if err != nil {
		return false, nil // Service not enabled
	}
	return strings.TrimSpace(output) == "enabled", nil
}

func (s *SystemdManagerFixed) GetStatus(name string) (ServiceStatus, error) {
	status := ServiceStatus{Name: name}

	// Check if running
	running, _ := s.IsRunning(name)
	status.Running = running

	// Check if enabled
	enabled, _ := s.IsEnabled(name)
	status.Enabled = enabled

	// Get detailed status
	if s.executor != nil {
		output, err := s.executor.Execute("systemctl", "show", name, "--property=LoadState,ActiveState,SubState,MainPID,Description")
		if err == nil {
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key, value := parts[0], parts[1]
					switch key {
					case "LoadState":
						status.LoadState = value
					case "ActiveState":
						status.ActiveState = value
					case "SubState":
						status.SubState = value
					case "MainPID":
						if value != "0" {
							_, _ = fmt.Sscanf(value, "%d", &status.PID)
						}
					case "Description":
						status.Description = value
					}
				}
			}
		}
	}

	return status, nil
}

func (s *SystemdManagerFixed) runSystemctl(action, name string) error {
	if s.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := s.executor.Execute("systemctl", action, name)
	return err
}

func (s *SystemdManagerFixed) SetExecutor(executor *executor.CommandExecutor) {
	s.executor = executor
}

// LaunchdManagerFixed implements service management for macOS launchd
type LaunchdManagerFixed struct {
	executor *executor.CommandExecutor
}

func (l *LaunchdManagerFixed) Start(name string) error {
	if l.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := l.executor.Execute("launchctl", "start", name)
	return err
}

func (l *LaunchdManagerFixed) Stop(name string) error {
	if l.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := l.executor.Execute("launchctl", "stop", name)
	return err
}

func (l *LaunchdManagerFixed) Restart(name string) error {
	if err := l.Stop(name); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return l.Start(name)
}

func (l *LaunchdManagerFixed) Reload(name string) error {
	return l.Restart(name) // launchd doesn't have reload
}

func (l *LaunchdManagerFixed) Enable(name string) error {
	if l.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := l.executor.Execute("launchctl", "load", "-w", name)
	return err
}

func (l *LaunchdManagerFixed) Disable(name string) error {
	if l.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := l.executor.Execute("launchctl", "unload", "-w", name)
	return err
}

func (l *LaunchdManagerFixed) IsRunning(name string) (bool, error) {
	if l.executor == nil {
		return false, fmt.Errorf("executor not initialized")
	}
	_, err := l.executor.Execute("launchctl", "list", name)
	return err == nil, nil
}

func (l *LaunchdManagerFixed) IsEnabled(name string) (bool, error) {
	if l.executor == nil {
		return false, fmt.Errorf("executor not initialized")
	}
	_, err := l.executor.Execute("launchctl", "list", name)
	return err == nil, nil
}

func (l *LaunchdManagerFixed) GetStatus(name string) (ServiceStatus, error) {
	status := ServiceStatus{Name: name}

	running, _ := l.IsRunning(name)
	status.Running = running

	enabled, _ := l.IsEnabled(name)
	status.Enabled = enabled

	return status, nil
}

func (l *LaunchdManagerFixed) SetExecutor(executor *executor.CommandExecutor) {
	l.executor = executor
}

// SysVInitManagerFixed implements service management for SysV init
type SysVInitManagerFixed struct {
	executor *executor.CommandExecutor
}

func (s *SysVInitManagerFixed) Start(name string) error {
	if s.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := s.executor.Execute("service", name, "start")
	return err
}

func (s *SysVInitManagerFixed) Stop(name string) error {
	if s.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := s.executor.Execute("service", name, "stop")
	return err
}

func (s *SysVInitManagerFixed) Restart(name string) error {
	if s.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := s.executor.Execute("service", name, "restart")
	return err
}

func (s *SysVInitManagerFixed) Reload(name string) error {
	if s.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := s.executor.Execute("service", name, "reload")
	return err
}

func (s *SysVInitManagerFixed) Enable(name string) error {
	if s.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := s.executor.Execute("chkconfig", name, "on")
	return err
}

func (s *SysVInitManagerFixed) Disable(name string) error {
	if s.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := s.executor.Execute("chkconfig", name, "off")
	return err
}

func (s *SysVInitManagerFixed) IsRunning(name string) (bool, error) {
	if s.executor == nil {
		return false, fmt.Errorf("executor not initialized")
	}
	_, err := s.executor.Execute("service", name, "status")
	return err == nil, nil
}

func (s *SysVInitManagerFixed) IsEnabled(name string) (bool, error) {
	if s.executor == nil {
		return false, fmt.Errorf("executor not initialized")
	}
	output, err := s.executor.Execute("chkconfig", "--list", name)
	if err != nil {
		return false, err
	}
	return strings.Contains(output, ":on"), nil
}

func (s *SysVInitManagerFixed) GetStatus(name string) (ServiceStatus, error) {
	status := ServiceStatus{Name: name}

	running, _ := s.IsRunning(name)
	status.Running = running

	enabled, _ := s.IsEnabled(name)
	status.Enabled = enabled

	return status, nil
}

func (s *SysVInitManagerFixed) SetExecutor(executor *executor.CommandExecutor) {
	s.executor = executor
}

// WindowsServiceManagerFixed implements service management for Windows
type WindowsServiceManagerFixed struct {
	executor *executor.CommandExecutor
}

func (w *WindowsServiceManagerFixed) Start(name string) error {
	if w.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := w.executor.Execute("sc", "start", name)
	return err
}

func (w *WindowsServiceManagerFixed) Stop(name string) error {
	if w.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := w.executor.Execute("sc", "stop", name)
	return err
}

func (w *WindowsServiceManagerFixed) Restart(name string) error {
	if err := w.Stop(name); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return w.Start(name)
}

func (w *WindowsServiceManagerFixed) Reload(name string) error {
	return w.Restart(name) // Windows doesn't have reload
}

func (w *WindowsServiceManagerFixed) Enable(name string) error {
	if w.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := w.executor.Execute("sc", "config", name, "start=", "auto")
	return err
}

func (w *WindowsServiceManagerFixed) Disable(name string) error {
	if w.executor == nil {
		return fmt.Errorf("executor not initialized")
	}
	_, err := w.executor.Execute("sc", "config", name, "start=", "disabled")
	return err
}

func (w *WindowsServiceManagerFixed) IsRunning(name string) (bool, error) {
	if w.executor == nil {
		return false, fmt.Errorf("executor not initialized")
	}
	output, err := w.executor.Execute("sc", "query", name)
	if err != nil {
		return false, err
	}
	return strings.Contains(output, "RUNNING"), nil
}

func (w *WindowsServiceManagerFixed) IsEnabled(name string) (bool, error) {
	if w.executor == nil {
		return false, fmt.Errorf("executor not initialized")
	}
	output, err := w.executor.Execute("sc", "qc", name)
	if err != nil {
		return false, err
	}
	return strings.Contains(output, "AUTO_START"), nil
}

func (w *WindowsServiceManagerFixed) GetStatus(name string) (ServiceStatus, error) {
	status := ServiceStatus{Name: name}

	running, _ := w.IsRunning(name)
	status.Running = running

	enabled, _ := w.IsEnabled(name)
	status.Enabled = enabled

	return status, nil
}

func (w *WindowsServiceManagerFixed) SetExecutor(executor *executor.CommandExecutor) {
	w.executor = executor
}

// GenericManagerFixed implements basic service management
type GenericManagerFixed struct {
	executor *executor.CommandExecutor
}

func (g *GenericManagerFixed) Start(name string) error {
	return fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManagerFixed) Stop(name string) error {
	return fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManagerFixed) Restart(name string) error {
	return fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManagerFixed) Reload(name string) error {
	return fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManagerFixed) Enable(name string) error {
	return fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManagerFixed) Disable(name string) error {
	return fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManagerFixed) IsRunning(name string) (bool, error) {
	return false, fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManagerFixed) IsEnabled(name string) (bool, error) {
	return false, fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManagerFixed) GetStatus(name string) (ServiceStatus, error) {
	return ServiceStatus{Name: name}, fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManagerFixed) SetExecutor(executor *executor.CommandExecutor) {
	g.executor = executor
}

// detectServiceManager detects the service manager on the remote host
func (m *ServiceModuleFixed) detectServiceManager(exec *executor.CommandExecutor) (ServiceManagerFixed, error) {
	// Check for systemd
	_, err := exec.Execute("test", "-d", "/run/systemd/system")
	if err == nil {
		return &SystemdManagerFixed{}, nil
	}

	// Check for SysV init
	_, err = exec.Execute("test", "-d", "/etc/init.d")
	if err == nil {
		return &SysVInitManagerFixed{}, nil
	}

	// Check for launchd (macOS)
	_, err = exec.Execute("test", "-d", "/Library/LaunchDaemons")
	if err == nil {
		return &LaunchdManagerFixed{}, nil
	}

	// Fallback to generic manager
	return &GenericManagerFixed{}, nil
}

// failResult creates a failed result
func (m *ServiceModuleFixed) failResult(result types.TaskResult, message string) (types.TaskResult, error) {
	result.Success = false
	result.Failed = true
	result.Error = message
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// hasSystemd checks if systemd is available
func hasSystemd() bool {
	_, err := os.Stat("/run/systemd/system")
	return err == nil
}

// hasSysVInit checks if SysV init is available
func hasSysVInit() bool {
	_, err := os.Stat("/etc/init.d")
	return err == nil
}
