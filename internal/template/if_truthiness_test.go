package template

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// {% if not a and not b %} inside a command works for strings, numbers and
// lists, not only booleans; a condition the evaluator rejects is reported
func TestIfBlockTruthiness(t *testing.T) {
	e := NewEngine()
	vars := map[string]interface{}{"a": "", "b": []interface{}{}, "c": "set", "n": 0}
	cases := map[string]string{
		`run {% if not a and not b %}--bootstrap{% endif %} now`: "run --bootstrap now",
		`{% if not a and c %}yes{% else %}no{% endif %}`:         "yes",
		`{% if c and not n %}yes{% endif %}`:                     "yes",
		`{% if a or b %}yes{% else %}no{% endif %}`:              "no",
	}
	for tpl, want := range cases {
		out, err := e.Render(context.Background(), tpl, vars)
		require.NoError(t, err, tpl)
		assert.Equal(t, want, out, tpl)
	}
	_, err := e.Render(context.Background(), `{% if a | nosuchfilter %}x{% endif %}`, vars)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "not defined", "the evaluator's own error is shown, not Go's")
}
