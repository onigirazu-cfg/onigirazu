package inventory

import (
	"fmt"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Changes during a run (add_host, group_by): they live in memory for the
// rest of the run, as in Ansible.

// AddHost adds a host to "all" and to groups (created when missing), or, for
// a host the inventory has, merges vars into it and adds the groups.
// Connection variables (ansible_host, ansible_port, ansible_user, ...) work
// as in an inventory file.
func (m *Manager) AddHost(name string, vars map[string]interface{}, groups []string) error {
	if name == "" {
		return fmt.Errorf("add_host: no host name")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.inventory == nil {
		m.inventory = &types.Inventory{}
	}
	if m.inventory.Groups == nil {
		m.inventory.Groups = map[string]*types.Group{}
	}
	var entries []*types.Host
	for _, g := range m.inventory.Groups {
		if h, ok := g.Hosts[name]; ok {
			entries = append(entries, h)
		}
	}
	if len(entries) == 0 {
		h := &types.Host{Name: name, Address: name, Vars: map[string]interface{}{}}
		m.inventory.Hosts = append(m.inventory.Hosts, *h)
		entries = append(entries, h)
	}
	for _, h := range entries {
		if h.Vars == nil {
			h.Vars = map[string]interface{}{}
		}
		for k, v := range vars {
			h.Vars[k] = v
		}
	}
	m.addToGroup(entries[0], "all", nil)
	for _, g := range groups {
		if g != "" {
			m.addToGroup(entries[0], g, nil)
		}
	}
	return nil
}

// AddHostToGroup adds a host the inventory has to group (created when
// missing) and the group under parents (default "all"), as group_by does
func (m *Manager) AddHostToGroup(hostName, group string, parents []string) error {
	if group == "" {
		return fmt.Errorf("group_by: no group name")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.inventory == nil || m.inventory.Groups == nil {
		return fmt.Errorf("group_by: no inventory")
	}
	var host *types.Host
	for _, g := range m.inventory.Groups {
		if h, ok := g.Hosts[hostName]; ok {
			host = h
			break
		}
	}
	if host == nil {
		return fmt.Errorf("group_by: host %s is not in the inventory", hostName)
	}
	m.addToGroup(host, group, parents)
	return nil
}

// addToGroup puts host in group and in every ancestor of it (groups hold the
// hosts of their children, see resolveGroupInheritance); a new group gets
// parents (none: "all")
func (m *Manager) addToGroup(host *types.Host, name string, parents []string) {
	g := m.inventory.Groups[name]
	if g == nil {
		g = &types.Group{Name: name, Hosts: map[string]*types.Host{}}
		m.inventory.Groups[name] = g
		if m.groupVars == nil {
			m.groupVars = map[string]map[string]interface{}{}
		}
		if m.groupParents == nil {
			m.groupParents = map[string][]string{}
		}
		if _, ok := m.groupVars[name]; !ok {
			m.groupVars[name] = map[string]interface{}{}
		}
	}
	if g.Hosts == nil {
		g.Hosts = map[string]*types.Host{}
	}
	if _, ok := g.Hosts[host.Name]; !ok {
		g.Hosts[host.Name] = host
	}
	for _, p := range parents {
		if p == "" || p == name {
			continue
		}
		known := false
		for _, have := range m.groupParents[name] {
			known = known || have == p
		}
		if !known {
			m.groupParents[name] = append(m.groupParents[name], p)
			if pg := m.inventory.Groups[p]; pg != nil {
				pg.Children = append(pg.Children, name)
			}
		}
	}
	// the ancestors hold the host too
	seen := map[string]bool{name: true}
	var up func(string)
	up = func(child string) {
		for _, p := range m.groupParents[child] {
			if seen[p] {
				continue
			}
			seen[p] = true
			if m.inventory.Groups[p] == nil {
				m.inventory.Groups[p] = &types.Group{Name: p, Hosts: map[string]*types.Host{}}
			}
			if _, ok := m.inventory.Groups[p].Hosts[host.Name]; !ok {
				m.inventory.Groups[p].Hosts[host.Name] = host
			}
			up(p)
		}
	}
	up(name)
	if name != "all" {
		all := m.inventory.Groups["all"]
		if all == nil {
			all = &types.Group{Name: "all", Hosts: map[string]*types.Host{}}
			m.inventory.Groups["all"] = all
		}
		if _, ok := all.Hosts[host.Name]; !ok {
			all.Hosts[host.Name] = host
		}
	}
}
