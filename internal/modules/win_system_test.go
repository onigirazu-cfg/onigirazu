package modules

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWinSystemArgs(t *testing.T) {
	host := cannedHost(t, "")
	for name, args := range map[string]map[string]interface{}{
		"win_file":     {"state": "file"},
		"win_copy":     {"content": "x"},
		"win_service":  {"state": "started"},
		"win_timezone": {},
	} {
		m, _ := NewRegistry().GetModule(name)
		if res := run(t, m, host, args); !res.Failed {
			t.Errorf("%s %v accepted", name, args)
		}
	}
	bad := []struct {
		module string
		args   map[string]interface{}
	}{
		{"win_file", map[string]interface{}{"path": `C:\x`, "state": "link"}},
		{"win_copy", map[string]interface{}{"dest": `C:\x`, "src": "a", "content": "b"}},
		{"win_copy", map[string]interface{}{"dest": `C:\x`, "src": t.TempDir()}},
		{"win_copy", map[string]interface{}{"dest": `C:\x`, "remote_src": true}},
		{"win_service", map[string]interface{}{"name": "x", "start_mode": "sometimes"}},
		{"win_service", map[string]interface{}{"name": "x", "state": "gone"}},
	}
	for _, b := range bad {
		m, _ := NewRegistry().GetModule(b.module)
		if res := run(t, m, host, b.args); !res.Failed {
			t.Errorf("%s %v accepted", b.module, b.args)
		}
	}
}

// file and copy scripts run in a real pwsh (paths of the pwsh machine);
// the pwsh must keep its files between commands (docker exec into a
// running container, not docker run --rm)
func TestWinFileAndCopyWithPwsh(t *testing.T) {
	host := pwshHost(t)
	dir := "/tmp/onigirazu-win-" + strings.ReplaceAll(t.Name(), "/", "-")
	file, _ := NewRegistry().GetModule("win_file")
	copyMod, _ := NewRegistry().GetModule("win_copy")
	step := func(m interface{ GetName() string }, args map[string]interface{}, changed bool) map[string]interface{} {
		t.Helper()
		mod, _ := NewRegistry().GetModule(m.GetName())
		res := run(t, mod, host, args)
		if !res.Success || res.Changed != changed {
			t.Fatalf("%s %v = %+v (want changed %v)", m.GetName(), args, res, changed)
		}
		return res.Output
	}
	step(file, map[string]interface{}{"path": dir, "state": "absent"}, false)
	step(file, map[string]interface{}{"path": dir, "state": "directory"}, true)
	step(file, map[string]interface{}{"path": dir, "state": "directory"}, false)
	step(file, map[string]interface{}{"path": dir + "/t", "state": "touch"}, true)
	step(file, map[string]interface{}{"path": dir + "/t"}, false)

	// content bigger than one upload piece
	big := strings.Repeat("0123456789abcdef", 20000) + "\n"
	out := step(copyMod, map[string]interface{}{"dest": dir + "/sub/big.txt", "content": big}, true)
	if out["size"] != float64(len(big)) {
		t.Errorf("size = %v", out["size"])
	}
	step(copyMod, map[string]interface{}{"dest": dir + "/sub/big.txt", "content": big}, false)
	step(copyMod, map[string]interface{}{"dest": dir + "/sub/big.txt", "content": "other", "force": false}, false)
	step(copyMod, map[string]interface{}{"dest": dir + "/sub/big.txt", "content": "other", "_check_mode": true}, true)
	step(copyMod, map[string]interface{}{"dest": dir + "/copy.txt", "src": dir + "/sub/big.txt", "remote_src": true}, true)
	step(copyMod, map[string]interface{}{"dest": dir + "/copy.txt", "src": dir + "/sub/big.txt", "remote_src": true}, false)

	local := filepath.Join(t.TempDir(), "local.txt")
	if err := os.WriteFile(local, []byte("from the control machine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out = step(copyMod, map[string]interface{}{"dest": dir, "src": local}, true)
	if out["dest"] != dir+"/local.txt" {
		t.Errorf("dest directory: %v", out["dest"])
	}
	step(file, map[string]interface{}{"path": dir, "state": "absent"}, true)
}

// the service and time zone scripts parse (they need Windows to run)
func TestWinServiceTimezoneScriptsParse(t *testing.T) {
	host := pwshHost(t)
	for _, s := range []func() (string, error){
		func() (string, error) {
			return winServiceScript(context.TODO(), nil, map[string]interface{}{"name": "W32Time", "state": "started", "start_mode": "delayed"})
		},
		func() (string, error) {
			return winTimezoneScript(context.TODO(), nil, map[string]interface{}{"timezone": "UTC"})
		},
	} {
		script, err := s()
		if err != nil {
			t.Fatal(err)
		}
		res := run(t, NewWinPowerShellModule(), host, map[string]interface{}{
			"script":     "param($Text)\n$e = $null\n[void][System.Management.Automation.Language.Parser]::ParseInput($Text, [ref]$null, [ref]$e)\n$Ansible.Result = @($e | ForEach-Object { $_.Message })",
			"parameters": map[string]interface{}{"Text": script}})
		if errs, _ := res.Output["result"].([]interface{}); !res.Success || len(errs) > 0 {
			t.Errorf("parse: %+v", res)
		}
	}
}
