package ssh

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestIsWindowsSSH(t *testing.T) {
	for _, tc := range []struct {
		vars map[string]interface{}
		want bool
	}{
		{nil, false},
		{map[string]interface{}{"ansible_shell_type": "sh"}, false},
		{map[string]interface{}{"ansible_shell_type": "powershell"}, true},
		{map[string]interface{}{"ansible_shell_type": "cmd"}, true},
		{map[string]interface{}{"onigirazu_shell_type": "powershell"}, true},
	} {
		if got := isWindowsSSH(types.Host{Vars: tc.vars}); got != tc.want {
			t.Errorf("%v: got %v", tc.vars, got)
		}
	}
}

func TestWindowsAgentCommand(t *testing.T) {
	got := windowsAgentCommand(`C:\Users\o'neil\.onigirazu\bin\onigirazu-agent-1.exe`)
	want := `powershell.exe -NoProfile -NonInteractive -Command "& 'C:\Users\o''neil\.onigirazu\bin\onigirazu-agent-1.exe' serve"`
	if got != want {
		t.Errorf("got %s", got)
	}
}
