package expression

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// hosts evaluate in parallel: the filter tables must be safe to build and
// use from many goroutines at once
func TestEvalConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, e := range []string{`'a' | sha256`, `[1, 2] | product([3]) | list | length`, `'x' | type_debug`, `'a,b' | split(',') | length`} {
				_, err := Eval(e, map[string]interface{}{})
				require.NoError(t, err, e)
			}
		}()
	}
	wg.Wait()
}
