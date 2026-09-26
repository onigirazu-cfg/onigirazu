package adhoc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestExecuteOnHost_CheckModeChangesNothing(t *testing.T) {
	e := &Executor{moduleRegistry: modules.NewRegistry()}
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	path := filepath.Join(t.TempDir(), "probe")
	task := &types.Task{Name: "touch", Module: "file", Args: map[string]interface{}{"path": path, "state": "touch"}}

	r := e.executeOnHost(context.Background(), task, host, Options{Check: true})
	require.NoError(t, r.Error)
	assert.True(t, r.Result.Changed, "reports what it would do")
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "--check must not touch the host")

	shell := &types.Task{Name: "sh", Module: "shell", Args: map[string]interface{}{"cmd": "touch " + path}}
	r = e.executeOnHost(context.Background(), shell, host, Options{Check: true})
	require.NoError(t, r.Error)
	assert.True(t, r.Result.Skipped, "a module without check mode is skipped")
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))

	r = e.executeOnHost(context.Background(), task, host, Options{})
	require.NoError(t, r.Error)
	_, err = os.Stat(path)
	assert.NoError(t, err, "without --check it runs")
}
