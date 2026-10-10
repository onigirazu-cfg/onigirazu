package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// PipxModule manages Python applications with pipx (community.general.pipx):
// name, state present/absent/latest/install/uninstall/upgrade/reinstall,
// source (a spec such as "black==24.1" or a git URL), executable,
// install_deps, force, python, inject_packages (into an installed app)
type PipxModule struct {
	*BaseModule
}

// NewPipxModule creates the pipx module
func NewPipxModule() *PipxModule { return &PipxModule{BaseModule: NewBaseModule("pipx")} }

func (m *PipxModule) GetDescription() string { return "Manage Python applications with pipx" }

func (m *PipxModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "name", "") == "" {
		return fmt.Errorf("pipx requires 'name'")
	}
	switch getStringArg(args, "state", "install") {
	case "present", "install", "absent", "uninstall", "latest", "upgrade", "reinstall", "inject":
		return nil
	}
	return fmt.Errorf("pipx: state must be present, absent, latest, reinstall or inject")
}

func (m *PipxModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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
	name := getStringArg(args, "name", "")
	state := getStringArg(args, "state", "install")
	pipx := getStringArg(args, "executable", "")
	if pipx == "" {
		pipx = "$(command -v pipx || echo pipx)"
	} else {
		pipx = shellQuote(pipx)
	}
	// what pipx has: name -> version, from its JSON listing
	listed, _ := runShellOnHost(ctx, host, args, pipx+" list --json 2>/dev/null")
	installed := map[string]string{}
	var listing struct {
		Venvs map[string]struct {
			Metadata struct {
				MainPackage struct {
					Version string `json:"package_version"`
				} `json:"main_package"`
			} `json:"metadata"`
		} `json:"venvs"`
	}
	if json.Unmarshal([]byte(strings.TrimSpace(listed)), &listing) == nil {
		for venv, v := range listing.Venvs {
			installed[normalizePip(venv)] = v.Metadata.MainPackage.Version
		}
	}
	current, have := installed[normalizePip(name)]
	result.Output["version"] = current
	var cmd, msg string
	flags := ""
	if getBoolArg(args, "force", false) {
		flags += " --force"
	}
	if getBoolArg(args, "install_deps", false) {
		flags += " --include-deps"
	}
	if py := getStringArg(args, "python", ""); py != "" {
		flags += " --python " + shellQuote(py)
	}
	source := getStringArg(args, "source", name)
	switch state {
	case "present", "install":
		if have && !getBoolArg(args, "force", false) {
			break
		}
		cmd, msg = pipx+" install"+flags+" "+shellQuote(source), "installed "+name
	case "absent", "uninstall":
		if have {
			cmd, msg = pipx+" uninstall "+shellQuote(name), "removed "+name
		}
	case "latest", "upgrade":
		if !have {
			cmd, msg = pipx+" install"+flags+" "+shellQuote(source), "installed "+name
		} else {
			cmd, msg = pipx+" upgrade "+shellQuote(name), "upgraded "+name
		}
	case "reinstall":
		cmd, msg = pipx+" reinstall "+shellQuote(name), "reinstalled "+name
	case "inject":
		pkgs := listArg(args, "inject_packages")
		if len(pkgs) == 0 {
			return fail("pipx: inject needs inject_packages")
		}
		quoted := make([]string, len(pkgs))
		for i, p := range pkgs {
			quoted[i] = shellQuote(p)
		}
		cmd, msg = pipx+" inject "+shellQuote(name)+" "+strings.Join(quoted, " "), "injected into "+name
	}
	if cmd == "" {
		result.Output["msg"] = name + " is as wanted"
		result.Duration = time.Since(start)
		return result, nil
	}
	if inCheckMode(args) {
		result.Changed = true
		result.Output["msg"] = "would have " + msg
		result.Duration = time.Since(start)
		return result, nil
	}
	out, err := runShellOnHost(ctx, host, args, cmd+" 2>&1")
	if err != nil {
		return fail(fmt.Sprintf("pipx: %v: %s", err, strings.TrimSpace(out)))
	}
	// upgrade says so when nothing changed
	if state == "latest" || state == "upgrade" {
		if strings.Contains(out, "is already at latest version") {
			result.Output["msg"] = name + " is at its latest version"
			result.Duration = time.Since(start)
			return result, nil
		}
	}
	result.Changed = true
	result.Output["msg"] = msg
	result.Duration = time.Since(start)
	return result, nil
}
