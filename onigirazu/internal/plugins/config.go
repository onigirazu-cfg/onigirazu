package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents plugin configuration
type Config struct {
	// PluginsDir is the directory where plugins are located
	PluginsDir string `yaml:"plugins_dir"`
	// Plugins is a list of plugins to load
	Plugins []PluginConfig `yaml:"plugins"`
}

// PluginConfig represents configuration for a single plugin
type PluginConfig struct {
	// Name is the plugin name
	Name string `yaml:"name"`
	// Type is the plugin type (module, callback, inventory, filter)
	Type string `yaml:"type"`
	// Path is the path to the plugin file (for Go plugins)
	Path string `yaml:"path,omitempty"`
	// Enabled indicates if the plugin should be loaded
	Enabled bool `yaml:"enabled"`
	// Config is plugin-specific configuration
	Config map[string]interface{} `yaml:"config,omitempty"`
}

// LoadConfig loads plugin configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	// Clean the path to prevent directory traversal attacks
	cleanPath := filepath.Clean(configPath)

	// #nosec G304 - configPath is expected to be provided by the user/admin
	// and should be validated by the caller. This is a configuration file
	// that needs to be read from a user-specified location.
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default plugins directory if not specified
	if config.PluginsDir == "" {
		config.PluginsDir = "./plugins"
	}

	return &config, nil
}

// LoadPluginsFromConfig loads plugins from configuration
func LoadPluginsFromConfig(ctx context.Context, manager *Manager, config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	for _, pluginConfig := range config.Plugins {
		if !pluginConfig.Enabled {
			continue
		}

		// If path is relative, make it relative to plugins directory
		pluginPath := pluginConfig.Path
		if pluginPath != "" && !filepath.IsAbs(pluginPath) {
			pluginPath = filepath.Join(config.PluginsDir, pluginPath)
		}

		// Load plugin based on type
		switch pluginConfig.Type {
		case "module", "callback", "inventory", "filter":
			// For Go plugins, use the loader
			if pluginPath != "" {
				loader := NewGoPluginLoader()
				plugin, err := loader.Load(ctx, pluginPath)
				if err != nil {
					return fmt.Errorf("failed to load plugin %s: %w", pluginConfig.Name, err)
				}

				// Initialize plugin with config
				if err := plugin.Initialize(ctx, pluginConfig.Config); err != nil {
					return fmt.Errorf("failed to initialize plugin %s: %w", pluginConfig.Name, err)
				}

				// Register plugin
				if err := manager.Register(ctx, plugin); err != nil {
					return fmt.Errorf("failed to register plugin %s: %w", pluginConfig.Name, err)
				}
			}
		default:
			return fmt.Errorf("unknown plugin type: %s", pluginConfig.Type)
		}
	}

	return nil
}

// DefaultConfig returns a default plugin configuration
func DefaultConfig() *Config {
	return &Config{
		PluginsDir: "./plugins",
		Plugins: []PluginConfig{
			{
				Name:    "builtin_filters",
				Type:    "filter",
				Enabled: true,
			},
		},
	}
}

// SaveConfig saves plugin configuration to a YAML file
func SaveConfig(config *Config, configPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Use 0600 permissions for security - only owner can read/write
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
