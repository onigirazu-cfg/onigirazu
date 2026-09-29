package expression

import (
	"fmt"
	"testing"
)

func TestFilterKeywordArguments(t *testing.T) {
	vars := map[string]interface{}{
		"people": []interface{}{
			map[string]interface{}{"n": "x", "age": 30, "info": map[string]interface{}{"city": "b"}},
			map[string]interface{}{"n": "y", "age": 20, "info": map[string]interface{}{"city": "a"}},
			map[string]interface{}{"n": "z", "age": 30, "info": map[string]interface{}{"city": "c"}},
		},
		"words": []interface{}{"b", "A", "c"},
		"d1":    map[string]interface{}{"a": 1, "b": map[string]interface{}{"c": 2}, "l": []interface{}{1}},
		"d2":    map[string]interface{}{"b": map[string]interface{}{"d": 3}, "l": []interface{}{2}},
		"kv":    []interface{}{map[string]interface{}{"k": "a", "v": 1}},
	}
	cases := map[string]string{
		"people | sort(attribute='age') | map(attribute='n') | list":               "[y x z]",
		"people | sort(attribute='age', reverse=true) | map(attribute='n') | list": "[x z y]",
		"people | sort(attribute='age,n') | map(attribute='n') | first":            "y",
		"people | sort(attribute='info.city') | map(attribute='n') | list":         "[y x z]",
		"words | sort":                                  "[A b c]",
		"words | sort(case_sensitive=true)":             "[A b c]",
		"words | sort(reverse=true)":                    "[c b A]",
		"[3, 10, 2] | sort":                             "[2 3 10]",
		"d1 | combine(d2)":                              "map[a:1 b:map[d:3] l:[2]]",
		"d1 | combine(d2, recursive=true)":              "map[a:1 b:map[c:2 d:3] l:[2]]",
		"d1 | combine(d2, list_merge='append')":         "map[a:1 b:map[d:3] l:[1 2]]",
		"kv | items2dict(key_name='k', value_name='v')": "map[a:1]",
		"[{'key': 'x', 'value': 2}] | items2dict":       "map[x:2]",
		"1 == 1": "true",
	}
	for in, want := range cases {
		got, err := Eval(in, vars)
		if err != nil || fmt.Sprint(got) != want {
			t.Errorf("%s = %v, %v; want %s", in, got, err, want)
		}
	}
}
