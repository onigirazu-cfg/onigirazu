package expression

import "testing"

func TestDefinedHoldsNone(t *testing.T) {
	vars := map[string]interface{}{
		"n": nil,
		"d": map[string]interface{}{"k": nil, "l": []interface{}{nil}},
		"s": map[string]string{"a": "b"},
		"i": 0,
	}
	cases := map[string]bool{
		"n is defined":         true,
		"n is undefined":       false,
		"n is not defined":     false,
		"missing is defined":   false,
		"missing is undefined": true,
		"d.k is defined":       true,
		"d['zz'] is defined":   false,
		"d.zz is defined":      false,
		"d.k.x is defined":     false,
		"d.l[0] is defined":    true,
		"d.l[5] is defined":    false,
		"s.a is defined":       true,
		"s.b is defined":       false,
		"n is none":            true,
		"d.l[i] is defined":    false, // not a literal path: the value decides
	}
	for cond, want := range cases {
		got, err := Condition(cond, vars)
		if err != nil || got != want {
			t.Errorf("%s: got %v, %v; want %v", cond, got, err, want)
		}
	}
}
