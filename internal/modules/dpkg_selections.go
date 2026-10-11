package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// DpkgSelectionsModule sets a package's dpkg selection (Ansible's
// dpkg_selections): name, selection install/hold/deinstall/purge
type DpkgSelectionsModule struct{ *BaseModule }

func NewDpkgSelectionsModule() *DpkgSelectionsModule {
	return &DpkgSelectionsModule{BaseModule: NewBaseModule("dpkg_selections")}
}

func (m *DpkgSelectionsModule) GetDescription() string {
	return "Set a package's dpkg selection (hold, install, ...)"
}

func (m *DpkgSelectionsModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "name", "") == "" {
		return fmt.Errorf("dpkg_selections requires 'name'")
	}
	switch getStringArg(args, "selection", "") {
	case "install", "hold", "deinstall", "purge":
		return nil
	}
	return fmt.Errorf("dpkg_selections: selection must be install, hold, deinstall or purge")
}

func (m *DpkgSelectionsModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	if err := m.Validate(args); err != nil {
		result.Success, result.Error = false, err.Error()
		return result, nil
	}
	name, want := getStringArg(args, "name", ""), getStringArg(args, "selection", "")
	out, _ := runShellOnHost(ctx, host, args, "dpkg --get-selections "+shellQuote(name)+" 2>/dev/null | awk '{print $2}'")
	current := strings.TrimSpace(out)
	result.Output["before"] = current
	if current == want {
		result.Duration = time.Since(start)
		return result, nil
	}
	result.Changed = true
	if !inCheckMode(args) {
		if _, err := runShellOnHost(ctx, host, args, "printf '%s %s\\n' "+shellQuote(name)+" "+shellQuote(want)+" | dpkg --set-selections"); err != nil {
			result.Success, result.Error = false, "dpkg_selections: "+err.Error()
			return result, nil
		}
	}
	result.Output["after"] = want
	result.Duration = time.Since(start)
	return result, nil
}
