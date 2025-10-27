package modules

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// PauseModule pauses playbook execution for a specified time or until user input
type PauseModule struct {
	*BaseModule
}

// NewPauseModule creates a new pause module
func NewPauseModule() *PauseModule {
	return &PauseModule{
		BaseModule: NewBaseModule("pause"),
	}
}

func (m *PauseModule) GetDescription() string {
	return "Pause playbook execution for a specified duration or until user input"
}

func (m *PauseModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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

	// Get pause parameters
	seconds := 0
	if secsVal, exists := args["seconds"]; exists {
		if secsInt, ok := secsVal.(float64); ok {
			seconds = int(secsInt)
		} else if secsStr, ok := secsVal.(string); ok {
			fmt.Sscanf(secsStr, "%d", &seconds)
		}
	}

	minutes := 0
	if minsVal, exists := args["minutes"]; exists {
		if minsInt, ok := minsVal.(float64); ok {
			minutes = int(minsInt)
		} else if minsStr, ok := minsVal.(string); ok {
			fmt.Sscanf(minsStr, "%d", &minutes)
		}
	}

	// Convert minutes to seconds
	totalSeconds := seconds + (minutes * 60)

	prompt := ""
	if promptVal, exists := args["prompt"]; exists {
		if promptStr, ok := promptVal.(string); ok {
			prompt = promptStr
		}
	}

	// If there's a prompt, wait for user input
	if prompt != "" {
		fmt.Print(prompt)
		if len(prompt) > 0 && prompt[len(prompt)-1:] != "\n" && prompt[len(prompt)-1:] != " " {
			fmt.Print("\n")
		}
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			result.Output["user_input"] = scanner.Text()
		}
	} else if totalSeconds > 0 {
		// Sleep for the specified duration
		result.Output["seconds"] = totalSeconds
		result.Output["msg"] = fmt.Sprintf("Pausing for %d seconds", totalSeconds)
		time.Sleep(time.Duration(totalSeconds) * time.Second)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *PauseModule) Validate(args map[string]interface{}) error {
	return m.BaseModule.Validate(args)
}
