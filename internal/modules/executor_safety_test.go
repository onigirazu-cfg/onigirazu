package modules

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestExecutorSafety verifies that modules don't cache executors
// This test ensures that each host gets its own executor connection
// and commands are executed on the correct target host.
func TestExecutorSafety(t *testing.T) {
	// This test would require actual SSH connections to verify
	// For now, we test the BaseExecutorModule API

	module := NewBaseExecutorModule("test")

	if module == nil {
		t.Fatal("NewBaseExecutorModule returned nil")
	}

	if module.BaseModule == nil {
		t.Fatal("BaseModule is nil")
	}

	if module.GetName() != "test" {
		t.Errorf("Expected name 'test', got '%s'", module.GetName())
	}
}

// TestWithExecutorPattern tests the WithExecutor pattern
func TestWithExecutorPattern(t *testing.T) {
	module := NewBaseExecutorModule("test")

	// Create a local host for testing
	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
		Port:    22,
	}

	// Test WithExecutor
	executionCount := 0
	err := module.WithExecutor(host, func(exec *executor.CommandExecutor) error {
		executionCount++
		if exec == nil {
			t.Error("Executor is nil inside WithExecutor callback")
		}
		return nil
	})

	if err != nil {
		t.Errorf("WithExecutor failed: %v", err)
	}

	if executionCount != 1 {
		t.Errorf("Expected execution count 1, got %d", executionCount)
	}
}

// TestWithExecutorResult tests the WithExecutorResult pattern
func TestWithExecutorResult(t *testing.T) {
	module := NewBaseExecutorModule("test")

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
		Port:    22,
	}

	// Test WithExecutorResult
	result, err := module.WithExecutorResult(host, func(exec *executor.CommandExecutor) (string, error) {
		if exec == nil {
			t.Error("Executor is nil inside WithExecutorResult callback")
		}
		return "test-result", nil
	})

	if err != nil {
		t.Errorf("WithExecutorResult failed: %v", err)
	}

	if result != "test-result" {
		t.Errorf("Expected result 'test-result', got '%s'", result)
	}
}

// TestCreateExecutor tests the CreateExecutor method
func TestCreateExecutor(t *testing.T) {
	module := NewBaseExecutorModule("test")

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
		Port:    22,
	}

	exec, err := module.CreateExecutor(host)
	if err != nil {
		t.Errorf("CreateExecutor failed: %v", err)
	}

	if exec == nil {
		t.Fatal("CreateExecutor returned nil executor")
	}

	// Clean up
	defer exec.Close()
}

// TestMultipleHostsPattern demonstrates the correct pattern for multi-host execution
// This test requires actual SSH connectivity and is skipped in CI
func TestMultipleHostsPattern(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Skip this test as it requires actual SSH connectivity to multiple hosts
	t.Skip("This test requires actual SSH connectivity to multiple hosts - run manually with proper setup")

	module := NewBaseExecutorModule("test")

	hosts := []types.Host{
		{Name: "host1", Address: "127.0.0.1", Port: 22},
		{Name: "host2", Address: "127.0.0.2", Port: 22},
		{Name: "host3", Address: "127.0.0.3", Port: 22},
	}

	// Simulate executing on multiple hosts
	for _, host := range hosts {
		err := module.WithExecutor(host, func(exec *executor.CommandExecutor) error {
			// Each iteration should get a fresh executor
			// connected to the specific host
			if exec == nil {
				t.Errorf("Executor is nil for host %s", host.Name)
			}
			return nil
		})

		if err != nil {
			t.Errorf("Execution failed for host %s: %v", host.Name, err)
		}
	}
}

// BenchmarkWithExecutor benchmarks the WithExecutor pattern
func BenchmarkWithExecutor(b *testing.B) {
	module := NewBaseExecutorModule("test")
	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
		Port:    22,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.WithExecutor(host, func(exec *executor.CommandExecutor) error {
			return nil
		})
	}
}
