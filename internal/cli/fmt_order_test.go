package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatFile_KeepsOrderCommentsAndPutsNameFirst(t *testing.T) {
	p := filepath.Join(t.TempDir(), "site.yml")
	require.NoError(t, os.WriteFile(p, []byte("plays:\n  - hosts: all\n    name: p   # the play\n    tasks:\n    - debug: {msg: x}\n      when: y\n      name: t\n"), 0o644))
	_, err := formatFile(p, 2, false, false, true)
	require.NoError(t, err)
	out, _ := os.ReadFile(p)
	assert.Equal(t, "plays:\n  - name: p # the play\n    hosts: all\n    tasks:\n      - name: t\n        debug: {msg: x}\n        when: y\n", string(out))
}
