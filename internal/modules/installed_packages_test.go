package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// the capture taken before the task answers without asking the host
func TestInstalledPackagesFromCapture(t *testing.T) {
	args := map[string]interface{}{"_before": map[string]interface{}{
		"kind": "packages", "names": []interface{}{"curl", "vim"}, "installed": []interface{}{"curl"},
	}}
	got, err := installedPackages(context.Background(), types.Host{}, args, []string{"vim", "curl"})
	if err != nil {
		t.Fatal(err)
	}
	if !got["curl"] || got["vim"] {
		t.Fatalf("got %v, want curl only", got)
	}
	// a name the capture does not cover goes to the host (none here)
	if _, err := installedPackages(context.Background(), types.Host{}, args, []string{"git"}); err == nil {
		t.Fatal("expected a host query for a name outside the capture")
	}
}
