package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestAssertModule(t *testing.T) {
	vars := map[string]interface{}{"pkgs": []interface{}{"curl"}, "port": 80}
	run := func(args map[string]interface{}) types.TaskResult {
		args["_vars"] = vars
		r, err := NewAssertModule().Execute(context.Background(), types.Host{Name: "h"}, args)
		assert.NoError(t, err)
		return r
	}
	ok := run(map[string]interface{}{"that": []interface{}{"pkgs | length == 1", "'curl' in pkgs", "port > 10"}, "success_msg": "fine"})
	assert.True(t, ok.Success)
	assert.Equal(t, "fine", ok.Output["msg"])

	bad := run(map[string]interface{}{"that": []interface{}{"port > 10", "port > 100"}, "fail_msg": "port too low"})
	assert.False(t, bad.Success)
	assert.Equal(t, "port too low", bad.Error)
	assert.Equal(t, "port > 100", bad.Output["assertion"])

	single := run(map[string]interface{}{"that": "missing is defined"})
	assert.False(t, single.Success)

	assert.Error(t, NewAssertModule().Validate(map[string]interface{}{}))
}
