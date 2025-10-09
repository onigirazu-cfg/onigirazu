package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
)

// EnhancedBrewManager implements enhanced Homebrew package management
type EnhancedBrewManager struct {
	executor *executor.CommandExecutor
	cache    *PackageStateCache
}

// NewEnhancedBrewManager creates a new enhanced Brew manager
func NewEnhancedBrewManager(exec *executor.CommandExecutor) *EnhancedBrewManager {
	return &EnhancedBrewManager{
		executor: exec,
		cache:    NewPackageStateCache(10 * time.Minute),
	}
}

func (b *EnhancedBrewManager) Install(ctx context.Context, name, version string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "install",
		Success:   false,
	}

	// Check current state
	currentState, err := b.IsInstalled(ctx, name)
	if err != nil {
		operation.Error = fmt.Sprintf("failed to check current state: %v", err)
		operation.Duration = time.Since(startTime)
		return operation, err
	}

	if currentState.Installed {
		operation.Success = true
		operation.Changed = false
		operation.Output = "Package already installed"
		operation.Duration = time.Since(startTime)
		return operation, nil
	}

	// Execute installation
	output, err := b.executor.ExecuteWithContext(ctx, "brew", "install", name)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("installation failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = true

	// Update cache
	newState, _ := b.queryPackageState(ctx, name)
	b.cache.Set(name, newState)

	return operation, nil
}

func (b *EnhancedBrewManager) Remove(ctx context.Context, name string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "remove",
		Success:   false,
	}

	// Check current state
	currentState, err := b.IsInstalled(ctx, name)
	if err != nil {
		operation.Error = fmt.Sprintf("failed to check current state: %v", err)
		operation.Duration = time.Since(startTime)
		return operation, err
	}

	if !currentState.Installed {
		operation.Success = true
		operation.Changed = false
		operation.Output = "Package already removed"
		operation.Duration = time.Since(startTime)
		return operation, nil
	}

	// Execute removal
	output, err := b.executor.ExecuteWithContext(ctx, "brew", "uninstall", name)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("removal failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = true

	// Update cache
	newState, _ := b.queryPackageState(ctx, name)
	b.cache.Set(name, newState)

	return operation, nil
}

func (b *EnhancedBrewManager) Update(ctx context.Context, name string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "update",
		Success:   false,
	}

	// Execute update
	output, err := b.executor.ExecuteWithContext(ctx, "brew", "upgrade", name)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("update failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = !contains(output, "already installed")

	return operation, nil
}

func (b *EnhancedBrewManager) UpdateAll(ctx context.Context) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   "all",
		Operation: "update_all",
		Success:   false,
	}

	// Update brew first
	if _, err := b.executor.ExecuteWithContext(ctx, "brew", "update"); err != nil {
		operation.Error = fmt.Sprintf("brew update failed: %v", err)
		operation.Duration = time.Since(startTime)
		return operation, err
	}

	// Upgrade all packages
	output, err := b.executor.ExecuteWithContext(ctx, "brew", "upgrade")
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("brew upgrade failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = !contains(output, "Already up-to-date")

	return operation, nil
}

func (b *EnhancedBrewManager) IsInstalled(ctx context.Context, name string) (*EnhancedPackageState, error) {
	// Check cache first
	if state, found := b.cache.Get(name); found {
		return state, nil
	}

	// Query system
	state, err := b.queryPackageState(ctx, name)
	if err != nil {
		return nil, err
	}

	// Cache the result
	b.cache.Set(name, state)

	return state, nil
}

func (b *EnhancedBrewManager) queryPackageState(ctx context.Context, name string) (*EnhancedPackageState, error) {
	state := &EnhancedPackageState{
		Name:        name,
		Installed:   false,
		LastChecked: time.Now(),
	}

	output, err := b.executor.ExecuteWithContext(ctx, "brew", "list", "--versions", name)
	if err == nil && output != "" {
		state.Installed = true
		// Parse version from output
		parts := strings.Fields(output)
		if len(parts) >= 2 {
			state.Version = parts[1]
		}
	}

	state.Hash = generateStateHash(name, state.Version, "")
	return state, nil
}

func (b *EnhancedBrewManager) GetPackageInfo(ctx context.Context, name string) (*EnhancedPackageInfo, error) {
	info := &EnhancedPackageInfo{
		PackageInfo: PackageInfo{Name: name},
	}

	state, err := b.IsInstalled(ctx, name)
	if err != nil {
		return nil, err
	}

	info.Installed = state.Installed
	info.Version = state.Version

	return info, nil
}

func (b *EnhancedBrewManager) InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error) {
	startTime := time.Now()
	batch := &BatchOperation{
		Operations: make([]PackageOperation, 0, len(packages)),
		Success:    true,
	}

	for _, pkg := range packages {
		operation, err := b.Install(ctx, pkg.Name, pkg.Version)
		batch.Operations = append(batch.Operations, *operation)

		if err != nil {
			batch.Success = false
		}

		if operation.Changed {
			batch.Changed = true
		}
	}

	batch.Duration = time.Since(startTime)
	batch.Summary = fmt.Sprintf("Processed %d packages", len(packages))

	return batch, nil
}

func (b *EnhancedBrewManager) RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error) {
	startTime := time.Now()
	batch := &BatchOperation{
		Operations: make([]PackageOperation, 0, len(packages)),
		Success:    true,
	}

	for _, pkg := range packages {
		operation, err := b.Remove(ctx, pkg)
		batch.Operations = append(batch.Operations, *operation)

		if err != nil {
			batch.Success = false
		}

		if operation.Changed {
			batch.Changed = true
		}
	}

	batch.Duration = time.Since(startTime)
	batch.Summary = fmt.Sprintf("Processed %d packages", len(packages))

	return batch, nil
}

func (b *EnhancedBrewManager) RefreshCache(ctx context.Context) error {
	_, err := b.executor.ExecuteWithContext(ctx, "brew", "update")
	if err == nil {
		b.cache.Clear()
	}
	return err
}

func (b *EnhancedBrewManager) ValidateState(ctx context.Context) error {
	_, err := b.executor.ExecuteWithContext(ctx, "brew", "--version")
	return err
}

func (b *EnhancedBrewManager) DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error) {
	preview := &OperationPreview{
		WillChange: false,
		Actions:    []string{},
	}

	if len(args) == 0 {
		return preview, fmt.Errorf("package name required")
	}

	name := args[0]
	currentState, err := b.IsInstalled(ctx, name)
	if err != nil {
		return preview, err
	}

	switch operation {
	case "present":
		if !currentState.Installed {
			preview.WillChange = true
			preview.Actions = append(preview.Actions, fmt.Sprintf("Install %s", name))
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
		}
	}

	return preview, nil
}

// EnhancedYumManager implements enhanced YUM package management
type EnhancedYumManager struct {
	executor *executor.CommandExecutor
	cache    *PackageStateCache
}

// NewEnhancedYumManager creates a new enhanced YUM manager
func NewEnhancedYumManager(exec *executor.CommandExecutor) *EnhancedYumManager {
	return &EnhancedYumManager{
		executor: exec,
		cache:    NewPackageStateCache(10 * time.Minute),
	}
}

// Implement similar methods for YUM...
func (y *EnhancedYumManager) Install(ctx context.Context, name, version string) (*PackageOperation, error) {
	// Similar implementation to APT but using yum commands
	return &PackageOperation{
		Package:   name,
		Operation: "install",
		Success:   false,
		Error:     "YUM manager not fully implemented yet",
	}, fmt.Errorf("not implemented")
}

func (y *EnhancedYumManager) Remove(ctx context.Context, name string) (*PackageOperation, error) {
	return &PackageOperation{}, fmt.Errorf("not implemented")
}

func (y *EnhancedYumManager) Update(ctx context.Context, name string) (*PackageOperation, error) {
	return &PackageOperation{}, fmt.Errorf("not implemented")
}

func (y *EnhancedYumManager) UpdateAll(ctx context.Context) (*PackageOperation, error) {
	return &PackageOperation{}, fmt.Errorf("not implemented")
}

func (y *EnhancedYumManager) IsInstalled(ctx context.Context, name string) (*EnhancedPackageState, error) {
	return &EnhancedPackageState{}, fmt.Errorf("not implemented")
}

func (y *EnhancedYumManager) GetPackageInfo(ctx context.Context, name string) (*EnhancedPackageInfo, error) {
	return &EnhancedPackageInfo{}, fmt.Errorf("not implemented")
}

func (y *EnhancedYumManager) InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error) {
	return &BatchOperation{}, fmt.Errorf("not implemented")
}

func (y *EnhancedYumManager) RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error) {
	return &BatchOperation{}, fmt.Errorf("not implemented")
}

func (y *EnhancedYumManager) RefreshCache(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (y *EnhancedYumManager) ValidateState(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (y *EnhancedYumManager) DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error) {
	return &OperationPreview{}, fmt.Errorf("not implemented")
}

// EnhancedDnfManager implements enhanced DNF package management
type EnhancedDnfManager struct {
	EnhancedYumManager // Inherit from YUM for now
}

// NewEnhancedDnfManager creates a new enhanced DNF manager
func NewEnhancedDnfManager(exec *executor.CommandExecutor) *EnhancedDnfManager {
	return &EnhancedDnfManager{
		EnhancedYumManager: *NewEnhancedYumManager(exec),
	}
}

// EnhancedChocolateyManager implements enhanced Chocolatey package management
type EnhancedChocolateyManager struct {
	executor *executor.CommandExecutor
	cache    *PackageStateCache
}

// NewEnhancedChocolateyManager creates a new enhanced Chocolatey manager
func NewEnhancedChocolateyManager(exec *executor.CommandExecutor) *EnhancedChocolateyManager {
	return &EnhancedChocolateyManager{
		executor: exec,
		cache:    NewPackageStateCache(10 * time.Minute),
	}
}

// Implement placeholder methods for Chocolatey...
func (c *EnhancedChocolateyManager) Install(ctx context.Context, name, version string) (*PackageOperation, error) {
	return &PackageOperation{}, fmt.Errorf("not implemented")
}

func (c *EnhancedChocolateyManager) Remove(ctx context.Context, name string) (*PackageOperation, error) {
	return &PackageOperation{}, fmt.Errorf("not implemented")
}

func (c *EnhancedChocolateyManager) Update(ctx context.Context, name string) (*PackageOperation, error) {
	return &PackageOperation{}, fmt.Errorf("not implemented")
}

func (c *EnhancedChocolateyManager) UpdateAll(ctx context.Context) (*PackageOperation, error) {
	return &PackageOperation{}, fmt.Errorf("not implemented")
}

func (c *EnhancedChocolateyManager) IsInstalled(ctx context.Context, name string) (*EnhancedPackageState, error) {
	return &EnhancedPackageState{}, fmt.Errorf("not implemented")
}

func (c *EnhancedChocolateyManager) GetPackageInfo(ctx context.Context, name string) (*EnhancedPackageInfo, error) {
	return &EnhancedPackageInfo{}, fmt.Errorf("not implemented")
}

func (c *EnhancedChocolateyManager) InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error) {
	return &BatchOperation{}, fmt.Errorf("not implemented")
}

func (c *EnhancedChocolateyManager) RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error) {
	return &BatchOperation{}, fmt.Errorf("not implemented")
}

func (c *EnhancedChocolateyManager) RefreshCache(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (c *EnhancedChocolateyManager) ValidateState(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (c *EnhancedChocolateyManager) DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error) {
	return &OperationPreview{}, fmt.Errorf("not implemented")
}

// EnhancedGenericManager provides a fallback implementation
type EnhancedGenericManager struct {
	executor *executor.CommandExecutor
}

// NewEnhancedGenericManager creates a new enhanced generic manager
func NewEnhancedGenericManager(exec *executor.CommandExecutor) *EnhancedGenericManager {
	return &EnhancedGenericManager{
		executor: exec,
	}
}

func (g *EnhancedGenericManager) Install(ctx context.Context, name, version string) (*PackageOperation, error) {
	return &PackageOperation{
		Package:   name,
		Operation: "install",
		Success:   false,
		Error:     "package management not supported on this platform",
	}, fmt.Errorf("package management not supported on this platform")
}

func (g *EnhancedGenericManager) Remove(ctx context.Context, name string) (*PackageOperation, error) {
	return &PackageOperation{}, fmt.Errorf("package management not supported on this platform")
}

func (g *EnhancedGenericManager) Update(ctx context.Context, name string) (*PackageOperation, error) {
	return &PackageOperation{}, fmt.Errorf("package management not supported on this platform")
}

func (g *EnhancedGenericManager) UpdateAll(ctx context.Context) (*PackageOperation, error) {
	return &PackageOperation{}, fmt.Errorf("package management not supported on this platform")
}

func (g *EnhancedGenericManager) IsInstalled(ctx context.Context, name string) (*EnhancedPackageState, error) {
	return &EnhancedPackageState{}, fmt.Errorf("package management not supported on this platform")
}

func (g *EnhancedGenericManager) GetPackageInfo(ctx context.Context, name string) (*EnhancedPackageInfo, error) {
	return &EnhancedPackageInfo{}, fmt.Errorf("package management not supported on this platform")
}

func (g *EnhancedGenericManager) InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error) {
	return &BatchOperation{}, fmt.Errorf("package management not supported on this platform")
}

func (g *EnhancedGenericManager) RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error) {
	return &BatchOperation{}, fmt.Errorf("package management not supported on this platform")
}

func (g *EnhancedGenericManager) RefreshCache(ctx context.Context) error {
	return fmt.Errorf("package management not supported on this platform")
}

func (g *EnhancedGenericManager) ValidateState(ctx context.Context) error {
	return fmt.Errorf("package management not supported on this platform")
}

func (g *EnhancedGenericManager) DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error) {
	return &OperationPreview{}, fmt.Errorf("package management not supported on this platform")
}

// Helper function
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
