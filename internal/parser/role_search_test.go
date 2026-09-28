package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindRole(t *testing.T) {
	dir := t.TempDir()
	mk := func(rel string) string {
		p := filepath.Join(dir, rel)
		require.NoError(t, os.MkdirAll(filepath.Join(p, "tasks"), 0o755))
		return p
	}
	local := mk("playbooks/roles/local")
	shared := mk("roles/shared")
	coll := mk("colls/ansible_collections/clanred/infra/roles/users")
	byPath := mk("vendor/thing")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ansible.cfg"),
		[]byte("[defaults]\nroles_path = ./roles:~/nowhere\ncollections_path = ./colls\n"), 0o644))

	t.Chdir(dir)
	t.Setenv("ANSIBLE_CONFIG", "")
	t.Setenv("ANSIBLE_ROLES_PATH", "")
	t.Setenv("ANSIBLE_COLLECTIONS_PATH", "")
	playRoles := filepath.Join(dir, "playbooks", "roles")

	got, _ := findRole("local", playRoles)
	assert.Equal(t, local, got)
	got, _ = findRole("shared", playRoles)
	assert.Equal(t, shared, got, "roles_path of ansible.cfg, relative to it")
	got, _ = findRole("clanred.infra.users", playRoles)
	assert.Equal(t, coll, got, "a collection role")
	got, _ = findRole("../vendor/thing", playRoles)
	assert.Equal(t, filepath.Clean(byPath), filepath.Clean(got), "a path relative to the playbook")
	got, tried := findRole("nope", playRoles)
	assert.Empty(t, got)
	assert.Contains(t, tried, filepath.Join(dir, "roles", "nope"))

	SetRoleSearch([]string{filepath.Join(dir, "vendor")}, nil)
	defer SetRoleSearch(nil, nil)
	got, _ = findRole("thing", playRoles)
	assert.Equal(t, byPath, got, "roles_path of onigirazu.yml")
}
