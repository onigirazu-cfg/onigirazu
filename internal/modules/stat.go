package modules

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// StatModule retrieves file or directory status
type StatModule struct {
	*BaseModule
	executor *executor.CommandExecutor
}

func NewStatModule() *StatModule {
	return &StatModule{
		BaseModule: NewBaseModule("stat"),
	}
}

func (m *StatModule) GetDescription() string {
	return "Retrieves file or directory status"
}

func (m *StatModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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

	// Initialize executor if not already done
	if m.executor == nil {
		exec, err := executor.NewCommandExecutor(host)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create executor: %v", err)
			result.Duration = time.Since(startTime)
			return result, err
		}
		m.executor = exec
	}

	path := args["path"].(string)

	// Get file info using remote stat command
	statOutput, err := m.getRemoteFileStat(path)

	if err != nil {
		// Check if file doesn't exist
		if strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "cannot stat") {
			// File doesn't exist
			statOutput = make(map[string]interface{})
			statOutput["exists"] = false
			statOutput["path"] = path
			statOutput["stat"] = map[string]interface{}{
				"exists": false,
				"path":   path,
			}
			result.Success = true
			result.Changed = false
			result.Output = statOutput
			result.Duration = time.Since(startTime)
			return result, nil
		}
		// Other error
		result.Success = false
		result.Error = fmt.Sprintf("failed to stat file: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}

	result.Success = true
	result.Changed = false // stat never changes anything
	result.Output = statOutput
	result.Duration = time.Since(startTime)

	return result, nil
}

// getRemoteFileStat retrieves file information from remote host using stat command
func (m *StatModule) getRemoteFileStat(path string) (map[string]interface{}, error) {
	// Use stat command with JSON-like output format
	// Format: exists|type|size|mode|mtime
	// Build command with proper escaping - escape pipes in echo to avoid shell interpretation
	cmd := fmt.Sprintf(`if [ -e '%s' ]; then if [ -d '%s' ]; then TYPE=directory; elif [ -L '%s' ]; then TYPE=link; elif [ -f '%s' ]; then TYPE=file; else TYPE=other; fi; SIZE=$(stat -c %%s '%s' 2>/dev/null || stat -f %%z '%s' 2>/dev/null); MODE=$(stat -c %%a '%s' 2>/dev/null || stat -f %%A '%s' 2>/dev/null); MTIME=$(stat -c %%Y '%s' 2>/dev/null || stat -f %%m '%s' 2>/dev/null); echo "exists=true|type=$TYPE|size=$SIZE|mode=$MODE|mtime=$MTIME"; else echo "exists=false"; fi`,
		path, path, path, path, path, path, path, path, path, path)

	output, err := m.executor.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("stat command failed: %v, output: %s", err, output)
	}

	output = strings.TrimSpace(output)
	statOutput := make(map[string]interface{})

	// Parse output
	if strings.HasPrefix(output, "exists=false") {
		return nil, fmt.Errorf("no such file or directory")
	}

	// Parse key=value pairs
	pairs := strings.Split(output, "|")
	data := make(map[string]string)
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			data[kv[0]] = kv[1]
		}
	}

	// Build output structure
	exists := data["exists"] == "true"
	statOutput["exists"] = exists
	statOutput["path"] = path

	if exists {
		fileType := data["type"]
		statOutput["isdir"] = fileType == "directory"
		statOutput["isreg"] = fileType == "file"
		statOutput["islnk"] = fileType == "link"

		if size, err := strconv.ParseInt(data["size"], 10, 64); err == nil {
			statOutput["size"] = size
		} else {
			statOutput["size"] = 0
		}

		mode := data["mode"]
		statOutput["mode"] = mode

		if mtime, err := strconv.ParseInt(data["mtime"], 10, 64); err == nil {
			statOutput["mtime"] = mtime
			statOutput["atime"] = mtime // Simplified
			statOutput["ctime"] = mtime // Simplified
		}

		// Parse mode for permissions
		if modeInt, err := strconv.ParseInt(mode, 8, 32); err == nil {
			statOutput["readable"] = (modeInt & 0400) != 0
			statOutput["writable"] = (modeInt & 0200) != 0
			statOutput["executable"] = (modeInt & 0100) != 0
		}

		// For Ansible compatibility
		statOutput["stat"] = map[string]interface{}{
			"exists":     exists,
			"path":       path,
			"isdir":      statOutput["isdir"],
			"isreg":      statOutput["isreg"],
			"islnk":      statOutput["islnk"],
			"size":       statOutput["size"],
			"mode":       mode,
			"mtime":      statOutput["mtime"],
			"readable":   statOutput["readable"],
			"writable":   statOutput["writable"],
			"executable": statOutput["executable"],
		}
	}

	return statOutput, nil
}

func (m *StatModule) Validate(args map[string]interface{}) error {
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

	return nil
}
