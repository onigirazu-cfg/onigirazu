package modules

import (
	"context"
	"fmt"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// MockExecutor is a mock implementation of ModuleExecutor for testing
type MockExecutor struct {
	host types.Host

	// ExecuteFunc allows custom behavior for Execute calls
	ExecuteFunc func(command string, args ...string) (string, error)

	// ExecuteContextFunc allows custom behavior for ExecuteContext calls
	ExecuteContextFunc func(ctx context.Context, command string, args ...string) (string, error)

	// Track calls for verification
	ExecutedCommands    []string
	ExecuteContextCalls []string

	// Predefined responses for common commands
	Responses map[string]string

	// Predefined errors for specific commands
	Errors map[string]error

	// BecomeSettings stores privilege escalation settings
	BecomeSettings struct {
		Enabled bool
		User    string
		Method  string
	}
}

// NewMockExecutor creates a new MockExecutor for testing
func NewMockExecutor(host types.Host) *MockExecutor {
	return &MockExecutor{
		host:             host,
		ExecutedCommands: []string{},
		Responses:        make(map[string]string),
		Errors:           make(map[string]error),
	}
}

// Execute runs a command and returns a mocked response
func (m *MockExecutor) Execute(command string, args ...string) (string, error) {
	// Use custom function if provided
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(command, args...)
	}

	// Track the command
	fullCmd := command
	for _, arg := range args {
		fullCmd += " " + arg
	}
	m.ExecutedCommands = append(m.ExecutedCommands, fullCmd)

	// Check for predefined error first
	if err, exists := m.Errors[command]; exists {
		return "", err
	}

	// Return predefined response or empty string
	if response, exists := m.Responses[command]; exists {
		return response, nil
	}

	// Default: return empty string
	return "", nil
}

// ExecuteContext runs a command with context and returns a mocked response
func (m *MockExecutor) ExecuteContext(ctx context.Context, command string, args ...string) (string, error) {
	// Use custom function if provided
	if m.ExecuteContextFunc != nil {
		return m.ExecuteContextFunc(ctx, command, args...)
	}

	// Track the command
	fullCmd := command
	for _, arg := range args {
		fullCmd += " " + arg
	}
	m.ExecuteContextCalls = append(m.ExecuteContextCalls, fullCmd)

	// Check context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Delegate to Execute for consistency
	return m.Execute(command, args...)
}

// Close closes the mock executor (no-op for mock)
func (m *MockExecutor) Close() error {
	return nil
}

// SetBecome enables privilege escalation
func (m *MockExecutor) SetBecome(become bool, becomeUser, becomeMethod string) {
	m.BecomeSettings.Enabled = become
	m.BecomeSettings.User = becomeUser
	m.BecomeSettings.Method = becomeMethod
}

// GetHost returns the target host
func (m *MockExecutor) GetHost() types.Host {
	return m.host
}

// Helper methods for common test scenarios

// SetResponse sets a predefined response for a command
func (m *MockExecutor) SetResponse(command, response string) {
	m.Responses[command] = response
}

// SetError sets a predefined error for a command
func (m *MockExecutor) SetError(command string, err error) {
	m.Errors[command] = err
}

// WasCommandExecuted checks if a specific command was executed
func (m *MockExecutor) WasCommandExecuted(command string) bool {
	for _, cmd := range m.ExecutedCommands {
		if cmd == command {
			return true
		}
	}
	return false
}

// WasCommandExecutedContains checks if any executed command contains the substring
func (m *MockExecutor) WasCommandExecutedContains(substring string) bool {
	for _, cmd := range m.ExecutedCommands {
		if contains(cmd, substring) {
			return true
		}
	}
	return false
}

// GetLastCommand returns the last executed command
func (m *MockExecutor) GetLastCommand() string {
	if len(m.ExecutedCommands) == 0 {
		return ""
	}
	return m.ExecutedCommands[len(m.ExecutedCommands)-1]
}

// ClearHistory clears the command history
func (m *MockExecutor) ClearHistory() {
	m.ExecutedCommands = []string{}
	m.ExecuteContextCalls = []string{}
}

// SetupSystemctlResponses sets up common systemctl responses for service tests
func (m *MockExecutor) SetupSystemctlResponses() {
	m.SetResponse("systemctl is-enabled nginx.service", "enabled")
	m.SetResponse("systemctl is-active nginx.service", "active")
	m.SetResponse("sudo -n systemctl status nginx", "● nginx.service - Nginx HTTP server\n   Loaded: loaded (/etc/systemd/system/nginx.service; enabled)\n   Active: active (running) since...")
}

// SetupFirewallResponses sets up common firewall-cmd responses
func (m *MockExecutor) SetupFirewallResponses() {
	m.SetResponse("firewall-cmd --state", "running")
	m.SetResponse("firewall-cmd --list-all", "public (active)\n  target: default\n  icmp-block-inversion: no\n  interfaces: eth0\n  sources: \n  services: http https\n  ports: 8080/tcp\n  protocols: \n  forward: no\n  masquerade: no\n  forward-ports: \n  source-ports: \n  icmp-blocks: \n  rich rules:")
}

// SetupCronResponses sets up common cron responses
func (m *MockExecutor) SetupCronResponses() {
	m.SetResponse("crontab -l", "0 2 * * * /usr/local/bin/backup.sh")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// SetupCustom allows setting up the mock with a custom execute function
func (m *MockExecutor) SetupCustom(executeFunc func(command string, args ...string) (string, error)) {
	m.ExecuteFunc = executeFunc
}

// SetupFailingCommand sets up a command to fail with a specific error
func (m *MockExecutor) SetupFailingCommand(command string, errorMsg string) {
	m.Errors[command] = fmt.Errorf("%s", errorMsg)
}
