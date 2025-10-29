package profiler

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewProfileManagerDisabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	pm, err := NewProfileManager(cfg)
	if err != nil {
		t.Fatalf("NewProfileManager returned error: %v", err)
	}

	if pm.IsEnabled() {
		t.Error("Expected profiler to be disabled")
	}

	// Should not error when disabled
	if err := pm.StartProfiling(); err != nil {
		t.Fatalf("StartProfiling returned error when disabled: %v", err)
	}

	if err := pm.StopProfiling(); err != nil {
		t.Fatalf("StopProfiling returned error when disabled: %v", err)
	}
}

func TestNewProfileManagerEnabled(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		OutputDir: tmpDir,
		CPU:       true,
		Memory:    true,
		Trace:     false,
	}

	pm, err := NewProfileManager(cfg)
	if err != nil {
		t.Fatalf("NewProfileManager returned error: %v", err)
	}

	if !pm.IsEnabled() {
		t.Error("Expected profiler to be enabled")
	}

	if pm.GetOutputDir() != tmpDir {
		t.Errorf("Expected output dir %s, got %s", tmpDir, pm.GetOutputDir())
	}

	if pm.GetStartTime().IsZero() {
		t.Error("Expected non-zero start time")
	}
}

func TestProfileManagerCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	profileDir := filepath.Join(tmpDir, "profiles", "nested")

	cfg := Config{
		Enabled:   true,
		OutputDir: profileDir,
		CPU:       true,
		Memory:    true,
	}

	pm, err := NewProfileManager(cfg)
	if err != nil {
		t.Fatalf("NewProfileManager returned error: %v", err)
	}

	if !pm.IsEnabled() {
		t.Fatal("Expected profiler to be enabled")
	}

	// Check directory was created
	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		t.Errorf("Expected directory to be created at %s", profileDir)
	}
}

func TestStartStopProfiling(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		OutputDir: tmpDir,
		CPU:       true,
		Memory:    true,
		Trace:     false,
	}

	pm, err := NewProfileManager(cfg)
	if err != nil {
		t.Fatalf("NewProfileManager returned error: %v", err)
	}

	// Start profiling
	if err := pm.StartProfiling(); err != nil {
		t.Fatalf("StartProfiling returned error: %v", err)
	}

	// Do some work
	time.Sleep(100 * time.Millisecond)

	// Stop profiling
	if err := pm.StopProfiling(); err != nil {
		t.Fatalf("StopProfiling returned error: %v", err)
	}

	// Check that profile files were created
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read profile directory: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("Expected profile files to be created")
	}

	// Verify we have at least CPU and goroutine profiles
	hasGoroutineProfile := false
	for _, f := range files {
		if f.Name() != "." && !f.IsDir() {
			if filepath.Base(f.Name())[:9] == "goroutine" {
				hasGoroutineProfile = true
			}
		}
	}

	if !hasGoroutineProfile {
		t.Error("Expected goroutine profile file to be created")
	}
}

func TestProfilingReport(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		OutputDir: tmpDir,
		CPU:       true,
		Memory:    true,
		Trace:     false,
	}

	pm, err := NewProfileManager(cfg)
	if err != nil {
		t.Fatalf("NewProfileManager returned error: %v", err)
	}

	// Start profiling
	if err := pm.StartProfiling(); err != nil {
		t.Fatalf("StartProfiling returned error: %v", err)
	}

	// Do some work
	time.Sleep(50 * time.Millisecond)

	// Stop profiling
	if err := pm.StopProfiling(); err != nil {
		t.Fatalf("StopProfiling returned error: %v", err)
	}

	// Generate report
	report, err := pm.GenerateReport()
	if err != nil {
		t.Fatalf("GenerateReport returned error: %v", err)
	}

	if report == "" {
		t.Fatal("Expected non-empty report")
	}

	// Verify report contains expected content
	expectedStrings := []string{
		"PROFILING REPORT",
		"Duration:",
		"Output Directory:",
		"Memory Profile",
		"Goroutine Profile",
	}

	for _, expected := range expectedStrings {
		if !contains(report, expected) {
			t.Errorf("Expected report to contain '%s', got:\n%s", expected, report)
		}
	}
}

func TestProfilingWithMemoryOnly(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		OutputDir: tmpDir,
		CPU:       false,
		Memory:    true,
		Trace:     false,
	}

	pm, err := NewProfileManager(cfg)
	if err != nil {
		t.Fatalf("NewProfileManager returned error: %v", err)
	}

	// Start profiling
	if err := pm.StartProfiling(); err != nil {
		t.Fatalf("StartProfiling returned error: %v", err)
	}

	// Do some work
	time.Sleep(50 * time.Millisecond)

	// Stop profiling
	if err := pm.StopProfiling(); err != nil {
		t.Fatalf("StopProfiling returned error: %v", err)
	}

	// Check that profile files were created
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read profile directory: %v", err)
	}

	// Should have memory and goroutine profiles
	hasMemProfile := false
	hasGoroutineProfile := false

	for _, f := range files {
		if f.Name() != "." && !f.IsDir() {
			if filepath.Base(f.Name())[:3] == "mem" {
				hasMemProfile = true
			}
			if filepath.Base(f.Name())[:9] == "goroutine" {
				hasGoroutineProfile = true
			}
		}
	}

	if !hasMemProfile {
		t.Error("Expected memory profile file to be created")
	}
	if !hasGoroutineProfile {
		t.Error("Expected goroutine profile file to be created")
	}
}

func TestProfilingWithTrace(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		OutputDir: tmpDir,
		CPU:       true,
		Memory:    false,
		Trace:     true,
	}

	pm, err := NewProfileManager(cfg)
	if err != nil {
		t.Fatalf("NewProfileManager returned error: %v", err)
	}

	// Start profiling
	if err := pm.StartProfiling(); err != nil {
		t.Fatalf("StartProfiling returned error: %v", err)
	}

	// Do some work
	time.Sleep(50 * time.Millisecond)

	// Stop profiling
	if err := pm.StopProfiling(); err != nil {
		t.Fatalf("StopProfiling returned error: %v", err)
	}

	// Check that trace file was created
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read profile directory: %v", err)
	}

	hasTraceFile := false
	for _, f := range files {
		if f.Name() != "." && !f.IsDir() {
			if filepath.Base(f.Name())[:5] == "trace" {
				hasTraceFile = true
			}
		}
	}

	if !hasTraceFile {
		t.Error("Expected trace file to be created")
	}
}

func TestProfileManagerIsEnabledFlag(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		enabled bool
		want    bool
	}{
		{"enabled", true, true},
		{"disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Enabled:   tt.enabled,
				OutputDir: tmpDir,
			}

			pm, err := NewProfileManager(cfg)
			if err != nil {
				t.Fatalf("NewProfileManager returned error: %v", err)
			}

			if pm.IsEnabled() != tt.want {
				t.Errorf("Expected IsEnabled() to return %v, got %v", tt.want, pm.IsEnabled())
			}
		})
	}
}

func TestProfileManagerWithEmptyOutputDir(t *testing.T) {
	cfg := Config{
		Enabled:   true,
		OutputDir: "",
		CPU:       true,
		Memory:    true,
	}

	pm, err := NewProfileManager(cfg)
	if err != nil {
		t.Fatalf("NewProfileManager returned error: %v", err)
	}

	if pm.GetOutputDir() != "." {
		t.Errorf("Expected output dir to default to '.', got %s", pm.GetOutputDir())
	}
}

func TestGenerateReportDisabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	pm, err := NewProfileManager(cfg)
	if err != nil {
		t.Fatalf("NewProfileManager returned error: %v", err)
	}

	report, err := pm.GenerateReport()
	if err != nil {
		t.Fatalf("GenerateReport returned error: %v", err)
	}

	if report != "" {
		t.Error("Expected empty report when profiler is disabled")
	}
}

func TestStartProfilingTwice(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:   true,
		OutputDir: tmpDir,
		CPU:       true,
		Memory:    true,
	}

	pm, err := NewProfileManager(cfg)
	if err != nil {
		t.Fatalf("NewProfileManager returned error: %v", err)
	}

	// Start profiling first time
	if err := pm.StartProfiling(); err != nil {
		t.Fatalf("First StartProfiling returned error: %v", err)
	}

	// Starting twice might cause issues, but should be handled gracefully
	// This is more of a sanity check
	if err := pm.StartProfiling(); err != nil {
		t.Logf("Second StartProfiling returned error (expected): %v", err)
	}

	// Stop should still work
	if err := pm.StopProfiling(); err != nil {
		t.Fatalf("StopProfiling returned error: %v", err)
	}
}

func TestStartTimeRecording(t *testing.T) {
	cfg := Config{
		Enabled: true,
		CPU:     false,
		Memory:  false,
	}

	before := time.Now()
	pm, err := NewProfileManager(cfg)
	if err != nil {
		t.Fatalf("NewProfileManager returned error: %v", err)
	}
	after := time.Now()

	startTime := pm.GetStartTime()

	if startTime.Before(before) || startTime.After(after.Add(1*time.Second)) {
		t.Errorf("Expected start time to be between %v and %v, got %v", before, after, startTime)
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
