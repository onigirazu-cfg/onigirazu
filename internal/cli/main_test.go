package cli

import (
	"fmt"
	"os"
	"testing"
)

// The package's tests run real playbooks, whose audit records, facts cache and
// state default to $HOME: a throwaway HOME keeps them out of the developer's
// own ~/.onigirazu (they used to show up in `onigirazu audit` and serve's Runs)
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "onigirazu-cli-test-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = os.Setenv("HOME", home)
	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
}
