package engine

import (
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ansibleFacts gives the gathered facts their Ansible names, so playbooks
// written for Ansible work: ansible_os_family, ansible_hostname, ... and the
// same values without the prefix in ansible_facts
func ansibleFacts(sf *cache.SystemFacts, onigirazu map[string]interface{}) map[string]interface{} {
	short, _, _ := strings.Cut(sf.Hostname, ".")
	major, _, _ := strings.Cut(sf.OSVersion, ".")
	facts := map[string]interface{}{
		"hostname":                   short,
		"nodename":                   sf.Hostname,
		"fqdn":                       sf.FQDN,
		"os_family":                  sf.OSFamily,
		"distribution":               ansibleDistribution(sf.Distribution),
		"distribution_version":       sf.OSVersion,
		"distribution_major_version": major,
		"distribution_release":       sf.OSCodename,
		"architecture":               sf.Architecture,
		"system":                     sf.Kernel,
		"kernel":                     sf.KernelVersion,
		"processor_cores":            sf.CPUCores,
		"processor_vcpus":            sf.CPUCores,
		"memtotal_mb":                sf.MemTotalMB,
		"default_ipv4":               map[string]interface{}{"address": sf.DefaultIPv4},
		"virtualization_type":        sf.VirtualizationType,
		"virtualization_role":        sf.VirtualizationRole,
		"local":                      localFacts(sf.Local),
		"user_id":                    sf.Username,
		"pkg_mgr":                    packageManager(sf.OSFamily, sf.Distribution, major),
		"date_time":                  onigirazu["onigirazu_date_time"],
		"env":                        onigirazu["onigirazu_env"],
	}
	vars := map[string]interface{}{"ansible_facts": facts}
	for k, v := range facts {
		vars["ansible_"+k] = v
	}
	return vars
}

// localFacts is ansible_local, an empty map when there are none
func localFacts(local map[string]interface{}) map[string]interface{} {
	if local == nil {
		return map[string]interface{}{}
	}
	return local
}

// withMagicVariables adds hostvars and groups to the variables of a task:
// a snapshot taken before the task runs on its hosts
func (e *ExecutionEngine) withMagicVariables(variables map[string]interface{}) map[string]interface{} {
	hosts, groups, ok := e.inventoryView.get(e.inventoryMgr)
	if !ok {
		return variables
	}
	// nested task lists (blocks, run_once) come here again: start from the
	// variables without an earlier snapshot, or hostvars would nest
	base := make(map[string]interface{}, len(variables))
	for k, v := range variables {
		if k != "hostvars" && k != "groups" {
			base[k] = v
		}
	}
	out := make(map[string]interface{}, len(base)+2)
	for k, v := range base {
		out[k] = v
	}

	hostvars := make(map[string]interface{}, len(hosts))
	for i := range hosts {
		hostvars[hosts[i].Name] = e.hostVariables(&hosts[i], base)
	}
	out["hostvars"] = hostvars
	out["groups"] = groups
	return out
}

// inventorySnapshot holds the inventory's hosts and the groups variable for
// a run: building them for every task (a host view per host, every group's
// members) cost a share of the run that grew with the inventory
type inventorySnapshot struct {
	mu     sync.Mutex
	valid  bool
	hosts  []types.Host
	groups map[string]interface{}
}

func (s *inventorySnapshot) reset() {
	s.mu.Lock()
	s.valid, s.hosts, s.groups = false, nil, nil
	s.mu.Unlock()
}

// get returns the snapshot, taking it on first use; ok is false without an
// inventory. The groups map is shared: nothing changes it in place.
func (s *inventorySnapshot) get(inv interfaces.InventoryManager) (hosts []types.Host, groups map[string]interface{}, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.valid {
		return s.hosts, s.groups, true
	}
	all, err := inv.GetHosts("all")
	if err != nil {
		return nil, nil, false
	}
	groups = map[string]interface{}{}
	names := make([]interface{}, 0, len(all))
	for _, h := range all {
		names = append(names, h.Name)
	}
	groups["all"] = names
	groupNames := inv.ListGroups()
	sort.Strings(groupNames)
	for _, name := range groupNames {
		members, err := inv.GetHosts(name)
		if err != nil {
			continue
		}
		list := make([]interface{}, 0, len(members))
		for _, m := range members {
			list = append(list, m.Name)
		}
		groups[name] = list
	}
	s.valid, s.hosts, s.groups = true, all, groups
	return all, groups, true
}

// ansibleDistribution is the distribution name as Ansible spells it: the
// os-release ID "rocky" is "Rocky", "rhel" is "RedHat"
func ansibleDistribution(id string) string {
	names := map[string]string{
		"ubuntu": "Ubuntu", "debian": "Debian", "rocky": "Rocky", "almalinux": "AlmaLinux",
		"centos": "CentOS", "rhel": "RedHat", "redhat": "RedHat", "fedora": "Fedora", "amzn": "Amazon",
		"ol": "OracleLinux", "sles": "SLES", "opensuse-leap": "openSUSE Leap", "arch": "Archlinux",
		"alpine": "Alpine", "linuxmint": "Linux Mint", "darwin": "MacOSX", "macos": "MacOSX",
	}
	if name, ok := names[strings.ToLower(id)]; ok {
		return name
	}
	return id
}

// addMagicVars sets the Ansible variables about the run itself; the caller
// holds the read lock
func (e *ExecutionEngine) addMagicVars(vars map[string]interface{}) {
	vars["ansible_check_mode"] = e.checkMode
	vars["ansible_diff_mode"] = e.showDiff
	if e.limit != "" {
		vars["ansible_limit"] = e.limit
	}
	list := func(names []string, skipFailed bool) []interface{} {
		out := make([]interface{}, 0, len(names))
		for _, n := range names {
			if skipFailed && e.failedHosts[n] {
				continue
			}
			out = append(out, n)
		}
		return out
	}
	batch := e.batchHosts
	if len(batch) == 0 {
		batch = e.playHosts
	}
	vars["ansible_play_hosts_all"] = list(e.playHosts, false)
	vars["ansible_play_batch"] = list(batch, true)
	vars["ansible_play_hosts"] = list(batch, true)
	vars["play_hosts"] = list(batch, true) // the old name, still set by Ansible
}

// SetCheckMode tells the run it is a check (ansible_check_mode)
func (e *ExecutionEngine) SetCheckMode(check bool) {
	e.checkMode = check
}

// packageManager is Ansible's ansible_pkg_mgr from the OS family
func packageManager(family, distribution, major string) string {
	switch strings.ToLower(family) {
	case "debian":
		return "apt"
	case "redhat":
		if strings.EqualFold(distribution, "fedora") {
			return "dnf"
		}
		if n, err := strconv.Atoi(major); err == nil && n < 8 {
			return "yum"
		}
		return "dnf"
	case "suse":
		return "zypper"
	case "archlinux":
		return "pacman"
	case "alpine":
		return "apk"
	}
	return "unknown"
}
