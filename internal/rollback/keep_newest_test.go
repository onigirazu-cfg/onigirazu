package rollback

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeepNewest(t *testing.T) {
	sm := NewSnapshotManager(t.TempDir())
	base := time.Now()
	for i := 0; i < 5; i++ {
		s, err := sm.CreateSnapshot("pb", "d")
		require.NoError(t, err)
		s.ID = string(rune('a' + i))
		s.Timestamp = base.Add(time.Duration(i) * time.Minute)
		require.NoError(t, sm.SaveSnapshot(s))
		// pruning goes by the files' times
		p := filepath.Join(sm.snapshotDir, "snapshot_"+s.ID+".json")
		require.NoError(t, os.Chtimes(p, s.Timestamp, s.Timestamp))
	}
	require.NoError(t, sm.KeepNewest(2))
	left, err := sm.ListSnapshots()
	require.NoError(t, err)
	var ids []string
	for _, s := range left {
		ids = append(ids, s.ID)
	}
	assert.ElementsMatch(t, []string{"d", "e"}, ids)
}
