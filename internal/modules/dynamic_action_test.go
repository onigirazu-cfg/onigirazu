package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestResolveDynamicAction(t *testing.T) {
	task, err := resolveDynamicAction(&types.Task{Name: "x", Module: types.DynamicAction,
		Args: map[string]interface{}{"_action": `dnf name="gcc make" state=present`}})
	require.NoError(t, err)
	assert.Equal(t, "yum", task.Module, "dnf runs the yum module")
	assert.Equal(t, "gcc make", task.Args["name"])
	assert.Equal(t, "present", task.Args["state"])

	task, err = resolveDynamicAction(&types.Task{Module: types.DynamicAction,
		Args: map[string]interface{}{"_module": "ansible.builtin.apt", "name": "git"}})
	require.NoError(t, err)
	assert.Equal(t, "apt", task.Module)
	assert.Equal(t, map[string]interface{}{"name": "git"}, task.Args)

	_, err = resolveDynamicAction(&types.Task{Module: types.DynamicAction, Args: map[string]interface{}{"_module": "{{ x }}"}})
	assert.Error(t, err)
}
