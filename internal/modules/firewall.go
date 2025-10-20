package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// FirewallModule implements firewall management
type FirewallModule struct {
	*BaseExecutorModule
}

// FirewallManager interface for different firewall systems
type FirewallManager interface {
	Enable() error
	Disable() error
	IsEnabled() (bool, error)
	AllowPort(port, protocol string) error
	DenyPort(port, protocol string) error
	AllowService(service string) error
	DenyService(service string) error
	AllowFrom(source string) error
	DenyFrom(source string) error
	DeleteRule(rule string) error
	ListRules() ([]string, error)
	Reload() error
	SetExecutor(executor *executor.CommandExecutor)
	GetType() string
}

// NewFirewallModule creates a new firewall module
func NewFirewallModule() *FirewallModule {
	return &FirewallModule{
		BaseExecutorModule: NewBaseExecutorModule("firewall"),
	}
}

// GetDescription returns the module description
func (m *FirewallModule) GetDescription() string {
	return "Manage firewall rules (UFW, firewalld, iptables)"
}

// Execute manages firewall operations
func (m *FirewallModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "firewall",
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Use CreateExecutor to get fresh executor for this host (NO caching!)
	exec, err := m.CreateExecutor(host)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to create executor: %v", err))
	}
	defer exec.Close()

	// Detect firewall manager for this host (fresh each time)
	manager, err := m.detectFirewall(exec)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to detect firewall: %v", err))
	}
	manager.SetExecutor(exec)
	result.Output["firewall_type"] = manager.GetType()

	// Get operation type
	operation := getStringArg(args, "operation", "rule")

	switch operation {
	case "enable":
		return m.handleEnable(ctx, manager, host, args, result)
	case "disable":
		return m.handleDisable(ctx, manager, host, args, result)
	case "rule":
		return m.handleRule(ctx, manager, host, args, result)
	case "service":
		return m.handleService(ctx, manager, host, args, result)
	case "source":
		return m.handleSource(ctx, manager, host, args, result)
	case "list":
		return m.handleList(ctx, manager, host, args, result)
	case "reload":
		return m.handleReload(ctx, manager, host, args, result)
	default:
		return m.failResult(result, fmt.Sprintf("unknown operation: %s", operation))
	}
}

// detectFirewall detects which firewall system is available
func (m *FirewallModule) detectFirewall(exec *executor.CommandExecutor) (FirewallManager, error) {
	// Check for UFW
	if _, err := exec.Execute("which", "ufw"); err == nil {
		return &UFWManager{}, nil
	}

	// Check for firewalld
	if _, err := exec.Execute("which", "firewall-cmd"); err == nil {
		return &FirewalldManager{}, nil
	}

	// Check for iptables
	if _, err := exec.Execute("which", "iptables"); err == nil {
		return &IptablesManager{}, nil
	}

	return nil, fmt.Errorf("no supported firewall found (ufw, firewalld, iptables)")
}

// handleEnable enables the firewall
func (m *FirewallModule) handleEnable(ctx context.Context, manager FirewallManager, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	enabled, err := manager.IsEnabled()
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to check firewall status: %v", err))
	}

	if !enabled {
		if err := manager.Enable(); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to enable firewall: %v", err))
		}
		result.Changed = true
		result.Output["action"] = "enabled"
	} else {
		result.Output["action"] = "already_enabled"
	}

	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleDisable disables the firewall
func (m *FirewallModule) handleDisable(ctx context.Context, manager FirewallManager, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	enabled, err := manager.IsEnabled()
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to check firewall status: %v", err))
	}

	if enabled {
		if err := manager.Disable(); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to disable firewall: %v", err))
		}
		result.Changed = true
		result.Output["action"] = "disabled"
	} else {
		result.Output["action"] = "already_disabled"
	}

	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleRule manages firewall rules for ports
func (m *FirewallModule) handleRule(ctx context.Context, manager FirewallManager, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	port := getStringArg(args, "port", "")
	protocol := getStringArg(args, "protocol", "tcp")
	action := getStringArg(args, "action", "allow")
	state := getStringArg(args, "state", "present")

	if port == "" {
		return m.failResult(result, "port parameter is required")
	}

	changed := false

	if state == "present" {
		if action == "allow" {
			if err := manager.AllowPort(port, protocol); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to allow port: %v", err))
			}
			changed = true
			result.Output["action"] = fmt.Sprintf("allowed_%s_%s", port, protocol)
		} else if action == "deny" {
			if err := manager.DenyPort(port, protocol); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to deny port: %v", err))
			}
			changed = true
			result.Output["action"] = fmt.Sprintf("denied_%s_%s", port, protocol)
		}
	} else if state == "absent" {
		rule := fmt.Sprintf("%s/%s", port, protocol)
		if err := manager.DeleteRule(rule); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to delete rule: %v", err))
		}
		changed = true
		result.Output["action"] = fmt.Sprintf("deleted_%s_%s", port, protocol)
	}

	result.Changed = changed
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleService manages firewall rules for services
func (m *FirewallModule) handleService(ctx context.Context, manager FirewallManager, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	service := getStringArg(args, "service", "")
	action := getStringArg(args, "action", "allow")
	state := getStringArg(args, "state", "present")

	if service == "" {
		return m.failResult(result, "service parameter is required")
	}

	changed := false

	if state == "present" {
		if action == "allow" {
			if err := manager.AllowService(service); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to allow service: %v", err))
			}
			changed = true
			result.Output["action"] = fmt.Sprintf("allowed_service_%s", service)
		} else if action == "deny" {
			if err := manager.DenyService(service); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to deny service: %v", err))
			}
			changed = true
			result.Output["action"] = fmt.Sprintf("denied_service_%s", service)
		}
	} else if state == "absent" {
		if err := manager.DeleteRule(service); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to delete service rule: %v", err))
		}
		changed = true
		result.Output["action"] = fmt.Sprintf("deleted_service_%s", service)
	}

	result.Changed = changed
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleSource manages firewall rules for IP addresses/subnets
func (m *FirewallModule) handleSource(ctx context.Context, manager FirewallManager, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	source := getStringArg(args, "source", "")
	action := getStringArg(args, "action", "allow")
	state := getStringArg(args, "state", "present")

	if source == "" {
		return m.failResult(result, "source parameter is required")
	}

	changed := false

	if state == "present" {
		if action == "allow" {
			if err := manager.AllowFrom(source); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to allow from source: %v", err))
			}
			changed = true
			result.Output["action"] = fmt.Sprintf("allowed_from_%s", source)
		} else if action == "deny" {
			if err := manager.DenyFrom(source); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to deny from source: %v", err))
			}
			changed = true
			result.Output["action"] = fmt.Sprintf("denied_from_%s", source)
		}
	} else if state == "absent" {
		if err := manager.DeleteRule(source); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to delete source rule: %v", err))
		}
		changed = true
		result.Output["action"] = fmt.Sprintf("deleted_from_%s", source)
	}

	result.Changed = changed
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleList lists firewall rules
func (m *FirewallModule) handleList(ctx context.Context, manager FirewallManager, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	rules, err := manager.ListRules()
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to list rules: %v", err))
	}

	result.Output["rules"] = rules
	result.Output["rules_count"] = len(rules)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleReload reloads firewall configuration
func (m *FirewallModule) handleReload(ctx context.Context, manager FirewallManager, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	if err := manager.Reload(); err != nil {
		return m.failResult(result, fmt.Sprintf("failed to reload firewall: %v", err))
	}

	result.Changed = true
	result.Output["action"] = "reloaded"
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// UFWManager implements UFW firewall management
type UFWManager struct {
	executor *executor.CommandExecutor
}

func (u *UFWManager) Enable() error {
	_, err := u.executor.Execute("ufw", "--force", "enable")
	return err
}

func (u *UFWManager) Disable() error {
	_, err := u.executor.Execute("ufw", "disable")
	return err
}

func (u *UFWManager) IsEnabled() (bool, error) {
	output, err := u.executor.Execute("ufw", "status")
	if err != nil {
		return false, err
	}
	return strings.Contains(output, "Status: active"), nil
}

func (u *UFWManager) AllowPort(port, protocol string) error {
	_, err := u.executor.Execute("ufw", "allow", fmt.Sprintf("%s/%s", port, protocol))
	return err
}

func (u *UFWManager) DenyPort(port, protocol string) error {
	_, err := u.executor.Execute("ufw", "deny", fmt.Sprintf("%s/%s", port, protocol))
	return err
}

func (u *UFWManager) AllowService(service string) error {
	_, err := u.executor.Execute("ufw", "allow", service)
	return err
}

func (u *UFWManager) DenyService(service string) error {
	_, err := u.executor.Execute("ufw", "deny", service)
	return err
}

func (u *UFWManager) AllowFrom(source string) error {
	_, err := u.executor.Execute("ufw", "allow", "from", source)
	return err
}

func (u *UFWManager) DenyFrom(source string) error {
	_, err := u.executor.Execute("ufw", "deny", "from", source)
	return err
}

func (u *UFWManager) DeleteRule(rule string) error {
	_, err := u.executor.Execute("ufw", "delete", "allow", rule)
	return err
}

func (u *UFWManager) ListRules() ([]string, error) {
	output, err := u.executor.Execute("ufw", "status", "numbered")
	if err != nil {
		return nil, err
	}
	return strings.Split(output, "\n"), nil
}

func (u *UFWManager) Reload() error {
	_, err := u.executor.Execute("ufw", "reload")
	return err
}

func (u *UFWManager) SetExecutor(executor *executor.CommandExecutor) {
	u.executor = executor
}

func (u *UFWManager) GetType() string {
	return "ufw"
}

// FirewalldManager implements firewalld management
type FirewalldManager struct {
	executor *executor.CommandExecutor
}

func (f *FirewalldManager) Enable() error {
	if _, err := f.executor.Execute("systemctl", "enable", "firewalld"); err != nil {
		return err
	}
	_, err := f.executor.Execute("systemctl", "start", "firewalld")
	return err
}

func (f *FirewalldManager) Disable() error {
	if _, err := f.executor.Execute("systemctl", "stop", "firewalld"); err != nil {
		return err
	}
	_, err := f.executor.Execute("systemctl", "disable", "firewalld")
	return err
}

func (f *FirewalldManager) IsEnabled() (bool, error) {
	output, err := f.executor.Execute("firewall-cmd", "--state")
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(output) == "running", nil
}

func (f *FirewalldManager) AllowPort(port, protocol string) error {
	_, err := f.executor.Execute("firewall-cmd", "--permanent", "--add-port", fmt.Sprintf("%s/%s", port, protocol))
	if err != nil {
		return err
	}
	return f.Reload()
}

func (f *FirewalldManager) DenyPort(port, protocol string) error {
	_, err := f.executor.Execute("firewall-cmd", "--permanent", "--remove-port", fmt.Sprintf("%s/%s", port, protocol))
	if err != nil {
		return err
	}
	return f.Reload()
}

func (f *FirewalldManager) AllowService(service string) error {
	_, err := f.executor.Execute("firewall-cmd", "--permanent", "--add-service", service)
	if err != nil {
		return err
	}
	return f.Reload()
}

func (f *FirewalldManager) DenyService(service string) error {
	_, err := f.executor.Execute("firewall-cmd", "--permanent", "--remove-service", service)
	if err != nil {
		return err
	}
	return f.Reload()
}

func (f *FirewalldManager) AllowFrom(source string) error {
	_, err := f.executor.Execute("firewall-cmd", "--permanent", "--add-source", source)
	if err != nil {
		return err
	}
	return f.Reload()
}

func (f *FirewalldManager) DenyFrom(source string) error {
	_, err := f.executor.Execute("firewall-cmd", "--permanent", "--remove-source", source)
	if err != nil {
		return err
	}
	return f.Reload()
}

func (f *FirewalldManager) DeleteRule(rule string) error {
	// Try to remove as port
	if strings.Contains(rule, "/") {
		_, err := f.executor.Execute("firewall-cmd", "--permanent", "--remove-port", rule)
		if err == nil {
			return f.Reload()
		}
	}

	// Try to remove as service
	_, err := f.executor.Execute("firewall-cmd", "--permanent", "--remove-service", rule)
	if err == nil {
		return f.Reload()
	}

	// Try to remove as source
	_, err = f.executor.Execute("firewall-cmd", "--permanent", "--remove-source", rule)
	if err == nil {
		return f.Reload()
	}

	return err
}

func (f *FirewalldManager) ListRules() ([]string, error) {
	var rules []string

	// List ports
	output, err := f.executor.Execute("firewall-cmd", "--list-ports")
	if err == nil && output != "" {
		rules = append(rules, "Ports:")
		rules = append(rules, strings.Split(strings.TrimSpace(output), " ")...)
	}

	// List services
	output, err = f.executor.Execute("firewall-cmd", "--list-services")
	if err == nil && output != "" {
		rules = append(rules, "Services:")
		rules = append(rules, strings.Split(strings.TrimSpace(output), " ")...)
	}

	// List sources
	output, err = f.executor.Execute("firewall-cmd", "--list-sources")
	if err == nil && output != "" {
		rules = append(rules, "Sources:")
		rules = append(rules, strings.Split(strings.TrimSpace(output), " ")...)
	}

	return rules, nil
}

func (f *FirewalldManager) Reload() error {
	_, err := f.executor.Execute("firewall-cmd", "--reload")
	return err
}

func (f *FirewalldManager) SetExecutor(executor *executor.CommandExecutor) {
	f.executor = executor
}

func (f *FirewalldManager) GetType() string {
	return "firewalld"
}

// IptablesManager implements iptables management
type IptablesManager struct {
	executor *executor.CommandExecutor
}

func (i *IptablesManager) Enable() error {
	// iptables is always "enabled" if installed
	return nil
}

func (i *IptablesManager) Disable() error {
	// Flush all rules
	_, err := i.executor.Execute("iptables", "-F")
	return err
}

func (i *IptablesManager) IsEnabled() (bool, error) {
	// Check if iptables has any rules
	output, err := i.executor.Execute("iptables", "-L")
	if err != nil {
		return false, err
	}
	return len(output) > 0, nil
}

func (i *IptablesManager) AllowPort(port, protocol string) error {
	_, err := i.executor.Execute("iptables", "-A", "INPUT", "-p", protocol, "--dport", port, "-j", "ACCEPT")
	if err != nil {
		return err
	}
	return i.saveRules()
}

func (i *IptablesManager) DenyPort(port, protocol string) error {
	_, err := i.executor.Execute("iptables", "-A", "INPUT", "-p", protocol, "--dport", port, "-j", "DROP")
	if err != nil {
		return err
	}
	return i.saveRules()
}

func (i *IptablesManager) AllowService(service string) error {
	return fmt.Errorf("iptables does not support service names directly, use port numbers")
}

func (i *IptablesManager) DenyService(service string) error {
	return fmt.Errorf("iptables does not support service names directly, use port numbers")
}

func (i *IptablesManager) AllowFrom(source string) error {
	_, err := i.executor.Execute("iptables", "-A", "INPUT", "-s", source, "-j", "ACCEPT")
	if err != nil {
		return err
	}
	return i.saveRules()
}

func (i *IptablesManager) DenyFrom(source string) error {
	_, err := i.executor.Execute("iptables", "-A", "INPUT", "-s", source, "-j", "DROP")
	if err != nil {
		return err
	}
	return i.saveRules()
}

func (i *IptablesManager) DeleteRule(rule string) error {
	// This is simplified - in practice, you'd need to parse the rule
	return fmt.Errorf("delete rule not fully implemented for iptables")
}

func (i *IptablesManager) ListRules() ([]string, error) {
	output, err := i.executor.Execute("iptables", "-L", "-n", "-v")
	if err != nil {
		return nil, err
	}
	return strings.Split(output, "\n"), nil
}

func (i *IptablesManager) Reload() error {
	// iptables doesn't have a reload concept
	return nil
}

func (i *IptablesManager) saveRules() error {
	// Try iptables-save with different methods
	if _, err := i.executor.Execute("which", "iptables-save"); err == nil {
		if _, err := i.executor.Execute("sh", "-c", "iptables-save > /etc/iptables/rules.v4"); err == nil {
			return nil
		}
		if _, err := i.executor.Execute("sh", "-c", "iptables-save > /etc/sysconfig/iptables"); err == nil {
			return nil
		}
	}
	return nil // Don't fail if we can't save
}

func (i *IptablesManager) SetExecutor(executor *executor.CommandExecutor) {
	i.executor = executor
}

func (i *IptablesManager) GetType() string {
	return "iptables"
}

// Validate validates firewall module arguments
func (m *FirewallModule) Validate(args map[string]interface{}) error {
	operation := getStringArg(args, "operation", "rule")

	switch operation {
	case "enable", "disable", "reload", "list":
		// No additional validation needed
	case "rule":
		if _, exists := args["port"]; !exists {
			return fmt.Errorf("port parameter is required for rule operation")
		}
	case "service":
		if _, exists := args["service"]; !exists {
			return fmt.Errorf("service parameter is required for service operation")
		}
	case "source":
		if _, exists := args["source"]; !exists {
			return fmt.Errorf("source parameter is required for source operation")
		}
	default:
		return fmt.Errorf("invalid operation: %s", operation)
	}

	return nil
}
