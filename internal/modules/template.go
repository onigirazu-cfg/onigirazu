package modules

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TemplateModule handles template file processing
type TemplateModule struct {
	*BaseModule
	engine   *template.Engine
	executor *sshpkg.Client
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

	// Check if we need to get/cache SSH executor for this host
	if m.executor == nil {
		if !sshpkg.IsLocal(host) {
			pool := sshpkg.GetGlobalPool()
			executor, err := pool.GetConnection(host)
			if err != nil {
				result.Error = fmt.Sprintf("failed to get SSH connection: %v", err)
				result.Duration = time.Since(startTime)
				return result, fmt.Errorf("%s", result.Error)
			}
			m.executor = executor
		}
	}

	// Determine if this is a local or remote operation
	if sshpkg.IsLocal(host) {
		return m.executeLocal(ctx, host, args, result, startTime)
	}
	return m.executeRemote(ctx, host, args, result, startTime)
}

// executeLocal handles template operations on localhost
func (m *TemplateModule) executeLocal(ctx context.Context, host types.Host, args map[string]interface{}, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Get parameters
	src := getStringArg(args, "src", "")
	content := getStringArg(args, "content", "")
	dest, ok := args["dest"].(string)
	if !ok || dest == "" {
		result.Error = "dest parameter is required and must be a string"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Either src or content must be provided
	if src == "" && content == "" {
		result.Error = "either src or content parameter is required"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Get optional parameters
	backup := getBoolArg(args, "backup", false)
	mode := getStringArg(args, "mode", "0644")
	owner := getStringArg(args, "owner", "")
	group := getStringArg(args, "group", "")
	variables := getMapArg(args, "vars", make(map[string]interface{}))
	force := getBoolArg(args, "force", false)

	// Merge host variables
	allVars := make(map[string]interface{})
	for k, v := range host.Vars {
		allVars[k] = v
	}
	for k, v := range variables {
		allVars[k] = v
	}

	// Add host information to variables
	allVars["onigirazu_host"] = host.Address
	allVars["onigirazu_hostname"] = host.Name
	allVars["onigirazu_user"] = host.User
	allVars["onigirazu_port"] = host.Port

	// Render template
	var renderedContent string
	var err error

	if content != "" {
		// Render inline content
		renderedContent, err = m.engine.Render(ctx, content, allVars)
		if err != nil {
			result.Error = fmt.Sprintf("failed to render template content: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}
	} else {
		// Check if source template exists
		if _, err := os.Stat(src); os.IsNotExist(err) {
			result.Error = fmt.Sprintf("source template file does not exist: %s", src)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}

		// Render template file
		renderedContent, err = m.engine.RenderFile(ctx, src, allVars)
		if err != nil {
			result.Error = fmt.Sprintf("failed to render template: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}
	}

	// Calculate checksum of rendered content
	newChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte(renderedContent)))

	// Check if destination file exists and compare content
	var needsUpdate bool
	var originalContent []byte
	var oldChecksum string

	if _, err := os.Stat(dest); err == nil {
		// File exists, read current content
		originalContent, err = os.ReadFile(dest) // #nosec G304 -- dest is validated by security validator
		if err != nil {
			result.Error = fmt.Sprintf("failed to read existing file: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}

		oldChecksum = fmt.Sprintf("%x", sha256.Sum256(originalContent))
		needsUpdate = oldChecksum != newChecksum || force
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
		result.Output["checksum"] = oldChecksum
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Create backup if requested and file exists
	if backup && len(originalContent) > 0 {
		backupPath := dest + ".backup." + time.Now().Format("20060102-150405")
		if err := os.WriteFile(backupPath, originalContent, 0600); err != nil {
			result.Error = fmt.Sprintf("failed to create backup: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}
		result.Output["backup_file"] = backupPath
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0750); err != nil {
		result.Error = fmt.Sprintf("failed to create destination directory: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Parse file mode
	fileMode, err := parseFileMode(mode)
	if err != nil {
		result.Error = fmt.Sprintf("invalid file mode: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Write rendered content to destination
	if err := os.WriteFile(dest, []byte(renderedContent), fileMode); err != nil {
		result.Error = fmt.Sprintf("failed to write template to destination: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
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
	if src != "" {
		result.Output["src"] = src
	}
	result.Output["dest"] = dest
	result.Output["size"] = len(renderedContent)
	result.Output["checksum"] = newChecksum
	result.Duration = time.Since(startTime)

	return result, nil
}

// executeRemote handles template operations on remote hosts via SFTP
func (m *TemplateModule) executeRemote(ctx context.Context, host types.Host, args map[string]interface{}, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Get parameters
	src := getStringArg(args, "src", "")
	content := getStringArg(args, "content", "")
	dest, ok := args["dest"].(string)
	if !ok || dest == "" {
		result.Error = "dest parameter is required and must be a string"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Either src or content must be provided
	if src == "" && content == "" {
		result.Error = "either src or content parameter is required"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Get optional parameters
	backup := getBoolArg(args, "backup", false)
	mode := getStringArg(args, "mode", "0644")
	variables := getMapArg(args, "vars", make(map[string]interface{}))
	force := getBoolArg(args, "force", false)

	// Merge host variables
	allVars := make(map[string]interface{})
	for k, v := range host.Vars {
		allVars[k] = v
	}
	for k, v := range variables {
		allVars[k] = v
	}

	// Add host information to variables
	allVars["onigirazu_host"] = host.Address
	allVars["onigirazu_hostname"] = host.Name
	allVars["onigirazu_user"] = host.User
	allVars["onigirazu_port"] = host.Port

	// Render template
	var renderedContent string
	var err error

	if content != "" {
		// Render inline content
		renderedContent, err = m.engine.Render(ctx, content, allVars)
		if err != nil {
			result.Error = fmt.Sprintf("failed to render template content: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}
	} else {
		// Check if source template exists locally
		if _, err := os.Stat(src); os.IsNotExist(err) {
			result.Error = fmt.Sprintf("source template file does not exist: %s", src)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}

		// Render template file
		renderedContent, err = m.engine.RenderFile(ctx, src, allVars)
		if err != nil {
			result.Error = fmt.Sprintf("failed to render template: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}
	}

	// Calculate checksum of rendered content
	newChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte(renderedContent)))

	// Check if destination file exists on remote host
	var needsUpdate bool
	var oldChecksum string

	remoteFileInfo, err := m.executor.StatFile(dest)
	if err == nil {
		// File exists on remote host, read and compare
		remoteContent, err := m.executor.ReadFile(dest)
		if err != nil {
			result.Error = fmt.Sprintf("failed to read remote file: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}

		oldChecksum = fmt.Sprintf("%x", sha256.Sum256(remoteContent))
		needsUpdate = oldChecksum != newChecksum || force

		// Create backup if requested
		if backup && needsUpdate {
			backupPath := dest + ".backup." + time.Now().Format("20060102-150405")
			if err := m.executor.WriteFile(backupPath, remoteContent, remoteFileInfo.Mode()); err != nil {
				result.Error = fmt.Sprintf("failed to create backup on remote host: %v", err)
				result.Duration = time.Since(startTime)
				return result, fmt.Errorf("%s", result.Error)
			}
			result.Output["backup_file"] = backupPath
		}
	} else {
		// File doesn't exist on remote host
		needsUpdate = true
	}

	if !needsUpdate {
		// File is already up to date
		result.Success = true
		result.Changed = false
		result.Output["message"] = "Template is already up to date"
		result.Output["dest"] = dest
		result.Output["checksum"] = oldChecksum
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Parse file mode
	fileMode, err := parseFileMode(mode)
	if err != nil {
		result.Error = fmt.Sprintf("invalid file mode: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Write rendered content to remote destination
	if err := m.executor.WriteFile(dest, []byte(renderedContent), fileMode); err != nil {
		result.Error = fmt.Sprintf("failed to write template to remote destination: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	result.Success = true
	result.Changed = true
	result.Output["message"] = "Template processed successfully on remote host"
	if src != "" {
		result.Output["src"] = src
	}
	result.Output["dest"] = dest
	result.Output["size"] = len(renderedContent)
	result.Output["checksum"] = newChecksum
	result.Duration = time.Since(startTime)

	return result, nil
}

// Validate validates template module arguments
func (m *TemplateModule) Validate(args map[string]interface{}) error {
	// Check that either src or content is provided
	src, hasSrc := args["src"]
	content, hasContent := args["content"]

	if !hasSrc && !hasContent {
		return fmt.Errorf("either src or content parameter is required")
	}

	// Validate src if provided
	if hasSrc {
		if srcStr, ok := src.(string); !ok || srcStr == "" {
			return fmt.Errorf("src must be a non-empty string")
		}
	}

	// Validate content if provided
	if hasContent {
		if contentStr, ok := content.(string); !ok || contentStr == "" {
			return fmt.Errorf("content must be a non-empty string")
		}
	}

	// Check required dest argument
	if _, ok := args["dest"]; !ok {
		return fmt.Errorf("dest parameter is required")
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

// parseFileMode parses file mode string to os.FileMode
func parseFileMode(mode string) (os.FileMode, error) {
	if mode == "" {
		return 0644, nil
	}

	modeInt, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return 0644, fmt.Errorf("invalid mode %s: %v", mode, err)
	}

	return os.FileMode(modeInt), nil
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

// toTitle converts string to title case (first letter of each word capitalized)
func toTitle(s string) string {
	if s == "" {
		return s
	}

	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// getTemplateFunctions returns available template functions
func (m *TemplateModule) getTemplateFunctions() map[string]interface{} {
	return map[string]interface{}{
		// String functions
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"title":     toTitle,
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
