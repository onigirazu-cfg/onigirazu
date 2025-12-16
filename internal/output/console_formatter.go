package output

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

// ConsoleFormatter handles console output formatting
type ConsoleFormatter struct {
	noColor    bool
	width      int
	useSymbols bool
}

// NewConsoleFormatter creates a new console formatter
func NewConsoleFormatter(noColor bool) *ConsoleFormatter {
	width := 80
	// Try to detect terminal width from environment or use default
	if w := os.Getenv("COLUMNS"); w != "" {
		_, _ = fmt.Sscanf(w, "%d", &width)
	}

	return &ConsoleFormatter{
		noColor:    noColor,
		width:      width,
		useSymbols: !noColor,
	}
}

// FormatAggregatedResults formats aggregated results beautifully
func (cf *ConsoleFormatter) FormatAggregatedResults(
	aggregated []AggregatedResult,
	metrics ExecutionMetrics,
) string {
	var sb strings.Builder

	// Header
	sb.WriteString(cf.section("EXECUTION RESULTS"))

	// Results by status
	for _, group := range aggregated {
		sb.WriteString(cf.formatStatusGroup(group))
	}

	// Metrics section
	sb.WriteString(cf.section("PERFORMANCE METRICS"))
	sb.WriteString(cf.formatMetrics(metrics))

	// Footer
	sb.WriteString(cf.separator())

	return sb.String()
}

// formatStatusGroup formats a group of results by status
func (cf *ConsoleFormatter) formatStatusGroup(group AggregatedResult) string {
	var sb strings.Builder

	// Status header
	header := cf.statusHeader(group.Status, group.Count, group.Percentage)
	sb.WriteString(header)

	// Host results
	for i, host := range group.Hosts {
		result := cf.formatHostResult(host, i == len(group.Hosts)-1)
		sb.WriteString(result)
	}

	sb.WriteString("\n")

	return sb.String()
}

// statusHeader creates a status group header
func (cf *ConsoleFormatter) statusHeader(status ResultStatus, count int, percentage float64) string {
	symbol := cf.statusSymbol(status)
	label := cf.statusLabel(status)

	if cf.noColor {
		return fmt.Sprintf("%s %s (%d hosts, %.1f%%)\n", symbol, label, count, percentage)
	}

	coloredLabel := cf.colorizeStatus(label, status)
	return fmt.Sprintf("%s %s (%d hosts, %.1f%%)\n", symbol, coloredLabel, count, percentage)
}

// formatHostResult formats a single host result
func (cf *ConsoleFormatter) formatHostResult(host AggregatedHost, isLast bool) string {
	var sb strings.Builder

	// Tree branch
	branch := "  ├─ "
	if isLast {
		branch = "  └─ "
	}

	sb.WriteString(branch)

	// Host name (truncate if needed)
	hostName := host.Name
	if len(hostName) > 25 {
		hostName = hostName[:22] + "..."
	}

	if cf.noColor {
		sb.WriteString(fmt.Sprintf("%-25s", hostName))
	} else {
		sb.WriteString(utils.Colors.HostName(fmt.Sprintf("%-25s", hostName)))
	}

	// Status and duration
	sb.WriteString(fmt.Sprintf(" [%s]", formatDuration(host.Duration)))

	// Change indicator
	if host.Changed {
		sb.WriteString(" ")
		if cf.noColor {
			sb.WriteString("⚡ Changed")
		} else {
			sb.WriteString(utils.Colors.Changed("⚡ Changed"))
		}
	}

	sb.WriteString("\n")

	// Details
	if host.ErrorMessage != "" {
		sb.WriteString(cf.formatError(host))
	}

	return sb.String()
}

// formatError formats error details
func (cf *ConsoleFormatter) formatError(host AggregatedHost) string {
	var sb strings.Builder

	sb.WriteString("     │\n")
	sb.WriteString("     └─ ")

	if cf.noColor {
		sb.WriteString(fmt.Sprintf("Error: %s\n", host.ErrorMessage))
	} else {
		sb.WriteString(utils.Colors.Error(fmt.Sprintf("Error: %s\n", host.ErrorMessage)))
	}

	// Add suggestions if any
	if len(host.Suggestions) > 0 {
		sb.WriteString("        ")
		if cf.noColor {
			sb.WriteString("Suggestions:\n")
		} else {
			sb.WriteString(utils.Colors.Info("Suggestions:\n"))
		}

		for _, suggestion := range host.Suggestions {
			sb.WriteString(fmt.Sprintf("        • %s\n", suggestion))
		}
	}

	return sb.String()
}

// formatMetrics formats execution metrics
func (cf *ConsoleFormatter) formatMetrics(metrics ExecutionMetrics) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("  Total hosts:        %d\n", metrics.Total))
	sb.WriteString(fmt.Sprintf("  Successful:         %d\n", metrics.SuccessCount))
	sb.WriteString(fmt.Sprintf("  Failed:             %d\n", metrics.FailedCount))
	sb.WriteString(fmt.Sprintf("  Changed:            %d\n", metrics.ChangedCount))
	sb.WriteString(fmt.Sprintf("  Total duration:     %s\n", formatDuration(metrics.TotalDuration)))
	sb.WriteString(fmt.Sprintf("  Average per host:   %s\n", formatDuration(metrics.AverageDuration)))
	sb.WriteString(fmt.Sprintf("  Fastest:            %s\n", formatDuration(metrics.FastestDuration)))
	sb.WriteString(fmt.Sprintf("  Slowest:            %s\n", formatDuration(metrics.SlowestDuration)))

	sb.WriteString("\n")

	return sb.String()
}

// statusSymbol returns a symbol for the status
func (cf *ConsoleFormatter) statusSymbol(status ResultStatus) string {
	symbols := map[ResultStatus]string{
		StatusSuccess: "✓",
		StatusChanged: "⚡",
		StatusFailed:  "✗",
		StatusSkipped: "⊝",
	}

	symbol := symbols[status]
	if cf.noColor {
		return symbol
	}

	switch status {
	case StatusSuccess:
		return utils.Colors.Success(symbol)
	case StatusChanged:
		return utils.Colors.Changed(symbol)
	case StatusFailed:
		return utils.Colors.Error(symbol)
	case StatusSkipped:
		return utils.Colors.Skipped(symbol)
	default:
		return symbol
	}
}

// statusLabel returns a label for the status
func (cf *ConsoleFormatter) statusLabel(status ResultStatus) string {
	labels := map[ResultStatus]string{
		StatusSuccess: "SUCCESSFUL",
		StatusChanged: "CHANGED",
		StatusFailed:  "FAILED",
		StatusSkipped: "SKIPPED",
	}
	return labels[status]
}

// colorizeStatus colorizes a status label
func (cf *ConsoleFormatter) colorizeStatus(label string, status ResultStatus) string {
	if cf.noColor {
		return label
	}

	switch status {
	case StatusSuccess:
		return utils.Colors.Success(label)
	case StatusChanged:
		return utils.Colors.Changed(label)
	case StatusFailed:
		return utils.Colors.Error(label)
	case StatusSkipped:
		return utils.Colors.Skipped(label)
	default:
		return label
	}
}

// section creates a section header
func (cf *ConsoleFormatter) section(title string) string {
	if cf.noColor {
		return fmt.Sprintf("\n%s\n%s\n\n", title, strings.Repeat("=", len(title)))
	}

	return fmt.Sprintf("\n%s\n%s\n\n",
		utils.Colors.Header(title),
		utils.Colors.Dim(strings.Repeat("=", len(title))))
}

// separator creates a separator line
func (cf *ConsoleFormatter) separator() string {
	line := strings.Repeat("─", 50)
	if cf.noColor {
		return line + "\n"
	}
	return utils.Colors.Dim(line) + "\n"
}

// formatDuration formats a duration nicely
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%.0fms", d.Seconds()*1000)
	case d < time.Minute:
		return fmt.Sprintf("%.2fs", d.Seconds())
	default:
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
}
