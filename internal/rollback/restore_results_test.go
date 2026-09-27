package rollback

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestRestoreResults(t *testing.T) {
	registry := newMockModuleRegistry()
	var calls []string
	record := func(name string) *mockModule {
		return &mockModule{name: name, executeFunc: func(_ context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
			calls = append(calls, name+" "+host.Name+" "+args["dest"].(string))
			return types.TaskResult{Success: true}, nil
		}}
	}
	registry.RegisterModule("copy", record("copy"))
	ex := NewRollbackExecutor(nil, registry, &mockLogger{})

	file := func(host, dest, content string) types.TaskResult {
		return types.TaskResult{Host: host, TaskName: "write " + dest, Module: "copy", Changed: true,
			Before: map[string]interface{}{"kind": "file", "path": dest, "content": content, "mode": "0644"}}
	}
	results := []types.TaskResult{
		file("web1", "/etc/a", "old a"),
		{Host: "web1", TaskName: "unchanged", Module: "copy"},
		file("web1", "/etc/b", "old b"),
		{Host: "web1", TaskName: "restart", Module: "command", Changed: true},
	}
	rep := ex.RestoreResults(context.Background(), results)
	assert.Equal(t, 2, rep.Undone)
	assert.Equal(t, []string{"copy web1 /etc/b", "copy web1 /etc/a"}, calls) // newest first
	assert.Equal(t, []string{"web1: restart (command)"}, rep.Irreversible)
	assert.Empty(t, rep.Failed)
}
