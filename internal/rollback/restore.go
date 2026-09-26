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
