package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

type Logger struct {
	verbose bool
	logger  *log.Logger
}

func New(verbose bool) *Logger {
	return &Logger{
		verbose: verbose,
		logger:  log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.logger.Printf("[INFO] %s", message)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	if l.verbose {
		message := fmt.Sprintf(format, args...)
		l.logger.Printf("[DEBUG] %s", message)
	}
}

func (l *Logger) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.logger.Printf("[ERROR] %s", message)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.logger.Printf("[WARN] %s", message)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.logger.Printf("[FATAL] %s", message)
	os.Exit(1)
}

func (l *Logger) SetLevel(level string) {
	// For now, just set verbose based on level
	l.verbose = level == "debug"
}

// Task and play logging methods
func (l *Logger) TaskStart(taskName, hostName string) {
	l.Info("Starting task '%s' on host '%s'", taskName, hostName)
}

func (l *Logger) TaskEnd(taskName, hostName string, changed, success bool) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}
	changeStatus := ""
	if changed {
		changeStatus = " (changed)"
	}
	l.Info("Task '%s' on host '%s': %s%s", taskName, hostName, status, changeStatus)
}

func (l *Logger) PlayStart(playName string, playIndex, totalPlays int) {
	l.Info("Starting play %d/%d: '%s'", playIndex+1, totalPlays, playName)
}

func (l *Logger) PlayEnd(playName, hostName string, success bool, duration time.Duration) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}
	l.Info("Play '%s' on host '%s': %s (took %v)", playName, hostName, status, duration)
}

func (l *Logger) Progress(completed, total int, currentTask, currentHost string) {
	l.Info("Progress: %d/%d - Task: '%s' on Host: '%s'", completed, total, currentTask, currentHost)
}

func (l *Logger) Retry(taskName, hostName string, attempt, maxAttempts int, delay time.Duration, err error) {
	l.Warn("Task '%s' on host '%s' failed (attempt %d/%d), retrying in %v: %v",
		taskName, hostName, attempt, maxAttempts, delay, err)
}

// NewEnhancedLogger creates a new enhanced logger
func NewEnhancedLogger(level, format string, output io.Writer) (*Logger, error) {
	logger := New(level == "debug")
	logger.SetLevel(level)
	return logger, nil
}
