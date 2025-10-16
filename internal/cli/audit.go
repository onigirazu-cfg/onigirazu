package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/audit"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit playbook executions and view reports",
	Long: `Manage and view audit records of playbook executions.

This command allows you to:
- List execution history
- View detailed execution reports
- Export audit data in various formats
- View statistics and analytics
- Clear old records`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List execution records",
	Long:  `List audit records of playbook executions with optional filtering.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		reportFormat, _ := cmd.Flags().GetString("format")
		limit, _ := cmd.Flags().GetInt("limit")
		offset, _ := cmd.Flags().GetInt("offset")
		playbookFilter, _ := cmd.Flags().GetString("playbook")
		statusFilter, _ := cmd.Flags().GetString("status")
		hostFilter, _ := cmd.Flags().GetString("host")
		sortBy, _ := cmd.Flags().GetString("sort")
		sortOrder, _ := cmd.Flags().GetString("sort-order")

		logLevel := "warn"
		if verbose {
			logLevel = "info"
		}
		logFormat := "text"
		log, err := logger.NewEnhancedLogger(logLevel, logFormat, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}

		auditPath := getAuditPath()
		storage, err := audit.NewStorage(auditPath, log)
		if err != nil {
			return fmt.Errorf("failed to initialize audit storage: %w", err)
		}
		defer storage.Close()

		// Apply filters
		filterOpts := audit.FilterOptions{
			PlaybookPath: playbookFilter,
			Status:       audit.ExecutionStatus(statusFilter),
			HostFilter:   hostFilter,
			Limit:        limit,
			Offset:       offset,
			SortBy:       sortBy,
			SortOrder:    sortOrder,
		}

		records, err := storage.ListRecords(filterOpts)
		if err != nil {
			return fmt.Errorf("failed to list records: %w", err)
		}

		if len(records) == 0 {
			fmt.Println(utils.Colors.Warning("No execution records found."))
			return nil
		}

		reporter := audit.NewReporter(records)
		report, err := reporter.Generate(audit.FormatType(reportFormat))
		if err != nil {
			return fmt.Errorf("failed to generate report: %w", err)
		}

		output, _ := cmd.Flags().GetString("output")
		if output != "" {
			err := os.WriteFile(output, []byte(report), 0600)
			if err != nil {
				return fmt.Errorf("failed to write report to file: %w", err)
			}
			fmt.Printf("%s Report saved to: %s\n", utils.Colors.Success("✓"), output)
		} else {
			fmt.Print(report)
		}

		return nil
	},
}

var auditShowCmd = &cobra.Command{
	Use:   "show <execution-id>",
	Short: "Show detailed execution report",
	Long:  `Display detailed information about a specific execution.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		recordID := args[0]
		reportFormat, _ := cmd.Flags().GetString("format")

		logLevel := "warn"
		if verbose {
			logLevel = "info"
		}
		logFormat := "text"
		log, err := logger.NewEnhancedLogger(logLevel, logFormat, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}

		auditPath := getAuditPath()
		storage, err := audit.NewStorage(auditPath, log)
		if err != nil {
			return fmt.Errorf("failed to initialize audit storage: %w", err)
		}
		defer storage.Close()

		record, err := storage.LoadRecord(recordID)
		if err != nil {
			return fmt.Errorf("failed to load record: %w", err)
		}

		reporter := audit.NewReporter([]audit.ExecutionRecord{*record})
		report, err := reporter.GenerateDetailedReport(recordID, audit.FormatType(reportFormat))
		if err != nil {
			return fmt.Errorf("failed to generate detailed report: %w", err)
		}

		output, _ := cmd.Flags().GetString("output")
		if output != "" {
			err := os.WriteFile(output, []byte(report), 0600)
			if err != nil {
				return fmt.Errorf("failed to write report to file: %w", err)
			}
			fmt.Printf("%s Report saved to: %s\n", utils.Colors.Success("✓"), output)
		} else {
			fmt.Print(report)
		}

		return nil
	},
}

var auditStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show audit statistics",
	Long:  `Display statistics about execution history.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logLevel := "warn"
		if verbose {
			logLevel = "info"
		}
		format := "text"
		log, err := logger.NewEnhancedLogger(logLevel, format, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}

		auditPath := getAuditPath()
		storage, err := audit.NewStorage(auditPath, log)
		if err != nil {
			return fmt.Errorf("failed to initialize audit storage: %w", err)
		}
		defer storage.Close()

		stats, err := storage.GetStatistics(audit.FilterOptions{})
		if err != nil {
			return fmt.Errorf("failed to get statistics: %w", err)
		}

		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("AUDIT STATISTICS")
		fmt.Println(strings.Repeat("=", 60))

		fmt.Printf("\n%-30s %v\n", "Total Executions:", stats.TotalExecutions)
		fmt.Printf("%-30s %v\n", "Successful Runs:", utils.Colors.Success(fmt.Sprintf("%v", stats.SuccessfulRuns)))
		fmt.Printf("%-30s %v\n", "Failed Runs:", utils.Colors.Error(fmt.Sprintf("%v", stats.FailedRuns)))
		fmt.Printf("%-30s %v\n", "Success Rate:", fmt.Sprintf("%.1f%%", float64(stats.SuccessfulRuns)*100/float64(stats.TotalExecutions)))

		fmt.Printf("\n%-30s %v\n", "Total Tasks Executed:", stats.TotalTasks)
		fmt.Printf("%-30s %v\n", "Failed Tasks:", utils.Colors.Error(fmt.Sprintf("%v", stats.TotalFailedTasks)))
		fmt.Printf("%-30s %.2f hours\n", "Average Duration:", stats.AvgDuration/3600)

		if !stats.FirstExecution.IsZero() {
			fmt.Printf("\n%-30s %s\n", "First Execution:", stats.FirstExecution.Format(time.RFC3339))
		}
		if !stats.LastExecution.IsZero() {
			fmt.Printf("%-30s %s\n", "Last Execution:", stats.LastExecution.Format(time.RFC3339))
		}

		if len(stats.MostUsedModules) > 0 {
			fmt.Println("\nMost Used Modules:")
			for i, module := range stats.MostUsedModules {
				fmt.Printf("  %d. %s\n", i+1, module)
			}
		}

		if len(stats.CommonErrors) > 0 {
			fmt.Println("\nCommon Errors:")
			for i, errMsg := range stats.CommonErrors {
				fmt.Printf("  %d. %s\n", i+1, errMsg)
			}
		}

		fmt.Println("\n" + strings.Repeat("=", 60))
		return nil
	},
}

var auditHostCmd = &cobra.Command{
	Use:   "host <hostname>",
	Short: "Show statistics for a specific host",
	Long:  `Display execution statistics for a specific host.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostname := args[0]

		logLevel := "warn"
		if verbose {
			logLevel = "info"
		}
		format := "text"
		log, err := logger.NewEnhancedLogger(logLevel, format, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}

		auditPath := getAuditPath()
		storage, err := audit.NewStorage(auditPath, log)
		if err != nil {
			return fmt.Errorf("failed to initialize audit storage: %w", err)
		}
		defer storage.Close()

		stats, err := storage.GetHostStatistics(hostname)
		if err != nil {
			return fmt.Errorf("failed to get host statistics: %w", err)
		}

		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Printf("HOST STATISTICS: %s\n", hostname)
		fmt.Println(strings.Repeat("=", 60))

		fmt.Printf("\n%-30s %v\n", "Total Executions:", stats.TotalExecutions)
		fmt.Printf("%-30s %v\n", "Successful Runs:", utils.Colors.Success(fmt.Sprintf("%v", stats.SuccessfulRuns)))
		fmt.Printf("%-30s %v\n", "Failed Runs:", utils.Colors.Error(fmt.Sprintf("%v", stats.FailedRuns)))

		if stats.TotalExecutions > 0 {
			successRate := float64(stats.SuccessfulRuns) * 100 / float64(stats.TotalExecutions)
			fmt.Printf("%-30s %.1f%%\n", "Success Rate:", successRate)
		}

		fmt.Printf("\n%-30s %v\n", "Total Tasks:", stats.TotalTasks)
		fmt.Printf("%-30s %v\n", "Failed Tasks:", utils.Colors.Error(fmt.Sprintf("%v", stats.TotalFailedTasks)))

		if stats.AvgTaskDuration > 0 {
			fmt.Printf("%-30s %.2f seconds\n", "Avg Task Duration:", stats.AvgTaskDuration)
		}

		if !stats.LastExecution.IsZero() {
			fmt.Printf("%-30s %s\n", "Last Execution:", stats.LastExecution.Format(time.RFC3339))
		}

		if len(stats.MostCommonErrors) > 0 {
			fmt.Println("\nMost Common Errors:")
			for i, errMsg := range stats.MostCommonErrors {
				fmt.Printf("  %d. %s\n", i+1, errMsg)
			}
		}

		fmt.Println("\n" + strings.Repeat("=", 60))
		return nil
	},
}

var auditClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear old audit records",
	Long:  `Remove audit records older than the specified retention period.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		days, _ := cmd.Flags().GetInt("days")
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("This will delete audit records older than %d days. Continue? (y/n): ", days)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(response) != "y" {
				fmt.Println("Canceled.")
				return nil
			}
		}

		logLevel := "warn"
		if verbose {
			logLevel = "info"
		}
		format := "text"
		log, err := logger.NewEnhancedLogger(logLevel, format, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}

		auditPath := getAuditPath()
		storage, err := audit.NewStorage(auditPath, log)
		if err != nil {
			return fmt.Errorf("failed to initialize audit storage: %w", err)
		}
		defer storage.Close()

		deleted, err := storage.DeleteOldRecords(days)
		if err != nil {
			return fmt.Errorf("failed to delete records: %w", err)
		}

		fmt.Printf("%s Deleted %d old audit records (older than %d days)\n",
			utils.Colors.Success("✓"), deleted, days)
		return nil
	},
}

var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit records",
	Long:  `Export audit records to a file in the specified format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		reportFormat, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")

		if output == "" {
			return fmt.Errorf("output file path is required (use --output)")
		}

		logLevel := "warn"
		if verbose {
			logLevel = "info"
		}
		logFormat := "text"
		log, err := logger.NewEnhancedLogger(logLevel, logFormat, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}

		auditPath := getAuditPath()
		storage, err := audit.NewStorage(auditPath, log)
		if err != nil {
			return fmt.Errorf("failed to initialize audit storage: %w", err)
		}
		defer storage.Close()

		records, err := storage.ListRecords(audit.FilterOptions{})
		if err != nil {
			return fmt.Errorf("failed to list records: %w", err)
		}

		if len(records) == 0 {
			fmt.Println(utils.Colors.Warning("No records to export."))
			return nil
		}

		reporter := audit.NewReporter(records)
		report, err := reporter.Generate(audit.FormatType(reportFormat))
		if err != nil {
			return fmt.Errorf("failed to generate report: %w", err)
		}

		err = os.WriteFile(output, []byte(report), 0600)
		if err != nil {
			return fmt.Errorf("failed to write export file: %w", err)
		}

		fmt.Printf("%s Exported %d records to: %s\n",
			utils.Colors.Success("✓"), len(records), output)
		return nil
	},
}

// Helper function to get audit storage path
func getAuditPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".onigirazu", "audit")
}

// Initialize audit subcommands
func init() {
	// Main audit command
	auditCmd.AddCommand(auditListCmd)
	auditCmd.AddCommand(auditShowCmd)
	auditCmd.AddCommand(auditStatsCmd)
	auditCmd.AddCommand(auditHostCmd)
	auditCmd.AddCommand(auditClearCmd)
	auditCmd.AddCommand(auditExportCmd)

	// List flags
	auditListCmd.Flags().StringP("format", "f", "text", "Output format: text, json, csv, html, markdown")
	auditListCmd.Flags().IntP("limit", "l", 20, "Maximum number of records to display")
	auditListCmd.Flags().Int("offset", 0, "Offset for pagination")
	auditListCmd.Flags().String("playbook", "", "Filter by playbook path")
	auditListCmd.Flags().String("status", "", "Filter by status (success, failure)")
	auditListCmd.Flags().String("host", "", "Filter by affected host")
	auditListCmd.Flags().String("sort", "time", "Sort by: time, duration, status")
	auditListCmd.Flags().String("sort-order", "desc", "Sort order: asc, desc")
	auditListCmd.Flags().StringP("output", "o", "", "Save report to file")

	// Show flags
	auditShowCmd.Flags().StringP("format", "f", "markdown", "Output format: text, json, markdown")
	auditShowCmd.Flags().StringP("output", "o", "", "Save report to file")

	// Export flags
	auditExportCmd.Flags().StringP("format", "f", "json", "Output format: json, csv, html, markdown")
	auditExportCmd.Flags().StringP("output", "o", "", "Output file path (required)")
	auditExportCmd.MarkFlagRequired("output")

	// Clear flags
	auditClearCmd.Flags().IntP("days", "d", 30, "Delete records older than this many days")
	auditClearCmd.Flags().BoolP("force", "f", false, "Don't ask for confirmation")
}
