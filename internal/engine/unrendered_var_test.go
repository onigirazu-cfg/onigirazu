package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// an argument that still carries a variable's raw template fails the task
// with that variable's error; other arguments pass
func TestUnrenderedUsed(t *testing.T) {
	bad := map[string]unrenderedVar{"image": {text: "{{ (registry or 'x') }}/img", err: errors.New("interface conversion: string, not bool")}}
	name, err := unrenderedUsed(map[string]interface{}{"cmd": "docker pull {{ (registry or 'x') }}/img"}, bad)
	assert.Equal(t, "image", name)
	assert.ErrorContains(t, err, "not bool")

	name, err = unrenderedUsed(map[string]interface{}{"cmd": "echo fine", "env": map[string]interface{}{"A": []interface{}{"{{ (registry or 'x') }}/img"}}}, bad)
	assert.Equal(t, "image", name)
	assert.Error(t, err)

	_, err = unrenderedUsed(map[string]interface{}{"cmd": "echo fine"}, bad)
	assert.NoError(t, err)
	_, err = unrenderedUsed(map[string]interface{}{"cmd": "{{ (registry or 'x') }}"}, nil)
	assert.NoError(t, err)
}

func TestUnrenderedEvaluatedKeywords(t *testing.T) {
	for _, keyword := range []string{"when", "length", "changed_when", "failed_when", "until", "loop", "with_items", "with_lookup", "range", "args_filter", "environment", "delegate_to", "retries", "delay", "become", "become_user", "become_method", "task_var", "ignore_errors", "register"} {
		t.Run(keyword, func(t *testing.T) {
			e, module := becomeExprEngine(t)
			host := twoHosts()[0]
			vars := map[string]interface{}{"broken": "{{ .Names }}"}
			task := &types.Task{Name: "probe", Module: "become_probe"}
			switch keyword {
			case "when":
				task.When = "broken + 1 > 0"
			case "length":
				task.When = "broken | length > 0"
			case "changed_when":
				task.ChangedWhen = "broken | length > 0"
			case "failed_when":
				task.FailedWhen = "broken | length == 0"
			case "until":
				task.Until = "broken | length > 0"
			case "loop":
				task.Loop = &types.Loop{Expr: "[broken | length]"}
			case "with_items":
				task.Loop = &types.Loop{Items: []interface{}{"{{ broken | length }}"}}
			case "range":
				task.Loop = &types.Loop{Range: "1-{{ broken | length }}"}
			case "with_lookup":
				task.Loop = &types.Loop{Lookup: "nested", Terms: []interface{}{"{{ [broken | length] }}"}}
			case "environment":
				task.Environment = map[string]interface{}{"VALUE": "{{ broken | length }}"}
			case "delegate_to":
				task.DelegateTo = "{{ broken | length }}"
			case "retries":
				task.RetriesExpr = "{{ broken | length }}"
			case "delay":
				task.DelayExpr = "{{ broken | length }}"
			case "become_user":
				task.Become, task.BecomeUser = true, "{{ broken | length }}"
			case "become_method":
				task.Become, task.BecomeMethod = true, "{{ broken | length }}"
			case "task_var":
				task.Vars = map[string]interface{}{"value": "{{ broken | length }}"}
			case "ignore_errors":
				task.FailedWhen = "true"
				task.IgnoreErrorsExpr = "broken | length > 0"
			case "become":
				task.BecomeExpr, task.BecomeSet = "{{ broken | length > 0 }}", true
			case "register":
				task.RegisterVars = map[string]string{"out": "broken | length"}
			case "args_filter":
				task.Args = map[string]interface{}{"value": "{{ broken | length }}"}
			}
			result := &types.PlayResult{}
			err := e.executeTask(context.Background(), task, []types.Host{host}, vars, result)
			require.ErrorContains(t, err, `variable "broken":`)
			require.ErrorContains(t, err, "Names")
			// An unrelated task can still execute with the same bad play variable.
			before := len(module.calls)
			require.NoError(t, e.executeTaskOnHost(context.Background(), &types.Task{Name: "unrelated", Module: "become_probe"}, &host, vars, &types.PlayResult{}))
			require.Len(t, module.calls, before+1)
		})
	}
}

func TestUnrenderedSafeConditions(t *testing.T) {
	for _, condition := range []string{"false and broken | length > 0", "true or broken | length > 0", "broken is defined", "broken is undefined", "broken | default('fallback')", "'broken' | length > 0"} {
		t.Run(condition, func(t *testing.T) {
			e, _ := becomeExprEngine(t)
			host := twoHosts()[0]
			task := &types.Task{Name: "safe", Module: "become_probe", When: condition}
			result := &types.PlayResult{}
			require.NoError(t, e.executeTaskOnHost(context.Background(), task, &host, map[string]interface{}{"broken": "{{ .Names }}"}, result))
			require.Equal(t, condition == "broken is defined" || condition == "false and broken | length > 0", result.Hosts[0].Tasks[0].Skipped)
		})
	}
}

func TestUnrenderedReferenceDetection(t *testing.T) {
	bad := map[string]unrenderedVar{"broken": {text: "{{ .Names }}", err: errors.New("render failed")}}
	ctx := context.WithValue(context.Background(), unrenderedKey{}, bad)
	vars := map[string]interface{}{}
	for _, code := range []string{"broken", "not broken", "broken | length > 0", "broken is defined and broken", "broken | default(broken)", "[broken]", "broken.attr"} {
		require.ErrorContains(t, checkUnrenderedExpression(ctx, code, vars), `variable "broken": render failed`, code)
	}
	for _, code := range []string{"unbroken", "'broken'", `"escaped \" broken"`, "obj.broken", "obj | broken", "obj is broken", "broken is defined", "broken is undefined", "broken is not defined", "broken | default([])", "broken | ansible.builtin.default([])", "broken | d([])"} {
		require.NoError(t, checkUnrenderedExpression(ctx, code, vars), code)
	}
	require.NoError(t, checkUnrenderedExpression(ctx, "broken", map[string]interface{}{"broken": true}), "overridden values are usable")
	require.NoError(t, checkUnrenderedTemplates(ctx, "literal broken", vars))
}

func TestUnrenderedLaterPass(t *testing.T) {
	e, module := becomeExprEngine(t)
	host := twoHosts()[0]
	vars := map[string]interface{}{
		"answer": "{{ config.port }}", "config": "{{ source }}", "source": map[string]interface{}{"port": 22},
	}
	task := &types.Task{Name: "resolved", Module: "become_probe", When: "answer | int == 22"}
	require.NoError(t, e.executeTaskOnHost(context.Background(), task, &host, vars, &types.PlayResult{}))
	require.Len(t, module.calls, 1)
}

func TestUnrenderedIncludeCondition(t *testing.T) {
	e, _ := becomeExprEngine(t)
	task := &types.Task{Name: "include", Module: "include_role", When: "broken | length > 0", IncludedRole: &types.Role{Name: "sensor"}}
	result := &types.PlayResult{}
	err := e.executeTask(context.Background(), task, twoHosts()[:1], map[string]interface{}{"broken": "{{ .Names }}"}, result)
	require.ErrorContains(t, err, `variable "broken":`)
	require.True(t, result.Hosts[0].Tasks[0].Failed)
}

func TestUnrenderedThrottle(t *testing.T) {
	e, _ := becomeExprEngine(t)
	_, err := e.throttleSlots(context.Background(), &types.Task{Throttle: "{{ broken | length }}"}, map[string]interface{}{"broken": "{{ .Names }}"})
	require.ErrorContains(t, err, `variable "broken":`)
}

func TestUnrenderedDefaultArgument(t *testing.T) {
	e, _ := becomeExprEngine(t)
	host := twoHosts()[0]
	e.moduleRegistry = modules.NewRegistry()
	vars := map[string]interface{}{"broken": "{{ missing }}"}
	result := &types.PlayResult{}
	task := &types.Task{Name: "default", Module: "debug", Args: map[string]interface{}{"msg": "{{ broken | default('d') }}"}}
	require.NoError(t, e.executeTaskOnHost(context.Background(), task, &host, vars, result))
	require.Equal(t, "d", result.Hosts[0].Tasks[0].Output["msg"])
	require.Equal(t, "{{ missing }}", vars["broken"], "the caller's map remains intact")
}

func TestBreakWhenItemLazyVars(t *testing.T) {
	for _, stop := range []string{"{{ item == 2 }}", "{{ r is success and item == 2 }}"} {
		t.Run(stop, func(t *testing.T) {
			e, module := becomeExprEngine(t)
			host := twoHosts()[0]
			task := &types.Task{Name: "probe", Module: "become_probe", Register: "r", BreakWhen: "stop",
				Vars: map[string]interface{}{"stop": stop}, Loop: &types.Loop{Items: []interface{}{1, 2, 3}}}
			require.NoError(t, e.runLoopOnHost(context.Background(), task, &host, nil, &types.PlayResult{}))
			require.Len(t, module.calls, 2)
			result := e.getHostVar(host.Name, "r").(map[string]interface{})
			require.Len(t, result["results"], 2)
		})
	}
}

func TestBreakWhenUnrenderedError(t *testing.T) {
	e, _ := becomeExprEngine(t)
	host := twoHosts()[0]
	task := &types.Task{Name: "probe", Module: "become_probe", Register: "r", BreakWhen: "broken | length > 0", Loop: &types.Loop{Items: []interface{}{1}}}
	require.NoError(t, e.runLoopOnHost(context.Background(), task, &host, map[string]interface{}{"broken": "{{ .Names }}"}, &types.PlayResult{}))
	result := e.getHostVar(host.Name, "r").(map[string]interface{})["results"].([]interface{})[0].(map[string]interface{})
	require.Contains(t, result["break_when_suppressed_exception"], `variable "broken":`)
}

func TestLazyContextRemovesFailedCopies(t *testing.T) {
	e, _ := becomeExprEngine(t)
	vars := map[string]interface{}{"broken": "{{ missing }}", "nested": map[string]interface{}{"value": "{{ missing }}"}}
	ctx := e.lazyEvaluationContext(context.Background(), "", vars)
	require.Empty(t, vars)
	bad := ctx.Value(unrenderedKey{}).(map[string]unrenderedVar)
	require.Len(t, bad, 2)
}

func TestLoopIgnoreEarlyFailure(t *testing.T) {
	e, _ := becomeExprEngine(t)
	host := twoHosts()[0]
	task := &types.Task{Name: "probe", Module: "become_probe", IgnoreErrorsExpr: "ignore",
		Vars: map[string]interface{}{"ignore": "{{ item == 1 }}"},
		Args: map[string]interface{}{"value": "{{ missing }}"}, Loop: &types.Loop{Items: []interface{}{1}}}
	result := &types.PlayResult{}
	require.NoError(t, e.runLoopOnHost(context.Background(), task, &host, nil, result))
	require.True(t, result.Hosts[0].Tasks[0].Failed)
}

// registered results are data: a loop over them must not render the braces
// a command printed, as the item carries them again
func TestLoopItemsAreData(t *testing.T) {
	e, module := becomeExprEngine(t)
	host := twoHosts()[0]
	e.setHostVar(host.Name, "braces", map[string]interface{}{
		"results": []interface{}{map[string]interface{}{"stdout": "{{ .Id }}"}},
	})
	task := &types.Task{Name: "copy", Module: "become_probe", When: "item.stdout | length > 0",
		Args: map[string]interface{}{"value": "{{ item.stdout }}"}, Loop: &types.Loop{Expr: "braces.results"}}
	require.NoError(t, e.executeTask(context.Background(), task, []types.Host{host}, map[string]interface{}{}, &types.PlayResult{}))
	require.Len(t, module.calls, 1)
}
