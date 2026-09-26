package modules

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestConfigSet_NestedKeyIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"server":{"host":"0.0.0.0"}}`), 0o644))
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	args := map[string]interface{}{"path": path, "format": "json", "action": "set", "key": "server.port", "value": 8080}

	res, err := NewConfigModule().Execute(context.Background(), host, args)
	require.NoError(t, err)
	require.True(t, res.Success, res.Error)
	assert.True(t, res.Changed)

	var got map[string]map[string]interface{}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, float64(8080), got["server"]["port"])
	assert.Equal(t, "0.0.0.0", got["server"]["host"])

	res, err = NewConfigModule().Execute(context.Background(), host, args)
	require.NoError(t, err)
	assert.False(t, res.Changed, "second set changes nothing")
}

func TestConfig_CheckModeMergeAndDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.json")
	orig := []byte(`{"server":{"port":8080}}`)
	require.NoError(t, os.WriteFile(path, orig, 0o644))
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	run := func(args map[string]interface{}) types.TaskResult {
		args["path"], args["format"] = path, "json"
		res, err := NewConfigModule().Execute(context.Background(), host, args)
		require.NoError(t, err)
		require.True(t, res.Success, res.Error)
		return res
	}

	res := run(map[string]interface{}{"action": "set", "key": "server.host", "value": "x", "_check_mode": true})
	assert.True(t, res.Changed)
	res = run(map[string]interface{}{"action": "delete", "_check_mode": true})
	assert.True(t, res.Changed)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, orig, data, "check mode changed nothing")

	// merge compares by JSON form: 8080 (int) equals 8080 read back from JSON
	res = run(map[string]interface{}{"action": "merge", "values": map[string]interface{}{"server": map[string]interface{}{"port": 8080}}})
	assert.False(t, res.Changed)

	require.NoError(t, os.Remove(path))
	res = run(map[string]interface{}{"action": "delete"})
	assert.False(t, res.Changed, "deleting a missing file is no change")
}
