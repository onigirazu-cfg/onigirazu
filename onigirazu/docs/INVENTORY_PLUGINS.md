# Inventory Plugins Documentation

## Overview

Onigirazu supports dynamic inventory plugins that can fetch host information from various cloud providers and infrastructure sources. This document describes the inventory plugin system and provides examples for AWS EC2, Azure VM, and GCP Compute.

## Architecture

### Plugin Interface

All inventory plugins must implement the `InventoryPlugin` interface:

```go
type InventoryPlugin interface {
    Plugin
    GetHosts(ctx context.Context, pattern string) ([]types.Host, error)
    GetGroups(ctx context.Context, pattern string) (map[string]*types.Group, error)
    Refresh(ctx context.Context) error
}
```

### Base Plugin

The `BaseInventoryPlugin` provides common functionality:

- Plugin metadata (name, version, description)
- Configuration management
- Lifecycle management (Initialize, Shutdown)

## Available Plugins

### 1. AWS EC2 Inventory Plugin

**Location:** `examples/plugins/aws_ec2/plugin.go`

**Description:** Fetches inventory from AWS EC2 instances.

**Configuration:**

```yaml
plugins:
  inventory:
    - name: aws_ec2
      type: inventory
      path: ./examples/plugins/aws_ec2/plugin.so
      enabled: true
      config:
        region: us-east-1
        access_key_id: YOUR_ACCESS_KEY
        secret_access_key: YOUR_SECRET_KEY
        filters:
          instance-state-name: running
```

**Features:**

- Groups hosts by:
  - Instance tags (e.g., `tag_Name_web`, `tag_Environment_production`)
  - Instance type (e.g., `type_t2_micro`)
  - Availability zone (e.g., `us-east-1a`)
  - Security groups
- Provides host variables:
  - `instance_id` - EC2 instance ID
  - `instance_type` - EC2 instance type
  - `availability_zone` - AZ where instance is running
  - `private_ip` - Private IP address
  - `public_ip` - Public IP address (if available)
  - `tags` - All instance tags

**Mock Data (for testing):**

The plugin includes mock data with 4 sample hosts:

- `web-server-1`, `web-server-2` (frontend role)
- `api-server-1` (backend role)
- `db-server-1` (database role)

### 2. Azure VM Inventory Plugin

**Location:** `examples/plugins/azure_vm/plugin.go`

**Description:** Fetches inventory from Azure Virtual Machines.

**Configuration:**

```yaml
plugins:
  inventory:
    - name: azure_vm
      type: inventory
      path: ./examples/plugins/azure_vm/plugin.so
      enabled: true
      config:
        subscription_id: YOUR_SUBSCRIPTION_ID
        tenant_id: YOUR_TENANT_ID
        client_id: YOUR_CLIENT_ID
        client_secret: YOUR_CLIENT_SECRET
        resource_group: my-resource-group
        filters:
          powerState: running
```

**Features:**

- Groups hosts by:
  - VM tags (e.g., `tag_role_frontend`)
  - Resource group
  - Location (Azure region)
  - VM size
- Provides host variables:
  - `vm_id` - Azure VM resource ID
  - `vm_size` - VM size (e.g., Standard_D2s_v3)
  - `location` - Azure region
  - `resource_group` - Resource group name
  - `private_ip` - Private IP address
  - `public_ip` - Public IP address (if available)
  - `tags` - All VM tags

**Mock Data (for testing):**

The plugin includes mock data with 4 sample hosts:

- `frontend-vm-1`, `frontend-vm-2` (frontend role)
- `backend-vm-1` (backend role)
- `database-vm-1` (database role)

### 3. GCP Compute Inventory Plugin

**Location:** `examples/plugins/gcp_compute/plugin.go`

**Description:** Fetches inventory from Google Cloud Platform Compute Engine instances.

**Configuration:**

```yaml
plugins:
  inventory:
    - name: gcp_compute
      type: inventory
      path: ./examples/plugins/gcp_compute/plugin.so
      enabled: true
      config:
        project_id: my-gcp-project
        zone: us-central1-a
        credentials_file: /path/to/service-account.json
        filters:
          status: RUNNING
```

**Features:**

- Groups hosts by:
  - Instance labels (e.g., `label_role_frontend`)
  - Zone
  - Machine type
  - Network tags
- Provides host variables:
  - `instance_id` - GCP instance ID
  - `machine_type` - Machine type (e.g., e2-medium)
  - `zone` - GCP zone
  - `project_id` - GCP project ID
  - `private_ip` - Private IP address
  - `public_ip` - External IP address (if available)
  - `labels` - All instance labels

**Mock Data (for testing):**

The plugin includes mock data with 4 sample hosts:

- `frontend-1`, `frontend-2` (frontend role)
- `backend-1` (backend role)
- `database-1` (database role)

## Building Plugins

To build a plugin as a shared library (.so file):

```bash
# AWS EC2 Plugin
go build -buildmode=plugin -o aws_ec2.so examples/plugins/aws_ec2/plugin.go

# Azure VM Plugin
go build -buildmode=plugin -o azure_vm.so examples/plugins/azure_vm/plugin.go

# GCP Compute Plugin
go build -buildmode=plugin -o gcp_compute.so examples/plugins/gcp_compute/plugin.go
```

**Note:** Go plugins must be built with the same Go version and on the same platform as the main application.

## Using Inventory Plugins

### 1. Configuration File

Create a `plugins.yml` configuration file:

```yaml
plugins:
  inventory:
    - name: aws_ec2
      type: inventory
      path: ./plugins/aws_ec2.so
      enabled: true
      config:
        region: us-east-1
        filters:
          instance-state-name: running
```

### 2. Load Plugins in Code

```go
import (
    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
    "github.com/onigirazu-cfg/onigirazu/internal/inventory"
)

// Load plugin configuration
config, err := plugins.LoadConfig("plugins.yml")
if err != nil {
    log.Fatal(err)
}

// Create plugin manager
loader := plugins.NewGoPluginLoader()
manager := plugins.NewManager(loader)

// Load plugins from config
if err := plugins.LoadPluginsFromConfig(ctx, manager, config); err != nil {
    log.Fatal(err)
}

// Get inventory plugin
plugin, err := manager.Get("aws_ec2")
if err != nil {
    log.Fatal(err)
}

inventoryPlugin := plugin.(plugins.InventoryPlugin)

// Fetch hosts
hosts, err := inventoryPlugin.GetHosts(ctx, "*")
if err != nil {
    log.Fatal(err)
}

// Fetch groups
groups, err := inventoryPlugin.GetGroups(ctx, "*")
if err != nil {
    log.Fatal(err)
}
```

### 3. Integration with Inventory Manager

The inventory manager can use plugins as a source:

```go
// Create inventory manager
invManager := inventory.NewManager()

// Add plugin as source
invManager.AddPlugin(inventoryPlugin)

// Refresh inventory from all sources
if err := invManager.Refresh(ctx); err != nil {
    log.Fatal(err)
}

// Get hosts from inventory
hosts := invManager.GetHosts("*")
```

## Creating Custom Inventory Plugins

### Step 1: Define Plugin Structure

```go
package main

import (
    "context"
    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type MyInventoryPlugin struct {
    *plugins.BaseInventoryPlugin
    // Add your custom fields
    apiEndpoint string
    apiKey      string
}
```

### Step 2: Implement Constructor

```go
func NewMyInventoryPlugin() *MyInventoryPlugin {
    return &MyInventoryPlugin{
        BaseInventoryPlugin: plugins.NewBaseInventoryPlugin(
            "my_inventory",
            "1.0.0",
            "My custom inventory plugin",
        ),
    }
}
```

### Step 3: Implement Initialize

```go
func (p *MyInventoryPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
    // Call base initialization
    if err := p.BaseInventoryPlugin.Initialize(ctx, config); err != nil {
        return err
    }

    // Parse custom configuration
    if endpoint, ok := config["api_endpoint"].(string); ok {
        p.apiEndpoint = endpoint
    }
    if key, ok := config["api_key"].(string); ok {
        p.apiKey = key
    }

    return nil
}
```

### Step 4: Implement GetHosts

```go
func (p *MyInventoryPlugin) GetHosts(ctx context.Context, pattern string) ([]types.Host, error) {
    // Fetch hosts from your source
    hosts := []types.Host{
        {
            Name:    "host1",
            Address: "192.168.1.10",
            Port:    22,
            User:    "admin",
            Vars: map[string]interface{}{
                "custom_var": "value",
            },
        },
    }

    // Filter by pattern if needed
    return filterHosts(hosts, pattern), nil
}
```

### Step 5: Implement GetGroups

```go
func (p *MyInventoryPlugin) GetGroups(ctx context.Context, pattern string) (map[string]*types.Group, error) {
    groups := make(map[string]*types.Group)

    // Create groups
    webGroup := &types.Group{
        Name:  "web",
        Hosts: make(map[string]*types.Host),
        Variables: map[string]interface{}{
            "group_var": "value",
        },
    }

    // Add hosts to groups
    hosts, _ := p.GetHosts(ctx, "*")
    for i := range hosts {
        host := &hosts[i]
        webGroup.Hosts[host.Name] = host
    }

    groups["web"] = webGroup
    return groups, nil
}
```

### Step 6: Implement Refresh

```go
func (p *MyInventoryPlugin) Refresh(ctx context.Context) error {
    // Refresh inventory data from source
    // This is called periodically to update the cache
    return nil
}
```

### Step 7: Export Plugin

```go
// Plugin exports the plugin instance
var Plugin plugins.InventoryPlugin = NewMyInventoryPlugin()

// main is required for Go plugins but not used
func main() {}
```

### Step 8: Build Plugin

```bash
go build -buildmode=plugin -o my_inventory.so my_inventory_plugin.go
```

## Best Practices

1. **Caching:** Cache inventory data and refresh periodically to avoid excessive API calls
2. **Error Handling:** Handle API errors gracefully and provide meaningful error messages
3. **Filtering:** Support pattern matching for hosts and groups (wildcards, regex)
4. **Pagination:** Handle large inventories with pagination
5. **Rate Limiting:** Respect API rate limits of cloud providers
6. **Credentials:** Support multiple authentication methods (env vars, config files, IAM roles)
7. **Metadata:** Include rich metadata as host variables for use in playbooks
8. **Groups:** Create logical groups based on tags, regions, types, etc.
9. **Testing:** Include mock data for testing without real API calls
10. **Documentation:** Document configuration options and available variables

## Troubleshooting

### Plugin Not Loading

- Ensure plugin is built with the same Go version as main application
- Check plugin path in configuration
- Verify plugin exports `Plugin` variable
- Check plugin logs for initialization errors

### No Hosts Returned

- Verify API credentials are correct
- Check filters in configuration
- Ensure instances/VMs are in running state
- Check plugin logs for API errors

### Performance Issues

- Implement caching with appropriate TTL
- Use filters to reduce API response size
- Consider pagination for large inventories
- Monitor API rate limits

## Future Enhancements

- [ ] Support for more cloud providers (DigitalOcean, Linode, etc.)
- [ ] Support for container orchestrators (Kubernetes, Docker Swarm)
- [ ] Support for configuration management databases (CMDB)
- [ ] Automatic credential discovery (IAM roles, managed identities)
- [ ] Inventory caching to disk
- [ ] Inventory diff and change detection
- [ ] Parallel inventory refresh
- [ ] Inventory webhooks for real-time updates

## References

- [Plugin System Documentation](PLUGIN_INTEGRATION.md)
- [AWS EC2 API Documentation](https://docs.aws.amazon.com/ec2/)
- [Azure VM API Documentation](https://docs.microsoft.com/en-us/azure/virtual-machines/)
- [GCP Compute API Documentation](https://cloud.google.com/compute/docs)
