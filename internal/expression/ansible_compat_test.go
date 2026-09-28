package expression

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnsibleCompat(t *testing.T) {
	vars := map[string]interface{}{
		"r":     map[string]interface{}{"failed": false, "changed": true, "skipped": false},
		"bad":   map[string]interface{}{"failed": true},
		"names": []interface{}{"es", "fleet"},
		"lists": map[string]interface{}{"es": []interface{}{"x"}},
		"users": []interface{}{map[string]interface{}{"name": "a"}, map[string]interface{}{}},
	}
	cases := map[string]interface{}{
		"r is success":                                          true,
		"r is succeeded":                                        true,
		"bad is failed":                                         true,
		"bad is not success":                                    true,
		"r is changed":                                          true,
		"r is not skipped":                                      true,
		"'1.10' is version('1.9', '>')":                         true,
		"'2.0' is version('2.0.1', '<')":                        true,
		"'1.0rc1' is version('1.0', '>')":                       true, // LooseVersion, as Ansible's default
		"'22.04' is version('22.04')":                           true,
		"[1,2] is subset([1,2,3])":                              true,
		"{'a': 1} is mapping":                                   true,
		"4 is even":                                             true,
		"[1,2,3] | intersect([2,3,4])":                          []interface{}{2, 3},
		"[1,2,3] | difference([2])":                             []interface{}{1, 3},
		"[1,2] | union([2,3])":                                  []interface{}{1, 2, 3},
		"[1,2] | symmetric_difference([2,3])":                   []interface{}{1, 3},
		"[1,2] | product(['a'])":                                []interface{}{[]interface{}{1, "a"}, []interface{}{2, "a"}},
		"[1,2] | zip(['a','b'])":                                []interface{}{[]interface{}{1, "a"}, []interface{}{2, "b"}},
		"'a.b*' | regex_escape":                                 `a\.b\*`,
		"users | map(attribute='name', default='-')":            []interface{}{"a", "-"},
		"(names | intersect(lists.keys()) + ['Other']) | first": "es",
		"[1] + [2]":                                             []interface{}{1, 2},
		"'a' + 'b'":                                             "ab",
		"1 + 2":                                                 3,
		"1.5 + 1":                                               2.5,
	}
	for e, want := range cases {
		got, err := Eval(e, vars)
		require.NoError(t, err, e)
		assert.Equal(t, want, got, e)
	}
}

func TestLookupFQCN(t *testing.T) {
	t.Setenv("ONI_TEST_ENV", "v")
	got, err := Eval("lookup('ansible.builtin.env', 'ONI_TEST_ENV')", map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, "v", got)
}
