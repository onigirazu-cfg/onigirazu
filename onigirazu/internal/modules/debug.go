package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// DebugModule prints debug messages
type DebugModule struct {
	*BaseModule
}

// NewDebugModule creates a new debug module
func NewDebugModule() *DebugModule {
	return &DebugModule{
		BaseModule: NewBaseModule("debug"),
	}
}

func (m *DebugModule) GetDescription() string {
	return "Prints debug messages"
}

func (m *DebugModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
	}

	// Get the message to print
	msg := ""
	if msgVal, exists := args["msg"]; exists {
		if msgStr, ok := msgVal.(string); ok {
			msg = msgStr
		} else {
			msg = fmt.Sprintf("%v", msgVal)
		}
	} else if varVal, exists := args["var"]; exists {
		// Support for var parameter (print variable)
		if varStr, ok := varVal.(string); ok {
			msg = fmt.Sprintf("%s: %v", varStr, varVal)
		} else {
			msg = fmt.Sprintf("%v", varVal)
		}
	} else {
		result.Success = false
		result.Error = "debug module requires 'msg' or 'var' parameter"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Store the message in output
	result.Output["msg"] = msg

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *DebugModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	// Check that either msg or var is provided
	_, hasMsg := args["msg"]
	_, hasVar := args["var"]

	if !hasMsg && !hasVar {
		return fmt.Errorf("debug module requires either 'msg' or 'var' parameter")
	}

	return nil
}
