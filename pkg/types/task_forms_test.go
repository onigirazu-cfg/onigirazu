package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestTask_NotifyAsStringOrList(t *testing.T) {
	var one, many Task
	require.NoError(t, yaml.Unmarshal([]byte("name: a\ncopy: {dest: /x}\nnotify: restart app\n"), &one))
	require.NoError(t, yaml.Unmarshal([]byte("name: b\ncopy: {dest: /x}\nnotify: [a, b]\n"), &many))
	assert.Equal(t, []string{"restart app"}, one.Notify)
	assert.Equal(t, []string{"a", "b"}, many.Notify)
}

func TestRoleReference_AnsibleForms(t *testing.T) {
	var roles []RoleReference
	require.NoError(t, yaml.Unmarshal([]byte(`
- common
- role: web
  port: 8080
- name: db
  vars: {engine: pg}
`), &roles))
	require.Len(t, roles, 3)
	assert.Equal(t, "common", roles[0].Name)
	assert.Equal(t, "web", roles[1].Name)
	assert.Equal(t, 8080, roles[1].Vars["port"])
	assert.Equal(t, "db", roles[2].Name)
	assert.Equal(t, "pg", roles[2].Vars["engine"])
}

func TestPlay_HostsStringOrList(t *testing.T) {
	var s, l Play
	require.NoError(t, yaml.Unmarshal([]byte("name: a\nhosts: web\ntasks: []\n"), &s))
	require.NoError(t, yaml.Unmarshal([]byte("name: b\nhosts: [web, db]\ntasks: []\n"), &l))
	assert.Equal(t, "web", s.Hosts)
	assert.Equal(t, "web,db", l.Hosts)
}

func TestTask_WithDict(t *testing.T) {
	var lit, expr Task
	require.NoError(t, yaml.Unmarshal([]byte("name: a\ndebug: {msg: x}\nwith_dict: {b: 2, a: 1}\n"), &lit))
	require.NoError(t, yaml.Unmarshal([]byte("name: b\ndebug: {msg: x}\nwith_dict: \"{{ users }}\"\n"), &expr))
	require.NotNil(t, lit.Loop)
	assert.Equal(t, []interface{}{
		map[string]interface{}{"key": "a", "value": 1},
		map[string]interface{}{"key": "b", "value": 2},
	}, lit.Loop.Items)
	require.NotNil(t, expr.Loop)
	assert.Equal(t, "(users) | dict2items", expr.Loop.Expr)
}
