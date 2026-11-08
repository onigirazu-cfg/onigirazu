package engine

import (
	"context"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

const roleScopeKey = "__onigirazu_role_scope"

// Keep published defaults separate from play variables: the next role's
// defaults may replace earlier defaults, but never inventory or play vars.
// Scope snapshots are immutable while host tasks run in parallel.
type roleVariableScope struct{ defaults, vars, params map[string]interface{} }

func (e *ExecutionEngine) exposeRole(ctx context.Context, role *types.Role, variables map[string]interface{}) error {
	order, err := e.roleLoader.ResolveDependencyOrder(ctx, role)
	if err != nil {
		return err
	}
	scope, _ := variables[roleScopeKey].(roleVariableScope)
	for _, r := range order {
		scope.defaults = e.mergeVariables(scope.defaults, r.Defaults)
		scope.vars = e.mergeVariables(scope.vars, r.Vars)
	}
	variables[roleScopeKey] = scope
	return nil
}

func rolePublic(task *types.Task) bool {
	if v, ok := task.Args["public"]; ok {
		return v == true || v == "true" || v == "yes"
	}
	return types.ShortModuleName(task.Module) == "import_role"
}

// Static imports expose variables even to tasks preceding the import, also
// a role imported by an imported role
func (e *ExecutionEngine) exposeImportedRoles(ctx context.Context, tasks []types.Task, vars map[string]interface{}) error {
	return e.exposeImportedRolesDepth(ctx, tasks, vars, 0)
}

// maxImportDepth stops a role that imports itself
const maxImportDepth = 32

func (e *ExecutionEngine) exposeImportedRolesDepth(ctx context.Context, tasks []types.Task, vars map[string]interface{}, depth int) error {
	if depth > maxImportDepth {
		return nil
	}
	for i := range tasks {
		t := &tasks[i]
		if t.IncludedRole != nil && types.ShortModuleName(t.Module) == "import_role" && rolePublic(t) {
			if err := e.exposeRole(ctx, t.IncludedRole, vars); err != nil {
				return err
			}
			if err := e.exposeImportedRolesDepth(ctx, t.IncludedRole.Tasks, vars, depth+1); err != nil {
				return err
			}
		}
		for _, list := range [][]types.Task{t.Block, t.Rescue, t.Always} {
			if err := e.exposeImportedRolesDepth(ctx, list, vars, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}
