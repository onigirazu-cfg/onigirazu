package expression

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// and/or/not follow Jinja: truthiness of any value, an operand as the result,
// lazy right side
func TestBooleanOperatorsAreJinja(t *testing.T) {
	vars := map[string]interface{}{"a": false, "x": "", "y": "v", "n": 0, "m": 2, "lst": []interface{}{}, "d": map[string]interface{}{"k": 1}}
	cases := map[string]interface{}{
		`not x`:                   true,
		`not y`:                   false,
		`not n`:                   true,
		`not lst`:                 true,
		`not d`:                   false,
		`x and y`:                 "",
		`y and m`:                 2,
		`not x and y`:             "v",
		`not x and not y`:         false,
		`x or y`:                  "v",
		`y or x`:                  "v",
		`x or 'default'`:          "default",
		`not a and not x`:         true,
		`u is defined and u.k`:    false,
		`u is not defined or u.k`: true,
		`d.k and not x`:           true,
		`(not x) and (m > 1)`:     true,
		`not (x or lst)`:          true,
	}
	for e, want := range cases {
		got, err := Eval(e, vars)
		require.NoError(t, err, e)
		assert.Equal(t, want, got, e)
	}
	holds, err := Condition("not x and y", vars)
	require.NoError(t, err)
	assert.True(t, holds)
}
