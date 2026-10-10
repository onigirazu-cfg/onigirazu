package cache

import (
	"sync"
	"testing"
	"time"
)

// facts are gathered on all hosts at once: Get and Set run concurrently
func TestFactsCacheConcurrent(t *testing.T) {
	fc := NewFactsCache(time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			host := string(rune('a' + i%5))
			if _, ok := fc.Get(host); !ok {
				fc.Set(host, &SystemFacts{})
			}
		}(i)
	}
	wg.Wait()
	if s := fc.GetStats(); s.Hits+s.Misses != 20 {
		t.Errorf("stats %+v", s)
	}
}
