package parser

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// InventoryParser handles parsing of inventory files in multiple formats
type InventoryParser struct {
	logger interface {
		Debug(format string, args ...interface{})
		Info(format string, args ...interface{})
		Warn(format string, args ...interface{})
	}
}

// NewInventoryParser creates a new inventory parser
func NewInventoryParser(logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
}) *InventoryParser {
	return &InventoryParser{
		logger: logger,
	}
}

// FindInventoryFile searches for inventory file in common locations
func (p *InventoryParser) FindInventoryFile(baseDir string) (string, error) {
	// List of common inventory file names
	inventoryFiles := []string{
		"inventory.yml",
		"inventory.yaml",
		"inventory.toml",
		"hosts",
		"hosts.yml",
		"hosts.yaml",
		"hosts.toml",
		"inventory",
	}

	for _, filename := range inventoryFiles {
		path := filepath.Join(baseDir, filename)
		if _, err := os.Stat(path); err == nil {
			p.logger.Debug("Found inventory file: %s", path)
			return path, nil
		}
	}

	return "", fmt.Errorf("no inventory file found in %s (searched: %v)", baseDir, inventoryFiles)
}

// ParseInventoryFile parses inventory file and auto-detects format
func (p *InventoryParser) ParseInventoryFile(ctx context.Context, filePath string) (*types.Inventory, error) {
	// Read file content
	data, err := os.ReadFile(filePath) // #nosec G304 -- filePath is provided by user
	if err != nil {
		return nil, fmt.Errorf("error reading inventory file: %w", err)
	}

	// Detect format based on file extension and content
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".toml":
		return p.parseTomlInventory(data)
	case ".yml", ".yaml":
		return p.parseYamlInventory(data)
	default:
		// Try to auto-detect format
		return p.autoDetectAndParse(data, filePath)
	}
}

// autoDetectAndParse tries to detect format and parse accordingly
func (p *InventoryParser) autoDetectAndParse(data []byte, filePath string) (*types.Inventory, error) {
	content := string(data)

	// Check if it's a simple list (no YAML/TOML markers)
	if p.isSimpleList(content) {
		p.logger.Debug("Detected simple list format for %s", filePath)
		return p.parseSimpleList(data)
	}

	// Try YAML first (most common)
	if inv, err := p.parseYamlInventory(data); err == nil {
		p.logger.Debug("Successfully parsed as YAML: %s", filePath)
		return inv, nil
	}

	// Try TOML
	if inv, err := p.parseTomlInventory(data); err == nil {
		p.logger.Debug("Successfully parsed as TOML: %s", filePath)
		return inv, nil
	}

	// Try simple list as fallback
	p.logger.Debug("Falling back to simple list format for %s", filePath)
	return p.parseSimpleList(data)
}

// isSimpleList checks if content looks like a simple list
func (p *InventoryParser) isSimpleList(content string) bool {
	lines := strings.Split(content, "\n")

	// Check if most lines look like addresses (no YAML/TOML syntax)
	simpleLines := 0
	totalLines := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		totalLines++

		// If line contains YAML/TOML markers, it's not a simple list
		if strings.Contains(line, ":") && !strings.Contains(line, "://") {
			// Could be YAML or TOML
			return false
		}
		if strings.Contains(line, "=") || strings.HasPrefix(line, "[") {
			// Likely TOML
			return false
		}

		simpleLines++
	}

	// If most lines are simple, treat as simple list
	return totalLines > 0 && simpleLines >= totalLines/2
}

// parseSimpleList parses a simple list of addresses (one per line)
func (p *InventoryParser) parseSimpleList(data []byte) (*types.Inventory, error) {
	inventory := &types.Inventory{
		Groups: make(map[string]*types.Group),
		Hosts:  make([]types.Host, 0),
	}

	// Create a default "all" group
	allGroup := &types.Group{
		Name:  "all",
		Hosts: make(map[string]*types.Host),
		Vars:  make(map[string]interface{}),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the line as host address
		host := p.parseSimpleHostLine(line, lineNum)
		if host != nil {
			inventory.Hosts = append(inventory.Hosts, *host)
			allGroup.Hosts[host.Name] = host
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading inventory: %w", err)
	}

	if len(inventory.Hosts) == 0 {
		return nil, fmt.Errorf("no valid hosts found in inventory")
	}

	inventory.Groups["all"] = allGroup

	p.logger.Info("Parsed simple list inventory: %d hosts", len(inventory.Hosts))
	return inventory, nil
}

// parseSimpleHostLine parses a single line from simple list format
func (p *InventoryParser) parseSimpleHostLine(line string, lineNum int) *types.Host {
	// Format can be:
	// - IP address: 192.168.1.10
	// - Hostname: server.example.com
	// - IP with port: 192.168.1.10:2222
	// - Hostname with port: server.example.com:2222
	// - With user: user@192.168.1.10
	// - With user and port: user@192.168.1.10:2222

	var user, address, name string
	port := 22

	// Check for user@host format
	if strings.Contains(line, "@") {
		parts := strings.SplitN(line, "@", 2)
		user = parts[0]
		line = parts[1]
	}

	// Check for host:port format
	if strings.Contains(line, ":") && !strings.Contains(line, "://") {
		parts := strings.SplitN(line, ":", 2)
		address = parts[0]
		if _, err := fmt.Sscanf(parts[1], "%d", &port); err != nil {
			p.logger.Warn("Invalid port in line %d: %s, using default 22", lineNum, line)
			port = 22
		}
	} else {
		address = line
	}

	// Generate host name
	if user != "" {
		name = fmt.Sprintf("%s@%s", user, address)
	} else {
		name = address
	}

	// Set default user if not specified
	if user == "" {
		user = "root"
	}

	host := &types.Host{
		Name:    name,
		Address: address,
		Port:    port,
		User:    user,
		Vars:    make(map[string]interface{}),
	}

	return host
}

// parseYamlInventory parses YAML format inventory
func (p *InventoryParser) parseYamlInventory(data []byte) (*types.Inventory, error) {
	var inventory types.Inventory
	if err := yaml.Unmarshal(data, &inventory); err != nil {
		return nil, fmt.Errorf("error parsing YAML inventory: %w", err)
	}

	// Initialize maps if nil
	if inventory.Groups == nil {
		inventory.Groups = make(map[string]*types.Group)
	}
	if inventory.Hosts == nil {
		inventory.Hosts = make([]types.Host, 0)
	}

	return &inventory, nil
}

// TOML structure for inventory
type tomlInventory struct {
	Hosts  map[string]tomlHost  `toml:"hosts"`
	Groups map[string]tomlGroup `toml:"groups"`
}

type tomlHost struct {
	Address string                 `toml:"address"`
	Port    int                    `toml:"port"`
	User    string                 `toml:"user"`
	Vars    map[string]interface{} `toml:"vars"`
}

type tomlGroup struct {
	Hosts    []string               `toml:"hosts"`
	Children []string               `toml:"children"`
	Vars     map[string]interface{} `toml:"vars"`
}

// parseTomlInventory parses TOML format inventory
func (p *InventoryParser) parseTomlInventory(data []byte) (*types.Inventory, error) {
	var tomlInv tomlInventory
	if err := toml.Unmarshal(data, &tomlInv); err != nil {
		return nil, fmt.Errorf("error parsing TOML inventory: %w", err)
	}

	// Convert to standard inventory format
	inventory := &types.Inventory{
		Groups: make(map[string]*types.Group),
		Hosts:  make([]types.Host, 0),
	}

	// Convert hosts
	for name, tomlHost := range tomlInv.Hosts {
		host := types.Host{
			Name:    name,
			Address: tomlHost.Address,
			Port:    tomlHost.Port,
			User:    tomlHost.User,
			Vars:    tomlHost.Vars,
		}

		// Set defaults
		if host.Port == 0 {
			host.Port = 22
		}
		if host.User == "" {
			host.User = "root"
		}
		if host.Address == "" {
			host.Address = name
		}
		if host.Vars == nil {
			host.Vars = make(map[string]interface{})
		}

		inventory.Hosts = append(inventory.Hosts, host)
	}

	// Convert groups
	for name, tomlGroup := range tomlInv.Groups {
		group := &types.Group{
			Name:     name,
			Hosts:    make(map[string]*types.Host),
			Children: tomlGroup.Children,
			Vars:     tomlGroup.Vars,
		}

		if group.Vars == nil {
			group.Vars = make(map[string]interface{})
		}

		// Link hosts to group
		for _, hostName := range tomlGroup.Hosts {
			// Find host in inventory
			for i := range inventory.Hosts {
				if inventory.Hosts[i].Name == hostName {
					group.Hosts[hostName] = &inventory.Hosts[i]
					break
				}
			}
		}

		inventory.Groups[name] = group
	}

	p.logger.Info("Parsed TOML inventory: %d groups, %d hosts", len(inventory.Groups), len(inventory.Hosts))
	return inventory, nil
}
