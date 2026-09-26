package rollback

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// RollbackExecutor executes rollback operations
type RollbackExecutor struct {
	snapshotManager *SnapshotManager
	moduleRegistry  interfaces.ModuleRegistry
	logger          interfaces.Logger
	maxConcurrency  int
	// hosts resolves a host name of the snapshot to how to reach it
	hosts func(name string) (types.Host, error)
}

// WithHosts sets how host names are resolved (from the inventory)
func (re *RollbackExecutor) WithHosts(resolve func(name string) (types.Host, error)) *RollbackExecutor {
	re.hosts = resolve
	return re
}

// NewRollbackExecutor creates a new rollback executor
func NewRollbackExecutor(
	snapshotManager *SnapshotManager,
	moduleRegistry interfaces.ModuleRegistry,
	logger interfaces.Logger,
) *RollbackExecutor {
	return &RollbackExecutor{
		snapshotManager: snapshotManager,
		moduleRegistry:  moduleRegistry,
		logger:          logger,
		maxConcurrency:  5, // default value
	}
}

// WithMaxConcurrency sets the maximum concurrency for rollback operations
func (re *RollbackExecutor) WithMaxConcurrency(maxConcurrency int) *RollbackExecutor {
	re.maxConcurrency = maxConcurrency
	return re
}

// RollbackResult represents the result of a rollback operation
type RollbackResult struct {
	SnapshotID      string
	Success         bool
	Error           string
	ResourcesRolled int
	ResourcesFailed int
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	FailedResources []string
}

// Rollback performs a rollback to a specific snapshot
func (re *RollbackExecutor) Rollback(ctx context.Context, snapshotID string) (*RollbackResult, error) {
	re.logger.Info("Starting rollback to snapshot: %s", snapshotID)

	result := &RollbackResult{
		SnapshotID:      snapshotID,
		StartTime:       time.Now(),
		FailedResources: []string{},
	}

	// Load snapshot
	snapshot, err := re.snapshotManager.LoadSnapshot(snapshotID)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to load snapshot: %v", err)
		return result, err
	}

	re.logger.Info("Loaded snapshot with %d resources", len(snapshot.Resources))

	// Sort resources by rollback order (higher order first)
	sortedResources := make([]ResourceSnapshot, len(snapshot.Resources))
	copy(sortedResources, snapshot.Resources)
	sort.Slice(sortedResources, func(i, j int) bool {
		orderI := 50
		orderJ := 50
		if sortedResources[i].RollbackOp != nil {
			orderI = sortedResources[i].RollbackOp.Order
		}
		if sortedResources[j].RollbackOp != nil {
			orderJ = sortedResources[j].RollbackOp.Order
		}
		return orderI > orderJ
	})

	// Execute rollback operations
	for _, resource := range sortedResources {
		if !resource.Reversible || resource.RollbackOp == nil {
			re.logger.Warn("Skipping non-reversible resource: %s (%s)", resource.Identifier, resource.Type)
			continue
		}

		re.logger.Info("Rolling back: %s on %s", resource.Identifier, resource.Host)

		if err := re.executeRollbackOperation(ctx, &resource); err != nil {
			re.logger.Error("Failed to rollback resource %s: %v", resource.Identifier, err)
			result.ResourcesFailed++
			result.FailedResources = append(result.FailedResources,
				fmt.Sprintf("%s (%s): %v", resource.Identifier, resource.Host, err))

			// Continue with other resources even if one fails
			continue
		}

		result.ResourcesRolled++
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = result.ResourcesFailed == 0

	if result.Success {
		re.logger.Info("Rollback completed successfully: %d resources rolled back", result.ResourcesRolled)
	} else {
		re.logger.Error("Rollback completed with errors: %d succeeded, %d failed",
			result.ResourcesRolled, result.ResourcesFailed)
	}

	return result, nil
}

// executeRollbackOperation executes a single rollback operation
func (re *RollbackExecutor) executeRollbackOperation(ctx context.Context, resource *ResourceSnapshot) error {
	// Get the module
	module, err := re.moduleRegistry.Get(resource.RollbackOp.Module)
	if err != nil {
		return fmt.Errorf("failed to get module %s: %w", resource.RollbackOp.Module, err)
	}

	host := types.Host{Name: resource.Host}
	if re.hosts != nil {
		var err error
		if host, err = re.hosts(resource.Host); err != nil {
			return fmt.Errorf("host %s: %w", resource.Host, err)
		}
	}
	// the escalation the task had, for modules that take it from the host
	if become, _ := resource.RollbackOp.Args["_become"].(bool); become {
		host.Become = true
		host.BecomeUser, _ = resource.RollbackOp.Args["_become_user"].(string)
		host.BecomeMethod, _ = resource.RollbackOp.Args["_become_method"].(string)
	}
	// modules may add keys; the snapshot keeps its own copy
	args := make(map[string]interface{}, len(resource.RollbackOp.Args))
	for k, v := range resource.RollbackOp.Args {
		args[k] = v
	}

	// Validate arguments
	if err := module.Validate(args); err != nil {
		return fmt.Errorf("invalid rollback arguments: %w", err)
	}

	// Execute the rollback operation
	taskResult, err := module.Execute(ctx, host, args)
	if err != nil {
		return fmt.Errorf("rollback execution failed: %w", err)
	}

	if !taskResult.Success {
		return fmt.Errorf("rollback operation failed: %s", taskResult.Error)
	}

	return nil
}

// DryRunRollback performs a dry-run of a rollback operation
func (re *RollbackExecutor) DryRunRollback(ctx context.Context, snapshotID string) (*RollbackPlan, error) {
	re.logger.Info("Generating rollback plan for snapshot: %s", snapshotID)

	// Load snapshot
	snapshot, err := re.snapshotManager.LoadSnapshot(snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot: %w", err)
	}

	plan := &RollbackPlan{
		SnapshotID:  snapshotID,
		Timestamp:   snapshot.Timestamp,
		Description: snapshot.Description,
		Operations:  []PlannedOperation{},
	}

	// Sort resources by rollback order
	sortedResources := make([]ResourceSnapshot, len(snapshot.Resources))
	copy(sortedResources, snapshot.Resources)
	sort.Slice(sortedResources, func(i, j int) bool {
		orderI := 50
		orderJ := 50
		if sortedResources[i].RollbackOp != nil {
			orderI = sortedResources[i].RollbackOp.Order
		}
		if sortedResources[j].RollbackOp != nil {
			orderJ = sortedResources[j].RollbackOp.Order
		}
		return orderI > orderJ
	})

	// Generate planned operations
	for i, resource := range sortedResources {
		op := PlannedOperation{
			Order:      i + 1,
			Host:       resource.Host,
			Resource:   resource.Identifier,
			Module:     resource.Type,
			Action:     "rollback",
			Reversible: resource.Reversible,
		}

		if resource.RollbackOp != nil {
			op.Details = describeOperation(resource.RollbackOp)
		} else if reason, ok := resource.State["reason"].(string); ok {
			op.Details = "skipped: " + reason
		} else {
			op.Details = "skipped: not reversible"
		}

		plan.Operations = append(plan.Operations, op)
	}

	plan.TotalOperations = len(plan.Operations)
	plan.ReversibleOperations = countReversible(sortedResources)

	return plan, nil
}

// RollbackPlan represents a planned rollback operation
type RollbackPlan struct {
	SnapshotID           string
	Timestamp            time.Time
	Description          string
	Operations           []PlannedOperation
	TotalOperations      int
	ReversibleOperations int
}

// PlannedOperation represents a single planned rollback operation
type PlannedOperation struct {
	Order      int
	Host       string
	Resource   string
	Module     string
	Action     string
	Details    string
	Reversible bool
}

// countReversible counts the number of reversible resources
func countReversible(resources []ResourceSnapshot) int {
	count := 0
	for _, r := range resources {
		if r.Reversible && r.RollbackOp != nil {
			count++
		}
	}
	return count
}

// GetSnapshotInfo returns information about a snapshot
func (re *RollbackExecutor) GetSnapshotInfo(snapshotID string) (*SnapshotInfo, error) {
	snapshot, err := re.snapshotManager.LoadSnapshot(snapshotID)
	if err != nil {
		return nil, err
	}

	info := &SnapshotInfo{
		ID:              snapshot.ID,
		Timestamp:       snapshot.Timestamp,
		PlaybookID:      snapshot.PlaybookID,
		Description:     snapshot.Description,
		TotalResources:  len(snapshot.Resources),
		ReversibleCount: countReversible(snapshot.Resources),
		ResourcesByType: make(map[string]int),
		ResourcesByHost: make(map[string]int),
	}

	for _, resource := range snapshot.Resources {
		info.ResourcesByType[resource.Type]++
		info.ResourcesByHost[resource.Host]++
	}

	return info, nil
}

// SnapshotInfo provides summary information about a snapshot
type SnapshotInfo struct {
	ID              string
	Timestamp       time.Time
	PlaybookID      string
	Description     string
	TotalResources  int
	ReversibleCount int
	ResourcesByType map[string]int
	ResourcesByHost map[string]int
}

// ListSnapshots returns a list of all available snapshots
func (re *RollbackExecutor) ListSnapshots() ([]SnapshotInfo, error) {
	snapshots, err := re.snapshotManager.ListSnapshots()
	if err != nil {
		return nil, err
	}

	infos := make([]SnapshotInfo, 0, len(snapshots))
	for _, snapshot := range snapshots {
		info := SnapshotInfo{
			ID:              snapshot.ID,
			Timestamp:       snapshot.Timestamp,
			PlaybookID:      snapshot.PlaybookID,
			Description:     snapshot.Description,
			TotalResources:  len(snapshot.Resources),
			ReversibleCount: countReversible(snapshot.Resources),
			ResourcesByType: make(map[string]int),
			ResourcesByHost: make(map[string]int),
		}

		for _, resource := range snapshot.Resources {
			info.ResourcesByType[resource.Type]++
			info.ResourcesByHost[resource.Host]++
		}

		infos = append(infos, info)
	}

	return infos, nil
}

// describeOperation says in a few words what a rollback operation does
func describeOperation(op *RollbackOperation) string {
	a := op.Args
	attrs := ""
	for _, k := range []string{"mode", "owner", "group"} {
		if v, ok := a[k]; ok {
			attrs += fmt.Sprintf(" %s=%v", k, v)
		}
	}
	switch {
	case op.Module == "file" && a["state"] == "absent":
		return "remove (did not exist before)"
	case op.Module == "file" && a["state"] == "directory":
		return "restore directory" + attrs
	case op.Module == "copy":
		content, _ := a["content"].(string)
		return fmt.Sprintf("restore content (%d bytes)%s", len(content), attrs)
	}
	return "run " + op.Module
}
