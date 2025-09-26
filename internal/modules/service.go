package modules

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ServiceModule implements service management
type ServiceModule struct {
	BaseModule
	serviceManager ServiceManager
}

// ServiceManager interface for different service management systems
type ServiceManager interface {
	Start(name string) error
	Stop(name string) error
	Restart(name string) error
	Reload(name string) error
	Enable(name string) error
	Disable(name string) error
	IsRunning(name string) (bool, error)
	IsEnabled(name string) (bool, error)
	GetStatus(name string) (ServiceStatus, error)
}

// ServiceStatus represents service status information
type ServiceStatus struct {
	Name        string    `json:"name"`
	Running     bool      `json:"running"`
	Enabled     bool      `json:"enabled"`
	PID         int       `json:"pid,omitempty"`
	Uptime      string    `json:"uptime,omitempty"`
	Memory      string    `json:"memory,omitempty"`
	CPU         string    `json:"cpu,omitempty"`
	Description string    `json:"description,omitempty"`
	LoadState   string    `json:"load_state,omitempty"`
	ActiveState string    `json:"active_state,omitempty"`
	SubState    string    `json:"sub_state,omitempty"`
	LastStarted time.Time `json:"last_started,omitempty"`
}

// ServiceAction represents the action to perform
type ServiceAction string

const (
	ServiceActionStart   ServiceAction = "start"
	ServiceActionStop    ServiceAction = "stop"
	ServiceActionRestart ServiceAction = "restart"
	ServiceActionReload  ServiceAction = "reload"
	ServiceActionEnable  ServiceAction = "enable"
	ServiceActionDisable ServiceAction = "disable"
	ServiceActionStatus  ServiceAction = "status"
)

// NewServiceModule creates a new service module
func NewServiceModule() *ServiceModule {
	var manager ServiceManager

	// Detect service management system
	switch runtime.GOOS {
	case "linux":
		if m.hasSystemd() {
			manager = &SystemdManager{}
		} else if m.hasSysVInit() {
			manager = &SysVInitManager{}
		} else {
			manager = &GenericManager{}
		}
	case "darwin":
		manager = &LaunchdManager{}
	case "windows":
		manager = &WindowsServiceManager{}
	default:
		manager = &GenericManager{}
	}

	return &ServiceModule{
		BaseModule: BaseModule{
			name:        "service",
			description: "Manage system services",
		},
		serviceManager: manager,
	}
}

// Execute manages system services
func (m *ServiceModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "service",
		Host:      host.Name,
		Module:    m.name,
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Get required parameters
	name, ok := args["name"].(string)
	if !ok {
		return m.failResult(result, "name parameter is required")
	}

	state := getStringArg(args, "state", "started")
	enabled := getBoolArg(args, "enabled", false)

	// Get current status
	currentStatus, err := m.serviceManager.GetStatus(name)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to get service status: %v", err))
	}

	result.Output["service_status"] = currentStatus

	// Handle state changes
	changed := false
	switch state {
	case "started":
		if !currentStatus.Running {
			if err := m.serviceManager.Start(name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to start service: %v", err))
			}
			changed = true
			result.Output["action"] = "started"
		}
	case "stopped":
		if currentStatus.Running {
			if err := m.serviceManager.Stop(name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to stop service: %v", err))
			}
			changed = true
			result.Output["action"] = "stopped"
		}
	case "restarted":
		if err := m.serviceManager.Restart(name); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to restart service: %v", err))
		}
		changed = true
		result.Output["action"] = "restarted"
	case "reloaded":
		if err := m.serviceManager.Reload(name); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to reload service: %v", err))
		}
		changed = true
		result.Output["action"] = "reloaded"
	}

	// Handle enabled state
	if args["enabled"] != nil {
		if enabled && !currentStatus.Enabled {
			if err := m.serviceManager.Enable(name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to enable service: %v", err))
			}
			changed = true
			result.Output["enabled"] = true
		} else if !enabled && currentStatus.Enabled {
			if err := m.serviceManager.Disable(name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to disable service: %v", err))
			}
			changed = true
			result.Output["enabled"] = false
		}
	}

	// Get updated status
	if changed {
		updatedStatus, err := m.serviceManager.GetStatus(name)
		if err == nil {
			result.Output["service_status"] = updatedStatus
		}
	}

	result.Changed = changed
	result.Duration = time.Since(startTime)

	return result, nil
}

// Validate validates service module arguments
func (m *ServiceModule) Validate(args map[string]interface{}) error {
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

// SystemdManager implements service management for systemd
type SystemdManager struct{}

func (s *SystemdManager) Start(name string) error {
	return s.runSystemctl("start", name)
}

func (s *SystemdManager) Stop(name string) error {
	return s.runSystemctl("stop", name)
}

func (s *SystemdManager) Restart(name string) error {
	return s.runSystemctl("restart", name)
}

func (s *SystemdManager) Reload(name string) error {
	return s.runSystemctl("reload", name)
}

func (s *SystemdManager) Enable(name string) error {
	return s.runSystemctl("enable", name)
}

func (s *SystemdManager) Disable(name string) error {
	return s.runSystemctl("disable", name)
}

func (s *SystemdManager) IsRunning(name string) (bool, error) {
	cmd := exec.Command("systemctl", "is-active", name)
	output, err := cmd.Output()
	if err != nil {
		return false, nil // Service not running
	}
	return strings.TrimSpace(string(output)) == "active", nil
}

func (s *SystemdManager) IsEnabled(name string) (bool, error) {
	cmd := exec.Command("systemctl", "is-enabled", name)
	output, err := cmd.Output()
	if err != nil {
		return false, nil // Service not enabled
	}
	return strings.TrimSpace(string(output)) == "enabled", nil
}

func (s *SystemdManager) GetStatus(name string) (ServiceStatus, error) {
	status := ServiceStatus{Name: name}

	// Check if running
	running, _ := s.IsRunning(name)
	status.Running = running

	// Check if enabled
	enabled, _ := s.IsEnabled(name)
	status.Enabled = enabled

	// Get detailed status
	cmd := exec.Command("systemctl", "show", name, "--property=LoadState,ActiveState,SubState,MainPID,Description")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
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
						fmt.Sscanf(value, "%d", &status.PID)
					}
				case "Description":
					status.Description = value
				}
			}
		}
	}

	return status, nil
}

func (s *SystemdManager) runSystemctl(action, name string) error {
	cmd := exec.Command("systemctl", action, name)
	return cmd.Run()
}

// LaunchdManager implements service management for macOS launchd
type LaunchdManager struct{}

func (l *LaunchdManager) Start(name string) error {
	cmd := exec.Command("launchctl", "start", name)
	return cmd.Run()
}

func (l *LaunchdManager) Stop(name string) error {
	cmd := exec.Command("launchctl", "stop", name)
	return cmd.Run()
}

func (l *LaunchdManager) Restart(name string) error {
	if err := l.Stop(name); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return l.Start(name)
}

func (l *LaunchdManager) Reload(name string) error {
	return l.Restart(name) // launchd doesn't have reload
}

func (l *LaunchdManager) Enable(name string) error {
	cmd := exec.Command("launchctl", "load", "-w", name)
	return cmd.Run()
}

func (l *LaunchdManager) Disable(name string) error {
	cmd := exec.Command("launchctl", "unload", "-w", name)
	return cmd.Run()
}

func (l *LaunchdManager) IsRunning(name string) (bool, error) {
	cmd := exec.Command("launchctl", "list", name)
	err := cmd.Run()
	return err == nil, nil
}

func (l *LaunchdManager) IsEnabled(name string) (bool, error) {
	// Check if service file exists and is loaded
	cmd := exec.Command("launchctl", "list", name)
	err := cmd.Run()
	return err == nil, nil
}

func (l *LaunchdManager) GetStatus(name string) (ServiceStatus, error) {
	status := ServiceStatus{Name: name}

	running, _ := l.IsRunning(name)
	status.Running = running

	enabled, _ := l.IsEnabled(name)
	status.Enabled = enabled

	return status, nil
}

// SysVInitManager implements service management for SysV init
type SysVInitManager struct{}

func (s *SysVInitManager) Start(name string) error {
	cmd := exec.Command("service", name, "start")
	return cmd.Run()
}

func (s *SysVInitManager) Stop(name string) error {
	cmd := exec.Command("service", name, "stop")
	return cmd.Run()
}

func (s *SysVInitManager) Restart(name string) error {
	cmd := exec.Command("service", name, "restart")
	return cmd.Run()
}

func (s *SysVInitManager) Reload(name string) error {
	cmd := exec.Command("service", name, "reload")
	return cmd.Run()
}

func (s *SysVInitManager) Enable(name string) error {
	cmd := exec.Command("chkconfig", name, "on")
	return cmd.Run()
}

func (s *SysVInitManager) Disable(name string) error {
	cmd := exec.Command("chkconfig", name, "off")
	return cmd.Run()
}

func (s *SysVInitManager) IsRunning(name string) (bool, error) {
	cmd := exec.Command("service", name, "status")
	err := cmd.Run()
	return err == nil, nil
}

func (s *SysVInitManager) IsEnabled(name string) (bool, error) {
	cmd := exec.Command("chkconfig", "--list", name)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(output), ":on"), nil
}

func (s *SysVInitManager) GetStatus(name string) (ServiceStatus, error) {
	status := ServiceStatus{Name: name}

	running, _ := s.IsRunning(name)
	status.Running = running

	enabled, _ := s.IsEnabled(name)
	status.Enabled = enabled

	return status, nil
}

// WindowsServiceManager implements service management for Windows
type WindowsServiceManager struct{}

func (w *WindowsServiceManager) Start(name string) error {
	cmd := exec.Command("sc", "start", name)
	return cmd.Run()
}

func (w *WindowsServiceManager) Stop(name string) error {
	cmd := exec.Command("sc", "stop", name)
	return cmd.Run()
}

func (w *WindowsServiceManager) Restart(name string) error {
	if err := w.Stop(name); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return w.Start(name)
}

func (w *WindowsServiceManager) Reload(name string) error {
	return w.Restart(name) // Windows doesn't have reload
}

func (w *WindowsServiceManager) Enable(name string) error {
	cmd := exec.Command("sc", "config", name, "start=", "auto")
	return cmd.Run()
}

func (w *WindowsServiceManager) Disable(name string) error {
	cmd := exec.Command("sc", "config", name, "start=", "disabled")
	return cmd.Run()
}

func (w *WindowsServiceManager) IsRunning(name string) (bool, error) {
	cmd := exec.Command("sc", "query", name)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(output), "RUNNING"), nil
}

func (w *WindowsServiceManager) IsEnabled(name string) (bool, error) {
	cmd := exec.Command("sc", "qc", name)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(output), "AUTO_START"), nil
}

func (w *WindowsServiceManager) GetStatus(name string) (ServiceStatus, error) {
	status := ServiceStatus{Name: name}

	running, _ := w.IsRunning(name)
	status.Running = running

	enabled, _ := w.IsEnabled(name)
	status.Enabled = enabled

	return status, nil
}

// GenericManager implements basic service management
type GenericManager struct{}

func (g *GenericManager) Start(name string) error {
	return fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManager) Stop(name string) error {
	return fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManager) Restart(name string) error {
	return fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManager) Reload(name string) error {
	return fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManager) Enable(name string) error {
	return fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManager) Disable(name string) error {
	return fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManager) IsRunning(name string) (bool, error) {
	return false, fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManager) IsEnabled(name string) (bool, error) {
	return false, fmt.Errorf("service management not supported on this platform")
}

func (g *GenericManager) GetStatus(name string) (ServiceStatus, error) {
	return ServiceStatus{Name: name}, fmt.Errorf("service management not supported on this platform")
}

// Helper functions for service detection
func (m *ServiceModule) hasSystemd() bool {
	_, err := os.Stat("/run/systemd/system")
	return err == nil
}

func (m *ServiceModule) hasSysVInit() bool {
	_, err := os.Stat("/etc/init.d")
	return err == nil
}

// failResult creates a failed result
func (m *ServiceModule) failResult(result types.TaskResult, message string) (types.TaskResult, error) {
	result.Success = false
	result.Failed = true
	result.Error = message
	result.Duration = time.Since(result.Timestamp)
	return result, fmt.Errorf(message)
}
