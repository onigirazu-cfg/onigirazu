//go:build !windows

package executor

import (
	"context"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// a shell's child holds the output pipes: canceling must stop it too, not
// only the shell (a compound line keeps sh from exec'ing the command)
func TestRunCancelStopsChildren(t *testing.T) {
	e, err := NewCommandExecutor(types.Host{Name: "localhost", Address: "localhost"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := e.Run(ctx, "sleep 30; true"); err == nil {
		t.Fatal("a canceled command must fail")
	}
	if elapsed := time.Since(started); elapsed > 3*time.Second {
		t.Fatalf("returned after %s: the child kept running", elapsed)
	}
}
