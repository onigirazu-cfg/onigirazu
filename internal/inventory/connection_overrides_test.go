package inventory

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/internal/parser"
)

func TestConnectionVars(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"hosts.yml": `all:
  vars:
    ansible_user: usx
    ansible_ssh_common_args: "-o StrictHostKeyChecking=no -o PreferredAuthentications=publickey"
  children:
    fleet:
      vars:
        ansible_become_password: from-group
      hosts:
        h1: {ansible_host: 10.0.0.1}
        h2: {ansible_host: 10.0.0.2, ansible_become_password: from-host}
`})
	p := parser.NewEnhancedParser(nil, &mockLogger{})
	inv, err := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0).
		LoadFromMultipleSources(context.Background(), []string{filepath.Join(dir, "hosts.yml")})
	require.NoError(t, err)
	m := NewManager(p, &mockLogger{}, newMockCache())
	require.NoError(t, m.SetInventory(inv))

	host := func(name string) (user, pass, become string, insecure bool, port int) {
		h, err := m.GetHosts(name)
		require.NoError(t, err)
		require.Len(t, h, 1)
		return h[0].User, h[0].Password, h[0].BecomePassword, h[0].InsecureIgnoreHostKey, h[0].Port
	}
	user, _, become, insecure, _ := host("h1")
	assert.Equal(t, "usx", user)
	assert.Equal(t, "from-group", become)
	assert.True(t, insecure, "StrictHostKeyChecking=no in ansible_ssh_common_args")
	_, _, become, _, _ = host("h2")
	assert.Equal(t, "from-host", become)

	// -e wins over the inventory, as in Ansible
	m.SetConnectionOverrides(map[string]interface{}{"ansible_user": "ansible", "ansible_password": "pw",
		"ansible_become_password": "bp", "ansible_port": 2222, "other": "x"})
	user, pass, become, _, port := host("h2")
	assert.Equal(t, "ansible", user)
	assert.Equal(t, "pw", pass)
	assert.Equal(t, "bp", become)
	assert.Equal(t, 2222, port)
}
