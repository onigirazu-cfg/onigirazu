package engine

import (
	"context"
	"errors"
	"testing"

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
	for _, keyword := range []string{"when", "length", "changed_when", "failed_when", "until", "loop", "with_items", "with_lookup", "break_when", "args_filter", "environment", "delegate_to", "retries", "delay", "become", "register"} {
		t.Run(keyword, func(t *testing.T) {
			e, module := becomeExprEngine(t)
			host := twoHosts()[0]
			vars := map[string]interface{}{"broken": "{{ .Names }}"}
			task := &types.Task{Name: "probe", Module: "become_probe"}
			switch keyword {
			case "when":
				task.When = "broken"
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
			case "with_lookup":
				task.Loop = &types.Loop{Lookup: "nested", Terms: []interface{}{"{{ [broken | length] }}"}}
			case "break_when":
				task.Loop = &types.Loop{Items: []interface{}{1}}
				task.BreakWhen = "broken | length > 0"
			case "environment":
				task.Environment = map[string]interface{}{"VALUE": "{{ broken | length }}"}
			case "delegate_to":
				task.DelegateTo = "{{ broken | length }}"
			case "retries":
				task.RetriesExpr = "{{ broken | length }}"
			case "delay":
				task.DelayExpr = "{{ broken | length }}"
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
	for _, condition := range []string{"broken is defined", "broken is undefined", "broken | default('fallback')", "'broken' | length > 0"} {
		t.Run(condition, func(t *testing.T) {
			e, _ := becomeExprEngine(t)
			host := twoHosts()[0]
			task := &types.Task{Name: "safe", Module: "become_probe", When: condition}
			result := &types.PlayResult{}
			require.NoError(t, e.executeTaskOnHost(context.Background(), task, &host, map[string]interface{}{"broken": "{{ .Names }}"}, result))
			require.Equal(t, condition == "broken is undefined", result.Hosts[0].Tasks[0].Skipped)
		})
	}
}

func TestUnrenderedReferenceDetection(t *testing.T) {
	bad := map[string]unrenderedVar{"broken": {text: "{{ .Names }}", err: errors.New("render failed")}}
	ctx := context.WithValue(context.Background(), unrenderedKey{}, bad)
	vars := map[string]interface{}{"broken": "{{ .Names }}"}
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
