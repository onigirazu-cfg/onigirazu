package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// SysvinitModule manages a SysV init service (Ansible's sysvinit): name,
// state started/stopped/restarted/reloaded, enabled, runlevels, pattern
type SysvinitModule struct{ *BaseModule }

func NewSysvinitModule() *SysvinitModule {
	return &SysvinitModule{BaseModule: NewBaseModule("sysvinit")}
}

func (m *SysvinitModule) GetDescription() string { return "Manage a SysV init service" }

func (m *SysvinitModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	name := getStringArg(args, "name", "")
	if name == "" {
		return fail("sysvinit requires 'name'")
	}
	script := "/etc/init.d/" + name
	state := getStringArg(args, "state", "")
	// running? the script's status, or a process matching pattern
	check := shellQuote(script) + " status >/dev/null 2>&1"
	if p := getStringArg(args, "pattern", ""); p != "" {
		check = "pgrep -f " + shellQuote(p) + " >/dev/null 2>&1"
	}
	out, _ := runShellOnHost(ctx, host, args, check+" && echo running || echo stopped")
	running := strings.TrimSpace(out) == "running"
	result.Output["before"] = map[bool]string{true: "running", false: "stopped"}[running]
	run := func(action string) error {
		if inCheckMode(args) {
			return nil
		}
		_, err := runShellOnHost(ctx, host, args, shellQuote(script)+" "+action)
		return err
	}
	switch state {
	case "started":
		if !running {
			result.Changed = true
			if err := run("start"); err != nil {
				return fail("sysvinit: " + err.Error())
			}
		}
	case "stopped":
		if running {
			result.Changed = true
			if err := run("stop"); err != nil {
				return fail("sysvinit: " + err.Error())
			}
		}
	case "restarted", "reloaded":
		result.Changed = true
		if err := run(strings.TrimSuffix(state, "ed")); err != nil {
			return fail("sysvinit: " + err.Error())
		}
	case "":
	default:
		return fail(fmt.Sprintf("sysvinit: state %q", state))
	}
	if v, set := args["enabled"]; set {
		want, _ := v.(bool)
		have, _ := runShellOnHost(ctx, host, args, "ls /etc/rc[2-5].d/S*"+name+" >/dev/null 2>&1 && echo yes || (chkconfig --list "+shellQuote(name)+" 2>/dev/null | grep -q ':on' && echo yes || echo no)")
		enabled := strings.TrimSpace(have) == "yes"
		if enabled != want {
			result.Changed = true
			if !inCheckMode(args) {
				cmd := map[bool]string{true: "update-rc.d " + shellQuote(name) + " defaults 2>/dev/null || chkconfig " + shellQuote(name) + " on", false: "update-rc.d -f " + shellQuote(name) + " remove 2>/dev/null || chkconfig " + shellQuote(name) + " off"}[want]
				if _, err := runShellOnHost(ctx, host, args, cmd); err != nil {
					return fail("sysvinit: enable: " + err.Error())
				}
			}
		}
	}
	result.Duration = time.Since(start)
	return result, nil
}
