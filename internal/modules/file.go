package modules

import (
	"context"
	"fmt"
	"path/filepath"
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
		TaskName:  args["name"].(string),
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

	path := args["path"].(string)
	state := args["state"].(string)

	switch state {
	case "present":
		return m.ensureFilePresent(exec, path, result, startTime, args)
	case "absent":
		return m.ensureFileAbsent(exec, path, result, startTime)
	case "directory":
		return m.ensureDirectory(exec, path, result, startTime)
	case "touch":
		return m.touchFile(exec, path, result, startTime)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("unsupported state: %s", state)
		result.Duration = time.Since(startTime)
		return result, nil
	}
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
	stateStr := state.(string)
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
	checkCmd := fmt.Sprintf(`test -e '%s' && cat '%s' || echo __NOTEXISTS__`, path, path)
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

	if needsUpdate {
		// Create directory if needed
		dir := filepath.Dir(path)
		mkdirCmd := fmt.Sprintf(`mkdir -p '%s'`, dir)
		_, err := exec.Execute(mkdirCmd)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("error creating directory: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		// Write file with content - escape content for shell
		escapedContent := strings.ReplaceAll(content, "'", "'\\''")
		writeCmd := fmt.Sprintf(`printf '%%s' '%s' > '%s'`, escapedContent, path)
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

func (m *FileModule) ensureFileAbsent(exec *executor.CommandExecutor, path string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Check if file exists
	checkCmd := fmt.Sprintf(`test -e '%s' && echo exists || echo notexists`, path)
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

	// Remove file
	removeCmd := fmt.Sprintf(`rm -rf '%s'`, path)
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

func (m *FileModule) ensureDirectory(exec *executor.CommandExecutor, path string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Check if directory exists
	checkCmd := fmt.Sprintf(`test -d '%s' && echo exists || echo notexists`, path)
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

	// Create directory
	createCmd := fmt.Sprintf(`mkdir -p '%s'`, path)
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

func (m *FileModule) touchFile(exec *executor.CommandExecutor, path string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Check if file exists
	// Note: executor.Execute will automatically use shell if needed
	checkCmd := fmt.Sprintf(`test -e '%s' && echo exists || echo notexists`, path)
	output, err := exec.Execute(checkCmd)
	fileExists := (err == nil && strings.TrimSpace(output) == "exists")

	if !fileExists {
		// Create the file
		touchCmd := fmt.Sprintf(`touch '%s'`, path)
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
		touchCmd := fmt.Sprintf(`touch '%s'`, path)
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
