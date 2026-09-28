package managed

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func fileRes(path, kind string) types.ManagedResource {
	return types.ManagedResource{Type: "file", ID: path, Before: map[string]interface{}{"kind": kind, "content": "old\n"}}
}

func task(key string, res ...types.ManagedResource) types.TaskResult {
	return types.TaskResult{TaskName: key, TaskKey: key, Success: true, Resources: res}
}

func run(complete bool, keys []string, kept []string, tasks ...types.TaskResult) Run {
	k := map[string]bool{}
	for _, key := range keys {
		k[key] = true
	}
	return Run{Complete: complete, Scopes: Scopes{Keys: k, Kept: kept},
		Result: &types.PlaybookResult{Plays: []types.PlayResult{{Hosts: []types.HostResult{{Host: "h", Tasks: tasks}}}}}}
}

func TestUpdateOriginsAndOrphans(t *testing.T) {
	st := &State{}
	orphans := st.Update(run(true, []string{"p/a", "p/b", "p/c", "p/d"}, nil,
		task("p/a", fileRes("/a", "absent")),
		task("p/b", fileRes("/b", "file")),
		task("p/c", types.ManagedResource{Type: "package", ID: "nginx", Before: map[string]interface{}{"installed": []interface{}{}}}),
		task("p/d", types.ManagedResource{Type: "user", ID: "u", Before: map[string]interface{}{"exists": true}}),
	))
	assert.Empty(t, orphans)
	assert.Equal(t, OriginCreated, st.Find("h", "file", "/a").Origin)
	assert.Equal(t, OriginAdopted, st.Find("h", "file", "/b").Origin)
	assert.Equal(t, OriginCreated, st.Find("h", "package", "nginx").Origin)
	assert.Equal(t, OriginAdopted, st.Find("h", "user", "u").Origin)

	// a, c and d left the playbook
	orphans = st.Update(run(true, []string{"p/b"}, nil, task("p/b", fileRes("/b", "file"))))
	got := map[string]string{}
	for _, r := range orphans {
		got[r.Type+" "+r.ID] = r.Action()
	}
	assert.Equal(t, map[string]string{
		"file /a": ActionDestroy, "package nginx": ActionDestroy, "user u": ActionForget,
	}, got)
	assert.Equal(t, []string{"p/b"}, st.Find("h", "file", "/b").Tasks)
}

func TestUpdateKeepsClaims(t *testing.T) {
	base := func() *State {
		st := &State{}
		st.Update(run(true, []string{"p/a", "p/role:x/tasks/t"}, nil,
			task("p/a", fileRes("/a", "absent")), task("p/role:x/tasks/t", fileRes("/r", "absent"))))
		return st
	}

	// the task is still there but skipped (when): kept
	st := base()
	assert.Empty(t, st.Update(run(true, []string{"p/a", "p/role:x/tasks/t"}, nil)))

	// the role was skipped whole: its keys were never assigned
	st = base()
	assert.Empty(t, st.Update(run(true, []string{"p/a"}, []string{"p/role:x"})))

	// a partial run (--tags) drops nothing
	st = base()
	assert.Empty(t, st.Update(run(false, nil, nil)))

	// a failed task on the host drops nothing
	st = base()
	failed := types.TaskResult{TaskKey: "p/z", Failed: true}
	assert.Empty(t, st.Update(run(true, nil, nil, failed)))

	// the task ran and manages another path now: the old one is an orphan
	st = base()
	orphans := st.Update(run(true, []string{"p/a", "p/role:x/tasks/t"}, nil,
		task("p/a", fileRes("/a2", "absent")), task("p/role:x/tasks/t", fileRes("/r", "file"))))
	require.Len(t, orphans, 1)
	assert.Equal(t, "/a", orphans[0].ID)

	// state: absent forgets the record
	st = base()
	gone := fileRes("/a", "file")
	gone.Absent = true
	st.Update(run(true, []string{"p/a", "p/role:x/tasks/t"}, nil, task("p/a", gone)))
	assert.Nil(t, st.Find("h", "file", "/a"))
}

func TestActions(t *testing.T) {
	adoptedFile := &Record{Type: "file", Origin: OriginAdopted, Before: map[string]interface{}{"kind": "file", "content": "x"}}
	assert.Equal(t, ActionRestore, adoptedFile.Action())
	binary := &Record{Type: "file", Origin: OriginAdopted, Before: map[string]interface{}{"kind": "file"}}
	assert.Equal(t, ActionForget, binary.Action())
	kept := &Record{Type: "file", Origin: OriginCreated, PreventDestroy: true}
	assert.Equal(t, ActionForget, kept.Action())
	svc := &Record{Type: "service", Origin: OriginAdopted, Before: map[string]interface{}{"active": "active"}}
	assert.Equal(t, ActionRestore, svc.Action())
	assert.Equal(t, ActionForget, (&Record{Type: "file", Origin: OriginUnknown}).Action())
}

func TestSaveLoad(t *testing.T) {
	path := Path(filepath.Join(t.TempDir(), "site.yml"))
	st, err := Load(path)
	require.NoError(t, err)
	assert.Empty(t, st.Resources)
	st.Update(run(true, []string{"p/a"}, nil, task("p/a", fileRes("/a", "absent"))))
	require.NoError(t, st.Save(path))
	back, err := Load(path)
	require.NoError(t, err)
	require.Len(t, back.Resources, 1)
	assert.Equal(t, []string{"p/a"}, back.Resources[0].Tasks)
	assert.True(t, back.Remove("h", "file", "/a"))
	assert.False(t, back.Remove("h", "file", "/a"))
}

func TestSortForDestroyAndUndoResult(t *testing.T) {
	recs := []*Record{
		{Type: "user", ID: "u"},
		{Type: "file", ID: "/d"},
		{Type: "file", ID: "/d/f"},
		{Type: "package", ID: "p"},
	}
	SortForDestroy(recs)
	var got []string
	for _, r := range recs {
		got = append(got, r.ID)
	}
	assert.Equal(t, []string{"/d/f", "/d", "p", "u"}, got)

	res := UndoResult(&Record{Host: "h", Type: "file", Origin: OriginCreated, Before: map[string]interface{}{"kind": "absent"}})
	assert.Equal(t, true, res.Before["_keep_nonempty_dir"])
	assert.True(t, res.Changed)
	res = UndoResult(&Record{Type: "file", Origin: OriginAdopted, Before: map[string]interface{}{"kind": "file"}})
	assert.Nil(t, res.Before["_keep_nonempty_dir"])
}

func TestLock(t *testing.T) {
	path := Path(filepath.Join(t.TempDir(), "site.yml"))
	l, err := AcquireLock(path, "apply", 0)
	require.NoError(t, err)
	_, err = AcquireLock(path, "apply", 0)
	var locked *LockedError
	require.ErrorAs(t, err, &locked)
	assert.Equal(t, l.info.ID, locked.Info.ID)
	assert.Contains(t, err.Error(), "state unlock")

	assert.Error(t, ForceUnlock(path, "wrong"))
	require.NoError(t, ForceUnlock(path, l.info.ID))
	assert.Error(t, l.Release(), "the lock is gone")

	l2, err := AcquireLock(path, "apply", 0)
	require.NoError(t, err)
	require.NoError(t, l2.Release())
	assert.Error(t, ForceUnlock(path, l2.info.ID))
}
