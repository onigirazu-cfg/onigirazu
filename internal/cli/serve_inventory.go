package cli

import (
	"context"
	"encoding/json"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type serveInventorySnapshot struct {
	mu        sync.Mutex // one load of this source at a time
	inventory *types.Inventory
	sources   inventory.InventorySources
	err       error
	loadedAt  time.Time
}

type serveInventorySource struct {
	Source   string                      `json:"source"`
	Jobs     []string                    `json:"jobs"`
	LoadedAt time.Time                   `json:"loaded_at"`
	Error    string                      `json:"error,omitempty"`
	Groups   []serveInventoryDetailGroup `json:"groups"`
	Hosts    []serveInventoryHost        `json:"hosts"`
}

type serveInventoryDetailGroup struct {
	Name       string             `json:"name"`
	Children   []string           `json:"children"`
	Hosts      []string           `json:"hosts"`
	Vars       []string           `json:"vars"`
	VarSources *map[string]string `json:"var_sources,omitempty"`
}

type serveInventoryHost struct {
	Name      string                          `json:"name"`
	Address   *string                         `json:"address,omitempty"`
	Groups    []string                        `json:"groups"`
	Vars      []string                        `json:"vars"`
	Effective *[]inventory.VariableProvenance `json:"effective,omitempty"`
}

func (s *server) inventoryPaths(j *ServeJob) []string {
	paths := s.inventory
	if j.Inventory != "" {
		paths = []string{j.Inventory}
	}
	log, _ := logger.NewEnhancedLogger("error", "text", s.out)
	return inventorySources(log, j.Playbook, paths)
}

func inventorySourceKey(paths []string) string {
	sorted := append([]string{}, paths...)
	sort.Strings(sorted)
	key, _ := json.Marshal(sorted)
	return string(key)
}

// inventoryLoadTimeout bounds a dynamic inventory that never answers
const inventoryLoadTimeout = time.Minute

// inventorySnapshot returns the source's snapshot, loading it when missing or
// older than 5 minutes; readers of the same source wait for one load, other
// sources are not held up by it
func (s *server) inventorySnapshot(paths []string) (inv *types.Inventory, sources inventory.InventorySources, loadedAt time.Time, err error) {
	key := inventorySourceKey(paths)
	s.inventoryMu.Lock()
	if s.inventorySnapshots == nil {
		s.inventorySnapshots = make(map[string]*serveInventorySnapshot)
	}
	snapshot := s.inventorySnapshots[key]
	if snapshot == nil {
		snapshot = &serveInventorySnapshot{}
		s.inventorySnapshots[key] = snapshot
	}
	s.inventoryMu.Unlock()
	snapshot.mu.Lock()
	defer snapshot.mu.Unlock()
	if !snapshot.loadedAt.IsZero() && time.Since(snapshot.loadedAt) < 5*time.Minute {
		return snapshot.inventory, snapshot.sources, snapshot.loadedAt, snapshot.err
	}
	ctx, cancel := context.WithTimeout(context.Background(), inventoryLoadTimeout)
	defer cancel()
	if s.inventoryLoad != nil {
		snapshot.inventory, snapshot.err = s.inventoryLoad(paths)
	} else {
		log, err := logger.NewEnhancedLogger("error", "text", s.out)
		if err != nil {
			snapshot.err = err
		} else {
			p := parser.NewEnhancedParser(template.NewEngine(), log)
			p.PreserveInventoryStructure()
			loader := inventory.NewMultiSourceLoader(p, log, cache.NewManager(5*time.Minute), 10*time.Minute).TrackSources()
			snapshot.inventory, snapshot.err = loader.LoadFromMultipleSources(ctx, paths)
			snapshot.sources = loader.Sources()
		}
	}
	snapshot.loadedAt = time.Now().UTC()
	return snapshot.inventory, snapshot.sources, snapshot.loadedAt, snapshot.err
}

// inventorySourceLabel: operators see every source; viewers see file paths but
// not an inline inventory (user@host:port,...), which is connection data
func inventorySourceLabel(paths []string, operator bool) string {
	if operator {
		return strings.Join(paths, ", ")
	}
	labels := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.Contains(path, ",") {
			labels = append(labels, "inline")
		} else {
			labels = append(labels, path)
		}
	}
	return strings.Join(labels, ", ")
}

func inventoryVariableSourceLabel(source string, operator bool) string {
	if filepath.IsAbs(source) {
		return source
	}
	return inventorySourceLabel([]string{source}, operator)
}

func inventoryVarNames(vars map[string]interface{}) []string {
	names := make([]string, 0, len(vars))
	for name := range vars {
		if name != "_original_port" { // the parser's own bookkeeping
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (s *server) inventoryView(operator bool) []serveInventorySource {
	out := make([]serveInventorySource, 0)
	indices := make(map[string]int)
	for i := range s.cfg.Jobs {
		j := &s.cfg.Jobs[i]
		paths := s.inventoryPaths(j)
		key := inventorySourceKey(paths)
		if index, ok := indices[key]; ok {
			out[index].Jobs = append(out[index].Jobs, j.Name)
			continue
		}
		inv, sources, loadedAt, loadErr := s.inventorySnapshot(paths)
		source := serveInventorySource{Source: inventorySourceLabel(paths, operator), Jobs: []string{j.Name}, LoadedAt: loadedAt,
			Groups: make([]serveInventoryDetailGroup, 0), Hosts: make([]serveInventoryHost, 0)}
		if loadErr != nil {
			// Parser and dynamic inventory errors can contain variable values.
			source.Error = "Failed to load inventory"
		} else {
			groupNames := make([]string, 0, len(inv.Groups))
			for name := range inv.Groups {
				groupNames = append(groupNames, name)
			}
			if _, ok := inv.Groups["all"]; !ok {
				groupNames = append(groupNames, "all")
			}
			sort.Strings(groupNames)
			groupsByHost := make(map[string][]string)
			for _, name := range groupNames {
				group := inv.Groups[name]
				if group == nil {
					group = inventoryImplicitAll(inv)
				}
				hosts := make([]string, 0, len(group.Hosts))
				for host := range group.Hosts {
					hosts = append(hosts, host)
				}
				sort.Strings(hosts)
				children := append([]string{}, group.Children...)
				sort.Strings(children)
				entry := serveInventoryDetailGroup{Name: name, Children: children, Hosts: hosts, Vars: inventoryVarNames(group.Vars)}
				if operator {
					labels := make(map[string]string, len(group.Vars))
					for variable, path := range sources.Groups[name] {
						labels[variable] = inventoryVariableSourceLabel(path, operator)
					}
					entry.VarSources = &labels
				}
				source.Groups = append(source.Groups, entry)
				members := inventoryGroupHosts(inv, name, make(map[string]bool))
				for _, host := range members {
					groupsByHost[host] = append(groupsByHost[host], name)
				}
			}
			hosts := inventoryHosts(inv)
			for _, host := range hosts {
				name := host.Name
				groups := groupsByHost[name]
				if groups == nil {
					groups = []string{}
				}
				entry := serveInventoryHost{Name: name, Groups: groups, Vars: inventoryVarNames(host.Vars)}
				if operator {
					address := host.Address
					entry.Address = &address
					effective, err := inventory.EffectiveProvenance(inv, name, sources)
					if err != nil {
						source.Error = "Failed to resolve inventory variables"
						effective = []inventory.VariableProvenance{}
					}
					for i := range effective {
						effective[i].Source = inventoryVariableSourceLabel(effective[i].Source, operator)
					}
					entry.Effective = &effective
				}
				source.Hosts = append(source.Hosts, entry)
			}
		}
		indices[key] = len(out)
		out = append(out, source)
	}
	for i := range out {
		sort.Strings(out[i].Jobs)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Source < out[j].Source })
	return out
}

func inventoryGroupHosts(inv *types.Inventory, name string, seen map[string]bool) []string {
	if seen[name] {
		return nil
	}
	seen[name] = true
	hosts := make([]string, 0)
	if name == "all" {
		for _, host := range inventoryHosts(inv) {
			hosts = append(hosts, host.Name)
		}
		return hosts
	}
	if group := inv.Groups[name]; group != nil {
		for host := range group.Hosts {
			hosts = append(hosts, host)
		}
		for _, child := range group.Children {
			hosts = append(hosts, inventoryGroupHosts(inv, child, seen)...)
		}
	}
	sort.Strings(hosts)
	return slices.Compact(hosts)
}

func inventoryHosts(inv *types.Inventory) []types.Host {
	byName := make(map[string]types.Host, len(inv.Hosts))
	for _, host := range inv.Hosts {
		byName[host.Name] = host
	}
	names := make([]string, 0, len(inv.Groups))
	for name := range inv.Groups {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		for name, host := range inv.Groups[name].Hosts {
			if existing, ok := byName[name]; ok && host != nil && len(host.Vars) > 0 {
				// the same host in several places: show every variable name it gets
				merged := maps.Clone(existing.Vars)
				if merged == nil {
					merged = make(map[string]interface{}, len(host.Vars))
				}
				for k := range host.Vars {
					if _, set := merged[k]; !set {
						merged[k] = nil
					}
				}
				existing.Vars = merged
				byName[name] = existing
			}
			if _, ok := byName[name]; !ok {
				entry := types.Host{Name: name, Address: name}
				if host != nil {
					entry = *host
				}
				entry.Name = name
				if entry.Address == "" {
					entry.Address = name
				}
				byName[name] = entry
			}
		}
	}
	hosts := make([]types.Host, 0, len(byName))
	for _, host := range byName {
		hosts = append(hosts, host)
	}
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].Name < hosts[j].Name })
	return hosts
}

func inventoryImplicitAll(inv *types.Inventory) *types.Group {
	group := &types.Group{Name: "all", Hosts: make(map[string]*types.Host), Children: []string{}}
	grouped := make(map[string]bool)
	children := make(map[string]bool)
	for _, g := range inv.Groups {
		for name := range g.Hosts {
			grouped[name] = true
		}
		for _, name := range g.Children {
			children[name] = true
		}
	}
	for name := range inv.Groups {
		if !children[name] {
			group.Children = append(group.Children, name)
		}
	}
	for _, host := range inventoryHosts(inv) {
		if !grouped[host.Name] {
			entry := host
			group.Hosts[host.Name] = &entry
		}
	}
	return group
}
