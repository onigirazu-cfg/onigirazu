package logger

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
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

func TestLoggerSetLevel(t *testing.T) {
	logger := New(false)

	// Test setting debug level
	logger.SetLevel("debug")
	if !logger.verbose {
		t.Error("Expected verbose to be true after setting level to debug")
	}

	// Test setting other level
	logger.SetLevel("info")
	if logger.verbose {
		t.Error("Expected verbose to be false after setting level to info")
	}
}

func TestLoggerProgress(t *testing.T) {
	var buf bytes.Buffer
	logger := New(false)
	logger.logger.SetOutput(&buf)

	logger.Progress(5, 10, "test-task", "test-host")

	output := buf.String()
	if !strings.Contains(output, "5/10") ||
		!strings.Contains(output, "test-task") ||
		!strings.Contains(output, "test-host") {
		t.Error("Progress output should contain all expected information")
	}
}

func TestNewEnhancedLogger(t *testing.T) {
	var buf bytes.Buffer

	// Test with debug level
	logger, err := NewEnhancedLogger("debug", "text", &buf)
	if err != nil {
		t.Fatalf("NewEnhancedLogger() returned error: %v", err)
	}
	if logger == nil {
		t.Fatal("NewEnhancedLogger() returned nil")
	}
	if !logger.verbose {
		t.Error("Expected verbose to be true for debug level")
	}

	// Test with info level
	logger, err = NewEnhancedLogger("info", "json", &buf)
	if err != nil {
		t.Fatalf("NewEnhancedLogger() returned error: %v", err)
	}
	if logger == nil {
		t.Fatal("NewEnhancedLogger() returned nil")
	}
	if logger.verbose {
		t.Error("Expected verbose to be false for info level")
	}
}

// Helper type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestEnhancedLoggerTextFormat tests text format logging
func TestEnhancedLoggerTextFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("debug", FormatText, &buf)
	defer logger.Close()

	// Test different log levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warning message")
	logger.Error("error message")

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Error("Expected debug message in output")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Expected info message in output")
	}
	if !strings.Contains(output, "warning message") {
		t.Error("Expected warning message in output")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Expected error message in output")
	}
}

// TestEnhancedLoggerJSONFormat tests JSON format logging
func TestEnhancedLoggerJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatJSON, &buf)
	defer logger.Close()

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, `"message":"test message"`) {
		t.Error("Expected JSON formatted message in output")
	}
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Error("Expected level field in JSON output")
	}
}

// TestEnhancedLoggerWithField tests field addition
func TestEnhancedLoggerWithField(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	loggerWithField := logger.WithField("key", "value")
	loggerWithField.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "key=value") {
		t.Error("Expected field in output")
	}
}

// TestEnhancedLoggerWithFields tests multiple fields addition
func TestEnhancedLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatJSON, &buf)
	defer logger.Close()

	fields := map[string]interface{}{
		"host": "localhost",
		"port": 22,
	}
	loggerWithFields := logger.WithFields(fields)
	loggerWithFields.Info("test message")

	output := buf.String()
	if !strings.Contains(output, `"host":"localhost"`) {
		t.Error("Expected host field in JSON output")
	}
	if !strings.Contains(output, `"port":22`) {
		t.Error("Expected port field in JSON output")
	}
}

// TestEnhancedLoggerSetLevel tests level changing
func TestEnhancedLoggerSetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	// Debug should not be logged at info level
	logger.Debug("debug message 1")
	if strings.Contains(buf.String(), "debug message 1") {
		t.Error("Debug message should not be logged at info level")
	}

	// Change to debug level
	logger.SetLevel("debug")
	logger.Debug("debug message 2")
	if !strings.Contains(buf.String(), "debug message 2") {
		t.Error("Debug message should be logged at debug level")
	}
}

// TestEnhancedLoggerTaskStart tests task start logging
func TestEnhancedLoggerTaskStart(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	logger.TaskStart("host1", "install package")

	output := buf.String()
	if !strings.Contains(output, "host1") {
		t.Error("Expected host in output")
	}
	if !strings.Contains(output, "install package") {
		t.Error("Expected task name in output")
	}
}

// TestEnhancedLoggerTaskSuccess tests task success logging
func TestEnhancedLoggerTaskSuccess(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	duration := 100 * time.Millisecond
	logger.TaskSuccess("host1", "install package", true, duration)

	output := buf.String()
	if !strings.Contains(output, "host1") {
		t.Error("Expected host in output")
	}
	if !strings.Contains(output, "install package") {
		t.Error("Expected task name in output")
	}
	if !strings.Contains(output, "CHANGED") {
		t.Error("Expected CHANGED status in output")
	}
}

// TestEnhancedLoggerTaskError tests task error logging
func TestEnhancedLoggerTaskError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	duration := 50 * time.Millisecond
	err := fmt.Errorf("connection failed")
	logger.TaskError("host1", "install package", err, duration)

	output := buf.String()
	if !strings.Contains(output, "host1") {
		t.Error("Expected host in output")
	}
	if !strings.Contains(output, "install package") {
		t.Error("Expected task name in output")
	}
	if !strings.Contains(output, "connection failed") {
		t.Error("Expected error message in output")
	}
}

// TestEnhancedLoggerTaskEnd tests task end logging
func TestEnhancedLoggerTaskEnd(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	// Test successful task
	logger.TaskEnd("install package", "host1", true, true)
	output := buf.String()
	if !strings.Contains(output, "SUCCESS") {
		t.Error("Expected SUCCESS status in output")
	}
	if !strings.Contains(output, "changed") {
		t.Error("Expected changed indicator in output")
	}

	// Test failed task
	buf.Reset()
	logger.TaskEnd("install package", "host1", false, false)
	output = buf.String()
	if !strings.Contains(output, "FAILED") {
		t.Error("Expected FAILED status in output")
	}
}

// TestEnhancedLoggerBuffering tests log buffering
func TestEnhancedLoggerBuffering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhancedWithBuffer("info", FormatText, &buf, 10)
	defer logger.Close()

	// Log should be buffered initially
	logger.Info("test message")

	// Flush buffer
	logger.Flush()

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("Expected message after flush")
	}
}

// TestEnhancedLoggerGetStats tests statistics retrieval
func TestEnhancedLoggerGetStats(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	// Log some messages
	logger.Info("message 1")
	logger.Warn("message 2")
	logger.Error("message 3")

	stats := logger.GetStats()
	if stats.TotalLogs != 3 {
		t.Errorf("Expected 3 total logs, got %d", stats.TotalLogs)
	}
	if stats.Uptime == 0 {
		t.Error("Expected non-zero uptime")
	}
}

// TestEnhancedLoggerSetBufferSize tests buffer size changing
func TestEnhancedLoggerSetBufferSize(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhancedWithBuffer("info", FormatText, &buf, 100)
	defer logger.Close()

	logger.SetBufferSize(50)

	stats := logger.GetStats()
	if stats.MaxBuffer != 50 {
		t.Errorf("Expected max buffer 50, got %d", stats.MaxBuffer)
	}
}

// TestEnhancedLoggerEnableBuffering tests buffering enable/disable
func TestEnhancedLoggerEnableBuffering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhancedWithBuffer("info", FormatText, &buf, 100)
	defer logger.Close()

	// Disable buffering
	logger.EnableBuffering(false)
	stats := logger.GetStats()
	if stats.MaxBuffer != 0 {
		t.Errorf("Expected max buffer 0 when disabled, got %d", stats.MaxBuffer)
	}

	// Re-enable buffering
	logger.EnableBuffering(true)
	stats = logger.GetStats()
	if stats.MaxBuffer == 0 {
		t.Error("Expected non-zero max buffer when enabled")
	}
}

// TestEnhancedLoggerClose tests logger closing
func TestEnhancedLoggerClose(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)

	logger.Info("test message")

	err := logger.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("Expected message after close")
	}
}

// TestParseLogLevel tests log level parsing
func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"DEBUG", LevelDebug},
		{"debug", LevelDebug},
		{"INFO", LevelInfo},
		{"info", LevelInfo},
		{"WARN", LevelWarn},
		{"WARNING", LevelWarn},
		{"ERROR", LevelError},
		{"FATAL", LevelFatal},
		{"unknown", LevelInfo}, // default
	}

	for _, tt := range tests {
		result := parseLogLevel(tt.input)
		if result != tt.expected {
			t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// TestEnhancedLoggerConcurrency tests concurrent logging
func TestEnhancedLoggerConcurrency(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Info("goroutine %d message %d", id, j)
			}
		}(i)
	}

	wg.Wait()

	stats := logger.GetStats()
	expectedLogs := int64(numGoroutines * messagesPerGoroutine)
	if stats.TotalLogs != expectedLogs {
		t.Errorf("Expected %d total logs, got %d", expectedLogs, stats.TotalLogs)
	}
}
