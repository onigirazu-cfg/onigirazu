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
