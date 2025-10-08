package plugins

import (
	"context"
	"fmt"
	"strings"
)

// BaseFilterPlugin provides a base implementation for filter plugins
type BaseFilterPlugin struct {
	name        string
	version     string
	description string
	filters     map[string]FilterFunc
}

// NewBaseFilterPlugin creates a new base filter plugin
func NewBaseFilterPlugin(name, version, description string) *BaseFilterPlugin {
	return &BaseFilterPlugin{
		name:        name,
		version:     version,
		description: description,
		filters:     make(map[string]FilterFunc),
	}
}

// GetName returns the plugin name
func (p *BaseFilterPlugin) GetName() string {
	return p.name
}

// GetType returns the plugin type
func (p *BaseFilterPlugin) GetType() PluginType {
	return PluginTypeFilter
}

// GetVersion returns the plugin version
func (p *BaseFilterPlugin) GetVersion() string {
	return p.version
}

// GetDescription returns the plugin description
func (p *BaseFilterPlugin) GetDescription() string {
	return p.description
}

// Initialize initializes the plugin with configuration
func (p *BaseFilterPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	// Default implementation does nothing
	return nil
}

// Cleanup performs cleanup when plugin is unloaded
func (p *BaseFilterPlugin) Cleanup(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// GetFilters returns map of filter names to filter functions
func (p *BaseFilterPlugin) GetFilters() map[string]FilterFunc {
	return p.filters
}

// AddFilter adds a filter function
func (p *BaseFilterPlugin) AddFilter(name string, fn FilterFunc) {
	p.filters[name] = fn
}

// RemoveFilter removes a filter function
func (p *BaseFilterPlugin) RemoveFilter(name string) {
	delete(p.filters, name)
}

// BuiltinFiltersPlugin provides built-in filter functions
type BuiltinFiltersPlugin struct {
	*BaseFilterPlugin
}

// NewBuiltinFiltersPlugin creates a new built-in filters plugin
func NewBuiltinFiltersPlugin() *BuiltinFiltersPlugin {
	plugin := &BuiltinFiltersPlugin{
		BaseFilterPlugin: NewBaseFilterPlugin("builtin", "1.0.0", "Built-in filter functions"),
	}

	// Register built-in filters
	plugin.AddFilter("upper", UpperFilter)
	plugin.AddFilter("lower", LowerFilter)
	plugin.AddFilter("title", TitleFilter)
	plugin.AddFilter("trim", TrimFilter)
	plugin.AddFilter("replace", ReplaceFilter)
	plugin.AddFilter("default", DefaultFilter)
	plugin.AddFilter("length", LengthFilter)
	plugin.AddFilter("join", JoinFilter)
	plugin.AddFilter("split", SplitFilter)

	return plugin
}

// UpperFilter converts string to uppercase
func UpperFilter(input interface{}, args ...interface{}) (interface{}, error) {
	str, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("upper filter expects string input, got %T", input)
	}
	return strings.ToUpper(str), nil
}

// LowerFilter converts string to lowercase
func LowerFilter(input interface{}, args ...interface{}) (interface{}, error) {
	str, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("lower filter expects string input, got %T", input)
	}
	return strings.ToLower(str), nil
}

// TitleFilter converts string to title case
func TitleFilter(input interface{}, args ...interface{}) (interface{}, error) {
	str, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("title filter expects string input, got %T", input)
	}
	return strings.Title(str), nil
}

// TrimFilter trims whitespace from string
func TrimFilter(input interface{}, args ...interface{}) (interface{}, error) {
	str, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("trim filter expects string input, got %T", input)
	}
	return strings.TrimSpace(str), nil
}

// ReplaceFilter replaces occurrences in string
func ReplaceFilter(input interface{}, args ...interface{}) (interface{}, error) {
	str, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("replace filter expects string input, got %T", input)
	}

	if len(args) < 2 {
		return nil, fmt.Errorf("replace filter requires 2 arguments: old and new")
	}

	old, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("replace filter expects string for old argument, got %T", args[0])
	}

	new, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("replace filter expects string for new argument, got %T", args[1])
	}

	return strings.ReplaceAll(str, old, new), nil
}

// DefaultFilter returns default value if input is empty
func DefaultFilter(input interface{}, args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("default filter requires 1 argument: default value")
	}

	// Check if input is nil or empty
	if input == nil {
		return args[0], nil
	}

	if str, ok := input.(string); ok && str == "" {
		return args[0], nil
	}

	return input, nil
}

// LengthFilter returns length of string or slice
func LengthFilter(input interface{}, args ...interface{}) (interface{}, error) {
	switch v := input.(type) {
	case string:
		return len(v), nil
	case []interface{}:
		return len(v), nil
	case []string:
		return len(v), nil
	case map[string]interface{}:
		return len(v), nil
	default:
		return nil, fmt.Errorf("length filter expects string, slice, or map input, got %T", input)
	}
}

// JoinFilter joins slice elements with separator
func JoinFilter(input interface{}, args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("join filter requires 1 argument: separator")
	}

	separator, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("join filter expects string for separator argument, got %T", args[0])
	}

	switch v := input.(type) {
	case []interface{}:
		strs := make([]string, len(v))
		for i, item := range v {
			strs[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(strs, separator), nil
	case []string:
		return strings.Join(v, separator), nil
	default:
		return nil, fmt.Errorf("join filter expects slice input, got %T", input)
	}
}

// SplitFilter splits string by separator
func SplitFilter(input interface{}, args ...interface{}) (interface{}, error) {
	str, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("split filter expects string input, got %T", input)
	}

	if len(args) < 1 {
		return nil, fmt.Errorf("split filter requires 1 argument: separator")
	}

	separator, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("split filter expects string for separator argument, got %T", args[0])
	}

	return strings.Split(str, separator), nil
}
