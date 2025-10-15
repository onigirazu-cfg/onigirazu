package modules

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
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
		// Read from source file (always local)
		sourceData, err = os.ReadFile(src) // #nosec G304 -- src is validated by security validator
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

	// Calculate source checksum
	sourceChecksum := fmt.Sprintf("%x", sha256.Sum256(sourceData))
	result.Output["checksum"] = sourceChecksum

	// Check if we're working with a local or remote host
	isLocal := sshpkg.IsLocal(host)

	if isLocal {
		return m.executeLocal(dest, sourceData, backup, force, mode, owner, group, sourceChecksum, result)
	} else {
		return m.executeRemote(host, dest, sourceData, backup, force, mode, owner, group, sourceChecksum, result)
	}
}

// executeLocal handles file copying on local host
func (m *CopyModule) executeLocal(dest string, sourceData []byte, backup, force bool, mode, owner, group, sourceChecksum string, result types.TaskResult) (types.TaskResult, error) {
	// Check if destination exists
	destExists := true
	_, err := os.Stat(dest)
	if os.IsNotExist(err) {
		destExists = false
	} else if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to stat destination %s: %v", dest, err)
		return result, err
	}

	var destChecksum string
	if destExists {
		destData, err := os.ReadFile(dest) // #nosec G304 -- dest is validated by security validator
		if err != nil {
			result.Failed = true
			result.Error = fmt.Sprintf("failed to read destination file %s: %v", dest, err)
			return result, err
		}
		destChecksum = fmt.Sprintf("%x", sha256.Sum256(destData))
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
	if err := os.MkdirAll(destDir, 0750); err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to create destination directory %s: %v", destDir, err)
		return result, err
	}

	// Determine file mode
	fileMode := os.FileMode(0644)
	if mode != "" {
		modeInt, err := strconv.ParseUint(mode, 8, 32)
		if err != nil {
			result.Output["mode_warning"] = fmt.Sprintf("invalid mode %s, using default 0644", mode)
		} else {
			fileMode = os.FileMode(modeInt)
		}
	}

	// Write the file
	if err := os.WriteFile(dest, sourceData, fileMode); err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to write file %s: %v", dest, err)
		return result, err
	}

	result.Changed = true
	result.Success = true
	result.Output["dest"] = dest
	result.Output["size"] = len(sourceData)

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

// executeRemote handles file copying on remote host via SFTP
func (m *CopyModule) executeRemote(host types.Host, dest string, sourceData []byte, backup, force bool, mode, owner, group, sourceChecksum string, result types.TaskResult) (types.TaskResult, error) {
	// Get SSH client from connection pool
	pool := sshpkg.GetGlobalPool()
	sshClient, err := pool.GetConnection(host)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to get SSH connection: %v", err)
		return result, err
	}
	defer pool.ReleaseConnection(host)

	// Check if destination exists on remote host
	destExists := true
	_, err = sshClient.StatFile(dest)
	if err != nil {
		if os.IsNotExist(err) {
			destExists = false
		} else {
			result.Failed = true
			result.Error = fmt.Sprintf("failed to stat remote destination %s: %v", dest, err)
			return result, err
		}
	}

	var destChecksum string
	if destExists {
		destData, err := sshClient.ReadFile(dest)
		if err != nil {
			result.Failed = true
			result.Error = fmt.Sprintf("failed to read remote destination file %s: %v", dest, err)
			return result, err
		}
		destChecksum = fmt.Sprintf("%x", sha256.Sum256(destData))
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
		// Use command to create backup on remote host
		exec, err := executor.NewCommandExecutor(host)
		if err != nil {
			result.Failed = true
			result.Error = fmt.Sprintf("failed to create executor for backup: %v", err)
			return result, err
		}
		defer exec.Close()

		_, err = exec.Execute("cp", dest, backupPath)
		if err != nil {
			result.Failed = true
			result.Error = fmt.Sprintf("failed to create backup: %v", err)
			return result, err
		}
		result.Output["backup_file"] = backupPath
	}

	// Determine file mode
	fileMode := os.FileMode(0644)
	if mode != "" {
		modeInt, err := strconv.ParseUint(mode, 8, 32)
		if err != nil {
			result.Output["mode_warning"] = fmt.Sprintf("invalid mode %s, using default 0644", mode)
		} else {
			fileMode = os.FileMode(modeInt)
		}
	}

	// Write file to remote host using SFTP
	if err := sshClient.WriteFile(dest, sourceData, fileMode); err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to write remote file %s: %v", dest, err)
		return result, err
	}

	result.Changed = true
	result.Success = true
	result.Output["dest"] = dest
	result.Output["size"] = len(sourceData)

	// Set ownership if specified
	if owner != "" || group != "" {
		result.Output["ownership_warning"] = "ownership setting not implemented yet"
	}

	// Get final file info
	if finalInfo, err := sshClient.StatFile(dest); err == nil {
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

	// If src is provided, check if it exists (locally)
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

// copyFile copies a file from src to dst (local only)
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src) // #nosec G304 -- src is validated by security validator
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst) // #nosec G304 -- dst is validated by security validator
	if err != nil {
		return err
	}
	defer func() {
		// Best-effort cleanup in case of panic
		_ = destFile.Close()
	}()

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Ensure data is flushed to disk before closing
	if err = destFile.Sync(); err != nil {
		return err
	}

	// Explicitly close and handle any errors
	if err = destFile.Close(); err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
