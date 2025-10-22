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

// FindModule searches for files matching patterns
type FindModule struct {
	*BaseModule
}

// NewFindModule creates a new find module instance
func NewFindModule() *FindModule {
	return &FindModule{
		BaseModule: NewBaseModule("find"),
	}
}

// GetDescription returns the module description
func (m *FindModule) GetDescription() string {
	return "Search for files matching patterns"
}

// Execute runs the find module
func (m *FindModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Output:    make(map[string]interface{}),
	}

	// Validate arguments
	if err := m.Validate(args); err != nil {
		result.Failed = true
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	path := getStringArg(args, "path", ".")
	pattern := getStringArg(args, "pattern", "*")
	fileType := getStringArg(args, "type", "")
	limit := getIntArg(args, "limit", 0)

	// Initialize executor for remote execution
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}
	defer exec.Close()

	// Find files matching pattern
	files, err := m.findFiles(exec, path, pattern, fileType, limit)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("find failed: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}

	result.Success = true
	result.Changed = false // find never changes anything
	result.Output["files"] = files
	result.Output["file_count"] = len(files)
	result.Duration = time.Since(startTime)

	return result, nil
}

// findFiles searches for files matching the pattern
func (m *FindModule) findFiles(exec *executor.CommandExecutor, path, pattern, fileType string, limit int) ([]map[string]interface{}, error) {
	var files []map[string]interface{}

	// Build find command
	cmd := fmt.Sprintf("find '%s' -type %s -name '%s' -print0 2>/dev/null | tr '\\0' '\\n' | head -n %d",
		path, m.getTypeFlag(fileType), escapeSingleQuotes(pattern), m.getLimitValue(limit))

	// Execute find command
	output, err := exec.Execute(cmd)
	if err != nil {
		// If path doesn't exist, return empty list
		if strings.Contains(err.Error(), "No such file") || strings.Contains(err.Error(), "no such file") {
			return files, nil
		}
		return nil, fmt.Errorf("find command failed: %v", err)
	}

	// Parse output lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Get file stats
		if fileInfo, err := getFileStats(exec, line); err == nil {
			files = append(files, fileInfo)
		}
	}

	return files, nil
}

// getFileStats retrieves stats for a single file
func getFileStats(exec *executor.CommandExecutor, filePath string) (map[string]interface{}, error) {
	fileInfo := make(map[string]interface{})
	fileInfo["path"] = filePath

	// Extract filename
	fileInfo["name"] = filepath.Base(filePath)

	// Get file size and type using stat command
	cmd := fmt.Sprintf(`if [ -e '%s' ]; then if [ -d '%s' ]; then TYPE=directory; elif [ -L '%s' ]; then TYPE=link; elif [ -f '%s' ]; then TYPE=file; else TYPE=other; fi; SIZE=$(stat -c %%s '%s' 2>/dev/null || stat -f %%z '%s' 2>/dev/null); MODE=$(stat -c %%a '%s' 2>/dev/null || stat -f %%A '%s' 2>/dev/null); MTIME=$(stat -c %%Y '%s' 2>/dev/null || stat -f %%m '%s' 2>/dev/null); echo "type=$TYPE|size=$SIZE|mode=$MODE|mtime=$MTIME"; fi`,
		filePath, filePath, filePath, filePath, filePath, filePath, filePath, filePath, filePath, filePath)

	output, err := exec.Execute(cmd)
	if err != nil {
		return fileInfo, err
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return fileInfo, nil
	}

	// Parse stats output
	pairs := strings.Split(output, "|")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			switch kv[0] {
			case "type":
				fileInfo["type"] = kv[1]
				fileInfo["isdir"] = kv[1] == "directory"
				fileInfo["isfile"] = kv[1] == "file"
				fileInfo["islink"] = kv[1] == "link"
			case "size":
				if sizeStr := strings.TrimSpace(kv[1]); sizeStr != "" {
					fileInfo["size"] = sizeStr
				}
			case "mode":
				fileInfo["mode"] = strings.TrimSpace(kv[1])
			case "mtime":
				fileInfo["mtime"] = strings.TrimSpace(kv[1])
			}
		}
	}

	return fileInfo, nil
}

// getTypeFlag returns the find -type flag value
func (m *FindModule) getTypeFlag(fileType string) string {
	switch fileType {
	case "file":
		return "f"
	case "directory":
		return "d"
	case "link":
		return "l"
	case "socket":
		return "s"
	case "pipe":
		return "p"
	case "block":
		return "b"
	case "char":
		return "c"
	default:
		return "f" // Default to files
	}
}

// getLimitValue returns the appropriate limit value for head command
func (m *FindModule) getLimitValue(limit int) int {
	if limit <= 0 {
		return 999999 // Very large number
	}
	return limit
}

// escapeSingleQuotes escapes single quotes in pattern
func escapeSingleQuotes(pattern string) string {
	return strings.ReplaceAll(pattern, "'", "'\\''")
}

// Validate checks if the provided arguments are valid
func (m *FindModule) Validate(args map[string]interface{}) error {
	// path is optional, defaults to "."
	if path, ok := args["path"].(string); ok && path == "" {
		return fmt.Errorf("'path' must not be empty")
	}

	// pattern is optional, defaults to "*"
	if pattern, ok := args["pattern"].(string); ok && pattern == "" {
		return fmt.Errorf("'pattern' must not be empty")
	}

	// type is optional
	if fileType, ok := args["type"].(string); ok {
		validTypes := []string{"file", "directory", "link", "socket", "pipe", "block", "char", ""}
		validType := false
		for _, t := range validTypes {
			if fileType == t {
				validType = true
				break
			}
		}
		if !validType {
			return fmt.Errorf("invalid type '%s': must be one of file, directory, link, socket, pipe, block, char", fileType)
		}
	}

	// limit is optional, must be non-negative
	if limit, ok := args["limit"].(float64); ok && limit < 0 {
		return fmt.Errorf("'limit' must be non-negative")
	}

	return nil
}
