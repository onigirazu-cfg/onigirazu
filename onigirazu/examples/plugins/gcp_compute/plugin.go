package main

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/plugins"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// GCPComputeInventoryPlugin provides dynamic inventory from GCP Compute Engine
type GCPComputeInventoryPlugin struct {
	*plugins.BaseInventoryPlugin
	projectID       string
	zone            string
	credentialsFile string
	filters         map[string]string
	cachedHosts     []types.Host
	cachedGroups    map[string]*types.Group
	lastRefresh     time.Time
}

// NewGCPComputeInventoryPlugin creates a new GCP Compute inventory plugin
func NewGCPComputeInventoryPlugin() *GCPComputeInventoryPlugin {
	return &GCPComputeInventoryPlugin{
		BaseInventoryPlugin: plugins.NewBaseInventoryPlugin(
			"gcp_compute",
			"1.0.0",
			"Dynamic inventory from GCP Compute Engine instances",
		),
		filters:      make(map[string]string),
		cachedGroups: make(map[string]*types.Group),
	}
}

// Initialize initializes the plugin with configuration
func (p *GCPComputeInventoryPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	// Call base initialization
	if err := p.BaseInventoryPlugin.Initialize(ctx, config); err != nil {
		return err
	}

	// Extract GCP-specific configuration
	if projectID, ok := config["project_id"].(string); ok {
		p.projectID = projectID
	}

	if zone, ok := config["zone"].(string); ok {
		p.zone = zone
	} else {
		p.zone = "us-central1-a" // Default zone
	}

	if credentialsFile, ok := config["credentials_file"].(string); ok {
		p.credentialsFile = credentialsFile
	}

	// Extract filters
	if filters, ok := config["filters"].(map[string]interface{}); ok {
		for key, value := range filters {
			if strValue, ok := value.(string); ok {
				p.filters[key] = strValue
			}
		}
	}

	return nil
}

// GetHosts returns list of hosts from GCP Compute Engine
func (p *GCPComputeInventoryPlugin) GetHosts(ctx context.Context, pattern string) ([]types.Host, error) {
	// Check if cache is still valid
	if time.Since(p.lastRefresh) < p.GetCacheTTL() && len(p.cachedHosts) > 0 {
		return p.filterHosts(p.cachedHosts, pattern), nil
	}

	// Refresh inventory from GCP
	if err := p.Refresh(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh GCP Compute inventory: %w", err)
	}

	return p.filterHosts(p.cachedHosts, pattern), nil
}

// GetGroups returns list of groups from GCP Compute Engine
func (p *GCPComputeInventoryPlugin) GetGroups(ctx context.Context, pattern string) (map[string]*types.Group, error) {
	// Check if cache is still valid
	if time.Since(p.lastRefresh) < p.GetCacheTTL() && len(p.cachedGroups) > 0 {
		return p.filterGroups(p.cachedGroups, pattern), nil
	}

	// Refresh inventory from GCP
	if err := p.Refresh(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh GCP Compute inventory: %w", err)
	}

	return p.filterGroups(p.cachedGroups, pattern), nil
}

// Refresh refreshes the inventory data from GCP Compute Engine
func (p *GCPComputeInventoryPlugin) Refresh(ctx context.Context) error {
	// NOTE: This is a mock implementation for demonstration purposes
	// In a real implementation, you would use the GCP SDK to query Compute instances
	//
	// Example with GCP SDK:
	// import "cloud.google.com/go/compute/apiv1"
	//
	// client, err := compute.NewInstancesRESTClient(ctx)
	// if err != nil {
	//     return fmt.Errorf("failed to create compute client: %w", err)
	// }
	// defer client.Close()
	//
	// req := &computepb.ListInstancesRequest{
	//     Project: p.projectID,
	//     Zone:    p.zone,
	//     Filter:  proto.String(buildFilter(p.filters)),
	// }
	//
	// it := client.List(ctx, req)
	// for {
	//     instance, err := it.Next()
	//     if err == iterator.Done {
	//         break
	//     }
	//     if err != nil {
	//         return fmt.Errorf("failed to list instances: %w", err)
	//     }
	//     // Parse instance and populate p.cachedHosts and p.cachedGroups
	// }

	// Mock implementation - returns example hosts
	p.cachedHosts = []types.Host{
		{
			Name:    "frontend-1",
			Address: "10.128.0.10",
			Port:    22,
			User:    "gcp-user",
			Vars: map[string]interface{}{
				"instance_id":  "1234567890123456789",
				"machine_type": "e2-medium",
				"zone":         p.zone,
				"project_id":   p.projectID,
				"labels": map[string]string{
					"name":        "frontend-1",
					"environment": "production",
					"role":        "frontend",
				},
			},
		},
		{
			Name:    "frontend-2",
			Address: "10.128.0.11",
			Port:    22,
			User:    "gcp-user",
			Vars: map[string]interface{}{
				"instance_id":  "1234567890123456790",
				"machine_type": "e2-medium",
				"zone":         p.zone,
				"project_id":   p.projectID,
				"labels": map[string]string{
					"name":        "frontend-2",
					"environment": "production",
					"role":        "frontend",
				},
			},
		},
		{
			Name:    "backend-1",
			Address: "10.128.1.10",
			Port:    22,
			User:    "gcp-user",
			Vars: map[string]interface{}{
				"instance_id":  "1234567890123456791",
				"machine_type": "e2-standard-2",
				"zone":         p.zone,
				"project_id":   p.projectID,
				"labels": map[string]string{
					"name":        "backend-1",
					"environment": "production",
					"role":        "backend",
				},
			},
		},
		{
			Name:    "database-1",
			Address: "10.128.2.10",
			Port:    22,
			User:    "gcp-user",
			Vars: map[string]interface{}{
				"instance_id":  "1234567890123456792",
				"machine_type": "n2-standard-4",
				"zone":         p.zone,
				"project_id":   p.projectID,
				"labels": map[string]string{
					"name":        "database-1",
					"environment": "production",
					"role":        "database",
				},
			},
		},
	}

	// Create groups based on labels
	p.cachedGroups = make(map[string]*types.Group)

	// Group by role
	frontendGroup := &types.Group{
		Name:  "frontend",
		Hosts: make(map[string]*types.Host),
		Variables: map[string]interface{}{
			"group_type": "role",
		},
	}
	backendGroup := &types.Group{
		Name:  "backend",
		Hosts: make(map[string]*types.Host),
		Variables: map[string]interface{}{
			"group_type": "role",
		},
	}
	dbGroup := &types.Group{
		Name:  "database",
		Hosts: make(map[string]*types.Host),
		Variables: map[string]interface{}{
			"group_type": "role",
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

	// Group by zone
	zoneGroup := &types.Group{
		Name:  p.zone,
		Hosts: make(map[string]*types.Host),
		Variables: map[string]interface{}{
			"group_type": "zone",
		},
	}

	// Assign hosts to groups
	for i := range p.cachedHosts {
		host := &p.cachedHosts[i]
		if labels, ok := host.Vars["labels"].(map[string]string); ok {
			if role, ok := labels["role"]; ok {
				switch role {
				case "frontend":
					frontendGroup.Hosts[host.Name] = host
				case "backend":
					backendGroup.Hosts[host.Name] = host
				case "database":
					dbGroup.Hosts[host.Name] = host
				}
			}
			if env, ok := labels["environment"]; ok && env == "production" {
				prodGroup.Hosts[host.Name] = host
			}
		}
		// All hosts are in the same zone in this mock
		zoneGroup.Hosts[host.Name] = host
	}

	p.cachedGroups["frontend"] = frontendGroup
	p.cachedGroups["backend"] = backendGroup
	p.cachedGroups["database"] = dbGroup
	p.cachedGroups["production"] = prodGroup
	p.cachedGroups[p.zone] = zoneGroup

	p.lastRefresh = time.Now()
	return nil
}

// filterHosts filters hosts by pattern
func (p *GCPComputeInventoryPlugin) filterHosts(hosts []types.Host, pattern string) []types.Host {
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
func (p *GCPComputeInventoryPlugin) filterGroups(groups map[string]*types.Group, pattern string) map[string]*types.Group {
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
var Plugin plugins.InventoryPlugin = NewGCPComputeInventoryPlugin()

// main is required for Go plugins but not used
func main() {}
