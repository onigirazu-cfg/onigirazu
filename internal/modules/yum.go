package modules

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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

func (m *YumModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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
		if err := m.updateYumCache(ctx); err != nil {
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
			if err := m.installPackages(ctx, pkgNames, state == "latest", enableRepoList, disableRepoList, security); err != nil {
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

	result.Output["state"] = state
	result.Output["packages"] = pkgNames
	result.Output["msg"] = fmt.Sprintf("Package operation completed")

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *YumModule) updateYumCache(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "yum", "check-update")
	return cmd.Run()
}

func (m *YumModule) installPackages(ctx context.Context, packages []string, upgrade bool, enableRepo, disableRepo string, security bool) error {
	args := []string{"install", "-y"}

	// Add repo options
	if enableRepo != "" {
		args = append(args, "--enablerepo="+enableRepo)
	}
	if disableRepo != "" {
		args = append(args, "--disablerepo="+disableRepo)
	}

	// Handle security updates
	if security {
		args = []string{"update", "-y", "--security"}
		if enableRepo != "" {
			args = append(args, "--enablerepo="+enableRepo)
		}
		if disableRepo != "" {
			args = append(args, "--disablerepo="+disableRepo)
		}
	} else if upgrade {
		args = []string{"update", "-y"}
		if enableRepo != "" {
			args = append(args, "--enablerepo="+enableRepo)
		}
		if disableRepo != "" {
			args = append(args, "--disablerepo="+disableRepo)
		}
		args = append(args, packages...)
	} else {
		args = append(args, packages...)
	}

	cmd := exec.CommandContext(ctx, "yum", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yum failed: %s", stderr.String())
	}
	return nil
}

func (m *YumModule) removePackages(ctx context.Context, packages []string) error {
	args := []string{"remove", "-y"}
	args = append(args, packages...)

	cmd := exec.CommandContext(ctx, "yum", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yum failed: %s", stderr.String())
	}
	return nil
}

func (m *YumModule) Validate(args map[string]interface{}) error {
	return m.BaseModule.Validate(args)
}
