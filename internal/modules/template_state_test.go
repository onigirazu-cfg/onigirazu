package modules

import (
	"context"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func runTemplate(t *testing.T, args map[string]interface{}) types.TaskResult {
	t.Helper()
	host := types.Host{Name: "localhost", Address: "localhost"}
	result, err := NewTemplateModule().Execute(context.Background(), host, args)
	require.NoError(t, err)
	require.True(t, result.Success, result.Error)
	return result
}

func TestTemplate_IdempotentAndModeDrift(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "app.conf")
	args := map[string]interface{}{"content": "port = {{ port }}\n", "dest": dest, "mode": "0640", "vars": map[string]interface{}{"port": 8080}}

	assert.True(t, runTemplate(t, args).Changed, "first run creates the file")
	assert.False(t, runTemplate(t, args).Changed, "second run is a no-op")

	require.NoError(t, os.Chmod(dest, 0644))
	assert.True(t, runTemplate(t, args).Changed, "mode drift is corrected")
	info, err := os.Stat(dest)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0640), info.Mode().Perm())
	assert.False(t, runTemplate(t, args).Changed)
}

func TestTemplate_ModeNotEnforcedWhenUnset(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "app.conf")
	args := map[string]interface{}{"content": "x\n", "dest": dest}
	runTemplate(t, args)
	require.NoError(t, os.Chmod(dest, 0600))
	assert.False(t, runTemplate(t, args).Changed)
}

func TestTemplate_OwnerMatchingCurrentUser(t *testing.T) {
	u, err := user.Current()
	require.NoError(t, err)
	dest := filepath.Join(t.TempDir(), "app.conf")
	args := map[string]interface{}{"content": "x\n", "dest": dest, "owner": u.Username}

	result := runTemplate(t, args)
	assert.True(t, result.Changed)
	assert.NotContains(t, result.Output, "ownership_warning")
	assert.False(t, runTemplate(t, args).Changed)
}
