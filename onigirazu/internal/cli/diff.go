package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/state"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

// TaskDiff represents the difference status of a task
type TaskDiff struct {
	Task       *types.Task
	Status     DiffStatus
	Changes    []string
	PrevResult *types.TaskResult
}

// DiffStatus represents the status of a task in diff
type DiffStatus string

const (
	DiffStatusNew       DiffStatus = "new"       // Task is new (not in state)
	DiffStatusModified  DiffStatus = "modified"  // Task exists but args changed
	DiffStatusUnchanged DiffStatus = "unchanged" // Task exists and unchanged
	DiffStatusRemoved   DiffStatus = "removed"   // Task in state but not in playbook
)

// newDiffCmd creates the diff command
func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff [playbook]",
		Short: "Show differences between playbook and current state",
		Long: `Show what changes would be made by comparing the playbook with current state.

This command analyzes the playbook and compares it with the last execution state to show:
- New tasks that will be executed for the first time
- Modified tasks where arguments or configuration changed
- Unchanged tasks that match the previous execution
- Removed tasks that were in previous state but not in current playbook

This helps you understand the impact of changes before running 'apply'.`,
		Example: `  # Show differences
  onigirazu diff production.yml

  # Show differences with custom state file
  onigirazu diff production.yml --state custom-state.json

  # Show detailed differences
  onigirazu diff production.yml --verbose

  # Show only changed tasks
  onigirazu diff production.yml --changed-only`,
		Args: cobra.ExactArgs(1),
		RunE: runDiff,
	}

	// Add command-specific flags
	cmd.Flags().StringP("state", "s", ".onigirazu-state", "State file path")
	cmd.Flags().Bool("changed-only", false, "Show only new and modified tasks")
	cmd.Flags().Bool("detailed", false, "Show detailed changes for each task")
	cmd.Flags().StringP("output", "o", "text", "Output format: text, json, yaml")

	return cmd
}

func runDiff(cmd *cobra.Command, args []string) error {
	playbookPath := args[0]

	// Get flags
	verbose, _ := cmd.Flags().GetBool("verbose")
	statePath, _ := cmd.Flags().GetString("state")
	changedOnly, _ := cmd.Flags().GetBool("changed-only")
	detailed, _ := cmd.Flags().GetBool("detailed")
	outputFormat, _ := cmd.Flags().GetString("output")
	noColor, _ := cmd.Flags().GetBool("no-color")

	// Check if playbook exists
	if _, err := os.Stat(playbookPath); os.IsNotExist(err) {
		return fmt.Errorf("playbook file not found: %s", playbookPath)
	}

	// Parse playbook
	if verbose {
		fmt.Printf("📖 Parsing playbook: %s\n", playbookPath)
	}

	p := parser.New()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	playbook, err := p.ParsePlaybook(ctx, playbookPath)
	if err != nil {
		return fmt.Errorf("failed to parse playbook: %w", err)
	}

	// Load state
	if verbose {
		fmt.Printf("📊 Loading state from: %s\n", statePath)
	}

	stateManager := state.New(statePath)
	currentState, err := stateManager.LoadState()
	if err != nil {
		if verbose {
			fmt.Printf("⚠️  Could not load state (will treat all tasks as new): %v\n", err)
		}
		currentState = &types.State{
			Variables: make(map[string]interface{}),
			Checksums: make(map[string]string),
		}
	}

	// Calculate differences
	diffs := calculateDiffs(playbook, currentState)

	// Output results
	switch outputFormat {
	case "json":
		return outputDiffJSON(diffs)
	case "yaml":
		return outputDiffYAML(diffs)
	default:
		return outputDiffText(diffs, changedOnly, detailed, verbose, noColor)
	}
}

// calculateDiffs compares playbook with state and returns differences
func calculateDiffs(playbook *types.Playbook, currentState *types.State) map[string][]*TaskDiff {
	diffs := make(map[string][]*TaskDiff)

	// Build a map of previous task results by task ID
	prevResults := make(map[string]*types.TaskResult)
	for _, playResult := range currentState.Results {
		for _, taskResult := range playResult.Tasks {
			taskID := generateTaskID(taskResult.TaskName, taskResult.Host)
			prevResults[taskID] = &taskResult
		}
	}

	// Track which previous tasks we've seen
	seenTasks := make(map[string]bool)

	// Compare each task in the playbook
	for _, play := range playbook.Plays {
		playName := play.Name
		if playName == "" {
			playName = "Unnamed Play"
		}

		for _, task := range play.Tasks {
			// For now, we'll use "localhost" as default host
			// In real implementation, this would iterate over actual hosts
			taskID := generateTaskID(task.Name, "localhost")
			seenTasks[taskID] = true

			diff := &TaskDiff{
				Task: &task,
			}

			if prevResult, exists := prevResults[taskID]; exists {
				// Task exists in previous state
				diff.PrevResult = prevResult

				// Check if task has changed
				if hasTaskChanged(&task, prevResult) {
					diff.Status = DiffStatusModified
					diff.Changes = detectChanges(&task, prevResult)
				} else {
					diff.Status = DiffStatusUnchanged
				}
			} else {
				// Task is new
				diff.Status = DiffStatusNew
			}

			diffs[playName] = append(diffs[playName], diff)
		}
	}

	// Find removed tasks (in state but not in playbook)
	for taskID, prevResult := range prevResults {
		if !seenTasks[taskID] {
			diff := &TaskDiff{
				Task: &types.Task{
					Name:   prevResult.TaskName,
					Module: prevResult.Module,
				},
				Status:     DiffStatusRemoved,
				PrevResult: prevResult,
			}

			// Add to a special "Removed Tasks" play
			diffs["Removed Tasks"] = append(diffs["Removed Tasks"], diff)
		}
	}

	return diffs
}

// generateTaskID creates a unique ID for a task
func generateTaskID(taskName, host string) string {
	data := fmt.Sprintf("%s:%s", taskName, host)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:8])
}

// hasTaskChanged checks if a task has changed compared to previous execution
func hasTaskChanged(task *types.Task, prevResult *types.TaskResult) bool {
	// Check if module changed
	if task.Module != prevResult.Module {
		return true
	}

	// Check if args changed by comparing JSON representation
	// Since TaskResult doesn't store args, we'll compare the task args
	// with a hash stored in the output (if available)
	if argsHash, ok := prevResult.Output["args_hash"].(string); ok {
		currentHash := calculateArgsHash(task.Args)
		return argsHash != currentHash
	}

	// If no hash available, assume changed if task was previously changed
	// This is a conservative approach
	return prevResult.Changed
}

// detectChanges identifies specific changes in a task
func detectChanges(task *types.Task, prevResult *types.TaskResult) []string {
	var changes []string

	// Check module change
	if task.Module != prevResult.Module {
		changes = append(changes, fmt.Sprintf("Module changed: %s → %s", prevResult.Module, task.Module))
	}

	// Since TaskResult doesn't store args, we can only detect that args changed
	// by comparing hashes or noting that the task definition changed
	if argsHash, ok := prevResult.Output["args_hash"].(string); ok {
		currentHash := calculateArgsHash(task.Args)
		if argsHash != currentHash {
			changes = append(changes, "Task arguments have changed")
			// List current args for reference
			if len(task.Args) > 0 {
				changes = append(changes, "Current arguments:")
				for key, value := range task.Args {
					changes = append(changes, fmt.Sprintf("  %s: %v", key, value))
				}
			}
		}
	} else {
		// No hash available, just note that we can't determine specific changes
		changes = append(changes, "Task definition may have changed (no previous args hash)")
	}

	return changes
}

// calculateArgsHash calculates a hash of task arguments
func calculateArgsHash(args map[string]interface{}) string {
	data, _ := json.Marshal(args)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// outputDiffText outputs differences in text format
func outputDiffText(diffs map[string][]*TaskDiff, changedOnly, detailed, verbose, noColor bool) error {
	// Count statistics
	stats := struct {
		New       int
		Modified  int
		Unchanged int
		Removed   int
	}{}

	for _, playDiffs := range diffs {
		for _, diff := range playDiffs {
			switch diff.Status {
			case DiffStatusNew:
				stats.New++
			case DiffStatusModified:
				stats.Modified++
			case DiffStatusUnchanged:
				stats.Unchanged++
			case DiffStatusRemoved:
				stats.Removed++
			}
		}
	}

	// Print header
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("📊 Playbook Diff Analysis")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// Print summary
	fmt.Println("Summary:")
	if !noColor {
		fmt.Printf("  %s %d new tasks\n", utils.Colors.Success("✓"), stats.New)
		fmt.Printf("  %s %d modified tasks\n", utils.Colors.Warning("~"), stats.Modified)
		fmt.Printf("  %s %d unchanged tasks\n", utils.Colors.Info("="), stats.Unchanged)
		if stats.Removed > 0 {
			fmt.Printf("  %s %d removed tasks\n", utils.Colors.Error("✗"), stats.Removed)
		}
	} else {
		fmt.Printf("  + %d new tasks\n", stats.New)
		fmt.Printf("  ~ %d modified tasks\n", stats.Modified)
		fmt.Printf("  = %d unchanged tasks\n", stats.Unchanged)
		if stats.Removed > 0 {
			fmt.Printf("  - %d removed tasks\n", stats.Removed)
		}
	}
	fmt.Println()

	// Print details for each play
	for playName, playDiffs := range diffs {
		// Skip if no changes and changedOnly is set
		if changedOnly {
			hasChanges := false
			for _, diff := range playDiffs {
				if diff.Status != DiffStatusUnchanged {
					hasChanges = true
					break
				}
			}
			if !hasChanges {
				continue
			}
		}

		fmt.Println("───────────────────────────────────────────────────────────")
		fmt.Printf("Play: %s\n", playName)
		fmt.Println("───────────────────────────────────────────────────────────")
		fmt.Println()

		for _, diff := range playDiffs {
			// Skip unchanged tasks if changedOnly is set
			if changedOnly && diff.Status == DiffStatusUnchanged {
				continue
			}

			printTaskDiff(diff, detailed, verbose, noColor)
		}

		fmt.Println()
	}

	// Print footer
	fmt.Println("═══════════════════════════════════════════════════════════")
	if stats.New > 0 || stats.Modified > 0 {
		fmt.Println("⚠️  Changes detected! Review carefully before applying.")
	} else if stats.Removed > 0 {
		fmt.Println("⚠️  Some tasks were removed from the playbook.")
	} else {
		fmt.Println("✓ No changes detected. Playbook matches current state.")
	}
	fmt.Println("═══════════════════════════════════════════════════════════")

	return nil
}

// printTaskDiff prints a single task diff
func printTaskDiff(diff *TaskDiff, detailed, verbose, noColor bool) {
	var statusSymbol, statusColor string

	switch diff.Status {
	case DiffStatusNew:
		statusSymbol = "+"
		if !noColor {
			statusColor = utils.Colors.Success(statusSymbol)
		} else {
			statusColor = statusSymbol
		}
	case DiffStatusModified:
		statusSymbol = "~"
		if !noColor {
			statusColor = utils.Colors.Warning(statusSymbol)
		} else {
			statusColor = statusSymbol
		}
	case DiffStatusUnchanged:
		statusSymbol = "="
		if !noColor {
			statusColor = utils.Colors.Info(statusSymbol)
		} else {
			statusColor = statusSymbol
		}
	case DiffStatusRemoved:
		statusSymbol = "-"
		if !noColor {
			statusColor = utils.Colors.Error(statusSymbol)
		} else {
			statusColor = statusSymbol
		}
	}

	fmt.Printf("  %s %s\n", statusColor, diff.Task.Name)
	fmt.Printf("    Module: %s\n", diff.Task.Module)

	if diff.Status == DiffStatusNew {
		fmt.Printf("    Status: New task (will be executed)\n")
	} else if diff.Status == DiffStatusRemoved {
		fmt.Printf("    Status: Removed from playbook\n")
	} else if diff.Status == DiffStatusModified {
		fmt.Printf("    Status: Modified\n")
		if len(diff.Changes) > 0 {
			fmt.Println("    Changes:")
			for _, change := range diff.Changes {
				fmt.Printf("      • %s\n", change)
			}
		}
	} else {
		fmt.Printf("    Status: Unchanged\n")
	}

	// Show detailed info if requested
	if detailed && len(diff.Task.Args) > 0 {
		fmt.Println("    Arguments:")
		for key, value := range diff.Task.Args {
			fmt.Printf("      %s: %v\n", key, value)
		}
	}

	// Show previous result if verbose
	if verbose && diff.PrevResult != nil {
		status := "success"
		if diff.PrevResult.Failed {
			status = "failed"
		} else if diff.PrevResult.Skipped {
			status = "skipped"
		}
		fmt.Printf("    Previous execution: %s\n", status)
		if diff.PrevResult.Changed {
			fmt.Println("    Previous result: Changed")
		} else {
			fmt.Println("    Previous result: No change")
		}
	}

	fmt.Println()
}

// outputDiffJSON outputs differences in JSON format
func outputDiffJSON(diffs map[string][]*TaskDiff) error {
	type JSONDiff struct {
		Play    string     `json:"play"`
		Task    string     `json:"task"`
		Module  string     `json:"module"`
		Status  DiffStatus `json:"status"`
		Changes []string   `json:"changes,omitempty"`
	}

	var output []JSONDiff

	for playName, playDiffs := range diffs {
		for _, diff := range playDiffs {
			output = append(output, JSONDiff{
				Play:    playName,
				Task:    diff.Task.Name,
				Module:  diff.Task.Module,
				Status:  diff.Status,
				Changes: diff.Changes,
			})
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// outputDiffYAML outputs differences in YAML format
func outputDiffYAML(diffs map[string][]*TaskDiff) error {
	// Simple YAML output (without external dependencies)
	for playName, playDiffs := range diffs {
		fmt.Printf("- play: %s\n", playName)
		fmt.Println("  tasks:")
		for _, diff := range playDiffs {
			fmt.Printf("    - task: %s\n", diff.Task.Name)
			fmt.Printf("      module: %s\n", diff.Task.Module)
			fmt.Printf("      status: %s\n", diff.Status)
			if len(diff.Changes) > 0 {
				fmt.Println("      changes:")
				for _, change := range diff.Changes {
					fmt.Printf("        - %s\n", strings.ReplaceAll(change, "\n", " "))
				}
			}
		}
	}

	return nil
}
