package drift

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/onigirazu-cfg/onigirazu/internal/brand"
)

// Reporter generates drift reports in various formats
type Reporter struct {
	config *DriftConfig
}

// NewReporter creates a new reporter
func NewReporter(config *DriftConfig) *Reporter {
	return &Reporter{
		config: config,
	}
}

// GenerateReport generates a report in the specified format
func (r *Reporter) GenerateReport(report *DriftReport, format string) (string, error) {
	switch format {
	case "text", "":
		return r.generateTextReport(report), nil
	case "json":
		return r.generateJSONReport(report)
	case "html":
		return r.generateHTMLReport(report)
	default:
		return "", fmt.Errorf("unsupported report format: %s", format)
	}
}

// generateTextReport generates a text report
func (r *Reporter) generateTextReport(report *DriftReport) string {
	var sb strings.Builder

	// Header
	sb.WriteString(color.CyanString("=== Drift Detection Report ===\n"))
	sb.WriteString(fmt.Sprintf("Report ID: %s\n", report.ID))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", report.Timestamp.Format(time.RFC3339)))
	if report.SnapshotID != "" {
		sb.WriteString(fmt.Sprintf("Snapshot ID: %s\n", report.SnapshotID))
	}
	sb.WriteString(fmt.Sprintf("Duration: %v\n", report.Duration))
	sb.WriteString("\n")

	// Summary
	sb.WriteString(color.YellowString("Summary:\n"))
	sb.WriteString(fmt.Sprintf("  Total Drifts: %d\n", report.TotalDrifts))

	if report.CriticalDrifts > 0 {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", color.RedString("Critical"), report.CriticalDrifts))
	}
	if report.HighDrifts > 0 {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", color.HiRedString("High"), report.HighDrifts))
	}
	if report.MediumDrifts > 0 {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", color.YellowString("Medium"), report.MediumDrifts))
	}
	if report.LowDrifts > 0 {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", color.GreenString("Low"), report.LowDrifts))
	}

	if report.FixedDrifts > 0 {
		sb.WriteString(fmt.Sprintf("  Fixed: %d\n", report.FixedDrifts))
	}
	if report.FailedFixes > 0 {
		sb.WriteString(fmt.Sprintf("  Failed Fixes: %d\n", report.FailedFixes))
	}
	sb.WriteString("\n")

	// Drifts by type
	if len(report.DriftsByType) > 0 {
		sb.WriteString(color.YellowString("Drifts by Type:\n"))
		for driftType, count := range report.DriftsByType {
			sb.WriteString(fmt.Sprintf("  %s: %d\n", driftType, count))
		}
		sb.WriteString("\n")
	}

	// Drifts by host
	if len(report.DriftsByHost) > 0 {
		sb.WriteString(color.YellowString("Drifts by Host:\n"))
		for host, count := range report.DriftsByHost {
			sb.WriteString(fmt.Sprintf("  %s: %d\n", host, count))
		}
		sb.WriteString("\n")
	}

	// Detailed drift items
	if len(report.Items) > 0 {
		sb.WriteString(color.CyanString("=== Detected Drifts ===\n\n"))

		for i, item := range report.Items {
			sb.WriteString(fmt.Sprintf("%d. ", i+1))
			sb.WriteString(r.formatDriftItem(&item))
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(color.GreenString("✓ No drift detected!\n"))
	}

	return sb.String()
}

// formatDriftItem formats a single drift item
func (r *Reporter) formatDriftItem(item *DriftItem) string {
	var sb strings.Builder

	// Resource info with severity color
	severityColor := r.getSeverityColor(item.Severity)
	sb.WriteString(severityColor(fmt.Sprintf("[%s] ", strings.ToUpper(string(item.Severity)))))
	sb.WriteString(fmt.Sprintf("%s: %s on %s\n", item.Type, item.Resource, item.Host))

	// Status
	statusStr := r.formatStatus(item.Status)
	sb.WriteString(fmt.Sprintf("   Status: %s\n", statusStr))

	// Message
	if item.Message != "" {
		sb.WriteString(fmt.Sprintf("   Message: %s\n", item.Message))
	}

	// Diff
	if item.Diff != "" {
		sb.WriteString("   Changes:\n")
		for _, line := range strings.Split(item.Diff, "\n") {
			if line != "" {
				sb.WriteString(fmt.Sprintf("     %s\n", line))
			}
		}
	}

	// Auto-fix info
	if item.CanAutoFix {
		sb.WriteString(color.GreenString("   ✓ Can be auto-fixed\n"))
		if item.FixOperation != nil {
			sb.WriteString(fmt.Sprintf("   Fix: %s\n", item.FixOperation.Module))
		}
	} else {
		sb.WriteString(color.YellowString("   ⚠ Manual fix required\n"))
	}

	// Fixed timestamp
	if item.FixedAt != nil {
		sb.WriteString(fmt.Sprintf("   Fixed at: %s\n", item.FixedAt.Format(time.RFC3339)))
	}

	return sb.String()
}

// getSeverityColor returns color function for severity
func (r *Reporter) getSeverityColor(severity DriftSeverity) func(string, ...interface{}) string {
	switch severity {
	case SeverityCritical:
		return color.RedString
	case SeverityHigh:
		return color.HiRedString
	case SeverityMedium:
		return color.YellowString
	case SeverityLow:
		return color.GreenString
	default:
		return color.WhiteString
	}
}

// formatStatus formats drift status
func (r *Reporter) formatStatus(status DriftStatus) string {
	switch status {
	case StatusDetected:
		return color.YellowString("Detected")
	case StatusFixed:
		return color.GreenString("Fixed")
	case StatusIgnored:
		return color.BlueString("Ignored")
	case StatusFailed:
		return color.RedString("Failed")
	default:
		return string(status)
	}
}

// generateJSONReport generates a JSON report
func (r *Reporter) generateJSONReport(report *DriftReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// generateHTMLReport generates an HTML report
func (r *Reporter) generateHTMLReport(report *DriftReport) (string, error) {
	tmpl := `<!doctype html><html lang="en"><head>` + brand.ReportHead + `
<title>Drift detection report - {{.ID}}</title></head><body><main class="onz-page">
` + brand.HeaderOpen + `Drift report` + brand.HeaderClose + `
<h1 class="onz-title">Drift detection report</h1>
<p class="onz-sub">{{.ID}} · {{.Timestamp.Format "2006-01-02 15:04:05 MST"}} · {{.Duration}}</p>
<div class="onz-stats">
<div class="onz-stat"><b>{{.TotalDrifts}}</b><span>Total drifts</span></div>
<div class="onz-stat bad"><b>{{.CriticalDrifts}}</b><span>Critical</span></div>
<div class="onz-stat warn"><b>{{.HighDrifts}}</b><span>High</span></div>
<div class="onz-stat warn"><b>{{.MediumDrifts}}</b><span>Medium</span></div>
<div class="onz-stat muted"><b>{{.LowDrifts}}</b><span>Low</span></div>
<div class="onz-stat ok"><b>{{.FixedDrifts}}</b><span>Fixed</span></div>
<div class="onz-stat bad"><b>{{.FailedFixes}}</b><span>Failed fixes</span></div>
</div>
{{if or .SnapshotID .PlaybookID}}<section class="onz-card"><h2>Report details</h2><div class="onz-body">
{{if .SnapshotID}}<p>Snapshot ID: {{.SnapshotID}}</p>{{end}}
{{if .PlaybookID}}<p>Playbook ID: {{.PlaybookID}}</p>{{end}}
</div></section>{{end}}
{{if .DriftsByType}}<section class="onz-card"><h2>Drifts by type</h2><div class="onz-scroll"><table class="onz-table"><tr><th>Type</th><th class="num">Count</th></tr>{{range $type, $count := .DriftsByType}}<tr><td>{{$type}}</td><td class="num">{{$count}}</td></tr>{{end}}</table></div></section>{{end}}
{{if .DriftsByHost}}<section class="onz-card"><h2>Drifts by host</h2><div class="onz-scroll"><table class="onz-table"><tr><th>Host</th><th class="num">Count</th></tr>{{range $host, $count := .DriftsByHost}}<tr><td>{{$host}}</td><td class="num">{{$count}}</td></tr>{{end}}</table></div></section>{{end}}
{{range .Items}}<section class="onz-card">
<h2>{{.Type}}: {{.Resource}} on {{.Host}} <span class="onz-badge {{tone (printf "%s" .Severity)}}">{{.Severity}}</span> <span class="onz-badge {{tone (printf "%s" .Status)}}">{{.Status}}</span></h2>
<div class="onz-body">
{{if .Message}}<p>{{.Message}}</p>{{end}}
<p class="muted">Detected: {{.DetectedAt.Format "2006-01-02 15:04:05 MST"}}{{if .FixedAt}} · Fixed: {{.FixedAt.Format "2006-01-02 15:04:05 MST"}}{{end}}</p>
{{if .CanAutoFix}}<span class="onz-badge ok">auto-fixable</span>{{end}}
{{if .Diff}}<pre class="onz-diff">{{range diffLines .Diff}}<span class="{{.Class}}">{{.Text}}</span>{{end}}</pre>{{end}}
</div></section>{{else}}<p class="muted">No drift detected.</p>{{end}}
<footer class="onz-foot">Generated {{.Timestamp.Format "2006-01-02 15:04:05 MST"}}</footer></main></body></html>`

	t, err := template.New("report").Funcs(template.FuncMap{
		"tone":      brand.Tone,
		"diffLines": reportDiffLines,
	}).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var sb strings.Builder
	if err := t.Execute(&sb, report); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return sb.String(), nil
}

// reportDiffLines classifies diff lines without introducing trusted HTML.
func reportDiffLines(diff string) []struct{ Class, Text string } {
	var lines []struct{ Class, Text string }
	for _, text := range strings.Split(strings.TrimSuffix(diff, "\n"), "\n") {
		class := ""
		switch {
		case strings.HasPrefix(text, "---"), strings.HasPrefix(text, "+++"):
			class = "hdr"
		case strings.HasPrefix(text, "+"):
			class = "add"
		case strings.HasPrefix(text, "-"):
			class = "del"
		case strings.HasPrefix(text, "@@"):
			class = "hunk"
		}
		lines = append(lines, struct{ Class, Text string }{class, text})
	}
	return lines
}

// SaveReportToFile saves a report to a file
func (r *Reporter) SaveReportToFile(report *DriftReport, filename string, format string) error {
	content, err := r.GenerateReport(report, format)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(content), 0600)
}
