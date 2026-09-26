package modules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func firstDiff(t *testing.T, r types.TaskResult) map[string]interface{} {
	t.Helper()
	list, ok := r.Output["diff"].([]interface{})
	require.True(t, ok, "diff in %v", r.Output)
	require.NotEmpty(t, list)
	return list[0].(map[string]interface{})
}

func TestModulesReportDiffs(t *testing.T) {
	dir := t.TempDir()
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
		return p
	}
	ctx := context.Background()
	with := func(args map[string]interface{}) map[string]interface{} {
		args["_diff"], args["_check_mode"] = true, true
		return args
	}

	p := write("line", "x=1\ny=1\n")
	r, err := NewLineinfileModule().Execute(ctx, host, with(map[string]interface{}{"path": p, "line": "x=2", "regexp": "^x="}))
	require.NoError(t, err)
	d := firstDiff(t, r)
	assert.Equal(t, "x=1\ny=1\n", d["before"])
	assert.Equal(t, "x=2\ny=1\n", d["after"])

	p = write("replace", "a b\n")
	r, err = NewReplaceModule().Execute(ctx, host, with(map[string]interface{}{"path": p, "regexp": "b", "replace": "c"}))
	require.NoError(t, err)
	assert.Equal(t, "a c\n", firstDiff(t, r)["after"])

	p = write("block", "top\n")
	r, err = NewBlockinfileModule().Execute(ctx, host, with(map[string]interface{}{"path": p, "block": "inside"}))
	require.NoError(t, err)
	assert.Contains(t, firstDiff(t, r)["after"], "inside")

	p = write("copy", "old\n")
	r, err = NewCopyModule().Execute(ctx, host, with(map[string]interface{}{"dest": p, "content": "new\n"}))
	require.NoError(t, err)
	d = firstDiff(t, r)
	assert.Equal(t, "old\n", d["before"])
	assert.Equal(t, "new\n", d["after"])

	// no --diff, no diff
	r, err = NewReplaceModule().Execute(ctx, host, map[string]interface{}{"path": p, "regexp": "old", "replace": "x", "_check_mode": true})
	require.NoError(t, err)
	assert.Nil(t, r.Output["diff"])
}

func TestDiffText(t *testing.T) {
	assert.Equal(t, "", diffText([]byte("x"), false))
	assert.Equal(t, "(binary, 3 bytes)\n", diffText([]byte{1, 0, 2}, true))
}

func TestCaptureBefore(t *testing.T) {
	dir := t.TempDir()
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	p := filepath.Join(dir, "conf")
	require.NoError(t, os.WriteFile(p, []byte("v1\n"), 0o640))
	before := captureBefore(context.Background(), host, "copy", map[string]interface{}{"dest": p, "_become": false})
	assert.Equal(t, "file", before["kind"])
	assert.Equal(t, "v1\n", before["content"])
	assert.Equal(t, "0640", before["mode"])
	assert.Equal(t, false, before["_become"])

	absent := captureBefore(context.Background(), host, "file", map[string]interface{}{"path": filepath.Join(dir, "none")})
	assert.Equal(t, "absent", absent["kind"])
	assert.Nil(t, captureBefore(context.Background(), host, "apt", map[string]interface{}{"name": "x"}))
}
