package engine

import (
	"reflect"
	"testing"
)

func TestFilterFacts(t *testing.T) {
	facts := map[string]interface{}{
		"ansible_local": 1, "ansible_distribution": 2, "ansible_distribution_version": 3,
		"ansible_facts": 4, "onigirazu_kernel": 5,
	}
	cases := map[string]struct {
		filter interface{}
		want   []string
	}{
		"all":    {nil, []string{"ansible_local", "ansible_distribution", "ansible_distribution_version"}},
		"one":    {"ansible_local", []string{"ansible_local"}},
		"glob":   {"ansible_distribution*", []string{"ansible_distribution", "ansible_distribution_version"}},
		"list":   {[]interface{}{"ansible_local", "ansible_distribution"}, []string{"ansible_local", "ansible_distribution"}},
		"commas": {"ansible_local, ansible_distribution", []string{"ansible_local", "ansible_distribution"}},
	}
	for name, c := range cases {
		got := filterFacts(facts, factFilters(c.filter))
		want := map[string]interface{}{}
		for _, k := range c.want {
			want[k] = facts[k]
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
}
