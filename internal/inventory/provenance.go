package inventory

import (
	"fmt"
	"maps"
	"reflect"
	"sort"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// InventorySources records the last source of each inventory variable.
type InventorySources struct {
	Hosts  map[string]map[string]string
	Groups map[string]map[string]string
	// Separate host definitions can survive in different groups after a merge.
	hostDefinitions map[*types.Host]map[string]string
	// Values of the same variable map share labels even when hosts are copied.
	hostVariables map[reflect.Value]map[string]string
}

func newInventorySources() InventorySources {
	return InventorySources{Hosts: make(map[string]map[string]string), Groups: make(map[string]map[string]string), hostDefinitions: make(map[*types.Host]map[string]string), hostVariables: make(map[reflect.Value]map[string]string)}
}

func (sources InventorySources) hostLabels(host *types.Host) map[string]string {
	key := reflect.ValueOf(host.Vars)
	labels := sources.hostVariables[key]
	if labels == nil {
		labels = make(map[string]string, len(host.Vars))
		sources.hostVariables[key] = labels
	}
	sources.hostDefinitions[host] = labels
	return labels
}

func (sources InventorySources) recordHostVars(host *types.Host, vars map[string]interface{}, source string) {
	if host.Vars == nil {
		return
	}
	labels := sources.hostLabels(host)
	for name := range vars {
		labels[name] = source
	}
}

func recordSource(target map[string]map[string]string, scope, name, source string) {
	if target[scope] == nil {
		target[scope] = make(map[string]string)
	}
	target[scope][name] = source
}

func recordVarSources(target map[string]map[string]string, scope string, vars map[string]interface{}, source string) {
	for name := range vars {
		recordSource(target, scope, name, source)
	}
}

// VariableProvenance describes an effective inventory variable without its value.
type VariableProvenance struct {
	Name   string `json:"name"`
	From   string `json:"from"`
	Source string `json:"source"`
}

// EffectiveProvenance resolves a host's inventory variable origins using the
// engine's group order and variable visitor. It does not mutate the inventory.
func EffectiveProvenance(inv *types.Inventory, hostName string, sources InventorySources) ([]VariableProvenance, error) {
	if inv == nil {
		return nil, fmt.Errorf("no inventory loaded")
	}
	view := &types.Inventory{Groups: make(map[string]*types.Group, len(inv.Groups))}
	m := &Manager{inventory: view, groupVars: make(map[string]map[string]interface{}, len(inv.Groups)), groupParents: make(map[string][]string)}
	for name, group := range inv.Groups {
		copy := *group
		copy.Hosts = maps.Clone(group.Hosts)
		view.Groups[name] = &copy
		m.groupVars[name] = group.Vars
		for _, child := range group.Children {
			m.groupParents[child] = append(m.groupParents[child], name)
		}
	}
	// Validate the graph before using the engine's inheritance resolver.
	visiting, done := make(map[string]bool), make(map[string]bool)
	var check func(string) error
	check = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("inventory group cycle at %s", name)
		}
		if done[name] {
			return nil
		}
		group := view.Groups[name]
		if group == nil {
			return fmt.Errorf("inventory group %s not found", name)
		}
		visiting[name] = true
		for _, child := range group.Children {
			if err := check(child); err != nil {
				return err
			}
		}
		delete(visiting, name)
		done[name] = true
		return nil
	}
	for name := range view.Groups {
		if err := check(name); err != nil {
			return nil, err
		}
	}
	if err := m.resolveGroupInheritance(view); err != nil {
		return nil, err
	}
	var host *types.Host
	// Prefer a group definition, as GetHosts does; top-level hosts may also be
	// inspected even when the inventory does not place them in a group.
	names := make([]string, 0, len(view.Groups))
	for name := range view.Groups {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if entry := view.Groups[name].Hosts[hostName]; entry != nil {
			host = entry
			break
		}
	}
	if host == nil {
		for i := range inv.Hosts {
			if inv.Hosts[i].Name == hostName {
				host = &inv.Hosts[i]
				break
			}
		}
	}
	if host == nil {
		host = &types.Host{Name: hostName}
	}
	selected := *host
	selected.Name = hostName
	effective := make(map[string]VariableProvenance)
	visit := func(group string, entry *types.Host, vars map[string]interface{}) {
		labels := sources.Groups[group]
		from := "group:" + group
		if entry != nil {
			from = "host"
			labels = sources.hostDefinitions[entry]
			if labels == nil {
				labels = sources.Hosts[hostName]
			}
		}
		for name, value := range vars {
			if name == "_original_port" || name == "group_names" {
				continue
			}
			// Inspection keeps names for fields absent from execution vars.
			if _, nameOnly := value.(types.InventoryNameOnly); nameOnly {
				continue
			}
			effective[name] = VariableProvenance{Name: name, From: from, Source: labels[name]}
		}
	}
	m.visitHostVariables(&selected, m.orderedHostGroups(hostName), visit)
	base := maps.Clone(effective)
	visit("", host, host.Vars)
	resolved := effective
	// The engine reapplies the selected group's host definition last. Native
	// inventories may keep different definitions; report only unambiguous origins.
	for _, name := range names {
		candidate := view.Groups[name].Hosts[hostName]
		if candidate == nil || candidate == host {
			continue
		}
		effective = maps.Clone(base)
		visit("", candidate, candidate.Vars)
		if !maps.Equal(resolved, effective) {
			return nil, fmt.Errorf("host %s has ambiguous variable origins across group definitions", hostName)
		}
	}
	effective = resolved
	out := make([]VariableProvenance, 0, len(effective))
	for _, origin := range effective {
		out = append(out, origin)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
