package engine

import (
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
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
	hosts, err := e.inventoryMgr.GetHosts("all")
	if err != nil {
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
	out := make(map[string]interface{})
	for k, v := range base {
		out[k] = v
	}

	hostvars := make(map[string]interface{}, len(hosts))
	for i := range hosts {
		hostvars[hosts[i].Name] = e.hostVariables(&hosts[i], base)
	}
	out["hostvars"] = hostvars

	groups := map[string]interface{}{}
	all := make([]interface{}, 0, len(hosts))
	for _, h := range hosts {
		all = append(all, h.Name)
	}
	groups["all"] = all
	names := e.inventoryMgr.ListGroups()
	sort.Strings(names)
	for _, name := range names {
		members, err := e.inventoryMgr.GetHosts(name)
		if err != nil {
			continue
		}
		list := make([]interface{}, 0, len(members))
		for _, m := range members {
			list = append(list, m.Name)
		}
		groups[name] = list
	}
	out["groups"] = groups
	return out
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
	all, batch := e.playHostLists()
	vars["ansible_play_hosts_all"] = all
	vars["ansible_play_batch"] = batch
	vars["ansible_play_hosts"] = batch
	vars["play_hosts"] = batch // the old name, still set by Ansible
}

// magicLists caches the play host lists of the magic variables: built for
// every host of every task they made each task quadratic in the hosts
type magicLists struct {
	mu         sync.Mutex
	version    uint64
	valid      bool
	all, batch []interface{}
}

// playHostLists are ansible_play_hosts_all and ansible_play_batch, built
// again only when the hosts changed (the caller holds e.mutex for reading).
// The lists are shared: nothing changes them in place.
func (e *ExecutionEngine) playHostLists() (all, batch []interface{}) {
	m := &e.magic
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.valid && m.version == e.hostsVersion {
		return m.all, m.batch
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
	names := e.batchHosts
	if len(names) == 0 {
		names = e.playHosts
	}
	m.all, m.batch = list(e.playHosts, false), list(names, true)
	m.version, m.valid = e.hostsVersion, true
	return m.all, m.batch
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
