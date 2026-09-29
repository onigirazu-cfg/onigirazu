package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// a fake runtime: records its arguments and runs the shell line locally
func fakeRuntime(t *testing.T, name string) string {
	dir := t.TempDir()
	log := filepath.Join(dir, "args")
	script := "#!/bin/sh\necho \"$@\" >> " + log + "\nfor last; do :; done\nexec sh -c \"$last\"\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func TestContainerConnection(t *testing.T) {
	log := fakeRuntime(t, "docker")
	host := types.Host{Name: "web1", User: "app", Vars: map[string]interface{}{"ansible_connection": "community.docker.docker"}}
	if runtime, name, ok := sshpkg.Container(host); !ok || runtime != "docker" || name != "web1" {
		t.Fatalf("Container = %s %s %v", runtime, name, ok)
	}
	if sshpkg.IsLocal(host) {
		t.Fatal("a container is not local")
	}
	e, err := NewCommandExecutor(host)
	if err != nil {
		t.Fatal(err)
	}
	out, err := e.Execute("echo", "a b")
	if err != nil || strings.TrimSpace(out) != "a b" {
		t.Fatalf("Execute = %q, %v", out, err)
	}
	res, err := e.Run(context.Background(), "echo out; echo err >&2; exit 3")
	if err != nil || res.RC != 3 || strings.TrimSpace(res.Stdout) != "out" || strings.TrimSpace(res.Stderr) != "err" {
		t.Fatalf("Run = %+v, %v", res, err)
	}
	args, _ := os.ReadFile(log)
	if !strings.HasPrefix(string(args), "exec -i -u app web1 sh -c ") {
		t.Errorf("runtime args = %q", args)
	}
	if !e.IsRemote() {
		t.Error("a container executor is remote")
	}
}

func TestPodmanConnectionUsesAnsibleHost(t *testing.T) {
	fakeRuntime(t, "podman")
	host := types.Host{Name: "db", Address: "db-container", Vars: map[string]interface{}{"ansible_connection": "podman"}}
	runtime, name, ok := sshpkg.Container(host)
	if !ok || runtime != "podman" || name != "db-container" {
		t.Fatalf("Container = %s %s %v", runtime, name, ok)
	}
}
