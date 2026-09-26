package rollback

import (
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
