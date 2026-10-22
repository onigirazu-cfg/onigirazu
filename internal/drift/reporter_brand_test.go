package drift

import (
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/brand"
)

func renderBrandedReport(t *testing.T, name string) string {
	t.Helper()
	out, err := NewReporter(&DriftConfig{}).generateHTMLReport(&DriftReport{ID: name, PlaybookID: name})
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
	want := brand.Header("Drift report")
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

func TestHTMLReportContentEscapingAndDiff(t *testing.T) {
	const payload = `<script>alert("x")</script>`
	out, err := NewReporter(&DriftConfig{}).generateHTMLReport(&DriftReport{
		ID: payload, SnapshotID: payload, PlaybookID: payload,
		DriftsByHost: map[string]int{payload: 1},
		DriftsByType: map[DriftType]int{DriftType(payload): 1},
		Items: []DriftItem{{
			Type: DriftType(payload), Host: payload, Resource: payload, Message: payload,
			Severity: SeverityCritical, Status: StatusFailed, CanAutoFix: true,
			Diff: "--- old\n+++ new\n@@ change @@\n-" + payload + "\n+" + payload + "\n context\n\n tail\n",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "<script>") {
		t.Fatal("raw script in report content")
	}
	for _, want := range []string{
		`<span class="onz-badge bad">critical</span>`,
		`<span class="onz-badge bad">failed</span>`,
		`<span class="onz-badge ok">auto-fixable</span>`,
		`<pre class="onz-diff">`, `<span class="hdr">--- old</span>`,
		`<span class="hdr">&#43;&#43;&#43; new</span>`, `<span class="hunk">@@ change @@</span>`,
		`<span class="del">-&lt;script&gt;`, `<span class="add">&#43;&lt;script&gt;`,
		`<span class=""> context</span><span class=""></span><span class=""> tail</span>`,
		`<table class="onz-table">`, `<td class="num">1</td>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}
