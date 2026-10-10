package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRepo(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		run("init", "-q", "-b", "main")
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "-A")
	run("commit", "-q", "-m", "x")
}

func TestPullSyncRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	src := t.TempDir()
	gitRepo(t, src, map[string]string{"site.yml": "- hosts: all\n  tasks: []\n"})
	o := &pullOptions{repo: src, branch: "main"}
	dir := filepath.Join(t.TempDir(), "checkout")
	c1, changed, err := syncRepo(context.Background(), o, dir)
	if err != nil || !changed || len(c1) != 40 {
		t.Fatalf("clone: %v changed=%v commit=%q", err, changed, c1)
	}
	c2, changed, err := syncRepo(context.Background(), o, dir)
	if err != nil || changed || c2 != c1 {
		t.Fatalf("no change: %v changed=%v", err, changed)
	}
	gitRepo(t, src, map[string]string{"site.yml": "- hosts: all\n  tasks: [{debug: {msg: x}}]\n"})
	c3, changed, err := syncRepo(context.Background(), o, dir)
	if err != nil || !changed || c3 == c1 {
		t.Fatalf("after a push: %v changed=%v", err, changed)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "site.yml")); !strings.Contains(string(b), "debug") {
		t.Fatal("checkout not updated")
	}
}

func TestPullArgsAndUnits(t *testing.T) {
	o := &pullOptions{repo: "git@x:ops/site.git", branch: "main", playbook: "site.yml", driftOnly: true, limit: "web", become: true, extraVars: []string{"a=1"}}
	args := pullApplyArgs(o, "/d")
	want := []string{"/d/site.yml", "--check", "--limit", "web", "--become", "-e", "a=1"}
	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Fatalf("args: %v", args)
	}
	dir, err := pullDir(o)
	if err != nil || !strings.HasSuffix(dir, filepath.Join(".onigirazu", "pull", "site")) {
		t.Fatalf("dir: %q %v", dir, err)
	}
	service, timer := pullUnitFiles(o, "/usr/local/bin/onigirazu")
	if !strings.Contains(service, "ExecStart=/usr/local/bin/onigirazu 'pull' '--repo' 'git@x:ops/site.git'") || !strings.Contains(service, "'--drift-only'") {
		t.Fatalf("service:\n%s", service)
	}
	if !strings.Contains(timer, "OnUnitActiveSec=30m0s") {
		t.Fatalf("timer:\n%s", timer)
	}
}
