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
