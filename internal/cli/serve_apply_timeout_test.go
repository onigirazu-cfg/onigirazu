package cli

import (
	"slices"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestApplyArgsNoApplyTimeout(t *testing.T) {
	if args := (driftCheckOptions{}).applyArgs("p.yml", false); slices.Contains(args, "--timeout=0") {
		t.Fatalf("the default keeps apply's timeout: %v", args)
	}
	if args := (driftCheckOptions{noApplyTimeout: true}).applyArgs("p.yml", false); !slices.Contains(args, "--timeout=0") {
		t.Fatalf("serve's jobs must switch apply's timeout off: %v", args)
	}
}

func TestApplyStatus(t *testing.T) {
	failedHost := map[string][]DriftItem{"h": {{}}}
	for _, c := range []struct {
		name   string
		report DriftReport
		result types.PlaybookResult
		check  bool
		want   string
	}{
		{name: "clean", want: "ok"},
		{name: "changed", report: DriftReport{Drift: failedHost}, want: "ok"},
		{name: "drift in a check", report: DriftReport{Drift: failedHost}, check: true, want: "drift"},
		{name: "failed host", report: DriftReport{Errors: failedHost}, want: "failed"},
		{
			name:   "play cut with no failed host",
			report: DriftReport{Drift: failedHost},
			result: types.PlaybookResult{Failed: true, Error: "task execution canceled: context deadline exceeded"},
			want:   "failed",
		},
	} {
		if got := applyStatus(&c.report, &c.result, c.check); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}
