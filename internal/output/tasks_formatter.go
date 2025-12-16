package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/internal/taskpreview"
	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

// FormatTasksText formats task preview as human-readable text
func FormatTasksText(result *taskpreview.PreviewResult) string {
	var output strings.Builder

	output.WriteString("Tasks that would execute:\n\n")

	// Show filter information
	if len(result.Tags) > 0 {
		output.WriteString(fmt.Sprintf("Filters applied: --tags %s\n", strings.Join(result.Tags, ",")))
	}
	if len(result.SkipTags) > 0 {
		output.WriteString(fmt.Sprintf("Skip filters: --skip-tags %s\n", strings.Join(result.SkipTags, ",")))
	}
	if len(result.Tags) > 0 || len(result.SkipTags) > 0 {
		output.WriteString("\n")
	}

	// Show plays and tasks
	for _, play := range result.Plays {
		output.WriteString(fmt.Sprintf("Play %d: %s (Hosts: %s)\n", play.Index+1, play.Name, play.Hosts))

		for _, task := range play.Tasks {
			var statusSymbol string
			statusText := ""

			if task.Status == taskpreview.StatusExecute || task.Status == taskpreview.StatusUnconditional {
				statusSymbol = "✓"
				if task.Status == taskpreview.StatusUnconditional {
					statusText = " [ALWAYS TAG]"
				}
			} else {
				statusSymbol = "✗"
				statusText = fmt.Sprintf(" [SKIPPED: %s]", formatSkipReason(task.Status))
			}

			// Format tags
			tagsStr := ""
			if len(task.Tags) > 0 {
				tagsStr = "[" + strings.Join(task.Tags, ", ") + "] "
			}

			// Apply color only if colors are enabled
			if utils.ColorsEnabled {
				// Use color codes for execute/skip
				color := utils.Green
				if task.Status != taskpreview.StatusExecute && task.Status != taskpreview.StatusUnconditional {
					color = utils.Red
				}
				output.WriteString(fmt.Sprintf("  %s%s%s %s%s%s\n",
					color, statusSymbol, utils.Reset,
					tagsStr, task.Name, statusText))
			} else {
				output.WriteString(fmt.Sprintf("  %s %s%s%s\n",
					statusSymbol, tagsStr, task.Name, statusText))
			}
		}

		// Play summary
		if play.Summary.Skipped > 0 {
			output.WriteString(fmt.Sprintf("  Summary: %d would execute, %d skipped\n",
				play.Summary.Would, play.Summary.Skipped))
		} else {
			output.WriteString(fmt.Sprintf("  Summary: %d would execute\n", play.Summary.Would))
		}
		output.WriteString("\n")
	}

	// Global summary
	output.WriteString("Overall Summary:\n")
	output.WriteString(fmt.Sprintf("  Total tasks:       %d\n", result.GlobalSummary.TotalTasks))
	output.WriteString(fmt.Sprintf("  Would execute:     %d\n", result.GlobalSummary.WouldExecute))
	output.WriteString(fmt.Sprintf("  Would skip:        %d\n", result.GlobalSummary.Skipped))

	if len(result.GlobalSummary.SkipDetails) > 0 {
		output.WriteString("  Skip reasons:\n")
		for reason, count := range result.GlobalSummary.SkipDetails {
			output.WriteString(fmt.Sprintf("    - %s: %d\n", reason, count))
		}
	}

	return output.String()
}

// FormatTasksJSON formats task preview as JSON
func FormatTasksJSON(result *taskpreview.PreviewResult) string {
	output := map[string]interface{}{
		"filters":   result.Tags,
		"skip_tags": result.SkipTags,
		"plays":     formatPlaysForJSON(result.Plays),
		"summary":   result.GlobalSummary,
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	return string(data)
}

// FormatTasksYAML formats task preview as YAML
func FormatTasksYAML(result *taskpreview.PreviewResult) string {
	output := map[string]interface{}{
		"filters":   result.Tags,
		"skip_tags": result.SkipTags,
		"plays":     formatPlaysForJSON(result.Plays),
		"summary":   result.GlobalSummary,
	}

	data, _ := yaml.Marshal(output)
	return string(data)
}

// FormatTasksCSV formats task preview as CSV
func FormatTasksCSV(result *taskpreview.PreviewResult) string {
	var output strings.Builder
	w := csv.NewWriter(&output)

	// Write header
	_ = w.Write([]string{"Play", "Task Name", "Tags", "Module", "Status", "Skip Reason"})

	for _, play := range result.Plays {
		for _, task := range play.Tasks {
			status := "execute"
			if task.Status != taskpreview.StatusExecute && task.Status != taskpreview.StatusUnconditional {
				status = "skip"
			}

			_ = w.Write([]string{
				play.Name,
				task.Name,
				strings.Join(task.Tags, "; "),
				task.Module,
				status,
				task.SkipReason,
			})
		}
	}

	w.Flush()
	return output.String()
}

// formatPlaysForJSON formats plays for JSON/YAML output
func formatPlaysForJSON(plays []taskpreview.PlayPreview) []map[string]interface{} {
	var result []map[string]interface{}

	for _, play := range plays {
		tasks := []map[string]interface{}{}
		for _, task := range play.Tasks {
			taskMap := map[string]interface{}{
				"name":        task.Name,
				"module":      task.Module,
				"tags":        task.Tags,
				"status":      string(task.Status),
				"skip_reason": task.SkipReason,
			}
			tasks = append(tasks, taskMap)
		}

		result = append(result, map[string]interface{}{
			"name":    play.Name,
			"hosts":   play.Hosts,
			"tasks":   tasks,
			"summary": play.Summary,
		})
	}

	return result
}

// formatSkipReason formats skip reason for display
func formatSkipReason(status taskpreview.ExecutionStatus) string {
	switch status {
	case taskpreview.StatusSkipNever:
		return "NEVER TAG"
	case taskpreview.StatusSkipTags:
		return "TAG MISMATCH"
	case taskpreview.StatusSkipSkipTags:
		return "SKIP-TAG MATCH"
	case taskpreview.StatusSkipCondition:
		return "CONDITION FAILED"
	default:
		return "UNKNOWN"
	}
}
