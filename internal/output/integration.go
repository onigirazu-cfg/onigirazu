package output

import (
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// IntegrationHelper provides convenience methods for integrating console output improvements
type IntegrationHelper struct {
	formatter *ConsoleFormatter
	analyzer  *ErrorAnalyzer
}

// NewIntegrationHelper creates a new integration helper
func NewIntegrationHelper(noColor bool) *IntegrationHelper {
	return &IntegrationHelper{
		formatter: NewConsoleFormatter(noColor),
		analyzer:  NewErrorAnalyzer(),
	}
}

// ProcessPlayResult processes a PlayResult and returns formatted output
func (h *IntegrationHelper) ProcessPlayResult(pr types.PlayResult) (string, ExecutionMetrics) {
	// Aggregate the results
	aggregator := FromPlayResult(pr)

	// Get aggregated results
	aggregated := aggregator.Aggregate()

	// Analyze errors and add suggestions
	for i := range aggregated {
		for j := range aggregated[i].Hosts {
			if aggregated[i].Hosts[j].ErrorMessage != "" {
				_, suggestions := h.analyzer.AnalyzeError(aggregated[i].Hosts[j].ErrorMessage)
				aggregated[i].Hosts[j].Suggestions = suggestions
			}
		}
	}

	// Get metrics
	metrics := aggregator.GetMetrics()

	// Format output
	output := h.formatter.FormatAggregatedResults(aggregated, metrics)

	return output, metrics
}

// ProcessTaskResults processes TaskResult slice and returns formatted output
func (h *IntegrationHelper) ProcessTaskResults(tasks []types.TaskResult) (string, ExecutionMetrics) {
	// Aggregate the results
	aggregator := FromTaskResults(tasks)

	// Get aggregated results
	aggregated := aggregator.Aggregate()

	// Analyze errors and add suggestions
	for i := range aggregated {
		for j := range aggregated[i].Hosts {
			if aggregated[i].Hosts[j].ErrorMessage != "" {
				_, suggestions := h.analyzer.AnalyzeError(aggregated[i].Hosts[j].ErrorMessage)
				aggregated[i].Hosts[j].Suggestions = suggestions
			}
		}
	}

	// Get metrics
	metrics := aggregator.GetMetrics()

	// Format output
	output := h.formatter.FormatAggregatedResults(aggregated, metrics)

	return output, metrics
}

// FormatHostResults formats a slice of types.HostResult
func (h *IntegrationHelper) FormatHostResults(hosts []types.HostResult, playName string) (string, ExecutionMetrics) {
	aggregator := NewResultAggregator()

	for _, hostResult := range hosts {
		host := AggregatedHost{
			Name:    hostResult.Host,
			Details: make(map[string]interface{}),
		}

		var totalDuration time.Duration
		var changedCount, failedCount int
		var lastError string

		for _, task := range hostResult.Tasks {
			totalDuration += task.Duration
			if task.Changed {
				changedCount++
			}
			if task.Failed {
				failedCount++
				lastError = task.Error
			}
		}

		host.Duration = totalDuration

		// Determine status
		if hostResult.Failed {
			host.Status = StatusFailed
			host.ErrorMessage = lastError
		} else if changedCount > 0 {
			host.Status = StatusChanged
			host.Changed = true
		} else {
			host.Status = StatusSuccess
		}

		host.Details["play"] = playName
		host.Details["task_count"] = len(hostResult.Tasks)
		host.Details["changed_count"] = changedCount

		aggregator.Add(host)
	}

	// Get aggregated results
	aggregated := aggregator.Aggregate()

	// Analyze errors
	for i := range aggregated {
		for j := range aggregated[i].Hosts {
			if aggregated[i].Hosts[j].ErrorMessage != "" {
				_, suggestions := h.analyzer.AnalyzeError(aggregated[i].Hosts[j].ErrorMessage)
				aggregated[i].Hosts[j].Suggestions = suggestions
			}
		}
	}

	// Get metrics
	metrics := aggregator.GetMetrics()

	// Format output
	output := h.formatter.FormatAggregatedResults(aggregated, metrics)

	return output, metrics
}

// ErrorSummaryReport generates a human-readable error summary report
func (h *IntegrationHelper) ErrorSummaryReport(errors []string) string {
	summary := h.analyzer.SummarizeErrors(errors)

	var report string
	report += "\n╔════════════════════════════════════════╗\n"
	report += "║        ERROR SUMMARY REPORT            ║\n"
	report += "╚════════════════════════════════════════╝\n\n"

	report += "Total Errors: " + fmt.Sprintf("%d", summary.Total) + "\n"
	report += "Average Error Length: " + fmt.Sprintf("%d", summary.AverageErrorLength) + " characters\n\n"

	if summary.MostCommonCount > 0 {
		report += "Most Common Error:\n"
		report += "  - " + summary.MostCommonError + "\n"
		report += "  - Occurrences: " + fmt.Sprintf("%d", summary.MostCommonCount) + "\n\n"
	}

	report += "Errors by Type:\n"
	for errorType, count := range summary.ByType {
		report += "  - " + string(errorType) + ": " + fmt.Sprintf("%d", count) + "\n"
	}

	return report
}
