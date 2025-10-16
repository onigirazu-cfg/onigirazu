package audit

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Reporter generates audit reports in various formats
type Reporter struct {
	records []ExecutionRecord
}

// NewReporter creates a new audit reporter
func NewReporter(records []ExecutionRecord) *Reporter {
	return &Reporter{
		records: records,
	}
}

// FormatType represents the output format
type FormatType string

const (
	FormatText     FormatType = "text"
	FormatJSON     FormatType = "json"
	FormatCSV      FormatType = "csv"
	FormatHTML     FormatType = "html"
	FormatMarkdown FormatType = "markdown"
)

// Generate generates a report in the specified format
func (r *Reporter) Generate(format FormatType) (string, error) {
	switch format {
	case FormatText:
		return r.generateTextReport()
	case FormatJSON:
		return r.generateJSONReport()
	case FormatCSV:
		return r.generateCSVReport()
	case FormatHTML:
		return r.generateHTMLReport()
	case FormatMarkdown:
		return r.generateMarkdownReport()
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}
}

// generateTextReport generates a text format report
func (r *Reporter) generateTextReport() (string, error) {
	var buf bytes.Buffer

	buf.WriteString("\n" + strings.Repeat("=", 80) + "\n")
	buf.WriteString("AUDIT REPORT - EXECUTION HISTORY\n")
	buf.WriteString(strings.Repeat("=", 80) + "\n\n")

	if len(r.records) == 0 {
		buf.WriteString("No execution records found.\n")
		return buf.String(), nil
	}

	buf.WriteString(fmt.Sprintf("Total Executions: %d\n", len(r.records)))
	buf.WriteString(strings.Repeat("-", 80) + "\n\n")

	for i, record := range r.records {
		buf.WriteString(fmt.Sprintf("[%d] Execution ID: %s\n", i+1, record.ID))
		buf.WriteString(fmt.Sprintf("    Playbook: %s\n", record.PlaybookPath))
		buf.WriteString(fmt.Sprintf("    User: %s\n", record.User))
		buf.WriteString(fmt.Sprintf("    Status: %s\n", colorizeStatus(string(record.Status))))
		buf.WriteString(fmt.Sprintf("    Start Time: %s\n", record.StartTime.Format(time.RFC3339)))
		buf.WriteString(fmt.Sprintf("    Duration: %.2f seconds\n", record.Duration))
		buf.WriteString(fmt.Sprintf("    Total Tasks: %d (Success: %d, Failed: %d, Skipped: %d)\n",
			record.TotalTasks, record.SuccessfulTasks, record.FailedTasks, record.SkippedTasks))

		if len(record.AffectedHosts) > 0 {
			buf.WriteString(fmt.Sprintf("    Affected Hosts: %s\n", strings.Join(record.AffectedHosts, ", ")))
		}

		if record.FailedTasks > 0 && record.ErrorMessage != "" {
			buf.WriteString(fmt.Sprintf("    Error: %s\n", record.ErrorMessage))
		}

		if len(record.Plays) > 0 {
			buf.WriteString("    Plays:\n")
			for _, play := range record.Plays {
				buf.WriteString(fmt.Sprintf("      - %s: %d tasks\n", play.Name, len(play.Tasks)))
			}
		}

		buf.WriteString("\n")
	}

	buf.WriteString(strings.Repeat("=", 80) + "\n")
	return buf.String(), nil
}

// generateJSONReport generates a JSON format report
func (r *Reporter) generateJSONReport() (string, error) {
	data, err := json.MarshalIndent(map[string]interface{}{
		"timestamp": time.Now(),
		"total":     len(r.records),
		"records":   r.records,
	}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(data), nil
}

// generateCSVReport generates a CSV format report
func (r *Reporter) generateCSVReport() (string, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	headers := []string{
		"Execution ID", "Playbook", "User", "Status", "Start Time",
		"End Time", "Duration (s)", "Total Tasks", "Successful", "Failed",
		"Skipped", "Exit Code", "Hosts Count", "Error",
	}
	if err := writer.Write(headers); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write records
	for _, record := range r.records {
		row := []string{
			record.ID,
			record.PlaybookPath,
			record.User,
			string(record.Status),
			record.StartTime.Format(time.RFC3339),
			record.EndTime.Format(time.RFC3339),
			fmt.Sprintf("%.2f", record.Duration),
			fmt.Sprintf("%d", record.TotalTasks),
			fmt.Sprintf("%d", record.SuccessfulTasks),
			fmt.Sprintf("%d", record.FailedTasks),
			fmt.Sprintf("%d", record.SkippedTasks),
			fmt.Sprintf("%d", record.ExitCode),
			fmt.Sprintf("%d", len(record.AffectedHosts)),
			record.ErrorMessage,
		}

		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.String(), nil
}

// generateHTMLReport generates an HTML format report
func (r *Reporter) generateHTMLReport() (string, error) {
	var buf bytes.Buffer

	// Write HTML header
	buf.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Onigirazu Audit Report</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            border-bottom: 3px solid #007bff;
            padding-bottom: 10px;
        }
        h2 {
            color: #555;
            margin-top: 20px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 10px;
        }
        th {
            background-color: #007bff;
            color: white;
            padding: 12px;
            text-align: left;
            font-weight: bold;
        }
        tr:nth-child(even) {
            background-color: #f9f9f9;
        }
        tr:hover {
            background-color: #f0f0f0;
        }
        td {
            border-bottom: 1px solid #ddd;
            padding: 12px;
        }
        .status-success {
            background-color: #d4edda;
            color: #155724;
            padding: 4px 8px;
            border-radius: 4px;
        }
        .status-failure {
            background-color: #f8d7da;
            color: #721c24;
            padding: 4px 8px;
            border-radius: 4px;
        }
        .status-running {
            background-color: #d1ecf1;
            color: #0c5460;
            padding: 4px 8px;
            border-radius: 4px;
        }
        .metric {
            display: inline-block;
            margin-right: 30px;
            margin-bottom: 15px;
        }
        .metric-value {
            font-size: 24px;
            font-weight: bold;
            color: #007bff;
        }
        .metric-label {
            color: #666;
            font-size: 14px;
        }
        .timestamp {
            color: #999;
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🍙 Onigirazu Audit Report</h1>
`)

	// Write summary metrics
	successCount := 0
	failureCount := 0
	for _, record := range r.records {
		if record.Status == StatusSuccess {
			successCount++
		} else if record.Status == StatusFailure {
			failureCount++
		}
	}

	buf.WriteString(`        <h2>Summary</h2>
`)
	buf.WriteString(fmt.Sprintf(`        <div class="metric">
            <div class="metric-value">%d</div>
            <div class="metric-label">Total Executions</div>
        </div>
`, len(r.records)))
	buf.WriteString(fmt.Sprintf(`        <div class="metric">
            <div class="metric-value" style="color: #28a745;">%d</div>
            <div class="metric-label">Successful</div>
        </div>
`, successCount))
	buf.WriteString(fmt.Sprintf(`        <div class="metric">
            <div class="metric-value" style="color: #dc3545;">%d</div>
            <div class="metric-label">Failed</div>
        </div>
`, failureCount))

	buf.WriteString(`        <h2>Execution Records</h2>
        <table>
            <thead>
                <tr>
                    <th>Playbook</th>
                    <th>User</th>
                    <th>Status</th>
                    <th>Start Time</th>
                    <th>Duration (s)</th>
                    <th>Tasks</th>
                    <th>Exit Code</th>
                </tr>
            </thead>
            <tbody>
`)

	// Write records
	for _, record := range r.records {
		statusClass := "status-" + string(record.Status)
		if record.Status == "running" {
			statusClass = "status-running"
		}

		buf.WriteString(fmt.Sprintf(`                <tr>
                    <td>%s</td>
                    <td>%s</td>
                    <td><span class="%s">%s</span></td>
                    <td>%s</td>
                    <td>%.2f</td>
                    <td>%d (✓%d/✗%d/⊘%d)</td>
                    <td>%d</td>
                </tr>
`,
			record.PlaybookPath, record.User, statusClass, record.Status,
			record.StartTime.Format("2006-01-02 15:04:05"),
			record.Duration, record.TotalTasks, record.SuccessfulTasks,
			record.FailedTasks, record.SkippedTasks, record.ExitCode))
	}

	buf.WriteString(`            </tbody>
        </table>
        <p class="timestamp">Generated: `)
	buf.WriteString(time.Now().Format("2006-01-02 15:04:05 MST"))
	buf.WriteString(`</p>
    </div>
</body>
</html>
`)

	return buf.String(), nil
}

// generateMarkdownReport generates a Markdown format report
func (r *Reporter) generateMarkdownReport() (string, error) {
	var buf bytes.Buffer

	buf.WriteString("# Onigirazu Audit Report\n\n")
	buf.WriteString("## Summary\n\n")

	if len(r.records) == 0 {
		buf.WriteString("No execution records found.\n")
		return buf.String(), nil
	}

	successCount := 0
	failureCount := 0
	totalTasks := 0
	totalFailed := 0

	for _, record := range r.records {
		if record.Status == StatusSuccess {
			successCount++
		} else if record.Status == StatusFailure {
			failureCount++
		}
		totalTasks += record.TotalTasks
		totalFailed += record.FailedTasks
	}

	buf.WriteString("| Metric | Value |\n")
	buf.WriteString("|--------|-------|\n")
	buf.WriteString(fmt.Sprintf("| Total Executions | %d |\n", len(r.records)))
	buf.WriteString(fmt.Sprintf("| Successful | %d |\n", successCount))
	buf.WriteString(fmt.Sprintf("| Failed | %d |\n", failureCount))
	buf.WriteString(fmt.Sprintf("| Total Tasks | %d |\n", totalTasks))
	buf.WriteString(fmt.Sprintf("| Failed Tasks | %d |\n", totalFailed))
	buf.WriteString("\n")

	// Write execution records table
	buf.WriteString("## Execution Records\n\n")
	buf.WriteString("| Playbook | User | Status | Start Time | Duration | Tasks | Error |\n")
	buf.WriteString("|----------|------|--------|------------|----------|-------|-------|\n")

	for _, record := range r.records {
		taskSummary := fmt.Sprintf("%d (✓%d/✗%d)", record.TotalTasks, record.SuccessfulTasks, record.FailedTasks)
		errMsg := record.ErrorMessage
		if len(errMsg) > 50 {
			errMsg = errMsg[:50] + "..."
		}

		buf.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %.2fs | %s | %s |\n",
			record.PlaybookPath, record.User, record.Status,
			record.StartTime.Format("2006-01-02 15:04:05"),
			record.Duration, taskSummary, errMsg))
	}

	buf.WriteString("\n")

	// Write detailed records if not too many
	if len(r.records) <= 10 {
		buf.WriteString("## Detailed Records\n\n")

		for i, record := range r.records {
			buf.WriteString(fmt.Sprintf("### Execution %d: %s\n\n", i+1, record.ID))
			buf.WriteString(fmt.Sprintf("- **Playbook**: %s\n", record.PlaybookPath))
			buf.WriteString(fmt.Sprintf("- **User**: %s\n", record.User))
			buf.WriteString(fmt.Sprintf("- **Status**: %s\n", record.Status))
			buf.WriteString(fmt.Sprintf("- **Start**: %s\n", record.StartTime.Format(time.RFC3339)))
			buf.WriteString(fmt.Sprintf("- **Duration**: %.2f seconds\n", record.Duration))
			buf.WriteString(fmt.Sprintf("- **Exit Code**: %d\n", record.ExitCode))

			if len(record.AffectedHosts) > 0 {
				buf.WriteString(fmt.Sprintf("- **Hosts**: %s\n", strings.Join(record.AffectedHosts, ", ")))
			}

			if record.ErrorMessage != "" {
				buf.WriteString(fmt.Sprintf("- **Error**: %s\n", record.ErrorMessage))
			}

			buf.WriteString("\n")
		}
	}

	buf.WriteString("---\n\n")
	buf.WriteString(fmt.Sprintf("*Generated at %s*\n", time.Now().Format(time.RFC3339)))

	return buf.String(), nil
}

// GenerateDetailedReport generates a detailed report for a single execution
func (r *Reporter) GenerateDetailedReport(recordID string, format FormatType) (string, error) {
	if len(r.records) == 0 {
		return "", fmt.Errorf("no records found")
	}

	var record *ExecutionRecord
	for i := range r.records {
		if r.records[i].ID == recordID {
			record = &r.records[i]
			break
		}
	}

	if record == nil {
		return "", fmt.Errorf("record not found: %s", recordID)
	}

	switch format {
	case FormatJSON:
		data, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil

	case FormatMarkdown:
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("# Execution Report: %s\n\n", record.ID))
		buf.WriteString("## Overview\n\n")
		buf.WriteString(fmt.Sprintf("- **Playbook**: %s\n", record.PlaybookPath))
		buf.WriteString(fmt.Sprintf("- **Inventory**: %s\n", record.InventoryPath))
		buf.WriteString(fmt.Sprintf("- **User**: %s\n", record.User))
		buf.WriteString(fmt.Sprintf("- **Status**: %s\n", record.Status))
		buf.WriteString(fmt.Sprintf("- **Start Time**: %s\n", record.StartTime.Format(time.RFC3339)))
		buf.WriteString(fmt.Sprintf("- **End Time**: %s\n", record.EndTime.Format(time.RFC3339)))
		buf.WriteString(fmt.Sprintf("- **Duration**: %.2f seconds\n", record.Duration))
		buf.WriteString(fmt.Sprintf("- **Exit Code**: %d\n", record.ExitCode))

		if record.ErrorMessage != "" {
			buf.WriteString(fmt.Sprintf("- **Error**: %s\n", record.ErrorMessage))
		}

		buf.WriteString("\n## Task Summary\n\n")
		buf.WriteString("| Status | Count |\n|--------|-------|\n")
		buf.WriteString(fmt.Sprintf("| Successful | %d |\n", record.SuccessfulTasks))
		buf.WriteString(fmt.Sprintf("| Failed | %d |\n", record.FailedTasks))
		buf.WriteString(fmt.Sprintf("| Skipped | %d |\n", record.SkippedTasks))
		buf.WriteString(fmt.Sprintf("| **Total** | **%d** |\n", record.TotalTasks))

		if len(record.AffectedHosts) > 0 {
			buf.WriteString("\n## Affected Hosts\n\n")
			for _, host := range record.AffectedHosts {
				buf.WriteString(fmt.Sprintf("- %s\n", host))
			}
		}

		if len(record.Plays) > 0 {
			buf.WriteString("\n## Plays\n\n")
			for i, play := range record.Plays {
				buf.WriteString(fmt.Sprintf("### Play %d: %s\n\n", i+1, play.Name))
				buf.WriteString(fmt.Sprintf("- **Hosts**: %s\n", strings.Join(play.Hosts, ", ")))
				buf.WriteString(fmt.Sprintf("- **Duration**: %.2f seconds\n", play.Duration))
				buf.WriteString(fmt.Sprintf("- **Status**: %s\n\n", play.Status))

				if len(play.Tasks) > 0 {
					buf.WriteString("#### Tasks\n\n")
					buf.WriteString("| Task | Module | Host | Status | Duration |\n")
					buf.WriteString("|------|--------|------|--------|----------|\n")
					for _, task := range play.Tasks {
						buf.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %.2fs |\n",
							task.Name, task.Module, task.Host, task.Status, task.Duration))
					}
					buf.WriteString("\n")
				}
			}
		}

		return buf.String(), nil

	case FormatText:
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("Execution Report: %s\n", record.ID))
		buf.WriteString(fmt.Sprintf("Playbook: %s\n", record.PlaybookPath))
		buf.WriteString(fmt.Sprintf("Status: %s\n", record.Status))
		buf.WriteString(fmt.Sprintf("Duration: %.2f seconds\n", record.Duration))
		buf.WriteString(fmt.Sprintf("Total Tasks: %d\n", record.TotalTasks))
		return buf.String(), nil

	default:
		return "", fmt.Errorf("unsupported format for detailed report: %s", format)
	}
}

// Helper functions

func colorizeStatus(status string) string {
	switch status {
	case "success":
		return "✓ SUCCESS"
	case "failure":
		return "✗ FAILURE"
	case "running":
		return "⟳ RUNNING"
	case "skipped":
		return "⊘ SKIPPED"
	default:
		return status
	}
}

// WriteReport writes a report to an io.Writer
func (r *Reporter) WriteReport(w io.Writer, format FormatType) error {
	report, err := r.Generate(format)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, report)
	return err
}
