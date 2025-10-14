package plugins

import (
	"context"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// BaseCallbackPlugin provides a base implementation for callback plugins
type BaseCallbackPlugin struct {
	name        string
	version     string
	description string
}

// NewBaseCallbackPlugin creates a new base callback plugin
func NewBaseCallbackPlugin(name, version, description string) *BaseCallbackPlugin {
	return &BaseCallbackPlugin{
		name:        name,
		version:     version,
		description: description,
	}
}

// GetName returns the plugin name
func (p *BaseCallbackPlugin) GetName() string {
	return p.name
}

// GetType returns the plugin type
func (p *BaseCallbackPlugin) GetType() PluginType {
	return PluginTypeCallback
}

// GetVersion returns the plugin version
func (p *BaseCallbackPlugin) GetVersion() string {
	return p.version
}

// GetDescription returns the plugin description
func (p *BaseCallbackPlugin) GetDescription() string {
	return p.description
}

// Initialize initializes the plugin with configuration
func (p *BaseCallbackPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	// Default implementation does nothing
	return nil
}

// Cleanup performs cleanup when plugin is unloaded
func (p *BaseCallbackPlugin) Cleanup(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// OnPlaybookStart is called when playbook execution starts
func (p *BaseCallbackPlugin) OnPlaybookStart(ctx context.Context, playbook *types.Playbook) error {
	// Default implementation does nothing
	return nil
}

// OnPlaybookEnd is called when playbook execution ends
func (p *BaseCallbackPlugin) OnPlaybookEnd(ctx context.Context, playbook *types.Playbook, success bool, duration time.Duration) error {
	// Default implementation does nothing
	return nil
}

// OnPlayStart is called when play execution starts
func (p *BaseCallbackPlugin) OnPlayStart(ctx context.Context, play *types.Play) error {
	// Default implementation does nothing
	return nil
}

// OnPlayEnd is called when play execution ends
func (p *BaseCallbackPlugin) OnPlayEnd(ctx context.Context, play *types.Play, success bool, duration time.Duration) error {
	// Default implementation does nothing
	return nil
}

// OnTaskStart is called when task execution starts
func (p *BaseCallbackPlugin) OnTaskStart(ctx context.Context, task *types.Task, host types.Host) error {
	// Default implementation does nothing
	return nil
}

// OnTaskEnd is called when task execution ends
func (p *BaseCallbackPlugin) OnTaskEnd(ctx context.Context, task *types.Task, host types.Host, result types.TaskResult) error {
	// Default implementation does nothing
	return nil
}

// OnTaskRetry is called when task is retried
func (p *BaseCallbackPlugin) OnTaskRetry(ctx context.Context, task *types.Task, host types.Host, attempt int, err error) error {
	// Default implementation does nothing
	return nil
}

// CallbackManager manages callback plugins and dispatches events
type CallbackManager struct {
	plugins []CallbackPlugin
}

// NewCallbackManager creates a new callback manager
func NewCallbackManager() *CallbackManager {
	return &CallbackManager{
		plugins: make([]CallbackPlugin, 0),
	}
}

// AddPlugin adds a callback plugin
func (m *CallbackManager) AddPlugin(plugin CallbackPlugin) {
	m.plugins = append(m.plugins, plugin)
}

// RemovePlugin removes a callback plugin
func (m *CallbackManager) RemovePlugin(name string) {
	for i, plugin := range m.plugins {
		if plugin.GetName() == name {
			m.plugins = append(m.plugins[:i], m.plugins[i+1:]...)
			return
		}
	}
}

// OnPlaybookStart dispatches playbook start event to all plugins
func (m *CallbackManager) OnPlaybookStart(ctx context.Context, playbook *types.Playbook) error {
	for _, plugin := range m.plugins {
		if err := plugin.OnPlaybookStart(ctx, playbook); err != nil {
			return err
		}
	}
	return nil
}

// OnPlaybookEnd dispatches playbook end event to all plugins
func (m *CallbackManager) OnPlaybookEnd(ctx context.Context, playbook *types.Playbook, success bool, duration time.Duration) error {
	for _, plugin := range m.plugins {
		if err := plugin.OnPlaybookEnd(ctx, playbook, success, duration); err != nil {
			return err
		}
	}
	return nil
}

// OnPlayStart dispatches play start event to all plugins
func (m *CallbackManager) OnPlayStart(ctx context.Context, play *types.Play) error {
	for _, plugin := range m.plugins {
		if err := plugin.OnPlayStart(ctx, play); err != nil {
			return err
		}
	}
	return nil
}

// OnPlayEnd dispatches play end event to all plugins
func (m *CallbackManager) OnPlayEnd(ctx context.Context, play *types.Play, success bool, duration time.Duration) error {
	for _, plugin := range m.plugins {
		if err := plugin.OnPlayEnd(ctx, play, success, duration); err != nil {
			return err
		}
	}
	return nil
}

// OnTaskStart dispatches task start event to all plugins
func (m *CallbackManager) OnTaskStart(ctx context.Context, task *types.Task, host types.Host) error {
	for _, plugin := range m.plugins {
		if err := plugin.OnTaskStart(ctx, task, host); err != nil {
			return err
		}
	}
	return nil
}

// OnTaskEnd dispatches task end event to all plugins
func (m *CallbackManager) OnTaskEnd(ctx context.Context, task *types.Task, host types.Host, result types.TaskResult) error {
	for _, plugin := range m.plugins {
		if err := plugin.OnTaskEnd(ctx, task, host, result); err != nil {
			return err
		}
	}
	return nil
}

// OnTaskRetry dispatches task retry event to all plugins
func (m *CallbackManager) OnTaskRetry(ctx context.Context, task *types.Task, host types.Host, attempt int, err error) error {
	for _, plugin := range m.plugins {
		if err := plugin.OnTaskRetry(ctx, task, host, attempt, err); err != nil {
			return err
		}
	}
	return nil
}
