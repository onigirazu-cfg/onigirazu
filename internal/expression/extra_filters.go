package expression

import (
	"fmt"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
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

const scopedFiltersKey = "__onigirazu_expression_filters"

type scopedFilters struct {
	functions map[string]func(...interface{}) (interface{}, error)
}

// WithFilters copies variables and attaches this engine's plugin filters.
func WithFilters(variables map[string]interface{}, filters map[string]func(...interface{}) (interface{}, error)) map[string]interface{} {
	out := make(map[string]interface{}, len(variables))
	for k, v := range variables {
		out[k] = v
	}
	if len(filters) != 0 {
		out[scopedFiltersKey] = scopedFilters{functions: filters}
	}
	return out
}

func evaluationFilters(variables map[string]interface{}) map[string]func(...interface{}) (interface{}, error) {
	filters, _ := variables[scopedFiltersKey].(scopedFilters)
	return filters.functions
}

type scopedFilterPatch struct {
	names map[string]func(...interface{}) (interface{}, error)
}

func (p scopedFilterPatch) Visit(node *ast.Node) {
	call, ok := (*node).(*ast.CallNode)
	if !ok {
		return
	}
	name, ok := call.Callee.(*ast.IdentifierNode)
	if !ok || p.names[name.Value] == nil {
		return
	}
	ast.Patch(node, &ast.CallNode{
		Callee:    &ast.IdentifierNode{Value: "jinja_scoped_filter"},
		Arguments: append([]ast.Node{&ast.IdentifierNode{Value: lookupVars}, &ast.StringNode{Value: name.Value}}, call.Arguments...),
	})
}

func callScopedFilter(params ...interface{}) (interface{}, error) {
	variables, _ := params[0].(map[string]interface{})
	name, _ := params[1].(string)
	fn := evaluationFilters(variables)[name]
	if fn == nil {
		return nil, fmt.Errorf("filter %q is not available here", name)
	}
	return fn(params[2:]...)
}
