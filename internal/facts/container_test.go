package facts

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestContainerRunner(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script stands in for docker")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "docker")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho \"$@\"\n"), 0o755); err != nil { // #nosec G306 -- a test executable
		t.Fatal(err)
	}
	out, err := containerRunner{runtime: fake, name: "c1", user: "app"}.ExecuteCommand("uname -s")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out); got != "exec -u app c1 sh -c uname -s" {
		t.Errorf("args = %q", got)
	}
}
