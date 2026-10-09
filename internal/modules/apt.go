package modules

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
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

	// a failed query counts as not installed: the task then runs
	installed, _ := installedPackages(ctx, host, args, pkgNames)
	for _, pkgName := range pkgNames {
		isInstalled := installed[pkgName]
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

	updateCache := getBoolArg(args, "update_cache", false)

	autoremove := getBoolArg(args, "autoremove", false)

	autoclean := getBoolArg(args, "autoclean", false)

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
		if inCheckMode(args) {
			result.Success = true
			result.Changed = true
			result.Output["state"] = state
			result.Output["packages"] = pkgNames
			result.Output["msg"] = "would change: " + preCheck.Reason
			result.Duration = time.Since(startTime)
			return result, nil
		}
	}
	upgrade, err := aptUpgradeArg(args)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, nil
	}
	if inCheckMode(args) && upgrade != "" {
		// apt-get -s shows what the upgrade would do without doing it
		out, err := aptGet(ctx, host, args, "-s", upgrade)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Changed = aptChanged(out)
			result.Output["msg"] = "check mode: " + aptSummary(out)
		}
		result.Duration = time.Since(startTime)
		return result, nil
	}
	if inCheckMode(args) {
		// cache updates and cleanups are not predicted
		result.Success = true
		result.Output["msg"] = "check mode: nothing done for this apt operation"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Update cache if requested and older than cache_valid_time seconds
	if updateCache && m.cacheValid(ctx, host, args) {
		result.Output["cache_updated"] = false
	} else if updateCache {
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

	if upgrade != "" {
		out, err := aptGet(ctx, host, args, "-y", "-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold", upgrade)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to upgrade: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		result.Changed = result.Changed || aptChanged(out)
		result.Output["upgrade"] = aptSummary(out)
	}

	// Autoremove if requested
	if autoremove {
		out, err := m.autoremovePackages(ctx, host, args)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to autoremove: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		result.Changed = result.Changed || aptChanged(out)
	}

	// Autoclean if requested
	if autoclean {
		out, err := m.autocleanPackages(ctx, host, args)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to autoclean: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		// a change is a removed archive: apt-get prints "Del <pkg> ..."
		result.Changed = result.Changed || autocleanChanged(out)
	}

	result.Output["state"] = state
	result.Output["packages"] = pkgNames
	result.Output["msg"] = "Package operation completed"

	result.Duration = time.Since(startTime)
	return result, nil
}

// aptGet runs apt-get non-interactively on the target host
func aptGet(ctx context.Context, host types.Host, args map[string]interface{}, aptArgs ...string) (string, error) {
	argv := []string{"env", "DEBIAN_FRONTEND=noninteractive", "apt-get"}
	// lock_timeout: wait that long for the dpkg lock (apt 1.9.11+); 60 s by
	// default as in Ansible (apt-daily holds it after a boot)
	if n := getIntArg(args, "lock_timeout", 60); n > 0 {
		argv = append(argv, "-o", "DPkg::Lock::Timeout="+strconv.Itoa(n))
	}
	argv = append(argv, aptArgs...)
	out, err := runOnHost(ctx, host, args, argv...)
	if err != nil {
		return out, fmt.Errorf("apt-get %s failed: %w", aptArgs[0], err)
	}
	return out, nil
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

func (m *AptModule) autoremovePackages(ctx context.Context, host types.Host, args map[string]interface{}) (string, error) {
	return aptGet(ctx, host, args, "autoremove", "-y")
}

// cacheValid tells whether the package lists were updated less than
// cache_valid_time seconds ago
func (m *AptModule) cacheValid(ctx context.Context, host types.Host, args map[string]interface{}) bool {
	validFor := getIntArg(args, "cache_valid_time", 0)
	if validFor <= 0 {
		return false
	}
	// the newest of the files apt-get update writes; none means no valid cache
	out, err := runOnHost(ctx, host, args, "sh", "-c",
		`t=$(stat -c %Y /var/cache/apt/pkgcache.bin /var/lib/apt/periodic/update-success-stamp 2>/dev/null | sort -n | tail -1); [ -n "$t" ] && echo $(( $(date +%s) - t ))`)
	if err != nil {
		return false
	}
	age, err := strconv.Atoi(strings.TrimSpace(out))
	return err == nil && age >= 0 && age < validFor
}

// aptUpgradeArg maps the upgrade argument to an apt-get command
func aptUpgradeArg(args map[string]interface{}) (string, error) {
	v, ok := args["upgrade"]
	if !ok || v == nil {
		return "", nil
	}
	s := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
	switch s {
	case "no", "false", "":
		return "", nil
	case "yes", "true", "safe":
		return "upgrade", nil
	case "full", "dist":
		return "dist-upgrade", nil
	}
	return "", fmt.Errorf("upgrade: expected yes, safe, full, dist or no, got %q", s)
}

var aptCounts = regexp.MustCompile(`(\d+) upgraded, (\d+) newly installed, (\d+) to remove`)

// aptChanged tells from apt-get output whether packages were (or would be)
// upgraded, installed or removed
func aptChanged(out string) bool {
	m := aptCounts.FindStringSubmatch(out)
	return len(m) == 4 && (m[1] != "0" || m[2] != "0" || m[3] != "0")
}

func aptSummary(out string) string {
	if m := aptCounts.FindString(out); m != "" {
		return m
	}
	return "nothing to do"
}

func (m *AptModule) autocleanPackages(ctx context.Context, host types.Host, args map[string]interface{}) (string, error) {
	return aptGet(ctx, host, args, "autoclean")
}

// autocleanChanged tells whether apt-get autoclean removed an archive
func autocleanChanged(out string) bool {
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Del ") {
			return true
		}
	}
	return false
}

func (m *AptModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}
	return validatePackageState(args)
}
