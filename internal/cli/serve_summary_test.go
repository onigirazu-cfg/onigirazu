package cli

import (
	"strings"
	"testing"
)

func TestFirstFailure(t *testing.T) {
	if got := firstFailure(nil, nil); got != "" {
		t.Fatalf("no errors: %q", got)
	}
	errs := map[string][]DriftItem{
		"web2": {{Task: "Explain", Detail: "SAFEGUARD STOP: Stopped by the veeam safeguard: job due.\n{\"ok\": false}"}},
		"web1": {{Task: "Install", Detail: ""}},
	}
	if got := firstFailure(errs, nil); got != "web1: Install failed" {
		t.Fatalf("host order or empty detail: %q", got)
	}
	delete(errs, "web1")
	if got := firstFailure(errs, nil); got != "web2: SAFEGUARD STOP: Stopped by the veeam safeguard: job due." {
		t.Fatalf("first line: %q", got)
	}
	errs["web2"][0].Detail = strings.Repeat("é", 300)
	if got := []rune(firstFailure(errs, nil)); len(got) != len("web2: ")+200 || got[len(got)-1] != '…' {
		t.Fatalf("truncation: %d runes", len(got))
	}
}

func TestFirstFailureMasksBeforeCutting(t *testing.T) {
	secret := "super-secret-token-value"
	errs := map[string][]DriftItem{"web1": {{Task: "t", Detail: strings.Repeat("x", 190) + secret}}}
	got := firstFailure(errs, strings.NewReplacer(secret, "***"))
	if strings.Contains(got, "super-sec") || !strings.Contains(got, "***") {
		t.Fatalf("secret not masked before cutting: %q", got)
	}
}

func TestReportFailure(t *testing.T) {
	if got := reportFailure(nil, nil); got != "" {
		t.Fatalf("empty: %q", got)
	}
	if got := reportFailure([]byte(`{"errors":{"h":"could not connect"}}`), nil); got != "" {
		t.Fatalf("verify-style report must be ignored: %q", got)
	}
	got := reportFailure([]byte(`{"errors":{"h":[{"task":"x","detail":"SAFEGUARD STOP: why\nmore"}]}}`), nil)
	if got != "h: SAFEGUARD STOP: why" {
		t.Fatalf("drift report: %q", got)
	}
}

func TestFirstFailureMasksBeforeTrimming(t *testing.T) {
	errs := map[string][]DriftItem{"h": {{Task: "t", Detail: "authentication failed: token-value "}}}
	got := firstFailure(errs, strings.NewReplacer("token-value ", "***"))
	if strings.Contains(got, "token-value") {
		t.Fatalf("secret leaked after trimming: %q", got)
	}
}
