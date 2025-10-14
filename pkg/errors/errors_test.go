package errors

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestErrorTypeString(t *testing.T) {
	tests := []struct {
		name     string
		errType  ErrorType
		expected string
	}{
		{"Connection", ErrorTypeConnection, "CONNECTION"},
		{"Authentication", ErrorTypeAuthentication, "AUTH"},
		{"Execution", ErrorTypeExecution, "EXEC"},
		{"Validation", ErrorTypeValidation, "VALIDATION"},
		{"Timeout", ErrorTypeTimeout, "TIMEOUT"},
		{"HostKey", ErrorTypeHostKey, "HOSTKEY"},
		{"Module", ErrorTypeModule, "MODULE"},
		{"Parser", ErrorTypeParser, "PARSER"},
		{"Unknown", ErrorType(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &OnigiraruError{Type: tt.errType}
			result := err.typeString()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestOnigiraruErrorError(t *testing.T) {
	err := &OnigiraruError{
		Type:    ErrorTypeConnection,
		Module:  "ssh",
		Host:    "example.com",
		Message: "connection refused",
	}

	result := err.Error()
	expected := "[CONNECTION] ssh on example.com: connection refused"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestOnigiraruErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &OnigiraruError{
		Type:  ErrorTypeExecution,
		Cause: cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Expected unwrapped error to be the cause")
	}
}

func TestNewConnectionError(t *testing.T) {
	cause := errors.New("connection refused")
	err := NewConnectionError("example.com", "ssh", cause)

	if err.Type != ErrorTypeConnection {
		t.Errorf("Expected type CONNECTION, got %v", err.Type)
	}
	if err.Host != "example.com" {
		t.Errorf("Expected host 'example.com', got '%s'", err.Host)
	}
	if err.Module != "ssh" {
		t.Errorf("Expected module 'ssh', got '%s'", err.Module)
	}
	if err.Cause != cause {
		t.Errorf("Expected cause to be set")
	}
	if err.Message != "connection refused" {
		t.Errorf("Expected message 'connection refused', got '%s'", err.Message)
	}
	if err.Timestamp.IsZero() {
		t.Errorf("Expected timestamp to be set")
	}
}

func TestNewExecutionError(t *testing.T) {
	cause := errors.New("command failed")
	err := NewExecutionError("example.com", "shell", "run command", "ls -la", cause)

	if err.Type != ErrorTypeExecution {
		t.Errorf("Expected type EXECUTION, got %v", err.Type)
	}
	if err.Host != "example.com" {
		t.Errorf("Expected host 'example.com', got '%s'", err.Host)
	}
	if err.Module != "shell" {
		t.Errorf("Expected module 'shell', got '%s'", err.Module)
	}
	if err.Task != "run command" {
		t.Errorf("Expected task 'run command', got '%s'", err.Task)
	}
	if err.Command != "ls -la" {
		t.Errorf("Expected command 'ls -la', got '%s'", err.Command)
	}
	if err.Cause != cause {
		t.Errorf("Expected cause to be set")
	}
}

func TestNewHostKeyError(t *testing.T) {
	err := NewHostKeyError("example.com", "host key verification failed")

	if err.Type != ErrorTypeHostKey {
		t.Errorf("Expected type HOSTKEY, got %v", err.Type)
	}
	if err.Host != "example.com" {
		t.Errorf("Expected host 'example.com', got '%s'", err.Host)
	}
	if err.Message != "host key verification failed" {
		t.Errorf("Expected message 'host key verification failed', got '%s'", err.Message)
	}
	if err.Timestamp.IsZero() {
		t.Errorf("Expected timestamp to be set")
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("file", "create file", "path is required")

	if err.Type != ErrorTypeValidation {
		t.Errorf("Expected type VALIDATION, got %v", err.Type)
	}
	if err.Module != "file" {
		t.Errorf("Expected module 'file', got '%s'", err.Module)
	}
	if err.Task != "create file" {
		t.Errorf("Expected task 'create file', got '%s'", err.Task)
	}
	if err.Message != "path is required" {
		t.Errorf("Expected message 'path is required', got '%s'", err.Message)
	}
}

func TestNewTimeoutError(t *testing.T) {
	timeout := 30 * time.Second
	err := NewTimeoutError("example.com", "shell", "long command", timeout)

	if err.Type != ErrorTypeTimeout {
		t.Errorf("Expected type TIMEOUT, got %v", err.Type)
	}
	if err.Host != "example.com" {
		t.Errorf("Expected host 'example.com', got '%s'", err.Host)
	}
	if err.Module != "shell" {
		t.Errorf("Expected module 'shell', got '%s'", err.Module)
	}
	if err.Task != "long command" {
		t.Errorf("Expected task 'long command', got '%s'", err.Task)
	}
	if !strings.Contains(err.Message, "30s") {
		t.Errorf("Expected message to contain timeout duration, got '%s'", err.Message)
	}
}

func TestNewModuleError(t *testing.T) {
	err := NewModuleError("custom", "module not found")

	if err.Type != ErrorTypeModule {
		t.Errorf("Expected type MODULE, got %v", err.Type)
	}
	if err.Module != "custom" {
		t.Errorf("Expected module 'custom', got '%s'", err.Module)
	}
	if err.Message != "module not found" {
		t.Errorf("Expected message 'module not found', got '%s'", err.Message)
	}
}

func TestWithContext(t *testing.T) {
	err := NewModuleError("test", "test error")

	// Add single context
	err.WithContext("key1", "value1")
	if err.Context["key1"] != "value1" {
		t.Errorf("Expected context key1='value1', got '%v'", err.Context["key1"])
	}

	// Add multiple contexts
	err.WithContext("key2", 123).WithContext("key3", true)
	if err.Context["key2"] != 123 {
		t.Errorf("Expected context key2=123, got '%v'", err.Context["key2"])
	}
	if err.Context["key3"] != true {
		t.Errorf("Expected context key3=true, got '%v'", err.Context["key3"])
	}

	// Verify all keys exist
	if len(err.Context) != 3 {
		t.Errorf("Expected 3 context keys, got %d", len(err.Context))
	}
}

func TestWithContextChaining(t *testing.T) {
	err := NewModuleError("test", "test error").
		WithContext("user", "admin").
		WithContext("action", "delete").
		WithContext("resource", "file.txt")

	if len(err.Context) != 3 {
		t.Errorf("Expected 3 context keys, got %d", len(err.Context))
	}
	if err.Context["user"] != "admin" {
		t.Errorf("Expected user='admin', got '%v'", err.Context["user"])
	}
	if err.Context["action"] != "delete" {
		t.Errorf("Expected action='delete', got '%v'", err.Context["action"])
	}
	if err.Context["resource"] != "file.txt" {
		t.Errorf("Expected resource='file.txt', got '%v'", err.Context["resource"])
	}
}

func TestIsType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		errType  ErrorType
		expected bool
	}{
		{
			name:     "matching type",
			err:      NewConnectionError("host", "module", errors.New("test")),
			errType:  ErrorTypeConnection,
			expected: true,
		},
		{
			name:     "non-matching type",
			err:      NewConnectionError("host", "module", errors.New("test")),
			errType:  ErrorTypeTimeout,
			expected: false,
		},
		{
			name:     "non-OnigiraruError",
			err:      errors.New("standard error"),
			errType:  ErrorTypeConnection,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			errType:  ErrorTypeConnection,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsType(tt.err, tt.errType)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetType(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedTyp ErrorType
		expectedOk  bool
	}{
		{
			name:        "OnigiraruError",
			err:         NewConnectionError("host", "module", errors.New("test")),
			expectedTyp: ErrorTypeConnection,
			expectedOk:  true,
		},
		{
			name:        "standard error",
			err:         errors.New("standard error"),
			expectedTyp: 0,
			expectedOk:  false,
		},
		{
			name:        "nil error",
			err:         nil,
			expectedTyp: 0,
			expectedOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, ok := GetType(tt.err)
			if ok != tt.expectedOk {
				t.Errorf("Expected ok=%v, got %v", tt.expectedOk, ok)
			}
			if ok && typ != tt.expectedTyp {
				t.Errorf("Expected type=%v, got %v", tt.expectedTyp, typ)
			}
		})
	}
}

func TestErrorTimestamp(t *testing.T) {
	before := time.Now()
	err := NewModuleError("test", "test error")
	after := time.Now()

	if err.Timestamp.Before(before) || err.Timestamp.After(after) {
		t.Errorf("Expected timestamp to be between %v and %v, got %v", before, after, err.Timestamp)
	}
}

func TestAllErrorTypes(t *testing.T) {
	// Test that all error types can be created and have correct type
	tests := []struct {
		name    string
		creator func() *OnigiraruError
		errType ErrorType
	}{
		{
			name:    "Connection",
			creator: func() *OnigiraruError { return NewConnectionError("host", "mod", errors.New("err")) },
			errType: ErrorTypeConnection,
		},
		{
			name:    "Execution",
			creator: func() *OnigiraruError { return NewExecutionError("host", "mod", "task", "cmd", errors.New("err")) },
			errType: ErrorTypeExecution,
		},
		{
			name:    "HostKey",
			creator: func() *OnigiraruError { return NewHostKeyError("host", "msg") },
			errType: ErrorTypeHostKey,
		},
		{
			name:    "Validation",
			creator: func() *OnigiraruError { return NewValidationError("mod", "task", "msg") },
			errType: ErrorTypeValidation,
		},
		{
			name:    "Timeout",
			creator: func() *OnigiraruError { return NewTimeoutError("host", "mod", "task", time.Second) },
			errType: ErrorTypeTimeout,
		},
		{
			name:    "Module",
			creator: func() *OnigiraruError { return NewModuleError("mod", "msg") },
			errType: ErrorTypeModule,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creator()
			if err.Type != tt.errType {
				t.Errorf("Expected type %v, got %v", tt.errType, err.Type)
			}
			if err.Timestamp.IsZero() {
				t.Errorf("Expected timestamp to be set")
			}
		})
	}
}
