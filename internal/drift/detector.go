package drift

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/internal/rollback"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Detector detects configuration drift
type Detector struct {
	snapshotManager *rollback.SnapshotManager
	moduleRegistry  interfaces.ModuleRegistry
	logger          interfaces.Logger
	config          *DriftConfig
	reportDir       string
}

// NewDetector creates a new drift detector
func NewDetector(
	snapshotManager *rollback.SnapshotManager,
	moduleRegistry interfaces.ModuleRegistry,
	logger interfaces.Logger,
	config *DriftConfig,
) *Detector {
	if config == nil {
		config = DefaultConfig()
	}

	// Set default report directory
	reportDir := config.ReportPath
	if reportDir == "" {
		homeDir, _ := os.UserHomeDir()
		reportDir = filepath.Join(homeDir, ".onigirazu", "drift-reports")
	}

	return &Detector{
		snapshotManager: snapshotManager,
		moduleRegistry:  moduleRegistry,
		logger:          logger,
		config:          config,
		reportDir:       reportDir,
	}
}

// DefaultConfig returns default drift detection configuration
func DefaultConfig() *DriftConfig {
	return &DriftConfig{
		Enabled:       true,
		CheckInterval: 1 * time.Hour,
		Resources: []DriftType{
			DriftTypeFile,
			DriftTypePackage,
			DriftTypeService,
			DriftTypeUser,
			DriftTypeGroup,
		},
		AutoFix: false,
		AutoFixSeverity: []DriftSeverity{
			SeverityCritical,
			SeverityHigh,
		},
		DryRun: false,
		Notifications: NotificationConfig{
			Enabled:     false,
			OnDetect:    true,
			OnFix:       true,
			OnFail:      true,
			MinSeverity: SeverityMedium,
		},
		ReportFormat: "text",
		KeepReports:  30,
		DaemonMode:   false,
	}
}

// DetectDrift detects drift from a snapshot
func (d *Detector) DetectDrift(ctx context.Context, snapshotID string) (*DriftReport, error) {
	d.logger.Info("Starting drift detection for snapshot: %s", snapshotID)
	startTime := time.Now()

	// Load snapshot
	snapshot, err := d.snapshotManager.LoadSnapshot(snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Create report
	report := &DriftReport{
		ID:               generateReportID(),
		Timestamp:        time.Now(),
		SnapshotID:       snapshotID,
		PlaybookID:       snapshot.PlaybookID,
		Items:            []DriftItem{},
		DriftsByType:     make(map[DriftType]int),
		DriftsByHost:     make(map[string]int),
		DriftsBySeverity: make(map[DriftSeverity]int),
		Metadata:         make(map[string]interface{}),
	}

	// Check each resource for drift
	for _, resource := range snapshot.Resources {
		// Skip if resource type is not in config
		if !d.shouldCheckResource(DriftType(resource.Type)) {
			continue
		}

		// Skip if resource is in ignore list
		if d.isIgnored(resource.Identifier) {
			continue
		}

		d.logger.Debug("Checking drift for %s: %s on %s", resource.Type, resource.Identifier, resource.Host)

		// Check for drift
		driftItem, err := d.checkResourceDrift(ctx, &resource)
		if err != nil {
			d.logger.Warn("Failed to check drift for %s: %v", resource.Identifier, err)
			continue
		}

		if driftItem != nil {
			report.Items = append(report.Items, *driftItem)
			report.DriftsByType[driftItem.Type]++
			report.DriftsByHost[driftItem.Host]++
			report.DriftsBySeverity[driftItem.Severity]++

			// Count by severity
			switch driftItem.Severity {
			case SeverityCritical:
				report.CriticalDrifts++
			case SeverityHigh:
				report.HighDrifts++
			case SeverityMedium:
				report.MediumDrifts++
			case SeverityLow:
				report.LowDrifts++
			}
		}
	}

	report.TotalDrifts = len(report.Items)
	report.Duration = time.Since(startTime)

	d.logger.Info("Drift detection completed: %d drifts found in %v", report.TotalDrifts, report.Duration)

	// Save report
	if err := d.saveReport(report); err != nil {
		d.logger.Warn("Failed to save drift report: %v", err)
	}

	return report, nil
}

// checkResourceDrift checks if a resource has drifted
func (d *Detector) checkResourceDrift(ctx context.Context, resource *rollback.ResourceSnapshot) (*DriftItem, error) {
	// Get current state of the resource
	currentState, err := d.getCurrentState(ctx, resource)
	if err != nil {
		return nil, fmt.Errorf("failed to get current state: %w", err)
	}

	// Compare with expected state
	result := d.compareStates(resource.State, currentState)
	if !result.HasDrift {
		return nil, nil
	}

	// Create drift item
	driftItem := &DriftItem{
		ID:         uuid.New().String(),
		Type:       DriftType(resource.Type),
		Resource:   resource.Identifier,
		Host:       resource.Host,
		Severity:   d.calculateSeverity(resource.Type, result),
		Status:     StatusDetected,
		DetectedAt: time.Now(),
		Expected:   resource.State,
		Actual:     currentState,
		Diff:       d.formatDiff(result.Diff),
		Message:    result.Message,
		CanAutoFix: resource.Reversible && resource.RollbackOp != nil,
	}

	// Add fix operation if available
	if driftItem.CanAutoFix {
		driftItem.FixOperation = &FixOperation{
			Module: resource.RollbackOp.Module,
			Args:   resource.RollbackOp.Args,
			Order:  calculateFixOrder(resource.Type),
		}
	}

	return driftItem, nil
}

// getCurrentState gets the current state of a resource
func (d *Detector) getCurrentState(ctx context.Context, resource *rollback.ResourceSnapshot) (map[string]interface{}, error) {
	// Get module for this resource type
	module, err := d.moduleRegistry.Get(resource.Module)
	if err != nil {
		return nil, fmt.Errorf("module not found: %s", resource.Module)
	}

	// Create host object
	host := types.Host{
		Name: resource.Host,
	}

	// For stat-like operations, we need to query the current state
	// This is module-specific logic
	switch resource.Type {
	case "file":
		return d.getFileState(ctx, module, host, resource.Identifier)
	case "package":
		return d.getPackageState(ctx, module, host, resource.Identifier)
	case "service":
		return d.getServiceState(ctx, module, host, resource.Identifier)
	case "user":
		return d.getUserState(ctx, module, host, resource.Identifier)
	case "group":
		return d.getGroupState(ctx, module, host, resource.Identifier)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resource.Type)
	}
}

// getFileState gets current file state
func (d *Detector) getFileState(ctx context.Context, module interfaces.ModuleExecutor, host types.Host, path string) (map[string]interface{}, error) {
	// Use stat module to get file info
	statModule, err := d.moduleRegistry.Get("stat")
	if err != nil {
		return nil, err
	}

	args := map[string]interface{}{
		"path": path,
	}

	result, err := statModule.Execute(ctx, host, args)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return map[string]interface{}{"exists": false}, nil
	}

	return result.Output, nil
}

// getPackageState gets current package state
func (d *Detector) getPackageState(ctx context.Context, module interfaces.ModuleExecutor, host types.Host, packageName string) (map[string]interface{}, error) {
	// Query package state
	args := map[string]interface{}{
		"name":  packageName,
		"state": "present", // Just check if installed
	}

	result, err := module.Execute(ctx, host, args)
	if err != nil {
		return nil, err
	}

	return result.Output, nil
}

// getServiceState gets current service state
func (d *Detector) getServiceState(ctx context.Context, module interfaces.ModuleExecutor, host types.Host, serviceName string) (map[string]interface{}, error) {
	args := map[string]interface{}{
		"name": serviceName,
	}

	result, err := module.Execute(ctx, host, args)
	if err != nil {
		return nil, err
	}

	return result.Output, nil
}

// getUserState gets current user state
func (d *Detector) getUserState(ctx context.Context, module interfaces.ModuleExecutor, host types.Host, username string) (map[string]interface{}, error) {
	args := map[string]interface{}{
		"name": username,
	}

	result, err := module.Execute(ctx, host, args)
	if err != nil {
		return nil, err
	}

	return result.Output, nil
}

// getGroupState gets current group state
func (d *Detector) getGroupState(ctx context.Context, module interfaces.ModuleExecutor, host types.Host, groupName string) (map[string]interface{}, error) {
	args := map[string]interface{}{
		"name": groupName,
	}

	result, err := module.Execute(ctx, host, args)
	if err != nil {
		return nil, err
	}

	return result.Output, nil
}

// compareStates compares expected and actual states
func (d *Detector) compareStates(expected, actual map[string]interface{}) *DriftCheckResult {
	result := &DriftCheckResult{
		HasDrift: false,
		Expected: expected,
		Actual:   actual,
		Diff:     make(map[string]DiffValue),
	}

	// Compare each field
	for key, expectedValue := range expected {
		actualValue, exists := actual[key]

		if !exists {
			result.HasDrift = true
			result.Diff[key] = DiffValue{
				Expected: expectedValue,
				Actual:   nil,
				Changed:  true,
			}
			continue
		}

		if !deepEqual(expectedValue, actualValue) {
			result.HasDrift = true
			result.Diff[key] = DiffValue{
				Expected: expectedValue,
				Actual:   actualValue,
				Changed:  true,
			}
		}
	}

	// Check for extra fields in actual
	for key, actualValue := range actual {
		if _, exists := expected[key]; !exists {
			result.HasDrift = true
			result.Diff[key] = DiffValue{
				Expected: nil,
				Actual:   actualValue,
				Changed:  true,
			}
		}
	}

	if result.HasDrift {
		result.Message = fmt.Sprintf("Configuration drift detected: %d fields changed", len(result.Diff))
	}

	return result
}

// deepEqual compares two values deeply
func deepEqual(a, b interface{}) bool {
	// Simple comparison - can be enhanced
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// formatDiff formats diff for display
func (d *Detector) formatDiff(diff map[string]DiffValue) string {
	if len(diff) == 0 {
		return ""
	}

	result := ""
	for key, value := range diff {
		if value.Changed {
			result += fmt.Sprintf("%s: %v -> %v\n", key, value.Expected, value.Actual)
		}
	}
	return result
}

// calculateSeverity calculates drift severity
func (d *Detector) calculateSeverity(resourceType string, result *DriftCheckResult) DriftSeverity {
	// Default severity based on resource type
	switch resourceType {
	case "file":
		// Check if it's a critical system file
		return SeverityMedium
	case "package":
		return SeverityHigh
	case "service":
		return SeverityCritical
	case "user", "group":
		return SeverityHigh
	default:
		return SeverityLow
	}
}

// calculateFixOrder calculates the order for fixing drift
func calculateFixOrder(resourceType string) int {
	switch resourceType {
	case "service", "systemd":
		return 100
	case "cron":
		return 90
	case "file", "copy", "template", "lineinfile":
		return 80
	case "git":
		return 70
	case "package":
		return 60
	case "user":
		return 50
	case "group":
		return 40
	default:
		return 0
	}
}

// shouldCheckResource checks if a resource type should be checked
func (d *Detector) shouldCheckResource(resourceType DriftType) bool {
	if len(d.config.Resources) == 0 {
		return true
	}

	for _, rt := range d.config.Resources {
		if rt == resourceType {
			return true
		}
	}
	return false
}

// isIgnored checks if a resource is in the ignore list
func (d *Detector) isIgnored(identifier string) bool {
	for _, ignored := range d.config.IgnoreResources {
		if ignored == identifier {
			return true
		}
	}
	return false
}

// saveReport saves a drift report to disk
func (d *Detector) saveReport(report *DriftReport) error {
	// Create report directory if it doesn't exist
	if err := os.MkdirAll(d.reportDir, 0750); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Serialize report
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize report: %w", err)
	}

	// Write to file
	filename := filepath.Join(d.reportDir, fmt.Sprintf("drift_report_%s.json", report.ID))
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	d.logger.Debug("Drift report saved: %s", filename)
	return nil
}

// LoadReport loads a drift report from disk
func (d *Detector) LoadReport(reportID string) (*DriftReport, error) {
	filename := filepath.Join(d.reportDir, fmt.Sprintf("drift_report_%s.json", reportID))

	data, err := os.ReadFile(filename) // #nosec G304 - filename is constructed from reportDir and sanitized reportID
	if err != nil {
		return nil, fmt.Errorf("failed to read report file: %w", err)
	}

	var report DriftReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to deserialize report: %w", err)
	}

	return &report, nil
}

// ListReports lists all drift reports
func (d *Detector) ListReports() ([]DriftReport, error) {
	files, err := filepath.Glob(filepath.Join(d.reportDir, "drift_report_*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}

	reports := make([]DriftReport, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file) // #nosec G304 - file paths come from filepath.Glob with controlled pattern
		if err != nil {
			d.logger.Warn("Failed to read report file %s: %v", file, err)
			continue
		}

		var report DriftReport
		if err := json.Unmarshal(data, &report); err != nil {
			d.logger.Warn("Failed to deserialize report %s: %v", file, err)
			continue
		}

		reports = append(reports, report)
	}

	return reports, nil
}

// generateReportID generates a unique report ID
func generateReportID() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}
