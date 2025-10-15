package drift

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
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
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Drift Detection Report - {{.ID}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            border-bottom: 3px solid #4CAF50;
            padding-bottom: 10px;
        }
        h2 {
            color: #555;
            margin-top: 30px;
        }
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin: 20px 0;
        }
        .summary-card {
            padding: 20px;
            border-radius: 6px;
            background-color: #f9f9f9;
            border-left: 4px solid #4CAF50;
        }
        .summary-card h3 {
            margin: 0 0 10px 0;
            color: #666;
            font-size: 14px;
            text-transform: uppercase;
        }
        .summary-card .value {
            font-size: 32px;
            font-weight: bold;
            color: #333;
        }
        .severity-critical { border-left-color: #f44336; }
        .severity-high { border-left-color: #ff9800; }
        .severity-medium { border-left-color: #ffc107; }
        .severity-low { border-left-color: #4CAF50; }
        .drift-item {
            margin: 15px 0;
            padding: 15px;
            border: 1px solid #ddd;
            border-radius: 6px;
            background-color: #fafafa;
        }
        .drift-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
        }
        .drift-title {
            font-weight: bold;
            font-size: 16px;
        }
        .severity-badge {
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: bold;
            color: white;
        }
        .badge-critical { background-color: #f44336; }
        .badge-high { background-color: #ff9800; }
        .badge-medium { background-color: #ffc107; color: #333; }
        .badge-low { background-color: #4CAF50; }
        .drift-details {
            margin-top: 10px;
            font-size: 14px;
            color: #666;
        }
        .diff {
            background-color: #f5f5f5;
            padding: 10px;
            border-radius: 4px;
            font-family: 'Courier New', monospace;
            font-size: 13px;
            margin-top: 10px;
            white-space: pre-wrap;
        }
        .status-fixed { color: #4CAF50; }
        .status-detected { color: #ff9800; }
        .status-failed { color: #f44336; }
        .can-autofix {
            display: inline-block;
            margin-top: 10px;
            padding: 4px 8px;
            background-color: #e8f5e9;
            color: #2e7d32;
            border-radius: 4px;
            font-size: 12px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #f5f5f5;
            font-weight: bold;
            color: #333;
        }
        .no-drift {
            text-align: center;
            padding: 40px;
            color: #4CAF50;
            font-size: 18px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔍 Drift Detection Report</h1>

        <div class="summary">
            <div class="summary-card">
                <h3>Report ID</h3>
                <div class="value" style="font-size: 16px;">{{.ID}}</div>
            </div>
            <div class="summary-card">
                <h3>Total Drifts</h3>
                <div class="value">{{.TotalDrifts}}</div>
            </div>
            <div class="summary-card severity-critical">
                <h3>Critical</h3>
                <div class="value">{{.CriticalDrifts}}</div>
            </div>
            <div class="summary-card severity-high">
                <h3>High</h3>
                <div class="value">{{.HighDrifts}}</div>
            </div>
            <div class="summary-card severity-medium">
                <h3>Medium</h3>
                <div class="value">{{.MediumDrifts}}</div>
            </div>
            <div class="summary-card severity-low">
                <h3>Low</h3>
                <div class="value">{{.LowDrifts}}</div>
            </div>
        </div>

        <h2>Report Details</h2>
        <table>
            <tr>
                <th>Timestamp</th>
                <td>{{.Timestamp.Format "2006-01-02 15:04:05"}}</td>
            </tr>
            {{if .SnapshotID}}
            <tr>
                <th>Snapshot ID</th>
                <td>{{.SnapshotID}}</td>
            </tr>
            {{end}}
            {{if .PlaybookID}}
            <tr>
                <th>Playbook ID</th>
                <td>{{.PlaybookID}}</td>
            </tr>
            {{end}}
            <tr>
                <th>Duration</th>
                <td>{{.Duration}}</td>
            </tr>
        </table>

        {{if .Items}}
        <h2>Detected Drifts</h2>
        {{range $index, $item := .Items}}
        <div class="drift-item">
            <div class="drift-header">
                <div class="drift-title">
                    {{$item.Type}}: {{$item.Resource}} on {{$item.Host}}
                </div>
                <span class="severity-badge badge-{{$item.Severity}}">{{$item.Severity}}</span>
            </div>
            <div class="drift-details">
                <strong>Status:</strong> <span class="status-{{$item.Status}}">{{$item.Status}}</span><br>
                {{if $item.Message}}
                <strong>Message:</strong> {{$item.Message}}<br>
                {{end}}
                <strong>Detected:</strong> {{$item.DetectedAt.Format "2006-01-02 15:04:05"}}<br>
                {{if $item.FixedAt}}
                <strong>Fixed:</strong> {{$item.FixedAt.Format "2006-01-02 15:04:05"}}<br>
                {{end}}
            </div>
            {{if $item.Diff}}
            <div class="diff">{{$item.Diff}}</div>
            {{end}}
            {{if $item.CanAutoFix}}
            <div class="can-autofix">✓ Can be auto-fixed</div>
            {{end}}
        </div>
        {{end}}
        {{else}}
        <div class="no-drift">
            ✓ No drift detected! Your infrastructure is in sync.
        </div>
        {{end}}
    </div>
</body>
</html>`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var sb strings.Builder
	if err := t.Execute(&sb, report); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return sb.String(), nil
}

// SaveReportToFile saves a report to a file
func (r *Reporter) SaveReportToFile(report *DriftReport, filename string, format string) error {
	content, err := r.GenerateReport(report, format)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(content), 0644)
}
