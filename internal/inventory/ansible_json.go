package inventory

import (
	"sort"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// AnsibleList is the inventory as `ansible-inventory --list` prints it:
// every group with its own hosts and its children, "all" with the top
// groups and "ungrouped", and _meta.hostvars with each host's variables
// (group variables resolved, connection variables included)
func (m *Manager) AnsibleList() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	out := map[string]interface{}{}
	hostvars := map[string]interface{}{}
	if m.inventory == nil {
		out["_meta"] = map[string]interface{}{"hostvars": hostvars}
		return out
	}
	children := map[string][]string{}
	for child, parents := range m.groupParents {
		for _, p := range parents {
			children[p] = append(children[p], child)
		}
	}
	grouped := map[string]bool{}
	var top []string
	names := make([]string, 0, len(m.inventory.Groups))
	for name := range m.inventory.Groups {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if name == "all" || name == "ungrouped" {
			continue
		}
		group := m.inventory.Groups[name]
		// hosts that came in through a child group belong to the child
		inChild := map[string]bool{}
		for _, c := range children[name] {
			if cg := m.inventory.Groups[c]; cg != nil {
				for h := range cg.Hosts {
					inChild[h] = true
				}
			}
		}
		entry := map[string]interface{}{}
		var hosts []string
		for h := range group.Hosts {
			grouped[h] = true
			if !inChild[h] {
				hosts = append(hosts, h)
			}
		}
		sort.Strings(hosts)
		if len(hosts) > 0 {
			entry["hosts"] = hosts
		}
		if c := children[name]; len(c) > 0 {
			sort.Strings(c)
			entry["children"] = c
		}
		out[name] = entry
		if len(m.groupParents[name]) == 0 || (len(m.groupParents[name]) == 1 && m.groupParents[name][0] == "all") {
			top = append(top, name)
		}
	}
	var ungrouped []string
	seen := map[string]bool{}
	for _, g := range m.inventory.Groups {
		for name, host := range g.Hosts {
			if seen[name] {
				continue
			}
			seen[name] = true
			if !grouped[name] {
				ungrouped = append(ungrouped, name)
			}
			hostvars[name] = ansibleHostVars(m.hostView(host))
		}
	}
	sort.Strings(ungrouped)
	if len(ungrouped) > 0 {
		out["ungrouped"] = map[string]interface{}{"hosts": ungrouped}
	}
	top = append(top, "ungrouped")
	out["all"] = map[string]interface{}{"children": top}
	out["_meta"] = map[string]interface{}{"hostvars": hostvars}
	return out
}

// AnsibleHostVars are the variables of one host as ansible-inventory
// --host prints them
func (m *Manager) AnsibleHostVars(name string) (map[string]interface{}, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if m.inventory == nil {
		return nil, false
	}
	for _, g := range m.inventory.Groups {
		if host, ok := g.Hosts[name]; ok {
			return ansibleHostVars(m.hostView(host)), true
		}
	}
	return nil, false
}

func ansibleHostVars(h types.Host) map[string]interface{} {
	vars := map[string]interface{}{}
	for k, v := range h.Vars {
		if k == "group_names" || (len(k) > 0 && k[0] == '_') {
			continue
		}
		vars[k] = v
	}
	if h.Address != "" && h.Address != h.Name {
		vars["ansible_host"] = h.Address
	}
	if h.Port != 0 {
		vars["ansible_port"] = h.Port
	}
	if h.User != "" {
		vars["ansible_user"] = h.User
	}
	if h.KeyFile != "" {
		vars["ansible_ssh_private_key_file"] = h.KeyFile
	}
	delete(vars, "ansible_password")
	delete(vars, "ansible_become_password") // secrets stay out of listings
	return vars
}
