package inventory

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
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

func TestTemplatedConnectionVars(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"hosts.yml": `all:
  vars:
    ssh_port: 22582
    ansible_port: "{{ ssh_port }}"
    admin: usx
    ansible_user: "{{ admin }}"
  hosts:
    h1: {ansible_host: 10.0.0.1}
    h2: {ansible_host: 10.0.0.2, ssh_port: 22}
`})
	p := parser.NewEnhancedParser(nil, &mockLogger{})
	inv, err := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0).
		LoadFromMultipleSources(context.Background(), []string{filepath.Join(dir, "hosts.yml")})
	require.NoError(t, err)
	m := NewManager(p, &mockLogger{}, newMockCache())
	require.NoError(t, m.SetInventory(inv))
	for name, port := range map[string]int{"h1": 22582, "h2": 22} {
		h, err := m.GetHosts(name)
		require.NoError(t, err)
		assert.Equal(t, port, h[0].Port, name)
		assert.Equal(t, "usx", h[0].User, name)
	}
}

func TestRenderConnectionVarsExtraVars(t *testing.T) {
	for _, tc := range []struct {
		name  string
		extra map[string]interface{}
		want  string
	}{
		{"inventory", nil, "-o ProxyJump=bastion"},
		{"extra", map[string]interface{}{"controller_on_lan": true}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vars := map[string]interface{}{
				"ansible_ssh_common_args": "{{ '' if controller_on_lan | default(false) else '-o ProxyJump=bastion' }}",
				"controller_on_lan":       false,
			}
			got := renderConnectionVars(vars, vars, tc.extra)
			assert.Equal(t, tc.want, got["ansible_ssh_common_args"])
			assert.Equal(t, false, vars["controller_on_lan"])
		})
	}
	vars := map[string]interface{}{"ansible_host": "{{ undefined_var.x }}"}
	got := renderConnectionVars(vars, vars)
	assert.NotContains(t, got, "ansible_host")
}

func TestHostConnectionTemplates(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"hosts.yml": `all:
  vars:
    target_address: 192.0.2.10
    ssh_user: deploy
    ssh_port: 2222
    key_path: /tmp/example-key
    controller_on_lan: false
  hosts:
    target:
      ansible_host: "{{ target_address }}"
      ansible_user: "{{ ssh_user }}"
      ansible_port: "{{ ssh_port }}"
      ansible_ssh_private_key_file: "{{ key_path }}"
      ansible_ssh_common_args: "{{ '' if controller_on_lan else '-o ProxyJump=bastion' }}"
    broken:
      ansible_host: "{{ undefined_var.x }}"
`})
	p := parser.NewEnhancedParser(nil, &mockLogger{})
	inv, err := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0).
		LoadFromMultipleSources(context.Background(), []string{filepath.Join(dir, "hosts.yml")})
	require.NoError(t, err)
	m := NewManager(p, &mockLogger{}, newMockCache())
	require.NoError(t, m.SetInventory(inv))
	extra := map[string]interface{}{"controller_on_lan": true}
	m.SetConnectionOverrides(extra)
	extra["controller_on_lan"] = false // The manager owns its copy.
	hosts, err := m.GetHosts("target")
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, "192.0.2.10", hosts[0].Address)
	assert.Equal(t, "deploy", hosts[0].User)
	assert.Equal(t, 2222, hosts[0].Port)
	assert.Equal(t, "/tmp/example-key", hosts[0].KeyFile)
	assert.Empty(t, hosts[0].SSHArgs)
	hosts, err = m.GetHosts("broken")
	require.NoError(t, err)
	assert.Equal(t, "broken", hosts[0].Address)
}

func TestConnectionArgsOverridesCombine(t *testing.T) {
	for _, tc := range []struct{ name, common, extra, want string }{
		{"empty", "", "", ""},
		{"rendered empty", "{{ '' }}", "", ""},
		{"preserve extra", "", "-o ConnectTimeout=5", "-o ConnectTimeout=5"},
		{"replace common", "-J other.example.com", "-o ConnectTimeout=5", "-J other.example.com -o ConnectTimeout=5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host := &types.Host{Name: "target", Address: "target", Vars: map[string]interface{}{
				"ansible_ssh_common_args": "-J bastion", "ansible_ssh_extra_args": tc.extra,
			}}
			m := NewManager(nil, &mockLogger{}, newMockCache())
			require.NoError(t, m.SetInventory(&types.Inventory{Groups: map[string]*types.Group{
				"all": {Name: "all", Hosts: map[string]*types.Host{"target": host}},
			}}))
			m.SetConnectionOverrides(map[string]interface{}{"ansible_ssh_common_args": tc.common})
			hosts, err := m.GetHosts("target")
			require.NoError(t, err)
			require.Len(t, hosts, 1)
			assert.Equal(t, tc.want, hosts[0].SSHArgs)
			assert.Equal(t, "-J bastion", hosts[0].Vars["ansible_ssh_common_args"])
		})
	}
}

func TestApplyHostOverridesPreservesEffectiveConnectionVars(t *testing.T) {
	host := &types.Host{Name: "target", Address: "target", Vars: map[string]interface{}{
		"ansible_user": "inventory-user", "ansible_ssh_private_key_file": "/tmp/inventory-key",
	}}
	m := NewManager(nil, &mockLogger{}, newMockCache())
	require.NoError(t, m.SetInventory(&types.Inventory{Groups: map[string]*types.Group{
		"all": {Name: "all", Hosts: map[string]*types.Host{"target": host}, Vars: map[string]interface{}{
			"ansible_ssh_user": "group-user", "ansible_private_key_file": "/tmp/group-key",
		}},
	}}))
	m.ApplyHostOverrides("cli-user", "/tmp/cli-key")
	hosts, err := m.GetHosts("target")
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, "cli-user", hosts[0].User)
	assert.Equal(t, "/tmp/cli-key", hosts[0].KeyFile)
	for _, name := range []string{"ansible_user", "ansible_ssh_user"} {
		assert.Equal(t, "cli-user", hosts[0].Vars[name])
	}
	for _, name := range []string{"ansible_ssh_private_key_file", "ansible_private_key_file"} {
		assert.Equal(t, "/tmp/cli-key", hosts[0].Vars[name])
	}
	assert.NotContains(t, hosts[0].Vars, "onigirazu_user")
	assert.NotContains(t, hosts[0].Vars, "onigirazu_ssh_private_key_file")
}
