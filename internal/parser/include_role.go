package parser

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// loadIncludedRole loads the role of an include_role/import_role task:
// name, and tasks_from for another file than tasks/main.yml. Its tasks get
// the task's tags, so --tags selects them with the include.
func (p *EnhancedParser) loadIncludedRole(ctx context.Context, task *types.Task, depth int) error {
	name, _ := task.Args["name"].(string)
	if name == "" {
		return fmt.Errorf("%s: argument 'name' is required", task.Module)
	}
	loaded, err := p.roleLoader.LoadRole(ctx, types.RoleReference{Name: name})
	if err != nil {
		return fmt.Errorf("%s %s: %w", task.Module, name, err)
	}
	role := *loaded // the cached role stays as it is
	if from, _ := task.Args["tasks_from"].(string); from != "" {
		if filepath.Ext(from) == "" {
			from += ".yml"
		}
		path := filepath.Join(role.Path, "tasks", from)
		if role.Tasks, err = p.loadIncludedTasks(ctx, path); err != nil {
			return fmt.Errorf("%s %s: %w", task.Module, name, err)
		}
		resolveRoleFiles(role.Tasks, role.Path)
	}
	if role.Tasks, err = p.expandIncludes(ctx, role.Tasks, filepath.Join(role.Path, "tasks"), depth+1); err != nil {
		return fmt.Errorf("%s %s: %w", task.Module, name, err)
	}
	if len(task.Tags) > 0 {
		tasks := make([]types.Task, len(role.Tasks))
		for i, t := range role.Tasks {
			t.Tags = append(append([]string{}, task.Tags...), t.Tags...)
			tasks[i] = t
		}
		role.Tasks = tasks
	}
	task.IncludedRole = &role
	if task.Name == "" || strings.HasSuffix(task.Name, " task") {
		task.Name = task.Module + " " + name
	}
	return nil
}
