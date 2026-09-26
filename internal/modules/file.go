package modules

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// FileModule manages files
type FileModule struct {
	*BaseModule
}

func NewFileModule() *FileModule {
	return &FileModule{
		BaseModule: NewBaseModule("file"),
	}
}

func (m *FileModule) GetDescription() string {
	return "Manages files and directories"
}

func (m *FileModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  taskName(args),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
	}

	// Validate arguments
	if err := m.Validate(args); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Initialize executor for remote execution
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}
	defer exec.Close()

	path, _ := args["path"].(string)
	state, _ := args["state"].(string)

	switch state {
	case "present":
		result, err = m.ensureFilePresent(exec, path, result, startTime, args)
	case "absent":
		return m.ensureFileAbsent(exec, path, result, startTime, inCheckMode(args))
	case "directory":
		result, err = m.ensureDirectory(exec, path, result, startTime, inCheckMode(args))
	case "touch":
		result, err = m.touchFile(exec, path, result, startTime, inCheckMode(args))
	default:
		result.Success = false
		result.Error = fmt.Sprintf("unsupported state: %s", state)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	if err != nil || !result.Success {
		return result, err
	}
	return m.applyAttributes(ctx, host, args, path, result, startTime)
}

// applyAttributes enforces mode, owner and group on path when they are given
func (m *FileModule) applyAttributes(ctx context.Context, host types.Host, args map[string]interface{}, path string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	fail := func(msg string) (types.TaskResult, error) {
		result.Success = false
		result.Error = msg
		result.Duration = time.Since(startTime)
		return result, nil
	}

	if mode := getStringArg(args, "mode", ""); mode != "" {
		want, err := strconv.ParseUint(mode, 8, 32)
		if err != nil {
			return fail(fmt.Sprintf("invalid mode %q", mode))
		}
		out, err := runOnHost(ctx, host, args, "stat", "-c", "%a", path)
		if err != nil && inCheckMode(args) {
			out = "" // the path does not exist yet; its mode would be set
		} else if err != nil {
			return fail(fmt.Sprintf("failed to read mode of %s: %v", path, err))
		}
		have, _ := strconv.ParseUint(strings.TrimSpace(out), 8, 32)
		if have != want && inCheckMode(args) {
			result.Changed = true
		} else if have != want {
			if _, err := runOnHost(ctx, host, args, "chmod", fmt.Sprintf("%04o", want), path); err != nil {
				return fail(fmt.Sprintf("failed to set mode of %s: %v", path, err))
			}
			result.Changed = true
		}
	}

	changed, err := ensureOwnership(ctx, host, args, path, getStringArg(args, "owner", ""), getStringArg(args, "group", ""))
	if err != nil {
		return fail(err.Error())
	}
	result.Changed = result.Changed || changed
	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *FileModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	path, exists := args["path"]
	if !exists {
		return fmt.Errorf("argument 'path' is required")
	}

	if _, ok := path.(string); !ok {
		return fmt.Errorf("argument 'path' must be a string")
	}

	state, exists := args["state"]
	if !exists {
		return fmt.Errorf("argument 'state' is required")
	}

	if _, ok := state.(string); !ok {
		return fmt.Errorf("argument 'state' must be a string")
	}

	validStates := []string{"present", "absent", "directory", "touch"}
	stateStr, _ := state.(string)
	for _, validState := range validStates {
		if stateStr == validState {
			return nil
		}
	}

	return fmt.Errorf("unsupported state: %s", stateStr)
}

func (m *FileModule) ensureFilePresent(exec *executor.CommandExecutor, path string, result types.TaskResult, startTime time.Time, args map[string]interface{}) (types.TaskResult, error) {
	// Get content if provided
	var content string
	if contentArg, exists := args["content"]; exists {
		if contentStr, ok := contentArg.(string); ok {
			content = contentStr
		}
	}

	// Check if file exists and get current content
	fileExists := false
	currentContent := ""
	checkCmd := fmt.Sprintf(`test -e %s && cat %s || echo __NOTEXISTS__`, shellQuote(path), shellQuote(path))
	output, err := exec.Execute(checkCmd)
	if err == nil {
		if strings.Contains(output, "__NOTEXISTS__") {
			fileExists = false
		} else {
			fileExists = true
			currentContent = output
		}
	}

	needsUpdate := !fileExists || (content != "" && currentContent != content)

	if needsUpdate && inCheckMode(args) {
		return wouldChange(result, startTime, fmt.Sprintf("%s would be written", path))
	}
	if needsUpdate {
		// Create directory if needed
		dir := filepath.Dir(path)
		mkdirCmd := fmt.Sprintf(`mkdir -p %s`, shellQuote(dir))
		_, err := exec.Execute(mkdirCmd)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error creating directory: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		writeCmd := fmt.Sprintf(`printf '%%s' %s > %s`, shellQuote(content), shellQuote(path))
		_, err = exec.Execute(writeCmd)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error writing file: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		result.Success = true
		result.Changed = true
		if fileExists {
			result.Output = map[string]interface{}{
				"message": fmt.Sprintf("File %s updated", path),
			}
		} else {
			result.Output = map[string]interface{}{
				"message": fmt.Sprintf("File %s created", path),
			}
		}
	} else {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("File %s already exists with correct content", path),
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *FileModule) ensureFileAbsent(exec *executor.CommandExecutor, path string, result types.TaskResult, startTime time.Time, check bool) (types.TaskResult, error) {
	// Check if file exists
	checkCmd := fmt.Sprintf(`test -e %s && echo exists || echo notexists`, shellQuote(path))
	output, err := exec.Execute(checkCmd)

	if err == nil && strings.TrimSpace(output) == "notexists" {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("File %s does not exist", path),
		}
		result.Duration = time.Since(startTime)
		return result, nil
	}

	if check {
		return wouldChange(result, startTime, fmt.Sprintf("%s would be removed", path))
	}

	// Remove file
	removeCmd := fmt.Sprintf(`rm -rf %s`, shellQuote(path))
	_, err = exec.Execute(removeCmd)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("error deleting file: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.Success = true
	result.Changed = true
	result.Output = map[string]interface{}{
		"message": fmt.Sprintf("File %s deleted", path),
	}
	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *FileModule) ensureDirectory(exec *executor.CommandExecutor, path string, result types.TaskResult, startTime time.Time, check bool) (types.TaskResult, error) {
	// Check if directory exists
	checkCmd := fmt.Sprintf(`test -d %s && echo exists || echo notexists`, shellQuote(path))
	output, err := exec.Execute(checkCmd)

	if err == nil && strings.TrimSpace(output) == "exists" {
		result.Success = true
		result.Changed = false
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("Directory %s already exists", path),
		}
		result.Duration = time.Since(startTime)
		return result, nil
	}

	if check {
		return wouldChange(result, startTime, fmt.Sprintf("directory %s would be created", path))
	}

	// Create directory
	createCmd := fmt.Sprintf(`mkdir -p %s`, shellQuote(path))
	_, err = exec.Execute(createCmd)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("error creating directory: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.Success = true
	result.Changed = true
	result.Output = map[string]interface{}{
		"message": fmt.Sprintf("Directory %s created", path),
	}
	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *FileModule) touchFile(exec *executor.CommandExecutor, path string, result types.TaskResult, startTime time.Time, check bool) (types.TaskResult, error) {
	// Check if file exists
	// Note: executor.Execute will automatically use shell if needed
	checkCmd := fmt.Sprintf(`test -e %s && echo exists || echo notexists`, shellQuote(path))
	output, err := exec.Execute(checkCmd)
	fileExists := (err == nil && strings.TrimSpace(output) == "exists")

	if check {
		return wouldChange(result, startTime, fmt.Sprintf("%s would be touched", path))
	}

	if !fileExists {
		// Create the file
		touchCmd := fmt.Sprintf(`touch %s`, shellQuote(path))
		_, err := exec.Execute(touchCmd)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error creating file: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		result.Success = true
		result.Changed = true
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("File %s created", path),
		}
	} else {
		// Update the modification time
		touchCmd := fmt.Sprintf(`touch %s`, shellQuote(path))
		_, err := exec.Execute(touchCmd)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error updating file times: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		result.Success = true
		result.Changed = false // Touch doesn't change content, just timestamps
		result.Output = map[string]interface{}{
			"message": fmt.Sprintf("File %s touched", path),
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// wouldChange is the check mode result of a change that was not made
func wouldChange(result types.TaskResult, startTime time.Time, msg string) (types.TaskResult, error) {
	result.Success = true
	result.Changed = true
	if result.Output == nil {
		result.Output = map[string]interface{}{}
	}
	result.Output["msg"] = msg
	result.Duration = time.Since(startTime)
	return result, nil
}
