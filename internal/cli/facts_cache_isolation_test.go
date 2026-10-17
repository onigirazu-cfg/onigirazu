package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/config"
)

func TestFactsCacheSettingsIsolation(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{FactCaching: "jsonfile", FactCachingDir: "first", FactCachingTimeout: time.Hour}
	first := configureFactsCache(cfg, dir, false)
	first.Set("host", &cache.SystemFacts{Hostname: "first"})
	if got := configureFactsCache(cfg, dir, false); got != first {
		t.Fatal("identical settings must reuse the cache")
	}
	cfg.FactCachingDir = "second"
	second := configureFactsCache(cfg, dir, true)
	if second == first {
		t.Fatal("different directories must have independent caches")
	}
	if _, found := second.Get("host"); found {
		t.Fatal("facts escaped their cache directory")
	}
	if got, found := first.Get("host"); !found || got.Hostname != "first" {
		t.Fatal("flushing another cache removed facts")
	}
	if first.GetTTL() != time.Hour {
		t.Fatal("unexpected first cache TTL")
	}
	cfg.FactCachingDir = "first"
	cfg.FactCachingTimeout = time.Minute
	third := configureFactsCache(cfg, dir, false)
	if third == first || first.GetTTL() != time.Hour {
		t.Fatal("different timeout reconfigured an existing cache")
	}
	if _, err := os.Stat(filepath.Join(dir, "first", "host.json")); err != nil {
		t.Fatal(err)
	}
}
