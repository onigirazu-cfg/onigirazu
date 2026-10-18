//go:build !windows

package modules

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// a canceled run stops the script and what it started, not only bash
func TestScriptCancelStopsChildren(t *testing.T) {
	path := t.TempDir() + "/hang.sh"
	if err := os.WriteFile(path, []byte("sleep 30\ntrue\n"), 0o755); err != nil { // #nosec G306 -- a test script
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	started := time.Now()
	res, _ := NewScriptModule().Execute(ctx, types.Host{Name: "localhost", Address: "localhost",
		Vars: map[string]interface{}{"ansible_connection": "local"}}, map[string]interface{}{"cmd": path})
	if res.Success {
		t.Fatal("a canceled script must fail")
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("returned after %s: the child kept running", elapsed)
	}
}
