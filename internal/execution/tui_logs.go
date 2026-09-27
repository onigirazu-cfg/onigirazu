package execution

import (
	"regexp"
	"strings"
)

var (
	ansiCode = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	// "2026-09-27 11:29:21 [INFO] message {fields}"
	logPrefix = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}\S*\s+)?\[(INFO|WARN|WARNING|ERROR|DEBUG|FATAL)\]\s*`)
	logFields = regexp.MustCompile(`\s+\{[^{}]*\}$`)
)

// parseLogLine takes a line of the logger apart: its level and its message
// without colors, timestamp and structured fields
func parseLogLine(line string) (level, message string) {
	line = strings.TrimSpace(ansiCode.ReplaceAllString(line, ""))
	level = "INFO"
	if m := logPrefix.FindStringSubmatch(line); m != nil {
		level = m[2]
		line = line[len(m[0]):]
	}
	switch level {
	case "WARNING":
		level = "WARN"
	case "FATAL":
		level = "ERROR"
	}
	return level, logFields.ReplaceAllString(line, "")
}
