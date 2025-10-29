package logger

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// Test HostResultSummary initialization
func TestHostResultSummaryStruct(t *testing.T) {
	hostResult := HostResultSummary{
		Hostname:     "web-01",
		TasksOK:      8,
		TasksChanged: 2,
		TasksFailed:  0,
		TasksSkipped: 1,
		Duration:     5 * time.Second,
		Status:       "CHANGED",
	}

	if hostResult.Hostname != "web-01" {
		t.Errorf("Expected hostname 'web-01', got '%s'", hostResult.Hostname)
	}
	if hostResult.TasksOK != 8 {
		t.Errorf("Expected 8 ok tasks, got %d", hostResult.TasksOK)
	}
	if hostResult.Status != "CHANGED" {
		t.Errorf("Expected status 'CHANGED', got '%s'", hostResult.Status)
	}
}

// Test ExecutionSummary with HostResults
func TestExecutionSummaryWithHostResults(t *testing.T) {
	summary := ExecutionSummary{
		TotalDuration: 2*time.Minute + 34*time.Second,
		PlayCount:     3,
		TaskCount:     21,
		SuccessCount:  18,
		ChangedCount:  2,
		FailedCount:   0,
		SkippedCount:  1,
		Stats:         nil,
		HostResults: map[string]HostResultSummary{
			"web-01": {
				Hostname:     "web-01",
				TasksOK:      8,
				TasksChanged: 2,
				TasksFailed:  0,
				TasksSkipped: 0,
				Status:       "CHANGED",
			},
			"web-02": {
				Hostname:     "web-02",
				TasksOK:      10,
				TasksChanged: 0,
				TasksFailed:  0,
				TasksSkipped: 0,
				Status:       "OK",
			},
			"db-01": {
				Hostname:     "db-01",
				TasksOK:      0,
				TasksChanged: 0,
				TasksFailed:  1,
				TasksSkipped: 1,
				Status:       "FAILED",
			},
		},
	}

	if len(summary.HostResults) != 3 {
		t.Errorf("Expected 3 hosts, got %d", len(summary.HostResults))
	}

	web01 := summary.HostResults["web-01"]
	if web01.TasksChanged != 2 {
		t.Errorf("Expected 2 changed tasks for web-01, got %d", web01.TasksChanged)
	}

	web02 := summary.HostResults["web-02"]
	if web02.Status != "OK" {
		t.Errorf("Expected status 'OK' for web-02, got '%s'", web02.Status)
	}

	db01 := summary.HostResults["db-01"]
	if db01.Status != "FAILED" {
		t.Errorf("Expected status 'FAILED' for db-01, got '%s'", db01.Status)
	}
}

// Test NormalFormatter with per-host breakdown
func TestNormalFormatterExecutionEndWithHostResults(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewNormalFormatter(false, &buf)

	summary := ExecutionSummary{
		TotalDuration: 2*time.Minute + 34*time.Second,
		PlayCount:     3,
		TaskCount:     21,
		SuccessCount:  18,
		ChangedCount:  2,
		FailedCount:   0,
		SkippedCount:  1,
		Stats:         nil,
		HostResults: map[string]HostResultSummary{
			"web-01": {
				Hostname:     "web-01",
				TasksOK:      8,
				TasksChanged: 2,
				TasksFailed:  0,
				TasksSkipped: 0,
				Status:       "CHANGED",
			},
			"web-02": {
				Hostname:     "web-02",
				TasksOK:      10,
				TasksChanged: 0,
				TasksFailed:  0,
				TasksSkipped: 0,
				Status:       "OK",
			},
		},
	}

	output := formatter.FormatExecutionEnd(summary)

	// Verify header
	if !strings.Contains(output, "EXECUTION COMPLETE") {
		t.Error("Expected 'EXECUTION COMPLETE' in output")
	}

	// Verify summary stats
	if !strings.Contains(output, "Total Duration:") {
		t.Error("Expected 'Total Duration:' in output")
	}
	if !strings.Contains(output, "Plays: 3") {
		t.Error("Expected 'Plays: 3' in output")
	}

	// Verify per-host breakdown
	if !strings.Contains(output, "Per-Host Results:") {
		t.Error("Expected 'Per-Host Results:' in output")
	}
	if !strings.Contains(output, "web-01") {
		t.Error("Expected 'web-01' in output")
	}
	if !strings.Contains(output, "web-02") {
		t.Error("Expected 'web-02' in output")
	}

	// Verify task counts
	if !strings.Contains(output, "OK:8") {
		t.Error("Expected 'OK:8' for web-01 in output")
	}
	if !strings.Contains(output, "CHANGED:2") {
		t.Error("Expected 'CHANGED:2' for web-01 in output")
	}

	t.Logf("NormalFormatter Output:\n%s", output)
}

// Test NormalFormatter without host results (backward compatibility)
func TestNormalFormatterExecutionEndWithoutHostResults(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewNormalFormatter(false, &buf)

	summary := ExecutionSummary{
		TotalDuration: 2*time.Minute + 34*time.Second,
		PlayCount:     3,
		TaskCount:     21,
		SuccessCount:  18,
		ChangedCount:  2,
		FailedCount:   0,
		SkippedCount:  1,
		Stats:         nil,
		HostResults:   make(map[string]HostResultSummary),
	}

	output := formatter.FormatExecutionEnd(summary)

	// Verify header and stats are present
	if !strings.Contains(output, "EXECUTION COMPLETE") {
		t.Error("Expected 'EXECUTION COMPLETE' in output")
	}
	if !strings.Contains(output, "Plays: 3") {
		t.Error("Expected 'Plays: 3' in output")
	}

	// Per-host section should not be present for empty HostResults
	if strings.Contains(output, "Per-Host Results:") {
		t.Error("Should not have 'Per-Host Results:' for empty HostResults")
	}

	t.Logf("NormalFormatter Output (no hosts):\n%s", output)
}

// Test VerboseFormatter with per-host breakdown
func TestVerboseFormatterExecutionEndWithHostResults(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewVerboseFormatter(false, &buf)

	summary := ExecutionSummary{
		TotalDuration: 2*time.Minute + 34*time.Second,
		PlayCount:     3,
		TaskCount:     21,
		SuccessCount:  18,
		ChangedCount:  2,
		FailedCount:   0,
		SkippedCount:  1,
		Stats:         nil,
		HostResults: map[string]HostResultSummary{
			"web-01": {
				Hostname:     "web-01",
				TasksOK:      8,
				TasksChanged: 2,
				TasksFailed:  0,
				TasksSkipped: 0,
				Status:       "CHANGED",
			},
			"web-02": {
				Hostname:     "web-02",
				TasksOK:      10,
				TasksChanged: 0,
				TasksFailed:  0,
				TasksSkipped: 0,
				Status:       "OK",
			},
		},
	}

	output := formatter.FormatExecutionEnd(summary)

	// Verify header
	if !strings.Contains(output, "EXECUTION COMPLETE") {
		t.Error("Expected 'EXECUTION COMPLETE' in output")
	}

	// Verify tree structure
	if !strings.Contains(output, "├─") && !strings.Contains(output, "└─") {
		t.Error("Expected tree structure with ├─ or └─")
	}

	// Verify per-host details
	if !strings.Contains(output, "Per-Host Details:") {
		t.Error("Expected 'Per-Host Details:' in output")
	}

	t.Logf("VerboseFormatter Output:\n%s", output)
}

// Test DebugFormatter with per-host breakdown
func TestDebugFormatterExecutionEndWithHostResults(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewDebugFormatter(&buf)

	summary := ExecutionSummary{
		TotalDuration: 2*time.Minute + 34*time.Second,
		PlayCount:     3,
		TaskCount:     21,
		SuccessCount:  18,
		ChangedCount:  2,
		FailedCount:   0,
		SkippedCount:  1,
		Stats:         nil,
		HostResults: map[string]HostResultSummary{
			"web-01": {
				Hostname:     "web-01",
				TasksOK:      8,
				TasksChanged: 2,
				TasksFailed:  0,
				TasksSkipped: 0,
				Status:       "CHANGED",
			},
		},
	}

	output := formatter.FormatExecutionEnd(summary)

	// JSON format check
	if !strings.Contains(output, "EXECUTION_END") {
		t.Error("Expected 'EXECUTION_END' phase in JSON output")
	}
	if !strings.Contains(output, "host_results") {
		t.Error("Expected 'host_results' field in JSON output")
	}
	if !strings.Contains(output, "web-01") {
		t.Error("Expected 'web-01' in JSON output")
	}

	t.Logf("DebugFormatter Output:\n%s", output)
}

// Test per-host status determination
func TestPerHostStatusDetermination(t *testing.T) {
	testCases := []struct {
		name           string
		tasksFailed    int
		tasksChanged   int
		expectedStatus string
	}{
		{"All OK", 0, 0, "OK"},
		{"Some changed", 0, 2, "CHANGED"},
		{"Some failed", 1, 0, "FAILED"},
		{"Both failed and changed", 1, 2, "FAILED"},
	}

	for _, tc := range testCases {
		// Determine status (same logic as in apply.go)
		status := "OK"
		if tc.tasksFailed > 0 {
			status = "FAILED"
		} else if tc.tasksChanged > 0 {
			status = "CHANGED"
		}

		if status != tc.expectedStatus {
			t.Errorf("%s: Expected status '%s', got '%s'", tc.name, tc.expectedStatus, status)
		}
	}
}

// Test host result accumulation for multi-play scenarios
func TestHostResultAccumulation(t *testing.T) {
	// Simulate 2 plays, both executing on the same host
	hostResults := make(map[string]HostResultSummary)

	// Play 1: web-01 has 5 OK, 1 CHANGED
	hostResults["web-01"] = HostResultSummary{
		Hostname:     "web-01",
		TasksOK:      5,
		TasksChanged: 1,
		TasksFailed:  0,
		TasksSkipped: 0,
		Status:       "CHANGED",
	}

	// Play 2: web-01 has 8 OK, 0 CHANGED (accumulate)
	existing := hostResults["web-01"]
	existing.TasksOK += 8
	existing.TasksChanged += 0
	hostResults["web-01"] = existing

	result := hostResults["web-01"]
	if result.TasksOK != 13 {
		t.Errorf("Expected 13 total OK tasks, got %d", result.TasksOK)
	}
	if result.TasksChanged != 1 {
		t.Errorf("Expected 1 total CHANGED task, got %d", result.TasksChanged)
	}
}
