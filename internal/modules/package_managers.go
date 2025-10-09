package modules

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/executor"
)

// ============================================================================
// PACKAGE MANAGER FACTORY
// ============================================================================

// createUnifiedPackageManager creates appropriate package manager with executor
func createUnifiedPackageManager(ctx context.Context, exec *executor.CommandExecutor, hostname string) UnifiedPackageManager {
	// Detect package management system on the REMOTE host
	// Try to detect by executing commands on the remote host

	// Try apt (Debian/Ubuntu)
	if _, err := exec.Execute("which", "apt-get"); err == nil {
		return NewUnifiedAptManager(exec, hostname)
	}

	// Try yum (RHEL/CentOS)
	if _, err := exec.Execute("which", "yum"); err == nil {
		return NewUnifiedYumManager(exec, hostname)
	}

	// Try dnf (Fedora/RHEL 8+)
	if _, err := exec.Execute("which", "dnf"); err == nil {
		return NewUnifiedDnfManager(exec, hostname)
	}

	// Try pacman (Arch)
	if _, err := exec.Execute("which", "pacman"); err == nil {
		return NewUnifiedPacmanManager(exec, hostname)
	}

	// Try zypper (openSUSE)
	if _, err := exec.Execute("which", "zypper"); err == nil {
		return NewUnifiedZypperManager(exec, hostname)
	}

	// Try brew (macOS)
	if _, err := exec.Execute("which", "brew"); err == nil {
		return NewUnifiedBrewManager(exec, hostname)
	}

	// Try choco (Windows)
	if _, err := exec.Execute("which", "choco"); err == nil {
		return NewUnifiedChocoManager(exec, hostname)
	}

	// Fallback to generic
	return NewUnifiedGenericManager(exec, hostname)
}

// ============================================================================
// APT MANAGER (Debian/Ubuntu)
// ============================================================================

// UnifiedAptManager implements unified APT package management
type UnifiedAptManager struct {
	executor    *executor.CommandExecutor
	hostname    string
	cache       *PackageStateCache
	lastUpdate  time.Time
	updateMutex sync.Mutex
}

// NewUnifiedAptManager creates a new unified APT manager
func NewUnifiedAptManager(exec *executor.CommandExecutor, hostname string) *UnifiedAptManager {
	return &UnifiedAptManager{
		executor: exec,
		hostname: hostname,
		cache:    NewPackageStateCache(10 * time.Minute),
	}
}

func (a *UnifiedAptManager) Install(ctx context.Context, name, version string) (*PackageOperation, error) {
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
			operation.NewVersion = currentState.Version
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

	// Invalidate global cache
	cache.GetGlobalPackageCache().Invalidate(a.hostname, name)

	return operation, nil
}

func (a *UnifiedAptManager) Remove(ctx context.Context, name string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "remove",
		Success:   false,
	}

	// Check current state
	currentState, err := a.IsInstalled(ctx, name)
	if err != nil {
		operation.Error = fmt.Sprintf("failed to check current state: %v", err)
		operation.Duration = time.Since(startTime)
		return operation, err
	}

	operation.OldVersion = currentState.Version

	if !currentState.Installed {
		operation.Success = true
		operation.Changed = false
		operation.Output = "Package already removed"
		operation.Duration = time.Since(startTime)
		return operation, nil
	}

	// Execute removal
	output, err := a.executeWithContext(ctx, "sudo", "apt-get", "remove", "-y", name)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("removal failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = true

	// Update cache
	newState := &PackageState{
		Name:      name,
		Installed: false,
		Version:   "",
	}
	a.cache.Set(name, newState)

	// Invalidate global cache
	cache.GetGlobalPackageCache().Invalidate(a.hostname, name)

	return operation, nil
}

func (a *UnifiedAptManager) Update(ctx context.Context, name string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "update",
		Success:   false,
	}

	// Get current version
	currentState, err := a.IsInstalled(ctx, name)
	if err != nil {
		operation.Error = fmt.Sprintf("failed to check current state: %v", err)
		operation.Duration = time.Since(startTime)
		return operation, err
	}

	operation.OldVersion = currentState.Version

	// Execute update
	output, err := a.executeWithContext(ctx, "sudo", "apt-get", "install", "-y", "--only-upgrade", name)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("update failed: %v", err)
		return operation, err
	}

	// Check new version
	newState, err := a.IsInstalled(ctx, name)
	if err == nil {
		operation.NewVersion = newState.Version
		operation.Changed = operation.OldVersion != operation.NewVersion
		a.cache.Set(name, newState)
	}

	operation.Success = true

	// Invalidate global cache
	cache.GetGlobalPackageCache().Invalidate(a.hostname, name)

	return operation, nil
}

func (a *UnifiedAptManager) UpdateAll(ctx context.Context) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   "all",
		Operation: "update_all",
		Success:   false,
	}

	// Update package cache first
	if _, err := a.executeWithContext(ctx, "sudo", "apt-get", "update"); err != nil {
		operation.Error = fmt.Sprintf("failed to update package cache: %v", err)
		operation.Duration = time.Since(startTime)
		return operation, err
	}

	// Upgrade all packages
	output, err := a.executeWithContext(ctx, "sudo", "apt-get", "upgrade", "-y")
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("failed to upgrade packages: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = !strings.Contains(output, "0 upgraded")

	// Clear cache
	a.cache.Clear()

	return operation, nil
}

func (a *UnifiedAptManager) IsInstalled(ctx context.Context, name string) (*PackageState, error) {
	// Try to get from cache first
	if cached, found := a.cache.Get(name); found {
		return cached, nil
	}

	// Try global cache
	pkgCache := cache.GetGlobalPackageCache()
	if cached, found := pkgCache.Get(a.hostname, name); found {
		state := &PackageState{
			Name:      name,
			Installed: cached.Installed,
			Version:   cached.Version,
		}
		a.cache.Set(name, state)
		return state, nil
	}

	// Query package state
	state, err := a.queryPackageState(ctx, name)
	if err != nil {
		return &PackageState{Name: name, Installed: false}, err
	}

	// Cache the result
	a.cache.Set(name, state)
	pkgCache.Set(a.hostname, name, state.Installed, state.Version)

	return state, nil
}

func (a *UnifiedAptManager) queryPackageState(ctx context.Context, name string) (*PackageState, error) {
	state := &PackageState{
		Name:      name,
		Installed: false,
		Version:   "",
	}

	// Use dpkg -l to check installation status
	output, err := a.executeWithContext(ctx, "dpkg", "-l", name)

	if err != nil {
		// Package not installed or dpkg failed
		return state, nil
	}

	// Parse dpkg -l output
	// Format: ii  package-name  version  architecture  description
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ii ") {
			// Package is installed
			parts := strings.Fields(line)
			if len(parts) >= 3 && parts[1] == name {
				state.Installed = true
				state.Version = parts[2]
				break
			}
		}
	}

	// Get available version
	if availOutput, err := a.executeWithContext(ctx, "apt-cache", "policy", name); err == nil {
		if version := parseAptAvailableVersion(availOutput); version != "" {
			state.AvailableVersion = version
		}
	}

	// Get repository info
	if showOutput, err := a.executeWithContext(ctx, "apt-cache", "show", name); err == nil {
		if repo := parseAptRepository(showOutput); repo != "" {
			state.Repository = repo
		}
	}

	state.Hash = generateStateHash(name, state.Version, state.Repository)

	return state, nil
}

func (a *UnifiedAptManager) GetPackageInfo(ctx context.Context, name string) (*PackageInfo, error) {
	info := &PackageInfo{Name: name}

	// Get installation status
	state, err := a.IsInstalled(ctx, name)
	if err != nil {
		return info, err
	}

	info.Installed = state.Installed
	info.Version = state.Version
	info.Repository = state.Repository

	// Get package information
	output, err := a.executeWithContext(ctx, "apt-cache", "show", name)
	if err != nil {
		return info, fmt.Errorf("failed to get package info for %s: %v", name, err)
	}

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
			info.Dependencies = parseAptDependencies(deps)
		}
	}

	// Check if upgradable
	if state.AvailableVersion != "" && state.Version != state.AvailableVersion {
		info.Upgradable = true
		info.NewVersion = state.AvailableVersion
	}

	return info, nil
}

func (a *UnifiedAptManager) InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error) {
	startTime := time.Now()
	batchOp := &BatchOperation{
		Operations: make([]PackageOperation, 0, len(packages)),
		TotalCount: len(packages),
	}

	// Build package list
	var packageList []string
	for _, pkg := range packages {
		if pkg.Version != "" {
			packageList = append(packageList, fmt.Sprintf("%s=%s", pkg.Name, pkg.Version))
		} else {
			packageList = append(packageList, pkg.Name)
		}
	}

	// Execute batch installation
	output, err := a.executeWithContext(ctx, append([]string{"sudo", "apt-get", "install", "-y", "--no-install-recommends"}, packageList...)...)

	// Create operations for each package
	for _, pkg := range packages {
		op := PackageOperation{
			Package:   pkg.Name,
			Operation: "install",
			Output:    output,
		}

		if err != nil {
			op.Success = false
			op.Error = fmt.Sprintf("batch installation failed: %v", err)
		} else {
			// Verify installation
			state, _ := a.IsInstalled(ctx, pkg.Name)
			op.Success = state != nil && state.Installed
			op.Changed = op.Success
			if state != nil {
				op.NewVersion = state.Version
			}
		}

		batchOp.Operations = append(batchOp.Operations, op)
		if op.Success {
			batchOp.SuccessCount++
			if op.Changed {
				batchOp.ChangedCount++
			}
		} else {
			batchOp.FailedCount++
		}
	}

	batchOp.Duration = time.Since(startTime)
	batchOp.Success = batchOp.FailedCount == 0
	batchOp.Changed = batchOp.ChangedCount > 0
	batchOp.Summary = fmt.Sprintf("Batch install: %d packages, %d succeeded, %d failed",
		batchOp.TotalCount, batchOp.SuccessCount, batchOp.FailedCount)

	return batchOp, nil
}

func (a *UnifiedAptManager) RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error) {
	startTime := time.Now()
	batchOp := &BatchOperation{
		Operations: make([]PackageOperation, 0, len(packages)),
		TotalCount: len(packages),
	}

	// Execute batch removal
	output, err := a.executeWithContext(ctx, append([]string{"sudo", "apt-get", "remove", "-y"}, packages...)...)

	// Create operations for each package
	for _, name := range packages {
		op := PackageOperation{
			Package:   name,
			Operation: "remove",
			Output:    output,
		}

		if err != nil {
			op.Success = false
			op.Error = fmt.Sprintf("batch removal failed: %v", err)
		} else {
			// Verify removal
			state, _ := a.IsInstalled(ctx, name)
			op.Success = state == nil || !state.Installed
			op.Changed = op.Success
		}

		batchOp.Operations = append(batchOp.Operations, op)
		if op.Success {
			batchOp.SuccessCount++
			if op.Changed {
				batchOp.ChangedCount++
			}
		} else {
			batchOp.FailedCount++
		}
	}

	batchOp.Duration = time.Since(startTime)
	batchOp.Success = batchOp.FailedCount == 0
	batchOp.Changed = batchOp.ChangedCount > 0
	batchOp.Summary = fmt.Sprintf("Batch remove: %d packages, %d succeeded, %d failed",
		batchOp.TotalCount, batchOp.SuccessCount, batchOp.FailedCount)

	return batchOp, nil
}

func (a *UnifiedAptManager) RefreshCache(ctx context.Context) error {
	a.updateMutex.Lock()
	defer a.updateMutex.Unlock()

	// Don't update too frequently
	if time.Since(a.lastUpdate) < 5*time.Minute {
		return nil
	}

	_, err := a.executeWithContext(ctx, "sudo", "apt-get", "update")
	if err != nil {
		return fmt.Errorf("failed to update package cache: %w", err)
	}

	a.lastUpdate = time.Now()
	a.cache.Clear()

	return nil
}

func (a *UnifiedAptManager) ValidateState(ctx context.Context) error {
	// Check if apt is working
	_, err := a.executeWithContext(ctx, "apt-get", "--version")
	return err
}

func (a *UnifiedAptManager) DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error) {
	preview := &OperationPreview{
		Actions: make([]string, 0),
	}

	if len(args) == 0 {
		return preview, fmt.Errorf("package name required")
	}

	name := args[0]
	version := ""
	if len(args) > 1 {
		version = args[1]
	}

	packageSpec := name
	if version != "" {
		packageSpec = fmt.Sprintf("%s=%s", name, version)
	}

	var output string
	var err error

	switch operation {
	case "present", "install":
		output, err = a.executeWithContext(ctx, "apt-get", "install", "-s", packageSpec)
	case "absent", "remove":
		output, err = a.executeWithContext(ctx, "apt-get", "remove", "-s", name)
	case "latest", "update":
		output, err = a.executeWithContext(ctx, "apt-get", "install", "-s", "--only-upgrade", name)
	default:
		return preview, fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		return preview, err
	}

	// Parse simulation output
	preview.WillChange = strings.Contains(output, "NEW packages will be installed") ||
		strings.Contains(output, "packages will be REMOVED") ||
		strings.Contains(output, "upgraded")

	// Extract actions
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Inst ") || strings.Contains(line, "Remv ") {
			preview.Actions = append(preview.Actions, strings.TrimSpace(line))
		}
	}

	// Extract size information
	if match := regexp.MustCompile(`Need to get ([\d.]+\s*[kMG]?B)`).FindStringSubmatch(output); len(match) > 1 {
		preview.Size = match[1]
	}

	return preview, nil
}

func (a *UnifiedAptManager) GetDependencies(ctx context.Context, name string) ([]string, error) {
	output, err := a.executeWithContext(ctx, "apt-cache", "depends", name)
	if err != nil {
		return nil, err
	}

	var deps []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Depends: ") {
			dep := strings.TrimPrefix(line, "Depends: ")
			deps = append(deps, dep)
		}
	}

	return deps, nil
}

func (a *UnifiedAptManager) VerifyChecksum(ctx context.Context, name, version string) (bool, error) {
	// APT doesn't provide easy checksum verification
	// This would require downloading package info and comparing checksums
	return true, nil
}

// executeWithContext executes a command with context support
func (a *UnifiedAptManager) executeWithContext(ctx context.Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no command specified")
	}

	// Check if executor supports context
	if a.executor != nil {
		return a.executor.ExecuteWithContext(ctx, args[0], args[1:]...)
	}

	// Fallback to regular execution
	return a.executor.Execute(args[0], args[1:]...)
}

// Helper functions for parsing APT output

func parseAptAvailableVersion(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Candidate:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

func parseAptRepository(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "APT-Sources: ") {
			return strings.TrimPrefix(line, "APT-Sources: ")
		}
	}
	return ""
}

func parseAptDependencies(deps string) []string {
	var result []string
	parts := strings.Split(deps, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Remove version constraints
		if idx := strings.Index(part, " ("); idx != -1 {
			part = part[:idx]
		}
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// Search searches for packages matching query
func (a *UnifiedAptManager) Search(ctx context.Context, query string) ([]PackageInfo, error) {
	output, err := a.executeWithContext(ctx, "apt-cache", "search", query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var packages []PackageInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) == 2 {
			packages = append(packages, PackageInfo{
				Name:        parts[0],
				Description: parts[1],
			})
		}
	}

	return packages, nil
}

// ListInstalled lists all installed packages
func (a *UnifiedAptManager) ListInstalled(ctx context.Context) ([]PackageInfo, error) {
	output, err := a.executeWithContext(ctx, "dpkg-query", "-W", "-f=${Package}\t${Version}\t${Architecture}\n")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	var packages []PackageInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			pkg := PackageInfo{
				Name:      parts[0],
				Version:   parts[1],
				Installed: true,
			}
			if len(parts) >= 3 {
				pkg.Architecture = parts[2]
			}
			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

// ListUpgradable lists packages that can be upgraded
func (a *UnifiedAptManager) ListUpgradable(ctx context.Context) ([]PackageInfo, error) {
	output, err := a.executeWithContext(ctx, "apt", "list", "--upgradable")
	if err != nil {
		return nil, fmt.Errorf("failed to list upgradable packages: %w", err)
	}

	var packages []PackageInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "Listing...") {
			continue
		}

		// Format: package/repo version arch [upgradable from: old_version]
		parts := strings.Fields(line)
		if len(parts) >= 4 {
			nameParts := strings.Split(parts[0], "/")
			pkg := PackageInfo{
				Name:         nameParts[0],
				NewVersion:   parts[1],
				Architecture: parts[2],
				Upgradable:   true,
				Installed:    true,
			}

			// Extract old version
			if len(parts) >= 6 && parts[3] == "[upgradable" {
				pkg.Version = parts[5]
				pkg.Version = strings.TrimSuffix(pkg.Version, "]")
			}

			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

// Clean cleans package cache
func (a *UnifiedAptManager) Clean(ctx context.Context) error {
	_, err := a.executeWithContext(ctx, "sudo", "apt-get", "clean")
	if err != nil {
		return fmt.Errorf("clean failed: %w", err)
	}
	return nil
}

// AutoRemove removes orphaned packages
func (a *UnifiedAptManager) AutoRemove(ctx context.Context) ([]string, error) {
	// First, do a dry run to see what would be removed
	output, err := a.executeWithContext(ctx, "apt-get", "autoremove", "--dry-run")
	if err != nil {
		return nil, fmt.Errorf("autoremove check failed: %w", err)
	}

	var packages []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "The following packages will be REMOVED:") {
			continue
		}
		// Parse package names from output
		if strings.HasPrefix(line, "  ") {
			pkgs := strings.Fields(strings.TrimSpace(line))
			packages = append(packages, pkgs...)
		}
	}

	return packages, nil
}

// VerifyIntegrity verifies package system integrity
func (a *UnifiedAptManager) VerifyIntegrity(ctx context.Context) error {
	// Check for broken packages
	output, err := a.executeWithContext(ctx, "dpkg", "--audit")
	if err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}

	if strings.TrimSpace(output) != "" {
		return fmt.Errorf("broken packages detected: %s", output)
	}

	// Verify package database
	_, err = a.executeWithContext(ctx, "apt-get", "check")
	if err != nil {
		return fmt.Errorf("package database check failed: %w", err)
	}

	return nil
}

// ============================================================================
// YUM MANAGER (RHEL/CentOS)
// ============================================================================

// UnifiedYumManager implements unified YUM package management
type UnifiedYumManager struct {
	executor *executor.CommandExecutor
	hostname string
	cache    *PackageStateCache
}

// NewUnifiedYumManager creates a new unified YUM manager
func NewUnifiedYumManager(exec *executor.CommandExecutor, hostname string) *UnifiedYumManager {
	return &UnifiedYumManager{
		executor: exec,
		hostname: hostname,
		cache:    NewPackageStateCache(10 * time.Minute),
	}
}

func (y *UnifiedYumManager) Install(ctx context.Context, name, version string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "install",
		Success:   false,
	}

	packageSpec := name
	if version != "" {
		packageSpec = fmt.Sprintf("%s-%s", name, version)
	}

	output, err := y.executeWithContext(ctx, "sudo", "yum", "install", "-y", packageSpec)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("installation failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = !strings.Contains(output, "already installed")

	// Get new version
	if state, err := y.IsInstalled(ctx, name); err == nil && state.Installed {
		operation.NewVersion = state.Version
	}

	return operation, nil
}

func (y *UnifiedYumManager) Remove(ctx context.Context, name string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "remove",
		Success:   false,
	}

	output, err := y.executeWithContext(ctx, "sudo", "yum", "remove", "-y", name)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("removal failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = true

	return operation, nil
}

func (y *UnifiedYumManager) Update(ctx context.Context, name string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "update",
		Success:   false,
	}

	output, err := y.executeWithContext(ctx, "sudo", "yum", "update", "-y", name)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("update failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = !strings.Contains(output, "Nothing to do")

	return operation, nil
}

func (y *UnifiedYumManager) UpdateAll(ctx context.Context) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   "all",
		Operation: "update_all",
		Success:   false,
	}

	output, err := y.executeWithContext(ctx, "sudo", "yum", "update", "-y")
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("update all failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = !strings.Contains(output, "Nothing to do")

	return operation, nil
}

func (y *UnifiedYumManager) IsInstalled(ctx context.Context, name string) (*PackageState, error) {
	if cached, found := y.cache.Get(name); found {
		return cached, nil
	}

	state := &PackageState{
		Name:      name,
		Installed: false,
	}

	output, err := y.executeWithContext(ctx, "rpm", "-q", name)
	if err != nil {
		return state, nil // Package not installed
	}

	outputStr := strings.TrimSpace(output)
	if strings.Contains(outputStr, "not installed") {
		return state, nil
	}

	state.Installed = true

	// Extract version from output like "package-1.2.3-4.el7.x86_64"
	parts := strings.Split(outputStr, "-")
	if len(parts) >= 2 {
		state.Version = strings.Join(parts[1:], "-")
	}

	y.cache.Set(name, state)

	return state, nil
}

func (y *UnifiedYumManager) GetPackageInfo(ctx context.Context, name string) (*PackageInfo, error) {
	info := &PackageInfo{Name: name}

	state, _ := y.IsInstalled(ctx, name)
	info.Installed = state.Installed
	info.Version = state.Version

	return info, nil
}

func (y *UnifiedYumManager) InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error) {
	// YUM supports batch installation
	var packageList []string
	for _, pkg := range packages {
		if pkg.Version != "" {
			packageList = append(packageList, fmt.Sprintf("%s-%s", pkg.Name, pkg.Version))
		} else {
			packageList = append(packageList, pkg.Name)
		}
	}

	startTime := time.Now()
	output, err := y.executeWithContext(ctx, append([]string{"sudo", "yum", "install", "-y"}, packageList...)...)

	batchOp := &BatchOperation{
		Operations: make([]PackageOperation, 0, len(packages)),
		TotalCount: len(packages),
		Duration:   time.Since(startTime),
	}

	for _, pkg := range packages {
		op := PackageOperation{
			Package:   pkg.Name,
			Operation: "install",
			Output:    output,
			Success:   err == nil,
			Changed:   err == nil,
		}
		batchOp.Operations = append(batchOp.Operations, op)
		if op.Success {
			batchOp.SuccessCount++
			if op.Changed {
				batchOp.ChangedCount++
			}
		} else {
			batchOp.FailedCount++
		}
	}

	batchOp.Success = err == nil
	batchOp.Changed = batchOp.ChangedCount > 0

	return batchOp, nil
}

func (y *UnifiedYumManager) RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error) {
	startTime := time.Now()
	output, err := y.executeWithContext(ctx, append([]string{"sudo", "yum", "remove", "-y"}, packages...)...)

	batchOp := &BatchOperation{
		Operations: make([]PackageOperation, 0, len(packages)),
		TotalCount: len(packages),
		Duration:   time.Since(startTime),
	}

	for _, name := range packages {
		op := PackageOperation{
			Package:   name,
			Operation: "remove",
			Output:    output,
			Success:   err == nil,
			Changed:   err == nil,
		}
		batchOp.Operations = append(batchOp.Operations, op)
		if op.Success {
			batchOp.SuccessCount++
		} else {
			batchOp.FailedCount++
		}
	}

	batchOp.Success = err == nil
	batchOp.Changed = batchOp.SuccessCount > 0

	return batchOp, nil
}

func (y *UnifiedYumManager) RefreshCache(ctx context.Context) error {
	_, err := y.executeWithContext(ctx, "sudo", "yum", "clean", "all")
	if err != nil {
		return err
	}
	y.cache.Clear()
	return nil
}

func (y *UnifiedYumManager) ValidateState(ctx context.Context) error {
	_, err := y.executeWithContext(ctx, "yum", "--version")
	return err
}

func (y *UnifiedYumManager) DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error) {
	// YUM doesn't have a built-in dry-run mode like APT
	return &OperationPreview{
		WillChange: true,
		Actions:    []string{fmt.Sprintf("%s %v", operation, args)},
	}, nil
}

func (y *UnifiedYumManager) GetDependencies(ctx context.Context, name string) ([]string, error) {
	output, err := y.executeWithContext(ctx, "yum", "deplist", name)
	if err != nil {
		return nil, err
	}

	var deps []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "dependency: ") {
			dep := strings.TrimPrefix(line, "dependency: ")
			deps = append(deps, dep)
		}
	}

	return deps, nil
}

func (y *UnifiedYumManager) VerifyChecksum(ctx context.Context, name, version string) (bool, error) {
	return true, nil
}

// Search searches for packages (stub implementation)
func (y *UnifiedYumManager) Search(ctx context.Context, query string) ([]PackageInfo, error) {
	output, err := y.executeWithContext(ctx, "yum", "search", query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var packages []PackageInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, ".") && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				nameParts := strings.Fields(parts[0])
				if len(nameParts) > 0 {
					packages = append(packages, PackageInfo{
						Name:        nameParts[0],
						Description: strings.TrimSpace(parts[1]),
					})
				}
			}
		}
	}

	return packages, nil
}

// ListInstalled lists all installed packages (stub implementation)
func (y *UnifiedYumManager) ListInstalled(ctx context.Context) ([]PackageInfo, error) {
	output, err := y.executeWithContext(ctx, "rpm", "-qa", "--queryformat", "%{NAME}\t%{VERSION}-%{RELEASE}\t%{ARCH}\n")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	var packages []PackageInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			pkg := PackageInfo{
				Name:      parts[0],
				Version:   parts[1],
				Installed: true,
			}
			if len(parts) >= 3 {
				pkg.Architecture = parts[2]
			}
			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

// ListUpgradable lists packages that can be upgraded (stub implementation)
func (y *UnifiedYumManager) ListUpgradable(ctx context.Context) ([]PackageInfo, error) {
	output, err := y.executeWithContext(ctx, "yum", "list", "updates")
	if err != nil {
		return nil, fmt.Errorf("failed to list upgradable packages: %w", err)
	}

	var packages []PackageInfo
	lines := strings.Split(output, "\n")
	inUpdates := false

	for _, line := range lines {
		if strings.Contains(line, "Updated Packages") || strings.Contains(line, "Available Upgrades") {
			inUpdates = true
			continue
		}

		if inUpdates && line != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				nameParts := strings.Split(parts[0], ".")
				pkg := PackageInfo{
					Name:       nameParts[0],
					NewVersion: parts[1],
					Upgradable: true,
					Installed:  true,
				}
				if len(nameParts) > 1 {
					pkg.Architecture = nameParts[1]
				}
				packages = append(packages, pkg)
			}
		}
	}

	return packages, nil
}

// Clean cleans package cache (stub implementation)
func (y *UnifiedYumManager) Clean(ctx context.Context) error {
	_, err := y.executeWithContext(ctx, "sudo", "yum", "clean", "all")
	if err != nil {
		return fmt.Errorf("clean failed: %w", err)
	}
	return nil
}

// AutoRemove removes orphaned packages (stub implementation)
func (y *UnifiedYumManager) AutoRemove(ctx context.Context) ([]string, error) {
	output, err := y.executeWithContext(ctx, "package-cleanup", "--leaves", "--quiet")
	if err != nil {
		// package-cleanup might not be available
		return []string{}, nil
	}

	var packages []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			packages = append(packages, line)
		}
	}

	return packages, nil
}

// VerifyIntegrity verifies package system integrity (stub implementation)
func (y *UnifiedYumManager) VerifyIntegrity(ctx context.Context) error {
	_, err := y.executeWithContext(ctx, "rpm", "-Va")
	if err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}
	return nil
}

func (y *UnifiedYumManager) executeWithContext(ctx context.Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no command specified")
	}
	if y.executor != nil {
		return y.executor.ExecuteWithContext(ctx, args[0], args[1:]...)
	}
	return y.executor.Execute(args[0], args[1:]...)
}

// ============================================================================
// BREW MANAGER (macOS)
// ============================================================================

// UnifiedBrewManager implements unified Homebrew package management
type UnifiedBrewManager struct {
	executor *executor.CommandExecutor
	hostname string
	cache    *PackageStateCache
}

// NewUnifiedBrewManager creates a new unified Brew manager
func NewUnifiedBrewManager(exec *executor.CommandExecutor, hostname string) *UnifiedBrewManager {
	return &UnifiedBrewManager{
		executor: exec,
		hostname: hostname,
		cache:    NewPackageStateCache(10 * time.Minute),
	}
}

func (b *UnifiedBrewManager) Install(ctx context.Context, name, version string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "install",
		Success:   false,
	}

	if version != "" {
		operation.Error = "specific version installation not supported with Homebrew"
		operation.Duration = time.Since(startTime)
		return operation, fmt.Errorf("specific version installation not supported")
	}

	// Check if already installed
	state, _ := b.IsInstalled(ctx, name)
	if state.Installed {
		operation.Success = true
		operation.Changed = false
		operation.Output = "Package already installed"
		operation.Duration = time.Since(startTime)
		return operation, nil
	}

	output, err := b.executeWithContext(ctx, "brew", "install", name)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("installation failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = true

	// Get version
	if newState, err := b.IsInstalled(ctx, name); err == nil {
		operation.NewVersion = newState.Version
	}

	return operation, nil
}

func (b *UnifiedBrewManager) Remove(ctx context.Context, name string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "remove",
		Success:   false,
	}

	output, err := b.executeWithContext(ctx, "brew", "uninstall", name)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("removal failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = true

	return operation, nil
}

func (b *UnifiedBrewManager) Update(ctx context.Context, name string) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   name,
		Operation: "update",
		Success:   false,
	}

	output, err := b.executeWithContext(ctx, "brew", "upgrade", name)
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("update failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = !strings.Contains(output, "already installed")

	return operation, nil
}

func (b *UnifiedBrewManager) UpdateAll(ctx context.Context) (*PackageOperation, error) {
	startTime := time.Now()
	operation := &PackageOperation{
		Package:   "all",
		Operation: "update_all",
		Success:   false,
	}

	// Update brew first
	if _, err := b.executeWithContext(ctx, "brew", "update"); err != nil {
		operation.Error = fmt.Sprintf("brew update failed: %v", err)
		operation.Duration = time.Since(startTime)
		return operation, err
	}

	output, err := b.executeWithContext(ctx, "brew", "upgrade")
	operation.Output = output
	operation.Duration = time.Since(startTime)

	if err != nil {
		operation.Error = fmt.Sprintf("brew upgrade failed: %v", err)
		return operation, err
	}

	operation.Success = true
	operation.Changed = !strings.Contains(output, "Already up-to-date")

	return operation, nil
}

func (b *UnifiedBrewManager) IsInstalled(ctx context.Context, name string) (*PackageState, error) {
	if cached, found := b.cache.Get(name); found {
		return cached, nil
	}

	state := &PackageState{
		Name:      name,
		Installed: false,
	}

	output, err := b.executeWithContext(ctx, "brew", "list", "--versions", name)
	if err != nil {
		return state, nil
	}

	outputStr := strings.TrimSpace(output)
	if outputStr == "" {
		return state, nil
	}

	state.Installed = true
	parts := strings.Fields(outputStr)
	if len(parts) >= 2 {
		state.Version = parts[1]
	}

	b.cache.Set(name, state)

	return state, nil
}

func (b *UnifiedBrewManager) GetPackageInfo(ctx context.Context, name string) (*PackageInfo, error) {
	info := &PackageInfo{Name: name}

	state, _ := b.IsInstalled(ctx, name)
	info.Installed = state.Installed
	info.Version = state.Version

	// Get description
	if output, err := b.executeWithContext(ctx, "brew", "info", name); err == nil {
		lines := strings.Split(output, "\n")
		if len(lines) > 1 {
			info.Description = lines[1]
		}
	}

	return info, nil
}

func (b *UnifiedBrewManager) InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error) {
	var packageList []string
	for _, pkg := range packages {
		packageList = append(packageList, pkg.Name)
	}

	startTime := time.Now()
	output, err := b.executeWithContext(ctx, append([]string{"brew", "install"}, packageList...)...)

	batchOp := &BatchOperation{
		Operations: make([]PackageOperation, 0, len(packages)),
		TotalCount: len(packages),
		Duration:   time.Since(startTime),
	}

	for _, pkg := range packages {
		op := PackageOperation{
			Package:   pkg.Name,
			Operation: "install",
			Output:    output,
			Success:   err == nil,
			Changed:   err == nil,
		}
		batchOp.Operations = append(batchOp.Operations, op)
		if op.Success {
			batchOp.SuccessCount++
		} else {
			batchOp.FailedCount++
		}
	}

	batchOp.Success = err == nil
	batchOp.Changed = batchOp.SuccessCount > 0

	return batchOp, nil
}

func (b *UnifiedBrewManager) RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error) {
	startTime := time.Now()
	output, err := b.executeWithContext(ctx, append([]string{"brew", "uninstall"}, packages...)...)

	batchOp := &BatchOperation{
		Operations: make([]PackageOperation, 0, len(packages)),
		TotalCount: len(packages),
		Duration:   time.Since(startTime),
	}

	for _, name := range packages {
		op := PackageOperation{
			Package:   name,
			Operation: "remove",
			Output:    output,
			Success:   err == nil,
			Changed:   err == nil,
		}
		batchOp.Operations = append(batchOp.Operations, op)
		if op.Success {
			batchOp.SuccessCount++
		} else {
			batchOp.FailedCount++
		}
	}

	batchOp.Success = err == nil
	batchOp.Changed = batchOp.SuccessCount > 0

	return batchOp, nil
}

func (b *UnifiedBrewManager) RefreshCache(ctx context.Context) error {
	_, err := b.executeWithContext(ctx, "brew", "update")
	if err != nil {
		return err
	}
	b.cache.Clear()
	return nil
}

func (b *UnifiedBrewManager) ValidateState(ctx context.Context) error {
	_, err := b.executeWithContext(ctx, "brew", "--version")
	return err
}

func (b *UnifiedBrewManager) DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error) {
	return &OperationPreview{
		WillChange: true,
		Actions:    []string{fmt.Sprintf("%s %v", operation, args)},
	}, nil
}

func (b *UnifiedBrewManager) GetDependencies(ctx context.Context, name string) ([]string, error) {
	output, err := b.executeWithContext(ctx, "brew", "deps", name)
	if err != nil {
		return nil, err
	}

	var deps []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			deps = append(deps, line)
		}
	}

	return deps, nil
}

func (b *UnifiedBrewManager) VerifyChecksum(ctx context.Context, name, version string) (bool, error) {
	return true, nil
}

// Search searches for packages
func (b *UnifiedBrewManager) Search(ctx context.Context, query string) ([]PackageInfo, error) {
	output, err := b.executeWithContext(ctx, "brew", "search", query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var packages []PackageInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "==>") {
			packages = append(packages, PackageInfo{
				Name: line,
			})
		}
	}

	return packages, nil
}

// ListInstalled lists all installed packages
func (b *UnifiedBrewManager) ListInstalled(ctx context.Context) ([]PackageInfo, error) {
	output, err := b.executeWithContext(ctx, "brew", "list", "--versions")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	var packages []PackageInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			packages = append(packages, PackageInfo{
				Name:      parts[0],
				Version:   parts[1],
				Installed: true,
			})
		}
	}

	return packages, nil
}

// ListUpgradable lists packages that can be upgraded
func (b *UnifiedBrewManager) ListUpgradable(ctx context.Context) ([]PackageInfo, error) {
	output, err := b.executeWithContext(ctx, "brew", "outdated")
	if err != nil {
		return nil, fmt.Errorf("failed to list upgradable packages: %w", err)
	}

	var packages []PackageInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			packages = append(packages, PackageInfo{
				Name:       parts[0],
				Version:    parts[1],
				NewVersion: parts[2],
				Upgradable: true,
				Installed:  true,
			})
		}
	}

	return packages, nil
}

// Clean cleans package cache
func (b *UnifiedBrewManager) Clean(ctx context.Context) error {
	_, err := b.executeWithContext(ctx, "brew", "cleanup")
	if err != nil {
		return fmt.Errorf("clean failed: %w", err)
	}
	return nil
}

// AutoRemove removes orphaned packages (not applicable for Homebrew)
func (b *UnifiedBrewManager) AutoRemove(ctx context.Context) ([]string, error) {
	// Homebrew doesn't have a direct equivalent to autoremove
	// We can use "brew autoremove" if available (newer versions)
	output, err := b.executeWithContext(ctx, "brew", "autoremove", "--dry-run")
	if err != nil {
		return []string{}, nil // Not an error, just not supported
	}

	var packages []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "==>") {
			packages = append(packages, line)
		}
	}

	return packages, nil
}

// VerifyIntegrity verifies package system integrity
func (b *UnifiedBrewManager) VerifyIntegrity(ctx context.Context) error {
	_, err := b.executeWithContext(ctx, "brew", "doctor")
	if err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}
	return nil
}

func (b *UnifiedBrewManager) executeWithContext(ctx context.Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no command specified")
	}

	// Brew commands should run locally, not through SSH
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ============================================================================
// GENERIC/STUB MANAGERS
// ============================================================================

// UnifiedDnfManager wraps YumManager (DNF is YUM-compatible)
type UnifiedDnfManager struct {
	*UnifiedYumManager
}

func NewUnifiedDnfManager(exec *executor.CommandExecutor, hostname string) *UnifiedDnfManager {
	return &UnifiedDnfManager{
		UnifiedYumManager: NewUnifiedYumManager(exec, hostname),
	}
}

// UnifiedPacmanManager - stub for Arch Linux
type UnifiedPacmanManager struct {
	executor *executor.CommandExecutor
	hostname string
	cache    *PackageStateCache
}

func NewUnifiedPacmanManager(exec *executor.CommandExecutor, hostname string) *UnifiedPacmanManager {
	return &UnifiedPacmanManager{
		executor: exec,
		hostname: hostname,
		cache:    NewPackageStateCache(10 * time.Minute),
	}
}

// Implement stub methods (similar pattern to YUM)
func (p *UnifiedPacmanManager) Install(ctx context.Context, name, version string) (*PackageOperation, error) {
	return &PackageOperation{Package: name, Operation: "install", Success: false, Error: "pacman support not fully implemented"}, fmt.Errorf("not implemented")
}

func (p *UnifiedPacmanManager) Remove(ctx context.Context, name string) (*PackageOperation, error) {
	return &PackageOperation{Package: name, Operation: "remove", Success: false, Error: "pacman support not fully implemented"}, fmt.Errorf("not implemented")
}

func (p *UnifiedPacmanManager) Update(ctx context.Context, name string) (*PackageOperation, error) {
	return &PackageOperation{Package: name, Operation: "update", Success: false, Error: "pacman support not fully implemented"}, fmt.Errorf("not implemented")
}

func (p *UnifiedPacmanManager) UpdateAll(ctx context.Context) (*PackageOperation, error) {
	return &PackageOperation{Package: "all", Operation: "update_all", Success: false, Error: "pacman support not fully implemented"}, fmt.Errorf("not implemented")
}

func (p *UnifiedPacmanManager) IsInstalled(ctx context.Context, name string) (*PackageState, error) {
	return &PackageState{Name: name, Installed: false}, nil
}

func (p *UnifiedPacmanManager) GetPackageInfo(ctx context.Context, name string) (*PackageInfo, error) {
	return &PackageInfo{Name: name}, nil
}

func (p *UnifiedPacmanManager) InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error) {
	return &BatchOperation{Success: false}, fmt.Errorf("not implemented")
}

func (p *UnifiedPacmanManager) RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error) {
	return &BatchOperation{Success: false}, fmt.Errorf("not implemented")
}

func (p *UnifiedPacmanManager) RefreshCache(ctx context.Context) error {
	return nil
}

func (p *UnifiedPacmanManager) ValidateState(ctx context.Context) error {
	return nil
}

func (p *UnifiedPacmanManager) DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error) {
	return &OperationPreview{}, nil
}

func (p *UnifiedPacmanManager) GetDependencies(ctx context.Context, name string) ([]string, error) {
	return nil, nil
}

func (p *UnifiedPacmanManager) VerifyChecksum(ctx context.Context, name, version string) (bool, error) {
	return true, nil
}

// Stub implementations for new methods
func (p *UnifiedPacmanManager) Search(ctx context.Context, query string) ([]PackageInfo, error) {
	return []PackageInfo{}, fmt.Errorf("not implemented")
}

func (p *UnifiedPacmanManager) ListInstalled(ctx context.Context) ([]PackageInfo, error) {
	return []PackageInfo{}, fmt.Errorf("not implemented")
}

func (p *UnifiedPacmanManager) ListUpgradable(ctx context.Context) ([]PackageInfo, error) {
	return []PackageInfo{}, fmt.Errorf("not implemented")
}

func (p *UnifiedPacmanManager) Clean(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (p *UnifiedPacmanManager) AutoRemove(ctx context.Context) ([]string, error) {
	return []string{}, fmt.Errorf("not implemented")
}

func (p *UnifiedPacmanManager) VerifyIntegrity(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

// Similar stubs for Zypper, Chocolatey, Generic
type UnifiedZypperManager struct{ *UnifiedPacmanManager }
type UnifiedChocoManager struct{ *UnifiedPacmanManager }
type UnifiedGenericManager struct{ *UnifiedPacmanManager }

func NewUnifiedZypperManager(exec *executor.CommandExecutor, hostname string) *UnifiedZypperManager {
	return &UnifiedZypperManager{NewUnifiedPacmanManager(exec, hostname)}
}

func NewUnifiedChocoManager(exec *executor.CommandExecutor, hostname string) *UnifiedChocoManager {
	return &UnifiedChocoManager{NewUnifiedPacmanManager(exec, hostname)}
}

func NewUnifiedGenericManager(exec *executor.CommandExecutor, hostname string) *UnifiedGenericManager {
	return &UnifiedGenericManager{NewUnifiedPacmanManager(exec, hostname)}
}
