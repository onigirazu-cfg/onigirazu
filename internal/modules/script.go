package modules

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ScriptModule executes a local script on the remote host
type ScriptModule struct {
	*BaseModule
}

// NewScriptModule creates a new script module
func NewScriptModule() *ScriptModule {
	return &ScriptModule{
		BaseModule: NewBaseModule("script"),
	}
}

func (m *ScriptModule) GetDescription() string {
	return "Execute a local script on the remote host"
}

func (m *ScriptModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   true,
		Changed:   true,
		Output:    make(map[string]interface{}),
	}

	// Get script path
	scriptPath := ""
	if scriptVal, exists := args["script"]; exists {
		if scriptStr, ok := scriptVal.(string); ok {
			scriptPath = scriptStr
		}
	}

	if scriptPath == "" {
		result.Success = false
		result.Error = "'script' parameter is required"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Make path absolute if relative
	if !filepath.IsAbs(scriptPath) {
		abs, err := filepath.Abs(scriptPath)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to resolve script path: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		scriptPath = abs
	}

	// Check if script exists
	if _, err := os.Stat(scriptPath); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("script not found: %s", scriptPath)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Get arguments
	argsStr := ""
	if argsVal, exists := args["args"]; exists {
		if argsStr2, ok := argsVal.(string); ok {
			argsStr = argsStr2
		}
	}

	// Prepare command
	var cmd *exec.Cmd
	if argsStr != "" {
		// Parse arguments
		cmdArgs := strings.Fields(argsStr)
		cmd = exec.CommandContext(ctx, "bash", append([]string{scriptPath}, cmdArgs...)...)
	} else {
		cmd = exec.CommandContext(ctx, "bash", scriptPath)
	}

	// Capture output
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = io.MultiWriter(&stdout)
	cmd.Stderr = io.MultiWriter(&stderr)

	// Execute script
	err := cmd.Run()

	// Get return code
	rc := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			rc = exitErr.ExitCode()
		}
		result.Success = false
		result.Error = fmt.Sprintf("script execution failed: %v", err)
	}

	result.Output["stdout"] = stdout.String()
	result.Output["stderr"] = stderr.String()
	result.Output["rc"] = rc
	result.Output["args"] = argsStr

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *ScriptModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	// Check that script parameter is provided
	if _, exists := args["script"]; !exists {
		return fmt.Errorf("script module requires 'script' parameter")
	}

	return nil
}
