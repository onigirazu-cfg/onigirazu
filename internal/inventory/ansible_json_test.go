package inventory

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/internal/parser"
)

func TestAnsibleList(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"hosts.ini": `solo ansible_connection=local
[web]
w1 ansible_host=10.0.0.1 role=front
[db]
d1
[prod:children]
web
db
[prod:vars]
env=prod
ansible_password=secret
[pending]
`})
	p := parser.NewEnhancedParser(nil, &mockLogger{})
	inv, err := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0).
		LoadFromMultipleSources(context.Background(), []string{filepath.Join(dir, "hosts.ini")})
	require.NoError(t, err)
	m := NewManager(p, &mockLogger{}, newMockCache())
	require.NoError(t, m.SetInventory(inv))

	list := m.AnsibleList()
	assert.Equal(t, map[string]interface{}{"hosts": []string{"solo"}}, list["ungrouped"])
	assert.Equal(t, map[string]interface{}{"hosts": []string{"w1"}}, list["web"])
	assert.Equal(t, map[string]interface{}{"children": []string{"db", "web"}}, list["prod"], "hosts of children stay with them")
	assert.Equal(t, map[string]interface{}{}, list["pending"], "an empty group")
	all := list["all"].(map[string]interface{})["children"].([]string)
	assert.Contains(t, all, "prod")
	assert.Contains(t, all, "ungrouped")
	assert.NotContains(t, all, "web")

	hv := list["_meta"].(map[string]interface{})["hostvars"].(map[string]interface{})
	w1 := hv["w1"].(map[string]interface{})
	assert.Equal(t, "10.0.0.1", w1["ansible_host"])
	assert.Equal(t, "front", w1["role"])
	assert.Equal(t, "prod", w1["env"], "group vars resolved")
	assert.NotContains(t, w1, "ansible_password", "no secrets in listings")

	vars, ok := m.AnsibleHostVars("d1")
	require.True(t, ok)
	assert.Equal(t, "prod", vars["env"])
}
