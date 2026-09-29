package inventory

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/stretchr/testify/require"
)

func TestHostLevelSSHArgsAnsibleYAML(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "h.yml")
	require.NoError(t, os.WriteFile(f, []byte("all:\n  hosts:\n    web1:\n      ansible_host: 10.0.0.5\n      ansible_ssh_common_args: \"-o StrictHostKeyChecking=no\"\n      ansible_ssh_extra_args: \"-J bastion\"\n    web2:\n      ansible_host: 10.0.0.6\n"), 0o600))
	p := parser.NewEnhancedParser(nil, &mockLogger{})
	inv, err := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0).LoadFromMultipleSources(context.Background(), []string{f})
	require.NoError(t, err)
	m := NewManager(p, &mockLogger{}, newMockCache())
	require.NoError(t, m.SetInventory(inv))
	h1, _ := m.GetHosts("web1")
	h2, _ := m.GetHosts("web2")
	require.True(t, h1[0].InsecureIgnoreHostKey)
	require.False(t, h2[0].InsecureIgnoreHostKey)
	require.Equal(t, "-o StrictHostKeyChecking=no -J bastion", h1[0].SSHArgs)
	require.Empty(t, h2[0].SSHArgs)
	all, _ := m.GetHosts("all")
	require.Len(t, all, 2)
}
