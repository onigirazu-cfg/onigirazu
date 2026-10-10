package engine

import (
	"fmt"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// inventoryWriter is the inventory manager's part that add_host and
// group_by change
type inventoryWriter interface {
	AddHost(name string, vars map[string]interface{}, groups []string) error
	AddHostToGroup(hostName, group string, parents []string) error
}

// applyInventoryChange puts what add_host and group_by returned into the
// inventory for the rest of the run (later plays, hostvars, groups)
func (e *ExecutionEngine) applyInventoryChange(task *types.Task, host *types.Host, result types.TaskResult) {
	module := types.ShortModuleName(task.Module)
	if module != "add_host" && module != "group_by" {
		return
	}
	inv, ok := e.inventoryMgr.(inventoryWriter)
	if !ok {
		e.logger.Warn("%s: the inventory cannot be changed during the run", module)
		return
	}
	var err error
	switch module {
	case "add_host":
		out, _ := result.Output["add_host"].(map[string]interface{})
		name, _ := out["host_name"].(string)
		vars, _ := out["host_vars"].(map[string]interface{})
		err = inv.AddHost(name, vars, stringList(out["groups"]))
	case "group_by":
		group, _ := result.Output["add_group"].(string)
		err = inv.AddHostToGroup(host.Name, group, stringList(result.Output["parent_groups"]))
	}
	if err != nil {
		e.logger.Warn("%s on %s: %v", module, host.Name, err)
		return
	}
	// hostvars and groups are read from the inventory again
	e.inventoryView.reset()
}

func stringList(v interface{}) []string {
	switch l := v.(type) {
	case []string:
		return l
	case []interface{}:
		out := make([]string, 0, len(l))
		for _, x := range l {
			out = append(out, fmt.Sprint(x))
		}
		return out
	}
	return nil
}
