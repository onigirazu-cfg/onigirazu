package modules

import "testing"

func TestFindSelection(t *testing.T) {
	now := 1_000_000.0
	day := 86400.0
	rec := func(name string, ageDays float64, size int64) map[string]interface{} {
		return map[string]interface{}{"name": name, "mtime": now - ageDays*day, "size": size}
	}
	cases := []struct {
		args map[string]interface{}
		rec  map[string]interface{}
		want bool
	}{
		{map[string]interface{}{}, rec(".hidden", 0, 1), false},
		{map[string]interface{}{"hidden": true}, rec(".hidden", 0, 1), true},
		{map[string]interface{}{"age": "30d"}, rec("old", 40, 1), true},
		{map[string]interface{}{"age": "30d"}, rec("new", 1, 1), false},
		{map[string]interface{}{"age": "-1d"}, rec("new", 0.5, 1), true},
		{map[string]interface{}{"age": "-1d"}, rec("old", 2, 1), false},
		{map[string]interface{}{"size": "1k"}, rec("big", 0, 5000), true},
		{map[string]interface{}{"size": "1k"}, rec("small", 0, 10), false},
		{map[string]interface{}{"size": "-1k"}, rec("small", 0, 10), true},
		{map[string]interface{}{"excludes": "b*"}, rec("big.log", 0, 1), false},
		{map[string]interface{}{"use_regex": true}, rec("a.log", 0, 1), true},
	}
	for i, c := range cases {
		patterns := []string{"*"}
		if c.args["use_regex"] == true {
			patterns = []string{`[ab]\.`}
		}
		sel, err := newFindSelection(c.args, patterns)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		if got := sel.keep(c.rec, now); got != c.want {
			t.Errorf("%d %v %v: got %v, want %v", i, c.args, c.rec["name"], got, c.want)
		}
	}
	for _, bad := range []map[string]interface{}{{"contains": "x"}, {"age_stamp": "atime"}, {"age": "soon"}} {
		if _, err := newFindSelection(bad, nil); err == nil {
			t.Errorf("%v: expected an error", bad)
		}
	}
}
