package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRoleFiles(t *testing.T) {
	role := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(role, "templates"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(role, "files"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(role, "templates", "a.j2"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(role, "files", "b.txt"), nil, 0o644))

	tasks := []types.Task{
		{Module: "template", Args: map[string]interface{}{"src": "a.j2"}},
		{Block: []types.Task{{Module: "copy", Args: map[string]interface{}{"src": "b.txt"}}}},
		{Module: "copy", Args: map[string]interface{}{"src": "b.txt", "remote_src": true}},
		{Module: "copy", Args: map[string]interface{}{"src": "missing.txt"}},
		{Module: "template", Args: map[string]interface{}{"src": "{{ name }}.j2"}},
		{Module: "script", Args: map[string]interface{}{"script": "b.txt"}},
	}
	resolveRoleFiles(tasks, role)
	assert.Equal(t, filepath.Join(role, "templates", "a.j2"), tasks[0].Args["src"])
	assert.Equal(t, filepath.Join(role, "files", "b.txt"), tasks[1].Block[0].Args["src"])
	assert.Equal(t, "b.txt", tasks[2].Args["src"], "remote_src is on the host")
	assert.Equal(t, "missing.txt", tasks[3].Args["src"])
	assert.Equal(t, filepath.Join(role, "templates", "{{ name }}.j2"), tasks[4].Args["src"])
	assert.Equal(t, filepath.Join(role, "files", "b.txt"), tasks[5].Args["script"])
}
