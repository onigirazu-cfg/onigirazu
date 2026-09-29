package ssh

import (
	"io"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestParseSSHArgs(t *testing.T) {
	tests := []struct {
		args string
		want sshOptions
	}{
		{"", sshOptions{}},
		{"-o ConnectTimeout=15", sshOptions{ConnectTimeout: 15 * time.Second}},
		{"-oConnectTimeout=7 -o StrictHostKeyChecking=no", sshOptions{ConnectTimeout: 7 * time.Second}},
		{`-o "ConnectTimeout 3"`, sshOptions{ConnectTimeout: 3 * time.Second}},
		{"-o ConnectTimeout=abc", sshOptions{}},
		{"-J bastion", sshOptions{ProxyJump: "bastion"}},
		{"-Jadmin@bastion:2222,inner", sshOptions{ProxyJump: "admin@bastion:2222,inner"}},
		{"-o ProxyJump=bastion", sshOptions{ProxyJump: "bastion"}},
		{"-o ProxyJump=none", sshOptions{}},
		{`-o ProxyCommand="ssh -W %h:%p bastion"`, sshOptions{ProxyCommand: "ssh -W %h:%p bastion"}},
		{`-o 'ProxyCommand ssh -W %h:%p bastion'`, sshOptions{ProxyCommand: "ssh -W %h:%p bastion"}},
		{"-o proxycommand=none -C -v", sshOptions{}},
	}
	for _, tt := range tests {
		if got := parseSSHArgs(tt.args); got != tt.want {
			t.Errorf("parseSSHArgs(%q) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}

func TestShellWords(t *testing.T) {
	got := shellWords(`-o 'A B' -o "C=D E"  x`)
	want := []string{"-o", "A B", "-o", "C=D E", "x"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("shellWords = %q, want %q", got, want)
	}
	if got := shellWords(`""`); !reflect.DeepEqual(got, []string{""}) {
		t.Errorf("empty quotes = %q", got)
	}
}

func TestJumpConfigExplicit(t *testing.T) {
	target := types.Host{User: "deploy", InsecureIgnoreHostKey: true}
	name, addr, cfg, err := jumpConfig("admin@203.0.113.5:2222", target, nil, nil, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if name != "203.0.113.5" || addr != "203.0.113.5:2222" || cfg.User != "admin" || cfg.Timeout != 5*time.Second {
		t.Errorf("got name=%s addr=%s user=%s timeout=%s", name, addr, cfg.User, cfg.Timeout)
	}
	if err := cfg.HostKeyCallback("x", nil, nil); err != nil {
		t.Errorf("insecure target should skip the jump host key check: %v", err)
	}
}

func TestProxyCommandConn(t *testing.T) {
	conn, err := proxyCommandConn("printf '%s' '%h %p %r %%'", types.Host{User: "bob"}, "192.0.2.1:2200")
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(conn)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "192.0.2.1 2200 bob %" {
		t.Errorf("output = %q", out)
	}
	if a := conn.RemoteAddr().String(); a != "192.0.2.1:2200" {
		t.Errorf("RemoteAddr = %q", a)
	}
	_ = conn.Close()
}

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got := expandHome("~/.ssh/id"); got != filepath.Join(home, ".ssh/id") {
		t.Errorf("expandHome = %q", got)
	}
	if got := expandHome("/abs"); got != "/abs" {
		t.Errorf("expandHome(/abs) = %q", got)
	}
}
