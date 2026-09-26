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

func TestCommandAndShell_CreatesRemoves(t *testing.T) {
	host := types.Host{Name: "local", Address: "localhost", Vars: map[string]interface{}{"onigirazu_connection": "local"}}
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker")
	missing := filepath.Join(dir, "missing")
	ran := filepath.Join(dir, "ran")

	for _, m := range []types.Module{NewCommandModule(), NewShellModule()} {
		require.NoError(t, os.WriteFile(marker, nil, 0o600))
		_ = os.Remove(ran)

		res, err := m.Execute(context.Background(), host, map[string]interface{}{"cmd": "touch " + ran, "command": "touch " + ran, "creates": marker})
		require.NoError(t, err)
		assert.False(t, res.Changed, "creates exists: skipped")
		assert.NoFileExists(t, ran)

		res, err = m.Execute(context.Background(), host, map[string]interface{}{"cmd": "touch " + ran, "command": "touch " + ran, "removes": missing})
		require.NoError(t, err)
		assert.False(t, res.Changed, "removes missing: skipped")
		assert.NoFileExists(t, ran)

		res, err = m.Execute(context.Background(), host, map[string]interface{}{"cmd": "touch " + ran, "command": "touch " + ran, "creates": missing})
		require.NoError(t, err)
		assert.True(t, res.Changed, "creates missing: runs")
		assert.FileExists(t, ran)
	}
}
