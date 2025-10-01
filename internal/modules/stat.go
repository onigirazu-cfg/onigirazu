package modules

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// StatModule retrieves file or directory status
type StatModule struct {
	*BaseModule
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

	path := args["path"].(string)

	// Get file info
	fileInfo, err := os.Stat(path)

	// Prepare stat output
	statOutput := make(map[string]interface{})

	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist
			statOutput["exists"] = false
			statOutput["path"] = path
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

	// File exists - populate all information
	statOutput["exists"] = true
	statOutput["path"] = path
	statOutput["isdir"] = fileInfo.IsDir()
	statOutput["isreg"] = fileInfo.Mode().IsRegular()
	statOutput["islnk"] = fileInfo.Mode()&os.ModeSymlink != 0
	statOutput["size"] = fileInfo.Size()
	statOutput["mode"] = fmt.Sprintf("%04o", fileInfo.Mode().Perm())
	statOutput["mtime"] = fileInfo.ModTime().Unix()
	statOutput["atime"] = fileInfo.ModTime().Unix() // Go doesn't provide atime easily
	statOutput["ctime"] = fileInfo.ModTime().Unix() // Go doesn't provide ctime easily

	// Readable/writable/executable checks (simplified)
	mode := fileInfo.Mode()
	statOutput["readable"] = mode&0400 != 0
	statOutput["writable"] = mode&0200 != 0
	statOutput["executable"] = mode&0100 != 0

	// For Ansible compatibility
	statOutput["stat"] = map[string]interface{}{
		"exists":     true,
		"path":       path,
		"isdir":      fileInfo.IsDir(),
		"isreg":      fileInfo.Mode().IsRegular(),
		"islnk":      fileInfo.Mode()&os.ModeSymlink != 0,
		"size":       fileInfo.Size(),
		"mode":       fmt.Sprintf("%04o", fileInfo.Mode().Perm()),
		"mtime":      fileInfo.ModTime().Unix(),
		"readable":   mode&0400 != 0,
		"writable":   mode&0200 != 0,
		"executable": mode&0100 != 0,
	}

	result.Success = true
	result.Changed = false // stat never changes anything
	result.Output = statOutput
	result.Duration = time.Since(startTime)

	return result, nil
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
