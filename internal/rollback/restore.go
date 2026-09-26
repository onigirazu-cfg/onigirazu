package rollback

import (
	"fmt"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ResourceFromResult turns a changed task that captured its target file
// (TaskResult.Before) into a snapshot resource whose rollback operation puts
// the file back: its old content, mode and owner, or its absence. seq is the
// task's position in the run; later changes are undone first.
func ResourceFromResult(t types.TaskResult, host string, seq int) (ResourceSnapshot, bool) {
	before := t.Before
	if before == nil {
		return ResourceSnapshot{}, false
	}
	path, _ := before["path"].(string)
	r := ResourceSnapshot{
		Type: "file", Identifier: path, Host: host, State: before,
		Action: "modified", Module: t.Module, TaskName: t.TaskName,
	}
	args := map[string]interface{}{}
	for _, k := range []string{"_become", "_become_user", "_become_method"} {
		if v, ok := before[k]; ok {
			args[k] = v
		}
	}
	kind, _ := before["kind"].(string)
	switch kind {
	case "packages", "service", "user", "group":
		return systemResource(t, host, seq, before, args)
	}
	switch {
	case kind == "absent":
		args["path"], args["state"] = path, "absent"
		r.RollbackOp = &RollbackOperation{Module: "file", Args: args, Order: seq}
	case kind == "file" && before["content"] != nil:
		args["dest"], args["content"] = path, before["content"]
		copyAttrs(args, before)
		r.RollbackOp = &RollbackOperation{Module: "copy", Args: args, Order: seq}
	case kind == "directory":
		args["path"], args["state"] = path, "directory"
		copyAttrs(args, before)
		r.RollbackOp = &RollbackOperation{Module: "file", Args: args, Order: seq}
	default:
		r.State = map[string]interface{}{"path": path, "kind": kind,
			"reason": fmt.Sprintf("a %s whose content was not kept (binary or larger than 1 MiB)", kind)}
		return r, true
	}
	r.Reversible = true
	return r, true
}

func copyAttrs(args, before map[string]interface{}) {
	for _, k := range []string{"mode", "owner", "group"} {
		if v, ok := before[k].(string); ok && v != "" {
			args[k] = v
		}
	}
}

// systemResource is the rollback of a package, service, user or group task
func systemResource(t types.TaskResult, host string, seq int, before, args map[string]interface{}) (ResourceSnapshot, bool) {
	kind, _ := before["kind"].(string)
	name, _ := before["name"].(string)
	r := ResourceSnapshot{Type: kind, Identifier: name, Host: host, State: before, Action: "modified", Module: t.Module, TaskName: t.TaskName}
	notReversible := func(reason string) (ResourceSnapshot, bool) {
		r.State = map[string]interface{}{"kind": kind, "reason": reason}
		return r, true
	}
	switch kind {
	case "packages":
		was := map[string]bool{}
		installed, _ := before["installed"].([]interface{})
		for _, n := range installed {
			was[fmt.Sprint(n)] = true
		}
		names, _ := before["names"].([]interface{})
		var add, remove []interface{}
		for _, n := range names {
			if was[fmt.Sprint(n)] {
				add = append(add, n)
			} else {
				remove = append(remove, n)
			}
		}
		r.Identifier = fmt.Sprint(names)
		switch state, _ := before["state"].(string); {
		case state == "absent" && len(add) > 0:
			args["name"], args["state"] = add, "present"
		case state != "absent" && len(remove) > 0:
			args["name"], args["state"] = remove, "absent"
		default:
			return notReversible("upgrades of installed packages are not undone")
		}
		r.RollbackOp = &RollbackOperation{Module: "package", Args: args, Order: seq}
	case "service":
		active, _ := before["active"].(string)
		args["name"], args["state"] = name, map[bool]string{true: "started", false: "stopped"}[active == "active"]
		switch enabled, _ := before["enabled"].(string); enabled {
		case "enabled":
			args["enabled"] = true
		case "disabled":
			args["enabled"] = false
		}
		r.RollbackOp = &RollbackOperation{Module: "service", Args: args, Order: seq}
	case "user", "group":
		if existed, _ := before["exists"].(bool); existed {
			return notReversible("changes to an existing " + kind + " are not undone")
		}
		// created by the run: removed again (a user's home stays)
		args["name"], args["state"] = name, "absent"
		r.RollbackOp = &RollbackOperation{Module: kind, Args: args, Order: seq}
	}
	r.Reversible = true
	return r, true
}
