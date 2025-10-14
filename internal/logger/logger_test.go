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

// TestEnhancedLoggerLogWithContext tests logging with context
func TestEnhancedLoggerLogWithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatJSON, &buf)
	defer logger.Close()

	ctx := map[string]interface{}{
		"request_id": "12345",
		"user":       "testuser",
	}

	logger.LogWithContext(LevelInfo, ctx, "test message with context")
	logger.Flush() // Flush buffer to get output

	output := buf.String()
	if !strings.Contains(output, "test message with context") {
		t.Error("Expected message in output")
	}
	if !strings.Contains(output, "request_id") {
		t.Error("Expected request_id field in output")
	}
	if !strings.Contains(output, "12345") {
		t.Error("Expected request_id value in output")
	}
}

// TestEnhancedLoggerLogWithContextBuffering tests buffering with LogWithContext
func TestEnhancedLoggerLogWithContextBuffering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhancedWithBuffer("info", FormatJSON, &buf, 5)
	defer logger.Close()

	ctx := map[string]interface{}{
		"key": "value",
	}

	// Add messages to buffer
	for i := 0; i < 3; i++ {
		logger.LogWithContext(LevelInfo, ctx, "message %d", i)
	}

	// Buffer should have messages
	stats := logger.GetStats()
	if stats.BufferSize != 3 {
		t.Errorf("Expected buffer size 3, got %d", stats.BufferSize)
	}

	// Flush buffer
	logger.Flush()

	output := buf.String()
	if !strings.Contains(output, "message 0") {
		t.Error("Expected message 0 after flush")
	}
	if !strings.Contains(output, "message 2") {
		t.Error("Expected message 2 after flush")
	}
}

// TestEnhancedLoggerLogWithContextBufferOverflow tests buffer overflow
func TestEnhancedLoggerLogWithContextBufferOverflow(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhancedWithBuffer("info", FormatText, &buf, 2)
	defer logger.Close()

	ctx := map[string]interface{}{
		"key": "value",
	}

	// Fill buffer and overflow
	for i := 0; i < 5; i++ {
		logger.LogWithContext(LevelInfo, ctx, "message %d", i)
	}

	output := buf.String()
	// Should have written messages when buffer overflowed
	if !strings.Contains(output, "message") {
		t.Error("Expected messages in output after buffer overflow")
	}
}

// TestEnhancedLoggerTrace tests trace logging
func TestEnhancedLoggerTrace(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("debug", FormatJSON, &buf)
	defer logger.Close()

	logger.Trace("trace message")
	logger.Flush() // Flush buffer to get output

	output := buf.String()
	if !strings.Contains(output, "trace message") {
		t.Error("Expected trace message in output")
	}
	if !strings.Contains(output, "TRACE") {
		t.Error("Expected TRACE level indicator in output")
	}
}

// TestEnhancedLoggerTraceNotLogged tests trace not logged at info level
func TestEnhancedLoggerTraceNotLogged(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	logger.Trace("trace message")

	output := buf.String()
	if strings.Contains(output, "trace message") {
		t.Error("Trace message should not be logged at info level")
	}
}

// TestEnhancedLoggerPerformance tests performance logging
func TestEnhancedLoggerPerformance(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatJSON, &buf)
	defer logger.Close()

	duration := 150 * time.Millisecond
	details := map[string]interface{}{
		"queries": 5,
		"cache":   "hit",
	}

	logger.Performance("database_query", duration, details)
	logger.Flush() // Flush buffer to get output

	output := buf.String()
	if !strings.Contains(output, "database_query") {
		t.Error("Expected operation name in output")
	}
	if !strings.Contains(output, "performance") {
		t.Error("Expected performance type in output")
	}
	if !strings.Contains(output, "queries") {
		t.Error("Expected queries detail in output")
	}
}

// TestEnhancedLoggerAudit tests audit logging
func TestEnhancedLoggerAudit(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatJSON, &buf)
	defer logger.Close()

	details := map[string]interface{}{
		"resource": "/api/users",
		"method":   "POST",
	}

	logger.Audit("admin", "create_user", details)
	logger.Flush() // Flush buffer to get output

	output := buf.String()
	if !strings.Contains(output, "admin") {
		t.Error("Expected user in output")
	}
	if !strings.Contains(output, "create_user") {
		t.Error("Expected action in output")
	}
	if !strings.Contains(output, "audit") {
		t.Error("Expected audit type in output")
	}
	if !strings.Contains(output, "resource") {
		t.Error("Expected resource detail in output")
	}
}

// TestEnhancedLoggerSecurity tests security logging
func TestEnhancedLoggerSecurity(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	details := map[string]interface{}{
		"ip":     "192.168.1.100",
		"reason": "too many attempts",
	}

	// Test with high severity (should log as ERROR)
	logger.Security("failed_login", "high", details)
	logger.Flush() // Flush buffer to get output

	output := buf.String()
	if !strings.Contains(output, "failed_login") {
		t.Error("Expected event in output")
	}
	if !strings.Contains(output, "high") {
		t.Error("Expected severity in output")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("Expected ERROR level for high severity")
	}

	// Test with low severity (should log as WARN)
	buf.Reset()
	logger.Security("suspicious_activity", "low", details)
	logger.Flush() // Flush buffer to get output

	output = buf.String()
	if !strings.Contains(output, "WARN") {
		t.Error("Expected WARN level for low severity")
	}
}

// TestEnhancedLoggerSecurityCritical tests critical security logging
func TestEnhancedLoggerSecurityCritical(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	details := map[string]interface{}{
		"attack_type": "sql_injection",
	}

	logger.Security("attack_detected", "critical", details)
	logger.Flush() // Flush buffer to get output

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Error("Expected ERROR level for critical severity")
	}
}

// TestEnhancedLoggerTaskSkipped tests task skipped logging
func TestEnhancedLoggerTaskSkipped(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatJSON, &buf)
	defer logger.Close()

	logger.TaskSkipped("host1", "install_package", "already installed")

	output := buf.String()
	if !strings.Contains(output, "host1") {
		t.Error("Expected host in output")
	}
	if !strings.Contains(output, "install_package") {
		t.Error("Expected task name in output")
	}
	if !strings.Contains(output, "already installed") {
		t.Error("Expected reason in output")
	}
	if !strings.Contains(output, "task_skipped") {
		t.Error("Expected task_skipped type in output")
	}
}

// TestEnhancedLoggerPlayStart tests play start logging
func TestEnhancedLoggerPlayStart(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	logger.PlayStart("setup_servers", 0, 3)

	output := buf.String()
	if !strings.Contains(output, "setup_servers") {
		t.Error("Expected play name in output")
	}
	// PlayStart uses playIndex directly, so it should be "0/3" not "1/3"
	if !strings.Contains(output, "0/3") {
		t.Error("Expected play index in output")
	}
}

// TestEnhancedLoggerPlayEnd tests play end logging
func TestEnhancedLoggerPlayEnd(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatJSON, &buf)
	defer logger.Close()

	duration := 5 * time.Second

	// Test successful play
	logger.PlayEnd("setup_servers", "host1", true, duration)

	output := buf.String()
	if !strings.Contains(output, "setup_servers") {
		t.Error("Expected play name in output")
	}
	if !strings.Contains(output, "host1") {
		t.Error("Expected host in output")
	}
	if !strings.Contains(output, "SUCCESSFULLY") {
		t.Error("Expected SUCCESSFULLY status in output")
	}

	// Test failed play
	buf.Reset()
	logger.PlayEnd("setup_servers", "host2", false, duration)

	output = buf.String()
	if !strings.Contains(output, "WITH ERRORS") {
		t.Error("Expected WITH ERRORS status in output")
	}
}

// TestEnhancedLoggerProgress tests progress logging
func TestEnhancedLoggerProgress(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatJSON, &buf)
	defer logger.Close()

	logger.Progress(7, 10, "install_nginx", "web-server-1")

	output := buf.String()
	if !strings.Contains(output, "7") {
		t.Error("Expected completed count in output")
	}
	if !strings.Contains(output, "10") {
		t.Error("Expected total count in output")
	}
	if !strings.Contains(output, "70.0") {
		t.Error("Expected percentage in output")
	}
	if !strings.Contains(output, "install_nginx") {
		t.Error("Expected task name in output")
	}
	if !strings.Contains(output, "web-server-1") {
		t.Error("Expected host name in output")
	}
}

// TestEnhancedLoggerRetry tests retry logging
func TestEnhancedLoggerRetry(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhanced("info", FormatText, &buf)
	defer logger.Close()

	err := fmt.Errorf("connection timeout")
	delay := 3 * time.Second

	logger.Retry("connect_database", "db-server-1", 2, 5, delay, err)

	output := buf.String()
	if !strings.Contains(output, "connect_database") {
		t.Error("Expected task name in output")
	}
	if !strings.Contains(output, "db-server-1") {
		t.Error("Expected host name in output")
	}
	if !strings.Contains(output, "2/5") {
		t.Error("Expected attempt count in output")
	}
	if !strings.Contains(output, "3s") {
		t.Error("Expected delay in output")
	}
	if !strings.Contains(output, "connection timeout") {
		t.Error("Expected error message in output")
	}
	if !strings.Contains(output, "WARN") {
		t.Error("Expected WARN level for retry")
	}
}

// TestEnhancedLoggerFlushBufferLocked tests buffer flushing
func TestEnhancedLoggerFlushBufferLocked(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhancedWithBuffer("info", FormatText, &buf, 10)
	defer logger.Close()

	ctx := map[string]interface{}{
		"test": "value",
	}

	// Add messages to buffer
	for i := 0; i < 5; i++ {
		logger.LogWithContext(LevelInfo, ctx, "buffered message %d", i)
	}

	// Verify buffer has messages
	stats := logger.GetStats()
	if stats.BufferSize == 0 {
		t.Error("Expected non-empty buffer")
	}

	// Flush buffer
	logger.Flush()

	// Verify buffer is empty
	stats = logger.GetStats()
	if stats.BufferSize != 0 {
		t.Errorf("Expected empty buffer after flush, got %d", stats.BufferSize)
	}

	// Verify messages were written
	output := buf.String()
	if !strings.Contains(output, "buffered message 0") {
		t.Error("Expected first message in output")
	}
	if !strings.Contains(output, "buffered message 4") {
		t.Error("Expected last message in output")
	}
}

// TestEnhancedLoggerSetBufferSizeFlush tests buffer size reduction triggers flush
func TestEnhancedLoggerSetBufferSizeFlush(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhancedWithBuffer("info", FormatText, &buf, 100)
	defer logger.Close()

	ctx := map[string]interface{}{
		"key": "value",
	}

	// Add messages to buffer
	for i := 0; i < 10; i++ {
		logger.LogWithContext(LevelInfo, ctx, "message %d", i)
	}

	// Reduce buffer size below current buffer length
	logger.SetBufferSize(5)

	// Buffer should be flushed
	output := buf.String()
	if !strings.Contains(output, "message 0") {
		t.Error("Expected messages flushed when buffer size reduced")
	}
}

// TestEnhancedLoggerWriteEntry tests writeEntry methods
func TestEnhancedLoggerWriteEntry(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhancedWithBuffer("info", FormatText, &buf, 5)
	defer logger.Close()

	ctx := map[string]interface{}{
		"field1": "value1",
		"field2": 42,
	}

	// Add messages to trigger writeEntry through buffer flush
	for i := 0; i < 6; i++ {
		logger.LogWithContext(LevelInfo, ctx, "message %d", i)
	}

	output := buf.String()
	if !strings.Contains(output, "message") {
		t.Error("Expected messages written via writeEntry")
	}
	if !strings.Contains(output, "field1=value1") {
		t.Error("Expected fields in text output")
	}
}

// TestEnhancedLoggerWriteEntryJSON tests writeEntry with JSON format
func TestEnhancedLoggerWriteEntryJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := NewEnhancedWithBuffer("info", FormatJSON, &buf, 3)
	defer logger.Close()

	ctx := map[string]interface{}{
		"user_id": 123,
		"action":  "login",
	}

	// Add messages to trigger writeEntryJSON through buffer flush
	for i := 0; i < 5; i++ {
		logger.LogWithContext(LevelInfo, ctx, "event %d", i)
	}

	output := buf.String()
	if !strings.Contains(output, `"user_id":123`) {
		t.Error("Expected user_id field in JSON output")
	}
	if !strings.Contains(output, `"action":"login"`) {
		t.Error("Expected action field in JSON output")
	}
	if !strings.Contains(output, "event") {
		t.Error("Expected event messages in output")
	}
}
