package template

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/expression"
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

// An inline if whose chosen branch is None renders as None (nothing), even
// when the other branch names an undefined variable; Ansible does the same.
func TestRenderInlineIfNone(t *testing.T) {
	e := NewEngine()
	vars := map[string]interface{}{"plat": "windows", "lin": "linux", "n": nil, "d": map[string]interface{}{"rc": 0}}
	ok := map[string]string{
		`{{ undef_x | default({}) if plat == "linux" else none }}`:   "",
		`{{ (undef_x | default({})) if plat == "linux" else none }}`: "",
		`{{ (undef_x | default({}) if plat == "linux" else none) }}`: "",
		`{{ undef_x if plat == "linux" else None }}`:                 "",
		`{{ "x" if plat == "linux" else none }}`:                     "",
		`{{ "x" if plat == "linux" else n }}`:                        "",
		`{{ none if lin == "linux" else undef_x }}`:                  "",
		`{{ d | default({}) if lin == "linux" else none }}`:          "{'rc': 0}",
		`{{ "x" if lin == "linux" else none }}`:                      "x",
		`a{{ "x" if plat == "linux" else none }}b`:                   "ab",
	}
	for tpl, want := range ok {
		got, err := e.Render(context.Background(), tpl, vars)
		if err != nil || got != want {
			t.Errorf("%s: got %q, %v; want %q", tpl, got, err, want)
		}
		args, err := e.RenderTaskArgs(context.Background(), map[string]interface{}{"v": tpl}, vars)
		if err != nil {
			t.Errorf("%s as a task argument: %v", tpl, err)
		} else if want == "" && args["v"] != nil && args["v"] != "" {
			t.Errorf("%s as a task argument: got %#v, want None", tpl, args["v"])
		}
	}
	// the chosen branch is still checked: an undefined name stays an error
	for _, tpl := range []string{
		`{{ undef_x if lin == "linux" else none }}`,
		`{{ none if plat == "linux" else undef_x }}`,
		`{{ (undef_x) if lin == "linux" else none }}`,
	} {
		if got, err := e.Render(context.Background(), tpl, vars); err == nil {
			t.Errorf("%s: an undefined chosen branch must be an error, got %q", tpl, got)
		}
	}
}

// The condition of an inline if runs once, so a filter with side effects or
// a changing result cannot pick a different branch on a second look.
func TestInlineIfConditionEvaluatedOnce(t *testing.T) {
	calls := 0
	vars := expression.WithFilters(map[string]interface{}{"flag": true}, map[string]func(...interface{}) (interface{}, error){
		"flip": func(...interface{}) (interface{}, error) { calls++; return calls%2 == 1, nil },
	})
	if got, err := NewEngine().Render(context.Background(), `{{ missing if flag | flip else none }}`, vars); err == nil {
		t.Errorf("the undefined chosen branch was accepted: %q", got)
	}
	if calls != 1 {
		t.Errorf("condition evaluated %d times, want 1", calls)
	}
}

func TestWrappedInParens(t *testing.T) {
	cases := map[string]string{"(a)": "a", "((a))": "(a)", `(")")`: `")"`, "(a) if (b)": "", "(a": "", "a": ""}
	for in, want := range cases {
		got, ok := wrappedInParens(in)
		if ok != (want != "") || got != want {
			t.Errorf("%q: got %q, %v; want %q", in, got, ok, want)
		}
	}
}
