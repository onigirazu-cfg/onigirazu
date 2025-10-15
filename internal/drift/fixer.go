package drift

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Fixer automatically fixes detected drift
type Fixer struct {
	moduleRegistry interfaces.ModuleRegistry
	logger         interfaces.Logger
	config         *DriftConfig
}

// NewFixer creates a new drift fixer
func NewFixer(
	moduleRegistry interfaces.ModuleRegistry,
	logger interfaces.Logger,
	config *DriftConfig,
) *Fixer {
	return &Fixer{
		moduleRegistry: moduleRegistry,
		logger:         logger,
		config:         config,
	}
}

// FixDrift automatically fixes drift in a report
func (f *Fixer) FixDrift(ctx context.Context, report *DriftReport) (*FixResult, error) {
	f.logger.Info("Starting auto-fix for %d drifts", report.TotalDrifts)
	startTime := time.Now()

	result := &FixResult{
		ReportID:     report.ID,
		StartTime:    startTime,
		FixedItems:   []FixedItem{},
		FailedItems:  []FailedItem{},
		SkippedItems: []SkippedItem{},
	}

	// Filter items that can be auto-fixed
	fixableItems := f.filterFixableItems(report.Items)
	f.logger.Info("Found %d fixable items out of %d total drifts", len(fixableItems), len(report.Items))

	if len(fixableItems) == 0 {
		f.logger.Info("No fixable items found")
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Sort by fix order (services first, groups last)
	sort.Slice(fixableItems, func(i, j int) bool {
		return fixableItems[i].FixOperation.Order > fixableItems[j].FixOperation.Order
	})

	// Fix each item
	for _, item := range fixableItems {
		f.logger.Info("Fixing drift: %s on %s", item.Resource, item.Host)

		if f.config.DryRun {
			result.SkippedItems = append(result.SkippedItems, SkippedItem{
				DriftItem: item,
				Reason:    "Dry-run mode enabled",
			})
			continue
		}

		// Execute fix operation
		fixErr := f.executeFix(ctx, &item)
		if fixErr != nil {
			f.logger.Error("Failed to fix drift for %s: %v", item.Resource, fixErr)
			result.FailedItems = append(result.FailedItems, FailedItem{
				DriftItem: item,
				Error:     fixErr.Error(),
			})
			result.FailedCount++
			continue
		}

		// Mark as fixed
		item.Status = StatusFixed
		fixedAt := time.Now()
		item.FixedAt = &fixedAt

		result.FixedItems = append(result.FixedItems, FixedItem{
			DriftItem: item,
			FixedAt:   fixedAt,
		})
		result.FixedCount++

		f.logger.Info("Successfully fixed drift: %s", item.Resource)
	}

	result.EndTime = time.Now()
	result.Duration = time.Since(startTime)
	result.TotalProcessed = len(fixableItems)

	f.logger.Info("Auto-fix completed: %d fixed, %d failed, %d skipped in %v",
		result.FixedCount, result.FailedCount, len(result.SkippedItems), result.Duration)

	return result, nil
}

// executeFix executes a fix operation
func (f *Fixer) executeFix(ctx context.Context, item *DriftItem) error {
	if item.FixOperation == nil {
		return fmt.Errorf("no fix operation available")
	}

	// Get module
	module, err := f.moduleRegistry.Get(item.FixOperation.Module)
	if err != nil {
		return fmt.Errorf("module not found: %s", item.FixOperation.Module)
	}

	// Create host object
	host := types.Host{
		Name: item.Host,
	}

	// Validate arguments
	if err := module.Validate(item.FixOperation.Args); err != nil {
		return fmt.Errorf("invalid fix arguments: %w", err)
	}

	// Execute fix operation
	taskResult, err := module.Execute(ctx, host, item.FixOperation.Args)
	if err != nil {
		return fmt.Errorf("fix execution failed: %w", err)
	}

	if !taskResult.Success {
		return fmt.Errorf("fix operation failed: %s", taskResult.Error)
	}

	return nil
}

// filterFixableItems filters items that can be auto-fixed
func (f *Fixer) filterFixableItems(items []DriftItem) []DriftItem {
	fixable := []DriftItem{}

	for _, item := range items {
		// Skip if can't auto-fix
		if !item.CanAutoFix {
			continue
		}

		// Skip if no fix operation
		if item.FixOperation == nil {
			continue
		}

		// Check if severity is in auto-fix list
		if !f.shouldAutoFix(item.Severity) {
			continue
		}

		fixable = append(fixable, item)
	}

	return fixable
}

// shouldAutoFix checks if a severity level should be auto-fixed
func (f *Fixer) shouldAutoFix(severity DriftSeverity) bool {
	if !f.config.AutoFix {
		return false
	}

	if len(f.config.AutoFixSeverity) == 0 {
		return true
	}

	for _, s := range f.config.AutoFixSeverity {
		if s == severity {
			return true
		}
	}

	return false
}

// FixResult represents the result of auto-fix operation
type FixResult struct {
	ReportID       string        `json:"report_id"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time"`
	Duration       time.Duration `json:"duration"`
	TotalProcessed int           `json:"total_processed"`
	FixedCount     int           `json:"fixed_count"`
	FailedCount    int           `json:"failed_count"`
	FixedItems     []FixedItem   `json:"fixed_items"`
	FailedItems    []FailedItem  `json:"failed_items"`
	SkippedItems   []SkippedItem `json:"skipped_items"`
}

// FixedItem represents a successfully fixed drift item
type FixedItem struct {
	DriftItem DriftItem `json:"drift_item"`
	FixedAt   time.Time `json:"fixed_at"`
}

// FailedItem represents a failed fix attempt
type FailedItem struct {
	DriftItem DriftItem `json:"drift_item"`
	Error     string    `json:"error"`
}

// SkippedItem represents a skipped fix
type SkippedItem struct {
	DriftItem DriftItem `json:"drift_item"`
	Reason    string    `json:"reason"`
}
