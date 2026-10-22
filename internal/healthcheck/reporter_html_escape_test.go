package healthcheck

import (
	"strings"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestReporterFormatHTMLEscapesValues(t *testing.T) {
	const payload = `<script>alert(1)</script>"`
	now := time.Now()
	report := &HealthCheckReport{
		Timestamp:     now,
		OverallStatus: HealthStatus(payload),
		HostReports: []HostHealthReport{{
			Host:   types.Host{Name: payload, Address: payload},
			Status: HealthStatus(payload),
			Checks: []HealthCheckResult{{
				CheckType: CheckType(payload),
				Status:    HealthStatus(payload),
				Message:   payload,
			}},
		}},
	}

	out := NewReporter(false).FormatHTML(report)

	if strings.Contains(out, "<script>") {
		t.Fatalf("raw <script> in HTML output:\n%s", out)
	}
	if strings.Contains(out, `class="<`) || strings.Contains(out, `class="&lt;`) {
		t.Errorf("record value reached a class attribute:\n%s", out)
	}
	if want := strings.Count(out, "&lt;script&gt;alert(1)&lt;/script&gt;&#34;"); want != 7 {
		t.Errorf("escaped payload count = %d, want 7", want)
	}
	if !strings.Contains(out, `<span class="onz-badge muted">`) {
		t.Errorf("unknown status should map to class muted")
	}
}
