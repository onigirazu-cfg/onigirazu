package audit

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateHTMLReportEscapesValues(t *testing.T) {
	const payload = `<script>alert(1)</script>"`
	records := []ExecutionRecord{{
		ID:           payload,
		PlaybookPath: payload,
		User:         payload,
		Status:       ExecutionStatus(payload),
		StartTime:    time.Now(),
	}}

	out, err := NewReporter(records).Generate(FormatHTML)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(out, "<script>") {
		t.Fatalf("raw <script> in HTML output:\n%s", out)
	}
	if strings.Contains(out, `class="status-<`) || strings.Contains(out, `class="status-&lt;`) {
		t.Errorf("record value reached a class attribute:\n%s", out)
	}
	if got := strings.Count(out, "&lt;script&gt;alert(1)&lt;/script&gt;&#34;"); got != 3 {
		t.Errorf("escaped payload count = %d, want 3", got)
	}
	if !strings.Contains(out, `<span class="onz-badge muted">`) {
		t.Errorf("unknown status should map to class onz-badge muted")
	}
}

func TestGenerateHTMLReportKnownStatusClass(t *testing.T) {
	out, err := NewReporter([]ExecutionRecord{{Status: StatusFailure, StartTime: time.Now()}}).Generate(FormatHTML)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `<span class="onz-badge bad">failure</span>`) {
		t.Errorf("known status lost its class:\n%s", out)
	}
}
