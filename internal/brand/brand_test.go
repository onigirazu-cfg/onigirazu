package brand

import (
	"strings"
	"testing"
)

func TestHeaderEscapesAndCarriesMark(t *testing.T) {
	h := Header(`<x> & "y"`)
	if !strings.Contains(h, Mark) || !strings.Contains(h, "<b>onigirazu</b>") {
		t.Fatalf("header misses the mark or wordmark: %s", h)
	}
	if strings.Contains(h, "<x>") || !strings.Contains(h, "&lt;x&gt; &amp; &#34;y&#34;") {
		t.Fatalf("report name not escaped: %s", h)
	}
}

func TestHeadIsOffline(t *testing.T) {
	head := ReportHead
	if !strings.Contains(head, FaviconLink) || !strings.Contains(head, "prefers-color-scheme:dark") {
		t.Fatalf("head misses the favicon or the dark tokens: %s", head)
	}
	for _, bad := range []string{"http://", "https://", "url("} {
		if strings.Contains(strings.ReplaceAll(head, "http://www.w3.org/2000/svg", ""), bad) {
			t.Errorf("head references %q, reports must stay self-contained", bad)
		}
	}
}

// Reports splice ReportHead and HeaderOpen into html/template text, where "{{"
// would start an action.
func TestMarkupHasNoTemplateActions(t *testing.T) {
	for name, s := range map[string]string{"ReportCSS": ReportCSS, "ReportHead": ReportHead, "HeaderOpen": HeaderOpen, "HeaderClose": HeaderClose} {
		if strings.Contains(s, "{{") {
			t.Errorf("%s contains a template action delimiter", name)
		}
	}
}

// Without color-scheme the browser keeps light scrollbars and form controls
// on dark report pages.
func TestCSSDeclaresColorScheme(t *testing.T) {
	if !strings.Contains(ReportCSS, "color-scheme:light dark") {
		t.Fatal("brand CSS must declare color-scheme: light dark")
	}
}

func TestTone(t *testing.T) {
	for status, want := range map[string]string{
		"ok": "ok", "success": "ok", "succeeded": "ok", "healthy": "ok",
		"pass": "ok", "passed": "ok", "fixed": "ok", "in sync": "ok", "in_sync": "ok",
		"warning": "warn", "warn": "warn", "drift": "warn", "drifting": "warn",
		"detected": "warn", "pending": "warn", "medium": "warn", "high": "warn",
		"skipped": "warn", "unreachable": "warn", "failure": "bad", "failed": "bad",
		"fail": "bad", "error": "bad", "critical": "bad", "bad": "bad", "timed_out": "bad",
		"changed": "changed", "running": "changed", "queued": "changed", "would change": "changed",
		" HEALTHY ": "ok", "low": "muted", "unknown": "muted", "": "muted", "<script>": "muted",
	} {
		t.Run(status, func(t *testing.T) {
			if got := Tone(status); got != want {
				t.Errorf("Tone(%q) = %q, want %q", status, got, want)
			}
		})
	}
}
