package expression

import (
	"strings"
	"testing"
)

func TestFilters221(t *testing.T) {
	vars := map[string]interface{}{"l": []interface{}{1, 2, 3}, "b": []interface{}{[]interface{}{1}, []interface{}{2}}}
	cases := map[string]interface{}{
		`{{ 'abc' | md5 }}`:                                                "900150983cd24fb0d6963f7d28e17f72",
		`{{ 'abc' | sha1 }}`:                                               "a9993e364706816aba3e25717850c26c9cd0d89d",
		`{{ 'abc' | checksum }}`:                                           "a9993e364706816aba3e25717850c26c9cd0d89d",
		`{{ l | combinations(2) | length }}`:                               3,
		`{{ l | permutations(2) | length }}`:                               6,
		`{{ ['/a/b/c', '/a/b/d'] | commonpath }}`:                          "/a/b",
		`{{ '/a/../b/./c' | normpath }}`:                                   "/b/c",
		`{{ ['a', 'b', 'c'] | path_join }}`:                                "a/b/c",
		`{{ '/a/b/c' | relpath('/a') }}`:                                   "b/c",
		`{{ 1048576 | human_readable }}`:                                   "1.00 MB",
		`{{ '2M' | human_to_bytes }}`:                                      2097152,
		`{{ 8 | log(2) }}`:                                                 3.0,
		`{{ 2 | pow(3) }}`:                                                 8.0,
		`{{ 9 | root }}`:                                                   3.0,
		`{{ 'http://u:p@h.example:8080/p?q=1#f' | urlsplit('hostname') }}`: "h.example",
		`{{ 'http://u:p@h.example:8080/p?q=1#f' | urlsplit('port') }}`:     8080,
		`{{ 'a%20b' | urldecode }}`:                                        "a b",
		`{{ 'C:\\dir\\f.txt' | win_basename }}`:                            "f.txt",
		`{{ 'C:\\dir\\f.txt' | win_dirname }}`:                             "C:\\dir",
		`{{ ('C:\\dir' | win_splitdrive)[0] }}`:                            "C:",
		`{{ l | type_debug }}`:                                             "list",
		`{{ 'x' | type_debug }}`:                                           "str",
		`{{ 'a: 1\n---\nb: 2' | from_yaml_all | length }}`:                 2,
		`{{ [{'n': 'a', 'v': 1}, {'n': 'b', 'v': 2}] | rekey_on_member('n') | length }}`: 2,
		`{{ '%Y' | strftime(0, utc=True) }}`:                                             "1970",
		`{{ 'hello' | to_uuid }}`:                                                        "e76ab201-8286-5535-ab8a-9bfd70298ee3",
		`{{ ('2024-05-06' | to_datetime('%Y-%m-%d')).Year() }}`:                          2024,
		`{{ [{'a': [1, 2]}] | subelements('a') | length }}`:                              2,
		`{{ ([1, 2] | zip_longest([3]))[1][1] is none }}`:                                true,
		`{{ l | product([0]) | length }}`:                                                3,
	}
	for code, want := range cases {
		got, err := Eval(code, vars)
		if err != nil || !equalLoose(got, want) {
			t.Errorf("%s = %v (%T), %v; want %v", code, got, got, err, want)
		}
	}
	if c, _ := Eval(`{{ 'line one\nline two' | comment }}`, nil); !strings.Contains(c.(string), "# line one\n# line two\n") {
		t.Errorf("comment: %q", c)
	}
}

func equalLoose(a, b interface{}) bool {
	if af, ok := toFloat(a); ok {
		if bf, ok := toFloat(b); ok {
			return af == bf
		}
	}
	return a == b
}
