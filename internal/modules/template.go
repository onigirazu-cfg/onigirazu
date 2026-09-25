package modules

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
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

	if sshpkg.IsLocal(host) {
		return m.executeLocal(ctx, host, args, result, startTime)
	}

	// The module instance is shared by all hosts, so the connection is per call
	pool := sshpkg.GetGlobalPool()
	client, err := pool.GetConnection(host)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get SSH connection: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}
	defer pool.ReleaseConnection(host)

	return m.executeRemote(ctx, host, client, args, result, startTime)
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

	fileMode, err := parseFileMode(mode)
	if err != nil {
		result.Error = fmt.Sprintf("invalid file mode: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	changed := false
	if needsUpdate {
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

		if err := os.MkdirAll(filepath.Dir(dest), 0750); err != nil {
			result.Error = fmt.Sprintf("failed to create destination directory: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}

		if err := os.WriteFile(dest, []byte(renderedContent), fileMode); err != nil {
			result.Error = fmt.Sprintf("failed to write template to destination: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}
		changed = true
	}

	// os.WriteFile keeps the mode of an existing file, so an explicit mode is enforced here
	if _, modeSet := args["mode"]; modeSet {
		info, err := os.Stat(dest)
		if err == nil && info.Mode().Perm() != fileMode.Perm() {
			if err := os.Chmod(dest, fileMode); err != nil {
				result.Error = fmt.Sprintf("failed to set mode: %v", err)
				result.Duration = time.Since(startTime)
				return result, fmt.Errorf("%s", result.Error)
			}
			changed = true
		}
	}

	ownershipChanged, err := ensureOwnership(ctx, host, args, dest, owner, group)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}
	changed = changed || ownershipChanged

	return m.finishResult(result, startTime, changed, src, dest, renderedContent, newChecksum), nil
}

// executeRemote handles template operations on remote hosts via SFTP
func (m *TemplateModule) executeRemote(ctx context.Context, host types.Host, client *sshpkg.Client, args map[string]interface{}, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
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

	fileMode, err := parseFileMode(mode)
	if err != nil {
		result.Error = fmt.Sprintf("invalid file mode: %v", err)
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Stat and hash on the host, so root-only files work with become
	current, err := statRemoteFile(ctx, host, args, dest)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}
	needsUpdate := !current.Exists || current.SHA256 != newChecksum || force

	changed := false
	if needsUpdate {
		if backup && current.Exists {
			backupPath := dest + ".backup." + time.Now().Format("20060102-150405")
			if _, err := runOnHost(ctx, host, args, "cp", "-p", dest, backupPath); err != nil {
				result.Error = fmt.Sprintf("failed to create backup on remote host: %v", err)
				result.Duration = time.Since(startTime)
				return result, fmt.Errorf("%s", result.Error)
			}
			result.Output["backup_file"] = backupPath
		}
		// install(1) keeps the owner of an existing file and applies the mode
		writeMode := fileMode
		if _, modeSet := args["mode"]; !modeSet && current.Exists {
			writeMode = current.Mode
		}
		if err := installRemoteFile(ctx, host, args, client, dest, []byte(renderedContent), writeMode, current); err != nil {
			result.Error = err.Error()
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}
		changed = true
	} else if _, modeSet := args["mode"]; modeSet && current.Mode.Perm() != fileMode.Perm() {
		if _, err := runOnHost(ctx, host, args, "chmod", fmt.Sprintf("%04o", fileMode.Perm()), dest); err != nil {
			result.Error = fmt.Sprintf("failed to set mode: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}
		changed = true
	}

	ownershipChanged, err := ensureOwnership(ctx, host, args, dest, owner, group)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}
	changed = changed || ownershipChanged

	return m.finishResult(result, startTime, changed, src, dest, renderedContent, newChecksum), nil
}

// finishResult fills a successful template result
func (m *TemplateModule) finishResult(result types.TaskResult, startTime time.Time, changed bool, src, dest, rendered, checksum string) types.TaskResult {
	result.Success = true
	result.Changed = changed
	if changed {
		result.Output["message"] = "Template processed successfully"
	} else {
		result.Output["message"] = "Template is already up to date"
	}
	if src != "" {
		result.Output["src"] = src
	}
	result.Output["dest"] = dest
	result.Output["size"] = len(rendered)
	result.Output["checksum"] = checksum
	result.Duration = time.Since(startTime)
	return result
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
