package output

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConsoleFormatter_FormatAggregatedResults(t *testing.T) {
	formatter := NewConsoleFormatter(true) // No color for testing

	aggregated := []AggregatedResult{
		{
			Status:     StatusSuccess,
			Count:      2,
			Percentage: 66.67,
			Hosts: []AggregatedHost{
				{
					Name:     "host1",
					Status:   StatusSuccess,
					Duration: 100 * time.Millisecond,
				},
				{
					Name:     "host2",
					Status:   StatusSuccess,
					Duration: 150 * time.Millisecond,
				},
			},
		},
		{
			Status:     StatusFailed,
			Count:      1,
			Percentage: 33.33,
			Hosts: []AggregatedHost{
				{
					Name:         "host3",
					Status:       StatusFailed,
					Duration:     200 * time.Millisecond,
					ErrorMessage: "Connection refused",
				},
			},
		},
	}

	metrics := ExecutionMetrics{
		Total:           3,
		SuccessCount:    2,
		FailedCount:     1,
		TotalDuration:   450 * time.Millisecond,
		AverageDuration: 150 * time.Millisecond,
		FastestDuration: 100 * time.Millisecond,
		SlowestDuration: 200 * time.Millisecond,
	}

	output := formatter.FormatAggregatedResults(aggregated, metrics)

	// Check that output contains expected sections
	assert.Contains(t, output, "EXECUTION RESULTS")
	assert.Contains(t, output, "PERFORMANCE METRICS")
	assert.Contains(t, output, "host1")
	assert.Contains(t, output, "host2")
	assert.Contains(t, output, "host3")
	assert.Contains(t, output, "Connection refused")

	// Check metrics are present
	assert.Contains(t, output, "Total hosts:        3")
	assert.Contains(t, output, "Successful:         2")
	assert.Contains(t, output, "Failed:             1")
}

func TestConsoleFormatter_StatusSymbol(t *testing.T) {
	formatter := NewConsoleFormatter(true) // No color

	tests := []struct {
		status   ResultStatus
		expected string
	}{
		{StatusSuccess, "✓"},
		{StatusFailed, "✗"},
		{StatusChanged, "⚡"},
		{StatusSkipped, "⊝"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, formatter.statusSymbol(test.status))
	}
}

func TestConsoleFormatter_StatusLabel(t *testing.T) {
	formatter := NewConsoleFormatter(false)

	tests := []struct {
		status   ResultStatus
		expected string
	}{
		{StatusSuccess, "SUCCESSFUL"},
		{StatusFailed, "FAILED"},
		{StatusChanged, "CHANGED"},
		{StatusSkipped, "SKIPPED"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, formatter.statusLabel(test.status))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		contains string
	}{
		{50 * time.Millisecond, "ms"},
		{1500 * time.Millisecond, "s"},
		{2 * time.Minute, "m"},
	}

	for _, test := range tests {
		result := formatDuration(test.duration)
		assert.Contains(t, result, test.contains, "Duration %v should contain %s", test.duration, test.contains)
	}
}

func TestConsoleFormatter_NoColor_Output(t *testing.T) {
	formatter := NewConsoleFormatter(true)

	host := AggregatedHost{
		Name:         "testhost",
		Status:       StatusFailed,
		Duration:     100 * time.Millisecond,
		ErrorMessage: "Test error",
		Suggestions:  []string{"Check network", "Verify credentials"},
	}

	output := formatter.formatHostResult(host, true)

	// Check structure without ANSI codes
	assert.Contains(t, output, "testhost")
	assert.Contains(t, output, "100ms")
	assert.Contains(t, output, "Test error")
	assert.Contains(t, output, "Check network")
	assert.Contains(t, output, "Verify credentials")
}

func TestConsoleFormatter_Changed_Indicator(t *testing.T) {
	formatter := NewConsoleFormatter(true)

	host := AggregatedHost{
		Name:     "host1",
		Status:   StatusSuccess,
		Duration: 100 * time.Millisecond,
		Changed:  true,
	}

	output := formatter.formatHostResult(host, true)
	assert.Contains(t, output, "Changed")
	assert.Contains(t, output, "⚡")
}

func TestConsoleFormatter_Section_Separator(t *testing.T) {
	formatter := NewConsoleFormatter(true)

	section := formatter.section("TEST SECTION")
	assert.Contains(t, section, "TEST SECTION")
	assert.Contains(t, section, "============") // Should have separator line matching length of title

	separator := formatter.separator()
	assert.Contains(t, separator, "─")
}
