package template

import (
	"context"
	"testing"
)

func TestRenderNone(t *testing.T) {
	e := NewEngine()
	vars := map[string]interface{}{
		"n":    nil,
		"d":    map[string]interface{}{"k": nil, "l": []interface{}{nil}},
		"name": "x",
	}
	ok := map[string]string{
		"a={{ n }}b":             "a=b",
		"{{ None }}":             "",
		"{{ d.k }}|{{ d['k'] }}": "|",
		"{{ d.l[0] }}":           "",
		"{{ n is none }}":        "True",
		"{{ n is defined }}":     "True",
		"{{ name }}{{ n }}":      "x",
	}
	for tpl, want := range ok {
		got, err := e.Render(context.Background(), tpl, vars)
		if err != nil || got != want {
			t.Errorf("%s: got %q, %v; want %q", tpl, got, err, want)
		}
	}
	for _, tpl := range []string{"{{ missing }}", "{{ d.zz }}", "{{ d.l[3] }}"} {
		if got, err := e.Render(context.Background(), tpl, vars); err == nil {
			t.Errorf("%s: an undefined name must be an error, got %q", tpl, got)
		}
	}
}
