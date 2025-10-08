package main

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/plugins"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// AWSEC2InventoryPlugin provides dynamic inventory from AWS EC2
type AWSEC2InventoryPlugin struct {
	*plugins.BaseInventoryPlugin
	region          string
	accessKeyID     string
	secretAccessKey string
	filters         map[string]string
	cachedHosts     []types.Host
	cachedGroups    map[string]*types.Group
	lastRefresh     time.Time
}

// NewAWSEC2InventoryPlugin creates a new AWS EC2 inventory plugin
func NewAWSEC2InventoryPlugin() *AWSEC2InventoryPlugin {
	return &AWSEC2InventoryPlugin{
		BaseInventoryPlugin: plugins.NewBaseInventoryPlugin(
			"aws_ec2",
			"1.0.0",
			"Dynamic inventory from AWS EC2 instances",
		),
		filters:      make(map[string]string),
		cachedGroups: make(map[string]*types.Group),
	}
}

// Initialize initializes the plugin with configuration
func (p *AWSEC2InventoryPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
	// Call base initialization
	if err := p.BaseInventoryPlugin.Initialize(ctx, config); err != nil {
		return err
	}

	// Extract AWS-specific configuration
	if region, ok := config["region"].(string); ok {
		p.region = region
	} else {
		p.region = "us-east-1" // Default region
	}

	if accessKeyID, ok := config["access_key_id"].(string); ok {
		p.accessKeyID = accessKeyID
	}

	if secretAccessKey, ok := config["secret_access_key"].(string); ok {
		p.secretAccessKey = secretAccessKey
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

// GetHosts returns list of hosts from AWS EC2
func (p *AWSEC2InventoryPlugin) GetHosts(ctx context.Context, pattern string) ([]types.Host, error) {
	// Check if cache is still valid
	if time.Since(p.lastRefresh) < p.GetCacheTTL() && len(p.cachedHosts) > 0 {
		return p.filterHosts(p.cachedHosts, pattern), nil
	}

	// Refresh inventory from AWS
	if err := p.Refresh(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh AWS EC2 inventory: %w", err)
	}

	return p.filterHosts(p.cachedHosts, pattern), nil
}

// GetGroups returns list of groups from AWS EC2
func (p *AWSEC2InventoryPlugin) GetGroups(ctx context.Context, pattern string) (map[string]*types.Group, error) {
	// Check if cache is still valid
	if time.Since(p.lastRefresh) < p.GetCacheTTL() && len(p.cachedGroups) > 0 {
		return p.filterGroups(p.cachedGroups, pattern), nil
	}

	// Refresh inventory from AWS
	if err := p.Refresh(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh AWS EC2 inventory: %w", err)
	}

	return p.filterGroups(p.cachedGroups, pattern), nil
}

// Refresh refreshes the inventory data from AWS EC2
func (p *AWSEC2InventoryPlugin) Refresh(ctx context.Context) error {
	// NOTE: This is a mock implementation for demonstration purposes
	// In a real implementation, you would use the AWS SDK to query EC2 instances
	//
	// Example with AWS SDK:
	// import "github.com/aws/aws-sdk-go-v2/service/ec2"
	//
	// cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(p.region))
	// if err != nil {
	//     return fmt.Errorf("failed to load AWS config: %w", err)
	// }
	//
	// client := ec2.NewFromConfig(cfg)
	// result, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
	//     Filters: buildFilters(p.filters),
	// })
	// if err != nil {
	//     return fmt.Errorf("failed to describe instances: %w", err)
	// }
	//
	// Parse instances and populate p.cachedHosts and p.cachedGroups

	// Mock implementation - returns example hosts
	p.cachedHosts = []types.Host{
		{
			Name:    "web-server-1",
			Address: "10.0.1.10",
			Port:    22,
			User:    "ec2-user",
			Vars: map[string]interface{}{
				"instance_id":       "i-1234567890abcdef0",
				"instance_type":     "t3.micro",
				"region":            p.region,
				"availability_zone": "us-east-1a",
				"tags": map[string]string{
					"Name":        "web-server-1",
					"Environment": "production",
					"Role":        "web",
				},
			},
		},
		{
			Name:    "web-server-2",
			Address: "10.0.1.11",
			Port:    22,
			User:    "ec2-user",
			Vars: map[string]interface{}{
				"instance_id":       "i-1234567890abcdef1",
				"instance_type":     "t3.micro",
				"region":            p.region,
				"availability_zone": "us-east-1b",
				"tags": map[string]string{
					"Name":        "web-server-2",
					"Environment": "production",
					"Role":        "web",
				},
			},
		},
		{
			Name:    "db-server-1",
			Address: "10.0.2.10",
			Port:    22,
			User:    "ec2-user",
			Vars: map[string]interface{}{
				"instance_id":       "i-1234567890abcdef2",
				"instance_type":     "t3.small",
				"region":            p.region,
				"availability_zone": "us-east-1a",
				"tags": map[string]string{
					"Name":        "db-server-1",
					"Environment": "production",
					"Role":        "database",
				},
			},
		},
	}

	// Create groups based on tags
	p.cachedGroups = make(map[string]*types.Group)

	// Group by role
	webGroup := &types.Group{
		Name:  "web",
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

	// Assign hosts to groups
	for i := range p.cachedHosts {
		host := &p.cachedHosts[i]
		if tags, ok := host.Vars["tags"].(map[string]string); ok {
			if role, ok := tags["Role"]; ok {
				switch role {
				case "web":
					webGroup.Hosts[host.Name] = host
				case "database":
					dbGroup.Hosts[host.Name] = host
				}
			}
			if env, ok := tags["Environment"]; ok && env == "production" {
				prodGroup.Hosts[host.Name] = host
			}
		}
	}

	p.cachedGroups["web"] = webGroup
	p.cachedGroups["database"] = dbGroup
	p.cachedGroups["production"] = prodGroup

	p.lastRefresh = time.Now()
	return nil
}

// filterHosts filters hosts by pattern
func (p *AWSEC2InventoryPlugin) filterHosts(hosts []types.Host, pattern string) []types.Host {
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
func (p *AWSEC2InventoryPlugin) filterGroups(groups map[string]*types.Group, pattern string) map[string]*types.Group {
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
var Plugin plugins.InventoryPlugin = NewAWSEC2InventoryPlugin()

// main is required for Go plugins but not used
func main() {}
