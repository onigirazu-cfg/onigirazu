package parser

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

// FindInventoryFile searches for inventory file in common locations with priority system:
// Priority 2: playbook directory (baseDir)
// Priority 3: /etc/onigirazu/
func (p *InventoryParser) FindInventoryFile(baseDir string) (string, error) {
	return p.FindInventoryFileWithPath("", baseDir)
}

// FindInventoryFileWithPath searches for inventory file with priority system:
// Priority 1: Explicitly specified path
// Priority 2: playbook directory (baseDir)
// Priority 3: /etc/onigirazu/
func (p *InventoryParser) FindInventoryFileWithPath(explicitPath, baseDir string) (string, error) {
	pathDiscovery := NewPathDiscovery(p.logger)
	path, priority, err := pathDiscovery.DiscoverInventoryFilePathWithPriority(explicitPath, baseDir)
	if err == nil {
		p.logger.Info("Auto-detected inventory file at priority %d: %s", priority, path)
		return path, nil
	}
	return "", err
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
	case ".json":
		return p.parseJsonInventory(data)
	case ".ini":
		return p.parseIniInventory(data)
	default:
		// Try to auto-detect format
		return p.autoDetectAndParse(data, filePath)
	}
}

// autoDetectAndParse tries to detect format and parse accordingly
func (p *InventoryParser) autoDetectAndParse(data []byte, filePath string) (*types.Inventory, error) {
	content := string(data)

	// Check if it's executable (dynamic inventory script)
	if p.isExecutable(filePath) {
		p.logger.Debug("Detected executable script for %s", filePath)
		return p.parseDynamicInventory(filePath)
	}

	// Check if it's a simple list (no YAML/TOML markers)
	if p.isSimpleList(content) {
		p.logger.Debug("Detected simple list format for %s", filePath)
		return p.parseSimpleList(data)
	}

	// Try JSON first
	if inv, err := p.parseJsonInventory(data); err == nil {
		p.logger.Debug("Successfully parsed as JSON: %s", filePath)
		return inv, nil
	}

	// Try YAML
	if inv, err := p.parseYamlInventory(data); err == nil {
		p.logger.Debug("Successfully parsed as YAML: %s", filePath)
		return inv, nil
	}

	// Try TOML
	if inv, err := p.parseTomlInventory(data); err == nil {
		p.logger.Debug("Successfully parsed as TOML: %s", filePath)
		return inv, nil
	}

	// Try INI
	if inv, err := p.parseIniInventory(data); err == nil {
		p.logger.Debug("Successfully parsed as INI: %s", filePath)
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

// parseYamlInventory parses YAML format inventory with auto-detection of Ansible format
func (p *InventoryParser) parseYamlInventory(data []byte) (*types.Inventory, error) {
	// First, try to detect if this is Ansible-style YAML
	if isAnsibleYaml(data) {
		p.logger.Debug("Detected Ansible-style YAML inventory format")
		return p.parseAnsibleYamlInventory(data)
	}

	// Otherwise, parse as standard Onigirazu YAML
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

// isAnsibleYaml detects if YAML content is in Ansible format
func isAnsibleYaml(data []byte) bool {
	var rawMap map[string]interface{}
	if err := yaml.Unmarshal(data, &rawMap); err != nil {
		return false
	}

	// Ansible format has top-level "all" key or uses "ansible_" prefix in host vars
	if _, hasAll := rawMap["all"]; hasAll {
		return true
	}

	// Check for ansible_* variable names in host definitions
	if hosts, ok := rawMap["hosts"].(map[string]interface{}); ok {
		for _, hostData := range hosts {
			if hostMap, ok := hostData.(map[string]interface{}); ok {
				for key := range hostMap {
					if strings.HasPrefix(key, "ansible_") {
						return true
					}
				}
			}
		}
	}

	return false
}

// AnsibleYAML structures for parsing Ansible-format inventory
type ansibleYamlInventory struct {
	All map[string]interface{} `yaml:"all"`
}

type ansibleGroup struct {
	Hosts    map[string]interface{} `yaml:"hosts"`
	Children map[string]interface{} `yaml:"children"`
	Vars     map[string]interface{} `yaml:"vars"`
}

// parseAnsibleYamlInventory parses Ansible-format YAML inventory
func (p *InventoryParser) parseAnsibleYamlInventory(data []byte) (*types.Inventory, error) {
	var ansibleInv ansibleYamlInventory
	if err := yaml.Unmarshal(data, &ansibleInv); err != nil {
		return nil, fmt.Errorf("error parsing Ansible YAML inventory: %w", err)
	}

	inventory := &types.Inventory{
		Groups: make(map[string]*types.Group),
		Hosts:  make([]types.Host, 0),
	}

	// Parse the "all" group which contains all hosts and groups
	if ansibleInv.All != nil {
		allData := ansibleInv.All
		// First pass: collect all hosts from the "all" section
		parsedHosts := make(map[string]*types.Host)

		if hostsData, ok := allData["hosts"].(map[string]interface{}); ok {
			for hostName, hostData := range hostsData {
				host := p.parseAnsibleHost(hostName, hostData)
				if host != nil {
					parsedHosts[hostName] = host
					inventory.Hosts = append(inventory.Hosts, *host)
				}
			}
		}

		// Second pass: parse groups and their hosts
		if groupsData, ok := allData["children"].(map[string]interface{}); ok {
			for groupName, groupData := range groupsData {
				group := p.parseAnsibleGroup(groupName, groupData, parsedHosts)
				if group != nil {
					inventory.Groups[groupName] = group
				}
			}
		}

		// Parse group-level vars if present
		if vars, ok := allData["vars"].(map[string]interface{}); ok {
			allGroup := &types.Group{
				Name:     "all",
				Hosts:    make(map[string]*types.Host),
				Children: make([]string, 0),
				Vars:     vars,
			}
			// Add all hosts to "all" group
			for _, host := range parsedHosts {
				allGroup.Hosts[host.Name] = host
			}
			inventory.Groups["all"] = allGroup
		}
	}

	if len(inventory.Hosts) == 0 {
		return nil, fmt.Errorf("no valid hosts found in Ansible inventory")
	}

	p.logger.Info("Parsed Ansible YAML inventory: %d groups, %d hosts", len(inventory.Groups), len(inventory.Hosts))
	return inventory, nil
}

// parseAnsibleHost converts Ansible host definition to Onigirazu Host
func (p *InventoryParser) parseAnsibleHost(hostName string, hostData interface{}) *types.Host {
	host := &types.Host{
		Name:    hostName,
		Address: hostName,
		Port:    22,
		User:    "root",
		Vars:    make(map[string]interface{}),
	}

	if hostData == nil {
		return host
	}

	hostMap, ok := hostData.(map[string]interface{})
	if !ok {
		return host
	}

	// Map Ansible variables to Onigirazu fields
	for key, value := range hostMap {
		switch key {
		case "ansible_host":
			if v, ok := value.(string); ok {
				host.Address = v
			}
		case "ansible_port":
			switch v := value.(type) {
			case int:
				host.Port = v
			case float64:
				host.Port = int(v)
			case string:
				if _, err := fmt.Sscanf(v, "%d", &host.Port); err != nil {
					p.logger.Warn("Invalid ansible_port value: %v", value)
				}
			}
		case "ansible_user":
			if v, ok := value.(string); ok {
				host.User = v
			}
		case "ansible_ssh_private_key_file":
			if v, ok := value.(string); ok {
				host.KeyFile = v
			}
		case "ansible_password":
			if v, ok := value.(string); ok {
				host.Password = v
			}
		case "ansible_ssh_host_key_checking":
			if v, ok := value.(bool); ok && !v {
				host.InsecureIgnoreHostKey = true
			}
		default:
			// Store other Ansible variables (including custom ones) in Vars
			// Remove ansible_ prefix for cleaner variable names
			varName := strings.TrimPrefix(key, "ansible_")
			host.Vars[varName] = value
		}
	}

	return host
}

// parseAnsibleGroup converts Ansible group definition to Onigirazu Group
func (p *InventoryParser) parseAnsibleGroup(groupName string, groupData interface{}, allHosts map[string]*types.Host) *types.Group {
	group := &types.Group{
		Name:     groupName,
		Hosts:    make(map[string]*types.Host),
		Children: make([]string, 0),
		Vars:     make(map[string]interface{}),
	}

	if groupData == nil {
		return group
	}

	groupMap, ok := groupData.(map[string]interface{})
	if !ok {
		return group
	}

	// Parse hosts in this group
	if hostsData, ok := groupMap["hosts"].(map[string]interface{}); ok {
		for hostName := range hostsData {
			if host, exists := allHosts[hostName]; exists {
				group.Hosts[hostName] = host
			}
		}
	}

	// Parse child groups
	if childrenData, ok := groupMap["children"].([]interface{}); ok {
		for _, childName := range childrenData {
			if name, ok := childName.(string); ok {
				group.Children = append(group.Children, name)
			}
		}
	} else if childrenData, ok := groupMap["children"].(map[string]interface{}); ok {
		// Handle children as map (Ansible format can use both)
		for childName := range childrenData {
			group.Children = append(group.Children, childName)
		}
	}

	// Parse group variables
	if vars, ok := groupMap["vars"].(map[string]interface{}); ok {
		for key, value := range vars {
			group.Vars[key] = value
		}
	}

	return group
}

// TOML structure for inventory
type tomlInventory struct {
	Hosts  map[string]tomlHost  `toml:"hosts"`
	Groups map[string]tomlGroup `toml:"groups"`
}

type tomlHost struct {
	Address               string                 `toml:"address"`
	Port                  int                    `toml:"port"`
	User                  string                 `toml:"user"`
	KeyFile               string                 `toml:"key_file"`
	Password              string                 `toml:"password"`
	InsecureIgnoreHostKey bool                   `toml:"insecure_ignore_host_key"`
	Vars                  map[string]interface{} `toml:"vars"`
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
			Name:                  name,
			Address:               tomlHost.Address,
			Port:                  tomlHost.Port,
			User:                  tomlHost.User,
			KeyFile:               tomlHost.KeyFile,
			Password:              tomlHost.Password,
			InsecureIgnoreHostKey: tomlHost.InsecureIgnoreHostKey,
			Vars:                  tomlHost.Vars,
		}

		// Set defaults
		if host.Port == 0 {
			host.Port = 22
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

// parseJsonInventory parses JSON format inventory
func (p *InventoryParser) parseJsonInventory(data []byte) (*types.Inventory, error) {
	var inventory types.Inventory
	if err := json.Unmarshal(data, &inventory); err != nil {
		return nil, fmt.Errorf("error parsing JSON inventory: %w", err)
	}

	if inventory.Groups == nil {
		inventory.Groups = make(map[string]*types.Group)
	}
	if inventory.Hosts == nil {
		inventory.Hosts = make([]types.Host, 0)
	}

	p.logger.Info("Parsed JSON inventory: %d groups, %d hosts", len(inventory.Groups), len(inventory.Hosts))
	return &inventory, nil
}

// parseIniInventory parses INI/Ansible format inventory
func (p *InventoryParser) parseIniInventory(data []byte) (*types.Inventory, error) {
	inventory := &types.Inventory{
		Groups: make(map[string]*types.Group),
		Hosts:  make([]types.Host, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var currentGroup *types.Group
	var isChildrenSection bool
	var isVarsSection bool
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			groupName := strings.Trim(line, "[]")
			isChildrenSection = false
			isVarsSection = false

			if strings.Contains(groupName, ":") {
				parts := strings.SplitN(groupName, ":", 2)
				groupName = parts[0]
				groupType := parts[1]

				if groupType == "children" {
					isChildrenSection = true
					if existingGroup, exists := inventory.Groups[groupName]; exists {
						currentGroup = existingGroup
					} else {
						currentGroup = &types.Group{
							Name:     groupName,
							Hosts:    make(map[string]*types.Host),
							Children: make([]string, 0),
							Vars:     make(map[string]interface{}),
						}
						inventory.Groups[groupName] = currentGroup
					}
					continue
				} else if groupType == "vars" {
					isVarsSection = true
					if existingGroup, exists := inventory.Groups[groupName]; exists {
						currentGroup = existingGroup
					} else {
						currentGroup = &types.Group{
							Name:     groupName,
							Hosts:    make(map[string]*types.Host),
							Children: make([]string, 0),
							Vars:     make(map[string]interface{}),
						}
						inventory.Groups[groupName] = currentGroup
					}
					continue
				}
			}

			if existingGroup, exists := inventory.Groups[groupName]; exists {
				currentGroup = existingGroup
			} else {
				currentGroup = &types.Group{
					Name:     groupName,
					Hosts:    make(map[string]*types.Host),
					Children: make([]string, 0),
					Vars:     make(map[string]interface{}),
				}
				inventory.Groups[groupName] = currentGroup
			}
			continue
		}

		if currentGroup != nil {
			if isChildrenSection {
				currentGroup.Children = append(currentGroup.Children, line)
			} else if isVarsSection {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					currentGroup.Vars[key] = value
				}
			} else {
				host := p.parseIniHostLine(line, lineNum)
				if host != nil {
					inventory.Hosts = append(inventory.Hosts, *host)
					currentGroup.Hosts[host.Name] = host
				}
			}
		} else {
			host := p.parseIniHostLine(line, lineNum)
			if host != nil {
				inventory.Hosts = append(inventory.Hosts, *host)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading INI inventory: %w", err)
	}

	p.logger.Info("Parsed INI inventory: %d groups, %d hosts", len(inventory.Groups), len(inventory.Hosts))
	return inventory, nil
}

// parseIniHostLine parses a single host line from INI format
func (p *InventoryParser) parseIniHostLine(line string, lineNum int) *types.Host {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	hostSpec := parts[0]
	var user, address, name string
	port := 22

	if strings.Contains(hostSpec, "@") {
		userParts := strings.SplitN(hostSpec, "@", 2)
		user = userParts[0]
		hostSpec = userParts[1]
	}

	if strings.Contains(hostSpec, ":") && !strings.Contains(hostSpec, "://") {
		portParts := strings.SplitN(hostSpec, ":", 2)
		address = portParts[0]
		if _, err := fmt.Sscanf(portParts[1], "%d", &port); err != nil {
			p.logger.Warn("Invalid port in line %d: %s, using default 22", lineNum, line)
			port = 22
		}
	} else {
		address = hostSpec
	}

	name = address

	host := &types.Host{
		Name:    name,
		Address: address,
		Port:    port,
		User:    user,
		Vars:    make(map[string]interface{}),
	}

	for i := 1; i < len(parts); i++ {
		if strings.Contains(parts[i], "=") {
			varParts := strings.SplitN(parts[i], "=", 2)
			key := strings.TrimSpace(varParts[0])
			value := strings.TrimSpace(varParts[1])

			switch key {
			case "ansible_host", "onigirazu_host":
				host.Address = value
			case "ansible_port", "onigirazu_port":
				if p, err := fmt.Sscanf(value, "%d", &port); err == nil && p > 0 {
					host.Port = port
				}
			case "ansible_user", "onigirazu_user":
				host.User = value
			case "ansible_ssh_private_key_file", "onigirazu_ssh_private_key_file":
				host.KeyFile = value
			case "ansible_password", "onigirazu_password":
				host.Password = value
			default:
				host.Vars[key] = value
			}
		}
	}

	return host
}

// isExecutable checks if file is executable
func (p *InventoryParser) isExecutable(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}

// parseDynamicInventory executes a dynamic inventory script and parses its JSON output
func (p *InventoryParser) parseDynamicInventory(scriptPath string) (*types.Inventory, error) {
	cmd := exec.Command(scriptPath, "--list") // #nosec G204 -- scriptPath is user-provided inventory file
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error executing dynamic inventory script: %w", err)
	}

	return p.parseJsonInventory(output)
}

// ParseInventoryOrInline parses inventory from either a file path or inline specification.
// If the input looks like a file path, it's parsed as a file.
// If the input looks like an inline host specification (e.g., "192.168.1.1," or "host1,host2"),
// it's parsed as inline inventory.
func (p *InventoryParser) ParseInventoryOrInline(ctx context.Context, inventoryPath string) (*types.Inventory, error) {
	detector := NewInlineInventoryDetector(p.logger)

	// Check if this looks like inline inventory
	if detector.IsInlineInventory(inventoryPath) {
		p.logger.Debug("Detected inline inventory specification: %s", inventoryPath)
		return detector.ParseInlineInventory(inventoryPath)
	}

	// Otherwise, treat it as a file path
	p.logger.Debug("Treating as inventory file path: %s", inventoryPath)
	return p.ParseInventoryFile(ctx, inventoryPath)
}
