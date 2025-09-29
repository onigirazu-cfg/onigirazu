package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// LogLevel represents logging levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var levelNames = map[LogLevel]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
	LevelFatal: "FATAL",
}

var levelColors = map[LogLevel]string{
	LevelDebug: "\033[36m", // Cyan
	LevelInfo:  "\033[32m", // Green
	LevelWarn:  "\033[33m", // Yellow
	LevelError: "\033[31m", // Red
	LevelFatal: "\033[35m", // Magenta
}

const colorReset = "\033[0m"

// LogFormat represents output format
type LogFormat string

const (
	FormatText LogFormat = "text"
	FormatJSON LogFormat = "json"
)

// EnhancedLogger provides structured logging with multiple levels and formats
type EnhancedLogger struct {
	level     LogLevel
	format    LogFormat
	output    io.Writer
	logger    *log.Logger
	useColors bool
	mutex     sync.RWMutex
	fields    map[string]interface{}

	// Performance metrics
	logCount  map[LogLevel]int64
	startTime time.Time

	// Buffering for performance
	buffer     []LogEntry
	bufferSize int
	flushTimer *time.Timer
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// NewEnhanced creates a new enhanced logger
func NewEnhanced(level string, format LogFormat, output io.Writer) *EnhancedLogger {
	if output == nil {
		output = os.Stdout
	}

	logLevel := parseLogLevel(level)
	useColors := format == FormatText && isTerminal(output)

	logger := &EnhancedLogger{
		level:      logLevel,
		format:     format,
		output:     output,
		logger:     log.New(output, "", 0),
		useColors:  useColors,
		fields:     make(map[string]interface{}),
		logCount:   make(map[LogLevel]int64),
		startTime:  time.Now(),
		buffer:     make([]LogEntry, 0),
		bufferSize: 100, // Default buffer size
	}

	// Start flush timer for buffered logging
	logger.flushTimer = time.AfterFunc(time.Second, logger.flushBuffer)

	return logger
}

// NewEnhancedWithBuffer creates a new enhanced logger with custom buffer size
func NewEnhancedWithBuffer(level string, format LogFormat, output io.Writer, bufferSize int) *EnhancedLogger {
	logger := NewEnhanced(level, format, output)
	logger.bufferSize = bufferSize
	return logger
}

// parseLogLevel converts string to LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "FATAL":
		return LevelFatal
	default:
		return LevelInfo
	}
}

// isTerminal checks if output is a terminal (simplified check)
func isTerminal(w io.Writer) bool {
	if w == os.Stdout || w == os.Stderr {
		return true
	}
	return false
}

// WithField adds a field to the logger context
func (l *EnhancedLogger) WithField(key string, value interface{}) *EnhancedLogger {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	newLogger := &EnhancedLogger{
		level:     l.level,
		format:    l.format,
		output:    l.output,
		logger:    l.logger,
		useColors: l.useColors,
		fields:    make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new field
	newLogger.fields[key] = value

	return newLogger
}

// WithFields adds multiple fields to the logger context
func (l *EnhancedLogger) WithFields(fields map[string]interface{}) *EnhancedLogger {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	newLogger := &EnhancedLogger{
		level:     l.level,
		format:    l.format,
		output:    l.output,
		logger:    l.logger,
		useColors: l.useColors,
		fields:    make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// SetLevel changes the logging level
func (l *EnhancedLogger) SetLevel(level string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.level = parseLogLevel(level)
}

// Debug logs a debug message
func (l *EnhancedLogger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message
func (l *EnhancedLogger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *EnhancedLogger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message
func (l *EnhancedLogger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatal logs a fatal message and exits
func (l *EnhancedLogger) Fatal(format string, args ...interface{}) {
	l.log(LevelFatal, format, args...)
	os.Exit(1)
}

// log is the internal logging method
func (l *EnhancedLogger) log(level LogLevel, format string, args ...interface{}) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if level < l.level {
		return
	}

	message := fmt.Sprintf(format, args...)

	switch l.format {
	case FormatJSON:
		l.logJSON(level, message)
	default:
		l.logText(level, message)
	}
}

// logText logs in text format
func (l *EnhancedLogger) logText(level LogLevel, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelName := levelNames[level]

	var output string
	if l.useColors {
		color := levelColors[level]
		output = fmt.Sprintf("%s [%s%s%s] %s", timestamp, color, levelName, colorReset, message)
	} else {
		output = fmt.Sprintf("%s [%s] %s", timestamp, levelName, message)
	}

	// Add fields if any
	if len(l.fields) > 0 {
		var fieldStrs []string
		for k, v := range l.fields {
			fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", k, v))
		}
		output += fmt.Sprintf(" {%s}", strings.Join(fieldStrs, ", "))
	}

	l.logger.Println(output)
}

// logJSON logs in JSON format
func (l *EnhancedLogger) logJSON(level LogLevel, message string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     levelNames[level],
		Message:   message,
		Fields:    l.fields,
	}

	if len(l.fields) == 0 {
		entry.Fields = nil
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple text logging
		l.logger.Printf("JSON marshal error: %v, message: %s", err, message)
		return
	}

	l.logger.Println(string(jsonData))
}

// writeEntry writes a log entry in the appropriate format
func (l *EnhancedLogger) writeEntry(entry LogEntry) {
	switch l.format {
	case FormatJSON:
		l.writeEntryJSON(entry)
	default:
		l.writeEntryText(entry)
	}
}

// writeEntryText writes a log entry in text format
func (l *EnhancedLogger) writeEntryText(entry LogEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	levelName := entry.Level

	var output string
	if l.useColors {
		// Find color for level
		var color string
		for level, name := range levelNames {
			if name == levelName {
				color = levelColors[level]
				break
			}
		}
		output = fmt.Sprintf("%s [%s%s%s] %s", timestamp, color, levelName, colorReset, entry.Message)
	} else {
		output = fmt.Sprintf("%s [%s] %s", timestamp, levelName, entry.Message)
	}

	// Add fields if any
	if len(entry.Fields) > 0 {
		var fieldStrs []string
		for k, v := range entry.Fields {
			fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", k, v))
		}
		output += fmt.Sprintf(" {%s}", strings.Join(fieldStrs, ", "))
	}

	l.logger.Println(output)
}

// writeEntryJSON writes a log entry in JSON format
func (l *EnhancedLogger) writeEntryJSON(entry LogEntry) {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple text logging
		l.logger.Printf("JSON marshal error: %v, message: %s", err, entry.Message)
		return
	}

	l.logger.Println(string(jsonData))
}

// TaskStart logs task start with context
func (l *EnhancedLogger) TaskStart(host, taskName string) {
	l.WithFields(map[string]interface{}{
		"host": host,
		"task": taskName,
		"type": "task_start",
	}).Info("Starting task '%s' on host '%s'", taskName, host)
}

// TaskSuccess logs successful task completion
func (l *EnhancedLogger) TaskSuccess(host, taskName string, changed bool, duration time.Duration) {
	status := "OK"
	if changed {
		status = "CHANGED"
	}

	l.WithFields(map[string]interface{}{
		"host":     host,
		"task":     taskName,
		"type":     "task_end",
		"status":   status,
		"changed":  changed,
		"duration": duration.String(),
	}).Info("Task '%s' on host '%s': %s (%v)", taskName, host, status, duration)
}

// TaskError logs task failure
func (l *EnhancedLogger) TaskError(host, taskName string, err error, duration time.Duration) {
	l.WithFields(map[string]interface{}{
		"host":     host,
		"task":     taskName,
		"type":     "task_end",
		"status":   "FAILED",
		"error":    err.Error(),
		"duration": duration.String(),
	}).Error("Task '%s' on host '%s' failed: %v (%v)", taskName, host, err, duration)
}

// TaskEnd logs task completion
func (l *EnhancedLogger) TaskEnd(taskName, hostName string, changed, success bool) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}
	changeStatus := ""
	if changed {
		changeStatus = " (changed)"
	}

	l.WithFields(map[string]interface{}{
		"host":    hostName,
		"task":    taskName,
		"type":    "task_end",
		"success": success,
		"changed": changed,
	}).Info("Task '%s' on host '%s': %s%s", taskName, hostName, status, changeStatus)
}

// flushBuffer flushes the log buffer to output
func (l *EnhancedLogger) flushBuffer() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if len(l.buffer) == 0 {
		return
	}

	// Write all buffered entries
	for _, entry := range l.buffer {
		l.writeEntry(entry)
	}

	// Clear buffer
	l.buffer = l.buffer[:0]

	// Reset timer
	if l.flushTimer != nil {
		l.flushTimer.Reset(time.Second)
	}
}

// Flush manually flushes the buffer
func (l *EnhancedLogger) Flush() {
	l.flushBuffer()
}

// Close flushes the buffer and stops the timer
func (l *EnhancedLogger) Close() error {
	if l.flushTimer != nil {
		l.flushTimer.Stop()
	}
	l.flushBuffer()
	return nil
}

// GetStats returns logging statistics
func (l *EnhancedLogger) GetStats() LogStats {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	uptime := time.Since(l.startTime)
	totalLogs := int64(0)
	for _, count := range l.logCount {
		totalLogs += count
	}

	return LogStats{
		Uptime:      uptime,
		TotalLogs:   totalLogs,
		LogsByLevel: l.logCount,
		BufferSize:  len(l.buffer),
		MaxBuffer:   l.bufferSize,
	}
}

// LogStats holds logging statistics
type LogStats struct {
	Uptime      time.Duration      `json:"uptime"`
	TotalLogs   int64              `json:"total_logs"`
	LogsByLevel map[LogLevel]int64 `json:"logs_by_level"`
	BufferSize  int                `json:"buffer_size"`
	MaxBuffer   int                `json:"max_buffer"`
}

// SetBufferSize changes the buffer size
func (l *EnhancedLogger) SetBufferSize(size int) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.bufferSize = size

	// If current buffer is larger than new size, flush it
	if len(l.buffer) >= size {
		l.flushBuffer()
	}
}

// EnableBuffering enables or disables buffering
func (l *EnhancedLogger) EnableBuffering(enabled bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if !enabled {
		// Flush current buffer and set size to 0
		l.flushBuffer()
		l.bufferSize = 0
	} else if l.bufferSize == 0 {
		// Re-enable with default size
		l.bufferSize = 100
	}
}

// LogWithContext logs a message with additional context
func (l *EnhancedLogger) LogWithContext(level LogLevel, ctx map[string]interface{}, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	message := fmt.Sprintf(format, args...)

	// Merge context with existing fields
	allFields := make(map[string]interface{})
	for k, v := range l.fields {
		allFields[k] = v
	}
	for k, v := range ctx {
		allFields[k] = v
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     levelNames[level],
		Message:   message,
		Fields:    allFields,
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Update statistics
	l.logCount[level]++

	// Add to buffer or write immediately
	if l.bufferSize > 0 && len(l.buffer) < l.bufferSize {
		l.buffer = append(l.buffer, entry)
	} else {
		// Buffer full or buffering disabled, write immediately
		if len(l.buffer) > 0 {
			// Flush buffer first
			for _, bufferedEntry := range l.buffer {
				l.writeEntry(bufferedEntry)
			}
			l.buffer = l.buffer[:0]
		}
		l.writeEntry(entry)
	}
}

// Trace logs a trace message (below debug level)
func (l *EnhancedLogger) Trace(format string, args ...interface{}) {
	// Trace is below debug level, only log if debug is enabled
	if l.level <= LevelDebug {
		l.LogWithContext(LevelDebug, map[string]interface{}{"level": "TRACE"}, format, args...)
	}
}

// Performance logs performance-related information
func (l *EnhancedLogger) Performance(operation string, duration time.Duration, details map[string]interface{}) {
	ctx := map[string]interface{}{
		"type":      "performance",
		"operation": operation,
		"duration":  duration.String(),
	}

	for k, v := range details {
		ctx[k] = v
	}

	l.LogWithContext(LevelInfo, ctx, "Performance: %s took %v", operation, duration)
}

// Audit logs audit trail information
func (l *EnhancedLogger) Audit(user, action string, details map[string]interface{}) {
	ctx := map[string]interface{}{
		"type":   "audit",
		"user":   user,
		"action": action,
	}

	for k, v := range details {
		ctx[k] = v
	}

	l.LogWithContext(LevelInfo, ctx, "Audit: %s performed %s", user, action)
}

// Security logs security-related events
func (l *EnhancedLogger) Security(event string, severity string, details map[string]interface{}) {
	ctx := map[string]interface{}{
		"type":     "security",
		"event":    event,
		"severity": severity,
	}

	for k, v := range details {
		ctx[k] = v
	}

	level := LevelWarn
	if severity == "high" || severity == "critical" {
		level = LevelError
	}

	l.LogWithContext(level, ctx, "Security: %s (%s)", event, severity)
}

// TaskSkipped logs skipped task
func (l *EnhancedLogger) TaskSkipped(host, taskName, reason string) {
	l.WithFields(map[string]interface{}{
		"host":   host,
		"task":   taskName,
		"type":   "task_skipped",
		"reason": reason,
	}).Info("Task '%s' on host '%s' skipped: %s", taskName, host, reason)
}

// PlayStart logs play start
func (l *EnhancedLogger) PlayStart(playName string, playIndex, totalPlays int) {
	l.WithFields(map[string]interface{}{
		"play":        playName,
		"play_index":  playIndex,
		"total_plays": totalPlays,
		"type":        "play_start",
	}).Info("=== Starting play %d/%d: '%s' ===", playIndex, totalPlays, playName)
}

// PlayEnd logs play completion
func (l *EnhancedLogger) PlayEnd(playName, host string, success bool, duration time.Duration) {
	status := "SUCCESSFULLY"
	if !success {
		status = "WITH ERRORS"
	}

	l.WithFields(map[string]interface{}{
		"host":     host,
		"play":     playName,
		"type":     "play_end",
		"success":  success,
		"duration": duration.String(),
	}).Info("=== Play '%s' on host '%s' completed %s in %v ===", playName, host, status, duration)
}

// Progress logs progress information
func (l *EnhancedLogger) Progress(completed, total int, currentTask, currentHost string) {
	percentage := float64(completed) / float64(total) * 100

	l.WithFields(map[string]interface{}{
		"completed":    completed,
		"total":        total,
		"percentage":   fmt.Sprintf("%.1f%%", percentage),
		"current_task": currentTask,
		"current_host": currentHost,
		"type":         "progress",
	}).Info("Progress: %d/%d (%.1f%%) - %s on %s", completed, total, percentage, currentTask, currentHost)
}

// Retry logs retry attempt
func (l *EnhancedLogger) Retry(taskName, hostName string, attempt, maxAttempts int, delay time.Duration, err error) {
	l.WithFields(map[string]interface{}{
		"host":         hostName,
		"task":         taskName,
		"attempt":      attempt,
		"max_attempts": maxAttempts,
		"error":        err.Error(),
		"delay":        delay.String(),
		"type":         "retry",
	}).Warn("Task '%s' on host '%s' failed (attempt %d/%d), retrying in %v: %v",
		taskName, hostName, attempt, maxAttempts, delay, err)
}
