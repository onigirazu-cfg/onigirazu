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

func TestFromPlaybookResultKeepsTasksWithTheSameName(t *testing.T) {
	res := &types.PlaybookResult{Plays: []types.PlayResult{{Name: "p", Hosts: []types.HostResult{
		{Host: "a", Tasks: []types.TaskResult{
			{TaskName: "debug", Module: "debug", Output: map[string]interface{}{"msg": "one"}},
			{TaskName: "debug", Module: "debug", Skipped: true},
			{TaskName: "fail", Module: "fail", Failed: true, Ignored: true, Error: "boom"},
		}},
		{Host: "b", Tasks: []types.TaskResult{
			{TaskName: "debug", Module: "debug", Output: map[string]interface{}{"msg": "one"}},
			{TaskName: "debug", Module: "debug", Changed: true},
			{TaskName: "fail", Module: "fail"},
		}},
	}}}}
	exec := FromPlaybookResult(res, "p.yml", "p.yml", time.Now(), time.Second)
	if len(exec.Tasks) != 3 {
		t.Fatalf("tasks = %d, want 3", len(exec.Tasks))
	}
	if got := exec.Tasks[0].HostResults["a"].Output; got != "one" {
		t.Errorf("debug output = %q", got)
	}
	if exec.Tasks[1].HostResults["a"].Status != "skipped" || exec.Tasks[1].HostResults["b"].Status != "changed" {
		t.Errorf("second task = %+v", exec.Tasks[1].HostResults)
	}
	if r := exec.Tasks[2].HostResults["a"]; r.Status != "ignored" || r.Output != "boom" {
		t.Errorf("ignored failure = %+v", r)
	}
}
