package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ConfigModule implements configuration file management
type ConfigModule struct {
	BaseModule
	executor *executor.CommandExecutor
}

// NewConfigModule creates a new config module
func NewConfigModule() *ConfigModule {
	return &ConfigModule{
		BaseModule: BaseModule{
			name:        "config",
			description: "Manage configuration files with validation and backup",
		},
	}
}

// ConfigAction represents the action to perform
type ConfigAction string

const (
	ConfigActionSet      ConfigAction = "set"
	ConfigActionGet      ConfigAction = "get"
	ConfigActionDelete   ConfigAction = "delete"
	ConfigActionMerge    ConfigAction = "merge"
	ConfigActionBackup   ConfigAction = "backup"
	ConfigActionRestore  ConfigAction = "restore"
	ConfigActionValidate ConfigAction = "validate"
)

// ConfigFormat represents supported configuration formats
type ConfigFormat string

const (
	FormatJSON ConfigFormat = "json"
	FormatYAML ConfigFormat = "yaml"
	FormatINI  ConfigFormat = "ini"
	FormatTOML ConfigFormat = "toml"
	FormatXML  ConfigFormat = "xml"
)

// Execute manages configuration files
func (m *ConfigModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "config",
		Host:      host.Name,
		Module:    m.name,
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Initialize executor
	var err error
	m.executor, err = executor.NewCommandExecutor(host)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to create executor: %v", err))
	}

	// Get required parameters
	path, ok := args["path"].(string)
	if !ok {
		return m.failResult(result, "path parameter is required")
	}

	action := ConfigAction(getStringArg(args, "action", "set"))
	format := ConfigFormat(getStringArg(args, "format", "json"))

	// Execute based on action
	switch action {
	case ConfigActionSet:
		return m.executeSet(ctx, result, path, format, args)
	case ConfigActionGet:
		return m.executeGet(ctx, result, path, format, args)
	case ConfigActionDelete:
		return m.executeDelete(ctx, result, path, args)
	case ConfigActionMerge:
		return m.executeMerge(ctx, result, path, format, args)
	case ConfigActionBackup:
		return m.executeBackup(ctx, result, path, args)
	case ConfigActionRestore:
		return m.executeRestore(ctx, result, path, args)
	case ConfigActionValidate:
		return m.executeValidate(ctx, result, path, format, args)
	default:
		return m.failResult(result, fmt.Sprintf("unsupported action: %s", action))
	}
}

// executeSet sets configuration values
func (m *ConfigModule) executeSet(ctx context.Context, result types.TaskResult, path string, format ConfigFormat, args map[string]interface{}) (types.TaskResult, error) {
	// Create backup if requested
	if getBoolArg(args, "backup", false) {
		backupPath, err := m.createBackup(path)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to create backup: %v", err))
		}
		if backupPath != "" {
			result.Output["backup_path"] = backupPath
		}
		result.Output["backup_created"] = true
	}

	// Load existing config or create new
	config := make(map[string]interface{})
	if m.checkFileExists(path) {
		existingConfig, err := m.loadConfig(path, format)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to load existing config: %v", err))
		}
		config = existingConfig
	}

	// Apply new values - support both "values" map and "key"+"value" pair
	var values map[string]interface{}

	if valuesMap, ok := args["values"].(map[string]interface{}); ok {
		// Use values map directly
		values = valuesMap
	} else if key, keyOk := args["key"].(string); keyOk {
		// Use key+value pair
		if value, valueOk := args["value"]; valueOk {
			values = map[string]interface{}{key: value}
		} else {
			return m.failResult(result, "value parameter is required when using key parameter")
		}
	} else {
		return m.failResult(result, "either 'values' map or 'key'+'value' parameters are required for set action")
	}

	originalConfig := m.deepCopy(config)
	m.mergeConfig(config, values)

	// Check if config changed
	if !reflect.DeepEqual(originalConfig, config) {
		result.Changed = true
	}

	// Save config
	if err := m.saveConfig(path, format, config); err != nil {
		return m.failResult(result, fmt.Sprintf("failed to save config: %v", err))
	}

	// Validate if schema provided
	if schema, exists := args["schema"]; exists {
		if err := m.validateConfig(config, schema); err != nil {
			return m.failResult(result, fmt.Sprintf("config validation failed: %v", err))
		}
		result.Output["validation"] = "passed"
	}

	result.Output["config"] = config
	result.Output["path"] = path
	result.Duration = time.Since(result.Timestamp)

	return result, nil
}

// executeGet retrieves configuration values
func (m *ConfigModule) executeGet(ctx context.Context, result types.TaskResult, path string, format ConfigFormat, args map[string]interface{}) (types.TaskResult, error) {
	config, err := m.loadConfig(path, format)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to load config: %v", err))
	}

	// Get specific key if provided
	if key, exists := args["key"]; exists {
		if keyStr, ok := key.(string); ok {
			value := m.getNestedValue(config, keyStr)
			result.Output["value"] = value
			result.Output["key"] = keyStr
		}
	} else {
		result.Output["config"] = config
	}

	result.Output["path"] = path
	result.Duration = time.Since(result.Timestamp)

	return result, nil
}

// executeDelete deletes configuration keys or files
func (m *ConfigModule) executeDelete(ctx context.Context, result types.TaskResult, path string, args map[string]interface{}) (types.TaskResult, error) {
	if key, exists := args["key"]; exists {
		// Delete specific key
		format := ConfigFormat(getStringArg(args, "format", "json"))
		config, err := m.loadConfig(path, format)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to load config: %v", err))
		}

		keyStr := key.(string)
		if m.deleteNestedKey(config, keyStr) {
			result.Changed = true
			if err := m.saveConfig(path, format, config); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to save config: %v", err))
			}
		}

		result.Output["deleted_key"] = keyStr
	} else {
		// Delete entire file
		if getBoolArg(args, "backup", false) {
			backupPath, err := m.createBackup(path)
			if err != nil {
				return m.failResult(result, fmt.Sprintf("failed to create backup: %v", err))
			}
			if backupPath != "" {
				result.Output["backup_path"] = backupPath
			}
			result.Output["backup_created"] = true
		}

		if err := m.removeRemoteFile(path); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to delete file: %v", err))
		}
		result.Changed = true
		result.Output["deleted_file"] = path
	}

	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// executeMerge merges configuration with existing values
func (m *ConfigModule) executeMerge(ctx context.Context, result types.TaskResult, path string, format ConfigFormat, args map[string]interface{}) (types.TaskResult, error) {
	config, err := m.loadConfig(path, format)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to load config: %v", err))
	}

	values, ok := args["values"].(map[string]interface{})
	if !ok {
		return m.failResult(result, "values parameter is required for merge action")
	}

	originalConfig := m.deepCopy(config)
	m.mergeConfig(config, values)

	if !reflect.DeepEqual(originalConfig, config) {
		result.Changed = true
		if err := m.saveConfig(path, format, config); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to save config: %v", err))
		}
	}

	result.Output["config"] = config
	result.Duration = time.Since(result.Timestamp)

	return result, nil
}

// executeBackup creates a backup of the configuration file
func (m *ConfigModule) executeBackup(ctx context.Context, result types.TaskResult, path string, args map[string]interface{}) (types.TaskResult, error) {
	backupPath, err := m.createBackup(path)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to create backup: %v", err))
	}

	result.Changed = true
	result.Output["backup_path"] = backupPath
	result.Output["original_path"] = path
	result.Duration = time.Since(result.Timestamp)

	return result, nil
}

// executeRestore restores from a backup
func (m *ConfigModule) executeRestore(ctx context.Context, result types.TaskResult, path string, args map[string]interface{}) (types.TaskResult, error) {
	backupPath, ok := args["backup_path"].(string)
	if !ok {
		return m.failResult(result, "backup_path parameter is required for restore action")
	}

	// Copy backup to original location
	data, err := m.readRemoteFile(backupPath)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to read backup: %v", err))
	}

	if err := m.writeRemoteFile(path, data); err != nil {
		return m.failResult(result, fmt.Sprintf("failed to restore config: %v", err))
	}

	result.Changed = true
	result.Output["restored_from"] = backupPath
	result.Output["restored_to"] = path
	result.Duration = time.Since(result.Timestamp)

	return result, nil
}

// executeValidate validates configuration against schema
func (m *ConfigModule) executeValidate(ctx context.Context, result types.TaskResult, path string, format ConfigFormat, args map[string]interface{}) (types.TaskResult, error) {
	config, err := m.loadConfig(path, format)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to load config: %v", err))
	}

	schema, exists := args["schema"]
	if !exists {
		return m.failResult(result, "schema parameter is required for validate action")
	}

	if err := m.validateConfig(config, schema); err != nil {
		result.Output["validation"] = "failed"
		result.Output["validation_error"] = err.Error()
		result.Duration = time.Since(result.Timestamp)
		return result, nil
	}

	result.Output["validation"] = "passed"
	result.Output["config"] = config
	result.Duration = time.Since(result.Timestamp)

	return result, nil
}

// Validate validates config module arguments
func (m *ConfigModule) Validate(args map[string]interface{}) error {
	if _, exists := args["path"]; !exists {
		return fmt.Errorf("path parameter is required")
	}

	action := getStringArg(args, "action", "set")
	switch ConfigAction(action) {
	case ConfigActionSet, ConfigActionMerge:
		if _, exists := args["values"]; !exists {
			return fmt.Errorf("values parameter is required for %s action", action)
		}
	case ConfigActionRestore:
		if _, exists := args["backup_path"]; !exists {
			return fmt.Errorf("backup_path parameter is required for restore action")
		}
	}

	return nil
}

// loadConfig loads configuration from file
func (m *ConfigModule) loadConfig(path string, format ConfigFormat) (map[string]interface{}, error) {
	data, err := m.readRemoteFile(path)
	if err != nil {
		return nil, err
	}

	config := make(map[string]interface{})

	switch format {
	case FormatJSON:
		err = json.Unmarshal([]byte(data), &config)
	case FormatYAML:
		err = yaml.Unmarshal([]byte(data), &config)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	return config, err
}

// saveConfig saves configuration to file
func (m *ConfigModule) saveConfig(path string, format ConfigFormat, config map[string]interface{}) error {
	var data []byte
	var err error

	switch format {
	case FormatJSON:
		data, err = json.MarshalIndent(config, "", "  ")
	case FormatYAML:
		data, err = yaml.Marshal(config)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return err
	}

	return m.writeRemoteFile(path, string(data))
}

// createBackup creates a backup of the file
func (m *ConfigModule) createBackup(path string) (string, error) {
	if !m.checkFileExists(path) {
		return "", nil // No file to backup
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.backup.%s", path, timestamp)

	data, err := m.readRemoteFile(path)
	if err != nil {
		return "", err
	}

	err = m.writeRemoteFile(backupPath, data)
	return backupPath, err
}

// Helper methods for remote file operations

// checkFileExists checks if a file exists on the remote host
func (m *ConfigModule) checkFileExists(path string) bool {
	escapedPath := strings.ReplaceAll(path, "'", "'\\''")
	cmd := fmt.Sprintf(`test -e '%s' && echo exists || echo notexists`, escapedPath)
	output, err := m.executor.Execute(cmd)
	return err == nil && strings.TrimSpace(output) == "exists"
}

// readRemoteFile reads a file from the remote host
func (m *ConfigModule) readRemoteFile(path string) (string, error) {
	escapedPath := strings.ReplaceAll(path, "'", "'\\''")
	cmd := fmt.Sprintf(`cat '%s'`, escapedPath)
	output, err := m.executor.Execute(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}
	return output, nil
}

// writeRemoteFile writes content to a file on the remote host
func (m *ConfigModule) writeRemoteFile(path string, content string) error {
	escapedPath := strings.ReplaceAll(path, "'", "'\\''")
	escapedContent := strings.ReplaceAll(content, "'", "'\\''")

	// Create directory if needed
	dir := filepath.Dir(path)
	escapedDir := strings.ReplaceAll(dir, "'", "'\\''")
	mkdirCmd := fmt.Sprintf(`mkdir -p '%s'`, escapedDir)
	if _, err := m.executor.Execute(mkdirCmd); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Write file
	cmd := fmt.Sprintf(`printf '%%s' '%s' > '%s'`, escapedContent, escapedPath)
	if _, err := m.executor.Execute(cmd); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	return nil
}

// removeRemoteFile removes a file from the remote host
func (m *ConfigModule) removeRemoteFile(path string) error {
	escapedPath := strings.ReplaceAll(path, "'", "'\\''")
	cmd := fmt.Sprintf(`rm -f '%s'`, escapedPath)
	if _, err := m.executor.Execute(cmd); err != nil {
		return fmt.Errorf("failed to remove file: %v", err)
	}
	return nil
}

// mergeConfig merges source into target
func (m *ConfigModule) mergeConfig(target, source map[string]interface{}) {
	for key, value := range source {
		if existingValue, exists := target[key]; exists {
			if existingMap, ok := existingValue.(map[string]interface{}); ok {
				if sourceMap, ok := value.(map[string]interface{}); ok {
					m.mergeConfig(existingMap, sourceMap)
					continue
				}
			}
		}
		target[key] = value
	}
}

// getNestedValue gets a nested value using dot notation
func (m *ConfigModule) getNestedValue(config map[string]interface{}, key string) interface{} {
	keys := strings.Split(key, ".")
	current := config

	for i, k := range keys {
		if i == len(keys)-1 {
			return current[k]
		}

		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}

	return nil
}

// deleteNestedKey deletes a nested key using dot notation
func (m *ConfigModule) deleteNestedKey(config map[string]interface{}, key string) bool {
	keys := strings.Split(key, ".")
	current := config

	for i, k := range keys {
		if i == len(keys)-1 {
			if _, exists := current[k]; exists {
				delete(current, k)
				return true
			}
			return false
		}

		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			return false
		}
	}

	return false
}

// validateConfig validates configuration against schema (simplified)
func (m *ConfigModule) validateConfig(config map[string]interface{}, schema interface{}) error {
	// This is a simplified validation
	// In a real implementation, you would use a proper JSON schema validator
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return fmt.Errorf("schema must be a map")
	}

	return m.validateAgainstSchema(config, schemaMap)
}

// validateAgainstSchema performs basic schema validation
func (m *ConfigModule) validateAgainstSchema(config, schema map[string]interface{}) error {
	required, ok := schema["required"].([]interface{})
	if ok {
		for _, req := range required {
			if reqStr, ok := req.(string); ok {
				if _, exists := config[reqStr]; !exists {
					return fmt.Errorf("required field '%s' is missing", reqStr)
				}
			}
		}
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if ok {
		for key, value := range config {
			if propSchema, exists := properties[key]; exists {
				if propMap, ok := propSchema.(map[string]interface{}); ok {
					if err := m.validateValue(value, propMap); err != nil {
						return fmt.Errorf("validation failed for field '%s': %v", key, err)
					}
				}
			}
		}
	}

	return nil
}

// validateValue validates a single value against its schema
func (m *ConfigModule) validateValue(value interface{}, schema map[string]interface{}) error {
	expectedType, ok := schema["type"].(string)
	if !ok {
		return nil // No type constraint
	}

	actualType := reflect.TypeOf(value).Kind().String()

	// Simple type mapping
	typeMap := map[string]string{
		"string":  "string",
		"int":     "number",
		"float64": "number",
		"bool":    "boolean",
		"map":     "object",
		"slice":   "array",
	}

	if mappedType, exists := typeMap[actualType]; exists {
		if mappedType != expectedType && expectedType != "number" {
			return fmt.Errorf("expected type %s, got %s", expectedType, mappedType)
		}
	}

	return nil
}

// deepCopy creates a deep copy of a map
func (m *ConfigModule) deepCopy(original map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for key, value := range original {
		if valueMap, ok := value.(map[string]interface{}); ok {
			copy[key] = m.deepCopy(valueMap)
		} else {
			copy[key] = value
		}
	}
	return copy
}

// failResult creates a failed result
func (m *ConfigModule) failResult(result types.TaskResult, message string) (types.TaskResult, error) {
	result.Success = false
	result.Failed = true
	result.Error = message
	result.Duration = time.Since(result.Timestamp)
	return result, fmt.Errorf("%s", message)
}
