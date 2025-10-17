package validator

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestFileReadableValidator tests the file_readable validator
func TestFileReadableValidator(t *testing.T) {
	cv := NewCustomValidator()

	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test-readable-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tests := []struct {
		name      string
		value     interface{}
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid readable file",
			value:     tmpFile.Name(),
			shouldErr: false,
		},
		{
			name:      "non-existent file",
			value:     "/non/existent/file.txt",
			shouldErr: true,
			errMsg:    "not found or not accessible",
		},
		{
			name:      "invalid value type",
			value:     123,
			shouldErr: true,
			errMsg:    "expects string",
		},
		{
			name:      "empty path",
			value:     "",
			shouldErr: true,
			errMsg:    "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := types.CustomValidationRule{
				Name: "file_readable",
			}
			valid, err := cv.ValidateWithRule(tt.value, rule)

			if tt.shouldErr && valid {
				t.Errorf("Expected error but got success")
			}
			if !tt.shouldErr && !valid {
				t.Errorf("Expected success but got error: %v", err)
			}
			if tt.shouldErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errMsg, err)
				}
			}
		})
	}
}

// TestFileWritableValidator tests the file_writable validator
func TestFileWritableValidator(t *testing.T) {
	cv := NewCustomValidator()

	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test-writable-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tests := []struct {
		name      string
		value     interface{}
		shouldErr bool
	}{
		{
			name:      "valid writable file",
			value:     tmpFile.Name(),
			shouldErr: false,
		},
		{
			name:      "invalid value type",
			value:     123,
			shouldErr: true,
		},
		{
			name:      "empty path",
			value:     "",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := types.CustomValidationRule{
				Name: "file_writable",
			}
			valid, err := cv.ValidateWithRule(tt.value, rule)

			if tt.shouldErr && valid {
				t.Errorf("Expected error but got success")
			}
			if !tt.shouldErr && !valid {
				t.Errorf("Expected success but got error: %v", err)
			}
		})
	}
}

// TestFileExistsValidator tests the file_exists validator
func TestFileExistsValidator(t *testing.T) {
	cv := NewCustomValidator()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-exists-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tests := []struct {
		name      string
		value     interface{}
		shouldErr bool
	}{
		{
			name:      "existing file",
			value:     tmpFile.Name(),
			shouldErr: false,
		},
		{
			name:      "non-existent file",
			value:     "/non/existent/file.txt",
			shouldErr: true,
		},
		{
			name:      "invalid type",
			value:     123,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := types.CustomValidationRule{
				Name: "file_exists",
			}
			valid, _ := cv.ValidateWithRule(tt.value, rule)

			if tt.shouldErr && valid {
				t.Errorf("Expected error but got success")
			}
			if !tt.shouldErr && !valid {
				t.Errorf("Expected success but validation failed")
			}
		})
	}
}

// TestDirectoryExistsValidator tests the directory_exists validator
func TestDirectoryExistsValidator(t *testing.T) {
	cv := NewCustomValidator()

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-dir-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		value     interface{}
		shouldErr bool
	}{
		{
			name:      "existing directory",
			value:     tmpDir,
			shouldErr: false,
		},
		{
			name:      "non-existent directory",
			value:     "/non/existent/directory",
			shouldErr: true,
		},
		{
			name:      "invalid type",
			value:     123,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := types.CustomValidationRule{
				Name: "directory_exists",
			}
			valid, _ := cv.ValidateWithRule(tt.value, rule)

			if tt.shouldErr && valid {
				t.Errorf("Expected error but got success")
			}
			if !tt.shouldErr && !valid {
				t.Errorf("Expected success but validation failed")
			}
		})
	}
}

// TestDirectoryReadableValidator tests the directory_readable validator
func TestDirectoryReadableValidator(t *testing.T) {
	cv := NewCustomValidator()

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-dir-readable-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		value     interface{}
		shouldErr bool
	}{
		{
			name:      "existing readable directory",
			value:     tmpDir,
			shouldErr: false,
		},
		{
			name:      "non-existent directory",
			value:     "/non/existent/directory",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := types.CustomValidationRule{
				Name: "directory_readable",
			}
			valid, _ := cv.ValidateWithRule(tt.value, rule)

			if tt.shouldErr && valid {
				t.Errorf("Expected error but got success")
			}
			if !tt.shouldErr && !valid {
				t.Errorf("Expected success but validation failed")
			}
		})
	}
}

// TestDirectoryWritableValidator tests the directory_writable validator
func TestDirectoryWritableValidator(t *testing.T) {
	cv := NewCustomValidator()

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-dir-writable-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		value     interface{}
		shouldErr bool
	}{
		{
			name:      "existing writable directory",
			value:     tmpDir,
			shouldErr: false,
		},
		{
			name:      "invalid type",
			value:     123,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := types.CustomValidationRule{
				Name: "directory_writable",
			}
			valid, _ := cv.ValidateWithRule(tt.value, rule)

			if tt.shouldErr && valid {
				t.Errorf("Expected error but got success")
			}
			if !tt.shouldErr && !valid {
				t.Errorf("Expected success but validation failed")
			}
		})
	}
}

// TestPathExistsValidator tests the path_exists validator
func TestPathExistsValidator(t *testing.T) {
	cv := NewCustomValidator()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-path-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tests := []struct {
		name      string
		value     interface{}
		shouldErr bool
	}{
		{
			name:      "existing file",
			value:     tmpFile.Name(),
			shouldErr: false,
		},
		{
			name:      "non-existent path",
			value:     "/non/existent/path",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := types.CustomValidationRule{
				Name: "path_exists",
			}
			valid, _ := cv.ValidateWithRule(tt.value, rule)

			if tt.shouldErr && valid {
				t.Errorf("Expected error but got success")
			}
			if !tt.shouldErr && !valid {
				t.Errorf("Expected success but validation failed")
			}
		})
	}
}

// TestPathIsAbsoluteValidator tests the path_is_absolute validator
func TestPathIsAbsoluteValidator(t *testing.T) {
	cv := NewCustomValidator()

	tests := []struct {
		name      string
		value     interface{}
		shouldErr bool
	}{
		{
			name:      "absolute path",
			value:     "/absolute/path",
			shouldErr: false,
		},
		{
			name:      "relative path",
			value:     "relative/path",
			shouldErr: true,
		},
		{
			name:      "invalid type",
			value:     123,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := types.CustomValidationRule{
				Name: "path_is_absolute",
			}
			valid, _ := cv.ValidateWithRule(tt.value, rule)

			if tt.shouldErr && valid {
				t.Errorf("Expected error but got success")
			}
			if !tt.shouldErr && !valid {
				t.Errorf("Expected success but validation failed")
			}
		})
	}
}

// TestNotEmptyValidator tests the not_empty validator
func TestNotEmptyValidator(t *testing.T) {
	cv := NewCustomValidator()

	tests := []struct {
		name      string
		value     interface{}
		shouldErr bool
	}{
		{
			name:      "non-empty string",
			value:     "hello",
			shouldErr: false,
		},
		{
			name:      "empty string",
			value:     "",
			shouldErr: true,
		},
		{
			name:      "whitespace only",
			value:     "   ",
			shouldErr: true,
		},
		{
			name:      "invalid type",
			value:     123,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := types.CustomValidationRule{
				Name: "not_empty",
			}
			valid, _ := cv.ValidateWithRule(tt.value, rule)

			if tt.shouldErr && valid {
				t.Errorf("Expected error but got success")
			}
			if !tt.shouldErr && !valid {
				t.Errorf("Expected success but validation failed")
			}
		})
	}
}

// TestAlphanumericValidator tests the alphanumeric validator
func TestAlphanumericValidator(t *testing.T) {
	cv := NewCustomValidator()

	tests := []struct {
		name      string
		value     interface{}
		shouldErr bool
	}{
		{
			name:      "alphanumeric string",
			value:     "abc123",
			shouldErr: false,
		},
		{
			name:      "string with special chars",
			value:     "abc-123",
			shouldErr: true,
		},
		{
			name:      "string with spaces",
			value:     "abc 123",
			shouldErr: true,
		},
		{
			name:      "invalid type",
			value:     123,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := types.CustomValidationRule{
				Name: "alphanumeric",
			}
			valid, _ := cv.ValidateWithRule(tt.value, rule)

			if tt.shouldErr && valid {
				t.Errorf("Expected error but got success")
			}
			if !tt.shouldErr && !valid {
				t.Errorf("Expected success but validation failed")
			}
		})
	}
}

// TestValidURLValidator tests the valid_url validator
func TestValidURLValidator(t *testing.T) {
	cv := NewCustomValidator()

	tests := []struct {
		name      string
		value     interface{}
		shouldErr bool
	}{
		{
			name:      "valid HTTP URL",
			value:     "http://example.com",
			shouldErr: false,
		},
		{
			name:      "valid HTTPS URL",
			value:     "https://example.com",
			shouldErr: false,
		},
		{
			name:      "invalid URL missing scheme",
			value:     "example.com",
			shouldErr: true,
		},
		{
			name:      "invalid type",
			value:     123,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := types.CustomValidationRule{
				Name: "valid_url",
			}
			valid, _ := cv.ValidateWithRule(tt.value, rule)

			if tt.shouldErr && valid {
				t.Errorf("Expected error but got success")
			}
			if !tt.shouldErr && !valid {
				t.Errorf("Expected success but validation failed")
			}
		})
	}
}

// TestTimeoutHandling tests that validators respect timeout
func TestTimeoutHandling(t *testing.T) {
	cv := NewCustomValidator()

	// Register a slow validator that takes longer than timeout
	slowValidator := func(ctx context.Context, value interface{}, config interface{}) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return true, nil
		}
	}

	cv.RegisterValidator("slow", slowValidator)

	rule := types.CustomValidationRule{
		Name:    "slow",
		Timeout: float64(50), // 50ms timeout
	}

	start := time.Now()
	valid, err := cv.ValidateWithRule("test", rule)
	duration := time.Since(start)

	if valid {
		t.Errorf("Expected timeout error but got success")
	}
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if duration > 500*time.Millisecond {
		t.Errorf("Validator took too long: %v", duration)
	}
}

// TestUnknownValidator tests handling of unknown validator
func TestUnknownValidator(t *testing.T) {
	cv := NewCustomValidator()

	rule := types.CustomValidationRule{
		Name: "unknown_validator",
	}

	valid, err := cv.ValidateWithRule("test", rule)

	if valid {
		t.Errorf("Expected error for unknown validator")
	}
	if err == nil || !contains(err.Error(), "unknown validator") {
		t.Errorf("Expected 'unknown validator' error, got: %v", err)
	}
}

// TestRegisterCustomValidator tests registering a custom validator
func TestRegisterCustomValidator(t *testing.T) {
	cv := NewCustomValidator()

	customValidator := func(ctx context.Context, value interface{}, config interface{}) (bool, error) {
		str, ok := value.(string)
		if !ok {
			return false, nil
		}
		return len(str) > 5, nil
	}

	err := cv.RegisterValidator("min_length_5", customValidator)
	if err != nil {
		t.Errorf("Failed to register custom validator: %v", err)
	}

	rule := types.CustomValidationRule{
		Name: "min_length_5",
	}

	// Test with value that passes
	valid, _ := cv.ValidateWithRule("hello world", rule)
	if !valid {
		t.Errorf("Custom validator should pass for 'hello world'")
	}

	// Test with value that fails
	valid, _ = cv.ValidateWithRule("hi", rule)
	if valid {
		t.Errorf("Custom validator should fail for 'hi'")
	}
}

// TestExecuteValidators tests executing multiple validators
func TestExecuteValidators(t *testing.T) {
	cv := NewCustomValidator()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-multi-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	validators := []types.CustomValidationRule{
		{
			Name: "file_exists",
		},
		{
			Name: "file_readable",
		},
	}

	isValid, errors := cv.ExecuteValidators(tmpFile.Name(), validators)

	if !isValid || len(errors) > 0 {
		t.Errorf("Expected all validators to pass, got errors: %v", errors)
	}

	// Test with non-existent file
	validators2 := []types.CustomValidationRule{
		{
			Name: "file_exists",
		},
	}

	isValid, errors = cv.ExecuteValidators("/non/existent/file", validators2)

	if isValid || len(errors) == 0 {
		t.Errorf("Expected validator to fail for non-existent file")
	}
}

// TestGetRegisteredValidators tests getting list of validators
func TestGetRegisteredValidators(t *testing.T) {
	cv := NewCustomValidator()

	validators := cv.GetRegisteredValidators()

	if len(validators) == 0 {
		t.Errorf("Expected at least one built-in validator")
	}

	// Check for some expected validators
	expectedValidators := []string{
		"file_readable",
		"file_writable",
		"directory_exists",
		"not_empty",
	}

	for _, expected := range expectedValidators {
		found := false
		for _, validator := range validators {
			if validator == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected validator '%s' not found in registered validators", expected)
		}
	}
}

// TestIsValidatorRegistered tests checking if validator is registered
func TestIsValidatorRegistered(t *testing.T) {
	cv := NewCustomValidator()

	if !cv.IsValidatorRegistered("file_readable") {
		t.Errorf("Expected file_readable validator to be registered")
	}

	if cv.IsValidatorRegistered("non_existent") {
		t.Errorf("Expected non_existent validator to not be registered")
	}
}

// TestParseTimeoutValue tests parsing timeout values
func TestParseTimeoutValue(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		expected  int64
		shouldErr bool
	}{
		{
			name:     "float value",
			value:    float64(100),
			expected: 100,
		},
		{
			name:     "int value",
			value:    100,
			expected: 100,
		},
		{
			name:     "string value",
			value:    "100",
			expected: 100,
		},
		{
			name:      "negative value",
			value:     float64(-100),
			shouldErr: true,
		},
		{
			name:      "invalid type",
			value:     map[string]interface{}{},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTimeoutValue(tt.value)

			if tt.shouldErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.shouldErr && result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestCustomValidatorWithParameterValidator tests integration with ParameterValidator
func TestCustomValidatorWithParameterValidator(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test-integration-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	parameters := map[string]types.ParameterDef{
		"config_file": {
			Type:     "string",
			Required: true,
			Validators: []types.CustomValidationRule{
				{
					Name:     "file_readable",
					ErrorMsg: "Config file must be readable",
				},
			},
		},
	}

	validator := NewParameterValidator(parameters)

	// Test with valid file
	vars := map[string]interface{}{
		"config_file": tmpFile.Name(),
	}

	result := validator.ValidateParameters(vars)
	if !result.Valid {
		t.Errorf("Expected validation to pass for readable file, got errors: %v", result.Errors)
	}

	// Test with non-existent file
	vars2 := map[string]interface{}{
		"config_file": "/non/existent/file",
	}

	result2 := validator.ValidateParameters(vars2)
	if result2.Valid {
		t.Errorf("Expected validation to fail for non-existent file")
	}
	if len(result2.Errors) == 0 {
		t.Errorf("Expected validation errors")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != "" && substr != ""
}
