package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorAnalyzer_AnalyzeError(t *testing.T) {
	analyzer := NewErrorAnalyzer()

	tests := []struct {
		errorMsg  string
		errorType ErrorType
	}{
		{"connection refused", ErrorTypeConnection},
		{"Connection timeout", ErrorTypeTimeout},
		{"permission denied", ErrorTypePermission},
		{"SSH: Permission denied (publickey)", ErrorTypeAuth},
		{"Network unreachable", ErrorTypeNetwork},
		{"command not found", ErrorTypeCommand},
		{"exit code 127", ErrorTypeCommand},
		{"Authentication failed", ErrorTypeAuth},
		{"Some random error", ErrorTypeUnknown},
	}

	for _, test := range tests {
		errorType, suggestions := analyzer.AnalyzeError(test.errorMsg)
		assert.Equal(t, test.errorType, errorType, "Error message: %s", test.errorMsg)
		assert.NotEmpty(t, suggestions, "Should have suggestions for error type: %s", test.errorType)
	}
}

func TestErrorAnalyzer_SummarizeErrors(t *testing.T) {
	analyzer := NewErrorAnalyzer()

	errors := []string{
		"connection refused",
		"connection refused",
		"permission denied",
		"timeout",
		"timeout",
		"timeout",
	}

	summary := analyzer.SummarizeErrors(errors)

	assert.Equal(t, 6, summary.Total)
	assert.Equal(t, 2, summary.ByType[ErrorTypeConnection])
	assert.Equal(t, 1, summary.ByType[ErrorTypePermission])
	assert.Equal(t, 3, summary.ByType[ErrorTypeTimeout])

	// Most common error
	assert.Equal(t, "timeout", summary.MostCommonError)
	assert.Equal(t, 3, summary.MostCommonCount)
}

func TestErrorAnalyzer_AnalyzeHostError(t *testing.T) {
	analyzer := NewErrorAnalyzer()

	analyzed := analyzer.AnalyzeHostError("testhost", "Connection timeout")

	assert.Equal(t, "testhost", analyzed.Host)
	assert.Equal(t, "Connection timeout", analyzed.Message)
	assert.Equal(t, ErrorTypeTimeout, analyzed.Type)
	assert.NotEmpty(t, analyzed.Suggestions)
	assert.NotEmpty(t, analyzed.RetryAdvice)
	assert.Contains(t, analyzed.RetryAdvice, "timeout")
}

func TestErrorAnalyzer_GroupErrorsByType(t *testing.T) {
	analyzer := NewErrorAnalyzer()

	errors := []AnalyzedError{
		{
			Host:    "host1",
			Message: "connection refused",
			Type:    ErrorTypeConnection,
		},
		{
			Host:    "host2",
			Message: "connection refused",
			Type:    ErrorTypeConnection,
		},
		{
			Host:    "host3",
			Message: "permission denied",
			Type:    ErrorTypePermission,
		},
		{
			Host:    "host4",
			Message: "timeout",
			Type:    ErrorTypeTimeout,
		},
	}

	grouped := analyzer.GroupErrorsByType(errors)

	assert.Equal(t, 2, len(grouped[ErrorTypeConnection]))
	assert.Equal(t, 1, len(grouped[ErrorTypePermission]))
	assert.Equal(t, 1, len(grouped[ErrorTypeTimeout]))
}

func TestErrorAnalyzer_RetryAdvice(t *testing.T) {
	analyzer := NewErrorAnalyzer()

	tests := []struct {
		errMsg        string
		errorType     ErrorType
		adviceKeyword string
	}{
		{"timeout", ErrorTypeTimeout, "timeout"},
		{"connection refused", ErrorTypeConnection, "connectivity"},
		{"permission denied", ErrorTypePermission, "permissions"},
		{"authentication failed", ErrorTypeAuth, "credentials"},
	}

	for _, test := range tests {
		analyzed := analyzer.AnalyzeHostError("host", test.errMsg)
		assert.Equal(t, test.errorType, analyzed.Type)
		assert.Contains(t, analyzed.RetryAdvice, test.adviceKeyword)
	}
}

func TestErrorAnalyzer_CaseInsensitive(t *testing.T) {
	analyzer := NewErrorAnalyzer()

	// Test different case variations
	tests := []string{
		"CONNECTION REFUSED",
		"Connection Refused",
		"connection refused",
		"CONNECTION refused",
	}

	for _, errMsg := range tests {
		errorType, _ := analyzer.AnalyzeError(errMsg)
		assert.Equal(t, ErrorTypeConnection, errorType, "Should match case-insensitively: %s", errMsg)
	}
}

func TestErrorAnalyzer_EmptyErrors(t *testing.T) {
	analyzer := NewErrorAnalyzer()

	summary := analyzer.SummarizeErrors([]string{})
	assert.Equal(t, 0, summary.Total)
	assert.Equal(t, 0, summary.AverageErrorLength)
}
