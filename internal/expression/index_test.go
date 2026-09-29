package expression

import (
	"fmt"
	"testing"
)

func TestJinjaIndexing(t *testing.T) {
	vars := map[string]interface{}{
		"item":  []interface{}{"a", []interface{}{"b", "c"}},
		"pairs": []interface{}{[]interface{}{1, 2}},
		"s":     "abc",
		"l":     []interface{}{1, 2, 3},
		"i":     1,
		"f":     3.14,
	}
	cases := map[string]string{
		"item.0":          "a",
		"item.1.0":        "b",
		"item[1].1":       "c",
		"pairs[0].1":      "2",
		"'abc'[-1]":       "c",
		"s[0]":            "a",
		"s[i]":            "b",
		"l[-1]":           "3",
		"l[i]":            "2",
		"'abc'[1:]":       "bc",
		"l[1:]":           "[2 3]",
		"l[:-1]":          "[1 2]",
		"f * 2":           "6.28",
		"3.5 + 1":         "4.5",
		"l[5] is defined": "false",
	}
	for in, want := range cases {
		got, err := Eval(in, vars)
		if err != nil || fmt.Sprint(got) != want {
			t.Errorf("%s = %v, %v; want %s", in, got, err, want)
		}
	}
}
