package inventory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An Ansible inventory script: called with --list, prints groups at the top
// level and host variables under _meta
func TestSources_AnsibleInventoryScript(t *testing.T) {
	script := filepath.Join(t.TempDir(), "inv")
	require.NoError(t, os.WriteFile(script, []byte(`#!/bin/sh
[ "$1" = "--list" ] || { echo '{}'; exit 0; }
cat <<'J'
{"web": {"hosts": ["w1", "w2"], "vars": {"tier": "front"}},
 "db": ["d1"],
 "prod": {"children": ["web", "db"]},
 "_meta": {"hostvars": {"w1": {"ansible_host": "10.0.0.1", "ansible_port": 2200}, "lonely": {}}}}
J
`), 0o700))

	inv := loadSources(t, script)
	// as in Ansible, _meta.hostvars alone does not add a host
	assert.Equal(t, []string{"d1", "w1", "w2"}, hostNames(inv))
	for _, h := range inv.Hosts {
		if h.Name == "w1" {
			assert.Equal(t, "10.0.0.1", h.Address)
			assert.Equal(t, 2200, h.Port)
		}
	}
	require.Contains(t, inv.Groups, "web")
	assert.Len(t, inv.Groups["web"].Hosts, 2)
	assert.Equal(t, "front", inv.Groups["web"].Vars["tier"])
	assert.ElementsMatch(t, []string{"web", "db"}, inv.Groups["prod"].Children)
}
