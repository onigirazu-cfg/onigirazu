//go:build !windows

package modules

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestRunShellOnHostKeepsOutputAndStops(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "localhost", Vars: map[string]interface{}{"ansible_connection": "local"}}
	_, err := runShellOnHost(context.Background(), host, nil, `echo "gpg is not installed" >&2; exit 3`)
	if err == nil || !strings.Contains(err.Error(), "gpg is not installed") {
		t.Fatalf("the error must carry the output: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := runShellOnHost(ctx, host, nil, "sleep 30; true"); err == nil {
		t.Fatal("a canceled script must fail")
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("returned after %s", elapsed)
	}
}
