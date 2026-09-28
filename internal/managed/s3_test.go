package managed

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestS3Store runs against a real server: ONIGIRAZU_S3_TEST_ENDPOINT,
// ONIGIRAZU_S3_TEST_BUCKET (existing), ONIGIRAZU_S3_TEST_REGION and the
// AWS_* credentials
func TestS3Store(t *testing.T) {
	endpoint := os.Getenv("ONIGIRAZU_S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("ONIGIRAZU_S3_TEST_ENDPOINT not set")
	}
	ctx := context.Background()
	store, err := NewS3Store(S3Config{Bucket: os.Getenv("ONIGIRAZU_S3_TEST_BUCKET"), Prefix: "test/" + t.Name() + "/",
		Endpoint: endpoint, Region: os.Getenv("ONIGIRAZU_S3_TEST_REGION"), Insecure: true, PathStyle: true}, "/x/site.yml")
	require.NoError(t, err)
	_ = store.client.RemoveObject(ctx, store.bucket, store.key, minio.RemoveObjectOptions{})
	for id := range must(store.lockObjects(ctx)) {
		_ = store.client.RemoveObject(ctx, store.bucket, store.lockPrefix+id, minio.RemoveObjectOptions{})
	}

	st, err := store.Load(ctx)
	require.NoError(t, err)
	assert.Empty(t, st.Resources)
	st.Update(run(true, []string{"p/a"}, nil, task("p/a", fileRes("/a", "absent"))))
	require.NoError(t, store.Save(ctx, st))
	back, err := store.Load(ctx)
	require.NoError(t, err)
	require.Len(t, back.Resources, 1)

	l, err := store.Lock(ctx, "apply", 0)
	require.NoError(t, err)
	_, err = store.Lock(ctx, "apply", 0)
	var locked *LockedError
	require.ErrorAs(t, err, &locked, "a second lock must fail")
	require.NoError(t, l.Release())
	l2, err := store.Lock(ctx, "apply", 0)
	require.NoError(t, err)
	assert.Error(t, store.ForceUnlock(ctx, "wrong"))
	require.NoError(t, l2.Release())

	// many runs at once: exactly one holds the lock at a time
	var held, maxHeld int32
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l, err := store.Lock(ctx, "apply", time.Minute)
			if !assert.NoError(t, err) {
				return
			}
			n := atomic.AddInt32(&held, 1)
			for {
				m := atomic.LoadInt32(&maxHeld)
				if n <= m || atomic.CompareAndSwapInt32(&maxHeld, m, n) {
					break
				}
			}
			time.Sleep(200 * time.Millisecond)
			atomic.AddInt32(&held, -1)
			assert.NoError(t, l.Release())
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(1), maxHeld)
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
