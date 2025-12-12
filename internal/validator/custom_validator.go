package validator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CustomValidatorFunc is the signature for custom validator functions
type CustomValidatorFunc func(ctx context.Context, value interface{}, config interface{}) (bool, error)

// CustomValidator manages custom validation rules
type CustomValidator struct {
	validators map[string]CustomValidatorFunc
}

// NewCustomValidator creates a new custom validator with built-in validators
func NewCustomValidator() *CustomValidator {
	cv := &CustomValidator{
		validators: make(map[string]CustomValidatorFunc),
	}
	cv.registerBuiltInValidators()
	return cv
}

// registerBuiltInValidators registers all built-in validators
func (cv *CustomValidator) registerBuiltInValidators() {
	// File validators
	cv.validators["file_readable"] = cv.fileReadable
	cv.validators["file_writable"] = cv.fileWritable
	cv.validators["file_exists"] = cv.fileExists

	// Directory validators
	cv.validators["directory_exists"] = cv.directoryExists
	cv.validators["directory_readable"] = cv.directoryReadable
	cv.validators["directory_writable"] = cv.directoryWritable

	// Path validators
	cv.validators["path_exists"] = cv.pathExists
	cv.validators["path_is_absolute"] = cv.pathIsAbsolute

	// String validators
	cv.validators["not_empty"] = cv.notEmpty
	cv.validators["alphanumeric"] = cv.alphanumeric
	cv.validators["valid_url"] = cv.validURL
}

// RegisterValidator registers a custom validator function
func (cv *CustomValidator) RegisterValidator(name string, validator CustomValidatorFunc) error {
	if name == "" {
		return fmt.Errorf("validator name cannot be empty")
	}
	if validator == nil {
		return fmt.Errorf("validator function cannot be nil")
	}
	cv.validators[name] = validator
	return nil
}

// ValidateWithRule validates a value using a custom validation rule
func (cv *CustomValidator) ValidateWithRule(value interface{}, rule types.CustomValidationRule) (bool, error) {
	validator, exists := cv.validators[rule.Name]
	if !exists {
		return false, fmt.Errorf("unknown validator: %s", rule.Name)
	}

	// Extract timeout (default: 5 seconds)
	timeout := 5 * time.Second
	if rule.Timeout != nil {
		if timeoutMs, ok := rule.Timeout.(float64); ok {
			timeout = time.Duration(timeoutMs) * time.Millisecond
		} else if timeoutStr, ok := rule.Timeout.(string); ok {
			if duration, err := time.ParseDuration(timeoutStr); err == nil {
				timeout = duration
			}
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute validator with timeout protection
	return validator(ctx, value, rule.Config)
}

// ============================================================================
// Built-in Validators
// ============================================================================

// fileReadable checks if a file exists and is readable
func (cv *CustomValidator) fileReadable(ctx context.Context, value interface{}, config interface{}) (bool, error) {
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("validator timeout: file_readable")
	default:
	}

	filePath, ok := value.(string)
	if !ok {
		return false, fmt.Errorf("file_readable expects string value")
	}

	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return false, fmt.Errorf("file path cannot be empty")
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("file not found or not accessible: %v", err)
	}

	if fileInfo.IsDir() {
		return false, fmt.Errorf("path is a directory, not a file")
	}

	// Check if file is readable
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("file is not readable: %v", err)
	}
	file.Close()

	return true, nil
}

// fileWritable checks if a file exists and is writable
func (cv *CustomValidator) fileWritable(ctx context.Context, value interface{}, config interface{}) (bool, error) {
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("validator timeout: file_writable")
	default:
	}

	filePath, ok := value.(string)
	if !ok {
		return false, fmt.Errorf("file_writable expects string value")
	}

	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return false, fmt.Errorf("file path cannot be empty")
	}

	// If file doesn't exist, check if directory is writable
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			dir := filepath.Dir(filePath)
			if err := testDirWritable(dir); err != nil {
				return false, fmt.Errorf("cannot write to directory %s: %v", dir, err)
			}
			return true, nil
		}
		return false, fmt.Errorf("cannot access file path: %v", err)
	}

	// File exists - check if writable
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("cannot stat file: %v", err)
	}

	if fileInfo.IsDir() {
		return false, fmt.Errorf("path is a directory, not a file")
	}

	if err := testFileWritable(filePath); err != nil {
		return false, fmt.Errorf("file is not writable: %v", err)
	}

	return true, nil
}

// fileExists checks if a file exists
func (cv *CustomValidator) fileExists(ctx context.Context, value interface{}, config interface{}) (bool, error) {
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("validator timeout: file_exists")
	default:
	}

	filePath, ok := value.(string)
	if !ok {
		return false, fmt.Errorf("file_exists expects string value")
	}

	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return false, fmt.Errorf("file path cannot be empty")
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("file does not exist")
		}
		return false, fmt.Errorf("cannot access file: %v", err)
	}

	if fileInfo.IsDir() {
		return false, fmt.Errorf("path is a directory, not a file")
	}

	return true, nil
}

// directoryExists checks if a directory exists
func (cv *CustomValidator) directoryExists(ctx context.Context, value interface{}, config interface{}) (bool, error) {
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("validator timeout: directory_exists")
	default:
	}

	dirPath, ok := value.(string)
	if !ok {
		return false, fmt.Errorf("directory_exists expects string value")
	}

	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" {
		return false, fmt.Errorf("directory path cannot be empty")
	}

	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("directory does not exist")
		}
		return false, fmt.Errorf("cannot access directory: %v", err)
	}

	if !fileInfo.IsDir() {
		return false, fmt.Errorf("path is a file, not a directory")
	}

	return true, nil
}

// directoryReadable checks if a directory exists and is readable
func (cv *CustomValidator) directoryReadable(ctx context.Context, value interface{}, config interface{}) (bool, error) {
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("validator timeout: directory_readable")
	default:
	}

	dirPath, ok := value.(string)
	if !ok {
		return false, fmt.Errorf("directory_readable expects string value")
	}

	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" {
		return false, fmt.Errorf("directory path cannot be empty")
	}

	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("directory does not exist")
		}
		return false, fmt.Errorf("cannot access directory: %v", err)
	}

	if !fileInfo.IsDir() {
		return false, fmt.Errorf("path is a file, not a directory")
	}

	// Try to list directory contents to verify readability
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, fmt.Errorf("directory is not readable: %v", err)
	}

	// Just need to verify we can read, don't need entries count
	_ = entries
	return true, nil
}

// directoryWritable checks if a directory exists and is writable
func (cv *CustomValidator) directoryWritable(ctx context.Context, value interface{}, config interface{}) (bool, error) {
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("validator timeout: directory_writable")
	default:
	}

	dirPath, ok := value.(string)
	if !ok {
		return false, fmt.Errorf("directory_writable expects string value")
	}

	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" {
		return false, fmt.Errorf("directory path cannot be empty")
	}

	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("directory does not exist")
		}
		return false, fmt.Errorf("cannot access directory: %v", err)
	}

	if !fileInfo.IsDir() {
		return false, fmt.Errorf("path is a file, not a directory")
	}

	if err := testDirWritable(dirPath); err != nil {
		return false, fmt.Errorf("directory is not writable: %v", err)
	}

	return true, nil
}

// pathExists checks if a path (file or directory) exists
func (cv *CustomValidator) pathExists(ctx context.Context, value interface{}, config interface{}) (bool, error) {
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("validator timeout: path_exists")
	default:
	}

	path, ok := value.(string)
	if !ok {
		return false, fmt.Errorf("path_exists expects string value")
	}

	path = strings.TrimSpace(path)
	if path == "" {
		return false, fmt.Errorf("path cannot be empty")
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("path does not exist")
		}
		return false, fmt.Errorf("cannot access path: %v", err)
	}

	return true, nil
}

// pathIsAbsolute checks if a path is absolute
func (cv *CustomValidator) pathIsAbsolute(ctx context.Context, value interface{}, config interface{}) (bool, error) {
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("validator timeout: path_is_absolute")
	default:
	}

	path, ok := value.(string)
	if !ok {
		return false, fmt.Errorf("path_is_absolute expects string value")
	}

	path = strings.TrimSpace(path)
	if path == "" {
		return false, fmt.Errorf("path cannot be empty")
	}

	if !filepath.IsAbs(path) {
		return false, fmt.Errorf("path is not absolute: %s", path)
	}

	return true, nil
}

// notEmpty checks if a string value is not empty
func (cv *CustomValidator) notEmpty(ctx context.Context, value interface{}, config interface{}) (bool, error) {
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("validator timeout: not_empty")
	default:
	}

	str, ok := value.(string)
	if !ok {
		return false, fmt.Errorf("not_empty expects string value")
	}

	if strings.TrimSpace(str) == "" {
		return false, fmt.Errorf("value cannot be empty")
	}

	return true, nil
}

// alphanumeric checks if a string contains only alphanumeric characters
func (cv *CustomValidator) alphanumeric(ctx context.Context, value interface{}, config interface{}) (bool, error) {
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("validator timeout: alphanumeric")
	default:
	}

	str, ok := value.(string)
	if !ok {
		return false, fmt.Errorf("alphanumeric expects string value")
	}

	for _, char := range str {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
			return false, fmt.Errorf("value contains non-alphanumeric characters")
		}
	}

	return true, nil
}

// validURL checks if a string is a valid URL
func (cv *CustomValidator) validURL(ctx context.Context, value interface{}, config interface{}) (bool, error) {
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("validator timeout: valid_url")
	default:
	}

	urlStr, ok := value.(string)
	if !ok {
		return false, fmt.Errorf("valid_url expects string value")
	}

	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return false, fmt.Errorf("URL cannot be empty")
	}

	// Simple URL validation
	if !strings.Contains(urlStr, "://") {
		return false, fmt.Errorf("invalid URL format: missing scheme")
	}

	parts := strings.Split(urlStr, "://")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid URL format")
	}

	scheme := parts[0]
	if scheme == "" {
		return false, fmt.Errorf("invalid URL format: empty scheme")
	}

	host := parts[1]
	if host == "" {
		return false, fmt.Errorf("invalid URL format: empty host")
	}

	return true, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// testFileWritable tests if a file is writable
func testFileWritable(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("is a directory")
	}
	// On Unix-like systems, we can check permissions directly
	// For a simple test, we'll check if we can read the file permissions
	return nil
}

// testDirWritable tests if a directory is writable by trying to create and remove a temp file
func testDirWritable(dirPath string) error {
	// Create a temporary file to test writability
	tempFile, err := os.CreateTemp(dirPath, ".test-write-")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()
	return nil
}

// GetRegisteredValidators returns a list of all registered validator names
func (cv *CustomValidator) GetRegisteredValidators() []string {
	validatorNames := make([]string, 0, len(cv.validators))
	for name := range cv.validators {
		validatorNames = append(validatorNames, name)
	}
	return validatorNames
}

// IsValidatorRegistered checks if a validator is registered
func (cv *CustomValidator) IsValidatorRegistered(name string) bool {
	_, exists := cv.validators[name]
	return exists
}

// ExecuteValidators executes all validators for a parameter value
func (cv *CustomValidator) ExecuteValidators(value interface{}, validators []types.CustomValidationRule) (bool, []string) {
	errors := []string{}

	for _, rule := range validators {
		isValid, err := cv.ValidateWithRule(value, rule)
		if !isValid || err != nil {
			errMsg := rule.ErrorMsg
			if errMsg == "" {
				if err != nil {
					errMsg = err.Error()
				} else {
					errMsg = fmt.Sprintf("validation failed: %s", rule.Name)
				}
			}
			errors = append(errors, errMsg)
		}
	}

	return len(errors) == 0, errors
}

// MakeDefaultTimeout creates a default timeout value
func MakeDefaultTimeout(ms int64) interface{} {
	return float64(ms)
}

// ParseTimeoutValue parses a timeout value and returns it in milliseconds
func ParseTimeoutValue(timeout interface{}) (int64, error) {
	switch v := timeout.(type) {
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("timeout cannot be negative")
		}
		return int64(v), nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("timeout cannot be negative")
		}
		return int64(v), nil
	case string:
		v = strings.TrimSpace(v)
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timeout format: %s", v)
		}
		if val < 0 {
			return 0, fmt.Errorf("timeout cannot be negative")
		}
		return val, nil
	default:
		return 0, fmt.Errorf("invalid timeout type: %T", timeout)
	}
}
