package audit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type mockLogger struct{}

func (m *mockLogger) Debug(format string, args ...interface{})                                {}
func (m *mockLogger) Info(format string, args ...interface{})                                 {}
func (m *mockLogger) Warn(format string, args ...interface{})                                 {}
func (m *mockLogger) Error(format string, args ...interface{})                                {}
func (m *mockLogger) Fatal(format string, args ...interface{})                                {}
func (m *mockLogger) SetLevel(level string)                                                   {}
func (m *mockLogger) TaskStart(taskName, hostName string)                                     {}
func (m *mockLogger) TaskEnd(taskName, hostName string, changed, success bool)                {}
func (m *mockLogger) PlayStart(playName string, playIndex, totalPlays int)                    {}
func (m *mockLogger) PlayEnd(playName, hostName string, success bool, duration time.Duration) {}
func (m *mockLogger) Progress(completed, total int, currentTask, currentHost string)          {}
func (m *mockLogger) Retry(taskName, hostName string, attempt, maxAttempts int, delay time.Duration, err error) {
}
func (m *mockLogger) HostStats(format string, args ...interface{}) {}

func TestStorage_ListRecords_WithPagination(t *testing.T) {
	tmpDir := t.TempDir()
	storage := &Storage{
		path:   tmpDir,
		logger: &mockLogger{},
	}

	// Create 10 test records
	for i := 0; i < 10; i++ {
		record := &ExecutionRecord{
			ID:            string(rune(48 + i)),
			PlaybookPath:  "/path/playbook.yml",
			Status:        StatusSuccess,
			StartTime:     time.Now().Add(-time.Minute * time.Duration(10-i)),
			Duration:      float64(i),
			User:          "testuser",
			TotalTasks:    10,
			FailedTasks:   0,
			AffectedHosts: []string{"localhost"},
		}
		if err := storage.SaveRecord(record); err != nil {
			t.Errorf("SaveRecord failed: %v", err)
		}
	}

	// Test limit
	records, err := storage.ListRecords(FilterOptions{Limit: 5, Offset: 0})
	if err != nil {
		t.Errorf("ListRecords failed: %v", err)
	}

	if len(records) != 5 {
		t.Errorf("Expected 5 records with Limit=5, got %d", len(records))
	}

	// Test offset
	records, err = storage.ListRecords(FilterOptions{Limit: 5, Offset: 5})
	if err != nil {
		t.Errorf("ListRecords with offset failed: %v", err)
	}

	if len(records) != 5 {
		t.Errorf("Expected 5 records with Offset=5, got %d", len(records))
	}

	// Test offset beyond available records
	records, err = storage.ListRecords(FilterOptions{Limit: 5, Offset: 20})
	if err != nil {
		t.Errorf("ListRecords with large offset failed: %v", err)
	}

	if len(records) != 0 {
		t.Errorf("Expected 0 records with Offset=20, got %d", len(records))
	}
}

func TestStorage_ListRecords_UsesMetadataOnly(t *testing.T) {
	tmpDir := t.TempDir()
	storage := &Storage{
		path:   tmpDir,
		logger: &mockLogger{},
	}

	// Create a test record
	record := &ExecutionRecord{
		ID:            "test-record-1",
		PlaybookPath:  "/path/playbook.yml",
		Status:        StatusSuccess,
		StartTime:     time.Now(),
		Duration:      5.0,
		User:          "testuser",
		TotalTasks:    10,
		FailedTasks:   0,
		AffectedHosts: []string{"localhost"},
	}
	if err := storage.SaveRecord(record); err != nil {
		t.Errorf("SaveRecord failed: %v", err)
	}

	// Verify metadata file exists
	metadataPath := filepath.Join(tmpDir, "test-record-1", "metadata.json")
	if _, err := os.Stat(metadataPath); err != nil {
		t.Errorf("Metadata file not created: %v", err)
	}

	// Load metadata
	meta, err := storage.loadRecordMetadata("test-record-1")
	if err != nil {
		t.Errorf("loadRecordMetadata failed: %v", err)
	}

	// Verify metadata content
	if status, ok := meta["status"]; ok {
		statusStr, ok := status.(string)
		if !ok {
			t.Errorf("Expected status to be string, got %T", status)
		}
		if statusStr != string(StatusSuccess) {
			t.Errorf("Expected status to be %s, got %s", StatusSuccess, statusStr)
		}
	} else {
		t.Errorf("Expected status in metadata")
	}

	if totalTasks, ok := meta["total_tasks"]; ok {
		val, ok := totalTasks.(float64)
		if !ok {
			t.Errorf("Expected total_tasks to be float64, got %T", totalTasks)
		}
		if val != 10 {
			t.Errorf("Expected total_tasks to be 10, got %v", val)
		}
	} else {
		t.Errorf("Expected total_tasks in metadata")
	}
}

func TestStorage_LoadRecordMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	storage := &Storage{
		path:   tmpDir,
		logger: &mockLogger{},
	}

	// Create a test record
	record := &ExecutionRecord{
		ID:            "test-record-2",
		PlaybookPath:  "/test/playbook.yml",
		Status:        StatusFailure,
		StartTime:     time.Now(),
		Duration:      10.5,
		User:          "admin",
		TotalTasks:    20,
		FailedTasks:   5,
		AffectedHosts: []string{"host1", "host2"},
	}
	if err := storage.SaveRecord(record); err != nil {
		t.Errorf("SaveRecord failed: %v", err)
	}

	// Load and verify metadata
	meta, err := storage.loadRecordMetadata("test-record-2")
	if err != nil {
		t.Errorf("loadRecordMetadata failed: %v", err)
	}

	// Check all expected fields
	expectedFields := map[string]interface{}{
		"status":         string(StatusFailure),
		"total_tasks":    float64(20),
		"failed_tasks":   float64(5),
		"affected_hosts": float64(2),
		"user":           "admin",
	}

	for field, expectedVal := range expectedFields {
		if actualVal, ok := meta[field]; ok {
			if actualVal != expectedVal {
				t.Errorf("Field %s: expected %v, got %v", field, expectedVal, actualVal)
			}
		} else {
			t.Errorf("Expected field %s in metadata", field)
		}
	}
}

func TestStorage_MatchesMetadataFilter_ByStatus(t *testing.T) {
	metadata := map[string]interface{}{
		"status":       string(StatusSuccess),
		"total_tasks":  float64(10),
		"failed_tasks": float64(0),
	}

	// Should match
	if !matchesMetadataFilter(metadata, FilterOptions{Status: StatusSuccess}) {
		t.Errorf("Should match Status=success")
	}

	// Should not match
	if matchesMetadataFilter(metadata, FilterOptions{Status: StatusFailure}) {
		t.Errorf("Should not match Status=failure")
	}

	// Should match empty filter
	if !matchesMetadataFilter(metadata, FilterOptions{}) {
		t.Errorf("Should match empty filter")
	}
}

func TestStorage_GetStatistics_OptimizedForLargeDatasets(t *testing.T) {
	tmpDir := t.TempDir()
	storage := &Storage{
		path:   tmpDir,
		logger: &mockLogger{},
	}

	// Create 150 test records (more than the 100 limit for detailed analysis)
	for i := 0; i < 150; i++ {
		status := StatusSuccess
		if i%10 == 0 {
			status = StatusFailure
		}

		record := &ExecutionRecord{
			ID:            "record-" + string(rune(48+i%10)) + "-" + string(rune(48+i/10)),
			PlaybookPath:  "/path/playbook.yml",
			Status:        status,
			StartTime:     time.Now().Add(-time.Minute * time.Duration(150-i)),
			EndTime:       time.Now().Add(-time.Minute * time.Duration(149-i)),
			Duration:      1.0,
			User:          "testuser",
			TotalTasks:    10,
			FailedTasks:   0,
			AffectedHosts: []string{"localhost"},
		}

		if err := storage.SaveRecord(record); err != nil {
			t.Errorf("SaveRecord failed: %v", err)
		}
	}

	// Get statistics
	stats, err := storage.GetStatistics(FilterOptions{})
	if err != nil {
		t.Errorf("GetStatistics failed: %v", err)
	}

	// Verify basic statistics
	if stats.TotalExecutions != 150 {
		t.Errorf("Expected 150 total executions, got %d", stats.TotalExecutions)
	}

	// Should have both successes and failures
	if stats.SuccessfulRuns == 0 || stats.FailedRuns == 0 {
		t.Errorf("Expected both successful and failed runs")
	}

	if stats.TotalTasks != 1500 {
		t.Errorf("Expected 1500 total tasks (150 records * 10), got %d", stats.TotalTasks)
	}
}

func TestStorage_SaveAndLoad_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	storage := &Storage{
		path:   tmpDir,
		logger: &mockLogger{},
	}

	// Create and save a record
	originalRecord := &ExecutionRecord{
		ID:              "test-roundtrip",
		PlaybookPath:    "/path/playbook.yml",
		InventoryPath:   "/path/inventory",
		Status:          StatusSuccess,
		StartTime:       time.Now().Round(time.Second),
		EndTime:         time.Now().Round(time.Second).Add(time.Second * 10),
		Duration:        10.0,
		User:            "testuser",
		TotalTasks:      5,
		SuccessfulTasks: 5,
		FailedTasks:     0,
		SkippedTasks:    0,
		AffectedHosts:   []string{"host1", "host2"},
	}

	err := storage.SaveRecord(originalRecord)
	if err != nil {
		t.Errorf("SaveRecord failed: %v", err)
	}

	// Load it back
	loadedRecord, err := storage.LoadRecord("test-roundtrip")
	if err != nil {
		t.Errorf("LoadRecord failed: %v", err)
	}

	// Compare key fields
	if loadedRecord.ID != originalRecord.ID {
		t.Errorf("ID mismatch: %s vs %s", loadedRecord.ID, originalRecord.ID)
	}

	if loadedRecord.Status != originalRecord.Status {
		t.Errorf("Status mismatch: %s vs %s", loadedRecord.Status, originalRecord.Status)
	}

	if loadedRecord.PlaybookPath != originalRecord.PlaybookPath {
		t.Errorf("PlaybookPath mismatch: %s vs %s", loadedRecord.PlaybookPath, originalRecord.PlaybookPath)
	}

	if len(loadedRecord.AffectedHosts) != len(originalRecord.AffectedHosts) {
		t.Errorf("AffectedHosts count mismatch: %d vs %d", len(loadedRecord.AffectedHosts), len(originalRecord.AffectedHosts))
	}
}

func TestStorage_SortMetadata_ByStartTime(t *testing.T) {
	// Create test metadata records
	now := time.Now()
	records := []recordMeta{
		{
			ID: "record-1",
			Metadata: map[string]interface{}{
				"start_time": now.Add(time.Minute * 5).Format(time.RFC3339),
				"status":     StatusSuccess,
			},
		},
		{
			ID: "record-2",
			Metadata: map[string]interface{}{
				"start_time": now.Format(time.RFC3339),
				"status":     StatusSuccess,
			},
		},
		{
			ID: "record-3",
			Metadata: map[string]interface{}{
				"start_time": now.Add(time.Minute * 10).Format(time.RFC3339),
				"status":     StatusSuccess,
			},
		},
	}

	// Sort ascending
	sortMetadata(records, "start_time", "asc")

	// Verify order (should be 2, 1, 3)
	if records[0].ID != "record-2" {
		t.Errorf("Expected record-2 first (ascending), got %s", records[0].ID)
	}

	// Sort descending
	sortMetadata(records, "start_time", "desc")

	// Verify order (should be 3, 1, 2)
	if records[0].ID != "record-3" {
		t.Errorf("Expected record-3 first (descending), got %s", records[0].ID)
	}
}

func TestStorage_MetadataIncludesAffectedHostCount(t *testing.T) {
	tmpDir := t.TempDir()
	storage := &Storage{
		path:   tmpDir,
		logger: &mockLogger{},
	}

	// Create record with multiple affected hosts
	record := &ExecutionRecord{
		ID:            "test-hosts",
		PlaybookPath:  "/path/playbook.yml",
		Status:        StatusSuccess,
		StartTime:     time.Now(),
		Duration:      5.0,
		User:          "testuser",
		TotalTasks:    10,
		FailedTasks:   0,
		AffectedHosts: []string{"host1", "host2", "host3", "host4"},
	}

	if err := storage.SaveRecord(record); err != nil {
		t.Errorf("SaveRecord failed: %v", err)
	}

	// Load metadata
	meta, err := storage.loadRecordMetadata("test-hosts")
	if err != nil {
		t.Errorf("loadRecordMetadata failed: %v", err)
	}

	// Verify affected hosts count
	if hostCount, ok := meta["affected_hosts"]; ok {
		val, ok := hostCount.(float64)
		if !ok {
			t.Errorf("Expected affected_hosts to be float64, got %T", hostCount)
		}
		if val != 4 {
			t.Errorf("Expected affected_hosts count to be 4, got %v", val)
		}
	} else {
		t.Errorf("Expected affected_hosts in metadata")
	}
}

func TestStorage_ListRecords_PerformanceWithManyRecords(t *testing.T) {
	tmpDir := t.TempDir()
	storage := &Storage{
		path:   tmpDir,
		logger: &mockLogger{},
	}

	// Create 50 records
	for i := 0; i < 50; i++ {
		record := &ExecutionRecord{
			ID:            "perf-record-" + string(rune(48+i%10)) + "-" + string(rune(48+i/10)),
			PlaybookPath:  "/path/playbook.yml",
			Status:        StatusSuccess,
			StartTime:     time.Now(),
			Duration:      1.0,
			User:          "testuser",
			TotalTasks:    10,
			FailedTasks:   0,
			AffectedHosts: []string{"localhost"},
		}
		if err := storage.SaveRecord(record); err != nil {
			t.Errorf("SaveRecord failed: %v", err)
		}
	}

	// List with pagination should be fast
	start := time.Now()
	records, err := storage.ListRecords(FilterOptions{Limit: 10, Offset: 0})
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("ListRecords failed: %v", err)
	}

	if len(records) != 10 {
		t.Errorf("Expected 10 records, got %d", len(records))
	}

	// Should complete in reasonable time (under 1 second)
	if elapsed > time.Second {
		t.Errorf("ListRecords took too long: %v", elapsed)
	}
}
