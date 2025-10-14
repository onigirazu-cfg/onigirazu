package plugins

import (
	"context"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// PluginType represents the type of plugin
type PluginType string

const (
	// PluginTypeModule represents a module plugin
	PluginTypeModule PluginType = "module"
	// PluginTypeCallback represents a callback plugin
	PluginTypeCallback PluginType = "callback"
	// PluginTypeInventory represents an inventory plugin
	PluginTypeInventory PluginType = "inventory"
	// PluginTypeFilter represents a filter plugin
	PluginTypeFilter PluginType = "filter"
)

// Plugin represents a generic plugin interface
type Plugin interface {
	// GetName returns the plugin name
	GetName() string
	// GetType returns the plugin type
	GetType() PluginType
	// GetVersion returns the plugin version
	GetVersion() string
	// GetDescription returns the plugin description
	GetDescription() string
	// Initialize initializes the plugin with configuration
	Initialize(ctx context.Context, config map[string]interface{}) error
	// Cleanup performs cleanup when plugin is unloaded
	Cleanup(ctx context.Context) error
}

// ModulePlugin represents a module plugin that can execute tasks
type ModulePlugin interface {
	Plugin
	// Execute executes the module with given arguments
	Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
	// Validate validates the module arguments
	Validate(args map[string]interface{}) error
	// GetRequiredArgs returns list of required arguments
	GetRequiredArgs() []string
	// GetOptionalArgs returns list of optional arguments with defaults
	GetOptionalArgs() map[string]interface{}
}

// CallbackPlugin represents a callback plugin that can hook into execution events
type CallbackPlugin interface {
	Plugin
	// OnPlaybookStart is called when playbook execution starts
	OnPlaybookStart(ctx context.Context, playbook *types.Playbook) error
	// OnPlaybookEnd is called when playbook execution ends
	OnPlaybookEnd(ctx context.Context, playbook *types.Playbook, success bool, duration time.Duration) error
	// OnPlayStart is called when play execution starts
	OnPlayStart(ctx context.Context, play *types.Play) error
	// OnPlayEnd is called when play execution ends
	OnPlayEnd(ctx context.Context, play *types.Play, success bool, duration time.Duration) error
	// OnTaskStart is called when task execution starts
	OnTaskStart(ctx context.Context, task *types.Task, host types.Host) error
	// OnTaskEnd is called when task execution ends
	OnTaskEnd(ctx context.Context, task *types.Task, host types.Host, result types.TaskResult) error
	// OnTaskRetry is called when task is retried
	OnTaskRetry(ctx context.Context, task *types.Task, host types.Host, attempt int, err error) error
}

// InventoryPlugin represents an inventory plugin that can provide dynamic inventory
type InventoryPlugin interface {
	Plugin
	// GetHosts returns list of hosts from the inventory source
	GetHosts(ctx context.Context, pattern string) ([]types.Host, error)
	// GetGroups returns list of groups from the inventory source
	GetGroups(ctx context.Context, pattern string) (map[string]*types.Group, error)
	// Refresh refreshes the inventory data from source
	Refresh(ctx context.Context) error
	// GetCacheTTL returns the cache TTL for inventory data
	GetCacheTTL() time.Duration
}

// FilterPlugin represents a filter plugin that can transform data in templates
type FilterPlugin interface {
	Plugin
	// GetFilters returns map of filter names to filter functions
	GetFilters() map[string]FilterFunc
}

// FilterFunc represents a filter function that transforms input data
type FilterFunc func(input interface{}, args ...interface{}) (interface{}, error)

// PluginMetadata contains metadata about a plugin
type PluginMetadata struct {
	Name        string                 `json:"name"`
	Type        PluginType             `json:"type"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author"`
	License     string                 `json:"license"`
	Homepage    string                 `json:"homepage"`
	Tags        []string               `json:"tags"`
	Config      map[string]interface{} `json:"config"`
}

// PluginLoader defines interface for loading plugins
type PluginLoader interface {
	// Load loads a plugin from the given path
	Load(ctx context.Context, path string) (Plugin, error)
	// Unload unloads a plugin
	Unload(ctx context.Context, plugin Plugin) error
	// GetSupportedTypes returns list of supported plugin types
	GetSupportedTypes() []PluginType
}

// PluginRegistry defines interface for plugin registration and management
type PluginRegistry interface {
	// Register registers a plugin
	Register(ctx context.Context, plugin Plugin) error
	// Unregister unregisters a plugin by name
	Unregister(ctx context.Context, name string, pluginType PluginType) error
	// Get retrieves a plugin by name and type
	Get(name string, pluginType PluginType) (Plugin, error)
	// List returns list of registered plugins
	List(pluginType PluginType) []Plugin
	// ListAll returns all registered plugins
	ListAll() map[PluginType][]Plugin
	// GetMetadata returns metadata for a plugin
	GetMetadata(name string, pluginType PluginType) (*PluginMetadata, error)
}
