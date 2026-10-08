package modules

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Windows modules run over WinRM (ansible_connection: winrm)

// windowsSafe are the modules that run on WinRM hosts besides win_*: they
// work on the control machine or only on variables
var windowsSafe = map[string]bool{
	"debug": true, "set_fact": true, "assert": true, "fail": true, "meta": true, "include_vars": true,
	"pause": true, "async_status": true, "setup": true, "gather_facts": true,
	"add_host": true, "group_by": true,
}

// winModuleError explains a Linux module on a Windows host
func winModuleError(module string) string {
	return fmt.Sprintf("module %s does not run on Windows hosts (WinRM): use its win_ counterpart "+
		"or allow it through the Ansible bridge", module)
}

// psQuote is s as a single-quoted PowerShell string
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// winEnvPrefix sets the task's environment in PowerShell
func winEnvPrefix(host types.Host) string {
	keys := make([]string, 0, len(host.Environment))
	for k := range host.Environment {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "[Environment]::SetEnvironmentVariable(%s, %s)\n", psQuote(k), psQuote(host.Environment[k]))
	}
	return b.String()
}

func winClient(host types.Host, args map[string]interface{}) (winrm.Runner, error) {
	if getBoolArg(args, "_become", false) {
		return nil, fmt.Errorf("become is not supported on WinRM hosts yet")
	}
	return winrm.For(host)
}

func lines(s string) []interface{} {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSuffix(s, "\n")
	out := []interface{}{}
	if s == "" {
		return out
	}
	for _, l := range strings.Split(s, "\n") {
		out = append(out, l)
	}
	return out
}

// winSkip reports creates/removes that make a command unnecessary
func winSkip(ctx context.Context, c winrm.Runner, args map[string]interface{}) (bool, string, error) {
	for _, key := range []string{"creates", "removes"} {
		path := getStringArg(args, key, "")
		if path == "" {
			continue
		}
		res, err := c.RunPS(ctx, "if (Test-Path -LiteralPath "+psQuote(path)+") { 'yes' } else { 'no' }")
		if err != nil {
			return false, "", err
		}
		exists := strings.TrimSpace(res.Stdout) == "yes"
		if key == "creates" && exists {
			return true, fmt.Sprintf("skipped, since %s exists", path), nil
		}
		if key == "removes" && !exists {
			return true, fmt.Sprintf("skipped, since %s does not exist", path), nil
		}
	}
	return false, "", nil
}

// WinPingModule checks that PowerShell runs on the host
type WinPingModule struct{ *BaseModule }

// NewWinPingModule creates win_ping
func NewWinPingModule() *WinPingModule { return &WinPingModule{NewBaseModule("win_ping")} }

func (m *WinPingModule) GetDescription() string {
	return "Check the WinRM connection to a Windows host"
}

func (m *WinPingModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	res := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start,
		Output: map[string]interface{}{}}
	c, err := winClient(host, args)
	if err == nil {
		data := getStringArg(args, "data", "pong")
		var out winrm.Result
		out, err = c.RunPS(ctx, "Write-Output "+psQuote(data))
		if err == nil && out.ExitCode != 0 {
			err = fmt.Errorf("%s", strings.TrimSpace(out.Stderr))
		}
		if err == nil {
			res.Output["ping"] = strings.TrimSpace(out.Stdout)
		}
	}
	if err != nil {
		res.Failed, res.Error = true, err.Error()
	} else {
		res.Success = true
	}
	res.Duration = time.Since(start)
	return res, nil
}

// WinCommandModule runs a command (win_command) or a PowerShell/cmd script
// (win_shell)
type WinCommandModule struct {
	*BaseModule
	shell bool
}

// NewWinCommandModule creates win_command
func NewWinCommandModule() *WinCommandModule {
	return &WinCommandModule{BaseModule: NewBaseModule("win_command")}
}

// NewWinShellModule creates win_shell
func NewWinShellModule() *WinCommandModule {
	return &WinCommandModule{BaseModule: NewBaseModule("win_shell"), shell: true}
}

func (m *WinCommandModule) GetDescription() string {
	if m.shell {
		return "Run a PowerShell (or cmd) script on a Windows host"
	}
	return "Run a command on a Windows host"
}

func (m *WinCommandModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "cmd", getStringArg(args, "_raw_params", "")) == "" {
		return fmt.Errorf("%s needs a command", m.name)
	}
	return nil
}

func (m *WinCommandModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	res := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start,
		Output: map[string]interface{}{}}
	fail := func(err error) (types.TaskResult, error) {
		res.Failed, res.Error = true, err.Error()
		res.Duration = time.Since(start)
		return res, nil
	}
	if err := m.Validate(args); err != nil {
		return fail(err)
	}
	cmd := getStringArg(args, "cmd", getStringArg(args, "_raw_params", ""))
	c, err := winClient(host, args)
	if err != nil {
		return fail(err)
	}
	skip, msg, err := winSkip(ctx, c, args)
	if err != nil {
		return fail(err)
	}
	if skip {
		res.Success = true
		res.Output["msg"] = msg
		res.Output["cmd"] = cmd
		res.Duration = time.Since(start)
		return res, nil
	}
	if inCheckMode(args) {
		res.Success, res.Skipped = true, true
		res.Output["msg"] = "skipped in check mode"
		return res, nil
	}

	out, err := m.run(ctx, c, host, args, cmd)
	if err != nil {
		return fail(err)
	}
	res.Changed = true
	res.Output["cmd"] = cmd
	res.Output["rc"] = out.ExitCode
	res.Output["stdout"] = out.Stdout
	res.Output["stderr"] = out.Stderr
	res.Output["stdout_lines"] = lines(out.Stdout)
	res.Output["stderr_lines"] = lines(out.Stderr)
	if out.ExitCode != 0 {
		res.Failed = true
		res.Error = fmt.Sprintf("non-zero return code %d", out.ExitCode)
		if s := strings.TrimSpace(out.Stderr); s != "" {
			res.Error += ": " + s
		}
	} else {
		res.Success = true
	}
	res.Duration = time.Since(start)
	return res, nil
}

// run runs cmd: win_shell as a PowerShell script (or through executable),
// win_command as a command line
func (m *WinCommandModule) run(ctx context.Context, c winrm.Runner, host types.Host, args map[string]interface{}, cmd string) (winrm.Result, error) {
	chdir := getStringArg(args, "chdir", "")
	stdin := getStringArg(args, "stdin", "")
	executable := getStringArg(args, "executable", "")
	isPS := strings.EqualFold(executable, "") || strings.EqualFold(executable, "powershell") ||
		strings.EqualFold(executable, "powershell.exe")
	if m.shell && isPS {
		script := cmd
		if stdin != "" {
			script = fmt.Sprintf("%s | & {\n%s\n}", psQuote(stdin), script)
		}
		if chdir != "" {
			script = "Set-Location -LiteralPath " + psQuote(chdir) + "\n" + script
		}
		return c.RunPS(ctx, winEnvPrefix(host)+script)
	}
	line := cmd
	if m.shell {
		line = executable + " /c " + cmd
	}
	var prefix strings.Builder
	keys := make([]string, 0, len(host.Environment))
	for k := range host.Environment {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&prefix, "set \"%s=%s\" && ", k, host.Environment[k])
	}
	if chdir != "" {
		fmt.Fprintf(&prefix, "cd /d \"%s\" && ", chdir)
	}
	if stdin != "" {
		return c.RunCmdInput(ctx, prefix.String()+line, stdin)
	}
	return c.RunCmd(ctx, prefix.String()+line)
}
