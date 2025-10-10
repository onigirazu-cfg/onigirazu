package errors

import (
	"fmt"
	"time"
)

type ErrorType int

const (
	ErrorTypeConnection ErrorType = iota
	ErrorTypeAuthentication
	ErrorTypeExecution
	ErrorTypeValidation
	ErrorTypeTimeout
	ErrorTypeHostKey
	ErrorTypeModule
	ErrorTypeParser
)

type OnigiraruError struct {
	Type      ErrorType              `json:"type"`
	Module    string                 `json:"module"`
	Host      string                 `json:"host"`
	Task      string                 `json:"task"`
	Command   string                 `json:"command,omitempty"`
	Cause     error                  `json:"-"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

func (e *OnigiraruError) Error() string {
	return fmt.Sprintf("[%s] %s on %s: %s",
		e.typeString(), e.Module, e.Host, e.Message)
}

func (e *OnigiraruError) Unwrap() error {
	return e.Cause
}

func (e *OnigiraruError) typeString() string {
	switch e.Type {
	case ErrorTypeConnection:
		return "CONNECTION"
	case ErrorTypeAuthentication:
		return "AUTH"
	case ErrorTypeExecution:
		return "EXEC"
	case ErrorTypeValidation:
		return "VALIDATION"
	case ErrorTypeTimeout:
		return "TIMEOUT"
	case ErrorTypeHostKey:
		return "HOSTKEY"
	case ErrorTypeModule:
		return "MODULE"
	case ErrorTypeParser:
		return "PARSER"
	default:
		return "UNKNOWN"
	}
}

// Конструкторы для разных типов ошибок
func NewConnectionError(host, module string, cause error) *OnigiraruError {
	return &OnigiraruError{
		Type:      ErrorTypeConnection,
		Module:    module,
		Host:      host,
		Cause:     cause,
		Message:   cause.Error(),
		Timestamp: time.Now(),
	}
}

func NewExecutionError(host, module, task, command string, cause error) *OnigiraruError {
	return &OnigiraruError{
		Type:      ErrorTypeExecution,
		Module:    module,
		Host:      host,
		Task:      task,
		Command:   command,
		Cause:     cause,
		Message:   cause.Error(),
		Timestamp: time.Now(),
	}
}

func NewHostKeyError(host string, message string) *OnigiraruError {
	return &OnigiraruError{
		Type:      ErrorTypeHostKey,
		Host:      host,
		Message:   message,
		Timestamp: time.Now(),
	}
}

func NewValidationError(module, task string, message string) *OnigiraruError {
	return &OnigiraruError{
		Type:      ErrorTypeValidation,
		Module:    module,
		Task:      task,
		Message:   message,
		Timestamp: time.Now(),
	}
}

func NewTimeoutError(host, module, task string, timeout time.Duration) *OnigiraruError {
	return &OnigiraruError{
		Type:      ErrorTypeTimeout,
		Module:    module,
		Host:      host,
		Task:      task,
		Message:   fmt.Sprintf("operation timed out after %v", timeout),
		Timestamp: time.Now(),
	}
}

func NewModuleError(module, message string) *OnigiraruError {
	return &OnigiraruError{
		Type:      ErrorTypeModule,
		Module:    module,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// WithContext adds context information to the error
func (e *OnigiraruError) WithContext(key string, value interface{}) *OnigiraruError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// IsType checks if the error is of a specific type
func IsType(err error, errorType ErrorType) bool {
	if onigiraruErr, ok := err.(*OnigiraruError); ok {
		return onigiraruErr.Type == errorType
	}
	return false
}

// GetType returns the error type if it's an OnigiraruError
func GetType(err error) (ErrorType, bool) {
	if onigiraruErr, ok := err.(*OnigiraruError); ok {
		return onigiraruErr.Type, true
	}
	return 0, false
}
