package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVarsFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"site.yml": `plays:
  - name: p
    hosts: all
    vars: {port: 80, name: web}
    vars_files: [vars/common.yml, vars/prod.yml]
    tasks:
      - name: t
        debug: {msg: x}
`,
		"vars/common.yml": "port: 8080\nregion: eu\n",
		"vars/prod.yml":   "region: eu-west\n",
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	p := NewEnhancedParser(&mockTemplateEngine{}, &mockLogger{})
	pb, err := p.ParsePlaybook(context.Background(), filepath.Join(dir, "site.yml"))
	require.NoError(t, err)
	vars := pb.Plays[0].Vars
	assert.Equal(t, 8080, vars["port"], "vars_files over vars")
	assert.Equal(t, "web", vars["name"])
	assert.Equal(t, "eu-west", vars["region"], "later files win")

	require.NoError(t, os.Remove(filepath.Join(dir, "vars/prod.yml")))
	_, err = p.ParsePlaybook(context.Background(), filepath.Join(dir, "site.yml"))
	assert.Error(t, err, "a missing vars file is an error")
}
