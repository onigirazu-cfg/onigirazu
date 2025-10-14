package plugins

import (
	"context"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// BaseInventoryPlugin provides a base implementation for inventory plugins
type BaseInventoryPlugin struct {
	name        string
	version     string
	description string
	cacheTTL    time.Duration
}

// NewBaseInventoryPlugin creates a new base inventory plugin
func NewBaseInventoryPlugin(name, version, description string) *BaseInventoryPlugin {
	return &BaseInventoryPlugin{
		name:        name,
		version:     version,
		description: description,
		cacheTTL:    5 * time.Minute, // Default cache TTL
	}
}

// GetName returns the plugin name
func (p *BaseInventoryPlugin) GetName() string {
	return p.name
}

// GetType returns the plugin type
func (p *BaseInventoryPlugin) GetType() PluginType {
	return PluginTypeInventory
}

// GetVersion returns the plugin version
func (p *BaseInventoryPlugin) GetVersion() string {
	return p.version
}

// GetDescription returns the plugin description
func (p *BaseInventoryPlugin) GetDescription() string {
	return p.description
}

// Initialize initializes the plugin with configuration
func (p *BaseInventoryPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	// Set cache TTL from config if provided
	if ttl, exists := config["cache_ttl"]; exists {
		if duration, ok := ttl.(time.Duration); ok {
			p.cacheTTL = duration
		} else if seconds, ok := ttl.(int); ok {
			p.cacheTTL = time.Duration(seconds) * time.Second
		}
	}
	return nil
}

// Cleanup performs cleanup when plugin is unloaded
func (p *BaseInventoryPlugin) Cleanup(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// GetCacheTTL returns the cache TTL for inventory data
func (p *BaseInventoryPlugin) GetCacheTTL() time.Duration {
	return p.cacheTTL
}

// SetCacheTTL sets the cache TTL for inventory data
func (p *BaseInventoryPlugin) SetCacheTTL(ttl time.Duration) {
	p.cacheTTL = ttl
}

// GetHosts returns list of hosts from the inventory source
// This is a default implementation that should be overridden
func (p *BaseInventoryPlugin) GetHosts(ctx context.Context, pattern string) ([]types.Host, error) {
	return []types.Host{}, nil
}

// GetGroups returns list of groups from the inventory source
// This is a default implementation that should be overridden
func (p *BaseInventoryPlugin) GetGroups(ctx context.Context, pattern string) (map[string]*types.Group, error) {
	return make(map[string]*types.Group), nil
}

// Refresh refreshes the inventory data from source
// This is a default implementation that should be overridden
func (p *BaseInventoryPlugin) Refresh(ctx context.Context) error {
	return nil
}
