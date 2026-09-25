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
