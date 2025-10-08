//go:build plugin
// +build plugin

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/plugins"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// HelloModule is a simple example module plugin
type HelloModule struct {
	*plugins.BaseModulePlugin
}

// NewPlugin is the entry point for the plugin
// This function must be exported and return a Plugin interface
func NewPlugin() plugins.Plugin {
	module := &HelloModule{
		BaseModulePlugin: plugins.NewBaseModulePlugin(
			"hello",
			"1.0.0",
			"A simple hello world module plugin",
		),
	}

	// Define required and optional arguments
	module.AddRequiredArg("message")
	module.AddOptionalArg("uppercase", false)
	module.AddOptionalArg("repeat", 1)

	return module
}

// Execute executes the hello module
func (m *HelloModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	// Create result
	result := types.TaskResult{
		TaskName:  plugins.GetStringArg(args, "name", "hello"),
		Host:      host.Name,
		Module:    m.GetName(),
		Timestamp: startTime,
	}

	// Validate arguments
	if err := m.Validate(args); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Get arguments
	message := plugins.GetStringArg(args, "message", "")
	uppercase := plugins.GetBoolArg(args, "uppercase", false)
	repeat := plugins.GetIntArg(args, "repeat", 1)

	// Process message
	if uppercase {
		message = fmt.Sprintf("%s", message)
		// Convert to uppercase
		runes := []rune(message)
		for i, r := range runes {
			if r >= 'a' && r <= 'z' {
				runes[i] = r - 32
			}
		}
		message = string(runes)
	}

	// Repeat message
	var finalMessage string
	for i := 0; i < repeat; i++ {
		if i > 0 {
			finalMessage += " "
		}
		finalMessage += message
	}

	// Set result
	result.Success = true
	result.Changed = false
	result.Output = map[string]interface{}{
		"message":   finalMessage,
		"uppercase": uppercase,
		"repeat":    repeat,
	}
	result.Duration = time.Since(startTime)

	return result, nil
}

// Validate validates the module arguments
func (m *HelloModule) Validate(args map[string]interface{}) error {
	// Call base validation
	if err := m.BaseModulePlugin.Validate(args); err != nil {
		return err
	}

	// Additional validation
	message := plugins.GetStringArg(args, "message", "")
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	repeat := plugins.GetIntArg(args, "repeat", 1)
	if repeat < 1 || repeat > 100 {
		return fmt.Errorf("repeat must be between 1 and 100")
	}

	return nil
}
