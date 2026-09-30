package modules

import (
	"context"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm/winrmtest"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// fakeWindows answers PowerShell scripts and command lines like a small
// Windows host
func fakeWindows(t *testing.T) (types.Host, *winrmtest.Server) {
	f := &winrmtest.Server{Handle: func(command, stdin string) (string, string, int) {
		if script := winrmtest.Script(command, stdin); script != "" {
			switch {
			case strings.Contains(script, "Test-Path -LiteralPath 'C:\\exists'"):
				return "yes\r\n", "", 0
			case strings.Contains(script, "Test-Path"):
				return "no\r\n", "", 0
			case strings.HasPrefix(script, "Write-Output"):
				return strings.Trim(strings.TrimPrefix(script, "Write-Output "), "'") + "\r\n", "", 0
			case strings.Contains(script, "throw"):
				return "", "boom", 1
			}
			return "ps:" + script + "\r\nline2\r\n", "", 0
		}
		if strings.Contains(command, "exit 5") {
			return "", "bad\r\n", 5
		}
		return "cmd:" + command + "|stdin:" + stdin, "", 0
	}}
	addr, port := winrmtest.Start(t, f)
	return winrmtest.Host("w1", addr, port), f
}

func run(t *testing.T, m types.Module, host types.Host, args map[string]interface{}) types.TaskResult {
	t.Helper()
	res, err := m.Execute(context.Background(), host, args)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestWinPing(t *testing.T) {
	host, _ := fakeWindows(t)
	res := run(t, NewWinPingModule(), host, map[string]interface{}{})
	if !res.Success || res.Output["ping"] != "pong" {
		t.Errorf("win_ping = %+v", res)
	}
	res = run(t, NewWinPingModule(), host, map[string]interface{}{"_become": true})
	if !res.Failed || !strings.Contains(res.Error, "become") {
		t.Errorf("become = %+v", res)
	}
}

func TestWinCommand(t *testing.T) {
	host, _ := fakeWindows(t)
	host.Environment = map[string]string{"A": "1"}
	res := run(t, NewWinCommandModule(), host, map[string]interface{}{"cmd": "whoami /all", "chdir": `C:\tmp`, "stdin": "in"})
	out, _ := res.Output["stdout"].(string)
	if !res.Success || !res.Changed || !strings.Contains(out, `cmd:set "A=1" && cd /d "C:\tmp" && whoami /all|stdin:in`) {
		t.Errorf("win_command = %+v", res)
	}
	res = run(t, NewWinCommandModule(), host, map[string]interface{}{"cmd": "exit 5"})
	if !res.Failed || res.Output["rc"] != 5 || !strings.Contains(res.Error, "bad") {
		t.Errorf("failing command = %+v", res)
	}
	res = run(t, NewWinCommandModule(), host, map[string]interface{}{"cmd": "setup.exe", "creates": `C:\exists`})
	if !res.Success || res.Changed || !strings.Contains(res.Output["msg"].(string), "exists") {
		t.Errorf("creates = %+v", res)
	}
	res = run(t, NewWinCommandModule(), host, map[string]interface{}{"cmd": "x", "_check_mode": true})
	if !res.Skipped {
		t.Errorf("check mode = %+v", res)
	}
	res = run(t, NewWinCommandModule(), host, map[string]interface{}{})
	if !res.Failed {
		t.Error("a command is required")
	}
}

func TestWinShell(t *testing.T) {
	host, _ := fakeWindows(t)
	res := run(t, NewWinShellModule(), host, map[string]interface{}{"cmd": "Get-Date", "chdir": `C:\x`})
	out, _ := res.Output["stdout"].(string)
	if !res.Success || !strings.Contains(out, "ps:Set-Location -LiteralPath 'C:\\x'\nGet-Date") {
		t.Errorf("win_shell = %+v", res)
	}
	if l := res.Output["stdout_lines"].([]interface{}); len(l) != 3 || l[2] != "line2" {
		t.Errorf("stdout_lines = %v", l)
	}
	res = run(t, NewWinShellModule(), host, map[string]interface{}{"cmd": "throw 'x'"})
	if !res.Failed || res.Output["rc"] != 1 || !strings.Contains(res.Error, "boom") {
		t.Errorf("failing script = %+v", res)
	}
	res = run(t, NewWinShellModule(), host, map[string]interface{}{"cmd": "dir", "executable": "cmd"})
	out, _ = res.Output["stdout"].(string)
	if !strings.Contains(out, "cmd:cmd /c dir") {
		t.Errorf("executable cmd = %+v", res)
	}
}

func TestLinuxModuleOnWindowsHost(t *testing.T) {
	host, _ := fakeWindows(t)
	r := NewRegistry()
	res, err := r.ExecuteTask(context.Background(), &types.Task{Name: "t", Module: "copy",
		Args: map[string]interface{}{"dest": "C:\\x", "content": "y"}}, host, nil)
	if err != nil || !res.Failed || !strings.Contains(res.Error, "win_") {
		t.Errorf("copy on Windows = %+v, %v", res, err)
	}
	res, err = r.ExecuteTask(context.Background(), &types.Task{Name: "t", Module: "debug",
		Args: map[string]interface{}{"msg": "hi"}}, host, nil)
	if err != nil || res.Failed {
		t.Errorf("debug on Windows = %+v, %v", res, err)
	}
}
