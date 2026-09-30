package validator

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/bridge"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestBridgedModulesAreKnown(t *testing.T) {
	v := NewModuleSyntaxValidator([]string{"copy"})
	task := &types.Task{Name: "t", Module: "community.general.nmcli"}
	if err := v.ValidateTaskModule(task, 0, 0); err == nil {
		t.Fatal("a module neither built in nor allowed is unknown")
	}
	bridge.Configure(bridge.Config{Modules: []string{"community.general.*"}})
	defer bridge.Configure(bridge.Config{})
	if err := v.ValidateTaskModule(task, 0, 0); err != nil {
		t.Errorf("an allowed module is known: %v", err)
	}
}
