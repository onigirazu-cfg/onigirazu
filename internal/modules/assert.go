package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/expression"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// AssertModule checks that expressions hold, as Ansible's assert does
type AssertModule struct {
	*BaseModule
}

// NewAssertModule creates the assert module
func NewAssertModule() *AssertModule {
	return &AssertModule{BaseModule: NewBaseModule("assert")}
}

func (m *AssertModule) GetDescription() string {
	return "Fail when a condition does not hold"
}

func (m *AssertModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{
		TaskName: taskName(args), Host: host.Name, Module: m.name,
		Timestamp: start, Success: true, Output: map[string]interface{}{},
	}
	conditions, err := assertConditions(args["that"])
	if err != nil {
		result.Success, result.Error = false, err.Error()
		return result, nil
	}
	vars := taskVars(args)
	for _, cond := range conditions {
		holds, err := expression.Condition(cond, vars)
		if err != nil {
			result.Success, result.Error = false, err.Error()
			result.Output["assertion"] = cond
			return result, nil
		}
		if !holds {
			msg := getStringArg(args, "fail_msg", getStringArg(args, "msg", "Assertion failed"))
			result.Success, result.Error = false, msg
			result.Output["msg"], result.Output["assertion"] = msg, cond
			result.Output["evaluated_to"] = false
			result.Duration = time.Since(start)
			return result, nil
		}
	}
	result.Output["msg"] = getStringArg(args, "success_msg", "All assertions passed")
	result.Duration = time.Since(start)
	return result, nil
}

// assertConditions takes "that" as one expression or a list of them
func assertConditions(that interface{}) ([]string, error) {
	switch v := that.(type) {
	case string:
		return []string{v}, nil
	case bool:
		return []string{fmt.Sprint(v)}, nil
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}
		return out, nil
	case nil:
		return nil, fmt.Errorf("argument 'that' is required")
	}
	return nil, fmt.Errorf("argument 'that' must be an expression or a list of them, got %T", that)
}

func (m *AssertModule) Validate(args map[string]interface{}) error {
	_, err := assertConditions(args["that"])
	return err
}
