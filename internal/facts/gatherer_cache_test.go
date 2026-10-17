package facts

import (
	"context"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestGathererUsesProvidedCache(t *testing.T) {
	for _, name := range []string{"first", "second"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fc := cache.NewFactsCache(time.Hour)
			fc.Set("same-host", &cache.SystemFacts{Hostname: name})
			gatherer := NewGathererWithCache(fc)
			got, err := gatherer.GatherFacts(context.Background(), types.Host{Name: "same-host"})
			if err != nil {
				t.Fatal(err)
			}
			if got.Hostname != name {
				t.Fatalf("facts from another run: %+v", got)
			}
		})
	}
}
