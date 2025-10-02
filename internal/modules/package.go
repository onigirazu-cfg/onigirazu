package modules

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// PackageModule implements package management
type PackageModule struct {
	BaseModule
	packageManager PackageManager
}

// PackageManager interface for different package management systems
type PackageManager interface {
	Install(name, version string) error
	Remove(name string) error
	Update(name string) error
	UpdateAll() error
	IsInstalled(name string) (bool, string, error)
	Search(query string) ([]PackageInfo, error)
	ListInstalled() ([]PackageInfo, error)
	GetInfo(name string) (PackageInfo, error)
	Clean() error
}

// PackageInfo represents package information
type PackageInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Description  string `json:"description"`
	Architecture string `json:"architecture"`
	Repository   string `json:"repository"`
	Size         string `json:"size"`
	Installed    bool   `json:"installed"`
	Upgradable   bool   `json:"upgradable"`
	NewVersion   string `json:"new_version,omitempty"`
}

// PackageState represents the desired package state
type PackageState string

const (
	PackageStatePresent PackageState = "present"
	PackageStateAbsent  PackageState = "absent"
	PackageStateLatest  PackageState = "latest"
)

// NewPackageModule creates a new package module
func NewPackageModule() *PackageModule {
	return &PackageModule{
		BaseModule: BaseModule{
			name:        "package",
			description: "Manage system packages",
		},
		packageManager: nil, // Will be created in Execute method
	}
}

// createPackageManager creates appropriate package manager with executor
func createPackageManager(exec *executor.CommandExecutor, hostname string) PackageManager {
	// Detect package management system on the REMOTE host, not local
	// Try to detect by executing commands on the remote host

	// Try apt (Debian/Ubuntu)
	if _, err := exec.Execute("which", "apt-get"); err == nil {
		return &AptManager{executor: exec, hostname: hostname}
	}

	// Try yum (RHEL/CentOS)
	if _, err := exec.Execute("which", "yum"); err == nil {
		return &YumManager{executor: exec}
	}

	// Try dnf (Fedora/RHEL 8+)
	if _, err := exec.Execute("which", "dnf"); err == nil {
		return &DnfManager{YumManager{executor: exec}}
	}

	// Try pacman (Arch)
	if _, err := exec.Execute("which", "pacman"); err == nil {
		return &PacmanManager{GenericPackageManager{executor: exec}}
	}

	// Try zypper (openSUSE)
	if _, err := exec.Execute("which", "zypper"); err == nil {
		return &ZypperManager{GenericPackageManager{executor: exec}}
	}

	// Try brew (macOS)
	if _, err := exec.Execute("which", "brew"); err == nil {
		return &BrewManager{executor: exec}
	}

	// Try choco (Windows)
	if _, err := exec.Execute("which", "choco"); err == nil {
		return &ChocolateyManager{GenericPackageManager{executor: exec}}
	}

	// Fallback to generic
	return &GenericPackageManager{executor: exec}
}

// Execute manages system packages
func (m *PackageModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {

	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "package",
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Create executor for this host
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to create executor: %v", err))
	}
	defer exec.Close()

	// Create package manager with executor
	m.packageManager = createPackageManager(exec, host.Name)

	// PackageSpec represents a package with optional version
	type PackageSpec struct {
		Name    string
		Version string
	}

	// Get required parameters - support multiple formats
	var packageSpecs []PackageSpec

	nameArg, exists := args["name"]
	if !exists {
		return m.failResult(result, "name parameter is required")
	}

	// Global version (applies to all packages if not specified individually)
	globalVersion := getStringArg(args, "version", "")

	// Handle multiple formats:
	// 1. Single string: "git"
	// 2. List of strings: ["git", "curl"]
	// 3. List of objects: [{name: "git", version: "1.2.3"}, {name: "curl"}]
	switch v := nameArg.(type) {
	case string:
		// Single package name
		packageSpecs = []PackageSpec{{Name: v, Version: globalVersion}}

	case []interface{}:
		for i, item := range v {
			switch itemVal := item.(type) {
			case string:
				// Simple string in list
				packageSpecs = append(packageSpecs, PackageSpec{
					Name:    itemVal,
					Version: globalVersion,
				})

			case map[string]interface{}:
				// Object with name and optional version
				pkgName, ok := itemVal["name"].(string)
				if !ok {
					return m.failResult(result, fmt.Sprintf("name[%d].name must be a string", i))
				}
				pkgVersion := ""
				if v, ok := itemVal["version"].(string); ok {
					pkgVersion = v
				} else if globalVersion != "" {
					pkgVersion = globalVersion
				}
				packageSpecs = append(packageSpecs, PackageSpec{
					Name:    pkgName,
					Version: pkgVersion,
				})

			default:
				return m.failResult(result, fmt.Sprintf("name[%d] must be a string or object with 'name' field", i))
			}
		}

	case []string:
		// List of strings
		for _, name := range v {
			packageSpecs = append(packageSpecs, PackageSpec{
				Name:    name,
				Version: globalVersion,
			})
		}

	default:
		return m.failResult(result, "name parameter must be a string, list of strings, or list of objects")
	}

	if len(packageSpecs) == 0 {
		return m.failResult(result, "at least one package name is required")
	}

	state := PackageState(getStringArg(args, "state", "present"))
	updateCache := getBoolArg(args, "update_cache", false)

	// Update cache if requested
	if updateCache {
		if err := m.packageManager.Clean(); err != nil {
			// Don't fail on cache update errors, just log
			result.Output["cache_update_warning"] = err.Error()
		}
	}

	// Process each package
	packagesInfo := make(map[string]interface{})
	overallChanged := false

	for _, pkg := range packageSpecs {
		packageResult := make(map[string]interface{})

		// Store requested version if specified
		if pkg.Version != "" {
			packageResult["requested_version"] = pkg.Version
		}

		// Check current installation status
		installed, currentVersion, err := m.packageManager.IsInstalled(pkg.Name)
		if err != nil {
			packageResult["error"] = fmt.Sprintf("failed to check package status: %v", err)
			packagesInfo[pkg.Name] = packageResult
			continue
		}

		packageResult["installed"] = installed
		packageResult["current_version"] = currentVersion

		// Handle different states
		changed := false
		switch state {
		case PackageStatePresent:
			if !installed {
				if err := m.packageManager.Install(pkg.Name, pkg.Version); err != nil {
					packageResult["error"] = fmt.Sprintf("failed to install: %v", err)
				} else {
					changed = true
					packageResult["action"] = "installed"
				}
			} else if pkg.Version != "" && currentVersion != pkg.Version {
				if err := m.packageManager.Install(pkg.Name, pkg.Version); err != nil {
					packageResult["error"] = fmt.Sprintf("failed to install specific version: %v", err)
				} else {
					changed = true
					packageResult["action"] = "version_changed"
				}
			} else {
				packageResult["action"] = "already_installed"
			}

		case PackageStateAbsent:
			if installed {
				if err := m.packageManager.Remove(pkg.Name); err != nil {
					packageResult["error"] = fmt.Sprintf("failed to remove: %v", err)
				} else {
					changed = true
					packageResult["action"] = "removed"
				}
			} else {
				packageResult["action"] = "already_absent"
			}

		case PackageStateLatest:
			if !installed {
				if err := m.packageManager.Install(pkg.Name, ""); err != nil {
					packageResult["error"] = fmt.Sprintf("failed to install: %v", err)
				} else {
					changed = true
					packageResult["action"] = "installed"
				}
			} else {
				if err := m.packageManager.Update(pkg.Name); err != nil {
					packageResult["error"] = fmt.Sprintf("failed to update: %v", err)
				} else {
					// Check if version actually changed
					newInstalled, newVersion, err := m.packageManager.IsInstalled(pkg.Name)
					if err == nil && newInstalled && newVersion != currentVersion {
						changed = true
						packageResult["action"] = "updated"
						packageResult["new_version"] = newVersion
					} else {
						packageResult["action"] = "already_latest"
					}
				}
			}
		}

		packageResult["changed"] = changed
		if changed {
			overallChanged = true
		}

		// Get final package info
		if info, err := m.packageManager.GetInfo(pkg.Name); err == nil {
			packageResult["package_info"] = info
		}

		packagesInfo[pkg.Name] = packageResult
	}

	result.Output["packages"] = packagesInfo
	result.Output["package_count"] = len(packageSpecs)
	result.Changed = overallChanged
	result.Duration = time.Since(startTime)

	return result, nil
}

// Validate validates package module arguments
func (m *PackageModule) Validate(args map[string]interface{}) error {
	if _, exists := args["name"]; !exists {
		return fmt.Errorf("name parameter is required")
	}

	if state, exists := args["state"]; exists {
		if stateStr, ok := state.(string); ok {
			validStates := []string{"present", "absent", "latest"}
			valid := false
			for _, validState := range validStates {
				if stateStr == validState {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid state: %s", stateStr)
			}
		}
	}

	return nil
}

// AptManager implements package management for APT (Debian/Ubuntu)
type AptManager struct {
	executor *executor.CommandExecutor
	hostname string
}

func (a *AptManager) Install(name, version string) error {
	packageSpec := name
	if version != "" {
		packageSpec = fmt.Sprintf("%s=%s", name, version)
	}

	output, err := a.executor.Execute("sudo", "apt-get", "install", "-y", packageSpec)
	if err != nil {
		return fmt.Errorf("failed to install package %s: %v (output: %s)", packageSpec, err, output)
	}

	// Invalidate cache after installation
	cache.GetGlobalPackageCache().Invalidate(a.hostname, name)

	return nil
}

func (a *AptManager) Remove(name string) error {
	output, err := a.executor.Execute("sudo", "apt-get", "remove", "-y", name)
	if err != nil {
		return fmt.Errorf("failed to remove package %s: %v (output: %s)", name, err, output)
	}

	// Invalidate cache after removal
	cache.GetGlobalPackageCache().Invalidate(a.hostname, name)

	return nil
}

func (a *AptManager) Update(name string) error {
	output, err := a.executor.Execute("sudo", "apt-get", "install", "-y", "--only-upgrade", name)
	if err != nil {
		return fmt.Errorf("failed to update package %s: %v (output: %s)", name, err, output)
	}

	// Invalidate cache after update
	cache.GetGlobalPackageCache().Invalidate(a.hostname, name)

	return nil
}

func (a *AptManager) UpdateAll() error {
	if output, err := a.executor.Execute("sudo", "apt-get", "update"); err != nil {
		return fmt.Errorf("failed to update package cache: %v (output: %s)", err, output)
	}

	output, err := a.executor.Execute("sudo", "apt-get", "upgrade", "-y")
	if err != nil {
		return fmt.Errorf("failed to upgrade packages: %v (output: %s)", err, output)
	}
	return nil
}

func (a *AptManager) IsInstalled(name string) (bool, string, error) {
	// Try to get from cache first
	pkgCache := cache.GetGlobalPackageCache()
	if cached, found := pkgCache.Get(a.hostname, name); found {
		return cached.Installed, cached.Version, nil
	}

	// Use dpkg -l instead of dpkg-query to avoid shell variable expansion issues
	output, err := a.executor.Execute("dpkg", "-l", name)

	installed := false
	version := ""

	if err != nil {
		// Package not installed or dpkg failed
		installed = false
		version = ""
	} else {
		// Parse dpkg -l output
		// Format: ii  package-name  version  architecture  description
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "ii ") {
				// Package is installed
				parts := strings.Fields(line)
				if len(parts) >= 3 && parts[1] == name {
					installed = true
					version = parts[2]
					break
				}
			}
		}
	}

	// Cache the result
	pkgCache.Set(a.hostname, name, installed, version)

	return installed, version, nil
}

func (a *AptManager) Search(query string) ([]PackageInfo, error) {
	output, err := a.executor.Execute("apt-cache", "search", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search packages: %v", err)
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

func (a *AptManager) ListInstalled() ([]PackageInfo, error) {
	output, err := a.executor.Execute("dpkg-query", "-W", "-f=${Package} ${Version} ${Architecture}\n")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %v", err)
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
				Name:         parts[0],
				Version:      parts[1],
				Architecture: parts[2],
				Installed:    true,
			})
		}
	}

	return packages, nil
}

func (a *AptManager) GetInfo(name string) (PackageInfo, error) {
	info := PackageInfo{Name: name}

	// Check if installed
	installed, version, _ := a.IsInstalled(name)
	info.Installed = installed
	info.Version = version

	// Get package information
	output, err := a.executor.Execute("apt-cache", "show", name)
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
		}
	}

	return info, nil
}

func (a *AptManager) Clean() error {
	output, err := a.executor.Execute("sudo", "apt-get", "update")
	if err != nil {
		return fmt.Errorf("failed to update package cache: %v (output: %s)", err, output)
	}
	return nil
}

// YumManager implements package management for YUM (RHEL/CentOS)
type YumManager struct {
	executor *executor.CommandExecutor
}

func (y *YumManager) Install(name, version string) error {
	packageSpec := name
	if version != "" {
		packageSpec = fmt.Sprintf("%s-%s", name, version)
	}

	output, err := y.executor.Execute("sudo", "yum", "install", "-y", packageSpec)
	if err != nil {
		return fmt.Errorf("failed to install package %s: %v (output: %s)", packageSpec, err, output)
	}
	return nil
}

func (y *YumManager) Remove(name string) error {
	output, err := y.executor.Execute("sudo", "yum", "remove", "-y", name)
	if err != nil {
		return fmt.Errorf("failed to remove package %s: %v (output: %s)", name, err, output)
	}
	return nil
}

func (y *YumManager) Update(name string) error {
	output, err := y.executor.Execute("sudo", "yum", "update", "-y", name)
	if err != nil {
		return fmt.Errorf("failed to update package %s: %v (output: %s)", name, err, output)
	}
	return nil
}

func (y *YumManager) UpdateAll() error {
	output, err := y.executor.Execute("sudo", "yum", "update", "-y")
	if err != nil {
		return fmt.Errorf("failed to update all packages: %v (output: %s)", err, output)
	}
	return nil
}

func (y *YumManager) IsInstalled(name string) (bool, string, error) {
	output, err := y.executor.Execute("rpm", "-q", name)
	if err != nil {
		return false, "", nil // Package not installed
	}

	outputStr := strings.TrimSpace(output)
	if strings.Contains(outputStr, "not installed") {
		return false, "", nil
	}

	// Extract version from output like "package-1.2.3-4.el7.x86_64"
	parts := strings.Split(outputStr, "-")
	if len(parts) >= 2 {
		version := strings.Join(parts[1:], "-")
		return true, version, nil
	}

	return true, "", nil
}

func (y *YumManager) Search(query string) ([]PackageInfo, error) {
	output, err := y.executor.Execute("yum", "search", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search packages: %v", err)
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

func (y *YumManager) ListInstalled() ([]PackageInfo, error) {
	output, err := y.executor.Execute("rpm", "-qa", "--queryformat", "%{NAME} %{VERSION}-%{RELEASE} %{ARCH}\n")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %v", err)
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
				Name:         parts[0],
				Version:      parts[1],
				Architecture: parts[2],
				Installed:    true,
			})
		}
	}

	return packages, nil
}

func (y *YumManager) GetInfo(name string) (PackageInfo, error) {
	info := PackageInfo{Name: name}

	installed, version, _ := y.IsInstalled(name)
	info.Installed = installed
	info.Version = version

	return info, nil
}

func (y *YumManager) Clean() error {
	output, err := y.executor.Execute("sudo", "yum", "clean", "all")
	if err != nil {
		return fmt.Errorf("failed to clean package cache: %v (output: %s)", err, output)
	}
	return nil
}

// BrewManager implements package management for Homebrew (macOS)
type BrewManager struct {
	executor *executor.CommandExecutor
}

func (b *BrewManager) Install(name, version string) error {
	if version != "" {
		// Homebrew doesn't support specific versions easily
		return fmt.Errorf("specific version installation not supported with Homebrew")
	}
	cmd := exec.Command("brew", "install", name)
	return cmd.Run()
}

func (b *BrewManager) Remove(name string) error {
	cmd := exec.Command("brew", "uninstall", name)
	return cmd.Run()
}

func (b *BrewManager) Update(name string) error {
	cmd := exec.Command("brew", "upgrade", name)
	return cmd.Run()
}

func (b *BrewManager) UpdateAll() error {
	if err := exec.Command("brew", "update").Run(); err != nil {
		return err
	}
	return exec.Command("brew", "upgrade").Run()
}

func (b *BrewManager) IsInstalled(name string) (bool, string, error) {
	cmd := exec.Command("brew", "list", "--versions", name)
	output, err := cmd.Output()
	if err != nil {
		return false, "", nil // Package not installed
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return false, "", nil
	}

	parts := strings.Fields(outputStr)
	if len(parts) >= 2 {
		return true, parts[1], nil
	}

	return true, "", nil
}

func (b *BrewManager) Search(query string) ([]PackageInfo, error) {
	cmd := exec.Command("brew", "search", query)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var packages []PackageInfo
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "=") {
			packages = append(packages, PackageInfo{
				Name: line,
			})
		}
	}

	return packages, nil
}

func (b *BrewManager) ListInstalled() ([]PackageInfo, error) {
	cmd := exec.Command("brew", "list", "--versions")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var packages []PackageInfo
	lines := strings.Split(string(output), "\n")
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

func (b *BrewManager) GetInfo(name string) (PackageInfo, error) {
	info := PackageInfo{Name: name}

	installed, version, _ := b.IsInstalled(name)
	info.Installed = installed
	info.Version = version

	return info, nil
}

func (b *BrewManager) Clean() error {
	return exec.Command("brew", "update").Run()
}

// Additional package managers (DnfManager, PacmanManager, etc.) would follow similar patterns...

// GenericPackageManager provides a fallback implementation
type GenericPackageManager struct {
	executor *executor.CommandExecutor
}

func (g *GenericPackageManager) Install(name, version string) error {
	return fmt.Errorf("package management not supported on this platform")
}

func (g *GenericPackageManager) Remove(name string) error {
	return fmt.Errorf("package management not supported on this platform")
}

func (g *GenericPackageManager) Update(name string) error {
	return fmt.Errorf("package management not supported on this platform")
}

func (g *GenericPackageManager) UpdateAll() error {
	return fmt.Errorf("package management not supported on this platform")
}

func (g *GenericPackageManager) IsInstalled(name string) (bool, string, error) {
	return false, "", fmt.Errorf("package management not supported on this platform")
}

func (g *GenericPackageManager) Search(query string) ([]PackageInfo, error) {
	return nil, fmt.Errorf("package management not supported on this platform")
}

func (g *GenericPackageManager) ListInstalled() ([]PackageInfo, error) {
	return nil, fmt.Errorf("package management not supported on this platform")
}

func (g *GenericPackageManager) GetInfo(name string) (PackageInfo, error) {
	return PackageInfo{}, fmt.Errorf("package management not supported on this platform")
}

func (g *GenericPackageManager) Clean() error {
	return fmt.Errorf("package management not supported on this platform")
}

// Helper functions
func hasCommand(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// failResult creates a failed result
func (m *PackageModule) failResult(result types.TaskResult, message string) (types.TaskResult, error) {
	result.Success = false
	result.Failed = true
	result.Error = message
	result.Duration = time.Since(result.Timestamp)
	return result, fmt.Errorf("%s", message)
}

// Placeholder implementations for other package managers
type DnfManager struct{ YumManager }
type PacmanManager struct{ GenericPackageManager }
type ZypperManager struct{ GenericPackageManager }
type ChocolateyManager struct{ GenericPackageManager }
