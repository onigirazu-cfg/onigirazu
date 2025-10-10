package adhoc

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

// Formatter handles formatting of ad-hoc command results
type Formatter struct {
	noColor bool
}

// NewFormatter creates a new result formatter
func NewFormatter(noColor bool) *Formatter {
	return &Formatter{
		noColor: noColor,
	}
}

// Format formats the summary based on output format
func (f *Formatter) Format(summary *Summary, format string) (string, error) {
	switch format {
	case "json":
		return f.formatJSON(summary)
	case "yaml":
		return f.formatYAML(summary)
	case "table":
		return f.formatTable(summary), nil
	default:
		return f.formatText(summary), nil
	}
}

// formatText formats results as colored text (default)
func (f *Formatter) formatText(summary *Summary) string {
	var sb strings.Builder

	// Print individual results
	for _, result := range summary.Results {
		hostName := result.Host.Name
		if hostName == "" {
			hostName = result.Host.Address
		}

		if result.Error != nil {
			// Error occurred
			if f.noColor {
				sb.WriteString(fmt.Sprintf("%s | FAILED => %v\n", hostName, result.Error))
			} else {
				sb.WriteString(fmt.Sprintf("%s | %s => %v\n",
					hostName,
					utils.Colors.Error("FAILED"),
					result.Error))
			}
			continue
		}

		if result.Result == nil {
			// Skipped
			if f.noColor {
				sb.WriteString(fmt.Sprintf("%s | SKIPPED\n", hostName))
			} else {
				sb.WriteString(fmt.Sprintf("%s | %s\n",
					hostName,
					utils.Colors.Warning("SKIPPED")))
			}
			continue
		}

		// Success or failure
		if result.Result.Failed {
			if f.noColor {
				sb.WriteString(fmt.Sprintf("%s | FAILED => %s\n", hostName, result.Result.Error))
			} else {
				sb.WriteString(fmt.Sprintf("%s | %s => %s\n",
					hostName,
					utils.Colors.Error("FAILED"),
					result.Result.Error))
			}
		} else {
			status := "SUCCESS"
			if result.Result.Changed {
				status = "CHANGED"
			}

			if f.noColor {
				sb.WriteString(fmt.Sprintf("%s | %s", hostName, status))
			} else {
				color := utils.Colors.Success
				if result.Result.Changed {
					color = utils.Colors.Changed
				}
				sb.WriteString(fmt.Sprintf("%s | %s", hostName, color(status)))
			}

			// Add output if present
			if len(result.Result.Output) > 0 {
				// Prioritize stdout if available (for command/shell modules)
				if stdout, ok := result.Result.Output["stdout"]; ok {
					stdoutStr := fmt.Sprintf("%v", stdout)
					if stdoutStr != "" {
						sb.WriteString(" => \n")
						// Indent stdout output
						lines := strings.Split(stdoutStr, "\n")
						for _, line := range lines {
							if line != "" {
								sb.WriteString(fmt.Sprintf("    %s\n", line))
							}
						}
					}
				} else {
					// Show meaningful output fields based on priority
					// Priority order: ping, connection, msg, message, then others
					priorityKeys := []string{"ping", "connection", "msg"}
					shown := false

					for _, key := range priorityKeys {
						if v, ok := result.Result.Output[key]; ok {
							if !shown {
								sb.WriteString(" => ")
								shown = true
							} else {
								sb.WriteString(", ")
							}
							sb.WriteString(fmt.Sprintf("%s: %v", key, v))
						}
					}

					// If nothing shown yet, show first non-generic field
					if !shown {
						for k, v := range result.Result.Output {
							if k != "message" && k != "command" && k != "host" && k != "address" && k != "user" && k != "port" {
								sb.WriteString(fmt.Sprintf(" => %s: %v", k, v))
								shown = true
								break
							}
						}
					}

					// If still nothing shown, show message
					if !shown {
						if msg, ok := result.Result.Output["message"]; ok {
							sb.WriteString(fmt.Sprintf(" => %v", msg))
						}
					}
				}
			}
			sb.WriteString("\n")
		}
	}

	// Print summary
	sb.WriteString("\n")
	sb.WriteString(f.formatSummaryLine(summary))

	return sb.String()
}

// formatSummaryLine formats the summary line
func (f *Formatter) formatSummaryLine(summary *Summary) string {
	if f.noColor {
		return fmt.Sprintf("=== Summary: Total: %d | Success: %d | Failed: %d | Changed: %d | Duration: %s ===\n",
			summary.Total,
			summary.Success,
			summary.Failed,
			summary.Changed,
			summary.Duration.Round(10*1000000), // Round to 10ms
		)
	}

	return fmt.Sprintf("=== Summary: Total: %d | %s: %d | %s: %d | %s: %d | Duration: %s ===\n",
		summary.Total,
		utils.Colors.Success("Success"),
		summary.Success,
		utils.Colors.Error("Failed"),
		summary.Failed,
		utils.Colors.Changed("Changed"),
		summary.Changed,
		summary.Duration.Round(10*1000000),
	)
}

// formatJSON formats results as JSON
func (f *Formatter) formatJSON(summary *Summary) (string, error) {
	// Create simplified structure for JSON output
	type jsonResult struct {
		Host     string `json:"host"`
		Status   string `json:"status"`
		Changed  bool   `json:"changed,omitempty"`
		Message  string `json:"message,omitempty"`
		Error    string `json:"error,omitempty"`
		Duration string `json:"duration"`
	}

	type jsonSummary struct {
		Total    int          `json:"total"`
		Success  int          `json:"success"`
		Failed   int          `json:"failed"`
		Changed  int          `json:"changed"`
		Skipped  int          `json:"skipped"`
		Duration string       `json:"duration"`
		Results  []jsonResult `json:"results"`
	}

	results := make([]jsonResult, len(summary.Results))
	for i, result := range summary.Results {
		hostName := result.Host.Name
		if hostName == "" {
			hostName = result.Host.Address
		}

		jr := jsonResult{
			Host:     hostName,
			Duration: result.Duration.String(),
		}

		if result.Error != nil {
			jr.Status = "failed"
			jr.Error = result.Error.Error()
		} else if result.Result == nil {
			jr.Status = "skipped"
		} else if result.Result.Failed {
			jr.Status = "failed"
			jr.Error = result.Result.Error
		} else {
			jr.Status = "success"
			jr.Changed = result.Result.Changed
			// Build message from output with priority
			if len(result.Result.Output) > 0 {
				// Prioritize stdout if available (for command/shell modules)
				if stdout, ok := result.Result.Output["stdout"]; ok {
					jr.Message = fmt.Sprintf("%v", stdout)
				} else {
					// Show meaningful output fields based on priority
					priorityKeys := []string{"ping", "connection", "msg"}
					var parts []string

					for _, key := range priorityKeys {
						if v, ok := result.Result.Output[key]; ok {
							parts = append(parts, fmt.Sprintf("%s: %v", key, v))
						}
					}

					if len(parts) > 0 {
						jr.Message = strings.Join(parts, ", ")
					} else {
						// If nothing shown yet, show first non-generic field
						for k, v := range result.Result.Output {
							if k != "message" && k != "command" && k != "host" && k != "address" && k != "user" && k != "port" {
								jr.Message = fmt.Sprintf("%s: %v", k, v)
								break
							}
						}
						// If still nothing shown, show message field
						if jr.Message == "" {
							if msg, ok := result.Result.Output["message"]; ok {
								jr.Message = fmt.Sprintf("%v", msg)
							}
						}
					}
				}
			}
		}

		results[i] = jr
	}

	output := jsonSummary{
		Total:    summary.Total,
		Success:  summary.Success,
		Failed:   summary.Failed,
		Changed:  summary.Changed,
		Skipped:  summary.Skipped,
		Duration: summary.Duration.String(),
		Results:  results,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// formatYAML formats results as YAML
func (f *Formatter) formatYAML(summary *Summary) (string, error) {
	// Reuse JSON structure
	jsonStr, err := f.formatJSON(summary)
	if err != nil {
		return "", err
	}

	// Convert JSON to map
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return "", err
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(yamlData), nil
}

// formatTable formats results as a table
func (f *Formatter) formatTable(summary *Summary) string {
	var sb strings.Builder

	// Header
	sb.WriteString("+------------------+----------+---------+------------------+\n")
	sb.WriteString("| Host             | Status   | Changed | Duration         |\n")
	sb.WriteString("+------------------+----------+---------+------------------+\n")

	// Rows
	for _, result := range summary.Results {
		hostName := result.Host.Name
		if hostName == "" {
			hostName = result.Host.Address
		}

		// Truncate host name if too long
		if len(hostName) > 16 {
			hostName = hostName[:13] + "..."
		}

		var status string
		var changed string

		if result.Error != nil {
			status = "FAILED"
			changed = "-"
		} else if result.Result == nil {
			status = "SKIPPED"
			changed = "-"
		} else if result.Result.Failed {
			status = "FAILED"
			changed = "-"
		} else {
			status = "SUCCESS"
			if result.Result.Changed {
				changed = "Yes"
			} else {
				changed = "No"
			}
		}

		sb.WriteString(fmt.Sprintf("| %-16s | %-8s | %-7s | %-16s |\n",
			hostName,
			status,
			changed,
			result.Duration.Round(1000000), // Round to 1ms
		))
	}

	// Footer
	sb.WriteString("+------------------+----------+---------+------------------+\n")

	// Summary
	sb.WriteString(fmt.Sprintf("\nTotal: %d | Success: %d | Failed: %d | Changed: %d | Duration: %s\n",
		summary.Total,
		summary.Success,
		summary.Failed,
		summary.Changed,
		summary.Duration.Round(10*1000000),
	))

	return sb.String()
}
