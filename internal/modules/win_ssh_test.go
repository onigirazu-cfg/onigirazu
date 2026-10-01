package modules

import (
	"os"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// sshPwshHost is a host whose sshd runs powershell.exe = pwsh
// (ONIGIRAZU_TEST_WINSSH=user@address:port,keyfile)
func sshPwshHost(t *testing.T) types.Host {
	spec := os.Getenv("ONIGIRAZU_TEST_WINSSH")
	if spec == "" {
		t.Skip("ONIGIRAZU_TEST_WINSSH is not set")
	}
	target, key, _ := strings.Cut(spec, ",")
	user, addr, _ := strings.Cut(target, "@")
	host, port, _ := strings.Cut(addr, ":")
	p := 22
	if port != "" {
		p = atoiOr(port, 22)
	}
	return types.Host{Name: "winssh", Address: host, Port: p, User: user, KeyFile: key, InsecureIgnoreHostKey: true,
		Vars: map[string]interface{}{"ansible_connection": "ssh", "ansible_shell_type": "powershell"}}
}

func atoiOr(s string, def int) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func TestWindowsOverSSHIsWindows(t *testing.T) {
	h := types.Host{Name: "w", Vars: map[string]interface{}{"ansible_shell_type": "powershell"}}
	if !winrm.IsWindows(h) || winrm.IsWinRM(h) || !winrm.IsWindowsSSH(h) {
		t.Error("ansible_shell_type powershell is Windows over SSH")
	}
	if winrm.IsWindows(types.Host{Name: "l"}) {
		t.Error("a Linux host is not Windows")
	}
	r, err := winrm.For(h)
	if err != nil || r == nil {
		t.Errorf("For = %v, %v", r, err)
	}
}

func TestWinModulesOverSSH(t *testing.T) {
	host := sshPwshHost(t)
	res := run(t, NewWinPingModule(), host, map[string]interface{}{})
	if !res.Success || res.Output["ping"] != "pong" {
		t.Fatalf("win_ping = %+v", res)
	}
	res = run(t, NewWinPowerShellModule(), host, map[string]interface{}{
		"script": "param($N)\n$Ansible.Changed = $false\n$Ansible.Result = @{ n = $N * 2 }", "parameters": map[string]interface{}{"N": 21}})
	if !res.Success || res.Changed || res.Output["result"].(map[string]interface{})["n"] != float64(42) {
		t.Fatalf("win_powershell = %+v", res)
	}
	dir := "/tmp/onigirazu-winssh"
	file, _ := NewRegistry().GetModule("win_file")
	copyMod, _ := NewRegistry().GetModule("win_copy")
	run(t, file, host, map[string]interface{}{"path": dir, "state": "absent"})
	big := strings.Repeat("ssh-transport ", 20000)
	res = run(t, copyMod, host, map[string]interface{}{"dest": dir + "/big.txt", "content": big})
	if !res.Success || !res.Changed || res.Output["size"] != float64(len(big)) {
		t.Fatalf("win_copy = %+v", res)
	}
	res = run(t, copyMod, host, map[string]interface{}{"dest": dir + "/big.txt", "content": big})
	if !res.Success || res.Changed {
		t.Fatalf("second win_copy = %+v", res)
	}
	res = run(t, NewWinShellModule(), host, map[string]interface{}{"cmd": "throw 'over ssh'"})
	if !res.Failed || !strings.Contains(res.Error, "over ssh") {
		t.Errorf("failing script = %+v", res)
	}
	run(t, file, host, map[string]interface{}{"path": dir, "state": "absent"})
}
