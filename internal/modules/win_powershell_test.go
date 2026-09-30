package modules

import (
	"os"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm/winrmtest"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// pwshHost is a host whose PowerShell is a real pwsh (ONIGIRAZU_TEST_PWSH,
// e.g. "docker run --rm -i mcr.microsoft.com/powershell pwsh")
func pwshHost(t *testing.T) types.Host {
	pwsh := os.Getenv("ONIGIRAZU_TEST_PWSH")
	if pwsh == "" {
		t.Skip("ONIGIRAZU_TEST_PWSH is not set")
	}
	addr, port := winrmtest.Start(t, &winrmtest.Server{Handle: winrmtest.PwshHandler(pwsh)})
	return winrmtest.Host("pwsh", addr, port)
}

func cannedHost(t *testing.T, stdout string) types.Host {
	addr, port := winrmtest.Start(t, &winrmtest.Server{Handle: func(command, stdin string) (string, string, int) {
		if strings.Contains(winrmtest.Script(command, stdin), "Test-Path -LiteralPath 'C:\\exists'") {
			return "yes", "", 0
		}
		return stdout, "", 0
	}})
	return winrmtest.Host("w", addr, port)
}

func TestWinPowerShellResult(t *testing.T) {
	host := cannedHost(t, "banner\n"+winResultMarker+"\n"+
		`{"changed":false,"failed":false,"result":{"configured":true},"output":["a"],"error":[],"host_out":"hi"}`+"\n"+winResultMarker+"\n")
	res := run(t, NewWinPowerShellModule(), host, map[string]interface{}{"script": "$Ansible.Changed = $false"})
	if !res.Success || res.Changed || res.Output["result"].(map[string]interface{})["configured"] != true || res.Output["host_out"] != "hi" {
		t.Errorf("result = %+v", res)
	}
	host = cannedHost(t, winResultMarker+`{"changed":true,"failed":true,"error":[{"output":"Get-WsusServer: not found"}]}`+winResultMarker)
	res = run(t, NewWinPowerShellModule(), host, map[string]interface{}{"script": "Get-WsusServer"})
	if !res.Failed || res.Error != "Get-WsusServer: not found" {
		t.Errorf("failure = %+v", res)
	}
	host = cannedHost(t, "no marker")
	res = run(t, NewWinPowerShellModule(), host, map[string]interface{}{"script": "x"})
	if !res.Failed || !strings.Contains(res.Error, "no result") {
		t.Errorf("no marker = %+v", res)
	}
	res = run(t, NewWinPowerShellModule(), host, map[string]interface{}{"script": "x", "_check_mode": true})
	if !res.Skipped {
		t.Errorf("check mode = %+v", res)
	}
	res = run(t, NewWinPowerShellModule(), host, map[string]interface{}{"script": "x", "creates": `C:\exists`})
	if !res.Success || res.Changed {
		t.Errorf("creates = %+v", res)
	}
	for _, bad := range []map[string]interface{}{{}, {"script": "x", "error_action": "panic"}} {
		if res := run(t, NewWinPowerShellModule(), host, bad); !res.Failed {
			t.Errorf("%v accepted", bad)
		}
	}
}

func TestWinPowerShellWithPwsh(t *testing.T) {
	host := pwshHost(t)
	res := run(t, NewWinPowerShellModule(), host, map[string]interface{}{
		"script": "param($Name)\n$Ansible.Changed = $false\n$Ansible.Result = @{ greeting = \"hello $Name\" }\n" +
			"Write-Output 'out1'\nWrite-Warning 'careful'\nWrite-Host 'to host'\nWrite-Error 'soft'\n",
		"parameters": map[string]interface{}{"Name": "win"}})
	if !res.Success || res.Changed {
		t.Fatalf("result = %+v", res)
	}
	o := res.Output
	if o["result"].(map[string]interface{})["greeting"] != "hello win" || o["host_out"] != "to host" {
		t.Errorf("result/host_out = %v / %v", o["result"], o["host_out"])
	}
	if out := o["output"].([]interface{}); len(out) != 1 || out[0] != "out1" {
		t.Errorf("output = %v", o["output"])
	}
	if w := o["warning"].([]interface{}); len(w) != 1 || w[0] != "careful" {
		t.Errorf("warning = %v", o["warning"])
	}
	if e := o["error"].([]interface{}); len(e) != 1 {
		t.Errorf("a non-terminating error is listed and does not fail: %v", o["error"])
	}
	res = run(t, NewWinPowerShellModule(), host, map[string]interface{}{"script": "throw 'hard stop'"})
	if !res.Failed || !strings.Contains(res.Error, "hard stop") {
		t.Errorf("throw = %+v", res)
	}
	res = run(t, NewWinPowerShellModule(), host, map[string]interface{}{"script": "Write-Error 'as stop'; 'not reached'", "error_action": "stop"})
	if !res.Failed || !strings.Contains(res.Error, "as stop") {
		t.Errorf("error_action stop = %+v", res)
	}
	res = run(t, NewWinPowerShellModule(), host, map[string]interface{}{"script": "$Ansible.Failed = $true"})
	if !res.Failed {
		t.Errorf("$Ansible.Failed = %+v", res)
	}
}
