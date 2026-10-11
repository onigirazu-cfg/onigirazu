package engine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// a host whose facts cannot be gathered leaves the run: UNREACHABLE on a
// connection failure, FAILED otherwise; the play goes on with the rest
func TestHostsLeftAfterFacts(t *testing.T) {
	e, _, lg, _, _, _ := createTestEngine()
	lg.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	hosts := []types.Host{{Name: "up"}, {Name: "down"}, {Name: "broken"}}
	result := &types.PlayResult{Success: true}
	failed := map[string]error{
		"down":   errors.New("failed to connect to 10.0.0.9:22: dial tcp: i/o timeout"),
		"broken": errors.New("setup: command not found"),
	}
	require.NoError(t, e.hostsLeftAfterFacts(failed, hosts, result))
	assert.False(t, result.Success)
	active := e.activeHosts(hosts)
	require.Len(t, active, 1)
	assert.Equal(t, "up", active[0].Name)
	require.Len(t, result.Hosts, 2)
	assert.Equal(t, "broken", result.Hosts[0].Host)
	assert.Contains(t, result.Hosts[0].Tasks[0].Error, "gathering facts failed")
	assert.Equal(t, "down", result.Hosts[1].Host)
	assert.Contains(t, result.Hosts[1].Tasks[0].Error, "unreachable")

	// nobody left: the play fails
	e2, _, lg2, _, _, _ := createTestEngine()
	lg2.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	err := e2.hostsLeftAfterFacts(map[string]error{"only": errors.New("failed to connect to x")}, []types.Host{{Name: "only"}}, &types.PlayResult{})
	assert.ErrorContains(t, err, "host only unreachable")
}

// a templated src of a role task is placed after rendering: absolute as it
// is, relative in the role's files
func TestResolveRoleSource(t *testing.T) {
	role := t.TempDir()
	files := filepath.Join(role, "files")
	require.NoError(t, os.MkdirAll(files, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(files, "in-files.txt"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(role, "in-role.txt"), nil, 0o644))

	abs := map[string]interface{}{"_role_dir": files, "src": "/Users/me/topology.yaml", "dest": "/etc/t"}
	resolveRoleSource("copy", abs)
	assert.Equal(t, "/Users/me/topology.yaml", abs["src"])
	_, has := abs["_role_dir"]
	assert.False(t, has)

	rel := map[string]interface{}{"_role_dir": files, "src": "in-files.txt"}
	resolveRoleSource("copy", rel)
	assert.Equal(t, filepath.Join(files, "in-files.txt"), rel["src"])

	up := map[string]interface{}{"_role_dir": files, "src": "in-role.txt"}
	resolveRoleSource("copy", up)
	assert.Equal(t, filepath.Join(role, "in-role.txt"), up["src"])

	remote := map[string]interface{}{"_role_dir": files, "src": "on-the-host.txt", "remote_src": true}
	resolveRoleSource("copy", remote)
	assert.Equal(t, "on-the-host.txt", remote["src"])

	script := map[string]interface{}{"_role_dir": files, "script": "run.sh"}
	resolveRoleSource("script", script)
	assert.Equal(t, filepath.Join(files, "run.sh"), script["script"])
}
