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

// Paths and content with shell metacharacters must arrive unchanged
func TestFileAndStat_ShellMetacharacters(t *testing.T) {
	host := types.Host{Name: "local", Address: "localhost", Vars: map[string]interface{}{"onigirazu_connection": "local"}}
	dir := filepath.Join(t.TempDir(), `it's $(dir) "x"`)
	path := filepath.Join(dir, "a'b; touch pwned")
	content := "don't $(expand) `me` \\ \"here\"\n"

	res, err := NewFileModule().Execute(context.Background(), host, map[string]interface{}{"path": dir, "state": "directory"})
	require.NoError(t, err)
	require.True(t, res.Success, res.Error)

	res, err = NewFileModule().Execute(context.Background(), host, map[string]interface{}{"path": path, "state": "present", "content": content})
	require.NoError(t, err)
	require.True(t, res.Success, res.Error)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(got))
	assert.NoFileExists(t, "pwned")

	res, err = NewStatModule().Execute(context.Background(), host, map[string]interface{}{"path": path})
	require.NoError(t, err)
	assert.Equal(t, true, res.Output["exists"])

	res, err = NewFileModule().Execute(context.Background(), host, map[string]interface{}{"path": path, "state": "absent"})
	require.NoError(t, err)
	assert.True(t, res.Changed)
	assert.NoFileExists(t, path)
}
