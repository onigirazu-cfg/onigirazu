package modules

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
)

// EnhancedAptManager implements enhanced APT package management
type EnhancedAptManager struct {
	executor    *executor.CommandExecutor
	cache       *PackageStateCache
	lastUpdate  time.Time
	updateMutex sync.Mutex
}

// NewEnhancedAptManager creates a new enhanced APT manager
func NewEnhancedAptManager(exec *executor.CommandExecutor) *EnhancedAptManager {
	return &EnhancedAptManager{
		executor: exec,
		cache:    NewPackageStateCache(10 * time.Minute),
	}
}

// Install installs a package with enhanced idempotency
func (a *EnhancedAptManager) Install(ctx context.Context, name, version string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "install",
		Success:   false,
	}

	// Check current state first
	currentState, err := a.IsInstalled(ctx, name)
	if err != nil {
		operation.Error = fmt.Sprintf("failed to check current state: %v", err)
		operation.Duration = time.Since(startTime)
		return operation, err
	}

	operation.OldVersion = currentState.Version

	// Check if already installed with correct version
	if currentState.Installed {
		if version == "" || currentState.Version == version {
			operation.Success = true
			operation.Changed = false
			operation.Output = "Package already installed with correct version"
			operation.Duration = time.Since(startTime)
			return operation, nil
		}
	}

	// Prepare package specification
	packageSpec := name
	if version != "" {
		packageSpec = fmt.Sprintf("%s=%s", name, version)
	}

	// Execute installation with context
	output, err := a.executeWithContext(ctx, "sudo", "apt-get", "install", "-y", "--no-install-recommends", packageSpec)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("installation failed: %v", err)
		return operation, err
	}

	// Verify installation
	newState, err := a.IsInstalled(ctx, name)
	if err != nil {
		operation.Error = fmt.Sprintf("failed to verify installation: %v", err)
		return operation, err
	}

	if !newState.Installed {
		operation.Error = "package not found after installation"
		return operation, fmt.Errorf("package not found after installation")
	}

	operation.Success = true
	operation.Changed = true
	operation.NewVersion = newState.Version

	// Update cache
	a.cache.Set(name, newState)

	return operation, nil
}

// Remove removes a package with enhanced idempotency
func (a *EnhancedAptManager) Remove(ctx context.Context, name string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "remove",
		Success:   false,
	}

	// Check current state first
	currentState, err := a.IsInstalled(ctx, name)
	if err != nil {
		operation.Error = fmt.Sprintf("failed to check current state: %v", err)
		operation.Duration = time.Since(startTime)
		return operation, err
	}

	operation.OldVersion = currentState.Version

	// Check if already removed
	if !currentState.Installed {
		operation.Success = true
		operation.Changed = false
		operation.Output = "Package already removed"
		operation.Duration = time.Since(startTime)
		return operation, nil
	}

	// Execute removal with context
	output, err := a.executeWithContext(ctx, "sudo", "apt-get", "remove", "-y", name)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("removal failed: %v", err)
		return operation, err
	}

	// Clear cache to force fresh check
	a.cache.Delete(name)

	// Verify removal
	newState, err := a.IsInstalled(ctx, name)
	if err != nil {
		operation.Error = fmt.Sprintf("failed to verify removal: %v", err)
		return operation, err
	}

	if newState.Installed {
		operation.Error = "package still installed after removal"
		return operation, fmt.Errorf("package still installed after removal")
	}

	operation.Success = true
	operation.Changed = true

	// Update cache
	a.cache.Set(name, newState)

	return operation, nil
}

// Update updates a package with enhanced idempotency
func (a *EnhancedAptManager) Update(ctx context.Context, name string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "update",
		Success:   false,
	}

	// Check current state first
	currentState, err := a.IsInstalled(ctx, name)
	if err != nil {
		operation.Error = fmt.Sprintf("failed to check current state: %v", err)
		operation.Duration = time.Since(startTime)
		return operation, err
	}

	if !currentState.Installed {
		operation.Error = "package not installed, cannot update"
		operation.Duration = time.Since(startTime)
		return operation, fmt.Errorf("package not installed")
	}

	operation.OldVersion = currentState.Version

	// Check if update is available
	if currentState.AvailableVersion == "" || currentState.Version == currentState.AvailableVersion {
		operation.Success = true
		operation.Changed = false
		operation.Output = "Package already at latest version"
		operation.Duration = time.Since(startTime)
		return operation, nil
	}

	// Execute update with context
	output, err := a.executeWithContext(ctx, "sudo", "apt-get", "install", "-y", "--only-upgrade", name)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("update failed: %v", err)
		return operation, err
	}

	// Verify update
	newState, err := a.IsInstalled(ctx, name)
	if err != nil {
		operation.Error = fmt.Sprintf("failed to verify update: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.NewVersion = newState.Version
	operation.Changed = (currentState.Version != newState.Version)

	// Update cache
	a.cache.Set(name, newState)

	return operation, nil
}

// UpdateAll updates all packages
func (a *EnhancedAptManager) UpdateAll(ctx context.Context) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   "all",
		Operation: "update_all",
		Success:   false,
	}

	// First update package lists
	if err := a.RefreshCache(ctx); err != nil {
		operation.Error = fmt.Sprintf("failed to update package lists: %v", err)
		operation.Duration = time.Since(startTime)
		return operation, err
	}

	// Execute upgrade
	output, err := a.executeWithContext(ctx, "sudo", "apt-get", "upgrade", "-y")
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("upgrade failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = strings.Contains(output, "upgraded")

	// Clear cache to force refresh
	a.cache.Clear()

	return operation, nil
}

// IsInstalled checks if a package is installed with caching
func (a *EnhancedAptManager) IsInstalled(ctx context.Context, name string) (*EnhancedPackageState, error) {
	// Check cache first
	if state, found := a.cache.Get(name); found {
		return state, nil
	}

	// Query system
	state, err := a.queryPackageState(ctx, name)
	if err != nil {
		return nil, err
	}

	// Cache the result
	a.cache.Set(name, state)

	return state, nil
}

// queryPackageState queries the actual package state from the system
func (a *EnhancedAptManager) queryPackageState(ctx context.Context, name string) (*EnhancedPackageState, error) {
	state := &EnhancedPackageState{
		Name:        name,
		Installed:   false,
		LastChecked: time.Now(),
	}

	// Check installation status
	output, err := a.executeWithContext(ctx, "dpkg-query", "-W", "-f='${Status} ${Version}'", name)
	if err == nil && strings.Contains(output, "install ok installed") {
		state.Installed = true
		parts := strings.Fields(output)
		if len(parts) >= 4 {
			state.Version = parts[3]
		}
	}

	// Check available version if installed
	if state.Installed {
		availOutput, err := a.executeWithContext(ctx, "apt-cache", "policy", name)
		if err == nil {
			state.AvailableVersion = a.parseAvailableVersion(availOutput)
		}
	}

	// Generate hash for change detection
	state.Hash = generateStateHash(name, state.Version, "")

	return state, nil
}

// parseAvailableVersion parses available version from apt-cache policy output
func (a *EnhancedAptManager) parseAvailableVersion(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Candidate:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 && parts[1] != "(none)" {
				return parts[1]
			}
		}
	}
	return ""
}

// GetPackageInfo gets detailed package information
func (a *EnhancedAptManager) GetPackageInfo(ctx context.Context, name string) (*EnhancedPackageInfo, error) {
	info := &EnhancedPackageInfo{
		PackageInfo: PackageInfo{Name: name},
	}

	// Get basic state
	state, err := a.IsInstalled(ctx, name)
	if err != nil {
		return nil, err
	}

	info.Installed = state.Installed
	info.Version = state.Version

	// Get detailed information
	output, err := a.executeWithContext(ctx, "apt-cache", "show", name)
	if err != nil {
		return info, nil // Return basic info even if detailed info fails
	}

	// Parse detailed information
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Description: ") {
			info.Description = strings.TrimPrefix(line, "Description: ")
		} else if strings.HasPrefix(line, "Architecture: ") {
			info.Architecture = strings.TrimPrefix(line, "Architecture: ")
		} else if strings.HasPrefix(line, "Size: ") {
			info.Size = strings.TrimPrefix(line, "Size: ")
		} else if strings.HasPrefix(line, "Depends: ") {
			deps := strings.TrimPrefix(line, "Depends: ")
			info.Dependencies = a.parseDependencies(deps)
		}
	}

	return info, nil
}

// parseDependencies parses dependency string
func (a *EnhancedAptManager) parseDependencies(deps string) []string {
	// Simple parsing - can be enhanced
	parts := strings.Split(deps, ",")
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Remove version constraints
		re := regexp.MustCompile(`^([^\s(]+)`)
		matches := re.FindStringSubmatch(part)
		if len(matches) > 1 {
			result = append(result, matches[1])
		}
	}
	return result
}

// InstallMultiple installs multiple packages in batch
func (a *EnhancedAptManager) InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error) {
	startTime := time.Now()
	batch := &BatchOperation{
		Operations: make([]PackageOperation, 0, len(packages)),
		Success:    true,
	}

	// Process each package
	for _, pkg := range packages {
		operation, err := a.Install(ctx, pkg.Name, pkg.Version)
		batch.Operations = append(batch.Operations, *operation)

		if err != nil {
			batch.Success = false
		}

		if operation.Changed {
			batch.Changed = true
		}
	}

	batch.Duration = time.Since(startTime)
	batch.Summary = fmt.Sprintf("Processed %d packages, %d changed",
		len(packages), a.countChangedOperations(batch.Operations))

	return batch, nil
}

// RemoveMultiple removes multiple packages in batch
func (a *EnhancedAptManager) RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error) {
	startTime := time.Now()
	batch := &BatchOperation{
		Operations: make([]PackageOperation, 0, len(packages)),
		Success:    true,
	}

	// Process each package
	for _, pkg := range packages {
		operation, err := a.Remove(ctx, pkg)
		batch.Operations = append(batch.Operations, *operation)

		if err != nil {
			batch.Success = false
		}

		if operation.Changed {
			batch.Changed = true
		}
	}

	batch.Duration = time.Since(startTime)
	batch.Summary = fmt.Sprintf("Processed %d packages, %d changed",
		len(packages), a.countChangedOperations(batch.Operations))

	return batch, nil
}

// RefreshCache refreshes the package cache
func (a *EnhancedAptManager) RefreshCache(ctx context.Context) error {
	a.updateMutex.Lock()
	defer a.updateMutex.Unlock()

	// Don't update too frequently
	if time.Since(a.lastUpdate) < 5*time.Minute {
		return nil
	}

	_, err := a.executeWithContext(ctx, "sudo", "apt-get", "update")
	if err == nil {
		a.lastUpdate = time.Now()
		// Clear package cache to force refresh
		a.cache.Clear()
	}

	return err
}

// ValidateState validates the current state
func (a *EnhancedAptManager) ValidateState(ctx context.Context) error {
	// Check if apt is available and working
	_, err := a.executeWithContext(ctx, "apt-get", "--version")
	return err
}

// DryRun performs a dry run of the operation
func (a *EnhancedAptManager) DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error) {
	preview := &OperationPreview{
		WillChange: false,
		Actions:    []string{},
	}

	if len(args) == 0 {
		return preview, fmt.Errorf("package name required")
	}

	name := args[0]
	version := ""
	if len(args) > 1 {
		version = args[1]
	}

	// Check current state
	currentState, err := a.IsInstalled(ctx, name)
	if err != nil {
		return preview, err
	}

	switch operation {
	case "present":
		if !currentState.Installed {
			preview.WillChange = true
			preview.Actions = append(preview.Actions, fmt.Sprintf("Install %s", name))
		} else if version != "" && currentState.Version != version {
			preview.WillChange = true
			preview.Actions = append(preview.Actions, fmt.Sprintf("Change %s from %s to %s", name, currentState.Version, version))
		}

	case "absent":
		if currentState.Installed {
			preview.WillChange = true
			preview.Actions = append(preview.Actions, fmt.Sprintf("Remove %s", name))
		}

	case "latest":
		if !currentState.Installed {
			preview.WillChange = true
			preview.Actions = append(preview.Actions, fmt.Sprintf("Install %s (latest)", name))
		} else if currentState.AvailableVersion != "" && currentState.Version != currentState.AvailableVersion {
			preview.WillChange = true
			preview.Actions = append(preview.Actions, fmt.Sprintf("Update %s from %s to %s", name, currentState.Version, currentState.AvailableVersion))
		}
	}

	// Get size information for install operations
	if preview.WillChange && (operation == "present" || operation == "latest") {
		if sizeInfo, err := a.getPackageSize(ctx, name); err == nil {
			preview.Size = sizeInfo
		}
	}

	return preview, nil
}

// getPackageSize gets the package size information
func (a *EnhancedAptManager) getPackageSize(ctx context.Context, name string) (string, error) {
	output, err := a.executeWithContext(ctx, "apt-cache", "show", name)
	if err != nil {
		return "", err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Size: ") {
			sizeStr := strings.TrimPrefix(line, "Size: ")
			if size, err := strconv.Atoi(sizeStr); err == nil {
				return formatSize(size), nil
			}
		}
	}

	return "", nil
}

// executeWithContext executes a command with context support
func (a *EnhancedAptManager) executeWithContext(ctx context.Context, command string, args ...string) (string, error) {
	// Use the executor with context support
	return a.executor.ExecuteWithContext(ctx, command, args...)
}

// countChangedOperations counts how many operations resulted in changes
func (a *EnhancedAptManager) countChangedOperations(operations []PackageOperation) int {
	count := 0
	for _, op := range operations {
		if op.Changed {
			count++
		}
	}
	return count
}

// formatSize formats byte size to human readable format
func formatSize(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
