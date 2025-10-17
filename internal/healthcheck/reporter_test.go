package healthcheck

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNewReporter(t *testing.T) {
	tests := []struct {
		name     string
		useColor bool
	}{
		{
			name:     "with color",
			useColor: true,
		},
		{
			name:     "without color",
			useColor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewReporter(tt.useColor)
			if reporter == nil {
				t.Error("Expected reporter to be created")
			}
		})
	}
}

func TestReporterFormatStatus(t *testing.T) {
	reporter := NewReporter(false)

	tests := []struct {
		status   HealthStatus
		name     string
		expected string
	}{
		{StatusHealthy, "healthy", "healthy"},
		{StatusWarning, "warning", "warning"},
		{StatusCritical, "critical", "critical"},
		{StatusUnknown, "unknown", "unknown"},
		{HealthStatus("invalid"), "invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reporter.formatStatus(tt.status)

			if tt.expected != "" && !strings.Contains(result, tt.expected) {
				t.Errorf("expected %q in result, got %q", tt.expected, result)
			}
		})
	}
}

func TestReporterPrintReport(t *testing.T) {
	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	reporter := NewReporter(false)

	now := time.Now()
	report := &HealthCheckReport{
		Timestamp:     now,
		OverallStatus: StatusHealthy,
		HostReports: []HostHealthReport{
			{
				Host: types.Host{
					Name:    "host1",
					Address: "192.168.1.1",
				},
				Status:    StatusHealthy,
				Timestamp: now,
				Checks: []HealthCheckResult{
					{
						CheckType: CheckConnectivity,
						Host:      "host1",
						Status:    StatusHealthy,
						Message:   "Connected",
						Duration:  100 * time.Millisecond,
						Timestamp: now,
					},
				},
			},
		},
		Summary: map[string]int{
			"healthy": 1,
		},
		TotalDuration: 100 * time.Millisecond,
	}

	reporter.PrintReport(report)

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	if !strings.Contains(outputStr, "Health Check Report") {
		t.Error("Expected 'Health Check Report' in output")
	}

	if !strings.Contains(outputStr, "host1") {
		t.Error("Expected 'host1' in output")
	}
}

func TestReporterPrintHeader(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	reporter := NewReporter(false)

	now := time.Now()
	report := &HealthCheckReport{
		Timestamp:     now,
		OverallStatus: StatusHealthy,
		TotalDuration: 5 * time.Second,
	}

	reporter.printHeader(report)

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	if !strings.Contains(outputStr, "Health Check Report") {
		t.Error("Expected 'Health Check Report' in output")
	}

	if !strings.Contains(outputStr, "Overall Status") {
		t.Error("Expected 'Overall Status' in output")
	}

	if !strings.Contains(outputStr, "Total Duration") {
		t.Error("Expected 'Total Duration' in output")
	}
}

func TestReporterPrintHostReport(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	reporter := NewReporter(false)

	now := time.Now()
	hostReport := &HostHealthReport{
		Host: types.Host{
			Name:    "test-host",
			Address: "192.168.1.100",
		},
		Status:    StatusHealthy,
		Timestamp: now,
		Checks: []HealthCheckResult{
			{
				CheckType: CheckConnectivity,
				Status:    StatusHealthy,
				Message:   "Host is reachable",
				Duration:  100 * time.Millisecond,
				Timestamp: now,
			},
		},
		Summary: map[string]int{
			"healthy": 1,
		},
	}

	reporter.printHostReport(hostReport)

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	if !strings.Contains(outputStr, "test-host") {
		t.Error("Expected 'test-host' in output")
	}

	if !strings.Contains(outputStr, "192.168.1.100") {
		t.Error("Expected IP address in output")
	}
}

func TestReporterFormatJSON(t *testing.T) {
	reporter := NewReporter(false)

	now := time.Now()
	report := &HealthCheckReport{
		Timestamp:     now,
		OverallStatus: StatusHealthy,
		HostReports: []HostHealthReport{
			{
				Host: types.Host{
					Name:    "host1",
					Address: "192.168.1.1",
				},
				Status:    StatusHealthy,
				Timestamp: now,
			},
		},
		Summary: map[string]int{
			"healthy": 1,
		},
	}

	result, err := reporter.FormatJSON(report)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(result, "host1") {
		t.Error("Expected 'host1' in JSON output")
	}

	if !strings.Contains(result, "healthy") {
		t.Error("Expected 'healthy' in JSON output")
	}

	// Verify it's valid JSON
	if !strings.Contains(result, "{") || !strings.Contains(result, "}") {
		t.Error("Expected valid JSON structure")
	}
}

func TestReporterFormatCSV(t *testing.T) {
	reporter := NewReporter(false)

	now := time.Now()
	report := &HealthCheckReport{
		Timestamp:     now,
		OverallStatus: StatusHealthy,
		HostReports: []HostHealthReport{
			{
				Host: types.Host{
					Name:    "host1",
					Address: "192.168.1.1",
				},
				Status:    StatusHealthy,
				Timestamp: now,
				Checks: []HealthCheckResult{
					{
						CheckType: CheckConnectivity,
						Status:    StatusHealthy,
						Duration:  100 * time.Millisecond,
						Timestamp: now,
					},
				},
			},
		},
	}

	result := reporter.FormatCSV(report)

	if !strings.Contains(result, "host1") {
		t.Error("Expected 'host1' in CSV output")
	}

	if !strings.Contains(result, "connectivity") {
		t.Error("Expected 'connectivity' in CSV output")
	}

	if !strings.Contains(result, "Host,CheckType") {
		t.Error("Expected CSV header in output")
	}
}

func TestReporterFormatHTML(t *testing.T) {
	reporter := NewReporter(false)

	now := time.Now()
	report := &HealthCheckReport{
		Timestamp:     now,
		OverallStatus: StatusHealthy,
		HostReports: []HostHealthReport{
			{
				Host: types.Host{
					Name:    "host1",
					Address: "192.168.1.1",
				},
				Status:    StatusHealthy,
				Timestamp: now,
			},
		},
	}

	result := reporter.FormatHTML(report)

	if !strings.Contains(result, "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype")
	}

	if !strings.Contains(result, "host1") {
		t.Error("Expected 'host1' in HTML output")
	}
}

func TestReporterFormatMarkdown(t *testing.T) {
	reporter := NewReporter(false)

	now := time.Now()
	report := &HealthCheckReport{
		Timestamp:     now,
		OverallStatus: StatusHealthy,
		HostReports: []HostHealthReport{
			{
				Host: types.Host{
					Name:    "host1",
					Address: "192.168.1.1",
				},
				Status:    StatusHealthy,
				Timestamp: now,
			},
		},
	}

	result := reporter.FormatMarkdown(report)

	if !strings.Contains(result, "# Health Check Report") {
		t.Error("Expected markdown header")
	}

	if !strings.Contains(result, "host1") {
		t.Error("Expected 'host1' in markdown output")
	}
}

func TestReporterShouldFormatWithColor(t *testing.T) {
	tests := []struct {
		name      string
		useColor  bool
		status    HealthStatus
		expectLen bool
	}{
		{
			name:      "color enabled",
			useColor:  true,
			status:    StatusHealthy,
			expectLen: true,
		},
		{
			name:      "color disabled",
			useColor:  false,
			status:    StatusHealthy,
			expectLen: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewReporter(tt.useColor)
			result := reporter.formatStatus(tt.status)

			if tt.expectLen && len(result) == 0 {
				t.Error("Expected non-empty result")
			}
		})
	}
}

func TestReporterPrintSummary(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	reporter := NewReporter(false)

	report := &HealthCheckReport{
		Summary: map[string]int{
			"healthy":  5,
			"warning":  2,
			"critical": 1,
		},
		Statistics: HealthCheckStatistics{
			TotalHosts:    8,
			HealthyHosts:  5,
			WarningHosts:  2,
			CriticalHosts: 1,
		},
	}

	reporter.printSummary(report)

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	outputStr := string(output)

	if !strings.Contains(outputStr, "Summary") {
		t.Error("Expected 'Summary' in output")
	}
}

func TestReporterMultipleHosts(t *testing.T) {
	reporter := NewReporter(false)

	now := time.Now()
	report := &HealthCheckReport{
		Timestamp:     now,
		OverallStatus: StatusWarning,
		HostReports: []HostHealthReport{
			{
				Host:   types.Host{Name: "host1", Address: "192.168.1.1"},
				Status: StatusHealthy,
				Checks: []HealthCheckResult{
					{CheckType: CheckConnectivity, Status: StatusHealthy, Duration: 100 * time.Millisecond, Timestamp: now},
				},
			},
			{
				Host:   types.Host{Name: "host2", Address: "192.168.1.2"},
				Status: StatusWarning,
				Checks: []HealthCheckResult{
					{CheckType: CheckDiskSpace, Status: StatusWarning, Duration: 200 * time.Millisecond, Timestamp: now},
				},
			},
			{
				Host:   types.Host{Name: "host3", Address: "192.168.1.3"},
				Status: StatusCritical,
				Checks: []HealthCheckResult{
					{CheckType: CheckMemory, Status: StatusCritical, Duration: 300 * time.Millisecond, Timestamp: now},
				},
			},
		},
		Summary: map[string]int{
			"healthy":  1,
			"warning":  1,
			"critical": 1,
		},
	}

	result := reporter.FormatCSV(report)

	if !strings.Contains(result, "host1") {
		t.Error("Expected 'host1' in output")
	}
	if !strings.Contains(result, "host2") {
		t.Error("Expected 'host2' in output")
	}
	if !strings.Contains(result, "host3") {
		t.Error("Expected 'host3' in output")
	}
}

func TestReporterHandlesEmptyReport(t *testing.T) {
	reporter := NewReporter(false)

	report := &HealthCheckReport{
		Timestamp:     time.Now(),
		OverallStatus: StatusUnknown,
		HostReports:   []HostHealthReport{},
		Summary:       map[string]int{},
	}

	result := reporter.FormatCSV(report)

	if result == "" {
		t.Error("Expected non-empty result even for empty report")
	}
}

// ============================================================================
// Extended Reporter Tests for Edge Cases and Complex Data
// ============================================================================

// TestReporterFormatJSONWithComplexData tests JSON formatting with complex data
func TestReporterFormatJSONWithComplexData(t *testing.T) {
	reporter := NewReporter(false)

	now := time.Now()
	report := &HealthCheckReport{
		Timestamp:     now,
		OverallStatus: StatusWarning,
		HostReports: []HostHealthReport{
			{
				Host:   types.Host{Name: "host1", Address: "192.168.1.1", User: "admin", Port: 2222},
				Status: StatusHealthy,
				Checks: []HealthCheckResult{
					{
						CheckType: CheckDiskSpace,
						Status:    StatusHealthy,
						Message:   "Disk usage 45%",
						Details: map[string]interface{}{
							"disk_usage_percent": 45,
							"threshold":          80,
						},
						Duration:  150 * time.Millisecond,
						Timestamp: now,
					},
				},
				Summary:  map[string]int{string(StatusHealthy): 1},
				Duration: 150 * time.Millisecond,
			},
		},
		Summary: map[string]int{"healthy": 1},
		Statistics: HealthCheckStatistics{
			TotalHosts:      1,
			HealthyHosts:    1,
			AverageDuration: 150.0,
		},
	}

	result, err := reporter.FormatJSON(report)

	if err != nil {
		t.Fatalf("FormatJSON failed: %v", err)
	}
	if !strings.Contains(result, "disk_usage_percent") {
		t.Error("Expected complex details in JSON")
	}
	if !strings.Contains(result, "admin") {
		t.Error("Expected username in JSON")
	}
	if !strings.Contains(result, "2222") {
		t.Error("Expected port number in JSON")
	}
}

// TestReporterFormatHTMLWithMultipleStatus tests HTML formatting with various statuses
func TestReporterFormatHTMLWithMultipleStatus(t *testing.T) {
	reporter := NewReporter(true) // With color support

	now := time.Now()
	report := &HealthCheckReport{
		Timestamp:     now,
		OverallStatus: StatusCritical,
		HostReports: []HostHealthReport{
			{
				Host:   types.Host{Name: "healthy-host", Address: "192.168.1.1"},
				Status: StatusHealthy,
				Checks: []HealthCheckResult{
					{CheckType: CheckConnectivity, Status: StatusHealthy, Duration: 100 * time.Millisecond, Timestamp: now},
				},
			},
			{
				Host:   types.Host{Name: "warning-host", Address: "192.168.1.2"},
				Status: StatusWarning,
				Checks: []HealthCheckResult{
					{CheckType: CheckDiskSpace, Status: StatusWarning, Duration: 200 * time.Millisecond, Timestamp: now},
				},
			},
			{
				Host:   types.Host{Name: "critical-host", Address: "192.168.1.3"},
				Status: StatusCritical,
				Checks: []HealthCheckResult{
					{CheckType: CheckMemory, Status: StatusCritical, Duration: 300 * time.Millisecond, Timestamp: now},
				},
			},
		},
		Summary: map[string]int{
			"healthy":  1,
			"warning":  1,
			"critical": 1,
		},
	}

	result := reporter.FormatHTML(report)

	if !strings.Contains(result, "healthy-host") {
		t.Error("Expected 'healthy-host' in HTML")
	}
	if !strings.Contains(result, "warning-host") {
		t.Error("Expected 'warning-host' in HTML")
	}
	if !strings.Contains(result, "critical-host") {
		t.Error("Expected 'critical-host' in HTML")
	}
	if !strings.Contains(result, "<table>") || !strings.Contains(result, "</table>") {
		t.Error("Expected HTML table structure")
	}
}

// TestReporterFormatMarkdownWithStats tests Markdown formatting with statistics
func TestReporterFormatMarkdownWithStats(t *testing.T) {
	reporter := NewReporter(false)

	now := time.Now()
	report := &HealthCheckReport{
		Timestamp:     now,
		OverallStatus: StatusHealthy,
		HostReports: []HostHealthReport{
			{
				Host:   types.Host{Name: "host1", Address: "192.168.1.1"},
				Status: StatusHealthy,
				Checks: []HealthCheckResult{
					{CheckType: CheckConnectivity, Status: StatusHealthy, Duration: 100 * time.Millisecond, Timestamp: now},
					{CheckType: CheckDiskSpace, Status: StatusHealthy, Duration: 50 * time.Millisecond, Timestamp: now},
				},
			},
		},
		Summary: map[string]int{"healthy": 2},
		Statistics: HealthCheckStatistics{
			TotalHosts:   1,
			HealthyHosts: 1,
			ChecksPerHost: map[CheckType]int{
				CheckConnectivity: 1,
				CheckDiskSpace:    1,
			},
			AverageDuration: 75.0,
		},
	}

	result := reporter.FormatMarkdown(report)

	if !strings.Contains(result, "# Health Check Report") {
		t.Error("Expected header in markdown")
	}
	if !strings.Contains(result, "host1") {
		t.Error("Expected host name in markdown")
	}
}
