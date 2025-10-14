package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestPingModule_Execute_Local(t *testing.T) {
	module := NewPingModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	args := map[string]interface{}{
		"name": "test ping",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Changed {
		t.Errorf("Expected changed=false, got changed=true")
	}

	// Check output
	if ping, ok := result.Output["ping"].(string); !ok || ping != "pong" {
		t.Errorf("Expected ping='pong', got %v", result.Output["ping"])
	}

	if conn, ok := result.Output["connection"].(string); !ok || conn != "local" {
		t.Errorf("Expected connection='local', got %v", result.Output["connection"])
	}
}

func TestPingModule_Execute_CustomData(t *testing.T) {
	module := NewPingModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	args := map[string]interface{}{
		"name": "test ping",
		"data": "custom response",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Check custom data
	if ping, ok := result.Output["ping"].(string); !ok || ping != "custom response" {
		t.Errorf("Expected ping='custom response', got %v", result.Output["ping"])
	}
}

func TestPingModule_Validate(t *testing.T) {
	module := NewPingModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid - no args",
			args: map[string]interface{}{
				"name": "test",
			},
			wantErr: false,
		},
		{
			name: "valid - with data",
			args: map[string]interface{}{
				"name": "test",
				"data": "custom",
			},
			wantErr: false,
		},
		{
			name:    "invalid - missing name",
			args:    map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPingModule_GetDescription(t *testing.T) {
	module := NewPingModule()
	desc := module.GetDescription()

	if desc == "" {
		t.Error("Expected non-empty description")
	}
}

func TestPingModule_Execute_WithHostDetails(t *testing.T) {
	module := NewPingModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
		User:    "testuser",
		Port:    2222,
	}

	args := map[string]interface{}{
		"name": "test ping with details",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Check that user and port are included in output
	if user, ok := result.Output["user"].(string); !ok || user != "testuser" {
		t.Errorf("Expected user='testuser', got %v", result.Output["user"])
	}

	if port, ok := result.Output["port"].(int); !ok || port != 2222 {
		t.Errorf("Expected port=2222, got %v", result.Output["port"])
	}

	if host, ok := result.Output["host"].(string); !ok || host != "localhost" {
		t.Errorf("Expected host='localhost', got %v", result.Output["host"])
	}

	if address, ok := result.Output["address"].(string); !ok || address != "127.0.0.1" {
		t.Errorf("Expected address='127.0.0.1', got %v", result.Output["address"])
	}
}

func TestPingModule_Execute_WithoutHostDetails(t *testing.T) {
	module := NewPingModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
		// No User or Port specified
	}

	args := map[string]interface{}{
		"name": "test ping without details",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Check that user and port are NOT included in output
	if _, ok := result.Output["user"]; ok {
		t.Errorf("Expected user to not be in output, but it was present")
	}

	if _, ok := result.Output["port"]; ok {
		t.Errorf("Expected port to not be in output, but it was present")
	}
}

func TestPingModule_Execute_InvalidDataType(t *testing.T) {
	module := NewPingModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	// Pass data as non-string type
	args := map[string]interface{}{
		"name": "test ping",
		"data": 123, // Invalid type
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Should fall back to default "pong"
	if ping, ok := result.Output["ping"].(string); !ok || ping != "pong" {
		t.Errorf("Expected ping='pong' (fallback), got %v", result.Output["ping"])
	}
}

func TestPingModule_GetTaskName_WithName(t *testing.T) {
	module := NewPingModule()

	args := map[string]interface{}{
		"name": "custom task name",
	}

	taskName := module.getTaskName(args)
	if taskName != "custom task name" {
		t.Errorf("Expected task name 'custom task name', got %s", taskName)
	}
}

func TestPingModule_GetTaskName_WithoutName(t *testing.T) {
	module := NewPingModule()

	args := map[string]interface{}{}

	taskName := module.getTaskName(args)
	if taskName != "ping" {
		t.Errorf("Expected default task name 'ping', got %s", taskName)
	}
}

func TestPingModule_GetTaskName_InvalidType(t *testing.T) {
	module := NewPingModule()

	args := map[string]interface{}{
		"name": 123, // Invalid type
	}

	taskName := module.getTaskName(args)
	if taskName != "ping" {
		t.Errorf("Expected default task name 'ping' when type is invalid, got %s", taskName)
	}
}

// TestPingModule_Execute_RemoteHost documents that SSH connection testing
// requires a real SSH server and is better suited for integration tests
func TestPingModule_Execute_RemoteHost(t *testing.T) {
	t.Skip("SSH connection testing requires a real SSH server - use integration tests")

	// This test would require:
	// 1. A running SSH server (e.g., Docker container with SSH)
	// 2. Valid SSH credentials
	// 3. Network connectivity
	//
	// Example integration test setup:
	// - Start SSH container: docker run -d -p 2222:22 linuxserver/openssh-server
	// - Configure host with SSH details
	// - Test both successful and failed connections
	// - Verify error handling for connection failures
	//
	// Coverage for checkSSHConnection() and remote connection paths
	// should be measured in integration test suite
}
