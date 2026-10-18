package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestUnitRunning(t *testing.T) {
	for state, want := range map[string]bool{"active": true, "activating": true, "reloading": false,
		"inactive": false, "failed": false, "deactivating": false, "": false} {
		if got := unitRunning(state); got != want {
			t.Errorf("%q: %v", state, got)
		}
	}
	if got := activeState("activating\n"); got != "activating" {
		t.Errorf("activeState: %q", got)
	}
}

// a unit still activating is started, as in Ansible: no start, no change
func TestStartedOnActivatingUnit(t *testing.T) {
	before := map[string]interface{}{"kind": "service", "name": "app.service", "active": "activating", "enabled": "enabled"}
	if running, _, ok := capturedService(map[string]interface{}{"_before": before}, "app.service"); !ok || !running {
		t.Fatalf("service capture: running=%v ok=%v", running, ok)
	}
	host := types.Host{Name: "localhost", Address: "localhost", Vars: map[string]interface{}{"ansible_connection": "local"}}
	res, err := NewSystemdModule().Execute(context.Background(), host, map[string]interface{}{
		"name": "app.service", "state": "started", "_before": before,
	})
	if err != nil || res.Failed || res.Changed {
		t.Fatalf("systemd started on activating: changed=%v failed=%v err=%v %v", res.Changed, res.Failed, err, res.Error)
	}
}

// as in Ansible, state: stopped stops a unit that is still deactivating
func TestServiceStoppedOnDeactivatingUnit(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "localhost", Vars: map[string]interface{}{"ansible_connection": "local"}}
	for active, want := range map[string]bool{"deactivating": true, "inactive": false, "activating": true} {
		before := map[string]interface{}{"kind": "service", "name": "app.service", "active": active, "enabled": "enabled"}
		pre, err := NewServiceModuleFixed().PreCheckState(context.Background(), host, map[string]interface{}{
			"name": "app.service", "state": "stopped", "_before": before,
		})
		if err != nil || pre.ShouldExecute != want {
			t.Errorf("%s: should execute %v, want %v (%v)", active, pre != nil && pre.ShouldExecute, want, err)
		}
	}
}
