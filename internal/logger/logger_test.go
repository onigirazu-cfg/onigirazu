package logger

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	logger := New(false)
	if logger == nil {
		t.Fatal("New() returned nil")
	}

	if logger.verbose {
		t.Error("Expected verbose to be false")
	}
}

func TestLoggerVerbose(t *testing.T) {
	logger := New(true)
	if !logger.verbose {
		t.Error("Expected verbose to be true")
	}
}

func TestLoggerInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := New(false)
	logger.logger.SetOutput(&buf)

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Error("Expected output to contain [INFO]")
	}
	if !strings.Contains(output, "test message") {
		t.Error("Expected output to contain 'test message'")
	}
}

func TestLoggerDebug(t *testing.T) {
	var buf bytes.Buffer

	// Test with verbose=false
	logger := New(false)
	logger.logger.SetOutput(&buf)
	logger.Debug("debug message")

	if buf.String() != "" {
		t.Error("Expected no output when verbose=false")
	}

	// Test with verbose=true
	buf.Reset()
	logger = New(true)
	logger.logger.SetOutput(&buf)
	logger.Debug("debug message")

	output := buf.String()
	if !strings.Contains(output, "[DEBUG]") {
		t.Error("Expected output to contain [DEBUG]")
	}
	if !strings.Contains(output, "debug message") {
		t.Error("Expected output to contain 'debug message'")
	}
}

func TestLoggerError(t *testing.T) {
	var buf bytes.Buffer
	logger := New(false)
	logger.logger.SetOutput(&buf)

	logger.Error("error message")

	output := buf.String()
	if !strings.Contains(output, "[ERROR]") {
		t.Error("Expected output to contain [ERROR]")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Expected output to contain 'error message'")
	}
}

func TestLoggerWarn(t *testing.T) {
	var buf bytes.Buffer
	logger := New(false)
	logger.logger.SetOutput(&buf)

	logger.Warn("warning message")

	output := buf.String()
	if !strings.Contains(output, "[WARN]") {
		t.Error("Expected output to contain [WARN]")
	}
	if !strings.Contains(output, "warning message") {
		t.Error("Expected output to contain 'warning message'")
	}
}

func TestTaskLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := New(false)
	logger.logger.SetOutput(&buf)

	// Test TaskStart
	logger.TaskStart("test-task", "test-host")
	output := buf.String()
	if !strings.Contains(output, "test-task") || !strings.Contains(output, "test-host") {
		t.Error("TaskStart output should contain task and host names")
	}

	// Test TaskEnd
	buf.Reset()
	logger.TaskEnd("test-task", "test-host", true, true)
	output = buf.String()
	if !strings.Contains(output, "SUCCESS") || !strings.Contains(output, "(changed)") {
		t.Error("TaskEnd output should contain SUCCESS and (changed)")
	}

	// Test TaskEnd failure
	buf.Reset()
	logger.TaskEnd("test-task", "test-host", false, false)
	output = buf.String()
	if !strings.Contains(output, "FAILED") {
		t.Error("TaskEnd output should contain FAILED")
	}
}

func TestPlayLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := New(false)
	logger.logger.SetOutput(&buf)

	// Test PlayStart
	logger.PlayStart("test-play", 1, 3)
	output := buf.String()
	if !strings.Contains(output, "2/3") || !strings.Contains(output, "test-play") {
		t.Error("PlayStart output should contain play index and name")
	}

	// Test PlayEnd success
	buf.Reset()
	logger.PlayEnd("test-play", "test-host", true, 5*time.Second)
	output = buf.String()
	if !strings.Contains(output, "SUCCESS") || !strings.Contains(output, "5s") {
		t.Error("PlayEnd output should contain SUCCESS and duration")
	}

	// Test PlayEnd failure
	buf.Reset()
	logger.PlayEnd("test-play", "test-host", false, 3*time.Second)
	output = buf.String()
	if !strings.Contains(output, "FAILED") {
		t.Error("PlayEnd output should contain FAILED")
	}
}

func TestRetryLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := New(false)
	logger.logger.SetOutput(&buf)

	err := &testError{"test error"}
	logger.Retry("test-task", "test-host", 2, 3, 5*time.Second, err)

	output := buf.String()
	if !strings.Contains(output, "test-task") ||
	   !strings.Contains(output, "test-host") ||
	   !strings.Contains(output, "2/3") ||
	   !strings.Contains(output, "5s") ||
	   !strings.Contains(output, "test error") {
		t.Error("Retry output should contain all expected information")
	}
}

// Helper type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
