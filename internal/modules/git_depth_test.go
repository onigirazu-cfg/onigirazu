package modules

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestGitShallowClone(t *testing.T) {
	src := t.TempDir()
	git := func(dir string, args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git(src, "init", "-q", "-b", "main")
	for _, msg := range []string{"one", "two", "three"} {
		git(src, "commit", "-q", "--allow-empty", "-m", msg)
		if msg == "two" {
			git(src, "tag", "v2")
		}
	}
	dest := filepath.Join(t.TempDir(), "clone")
	host := types.Host{Name: "h", Address: "localhost"}
	run := func(version string) types.TaskResult {
		res, err := NewGitModule().Execute(context.Background(), host, map[string]interface{}{
			"repo": "file://" + src, "dest": dest, "version": version, "depth": 1})
		if err != nil || !res.Success {
			t.Fatalf("%s: %+v, %v", version, res, err)
		}
		return res
	}
	if !run("v2").Changed {
		t.Error("the clone must report a change")
	}
	if n := git(dest, "rev-list", "--count", "HEAD"); n != "1" {
		t.Errorf("a depth 1 clone has %s commits", n)
	}
	if run("v2").Changed {
		t.Error("the same version again must not change")
	}
	run("main")
	if msg := git(dest, "log", "-1", "--format=%s"); msg != "three" {
		t.Errorf("after switching to main HEAD is %q", msg)
	}
}
