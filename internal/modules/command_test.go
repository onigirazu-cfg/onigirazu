package modules

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestCommandModuleCreation(t *testing.T) {
	module := NewCommandModule()
	if module == nil {
		t.Fatal("NewCommandModule() returned nil")
	}

	if module.GetName() != "command" {
		t.Errorf("Expected module name to be 'command', got '%s'", module.GetName())
	}

	if module.GetDescription() != "Executes commands on remote hosts" {
		t.Errorf("Expected description to be 'Executes commands on remote hosts', got '%s'", module.GetDescription())
	}
}

func TestCommandModuleValidation(t *testing.T) {
	module := NewCommandModule()

	// Test missing required arguments
	args := map[string]interface{}{
		"name": "test-task",
	}

	err := module.Validate(args)
	if err == nil {
		t.Error("Expected validation error for missing 'command' argument")
	}

	// Test valid arguments
	args["command"] = "echo hello"
	err = module.Validate(args)
	if err != nil {
		t.Errorf("Expected no validation error, got: %v", err)
	}

	// Test with shell option
	args["shell"] = true
	err = module.Validate(args)
	if err != nil {
		t.Errorf("Expected no validation error with shell option, got: %v", err)
	}
}

func TestCommandModuleExecuteSimple(t *testing.T) {
	module := NewCommandModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	// Test simple echo command
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo hello"
	} else {
		cmd = "echo hello"
	}

	args := map[string]interface{}{
		"name":    "test-echo",
		"command": cmd,
		"shell":   false,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Error("Expected changed to be true for command execution")
	}

	// Check output
	if output, ok := result.Output["stdout"].(string); ok {
		if !strings.Contains(strings.TrimSpace(output), "hello") {
			t.Errorf("Expected output to contain 'hello', got: %s", output)
		}
	} else {
		t.Error("Expected stdout in output")
	}
}

func TestCommandModuleExecuteWithShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell test on Windows")
	}

	module := NewCommandModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	// Test command with shell features
	args := map[string]interface{}{
		"name":    "test-shell",
		"command": "echo $HOME | wc -c",
		"shell":   true,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Error("Expected changed to be true for command execution")
	}
}

func TestCommandModuleExecuteFailure(t *testing.T) {
	module := NewCommandModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	// Test command that should fail
	args := map[string]interface{}{
		"name":    "test-failure",
		"command": "nonexistentcommand12345",
		"shell":   false,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	// The module should return an error for failed commands
	if err == nil {
		t.Error("Expected error from Execute for nonexistent command")
	}

	if result.Success {
		t.Error("Expected failure for nonexistent command")
	}

	if !result.Failed {
		t.Error("Expected Failed flag to be true")
	}

	if result.Error == "" {
		t.Error("Expected error message in result")
	}
}

func TestShellModuleCreation(t *testing.T) {
	module := NewShellModule()
	if module == nil {
		t.Fatal("NewShellModule() returned nil")
	}

	if module.GetName() != "shell" {
		t.Errorf("Expected module name to be 'shell', got '%s'", module.GetName())
	}

	if module.GetDescription() != "Executes shell commands with shell interpretation" {
		t.Errorf("Expected description to be 'Executes shell commands with shell interpretation', got '%s'", module.GetDescription())
	}
}

func TestShellModuleExecute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell test on Windows")
	}

	module := NewShellModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	// Test shell command
	args := map[string]interface{}{
		"name":    "test-shell-module",
		"command": "echo 'hello world' | wc -w",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if !result.Changed {
		t.Error("Expected changed to be true for shell command execution")
	}

	// Check that shell was forced to true
	if output, ok := result.Output["stdout"].(string); ok {
		// The command should count words and return "2"
		if !strings.Contains(strings.TrimSpace(output), "2") {
			t.Errorf("Expected output to contain '2', got: %s", output)
		}
	} else {
		t.Error("Expected stdout in output")
	}
}

func TestShellModuleValidation(t *testing.T) {
	module := NewShellModule()

	// Test missing command
	args := map[string]interface{}{
		"name": "test-task",
	}

	err := module.Validate(args)
	if err == nil {
		t.Error("Expected validation error for missing 'command' argument")
	}

	// Test valid arguments
	args["command"] = "echo hello"
	err = module.Validate(args)
	if err != nil {
		t.Errorf("Expected no validation error, got: %v", err)
	}
}

// Additional tests for better coverage

func TestCommandModule_ValidateWithCmd(t *testing.T) {
	module := NewCommandModule()

	// Test with 'cmd' instead of 'command'
	args := map[string]interface{}{
		"name": "test-task",
		"cmd":  "echo test",
	}

	err := module.Validate(args)
	if err != nil {
		t.Errorf("Expected no validation error with 'cmd', got: %v", err)
	}

	// Verify that 'cmd' was copied to 'command'
	if args["command"] != "echo test" {
		t.Error("Expected 'cmd' to be copied to 'command'")
	}
}

func TestCommandModule_ValidateEmptyCommand(t *testing.T) {
	module := NewCommandModule()

	args := map[string]interface{}{
		"name":    "test-task",
		"command": "   ",
	}

	err := module.Validate(args)
	if err == nil {
		t.Error("Expected validation error for empty command")
	}
}

func TestCommandModule_ValidateInvalidCommandType(t *testing.T) {
	module := NewCommandModule()

	args := map[string]interface{}{
		"name":    "test-task",
		"command": 123,
	}

	err := module.Validate(args)
	if err == nil {
		t.Error("Expected validation error for non-string command")
	}
}

func TestCommandModule_ValidateInvalidShellType(t *testing.T) {
	module := NewCommandModule()

	args := map[string]interface{}{
		"name":    "test-task",
		"command": "echo test",
		"shell":   "yes",
	}

	err := module.Validate(args)
	if err == nil {
		t.Error("Expected validation error for non-boolean shell")
	}
}

func TestShellModule_ValidateWithCmd(t *testing.T) {
	module := NewShellModule()

	// Test with 'cmd' instead of 'command'
	args := map[string]interface{}{
		"cmd": "echo test",
	}

	err := module.Validate(args)
	if err != nil {
		t.Errorf("Expected no validation error with 'cmd', got: %v", err)
	}

	// Verify that 'cmd' was copied to 'command'
	if args["command"] != "echo test" {
		t.Error("Expected 'cmd' to be copied to 'command'")
	}
}

func TestShellModule_ValidateInvalidCommandType(t *testing.T) {
	module := NewShellModule()

	args := map[string]interface{}{
		"command": []string{"echo", "test"},
	}

	err := module.Validate(args)
	if err == nil {
		t.Error("Expected validation error for non-string command")
	}
}

func TestShellModule_ValidateInvalidChdirType(t *testing.T) {
	module := NewShellModule()

	args := map[string]interface{}{
		"command": "echo test",
		"chdir":   123,
	}

	err := module.Validate(args)
	if err == nil {
		t.Error("Expected validation error for non-string chdir")
	}
}

func TestShellModule_ValidateValidChdir(t *testing.T) {
	module := NewShellModule()

	args := map[string]interface{}{
		"command": "pwd",
		"chdir":   "/tmp",
	}

	err := module.Validate(args)
	if err != nil {
		t.Errorf("Expected no validation error with valid chdir, got: %v", err)
	}
}

func TestShellModule_ValidateInvalidEnvironmentType(t *testing.T) {
	module := NewShellModule()

	args := map[string]interface{}{
		"command":     "echo test",
		"environment": "invalid",
	}

	err := module.Validate(args)
	if err == nil {
		t.Error("Expected validation error for non-map environment")
	}
}

func TestShellModule_ValidateInvalidEnvironmentValueType(t *testing.T) {
	module := NewShellModule()

	args := map[string]interface{}{
		"command": "echo test",
		"environment": map[string]interface{}{
			"VAR1": "value1",
			"VAR2": 123,
		},
	}

	err := module.Validate(args)
	if err == nil {
		t.Error("Expected validation error for non-string environment value")
	}
}

func TestShellModule_ValidateValidEnvironment(t *testing.T) {
	module := NewShellModule()

	args := map[string]interface{}{
		"command": "echo $TEST_VAR",
		"environment": map[string]interface{}{
			"TEST_VAR": "test_value",
		},
	}

	err := module.Validate(args)
	if err != nil {
		t.Errorf("Expected no validation error with valid environment, got: %v", err)
	}
}

func TestShellModule_ExecuteWithChdir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell test on Windows")
	}

	module := NewShellModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	args := map[string]interface{}{
		"name":    "test-chdir",
		"command": "pwd",
		"chdir":   "/tmp",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Output should contain /tmp
	if output, ok := result.Output["stdout"].(string); ok {
		if !strings.Contains(output, "/tmp") && !strings.Contains(output, "/private/tmp") {
			t.Errorf("Expected output to contain '/tmp', got: %s", output)
		}
	}
}

func TestShellModule_ExecuteWithEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell test on Windows")
	}

	module := NewShellModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	args := map[string]interface{}{
		"name":    "test-env",
		"command": "printenv TEST_VAR",
		"environment": map[string]interface{}{
			"TEST_VAR": "hello_world",
		},
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Output should contain the environment variable value
	if output, ok := result.Output["stdout"].(string); ok {
		if !strings.Contains(output, "hello_world") {
			t.Errorf("Expected output to contain 'hello_world', got: %s", output)
		}
	}
}

func TestShellModule_ExecuteWithCmdParameter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping shell test on Windows")
	}

	module := NewShellModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	// Use 'cmd' instead of 'command'
	args := map[string]interface{}{
		"name": "test-cmd-param",
		"cmd":  "echo test_cmd",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if output, ok := result.Output["stdout"].(string); ok {
		if !strings.Contains(output, "test_cmd") {
			t.Errorf("Expected output to contain 'test_cmd', got: %s", output)
		}
	}
}

func TestShellModule_IsIdempotent(t *testing.T) {
	module := NewShellModule()

	if module.IsIdempotent() {
		t.Error("Shell module should not be idempotent")
	}
}

func TestCommandModule_ExecuteWithShellFailure(t *testing.T) {
	module := NewCommandModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	args := map[string]interface{}{
		"name":    "test-shell-failure",
		"command": "nonexistentcommand12345",
		"shell":   true,
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err == nil {
		t.Error("Expected error from Execute for nonexistent command with shell")
	}

	if result.Success {
		t.Error("Expected failure for nonexistent command with shell")
	}

	if !result.Failed {
		t.Error("Expected Failed flag to be true")
	}
}

func TestShellModule_ExecuteFailure(t *testing.T) {
	module := NewShellModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	args := map[string]interface{}{
		"name":    "test-shell-failure",
		"command": "exit 1",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err == nil {
		t.Error("Expected error from Execute for failing command")
	}

	if result.Success {
		t.Error("Expected failure for command that exits with error")
	}
}
