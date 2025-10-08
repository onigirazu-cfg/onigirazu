package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"
)

// GoPluginLoader loads Go plugins using the plugin package
type GoPluginLoader struct {
	mu      sync.RWMutex
	plugins map[string]*plugin.Plugin
}

// NewGoPluginLoader creates a new Go plugin loader
func NewGoPluginLoader() *GoPluginLoader {
	return &GoPluginLoader{
		plugins: make(map[string]*plugin.Plugin),
	}
}

// Load loads a plugin from the given path
func (l *GoPluginLoader) Load(ctx context.Context, path string) (Plugin, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin file not found: %s", path)
	}

	// Load plugin
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for NewPlugin symbol
	symbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return nil, fmt.Errorf("plugin must export 'NewPlugin' function: %w", err)
	}

	// Call NewPlugin function
	newPluginFunc, ok := symbol.(func() Plugin)
	if !ok {
		return nil, fmt.Errorf("NewPlugin must be a function that returns Plugin")
	}

	pluginInstance := newPluginFunc()
	if pluginInstance == nil {
		return nil, fmt.Errorf("NewPlugin returned nil")
	}

	// Store plugin reference
	l.plugins[path] = p

	return pluginInstance, nil
}

// Unload unloads a plugin
func (l *GoPluginLoader) Unload(ctx context.Context, plugin Plugin) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Note: Go plugins cannot be unloaded once loaded
	// This is a limitation of the Go plugin system
	// We just remove the reference from our map
	for path, p := range l.plugins {
		// Try to find the plugin by comparing symbols
		symbol, err := p.Lookup("NewPlugin")
		if err != nil {
			continue
		}

		newPluginFunc, ok := symbol.(func() Plugin)
		if !ok {
			continue
		}

		testPlugin := newPluginFunc()
		if testPlugin.GetName() == plugin.GetName() && testPlugin.GetType() == plugin.GetType() {
			delete(l.plugins, path)
			return nil
		}
	}

	return fmt.Errorf("plugin not found in loader")
}

// GetSupportedTypes returns list of supported plugin types
func (l *GoPluginLoader) GetSupportedTypes() []PluginType {
	return []PluginType{
		PluginTypeModule,
		PluginTypeCallback,
		PluginTypeInventory,
		PluginTypeFilter,
	}
}

// DirectoryLoader loads all plugins from a directory
type DirectoryLoader struct {
	loader PluginLoader
}

// NewDirectoryLoader creates a new directory loader
func NewDirectoryLoader(loader PluginLoader) *DirectoryLoader {
	return &DirectoryLoader{
		loader: loader,
	}
}

// LoadFromDirectory loads all plugins from a directory
func (l *DirectoryLoader) LoadFromDirectory(ctx context.Context, dir string) ([]Plugin, error) {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin directory not found: %s", dir)
	}

	var plugins []Plugin
	var errors []error

	// Walk through directory
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only load .so files (Go plugins)
		if filepath.Ext(path) != ".so" {
			return nil
		}

		// Try to load plugin
		plugin, err := l.loader.Load(ctx, path)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to load plugin from %s: %w", path, err))
			return nil // Continue loading other plugins
		}

		plugins = append(plugins, plugin)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk plugin directory: %w", err)
	}

	if len(errors) > 0 {
		return plugins, fmt.Errorf("some plugins failed to load: %v", errors)
	}

	return plugins, nil
}

// InMemoryLoader loads plugins that are already in memory (for testing)
type InMemoryLoader struct {
	plugins map[string]Plugin
}

// NewInMemoryLoader creates a new in-memory loader
func NewInMemoryLoader() *InMemoryLoader {
	return &InMemoryLoader{
		plugins: make(map[string]Plugin),
	}
}

// AddPlugin adds a plugin to the in-memory loader
func (l *InMemoryLoader) AddPlugin(name string, plugin Plugin) {
	l.plugins[name] = plugin
}

// Load loads a plugin by name
func (l *InMemoryLoader) Load(ctx context.Context, path string) (Plugin, error) {
	plugin, exists := l.plugins[path]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", path)
	}
	return plugin, nil
}

// Unload unloads a plugin
func (l *InMemoryLoader) Unload(ctx context.Context, plugin Plugin) error {
	for name, p := range l.plugins {
		if p.GetName() == plugin.GetName() && p.GetType() == plugin.GetType() {
			delete(l.plugins, name)
			return nil
		}
	}
	return fmt.Errorf("plugin not found in loader")
}

// GetSupportedTypes returns list of supported plugin types
func (l *InMemoryLoader) GetSupportedTypes() []PluginType {
	return []PluginType{
		PluginTypeModule,
		PluginTypeCallback,
		PluginTypeInventory,
		PluginTypeFilter,
	}
}
