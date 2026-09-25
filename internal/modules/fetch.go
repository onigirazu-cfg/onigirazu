package modules

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// FetchModule implements file fetching from remote hosts
type FetchModule struct {
	BaseModule
}

// NewFetchModule creates a new fetch module instance
func NewFetchModule() *FetchModule {
	return &FetchModule{
		BaseModule: BaseModule{
			name:        "fetch",
			description: "Fetch files from remote hosts to local machine",
		},
	}
}

// GetDescription returns the module description
func (m *FetchModule) GetDescription() string {
	return m.description
}

// Validate validates the module arguments
func (m *FetchModule) Validate(args map[string]interface{}) error {
	// Required: src
	if _, ok := args["src"]; !ok {
		return fmt.Errorf("src parameter is required")
	}

	// Required: dest
	if _, ok := args["dest"]; !ok {
		return fmt.Errorf("dest parameter is required")
	}

	return nil
}

// Execute executes the fetch module
func (m *FetchModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	result := types.TaskResult{
		Host:      host.Name,
		Module:    m.name,
		Timestamp: time.Now(),
		Output:    make(map[string]interface{}),
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// Validate arguments
	if err := m.Validate(args); err != nil {
		result.Failed = true
		result.Error = err.Error()
		return result, err
	}

	// Get parameters
	src, _ := args["src"].(string)
	dest, _ := args["dest"].(string)
	flat := getBoolArg(args, "flat", false)
	failOnMissing := getBoolArg(args, "fail_on_missing", true)
	validate := getBoolArg(args, "validate", true)

	// Read on the host (base64, with become): root-only and binary files work
	data, exists, err := readHostFile(ctx, host, args, src)
	if err != nil {
		result.Failed = true
		result.Error = err.Error()
		return result, err
	}
	if !exists {
		if failOnMissing {
			result.Failed = true
			result.Error = fmt.Sprintf("Source file does not exist: %s", src)
			return result, fmt.Errorf("source file does not exist: %s", src)
		}
		result.Success = true
		result.Output["msg"] = fmt.Sprintf("Source file does not exist (skipped): %s", src)
		return result, nil
	}
	srcChecksum := fmt.Sprintf("%x", sha256.Sum256(data))

	// Destination on the control machine
	var destPath string
	if flat {
		destPath = filepath.Join(dest, filepath.Base(src))
		if !strings.HasSuffix(dest, "/") {
			destPath = dest
		}
	} else {
		destPath = filepath.Join(dest, host.Name, src)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to create destination directory: %v", err)
		return result, err
	}

	result.Success = true
	result.Output["src"] = src
	result.Output["dest"] = destPath
	result.Output["checksum"] = srcChecksum

	if current, err := os.ReadFile(destPath); err == nil && fmt.Sprintf("%x", sha256.Sum256(current)) == srcChecksum { // #nosec G304 -- destination chosen by the playbook
		result.Output["msg"] = "File already exists with same checksum"
		return result, nil
	}
	if err := os.WriteFile(destPath, data, 0600); err != nil {
		result.Failed = true
		result.Success = false
		result.Error = fmt.Sprintf("Failed to write destination file: %v", err)
		return result, err
	}
	if validate {
		written, err := os.ReadFile(destPath) // #nosec G304 -- just written
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(written)) != srcChecksum {
			_ = os.Remove(destPath)
			result.Failed = true
			result.Success = false
			result.Error = "Checksum mismatch after fetch"
			return result, fmt.Errorf("checksum mismatch after fetch")
		}
	}
	result.Changed = true
	result.Output["msg"] = "File fetched successfully"
	return result, nil
}
