package parser

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	detector := NewInlineInventoryDetector(p.logger)
	simpleLines := 0
	totalLines := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		totalLines++

		// [user@]host[:port]
		if detector.isValidHostSpecification(line) {
			simpleLines++
			continue
		}

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

	uniqueHostNames(inventory, allGroup)
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

	name = address

	// no user: SSH connects as the local user, as Ansible does
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

// parseAnsibleYamlInventory parses Ansible-format YAML inventory
func (p *InventoryParser) parseAnsibleYamlInventory(data []byte) (*types.Inventory, error) {
	var ansibleInv ansibleYamlInventory
	if err := yaml.Unmarshal(data, &ansibleInv); err != nil {
		return nil, fmt.Errorf("error parsing Ansible YAML inventory: %w", err)
	}
	return p.parseAnsibleTree(ansibleInv.All)
}

// parseAnsibleTree builds an inventory from the Ansible tree under "all"
func (p *InventoryParser) parseAnsibleTree(all map[string]interface{}) (*types.Inventory, error) {
	inventory := &types.Inventory{
		Groups: make(map[string]*types.Group),
		Hosts:  make([]types.Host, 0),
	}

	// The tree under "all": hosts may be defined in any group, groups nest
	// through children (as a map of groups or a list of names), and every
	// host belongs to "all"
	if all != nil {
		w := &ansibleWalk{inventory: inventory, raw: map[string]map[string]interface{}{}, members: map[string][]string{}}
		w.group("all", all, 0)
		// a host may be listed in several groups: its settings from all of
		// them are merged, then it is parsed once
		hosts := map[string]*types.Host{}
		for _, name := range w.order {
			var data interface{}
			if raw := w.raw[name]; len(raw) > 0 {
				data = raw
			}
			if host := p.parseAnsibleHost(name, data); host != nil {
				hosts[name] = host
				inventory.Hosts = append(inventory.Hosts, *host)
			}
		}
		w.members["all"] = w.order
		for groupName, names := range w.members {
			for _, name := range names {
				if host, ok := hosts[name]; ok {
					inventory.Groups[groupName].Hosts[name] = host
				}
			}
		}
		// "all" is implicit; it is kept as a group only to carry its vars
		if len(inventory.Groups["all"].Vars) == 0 {
			delete(inventory.Groups, "all")
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
		case "ansible_become_password", "ansible_become_pass", "ansible_sudo_pass", "onigirazu_become_password":
			host.BecomePassword = fmt.Sprint(value)
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
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err == nil && isAnsibleScriptOutput(raw) {
		return p.parseAnsibleTree(ansibleScriptTree(raw))
	}

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

		// hosts before any section are "ungrouped", as in Ansible
		if currentGroup == nil {
			currentGroup = &types.Group{
				Name:     "ungrouped",
				Hosts:    make(map[string]*types.Host),
				Children: make([]string, 0),
				Vars:     make(map[string]interface{}),
			}
			inventory.Groups["ungrouped"] = currentGroup
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
			case "ansible_become_password", "ansible_become_pass", "onigirazu_become_password":
				host.BecomePassword = value
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

	// An existing file wins over a name that also looks like a host
	if _, err := os.Stat(inventoryPath); err != nil && detector.IsInlineInventory(inventoryPath) {
		p.logger.Debug("Detected inline inventory specification: %s", inventoryPath)
		return detector.ParseInlineInventory(inventoryPath)
	}

	// Otherwise, treat it as a file path
	p.logger.Debug("Treating as inventory file path: %s", inventoryPath)
	return p.ParseInventoryFile(ctx, inventoryPath)
}

// ansibleWalk collects the hosts and groups of an Ansible YAML inventory
type ansibleWalk struct {
	inventory *types.Inventory
	raw       map[string]map[string]interface{} // host -> merged settings
	members   map[string][]string               // group -> host names
	order     []string                          // hosts as they first appear
}

func (w *ansibleWalk) group(name string, data interface{}, depth int) *types.Group {
	group, ok := w.inventory.Groups[name]
	if !ok {
		group = &types.Group{Name: name, Hosts: map[string]*types.Host{}, Children: []string{}, Vars: map[string]interface{}{}}
		w.inventory.Groups[name] = group
	}
	m, ok := data.(map[string]interface{})
	if !ok || depth > 32 {
		return group
	}
	if hostsData, ok := m["hosts"].(map[string]interface{}); ok {
		names := make([]string, 0, len(hostsData))
		for hostName := range hostsData {
			names = append(names, hostName)
		}
		sort.Strings(names)
		for _, hostName := range names {
			if _, seen := w.raw[hostName]; !seen {
				w.raw[hostName] = map[string]interface{}{}
				w.order = append(w.order, hostName)
			}
			if settings, ok := hostsData[hostName].(map[string]interface{}); ok {
				for k, v := range settings {
					w.raw[hostName][k] = v
				}
			}
			w.members[name] = appendUnique(w.members[name], hostName)
		}
	}
	if vars, ok := m["vars"].(map[string]interface{}); ok {
		for k, v := range vars {
			group.Vars[k] = v
		}
	}
	switch children := m["children"].(type) {
	case map[string]interface{}:
		names := make([]string, 0, len(children))
		for childName := range children {
			names = append(names, childName)
		}
		sort.Strings(names)
		for _, childName := range names {
			w.group(childName, children[childName], depth+1)
			group.Children = appendUnique(group.Children, childName)
		}
	case []interface{}:
		for _, c := range children {
			if childName, ok := c.(string); ok {
				w.group(childName, nil, depth+1)
				group.Children = appendUnique(group.Children, childName)
			}
		}
	}
	return group
}

func appendUnique(list []string, s string) []string {
	for _, v := range list {
		if v == s {
			return list
		}
	}
	return append(list, s)
}

// isAnsibleScriptOutput tells the JSON of an Ansible inventory script
// ({"web": {"hosts": [...]}, "_meta": {"hostvars": {...}}}) from onigirazu's
// own JSON inventory ({"hosts": [...], "groups": {...}})
func isAnsibleScriptOutput(raw map[string]interface{}) bool {
	if _, ok := raw["_meta"]; ok {
		return true
	}
	if _, ok := raw["all"]; ok {
		return true
	}
	_, hosts := raw["hosts"]
	_, groups := raw["groups"]
	return len(raw) > 0 && !hosts && !groups
}

// ansibleScriptTree turns Ansible script output into the tree under "all"
// that an Ansible YAML inventory has
func ansibleScriptTree(raw map[string]interface{}) map[string]interface{} {
	hostvars := map[string]interface{}{}
	if meta, ok := raw["_meta"].(map[string]interface{}); ok {
		if hv, ok := meta["hostvars"].(map[string]interface{}); ok {
			hostvars = hv
		}
	}
	hostMap := func(list interface{}) map[string]interface{} {
		hosts := map[string]interface{}{}
		items, _ := list.([]interface{})
		for _, item := range items {
			if name, ok := item.(string); ok {
				hosts[name] = hostvars[name]
			}
		}
		return hosts
	}
	group := func(data interface{}) map[string]interface{} {
		out := map[string]interface{}{}
		switch g := data.(type) {
		case []interface{}: // a bare list of hosts
			out["hosts"] = hostMap(g)
		case map[string]interface{}:
			out["hosts"] = hostMap(g["hosts"])
			if vars, ok := g["vars"]; ok {
				out["vars"] = vars
			}
			if children, ok := g["children"]; ok {
				out["children"] = children
			}
		}
		return out
	}

	all := map[string]interface{}{}
	children := map[string]interface{}{}
	for name, data := range raw {
		switch name {
		case "_meta":
		case "all":
			for k, v := range group(data) {
				all[k] = v
			}
		default:
			children[name] = group(data)
		}
	}
	if existing, ok := all["children"].([]interface{}); ok {
		for _, c := range existing {
			if name, ok := c.(string); ok {
				if _, defined := children[name]; !defined {
					children[name] = nil
				}
			}
		}
	}
	all["children"] = children
	return all
}
