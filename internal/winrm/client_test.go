package winrm

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm/winrmtest"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestRunPSSendsTheScriptOnStdin(t *testing.T) {
	long := strings.Repeat("# padding\n", 2000) + "Write-Output 'hi'"
	f := &winrmtest.Server{Handle: func(command, stdin string) (string, string, int) {
		if !strings.Contains(winrmtest.DecodePS(command), "[Console]::In.ReadToEnd()") {
			return "", "not the bootstrap: " + command, 9
		}
		script, err := base64.StdEncoding.DecodeString(stdin)
		if err != nil {
			return "", err.Error(), 8
		}
		if string(script) != long {
			return "", "script changed on the way", 7
		}
		return "hi\r\n", "", 0
	}}
	c := startFake(t, f)
	res, err := c.RunPS(context.Background(), long)
	if err != nil || res.ExitCode != 0 || res.Stdout != "hi\r\n" {
		t.Fatalf("RunPS = %+v, %v", res, err)
	}
	if len(f.Commands[0]) > 8000 {
		t.Errorf("command line is %d bytes: the script must not be on it", len(f.Commands[0]))
	}
}

func TestRunCmdAndExitCodes(t *testing.T) {
	f := &winrmtest.Server{Handle: func(command, stdin string) (string, string, int) {
		script, _ := base64.StdEncoding.DecodeString(stdin)
		if strings.Contains(command, "fail") || string(script) == "fail" {
			return "", `#< CLIXML
<Objs Version="1.1.0.1" xmlns="http://schemas.microsoft.com/powershell/2004/04"><S S="Error">bad &lt;thing&gt;_x000D__x000A_</S></Objs>`, 3
		}
		return command, "", 0
	}}
	c := startFake(t, f)
	res, err := c.RunCmd(context.Background(), "ipconfig /all")
	if err != nil || !strings.Contains(res.Stdout, "ipconfig /all") {
		t.Fatalf("RunCmd = %+v, %v", res, err)
	}
	res, err = c.RunPS(context.Background(), "fail")
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 3 {
		t.Errorf("exit code = %d", res.ExitCode)
	}
	res, _ = c.RunCmd(context.Background(), "fail")
	if res.ExitCode != 3 {
		t.Errorf("cmd exit code = %d", res.ExitCode)
	}
	if got := cleanCLIXML(res.Stderr); got != "bad <thing>" {
		t.Errorf("cleanCLIXML = %q", got)
	}
}

func TestSettingsFor(t *testing.T) {
	host := func(vars map[string]interface{}) types.Host {
		return types.Host{Name: "w1", Address: "10.0.0.9", User: "admin", Vars: vars}
	}
	s, err := SettingsFor(host(map[string]interface{}{"ansible_connection": "winrm"}))
	if err != nil || s.Port != 5986 || !s.HTTPS || s.Transport != "ntlm" || s.Encryption != "auto" {
		t.Errorf("defaults = %+v, %v", s, err)
	}
	s, err = SettingsFor(host(map[string]interface{}{"ansible_port": 5985, "ansible_winrm_transport": "ntlm",
		"ansible_winrm_message_encryption": "always", "ansible_winrm_server_cert_validation": "ignore",
		"ansible_winrm_scheme": "http", "ansible_winrm_read_timeout_sec": "120", "ansible_password": "p"}))
	if err != nil || s.Port != 5986 && s.Port != 5985 || s.HTTPS || s.Encryption != "always" || !s.Insecure ||
		s.Timeout != 120*time.Second || s.Password != "p" {
		t.Errorf("ClanRed style = %+v, %v", s, err)
	}
	h := host(nil)
	h.Port = 5985
	if s, _ := SettingsFor(h); s.HTTPS {
		t.Error("port 5985 means http")
	}
	for _, bad := range []map[string]interface{}{
		{"ansible_winrm_transport": "kerberos"},
		{"ansible_winrm_scheme": "ftp"},
		{"ansible_winrm_message_encryption": "sometimes"},
		{"ansible_winrm_transport": "basic", "ansible_winrm_message_encryption": "always"},
		{"ansible_winrm_read_timeout_sec": "soon"},
	} {
		if _, err := SettingsFor(host(bad)); err == nil {
			t.Errorf("SettingsFor(%v) accepted", bad)
		}
	}
	if _, err := SettingsFor(types.Host{Name: "w2"}); err == nil {
		t.Error("a user is required")
	}
	if !IsWinRM(host(map[string]interface{}{"ansible_connection": "winrm"})) || IsWinRM(host(nil)) {
		t.Error("IsWinRM")
	}
}

func TestPoolReusesClients(t *testing.T) {
	h := types.Host{Name: "w1", Address: "10.0.0.9", User: "admin", Vars: map[string]interface{}{"ansible_connection": "winrm"}}
	a, err := For(h)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := For(h)
	if a != b {
		t.Error("one client per host")
	}
}

func startFake(t *testing.T, f *winrmtest.Server) *Client {
	t.Helper()
	addr, port := winrmtest.Start(t, f)
	c, err := Connect(Settings{Address: addr, Port: port, Transport: "basic", Encryption: "never",
		User: "admin", Password: "secret", Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// the cmd.exe wrapper of the SSH runner is valid PowerShell
func TestCmdScriptParses(t *testing.T) {
	pwsh := os.Getenv("ONIGIRAZU_TEST_PWSH")
	if pwsh == "" {
		t.Skip("ONIGIRAZU_TEST_PWSH is not set")
	}
	c := startFake(t, &winrmtest.Server{Handle: winrmtest.PwshHandler(pwsh)})
	script := fmt.Sprintf(cmdScript, psString(`set "A=1" && echo %A% 'x'`), psString("in"))
	res, err := c.RunPS(context.Background(), "$e = $null\n[void][System.Management.Automation.Language.Parser]::ParseInput("+
		psString(script)+", [ref]$null, [ref]$e)\n$e.Count")
	if err != nil || strings.TrimSpace(res.Stdout) != "0" {
		t.Errorf("parse: %+v %v", res, err)
	}
}

func TestRetryShell(t *testing.T) {
	calls := 0
	res, err := retryShell(context.Background(), func() (Result, error) {
		calls++
		if calls < 2 {
			return Result{}, errors.New("http error 500: Illegal operation attempted on a registry key that has been marked for deletion.")
		}
		return Result{Stdout: "ok"}, nil
	})
	if err != nil || res.Stdout != "ok" || calls != 2 {
		t.Errorf("retryShell = %+v, %v after %d calls", res, err, calls)
	}
	calls = 0
	_, err = retryShell(context.Background(), func() (Result, error) { calls++; return Result{}, errors.New("access denied") })
	if err == nil || calls != 1 {
		t.Errorf("other errors are not retried: %d calls", calls)
	}
}
