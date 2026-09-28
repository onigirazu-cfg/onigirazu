package modules

import (
	"reflect"
	"testing"
)

func TestParseDockerBytes(t *testing.T) {
	cases := map[string]int64{"512m": 512 << 20, "1G": 1 << 30, "1.5g": 3 << 29, "2048": 2048, "64kb": 64 << 10}
	for in, want := range cases {
		got, err := parseDockerBytes(in)
		if err != nil || got != want {
			t.Errorf("parseDockerBytes(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, bad := range []string{"x", "-1m", "1q"} {
		if _, err := parseDockerBytes(bad); err == nil {
			t.Errorf("parseDockerBytes(%q) succeeded", bad)
		}
	}
}

func TestContainerLimits(t *testing.T) {
	l, err := containerLimits(map[string]interface{}{"cpus": "1.5", "memory": "512m"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"--cpus", "1.5", "--memory", "536870912"}; !reflect.DeepEqual(l.runArgs(), want) {
		t.Errorf("runArgs = %v, want %v", l.runArgs(), want)
	}
	same := &ContainerState{NanoCPUs: 1500000000, Memory: 512 << 20}
	if got := l.updateArgs(same); got != nil {
		t.Errorf("updateArgs on matching limits = %v", got)
	}
	if got := l.updateArgs(&ContainerState{NanoCPUs: 1e9, Memory: 512 << 20}); !reflect.DeepEqual(got, []string{"--cpus", "1.5"}) {
		t.Errorf("updateArgs = %v", got)
	}
	unlimited := l.updateArgs(&ContainerState{NanoCPUs: 1500000000, MemorySwap: -1})
	if want := []string{"--memory", "536870912", "--memory-swap", "-1"}; !reflect.DeepEqual(unlimited, want) {
		t.Errorf("updateArgs with unlimited swap = %v, want %v", unlimited, want)
	}
	swapped := l.updateArgs(&ContainerState{NanoCPUs: 1500000000, Memory: 256 << 20, MemorySwap: 512 << 20})
	if want := []string{"--memory", "536870912", "--memory-swap", "1073741824"}; !reflect.DeepEqual(swapped, want) {
		t.Errorf("updateArgs with swap = %v, want %v", swapped, want)
	}
	none, _ := containerLimits(map[string]interface{}{"cpus": 2})
	if got := none.updateArgs(&ContainerState{NanoCPUs: 2e9, Memory: 1 << 30}); got != nil {
		t.Errorf("memory not set must not update memory: %v", got)
	}
	if _, err := containerLimits(map[string]interface{}{"cpus": "many"}); err == nil {
		t.Error("bad cpus accepted")
	}
}
