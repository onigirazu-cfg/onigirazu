package modules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// In check mode every file module reports the change it would make and
// leaves the disk exactly as it was
func TestCheckMode_FileModulesChangeNothing(t *testing.T) {
	host := types.Host{Name: "local", Address: "localhost", Vars: map[string]interface{}{"onigirazu_connection": "local"}}
	dir := t.TempDir()
	existing := filepath.Join(dir, "existing.txt")
	require.NoError(t, os.WriteFile(existing, []byte("old\n"), 0o600))

	snapshot := func() map[string]string {
		out := map[string]string{}
		_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				b, _ := os.ReadFile(p)
				out[p] = info.Mode().String() + " " + string(b)
			} else if err == nil {
				out[p] = "dir"
			}
			return nil
		})
		return out
	}
	before := snapshot()

	cases := []struct {
		name   string
		module types.Module
		args   map[string]interface{}
	}{
		{"copy new", NewCopyModule(), map[string]interface{}{"content": "x\n", "dest": filepath.Join(dir, "new.txt")}},
		{"copy changed", NewCopyModule(), map[string]interface{}{"content": "new\n", "dest": existing}},
		{"copy mode", NewCopyModule(), map[string]interface{}{"content": "old\n", "dest": existing, "mode": "0644"}},
		{"template", NewTemplateModule(), map[string]interface{}{"content": "t\n", "dest": filepath.Join(dir, "t.conf")}},
		{"file present", NewFileModule(), map[string]interface{}{"path": filepath.Join(dir, "p"), "state": "present", "content": "p"}},
		{"file directory", NewFileModule(), map[string]interface{}{"path": filepath.Join(dir, "d"), "state": "directory", "mode": "0700"}},
		{"file absent", NewFileModule(), map[string]interface{}{"path": existing, "state": "absent"}},
		{"file touch", NewFileModule(), map[string]interface{}{"path": filepath.Join(dir, "touched"), "state": "touch"}},
		{"lineinfile", NewLineinfileModule(), map[string]interface{}{"path": existing, "line": "added", "backup": true}},
		{"blockinfile", NewBlockinfileModule(), map[string]interface{}{"path": existing, "block": "b", "backup": true}},
	}
	for _, c := range cases {
		args := map[string]interface{}{"_check_mode": true}
		for k, v := range c.args {
			args[k] = v
		}
		res, err := c.module.Execute(context.Background(), host, args)
		require.NoError(t, err, c.name)
		require.True(t, res.Success, "%s: %s", c.name, res.Error)
		assert.True(t, res.Changed, "%s reports the change it would make", c.name)
		assert.Equal(t, before, snapshot(), "%s changed nothing on disk", c.name)
	}

	// no change needed: check mode says so
	res, err := NewCopyModule().Execute(context.Background(), host, map[string]interface{}{"_check_mode": true, "content": "old\n", "dest": existing})
	require.NoError(t, err)
	assert.False(t, res.Changed)
}

func TestCheckMode_UnsupportedModuleIsSkipped(t *testing.T) {
	check := true
	host := types.Host{Name: "local", Address: "localhost", Vars: map[string]interface{}{"onigirazu_connection": "local"}}
	marker := filepath.Join(t.TempDir(), "ran")
	res, err := NewRegistry().ExecuteTask(context.Background(), &types.Task{
		Name: "cmd", Module: "command", CheckMode: &check,
		Args: map[string]interface{}{"cmd": "touch " + marker},
	}, host, nil)
	require.NoError(t, err)
	assert.True(t, res.Skipped)
	assert.NoFileExists(t, marker)
}
