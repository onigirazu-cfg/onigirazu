package comply

import (
	"bytes"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/brand"
)

func renderBrandedReport(t *testing.T, name string) string {
	t.Helper()
	var out bytes.Buffer
	(&Report{Profile: name}).WriteHTML(&out)
	return out.String()
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

func TestHTMLReportHeaderEscaping(t *testing.T) {
	name := `<script>alert("x")</script> & example`
	out := renderBrandedReport(t, name)
	if strings.Contains(out, "<script>") {
		t.Fatal("raw script in report")
	}
	want := brand.Header("Compliance: " + name)
	if !strings.Contains(out, want) {
		t.Fatalf("missing header %q", want)
	}
	start := strings.Index(out, `<div class="onz-head">`)
	end := strings.Index(out[start:], "</div>")
	header := out[start : start+end]
	if !strings.Contains(header, "&lt;script&gt;") || strings.Contains(header, "<script>") || strings.Contains(header, "&amp;lt;script") {
		t.Error("header contains raw or double-escaped input")
	}
}

func TestHTMLReportContentEscapingAndBadges(t *testing.T) {
	const payload = `<script>alert("x")</script>`
	var out bytes.Buffer
	(&Report{Profile: payload, Title: payload, Failed: 1,
		Hosts: map[string]*HostResult{payload: {Controls: []ControlResult{{
			ID: payload, Title: payload, Severity: "critical", Details: []string{payload}, Remediation: payload,
		}, {Severity: "high", OK: true}, {Severity: "medium", OK: true}, {Severity: payload}}}},
		Errors: map[string]string{payload: payload},
	}).WriteHTML(&out)
	if strings.Contains(out.String(), "<script>") {
		t.Fatal("raw script in report content")
	}
	for _, want := range []string{
		`<span class="onz-badge bad">critical</span>`,
		`<span class="onz-badge warn">high</span>`,
		`<span class="onz-badge warn">medium</span>`,
		`<span class="onz-badge muted">&lt;script&gt;`,
		`<span class="onz-badge bad">fail</span>`,
		`<span class="onz-badge ok">pass</span>`,
		`<span class="onz-badge bad">could not be checked</span>`,
		`<table class="onz-table">`, `<td class="num">`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
}
