package modules

import (
	"context"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// SetupModule stands for setup and gather_facts: the engine gathers the
// facts again, this only makes the names known
type SetupModule struct {
	*BaseModule
}

// NewSetupModule creates the module for one of the two names
func NewSetupModule(name string) *SetupModule {
	return &SetupModule{BaseModule: NewBaseModule(name)}
}

func (m *SetupModule) GetDescription() string {
	return "Gather the facts of the host again (filter, fact_path)"
}

func (m *SetupModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	return types.TaskResult{Host: host.Name, Module: m.name, Success: true, Skipped: true}, nil
}

func (m *SetupModule) Validate(args map[string]interface{}) error { return nil }
