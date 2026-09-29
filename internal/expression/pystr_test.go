package expression

import "testing"

func TestPyStr(t *testing.T) {
	cases := []struct {
		in   interface{}
		want string
	}{
		{"text", "text"},
		{true, "True"},
		{false, "False"},
		{nil, "None"},
		{3, "3"},
		{3.0, "3.0"},
		{2.5, "2.5"},
		{1e16, "1e+16"},
		{0.00001, "1e-05"},
		{[]interface{}{"a", 1, true, nil}, "['a', 1, True, None]"},
		{map[string]interface{}{"b": []interface{}{1}, "a": "x"}, "{'a': 'x', 'b': [1]}"},
		{[]interface{}{"it's"}, `["it's"]`},
		{[]interface{}{"a\nb\\"}, `['a\nb\\']`},
	}
	for _, c := range cases {
		if got := PyStr(c.in); got != c.want {
			t.Errorf("PyStr(%#v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestJinjaStringsPrintLikePython(t *testing.T) {
	vars := map[string]interface{}{"flag": true, "l": []interface{}{"a", "b"}}
	cases := map[string]string{
		"'x' ~ flag":            "xTrue",
		"flag | string":         "True",
		"[1, true] | join(',')": "1,True",
		"','.join(l)":           "a,b",
		"flag | lower":          "true",
	}
	for in, want := range cases {
		got, err := Eval(in, vars)
		if err != nil || got != want {
			t.Errorf("%s = %v, %v; want %s", in, got, err, want)
		}
	}
}

func TestPyJSON(t *testing.T) {
	vars := map[string]interface{}{
		"d": map[string]interface{}{"k": []interface{}{1, 2}, "é": "<a&b>", "n": nil, "f": 1.0},
		"n": map[string]interface{}{"b": map[string]interface{}{"x": 1}, "a": []interface{}{1, "y"}},
	}
	cases := map[string]string{
		"d | to_json":      `{"f": 1.0, "k": [1, 2], "n": null, "\u00e9": "<a&b>"}`,
		"n | to_nice_json": "{\n    \"a\": [\n        1,\n        \"y\"\n    ],\n    \"b\": {\n        \"x\": 1\n    }\n}",
		"[] | to_json":     "[]",
		"'{\"x\": 1, \"y\": 2.5}' | from_json | string": "{'x': 1, 'y': 2.5}",
	}
	for in, want := range cases {
		got, err := Eval(in, vars)
		if err != nil || got != want {
			t.Errorf("%s = %q, %v; want %q", in, got, err, want)
		}
	}
}
