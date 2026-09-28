package facts

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
)

type fixedRunner string

func (f fixedRunner) ExecuteCommand(string) (string, error) { return string(f), nil }

func TestGatherVirtualization(t *testing.T) {
	g := &Gatherer{}
	cases := map[string][2]string{
		"docker\n": {"docker", "guest"},
		"vmware\n": {"VMware", "guest"},
		"kvm\n":    {"kvm", "guest"},
		"none\n":   {"NA", "NA"},
	}
	for out, want := range cases {
		f := &cache.SystemFacts{Kernel: "Linux"}
		g.gatherVirtualization(fixedRunner(out), f)
		if f.VirtualizationType != want[0] || f.VirtualizationRole != want[1] {
			t.Errorf("%q: got %s/%s, want %s/%s", out, f.VirtualizationType, f.VirtualizationRole, want[0], want[1])
		}
	}
	f := &cache.SystemFacts{Kernel: "Darwin"}
	g.gatherVirtualization(fixedRunner("docker"), f)
	if f.VirtualizationType != "NA" {
		t.Errorf("non-Linux: %s", f.VirtualizationType)
	}
}
