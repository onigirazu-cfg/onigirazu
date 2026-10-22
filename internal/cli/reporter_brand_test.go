package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/brand"
)

func renderBrandedReport(t *testing.T, name string) string {
	t.Helper()
	var out bytes.Buffer
	if err := writeDriftHTML(&out, &DriftReport{Playbook: name}); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestHTMLReportBrand(t *testing.T) {
	out := renderBrandedReport(t, "example")
	for _, want := range []string{brand.FaviconLink, brand.Mark, `class="onz-head"`, "prefers-color-scheme:dark", `<meta charset=`, `<meta name="viewport"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Index(out, brand.FaviconLink) > strings.Index(out, "<style>") {
		t.Error("brand head must precede report styles")
	}
	if strings.Index(out, `class="onz-head"`) > strings.Index(out, "<h1>") {
		t.Error("brand header must precede heading")
	}
}

func TestHTMLReportHeaderEscaping(t *testing.T) {
	name := `<script>alert("x")</script> & example`
	out := renderBrandedReport(t, name)
	want := brand.Header("Drift: " + name)
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

func TestPlanHTMLReportHeaderEscaping(t *testing.T) {
	name := `<script>alert("x")</script> & example`
	var out bytes.Buffer
	if err := writeDriftHTML(&out, &DriftReport{Plan: true, Playbook: name}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), brand.Header("Plan: "+name)) {
		t.Fatal("missing escaped plan header")
	}
	if strings.Contains(out.String(), "<script>") {
		t.Fatal("unescaped playbook")
	}
}
