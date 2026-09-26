package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExtraVars(t *testing.T) {
	file := filepath.Join(t.TempDir(), "vars.yml")
	require.NoError(t, os.WriteFile(file, []byte("region: eu\nreplicas: 3\n"), 0o600))

	vars, err := parseExtraVars([]string{
		"env=prod version=1.2",
		`{"port": 8080, "tags": ["a", "b"]}`,
		"@" + file,
		"env=staging",
	})
	require.NoError(t, err)
	assert.Equal(t, "staging", vars["env"], "later values win")
	assert.Equal(t, "1.2", vars["version"], "key=value gives strings")
	assert.Equal(t, 8080, vars["port"])
	assert.Equal(t, []interface{}{"a", "b"}, vars["tags"])
	assert.Equal(t, "eu", vars["region"])
	assert.Equal(t, 3, vars["replicas"])

	_, err = parseExtraVars([]string{"novalue"})
	assert.Error(t, err)
	_, err = parseExtraVars([]string{"@/does/not/exist"})
	assert.Error(t, err)
}
