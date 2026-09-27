package expression

import (
	"sync"

	"github.com/expr-lang/expr"
)

// Filters from filter plugins, usable in expressions as in templates
var (
	extraMu      sync.RWMutex
	extraFilters = map[string]func(params ...interface{}) (interface{}, error){}
)

// RegisterFilter makes a plugin filter available to expressions: x | name(args)
func RegisterFilter(name string, fn func(params ...interface{}) (interface{}, error)) {
	extraMu.Lock()
	defer extraMu.Unlock()
	extraFilters[name] = fn
}

func extraFilterOptions() []expr.Option {
	extraMu.RLock()
	defer extraMu.RUnlock()
	opts := make([]expr.Option, 0, len(extraFilters))
	for name, fn := range extraFilters {
		opts = append(opts, expr.Function(name, fn))
	}
	return opts
}
