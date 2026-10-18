package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// the comply flags must coexist with the root's persistent flags (no
// shorthand clash) and `comply list` must run
func TestComplyCommandFlags(t *testing.T) {
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"comply", "list"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "linux-baseline") || !strings.Contains(out.String(), "ssh ") {
		t.Errorf("comply list:\n%s", out.String())
	}
}

func writeComplyRunFiles(t *testing.T, dir string, slow bool) (string, string) {
	t.Helper()
	profile := filepath.Join(dir, "profile.yml")
	inventory := filepath.Join(dir, "hosts.yml")
	data := "name: local\nbecome: false\ncontrols:\n  - id: quick\n    title: directory exists\n    check: {file: {path: /tmp, type: directory}}\n"
	if slow {
		data += "  - id: slow\n    title: slow command\n    check: {command: {cmd: 'sleep 30; true'}}\n"
	}
	if err := os.WriteFile(profile, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inventory, []byte("all:\n  hosts:\n    localhost:\n      ansible_connection: local\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return profile, inventory
}

func TestComplyReportExecutionID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	profile, inventory := writeComplyRunFiles(t, dir, false)
	report, executionID, err := complyReport(context.Background(), &complyOptions{profile: profile, driftCheckOptions: driftCheckOptions{inventory: []string{inventory}}})
	if err != nil {
		t.Fatal(err)
	}
	if report == nil || report.Passed != 1 || executionID == "" {
		t.Fatalf("report=%+v executionID=%q", report, executionID)
	}
}

func TestComplyReportKeepsPartialResult(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	profile, inventory := writeComplyRunFiles(t, dir, true)
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(2*time.Second, cancel)
	defer timer.Stop()
	defer cancel()
	report, executionID, err := complyReport(ctx, &complyOptions{profile: profile, driftCheckOptions: driftCheckOptions{inventory: []string{inventory}}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context error, got %v", err)
	}
	if report == nil || report.Hosts["localhost"] == nil || report.Passed != 1 || report.Failed != 0 || executionID == "" {
		t.Fatalf("partial report=%+v executionID=%q", report, executionID)
	}
}
