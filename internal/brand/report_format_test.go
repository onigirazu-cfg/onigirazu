package brand

import (
	"testing"
	"time"
)

func TestReportDuration(t *testing.T) {
	for _, tt := range []struct {
		d    time.Duration
		want string
	}{
		{0, "<1 ms"}, {54 * time.Microsecond, "<1 ms"}, {time.Millisecond, "1 ms"},
		{12 * time.Millisecond, "12 ms"}, {999 * time.Millisecond, "999 ms"},
		{time.Second, "1.0 s"}, {1200 * time.Millisecond, "1.2 s"},
		{time.Minute, "1m 0s"}, {184 * time.Second, "3m 4s"},
	} {
		if got := ReportDuration(tt.d); got != tt.want {
			t.Errorf("ReportDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestReportSemanticClasses(t *testing.T) {
	if StatTone(0, "bad") != "" || StatTone(1, "bad") != "bad" {
		t.Fatal("only non-zero counts should carry a tone")
	}
	for _, severity := range []string{"high", "medium", "low", "unknown"} {
		if SeverityTone(severity) != "muted" {
			t.Errorf("severity %s is not neutral", severity)
		}
	}
	if SeverityTone("critical") != "bad" {
		t.Fatal("critical severity needs the red tone")
	}
	for _, status := range []string{"drift", "drifting", " DRIFT "} {
		if StatusClass(status) != "warn drift" {
			t.Errorf("signature missing for %q", status)
		}
	}
	if StatusClass("warning") != "warn" {
		t.Fatal("other warnings must not shift")
	}
	if Plural(1, "host", "hosts") != "host" || Plural(0, "host", "hosts") != "hosts" {
		t.Fatal("incorrect count noun")
	}
}
