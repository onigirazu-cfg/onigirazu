package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/expression"
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
		TaskName:  taskName(args),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
	}

	// Get the message to print
	// msg keeps its type (a list stays a list, as in Ansible); var= prints
	// "name: value"; values that are not text print as JSON
	var msg interface{}
	if msgVal, exists := args["msg"]; exists {
		msg = msgVal
	} else if varVal, exists := args["var"]; exists {
		// Support for var parameter (print variable)
		if varStr, ok := varVal.(string); ok {
			vars := taskVars(args)
			value, found := lookupVar(vars, varStr)
			if !found {
				// an expression, as in Ansible: result['stdout'], x | length
				if v, err := expression.Eval(varStr, vars); err == nil {
					value, found = v, true
				}
			}
			if !found {
				value = "VARIABLE IS NOT DEFINED!"
			}
			msg = varStr + ": " + debugText(value)
			result.Output[varStr] = value
		} else {
			msg = debugText(varVal)
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

// lookupVar resolves a dotted path ("result.stdout") in vars
func lookupVar(vars map[string]interface{}, path string) (interface{}, bool) {
	var cur interface{} = vars
	for _, key := range strings.Split(path, ".") {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		if cur, ok = m[key]; !ok {
			return nil, false
		}
	}
	return cur, true
}

// debugText prints a value the way Ansible's output shows it: text as is,
// anything else as JSON
func debugText(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	if b, err := json.Marshal(v); err == nil {
		return string(b)
	}
	return fmt.Sprint(v)
}
