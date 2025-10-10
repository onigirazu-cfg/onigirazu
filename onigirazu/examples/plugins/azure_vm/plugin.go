package main

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/plugins"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// AzureVMInventoryPlugin provides dynamic inventory from Azure Virtual Machines
type AzureVMInventoryPlugin struct {
	*plugins.BaseInventoryPlugin
	subscriptionID string
	tenantID       string
	clientID       string
	clientSecret   string
	resourceGroup  string
	cachedHosts    []types.Host
	cachedGroups   map[string]*types.Group
	lastRefresh    time.Time
}

// NewAzureVMInventoryPlugin creates a new Azure VM inventory plugin
func NewAzureVMInventoryPlugin() *AzureVMInventoryPlugin {
	return &AzureVMInventoryPlugin{
		BaseInventoryPlugin: plugins.NewBaseInventoryPlugin(
			"azure_vm",
			"1.0.0",
			"Dynamic inventory from Azure Virtual Machines",
		),
		cachedGroups: make(map[string]*types.Group),
	}
}

// Initialize initializes the plugin with configuration
func (p *AzureVMInventoryPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	// Call base initialization
	if err := p.BaseInventoryPlugin.Initialize(ctx, config); err != nil {
		return err
	}

	// Extract Azure-specific configuration
	if subscriptionID, ok := config["subscription_id"].(string); ok {
		p.subscriptionID = subscriptionID
	}

	if tenantID, ok := config["tenant_id"].(string); ok {
		p.tenantID = tenantID
	}

	if clientID, ok := config["client_id"].(string); ok {
		p.clientID = clientID
	}

	if clientSecret, ok := config["client_secret"].(string); ok {
		p.clientSecret = clientSecret
	}

	if resourceGroup, ok := config["resource_group"].(string); ok {
		p.resourceGroup = resourceGroup
	}

	return nil
}

// GetHosts returns list of hosts from Azure VMs
func (p *AzureVMInventoryPlugin) GetHosts(ctx context.Context, pattern string) ([]types.Host, error) {
	// Check if cache is still valid
	if time.Since(p.lastRefresh) < p.GetCacheTTL() && len(p.cachedHosts) > 0 {
		return p.filterHosts(p.cachedHosts, pattern), nil
	}

	// Refresh inventory from Azure
	if err := p.Refresh(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh Azure VM inventory: %w", err)
	}

	return p.filterHosts(p.cachedHosts, pattern), nil
}

// GetGroups returns list of groups from Azure VMs
func (p *AzureVMInventoryPlugin) GetGroups(ctx context.Context, pattern string) (map[string]*types.Group, error) {
	// Check if cache is still valid
	if time.Since(p.lastRefresh) < p.GetCacheTTL() && len(p.cachedGroups) > 0 {
		return p.filterGroups(p.cachedGroups, pattern), nil
	}

	// Refresh inventory from Azure
	if err := p.Refresh(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh Azure VM inventory: %w", err)
	}

	return p.filterGroups(p.cachedGroups, pattern), nil
}

// Refresh refreshes the inventory data from Azure VMs
func (p *AzureVMInventoryPlugin) Refresh(ctx context.Context) error {
	// NOTE: This is a mock implementation for demonstration purposes
	// In a real implementation, you would use the Azure SDK to query VMs
	//
	// Example with Azure SDK:
	// import "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	//
	// cred, err := azidentity.NewClientSecretCredential(p.tenantID, p.clientID, p.clientSecret, nil)
	// if err != nil {
	//     return fmt.Errorf("failed to create credential: %w", err)
	// }
	//
	// client, err := armcompute.NewVirtualMachinesClient(p.subscriptionID, cred, nil)
	// if err != nil {
	//     return fmt.Errorf("failed to create VM client: %w", err)
	// }
	//
	// pager := client.NewListPager(p.resourceGroup, nil)
	// for pager.More() {
	//     page, err := pager.NextPage(ctx)
	//     if err != nil {
	//         return fmt.Errorf("failed to list VMs: %w", err)
	//     }
	//     // Parse VMs and populate p.cachedHosts and p.cachedGroups
	// }

	// Mock implementation - returns example hosts
	p.cachedHosts = []types.Host{
		{
			Name:    "app-vm-1",
			Address: "10.1.0.10",
			Port:    22,
			User:    "azureuser",
			Vars: map[string]interface{}{
				"vm_id":           "/subscriptions/xxx/resourceGroups/mygroup/providers/Microsoft.Compute/virtualMachines/app-vm-1",
				"vm_size":         "Standard_B2s",
				"location":        "eastus",
				"resource_group":  p.resourceGroup,
				"subscription_id": p.subscriptionID,
				"tags": map[string]string{
					"Name":        "app-vm-1",
					"Environment": "production",
					"Tier":        "application",
				},
			},
		},
		{
			Name:    "app-vm-2",
			Address: "10.1.0.11",
			Port:    22,
			User:    "azureuser",
			Vars: map[string]interface{}{
				"vm_id":           "/subscriptions/xxx/resourceGroups/mygroup/providers/Microsoft.Compute/virtualMachines/app-vm-2",
				"vm_size":         "Standard_B2s",
				"location":        "eastus",
				"resource_group":  p.resourceGroup,
				"subscription_id": p.subscriptionID,
				"tags": map[string]string{
					"Name":        "app-vm-2",
					"Environment": "production",
					"Tier":        "application",
				},
			},
		},
		{
			Name:    "db-vm-1",
			Address: "10.1.1.10",
			Port:    22,
			User:    "azureuser",
			Vars: map[string]interface{}{
				"vm_id":           "/subscriptions/xxx/resourceGroups/mygroup/providers/Microsoft.Compute/virtualMachines/db-vm-1",
				"vm_size":         "Standard_D2s_v3",
				"location":        "eastus",
				"resource_group":  p.resourceGroup,
				"subscription_id": p.subscriptionID,
				"tags": map[string]string{
					"Name":        "db-vm-1",
					"Environment": "production",
					"Tier":        "database",
				},
			},
		},
	}

	// Create groups based on tags
	p.cachedGroups = make(map[string]*types.Group)

	// Group by tier
	appGroup := &types.Group{
		Name:  "application",
		Hosts: make(map[string]*types.Host),
		Variables: map[string]interface{}{
			"group_type": "tier",
		},
	}
	dbGroup := &types.Group{
		Name:  "database",
		Hosts: make(map[string]*types.Host),
		Variables: map[string]interface{}{
			"group_type": "tier",
		},
	}

	// Group by location
	eastusGroup := &types.Group{
		Name:  "eastus",
		Hosts: make(map[string]*types.Host),
		Variables: map[string]interface{}{
			"group_type": "location",
		},
	}

	// Group by environment
	prodGroup := &types.Group{
		Name:  "production",
		Hosts: make(map[string]*types.Host),
		Variables: map[string]interface{}{
			"group_type": "environment",
		},
	}

	// Assign hosts to groups
	for i := range p.cachedHosts {
		host := &p.cachedHosts[i]
		if tags, ok := host.Vars["tags"].(map[string]string); ok {
			if tier, ok := tags["Tier"]; ok {
				switch tier {
				case "application":
					appGroup.Hosts[host.Name] = host
				case "database":
					dbGroup.Hosts[host.Name] = host
				}
			}
			if env, ok := tags["Environment"]; ok && env == "production" {
				prodGroup.Hosts[host.Name] = host
			}
		}
		if location, ok := host.Vars["location"].(string); ok && location == "eastus" {
			eastusGroup.Hosts[host.Name] = host
		}
	}

	p.cachedGroups["application"] = appGroup
	p.cachedGroups["database"] = dbGroup
	p.cachedGroups["eastus"] = eastusGroup
	p.cachedGroups["production"] = prodGroup

	p.lastRefresh = time.Now()
	return nil
}

// filterHosts filters hosts by pattern
func (p *AzureVMInventoryPlugin) filterHosts(hosts []types.Host, pattern string) []types.Host {
	if pattern == "" || pattern == "*" || pattern == "all" {
		return hosts
	}

	filtered := []types.Host{}
	for _, host := range hosts {
		if matchesPattern(host.Name, pattern) {
			filtered = append(filtered, host)
		}
	}
	return filtered
}

// filterGroups filters groups by pattern
func (p *AzureVMInventoryPlugin) filterGroups(groups map[string]*types.Group, pattern string) map[string]*types.Group {
	if pattern == "" || pattern == "*" || pattern == "all" {
		return groups
	}

	filtered := make(map[string]*types.Group)
	for name, group := range groups {
		if matchesPattern(name, pattern) {
			filtered[name] = group
		}
	}
	return filtered
}

// matchesPattern checks if a string matches a pattern (simple wildcard matching)
func matchesPattern(str, pattern string) bool {
	// Simple implementation - in production, use proper glob matching
	if pattern == "*" || pattern == "all" {
		return true
	}
	return str == pattern
}

// Plugin exports the plugin instance
var Plugin plugins.InventoryPlugin = NewAzureVMInventoryPlugin()

// main is required for Go plugins but not used
func main() {}
