package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// MetricsCollector collects and manages system metrics
type MetricsCollector struct {
	metrics     map[string]*Metric
	mutex       sync.RWMutex
	collectors  []Collector
	startTime   time.Time
	enabled     bool
	interval    time.Duration
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// Metric represents a single metric
type Metric struct {
	Name        string                 `json:"name"`
	Type        MetricType             `json:"type"`
	Value       interface{}            `json:"value"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Description string                 `json:"description,omitempty"`
	Unit        string                 `json:"unit,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
	MetricTypeTimer     MetricType = "timer"
)

// Collector interface for metric collectors
type Collector interface {
	Collect() ([]*Metric, error)
	Name() string
	Interval() time.Duration
}

// SystemMetrics holds system-level metrics
type SystemMetrics struct {
	CPU        CPUMetrics        `json:"cpu"`
	Memory     MemoryMetrics     `json:"memory"`
	Goroutines GoroutineMetrics  `json:"goroutines"`
	GC         GCMetrics         `json:"gc"`
	Runtime    RuntimeMetrics    `json:"runtime"`
	Uptime     time.Duration     `json:"uptime"`
	Timestamp  time.Time         `json:"timestamp"`
}

// CPUMetrics holds CPU-related metrics
type CPUMetrics struct {
	NumCPU      int     `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	CGOCalls    int64   `json:"cgo_calls"`
}

// MemoryMetrics holds memory-related metrics
type MemoryMetrics struct {
	Alloc         uint64 `json:"alloc"`
	TotalAlloc    uint64 `json:"total_alloc"`
	Sys           uint64 `json:"sys"`
	Lookups       uint64 `json:"lookups"`
	Mallocs       uint64 `json:"mallocs"`
	Frees         uint64 `json:"frees"`
	HeapAlloc     uint64 `json:"heap_alloc"`
	HeapSys       uint64 `json:"heap_sys"`
	HeapIdle      uint64 `json:"heap_idle"`
	HeapInuse     uint64 `json:"heap_inuse"`
	HeapReleased  uint64 `json:"heap_released"`
	HeapObjects   uint64 `json:"heap_objects"`
	StackInuse    uint64 `json:"stack_inuse"`
	StackSys      uint64 `json:"stack_sys"`
	MSpanInuse    uint64 `json:"mspan_inuse"`
	MSpanSys      uint64 `json:"mspan_sys"`
	MCacheInuse   uint64 `json:"mcache_inuse"`
	MCacheSys     uint64 `json:"mcache_sys"`
	BuckHashSys   uint64 `json:"buck_hash_sys"`
	GCSys         uint64 `json:"gc_sys"`
	OtherSys      uint64 `json:"other_sys"`
	NextGC        uint64 `json:"next_gc"`
}

// GoroutineMetrics holds goroutine-related metrics
type GoroutineMetrics struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Waiting int `json:"waiting"`
}

// GCMetrics holds garbage collection metrics
type GCMetrics struct {
	NumGC        uint32        `json:"num_gc"`
	NumForcedGC  uint32        `json:"num_forced_gc"`
	PauseTotalNs uint64        `json:"pause_total_ns"`
	PauseNs      []uint64      `json:"pause_ns"`
	PauseEnd     []uint64      `json:"pause_end"`
	LastGC       time.Time     `json:"last_gc"`
	NextGC       uint64        `json:"next_gc"`
	GCCPUFraction float64      `json:"gc_cpu_fraction"`
}

// RuntimeMetrics holds Go runtime metrics
type RuntimeMetrics struct {
	Version   string `json:"version"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	Compiler  string `json:"compiler"`
	NumCPU    int    `json:"num_cpu"`
}

// TaskMetrics holds task execution metrics
type TaskMetrics struct {
	TaskName      string        `json:"task_name"`
	Module        string        `json:"module"`
	Host          string        `json:"host"`
	Duration      time.Duration `json:"duration"`
	Success       bool          `json:"success"`
	Changed       bool          `json:"changed"`
	RetryCount    int           `json:"retry_count"`
	Timestamp     time.Time     `json:"timestamp"`
	MemoryBefore  uint64        `json:"memory_before"`
	MemoryAfter   uint64        `json:"memory_after"`
	MemoryDelta   int64         `json:"memory_delta"`
}

// PlaybookMetrics holds playbook execution metrics
type PlaybookMetrics struct {
	PlaybookName   string                 `json:"playbook_name"`
	TotalTasks     int                    `json:"total_tasks"`
	SuccessfulTasks int                   `json:"successful_tasks"`
	FailedTasks    int                    `json:"failed_tasks"`
	ChangedTasks   int                    `json:"changed_tasks"`
	SkippedTasks   int                    `json:"skipped_tasks"`
	TotalHosts     int                    `json:"total_hosts"`
	Duration       time.Duration          `json:"duration"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	HostMetrics    map[string]HostMetrics `json:"host_metrics"`
	TaskMetrics    []TaskMetrics          `json:"task_metrics"`
}

// HostMetrics holds per-host metrics
type HostMetrics struct {
	HostName        string        `json:"host_name"`
	TotalTasks      int           `json:"total_tasks"`
	SuccessfulTasks int           `json:"successful_tasks"`
	FailedTasks     int           `json:"failed_tasks"`
	ChangedTasks    int           `json:"changed_tasks"`
	SkippedTasks    int           `json:"skipped_tasks"`
	Duration        time.Duration `json:"duration"`
	AverageTaskTime time.Duration `json:"average_task_time"`
	ConnectionTime  time.Duration `json:"connection_time"`
	LastSeen        time.Time     `json:"last_seen"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		metrics:   make(map[string]*Metric),
		startTime: time.Now(),
		enabled:   true,
		interval:  30 * time.Second,
		stopChan:  make(chan struct{}),
	}

	// Add default collectors
	mc.AddCollector(&SystemCollector{})
	mc.AddCollector(&RuntimeCollector{})

	return mc
}

// Start starts the metrics collection
func (mc *MetricsCollector) Start() {
	if !mc.enabled {
		return
	}

	mc.wg.Add(1)
	go mc.collectLoop()
}

// Stop stops the metrics collection
func (mc *MetricsCollector) Stop() {
	close(mc.stopChan)
	mc.wg.Wait()
}

// AddCollector adds a new collector
func (mc *MetricsCollector) AddCollector(collector Collector) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.collectors = append(mc.collectors, collector)
}

// RecordMetric records a single metric
func (mc *MetricsCollector) RecordMetric(name string, metricType MetricType, value interface{}, labels map[string]string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	metric := &Metric{
		Name:      name,
		Type:      metricType,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}

	mc.metrics[name] = metric
}

// RecordCounter records a counter metric
func (mc *MetricsCollector) RecordCounter(name string, value int64, labels map[string]string) {
	mc.RecordMetric(name, MetricTypeCounter, value, labels)
}

// RecordGauge records a gauge metric
func (mc *MetricsCollector) RecordGauge(name string, value float64, labels map[string]string) {
	mc.RecordMetric(name, MetricTypeGauge, value, labels)
}

// RecordTimer records a timer metric
func (mc *MetricsCollector) RecordTimer(name string, duration time.Duration, labels map[string]string) {
	mc.RecordMetric(name, MetricTypeTimer, duration.Nanoseconds(), labels)
}

// RecordTaskMetrics records task execution metrics
func (mc *MetricsCollector) RecordTaskMetrics(result types.TaskResult, retryCount int, memoryBefore, memoryAfter uint64) {
	metrics := TaskMetrics{
		TaskName:     result.TaskName,
		Module:       result.Module,
		Host:         result.Host,
		Duration:     result.Duration,
		Success:      result.Success,
		Changed:      result.Changed,
		RetryCount:   retryCount,
		Timestamp:    result.Timestamp,
		MemoryBefore: memoryBefore,
		MemoryAfter:  memoryAfter,
		MemoryDelta:  int64(memoryAfter) - int64(memoryBefore),
	}

	// Record individual metrics
	labels := map[string]string{
		"task":   result.TaskName,
		"module": result.Module,
		"host":   result.Host,
	}

	mc.RecordTimer("task_duration", result.Duration, labels)
	mc.RecordCounter("task_total", 1, labels)

	if result.Success {
		mc.RecordCounter("task_success", 1, labels)
	} else {
		mc.RecordCounter("task_failure", 1, labels)
	}

	if result.Changed {
		mc.RecordCounter("task_changed", 1, labels)
	}

	if retryCount > 0 {
		mc.RecordCounter("task_retries", int64(retryCount), labels)
	}

	mc.RecordGauge("task_memory_delta", float64(metrics.MemoryDelta), labels)
}

// GetMetrics returns all current metrics
func (mc *MetricsCollector) GetMetrics() map[string]*Metric {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	result := make(map[string]*Metric)
	for k, v := range mc.metrics {
		result[k] = v
	}

	return result
}

// GetSystemMetrics returns current system metrics
func (mc *MetricsCollector) GetSystemMetrics() SystemMetrics {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return SystemMetrics{
		CPU: CPUMetrics{
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			CGOCalls:     runtime.NumCgoCall(),
		},
		Memory: MemoryMetrics{
			Alloc:        memStats.Alloc,
			TotalAlloc:   memStats.TotalAlloc,
			Sys:          memStats.Sys,
			Lookups:      memStats.Lookups,
			Mallocs:      memStats.Mallocs,
			Frees:        memStats.Frees,
			HeapAlloc:    memStats.HeapAlloc,
			HeapSys:      memStats.HeapSys,
			HeapIdle:     memStats.HeapIdle,
			HeapInuse:    memStats.HeapInuse,
			HeapReleased: memStats.HeapReleased,
			HeapObjects:  memStats.HeapObjects,
			StackInuse:   memStats.StackInuse,
			StackSys:     memStats.StackSys,
			MSpanInuse:   memStats.MSpanInuse,
			MSpanSys:     memStats.MSpanSys,
			MCacheInuse:  memStats.MCacheInuse,
			MCacheSys:    memStats.MCacheSys,
			BuckHashSys:  memStats.BuckHashSys,
			GCSys:        memStats.GCSys,
			OtherSys:     memStats.OtherSys,
			NextGC:       memStats.NextGC,
		},
		Goroutines: GoroutineMetrics{
			Total: runtime.NumGoroutine(),
		},
		GC: GCMetrics{
			NumGC:         memStats.NumGC,
			NumForcedGC:   memStats.NumForcedGC,
			PauseTotalNs:  memStats.PauseTotalNs,
			PauseNs:       memStats.PauseNs[:],
			PauseEnd:      memStats.PauseEnd[:],
			NextGC:        memStats.NextGC,
			GCCPUFraction: memStats.GCCPUFraction,
		},
		Runtime: RuntimeMetrics{
			Version:  runtime.Version(),
			GOOS:     runtime.GOOS,
			GOARCH:   runtime.GOARCH,
			Compiler: runtime.Compiler,
			NumCPU:   runtime.NumCPU(),
		},
		Uptime:    time.Since(mc.startTime),
		Timestamp: time.Now(),
	}
}

// ExportMetrics exports metrics in various formats
func (mc *MetricsCollector) ExportMetrics(format string) ([]byte, error) {
	metrics := mc.GetMetrics()

	switch format {
	case "json":
		return json.MarshalIndent(metrics, "", "  ")
	case "prometheus":
		return mc.exportPrometheus(metrics)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportPrometheus exports metrics in Prometheus format
func (mc *MetricsCollector) exportPrometheus(metrics map[string]*Metric) ([]byte, error) {
	var output []string

	for _, metric := range metrics {
		line := fmt.Sprintf("# TYPE %s %s", metric.Name, metric.Type)
		output = append(output, line)

		labelStr := ""
		if len(metric.Labels) > 0 {
			var labels []string
			for k, v := range metric.Labels {
				labels = append(labels, fmt.Sprintf(`%s="%s"`, k, v))
			}
			labelStr = fmt.Sprintf("{%s}", strings.Join(labels, ","))
		}

		valueLine := fmt.Sprintf("%s%s %v %d", metric.Name, labelStr, metric.Value, metric.Timestamp.Unix())
		output = append(output, valueLine)
	}

	return []byte(strings.Join(output, "\n")), nil
}

// collectLoop runs the collection loop
func (mc *MetricsCollector) collectLoop() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.collectMetrics()
		case <-mc.stopChan:
			return
		}
	}
}

// collectMetrics collects metrics from all collectors
func (mc *MetricsCollector) collectMetrics() {
	for _, collector := range mc.collectors {
		metrics, err := collector.Collect()
		if err != nil {
			continue // Skip failed collections
		}

		mc.mutex.Lock()
		for _, metric := range metrics {
			mc.metrics[metric.Name] = metric
		}
		mc.mutex.Unlock()
	}
}

// SystemCollector collects system metrics
type SystemCollector struct{}

func (sc *SystemCollector) Name() string {
	return "system"
}

func (sc *SystemCollector) Interval() time.Duration {
	return 30 * time.Second
}

func (sc *SystemCollector) Collect() ([]*Metric, error) {
	var metrics []*Metric
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	now := time.Now()

	// Memory metrics
	metrics = append(metrics, &Metric{
		Name:      "memory_alloc_bytes",
		Type:      MetricTypeGauge,
		Value:     memStats.Alloc,
		Timestamp: now,
		Unit:      "bytes",
	})

	metrics = append(metrics, &Metric{
		Name:      "memory_sys_bytes",
		Type:      MetricTypeGauge,
		Value:     memStats.Sys,
		Timestamp: now,
		Unit:      "bytes",
	})

	// Goroutine metrics
	metrics = append(metrics, &Metric{
		Name:      "goroutines_total",
		Type:      MetricTypeGauge,
		Value:     runtime.NumGoroutine(),
		Timestamp: now,
	})

	// GC metrics
	metrics = append(metrics, &Metric{
		Name:      "gc_runs_total",
		Type:      MetricTypeCounter,
		Value:     memStats.NumGC,
		Timestamp: now,
	})

	return metrics, nil
}

// RuntimeCollector collects runtime metrics
type RuntimeCollector struct{}

func (rc *RuntimeCollector) Name() string {
	return "runtime"
}

func (rc *RuntimeCollector) Interval() time.Duration {
	return 60 * time.Second
}

func (rc *RuntimeCollector) Collect() ([]*Metric, error) {
	var metrics []*Metric
	now := time.Now()

	metrics = append(metrics, &Metric{
		Name:      "runtime_num_cpu",
		Type:      MetricTypeGauge,
		Value:     runtime.NumCPU(),
		Timestamp: now,
	})

	metrics = append(metrics, &Metric{
		Name:      "runtime_cgo_calls_total",
		Type:      MetricTypeCounter,
		Value:     runtime.NumCgoCall(),
		Timestamp: now,
	})

	return metrics, nil
}

// MetricsReporter provides reporting capabilities
type MetricsReporter struct {
	collector *MetricsCollector
	reports   []Report
	mutex     sync.RWMutex
}

// Report represents a metrics report
type Report struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    time.Duration          `json:"duration"`
	Metrics     map[string]*Metric     `json:"metrics"`
	Summary     map[string]interface{} `json:"summary"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewMetricsReporter creates a new metrics reporter
func NewMetricsReporter(collector *MetricsCollector) *MetricsReporter {
	return &MetricsReporter{
		collector: collector,
		reports:   make([]Report, 0),
	}
}

// GenerateReport generates a metrics report
func (mr *MetricsReporter) GenerateReport(name, reportType string) Report {
	report := Report{
		ID:        fmt.Sprintf("%s_%d", name, time.Now().Unix()),
		Name:      name,
		Type:      reportType,
		Timestamp: time.Now(),
		Metrics:   mr.collector.GetMetrics(),
		Summary:   make(map[string]interface{}),
		Metadata:  make(map[string]interface{}),
	}

	// Generate summary based on report type
	switch reportType {
	case "system":
		report.Summary = mr.generateSystemSummary()
	case "performance":
		report.Summary = mr.generatePerformanceSummary()
	case "tasks":
		report.Summary = mr.generateTaskSummary()
	}

	mr.mutex.Lock()
	mr.reports = append(mr.reports, report)
	mr.mutex.Unlock()

	return report
}

// generateSystemSummary generates system metrics summary
func (mr *MetricsReporter) generateSystemSummary() map[string]interface{} {
	systemMetrics := mr.collector.GetSystemMetrics()

	return map[string]interface{}{
		"uptime":           systemMetrics.Uptime.String(),
		"memory_usage":     systemMetrics.Memory.Alloc,
		"goroutines":       systemMetrics.Goroutines.Total,
		"gc_runs":          systemMetrics.GC.NumGC,
		"cpu_count":        systemMetrics.CPU.NumCPU,
		"go_version":       systemMetrics.Runtime.Version,
	}
}

// generatePerformanceSummary generates performance metrics summary
func (mr *MetricsReporter) generatePerformanceSummary() map[string]interface{} {
	metrics := mr.collector.GetMetrics()

	summary := make(map[string]interface{})

	// Calculate averages and totals
	var totalTasks, successfulTasks, failedTasks int64
	var totalDuration time.Duration

	for _, metric := range metrics {
		switch metric.Name {
		case "task_total":
			if val, ok := metric.Value.(int64); ok {
				totalTasks += val
			}
		case "task_success":
			if val, ok := metric.Value.(int64); ok {
				successfulTasks += val
			}
		case "task_failure":
			if val, ok := metric.Value.(int64); ok {
				failedTasks += val
			}
		case "task_duration":
			if val, ok := metric.Value.(int64); ok {
				totalDuration += time.Duration(val)
			}
		}
	}

	summary["total_tasks"] = totalTasks
	summary["successful_tasks"] = successfulTasks
	summary["failed_tasks"] = failedTasks
	summary["success_rate"] = float64(successfulTasks) / float64(totalTasks) * 100

	if totalTasks > 0 {
		summary["average_task_duration"] = totalDuration / time.Duration(totalTasks)
	}

	return summary
}

// generateTaskSummary generates task metrics summary
func (mr *MetricsReporter) generateTaskSummary() map[string]interface{} {
	return mr.generatePerformanceSummary()
}

// GetReports returns all reports
func (mr *MetricsReporter) GetReports() []Report {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	reports := make([]Report, len(mr.reports))
	copy(reports, mr.reports)
	return reports
}

// Helper function to import strings package
import "strings"
