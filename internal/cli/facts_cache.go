package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/config"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
)

type factsCacheSettings struct {
	kind    string
	dir     string
	timeout time.Duration
}

var runFactsCaches = struct {
	sync.Mutex
	caches map[factsCacheSettings]*cache.FactsCache
}{caches: make(map[factsCacheSettings]*cache.FactsCache)}

// configureFactsCache applies the fact caching settings: onigirazu.yml's
// fact_caching keys, else ansible.cfg's [defaults] fact_caching,
// fact_caching_connection and fact_caching_timeout (seconds); jsonfile keeps
// the facts of every host under the directory across runs
func configureFactsCache(cfg *config.Config, playbookDir string, flush bool) *cache.FactsCache {
	kind, dir, timeout := cfg.FactCaching, cfg.FactCachingDir, cfg.FactCachingTimeout
	if kind == "" {
		kind = parser.AnsibleCfgDefault(playbookDir, "fact_caching")
		if dir == "" {
			dir = parser.AnsibleCfgDefault(playbookDir, "fact_caching_connection")
		}
		if timeout == 0 {
			if s, err := strconv.Atoi(parser.AnsibleCfgDefault(playbookDir, "fact_caching_timeout")); err == nil && s > 0 {
				timeout = time.Duration(s) * time.Second
			}
		}
	}
	if kind == "jsonfile" {
		if dir == "" {
			if home, err := os.UserHomeDir(); err == nil {
				dir = filepath.Join(home, ".onigirazu", "facts")
			}
		}
		if !filepath.IsAbs(dir) && playbookDir != "" {
			dir = filepath.Join(playbookDir, dir)
		}
		if timeout == 0 {
			timeout = 24 * time.Hour
		}
	}
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	if kind != "jsonfile" {
		dir = ""
	}
	settings := factsCacheSettings{kind: kind, dir: dir, timeout: timeout}
	runFactsCaches.Lock()
	fc := runFactsCaches.caches[settings]
	if fc == nil {
		fc = cache.NewFactsCache(timeout)
		fc.UseDir(dir)
		runFactsCaches.caches[settings] = fc
	}
	runFactsCaches.Unlock()
	if flush {
		fc.Flush()
	}
	return fc
}
