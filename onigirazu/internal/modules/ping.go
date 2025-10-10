package modules

import (
	"context"
	"fmt"
	"time"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// PingModule checks connectivity to hosts
type PingModule struct {
	*BaseModule
}

// NewPingModule creates a new ping module
func NewPingModule() *PingModule {
	return &PingModule{
		BaseModule: NewBaseModule("ping"),
	}
}

func (m *PingModule) GetDescription() string {
	return "Tests connectivity to hosts"
}

func (m *PingModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  m.getTaskName(args),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
	}

	// Get optional data parameter (custom response message)
	data := "pong"
	if dataVal, exists := args["data"]; exists {
		if dataStr, ok := dataVal.(string); ok {
			data = dataStr
		}
	}

	// Check if host is local or remote
	isLocal := sshpkg.IsLocal(host)
	connectionType := "local"
	if !isLocal {
		connectionType = "ssh"
	}

	// For SSH connections, try to establish connection
	if !isLocal {
		if err := m.checkSSHConnection(ctx, host); err != nil {
			result.Success = false
			result.Failed = true
			result.Error = fmt.Sprintf("SSH connection failed: %v", err)
			result.Output["ping"] = "failed"
			result.Output["connection"] = connectionType
			result.Output["error"] = err.Error()
			result.Duration = time.Since(startTime)
			return result, nil
		}
	}

	// Connection successful
	result.Output["ping"] = data
	result.Output["connection"] = connectionType
	result.Output["host"] = host.Name
	result.Output["address"] = host.Address

	// Add additional host information if available
	if host.User != "" {
		result.Output["user"] = host.User
	}
	if host.Port > 0 {
		result.Output["port"] = host.Port
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *PingModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	// Ping module has no required parameters
	// Optional: data (custom response message)

	return nil
}

// checkSSHConnection attempts to establish SSH connection to verify connectivity
func (m *PingModule) checkSSHConnection(ctx context.Context, host types.Host) error {
	// Try to create SSH client
	client, err := sshpkg.NewClient(host)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	// Try to execute a simple command to verify the connection works
	// Use 'true' command which is available on all Unix systems and returns immediately
	_, err = client.ExecuteCommand("true")
	if err != nil {
		return fmt.Errorf("failed to execute test command: %w", err)
	}

	return nil
}

// getTaskName extracts task name from args
func (m *PingModule) getTaskName(args map[string]interface{}) string {
	if name, exists := args["name"]; exists {
		if nameStr, ok := name.(string); ok {
			return nameStr
		}
	}
	return "ping"
}
