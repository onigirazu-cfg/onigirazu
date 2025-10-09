package modules

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// LineinfileModule manages lines in files
type LineinfileModule struct {
	*BaseModule
	executor *executor.CommandExecutor
}

func NewLineinfileModule() *LineinfileModule {
	return &LineinfileModule{
		BaseModule: NewBaseModule("lineinfile"),
	}
}

func (m *LineinfileModule) GetDescription() string {
	return "Manages lines in text files"
}

func (m *LineinfileModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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
	var err error
	m.executor, err = executor.NewCommandExecutor(host)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}

	path := args["path"].(string)
	line := args["line"].(string)

	// Get optional parameters
	state := "present"
	if s, ok := args["state"].(string); ok {
		state = s
	}

	create := false
	if c, ok := args["create"].(bool); ok {
		create = c
	}

	backup := false
	if b, ok := args["backup"].(bool); ok {
		backup = b
	}

	insertafter := ""
	if ia, ok := args["insertafter"].(string); ok {
		insertafter = ia
	}

	insertbefore := ""
	if ib, ok := args["insertbefore"].(string); ok {
		insertbefore = ib
	}

	// regexp pattern to search for line
	var regexpPattern *regexp.Regexp
	if regexpStr, ok := args["regexp"].(string); ok {
		var err error
		regexpPattern, err = regexp.Compile(regexpStr)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("invalid regexp: %v", err)
			result.Duration = time.Since(startTime)
			return result, err
		}
	}

	// Check if file exists on remote host
	fileExists, err := m.checkFileExists(path)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to check file existence: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}

	if !fileExists && !create {
		result.Success = false
		result.Error = fmt.Sprintf("file %s does not exist and create=false", path)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("file does not exist")
	}

	// Backup if requested
	if backup && fileExists {
		backupPath := fmt.Sprintf("%s.%d.backup", path, time.Now().Unix())
		if err := m.copyFileRemote(path, backupPath); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create backup: %v", err)
			result.Duration = time.Since(startTime)
			return result, err
		}
		result.Output = map[string]interface{}{
			"backup_file": backupPath,
		}
	}

	// Read existing lines from remote file
	var lines []string
	if fileExists {
		lines, err = m.readRemoteFile(path)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to read file: %v", err)
			result.Duration = time.Since(startTime)
			return result, err
		}
	}

	// Process based on state
	var newLines []string
	changed := false

	if state == "absent" {
		// Remove matching lines
		newLines, changed = m.removeLines(lines, line, regexpPattern)
	} else {
		// Add or replace line
		newLines, changed = m.ensureLine(lines, line, regexpPattern, insertafter, insertbefore)
	}

	// Write back if changed
	if changed {
		if err := m.writeRemoteFile(path, newLines); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to write file: %v", err)
			result.Duration = time.Since(startTime)
			return result, err
		}
	}

	result.Success = true
	result.Changed = changed
	result.Duration = time.Since(startTime)

	if result.Output == nil {
		result.Output = make(map[string]interface{})
	}
	result.Output["msg"] = fmt.Sprintf("line %s in file %s", state, path)

	return result, nil
}

func (m *LineinfileModule) Validate(args map[string]interface{}) error {
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

	line, exists := args["line"]
	if !exists {
		return fmt.Errorf("argument 'line' is required")
	}
	if _, ok := line.(string); !ok {
		return fmt.Errorf("argument 'line' must be a string")
	}

	// Validate state if provided
	if state, ok := args["state"].(string); ok {
		if state != "present" && state != "absent" {
			return fmt.Errorf("state must be 'present' or 'absent'")
		}
	}

	return nil
}

// removeLines removes lines matching the pattern or exact line
func (m *LineinfileModule) removeLines(lines []string, line string, pattern *regexp.Regexp) ([]string, bool) {
	newLines := []string{}
	changed := false

	for _, l := range lines {
		match := false
		if pattern != nil {
			match = pattern.MatchString(l)
		} else {
			match = strings.TrimSpace(l) == strings.TrimSpace(line)
		}

		if !match {
			newLines = append(newLines, l)
		} else {
			changed = true
		}
	}

	return newLines, changed
}

// ensureLine ensures the line exists in the file
func (m *LineinfileModule) ensureLine(lines []string, line string, pattern *regexp.Regexp, insertafter, insertbefore string) ([]string, bool) {
	// Check if line already exists
	lineIndex := -1
	for i, l := range lines {
		match := false
		if pattern != nil {
			match = pattern.MatchString(l)
		} else {
			match = strings.TrimSpace(l) == strings.TrimSpace(line)
		}

		if match {
			lineIndex = i
			break
		}
	}

	// If line exists and matches, no change needed
	if lineIndex >= 0 {
		if strings.TrimSpace(lines[lineIndex]) == strings.TrimSpace(line) {
			return lines, false
		}
		// Replace the line
		lines[lineIndex] = line
		return lines, true
	}

	// Line doesn't exist, need to add it
	newLines := []string{}

	// Handle insertafter
	if insertafter != "" {
		inserted := false
		afterPattern, _ := regexp.Compile(insertafter)

		for _, l := range lines {
			newLines = append(newLines, l)
			if !inserted && afterPattern != nil && afterPattern.MatchString(l) {
				newLines = append(newLines, line)
				inserted = true
			}
		}

		if !inserted {
			// Pattern not found, append at end
			newLines = append(newLines, line)
		}
		return newLines, true
	}

	// Handle insertbefore
	if insertbefore != "" {
		inserted := false
		beforePattern, _ := regexp.Compile(insertbefore)

		for _, l := range lines {
			if !inserted && beforePattern != nil && beforePattern.MatchString(l) {
				newLines = append(newLines, line)
				inserted = true
			}
			newLines = append(newLines, l)
		}

		if !inserted {
			// Pattern not found, append at end
			newLines = append(newLines, line)
		}
		return newLines, true
	}

	// No insert position specified, append at end
	newLines = append(lines, line)
	return newLines, true
}

// checkFileExists checks if a file exists on the remote host
func (m *LineinfileModule) checkFileExists(path string) (bool, error) {
	escapedPath := strings.ReplaceAll(path, "'", "'\\''")
	cmd := fmt.Sprintf("test -e '%s' && echo exists || echo notexists", escapedPath)

	// Note: executor.Execute will automatically use shell if needed
	output, err := m.executor.Execute(cmd)
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(output) == "exists", nil
}

// readRemoteFile reads a file from the remote host and returns lines
func (m *LineinfileModule) readRemoteFile(path string) ([]string, error) {
	escapedPath := strings.ReplaceAll(path, "'", "'\\''")
	cmd := fmt.Sprintf("cat '%s'", escapedPath)

	// Note: executor.Execute will automatically use shell if needed
	output, err := m.executor.Execute(cmd)
	if err != nil {
		return nil, err
	}

	// Split output into lines
	lines := strings.Split(output, "\n")
	// Remove the last empty line if present (cat adds trailing newline)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines, nil
}

// writeRemoteFile writes lines to a file on the remote host
func (m *LineinfileModule) writeRemoteFile(path string, lines []string) error {
	escapedPath := strings.ReplaceAll(path, "'", "'\\''")

	// Join lines with newline
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n" // Add trailing newline
	}

	// Escape content for shell
	escapedContent := strings.ReplaceAll(content, "'", "'\\''")

	// Write content to file using printf
	cmd := fmt.Sprintf("printf '%%s' '%s' > '%s'", escapedContent, escapedPath)

	// Note: executor.Execute will automatically use shell if needed
	_, err := m.executor.Execute(cmd)
	return err
}

// copyFileRemote creates a backup copy of a file on the remote host
func (m *LineinfileModule) copyFileRemote(src, dst string) error {
	escapedSrc := strings.ReplaceAll(src, "'", "'\\''")
	escapedDst := strings.ReplaceAll(dst, "'", "'\\''")
	cmd := fmt.Sprintf("cp '%s' '%s'", escapedSrc, escapedDst)

	// Note: executor.Execute will automatically use shell if needed
	_, err := m.executor.Execute(cmd)
	return err
}
