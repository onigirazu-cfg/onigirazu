package engine

import (
	"context"
	"reflect"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/tagfilter"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestCountPlayTasks(t *testing.T) {
	role := &types.Role{
		PreTasks:  []types.Task{{Name: "role pre"}},
		Tasks:     []types.Task{{Block: []types.Task{{Name: "role task"}}, Always: []types.Task{{Name: "role always"}}, Rescue: []types.Task{{Name: "role rescue"}}}},
		PostTasks: []types.Task{{Name: "role post"}},
		Handlers:  []types.Task{{Name: "role handler"}},
	}
	for _, tc := range []struct {
		name string
		play types.Play
		want int
	}{
		{"empty", types.Play{}, 0},
		{"phases", types.Play{PreTasks: []types.Task{{}}, Tasks: []types.Task{{}, {}}, PostTasks: []types.Task{{}}}, 4},
		{"nested blocks", types.Play{Tasks: []types.Task{{Block: []types.Task{{}, {Block: []types.Task{{}, {}}, Always: []types.Task{{}}, Rescue: []types.Task{{}}}}, Always: []types.Task{{}}, Rescue: []types.Task{{}, {}}}}}, 5},
		{"included role", types.Play{Tasks: []types.Task{{IncludedRole: &types.Role{Tasks: []types.Task{{IncludedRole: role}, {}}}}}}, 5},
		{"loaded roles", types.Play{RoleObjects: []*types.Role{nil, role, role}, Roles: []types.RoleReference{{Name: "unloaded"}}}, 8},
		{"runtime estimates", types.Play{Tasks: []types.Task{{Loop: &types.Loop{Items: []interface{}{1, 2, 3}}}, {Module: "include_tasks"}}}, 2},
		{"checks excluded", types.Play{Handlers: []types.Task{{}}, HealthCheck: []types.Task{{}}, Verify: []map[string]interface{}{{"command": "true"}}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := (&ExecutionEngine{}).countPlayTasks(context.Background(), &tc.play); got != tc.want {
				t.Fatalf("count = %d, want %d", got, tc.want)
			}
		})
	}
}

type planObserver struct {
	ExecutionObserverI
	play  string
	hosts []string
	tasks int
	calls int
}

func (o *planObserver) OnPlayPlan(play string, hosts []string, tasks int) {
	o.play, o.hosts, o.tasks = play, hosts, tasks
	o.calls++
}

func TestPlayPlanObserver(t *testing.T) {
	obs := &planObserver{}
	// An observer without the optional interface is still accepted.
	e := &ExecutionEngine{observers: []ExecutionObserverI{struct{ ExecutionObserverI }{}, obs}}
	play := &types.Play{Name: "plan", Tasks: []types.Task{{}, {}}}
	e.notifyPlayPlan(context.Background(), play, []types.Host{{Name: "first"}, {Name: "second"}})
	if obs.calls != 1 || obs.play != "plan" || obs.tasks != 2 || !reflect.DeepEqual(obs.hosts, []string{"first", "second"}) {
		t.Fatalf("plan: %+v", obs)
	}
	e.verifyOnly = true
	e.notifyPlayPlan(context.Background(), play, []types.Host{{Name: "first"}})
	if obs.calls != 2 || obs.tasks != 0 {
		t.Fatalf("verify plan: %+v", obs)
	}
}

func TestCountPlayTasksTagFilter(t *testing.T) {
	filter, err := tagfilter.New("deploy", "")
	if err != nil {
		t.Fatal(err)
	}
	e := &ExecutionEngine{tagFilter: filter}
	play := &types.Play{
		Tasks: []types.Task{
			{Tags: []string{"deploy"}},
			{},
			{Tags: []string{"deploy"}, Block: []types.Task{{}, {}}},
		},
		RoleObjects: []*types.Role{{Tasks: []types.Task{{}}}, {Tasks: []types.Task{{}}}},
		Roles:       []types.RoleReference{{Name: "tagged", Tags: []string{"deploy"}}, {Name: "plain"}},
	}
	if got := e.countPlayTasks(context.Background(), play); got != 4 {
		t.Fatalf("count = %d, want 4", got)
	}
}

func TestCountPlayTasksRoleDependenciesAndFilteredInclude(t *testing.T) {
	e := &ExecutionEngine{roleLoader: parser.NewRoleLoader(nil, t.TempDir())}
	dependency := &types.Role{Name: "dependency", Tasks: []types.Task{{}, {}}}
	e.roleLoader.GetCache()[dependency.Name] = dependency
	role := &types.Role{Name: "main", Meta: types.RoleMeta{Dependencies: []types.RoleDependency{{Name: dependency.Name}}}, Tasks: []types.Task{{}}}
	play := &types.Play{RoleObjects: []*types.Role{role}, Tasks: []types.Task{{IncludedRole: role}}}
	if got := e.countPlayTasks(context.Background(), play); got != 6 {
		t.Fatalf("count = %d, want 6", got)
	}
	filter, err := tagfilter.New("deploy", "")
	if err != nil {
		t.Fatal(err)
	}
	e.tagFilter = filter
	tagged := &types.Role{Name: "tagged", Tasks: []types.Task{{Tags: []string{"deploy"}}}}
	play = &types.Play{Tasks: []types.Task{{IncludedRole: tagged}, {Tags: []string{"deploy"}, IncludedRole: tagged}}}
	if got := e.countPlayTasks(context.Background(), play); got != 1 {
		t.Fatalf("filtered include count = %d, want 1", got)
	}
}
