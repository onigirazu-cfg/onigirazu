package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// AptModule manages packages on Debian/Ubuntu systems
type AptModule struct {
	*BaseModule
}

// NewAptModule creates a new apt module
func NewAptModule() *AptModule {
	return &AptModule{
		BaseModule: NewBaseModule("apt"),
	}
}

func (m *AptModule) GetDescription() string {
	return "Manage packages on Debian/Ubuntu systems using apt"
}

// PreCheckState checks if packages are already in the desired state
// This enables idempotency by avoiding unnecessary apt-get calls
func (m *AptModule) PreCheckState(ctx context.Context, host types.Host, args map[string]interface{}) (*PreCheckResult, error) {
	// Get package names
	var pkgNames []string
	if nameVal, exists := args["name"]; exists {
		switch v := nameVal.(type) {
		case string:
			if v != "" {
				pkgNames = []string{v}
			}
		case []interface{}:
			for _, pkg := range v {
				if pkgStr, ok := pkg.(string); ok {
					pkgNames = append(pkgNames, pkgStr)
				}
			}
		}
	}

	state := "present"
	if stateVal, exists := args["state"]; exists {
		if stateStr, ok := stateVal.(string); ok {
			state = stateStr
		}
	}

	// If no packages specified, execute anyway
	if len(pkgNames) == 0 {
		return &PreCheckResult{
			ShouldExecute: true,
			Reason:        "No packages specified",
		}, nil
	}

	// Check current package installation status using dpkg (fast: ~100ms)
	// This is much faster than apt-get (~5000ms)
	currentState := make(map[string]interface{})
	allCorrect := true

	for _, pkgName := range pkgNames {
		isInstalled := debPackageInstalled(ctx, host, args, pkgName)
		currentState[pkgName] = isInstalled

		// "latest" cannot be decided without asking apt, so it always runs
		if state == "latest" || (state == "present" && !isInstalled) || (state == "absent" && isInstalled) {
			allCorrect = false
		}
	}

	if allCorrect {
		// State is already correct - skip execution
		return &PreCheckResult{
			IsStateCorrect: true,
			ShouldExecute:  false,
			Reason:         fmt.Sprintf("Packages already in state: %s", state),
			CurrentState:   currentState,
		}, nil
	}

	// State needs to change - execute the operation
	return &PreCheckResult{
		IsStateCorrect: false,
		ShouldExecute:  true,
		Reason:         fmt.Sprintf("Packages need to be set to state: %s", state),
		CurrentState:   currentState,
	}, nil
}

func (m *AptModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  taskName(args),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
	}

	// Get parameters
	state := "present"
	if stateVal, exists := args["state"]; exists {
		if stateStr, ok := stateVal.(string); ok {
			state = stateStr
		}
	}

	updateCache := false
	if updateVal, exists := args["update_cache"]; exists {
		if updateBool, ok := updateVal.(bool); ok {
			updateCache = updateBool
		}
	}

	autoremove := false
	if autoremoveVal, exists := args["autoremove"]; exists {
		if autoremoveBool, ok := autoremoveVal.(bool); ok {
			autoremove = autoremoveBool
		}
	}

	autoclean := false
	if autocleanVal, exists := args["autoclean"]; exists {
		if autocleanBool, ok := autocleanVal.(bool); ok {
			autoclean = autocleanBool
		}
	}

	// Get package names
	pkgNames := []string{}
	if nameVal, exists := args["name"]; exists {
		switch v := nameVal.(type) {
		case string:
			if v != "" {
				pkgNames = []string{v}
			}
		case []interface{}:
			for _, pkg := range v {
				if pkgStr, ok := pkg.(string); ok {
					pkgNames = append(pkgNames, pkgStr)
				}
			}
		}
	}

	// ✅ IDEMPOTENCY CHECK: Pre-check if packages are already in desired state
	if len(pkgNames) > 0 && (state == "present" || state == "absent" || state == "latest") {
		preCheck, err := m.PreCheckState(ctx, host, args)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("pre-check failed: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		// If state is already correct, skip execution
		if preCheck.IsStateCorrect {
			result.Success = true
			result.Changed = false // ✅ IMPORTANT: No changes needed!
			result.Output["state"] = state
			result.Output["packages"] = pkgNames
			result.Output["msg"] = preCheck.Reason
			result.Duration = time.Since(startTime)
			return result, nil
		}
	}

	// Update cache if requested
	if updateCache {
		if err := m.updateAptCache(ctx, host, args); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to update cache: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		// Refreshing package lists changes no configuration of the host
		result.Output["cache_updated"] = true
	}

	// Handle package operations
	if len(pkgNames) > 0 {
		if state == "present" || state == "latest" {
			changed, err := m.installPackages(ctx, host, args, pkgNames)
			if err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to install packages: %v", err)
				result.Duration = time.Since(startTime)
				return result, nil
			}
			result.Changed = result.Changed || changed
		} else if state == "absent" {
			if err := m.removePackages(ctx, host, args, pkgNames); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to remove packages: %v", err)
				result.Duration = time.Since(startTime)
				return result, nil
			}
			result.Changed = true // ✅ CORRECT: We actually made changes
		}
	}

	// Autoremove if requested
	if autoremove {
		if err := m.autoremovePackages(ctx, host, args); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to autoremove: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		result.Changed = true
	}

	// Autoclean if requested
	if autoclean {
		if err := m.autocleanPackages(ctx, host, args); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to autoclean: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		result.Changed = true
	}

	result.Output["state"] = state
	result.Output["packages"] = pkgNames
	result.Output["msg"] = "Package operation completed"

	result.Duration = time.Since(startTime)
	return result, nil
}

// aptGet runs apt-get non-interactively on the target host
func aptGet(ctx context.Context, host types.Host, args map[string]interface{}, aptArgs ...string) (string, error) {
	argv := append([]string{"env", "DEBIAN_FRONTEND=noninteractive", "apt-get"}, aptArgs...)
	out, err := runOnHost(ctx, host, args, argv...)
	if err != nil {
		return out, fmt.Errorf("apt-get %s failed: %w", aptArgs[0], err)
	}
	return out, nil
}

// debPackageInstalled reports whether a package is fully installed on the host
// (dpkg -l also lists removed packages that left config files behind)
func debPackageInstalled(ctx context.Context, host types.Host, args map[string]interface{}, pkg string) bool {
	out, err := runOnHost(ctx, host, args, "dpkg-query", "-W", "-f=${Status}", pkg)
	return err == nil && strings.TrimSpace(out) == "install ok installed"
}

func (m *AptModule) updateAptCache(ctx context.Context, host types.Host, args map[string]interface{}) error {
	_, err := aptGet(ctx, host, args, "update")
	return err
}

// installPackages installs packages or upgrades them to the latest version and
// reports whether apt changed anything
func (m *AptModule) installPackages(ctx context.Context, host types.Host, args map[string]interface{}, packages []string) (bool, error) {
	out, err := aptGet(ctx, host, args, append([]string{"install", "-y"}, packages...)...)
	if err != nil {
		return false, err
	}
	return !strings.Contains(out, "0 upgraded, 0 newly installed"), nil
}

func (m *AptModule) removePackages(ctx context.Context, host types.Host, args map[string]interface{}, packages []string) error {
	_, err := aptGet(ctx, host, args, append([]string{"remove", "-y"}, packages...)...)
	return err
}

func (m *AptModule) autoremovePackages(ctx context.Context, host types.Host, args map[string]interface{}) error {
	_, err := aptGet(ctx, host, args, "autoremove", "-y")
	return err
}

func (m *AptModule) autocleanPackages(ctx context.Context, host types.Host, args map[string]interface{}) error {
	_, err := aptGet(ctx, host, args, "autoclean")
	return err
}

func (m *AptModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}
	return validatePackageState(args)
}
