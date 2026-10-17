package execution

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	output    io.Writer
	mode      DisplayMode
	useColors bool
}

// NewDisplayer creates a new result displayer
func NewDisplayer(mode DisplayMode, useColors bool) *Displayer {
	return NewDisplayerWithWriter(mode, useColors, os.Stdout)
}

// NewDisplayerWithWriter creates a displayer using output.
func NewDisplayerWithWriter(mode DisplayMode, useColors bool, output io.Writer) *Displayer {
	if output == nil {
		output = os.Stdout
	}
	return &Displayer{
		output:    output,
		mode:      mode,
		useColors: useColors,
	}
}

// DisplayExecution formats and displays a complete execution result
func (d *Displayer) DisplayExecution(result *ExecutionResult) {
	if d.output == nil {
		d.output = os.Stdout
	}
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
	fmt.Fprintln(d.output)
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

	fmt.Fprintf(d.output, "Status:          %s\n", statusStr)
	fmt.Fprintf(d.output, "Playbook:        %s\n", result.PlaybookName)
	fmt.Fprintf(d.output, "Total Hosts:     %d\n", result.TotalHosts)
	fmt.Fprintf(d.output, "Execution ID:    %s\n", result.ExecutionID)
	fmt.Fprintf(d.output, "Started:         %s\n", result.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(d.output, "Duration:        %v\n", result.Duration.Round(0))
	fmt.Fprintln(d.output)

	// Summary stats
	fmt.Fprintf(d.output, "Results:         ")
	if d.useColors {
		fmt.Fprintf(d.output, "%s / %s / %s / %s\n",
			utils.Colors.Success(fmt.Sprintf("✓ %d success", result.TotalSuccess)),
			utils.Colors.Warning(fmt.Sprintf("⟳ %d changed", result.TotalChanged)),
			utils.Colors.Error(fmt.Sprintf("✗ %d failed", result.TotalFailed)),
			fmt.Sprintf("⊝ %d skipped", result.TotalSkipped),
		)
	} else {
		fmt.Fprintf(d.output, "✓ %d success / ⟳ %d changed / ✗ %d failed / ⊝ %d skipped\n",
			result.TotalSuccess, result.TotalChanged, result.TotalFailed, result.TotalSkipped)
	}
	if result.TotalIgnored > 0 {
		fmt.Fprintf(d.output, "                 %d failed task(s) handled by ignore_errors or rescue\n", result.TotalIgnored)
	}
	fmt.Fprintln(d.output)

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

		fmt.Fprintf(d.output, "%d. %s %s [%d/%d] (%d%% success)\n",
			i+1, statusIcon, task.Name, task.Success, task.Total, successRate)

		// Show error types if any failed
		if task.Failed > 0 && len(task.ErrorsByType) > 0 {
			d.printErrorsByType(task, 3)
		}
	}

	fmt.Fprintln(d.output)
}

// displayVerbose shows verbose mode output with error details
func (d *Displayer) displayVerbose(result *ExecutionResult) {
	d.displayNormal(result)

	d.printHeader("Detailed Results by Task", "═")

	for i, task := range result.Tasks {
		fmt.Fprintf(d.output, "\n[Task %d] %s\n", i+1, task.Name)
		fmt.Fprintln(d.output, strings.Repeat("─", 80))

		// Show aggregated stats
		fmt.Fprintf(d.output, "  Total: %d  Success: %d  Changed: %d  Failed: %d  Skipped: %d\n",
			task.Total, task.Success, task.Changed, task.Failed, task.Skipped)
		fmt.Fprintf(d.output, "  Duration: %v\n", task.Duration.Round(0))

		// Show errors with examples
		if task.Failed > 0 {
			fmt.Fprintln(d.output, "\n  Errors by Type:")
			for errorType, hosts := range task.ErrorsByType {
				d.printErrorTypeDetails(task, errorType, hosts, 5)
			}
		}

		fmt.Fprintln(d.output)
	}
}

// displayDebug shows debug mode output (complete JSON structure)
func (d *Displayer) displayDebug(result *ExecutionResult) {
	d.displayVerbose(result)

	d.printHeader("Complete Execution Data (JSON)", "═")
	fmt.Fprintln(d.output)

	// Print as JSON
	encoder := NewJSONPrinterWithWriter(d.output)
	encoder.PrintJSON(result)

	fmt.Fprintln(d.output)
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
			fmt.Fprintf(d.output, "   %s [%d hosts]: %s\n",
				utils.Colors.Error(errorType),
				len(hosts),
				hostStr,
			)
		} else {
			fmt.Fprintf(d.output, "   %s [%d hosts]: %s\n", errorType, len(hosts), hostStr)
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
			fmt.Fprintf(d.output, "    %s [%d hosts]:\n", utils.Colors.Error(errorType), count)
		} else {
			fmt.Fprintf(d.output, "    %s [%d hosts]:\n", errorType, count)
		}

		// Show examples with error details
		for _, hostname := range displayed {
			if result, exists := task.HostResults[hostname]; exists {
				fmt.Fprintf(d.output, "      ├─ %s: %s\n", hostname, result.Error)
			}
		}

		fmt.Fprintf(d.output, "      └─ [%d more hosts with same error...]\n", remaining)
	} else {
		if d.useColors {
			fmt.Fprintf(d.output, "    %s [%d hosts]:\n", utils.Colors.Error(errorType), count)
		} else {
			fmt.Fprintf(d.output, "    %s [%d hosts]:\n", errorType, count)
		}

		for i, hostname := range hosts {
			if result, exists := task.HostResults[hostname]; exists {
				if i < len(hosts)-1 {
					fmt.Fprintf(d.output, "      ├─ %s: %s\n", hostname, result.Error)
				} else {
					fmt.Fprintf(d.output, "      └─ %s: %s\n", hostname, result.Error)
				}
			}
		}
	}
}

// printHeader prints a formatted section header
func (d *Displayer) printHeader(title string, charType string) {
	width := 80
	if charType == "═" {
		fmt.Fprintf(d.output, "\n%s %s\n", title, strings.Repeat(charType, width-len(title)-1))
	} else {
		fmt.Fprintf(d.output, "\n%s\n%s\n", title, strings.Repeat(charType, len(title)))
	}
}

// JSONPrinter handles JSON output
type JSONPrinter struct{ output io.Writer }

// NewJSONPrinter creates a new JSON printer
func NewJSONPrinter() *JSONPrinter {
	return NewJSONPrinterWithWriter(os.Stdout)
}

// NewJSONPrinterWithWriter creates a JSON printer using output.
func NewJSONPrinterWithWriter(output io.Writer) *JSONPrinter {
	if output == nil {
		output = os.Stdout
	}
	return &JSONPrinter{output: output}
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
	out := jp.output
	if out == nil {
		out = os.Stdout
	}
	fmt.Fprintln(out, string(data))
}
