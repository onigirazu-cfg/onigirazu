package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writePlugin(t *testing.T, dir, name, script string) string {
	t.Helper()
	p := filepath.Join(dir, pluginPrefix+name)
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil { // #nosec G306 -- a test executable
		t.Fatal(err)
	}
	return p
}

func TestCommandPlugins(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script plugins")
	}
	dir := t.TempDir()
	t.Setenv("ONIGIRAZU_PLUGIN_PATH", dir)
	t.Setenv("HOME", t.TempDir())
	hello := writePlugin(t, dir, "hello", "#!/bin/sh\nexit 3\n")
	_ = os.WriteFile(filepath.Join(dir, pluginPrefix+"noexec"), []byte("x"), 0o600)

	root := NewRootCommand()
	if got := pluginFor(root, []string{"hello", "x"}); got != hello {
		t.Errorf("pluginFor(hello) = %q", got)
	}
	if got := pluginFor(root, []string{"apply", "x"}); got != "" {
		t.Errorf("a command of onigirazu wins: %q", got)
	}
	for _, args := range [][]string{{"noexec"}, {"-v"}, {"../hello"}, {}} {
		if got := pluginFor(root, args); got != "" {
			t.Errorf("pluginFor(%q) = %q", args, got)
		}
	}
	var exit *ExitError
	if err := runPlugin(hello, nil); !errors.As(err, &exit) || exit.Code != 3 {
		t.Errorf("runPlugin exit = %v", err)
	}
	if found := listPlugins(); found["hello"] != hello || found["noexec"] != "" {
		t.Errorf("listPlugins = %v", found)
	}

	cmd := newPluginCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"list"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("hello")) {
		t.Errorf("plugin list = %q", out.String())
	}
}
