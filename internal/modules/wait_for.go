package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// WaitForModule waits for a specific condition to be met
type WaitForModule struct {
	*BaseModule
}

// NewWaitForModule creates a new wait_for module
func NewWaitForModule() *WaitForModule {
	return &WaitForModule{
		BaseModule: NewBaseModule("wait_for"),
	}
}

func (m *WaitForModule) GetDescription() string {
	return "Wait for a specific condition to be met before continuing"
}

func (m *WaitForModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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

	timeout := getIntArg(args, "timeout", 300)
	delay := getIntArg(args, "delay", 0)
	port := getIntArg(args, "port", 0)
	path := getStringArg(args, "path", "")
	searchRegex := getStringArg(args, "search_regex", "")
	target := getStringArg(args, "host", "127.0.0.1")
	state := getStringArg(args, "state", "started")
	// started/present and stopped/absent are the same states
	wantPresent := state == "started" || state == "present"
	if !wantPresent && state != "stopped" && state != "absent" {
		result.Success = false
		result.Error = fmt.Sprintf("unsupported state: %s", state)
		return result, nil
	}

	// The check runs on the target host, as the task describes that host
	var check string
	switch {
	case port > 0:
		check = fmt.Sprintf("timeout 2 bash -c %s", shellQuote(fmt.Sprintf("</dev/tcp/%s/%d", target, port)))
	case path != "" && searchRegex != "":
		check = fmt.Sprintf("grep -Eq %s %s", shellQuote(searchRegex), shellQuote(path))
	case path != "":
		check = "test -e " + shellQuote(path)
	default:
		result.Success = false
		result.Error = "wait_for needs port or path"
		return result, nil
	}

	sleep := func(d time.Duration) bool {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(d):
			return true
		}
	}
	if delay > 0 && !sleep(time.Duration(delay)*time.Second) {
		result.Success = false
		result.Error = "canceled"
		return result, nil
	}

	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		_, err := runShellOnHost(ctx, host, args, check)
		if (err == nil) == wantPresent {
			result.Output["elapsed"] = time.Since(startTime).Seconds()
			result.Output["msg"] = "Condition met"
			result.Duration = time.Since(startTime)
			return result, nil
		}
		if time.Now().After(deadline) {
			result.Success = false
			result.Error = fmt.Sprintf("timeout: condition not met after %d seconds", timeout)
			result.Output["elapsed"] = time.Since(startTime).Seconds()
			result.Duration = time.Since(startTime)
			return result, nil
		}
		if !sleep(time.Second) {
			result.Success = false
			result.Error = "canceled"
			return result, nil
		}
	}
}

func (m *WaitForModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	// Check that at least one condition is specified
	_, hasPort := args["port"]
	_, hasPath := args["path"]

	if !hasPort && !hasPath {
		return fmt.Errorf("wait_for module requires either 'port' or 'path' parameter")
	}

	// Validate state parameter if provided
	if stateVal, exists := args["state"]; exists {
		if stateStr, ok := stateVal.(string); ok {
			validStates := map[string]bool{
				"started": true,
				"stopped": true,
				"present": true,
				"absent":  true,
				"drained": true,
			}
			if !validStates[stateStr] {
				return fmt.Errorf("invalid state '%s', must be one of: started, stopped, present, absent, drained", stateStr)
			}
		}
	}

	return nil
}
