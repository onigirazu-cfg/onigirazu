package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/execution"
	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

// NewShowExecutionCommand creates the show-execution command
func NewShowExecutionCommand() *cobra.Command {
	var (
		verbose bool
		debug   bool
	)

	cmd := &cobra.Command{
		Use:   "show-execution [execution-id]",
		Short: "Display a cached execution result",
		Long: `Display a cached execution result by ID or show the latest execution.

This command retrieves stored execution results without re-running the playbook.
You can view results in different verbosity levels.

Examples:
  # Show latest execution in normal mode
  onigirazu show-execution

  # Show specific execution in verbose mode
  onigirazu show-execution exec-1234567890 --verbose

  # Show execution in debug mode (full JSON)
  onigirazu show-execution exec-1234567890 --debug`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create cache manager
			cacheManager, err := execution.NewCacheManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to initialize cache: %v\n", err)
				return err
			}

			var result *execution.ExecutionResult

			// Load execution
			if len(args) > 0 {
				// Load specific execution
				executionID := args[0]
				result, err = cacheManager.Load(executionID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: Failed to load execution '%s': %v\n", executionID, err)
					return err
				}
			} else {
				// Load latest
				result, err = cacheManager.LoadLatest()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					return err
				}
			}

			// Determine display mode
			displayMode := execution.DisplayNormal
			if debug {
				displayMode = execution.DisplayDebug
			} else if verbose {
				displayMode = execution.DisplayVerbose
			}

			// Display result
			useColors := !noColor && utils.IsColorTerminal()
			displayer := execution.NewDisplayer(displayMode, useColors)
			displayer.DisplayExecution(result)

			return nil
		},
	}

	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show verbose output with error details")
	cmd.Flags().BoolVar(&debug, "debug", false, "Show debug output with complete JSON data")

	return cmd
}

// NewShowLastExecutionCommand creates the show-last-execution command
func NewShowLastExecutionCommand() *cobra.Command {
	var (
		verbose bool
		debug   bool
	)

	cmd := &cobra.Command{
		Use:   "show-last-execution",
		Short: "Display the most recent cached execution",
		Long: `Display the most recent execution result without re-running the playbook.

This is a convenience command equivalent to: onigirazu show-execution

Examples:
  # Show latest execution in normal mode
  onigirazu show-last-execution

  # Show latest in verbose mode
  onigirazu show-last-execution --verbose

  # Show latest in debug mode
  onigirazu show-last-execution --debug`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create cache manager
			cacheManager, err := execution.NewCacheManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to initialize cache: %v\n", err)
				return err
			}

			// Load latest
			result, err := cacheManager.LoadLatest()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}

			// Determine display mode
			displayMode := execution.DisplayNormal
			if debug {
				displayMode = execution.DisplayDebug
			} else if verbose {
				displayMode = execution.DisplayVerbose
			}

			// Display result
			useColors := !noColor && utils.IsColorTerminal()
			displayer := execution.NewDisplayer(displayMode, useColors)
			displayer.DisplayExecution(result)

			return nil
		},
	}

	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show verbose output with error details")
	cmd.Flags().BoolVar(&debug, "debug", false, "Show debug output with complete JSON data")

	return cmd
}

// NewListExecutionsCommand creates the list-executions command
func NewListExecutionsCommand() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list-executions",
		Short: "List recent cached executions",
		Long: `List recently cached execution results.

Examples:
  # List last 10 executions
  onigirazu list-executions

  # List last 20 executions
  onigirazu list-executions --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create cache manager
			cacheManager, err := execution.NewCacheManager()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to initialize cache: %v\n", err)
				return err
			}

			// List executions
			results, err := cacheManager.ListExecutions(limit)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to list executions: %v\n", err)
				return err
			}

			if len(results) == 0 {
				fmt.Println("No cached executions found")
				return nil
			}

			// Display as table
			fmt.Println()
			fmt.Printf("%-20s | %-30s | %-15s | %-10s | %-20s\n",
				"Execution ID", "Playbook", "Status", "Result", "Started")
			fmt.Println(strings.Repeat("─", 100))

			for _, result := range results {
				statusStr := fmt.Sprintf("✓ %d / ✗ %d", result.TotalSuccess, result.TotalFailed)
				fmt.Printf("%-20s | %-30s | %-15s | %-10s | %-20s\n",
					result.ExecutionID,
					result.PlaybookName,
					result.Status,
					statusStr,
					result.StartTime.Format("2006-01-02 15:04:05"),
				)
			}
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Number of executions to list")

	return cmd
}
