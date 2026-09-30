package modules

import (
	"reflect"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm/winrmtest"
)

func TestRegProviderPath(t *testing.T) {
	for in, want := range map[string]string{
		`HKLM:\System\CurrentControlSet`: `Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet`,
		`HKEY_CURRENT_USER\Software\X\`:  `Registry::HKEY_CURRENT_USER\Software\X`,
		`hkcu:/Software/Y`:               `Registry::HKEY_CURRENT_USER\Software\Y`,
		`HKU:\S-1-5-18`:                  `Registry::HKEY_USERS\S-1-5-18`,
	} {
		if got, err := regProviderPath(in); err != nil || got != want {
			t.Errorf("regProviderPath(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := regProviderPath(`C:\Windows`); err == nil {
		t.Error("a file path is not a registry path")
	}
}

func TestRegData(t *testing.T) {
	cases := []struct {
		kind string
		in   interface{}
		want interface{}
	}{
		{"DWord", 1, "1"}, {"DWord", "0xFFFFFFFF", "4294967295"}, {"DWord", "", "0"}, {"QWord", "0x10", "16"},
		{"MultiString", []interface{}{"a", 1}, []string{"a", "1"}}, {"MultiString", "one", []string{"one"}},
		{"Binary", []interface{}{0, 255}, "00ff"}, {"Binary", "hex:be,ef", "beef"}, {"String", 3, "3"},
	}
	for _, c := range cases {
		got, err := regData(c.kind, c.in)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("regData(%s, %v) = %v, %v; want %v", c.kind, c.in, got, err, c.want)
		}
	}
	for _, bad := range []struct {
		kind string
		in   interface{}
	}{{"DWord", "0x1FFFFFFFF"}, {"DWord", "-1"}, {"Binary", []interface{}{300}}, {"Binary", "zz"}} {
		if _, err := regData(bad.kind, bad.in); err == nil {
			t.Errorf("regData(%s, %v) accepted", bad.kind, bad.in)
		}
	}
}

func TestWinRegeditScriptAndResult(t *testing.T) {
	var script string
	addr, port := winrmtest.Start(t, &winrmtest.Server{Handle: func(command, stdin string) (string, string, int) {
		script = winrmtest.Script(command, stdin)
		return winResultMarker + `{"changed":true,"data_changed":true,"data_type_changed":false}` + winResultMarker, "", 0
	}})
	host := winrmtest.Host("w", addr, port)
	res := run(t, NewWinRegeditModule(), host, map[string]interface{}{
		"path": `HKLM:\System\CurrentControlSet\Control\Terminal Server`, "name": "fDenyTSConnections", "data": 0, "type": "dword"})
	if !res.Success || !res.Changed || res.Output["data_changed"] != true {
		t.Errorf("result = %+v", res)
	}
	for _, want := range []string{`$path = 'Registry::HKEY_LOCAL_MACHINE\System\CurrentControlSet\Control\Terminal Server'`,
		`$name = 'fDenyTSConnections'`, `$kind = 'DWord'`, `$state = 'present'`, `(ConvertFrom-Json '"0"')`} {
		if !strings.Contains(script, want) {
			t.Errorf("script lacks %q", want)
		}
	}
	res = run(t, NewWinRegeditModule(), host, map[string]interface{}{"path": `HKLM:\Software\X`, "state": "absent"})
	if !strings.Contains(script, `$name = $null`) || !strings.Contains(script, `$state = 'absent'`) {
		t.Errorf("absent key script: %s", script)
	}
	for _, bad := range []map[string]interface{}{
		{"path": `C:\x`}, {"path": `HKLM:\X`, "type": "word"}, {"path": `HKLM:\X`, "state": "gone"},
		{"path": `HKLM:\X`, "name": "v", "type": "dword", "data": "abc"},
	} {
		if res := run(t, NewWinRegeditModule(), host, bad); !res.Failed {
			t.Errorf("%v accepted", bad)
		}
	}
}

// the registry scripts are valid PowerShell (pwsh parses them; the
// registry itself exists only on Windows)
func TestWinRegeditScriptParses(t *testing.T) {
	host := pwshHost(t)
	for _, args := range []map[string]interface{}{
		{"path": `HKLM:\X`, "name": "v", "type": "dword", "data": "0xFFFFFFFF"},
		{"path": `HKLM:\X`, "name": "m", "type": "multistring", "data": []interface{}{"a", "b"}},
		{"path": `HKLM:\X`, "state": "absent"},
	} {
		script, err := regeditScript(args)
		if err != nil {
			t.Fatal(err)
		}
		res := run(t, NewWinPowerShellModule(), host, map[string]interface{}{
			"script":     "param($Text)\n$e = $null\n[void][System.Management.Automation.Language.Parser]::ParseInput($Text, [ref]$null, [ref]$e)\n$Ansible.Result = @($e | ForEach-Object { $_.Message })",
			"parameters": map[string]interface{}{"Text": script}})
		if !res.Success {
			t.Fatalf("parse run = %+v", res)
		}
		if errs, _ := res.Output["result"].([]interface{}); len(errs) > 0 {
			t.Errorf("%v: parse errors %v", args, errs)
		}
	}
}
