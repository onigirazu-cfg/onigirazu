package expression

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvalCondition(t *testing.T) {
	vars := map[string]interface{}{
		"fam":   "RedHat",
		"r":     map[string]interface{}{"rc": 0, "stdout": "ok", "changed": true},
		"flag":  true,
		"env":   "production",
		"n":     3,
		"items": []interface{}{"a", "b"},
		"empty": "",
		"f":     2.0,
	}
	cases := map[string]bool{
		`fam == "RedHat"`:                    true,
		`fam == 'Debian'`:                    false,
		`fam != "Debian"`:                    true,
		`r.rc == 0`:                          true,
		`r.stdout == "ok" and r.changed`:     true,
		`flag`:                               true,
		`not flag`:                           false,
		`fam == "RedHat" and r.rc == 0`:      true,
		`fam != "RedHat" or flag`:            true,
		`env in ['production', 'staging']`:   true,
		`env not in ['production']`:          false,
		`n > 2`:                              true,
		`n >= 4`:                             false,
		`f == 2`:                             true,
		`missing is defined`:                 false,
		`missing is not defined`:             true,
		`missing is undefined`:               true,
		`missing.child is defined`:           false,
		`r.rc is defined`:                    true,
		`fam is defined and fam == "RedHat"`: true,
		`items | length > 1`:                 true,
		`(items | length) == 2`:              true,
		`env | upper == "PRODUCTION"`:        true,
		`empty`:                              false,
		`flag == True`:                       true,
		`"is defined" == "is defined"`:       true,
		`missing == "x"`:                     false,
	}
	for cond, want := range cases {
		got, err := Condition(cond, vars)
		require.NoError(t, err, cond)
		assert.Equal(t, want, got, cond)
	}

	_, err := Condition(`fam ==`, vars)
	assert.Error(t, err)
}

func TestTruthy(t *testing.T) {
	for _, v := range []interface{}{nil, false, "", "false", "False", "no", "0", 0, 0.0, []interface{}{}, "<no value>"} {
		assert.False(t, Truthy(v), "%#v", v)
	}
	for _, v := range []interface{}{true, "true", "yes", "x", 1, 2.5, []interface{}{1}} {
		assert.True(t, Truthy(v), "%#v", v)
	}
}

func TestItems(t *testing.T) {
	items, err := Items([]string{"a", "b"})
	require.NoError(t, err)
	assert.Equal(t, []interface{}{"a", "b"}, items)

	items, err = Items(nil)
	require.NoError(t, err)
	assert.Empty(t, items)

	_, err = Items("text")
	assert.Error(t, err)

	value, err := Eval(`missing | default("x")`, nil)
	require.NoError(t, err)
	assert.Equal(t, "x", value)
}

func TestInOperator(t *testing.T) {
	vars := map[string]interface{}{
		"out":  "Active: running",
		"list": []interface{}{"a", 1},
		"m":    map[string]interface{}{"k": 1},
	}
	cases := map[string]bool{
		`"running" in out`:     true,
		`'stopped' in out`:     false,
		`"stopped" not in out`: true,
		`"a" in list`:          true,
		`1 in list`:            true,
		`1.0 in list`:          true,
		`"k" in m`:             true,
		`"x" in m`:             false,
		`"x" in missing`:       false,
	}
	for cond, want := range cases {
		got, err := Condition(cond, vars)
		require.NoError(t, err, cond)
		assert.Equal(t, want, got, cond)
	}
}

func TestDict2Items(t *testing.T) {
	vars := map[string]interface{}{"d": map[string]interface{}{"b": 2, "a": 1}}
	out, err := Eval("{{ d | dict2items }}", vars)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{
		map[string]interface{}{"key": "a", "value": 1},
		map[string]interface{}{"key": "b", "value": 2},
	}, out)
	back, err := Eval("d | dict2items | items2dict", vars)
	require.NoError(t, err)
	assert.Equal(t, vars["d"], back)
}

func TestJinjaFilters(t *testing.T) {
	vars := map[string]interface{}{
		"pkgs":  []interface{}{"jq", "curl", "jq"},
		"users": []interface{}{map[string]interface{}{"name": "a", "admin": true}, map[string]interface{}{"name": "b", "admin": false}},
		"d":     map[string]interface{}{"x": 1},
		"e":     map[string]interface{}{"c": 3, "a": 1, "b": 2},
		"s":     "Hello World",
	}
	cases := map[string]interface{}{
		`pkgs | unique | sort | join(",")`:                    "curl,jq",
		`d | to_json`:                                         `{"x":1}`,
		`'{"a": [1]}' | from_json`:                            map[string]interface{}{"a": []interface{}{1.0}},
		`s | regex_replace("(\\w+) (\\w+)", "\\2 \\1")`:       "World Hello",
		`s | regex_search("W\\w+")`:                           "World",
		`s | lower | capitalize`:                              "Hello world",
		`users | map(attribute='name') | list`:                []interface{}{"a", "b"},
		`users | selectattr('admin') | map(attribute='name')`: []interface{}{"a"},
		`users | rejectattr('name', 'equalto', 'a') | length`: 1,
		`pkgs | map('upper') | first`:                         "JQ",
		`pkgs | select('match', '^c') | list`:                 []interface{}{"curl"},
		`'/etc/app/app.conf' | basename`:                      "app.conf",
		`'x' ~ 1 ~ 'y'`:                                       "x1y",
		`'yes' if pkgs | length > 2 else 'no'`:                "yes",
		`'yes' if missing is defined else 'no'`:               "no",
		`d | combine({'y': 2})`:                               map[string]interface{}{"x": 1, "y": 2},
		`d.keys() | list`:                                     []interface{}{"x"},
		`e.keys() | join(',')`:                                "a,b,c",
		`e | values`:                                          []interface{}{1, 2, 3},
		`range(1, 4) | sum`:                                   6,
		`true | ternary('on', 'off')`:                         "on",
		`'a b' | quote`:                                       "'a b'",
		`'aGk=' | b64decode`:                                  "hi",
	}
	for e, want := range cases {
		got, err := Eval(e, vars)
		require.NoError(t, err, e)
		assert.Equal(t, want, got, e)
	}
}

func TestExtractFilter(t *testing.T) {
	vars := map[string]interface{}{"hv": map[string]interface{}{"a": map[string]interface{}{"ip": "10.0.0.1"}}}
	out, err := Eval(`['a', 'missing'] | map('extract', hv, 'ip') | list`, vars)
	require.NoError(t, err)
	assert.Equal(t, []interface{}{"10.0.0.1", nil}, out)
}
