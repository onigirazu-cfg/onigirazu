package vault

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadReadsYAML11Booleans(t *testing.T) {
	in := `a: yes
b: No
c: "yes"
d: 'off'
e: [on, OFF, maybe]
on: kept
f: yes please
g:
  - no
`
	out, err := Load([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]interface{}
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]interface{}{
		"a": true, "b": false, "c": "yes", "d": "off",
		"e": []interface{}{true, false, "maybe"}, "on": "kept", "f": "yes please",
		"g": []interface{}{false},
	}
	for k, v := range want {
		if got[k] == nil || !equalYAML(got[k], v) {
			t.Errorf("%s = %#v, want %#v", k, got[k], v)
		}
	}
}

func equalYAML(a, b interface{}) bool {
	x, _ := yaml.Marshal(a)
	y, _ := yaml.Marshal(b)
	return string(x) == string(y)
}
