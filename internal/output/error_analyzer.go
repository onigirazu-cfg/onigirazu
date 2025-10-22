package output

import (
	"sort"
	"strings"
	"time"
)

// ErrorType categorizes errors
type ErrorType string

const (
	ErrorTypeConnection ErrorType = "connection"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeCommand    ErrorType = "command"
	ErrorTypePermission ErrorType = "permission"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeAuth       ErrorType = "auth"
	ErrorTypeUnknown    ErrorType = "unknown"
)

// ErrorAnalyzer analyzes and categorizes errors
type ErrorAnalyzer struct {
	commonPatterns map[string]ErrorType
	suggestions    map[ErrorType][]string
}

// NewErrorAnalyzer creates a new error analyzer
func NewErrorAnalyzer() *ErrorAnalyzer {
	return &ErrorAnalyzer{
		commonPatterns: map[string]ErrorType{
			"connection refused":            ErrorTypeConnection,
			"connection timeout":            ErrorTypeTimeout,
			"timeout":                       ErrorTypeTimeout,
			"permission denied":             ErrorTypePermission,
			"permission denied (publickey)": ErrorTypeAuth,
			"network unreachable":           ErrorTypeNetwork,
			"command not found":             ErrorTypeCommand,
			"exit code":                     ErrorTypeCommand,
			"authentication failed":         ErrorTypeAuth,
			"host unreachable":              ErrorTypeNetwork,
			"no such host":                  ErrorTypeNetwork,
			"operation timed out":           ErrorTypeTimeout,
			"connection reset":              ErrorTypeConnection,
			"connection closed":             ErrorTypeConnection,
		},
		suggestions: map[ErrorType][]string{
			ErrorTypeConnection: {
				"Check if the host is reachable",
				"Verify SSH port is open (default: 22)",
				"Check firewall rules",
			},
			ErrorTypeTimeout: {
				"Increase timeout value with --timeout",
				"Check network connectivity",
				"Reduce parallel execution (--parallel)",
			},
			ErrorTypeCommand: {
				"Verify the command syntax",
				"Check if the program is installed",
				"Review the command output for details",
			},
			ErrorTypePermission: {
				"Check user permissions",
				"Use appropriate sudo/become settings",
				"Verify file/directory permissions",
			},
			ErrorTypeAuth: {
				"Verify SSH key is correct",
				"Check if SSH key permissions are 600",
				"Ensure SSH agent has the key loaded",
			},
			ErrorTypeNetwork: {
				"Check DNS resolution",
				"Verify network connectivity",
				"Check routing configuration",
			},
			ErrorTypeUnknown: {
				"Check verbose output for more details",
				"Review host logs and system events",
				"Verify host configuration",
			},
		},
	}
}

// AnalyzeError analyzes an error and returns categorization and suggestions
func (ea *ErrorAnalyzer) AnalyzeError(errMsg string) (ErrorType, []string) {
	lowerErr := strings.ToLower(errMsg)

	// Sort patterns by length (longest first) to match specific patterns before generic ones
	patterns := make([]string, 0, len(ea.commonPatterns))
	for pattern := range ea.commonPatterns {
		patterns = append(patterns, pattern)
	}
	sort.Slice(patterns, func(i, j int) bool {
		return len(patterns[i]) > len(patterns[j])
	})

	// Try to match against common patterns (in order of specificity)
	for _, pattern := range patterns {
		if strings.Contains(lowerErr, pattern) {
			return ea.commonPatterns[pattern], ea.suggestions[ea.commonPatterns[pattern]]
		}
	}

	// Default to unknown
	return ErrorTypeUnknown, ea.suggestions[ErrorTypeUnknown]
}

// ErrorSummary provides a summary of errors with categorization
type ErrorSummary struct {
	Total              int
	ByType             map[ErrorType]int
	FailureReasons     map[string]int
	MostCommonError    string
	MostCommonCount    int
	AverageErrorLength int
}

// SummarizeErrors creates a summary of multiple errors
func (ea *ErrorAnalyzer) SummarizeErrors(errors []string) ErrorSummary {
	summary := ErrorSummary{
		Total:          len(errors),
		ByType:         make(map[ErrorType]int),
		FailureReasons: make(map[string]int),
	}

	reasonCounts := make(map[string]int)
	totalLength := 0

	for _, err := range errors {
		errType, _ := ea.AnalyzeError(err)
		summary.ByType[errType]++

		// Extract reason (first 50 chars)
		reason := err
		if len(reason) > 50 {
			reason = reason[:50]
		}
		reasonCounts[reason]++
		totalLength += len(err)
	}

	// Find most common error
	maxCount := 0
	for reason, count := range reasonCounts {
		summary.FailureReasons[reason] = count
		if count > maxCount {
			maxCount = count
			summary.MostCommonError = reason
			summary.MostCommonCount = count
		}
	}

	if summary.Total > 0 {
		summary.AverageErrorLength = totalLength / summary.Total
	}

	return summary
}

// AnalyzedError represents an error with analysis results
type AnalyzedError struct {
	Host        string
	Message     string
	Type        ErrorType
	Suggestions []string
	RetryAdvice string
	AnalyzedAt  time.Time
}

// AnalyzeHostError analyzes an error for a specific host
func (ea *ErrorAnalyzer) AnalyzeHostError(host, errMsg string) AnalyzedError {
	errType, suggestions := ea.AnalyzeError(errMsg)

	analyzed := AnalyzedError{
		Host:        host,
		Message:     errMsg,
		Type:        errType,
		Suggestions: suggestions,
		AnalyzedAt:  time.Now(),
	}

	// Generate retry advice based on error type
	switch errType {
	case ErrorTypeTimeout:
		analyzed.RetryAdvice = "Consider retrying with increased timeout or reducing parallel load"
	case ErrorTypeConnection:
		analyzed.RetryAdvice = "Check host connectivity and retry"
	case ErrorTypeAuth:
		analyzed.RetryAdvice = "Verify credentials and SSH configuration, then retry"
	case ErrorTypePermission:
		analyzed.RetryAdvice = "Adjust permissions and retry with appropriate user/sudo"
	default:
		analyzed.RetryAdvice = "Check error details and host logs before retrying"
	}

	return analyzed
}

// GroupErrorsByType groups multiple errors by their type
func (ea *ErrorAnalyzer) GroupErrorsByType(errors []AnalyzedError) map[ErrorType][]AnalyzedError {
	groups := make(map[ErrorType][]AnalyzedError)
	for _, err := range errors {
		groups[err.Type] = append(groups[err.Type], err)
	}
	return groups
}
