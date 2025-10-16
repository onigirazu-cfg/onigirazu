package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/drift"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/internal/rollback"
)

var (
	driftSnapshotID string
	driftAutoFix    bool
	driftDryRun     bool
	driftFormat     string
	driftOutput     string
	driftList       bool
	driftInfo       bool
	driftReportID   string
	driftParallel   int
)

func newDriftCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drift",
		Short: "Detect and fix configuration drift",
		Long: `Detect configuration drift by comparing current system state with snapshots.

Drift detection helps identify unauthorized or unexpected changes to your infrastructure
by comparing the current state with the expected state captured in snapshots.

Features:
  • Detect drift from snapshots
  • Auto-fix detected drift
  • Generate reports in multiple formats (text, JSON, HTML)
  • List all drift reports
  • View detailed drift information

Examples:
  # Detect drift from a snapshot
  onigirazu drift detect --snapshot <snapshot-id>

  # Detect and auto-fix drift
  onigirazu drift detect --snapshot <snapshot-id> --auto-fix

  # Dry-run (preview fixes without applying)
  onigirazu drift detect --snapshot <snapshot-id> --auto-fix --dry-run

  # Generate HTML report
  onigirazu drift detect --snapshot <snapshot-id> --format html --output report.html

  # List all drift reports
  onigirazu drift --list

  # Show drift report details
  onigirazu drift --info --report <report-id>
`,
		RunE: runDrift,
	}

	cmd.Flags().StringVar(&driftSnapshotID, "snapshot", "", "Snapshot ID to compare against")
	cmd.Flags().BoolVar(&driftAutoFix, "auto-fix", false, "Automatically fix detected drift")
	cmd.Flags().BoolVar(&driftDryRun, "dry-run", false, "Preview fixes without applying them")
	cmd.Flags().StringVar(&driftFormat, "format", "text", "Report format (text, json, html)")
	cmd.Flags().StringVar(&driftOutput, "output", "", "Output file for report (default: stdout)")
	cmd.Flags().BoolVar(&driftList, "list", false, "List all drift reports")
	cmd.Flags().BoolVar(&driftInfo, "info", false, "Show detailed drift report information")
	cmd.Flags().StringVar(&driftReportID, "report", "", "Drift report ID")
	cmd.Flags().IntVarP(&driftParallel, "parallel", "f", 10, "Number of parallel executions")

	return cmd
}

func runDrift(cmd *cobra.Command, args []string) error {
	// Get directories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	snapshotDir := filepath.Join(homeDir, ".onigirazu", "snapshots")
	reportDir := filepath.Join(homeDir, ".onigirazu", "drift-reports")

	// Create logger
	log := logger.New(showDebug)

	// Create snapshot manager
	sm := rollback.NewSnapshotManager(snapshotDir)

	// Create module registry (built-in modules are registered automatically)
	registry := modules.NewRegistry()

	// Create drift config
	config := &drift.DriftConfig{
		Enabled:        true,
		AutoFix:        driftAutoFix,
		DryRun:         driftDryRun,
		MaxConcurrency: driftParallel,
		Resources: []drift.DriftType{
			drift.DriftTypeFile,
			drift.DriftTypePackage,
			drift.DriftTypeService,
			drift.DriftTypeUser,
			drift.DriftTypeGroup,
		},
		AutoFixSeverity: []drift.DriftSeverity{
			drift.SeverityCritical,
			drift.SeverityHigh,
			drift.SeverityMedium,
		},
		ReportFormat: driftFormat,
		ReportPath:   reportDir,
	}

	// Create drift detector
	detector := drift.NewDetector(sm, registry, log, config)

	// Handle different operations
	if driftList {
		return listDriftReports(detector)
	}

	if driftInfo {
		if driftReportID == "" {
			return fmt.Errorf("--report is required with --info")
		}
		return showDriftReportInfo(detector, driftReportID)
	}

	// Detect drift
	if driftSnapshotID == "" {
		return fmt.Errorf("--snapshot is required for drift detection")
	}

	return detectDrift(detector, registry, log, config)
}

func detectDrift(detector *drift.Detector, registry *modules.Registry, log *logger.Logger, config *drift.DriftConfig) error {
	ctx := context.Background()

	log.Info("Detecting drift from snapshot: %s", driftSnapshotID)

	// Detect drift
	report, err := detector.DetectDrift(ctx, driftSnapshotID)
	if err != nil {
		return fmt.Errorf("drift detection failed: %w", err)
	}

	// Auto-fix if requested
	var fixResult *drift.FixResult
	if driftAutoFix && report.TotalDrifts > 0 {
		log.Info("Auto-fix enabled, attempting to fix drift...")

		fixer := drift.NewFixer(registry, log, config)
		fixResult, err = fixer.FixDrift(ctx, report)
		if err != nil {
			log.Warn("Auto-fix failed: %v", err)
		} else {
			// Update report with fix results
			report.FixedDrifts = fixResult.FixedCount
			report.FailedFixes = fixResult.FailedCount
		}
	}

	// Generate report
	reporter := drift.NewReporter(config)
	reportContent, err := reporter.GenerateReport(report, driftFormat)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Output report
	if driftOutput != "" {
		if err := os.WriteFile(driftOutput, []byte(reportContent), 0600); err != nil {
			return fmt.Errorf("failed to write report to file: %w", err)
		}
		log.Info("Report saved to: %s", driftOutput)
	} else {
		fmt.Println(reportContent)
	}

	// Show fix results if auto-fix was enabled
	if fixResult != nil {
		fmt.Println()
		printFixResults(fixResult)
	}

	// Exit with error code if drift detected
	if report.TotalDrifts > 0 {
		if report.CriticalDrifts > 0 || report.HighDrifts > 0 {
			os.Exit(2) // Critical/High drift detected
		}
		os.Exit(1) // Drift detected
	}

	return nil
}

func listDriftReports(detector *drift.Detector) error {
	reports, err := detector.ListReports()
	if err != nil {
		return fmt.Errorf("failed to list reports: %w", err)
	}

	if len(reports) == 0 {
		fmt.Println("No drift reports found")
		return nil
	}

	// Print header
	fmt.Printf("%-20s %-25s %-15s %-10s %-10s %-10s\n",
		"REPORT ID", "TIMESTAMP", "SNAPSHOT ID", "TOTAL", "CRITICAL", "HIGH")
	fmt.Println(strings.Repeat("-", 100))

	// Print reports
	for _, report := range reports {
		fmt.Printf("%-20s %-25s %-15s %-10d %-10d %-10d\n",
			report.ID,
			report.Timestamp.Format("2006-01-02 15:04:05"),
			report.SnapshotID,
			report.TotalDrifts,
			report.CriticalDrifts,
			report.HighDrifts,
		)
	}

	fmt.Printf("\nTotal reports: %d\n", len(reports))
	return nil
}

func showDriftReportInfo(detector *drift.Detector, reportID string) error {
	report, err := detector.LoadReport(reportID)
	if err != nil {
		return fmt.Errorf("failed to load report: %w", err)
	}

	// Generate text report
	config := drift.DefaultConfig()
	reporter := drift.NewReporter(config)
	reportContent, err := reporter.GenerateReport(report, "text")
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	fmt.Println(reportContent)
	return nil
}

func printFixResults(result *drift.FixResult) {
	fmt.Println("=== Auto-Fix Results ===")
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Total Processed: %d\n", result.TotalProcessed)
	fmt.Printf("Fixed: %d\n", result.FixedCount)
	fmt.Printf("Failed: %d\n", result.FailedCount)
	fmt.Printf("Skipped: %d\n", len(result.SkippedItems))
	fmt.Println()

	if len(result.FixedItems) > 0 {
		fmt.Println("Fixed Items:")
		for _, item := range result.FixedItems {
			fmt.Printf("  ✓ %s: %s on %s\n", item.DriftItem.Type, item.DriftItem.Resource, item.DriftItem.Host)
		}
		fmt.Println()
	}

	if len(result.FailedItems) > 0 {
		fmt.Println("Failed Items:")
		for _, item := range result.FailedItems {
			fmt.Printf("  ✗ %s: %s on %s - %s\n",
				item.DriftItem.Type, item.DriftItem.Resource, item.DriftItem.Host, item.Error)
		}
		fmt.Println()
	}

	if len(result.SkippedItems) > 0 {
		fmt.Println("Skipped Items:")
		for _, item := range result.SkippedItems {
			fmt.Printf("  ⊘ %s: %s on %s - %s\n",
				item.DriftItem.Type, item.DriftItem.Resource, item.DriftItem.Host, item.Reason)
		}
		fmt.Println()
	}
}
