package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/config"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
)

// configureFactsCache applies the fact caching settings: onigirazu.yml's
// fact_caching keys, else ansible.cfg's [defaults] fact_caching,
// fact_caching_connection and fact_caching_timeout (seconds); jsonfile keeps
// the facts of every host under the directory across runs
func configureFactsCache(cfg *config.Config, playbookDir string, flush bool) {
	fc := cache.GetGlobalFactsCache()
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
		fc.UseDir(dir)
		if timeout == 0 {
			timeout = 24 * time.Hour
		}
	}
	if timeout > 0 {
		fc.SetTTL(timeout)
	}
	if flush {
		fc.Flush()
	}
}
