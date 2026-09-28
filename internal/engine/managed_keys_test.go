package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestAssignKeys(t *testing.T) {
	e := &ExecutionEngine{}
	tasks := []types.Task{
		{Name: "Install", Module: "apt"},
		{Name: "Install", Module: "apt"},
		{Module: "copy"},
		{Module: "copy"},
		{Block: []types.Task{{Name: "inner", Module: "file"}}, Rescue: []types.Task{{Module: "debug"}}},
	}
	e.assignKeys(tasks, "play:web/tasks")
	assert.Equal(t, "play:web/tasks/Install", tasks[0].Key)
	assert.Equal(t, "play:web/tasks/Install#2", tasks[1].Key)
	assert.Equal(t, "play:web/tasks/copy#1", tasks[2].Key)
	assert.Equal(t, "play:web/tasks/copy#2", tasks[3].Key)
	assert.Equal(t, "play:web/tasks/block#1/block/inner", tasks[4].Block[0].Key)
	assert.Equal(t, "play:web/tasks/block#1/rescue/debug#1", tasks[4].Rescue[0].Key)

	role := &types.Role{Name: "nginx", Tasks: []types.Task{{Name: "conf"}}, Handlers: []types.Task{{Name: "reload"}}}
	e.assignRoleKeys(role, "play:web")
	e.keepScope("play:web/role:skipped")
	s := e.ManagedScopes()
	assert.True(t, s.Keys["play:web/role:nginx/tasks/conf"])
	assert.True(t, s.Keys["play:web/role:nginx/handlers/reload"])
	assert.Equal(t, []string{"play:web/role:skipped"}, s.Kept)
}

func TestPlayScopes(t *testing.T) {
	assert.Equal(t, []string{"play:a", "play:#2:b", "play:#3:b", "play:#4:"},
		playScopes([]types.Play{{Name: "a"}, {Name: "b"}, {Name: "b"}, {}}))
}
