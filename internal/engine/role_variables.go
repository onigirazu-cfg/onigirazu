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

// Static imports expose variables even to tasks preceding the import.
func (e *ExecutionEngine) exposeImportedRoles(ctx context.Context, tasks []types.Task, vars map[string]interface{}) error {
	for i := range tasks {
		t := &tasks[i]
		if t.IncludedRole != nil && types.ShortModuleName(t.Module) == "import_role" && rolePublic(t) {
			if err := e.exposeRole(ctx, t.IncludedRole, vars); err != nil {
				return err
			}
		}
		for _, list := range [][]types.Task{t.Block, t.Rescue, t.Always} {
			if err := e.exposeImportedRoles(ctx, list, vars); err != nil {
				return err
			}
		}
	}
	return nil
}
