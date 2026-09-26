package execution

import (
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromPlaybookResult(t *testing.T) {
	result := &types.PlaybookResult{
		Plays: []types.PlayResult{{
			Name: "web",
			Hosts: []types.HostResult{
				{Host: "a", Success: true, Tasks: []types.TaskResult{
					{TaskName: "copy", Changed: true, Success: true},
					{TaskName: "cmd", Success: true},
				}},
				{Host: "b", Failed: true, Tasks: []types.TaskResult{
					{TaskName: "copy", Success: true},
					{TaskName: "cmd", Failed: true, Error: "exit 1"},
				}},
			},
		}},
		Stats: map[string]interface{}{"a": 1, "b": 2, "c": 3},
	}
	start := time.Now()
	exec := FromPlaybookResult(result, "/p/site.yml", "site.yml", start, time.Second)

	assert.Equal(t, 2, exec.TotalHosts, "hosts, not stats entries")
	require.Len(t, exec.Tasks, 2)
	assert.Equal(t, "copy", exec.Tasks[0].Name)
	assert.Equal(t, 2, exec.Tasks[0].Total)
	assert.Equal(t, 1, exec.Tasks[0].Changed)
	assert.Equal(t, 1, exec.Tasks[1].Failed)
	assert.Equal(t, "failed", exec.Tasks[1].HostResults["b"].Status)
	assert.Equal(t, 3, exec.TotalSuccess)
	assert.Equal(t, 1, exec.TotalFailed)
	assert.Equal(t, "partial_success", exec.Status)

	other := FromPlaybookResult(result, "/p/site.yml", "site.yml", start.Add(time.Millisecond), time.Second)
	assert.NotEqual(t, exec.ExecutionID, other.ExecutionID, "runs within one second get their own id")
}
