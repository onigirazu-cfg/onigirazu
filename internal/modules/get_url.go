package modules

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// GetURLModule implements file downloading from URLs
type GetURLModule struct {
	BaseModule
}

// NewGetURLModule creates a new get_url module instance
func NewGetURLModule() *GetURLModule {
	return &GetURLModule{
		BaseModule: BaseModule{
			name:        "get_url",
			description: "Download files from HTTP, HTTPS, or FTP URLs",
		},
	}
}

// GetDescription returns the module description
func (m *GetURLModule) GetDescription() string {
	return m.description
}

// Validate validates the module arguments
func (m *GetURLModule) Validate(args map[string]interface{}) error {
	// Required: url
	if _, ok := args["url"]; !ok {
		return fmt.Errorf("url parameter is required")
	}

	// Required: dest
	if _, ok := args["dest"]; !ok {
		return fmt.Errorf("dest parameter is required")
	}

	return nil
}

// Execute executes the get_url module
func (m *GetURLModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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
	url, _ := args["url"].(string)
	dest, _ := args["dest"].(string)
	force := getBoolArg(args, "force", false)
	backup := getBoolArg(args, "backup", false)
	checksum := getStringArg(args, "checksum", "")
	mode := getStringArg(args, "mode", "0644")
	owner := getStringArg(args, "owner", "")
	group := getStringArg(args, "group", "")
	timeout := getIntArg(args, "timeout", 30)
	headers := make(map[string]string)
	if headersVal, ok := args["headers"].(map[string]interface{}); ok {
		for k, v := range headersVal {
			if strVal, ok := v.(string); ok {
				headers[k] = strVal
			}
		}
	}

	// Initialize executor
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to create executor: %v", err)
		return result, err
	}
	defer exec.Close()

	// Check if destination exists
	checkCmd := fmt.Sprintf("test -f %s && echo 'exists' || echo 'not_exists'", dest)
	checkOutput, _ := exec.ExecuteWithContext(ctx, "sh", "-c", checkCmd)
	destExists := strings.TrimSpace(checkOutput) == "exists"

	// If file exists and force is false, check if we need to download
	if destExists && !force {
		// Check if checksum matches (if provided)
		if checksum != "" {
			destChecksum, err := m.getRemoteFileChecksum(ctx, exec, dest, checksum)
			if err == nil && m.checksumMatches(checksum, destChecksum) {
				result.Output["msg"] = "File already exists with matching checksum"
				result.Output["dest"] = dest
				result.Output["url"] = url
				result.Success = true
				return result, nil
			}
		} else {
			// File exists and no checksum provided, skip download
			result.Output["msg"] = "File already exists (use force=true to overwrite)"
			result.Output["dest"] = dest
			result.Output["url"] = url
			result.Success = true
			return result, nil
		}
	}

	// Create backup if requested and file exists
	if backup && destExists {
		backupCmd := fmt.Sprintf("cp %s %s.%d.backup", dest, dest, time.Now().Unix())
		_, err := exec.ExecuteWithContext(ctx, "sh", "-c", backupCmd)
		if err != nil {
			result.Output["warning"] = fmt.Sprintf("Failed to create backup: %v", err)
		} else {
			result.Output["backup_file"] = fmt.Sprintf("%s.%d.backup", dest, time.Now().Unix())
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		return result, err
	}

	// Add custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to download file: %v", err)
		return result, err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		result.Failed = true
		result.Error = fmt.Sprintf("HTTP request failed with status: %d %s", resp.StatusCode, resp.Status)
		return result, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	// Create temporary local file
	localTmpFile, err := os.CreateTemp("", "onigirazu-download-*")
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to create temporary file: %v", err)
		return result, err
	}
	defer os.Remove(localTmpFile.Name())
	defer localTmpFile.Close()

	// Download and calculate checksum if needed
	var hasher hash.Hash
	if checksum != "" {
		checksumType := strings.Split(checksum, ":")[0]
		switch checksumType {
		case "md5":
			hasher = md5.New()
		case "sha1":
			hasher = md5.New()
		case "sha256":
			hasher = sha256.New()
		}
	}

	var writer io.Writer = localTmpFile
	if hasher != nil {
		writer = io.MultiWriter(localTmpFile, hasher)
	}

	// Copy response body to file
	bytesWritten, err := io.Copy(writer, resp.Body)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to write file: %v", err)
		return result, err
	}

	localTmpFile.Close()

	// Verify checksum if provided
	if checksum != "" && hasher != nil {
		downloadedChecksum := hex.EncodeToString(hasher.Sum(nil))
		if !m.checksumMatches(checksum, downloadedChecksum) {
			result.Failed = true
			result.Error = fmt.Sprintf("Checksum mismatch: expected %s, got %s", checksum, downloadedChecksum)
			return result, fmt.Errorf("checksum mismatch")
		}
		result.Output["checksum"] = downloadedChecksum
	}

	// Upload file to remote host
	fileContent, err := os.ReadFile(localTmpFile.Name())
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to read downloaded file: %v", err)
		return result, err
	}

	// Create destination directory if needed
	destDir := filepath.Dir(dest)
	mkdirCmd := fmt.Sprintf("mkdir -p %s", destDir)
	_, err = exec.ExecuteWithContext(ctx, "sh", "-c", mkdirCmd)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to create destination directory: %v", err)
		return result, err
	}

	// Write file to remote host using base64 encoding to handle binary files
	writeCmd := fmt.Sprintf("echo '%s' | base64 -d > %s", encodeBase64(fileContent), dest)
	_, err = exec.ExecuteWithContext(ctx, "sh", "-c", writeCmd)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to write file to remote host: %v", err)
		return result, err
	}

	// Set file permissions
	if mode != "" {
		chmodCmd := fmt.Sprintf("chmod %s %s", mode, dest)
		_, err = exec.ExecuteWithContext(ctx, "sh", "-c", chmodCmd)
		if err != nil {
			result.Output["warning"] = fmt.Sprintf("Failed to set permissions: %v", err)
		}
	}

	// Set file ownership
	if owner != "" || group != "" {
		chownArg := owner
		if group != "" {
			if owner != "" {
				chownArg = fmt.Sprintf("%s:%s", owner, group)
			} else {
				chownArg = fmt.Sprintf(":%s", group)
			}
		}
		chownCmd := fmt.Sprintf("chown %s %s", chownArg, dest)
		_, err = exec.ExecuteWithContext(ctx, "sh", "-c", chownCmd)
		if err != nil {
			result.Output["warning"] = fmt.Sprintf("Failed to set ownership: %v", err)
		}
	}

	result.Changed = true
	result.Success = true
	result.Output["msg"] = "File downloaded successfully"
	result.Output["url"] = url
	result.Output["dest"] = dest
	result.Output["size"] = bytesWritten
	result.Output["status_code"] = resp.StatusCode

	return result, nil
}

// getRemoteFileChecksum calculates checksum of a file on remote host
func (m *GetURLModule) getRemoteFileChecksum(ctx context.Context, exec *executor.CommandExecutor, path string, checksumType string) (string, error) {
	var cmd string
	checksumAlgo := strings.Split(checksumType, ":")[0]

	switch checksumAlgo {
	case "md5":
		cmd = fmt.Sprintf("md5sum %s 2>/dev/null || md5 -q %s 2>/dev/null", path, path)
	case "sha1":
		cmd = fmt.Sprintf("sha1sum %s 2>/dev/null || shasum -a 1 %s 2>/dev/null", path, path)
	case "sha256":
		cmd = fmt.Sprintf("sha256sum %s 2>/dev/null || shasum -a 256 %s 2>/dev/null", path, path)
	default:
		return "", fmt.Errorf("unsupported checksum type: %s", checksumAlgo)
	}

	output, err := exec.ExecuteWithContext(ctx, "sh", "-c", cmd)
	if err != nil {
		return "", err
	}

	// Extract checksum from output
	parts := strings.Fields(output)
	if len(parts) > 0 {
		return parts[0], nil
	}

	return "", fmt.Errorf("failed to extract checksum from output")
}

// checksumMatches checks if two checksums match
func (m *GetURLModule) checksumMatches(expected, actual string) bool {
	// Expected format: "algorithm:checksum" or just "checksum"
	expectedChecksum := expected
	if strings.Contains(expected, ":") {
		parts := strings.Split(expected, ":")
		if len(parts) == 2 {
			expectedChecksum = parts[1]
		}
	}

	return strings.EqualFold(expectedChecksum, actual)
}

// encodeBase64 encodes bytes to base64 string
func encodeBase64(data []byte) string {
	// Use standard base64 encoding
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	var result strings.Builder
	result.Grow((len(data) + 2) / 3 * 4)

	for i := 0; i < len(data); i += 3 {
		b1 := data[i]
		b2 := byte(0)
		b3 := byte(0)

		if i+1 < len(data) {
			b2 = data[i+1]
		}
		if i+2 < len(data) {
			b3 = data[i+2]
		}

		result.WriteByte(base64Chars[b1>>2])
		result.WriteByte(base64Chars[((b1&0x03)<<4)|(b2>>4)])

		if i+1 < len(data) {
			result.WriteByte(base64Chars[((b2&0x0f)<<2)|(b3>>6)])
		} else {
			result.WriteByte('=')
		}

		if i+2 < len(data) {
			result.WriteByte(base64Chars[b3&0x3f])
		} else {
			result.WriteByte('=')
		}
	}

	return result.String()
}
