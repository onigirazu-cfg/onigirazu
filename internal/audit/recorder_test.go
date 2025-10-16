package audit

import (
	"os"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/logger"
)

// createTestLogger creates a simple logger for testing
func createTestLogger() *logger.Logger {
	log, _ := logger.NewEnhancedLogger("warn", "text", os.Stdout)
	return log
}

// TestRecorderInitialization tests creating a new recorder
func TestRecorderInitialization(t *testing.T) {
	config := AuditConfig{
		Enabled:     true,
		StoragePath: t.TempDir(),
		MaxRecords:  100,
	}

	log := createTestLogger()
	recorder, err := NewRecorder(config, log)

	if err != nil {
		t.Fatalf("Failed to create recorder: %v", err)
	}

	if recorder == nil {
		t.Fatal("Recorder is nil")
	}
}

// TestRecorderDisabled tests that disabled recorder returns error
func TestRecorderDisabled(t *testing.T) {
	config := AuditConfig{
		Enabled: false,
	}

	log := createTestLogger()
	_, err := NewRecorder(config, log)

	if err == nil {
		t.Fatal("Expected error for disabled audit, but got nil")
	}
}

// TestStartExecution tests starting an execution record
func TestStartExecution(t *testing.T) {
	config := AuditConfig{
		Enabled:     true,
		StoragePath: t.TempDir(),
	}

	log := createTestLogger()
	recorder, _ := NewRecorder(config, log)

	id, err := recorder.StartExecution("test.yml", "inventory.yml", []string{})

	if err != nil {
		t.Fatalf("Failed to start execution: %v", err)
	}

	if id == "" {
		t.Fatal("Execution ID is empty")
	}

	// Verify we can't start another execution while one is running
	_, err = recorder.StartExecution("test2.yml", "inventory.yml", []string{})
	if err == nil {
		t.Fatal("Expected error when starting another execution while one is running")
	}

	defer recorder.Close()
}

// TestRecordTaskResult tests recording a task result
func TestRecordTaskResult(t *testing.T) {
	config := AuditConfig{
		Enabled:     true,
		StoragePath: t.TempDir(),
	}

	log := createTestLogger()
	recorder, _ := NewRecorder(config, log)
	recorder.StartExecution("test.yml", "inventory.yml", []string{})

	// Record a play before recording tasks
	recorder.RecordPlay("Setup", 0, []string{"localhost"})

	taskResult := TaskResult{
		Name:      "test task",
		Module:    "command",
		Status:    TaskStatusOk,
		Host:      "localhost",
		Duration:  1.5,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Changed:   false,
	}

	err := recorder.RecordTaskResult(taskResult)
	if err != nil {
		t.Fatalf("Failed to record task: %v", err)
	}

	record := recorder.GetCurrentRecord()
	if record.TotalTasks != 1 {
		t.Fatalf("Expected 1 task, got %d", record.TotalTasks)
	}

	defer recorder.Close()
}

// TestCompleteExecution tests completing an execution
func TestCompleteExecution(t *testing.T) {
	config := AuditConfig{
		Enabled:     true,
		StoragePath: t.TempDir(),
	}

	log := createTestLogger()
	recorder, _ := NewRecorder(config, log)

	id, _ := recorder.StartExecution("test.yml", "inventory.yml", []string{})

	// Add some task results
	for i := 0; i < 3; i++ {
		taskResult := TaskResult{
			Name:      "task_" + string(rune(i)),
			Module:    "command",
			Status:    TaskStatusOk,
			Host:      "localhost",
			Duration:  1.0,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		recorder.RecordTaskResult(taskResult)
	}

	recordID, err := recorder.CompleteExecution(StatusSuccess, 0, "")
	if err != nil {
		t.Fatalf("Failed to complete execution: %v", err)
	}

	if recordID != id {
		t.Fatalf("Expected record ID %s, got %s", id, recordID)
	}

	defer recorder.Close()
}

// TestRecordUnreachableHost tests recording unreachable hosts
func TestRecordUnreachableHost(t *testing.T) {
	config := AuditConfig{
		Enabled:     true,
		StoragePath: t.TempDir(),
	}

	log := createTestLogger()
	recorder, _ := NewRecorder(config, log)
	recorder.StartExecution("test.yml", "inventory.yml", []string{})

	err := recorder.RecordUnreachableHost("unreachable-host")
	if err != nil {
		t.Fatalf("Failed to record unreachable host: %v", err)
	}

	record := recorder.GetCurrentRecord()
	if len(record.UnreachableHosts) != 1 || record.UnreachableHosts[0] != "unreachable-host" {
		t.Fatal("Unreachable host not recorded correctly")
	}

	defer recorder.Close()
}

// TestSetVariables tests setting variables
func TestSetVariables(t *testing.T) {
	config := AuditConfig{
		Enabled:     true,
		StoragePath: t.TempDir(),
	}

	log := createTestLogger()
	recorder, _ := NewRecorder(config, log)
	recorder.StartExecution("test.yml", "inventory.yml", []string{})

	vars := map[string]interface{}{
		"app_name": "myapp",
		"version":  "1.0.0",
	}

	err := recorder.SetVariables(vars)
	if err != nil {
		t.Fatalf("Failed to set variables: %v", err)
	}

	record := recorder.GetCurrentRecord()
	if len(record.Variables) != 2 {
		t.Fatalf("Expected 2 variables, got %d", len(record.Variables))
	}

	defer recorder.Close()
}

// TestSetMetadata tests setting metadata
func TestSetMetadata(t *testing.T) {
	config := AuditConfig{
		Enabled:     true,
		StoragePath: t.TempDir(),
	}

	log := createTestLogger()
	recorder, _ := NewRecorder(config, log)
	recorder.StartExecution("test.yml", "inventory.yml", []string{})

	err := recorder.SetMetadata("custom_key", "custom_value")
	if err != nil {
		t.Fatalf("Failed to set metadata: %v", err)
	}

	record := recorder.GetCurrentRecord()
	if record.Metadata["custom_key"] != "custom_value" {
		t.Fatal("Metadata not set correctly")
	}

	defer recorder.Close()
}

// TestStorageAndRetrieval tests saving and loading records
func TestStorageAndRetrieval(t *testing.T) {
	storageDir := t.TempDir()
	config := AuditConfig{
		Enabled:     true,
		StoragePath: storageDir,
	}

	log := createTestLogger()
	recorder, _ := NewRecorder(config, log)

	id, _ := recorder.StartExecution("test.yml", "inventory.yml", []string{})

	taskResult := TaskResult{
		Name:      "test",
		Module:    "command",
		Status:    TaskStatusOk,
		Host:      "localhost",
		Duration:  1.0,
		StartTime: time.Now(),
		EndTime:   time.Now(),
	}
	recorder.RecordTaskResult(taskResult)

	recordID, err := recorder.CompleteExecution(StatusSuccess, 0, "")
	if err != nil {
		t.Fatalf("Failed to complete execution: %v", err)
	}

	recorder.Close()

	// Now load the record
	storage, _ := NewStorage(storageDir, log)
	defer storage.Close()

	loaded, err := storage.LoadRecord(recordID)
	if err != nil {
		t.Fatalf("Failed to load record: %v", err)
	}

	if loaded.ID != id {
		t.Fatalf("Record ID mismatch: expected %s, got %s", id, loaded.ID)
	}

	if loaded.Status != StatusSuccess {
		t.Fatalf("Record status mismatch: expected %s, got %s", StatusSuccess, loaded.Status)
	}
}

// TestListRecords tests listing records
func TestListRecords(t *testing.T) {
	storageDir := t.TempDir()
	config := AuditConfig{
		Enabled:     true,
		StoragePath: storageDir,
	}

	log := createTestLogger()

	// Create multiple records
	for i := 0; i < 3; i++ {
		recorder, _ := NewRecorder(config, log)
		recorder.StartExecution("test.yml", "inventory.yml", []string{})
		recorder.RecordTaskResult(TaskResult{
			Name:      "task",
			Module:    "command",
			Status:    TaskStatusOk,
			Host:      "localhost",
			Duration:  1.0,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		})
		recorder.CompleteExecution(StatusSuccess, 0, "")
		recorder.Close()
		time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	}

	// List records
	storage, _ := NewStorage(storageDir, log)
	defer storage.Close()

	records, err := storage.ListRecords(FilterOptions{Limit: 0, Offset: 0})
	if err != nil {
		t.Fatalf("Failed to list records: %v", err)
	}

	if len(records) < 3 {
		t.Fatalf("Expected at least 3 records, got %d", len(records))
	}
}

// TestFilterOptions tests filtering records
func TestFilterOptions(t *testing.T) {
	storageDir := t.TempDir()
	config := AuditConfig{
		Enabled:     true,
		StoragePath: storageDir,
	}

	log := createTestLogger()

	// Create records with different statuses
	recorder1, _ := NewRecorder(config, log)
	recorder1.StartExecution("test1.yml", "inventory.yml", []string{})
	recorder1.CompleteExecution(StatusSuccess, 0, "")
	recorder1.Close()

	recorder2, _ := NewRecorder(config, log)
	recorder2.StartExecution("test2.yml", "inventory.yml", []string{})
	recorder2.CompleteExecution(StatusFailure, 1, "Error")
	recorder2.Close()

	// List only successful records
	storage, _ := NewStorage(storageDir, log)
	defer storage.Close()

	records, err := storage.ListRecords(FilterOptions{
		Status: StatusSuccess,
	})

	if err != nil {
		t.Fatalf("Failed to list records: %v", err)
	}

	if len(records) == 0 {
		t.Fatal("No successful records found")
	}

	for _, record := range records {
		if record.Status != StatusSuccess {
			t.Fatalf("Found non-successful record in filtered results")
		}
	}
}

// TestReportGeneration tests report generation
func TestReportGeneration(t *testing.T) {
	storageDir := t.TempDir()
	config := AuditConfig{
		Enabled:     true,
		StoragePath: storageDir,
	}

	log := createTestLogger()
	recorder, _ := NewRecorder(config, log)
	recorder.StartExecution("test.yml", "inventory.yml", []string{})
	recorder.CompleteExecution(StatusSuccess, 0, "")
	recorder.Close()

	// List records and generate report
	storage, _ := NewStorage(storageDir, log)
	defer storage.Close()

	records, _ := storage.ListRecords(FilterOptions{})
	reporter := NewReporter(records)

	formats := []FormatType{FormatText, FormatJSON, FormatCSV, FormatHTML, FormatMarkdown}
	for _, format := range formats {
		report, err := reporter.Generate(format)
		if err != nil {
			t.Fatalf("Failed to generate %s report: %v", format, err)
		}

		if report == "" {
			t.Fatalf("Generated empty %s report", format)
		}
	}
}

// TestDeleteOldRecords tests deleting old records
func TestDeleteOldRecords(t *testing.T) {
	storageDir := t.TempDir()
	config := AuditConfig{
		Enabled:     true,
		StoragePath: storageDir,
	}

	log := createTestLogger()

	// Create a record
	recorder, _ := NewRecorder(config, log)
	recorder.StartExecution("test.yml", "inventory.yml", []string{})
	recorder.CompleteExecution(StatusSuccess, 0, "")
	recorder.Close()

	storage, _ := NewStorage(storageDir, log)

	// Delete records older than 0 days (should delete all records)
	deleted, err := storage.DeleteOldRecords(0)
	if err != nil {
		t.Fatalf("Failed to delete old records: %v", err)
	}

	if deleted != 1 {
		t.Fatalf("Expected 1 deleted record, got %d", deleted)
	}

	storage.Close()
}

// TestStatistics tests getting statistics
func TestStatistics(t *testing.T) {
	storageDir := t.TempDir()
	config := AuditConfig{
		Enabled:     true,
		StoragePath: storageDir,
	}

	log := createTestLogger()

	// Create records
	for i := 0; i < 2; i++ {
		recorder, _ := NewRecorder(config, log)
		recorder.StartExecution("test.yml", "inventory.yml", []string{})
		recorder.RecordPlay("Setup", 0, []string{"localhost"})

		recorder.RecordTaskResult(TaskResult{
			Name:      "task",
			Module:    "package",
			Status:    TaskStatusOk,
			Host:      "localhost",
			Duration:  1.0,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		})

		status := StatusSuccess
		if i == 1 {
			status = StatusFailure
		}

		recorder.CompleteExecution(status, 0, "")
		recorder.Close()
	}

	storage, _ := NewStorage(storageDir, log)
	defer storage.Close()

	stats, err := storage.GetStatistics(FilterOptions{})
	if err != nil {
		t.Fatalf("Failed to get statistics: %v", err)
	}

	if stats.TotalExecutions != 2 {
		t.Fatalf("Expected 2 total executions, got %d", stats.TotalExecutions)
	}

	if stats.SuccessfulRuns != 1 {
		t.Fatalf("Expected 1 successful run, got %d", stats.SuccessfulRuns)
	}

	if stats.FailedRuns != 1 {
		t.Fatalf("Expected 1 failed run, got %d", stats.FailedRuns)
	}
}

// BenchmarkRecordTaskResult benchmarks task recording
func BenchmarkRecordTaskResult(b *testing.B) {
	config := AuditConfig{
		Enabled:     true,
		StoragePath: os.TempDir(),
	}

	log := createTestLogger()
	recorder, _ := NewRecorder(config, log)
	recorder.StartExecution("test.yml", "inventory.yml", []string{})
	recorder.RecordPlay("Setup", 0, []string{"localhost"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		taskResult := TaskResult{
			Name:      "task",
			Module:    "command",
			Status:    TaskStatusOk,
			Host:      "localhost",
			Duration:  1.0,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		recorder.RecordTaskResult(taskResult)
	}

	recorder.Close()
}

// TestSensitiveVariableFiltering tests that sensitive variables are redacted
func TestSensitiveVariableFiltering(t *testing.T) {
	config := AuditConfig{
		Enabled:          true,
		StoragePath:      t.TempDir(),
		IncludeSensitive: false,
	}

	log := createTestLogger()
	recorder, _ := NewRecorder(config, log)
	recorder.StartExecution("test.yml", "inventory.yml", []string{})

	vars := map[string]interface{}{
		"app_name": "myapp",
		"password": "secret123",
		"api_key":  "key123",
	}

	recorder.SetVariables(vars)

	record := recorder.GetCurrentRecord()

	// Check that sensitive vars are redacted
	if record.Variables["password"] != "***REDACTED***" {
		t.Fatal("Password was not redacted")
	}

	if record.Variables["api_key"] != "***REDACTED***" {
		t.Fatal("API key was not redacted")
	}

	if record.Variables["app_name"] != "myapp" {
		t.Fatal("App name should not be redacted")
	}

	recorder.Close()
}
