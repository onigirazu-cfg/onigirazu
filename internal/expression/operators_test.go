package expression

import (
	"fmt"
	"testing"
)

func TestJinjaOperators(t *testing.T) {
	vars := map[string]interface{}{"n": 7, "size": 5000, "d": map[string]interface{}{"k": 9}, "l": []interface{}{1}}
	cases := map[string]string{
		"7 // 2":         "3",
		"-7 // 2":        "-4",
		"7.5 // 2":       "3",
		"n // 2 + 1":     "4",
		"size // 1024":   "4",
		"(n + 1) // 3":   "2",
		"d['k'] // 2":    "4",
		"(n | int) // 2": "3",
		"'a // b'":       "a // b",
		"'x' * 3":        "xxx",
		"3 * 'ab'":       "ababab",
		"l * 2":          "[1 1]",
		"2 * 3":          "6",
		"2.5 * 2":        "5",
		"7 % 3":          "1",
		"[1, 2] + [3]":   "[1 2 3]",
	}
	for in, want := range cases {
		got, err := Eval(in, vars)
		if err != nil || fmt.Sprint(got) != want {
			t.Errorf("%s = %v, %v; want %s", in, got, err, want)
		}
	}
	if got, err := Eval("7 // 0", vars); err == nil {
		t.Errorf("7 // 0 = %v, want an error", got)
	}
}
