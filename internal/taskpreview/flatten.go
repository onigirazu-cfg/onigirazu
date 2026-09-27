package taskpreview

import "github.com/onigirazu-cfg/onigirazu/pkg/types"

// FlatTask is a task as the engine runs it: in play order, with the tags it
// inherits from its play, roles: entry and blocks
type FlatTask struct {
	Task *types.Task
	Type string // pre_task, role, task, post_task
	Role string
	Tags []string
}

// Flatten lists the tasks of a play in the order the engine runs them:
// pre_tasks, roles, tasks, post_tasks. Blocks are replaced by their block,
// rescue and always tasks. Handlers run only when notified and are left out.
func Flatten(play *types.Play) []FlatTask {
	var out []FlatTask
	add := func(tasks []types.Task, typ, role string, inherited []string) {
		out = appendFlat(out, tasks, typ, role, inherited)
	}
	add(play.PreTasks, "pre_task", "", play.Tags)
	for i, role := range play.RoleObjects {
		if role == nil {
			continue
		}
		tags := play.Tags
		if i < len(play.Roles) {
			tags = append(append([]string{}, play.Tags...), play.Roles[i].Tags...)
		}
		add(role.Tasks, "role", role.Name, tags)
	}
	add(play.Tasks, "task", "", play.Tags)
	add(play.PostTasks, "post_task", "", play.Tags)
	return out
}

func appendFlat(out []FlatTask, tasks []types.Task, typ, role string, inherited []string) []FlatTask {
	for i := range tasks {
		t := &tasks[i]
		tags := append(append([]string{}, inherited...), t.Tags...)
		if len(t.Block) > 0 || len(t.Rescue) > 0 || len(t.Always) > 0 {
			out = appendFlat(out, t.Block, typ, role, tags)
			out = appendFlat(out, t.Rescue, typ, role, tags)
			out = appendFlat(out, t.Always, typ, role, tags)
			continue
		}
		out = append(out, FlatTask{Task: t, Type: typ, Role: role, Tags: tags})
	}
	return out
}
