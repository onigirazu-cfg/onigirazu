package healthcheck

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

// Reporter generates health check reports in various formats
type Reporter struct {
	useColor bool
}

// NewReporter creates a new reporter
func NewReporter(useColor bool) *Reporter {
	return &Reporter{
		useColor: useColor,
	}
}

// PrintReport prints the health check report in a human-readable format
func (r *Reporter) PrintReport(report *HealthCheckReport) {
	fmt.Println()
	r.printHeader(report)
	fmt.Println()

	for _, hostReport := range report.HostReports {
		r.printHostReport(&hostReport)
	}

	fmt.Println()
	r.printSummary(report)
	fmt.Println()
}

// printHeader prints the report header
func (r *Reporter) printHeader(report *HealthCheckReport) {
	status := r.formatStatus(report.OverallStatus)
	fmt.Printf("Health Check Report - %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Overall Status: %s\n", status)
	fmt.Printf("Total Duration: %v\n", report.TotalDuration)
}

// printHostReport prints a single host report
func (r *Reporter) printHostReport(report *HostHealthReport) {
	status := r.formatStatus(report.Status)
	fmt.Printf("\n[%s] %s (%s)\n", status, report.Host.Name, report.Host.Address)
	fmt.Printf("  Duration: %v\n", report.Duration)

	// Create table for checks
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  Check Type\tStatus\tMessage")
	fmt.Fprintln(w, "  -----------\t------\t-------")

	for _, check := range report.Checks {
		checkStatus := r.formatStatus(check.Status)
		fmt.Fprintf(w, "  %s\t%s\t%s\n", check.CheckType, checkStatus, check.Message)

		// Print details if available
		if len(check.Details) > 0 {
			for key, value := range check.Details {
				fmt.Fprintf(w, "    → %s: %v\n", key, value)
			}
		}
	}

	w.Flush()
}

// printSummary prints the report summary
func (r *Reporter) printSummary(report *HealthCheckReport) {
	fmt.Println("Summary")
	fmt.Println("-------")
	fmt.Printf("Total Hosts: %d\n", report.Statistics.TotalHosts)
	fmt.Printf("  ✓ Healthy: %d\n", report.Statistics.HealthyHosts)
	fmt.Printf("  ⚠ Warning: %d\n", report.Statistics.WarningHosts)
	fmt.Printf("  ✗ Critical: %d\n", report.Statistics.CriticalHosts)
	if report.Statistics.UnknownHosts > 0 {
		fmt.Printf("  ? Unknown: %d\n", report.Statistics.UnknownHosts)
	}

	fmt.Printf("\nChecks Performed:\n")
	for checkType, count := range report.Statistics.ChecksPerHost {
		fmt.Printf("  • %s: %d\n", checkType, count)
	}

	fmt.Printf("\nAverage Duration: %.2f ms\n", report.Statistics.AverageDuration)
}

// FormatJSON returns the report as JSON string
func (r *Reporter) FormatJSON(report *HealthCheckReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FormatCSV returns the report as CSV string
func (r *Reporter) FormatCSV(report *HealthCheckReport) string {
	csv := "Host,CheckType,Status,Message,Duration(ms)\n"

	for _, hostReport := range report.HostReports {
		for _, check := range hostReport.Checks {
			line := fmt.Sprintf("%s,%s,%s,%q,%v\n",
				hostReport.Host.Name,
				check.CheckType,
				check.Status,
				check.Message,
				check.Duration.Milliseconds(),
			)
			csv += line
		}
	}

	return csv
}

// FormatHTML returns the report as HTML string
func (r *Reporter) FormatHTML(report *HealthCheckReport) string {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Health Check Report</title>
	<style>
		body { font-family: Arial, sans-serif; margin: 20px; }
		.header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
		.summary { margin: 20px 0; }
		.host-report { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
		.healthy { color: green; }
		.warning { color: orange; }
		.critical { color: red; }
		.unknown { color: gray; }
		table { width: 100%; border-collapse: collapse; }
		th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
		th { background-color: #f0f0f0; }
	</style>
</head>
<body>
	<div class="header">
		<h1>Health Check Report</h1>
		<p><strong>Generated:</strong> ` + report.Timestamp.Format("2006-01-02 15:04:05") + `</p>
		<p><strong>Overall Status:</strong> <span class="` + string(report.OverallStatus) + `">` + string(report.OverallStatus) + `</span></p>
		<p><strong>Total Duration:</strong> ` + report.TotalDuration.String() + `</p>
	</div>

	<div class="summary">
		<h2>Summary</h2>
		<ul>
			<li>Total Hosts: ` + fmt.Sprintf("%d", report.Statistics.TotalHosts) + `</li>
			<li><span class="healthy">✓ Healthy:</span> ` + fmt.Sprintf("%d", report.Statistics.HealthyHosts) + `</li>
			<li><span class="warning">⚠ Warning:</span> ` + fmt.Sprintf("%d", report.Statistics.WarningHosts) + `</li>
			<li><span class="critical">✗ Critical:</span> ` + fmt.Sprintf("%d", report.Statistics.CriticalHosts) + `</li>
		</ul>
	</div>
`

	// Add host reports
	for _, hostReport := range report.HostReports {
		statusClass := string(hostReport.Status)
		html += `
	<div class="host-report">
		<h3><span class="` + statusClass + `">` + hostReport.Host.Name + `</span> (` + hostReport.Host.Address + `)</h3>
		<p>Duration: ` + hostReport.Duration.String() + `</p>
		<table>
			<tr>
				<th>Check Type</th>
				<th>Status</th>
				<th>Message</th>
			</tr>
`

		for _, check := range hostReport.Checks {
			checkStatusClass := string(check.Status)
			html += `
			<tr>
				<td>` + string(check.CheckType) + `</td>
				<td><span class="` + checkStatusClass + `">` + string(check.Status) + `</span></td>
				<td>` + check.Message + `</td>
			</tr>
`
		}

		html += `
		</table>
	</div>
`
	}

	html += `
</body>
</html>
`

	return html
}

// FormatMarkdown returns the report as Markdown string
func (r *Reporter) FormatMarkdown(report *HealthCheckReport) string {
	md := fmt.Sprintf(`# Health Check Report

**Generated:** %s
**Overall Status:** %s
**Total Duration:** %v

## Summary

| Metric | Count |
|--------|-------|
| Total Hosts | %d |
| Healthy | %d |
| Warning | %d |
| Critical | %d |

### Checks Performed

`,
		report.Timestamp.Format("2006-01-02 15:04:05"),
		report.OverallStatus,
		report.TotalDuration,
		report.Statistics.TotalHosts,
		report.Statistics.HealthyHosts,
		report.Statistics.WarningHosts,
		report.Statistics.CriticalHosts,
	)

	for checkType, count := range report.Statistics.ChecksPerHost {
		md += fmt.Sprintf("- %s: %d\n", checkType, count)
	}

	md += "\n## Host Reports\n\n"

	for _, hostReport := range report.HostReports {
		md += fmt.Sprintf(`### %s (%s)

**Status:** %s
**Duration:** %v

| Check Type | Status | Message |
|------------|--------|---------|
`,
			hostReport.Host.Name,
			hostReport.Host.Address,
			hostReport.Status,
			hostReport.Duration,
		)

		for _, check := range hostReport.Checks {
			md += fmt.Sprintf("| %s | %s | %s |\n",
				check.CheckType,
				check.Status,
				check.Message,
			)
		}

		md += "\n"
	}

	return md
}

// formatStatus formats a status with color
func (r *Reporter) formatStatus(status HealthStatus) string {
	if !r.useColor {
		return fmt.Sprintf("[%s]", status)
	}

	statusStr := fmt.Sprintf("[%s]", status)
	switch status {
	case StatusHealthy:
		return utils.Colors.Success(statusStr)
	case StatusWarning:
		return utils.Colors.Warning(statusStr)
	case StatusCritical:
		return utils.Colors.Error(statusStr)
	default:
		return statusStr
	}
}

// SaveReport saves the report to a file
func SaveReport(report *HealthCheckReport, filePath string, format string) error {
	reporter := NewReporter(false)

	var content string
	var err error

	switch format {
	case "json":
		content, err = reporter.FormatJSON(report)
	case "csv":
		content = reporter.FormatCSV(report)
	case "html":
		content = reporter.FormatHTML(report)
	case "markdown", "md":
		content = reporter.FormatMarkdown(report)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}
