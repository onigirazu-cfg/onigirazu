package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/internal/engine"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/managed"
	"github.com/onigirazu-cfg/onigirazu/internal/rollback"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type fakeUndo struct {
	calls []string
	errs  map[string]error
}

func (f *fakeUndo) Undo(_ context.Context, t types.TaskResult) (bool, error) {
	id, _ := t.Before["path"].(string)
	if id == "" {
		id, _ = t.Before["name"].(string)
	}
	f.calls = append(f.calls, id)
	return true, f.errs[id]
}

func runOf(host string, tasks ...types.TaskResult) *types.PlaybookResult {
	return &types.PlaybookResult{Plays: []types.PlayResult{{Hosts: []types.HostResult{{Host: host, Tasks: tasks}}}}}
}

func TestDestroyOrphans(t *testing.T) {
	log := logger.NewWithWriter(false, io.Discard)
	playbook := filepath.Join(t.TempDir(), "site.yml")
	created := func(path string) types.ManagedResource {
		return types.ManagedResource{Type: "file", ID: path, Before: map[string]interface{}{"path": path, "kind": "absent"}}
	}
	first := runOf("h",
		types.TaskResult{TaskKey: "p/a", Success: true, Resources: []types.ManagedResource{created("/a")}},
		types.TaskResult{TaskKey: "p/b", Success: true, Resources: []types.ManagedResource{created("/b")}},
		types.TaskResult{TaskKey: "p/c", Success: true, Resources: []types.ManagedResource{created("/c")}})
	keys := engine.ManagedScopes{Keys: map[string]bool{"p/a": true, "p/b": true, "p/c": true}}
	updateManagedState(playbook, first, keys, true, false, log)

	// p/b and p/c left the playbook; /c cannot be removed
	second := runOf("h", types.TaskResult{TaskKey: "p/a", Success: true, Resources: []types.ManagedResource{created("/a")}})
	m := updateManagedState(playbook, second, engine.ManagedScopes{Keys: map[string]bool{"p/a": true}}, true, false, log)
	require.Len(t, second.Orphans, 2)
	u := &fakeUndo{errs: map[string]error{"/c": errors.New("busy")}}
	var out bytes.Buffer
	err := m.destroyOrphans(context.Background(), u, second, func(int) bool { return true }, &out, log)
	assert.Error(t, err)
	assert.ElementsMatch(t, []string{"/b", "/c"}, u.calls)
	assert.Contains(t, out.String(), "removed file /b on h")
	assert.Contains(t, out.String(), "failed file /c on h")
	st, err := managed.Load(managed.Path(playbook))
	require.NoError(t, err)
	assert.Nil(t, st.Find("h", "file", "/b"))
	assert.NotNil(t, st.Find("h", "file", "/c"), "a failed removal stays an orphan")

	// declined: nothing happens; a failed host: nothing happens
	u = &fakeUndo{}
	assert.NoError(t, m.destroyOrphans(context.Background(), u, second, func(int) bool { return false }, &out, log))
	failed := runOf("h", types.TaskResult{TaskKey: "p/x", Failed: true})
	assert.NoError(t, m.destroyOrphans(context.Background(), u, failed, func(int) bool { return true }, &out, log))
	assert.Empty(t, u.calls)

	// a kept directory is forgotten
	u = &fakeUndo{errs: map[string]error{"/c": &rollback.KeptError{Reason: "not empty"}}}
	assert.NoError(t, m.destroyOrphans(context.Background(), u, second, func(int) bool { return true }, &out, log))
	st, _ = managed.Load(managed.Path(playbook))
	assert.Nil(t, st.Find("h", "file", "/c"))
}
