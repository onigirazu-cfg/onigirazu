package inventory

import (
	"context"
	"path/filepath"
	"sort"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadSources(t *testing.T, sources ...string) *types.Inventory {
	t.Helper()
	p := parser.NewEnhancedParser(nil, &mockLogger{})
	inv, err := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0).
		LoadFromMultipleSources(context.Background(), sources)
	require.NoError(t, err)
	return inv
}

// hostNames lists the hosts of "all", wherever the inventory defines them
func hostNames(inv *types.Inventory) []string {
	m := NewManager(parser.NewEnhancedParser(nil, &mockLogger{}), &mockLogger{}, newMockCache())
	if err := m.SetInventory(inv); err != nil {
		return nil
	}
	hosts, _ := m.GetHosts("all")
	var names []string
	for _, h := range hosts {
		names = append(names, h.Name)
	}
	sort.Strings(names)
	return names
}

func TestSources_HostList(t *testing.T) {
	inv := loadSources(t, "web1,deploy@web2:2222")
	assert.Equal(t, []string{"deploy@web2", "web1"}, hostNames(inv))
	for _, h := range inv.Hosts {
		if h.Address == "web2" {
			assert.Equal(t, 2222, h.Port)
			assert.Equal(t, "deploy", h.User)
		}
	}
	assert.Equal(t, []string{"web1"}, hostNames(loadSources(t, "web1,")))
}

func TestSources_CommaSeparatedFiles(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"a.yml": "groups:\n  a:\n    hosts:\n      ha: {}\n",
		"b.yml": "groups:\n  b:\n    hosts:\n      hb: {}\n",
	})
	inv := loadSources(t, filepath.Join(dir, "a.yml")+","+filepath.Join(dir, "b.yml"))
	assert.Equal(t, []string{"ha", "hb"}, hostNames(inv))
}

func TestSources_PlainListWithUsersAndPorts(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"hosts": "web1\ndeploy@web2:2222\n10.0.0.3:2200\n"})
	inv := loadSources(t, filepath.Join(dir, "hosts"))
	assert.Equal(t, []string{"10.0.0.3", "deploy@web2", "web1"}, hostNames(inv))
}

func TestSources_MissingFile(t *testing.T) {
	p := parser.NewEnhancedParser(nil, &mockLogger{})
	_, err := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0).
		LoadFromMultipleSources(context.Background(), []string{"no-such-inventory"})
	assert.ErrorContains(t, err, "not found")
}
