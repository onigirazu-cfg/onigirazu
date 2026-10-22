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
	head := Head
	if !strings.HasPrefix(head, FaviconLink) || !strings.Contains(head, "prefers-color-scheme:dark") {
		t.Fatalf("head misses the favicon or the dark tokens: %s", head)
	}
	for _, bad := range []string{"http://", "https://", "url("} {
		if strings.Contains(strings.ReplaceAll(head, "http://www.w3.org/2000/svg", ""), bad) {
			t.Errorf("head references %q, reports must stay self-contained", bad)
		}
	}
}
