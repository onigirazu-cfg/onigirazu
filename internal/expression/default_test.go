package expression

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultOnMissingItems(t *testing.T) {
	vars := map[string]interface{}{
		"empty": []interface{}{},
		"items": []interface{}{"a", "b"},
		"r":     map[string]interface{}{"packages": []interface{}{}, "m": map[string]interface{}{"k": "v"}},
	}
	for expr, want := range map[string]interface{}{
		"empty[0] | default('none')":                 "none",
		"items[1] | default('none')":                 "b",
		"items[-1] | default('none')":                "b",
		"r.packages[0] | default('none')":            "none",
		"r['m']['k'] | default('none')":              "v",
		"r['m']['x'] | default('none')":              "none",
		"r.missing.deeper | default('none')":         "none",
		"r.packages[0].name | d('none')":             "none",
		"(items | first) | default('none')":          "a",
		"items[0] ~ '-' ~ (empty[0] | default('x'))": "a-x",
	} {
		got, err := Eval(expr, vars)
		require.NoError(t, err, expr)
		assert.Equal(t, want, got, expr)
	}
}

func TestRegisteredFilter(t *testing.T) {
	RegisterFilter("shout_test", func(p ...interface{}) (interface{}, error) {
		return fmt.Sprint(p[0]) + "!", nil
	})
	got, err := Eval("name | shout_test", map[string]interface{}{"name": "hi"})
	require.NoError(t, err)
	assert.Equal(t, "hi!", got)
}
