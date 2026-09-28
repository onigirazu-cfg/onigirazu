package expression

import (
	"reflect"
	"testing"
)

func TestPythonMethods(t *testing.T) {
	vars := map[string]interface{}{
		"opts": "--enable-shared  --with-x",
		"csv":  "a,b,c",
		"d":    map[string]interface{}{"k": "v"},
		"l":    []interface{}{"x", "y", "x"},
		"host": "web01.example.com",
	}
	cases := map[string]interface{}{
		"opts.split()":                    []interface{}{"--enable-shared", "--with-x"},
		"csv.split(',')":                  []interface{}{"a", "b", "c"},
		"csv.split(',', 1)":               []interface{}{"a", "b,c"},
		"csv.rsplit(',', 1)":              []interface{}{"a,b", "c"},
		"csv.split(',')[1]":               "b",
		"'  x '.strip()":                  "x",
		"host.startswith('web')":          true,
		"host.endswith(['.org', '.com'])": true,
		"host.split('.')[0].upper()":      "WEB01",
		"csv.replace(',', ';')":           "a;b;c",
		"'-'.join(l)":                     "x-y-x",
		"d.get('k')":                      "v",
		"d.get('missing', 'dflt')":        "dflt",
		"l.count('x')":                    2,
		"l.index('y')":                    1,
		"'42'.isdigit()":                  true,
		"(opts.split() | length) == 2":    true,
	}
	for code, want := range cases {
		got, err := Eval(code, vars)
		if err != nil {
			t.Errorf("%s: %v", code, err)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %#v, want %#v", code, got, want)
		}
	}
}
