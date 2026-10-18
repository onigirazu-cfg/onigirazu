//go:build !windows

package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// a reader that goes away mid-run (apply | head) must not kill the run: it
// finishes, saves its state and releases the state lock
func TestApplySurvivesClosedOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "onigirazu")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	done := filepath.Join(dir, "done.txt")
	pb := "- hosts: localhost\n  gather_facts: false\n  tasks:\n    - name: wait\n      command: sleep 1\n    - name: last\n      copy: {dest: " + done + ", content: \"x\\n\"}\n"
	for name, body := range map[string]string{"pb.yml": pb, "hosts.yml": "all:\n  hosts:\n    localhost:\n      ansible_connection: local\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(bin, "apply", "pb.yml", "-i", "hosts.yml", "--no-color")
	cmd.Dir = dir
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// read until the run is under way, then close the pipe as head would
	lines := bufio.NewScanner(out)
	for lines.Scan() && !strings.Contains(lines.Text(), "wait") {
	}
	_ = out.Close()
	_ = cmd.Wait()
	if _, err := os.Stat(done); err != nil {
		t.Fatalf("the run stopped when its reader went away: %v", err)
	}
	if locks, _ := filepath.Glob(filepath.Join(dir, ".onigirazu", "*.lock")); len(locks) > 0 {
		t.Fatalf("state lock left behind: %v", locks)
	}
}
