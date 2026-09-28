package galaxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "r.yml")
	require.NoError(t, os.WriteFile(p, []byte(`collections:
  - name: community.general
  - name: git@github.com:ClanRed/infra-ansible.git
    type: git
    version: v1.0.2
roles:
  - staticdev.pyenv
  - src: https://example.org/role.git
    name: myrole
`), 0o644))
	req, err := Load(p)
	require.NoError(t, err)
	require.Len(t, req.Collections, 2)
	assert.Equal(t, "", gitSource(req.Collections[0]))
	assert.Equal(t, "git@github.com:ClanRed/infra-ansible.git", gitSource(req.Collections[1]))
	require.Len(t, req.Roles, 2)
	assert.Equal(t, "staticdev.pyenv", req.Roles[0].Name)
	assert.Equal(t, "https://example.org/role.git", gitSource(req.Roles[1]))

	require.NoError(t, os.WriteFile(p, []byte("- src: git+https://example.org/x.git\n"), 0o644))
	req, err = Load(p)
	require.NoError(t, err)
	assert.Equal(t, "https://example.org/x.git", gitSource(req.Roles[0]))
}

func TestNewerVersion(t *testing.T) {
	assert.True(t, newerVersion("v1.2.10", "v1.2.9"))
	assert.True(t, newerVersion("3.0.0", "2.9.9"))
	assert.False(t, newerVersion("1.0", "1.0.1"))
}

func gitRepo(t *testing.T, files map[string]string, tag string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "x"}, {"tag", tag}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	return "file://" + dir
}

func TestInstallGitCollectionAndRole(t *testing.T) {
	coll := gitRepo(t, map[string]string{"galaxy.yml": "namespace: clanred\nname: infra\n", "roles/users/tasks/main.yml": "- debug: {msg: hi}\n"}, "v1.0.2")
	role := gitRepo(t, map[string]string{"tasks/main.yml": "- debug: {msg: role}\n"}, "1.0")
	out := t.TempDir()
	req := &Requirements{
		Collections: []Requirement{{Name: coll, Type: "git", Version: "v1.0.2"}, {Name: "ansible.posix"}},
		Roles:       []Requirement{{Src: role, Name: "mine", Version: "1.0"}},
	}
	opts := Options{RolesPath: filepath.Join(out, "roles"), CollectionsPath: filepath.Join(out, "collections")}
	require.NoError(t, Install(context.Background(), req, opts))
	assert.FileExists(t, filepath.Join(out, "collections", "ansible_collections", "clanred", "infra", "roles", "users", "tasks", "main.yml"))
	assert.FileExists(t, filepath.Join(out, "roles", "mine", "tasks", "main.yml"))
	assert.NoDirExists(t, filepath.Join(out, "roles", "mine", ".git"))
	// the same version again: kept as it is
	require.NoError(t, Install(context.Background(), req, opts))
}

func TestGalaxyRoleRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "staticdev", r.URL.Query().Get("owner__username"))
		_, _ = w.Write([]byte(`{"results": [{"github_user": "staticdev", "github_repo": "ansible-role-pyenv",
			"summary_fields": {"versions": [{"name": "2.9.0"}, {"name": "3.0.0"}, {"name": "2.10.1"}]}}]}`))
	}))
	defer srv.Close()
	repo, latest, err := galaxyRoleRepo(context.Background(), srv.URL, "staticdev.pyenv")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/staticdev/ansible-role-pyenv.git", repo)
	assert.Equal(t, "3.0.0", latest)
}
