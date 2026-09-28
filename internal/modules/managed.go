package modules

import (
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Managed state: every task that runs reports the resources it keeps on the
// host (with what it found there before), so apply can remember them and
// clean up after a task leaves the playbook.

// declaredResources lists what a module call manages; before is the
// registry's capture (nil in check mode)
func declaredResources(module string, args map[string]interface{}, before map[string]interface{}) []types.ManagedResource {
	if before != nil && before["error"] != nil {
		before = nil
	}
	state := getStringArg(args, "state", "")
	switch module {
	case "apt", "yum", "dnf", "package":
		var out []types.ManagedResource
		for _, name := range packageNames(args) {
			out = append(out, types.ManagedResource{Type: "package", ID: name,
				Absent: state == "absent" || state == "removed", Before: packageBefore(before, name)})
		}
		return out
	case "service", "systemd":
		if name := getStringArg(args, "name", ""); name != "" {
			return []types.ManagedResource{{Type: "service", ID: name, Before: before}}
		}
		return nil
	case "user", "group":
		if name := getStringArg(args, "name", ""); name != "" {
			return []types.ManagedResource{{Type: module, ID: name, Absent: state == "absent", Before: before}}
		}
		return nil
	}
	keys, ok := captureModules[module]
	if !ok {
		return nil
	}
	path := ""
	for _, k := range keys {
		if path = getStringArg(args, k, ""); path != "" {
			break
		}
	}
	if path == "" {
		return nil
	}
	// lineinfile/blockinfile state: absent removes a line, the file stays
	absent := module == "file" && state == "absent"
	return []types.ManagedResource{{Type: "file", ID: path, Absent: absent, Before: before}}
}

// packageBefore narrows a package capture to one name
func packageBefore(before map[string]interface{}, name string) map[string]interface{} {
	if before == nil {
		return nil
	}
	one := map[string]interface{}{}
	for k, v := range before {
		one[k] = v
	}
	one["names"] = []interface{}{name}
	var installed []interface{}
	if list, ok := before["installed"].([]interface{}); ok {
		for _, n := range list {
			if n == name {
				installed = append(installed, n)
			}
		}
	}
	one["installed"] = installed
	return one
}
