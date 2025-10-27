package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// BlockinfileModule inserts/updates/removes text blocks in files
type BlockinfileModule struct {
	*BaseModule
}

// NewBlockinfileModule creates a new blockinfile module
func NewBlockinfileModule() *BlockinfileModule {
	return &BlockinfileModule{
		BaseModule: NewBaseModule("blockinfile"),
	}
}

func (m *BlockinfileModule) GetDescription() string {
	return "Insert/update/remove a text block in a file using markers"
}

func (m *BlockinfileModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
	}

	// Get parameters
	filePath := ""
	if pathVal, exists := args["path"]; exists {
		if pathStr, ok := pathVal.(string); ok {
			filePath = pathStr
		}
	}

	if filePath == "" {
		result.Success = false
		result.Error = "'path' parameter is required"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Make path absolute
	if !filepath.IsAbs(filePath) {
		var err error
		filePath, err = filepath.Abs(filePath)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to resolve path: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
	}

	block := ""
	if blockVal, exists := args["block"]; exists {
		if blockStr, ok := blockVal.(string); ok {
			block = blockStr
		}
	}

	marker := "# {mark} ANSIBLE MANAGED BLOCK"
	if markerVal, exists := args["marker"]; exists {
		if markerStr, ok := markerVal.(string); ok {
			marker = markerStr
		}
	}

	state := "present"
	if stateVal, exists := args["state"]; exists {
		if stateStr, ok := stateVal.(string); ok {
			state = stateStr
		}
	}

	insertAfter := ""
	if afterVal, exists := args["insertafter"]; exists {
		if afterStr, ok := afterVal.(string); ok {
			insertAfter = afterStr
		}
	}

	insertBefore := ""
	if beforeVal, exists := args["insertbefore"]; exists {
		if beforeStr, ok := beforeVal.(string); ok {
			insertBefore = beforeStr
		}
	}

	backup := false
	if backupVal, exists := args["backup"]; exists {
		if backupBool, ok := backupVal.(bool); ok {
			backup = backupBool
		}
	}

	// Read file content
	fileContent := ""
	fileExists := true
	if data, err := os.ReadFile(filePath); err == nil {
		fileContent = string(data)
	} else if os.IsNotExist(err) {
		fileExists = false
	} else {
		result.Success = false
		result.Error = fmt.Sprintf("failed to read file: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Create backup if requested
	if backup && fileExists {
		backupPath := filePath + ".bak"
		if err := os.WriteFile(backupPath, []byte(fileContent), 0644); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create backup: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		result.Output["backup"] = backupPath
	}

	// Create marker patterns
	markerBegin := strings.ReplaceAll(marker, "{mark}", "BEGIN")
	markerEnd := strings.ReplaceAll(marker, "{mark}", "END")

	// Process content based on state
	newContent := fileContent
	blockBeginPattern := regexp.MustCompile("(?m)^.*" + regexp.QuoteMeta(markerBegin) + ".*$")
	blockEndPattern := regexp.MustCompile("(?m)^.*" + regexp.QuoteMeta(markerEnd) + ".*$")

	if state == "present" {
		// Check if block already exists
		if blockBeginPattern.MatchString(fileContent) {
			// Replace existing block
			// Find positions of markers
			beginIdx := blockBeginPattern.FindStringIndex(fileContent)
			endIdx := blockEndPattern.FindStringIndex(fileContent)

			if beginIdx != nil && endIdx != nil && beginIdx[0] < endIdx[0] {
				// Replace content between markers
				newBlockContent := fmt.Sprintf("%s\n%s\n%s\n", markerBegin, block, markerEnd)
				newContent = fileContent[:beginIdx[0]] + newBlockContent + fileContent[endIdx[1]:]

				if newContent != fileContent {
					result.Changed = true
				}
			}
		} else {
			// Add new block
			newBlockContent := fmt.Sprintf("%s\n%s\n%s\n", markerBegin, block, markerEnd)

			if insertBefore != "" {
				// Insert before pattern
				pattern := regexp.MustCompile("(?m)^.*" + regexp.QuoteMeta(insertBefore) + ".*$")
				if matches := pattern.FindStringIndex(fileContent); matches != nil {
					newContent = fileContent[:matches[0]] + newBlockContent + fileContent[matches[0]:]
					result.Changed = true
				}
			} else if insertAfter != "" {
				// Insert after pattern
				pattern := regexp.MustCompile("(?m)^.*" + regexp.QuoteMeta(insertAfter) + ".*$")
				if matches := pattern.FindStringIndex(fileContent); matches != nil {
					newContent = fileContent[:matches[1]] + "\n" + newBlockContent + fileContent[matches[1]:]
					result.Changed = true
				}
			} else {
				// Append at end
				if fileContent != "" && !strings.HasSuffix(fileContent, "\n") {
					newContent = fileContent + "\n" + newBlockContent
				} else {
					newContent = fileContent + newBlockContent
				}
				result.Changed = true
			}
		}
	} else if state == "absent" {
		// Remove block
		if blockBeginPattern.MatchString(fileContent) {
			beginIdx := blockBeginPattern.FindStringIndex(fileContent)
			endIdx := blockEndPattern.FindStringIndex(fileContent)

			if beginIdx != nil && endIdx != nil && beginIdx[0] < endIdx[0] {
				newContent = fileContent[:beginIdx[0]] + fileContent[endIdx[1]:]
				// Remove trailing newline if it's there
				newContent = strings.TrimSuffix(newContent, "\n") + "\n"
				result.Changed = true
			}
		}
	}

	// Write file if changed
	if result.Changed {
		// Ensure parent directory exists
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create directory: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to write file: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
	}

	result.Output["path"] = filePath
	result.Output["state"] = state
	result.Output["msg"] = fmt.Sprintf("Block %s", state)

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *BlockinfileModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	if _, exists := args["path"]; !exists {
		return fmt.Errorf("blockinfile module requires 'path' parameter")
	}

	return nil
}
