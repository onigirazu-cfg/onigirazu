package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/state"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// newStateCmd creates the state command with subcommands
func newStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Manage Onigirazu state",
		Long: `Manage and inspect Onigirazu state file.

The state file tracks:
- Last execution results
- Resource states
- Variable values
- File checksums

Subcommands:
  list  - List all resources in state
  show  - Show detailed information about a resource`,
		Example: `  # List all resources
  onigirazu state list

  # Show specific resource
  onigirazu state show web-server

  # List with custom state file
  onigirazu state list --state custom-state.json`,
	}

	// Add subcommands
	cmd.AddCommand(newStateListCmd())
	cmd.AddCommand(newStateShowCmd())

	return cmd
}

// newStateListCmd creates the state list subcommand
func newStateListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all resources in state",
		Long: `List all resources tracked in the state file.

Shows:
- Resource names (hosts)
- Last execution status
- Number of tasks executed
- Success/failure status`,
		Example: `  # List all resources
  onigirazu state list

  # List with custom state file
  onigirazu state list --state custom-state.json

  # List with verbose output
  onigirazu state list --verbose

  # List in JSON format
  onigirazu state list --output json`,
		RunE: runStateList,
	}

	// Add command-specific flags
	cmd.Flags().StringP("state", "s", "onigirazu.state", "State file path")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json, yaml)")
	cmd.Flags().String("filter", "", "Filter resources by name (supports wildcards)")

	return cmd
}

// newStateShowCmd creates the state show subcommand
func newStateShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [resource]",
		Short: "Show detailed information about a resource",
		Long: `Show detailed information about a specific resource in state.

Displays:
- Resource details
- Last execution results
- Task results
- Variables
- Timestamps`,
		Example: `  # Show resource details
  onigirazu state show web-server

  # Show with custom state file
  onigirazu state show web-server --state custom-state.json

  # Show in JSON format
  onigirazu state show web-server --output json`,
		Args: cobra.ExactArgs(1),
		RunE: runStateShow,
	}

	// Add command-specific flags
	cmd.Flags().StringP("state", "s", "onigirazu.state", "State file path")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json, yaml)")

	return cmd
}

func runStateList(cmd *cobra.Command, args []string) error {
	// Get flags
	statePath, _ := cmd.Flags().GetString("state")
	outputFormat, _ := cmd.Flags().GetString("output")
	filter, _ := cmd.Flags().GetString("filter")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Check if state file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return fmt.Errorf("state file not found: %s (run 'onigirazu apply' first)", statePath)
	}

	// Load state
	stateManager := state.New(statePath)
	stateData, err := stateManager.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Check if state is empty
	if len(stateData.Results) == 0 {
		fmt.Println("No resources found in state.")
		fmt.Println("Run 'onigirazu apply' to create state.")
		return nil
	}

	// Filter results if needed
	var filteredResults []types.PlayResult
	for _, result := range stateData.Results {
		playName := result.Name
		if playName == "" {
			playName = result.PlayName
		}
		if filter == "" || matchesFilter(playName, filter) {
			filteredResults = append(filteredResults, result)
		}
	}

	// Output based on format
	switch outputFormat {
	case "json":
		return outputJSON(filteredResults)
	case "yaml":
		return outputYAML(filteredResults)
	default:
		return outputTable(filteredResults, stateData, verbose)
	}
}

func runStateShow(cmd *cobra.Command, args []string) error {
	resourceName := args[0]

	// Get flags
	statePath, _ := cmd.Flags().GetString("state")
	outputFormat, _ := cmd.Flags().GetString("output")

	// Check if state file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return fmt.Errorf("state file not found: %s (run 'onigirazu apply' first)", statePath)
	}

	// Load state
	stateManager := state.New(statePath)
	stateData, err := stateManager.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Find resource
	var foundResult *types.PlayResult
	var foundHost *types.HostResult
	for _, result := range stateData.Results {
		// Check if play name matches (check both Name and PlayName)
		if result.Name == resourceName || result.PlayName == resourceName {
			foundResult = &result
			break
		}
		// Check if any host matches
		for _, hostResult := range result.Hosts {
			if hostResult.Host == resourceName {
				foundResult = &result
				foundHost = &hostResult
				break
			}
		}
		if foundResult != nil {
			break
		}
	}

	// Debug: print what we loaded if not found
	if foundResult == nil && os.Getenv("DEBUG_STATE") != "" {
		fmt.Fprintf(os.Stderr, "DEBUG: Looking for '%s'\n", resourceName)
		for i, result := range stateData.Results {
			fmt.Fprintf(os.Stderr, "DEBUG: Result[%d] Name='%s' PlayName='%s' Hosts=%d\n", i, result.Name, result.PlayName, len(result.Hosts))
			for _, h := range result.Hosts {
				fmt.Fprintf(os.Stderr, "DEBUG:   Host='%s'\n", h.Host)
			}
		}
	}

	if foundResult == nil {
		return fmt.Errorf("resource not found: %s", resourceName)
	}

	// Output based on format
	switch outputFormat {
	case "json":
		if foundHost != nil {
			return outputJSON(foundHost)
		}
		return outputJSON(foundResult)
	case "yaml":
		if foundHost != nil {
			return outputYAML(foundHost)
		}
		return outputYAML(foundResult)
	default:
		return outputResourceDetails(foundResult, foundHost, stateData)
	}
}

// Helper functions

func matchesFilter(name, filter string) bool {
	// Simple wildcard matching
	if strings.Contains(filter, "*") {
		prefix := strings.TrimSuffix(filter, "*")
		return strings.HasPrefix(name, prefix)
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(filter))
}

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func outputYAML(data interface{}) error {
	// For now, just output JSON (YAML support can be added later)
	return outputJSON(data)
}

func outputTable(results []types.PlayResult, stateData *types.State, verbose bool) error {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("State File Resources\n")
	fmt.Printf("Last Run: %s\n", stateData.LastRun.Format(time.RFC3339))
	if stateData.Playbook != "" {
		fmt.Printf("Playbook: %s\n", stateData.Playbook)
	}
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	for _, result := range results {
		// Play header
		status := "✅"
		if !result.Success {
			status = "❌"
		}
		playName := result.Name
		if playName == "" {
			playName = result.PlayName
		}
		fmt.Printf("%s Play: %s\n", status, playName)
		fmt.Printf("   Duration: %v\n", result.EndTime.Sub(result.StartTime).Round(time.Millisecond))
		fmt.Printf("   Hosts: %d\n", len(result.Hosts))

		if verbose {
			// Show host details
			for _, hostResult := range result.Hosts {
				hostStatus := "✅"
				if hostResult.Failed {
					hostStatus = "❌"
				}
				fmt.Printf("   %s Host: %s\n", hostStatus, hostResult.Host)
				fmt.Printf("      Tasks: %d\n", len(hostResult.Tasks))

				// Count task statuses
				changed := 0
				failed := 0
				skipped := 0
				for _, task := range hostResult.Tasks {
					if task.Changed {
						changed++
					}
					if task.Failed {
						failed++
					}
					if task.Skipped {
						skipped++
					}
				}

				fmt.Printf("      Changed: %d, Failed: %d, Skipped: %d\n", changed, failed, skipped)
			}
		}

		fmt.Println()
	}

	// Summary
	totalPlays := len(results)
	totalHosts := 0
	totalTasks := 0
	for _, result := range results {
		totalHosts += len(result.Hosts)
		for _, hostResult := range result.Hosts {
			totalTasks += len(hostResult.Tasks)
		}
	}

	fmt.Println("───────────────────────────────────────────────────────────")
	fmt.Printf("Total: %d plays, %d hosts, %d tasks\n", totalPlays, totalHosts, totalTasks)
	fmt.Println("═══════════════════════════════════════════════════════════")

	return nil
}

func outputResourceDetails(result *types.PlayResult, host *types.HostResult, stateData *types.State) error {
	fmt.Println("═══════════════════════════════════════════════════════════")

	if host != nil {
		// Show host details
		fmt.Printf("Host: %s\n", host.Host)
		status := "Success"
		if host.Failed {
			status = "Failed"
		}
		fmt.Printf("Status: %s\n", status)
		fmt.Printf("Tasks: %d\n", len(host.Tasks))
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Println()

		// Show tasks
		for i, task := range host.Tasks {
			taskStatus := "✅"
			if task.Failed {
				taskStatus = "❌"
			} else if task.Skipped {
				taskStatus = "⏭️ "
			} else if task.Changed {
				taskStatus = "🔄"
			}

			fmt.Printf("%s Task %d: %s\n", taskStatus, i+1, task.TaskName)
			fmt.Printf("   Module: %s\n", task.Module)
			fmt.Printf("   Duration: %v\n", task.Duration.Round(time.Millisecond))

			if task.Changed {
				fmt.Println("   Changed: Yes")
			}
			if task.Failed {
				fmt.Printf("   Error: %s\n", task.Error)
			}
			if task.Skipped {
				fmt.Println("   Skipped: Yes")
			}

			// Show output if available
			if len(task.Output) > 0 {
				// Try to get a string representation
				if msg, ok := task.Output["message"].(string); ok && len(msg) < 200 {
					fmt.Printf("   Output: %s\n", msg)
				}
			}

			fmt.Println()
		}
	} else {
		// Show play details
		playName := result.Name
		if playName == "" {
			playName = result.PlayName
		}
		fmt.Printf("Play: %s\n", playName)
		status := "Success"
		if !result.Success {
			status = "Failed"
		}
		fmt.Printf("Status: %s\n", status)
		fmt.Printf("Start: %s\n", result.StartTime.Format(time.RFC3339))
		fmt.Printf("End: %s\n", result.EndTime.Format(time.RFC3339))
		fmt.Printf("Duration: %v\n", result.EndTime.Sub(result.StartTime).Round(time.Millisecond))
		fmt.Printf("Hosts: %d\n", len(result.Hosts))
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Println()

		// Show hosts
		for _, hostResult := range result.Hosts {
			hostStatus := "✅"
			if hostResult.Failed {
				hostStatus = "❌"
			}
			fmt.Printf("%s Host: %s\n", hostStatus, hostResult.Host)
			fmt.Printf("   Tasks: %d\n", len(hostResult.Tasks))

			// Count statuses
			changed := 0
			failed := 0
			skipped := 0
			for _, task := range hostResult.Tasks {
				if task.Changed {
					changed++
				}
				if task.Failed {
					failed++
				}
				if task.Skipped {
					skipped++
				}
			}

			fmt.Printf("   Changed: %d, Failed: %d, Skipped: %d\n", changed, failed, skipped)
			fmt.Println()
		}
	}

	return nil
}
