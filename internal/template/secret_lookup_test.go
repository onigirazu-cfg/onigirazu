package template

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/expression"
	"github.com/onigirazu-cfg/onigirazu/internal/plugins"
)

func TestEngineSecretLookupIsolation(t *testing.T) {
	for _, name := range []string{"first", "second", "third"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			e := NewEngine()
			defer e.Close()
			e.SetSecretLookup(func(string, string, string) (string, error) { return name, nil })
			for range 20 {
				got, err := e.Render(context.Background(), `{{ bitwarden('item') }} {{ lookup('bitwarden', 'item') }}`, nil)
				if err != nil || got != name+" "+name {
					t.Fatalf("got %q, %v; want own resolver", got, err)
				}
			}
		})
	}
}

func TestNilEngineRenderTaskArgsPlainValues(t *testing.T) {
	var e *Engine
	got, err := e.RenderTaskArgs(context.Background(), map[string]interface{}{"name": "plain", "enabled": true}, nil)
	if err != nil || got["name"] != "plain" || got["enabled"] != true {
		t.Fatalf("got %v, %v; want unchanged plain arguments", got, err)
	}
}

func TestEnginePluginFilterIsolation(t *testing.T) {
	for _, name := range []string{"first", "second", "third"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			manager := plugins.NewManager(plugins.NewInMemoryLoader())
			plugin := plugins.NewBaseFilterPlugin("custom", "1", "")
			plugin.AddFilter("run_value", func(interface{}, ...interface{}) (interface{}, error) { return name, nil })
			if err := manager.Register(context.Background(), plugin); err != nil {
				t.Fatal(err)
			}
			e := NewEngineWithPlugins(manager)
			defer e.Close()
			for range 20 {
				got, err := e.Render(context.Background(), `{{ 'value' | run_value }}`, nil)
				if err != nil || got != name {
					t.Fatalf("got %q, %v; want %s", got, err, name)
				}
				gotExpression, err := expression.Eval(`'value' | run_value`, e.ExpressionVariables(nil))
				if err != nil || gotExpression != name {
					t.Fatalf("expression got %v, %v; want %s", gotExpression, err, name)
				}
			}
			plain := NewEngine()
			defer plain.Close()
			if _, err := expression.Eval(`'value' | run_value`, plain.ExpressionVariables(nil)); err == nil {
				t.Fatal("filter leaked into a plain engine")
			}
		})
	}
}
