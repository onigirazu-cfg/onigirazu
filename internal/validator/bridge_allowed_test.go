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

func TestBridgeSettingsArePerValidator(t *testing.T) {
	first := NewModuleSyntaxValidatorWithBridge(nil, bridge.Config{Modules: []string{"first.*"}})
	second := NewModuleSyntaxValidatorWithBridge(nil, bridge.Config{Modules: []string{"second.*"}})
	for _, tc := range []struct {
		validator       *ModuleSyntaxValidator
		allowed, denied string
	}{
		{first, "first.task", "second.task"}, {second, "second.task", "first.task"},
	} {
		if err := tc.validator.ValidateTaskModule(&types.Task{Module: tc.allowed}, 0, 0); err != nil {
			t.Fatal(err)
		}
		if err := tc.validator.ValidateTaskModule(&types.Task{Module: tc.denied}, 0, 0); err == nil {
			t.Fatal("validator used another run's bridge settings")
		}
	}
}
