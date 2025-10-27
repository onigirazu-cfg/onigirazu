package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// FailModule fails the playbook with a custom message
type FailModule struct {
	*BaseModule
}

// NewFailModule creates a new fail module
func NewFailModule() *FailModule {
	return &FailModule{
		BaseModule: NewBaseModule("fail"),
	}
}

func (m *FailModule) GetDescription() string {
	return "Fail playbook execution with a custom message"
}

func (m *FailModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   false,
		Changed:   false,
		Output:    make(map[string]interface{}),
	}

	// Get the failure message
	msg := "Failed as requested"
	if msgVal, exists := args["msg"]; exists {
		if msgStr, ok := msgVal.(string); ok {
			msg = msgStr
		} else {
			msg = fmt.Sprintf("%v", msgVal)
		}
	}

	result.Error = msg
	result.Output["msg"] = msg

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *FailModule) Validate(args map[string]interface{}) error {
	return m.BaseModule.Validate(args)
}
