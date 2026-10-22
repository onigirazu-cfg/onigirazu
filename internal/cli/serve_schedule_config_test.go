package cli

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestServeAtYAML(t *testing.T) {
	for _, spec := range []string{"at: 'tue 18:00'", "at: ['mon,thu 03:30', 'sat 12:00']"} {
		var j ServeJob
		if err := yaml.Unmarshal([]byte(spec), &j); err != nil {
			t.Fatal(err)
		}
		if err := j.prepareSchedule(""); err != nil {
			t.Fatal(err)
		}
		if len(j.at) != len(j.At) || j.AtTZ != "UTC" {
			t.Fatalf("not prepared: %+v", j)
		}
		if err := j.prepareSchedule(""); err != nil {
			t.Fatalf("not idempotent: %v", err)
		}
	}
	for _, spec := range []string{"at: 123", "at: [123]", "at: {day: mon}"} {
		var j ServeJob
		if err := yaml.Unmarshal([]byte(spec), &j); err == nil {
			t.Errorf("accepted %q", spec)
		}
	}
}

func TestServePrepareScheduleValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		job  ServeJob
	}{
		{"exclusive", ServeJob{Every: time.Hour, At: ServeAt{"mon 12:00"}}},
		{"zone without at", ServeJob{AtTZ: "UTC"}},
		{"zone", ServeJob{At: ServeAt{"* 12:00"}, AtTZ: "Not/AZone"}},
		{"negative", ServeJob{Every: -time.Second}},
		{"day", ServeJob{At: ServeAt{"monday 12:00"}}},
		{"empty day", ServeJob{At: ServeAt{"mon, 12:00"}}},
		{"range", ServeJob{At: ServeAt{"mon-tue-wed 12:00"}}},
		{"clock", ServeJob{At: ServeAt{"mon 24:00"}}},
		{"clock width", ServeJob{At: ServeAt{"mon 2:00"}}},
		{"fields", ServeJob{At: ServeAt{"mon"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.job.Name = "example"
			if err := tc.job.prepareSchedule(""); err == nil || !strings.Contains(err.Error(), "job example:") {
				t.Fatalf("expected job-specific error, got %v", err)
			}
		})
	}
}

func TestServeMissedResolution(t *testing.T) {
	for _, tc := range []struct{ kind, job, global, want string }{
		{"apply", "", "", "skip"},
		{"drift", "", "", "run_once"},
		{"verify", "", "", "run_once"},
		{"comply", "", "", "run_once"},
		{"apply", "", "run_once", "run_once"},
		{"drift", "skip", "run_once", "skip"},
	} {
		j := ServeJob{Kind: tc.kind, Missed: tc.job}
		if err := j.prepareSchedule(tc.global); err != nil || j.missed != tc.want {
			t.Errorf("%+v: policy %q, err %v", tc, j.missed, err)
		}
	}
	for _, tc := range []struct{ job, global string }{{"wrong", ""}, {"skip", "wrong"}} {
		j := ServeJob{Missed: tc.job}
		if err := j.prepareSchedule(tc.global); err == nil {
			t.Errorf("accepted invalid policy %+v", tc)
		}
	}
}

func TestServeNextAt(t *testing.T) {
	for _, tc := range []struct {
		name, spec, zone, after, want string
	}{
		{"week boundary", "mon 03:30", "UTC", "2026-10-25T23:00:00Z", "2026-10-26T03:30:00Z"},
		{"strictly after", "mon 03:30", "UTC", "2026-10-26T03:30:00Z", "2026-11-02T03:30:00Z"},
		{"wrapping range", "fri-mon 12:00", "UTC", "2026-10-24T13:00:00Z", "2026-10-25T12:00:00Z"},
		{"daily", "* 04:00", "UTC", "2026-10-25T12:00:00Z", "2026-10-26T04:00:00Z"},
		{"spring forward", "sun 02:30", "Europe/Madrid", "2026-03-28T23:00:00Z", "2026-03-29T01:30:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			j := ServeJob{At: ServeAt{tc.spec}, AtTZ: tc.zone}
			if err := j.prepareSchedule(""); err != nil {
				t.Fatal(err)
			}
			after, _ := time.Parse(time.RFC3339, tc.after)
			if got := j.nextAt(after).Format(time.RFC3339); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
	j := ServeJob{At: ServeAt{"sun 02:30"}, AtTZ: "Europe/Madrid"}
	if err := j.prepareSchedule(""); err != nil {
		t.Fatal(err)
	}
	// The repeated hour fires once, at its first occurrence (summer time).
	want := time.Date(2026, 10, 25, 0, 30, 0, 0, time.UTC)
	after := time.Date(2026, 10, 25, 0, 0, 0, 0, j.atLoc)
	if got := j.nextAt(after); !got.Equal(want) {
		t.Fatalf("fall back: got %v, want %v", got, want)
	}
	if got := j.nextAt(want); got.In(j.atLoc).Day() == 25 {
		t.Fatalf("repeated hour fired twice: %v", got)
	}
}

func TestServeScheduleSignature(t *testing.T) {
	j := ServeJob{Every: 6 * time.Hour}
	if got := j.scheduleSignature(); got != "every=6h0m0s" {
		t.Fatal(got)
	}
	j = ServeJob{At: ServeAt{"mon,thu 03:30", "sat 12:00"}}
	if err := j.prepareSchedule(""); err != nil {
		t.Fatal(err)
	}
	if got := j.scheduleSignature(); got != "at=mon,thu 03:30;sat 12:00;tz=UTC" {
		t.Fatal(got)
	}
}

func TestServeInvalidAtNamesJob(t *testing.T) {
	var cfg ServeConfig
	err := yaml.Unmarshal([]byte("jobs:\n  - name: example\n    at: [123]\n"), &cfg)
	if err == nil || !strings.Contains(err.Error(), "job example:") {
		t.Fatalf("error=%v", err)
	}
}
