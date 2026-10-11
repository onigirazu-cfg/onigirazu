package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// FlatpakModule manages Flatpak applications (community.general.flatpak):
// name (an application id, a list, or a .flatpakref URL), state
// present/absent/latest, remote (flathub), method system/user, executable,
// no_dependencies
type FlatpakModule struct {
	*BaseModule
}

// NewFlatpakModule creates the flatpak module
func NewFlatpakModule() *FlatpakModule { return &FlatpakModule{BaseModule: NewBaseModule("flatpak")} }

func (m *FlatpakModule) GetDescription() string { return "Manage Flatpak applications" }

func (m *FlatpakModule) Validate(args map[string]interface{}) error {
	if len(packageNames(args)) == 0 {
		return fmt.Errorf("flatpak requires 'name'")
	}
	switch getStringArg(args, "state", "present") {
	case "present", "absent", "latest":
	default:
		return fmt.Errorf("flatpak: state must be present, absent or latest")
	}
	switch getStringArg(args, "method", "system") {
	case "system", "user":
		return nil
	}
	return fmt.Errorf("flatpak: method must be system or user")
}

func (m *FlatpakModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name,
		Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	if err := m.Validate(args); err != nil {
		return fail(err.Error())
	}
	state := getStringArg(args, "state", "present")
	flatpak := getStringArg(args, "executable", "flatpak")
	scope := "--" + getStringArg(args, "method", "system")
	remote := getStringArg(args, "remote", "flathub")
	listed, err := runShellOnHost(ctx, host, args, shellQuote(flatpak)+" list "+scope+" --app --columns=application 2>/dev/null")
	if err != nil {
		return fail(fmt.Sprintf("flatpak list failed: %v", err))
	}
	installed := map[string]bool{}
	for _, l := range strings.Split(listed, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			installed[l] = true
		}
	}
	// a .flatpakref names its application in the file; we take the last
	// path element without the suffix as the id
	appID := func(name string) string {
		if strings.HasSuffix(name, ".flatpakref") {
			parts := strings.Split(strings.TrimSuffix(name, ".flatpakref"), "/")
			return parts[len(parts)-1]
		}
		return name
	}
	var install, remove, upgrade []string
	for _, name := range packageNames(args) {
		id := appID(name)
		switch state {
		case "present":
			if !installed[id] {
				install = append(install, name)
			}
		case "absent":
			if installed[id] {
				remove = append(remove, id)
			}
		case "latest":
			if !installed[id] {
				install = append(install, name)
			} else {
				upgrade = append(upgrade, id)
			}
		}
	}
	result.Output["install"], result.Output["remove"], result.Output["upgrade"] = install, remove, upgrade
	if len(install)+len(remove)+len(upgrade) == 0 {
		result.Duration = time.Since(start)
		return result, nil
	}
	if inCheckMode(args) {
		result.Changed = len(install)+len(remove) > 0
		result.Output["msg"] = "would change"
		result.Duration = time.Since(start)
		return result, nil
	}
	noDeps := ""
	if getBoolArg(args, "no_dependencies", false) {
		noDeps = " --no-deps"
	}
	run := func(verb string, items []string, withRemote bool) error {
		if len(items) == 0 {
			return nil
		}
		quoted := make([]string, len(items))
		for i, it := range items {
			quoted[i] = shellQuote(it)
		}
		cmd := shellQuote(flatpak) + " " + verb + " " + scope + " -y --noninteractive" + noDeps
		if withRemote {
			// a .flatpakref or an URL carries its own remote
			plain := true
			for _, it := range items {
				if strings.Contains(it, "://") || strings.HasSuffix(it, ".flatpakref") {
					plain = false
				}
			}
			if plain {
				cmd += " " + shellQuote(remote)
			}
		}
		out, err := runShellOnHost(ctx, host, args, cmd+" "+strings.Join(quoted, " ")+" 2>&1")
		if err != nil {
			return fmt.Errorf("flatpak %s: %v: %s", verb, err, strings.TrimSpace(out))
		}
		if verb == "update" && !strings.Contains(out, "Updating") && !strings.Contains(out, "Installing") {
			return nil
		}
		result.Changed = true
		return nil
	}
	for _, step := range []struct {
		verb       string
		items      []string
		withRemote bool
	}{{"install", install, true}, {"uninstall", remove, false}, {"update", upgrade, false}} {
		if err := run(step.verb, step.items, step.withRemote); err != nil {
			return fail(err.Error())
		}
	}
	result.Duration = time.Since(start)
	return result, nil
}
