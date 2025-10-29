package modules

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
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
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
	}

	// Get parameters
	timeout := 300 // default 5 minutes
	if timeoutVal, exists := args["timeout"]; exists {
		if timeoutInt, ok := timeoutVal.(float64); ok {
			timeout = int(timeoutInt)
		}
	}

	delay := 0
	if delayVal, exists := args["delay"]; exists {
		if delayInt, ok := delayVal.(float64); ok {
			delay = int(delayInt)
		}
	}

	state := "started"
	if stateVal, exists := args["state"]; exists {
		if stateStr, ok := stateVal.(string); ok {
			state = stateStr
		}
	}

	port := 0
	if portVal, exists := args["port"]; exists {
		if portInt, ok := portVal.(float64); ok {
			port = int(portInt)
		}
	}

	path := ""
	if pathVal, exists := args["path"]; exists {
		if pathStr, ok := pathVal.(string); ok {
			path = pathStr
		}
	}

	searchRegex := ""
	if searchVal, exists := args["search_regex"]; exists {
		if searchStr, ok := searchVal.(string); ok {
			searchRegex = searchStr
		}
	}

	hostVal := "localhost"
	if hostVal2, exists := args["host"]; exists {
		if hostStr, ok := hostVal2.(string); ok {
			hostVal = hostStr
		}
	}

	// Initial delay
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Second)
	}

	// Start waiting
	elapsed := 0.0
	timeoutDuration := time.Duration(timeout) * time.Second
	checkInterval := 100 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			result.Success = false
			result.Error = "context canceled"
			result.Duration = time.Since(startTime)
			return result, nil
		default:
		}

		// Check condition based on parameters
		conditionMet := false
		var err error

		if port > 0 {
			conditionMet, err = m.checkPort(hostVal, port, state)
		} else if path != "" && searchRegex == "" {
			conditionMet, err = m.checkPath(path, state)
		} else if path != "" && searchRegex != "" {
			conditionMet, err = m.checkPathRegex(path, searchRegex)
		}

		if err != nil && state != "absent" {
			// Continue waiting if not absent state
		}

		if conditionMet {
			result.Output["elapsed"] = elapsed
			result.Output["msg"] = "Condition met"
			result.Duration = time.Since(startTime)
			return result, nil
		}

		// Check timeout
		if time.Since(startTime) > timeoutDuration {
			result.Success = false
			result.Error = fmt.Sprintf("timeout: condition not met after %d seconds", timeout)
			result.Output["elapsed"] = time.Since(startTime).Seconds()
			result.Duration = time.Since(startTime)
			return result, nil
		}

		time.Sleep(checkInterval)
		elapsed = time.Since(startTime).Seconds()
	}
}

func (m *WaitForModule) checkPort(host string, port int, state string) (bool, error) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.Dial("tcp", addr)

	if state == "started" {
		if err == nil {
			conn.Close()
			return true, nil
		}
		return false, err
	} else if state == "stopped" {
		if err != nil {
			return true, nil
		}
		if conn != nil {
			conn.Close()
		}
		return false, nil
	}

	return false, nil
}

func (m *WaitForModule) checkPath(path string, state string) (bool, error) {
	// Make path absolute
	if !filepath.IsAbs(path) {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return false, err
		}
	}

	_, err := os.Stat(path)

	if state == "present" {
		return err == nil, nil
	} else if state == "absent" {
		return os.IsNotExist(err), nil
	}

	return false, nil
}

func (m *WaitForModule) checkPathRegex(path string, searchRegex string) (bool, error) {
	// Make path absolute
	if !filepath.IsAbs(path) {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return false, err
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	regex, err := regexp.Compile(searchRegex)
	if err != nil {
		return false, fmt.Errorf("invalid regex: %v", err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if regex.MatchString(scanner.Text()) {
			return true, nil
		}
	}

	return false, scanner.Err()
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
