package modules

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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

func (m *AptModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  args["name"].(string),
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

	// Update cache if requested
	if updateCache {
		if err := m.updateAptCache(ctx); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to update cache: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		result.Changed = true
	}

	// Handle package operations
	if len(pkgNames) > 0 {
		if state == "present" || state == "latest" {
			if err := m.installPackages(ctx, pkgNames, state == "latest"); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to install packages: %v", err)
				result.Duration = time.Since(startTime)
				return result, nil
			}
			result.Changed = true
		} else if state == "absent" {
			if err := m.removePackages(ctx, pkgNames); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to remove packages: %v", err)
				result.Duration = time.Since(startTime)
				return result, nil
			}
			result.Changed = true
		}
	}

	// Autoremove if requested
	if autoremove {
		if err := m.autoremovePackages(ctx); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to autoremove: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		result.Changed = true
	}

	// Autoclean if requested
	if autoclean {
		if err := m.autocleanPackages(ctx); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to autoclean: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		result.Changed = true
	}

	result.Output["state"] = state
	result.Output["packages"] = pkgNames
	result.Output["msg"] = fmt.Sprintf("Package operation completed")

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *AptModule) updateAptCache(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "apt-get", "update")
	return cmd.Run()
}

func (m *AptModule) installPackages(ctx context.Context, packages []string, upgrade bool) error {
	args := []string{"install", "-y"}
	if upgrade {
		args = []string{"install", "-y", "--only-upgrade"}
	}
	args = append(args, packages...)

	cmd := exec.CommandContext(ctx, "apt-get", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apt-get failed: %s", stderr.String())
	}
	return nil
}

func (m *AptModule) removePackages(ctx context.Context, packages []string) error {
	args := []string{"remove", "-y"}
	args = append(args, packages...)

	cmd := exec.CommandContext(ctx, "apt-get", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apt-get failed: %s", stderr.String())
	}
	return nil
}

func (m *AptModule) autoremovePackages(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "apt-get", "autoremove", "-y")
	return cmd.Run()
}

func (m *AptModule) autocleanPackages(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "apt-get", "autoclean")
	return cmd.Run()
}

func (m *AptModule) Validate(args map[string]interface{}) error {
	return m.BaseModule.Validate(args)
}
