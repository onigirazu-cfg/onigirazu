package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestActionKeyword(t *testing.T) {
	parse := func(src string) Task {
		var task Task
		require.NoError(t, yaml.Unmarshal([]byte(src), &task))
		return task
	}
	task := parse("name: a\naction: copy src=a dest=/b\n")
	assert.Equal(t, "copy", task.Module)
	assert.Equal(t, "/b", task.Args["dest"])
	assert.Empty(t, task.DelegateTo)

	task = parse("action:\n  module: ansible.builtin.file\n  path: /x\n  state: touch\n")
	assert.Equal(t, "file", task.Module)
	assert.Equal(t, "/x", task.Args["path"])

	task = parse("action: >\n  {{ ansible_pkg_mgr }} name=\"{{ pkgs }}\" state=present\nbecome: true\n")
	assert.Equal(t, DynamicAction, task.Module)
	assert.Equal(t, "{{ ansible_pkg_mgr }}", task.Args["_module"])
	assert.Equal(t, "{{ pkgs }}", task.Args["name"], "each value renders on its own")
	assert.Equal(t, "present", task.Args["state"])

	task = parse("action:\n  module: '{{ mgr }}'\n  name: git\n")
	assert.Equal(t, DynamicAction, task.Module)
	assert.Equal(t, "{{ mgr }}", task.Args["_module"])

	task = parse("local_action: command true\n")
	assert.Equal(t, "command", task.Module)
	assert.Equal(t, "localhost", task.DelegateTo)
}
