package cli

import (
	"bytes"
	"strings"
	"testing"
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
