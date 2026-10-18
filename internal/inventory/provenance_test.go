package inventory

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func provenanceLoader() *MultiSourceLoader {
	p := parser.NewEnhancedParser(nil, &mockLogger{})
	p.PreserveInventoryStructure()
	return NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0)
}

func TestInventorySourcesLastOccurrenceAndReset(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"first.yml": `groups:
  web:
    hosts:
      app:
        shared: first
        retained: first
    vars:
      shared_group: first
      retained_group: first
`,
		"second.yml": `groups:
  web:
    hosts:
      app:
        shared: second
    vars:
      shared_group: second
`,
	})
	first, second := filepath.Join(dir, "first.yml"), filepath.Join(dir, "second.yml")
	loader := provenanceLoader().TrackSources()
	_, err := loader.LoadFromMultipleSources(context.Background(), []string{first, second})
	require.NoError(t, err)
	assert.Equal(t, second, loader.Sources().Hosts["app"]["shared"])
	assert.Equal(t, first, loader.Sources().Hosts["app"]["retained"])
	assert.Equal(t, map[string]string{"shared_group": second, "retained_group": first}, loader.Sources().Groups["web"])

	_, err = loader.LoadFromMultipleSources(context.Background(), []string{second})
	require.NoError(t, err)
	assert.Equal(t, second, loader.Sources().Hosts["app"]["shared"])
	assert.NotContains(t, loader.Sources().Hosts["app"], "retained")
	assert.Equal(t, map[string]string{"shared_group": second}, loader.Sources().Groups["web"])
}

func TestInventorySourcesVarsDirectories(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"hosts.yml": `groups:
  web:
    hosts:
      app:
        level: inventory
    vars:
      tier: inventory
`,
		"group_vars/all.yml":    "environment: production\n",
		"group_vars/web/a.yml":  "tier: first\nretained: first\n",
		"group_vars/web/b.json": `{"tier":"second"}`,
		"host_vars/app/a.yml":   "level: first\nhost_only: first\n",
		"host_vars/app/b.yml":   "level: second\n",
	})
	loader := provenanceLoader().TrackSources()
	inv, err := loader.LoadFromMultipleSources(context.Background(), []string{filepath.Join(dir, "hosts.yml")})
	require.NoError(t, err)
	sources := loader.Sources()
	assert.Equal(t, filepath.Join(dir, "group_vars/all.yml"), sources.Groups["all"]["environment"])
	assert.Equal(t, filepath.Join(dir, "group_vars/web/a.yml"), sources.Groups["web"]["retained"])
	assert.Equal(t, filepath.Join(dir, "group_vars/web/b.json"), sources.Groups["web"]["tier"])
	assert.Equal(t, filepath.Join(dir, "host_vars/app/a.yml"), sources.Hosts["app"]["host_only"])
	assert.Equal(t, filepath.Join(dir, "host_vars/app/b.yml"), sources.Hosts["app"]["level"])
	entries, err := EffectiveProvenance(inv, "app", sources)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Name == "level" {
			assert.Equal(t, "host", entry.From)
			assert.Equal(t, sources.Hosts["app"]["level"], entry.Source)
		}
	}
}

func TestInventorySourcesDynamicCache(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires a shell script")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.sh")
	script := "#!/bin/sh\ncat <<'JSON'\n" + `{"web":{"hosts":["app"],"vars":{"tier":"web"}},"_meta":{"hostvars":{"app":{"secret":"hidden"}}}}` + "\nJSON\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o700))
	loader := provenanceLoader().TrackSources()
	for i := 0; i < 2; i++ {
		_, err := loader.LoadFromMultipleSources(context.Background(), []string{path})
		require.NoError(t, err)
		assert.Equal(t, path, loader.Sources().Hosts["app"]["secret"])
		assert.Equal(t, path, loader.Sources().Groups["web"]["tier"])
		// The second load must use the cached inventory rather than execute again.
		require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\nexit 1\n"), 0o700))
	}
}

func TestInventorySourcesTrackingOff(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"hosts.yml": "groups:\n  web:\n    hosts:\n      app: {secret: hidden}\n    vars: {tier: web}\n"})
	loader := provenanceLoader()
	inv, err := loader.LoadFromMultipleSources(context.Background(), []string{filepath.Join(dir, "hosts.yml")})
	require.NoError(t, err)
	assert.Nil(t, loader.Sources().Hosts)
	assert.Nil(t, loader.Sources().Groups)
	assert.Zero(t, testing.AllocsPerRun(100, func() { _ = loader.Sources() }))
	assert.Nil(t, loader.sources.hostDefinitions)
	assert.Nil(t, loader.sources.hostVariables)
	assert.Nil(t, loader.varsSources.Hosts)
	baseline := testing.AllocsPerRun(100, func() { loader.mergeInventory(inv) })
	untracked := testing.AllocsPerRun(100, func() { loader.mergeInventorySource(inv, "unused") })
	assert.Equal(t, baseline, untracked, "disabled tracking adds no merge allocations")
}

func TestEffectiveProvenanceMatchesEngine(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"hosts.yml": `groups:
  parent:
    children: [child]
    vars:
      inherited: parent
      host_wins: parent
  child:
    hosts:
      app:
        host_wins: host
    vars:
      inherited: child
      host_wins: child
      child_only: child
  alpha:
    hosts:
      app: {}
    vars:
      sibling: alpha
  zeta:
    hosts:
      app: {}
    vars:
      sibling: zeta
`})
	path := filepath.Join(dir, "hosts.yml")
	loader := provenanceLoader().TrackSources()
	inv, err := loader.LoadFromMultipleSources(context.Background(), []string{path})
	require.NoError(t, err)
	entries, err := EffectiveProvenance(inv, "app", loader.Sources())
	require.NoError(t, err)
	byName := make(map[string]VariableProvenance, len(entries))
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		byName[entry.Name] = entry
		names = append(names, entry.Name)
	}
	assert.True(t, sort.StringsAreSorted(names))
	assert.Equal(t, "host", byName["host_wins"].From)
	assert.Equal(t, "group:child", byName["inherited"].From)
	assert.Equal(t, "group:child", byName["child_only"].From)

	p := parser.NewEnhancedParser(nil, &mockLogger{})
	engineInventory, err := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0).
		LoadFromMultipleSources(context.Background(), []string{path})
	require.NoError(t, err)
	manager := NewManager(p, &mockLogger{}, newMockCache())
	require.NoError(t, manager.SetInventory(engineInventory))
	hosts, err := manager.GetHosts("app")
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	assert.Equal(t, "host", hosts[0].Vars["host_wins"])
	assert.Equal(t, "group:"+hosts[0].Vars["inherited"].(string), byName["inherited"].From)
	assert.Equal(t, "group:"+hosts[0].Vars["sibling"].(string), byName["sibling"].From)
	for _, name := range []string{"host_wins", "inherited", "child_only", "sibling"} {
		assert.Equal(t, path, byName[name].Source, name)
	}
}

func TestEffectiveProvenanceConnectionVariablesMatchEngine(t *testing.T) {
	for _, tc := range []struct {
		name, content string
	}{
		{"hosts.ini", "[web]\napp ansible_host=192.0.2.1 ansible_port=2222 ansible_user=deploy ansible_password=hidden custom=present\n"},
		{"hosts.yml", `all:
  children:
    web:
      hosts:
        app:
          ansible_host: 192.0.2.1
          ansible_port: 2222
          ansible_user: deploy
          ansible_password: hidden
          custom: present
`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{tc.name: tc.content})
			path := filepath.Join(dir, tc.name)
			loader := provenanceLoader().TrackSources()
			inv, err := loader.LoadFromMultipleSources(context.Background(), []string{path})
			require.NoError(t, err)
			entries, err := EffectiveProvenance(inv, "app", loader.Sources())
			require.NoError(t, err)
			origins := make(map[string]VariableProvenance, len(entries))
			for _, entry := range entries {
				origins[entry.Name] = entry
			}

			p := parser.NewEnhancedParser(nil, &mockLogger{})
			engineInventory, err := NewMultiSourceLoader(p, &mockLogger{}, newMockCache(), 0).
				LoadFromMultipleSources(context.Background(), []string{path})
			require.NoError(t, err)
			manager := NewManager(p, &mockLogger{}, newMockCache())
			require.NoError(t, manager.SetInventory(engineInventory))
			hosts, err := manager.GetHosts("app")
			require.NoError(t, err)
			require.Len(t, hosts, 1)
			for _, name := range []string{"ansible_host", "ansible_port", "ansible_user", "ansible_password", "custom"} {
				_, engineHas := hosts[0].Vars[name]
				origin, provenanceHas := origins[name]
				assert.Equal(t, engineHas, provenanceHas, name)
				if provenanceHas {
					assert.Equal(t, "host", origin.From, name)
					assert.Equal(t, path, origin.Source, name)
				}
			}
		})
	}
}

func TestEffectiveProvenanceSharedHostVariables(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"first.yml":  "all:\n  children:\n    alpha:\n      hosts:\n        app: {shared: first}\n",
		"second.yml": "all:\n  children:\n    zeta:\n      hosts:\n        app: {shared: second}\n",
	})
	first, second := filepath.Join(dir, "first.yml"), filepath.Join(dir, "second.yml")
	loader := provenanceLoader().TrackSources()
	inv, err := loader.LoadFromMultipleSources(context.Background(), []string{first, second})
	require.NoError(t, err)
	// The loader merges into the first host's map, shared with alpha's entry.
	assert.Equal(t, "second", inv.Groups["alpha"].Hosts["app"].Vars["shared"])
	entries, err := EffectiveProvenance(inv, "app", loader.Sources())
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, VariableProvenance{Name: "shared", From: "host", Source: second}, entries[0])
}

func TestEffectiveProvenanceAmbiguousHostDefinitions(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"first.yml":  "hosts:\n  - name: app\n    address: app\ngroups:\n  alpha:\n    hosts:\n      app:\n        vars: {choice: first}\n",
		"second.yml": "hosts:\n  - name: app\n    address: app\ngroups:\n  zeta:\n    hosts:\n      app:\n        vars: {choice: second}\n",
	})
	paths := []string{filepath.Join(dir, "first.yml"), filepath.Join(dir, "second.yml")}
	loader := provenanceLoader().TrackSources()
	inv, err := loader.LoadFromMultipleSources(context.Background(), paths)
	require.NoError(t, err)
	entries, err := EffectiveProvenance(inv, "app", loader.Sources())
	require.ErrorContains(t, err, "ambiguous variable origins")
	assert.Nil(t, entries)
	// This inventory really does resolve differently with the selected group.
	manager := setupTestManager(t)
	require.NoError(t, manager.SetInventory(inv))
	alpha, err := manager.GetHosts("alpha")
	require.NoError(t, err)
	zeta, err := manager.GetHosts("zeta")
	require.NoError(t, err)
	assert.Equal(t, "first", alpha[0].Vars["choice"])
	assert.Equal(t, "second", zeta[0].Vars["choice"])
}

func TestEffectiveProvenanceNullHostVariable(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"hosts.yml": "groups:\n  web:\n    hosts:\n      app:\n        vars:\n          ansible_password: null\n",
	})
	loader := provenanceLoader().TrackSources()
	path := filepath.Join(dir, "hosts.yml")
	inv, err := loader.LoadFromMultipleSources(context.Background(), []string{path})
	require.NoError(t, err)
	entries, err := EffectiveProvenance(inv, "app", loader.Sources())
	require.NoError(t, err)
	assert.Contains(t, entries, VariableProvenance{Name: "ansible_password", From: "host", Source: path})
}
