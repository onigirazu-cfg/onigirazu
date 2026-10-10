package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// max_concurrency bounds the hosts worked on at once; nested work runs in
// the slot of its host
func TestHostSlot(t *testing.T) {
	e := &ExecutionEngine{hostSlots: make(chan struct{}, 2)}
	var running, peak atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, release := e.hostSlot(context.Background())
			defer release()
			// a block's task on the same host: must not wait for a slot
			_, inner := e.hostSlot(ctx)
			defer inner()
			n := running.Add(1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			running.Add(-1)
		}()
	}
	wg.Wait()
	if p := peak.Load(); p != 2 {
		t.Errorf("%d hosts at once, want 2", p)
	}
	if len(e.hostSlots) != 0 {
		t.Errorf("%d slots not released", len(e.hostSlots))
	}
}
