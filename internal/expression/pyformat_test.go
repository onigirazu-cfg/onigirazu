package expression

import (
	"fmt"
	"testing"
)

func TestPythonFormatting(t *testing.T) {
	vars := map[string]interface{}{"name": "web", "n": 3, "f": 2.5, "flag": true, "l": []interface{}{"a"}}
	cases := map[string]string{
		"'%-5s|' | format('ab')":           "ab   |",
		"'%s-%d' | format(name, n)":        "web-3",
		"'%05.2f' | format(f)":             "02.50",
		"'%x %X %o' | format(255, 255, 8)": "ff FF 10",
		"'%s %r' | format(flag, 'x')":      "True 'x'",
		"'%s' | format(l)":                 "['a']",
		"'100%% %s' | format(n)":           "100% 3",
		"'%e' | format(12345.678)":         "1.234568e+04",
		"'%s items' % n":                   "3 items",
		"'%.1f%%' % f":                     "2.5%",
		"7 % 3":                            "1",
		"-7 % 3":                           "2",
		"7 % -3":                           "-2",
		"7.5 % 2":                          "1.5",
	}
	for in, want := range cases {
		got, err := Eval(in, vars)
		if err != nil || fmt.Sprint(got) != want {
			t.Errorf("%s = %v, %v; want %s", in, got, err, want)
		}
	}
	for _, in := range []string{"'%s %s' | format('a')", "'%s' | format('a', 'b')", "'%d' | format('x')"} {
		if got, err := Eval(in, vars); err == nil {
			t.Errorf("%s = %v, want an error", in, got)
		}
	}
}
