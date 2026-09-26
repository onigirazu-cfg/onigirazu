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
		TaskName:  taskName(args),
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

// Mount states follow Ansible: present = fstab entry only; mounted = fstab
// entry, mount point and mounted; unmounted = not mounted, fstab untouched;
// absent = not mounted and no fstab entry.

func (m *MountModule) handlePresent(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, path string) (types.TaskResult, error) {
	changed, err := m.ensureFstab(ctx, host, args, path)
	if err != nil {
		return m.failResult(result, err.Error())
	}
	result.Changed = changed
	result.Output["path"] = path
	result.Output["msg"] = fmt.Sprintf("fstab entry for %s is present", path)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

func (m *MountModule) handleMounted(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, path string) (types.TaskResult, error) {
	changed, err := m.ensureFstab(ctx, host, args, path)
	if err != nil {
		return m.failResult(result, err.Error())
	}
	if inCheckMode(args) {
		result.Changed = changed || !m.isMounted(ctx, host, args, path)
		result.Output["path"] = path
		result.Output["msg"] = fmt.Sprintf("Mount point %s would be mounted", path)
		result.Duration = time.Since(result.Timestamp)
		return result, nil
	}
	if _, err := runOnHost(ctx, host, args, "mkdir", "-p", path); err != nil {
		return m.failResult(result, fmt.Sprintf("failed to create mount point %s: %v", path, err))
	}

	mounted := m.isMounted(ctx, host, args, path)
	switch {
	case !mounted:
		if _, err := runOnHost(ctx, host, args, "mount", path); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to mount %s: %v", path, err))
		}
		changed = true
	case changed:
		// New options for a mounted filesystem take effect on remount
		if _, err := runOnHost(ctx, host, args, "mount", "-o", "remount", path); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to remount %s: %v", path, err))
		}
	}
	if !m.isMounted(ctx, host, args, path) {
		return m.failResult(result, fmt.Sprintf("%s is not mounted after mount", path))
	}

	result.Changed = changed
	result.Output["path"] = path
	result.Output["msg"] = fmt.Sprintf("Mount point %s is mounted", path)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

func (m *MountModule) handleUnmounted(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, path string) (types.TaskResult, error) {
	changed, err := m.unmount(ctx, host, args, path)
	if err != nil {
		return m.failResult(result, err.Error())
	}
	result.Changed = changed
	result.Output["path"] = path
	result.Output["msg"] = fmt.Sprintf("Mount point %s is unmounted", path)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

func (m *MountModule) handleAbsent(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult, path string) (types.TaskResult, error) {
	unmounted, err := m.unmount(ctx, host, args, path)
	if err != nil {
		return m.failResult(result, err.Error())
	}
	lines, err := m.readFstab(ctx, host, args)
	if err != nil {
		return m.failResult(result, err.Error())
	}
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if fstabMountPoint(line) != path {
			kept = append(kept, line)
		}
	}
	removed := len(kept) != len(lines)
	if removed {
		if err := m.writeFstab(ctx, host, args, kept); err != nil {
			return m.failResult(result, err.Error())
		}
	}
	result.Changed = unmounted || removed
	result.Output["path"] = path
	result.Output["msg"] = fmt.Sprintf("Mount point %s is absent", path)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// unmount unmounts path if it is mounted and reports whether it did
func (m *MountModule) unmount(ctx context.Context, host types.Host, args map[string]interface{}, path string) (bool, error) {
	if !m.isMounted(ctx, host, args, path) {
		return false, nil
	}
	if inCheckMode(args) {
		return true, nil
	}
	if _, err := runOnHost(ctx, host, args, "umount", path); err != nil {
		return false, fmt.Errorf("failed to unmount %s: %v", path, err)
	}
	if m.isMounted(ctx, host, args, path) {
		return false, fmt.Errorf("%s is still mounted after umount", path)
	}
	return true, nil
}

// isMounted reports whether a filesystem is mounted exactly at path
func (m *MountModule) isMounted(ctx context.Context, host types.Host, args map[string]interface{}, path string) bool {
	_, err := runOnHost(ctx, host, args, "findmnt", "-rn", "--mountpoint", path)
	return err == nil
}

// ensureFstab makes /etc/fstab carry exactly the task's entry for path
func (m *MountModule) ensureFstab(ctx context.Context, host types.Host, args map[string]interface{}, path string) (bool, error) {
	source := getStringArg(args, "src", "")
	if source == "" {
		return false, fmt.Errorf("parameter 'src' (mount source) is required")
	}
	fstype := getStringArg(args, "fstype", "auto")
	opts := getStringArg(args, "opts", "defaults")
	want := strings.Join([]string{source, path, fstype, opts, getStringArg(args, "dump", "0"), getStringArg(args, "passno", "0")}, " ")

	lines, err := m.readFstab(ctx, host, args)
	if err != nil {
		return false, err
	}
	found := false
	out := make([]string, 0, len(lines)+1)
	for _, line := range lines {
		if fstabMountPoint(line) != path {
			out = append(out, line)
			continue
		}
		if found {
			continue // drop duplicates for the same mount point
		}
		found = true
		out = append(out, want)
	}
	if !found {
		out = append(out, want)
	}
	if strings.Join(out, "\n") == strings.Join(lines, "\n") {
		return false, nil
	}
	return true, m.writeFstab(ctx, host, args, out)
}

func (m *MountModule) readFstab(ctx context.Context, host types.Host, args map[string]interface{}) ([]string, error) {
	out, err := runOnHost(ctx, host, args, "cat", "/etc/fstab")
	if err != nil {
		return nil, fmt.Errorf("failed to read /etc/fstab: %v", err)
	}
	return strings.Split(strings.TrimRight(out, "\n"), "\n"), nil
}

func (m *MountModule) writeFstab(ctx context.Context, host types.Host, args map[string]interface{}, lines []string) error {
	if inCheckMode(args) {
		return nil
	}
	script := "printf '%s' " + shellQuote(strings.Join(lines, "\n")+"\n") + " > /etc/fstab"
	if getBoolArg(args, "backup", true) {
		script = "cp -p /etc/fstab /etc/fstab.bak && " + script
	}
	if _, err := runShellOnHost(ctx, host, args, script); err != nil {
		return fmt.Errorf("failed to write /etc/fstab: %v", err)
	}
	return nil
}

// fstabMountPoint returns the mount point field of an fstab line, or "" for
// comments and blank lines
func fstabMountPoint(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 2 || strings.HasPrefix(fields[0], "#") {
		return ""
	}
	return fields[1]
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
