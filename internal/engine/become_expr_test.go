package engine

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type becomeExprCall struct {
	host, task string
	become     bool
	user       string
	argument   interface{}
}

type becomeExprModule struct {
	*modules.BaseModule
	mu    sync.Mutex
	calls []becomeExprCall
}

func (m *becomeExprModule) Execute(_ context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	exec := modules.NewMockExecutor(host)
	exec.SetBecome(host.Become, host.BecomeUser, host.BecomeMethod)
	m.mu.Lock()
	m.calls = append(m.calls, becomeExprCall{host: host.Name, task: args["_task_name"].(string),
		become: exec.BecomeSettings.Enabled, user: exec.BecomeSettings.User, argument: args["_become"]})
	m.mu.Unlock()
	return types.TaskResult{Success: true}, nil
}

func becomeExprEngine(t *testing.T) (*ExecutionEngine, *becomeExprModule) {
	t.Helper()
	e, config, _, _, _, _ := createTestEngine()
	config.On("GetDryRun").Return(false)
	e.templateEngine = template.NewEngine()
	registry := modules.NewRegistry()
	module := &becomeExprModule{BaseModule: modules.NewBaseModule("become_probe")}
	require.NoError(t, registry.Register("become_probe", module))
	e.moduleRegistry = registry
	return e, module
}

func TestBecomeExprPerHost(t *testing.T) {
	for _, level := range []string{"task", "play", "block"} {
		t.Run(level, func(t *testing.T) {
			e, module := becomeExprEngine(t)
			hosts := twoHosts()
			hosts[0].Vars = map[string]interface{}{"use_sudo": true}
			hosts[1].Vars = map[string]interface{}{"use_sudo": false}
			task := types.Task{Name: "probe", Module: "become_probe"}
			play := &types.Play{Name: "become", Tasks: []types.Task{task}}
			switch level {
			case "task":
				play.Tasks[0].BecomeExpr, play.Tasks[0].BecomeSet = "{{ use_sudo }}", true
			case "play":
				play.BecomeExpr = "{{ use_sudo }}"
			case "block":
				play.Tasks = []types.Task{{Name: "block", BecomeExpr: "{{ use_sudo }}", BecomeSet: true, Block: []types.Task{task}}}
			}
			result, err := e.executePlayOn(context.Background(), play, hosts)
			require.NoError(t, err)
			require.True(t, result.Success)
			require.Len(t, module.calls, 2)
			got := map[string]bool{}
			for _, call := range module.calls {
				got[call.host] = call.become
				if call.become {
					assert.Equal(t, true, call.argument)
				} else {
					assert.Nil(t, call.argument)
				}
			}
			assert.Equal(t, map[string]bool{"h1": true, "h2": false}, got)
		})
	}
}

func TestBecomeExprBooleanValues(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  bool
	}{
		{"true", true}, {"yes", true}, {"on", true}, {"y", true}, {"1", true}, {" \tTrUe\n", true},
		{"false", false}, {"no", false}, {"off", false}, {"n", false}, {"0", false}, {"", false}, {" \tOFF\n", false},
	} {
		t.Run(fmt.Sprintf("%q", tt.value), func(t *testing.T) {
			e, module := becomeExprEngine(t)
			host := twoHosts()[0]
			task := &types.Task{Name: "probe", Module: "become_probe", BecomeSet: true, BecomeExpr: "{{ use_sudo }}",
				Vars: map[string]interface{}{"use_sudo": tt.value}}
			require.NoError(t, e.executeTaskOnHost(context.Background(), task, &host, nil, &types.PlayResult{}))
			require.Len(t, module.calls, 1)
			assert.Equal(t, tt.want, module.calls[0].become)
		})
	}
}

func TestBecomeExprPrecedence(t *testing.T) {
	for _, tt := range []struct {
		name        string
		task        types.Task
		force, want bool
	}{
		{name: "task false overrides play", task: types.Task{BecomeSet: true}},
		{name: "task expression overrides play", task: types.Task{BecomeSet: true, BecomeExpr: "{{ false }}"}},
		// --become forces the play only: the task's own become still wins
		{name: "task false wins over force", task: types.Task{BecomeSet: true}, force: true},
		{name: "task expression wins over force", task: types.Task{BecomeSet: true, BecomeExpr: "{{ false }}"}, force: true},
		{name: "force without task become", task: types.Task{}, force: true, want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e, module := becomeExprEngine(t)
			e.SetForceBecome(tt.force, "")
			task := tt.task
			task.Name, task.Module = "probe", "become_probe"
			play := &types.Play{Name: "precedence", BecomeExpr: "{{ true }}", Tasks: []types.Task{task}}
			result, err := e.executePlayOn(context.Background(), play, twoHosts()[:1])
			require.NoError(t, err)
			require.True(t, result.Success)
			require.Len(t, module.calls, 1)
			assert.Equal(t, tt.want, module.calls[0].become)
		})
	}
}

func TestBecomeExprInvalidIsFailedTask(t *testing.T) {
	e, module := becomeExprEngine(t)
	host := twoHosts()[0]
	result := &types.PlayResult{Success: true}
	task := &types.Task{Name: "invalid", Module: "become_probe", BecomeSet: true, BecomeExpr: "{{ use_sudo }}"}
	err := e.executeTaskOnHost(context.Background(), task, &host, map[string]interface{}{"use_sudo": "maybe"}, result)
	require.ErrorContains(t, err, `become "{{ use_sudo }}" is not a boolean (rendered "maybe")`)
	require.Empty(t, module.calls)
	require.Len(t, result.Hosts, 1)
	require.Len(t, result.Hosts[0].Tasks, 1)
	assert.True(t, result.Hosts[0].Tasks[0].Failed)
	assert.Contains(t, result.Hosts[0].Tasks[0].Error, `become "{{ use_sudo }}" is not a boolean (rendered "maybe")`)
}

func TestBecomeExprLoopItems(t *testing.T) {
	e, module := becomeExprEngine(t)
	task := &types.Task{Name: "loop", Module: "become_probe", BecomeSet: true, BecomeExpr: "{{ item }}",
		Loop: &types.Loop{Items: []interface{}{true, false}}}
	require.NoError(t, e.executeTask(context.Background(), task, twoHosts()[:1], nil, &types.PlayResult{}))
	require.Len(t, module.calls, 2)
	assert.True(t, module.calls[0].become)
	assert.False(t, module.calls[1].become)
}

func TestBecomeExprDelegateUsesSourceVariables(t *testing.T) {
	e, module := becomeExprEngine(t)
	inv := e.inventoryMgr.(*MockInventoryManager)
	inv.hosts = []types.Host{{Name: "delegate", Address: "192.0.2.20", Vars: map[string]interface{}{"use_sudo": false}}}
	inv.On("GetHosts", mock.Anything).Return(nil, nil)
	host := types.Host{Name: "source", Address: "192.0.2.10", Vars: map[string]interface{}{"use_sudo": true}}
	task := &types.Task{Name: "delegated", Module: "become_probe", DelegateTo: "delegate", BecomeSet: true, BecomeExpr: "{{ use_sudo }}"}
	require.NoError(t, e.executeTaskOnHost(context.Background(), task, &host, nil, &types.PlayResult{}))
	require.Len(t, module.calls, 1)
	assert.Equal(t, "delegate", module.calls[0].host)
	assert.True(t, module.calls[0].become)
}

func TestBecomeExprHandlerInheritsPlay(t *testing.T) {
	e, module := becomeExprEngine(t)
	hosts := twoHosts()[:1]
	hosts[0].Vars = map[string]interface{}{"use_sudo": true}
	defer e.enterPlay(&types.Play{BecomeExpr: "{{ use_sudo }}"})()
	e.startPlayHandlers(hosts, []types.Task{{Name: "restart", Module: "become_probe"}}, nil)
	e.notifyHandlers([]string{"restart"}, hosts[0].Name)
	require.NoError(t, e.flushHandlers(context.Background(), hosts, &types.PlayResult{Success: true}))
	require.Len(t, module.calls, 1)
	assert.True(t, module.calls[0].become)
}

func TestBecomeExprIncludedRoleInheritance(t *testing.T) {
	e, module := becomeExprEngine(t)
	host := twoHosts()[0]
	host.Vars = map[string]interface{}{"use_sudo": true}
	task := &types.Task{Name: "role", BecomeSet: true, BecomeExpr: "{{ use_sudo }}",
		IncludedRole: &types.Role{Name: "probe", Tasks: []types.Task{
			{Name: "inherited", Module: "become_probe"},
			{Name: "overridden", Module: "become_probe", BecomeSet: true},
		}}}
	require.NoError(t, e.executeTask(context.Background(), task, []types.Host{host}, nil, &types.PlayResult{}))
	require.Len(t, module.calls, 2)
	assert.Equal(t, "inherited", module.calls[0].task)
	assert.True(t, module.calls[0].become)
	assert.Equal(t, "overridden", module.calls[1].task)
	assert.False(t, module.calls[1].become)
}

func TestBecomeExprIncludedRoleDependencyInheritance(t *testing.T) {
	e, module := becomeExprEngine(t)
	host := twoHosts()[0]
	host.Vars = map[string]interface{}{"use_sudo": true}
	dependency := &types.Role{Name: "dependency", Tasks: []types.Task{
		{Name: "dependency inherited", Module: "become_probe"},
		{Name: "dependency overridden", Module: "become_probe", BecomeSet: true},
	}}
	e.roleLoader.GetCache()[dependency.Name] = dependency
	role := &types.Role{Name: "main", Meta: types.RoleMeta{Dependencies: []types.RoleDependency{{Name: dependency.Name}}},
		Tasks: []types.Task{{Name: "main inherited", Module: "become_probe"}}}
	task := &types.Task{Name: "include", BecomeSet: true, BecomeExpr: "{{ use_sudo }}", IncludedRole: role}
	require.NoError(t, e.executeTask(context.Background(), task, []types.Host{host}, nil, &types.PlayResult{}))
	require.Len(t, module.calls, 3)
	assert.Equal(t, "dependency inherited", module.calls[0].task)
	assert.True(t, module.calls[0].become)
	assert.Equal(t, "dependency overridden", module.calls[1].task)
	assert.False(t, module.calls[1].become)
	assert.Equal(t, "main inherited", module.calls[2].task)
	assert.True(t, module.calls[2].become)
	for _, original := range []*types.Task{&dependency.Tasks[0], &role.Tasks[0]} {
		assert.False(t, original.BecomeSet)
		assert.Empty(t, original.BecomeExpr)
	}
}

func TestBecomeExprDisablesLoopPrefetch(t *testing.T) {
	for _, level := range []string{"task", "play"} {
		t.Run(level, func(t *testing.T) {
			e, module := becomeExprEngine(t)
			renderer := new(MockTemplateEngine)
			e.templateEngine = renderer
			task := &types.Task{Name: "file", Module: "file", Args: map[string]interface{}{"dest": "{{ item }}"}}
			if level == "task" {
				task.BecomeSet, task.BecomeExpr = true, "{{ item }}"
			} else {
				defer e.enterPlay(&types.Play{BecomeExpr: "{{ item }}"})()
			}
			host := twoHosts()[0]
			probes := e.prefetchLoop(context.Background(), task, &host, nil, []interface{}{true, false}, "item", "index")
			assert.Nil(t, probes)
			assert.Empty(t, module.calls)
			renderer.AssertNotCalled(t, "Render", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestBecomeExprRenderErrorIsFailedTask(t *testing.T) {
	e, module := becomeExprEngine(t)
	host := twoHosts()[0]
	result := &types.PlayResult{Success: true}
	task := &types.Task{Name: "invalid", Module: "become_probe", BecomeSet: true, BecomeExpr: "{{ use_sudo | missing_filter }}"}
	err := e.executeTaskOnHost(context.Background(), task, &host, map[string]interface{}{"use_sudo": true}, result)
	require.ErrorContains(t, err, `become "{{ use_sudo | missing_filter }}":`)
	require.Empty(t, module.calls)
	require.Len(t, result.Hosts, 1)
	require.Len(t, result.Hosts[0].Tasks, 1)
	assert.True(t, result.Hosts[0].Tasks[0].Failed)
	assert.Contains(t, result.Hosts[0].Tasks[0].Error, "missing_filter")
}

func TestBecomeForceKeepsTaskFalse(t *testing.T) {
	// --become forces the play's become, a templated one too; a task's own
	// become: false still wins, as before templated become existed
	e, module := becomeExprEngine(t)
	e.SetForceBecome(true, "admin")
	play := &types.Play{Name: "forced", BecomeExpr: "{{ false }}", Tasks: []types.Task{
		{Name: "probe", Module: "become_probe"},
		{Name: "opt out", Module: "become_probe", BecomeSet: true},
	}}
	result, err := e.executePlayOn(context.Background(), play, twoHosts()[:1])
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Len(t, module.calls, 2)
	assert.True(t, module.calls[0].become)
	assert.Equal(t, "admin", module.calls[0].user)
	assert.False(t, module.calls[1].become)
}

func TestInheritedBecomeKeepsTaskUser(t *testing.T) {
	e, module := becomeExprEngine(t)
	play := &types.Play{Name: "block", Tasks: []types.Task{{Name: "block", Become: true, BecomeSet: true,
		Block: []types.Task{{Name: "probe", Module: "become_probe", BecomeUser: "postgres"}}}}}
	result, err := e.executePlayOn(context.Background(), play, twoHosts()[:1])
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Len(t, module.calls, 1)
	assert.True(t, module.calls[0].become)
	assert.Equal(t, "postgres", module.calls[0].user)
}
