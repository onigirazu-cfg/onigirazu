package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// MountModule implements filesystem mounting
type MountModule struct {
	*BaseExecutorModule
}

// MountInfo represents mount point information
type MountInfo struct {
	Source     string
	MountPoint string
	FSType     string
	Options    string
	DumpFreq   string
	PassNum    string
}

// NewMountModule creates a new mount module
func NewMountModule() *MountModule {
	return &MountModule{
		BaseExecutorModule: NewBaseExecutorModule("mount"),
	}
}

// GetDescription returns the module description
func (m *MountModule) GetDescription() string {
	return "Control active and persistent filesystem mounts"
}

// Execute manages filesystem mounts
func (m *MountModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  getStringArg(args, "name", "mount"),
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Validate required parameters
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return m.failResult(result, "parameter 'path' (mount point) is required")
	}

	state := getStringArg(args, "state", "present")

	var execResult types.TaskResult = result

	execErr := m.WithExecutor(host, func(exec *executor.CommandExecutor) error {
		var err error
		switch state {
		case "present":
			execResult, err = m.handlePresent(ctx, exec, host, args, result, path)
		case "absent":
			execResult, err = m.handleAbsent(ctx, exec, host, args, result, path)
		case "mounted":
			execResult, err = m.handleMounted(ctx, exec, host, args, result, path)
		case "unmounted":
			execResult, err = m.handleUnmounted(ctx, exec, host, args, result, path)
		default:
			execResult, err = m.failResult(result, fmt.Sprintf("invalid state: %s", state))
		}
		return err
	})

	if execErr != nil {
		return result, execErr
	}

	return execResult, nil
}

// handlePresent ensures mount is in fstab and mounted
func (m *MountModule) handlePresent(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, path string) (types.TaskResult, error) {
	source, ok := args["src"].(string)
	if !ok || source == "" {
		return m.failResult(result, "parameter 'src' (mount source) is required for state=present")
	}

	fstype := getStringArg(args, "fstype", "defaults")
	opts := getStringArg(args, "opts", "defaults")
	backup := getBoolArg(args, "backup", true)

	result.Output["path"] = path
	result.Output["src"] = source
	result.Output["fstype"] = fstype
	result.Output["opts"] = opts

	// Check if already mounted
	isMounted, err := m.isMounted(exec, path)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to check mount status: %v", err))
	}

	// Check fstab
	inFstab, _, err := m.findInFstab(exec, path)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to read fstab: %v", err))
	}

	// Add to fstab if not present
	if !inFstab {
		fstabEntry := fmt.Sprintf("%s %s %s %s 0 0", source, path, fstype, opts)
		cmd := fmt.Sprintf("echo '%s' >> /etc/fstab", strings.ReplaceAll(fstabEntry, "'", "'\\''"))
		if backup {
			// Backup fstab first
			exec.Execute("cp", "/etc/fstab", "/etc/fstab.bak")
		}
		_, err := exec.Execute("sh", "-c", cmd)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to add mount to fstab: %v", err))
		}
		result.Changed = true
		result.Output["added_to_fstab"] = true
	}

	// Mount if not mounted
	if !isMounted {
		// Ensure mount point exists
		exec.Execute("mkdir", "-p", path)

		// Mount the filesystem
		_, err := exec.Execute("mount", path)
		if err != nil {
			// Try full mount command with source
			_, err = exec.Execute("mount", "-t", fstype, "-o", opts, source, path)
			if err != nil {
				return m.failResult(result, fmt.Sprintf("failed to mount filesystem: %v", err))
			}
		}
		result.Changed = true
		result.Output["mounted"] = true
	}

	if result.Changed {
		result.Output["msg"] = fmt.Sprintf("Mount point %s configured and mounted", path)
	} else {
		result.Output["msg"] = fmt.Sprintf("Mount point %s already configured and mounted", path)
	}

	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleAbsent ensures mount is absent from fstab and unmounted
func (m *MountModule) handleAbsent(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, path string) (types.TaskResult, error) {
	backup := getBoolArg(args, "backup", true)

	result.Output["path"] = path

	// Check fstab
	inFstab, line, err := m.findInFstab(exec, path)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to read fstab: %v", err))
	}

	// Remove from fstab
	if inFstab {
		if backup {
			exec.Execute("cp", "/etc/fstab", "/etc/fstab.bak")
		}
		// Remove the line from fstab
		sedCmd := fmt.Sprintf("sed -i '%d d' /etc/fstab", line)
		_, err := exec.Execute("sh", "-c", sedCmd)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to remove from fstab: %v", err))
		}
		result.Changed = true
		result.Output["removed_from_fstab"] = true
	}

	// Unmount if mounted
	isMounted, err := m.isMounted(exec, path)
	if err == nil && isMounted {
		_, err := exec.Execute("umount", path)
		if err != nil {
			// Try force unmount
			_, err = exec.Execute("umount", "-f", path)
			if err != nil {
				return m.failResult(result, fmt.Sprintf("failed to unmount filesystem: %v", err))
			}
		}
		result.Changed = true
		result.Output["unmounted"] = true
	}

	if result.Changed {
		result.Output["msg"] = fmt.Sprintf("Mount point %s removed and unmounted", path)
	} else {
		result.Output["msg"] = fmt.Sprintf("Mount point %s already absent", path)
	}

	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleMounted ensures filesystem is mounted
func (m *MountModule) handleMounted(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, path string) (types.TaskResult, error) {
	result.Output["path"] = path

	isMounted, err := m.isMounted(exec, path)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to check mount status: %v", err))
	}

	if !isMounted {
		// Try to mount
		_, err := exec.Execute("mount", path)
		if err != nil {
			// If simple mount fails, might need to mount all from fstab
			_, err = exec.Execute("mount", "-a")
			if err != nil {
				return m.failResult(result, fmt.Sprintf("failed to mount filesystem: %v", err))
			}
		}
		result.Changed = true
	}

	result.Output["msg"] = fmt.Sprintf("Mount point %s is mounted", path)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleUnmounted ensures filesystem is unmounted
func (m *MountModule) handleUnmounted(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, path string) (types.TaskResult, error) {
	result.Output["path"] = path

	isMounted, err := m.isMounted(exec, path)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to check mount status: %v", err))
	}

	if isMounted {
		_, err := exec.Execute("umount", path)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to unmount filesystem: %v", err))
		}
		result.Changed = true
	}

	result.Output["msg"] = fmt.Sprintf("Mount point %s is unmounted", path)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// isMounted checks if a path is currently mounted
func (m *MountModule) isMounted(exec *executor.CommandExecutor, path string) (bool, error) {
	output, err := exec.Execute("grep", "-E", fmt.Sprintf("^[^#].+ %s ", path), "/proc/mounts")
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(output) != "", nil
}

// findInFstab finds a mount point in fstab and returns its line number (1-based)
func (m *MountModule) findInFstab(exec *executor.CommandExecutor, path string) (bool, int, error) {
	output, err := exec.Execute("cat", "/etc/fstab")
	if err != nil {
		return false, 0, fmt.Errorf("failed to read /etc/fstab: %v", err)
	}

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		// Skip comments and empty lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Parse fstab line: device mountpoint fstype options dump pass
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 && fields[1] == path {
			return true, i + 1, nil
		}
	}

	return false, 0, nil
}

// Validate validates argument correctness
func (m *MountModule) Validate(args map[string]interface{}) error {
	if _, exists := args["path"]; !exists {
		return fmt.Errorf("'path' parameter is required")
	}

	state := getStringArg(args, "state", "present")
	validStates := map[string]bool{"present": true, "absent": true, "mounted": true, "unmounted": true}
	if !validStates[state] {
		return fmt.Errorf("invalid state: %s", state)
	}

	if state == "present" {
		if _, exists := args["src"]; !exists {
			return fmt.Errorf("'src' parameter is required for state=present")
		}
	}

	return nil
}
