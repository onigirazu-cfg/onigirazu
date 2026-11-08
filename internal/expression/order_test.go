package expression

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLiteralSerializationOrder(t *testing.T) {
	for _, tc := range []struct{ code, want string }{
		{`{'z': 1, 'a': 2} | to_json`, `{"z": 1, "a": 2}`},
		{`{'z': {'y': 1, 'a': 2}, 'a': [true, none, 1.0, 'é😀']} | to_json`, `{"z": {"y": 1, "a": 2}, "a": [true, null, 1.0, "\u00e9\ud83d\ude00"]}`},
		{`{'z': 1, 'a': 2} | combine({'z': 3, 'b': 4}) | to_json`, `{"z": 3, "a": 2, "b": 4}`},
		{`{'z': {'y': 1, 'a': 2}} | combine({'z': {'y': 3, 'b': 4}}, recursive=true) | to_json`, `{"z": {"y": 3, "a": 2, "b": 4}}`},
		// to_nice_json and to_nice_yaml sort keys in Ansible
		{`{'z': 1, 'a': {'y': 2, 'b': 3}} | to_nice_json`, "{\n    \"a\": {\n        \"b\": 3,\n        \"y\": 2\n    },\n    \"z\": 1\n}"},
		{`[{'z': 1, 'a': 2}, {'z': 3, 'b': 4}] | combine | to_json`, `{"z": 3, "a": 2, "b": 4}`},
		{`{'z': 1, 'a': 2, 'z': 3} | to_json`, `{"z": 3, "a": 2}`},
		{`{'z': -2, 'a': 0.25, 's': '<a&b>\n'} | to_json`, `{"z": -2, "a": 0.25, "s": "<a&b>\n"}`},
		{`{'z': 1, 'a': 2} | to_nice_yaml`, "a: 2\nz: 1\n"},
		{`{'z': 1, 'a': 2} | dict2items | map(attribute='key') | join(',')`, "z,a"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			got, err := Eval(tc.code, nil)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDaemonJSONOrder(t *testing.T) {
	got, err := Eval(`{'userland-proxy': false, 'log-driver': 'json-file', 'log-opts': {'max-size': '20m', 'max-file': '5'}} | combine({'userns-remap': 'default'} if flag else {}) | combine({'runtimes': {}} if other else {}) | to_json`, map[string]interface{}{"flag": false, "other": false})
	require.NoError(t, err)
	require.Equal(t, `{"userland-proxy": false, "log-driver": "json-file", "log-opts": {"max-size": "20m", "max-file": "5"}}`, got)
}
