package modules

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// SlurpModule reads a file from the host, base64-encoded (Ansible's slurp)
type SlurpModule struct {
	*BaseModule
}

// NewSlurpModule creates the slurp module
func NewSlurpModule() *SlurpModule {
	return &SlurpModule{BaseModule: NewBaseModule("slurp")}
}

func (m *SlurpModule) GetDescription() string { return "Read a file from the host (base64)" }

func (m *SlurpModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "src", getStringArg(args, "path", "")) == "" {
		return fmt.Errorf("slurp requires 'src'")
	}
	return nil
}

func (m *SlurpModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name,
		Timestamp: start, Success: true, Output: map[string]interface{}{}}
	src := getStringArg(args, "src", getStringArg(args, "path", ""))
	data, exists, err := readHostFile(ctx, host, args, src)
	switch {
	case err != nil:
		result.Success, result.Error = false, err.Error()
	case !exists:
		result.Success, result.Error = false, fmt.Sprintf("file not found: %s", src)
	default:
		// armor: false (ansible-core 2.21) gives the text itself
		if getBoolArg(args, "armor", true) {
			result.Output["content"] = base64.StdEncoding.EncodeToString(data)
			result.Output["encoding"] = "base64"
		} else {
			result.Output["content"] = string(data)
			result.Output["encoding"] = "none"
		}
		result.Output["source"] = src
	}
	result.Duration = time.Since(start)
	return result, nil
}
