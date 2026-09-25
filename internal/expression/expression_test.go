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
