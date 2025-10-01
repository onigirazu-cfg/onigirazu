package executor

import (
	"context"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNewCommandExecutor_Local(t *testing.T) {
	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	executor, err := NewCommandExecutor(host)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	defer executor.Close()

	if executor.IsRemote() {
		t.Error("Expected local executor, got remote")
	}

	if executor.sshClient != nil {
		t.Error("Expected no SSH client for local executor")
	}
}

func TestNewCommandExecutor_WithoutPool(t *testing.T) {
	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	executor, err := NewCommandExecutorWithoutPool(host)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	defer executor.Close()

	if executor.usePool {
		t.Error("Expected usePool to be false")
	}
}

func TestCommandExecutor_ExecuteLocal(t *testing.T) {
	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	executor, err := NewCommandExecutor(host)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	defer executor.Close()

	// Test simple command
	output, err := executor.Execute("echo", "hello")
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	if output != "hello\n" {
		t.Errorf("Expected 'hello\\n', got '%s'", output)
	}
}

func TestCommandExecutor_ExecuteWithContext(t *testing.T) {
	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	executor, err := NewCommandExecutor(host)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	defer executor.Close()

	ctx := context.Background()
	output, err := executor.ExecuteWithContext(ctx, "echo", "test")
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	if output != "test\n" {
		t.Errorf("Expected 'test\\n', got '%s'", output)
	}
}

func TestCommandExecutor_ExecuteWithTimeout(t *testing.T) {
	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	executor, err := NewCommandExecutor(host)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	defer executor.Close()

	// Test command that completes within timeout
	output, err := executor.ExecuteWithTimeout("echo", 5*time.Second, "timeout-test")
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	if output != "timeout-test\n" {
		t.Errorf("Expected 'timeout-test\\n', got '%s'", output)
	}
}

func TestCommandExecutor_ExecuteWithTimeout_Exceeded(t *testing.T) {
	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	executor, err := NewCommandExecutor(host)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}
	defer executor.Close()

	// Test command that exceeds timeout
	_, err = executor.ExecuteWithTimeout("sleep", 100*time.Millisecond, "10")
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestCommandExecutor_Close_Multiple(t *testing.T) {
	host := types.Host{
		Name:    "localhost",
		Address: "localhost",
	}

	executor, err := NewCommandExecutor(host)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Close multiple times should not panic
	err = executor.Close()
	if err != nil {
		t.Errorf("First close failed: %v", err)
	}

	err = executor.Close()
	if err != nil {
		t.Errorf("Second close failed: %v", err)
	}
}

func TestCommandExecutor_IsRemote(t *testing.T) {
	tests := []struct {
		name     string
		host     types.Host
		expected bool
	}{
		{
			name: "localhost",
			host: types.Host{
				Name:    "localhost",
				Address: "localhost",
			},
			expected: false,
		},
		{
			name: "127.0.0.1",
			host: types.Host{
				Name:    "local",
				Address: "127.0.0.1",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := NewCommandExecutor(tt.host)
			if err != nil {
				t.Fatalf("Failed to create executor: %v", err)
			}
			defer executor.Close()

			if executor.IsRemote() != tt.expected {
				t.Errorf("Expected IsRemote()=%v, got %v", tt.expected, executor.IsRemote())
			}
		})
	}
}
