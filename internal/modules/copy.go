package modules

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CopyModule handles file copying operations
type CopyModule struct {
	BaseModule
}

// NewCopyModule creates a new copy module instance
func NewCopyModule() *CopyModule {
	return &CopyModule{
		BaseModule: BaseModule{
			name:        "copy",
			description: "Copy files to remote locations",
		},
	}
}

// GetDescription returns the module description
func (m *CopyModule) GetDescription() string {
	return m.description
}

// Execute runs the copy module
func (m *CopyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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

	src, _ := args["src"].(string)
	dest, _ := args["dest"].(string)
	backup := getBoolArg(args, "backup", false)
	force := getBoolArg(args, "force", false)
	mode := getStringArg(args, "mode", "")
	owner := getStringArg(args, "owner", "")
	group := getStringArg(args, "group", "")
	content := getStringArg(args, "content", "")

	// Handle content vs src
	var sourceData []byte
	var err error

	if content != "" {
		// Use provided content
		sourceData = []byte(content)
		result.Output["source"] = "content"
	} else if src != "" {
		// Read from source file
		sourceData, err = os.ReadFile(src)
		if err != nil {
			result.Failed = true
			result.Error = fmt.Sprintf("failed to read source file %s: %v", src, err)
			return result, err
		}
		result.Output["source"] = src
	} else {
		err := fmt.Errorf("either 'src' or 'content' must be specified")
		result.Failed = true
		result.Error = err.Error()
		return result, err
	}

	// Check if destination exists
	destExists := true
	_, err = os.Stat(dest)
	if os.IsNotExist(err) {
		destExists = false
	} else if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to stat destination %s: %v", dest, err)
		return result, err
	}

	// Calculate checksums
	sourceChecksum := fmt.Sprintf("%x", md5.Sum(sourceData))
	result.Output["checksum"] = sourceChecksum

	var destChecksum string
	if destExists {
		destData, err := os.ReadFile(dest)
		if err != nil {
			result.Failed = true
			result.Error = fmt.Sprintf("failed to read destination file %s: %v", dest, err)
			return result, err
		}
		destChecksum = fmt.Sprintf("%x", md5.Sum(destData))
		result.Output["dest_checksum"] = destChecksum
	}

	// Check if file needs to be copied
	needsCopy := !destExists || sourceChecksum != destChecksum || force

	if !needsCopy {
		result.Success = true
		result.Output["msg"] = "file already exists with correct content"
		return result, nil
	}

	// Create backup if requested and file exists
	if backup && destExists {
		backupPath := dest + ".backup." + time.Now().Format("20060102-150405")
		if err := copyFile(dest, backupPath); err != nil {
			result.Failed = true
			result.Error = fmt.Sprintf("failed to create backup: %v", err)
			return result, err
		}
		result.Output["backup_file"] = backupPath
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to create destination directory %s: %v", destDir, err)
		return result, err
	}

	// Write the file
	if err := os.WriteFile(dest, sourceData, 0644); err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to write file %s: %v", dest, err)
		return result, err
	}

	result.Changed = true
	result.Success = true
	result.Output["dest"] = dest
	result.Output["size"] = len(sourceData)

	// Set file permissions if specified
	if mode != "" {
		result.Output["mode_warning"] = "mode setting not implemented yet"
	}

	// Set ownership if specified
	if owner != "" || group != "" {
		result.Output["ownership_warning"] = "ownership setting not implemented yet"
	}

	// Get final file info
	if finalInfo, err := os.Stat(dest); err == nil {
		result.Output["mode_actual"] = finalInfo.Mode().String()
		result.Output["size_actual"] = finalInfo.Size()
		result.Output["modified"] = finalInfo.ModTime()
	}

	if destExists {
		result.Output["msg"] = "file updated"
	} else {
		result.Output["msg"] = "file created"
	}

	return result, nil
}

// Validate checks if the provided arguments are valid
func (m *CopyModule) Validate(args map[string]interface{}) error {
	// Must have dest
	dest, ok := args["dest"].(string)
	if !ok || dest == "" {
		return fmt.Errorf("'dest' is required and must be a non-empty string")
	}

	// Must have either src or content
	src, hasSrc := args["src"].(string)
	_, hasContent := args["content"].(string)

	if !hasSrc && !hasContent {
		return fmt.Errorf("either 'src' or 'content' must be specified")
	}

	if hasSrc && hasContent {
		return fmt.Errorf("'src' and 'content' are mutually exclusive")
	}

	// If src is provided, check if it exists
	if hasSrc && src != "" {
		if _, err := os.Stat(src); os.IsNotExist(err) {
			return fmt.Errorf("source file '%s' does not exist", src)
		}
	}

	// Validate mode if provided (basic validation)
	if mode, ok := args["mode"].(string); ok && mode != "" {
		// Basic mode validation - should be octal string like "0644"
		if len(mode) < 3 || len(mode) > 4 {
			return fmt.Errorf("invalid mode '%s': mode should be 3-4 digit octal string", mode)
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
