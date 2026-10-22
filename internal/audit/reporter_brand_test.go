package audit

import (
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/brand"
)

func renderBrandedReport(t *testing.T, name string) string {
	t.Helper()
	out, err := NewReporter([]ExecutionRecord{{PlaybookPath: name}}).generateHTMLReport()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestHTMLReportBrand(t *testing.T) {
	out := renderBrandedReport(t, "example")
	for _, want := range []string{brand.FaviconLink, brand.Mark, `class="onz-head"`, `class="onz-page"`, `class="onz-stats"`, `class="onz-foot"`, brand.ReportHead, "prefers-color-scheme:dark", `<meta charset=`, `<meta name="viewport"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if got := strings.Count(out, "<style>"); got != 1 {
		t.Errorf("style block count = %d, want 1", got)
	}
	if strings.Index(out, `class="onz-head"`) > strings.Index(out, `<h1 class="onz-title">`) {
		t.Error("brand header must precede heading")
	}
}

// The report name is fixed; user-provided report data must not replace it.
func TestHTMLReportHeaderEscaping(t *testing.T) {
	name := `<script>alert("x")</script> & example`
	out := renderBrandedReport(t, name)
	if strings.Contains(out, "<script>") {
		t.Fatal("raw script in report")
	}
	want := brand.Header("Audit report")
	if !strings.Contains(out, want) {
		t.Fatalf("missing header %q", want)
	}
	start := strings.Index(out, `<div class="onz-head">`)
	end := strings.Index(out[start:], "</div>")
	header := out[start : start+end]
	if strings.Contains(header, "<script>") || strings.Contains(header, "&amp;lt;script") {
		t.Error("header contains raw or double-escaped input")
	}
}
