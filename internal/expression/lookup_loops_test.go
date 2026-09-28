package expression

import (
	"reflect"
	"testing"
)

func TestLoopLookups(t *testing.T) {
	users := []interface{}{
		map[string]interface{}{"name": "ann", "keys": []interface{}{"k1", "k2"}},
		map[string]interface{}{"name": "cid"},
	}
	cases := []struct {
		plugin string
		terms  []interface{}
		want   []interface{}
	}{
		{"nested", []interface{}{[]interface{}{1, 2}, []interface{}{"a"}},
			[]interface{}{[]interface{}{1, "a"}, []interface{}{2, "a"}}},
		{"together", []interface{}{[]interface{}{"a", "b"}, []interface{}{1}},
			[]interface{}{[]interface{}{"a", 1}, []interface{}{"b", nil}}},
		{"indexed_items", []interface{}{"x", "y"},
			[]interface{}{[]interface{}{0, "x"}, []interface{}{1, "y"}}},
		{"subelements", []interface{}{users, "keys", map[string]interface{}{"skip_missing": true}},
			[]interface{}{[]interface{}{users[0], "k1"}, []interface{}{users[0], "k2"}}},
	}
	for _, c := range cases {
		got, err := LookupItems("", nil, c.plugin, c.terms)
		if err != nil {
			t.Errorf("%s: %v", c.plugin, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s = %#v, want %#v", c.plugin, got, c.want)
		}
	}
	if _, err := LookupItems("", nil, "subelements", []interface{}{users, "keys"}); err == nil {
		t.Error("subelements without skip_missing must fail on an element without the key")
	}
	got, err := LookupItems("", nil, "random_choice", []interface{}{"only"})
	if err != nil || !reflect.DeepEqual(got, []interface{}{"only"}) {
		t.Errorf("random_choice = %v, %v", got, err)
	}
}
