package expression

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomFilter(t *testing.T) {
	vars := map[string]interface{}{"items": []interface{}{"a", "b", "c"}, "host": "web1"}
	for i := 0; i < 20; i++ {
		v, err := Eval("items | random", vars)
		require.NoError(t, err)
		assert.Contains(t, []interface{}{"a", "b", "c"}, v)

		n, err := Eval("60 | random", vars)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, n, 0)
		assert.Less(t, n, 60)

		s, err := Eval("100 | random(start=10, step=10)", vars)
		require.NoError(t, err)
		assert.Equal(t, 0, s.(int)%10)
		assert.GreaterOrEqual(t, s, 10)
	}
	// the same seed gives the same value (cron minute per host)
	a, err := Eval("60 | random(seed=host)", vars)
	require.NoError(t, err)
	b, _ := Eval("60 | random(seed=host)", vars)
	assert.Equal(t, a, b)

	sh, err := Eval("items | shuffle | length", vars)
	require.NoError(t, err)
	assert.Equal(t, 3, sh)

	// a comparison is not a keyword argument
	ok, err := Condition("host == 'web1'", vars)
	require.NoError(t, err)
	assert.True(t, ok)
}
