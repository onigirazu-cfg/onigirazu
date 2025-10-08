package plugins

import (
	"context"
	"fmt"
	"sync"
)

// Manager manages plugin lifecycle and registration
type Manager struct {
	mu       sync.RWMutex
	plugins  map[PluginType]map[string]Plugin
	metadata map[PluginType]map[string]*PluginMetadata
	loader   PluginLoader
}

// NewManager creates a new plugin manager
func NewManager(loader PluginLoader) *Manager {
	return &Manager{
		plugins:  make(map[PluginType]map[string]Plugin),
		metadata: make(map[PluginType]map[string]*PluginMetadata),
		loader:   loader,
	}
}

// Register registers a plugin
func (m *Manager) Register(ctx context.Context, plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pluginType := plugin.GetType()
	name := plugin.GetName()

	// Initialize plugin type map if not exists
	if m.plugins[pluginType] == nil {
		m.plugins[pluginType] = make(map[string]Plugin)
	}
	if m.metadata[pluginType] == nil {
		m.metadata[pluginType] = make(map[string]*PluginMetadata)
	}

	// Check if plugin already registered
	if _, exists := m.plugins[pluginType][name]; exists {
		return fmt.Errorf("plugin '%s' of type '%s' already registered", name, pluginType)
	}

	// Initialize plugin
	if err := plugin.Initialize(ctx, nil); err != nil {
		return fmt.Errorf("failed to initialize plugin '%s': %w", name, err)
	}

	// Register plugin
	m.plugins[pluginType][name] = plugin

	// Store metadata
	m.metadata[pluginType][name] = &PluginMetadata{
		Name:        plugin.GetName(),
		Type:        plugin.GetType(),
		Version:     plugin.GetVersion(),
		Description: plugin.GetDescription(),
	}

	return nil
}

// Unregister unregisters a plugin by name and type
func (m *Manager) Unregister(ctx context.Context, name string, pluginType PluginType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if plugin exists
	plugin, exists := m.plugins[pluginType][name]
	if !exists {
		return fmt.Errorf("plugin '%s' of type '%s' not found", name, pluginType)
	}

	// Cleanup plugin
	if err := plugin.Cleanup(ctx); err != nil {
		return fmt.Errorf("failed to cleanup plugin '%s': %w", name, err)
	}

	// Remove plugin
	delete(m.plugins[pluginType], name)
	delete(m.metadata[pluginType], name)

	return nil
}

// Get retrieves a plugin by name and type
func (m *Manager) Get(name string, pluginType PluginType) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins, exists := m.plugins[pluginType]
	if !exists {
		return nil, fmt.Errorf("no plugins of type '%s' registered", pluginType)
	}

	plugin, exists := plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin '%s' of type '%s' not found", name, pluginType)
	}

	return plugin, nil
}

// GetModule retrieves a module plugin by name
func (m *Manager) GetModule(name string) (ModulePlugin, error) {
	plugin, err := m.Get(name, PluginTypeModule)
	if err != nil {
		return nil, err
	}

	modulePlugin, ok := plugin.(ModulePlugin)
	if !ok {
		return nil, fmt.Errorf("plugin '%s' is not a module plugin", name)
	}

	return modulePlugin, nil
}

// GetCallback retrieves a callback plugin by name
func (m *Manager) GetCallback(name string) (CallbackPlugin, error) {
	plugin, err := m.Get(name, PluginTypeCallback)
	if err != nil {
		return nil, err
	}

	callbackPlugin, ok := plugin.(CallbackPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin '%s' is not a callback plugin", name)
	}

	return callbackPlugin, nil
}

// GetInventory retrieves an inventory plugin by name
func (m *Manager) GetInventory(name string) (InventoryPlugin, error) {
	plugin, err := m.Get(name, PluginTypeInventory)
	if err != nil {
		return nil, err
	}

	inventoryPlugin, ok := plugin.(InventoryPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin '%s' is not an inventory plugin", name)
	}

	return inventoryPlugin, nil
}

// GetFilter retrieves a filter plugin by name
func (m *Manager) GetFilter(name string) (FilterPlugin, error) {
	plugin, err := m.Get(name, PluginTypeFilter)
	if err != nil {
		return nil, err
	}

	filterPlugin, ok := plugin.(FilterPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin '%s' is not a filter plugin", name)
	}

	return filterPlugin, nil
}

// List returns list of registered plugins of a specific type
func (m *Manager) List(pluginType PluginType) []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins, exists := m.plugins[pluginType]
	if !exists {
		return []Plugin{}
	}

	result := make([]Plugin, 0, len(plugins))
	for _, plugin := range plugins {
		result = append(result, plugin)
	}

	return result
}

// ListAll returns all registered plugins grouped by type
func (m *Manager) ListAll() map[PluginType][]Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[PluginType][]Plugin)
	for pluginType, plugins := range m.plugins {
		pluginList := make([]Plugin, 0, len(plugins))
		for _, plugin := range plugins {
			pluginList = append(pluginList, plugin)
		}
		result[pluginType] = pluginList
	}

	return result
}

// GetMetadata returns metadata for a plugin
func (m *Manager) GetMetadata(name string, pluginType PluginType) (*PluginMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metadata, exists := m.metadata[pluginType]
	if !exists {
		return nil, fmt.Errorf("no plugins of type '%s' registered", pluginType)
	}

	meta, exists := metadata[name]
	if !exists {
		return nil, fmt.Errorf("plugin '%s' of type '%s' not found", name, pluginType)
	}

	return meta, nil
}

// LoadPlugin loads a plugin from the given path
func (m *Manager) LoadPlugin(ctx context.Context, path string) error {
	if m.loader == nil {
		return fmt.Errorf("no plugin loader configured")
	}

	plugin, err := m.loader.Load(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to load plugin from '%s': %w", path, err)
	}

	return m.Register(ctx, plugin)
}

// UnloadPlugin unloads a plugin
func (m *Manager) UnloadPlugin(ctx context.Context, name string, pluginType PluginType) error {
	plugin, err := m.Get(name, pluginType)
	if err != nil {
		return err
	}

	if err := m.Unregister(ctx, name, pluginType); err != nil {
		return err
	}

	if m.loader != nil {
		if err := m.loader.Unload(ctx, plugin); err != nil {
			return fmt.Errorf("failed to unload plugin '%s': %w", name, err)
		}
	}

	return nil
}

// Shutdown shuts down all plugins
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errors []error

	// Cleanup all plugins
	for pluginType, plugins := range m.plugins {
		for name, plugin := range plugins {
			if err := plugin.Cleanup(ctx); err != nil {
				errors = append(errors, fmt.Errorf("failed to cleanup plugin '%s' of type '%s': %w", name, pluginType, err))
			}
		}
	}

	// Clear all maps
	m.plugins = make(map[PluginType]map[string]Plugin)
	m.metadata = make(map[PluginType]map[string]*PluginMetadata)

	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}

	return nil
}

// GetStats returns statistics about registered plugins
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total"] = 0

	for pluginType, plugins := range m.plugins {
		count := len(plugins)
		stats[string(pluginType)] = count
		stats["total"] = stats["total"].(int) + count
	}

	return stats
}
