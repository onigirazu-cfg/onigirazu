package engine

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// perHostEngine returns an engine whose module calls are recorded per host
func perHostEngine(t *testing.T, result types.TaskResult) (*ExecutionEngine, *MockModuleRegistry, *[]map[string]interface{}) {
	t.Helper()
	engine, mockConfig, _, _, mockRegistry, mockTemplate := createTestEngine()
	mockConfig.On("GetDryRun").Return(false)
	mockTemplate.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	var mu sync.Mutex
	calls := []map[string]interface{}{}
	mockRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			mu.Lock()
			defer mu.Unlock()
			host := args.Get(2).(types.Host)
			vars := args.Get(3).(map[string]interface{})
			calls = append(calls, map[string]interface{}{"host": host.Name, "item": vars["item"], "loop": vars["loop"]})
		}).
		Return(result, nil)
	return engine, mockRegistry, &calls
}

func twoHosts() []types.Host {
	return []types.Host{{Name: "h1", Address: "10.0.0.1"}, {Name: "h2", Address: "10.0.0.2"}}
}

func TestLoop_ItemsPerHost(t *testing.T) {
	engine, _, calls := perHostEngine(t, types.TaskResult{Success: true, Changed: true, Output: map[string]interface{}{}})
	engine.setHostVar("h1", "names", []interface{}{"a", "b"})
	engine.setHostVar("h2", "names", []interface{}{"c"})

	task := &types.Task{
		Name: "loop", Module: "debug", Register: "out",
		Loop: &types.Loop{Expr: "{{ names }}"},
	}
	err := engine.executeTask(context.Background(), task, twoHosts(), map[string]interface{}{}, &types.PlayResult{})
	require.NoError(t, err)

	items := map[string][]interface{}{}
	for _, c := range *calls {
		items[c["host"].(string)] = append(items[c["host"].(string)], c["item"])
	}
	assert.Equal(t, []interface{}{"a", "b"}, items["h1"])
	assert.Equal(t, []interface{}{"c"}, items["h2"])

	out, ok := engine.getHostVar("h1", "out").(map[string]interface{})
	require.True(t, ok)
	assert.Len(t, out["results"], 2)
	assert.Equal(t, true, out["changed"])
	last := (*calls)[len(*calls)-1]["loop"].(map[string]interface{})
	assert.Equal(t, true, last["last"])
}

func TestLoop_ExpressionMustBeList(t *testing.T) {
	engine, registry, _ := perHostEngine(t, types.TaskResult{Success: true})
	engine.setHostVar("h1", "name", "not a list")
	task := &types.Task{Name: "loop", Module: "debug", Loop: &types.Loop{Expr: "name"}}
	err := engine.executeTask(context.Background(), task, twoHosts()[:1], map[string]interface{}{}, &types.PlayResult{})
	assert.Error(t, err)
	registry.AssertNotCalled(t, "ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestWhen_EvaluatedPerHost(t *testing.T) {
	engine, _, calls := perHostEngine(t, types.TaskResult{Success: true})
	engine.facts["h1"] = map[string]interface{}{"onigirazu_os_family": "Debian"}
	engine.facts["h2"] = map[string]interface{}{"onigirazu_os_family": "RedHat"}

	task := &types.Task{Name: "only redhat", Module: "debug", When: `onigirazu_os_family == "RedHat"`}
	playResult := &types.PlayResult{}
	err := engine.executeTask(context.Background(), task, twoHosts(), map[string]interface{}{}, playResult)
	require.NoError(t, err)

	require.Len(t, *calls, 1)
	assert.Equal(t, "h2", (*calls)[0]["host"])
	for _, h := range playResult.Hosts {
		if h.Host == "h1" {
			require.Len(t, h.Tasks, 1)
			assert.True(t, h.Tasks[0].Skipped)
		}
	}
}

func TestFailedWhenAndChangedWhen(t *testing.T) {
	engine, _, _ := perHostEngine(t, types.TaskResult{
		Success: false, Failed: true, Changed: true, Error: "exit 1",
		Output: map[string]interface{}{"rc": 1},
	})
	task := &types.Task{
		Name: "probe", Module: "command", Register: "r",
		FailedWhen: "r.rc > 1", ChangedWhen: "false",
	}
	host := twoHosts()[0]
	err := engine.executeTaskOnHost(context.Background(), task, &host, map[string]interface{}{}, &types.PlayResult{})
	require.NoError(t, err)

	r := engine.getHostVar("h1", "r").(map[string]interface{})
	assert.Equal(t, false, r["failed"])
	assert.Equal(t, false, r["changed"])

	task.FailedWhen = "r.rc == 1"
	err = engine.executeTaskOnHost(context.Background(), task, &host, map[string]interface{}{}, &types.PlayResult{})
	assert.Error(t, err)
}

func TestUntil_RetriesUntilConditionHolds(t *testing.T) {
	engine, mockConfig, mockLogger, _, mockRegistry, mockTemplate := createTestEngine()
	mockConfig.On("GetDryRun").Return(false)
	mockLogger.On("Retry", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	mockTemplate.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	for n := 1; n <= 3; n++ {
		mockRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(types.TaskResult{Success: true, Output: map[string]interface{}{"n": n}}, nil).Once()
	}

	task := &types.Task{
		Name: "wait", Module: "command", Register: "r",
		Until: "r.n >= 3", Retries: 5, RetryDelay: time.Millisecond,
	}
	host := twoHosts()[0]
	err := engine.executeTaskOnHost(context.Background(), task, &host, map[string]interface{}{}, &types.PlayResult{})
	require.NoError(t, err)
	mockRegistry.AssertNumberOfCalls(t, "ExecuteTask", 3)
	assert.Equal(t, 3, engine.getHostVar("h1", "r").(map[string]interface{})["n"])
}

func TestRegisteredValue_Lines(t *testing.T) {
	v := registeredValue(types.TaskResult{Output: map[string]interface{}{"stdout": "a\nb\n", "stderr": ""}})
	assert.Equal(t, []interface{}{"a", "b"}, v["stdout_lines"])
	assert.Equal(t, []interface{}{}, v["stderr_lines"])
}

func TestIgnoredFailure_DoesNotFailPlay(t *testing.T) {
	engine, _, _ := perHostEngine(t, types.TaskResult{Success: false, Failed: true, Error: "exit 1"})
	task := &types.Task{Name: "may fail", Module: "command", IgnoreErrors: true}
	host := twoHosts()[0]
	play := &types.PlayResult{Success: true}
	require.NoError(t, engine.executeTaskOnHost(context.Background(), task, &host, map[string]interface{}{}, play))
	assert.True(t, play.Success)
	require.Len(t, play.Hosts, 1)
	assert.False(t, play.Hosts[0].Failed)
}

func TestNotify_OnlyWhenChanged(t *testing.T) {
	for _, changed := range []bool{false, true} {
		engine, _, _ := perHostEngine(t, types.TaskResult{Success: true, Changed: changed})
		task := &types.Task{Name: "conf", Module: "copy", Notify: []string{"restart"}}
		host := twoHosts()[0]
		play := &types.PlayResult{Success: true}
		engine.startPlayHandlers(twoHosts(), []types.Task{{Name: "restart", Module: "service"}}, nil)
		require.NoError(t, engine.executeTaskOnHost(context.Background(), task, &host, map[string]interface{}{}, play))
		toRun := engine.handlers.takeNotified(twoHosts())
		if changed {
			assert.Equal(t, []types.Host{host}, toRun[0])
		} else {
			assert.Empty(t, toRun[0])
		}
	}
}

func TestRunOnce_FirstHostOnlyAndRegisterShared(t *testing.T) {
	engine, _, calls := perHostEngine(t, types.TaskResult{Success: true, Output: map[string]interface{}{"stdout": "v"}})
	task := &types.Task{Name: "once", Module: "command", RunOnce: true, Register: "r"}
	require.NoError(t, engine.executeTask(context.Background(), task, twoHosts(), map[string]interface{}{}, &types.PlayResult{}))
	require.Len(t, *calls, 1)
	assert.Equal(t, "h1", (*calls)[0]["host"])
	assert.Equal(t, "v", engine.getHostVar("h2", "r").(map[string]interface{})["stdout"])
}

func TestDelegateTo_RunsOnDelegateWithOwnVariables(t *testing.T) {
	engine, mockConfig, _, mockInventory, mockRegistry, mockTemplate := createTestEngine()
	mockConfig.On("GetDryRun").Return(false)
	mockTemplate.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	mockTemplate.On("Render", mock.Anything, "{{ target }}", mock.Anything).Return("h2", nil)
	mockInventory.hosts = twoHosts()
	mockInventory.On("GetHosts", mock.Anything).Return(nil, nil)
	var ranOn string
	var sawHostname interface{}
	mockRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			ranOn = args.Get(2).(types.Host).Name
			sawHostname = args.Get(3).(map[string]interface{})["inventory_hostname"]
		}).
		Return(types.TaskResult{Success: true}, nil)

	task := &types.Task{Name: "d", Module: "command", DelegateTo: "{{ target }}"}
	host := twoHosts()[0]
	play := &types.PlayResult{}
	require.NoError(t, engine.executeTaskOnHost(context.Background(), task, &host, map[string]interface{}{"target": "h2"}, play))
	assert.Equal(t, "h2", ranOn, "the module runs on the delegate")
	assert.Equal(t, "h1", sawHostname, "with the variables of the original host")
	assert.Equal(t, "h1", play.Hosts[0].Host, "and the result belongs to the original host")
}

func blockEngine(t *testing.T) (*ExecutionEngine, *[]string) {
	t.Helper()
	engine, mockConfig, _, _, mockRegistry, mockTemplate := createTestEngine()
	mockConfig.On("GetDryRun").Return(false)
	mockTemplate.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	var mu sync.Mutex
	ran := []string{}
	record := func(args mock.Arguments) {
		mu.Lock()
		defer mu.Unlock()
		ran = append(ran, args.Get(1).(*types.Task).Name)
	}
	isFail := func(task *types.Task) bool { return task.Name == "fail" }
	mockRegistry.On("ExecuteTask", mock.Anything, mock.MatchedBy(isFail), mock.Anything, mock.Anything).
		Run(record).Return(types.TaskResult{Success: false, Failed: true, Error: "boom"}, nil)
	mockRegistry.On("ExecuteTask", mock.Anything, mock.MatchedBy(func(task *types.Task) bool { return !isFail(task) }), mock.Anything, mock.Anything).
		Run(record).Return(types.TaskResult{Success: true}, nil)
	return engine, &ran
}

func TestBlock_RescueHandlesFailure(t *testing.T) {
	engine, ran := blockEngine(t)
	block := types.Task{
		Name:   "b",
		Block:  []types.Task{{Name: "fail", Module: "command"}, {Name: "skipped", Module: "command"}},
		Rescue: []types.Task{{Name: "rescue", Module: "command"}},
		Always: []types.Task{{Name: "always", Module: "command"}},
	}
	play := &types.PlayResult{Success: true}
	require.NoError(t, engine.executeTaskList(context.Background(), []types.Task{block}, twoHosts()[:1], map[string]interface{}{}, play))
	assert.Equal(t, []string{"fail", "rescue", "always"}, *ran)
	assert.True(t, play.Success, "a rescued failure does not fail the play")
}

func TestBlock_WithoutRescueFailsAfterAlways(t *testing.T) {
	engine, ran := blockEngine(t)
	block := types.Task{
		Name:   "b",
		Block:  []types.Task{{Name: "fail", Module: "command"}},
		Always: []types.Task{{Name: "always", Module: "command"}},
	}
	play := &types.PlayResult{Success: true}
	err := engine.executeTaskList(context.Background(), []types.Task{block}, twoHosts()[:1], map[string]interface{}{}, play)
	assert.Error(t, err)
	assert.Equal(t, []string{"fail", "always"}, *ran)
	assert.False(t, play.Success)
}

func TestExecutePlaybook_FailedPlayKeepsItsResults(t *testing.T) {
	engine, mockConfig, _, mockInventory, mockRegistry, mockTemplate := createTestEngine()
	mockConfig.On("GetDryRun").Return(false)
	mockTemplate.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	hosts := twoHosts()[:1]
	mockInventory.hosts = hosts
	mockInventory.On("GetHosts", "all").Return(hosts, nil)
	mockRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(types.TaskResult{Success: false, Failed: true, Error: "boom"}, nil)

	result, _ := engine.ExecutePlaybook(context.Background(), &types.Playbook{Plays: []types.Play{{
		Name: "p", Hosts: "all", Tasks: []types.Task{{Name: "fails", Module: "command"}},
	}}})
	require.NotNil(t, result)
	assert.True(t, result.Failed)
	require.Len(t, result.Plays, 1, "the failed play is part of the result")
	require.Len(t, result.Plays[0].Hosts, 1)
	assert.True(t, result.Plays[0].Hosts[0].Tasks[0].Failed)
}

func TestLimit_NarrowsPlayHosts(t *testing.T) {
	engine, _, _, mockInventory, _, _ := createTestEngine()
	all := []types.Host{{Name: "web1"}, {Name: "web2"}, {Name: "db1"}}
	mockInventory.On("GetHosts", "all").Return(all, nil)
	mockInventory.On("GetHosts", "web2,db1").Return([]types.Host{{Name: "web2"}, {Name: "db1"}}, nil)

	engine.SetLimit("web2,db1")
	hosts, err := engine.getPlayHosts(&types.Play{Hosts: "all"})
	require.NoError(t, err)
	var names []string
	for _, h := range hosts {
		names = append(names, h.Name)
	}
	assert.Equal(t, []string{"web2", "db1"}, names)
}

func TestExtraVars_OverrideEverything(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()
	engine.SetExtraVars(map[string]interface{}{"port": 9090})
	host := &types.Host{Name: "h1", Vars: map[string]interface{}{"port": 22}}
	engine.setHostVar("h1", "port", 1234) // set_fact
	vars := engine.hostVariables(host, map[string]interface{}{"port": 80})
	assert.Equal(t, 9090, vars["port"])
}

func TestExtraVars_OverrideTaskVars(t *testing.T) {
	engine, mockConfig, _, _, mockRegistry, mockTemplate := createTestEngine()
	mockConfig.On("GetDryRun").Return(false)
	mockTemplate.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	var seen interface{}
	mockRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { seen = args.Get(3).(map[string]interface{})["port"] }).
		Return(types.TaskResult{Success: true}, nil)
	engine.SetExtraVars(map[string]interface{}{"port": 9090})
	host := twoHosts()[0]
	task := &types.Task{Name: "t", Module: "debug", Vars: map[string]interface{}{"port": 1}}
	require.NoError(t, engine.executeTaskOnHost(context.Background(), task, &host, map[string]interface{}{}, &types.PlayResult{}))
	assert.Equal(t, 9090, seen)
}

func TestRescue_SeesTheFailedTask(t *testing.T) {
	engine, _ := blockEngine(t)
	block := types.Task{
		Name:   "b",
		Block:  []types.Task{{Name: "fail", Module: "command"}},
		Rescue: []types.Task{{Name: "rescue", Module: "command"}},
	}
	require.NoError(t, engine.executeTaskList(context.Background(), []types.Task{block}, twoHosts()[:1], map[string]interface{}{}, &types.PlayResult{}))
	task, ok := engine.getHostVar("h1", "ansible_failed_task").(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "fail", task["name"])
	res, ok := engine.getHostVar("h1", "ansible_failed_result").(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "boom", res["msg"])
	assert.NotNil(t, engine.getHostVar("h1", "onigirazu_failed_task"))
}

func TestStartAtTask_SkipsUntilTheTask(t *testing.T) {
	engine, _, calls := perHostEngine(t, types.TaskResult{Success: true})
	engine.SetStartAtTask("two")
	tasks := []types.Task{
		{Name: "one", Module: "debug"},
		{Name: "outer", Block: []types.Task{{Name: "skipped in block", Module: "debug"}}},
		{Name: "wrapper", Block: []types.Task{{Name: "before", Module: "debug"}, {Name: "two", Module: "debug"}, {Name: "three", Module: "debug"}}},
		{Name: "four", Module: "debug"},
	}
	require.NoError(t, engine.executeTaskList(context.Background(), tasks, twoHosts()[:1], map[string]interface{}{}, &types.PlayResult{}))
	assert.Len(t, *calls, 3, "two, three and four ran")
	assert.True(t, engine.StartAtTaskFound())
}

func TestForceBecome_AppliesToEveryPlay(t *testing.T) {
	engine, mockConfig, _, mockInventory, mockRegistry, mockTemplate := createTestEngine()
	mockConfig.On("GetDryRun").Return(false)
	mockTemplate.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	hosts := twoHosts()[:1]
	mockInventory.hosts = hosts
	mockInventory.On("GetHosts", "all").Return(hosts, nil)
	var became bool
	mockRegistry.On("ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { became = args.Get(1).(*types.Task).Become }).
		Return(types.TaskResult{Success: true}, nil)
	engine.SetForceBecome(true, "")
	_, err := engine.ExecutePlaybook(context.Background(), &types.Playbook{Plays: []types.Play{{
		Name: "p", Hosts: "all", Tasks: []types.Task{{Name: "t", Module: "debug"}},
	}}})
	require.NoError(t, err)
	assert.True(t, became)
}

func TestRunOnce_InsideBlockRunsOnceForAllHosts(t *testing.T) {
	engine, _, calls := perHostEngine(t, types.TaskResult{Success: true, Output: map[string]interface{}{"stdout": "v"}})
	block := types.Task{Name: "b", Block: []types.Task{
		{Name: "once", Module: "command", RunOnce: true, Register: "r"},
		{Name: "every host", Module: "debug"},
	}}
	require.NoError(t, engine.executeTaskList(context.Background(), []types.Task{block}, twoHosts(), map[string]interface{}{}, &types.PlayResult{}))
	assert.Len(t, *calls, 3, "once + every host twice")
	for _, h := range []string{"h1", "h2"} {
		r, ok := engine.getHostVar(h, "r").(map[string]interface{})
		require.True(t, ok, h)
		assert.Equal(t, "v", r["stdout"])
	}
}

func TestRenderError_IsAFailedTaskResult(t *testing.T) {
	for _, ignore := range []bool{false, true} {
		engine, mockConfig, _, _, mockRegistry, mockTemplate := createTestEngine()
		mockConfig.On("GetDryRun").Return(false)
		mockTemplate.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, fmt.Errorf("undefined variable: nope"))
		playResult := &types.PlayResult{}
		task := &types.Task{Name: "bad", Module: "debug", IgnoreErrors: ignore}
		err := engine.executeTask(context.Background(), task, twoHosts()[:1], map[string]interface{}{}, playResult)
		if ignore {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
		mockRegistry.AssertNotCalled(t, "ExecuteTask", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		require.Len(t, playResult.Hosts, 1)
		require.Len(t, playResult.Hosts[0].Tasks, 1)
		got := playResult.Hosts[0].Tasks[0]
		assert.True(t, got.Failed)
		assert.Contains(t, got.Error, "undefined variable")
	}
}

func TestMagicVariables_HostvarsAndGroups(t *testing.T) {
	engine, mockConfig, _, mockInventory, _, mockTemplate := createTestEngine()
	mockConfig.On("GetDryRun").Return(false)
	mockTemplate.On("RenderTaskArgs", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	hosts := []types.Host{
		{Name: "h1", Address: "10.0.0.1", Vars: map[string]interface{}{"port": 80}},
		{Name: "h2", Address: "10.0.0.2", Vars: map[string]interface{}{"port": 81}},
	}
	mockInventory.On("GetHosts", "all").Return(hosts, nil)
	mockInventory.On("GetHosts", "web").Return(hosts[1:], nil)
	mockInventory.On("ListGroups").Return([]string{"web"})
	engine.setHostVar("h2", "token", "abc")

	vars := engine.withMagicVariables(map[string]interface{}{"hostvars": "stale"})
	again := engine.withMagicVariables(vars)
	for _, v := range []map[string]interface{}{vars, again} {
		hv := v["hostvars"].(map[string]interface{})
		assert.Equal(t, 81, hv["h2"].(map[string]interface{})["port"])
		assert.Equal(t, "abc", hv["h2"].(map[string]interface{})["token"])
		_, nested := hv["h1"].(map[string]interface{})["hostvars"]
		assert.False(t, nested, "no hostvars inside hostvars")
		groups := v["groups"].(map[string]interface{})
		assert.Equal(t, []interface{}{"h1", "h2"}, groups["all"])
		assert.Equal(t, []interface{}{"h2"}, groups["web"])
	}
}

func TestHandlers_OnlyOnNotifyingHostsOnce(t *testing.T) {
	engine, _, calls := perHostEngine(t, types.TaskResult{Success: true, Changed: true})
	hosts := twoHosts()
	engine.startPlayHandlers(hosts, []types.Task{
		{Name: "restart app", Module: "service", Listen: "app changed"},
		{Name: "other", Module: "service"},
	}, map[string]interface{}{})
	engine.notifyHandlers([]string{"app changed"}, "h2")
	play := &types.PlayResult{Success: true}

	require.NoError(t, engine.flushHandlers(context.Background(), hosts, play))
	require.Len(t, *calls, 1)
	assert.Equal(t, "h2", (*calls)[0]["host"])

	require.NoError(t, engine.flushHandlers(context.Background(), hosts, play))
	assert.Len(t, *calls, 1, "a flushed notification does not run again")
}

func TestMeta_EndHostStopsTheHost(t *testing.T) {
	engine, _, calls := perHostEngine(t, types.TaskResult{Success: true})
	hosts := twoHosts()
	engine.startPlayHandlers(hosts, nil, nil)
	play := &types.PlayResult{Success: true}
	end := &types.Task{Name: "stop", Module: "meta", Args: map[string]interface{}{"free_form": "end_host"}, When: "inventory_hostname == 'h1'"}
	require.NoError(t, engine.executeTask(context.Background(), end, hosts, map[string]interface{}{}, play))
	require.NoError(t, engine.executeTask(context.Background(), &types.Task{Name: "after", Module: "debug"}, hosts, map[string]interface{}{}, play))
	require.Len(t, *calls, 1)
	assert.Equal(t, "h2", (*calls)[0]["host"])
}

func TestBecomeFalseAndBlockInheritance(t *testing.T) {
	play := becomeSettings{Become: true, User: "root"}
	assert.False(t, effectiveBecome(&types.Task{BecomeSet: true}, play).Become, "become: false wins")
	assert.True(t, effectiveBecome(&types.Task{}, play).Become)

	block := &types.Task{BecomeSet: true, Environment: map[string]interface{}{"A": "1", "B": "1"}, Vars: map[string]interface{}{"x": 1}}
	out := inheritBlock(block, []types.Task{
		{Name: "t", Environment: map[string]interface{}{"B": "2"}},
		{Name: "own", Become: true, BecomeSet: true},
	}, false)
	assert.True(t, out[0].BecomeSet && !out[0].Become)
	assert.Equal(t, map[string]interface{}{"A": "1", "B": "2"}, out[0].Environment)
	assert.Equal(t, 1, out[0].Vars["x"])
	assert.True(t, out[1].Become, "a task's own become stays")
}

func TestRoleVariablePrecedence(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()
	engine.SetExtraVars(map[string]interface{}{"e": "extra"})
	role := &types.Role{
		Defaults: map[string]interface{}{"d": "default", "p": "default", "v": "default", "e": "default"},
		Vars:     map[string]interface{}{"v": "role-vars", "p": "role-vars"},
		Params:   map[string]interface{}{"p": "param"},
	}
	vars := engine.mergeRoleVariables(role, map[string]interface{}{"d": "play"})
	assert.Equal(t, "play", vars["d"], "play vars over defaults")
	assert.Equal(t, "role-vars", vars["v"])
	assert.Equal(t, "param", vars["p"], "role params over role vars")
	assert.Equal(t, "extra", vars["e"], "-e wins")
}
