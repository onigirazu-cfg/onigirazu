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

func TestJinjaStringEscapes(t *testing.T) {
	cases := map[string]string{
		`'Foo Bar' | regex_replace('(\w+) (\w+)', '\2 \1')`: "Bar Foo",
		`'a1b22' | regex_findall('\d+') | length`:           "2",
		`'it\'s'`:                      "it's",
		`'a\nb' | length`:              "3",
		`"x\\y" | length`:              "3",
		"'a\nb'.splitlines() | length": "2",
	}
	for in, want := range cases {
		got, err := Eval(in, map[string]interface{}{})
		if err != nil || fmt.Sprint(got) != want {
			t.Errorf("%s = %v, %v; want %s", in, got, err, want)
		}
	}
}

func TestWordcountAndCenter(t *testing.T) {
	cases := map[string]string{
		"'hello world, again' | wordcount": "3",
		"'x' | center(5)":                  "  x  ",
		"'ab' | center(5)":                 "  ab ",
		"'abc' | center(6)":                " abc  ",
		"'abc' | center(2)":                "abc",
	}
	for in, want := range cases {
		got, err := Eval(in, map[string]interface{}{})
		if err != nil || fmt.Sprint(got) != want {
			t.Errorf("%s = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestSplitFilter(t *testing.T) {
	cases := map[string]string{
		"'  a  b ' | split":         "[a b]",
		"'a,b,,c' | split(',')":     "[a b  c]",
		"'a,b,c' | split(',', 1)":   "[a b,c]",
		"'a b  c' | split(none, 1)": "[a b  c]",
	}
	for in, want := range cases {
		got, err := Eval(in, map[string]interface{}{})
		if err != nil || fmt.Sprint(got) != want {
			t.Errorf("%s = %v, %v; want %s", in, got, err, want)
		}
	}
}
