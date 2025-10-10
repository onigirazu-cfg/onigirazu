package plugins

import (
	"context"
	"fmt"
)

// BaseModulePlugin provides a base implementation for module plugins
type BaseModulePlugin struct {
	name         string
	version      string
	description  string
	requiredArgs []string
	optionalArgs map[string]interface{}
}

// NewBaseModulePlugin creates a new base module plugin
func NewBaseModulePlugin(name, version, description string) *BaseModulePlugin {
	return &BaseModulePlugin{
		name:         name,
		version:      version,
		description:  description,
		requiredArgs: []string{},
		optionalArgs: make(map[string]interface{}),
	}
}

// GetName returns the plugin name
func (p *BaseModulePlugin) GetName() string {
	return p.name
}

// GetType returns the plugin type
func (p *BaseModulePlugin) GetType() PluginType {
	return PluginTypeModule
}

// GetVersion returns the plugin version
func (p *BaseModulePlugin) GetVersion() string {
	return p.version
}

// GetDescription returns the plugin description
func (p *BaseModulePlugin) GetDescription() string {
	return p.description
}

// Initialize initializes the plugin with configuration
func (p *BaseModulePlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	// Default implementation does nothing
	return nil
}

// Cleanup performs cleanup when plugin is unloaded
func (p *BaseModulePlugin) Cleanup(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// GetRequiredArgs returns list of required arguments
func (p *BaseModulePlugin) GetRequiredArgs() []string {
	return p.requiredArgs
}

// GetOptionalArgs returns list of optional arguments with defaults
func (p *BaseModulePlugin) GetOptionalArgs() map[string]interface{} {
	return p.optionalArgs
}

// SetRequiredArgs sets the required arguments
func (p *BaseModulePlugin) SetRequiredArgs(args []string) {
	p.requiredArgs = args
}

// SetOptionalArgs sets the optional arguments with defaults
func (p *BaseModulePlugin) SetOptionalArgs(args map[string]interface{}) {
	p.optionalArgs = args
}

// AddRequiredArg adds a required argument
func (p *BaseModulePlugin) AddRequiredArg(arg string) {
	p.requiredArgs = append(p.requiredArgs, arg)
}

// AddOptionalArg adds an optional argument with default value
func (p *BaseModulePlugin) AddOptionalArg(arg string, defaultValue interface{}) {
	p.optionalArgs[arg] = defaultValue
}

// Validate validates the module arguments
func (p *BaseModulePlugin) Validate(args map[string]interface{}) error {
	// Check required arguments
	for _, requiredArg := range p.requiredArgs {
		if _, exists := args[requiredArg]; !exists {
			return fmt.Errorf("required argument '%s' is missing", requiredArg)
		}
	}

	return nil
}

// Helper functions for argument parsing

// GetStringArg gets a string argument with default value
func GetStringArg(args map[string]interface{}, key string, defaultValue string) string {
	if val, exists := args[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetBoolArg gets a boolean argument with default value
func GetBoolArg(args map[string]interface{}, key string, defaultValue bool) bool {
	if val, exists := args[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetIntArg gets an integer argument with default value
func GetIntArg(args map[string]interface{}, key string, defaultValue int) int {
	if val, exists := args[key]; exists {
		if i, ok := val.(int); ok {
			return i
		}
		if f, ok := val.(float64); ok {
			return int(f)
		}
	}
	return defaultValue
}

// GetMapArg gets a map argument with default value
func GetMapArg(args map[string]interface{}, key string, defaultValue map[string]interface{}) map[string]interface{} {
	if val, exists := args[key]; exists {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	return defaultValue
}

// GetSliceArg gets a slice argument with default value
func GetSliceArg(args map[string]interface{}, key string, defaultValue []interface{}) []interface{} {
	if val, exists := args[key]; exists {
		if s, ok := val.([]interface{}); ok {
			return s
		}
	}
	return defaultValue
}

// GetStringSliceArg gets a string slice argument with default value
func GetStringSliceArg(args map[string]interface{}, key string, defaultValue []string) []string {
	if val, exists := args[key]; exists {
		if s, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(s))
			for _, item := range s {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
		if s, ok := val.([]string); ok {
			return s
		}
	}
	return defaultValue
}
