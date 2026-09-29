package inventory

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/stretchr/testify/require"
)

func TestInsecureExampleInventory(t *testing.T) {
	p := parser.NewEnhancedParser(nil, &mockLogger{})
	inv, err := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0).
		LoadFromMultipleSources(context.Background(), []string{"../../docs/examples/inventory_with_insecure_host_key.yml"})
	require.NoError(t, err)
	m := NewManager(p, &mockLogger{}, newMockCache())
	require.NoError(t, m.SetInventory(inv))
	want := map[string]bool{"dev-server-01": true, "dev-server-02": true, "prod-server-01": false}
	for name, insecure := range want {
		h, err := m.GetHosts(name)
		require.NoError(t, err)
		require.Len(t, h, 1)
		require.Equal(t, insecure, h[0].InsecureIgnoreHostKey, name)
	}
}
