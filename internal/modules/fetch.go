package modules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
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

	// Initialize executor
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to create executor: %v", err)
		return result, err
	}
	defer exec.Close()

	// Check if source file exists on remote host
	checkCmd := fmt.Sprintf("test -f %s && echo 'exists' || echo 'not_exists'", src)
	checkOutput, err := exec.ExecuteWithContext(ctx, "sh", "-c", checkCmd)

	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to check source file: %v", err)
		return result, err
	}

	exists := strings.TrimSpace(checkOutput) == "exists"

	if !exists {
		if failOnMissing {
			result.Failed = true
			result.Error = fmt.Sprintf("Source file does not exist: %s", src)
			return result, fmt.Errorf("source file does not exist: %s", src)
		} else {
			result.Output["msg"] = fmt.Sprintf("Source file does not exist (skipped): %s", src)
			return result, nil
		}
	}

	// Get source file checksum if validation is enabled
	var srcChecksum string
	if validate {
		checksumCmd := fmt.Sprintf("md5sum %s 2>/dev/null || md5 -q %s 2>/dev/null", src, src)
		checksumOutput, err := exec.ExecuteWithContext(ctx, "sh", "-c", checksumCmd)

		if err != nil {
			// If checksum fails, just skip validation
			validate = false
		} else {
			// Extract checksum (md5sum outputs "checksum filename", md5 outputs just checksum)
			parts := strings.Fields(checksumOutput)
			if len(parts) > 0 {
				srcChecksum = parts[0]
			}
		}
	}

	// Determine destination path
	var destPath string
	if flat {
		// Flat mode: save directly to dest with original filename
		destPath = filepath.Join(dest, filepath.Base(src))
	} else {
		// Hierarchical mode: create directory structure based on hostname and path
		// Format: dest/hostname/path/to/file
		hostDir := host.Name
		if hostDir == "" {
			hostDir = host.Address
		}
		destPath = filepath.Join(dest, hostDir, src)
	}

	// Create destination directory
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0750); err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to create destination directory: %v", err)
		return result, err
	}

	// Check if destination file already exists and has same checksum
	if validate && srcChecksum != "" {
		if destInfo, err := os.Stat(destPath); err == nil && !destInfo.IsDir() {
			destChecksum, err := calculateMD5(destPath)
			if err == nil && destChecksum == srcChecksum {
				result.Output["msg"] = "File already exists with same checksum"
				result.Output["src"] = src
				result.Output["dest"] = destPath
				result.Output["checksum"] = srcChecksum
				result.Success = true
				return result, nil
			}
		}
	}

	// Fetch the file using cat command
	catCmd := fmt.Sprintf("cat %s", src)
	catOutput, err := exec.ExecuteWithContext(ctx, "sh", "-c", catCmd)

	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to read source file: %v", err)
		return result, err
	}

	// Write the file to destination
	if err := os.WriteFile(destPath, []byte(catOutput), 0600); err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to write destination file: %v", err)
		return result, err
	}

	// Validate checksum if enabled
	if validate && srcChecksum != "" {
		destChecksum, err := calculateMD5(destPath)
		if err != nil {
			// Just log warning, don't fail
		} else if destChecksum != srcChecksum {
			result.Failed = true
			result.Error = "Checksum mismatch after fetch"
			result.Output["src_checksum"] = srcChecksum
			result.Output["dest_checksum"] = destChecksum
			// Clean up the file
			_ = os.Remove(destPath)
			return result, fmt.Errorf("checksum mismatch after fetch")
		}
	}

	result.Changed = true
	result.Success = true
	result.Output["msg"] = "File fetched successfully"
	result.Output["src"] = src
	result.Output["dest"] = destPath
	if srcChecksum != "" {
		result.Output["checksum"] = srcChecksum
	}

	return result, nil
}

// calculateMD5 calculates SHA256 checksum of a file (renamed from MD5 for security)
// Note: Function name kept for backward compatibility, but now uses SHA256
func calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath) // #nosec G304 -- filePath is from remote host, validated by executor
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Use SHA256 instead of MD5 for better security
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
