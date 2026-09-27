package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestRolloutBatches(t *testing.T) {
	e := &ExecutionEngine{}
	b, canary, err := e.rolloutBatches(&types.Play{Serial: 2}, 5)
	require.NoError(t, err)
	assert.Equal(t, []int{2, 2, 1}, b)
	assert.False(t, canary)

	e.safe.Canary = "1"
	b, canary, err = e.rolloutBatches(&types.Play{}, 5)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 4}, b)
	assert.True(t, canary)

	b, _, _ = e.rolloutBatches(&types.Play{Serial: 2}, 5)
	assert.Equal(t, []int{1, 2, 2}, b)

	e.safe.Canary = "20%"
	b, _, _ = e.rolloutBatches(&types.Play{}, 10)
	assert.Equal(t, []int{2, 8}, b)

	e.safe.Canary = "9"
	b, _, _ = e.rolloutBatches(&types.Play{}, 3)
	assert.Equal(t, []int{3}, b)
}

func TestOnUnhealthy(t *testing.T) {
	e := &ExecutionEngine{}
	checks := []types.Task{{Name: "c"}}
	assert.Equal(t, "rollback", e.onUnhealthy(&types.Play{HealthCheck: checks}))
	assert.Equal(t, "stop", e.onUnhealthy(&types.Play{HealthCheck: checks, OnUnhealthy: "stop"}))
	assert.Equal(t, "stop", e.onUnhealthy(&types.Play{}))
	e.safe.AutoRollback = true
	assert.Equal(t, "rollback", e.onUnhealthy(&types.Play{}))
	assert.True(t, e.rolloutEnabled(&types.Play{}))
	assert.False(t, (&ExecutionEngine{}).rolloutEnabled(&types.Play{}))
}

func TestChangesAndMarkRolledBack(t *testing.T) {
	r := &types.PlayResult{Hosts: []types.HostResult{{Host: "web1", Tasks: []types.TaskResult{
		{TaskName: "a", Changed: true},
		{TaskName: "b"},
		{TaskName: "c", Changed: true, Failed: true},
		{TaskName: "a", Changed: true},
	}}}}
	changes := changesOf(r)
	require.Len(t, changes, 2)
	assert.Equal(t, "web1", changes[0].Host)

	e := &ExecutionEngine{}
	e.markRolledBack(r, changes[:1])
	assert.True(t, r.Hosts[0].Tasks[0].RolledBack)
	assert.False(t, r.Hosts[0].Tasks[3].RolledBack)
}
