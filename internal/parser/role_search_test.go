package parser

import (
	"os"
	"path/filepath"
	"strings"
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

func TestParserRoleSearchIsolation(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{"first", "second", "third"} {
		dir := filepath.Join(base, name)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "same"), 0o755))
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := NewEnhancedParser(&mockTemplateEngine{}, &mockLogger{})
			paths := []string{dir}
			p.SetRoleSearch(paths, nil)
			paths[0] = "changed"
			got, _ := findRoleWithSearch("same", filepath.Join(base, "roles"), p.roleLoader.search)
			require.Equal(t, filepath.Join(dir, "same"), got)
		})
	}
}

func TestCollectionSearchPaths(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfgDir := filepath.Join(dir, "config")
	require.NoError(t, os.Mkdir(cfgDir, 0o755))
	cfg := filepath.Join(cfgDir, "ansible.cfg")
	require.NoError(t, os.WriteFile(cfg, []byte("[defaults]\ncollections_path = ./collections:../shared\n"), 0o600))
	t.Setenv("ANSIBLE_CONFIG", cfg)
	t.Setenv("ANSIBLE_COLLECTIONS_PATH", "/env/one:/env/two")
	t.Setenv("ANSIBLE_COLLECTIONS_PATHS", "/env/legacy")
	t.Setenv("HOME", dir)
	configured := []string{"/configured"}
	got := CollectionSearchPaths(dir, configured)
	require.Equal(t, []string{
		"/configured", "/env/one", "/env/two", "/env/legacy",
		filepath.Join(cfgDir, "collections"), filepath.Join(dir, "shared"),
		filepath.Join(dir, ".ansible", "collections"), "/usr/share/ansible/collections",
	}, got)
	require.Equal(t, []string{"/configured"}, configured)
}

func TestAnsibleCfgIgnoredInWorldWritableCwd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ANSIBLE_CONFIG", "")
	t.Setenv("HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(dir, "ansible.cfg"), []byte("[defaults]\ncollections_path = ./evil\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o777); err != nil { // #nosec G302 -- the case under test
		t.Fatal(err)
	}
	t.Chdir(dir)
	for _, p := range CollectionSearchPaths("", nil) {
		if strings.HasSuffix(p, "evil") {
			t.Fatalf("collections from a world-writable ./ansible.cfg: %q", p)
		}
	}
}
