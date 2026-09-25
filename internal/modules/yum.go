package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// YumModule manages packages on RedHat/CentOS/Fedora systems
type YumModule struct {
	*BaseModule
}

// NewYumModule creates a new yum module
func NewYumModule() *YumModule {
	return &YumModule{
		BaseModule: NewBaseModule("yum"),
	}
}

func (m *YumModule) GetDescription() string {
	return "Manage packages on RedHat/CentOS/Fedora systems using yum"
}

// PreCheckState checks if packages are already in the desired state
func (m *YumModule) PreCheckState(ctx context.Context, host types.Host, args map[string]interface{}) (*PreCheckResult, error) {
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

	// Check current package installation status using rpm (fast: ~50ms)
	currentState := make(map[string]interface{})
	allCorrect := true

	for _, pkgName := range pkgNames {
		_, err := runOnHost(ctx, host, args, "rpm", "-q", pkgName)
		isInstalled := err == nil

		currentState[pkgName] = isInstalled

		// Check if desired state matches current state
		// "latest" cannot be decided without asking yum, so it always runs
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

func (m *YumModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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

	// Pre-check: if state is already correct, skip execution
	preCheck, err := m.PreCheckState(ctx, host, args)
	if err == nil && preCheck.IsStateCorrect {
		result.Output["state"] = args["state"]
		result.Output["packages"] = args["name"]
		result.Output["msg"] = preCheck.Reason
		result.Output["pre_checked"] = true
		result.Duration = time.Since(startTime)
		return result, nil
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

	enableRepoList := ""
	if enableVal, exists := args["enablerepo"]; exists {
		if enableStr, ok := enableVal.(string); ok {
			enableRepoList = enableStr
		}
	}

	disableRepoList := ""
	if disableVal, exists := args["disablerepo"]; exists {
		if disableStr, ok := disableVal.(string); ok {
			disableRepoList = disableStr
		}
	}

	security := false
	if secVal, exists := args["security"]; exists {
		if secBool, ok := secVal.(bool); ok {
			security = secBool
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

	// Update cache if requested
	if updateCache {
		if err := m.updateYumCache(ctx, host, args); err != nil {
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
			changed, err := m.installPackages(ctx, host, args, pkgNames, state == "latest", enableRepoList, disableRepoList, security)
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
			result.Changed = true
		}
	}

	result.Output["state"] = state
	result.Output["packages"] = pkgNames
	result.Output["msg"] = "Package operation completed"

	result.Duration = time.Since(startTime)
	return result, nil
}

// yumRun runs yum on the target host
func yumRun(ctx context.Context, host types.Host, args map[string]interface{}, yumArgs ...string) (string, error) {
	out, err := runOnHost(ctx, host, args, append([]string{"yum"}, yumArgs...)...)
	if err != nil {
		return out, fmt.Errorf("yum %s failed: %w", yumArgs[0], err)
	}
	return out, nil
}

// updateYumCache refreshes metadata; check-update is not used because it exits 100
// whenever updates are available
func (m *YumModule) updateYumCache(ctx context.Context, host types.Host, args map[string]interface{}) error {
	_, err := yumRun(ctx, host, args, "makecache")
	return err
}

// installPackages installs packages (upgrading them for latest/security) and reports
// whether yum changed anything
func (m *YumModule) installPackages(ctx context.Context, host types.Host, args map[string]interface{}, packages []string, upgrade bool, enableRepo, disableRepo string, security bool) (bool, error) {
	var repoArgs []string
	if enableRepo != "" {
		repoArgs = append(repoArgs, "--enablerepo="+enableRepo)
	}
	if disableRepo != "" {
		repoArgs = append(repoArgs, "--disablerepo="+disableRepo)
	}

	var runs [][]string
	switch {
	case security:
		runs = append(runs, append([]string{"update", "-y", "--security"}, repoArgs...))
	case upgrade:
		// update is a no-op for packages that are not installed yet
		runs = append(runs, append(append([]string{"install", "-y"}, repoArgs...), packages...))
		runs = append(runs, append(append([]string{"update", "-y"}, repoArgs...), packages...))
	default:
		runs = append(runs, append(append([]string{"install", "-y"}, repoArgs...), packages...))
	}

	changed := false
	for _, run := range runs {
		out, err := yumRun(ctx, host, args, run...)
		if err != nil {
			return changed, err
		}
		if !strings.Contains(out, "Nothing to do") {
			changed = true
		}
	}
	return changed, nil
}

func (m *YumModule) removePackages(ctx context.Context, host types.Host, args map[string]interface{}, packages []string) error {
	_, err := yumRun(ctx, host, args, append([]string{"remove", "-y"}, packages...)...)
	return err
}

func (m *YumModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}
	return validatePackageState(args)
}
