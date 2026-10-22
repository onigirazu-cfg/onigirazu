package healthcheck

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"text/tabwriter"

	"github.com/onigirazu-cfg/onigirazu/internal/brand"
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

	_ = w.Flush()
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
	out := `<!doctype html><html lang="en"><head>` + brand.ReportHead + `<title>Health check report</title></head><body><main class="onz-page">` + brand.Header("Health check report") + `
<h1 class="onz-title">Health check</h1><p class="onz-sub">Total duration ` + html.EscapeString(report.TotalDuration.String()) + `</p>
<div class="onz-stats"><div class="onz-stat ` + brand.Tone(string(report.OverallStatus)) + `"><b>` + html.EscapeString(string(report.OverallStatus)) + `</b><span>Overall</span></div>
<div class="onz-stat"><b>` + html.EscapeString(fmt.Sprint(report.Statistics.TotalHosts)) + `</b><span>Hosts</span></div>
<div class="onz-stat ok"><b>` + html.EscapeString(fmt.Sprint(report.Statistics.HealthyHosts)) + `</b><span>Healthy</span></div>
<div class="onz-stat warn"><b>` + html.EscapeString(fmt.Sprint(report.Statistics.WarningHosts)) + `</b><span>Warning</span></div>
<div class="onz-stat bad"><b>` + html.EscapeString(fmt.Sprint(report.Statistics.CriticalHosts)) + `</b><span>Critical</span></div></div>`
	for _, hostReport := range report.HostReports {
		out += `<section class="onz-card"><h2>` + html.EscapeString(hostReport.Host.Name) + ` (` + html.EscapeString(hostReport.Host.Address) + `) <span class="onz-badge ` + brand.Tone(string(hostReport.Status)) + `">` + html.EscapeString(string(hostReport.Status)) + `</span></h2>
<div class="onz-body">Duration: ` + html.EscapeString(hostReport.Duration.String()) + `</div>
<div class="onz-scroll"><table class="onz-table"><tr><th>Check type</th><th>Status</th><th>Message</th><th class="num">Duration</th></tr>`
		for _, check := range hostReport.Checks {
			out += `<tr><td>` + html.EscapeString(string(check.CheckType)) + `</td><td><span class="onz-badge ` + brand.Tone(string(check.Status)) + `">` + html.EscapeString(string(check.Status)) + `</span></td><td>` + html.EscapeString(check.Message) + `</td><td class="num">` + html.EscapeString(check.Duration.String()) + `</td></tr>`
		}
		out += `</table></div></section>`
	}
	out += `<footer class="onz-foot">Generated ` + html.EscapeString(report.Timestamp.Format("2006-01-02 15:04:05 MST")) + `</footer></main></body></html>`
	return out
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
