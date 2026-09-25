package inventory

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Manager manages inventory operations
type Manager struct {
	inventory    *types.Inventory
	parser       interfaces.PlaybookParser
	logger       interfaces.Logger
	cache        interfaces.CacheManager
	mutex        sync.RWMutex
	hostFilters  []HostFilter
	groupFilters []GroupFilter
	lastUpdated  time.Time
	lenient      bool // NEW: Lenient mode for parsing

	// Each group's own variables and parents, taken before inheritance
	// merges hosts into parent groups; host variables are resolved from these
	groupVars    map[string]map[string]interface{}
	groupParents map[string][]string
}

// HostFilter defines a function type for filtering hosts
type HostFilter func(*types.Host) bool

// GroupFilter defines a function type for filtering groups
type GroupFilter func(*types.Group) bool

// NewManager creates a new inventory manager
func NewManager(parser interfaces.PlaybookParser, logger interfaces.Logger, cache interfaces.CacheManager) *Manager {
	return &Manager{
		parser:       parser,
		logger:       logger,
		cache:        cache,
		hostFilters:  make([]HostFilter, 0),
		groupFilters: make([]GroupFilter, 0),
		lenient:      false, // Default to strict mode
	}
}

// SetLenient sets lenient mode for inventory processing
func (m *Manager) SetLenient(lenient bool) {
	m.lenient = lenient
	if lenient {
		m.logger.Info("Lenient mode enabled for inventory manager")
	}
}

// LoadInventory loads inventory from file
func (m *Manager) LoadInventory(ctx context.Context, filePath string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.logger.Debug("Loading inventory from: %s", filePath)

	// Check cache first
	cacheKey := fmt.Sprintf("inventory:%s", filePath)
	if cached, found := m.cache.Get(ctx, cacheKey); found {
		if inventory, ok := cached.(*types.Inventory); ok {
			m.inventory = inventory
			m.lastUpdated = time.Now()
			m.logger.Debug("Loaded inventory from cache")
			return nil
		}
	}

	// Parse inventory file
	inventory, err := m.parser.ParseInventory(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to parse inventory: %w", err)
	}

	// Process and validate inventory
	if err := m.processInventory(inventory); err != nil {
		return fmt.Errorf("failed to process inventory: %w", err)
	}

	m.inventory = inventory
	m.lastUpdated = time.Now()

	// Cache the inventory
	_ = m.cache.Set(ctx, cacheKey, inventory)

	m.logger.Info("Successfully loaded inventory: %d groups, %d hosts",
		len(inventory.Groups), m.getTotalHostCount())

	return nil
}

// SetInventory directly sets the inventory (used for pre-loaded/merged inventory)
func (m *Manager) SetInventory(inv *types.Inventory) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if inv == nil {
		return fmt.Errorf("cannot set nil inventory")
	}

	// Process and validate inventory
	if err := m.processInventory(inv); err != nil {
		return fmt.Errorf("failed to process inventory: %w", err)
	}

	m.inventory = inv
	m.lastUpdated = time.Now()

	m.logger.Info("Successfully set inventory: %d groups, %d hosts",
		len(inv.Groups), m.getTotalHostCount())

	return nil
}

// GetHosts returns all hosts matching the given pattern
func (m *Manager) GetHosts(pattern string) ([]types.Host, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.inventory == nil {
		return nil, fmt.Errorf("no inventory loaded")
	}

	hosts := m.hostsForPattern(pattern)

	// Apply filters
	filteredHosts := m.applyHostFilters(hosts)

	m.logger.Debug("Found %d hosts matching pattern '%s'", len(filteredHosts), pattern)
	return filteredHosts, nil
}

// hostsForPattern resolves a host pattern: "all", "localhost", a group, a
// host name or wildcard, or several of them joined with "," or ":" (union);
// a part starting with "!" removes its hosts, as in Ansible
func (m *Manager) hostsForPattern(pattern string) []types.Host {
	parts := strings.FieldsFunc(pattern, func(r rune) bool { return r == ',' || r == ':' })
	if len(parts) > 1 {
		var hosts []types.Host
		seen := make(map[string]bool)
		excluded := make(map[string]bool)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "!") {
				for _, h := range m.hostsForPattern(strings.TrimPrefix(part, "!")) {
					excluded[h.Name] = true
				}
				continue
			}
			for _, h := range m.hostsForPattern(part) {
				if !seen[h.Name] {
					seen[h.Name] = true
					hosts = append(hosts, h)
				}
			}
		}
		kept := hosts[:0]
		for _, h := range hosts {
			if !excluded[h.Name] {
				kept = append(kept, h)
			}
		}
		return kept
	}

	pattern = strings.TrimSpace(pattern)
	switch pattern {
	case "all":
		return m.getAllHosts()
	case "localhost":
		return []types.Host{m.getLocalhostHost()}
	}
	if group, exists := m.inventory.Groups[pattern]; exists {
		return m.getGroupHosts(group, pattern)
	}
	return m.getHostsByPattern(pattern)
}

// GetGroups returns all groups matching the given pattern
func (m *Manager) GetGroups(pattern string) (map[string]*types.Group, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.inventory == nil {
		return nil, fmt.Errorf("no inventory loaded")
	}

	groups := make(map[string]*types.Group)

	if pattern == "all" {
		// Return all groups
		for name, group := range m.inventory.Groups {
			groups[name] = group
		}
	} else {
		// Pattern matching for group names
		for name, group := range m.inventory.Groups {
			if m.matchPattern(name, pattern) {
				groups[name] = group
			}
		}
	}

	// Apply group filters
	filteredGroups := m.applyGroupFilters(groups)

	m.logger.Debug("Found %d groups matching pattern '%s'", len(filteredGroups), pattern)
	return filteredGroups, nil
}

// GetHostByName returns a specific host by name
func (m *Manager) GetHostByName(name string) (*types.Host, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.inventory == nil {
		return nil, fmt.Errorf("no inventory loaded")
	}

	// Search through direct hosts first
	for _, host := range m.inventory.Hosts {
		if host.Name == name {
			return &host, nil
		}
	}

	return nil, fmt.Errorf("host '%s' not found", name)
}

// GetGroupByName returns a specific group by name
func (m *Manager) GetGroupByName(name string) (*types.Group, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.inventory == nil {
		return nil, fmt.Errorf("no inventory loaded")
	}

	if group, exists := m.inventory.Groups[name]; exists {
		return group, nil
	}

	return nil, fmt.Errorf("group '%s' not found", name)
}

// AddHostFilter adds a host filter
func (m *Manager) AddHostFilter(filter HostFilter) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.hostFilters = append(m.hostFilters, filter)
}

// AddGroupFilter adds a group filter
func (m *Manager) AddGroupFilter(filter GroupFilter) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.groupFilters = append(m.groupFilters, filter)
}

// ClearFilters clears all filters
func (m *Manager) ClearFilters() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.hostFilters = make([]HostFilter, 0)
	m.groupFilters = make([]GroupFilter, 0)
}

// ApplyHostOverrides applies SSH user and key file overrides to all hosts
func (m *Manager) ApplyHostOverrides(user, keyFile string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.inventory == nil {
		return
	}

	for i := range m.inventory.Hosts {
		if user != "" {
			m.inventory.Hosts[i].User = user
		}
		if keyFile != "" {
			m.inventory.Hosts[i].KeyFile = keyFile
		}
	}

	for _, group := range m.inventory.Groups {
		for _, host := range group.Hosts {
			if user != "" {
				host.User = user
			}
			if keyFile != "" {
				host.KeyFile = keyFile
			}
		}
	}
}

// GetInventoryStats returns inventory statistics
func (m *Manager) GetInventoryStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.inventory == nil {
		return map[string]interface{}{
			"loaded": false,
		}
	}

	stats := map[string]interface{}{
		"loaded":       true,
		"last_updated": m.lastUpdated,
		"groups":       len(m.inventory.Groups),
		"total_hosts":  m.getTotalHostCount(),
		"group_stats":  m.getGroupStats(),
	}

	return stats
}

// ValidateConnectivity validates connectivity to hosts
func (m *Manager) ValidateConnectivity(ctx context.Context, hosts []types.Host, timeout time.Duration) map[string]error {
	results := make(map[string]error)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for _, host := range hosts {
		wg.Add(1)
		go func(h types.Host) {
			defer wg.Done()

			err := m.testHostConnectivity(ctx, &h, timeout)

			mutex.Lock()
			results[h.Name] = err
			mutex.Unlock()
		}(host)
	}

	wg.Wait()
	return results
}

// processInventory processes and validates inventory
func (m *Manager) processInventory(inventory *types.Inventory) error {
	m.groupVars = make(map[string]map[string]interface{}, len(inventory.Groups))
	m.groupParents = make(map[string][]string)
	for name, group := range inventory.Groups {
		own := make(map[string]interface{}, len(group.Vars))
		for k, v := range group.Vars {
			own[k] = v
		}
		m.groupVars[name] = own
		for _, child := range group.Children {
			m.groupParents[child] = append(m.groupParents[child], name)
		}
	}

	// Resolve group inheritance
	if err := m.resolveGroupInheritance(inventory); err != nil {
		return fmt.Errorf("failed to resolve group inheritance: %w", err)
	}

	// Validate host connectivity information
	if err := m.validateHosts(inventory); err != nil {
		return fmt.Errorf("host validation failed: %w", err)
	}

	// Process group variables
	m.processGroupVariables(inventory)

	return nil
}

// resolveGroupInheritance resolves parent-child relationships between groups
func (m *Manager) resolveGroupInheritance(inventory *types.Inventory) error {
	// Build dependency graph
	dependencies := make(map[string][]string)
	for groupName, group := range inventory.Groups {
		dependencies[groupName] = group.Children
	}

	// Resolve inheritance using topological sort
	resolved := make(map[string]bool)
	var resolve func(string) error

	resolve = func(groupName string) error {
		if resolved[groupName] {
			return nil
		}

		group, exists := inventory.Groups[groupName]
		if !exists {
			if m.lenient {
				m.logger.Warn("Group '%s' not found during inheritance resolution", groupName)
				return nil
			}
			return fmt.Errorf("group '%s' not found", groupName)
		}

		// Resolve children first
		for _, childName := range group.Children {
			if err := resolve(childName); err != nil {
				if m.lenient {
					m.logger.Warn("Failed to resolve child group '%s' of '%s': %v", childName, groupName, err)
					continue // Skip this child in lenient mode
				}
				return err
			}

			// Inherit hosts and variables from child
			childGroup := inventory.Groups[childName]
			if childGroup != nil {
				// Inherit hosts
				for hostName, host := range childGroup.Hosts {
					if group.Hosts == nil {
						group.Hosts = make(map[string]*types.Host)
					}
					if _, exists := group.Hosts[hostName]; !exists {
						group.Hosts[hostName] = host
					}
				}
			}
		}

		resolved[groupName] = true
		return nil
	}

	// Resolve all groups
	for groupName := range inventory.Groups {
		if err := resolve(groupName); err != nil {
			if m.lenient {
				m.logger.Warn("Failed to resolve group '%s': %v", groupName, err)
				continue // Skip this group in lenient mode
			}
			return err
		}
	}

	return nil
}

// findHostInOtherGroups searches for a host with the given name in other groups
func (m *Manager) findHostInOtherGroups(inventory *types.Inventory, hostName, excludeGroup string) *types.Host {
	for groupName, group := range inventory.Groups {
		if groupName == excludeGroup {
			continue
		}
		if host, exists := group.Hosts[hostName]; exists && host != nil {
			return host
		}
	}
	return nil
}

// validateHosts validates host configuration
func (m *Manager) validateHosts(inventory *types.Inventory) error {
	for groupName, group := range inventory.Groups {
		for hostName, host := range group.Hosts {
			// Skip nil hosts (hosts with no configuration)
			if host == nil {
				// Try to find this host in other groups and copy its configuration
				foundHost := m.findHostInOtherGroups(inventory, hostName, groupName)
				if foundHost != nil {
					// Create a copy of the found host
					hostCopy := *foundHost
					// Make a deep copy of Vars
					hostCopy.Vars = make(map[string]interface{})
					for k, v := range foundHost.Vars {
						hostCopy.Vars[k] = v
					}
					host = &hostCopy
					m.logger.Debug("Copied host '%s' configuration from another group to '%s'", hostName, groupName)
				} else {
					// Create a minimal host entry
					host = &types.Host{
						Name:    hostName,
						Address: hostName,
						Port:    22,
						Vars:    make(map[string]interface{}),
					}
				}
				group.Hosts[hostName] = host
			}

			// Initialize Vars if nil
			if host.Vars == nil {
				host.Vars = make(map[string]interface{})
			}

			// Debug: log what we have in Vars
			m.logger.Debug("Host '%s' in group '%s': Vars=%+v, User=%s, Address=%s",
				hostName, groupName, host.Vars, host.User, host.Address)

			// Process Onigirazu-style variables from host.Vars
			if onigirazuHost, ok := host.Vars["onigirazu_host"].(string); ok && onigirazuHost != "" {
				m.logger.Debug("Setting host.Address from onigirazu_host: %s", onigirazuHost)
				host.Address = onigirazuHost
			}
			if onigirazuUser, ok := host.Vars["onigirazu_user"].(string); ok && onigirazuUser != "" {
				m.logger.Debug("Setting host.User from onigirazu_user: %s", onigirazuUser)
				host.User = onigirazuUser
			}
			if onigirazuPort, ok := host.Vars["onigirazu_port"].(int); ok && onigirazuPort > 0 {
				host.Port = onigirazuPort
			}
			if keyFile, ok := host.Vars["onigirazu_ssh_private_key_file"].(string); ok && keyFile != "" {
				m.logger.Debug("Found onigirazu_ssh_private_key_file: %s", keyFile)
				// Expand ~ to home directory
				if strings.HasPrefix(keyFile, "~/") {
					if homeDir, err := os.UserHomeDir(); err == nil {
						keyFile = filepath.Join(homeDir, keyFile[2:])
						m.logger.Debug("Expanded key file path to: %s", keyFile)
					}
				}
				host.KeyFile = keyFile
				m.logger.Debug("Set host.KeyFile to: %s", host.KeyFile)
			} else {
				m.logger.Debug("No onigirazu_ssh_private_key_file found in Vars for host %s", hostName)
			}
			if password, ok := host.Vars["onigirazu_password"].(string); ok && password != "" {
				host.Password = password
			}

			// Set default values
			if host.Name == "" {
				host.Name = hostName
			}
			if host.Address == "" {
				host.Address = hostName
			}
			// Store original port value before setting default
			if host.Vars == nil {
				host.Vars = make(map[string]interface{})
			}
			if host.Port == 0 {
				host.Vars["_original_port"] = 0
				host.Port = 22
			} else {
				host.Vars["_original_port"] = host.Port
			}

			// Log final host configuration
			m.logger.Debug("Final host configuration for '%s': Name=%s, Address=%s, Port=%d, User=%s, KeyFile=%s",
				hostName, host.Name, host.Address, host.Port, host.User, host.KeyFile)

			// Validate address format
			if net.ParseIP(host.Address) == nil {
				// Not an IP, check if it's a valid hostname
				if !m.isValidHostname(host.Address) {
					m.logger.Warn("Host '%s' in group '%s' has invalid address: %s",
						hostName, groupName, host.Address)
				}
			}
		}
	}

	return nil
}

// processGroupVariables processes and merges group variables
func (m *Manager) processGroupVariables(inventory *types.Inventory) {
	for _, group := range inventory.Groups {
		if group.Vars == nil {
			group.Vars = make(map[string]interface{})
		}

		// Add group metadata
		group.Vars["group_name"] = group.Name
		group.Vars["group_hosts"] = len(group.Hosts)
		group.Vars["group_children"] = len(group.Children)
	}
}

// getAllHosts returns all hosts from all groups
func (m *Manager) getAllHosts() []types.Host {
	hosts := make([]types.Host, 0)
	seen := make(map[string]bool)

	for _, group := range m.inventory.Groups {
		for _, host := range group.Hosts {
			if !seen[host.Name] {
				hosts = append(hosts, m.hostView(host))
				seen[host.Name] = true
			}
		}
	}

	return hosts
}

// getLocalhostHost returns localhost host configuration
func (m *Manager) getLocalhostHost() types.Host {
	return types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
		Port:    22,
		User:    "root",
		Vars:    map[string]interface{}{"onigirazu_connection": "local"},
	}
}

// getGroupHosts returns all hosts from a specific group
func (m *Manager) getGroupHosts(group *types.Group, groupName string) []types.Host {
	hosts := make([]types.Host, 0, len(group.Hosts))
	for _, host := range group.Hosts {
		hosts = append(hosts, m.hostView(host))
	}
	return hosts
}

// hostView returns a copy of a host with the variables of every group it
// belongs to: "all" first, then parent groups before their children (by
// depth, then name), and the host's own variables over all of them. The same
// host gets the same variables whichever pattern selected it.
func (m *Manager) hostView(host *types.Host) types.Host {
	groups := make([]string, 0)
	for name, group := range m.inventory.Groups {
		if _, ok := group.Hosts[host.Name]; ok && name != "all" {
			groups = append(groups, name)
		}
	}
	depth := make(map[string]int)
	var depthOf func(string, int) int
	depthOf = func(g string, guard int) int {
		if d, ok := depth[g]; ok {
			return d
		}
		d := 0
		if guard < 64 { // cycles are reported elsewhere; do not loop here
			for _, parent := range m.groupParents[g] {
				if pd := depthOf(parent, guard+1) + 1; pd > d {
					d = pd
				}
			}
		}
		depth[g] = d
		return d
	}
	sort.Slice(groups, func(i, j int) bool {
		di, dj := depthOf(groups[i], 0), depthOf(groups[j], 0)
		if di != dj {
			return di < dj
		}
		return groups[i] < groups[j]
	})

	merged := make(map[string]interface{})
	for _, g := range append([]string{"all"}, groups...) {
		for k, v := range m.groupVars[g] {
			merged[k] = v
		}
	}

	// A host listed in several groups is a separate object in each; its own
	// variables are the union of all of them, over every group variable
	for _, g := range groups {
		if entry := m.inventory.Groups[g].Hosts[host.Name]; entry != nil {
			for k, v := range entry.Vars {
				merged[k] = v
			}
		}
	}

	hostCopy := *host
	hostCopy.Vars = make(map[string]interface{}, len(host.Vars)+len(merged)+1)
	for k, v := range merged {
		hostCopy.Vars[k] = v
	}
	for k, v := range host.Vars {
		hostCopy.Vars[k] = v
	}
	applyGroupConnectionVars(&hostCopy, host, merged)

	names := append([]string(nil), groups...)
	sort.Strings(names)
	hostCopy.Vars["group_names"] = names
	return hostCopy
}

// applyGroupConnectionVars fills connection fields the host leaves at their
// defaults from group variables
func applyGroupConnectionVars(hostCopy, host *types.Host, vars map[string]interface{}) {
	if address, ok := vars["address"].(string); ok && (host.Address == "" || host.Address == host.Name) {
		hostCopy.Address = address
	}
	if user, ok := vars["user"].(string); ok && host.User == "" {
		hostCopy.User = user
	}
	if port, ok := vars["port"].(int); ok {
		if origPort, hasOrig := host.Vars["_original_port"].(int); hasOrig && origPort == 0 {
			hostCopy.Port = port
		}
	}
	if password, ok := vars["password"].(string); ok && host.Password == "" {
		hostCopy.Password = password
	}
	if keyFile, ok := vars["key_file"].(string); ok && host.KeyFile == "" {
		hostCopy.KeyFile = keyFile
	}
	if insecure, ok := vars["insecure_ignore_host_key"].(bool); ok && !host.InsecureIgnoreHostKey {
		hostCopy.InsecureIgnoreHostKey = insecure
	}
}

// getHostsByPattern returns hosts matching a pattern
func (m *Manager) getHostsByPattern(pattern string) []types.Host {
	hosts := make([]types.Host, 0)
	seen := make(map[string]bool)

	for _, group := range m.inventory.Groups {
		for _, host := range group.Hosts {
			if !seen[host.Name] && m.matchPattern(host.Name, pattern) {
				hosts = append(hosts, m.hostView(host))
				seen[host.Name] = true
			}
		}
	}

	return hosts
}

// matchPattern checks if a string matches a pattern (supports wildcards)
func (m *Manager) matchPattern(str, pattern string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}

	if strings.Contains(pattern, "*") {
		// Convert wildcard pattern to regex-like matching
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			prefix, suffix := parts[0], parts[1]
			return strings.HasPrefix(str, prefix) && strings.HasSuffix(str, suffix)
		}
	}

	return str == pattern
}

// applyHostFilters applies all host filters
func (m *Manager) applyHostFilters(hosts []types.Host) []types.Host {
	if len(m.hostFilters) == 0 {
		return hosts
	}

	filtered := make([]types.Host, 0)
	for _, host := range hosts {
		include := true
		for _, filter := range m.hostFilters {
			if !filter(&host) {
				include = false
				break
			}
		}
		if include {
			filtered = append(filtered, host)
		}
	}

	return filtered
}

// applyGroupFilters applies all group filters
func (m *Manager) applyGroupFilters(groups map[string]*types.Group) map[string]*types.Group {
	if len(m.groupFilters) == 0 {
		return groups
	}

	filtered := make(map[string]*types.Group)
	for name, group := range groups {
		include := true
		for _, filter := range m.groupFilters {
			if !filter(group) {
				include = false
				break
			}
		}
		if include {
			filtered[name] = group
		}
	}

	return filtered
}

// getTotalHostCount returns total number of unique hosts
func (m *Manager) getTotalHostCount() int {
	if m.inventory == nil {
		return 0
	}

	seen := make(map[string]bool)
	count := 0

	for _, group := range m.inventory.Groups {
		for hostName := range group.Hosts {
			if !seen[hostName] {
				seen[hostName] = true
				count++
			}
		}
	}

	return count
}

// getGroupStats returns statistics for each group
func (m *Manager) getGroupStats() map[string]interface{} {
	stats := make(map[string]interface{})

	for groupName, group := range m.inventory.Groups {
		stats[groupName] = map[string]interface{}{
			"hosts":    len(group.Hosts),
			"children": len(group.Children),
			"vars":     len(group.Vars),
		}
	}

	return stats
}

// testHostConnectivity tests connectivity to a host
func (m *Manager) testHostConnectivity(ctx context.Context, host *types.Host, timeout time.Duration) error {
	// Use net.JoinHostPort to properly handle both IPv4 and IPv6 addresses
	address := net.JoinHostPort(host.Address, fmt.Sprintf("%d", host.Port))

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	defer conn.Close()

	return nil
}

// isValidHostname checks if a string is a valid hostname
func (m *Manager) isValidHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}

	// Check for valid characters
	for _, char := range hostname {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' && char != '.' {
			return false
		}
	}

	return true
}

// ListHosts returns a sorted list of all host names
func (m *Manager) ListHosts() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.inventory == nil {
		return []string{}
	}

	seen := make(map[string]bool)
	var names []string

	for _, group := range m.inventory.Groups {
		for hostName := range group.Hosts {
			if !seen[hostName] {
				names = append(names, hostName)
				seen[hostName] = true
			}
		}
	}

	sort.Strings(names)
	return names
}

// ListGroups returns a sorted list of all group names
func (m *Manager) ListGroups() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.inventory == nil {
		return []string{}
	}

	names := make([]string, 0, len(m.inventory.Groups))
	for groupName := range m.inventory.Groups {
		names = append(names, groupName)
	}

	sort.Strings(names)
	return names
}

// GetHostGroups returns all groups that contain the specified host
func (m *Manager) GetHostGroups(hostName string) []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.inventory == nil {
		return []string{}
	}

	groups := make([]string, 0)
	for groupName, group := range m.inventory.Groups {
		if _, exists := group.Hosts[hostName]; exists {
			groups = append(groups, groupName)
		}
	}

	sort.Strings(groups)
	return groups
}

// GetGroupHierarchy returns the full hierarchy of a group including all parent and child groups
func (m *Manager) GetGroupHierarchy(groupName string) (*GroupHierarchy, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.inventory == nil {
		return nil, fmt.Errorf("no inventory loaded")
	}

	group, exists := m.inventory.Groups[groupName]
	if !exists {
		return nil, fmt.Errorf("group '%s' not found", groupName)
	}

	hierarchy := &GroupHierarchy{
		Name:     groupName,
		Children: make([]string, len(group.Children)),
		Parents:  m.findParentGroups(groupName),
		Hosts:    make([]string, 0, len(group.Hosts)),
	}

	copy(hierarchy.Children, group.Children)

	for hostName := range group.Hosts {
		hierarchy.Hosts = append(hierarchy.Hosts, hostName)
	}

	sort.Strings(hierarchy.Children)
	sort.Strings(hierarchy.Parents)
	sort.Strings(hierarchy.Hosts)

	return hierarchy, nil
}

// findParentGroups finds all groups that have the specified group as a child
func (m *Manager) findParentGroups(childGroupName string) []string {
	parents := make([]string, 0)

	for groupName, group := range m.inventory.Groups {
		for _, child := range group.Children {
			if child == childGroupName {
				parents = append(parents, groupName)
				break
			}
		}
	}

	return parents
}

// IsHostInGroup checks if a host belongs to a group (including through inheritance)
func (m *Manager) IsHostInGroup(hostName, groupName string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.inventory == nil {
		return false
	}

	group, exists := m.inventory.Groups[groupName]
	if !exists {
		return false
	}

	_, found := group.Hosts[hostName]
	return found
}

// GetAllHostsInGroup returns all hosts in a group including hosts from child groups
func (m *Manager) GetAllHostsInGroup(groupName string) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.inventory == nil {
		return nil, fmt.Errorf("no inventory loaded")
	}

	group, exists := m.inventory.Groups[groupName]
	if !exists {
		return nil, fmt.Errorf("group '%s' not found", groupName)
	}

	hostSet := make(map[string]bool)
	m.collectHostsRecursive(group, hostSet)

	hosts := make([]string, 0, len(hostSet))
	for hostName := range hostSet {
		hosts = append(hosts, hostName)
	}

	sort.Strings(hosts)
	return hosts, nil
}

// collectHostsRecursive recursively collects all hosts from a group and its children
func (m *Manager) collectHostsRecursive(group *types.Group, hostSet map[string]bool) {
	for hostName := range group.Hosts {
		hostSet[hostName] = true
	}

	for _, childName := range group.Children {
		if childGroup, exists := m.inventory.Groups[childName]; exists {
			m.collectHostsRecursive(childGroup, hostSet)
		}
	}
}

// GroupHierarchy represents the hierarchy information for a group
type GroupHierarchy struct {
	Name     string
	Parents  []string
	Children []string
	Hosts    []string
}
