package monitoring

import (
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestNewMetricsCollector tests the constructor
func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()

	if mc == nil {
		t.Fatal("Expected non-nil MetricsCollector")
	}
	if !mc.enabled {
		t.Error("Expected enabled=true by default")
	}
	if mc.interval != 30*time.Second {
		t.Errorf("Expected interval=30s, got %v", mc.interval)
	}
	if len(mc.collectors) != 2 {
		t.Errorf("Expected 2 default collectors, got %d", len(mc.collectors))
	}
	if mc.metrics == nil {
		t.Error("Expected metrics map to be initialized")
	}
}

// TestRecordCounter tests counter recording
func TestRecordCounter(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCounter("test_counter", 42, map[string]string{"label": "value"})

	mc.mutex.RLock()
	metric, exists := mc.metrics["test_counter"]
	mc.mutex.RUnlock()

	if !exists {
		t.Fatal("Expected metric to be recorded")
	}
	if metric.Type != MetricTypeCounter {
		t.Errorf("Expected type=counter, got %s", metric.Type)
	}
	if metric.Value != int64(42) {
		t.Errorf("Expected value=42, got %v", metric.Value)
	}
	if metric.Labels["label"] != "value" {
		t.Errorf("Expected label='value', got '%s'", metric.Labels["label"])
	}
}

// TestRecordGauge tests gauge recording
func TestRecordGauge(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordGauge("test_gauge", 3.14, map[string]string{"type": "test"})

	mc.mutex.RLock()
	metric, exists := mc.metrics["test_gauge"]
	mc.mutex.RUnlock()

	if !exists {
		t.Fatal("Expected metric to be recorded")
	}
	if metric.Type != MetricTypeGauge {
		t.Errorf("Expected type=gauge, got %s", metric.Type)
	}
	if metric.Value != 3.14 {
		t.Errorf("Expected value=3.14, got %v", metric.Value)
	}
}

// TestRecordTimer tests timer recording
func TestRecordTimer(t *testing.T) {
	mc := NewMetricsCollector()

	duration := 500 * time.Millisecond
	mc.RecordTimer("test_timer", duration, nil)

	mc.mutex.RLock()
	metric, exists := mc.metrics["test_timer"]
	mc.mutex.RUnlock()

	if !exists {
		t.Fatal("Expected metric to be recorded")
	}
	if metric.Type != MetricTypeTimer {
		t.Errorf("Expected type=timer, got %s", metric.Type)
	}
	// RecordTimer stores duration.Nanoseconds() as int64
	if metric.Value != duration.Nanoseconds() {
		t.Errorf("Expected value=%v, got %v", duration.Nanoseconds(), metric.Value)
	}
}

// TestRecordMetric tests generic metric recording
func TestRecordMetric(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordMetric("custom_metric", MetricTypeHistogram, 100, map[string]string{
		"env": "test",
	})

	mc.mutex.RLock()
	metric, exists := mc.metrics["custom_metric"]
	mc.mutex.RUnlock()

	if !exists {
		t.Fatal("Expected metric to be recorded")
	}
	if metric.Name != "custom_metric" {
		t.Errorf("Expected name='custom_metric', got '%s'", metric.Name)
	}
	if metric.Type != MetricTypeHistogram {
		t.Errorf("Expected type=histogram, got %s", metric.Type)
	}
	if metric.Value != 100 {
		t.Errorf("Expected value=100, got %v", metric.Value)
	}
	if metric.Labels["env"] != "test" {
		t.Errorf("Expected env='test', got '%s'", metric.Labels["env"])
	}
}

// TestGetMetrics tests metrics retrieval
func TestGetMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCounter("counter1", 10, nil)
	mc.RecordGauge("gauge1", 20.5, nil)
	mc.RecordTimer("timer1", time.Second, nil)

	metrics := mc.GetMetrics()

	if len(metrics) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(metrics))
	}
	if _, exists := metrics["counter1"]; !exists {
		t.Error("Expected counter1 to exist")
	}
	if _, exists := metrics["gauge1"]; !exists {
		t.Error("Expected gauge1 to exist")
	}
	if _, exists := metrics["timer1"]; !exists {
		t.Error("Expected timer1 to exist")
	}
}

// TestGetSystemMetrics tests system metrics collection
func TestGetSystemMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	metrics := mc.GetSystemMetrics()

	if metrics.CPU.NumCPU != runtime.NumCPU() {
		t.Errorf("Expected NumCPU=%d, got %d", runtime.NumCPU(), metrics.CPU.NumCPU)
	}
	if metrics.CPU.NumGoroutine <= 0 {
		t.Error("Expected NumGoroutine > 0")
	}
	if metrics.Memory.Alloc == 0 {
		t.Error("Expected Alloc > 0")
	}
	if metrics.Runtime.Version == "" {
		t.Error("Expected non-empty runtime version")
	}
	if metrics.Runtime.GOOS == "" {
		t.Error("Expected non-empty GOOS")
	}
	if metrics.Runtime.GOARCH == "" {
		t.Error("Expected non-empty GOARCH")
	}
	if metrics.Uptime <= 0 {
		t.Error("Expected Uptime > 0")
	}
}

// TestRecordTaskMetrics tests task metrics recording
func TestRecordTaskMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	result := types.TaskResult{
		TaskName: "test_task",
		Module:   "shell",
		Host:     "localhost",
		Success:  true,
		Changed:  true,
		Duration: 100 * time.Millisecond,
	}

	mc.RecordTaskMetrics(result, 2, 1000, 1500)

	// Check that metrics were recorded
	metrics := mc.GetMetrics()

	// Should have recorded multiple metrics
	if len(metrics) == 0 {
		t.Error("Expected metrics to be recorded")
	}

	// Check for task duration metric
	found := false
	for name, metric := range metrics {
		if metric.Type == MetricTypeTimer {
			found = true
			if metric.Labels["task"] != "test_task" {
				t.Errorf("Expected task label='test_task', got '%s'", metric.Labels["task"])
			}
			if metric.Labels["module"] != "shell" {
				t.Errorf("Expected module label='shell', got '%s'", metric.Labels["module"])
			}
			break
		}
		_ = name
	}
	if !found {
		t.Error("Expected to find timer metric for task")
	}
}

// TestExportMetrics_JSON tests JSON export
func TestExportMetrics_JSON(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCounter("test_counter", 42, nil)
	mc.RecordGauge("test_gauge", 3.14, nil)

	data, err := mc.ExportMetrics("json")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var exported map[string]*Metric
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(exported) != 2 {
		t.Errorf("Expected 2 metrics in export, got %d", len(exported))
	}
}

// TestExportMetrics_Prometheus tests Prometheus export
func TestExportMetrics_Prometheus(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCounter("test_counter", 42, map[string]string{"env": "test"})

	data, err := mc.ExportMetrics("prometheus")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	output := string(data)
	if output == "" {
		t.Error("Expected non-empty Prometheus output")
	}
	// Should contain metric name
	if len(output) < 10 {
		t.Error("Expected Prometheus output to contain metric data")
	}
}

// TestExportMetrics_InvalidFormat tests invalid format handling
func TestExportMetrics_InvalidFormat(t *testing.T) {
	mc := NewMetricsCollector()

	_, err := mc.ExportMetrics("invalid_format")
	if err == nil {
		t.Error("Expected error for invalid format")
	}
}

// TestAddCollector tests adding custom collectors
func TestAddCollector(t *testing.T) {
	mc := NewMetricsCollector()
	initialCount := len(mc.collectors)

	// Add a custom collector
	mc.AddCollector(&SystemCollector{})

	if len(mc.collectors) != initialCount+1 {
		t.Errorf("Expected %d collectors, got %d", initialCount+1, len(mc.collectors))
	}
}

// TestSystemCollector tests the system collector
func TestSystemCollector(t *testing.T) {
	sc := &SystemCollector{}

	if sc.Name() != "system" {
		t.Errorf("Expected name='system', got '%s'", sc.Name())
	}
	if sc.Interval() != 30*time.Second {
		t.Errorf("Expected interval=30s, got %v", sc.Interval())
	}

	metrics, err := sc.Collect()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(metrics) == 0 {
		t.Error("Expected metrics to be collected")
	}

	// Check that we have expected metrics
	foundGoroutines := false
	foundMemory := false
	for _, m := range metrics {
		if m.Name == "goroutines_total" {
			foundGoroutines = true
		}
		if m.Name == "memory_alloc_bytes" {
			foundMemory = true
		}
	}
	if !foundGoroutines {
		t.Error("Expected to find goroutines metric")
	}
	if !foundMemory {
		t.Error("Expected to find memory metric")
	}
}

// TestRuntimeCollector tests the runtime collector
func TestRuntimeCollector(t *testing.T) {
	rc := &RuntimeCollector{}

	if rc.Name() != "runtime" {
		t.Errorf("Expected name='runtime', got '%s'", rc.Name())
	}
	if rc.Interval() != 60*time.Second {
		t.Errorf("Expected interval=60s, got %v", rc.Interval())
	}

	metrics, err := rc.Collect()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(metrics) == 0 {
		t.Error("Expected metrics to be collected")
	}

	// Check for runtime metrics
	foundCPU := false
	foundCGO := false
	for _, m := range metrics {
		if m.Name == "runtime_num_cpu" {
			foundCPU = true
			if cpuVal, ok := m.Value.(int); ok {
				if cpuVal <= 0 {
					t.Error("Expected NumCPU > 0")
				}
			} else {
				t.Errorf("Expected int value for NumCPU, got %T", m.Value)
			}
		}
		if m.Name == "runtime_cgo_calls_total" {
			foundCGO = true
		}
	}
	if !foundCPU {
		t.Error("Expected to find runtime_num_cpu metric")
	}
	if !foundCGO {
		t.Error("Expected to find runtime_cgo_calls_total metric")
	}
}

// TestStartStop tests starting and stopping the collector
func TestStartStop(t *testing.T) {
	mc := NewMetricsCollector()

	// Start collection
	mc.Start()

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Stop collection
	mc.Stop()

	// Should complete without hanging
}

// TestStartStop_Disabled tests that disabled collector doesn't start
func TestStartStop_Disabled(t *testing.T) {
	mc := NewMetricsCollector()
	mc.enabled = false

	// Should not start
	mc.Start()

	// Should complete immediately
	mc.Stop()
}

// TestConcurrentMetricRecording tests thread-safe metric recording
func TestConcurrentMetricRecording(t *testing.T) {
	mc := NewMetricsCollector()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				mc.RecordCounter("concurrent_counter", int64(id*10+j), nil)
				mc.RecordGauge("concurrent_gauge", float64(id*10+j), nil)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have recorded metrics without race conditions
	metrics := mc.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Expected metrics to be recorded")
	}
}

// TestMetricTimestamp tests that metrics have timestamps
func TestMetricTimestamp(t *testing.T) {
	mc := NewMetricsCollector()

	before := time.Now()
	mc.RecordCounter("timestamped_metric", 1, nil)
	after := time.Now()

	mc.mutex.RLock()
	metric := mc.metrics["timestamped_metric"]
	mc.mutex.RUnlock()

	if metric.Timestamp.Before(before) || metric.Timestamp.After(after) {
		t.Error("Expected metric timestamp to be within test time range")
	}
}

// TestMetricLabels tests metric labels
func TestMetricLabels(t *testing.T) {
	mc := NewMetricsCollector()

	labels := map[string]string{
		"env":     "production",
		"service": "api",
		"version": "1.0.0",
	}

	mc.RecordCounter("labeled_metric", 100, labels)

	mc.mutex.RLock()
	metric := mc.metrics["labeled_metric"]
	mc.mutex.RUnlock()

	if len(metric.Labels) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(metric.Labels))
	}
	for key, expectedValue := range labels {
		if metric.Labels[key] != expectedValue {
			t.Errorf("Expected label %s='%s', got '%s'", key, expectedValue, metric.Labels[key])
		}
	}
}

// TestMetricTypes tests all metric types
func TestMetricTypes(t *testing.T) {
	tests := []struct {
		metricType MetricType
		expected   string
	}{
		{MetricTypeCounter, "counter"},
		{MetricTypeGauge, "gauge"},
		{MetricTypeHistogram, "histogram"},
		{MetricTypeSummary, "summary"},
		{MetricTypeTimer, "timer"},
	}

	for _, tt := range tests {
		if string(tt.metricType) != tt.expected {
			t.Errorf("Expected metric type '%s', got '%s'", tt.expected, string(tt.metricType))
		}
	}
}
