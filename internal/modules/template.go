package modules

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TemplateModule handles template file processing
type TemplateModule struct {
	*BaseModule
	engine *template.Engine
}

// TemplateOptions holds template processing options
type TemplateOptions struct {
	TrimBlocks          bool   `json:"trim_blocks"`
	LStripBlocks        bool   `json:"lstrip_blocks"`
	KeepTrailingNewline bool   `json:"keep_trailing_newline"`
	BlockStartString    string `json:"block_start_string"`
	BlockEndString      string `json:"block_end_string"`
	VariableStartString string `json:"variable_start_string"`
	VariableEndString   string `json:"variable_end_string"`
	CommentStartString  string `json:"comment_start_string"`
	CommentEndString    string `json:"comment_end_string"`
}

// NewTemplateModule creates a new template module
func NewTemplateModule() *TemplateModule {
	return &TemplateModule{
		BaseModule: &BaseModule{
			name:        "template",
			description: "Process template files with variable substitution",
		},
		engine: template.NewEngine(),
	}
}

func (m *TemplateModule) GetDescription() string {
	return "Processes Jinja2-like templates with advanced features and creates files"
}

// Execute processes a template file
func (m *TemplateModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  "template",
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   false,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Validate required arguments
	src, ok := args["src"].(string)
	if !ok || src == "" {
		result.Error = "src parameter is required and must be a string"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	dest, ok := args["dest"].(string)
	if !ok || dest == "" {
		result.Error = "dest parameter is required and must be a string"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Get optional parameters
	backup := getBoolArg(args, "backup", false)
	mode := getStringArg(args, "mode", "0644")
	owner := getStringArg(args, "owner", "")
	group := getStringArg(args, "group", "")
	variables := getMapArg(args, "vars", make(map[string]interface{}))
	_ = getBoolArg(args, "force", false)      // force - not used in this implementation
	_ = getStringArg(args, "validate", "")    // validate - not used in this implementation

	// Template options
	_ = m.parseTemplateOptions(args) // options - not used in this implementation

	// Merge host variables
	allVars := make(map[string]interface{})
	for k, v := range host.Vars {
		allVars[k] = v
	}
	for k, v := range variables {
		allVars[k] = v
	}

	// Add host information to variables
	allVars["ansible_host"] = host.Address
	allVars["ansible_hostname"] = host.Name
	allVars["ansible_user"] = host.User
	allVars["ansible_port"] = host.Port

	// Add template functions
	allVars["template_functions"] = m.getTemplateFunctions()

	// Check if source template exists
	if _, err := os.Stat(src); os.IsNotExist(err) {
		result.Error = fmt.Sprintf("source template file does not exist: %s", src)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Render template
	renderedContent, err := m.engine.RenderFile(ctx, src, allVars)
	if err != nil {
		result.Error = fmt.Sprintf("failed to render template: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Check if destination file exists and compare content
	var needsUpdate bool
	var originalContent []byte

	if _, err := os.Stat(dest); err == nil {
		// File exists, read current content
		originalContent, err = ioutil.ReadFile(dest)
		if err != nil {
			result.Error = fmt.Sprintf("failed to read existing file: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}

		needsUpdate = string(originalContent) != renderedContent
	} else {
		// File doesn't exist, needs to be created
		needsUpdate = true
	}

	if !needsUpdate {
		// File is already up to date
		result.Success = true
		result.Changed = false
		result.Output["message"] = "Template is already up to date"
		result.Output["dest"] = dest
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Create backup if requested and file exists
	if backup && len(originalContent) > 0 {
		backupPath := dest + ".backup." + time.Now().Format("20060102-150405")
		if err := ioutil.WriteFile(backupPath, originalContent, 0644); err != nil {
			result.Error = fmt.Sprintf("failed to create backup: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}
		result.Output["backup_file"] = backupPath
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		result.Error = fmt.Sprintf("failed to create destination directory: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Write rendered content to destination
	if err := os.WriteFile(dest, []byte(renderedContent), 0644); err != nil {
		result.Error = fmt.Sprintf("failed to write template to destination: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Set file permissions
	if mode != "" {
		if err := m.setFileMode(dest, mode); err != nil {
			result.Error = fmt.Sprintf("failed to set file mode: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}
	}

	// Set file ownership (if specified and running as root)
	if owner != "" || group != "" {
		if err := m.setFileOwnership(dest, owner, group); err != nil {
			// Don't fail the task for ownership errors, just warn
			result.Output["ownership_warning"] = fmt.Sprintf("failed to set ownership: %v", err)
		}
	}

	result.Success = true
	result.Changed = true
	result.Output["message"] = "Template processed successfully"
	result.Output["src"] = src
	result.Output["dest"] = dest
	result.Output["size"] = len(renderedContent)
	result.Duration = time.Since(startTime)

	return result, nil
}

// Validate validates template module arguments
func (m *TemplateModule) Validate(args map[string]interface{}) error {
	// Check required arguments
	if _, ok := args["src"]; !ok {
		return fmt.Errorf("src parameter is required")
	}

	if _, ok := args["dest"]; !ok {
		return fmt.Errorf("dest parameter is required")
	}

	// Validate src is a string
	if src, ok := args["src"].(string); !ok || src == "" {
		return fmt.Errorf("src must be a non-empty string")
	}

	// Validate dest is a string
	if dest, ok := args["dest"].(string); !ok || dest == "" {
		return fmt.Errorf("dest must be a non-empty string")
	}

	// Validate optional parameters
	if mode, exists := args["mode"]; exists {
		if _, ok := mode.(string); !ok {
			return fmt.Errorf("mode must be a string")
		}
	}

	if owner, exists := args["owner"]; exists {
		if _, ok := owner.(string); !ok {
			return fmt.Errorf("owner must be a string")
		}
	}

	if group, exists := args["group"]; exists {
		if _, ok := group.(string); !ok {
			return fmt.Errorf("group must be a string")
		}
	}

	if backup, exists := args["backup"]; exists {
		if _, ok := backup.(bool); !ok {
			return fmt.Errorf("backup must be a boolean")
		}
	}

	if vars, exists := args["vars"]; exists {
		if _, ok := vars.(map[string]interface{}); !ok {
			return fmt.Errorf("vars must be a map")
		}
	}

	return nil
}

// setFileMode sets file permissions
func (m *TemplateModule) setFileMode(path, mode string) error {
	// Parse octal mode
	var perm os.FileMode
	switch mode {
	case "0644":
		perm = 0644
	case "0755":
		perm = 0755
	case "0600":
		perm = 0600
	case "0700":
		perm = 0700
	default:
		// Try to parse as octal
		if len(mode) == 4 && mode[0] == '0' {
			// Simple octal parsing for common cases
			switch mode {
			case "0644":
				perm = 0644
			case "0755":
				perm = 0755
			case "0600":
				perm = 0600
			case "0700":
				perm = 0700
			default:
				return fmt.Errorf("unsupported file mode: %s", mode)
			}
		} else {
			return fmt.Errorf("invalid file mode format: %s", mode)
		}
	}

	return os.Chmod(path, perm)
}

// setFileOwnership sets file ownership (simplified implementation)
func (m *TemplateModule) setFileOwnership(path, owner, group string) error {
	// This is a simplified implementation
	// In a real implementation, you would use os/user package to resolve
	// user/group names to UIDs/GIDs and use syscall.Chown
	return fmt.Errorf("ownership setting not implemented in this simplified version")
}

// parseTemplateOptions parses template-specific options
func (m *TemplateModule) parseTemplateOptions(args map[string]interface{}) map[string]interface{} {
	options := make(map[string]interface{})

	// Template processing options
	if _, exists := args["trim_blocks"]; exists {
		options["trim_blocks"] = getBoolArg(args, "trim_blocks", false)
	}

	if _, exists := args["lstrip_blocks"]; exists {
		options["lstrip_blocks"] = getBoolArg(args, "lstrip_blocks", false)
	}

	if _, exists := args["keep_trailing_newline"]; exists {
		options["keep_trailing_newline"] = getBoolArg(args, "keep_trailing_newline", true)
	}

	// Custom delimiters
	if _, exists := args["block_start_string"]; exists {
		options["block_start_string"] = getStringArg(args, "block_start_string", "{%")
	}

	if _, exists := args["block_end_string"]; exists {
		options["block_end_string"] = getStringArg(args, "block_end_string", "%}")
	}

	if _, exists := args["variable_start_string"]; exists {
		options["variable_start_string"] = getStringArg(args, "variable_start_string", "{{")
	}

	if _, exists := args["variable_end_string"]; exists {
		options["variable_end_string"] = getStringArg(args, "variable_end_string", "}}")
	}

	return options
}

// getTemplateFunctions returns available template functions
func (m *TemplateModule) getTemplateFunctions() map[string]interface{} {
	return map[string]interface{}{
		// String functions
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"title":     strings.Title,
		"trim":      strings.TrimSpace,
		"replace":   strings.ReplaceAll,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"split":     strings.Split,
		"join":      strings.Join,

		// Utility functions
		"default": func(value, defaultValue interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},

		"length": func(value interface{}) int {
			switch v := value.(type) {
			case string:
				return len(v)
			case []interface{}:
				return len(v)
			case map[string]interface{}:
				return len(v)
			default:
				return 0
			}
		},

		"keys": func(m map[string]interface{}) []string {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			return keys
		},

		"values": func(m map[string]interface{}) []interface{} {
			values := make([]interface{}, 0, len(m))
			for _, v := range m {
				values = append(values, v)
			}
			return values
		},

		// Type checking functions
		"isString": func(value interface{}) bool {
			_, ok := value.(string)
			return ok
		},

		"isNumber": func(value interface{}) bool {
			switch value.(type) {
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
				return true
			default:
				return false
			}
		},

		"isBool": func(value interface{}) bool {
			_, ok := value.(bool)
			return ok
		},

		"isList": func(value interface{}) bool {
			_, ok := value.([]interface{})
			return ok
		},

		"isDict": func(value interface{}) bool {
			_, ok := value.(map[string]interface{})
			return ok
		},

		// Math functions
		"add": func(a, b interface{}) interface{} {
			switch va := a.(type) {
			case int:
				if vb, ok := b.(int); ok {
					return va + vb
				}
			case float64:
				if vb, ok := b.(float64); ok {
					return va + vb
				}
			}
			return nil
		},

		"subtract": func(a, b interface{}) interface{} {
			switch va := a.(type) {
			case int:
				if vb, ok := b.(int); ok {
					return va - vb
				}
			case float64:
				if vb, ok := b.(float64); ok {
					return va - vb
				}
			}
			return nil
		},

		"multiply": func(a, b interface{}) interface{} {
			switch va := a.(type) {
			case int:
				if vb, ok := b.(int); ok {
					return va * vb
				}
			case float64:
				if vb, ok := b.(float64); ok {
					return va * vb
				}
			}
			return nil
		},

		"divide": func(a, b interface{}) interface{} {
			switch va := a.(type) {
			case int:
				if vb, ok := b.(int); ok && vb != 0 {
					return va / vb
				}
			case float64:
				if vb, ok := b.(float64); ok && vb != 0 {
					return va / vb
				}
			}
			return nil
		},
	}
}
