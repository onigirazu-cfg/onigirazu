package modules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// one probe for the loop; the items read their capture from it until one
// of them changes something
func TestPrefetchLoop(t *testing.T) {
	dir := t.TempDir()
	a, b, gone := filepath.Join(dir, "a b"), filepath.Join(dir, "it's"), filepath.Join(dir, "none")
	require.NoError(t, os.WriteFile(a, []byte("A\n"), 0o640))
	require.NoError(t, os.Mkdir(b, 0o755))
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}

	lp := PrefetchLoop(context.Background(), host, &types.Task{Module: "copy"}, []string{a, b, gone, a})
	require.NotNil(t, lp)
	ctx := WithLoopProbes(context.Background(), lp)

	// the files change after the prefetch: the items still see the capture
	require.NoError(t, os.Remove(a))
	before := captureBefore(ctx, host, "copy", map[string]interface{}{"dest": a})
	assert.Equal(t, "file", before["kind"])
	assert.Equal(t, "A\n", before["content"])
	assert.Equal(t, "0640", before["mode"])
	assert.Equal(t, "directory", captureBefore(ctx, host, "file", map[string]interface{}{"path": b})["kind"])
	assert.Equal(t, "absent", captureBefore(ctx, host, "file", map[string]interface{}{"path": gone})["kind"])

	// an item changed something: the next ones probe for themselves
	invalidateLoopProbes(ctx)
	assert.Equal(t, "absent", captureBefore(ctx, host, "copy", map[string]interface{}{"dest": a})["kind"])

	// fewer than two distinct paths: nothing to batch
	assert.Nil(t, PrefetchLoop(context.Background(), host, &types.Task{Module: "copy"}, []string{a, a}))
}
