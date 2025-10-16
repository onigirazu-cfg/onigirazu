package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/internal/rollback"
)

var (
	rollbackSnapshotID string
	rollbackDryRun     bool
	rollbackList       bool
	rollbackInfo       bool
	rollbackCleanup    bool
	rollbackMaxAge     string
	rollbackParallel   int
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback changes to a previous snapshot",
	Long: `Rollback changes made by a playbook execution to a previous snapshot.

Snapshots are automatically created before playbook execution and can be used
to restore the system to its previous state if something goes wrong.

Examples:
  # List available snapshots
  onigirazu rollback --list

  # Show information about a specific snapshot
  onigirazu rollback --info --snapshot <snapshot-id>

  # Preview rollback changes (dry-run)
  onigirazu rollback --dry-run --snapshot <snapshot-id>

  # Perform rollback
  onigirazu rollback --snapshot <snapshot-id>

  # Cleanup old snapshots (older than 30 days)
  onigirazu rollback --cleanup --max-age 30d
`,
	RunE: runRollback,
}

func init() {
	rollbackCmd.Flags().StringVar(&rollbackSnapshotID, "snapshot", "", "Snapshot ID to rollback to")
	rollbackCmd.Flags().BoolVar(&rollbackDryRun, "dry-run", false, "Show what would be rolled back without making changes")
	rollbackCmd.Flags().BoolVar(&rollbackList, "list", false, "List available snapshots")
	rollbackCmd.Flags().BoolVar(&rollbackInfo, "info", false, "Show detailed information about a snapshot")
	rollbackCmd.Flags().BoolVar(&rollbackCleanup, "cleanup", false, "Cleanup old snapshots")
	rollbackCmd.Flags().StringVar(&rollbackMaxAge, "max-age", "30d", "Maximum age for snapshots (e.g., 7d, 24h)")
	rollbackCmd.Flags().IntVarP(&rollbackParallel, "parallel", "f", 5, "Number of parallel executions")
}

func runRollback(cmd *cobra.Command, args []string) error {
	// Get snapshot directory from config or use default
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	snapshotDir := filepath.Join(homeDir, ".onigirazu", "snapshots")

	// Create logger
	log := logger.New(showDebug)

	// Create snapshot manager
	sm := rollback.NewSnapshotManager(snapshotDir)

	// Create module registry (built-in modules are registered automatically)
	registry := modules.NewRegistry()

	// Create rollback executor
	executor := rollback.NewRollbackExecutor(sm, registry, log).WithMaxConcurrency(rollbackParallel)

	// Handle different operations
	if rollbackList {
		return listSnapshots(executor)
	}

	if rollbackInfo {
		if rollbackSnapshotID == "" {
			return fmt.Errorf("--snapshot is required with --info")
		}
		return showSnapshotInfo(executor, rollbackSnapshotID)
	}

	if rollbackCleanup {
		return cleanupSnapshots(sm, rollbackMaxAge)
	}

	if rollbackSnapshotID == "" {
		return fmt.Errorf("--snapshot is required (use --list to see available snapshots)")
	}

	if rollbackDryRun {
		return dryRunRollback(executor, rollbackSnapshotID)
	}

	return performRollback(executor, rollbackSnapshotID)
}

func listSnapshots(executor *rollback.RollbackExecutor) error {
	snapshots, err := executor.ListSnapshots()
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SNAPSHOT ID\tTIMESTAMP\tPLAYBOOK\tRESOURCES\tREVERSIBLE\tDESCRIPTION")
	fmt.Fprintln(w, "-----------\t---------\t--------\t---------\t----------\t-----------")

	for _, snapshot := range snapshots {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%s\n",
			snapshot.ID,
			snapshot.Timestamp.Format("2006-01-02 15:04:05"),
			snapshot.PlaybookID,
			snapshot.TotalResources,
			snapshot.ReversibleCount,
			snapshot.Description,
		)
	}

	_ = w.Flush()
	return nil
}

func showSnapshotInfo(executor *rollback.RollbackExecutor, snapshotID string) error {
	info, err := executor.GetSnapshotInfo(snapshotID)
	if err != nil {
		return fmt.Errorf("failed to get snapshot info: %w", err)
	}

	fmt.Printf("Snapshot ID: %s\n", info.ID)
	fmt.Printf("Timestamp: %s\n", info.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Playbook: %s\n", info.PlaybookID)
	fmt.Printf("Description: %s\n", info.Description)
	fmt.Printf("Total Resources: %d\n", info.TotalResources)
	fmt.Printf("Reversible Resources: %d\n", info.ReversibleCount)
	fmt.Println()

	if len(info.ResourcesByType) > 0 {
		fmt.Println("Resources by Type:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for resType, count := range info.ResourcesByType {
			fmt.Fprintf(w, "  %s:\t%d\n", resType, count)
		}
		_ = w.Flush()
		fmt.Println()
	}

	if len(info.ResourcesByHost) > 0 {
		fmt.Println("Resources by Host:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for host, count := range info.ResourcesByHost {
			fmt.Fprintf(w, "  %s:\t%d\n", host, count)
		}
		_ = w.Flush()
	}

	return nil
}

func dryRunRollback(executor *rollback.RollbackExecutor, snapshotID string) error {
	fmt.Printf("Planning rollback to snapshot: %s\n\n", snapshotID)

	plan, err := executor.DryRunRollback(context.Background(), snapshotID)
	if err != nil {
		return fmt.Errorf("failed to plan rollback: %w", err)
	}

	fmt.Printf("Snapshot: %s\n", plan.SnapshotID)
	fmt.Printf("Created: %s\n", plan.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Description: %s\n", plan.Description)
	fmt.Printf("Total Operations: %d\n", plan.TotalOperations)
	fmt.Printf("Reversible Operations: %d\n", plan.ReversibleOperations)
	fmt.Println()

	if len(plan.Operations) > 0 {
		fmt.Println("Planned Operations:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ORDER\tHOST\tRESOURCE\tMODULE\tREVERSIBLE\tDETAILS")
		fmt.Fprintln(w, "-----\t----\t--------\t------\t----------\t-------")

		for _, op := range plan.Operations {
			reversible := "No"
			if op.Reversible {
				reversible = "Yes"
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
				op.Order,
				op.Host,
				op.Resource,
				op.Module,
				reversible,
				op.Details,
			)
		}
		_ = w.Flush()
	}

	fmt.Println()
	fmt.Println("This is a dry-run. No changes will be made.")
	fmt.Printf("To perform the rollback, run: onigirazu rollback --snapshot %s\n", snapshotID)

	return nil
}

func performRollback(executor *rollback.RollbackExecutor, snapshotID string) error {
	fmt.Printf("Rolling back to snapshot: %s\n", snapshotID)
	fmt.Println("This will revert changes made by the playbook execution.")
	fmt.Println()

	result, err := executor.Rollback(context.Background(), snapshotID)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Println()
	fmt.Println("Rollback Summary:")
	fmt.Printf("  Duration: %s\n", result.Duration)
	fmt.Printf("  Resources Rolled Back: %d\n", result.ResourcesRolled)
	fmt.Printf("  Resources Failed: %d\n", result.ResourcesFailed)
	fmt.Println()

	if len(result.FailedResources) > 0 {
		fmt.Println("Failed Resources:")
		for _, failed := range result.FailedResources {
			fmt.Printf("  - %s\n", failed)
		}
		fmt.Println()
	}

	if result.Success {
		fmt.Println("✓ Rollback completed successfully")
		return nil
	} else {
		fmt.Println("✗ Rollback completed with errors")
		return fmt.Errorf("rollback completed with %d errors", result.ResourcesFailed)
	}
}

func cleanupSnapshots(sm *rollback.SnapshotManager, maxAgeStr string) error {
	// Parse max age duration
	maxAge, err := time.ParseDuration(maxAgeStr)
	if err != nil {
		return fmt.Errorf("invalid max-age format: %w (use format like '7d', '24h', '30d')", err)
	}

	fmt.Printf("Cleaning up snapshots older than %s...\n", maxAge)

	if err := sm.CleanupOldSnapshots(maxAge); err != nil {
		return fmt.Errorf("failed to cleanup snapshots: %w", err)
	}

	fmt.Println("✓ Cleanup completed successfully")
	return nil
}
