package modules

import (
	"context"
	"fmt"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// MetaModule stands for meta tasks (flush_handlers, end_host, end_play,
// noop); the engine performs them, this only validates the action
type MetaModule struct {
	*BaseModule
}

// NewMetaModule creates the meta module
func NewMetaModule() *MetaModule {
	return &MetaModule{BaseModule: NewBaseModule("meta")}
}

func (m *MetaModule) GetDescription() string {
	return "Engine actions: flush_handlers, end_host, end_play, noop"
}

func (m *MetaModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	return types.TaskResult{Host: host.Name, Module: m.name, Success: true, Skipped: true}, nil
}

func (m *MetaModule) Validate(args map[string]interface{}) error {
	action := getStringArg(args, "free_form", "")
	switch action {
	case "flush_handlers", "end_host", "end_play", "noop", "clear_host_errors", "refresh_inventory", "reset_connection":
		return nil
	}
	return fmt.Errorf("unsupported meta action %q", action)
}

// IncludeRoleModule stands for include_role/import_role: the parser loads the
// role and the engine runs it; this only validates the arguments
type IncludeRoleModule struct {
	*BaseModule
}

// NewIncludeRoleModule creates the include_role or import_role module
func NewIncludeRoleModule(name string) *IncludeRoleModule {
	return &IncludeRoleModule{BaseModule: NewBaseModule(name)}
}

func (m *IncludeRoleModule) GetDescription() string {
	return "Run a role as a task"
}

func (m *IncludeRoleModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	return types.TaskResult{Host: host.Name, Module: m.name, Success: true, Skipped: true}, nil
}

func (m *IncludeRoleModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "name", "") == "" {
		return fmt.Errorf("argument 'name' is required")
	}
	return nil
}
