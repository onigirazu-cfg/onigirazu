package execution

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

// DisplayMode determines verbosity level when displaying results
type DisplayMode int

const (
	DisplayNormal DisplayMode = iota
	DisplayVerbose
	DisplayDebug
)

// Displayer formats and displays execution results
type Displayer struct {
	mode      DisplayMode
	useColors bool
}

// NewDisplayer creates a new result displayer
func NewDisplayer(mode DisplayMode, useColors bool) *Displayer {
	return &Displayer{
		mode:      mode,
		useColors: useColors,
	}
}

// DisplayExecution formats and displays a complete execution result
func (d *Displayer) DisplayExecution(result *ExecutionResult) {
	switch d.mode {
	case DisplayNormal:
		d.displayNormal(result)
	case DisplayVerbose:
		d.displayVerbose(result)
	case DisplayDebug:
		d.displayDebug(result)
	}
}

// displayNormal shows normal mode output
func (d *Displayer) displayNormal(result *ExecutionResult) {
	fmt.Println()
	d.printHeader("Execution Summary", "═")

	// Status line
	statusStr := result.Status
	if d.useColors {
		switch result.Status {
		case "success":
			statusStr = utils.Colors.Success(statusStr)
		case "partial_success":
			statusStr = utils.Colors.Warning(statusStr)
		case "failed":
			statusStr = utils.Colors.Error(statusStr)
		}
	}

	fmt.Printf("Status:          %s\n", statusStr)
	fmt.Printf("Playbook:        %s\n", result.PlaybookName)
	fmt.Printf("Total Hosts:     %d\n", result.TotalHosts)
	fmt.Printf("Execution ID:    %s\n", result.ExecutionID)
	fmt.Printf("Started:         %s\n", result.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration:        %v\n", result.Duration.Round(0))
	fmt.Println()

	// Summary stats
	fmt.Printf("Results:         ")
	if d.useColors {
		fmt.Printf("%s / %s / %s / %s\n",
			utils.Colors.Success(fmt.Sprintf("✓ %d success", result.TotalSuccess)),
			utils.Colors.Warning(fmt.Sprintf("⟳ %d changed", result.TotalChanged)),
			utils.Colors.Error(fmt.Sprintf("✗ %d failed", result.TotalFailed)),
			fmt.Sprintf("⊝ %d skipped", result.TotalSkipped),
		)
	} else {
		fmt.Printf("✓ %d success / ⟳ %d changed / ✗ %d failed / ⊝ %d skipped\n",
			result.TotalSuccess, result.TotalChanged, result.TotalFailed, result.TotalSkipped)
	}
	fmt.Println()

	// Per-task summary
	d.printHeader("Tasks", "─")
	for i, task := range result.Tasks {
		successRate := 0
		if task.Total > 0 {
			successRate = (task.Success * 100) / task.Total
		}

		statusIcon := "✓"
		if task.Failed > 0 {
			if d.useColors {
				statusIcon = utils.Colors.Error("✗")
			}
		}

		fmt.Printf("%d. %s %s [%d/%d] (%d%% success)\n",
			i+1, statusIcon, task.Name, task.Success, task.Total, successRate)

		// Show error types if any failed
		if task.Failed > 0 && len(task.ErrorsByType) > 0 {
			d.printErrorsByType(task, 3)
		}
	}

	fmt.Println()
}

// displayVerbose shows verbose mode output with error details
func (d *Displayer) displayVerbose(result *ExecutionResult) {
	d.displayNormal(result)

	d.printHeader("Detailed Results by Task", "═")

	for i, task := range result.Tasks {
		fmt.Printf("\n[Task %d] %s\n", i+1, task.Name)
		fmt.Println(strings.Repeat("─", 80))

		// Show aggregated stats
		fmt.Printf("  Total: %d  Success: %d  Changed: %d  Failed: %d  Skipped: %d\n",
			task.Total, task.Success, task.Changed, task.Failed, task.Skipped)
		fmt.Printf("  Duration: %v\n", task.Duration.Round(0))

		// Show errors with examples
		if task.Failed > 0 {
			fmt.Println("\n  Errors by Type:")
			for errorType, hosts := range task.ErrorsByType {
				d.printErrorTypeDetails(task, errorType, hosts, 5)
			}
		}

		fmt.Println()
	}
}

// displayDebug shows debug mode output (complete JSON structure)
func (d *Displayer) displayDebug(result *ExecutionResult) {
	d.displayVerbose(result)

	d.printHeader("Complete Execution Data (JSON)", "═")
	fmt.Println()

	// Print as JSON
	encoder := NewJSONPrinter()
	encoder.PrintJSON(result)

	fmt.Println()
}

// printErrorsByType displays errors grouped by type (normal mode - compact)
func (d *Displayer) printErrorsByType(task TaskResult, maxExamples int) {
	errorTypes := make([]string, 0, len(task.ErrorsByType))
	for errorType := range task.ErrorsByType {
		errorTypes = append(errorTypes, errorType)
	}
	sort.Strings(errorTypes)

	for _, errorType := range errorTypes {
		hosts := task.ErrorsByType[errorType]
		hostStr := strings.Join(hosts, ", ")
		if len(hostStr) > 80 {
			hostStr = hostStr[:77] + "..."
		}

		if d.useColors {
			fmt.Printf("   %s [%d hosts]: %s\n",
				utils.Colors.Error(errorType),
				len(hosts),
				hostStr,
			)
		} else {
			fmt.Printf("   %s [%d hosts]: %s\n", errorType, len(hosts), hostStr)
		}
	}
}

// printErrorTypeDetails displays detailed error information
func (d *Displayer) printErrorTypeDetails(task TaskResult, errorType string, hosts []string, maxExamples int) {
	count := len(hosts)
	if count > maxExamples {
		displayed := hosts[:maxExamples]
		remaining := count - maxExamples

		if d.useColors {
			fmt.Printf("    %s [%d hosts]:\n", utils.Colors.Error(errorType), count)
		} else {
			fmt.Printf("    %s [%d hosts]:\n", errorType, count)
		}

		// Show examples with error details
		for _, hostname := range displayed {
			if result, exists := task.HostResults[hostname]; exists {
				fmt.Printf("      ├─ %s: %s\n", hostname, result.Error)
			}
		}

		fmt.Printf("      └─ [%d more hosts with same error...]\n", remaining)
	} else {
		if d.useColors {
			fmt.Printf("    %s [%d hosts]:\n", utils.Colors.Error(errorType), count)
		} else {
			fmt.Printf("    %s [%d hosts]:\n", errorType, count)
		}

		for i, hostname := range hosts {
			if result, exists := task.HostResults[hostname]; exists {
				if i < len(hosts)-1 {
					fmt.Printf("      ├─ %s: %s\n", hostname, result.Error)
				} else {
					fmt.Printf("      └─ %s: %s\n", hostname, result.Error)
				}
			}
		}
	}
}

// printHeader prints a formatted section header
func (d *Displayer) printHeader(title string, charType string) {
	width := 80
	if charType == "═" {
		fmt.Printf("\n%s %s\n", title, strings.Repeat(charType, width-len(title)-1))
	} else {
		fmt.Printf("\n%s\n%s\n", title, strings.Repeat(charType, len(title)))
	}
}

// JSONPrinter handles JSON output
type JSONPrinter struct{}

// NewJSONPrinter creates a new JSON printer
func NewJSONPrinter() *JSONPrinter {
	return &JSONPrinter{}
}

// PrintJSON prints the complete execution result as JSON
func (jp *JSONPrinter) PrintJSON(result *ExecutionResult) {
	// Build complete output including all host details
	output := map[string]interface{}{
		"execution_id":  result.ExecutionID,
		"timestamp":     result.Timestamp,
		"playbook_path": result.PlaybookPath,
		"playbook_name": result.PlaybookName,
		"total_hosts":   result.TotalHosts,
		"status":        result.Status,
		"duration":      result.Duration.String(),
		"start_time":    result.StartTime,
		"end_time":      result.EndTime,
		"summary": map[string]int{
			"success": result.TotalSuccess,
			"failed":  result.TotalFailed,
			"changed": result.TotalChanged,
			"skipped": result.TotalSkipped,
		},
		"tasks": result.Tasks,
	}

	// Pretty print JSON
	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}
