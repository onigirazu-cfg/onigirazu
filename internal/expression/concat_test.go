package expression

import "testing"

func TestNestedConcat(t *testing.T) {
	vars := map[string]interface{}{
		"v":   "1.8.2",
		"out": map[string]interface{}{"stdout": "node_exporter, version 1.8.2 (branch", "stderr": ""},
		"n":   3,
	}
	cases := map[string]bool{
		"('version ' ~ v ~ ' ') in (out.stdout ~ out.stderr)":     true,
		"('version ' ~ v ~ ' ') not in (out.stdout ~ out.stderr)": false,
		"('x' ~ n) == 'x3'":            true,
		"['a' ~ n, 'b'] | length == 2": true,
		"'a ~ b' == 'a ~ b'":           true,
		"('a(' ~ '~') == 'a(~'":        true,
		"(['p' ~ n] | first) == 'p3'":  true,
	}
	for cond, want := range cases {
		got, err := Condition(cond, vars)
		if err != nil {
			t.Errorf("%s: %v", cond, err)
			continue
		}
		if got != want {
			t.Errorf("%s = %v, want %v", cond, got, want)
		}
	}
}

func TestInlineIfInsideBrackets(t *testing.T) {
	vars := map[string]interface{}{"auth": false, "out": "passwordauthentication no\n", "n": 2}
	cases := map[string]bool{
		"('passwordauthentication ' ~ ('yes' if auth | bool else 'no')) in out": true,
		"('yes' if auth else 'no') == 'no'":                                     true,
		"['a' if n > 1 else 'b'][0] == 'a'":                                     true,
		"(n if n > 5 else 0) == 0":                                              true,
		"('x' if auth) == ''":                                                   true,
	}
	for cond, want := range cases {
		got, err := Condition(cond, vars)
		if err != nil || got != want {
			t.Errorf("%s = %v, %v; want %v", cond, got, err, want)
		}
	}
}
