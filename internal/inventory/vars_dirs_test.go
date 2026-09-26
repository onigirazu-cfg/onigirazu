package inventory

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
}

// Variable precedence, lowest first: group_vars/all, parent groups, child
// groups, the host as written in the inventory (all its copies), host_vars
func TestGroupAndHostVarsFiles(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"hosts.yml": `groups:
  web:
    hosts:
      me:
        inline: from-inventory
        level: host-inline
  db:
    hosts:
      me: {}
      other: {}
  prod:
    children: [db]
`,
		"group_vars/all.yml":   "level: all\nfrom_all: true\n",
		"group_vars/web.yml":   "level: web\nfrom_web: true\n",
		"group_vars/prod.yml":  "tier: prod\nfrom_prod: true\n",
		"group_vars/db/a.yml":  "tier: db-a\n",
		"group_vars/db/b.json": `{"tier": "db-b"}`,
		"host_vars/me.yml":     "inline: from-host-vars\n",
	})

	p := parser.NewEnhancedParser(nil, &mockLogger{})
	inv, err := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0).
		LoadFromMultipleSources(context.Background(), []string{filepath.Join(dir, "hosts.yml")})
	require.NoError(t, err)
	m := NewManager(p, &mockLogger{}, newMockCache())
	require.NoError(t, m.SetInventory(inv))

	resolved := map[string]map[string]interface{}{}
	for _, pattern := range []string{"all", "db", "me"} {
		hosts, err := m.GetHosts(pattern)
		require.NoError(t, err)
		for _, h := range hosts {
			if prev, ok := resolved[h.Name]; ok {
				assert.Equal(t, prev, h.Vars, "same variables whichever pattern selected %s", h.Name)
			}
			resolved[h.Name] = h.Vars
		}
	}

	me := resolved["me"]
	assert.Equal(t, true, me["from_all"])
	assert.Equal(t, true, me["from_web"])
	assert.Equal(t, true, me["from_prod"], "parent group variables reach the child's hosts")
	assert.Equal(t, "db-b", me["tier"], "child group over parent; files of a directory in name order")
	assert.Equal(t, "host-inline", me["level"], "the host over its groups")
	assert.Equal(t, "from-host-vars", me["inline"], "host_vars over the inventory")
	assert.ElementsMatch(t, []string{"db", "prod", "web"}, me["group_names"])

	other := resolved["other"]
	assert.Equal(t, "all", other["level"])
	assert.Nil(t, other["from_web"], "variables of a sibling group do not leak")
}

func TestHostPatterns_UnionAndExclusion(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"hosts.yml": `groups:
  web:
    hosts: {w1: {}, w2: {}}
  db:
    hosts: {d1: {}, w2: {}}
`})
	p := parser.NewEnhancedParser(nil, &mockLogger{})
	inv, err := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0).
		LoadFromMultipleSources(context.Background(), []string{filepath.Join(dir, "hosts.yml")})
	require.NoError(t, err)
	m := NewManager(p, &mockLogger{}, newMockCache())
	require.NoError(t, m.SetInventory(inv))

	names := func(pattern string) []string {
		hosts, err := m.GetHosts(pattern)
		require.NoError(t, err)
		var out []string
		for _, h := range hosts {
			out = append(out, h.Name)
		}
		return out
	}
	assert.ElementsMatch(t, []string{"w1", "w2", "d1"}, names("web,db"))
	assert.ElementsMatch(t, []string{"w1", "w2", "d1"}, names("web:db"))
	assert.ElementsMatch(t, []string{"w1"}, names("web:!db"))
	assert.ElementsMatch(t, []string{"d1", "w1"}, names("all:!w2"))
}

func TestVarsDirNextToPlaybookWins(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{
		"inventories/prod/hosts.yml":          "groups:\n  web:\n    hosts:\n      w1: {}\n",
		"inventories/prod/group_vars/web.yml": "port: 80\nregion: eu\n",
		"playbooks/group_vars/web.yml":        "port: 8080\n",
		"playbooks/host_vars/w1.yml":          "role: primary\n",
	})
	p := parser.NewEnhancedParser(nil, &mockLogger{})
	loader := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0)
	loader.AddVarsDir(filepath.Join(root, "playbooks"))
	inv, err := loader.LoadFromMultipleSources(context.Background(), []string{filepath.Join(root, "inventories/prod/hosts.yml")})
	require.NoError(t, err)
	m := NewManager(p, &mockLogger{}, newMockCache())
	require.NoError(t, m.SetInventory(inv))
	hosts, err := m.GetHosts("all")
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, 8080, hosts[0].Vars["port"], "the playbook's group_vars win")
	assert.Equal(t, "eu", hosts[0].Vars["region"], "the inventory's still apply")
	assert.Equal(t, "primary", hosts[0].Vars["role"])
}
