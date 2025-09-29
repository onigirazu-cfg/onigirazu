package modules

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

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
	var manager PackageManager

	// Detect package management system
	switch runtime.GOOS {
	case "linux":
		if hasCommand("apt") {
			manager = &AptManager{}
		} else if hasCommand("yum") {
			manager = &YumManager{}
		} else if hasCommand("dnf") {
			manager = &DnfManager{}
		} else if hasCommand("pacman") {
			manager = &PacmanManager{}
		} else if hasCommand("zypper") {
			manager = &ZypperManager{}
		} else {
			manager = &GenericPackageManager{}
		}
	case "darwin":
		if hasCommand("brew") {
			manager = &BrewManager{}
		} else {
			manager = &GenericPackageManager{}
		}
	case "windows":
		if hasCommand("choco") {
			manager = &ChocolateyManager{}
		} else {
			manager = &GenericPackageManager{}
		}
	default:
		manager = &GenericPackageManager{}
	}

	return &PackageModule{
		BaseModule: BaseModule{
			name:        "package",
			description: "Manage system packages",
		},
		packageManager: manager,
	}
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

	// Get required parameters
	name, ok := args["name"].(string)
	if !ok {
		return m.failResult(result, "name parameter is required")
	}

	state := PackageState(getStringArg(args, "state", "present"))
	version := getStringArg(args, "version", "")
	updateCache := getBoolArg(args, "update_cache", false)

	// Update cache if requested
	if updateCache {
		if err := m.packageManager.Clean(); err != nil {
			// Don't fail on cache update errors, just log
			result.Output["cache_update_warning"] = err.Error()
		}
	}

	// Check current installation status
	installed, currentVersion, err := m.packageManager.IsInstalled(name)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to check package status: %v", err))
	}

	result.Output["package_name"] = name
	result.Output["installed"] = installed
	result.Output["current_version"] = currentVersion

	// Handle different states
	changed := false
	switch state {
	case PackageStatePresent:
		if !installed {
			if err := m.packageManager.Install(name, version); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to install package: %v", err))
			}
			changed = true
			result.Output["action"] = "installed"
		} else if version != "" && currentVersion != version {
			if err := m.packageManager.Install(name, version); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to install specific version: %v", err))
			}
			changed = true
			result.Output["action"] = "version_changed"
		}

	case PackageStateAbsent:
		if installed {
			if err := m.packageManager.Remove(name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to remove package: %v", err))
			}
			changed = true
			result.Output["action"] = "removed"
		}

	case PackageStateLatest:
		if !installed {
			if err := m.packageManager.Install(name, ""); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to install package: %v", err))
			}
			changed = true
			result.Output["action"] = "installed"
		} else {
			if err := m.packageManager.Update(name); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to update package: %v", err))
			}
			// Check if version actually changed
			newInstalled, newVersion, err := m.packageManager.IsInstalled(name)
			if err == nil && newInstalled && newVersion != currentVersion {
				changed = true
				result.Output["action"] = "updated"
				result.Output["new_version"] = newVersion
			}
		}
	}

	// Get final package info
	if info, err := m.packageManager.GetInfo(name); err == nil {
		result.Output["package_info"] = info
	}

	result.Changed = changed
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
type AptManager struct{}

func (a *AptManager) Install(name, version string) error {
	packageSpec := name
	if version != "" {
		packageSpec = fmt.Sprintf("%s=%s", name, version)
	}
	cmd := exec.Command("apt-get", "install", "-y", packageSpec)
	return cmd.Run()
}

func (a *AptManager) Remove(name string) error {
	cmd := exec.Command("apt-get", "remove", "-y", name)
	return cmd.Run()
}

func (a *AptManager) Update(name string) error {
	cmd := exec.Command("apt-get", "install", "-y", "--only-upgrade", name)
	return cmd.Run()
}

func (a *AptManager) UpdateAll() error {
	if err := exec.Command("apt-get", "update").Run(); err != nil {
		return err
	}
	return exec.Command("apt-get", "upgrade", "-y").Run()
}

func (a *AptManager) IsInstalled(name string) (bool, string, error) {
	cmd := exec.Command("dpkg-query", "-W", "-f=${Status} ${Version}", name)
	output, err := cmd.Output()
	if err != nil {
		return false, "", nil // Package not installed
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "install ok installed") {
		parts := strings.Fields(outputStr)
		if len(parts) >= 4 {
			return true, parts[3], nil
		}
		return true, "", nil
	}

	return false, "", nil
}

func (a *AptManager) Search(query string) ([]PackageInfo, error) {
	cmd := exec.Command("apt-cache", "search", query)
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
	cmd := exec.Command("dpkg-query", "-W", "-f=${Package} ${Version} ${Architecture}\n")
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
	cmd := exec.Command("apt-cache", "show", name)
	output, err := cmd.Output()
	if err != nil {
		return info, err
	}

	lines := strings.Split(string(output), "\n")
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
	return exec.Command("apt-get", "update").Run()
}

// YumManager implements package management for YUM (RHEL/CentOS)
type YumManager struct{}

func (y *YumManager) Install(name, version string) error {
	packageSpec := name
	if version != "" {
		packageSpec = fmt.Sprintf("%s-%s", name, version)
	}
	cmd := exec.Command("yum", "install", "-y", packageSpec)
	return cmd.Run()
}

func (y *YumManager) Remove(name string) error {
	cmd := exec.Command("yum", "remove", "-y", name)
	return cmd.Run()
}

func (y *YumManager) Update(name string) error {
	cmd := exec.Command("yum", "update", "-y", name)
	return cmd.Run()
}

func (y *YumManager) UpdateAll() error {
	return exec.Command("yum", "update", "-y").Run()
}

func (y *YumManager) IsInstalled(name string) (bool, string, error) {
	cmd := exec.Command("rpm", "-q", name)
	output, err := cmd.Output()
	if err != nil {
		return false, "", nil // Package not installed
	}

	outputStr := strings.TrimSpace(string(output))
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
	cmd := exec.Command("yum", "search", query)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var packages []PackageInfo
	lines := strings.Split(string(output), "\n")
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
	cmd := exec.Command("rpm", "-qa", "--queryformat", "%{NAME} %{VERSION}-%{RELEASE} %{ARCH}\n")
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
	return exec.Command("yum", "clean", "all").Run()
}

// BrewManager implements package management for Homebrew (macOS)
type BrewManager struct{}

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
type GenericPackageManager struct{}

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
	return result, fmt.Errorf(message)
}

// Placeholder implementations for other package managers
type DnfManager struct{ YumManager }
type PacmanManager struct{ GenericPackageManager }
type ZypperManager struct{ GenericPackageManager }
type ChocolateyManager struct{ GenericPackageManager }
