package expression

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBareTestArguments(t *testing.T) {
	for _, tc := range []struct {
		code string
		want interface{}
	}{
		{`g is sameas true`, true}, {`false is sameas false`, true}, {`none is sameas none`, true},
		{`g is sameas True`, true}, {`false is sameas False`, true}, {`none is sameas None`, true}, {`True is sameas true`, true}, {`False is sameas false`, true}, {`None is sameas none`, true},
		{`6 is divisibleby 3`, true}, {`'a' is equalto 'a'`, true}, {`2 is in [1, 2]`, true},
		{`g is sameas y.z`, true}, {`g is sameas xs[0]`, true}, {`g is not sameas true`, false},
		{`['a'] if (g is sameas true) else []`, []interface{}{"a"}},
		{`['a'] if g is sameas true else []`, []interface{}{"a"}},
		{`a if x is string else b`, "yes"}, {`x is defined and g`, true}, {`x is string or g`, true},
		{`g is sameas(true)`, true}, {`g is not sameas(false)`, true},
		{`1.5 is equalto 1.5`, true}, {`{'a': 1} is equalto {'a': 1}`, true},
	} {
		t.Run(tc.code, func(t *testing.T) {
			got, err := Eval("{{ "+tc.code+" }}", map[string]interface{}{"g": true, "x": "s", "a": "yes", "b": "no", "y": map[string]interface{}{"z": true}, "xs": []interface{}{true}})
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestBareTestArgumentBoundaries(t *testing.T) {
	// These expressions also succeed on origin/main, before bare arguments.
	for _, tc := range []struct {
		code string
		want bool
	}{
		{`x is string in [true]`, true},
		{`x is string is boolean`, true},
		{`x is string not in [false]`, true},
		{`x is string if true else false`, true},
		{`x is string if false else false`, false},
		{`x is string and true`, true},
		{`x is string or false`, true},
	} {
		t.Run(tc.code, func(t *testing.T) {
			got, err := Eval(tc.code, map[string]interface{}{"x": "s"})
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
