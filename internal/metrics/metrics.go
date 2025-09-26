package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds execution metrics
type Metrics struct {
	mu sync.RWMutex

	// Execution metrics
	PlaybooksExecuted int64 `json:"playbooks_executed"`
	PlaysExecuted     int64 `json:"plays_executed"`
	TasksExecuted     int64 `json:"tasks_executed"`
	TasksSucceeded    int64 `json:"tasks_succeeded"`
	TasksFailed       int64 `json:"tasks_failed"`
	TasksSkipped      int64 `json:"tasks_skipped"`
	TasksChanged      int64 `json:"tasks_changed"`

	// Timing metrics
	TotalExecutionTime time.Duration `json:"total_execution_time"`
	AverageTaskTime    time.Duration `json:"average_task_time"`
	MinTaskTime        time.Duration `json:"min_task_time"`
	MaxTaskTime        time.Duration `json:"max_task_time"`

	// Module usage
	ModuleUsage map[string]int64 `json:"module_usage"`

	// Error tracking
	ErrorsByModule map[string]int64 `json:"errors_by_module"`
	ErrorsByType   map[string]int64 `json:"errors_by_type"`

	// Performance metrics
	CacheHits   int64 `json:"cache_hits"`
	CacheMisses int64 `json:"cache_misses"`

	// Host metrics
	HostsConnected    int64            `json:"hosts_connected"`
	HostsUnreachable  int64            `json:"hosts_unreachable"`
	HostExecutionTime map[string]int64 `json:"host_execution_time_ms"`

	// Resource usage
	MemoryUsage  int64   `json:"memory_usage_bytes"`
	CPUUsage     float64 `json:"cpu_usage_percent"`
	NetworkBytes int64   `json:"network_bytes"`
	DiskIOBytes  int64   `json:"disk_io_bytes"`

	// Concurrent execution
	MaxConcurrentTasks     int64 `json:"max_concurrent_tasks"`
	CurrentConcurrentTasks int64 `json:"current_concurrent_tasks"`

	// Start time
	StartTime time.Time `json:"start_time"`

	// Prometheus metrics
	promRegistry *prometheus.Registry
	promMetrics  *PrometheusMetrics
}

// PrometheusMetrics holds Prometheus metric collectors
type PrometheusMetrics struct {
	TasksTotal      *prometheus.CounterVec
	TaskDuration    *prometheus.HistogramVec
	PlaybooksTotal  prometheus.Counter
	PlaysTotal      prometheus.Counter
	CacheHitRate    prometheus.Gauge
	HostsConnected  prometheus.Gauge
	ConcurrentTasks prometheus.Gauge
	ErrorsTotal     *prometheus.CounterVec
	ModuleUsage     *prometheus.CounterVec
}

// MetricsSummary provides a comprehensive summary of metrics
type MetricsSummary struct {
	Overview      *OverviewMetrics      `json:"overview"`
	Performance   *PerformanceMetrics   `json:"performance"`
	Modules       *ModuleMetrics        `json:"modules"`
	Errors        *ErrorMetrics         `json:"errors"`
	Hosts         *HostMetrics          `json:"hosts"`
	Cache         *CacheMetrics         `json:"cache"`
	ResourceUsage *ResourceUsageMetrics `json:"resource_usage"`
	Timestamp     time.Time             `json:"timestamp"`
}

type OverviewMetrics struct {
	PlaybooksExecuted int64         `json:"playbooks_executed"`
	PlaysExecuted     int64         `json:"plays_executed"`
	TasksExecuted     int64         `json:"tasks_executed"`
	TasksSucceeded    int64         `json:"tasks_succeeded"`
	TasksFailed       int64         `json:"tasks_failed"`
	TasksSkipped      int64         `json:"tasks_skipped"`
	TasksChanged      int64         `json:"tasks_changed"`
	SuccessRate       float64       `json:"success_rate"`
	Uptime            time.Duration `json:"uptime"`
}

type PerformanceMetrics struct {
	TotalExecutionTime     time.Duration `json:"total_execution_time"`
	AverageTaskTime        time.Duration `json:"average_task_time"`
	MinTaskTime            time.Duration `json:"min_task_time"`
	MaxTaskTime            time.Duration `json:"max_task_time"`
	MaxConcurrentTasks     int64         `json:"max_concurrent_tasks"`
	CurrentConcurrentTasks int64         `json:"current_concurrent_tasks"`
}

type ModuleMetrics struct {
	Usage      map[string]int64   `json:"usage"`
	TopModules []ModuleUsage      `json:"top_modules"`
	ErrorRates map[string]float64 `json:"error_rates"`
}

type ModuleUsage struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type ErrorMetrics struct {
	ByModule map[string]int64 `json:"by_module"`
	ByType   map[string]int64 `json:"by_type"`
	Total    int64            `json:"total"`
}

type HostMetrics struct {
	Connected      int64             `json:"connected"`
	Unreachable    int64             `json:"unreachable"`
	ExecutionTimes map[string]int64  `json:"execution_times_ms"`
	SlowestHosts   []HostPerformance `json:"slowest_hosts"`
}

type HostPerformance struct {
	Host string `json:"host"`
	Time int64  `json:"time_ms"`
}

type CacheMetrics struct {
	Hits    int64   `json:"hits"`
	Misses  int64   `json:"misses"`
	HitRate float64 `json:"hit_rate"`
	Total   int64   `json:"total"`
}

type ResourceUsageMetrics struct {
	MemoryUsage  int64   `json:"memory_usage_bytes"`
	CPUUsage     float64 `json:"cpu_usage_percent"`
	NetworkBytes int64   `json:"network_bytes"`
	DiskIOBytes  int64   `json:"disk_io_bytes"`
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		ModuleUsage:       make(map[string]int64),
		ErrorsByModule:    make(map[string]int64),
		ErrorsByType:      make(map[string]int64),
		HostExecutionTime: make(map[string]int64),
		StartTime:         time.Now(),
		MinTaskTime:       time.Duration(0),
		MaxTaskTime:       time.Duration(0),
	}
}

// NewMetricsWithPrometheus creates a new metrics instance with Prometheus support
func NewMetricsWithPrometheus() *Metrics {
	registry := prometheus.NewRegistry()

	promMetrics := &PrometheusMetrics{
		TasksTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "onigirazu_tasks_total",
				Help: "Total number of tasks executed",
			},
			[]string{"status", "module"},
		),
		TaskDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "onigirazu_task_duration_seconds",
				Help:    "Task execution duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"module", "host"},
		),
		PlaybooksTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "onigirazu_playbooks_total",
				Help: "Total number of playbooks executed",
			},
		),
		PlaysTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "onigirazu_plays_total",
				Help: "Total number of plays executed",
			},
		),
		CacheHitRate: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "onigirazu_cache_hit_rate",
				Help: "Cache hit rate percentage",
			},
		),
		HostsConnected: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "onigirazu_hosts_connected",
				Help: "Number of connected hosts",
			},
		),
		ConcurrentTasks: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "onigirazu_concurrent_tasks",
				Help: "Current number of concurrent tasks",
			},
		),
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "onigirazu_errors_total",
				Help: "Total number of errors",
			},
			[]string{"type", "module"},
		),
		ModuleUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "onigirazu_module_usage_total",
				Help: "Total usage count per module",
			},
			[]string{"module"},
		),
	}

	// Register all metrics
	registry.MustRegister(
		promMetrics.TasksTotal,
		promMetrics.TaskDuration,
		promMetrics.PlaybooksTotal,
		promMetrics.PlaysTotal,
		promMetrics.CacheHitRate,
		promMetrics.HostsConnected,
		promMetrics.ConcurrentTasks,
		promMetrics.ErrorsTotal,
		promMetrics.ModuleUsage,
	)

	return &Metrics{
		ModuleUsage:       make(map[string]int64),
		ErrorsByModule:    make(map[string]int64),
		ErrorsByType:      make(map[string]int64),
		HostExecutionTime: make(map[string]int64),
		StartTime:         time.Now(),
		MinTaskTime:       time.Duration(0),
		MaxTaskTime:       time.Duration(0),
		promRegistry:      registry,
		promMetrics:       promMetrics,
	}
}

// IncrementPlaybooksExecuted increments the playbooks executed counter
func (m *Metrics) IncrementPlaybooksExecuted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PlaybooksExecuted++
}

// IncrementPlaysExecuted increments the plays executed counter
func (m *Metrics) IncrementPlaysExecuted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PlaysExecuted++
}

// IncrementTasksExecuted increments the tasks executed counter
func (m *Metrics) IncrementTasksExecuted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TasksExecuted++
}

// IncrementTasksSucceeded increments the tasks succeeded counter
func (m *Metrics) IncrementTasksSucceeded() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TasksSucceeded++
}

// IncrementTasksFailed increments the tasks failed counter
func (m *Metrics) IncrementTasksFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TasksFailed++
}

// IncrementTasksSkipped increments the tasks skipped counter
func (m *Metrics) IncrementTasksSkipped() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TasksSkipped++
}

// IncrementTasksChanged increments the tasks changed counter
func (m *Metrics) IncrementTasksChanged() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TasksChanged++
}

// AddExecutionTime adds execution time to total
func (m *Metrics) AddExecutionTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalExecutionTime += duration

	// Update min/max task times
	if m.MinTaskTime == 0 || duration < m.MinTaskTime {
		m.MinTaskTime = duration
	}
	if duration > m.MaxTaskTime {
		m.MaxTaskTime = duration
	}

	if m.TasksExecuted > 0 {
		m.AverageTaskTime = m.TotalExecutionTime / time.Duration(m.TasksExecuted)
	}
}

// AddTaskExecutionTime adds execution time for a specific task and module
func (m *Metrics) AddTaskExecutionTime(module, host string, duration time.Duration) {
	m.AddExecutionTime(duration)

	// Update Prometheus metrics if available
	if m.promMetrics != nil {
		m.promMetrics.TaskDuration.WithLabelValues(module, host).Observe(duration.Seconds())
	}
}

// RecordHostExecutionTime records execution time for a specific host
func (m *Metrics) RecordHostExecutionTime(host string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.HostExecutionTime[host] += duration.Milliseconds()
}

// IncrementHostsConnected increments connected hosts counter
func (m *Metrics) IncrementHostsConnected() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.HostsConnected++

	if m.promMetrics != nil {
		m.promMetrics.HostsConnected.Set(float64(m.HostsConnected))
	}
}

// IncrementHostsUnreachable increments unreachable hosts counter
func (m *Metrics) IncrementHostsUnreachable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.HostsUnreachable++
}

// SetConcurrentTasks sets the current number of concurrent tasks
func (m *Metrics) SetConcurrentTasks(count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CurrentConcurrentTasks = count

	if count > m.MaxConcurrentTasks {
		m.MaxConcurrentTasks = count
	}

	if m.promMetrics != nil {
		m.promMetrics.ConcurrentTasks.Set(float64(count))
	}
}

// UpdateResourceUsage updates resource usage metrics
func (m *Metrics) UpdateResourceUsage(memory int64, cpu float64, network, diskIO int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MemoryUsage = memory
	m.CPUUsage = cpu
	m.NetworkBytes += network
	m.DiskIOBytes += diskIO
}

// IncrementModuleUsage increments usage counter for a module
func (m *Metrics) IncrementModuleUsage(module string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ModuleUsage[module]++

	if m.promMetrics != nil {
		m.promMetrics.ModuleUsage.WithLabelValues(module).Inc()
	}
}

// IncrementErrorByModule increments error counter for a module
func (m *Metrics) IncrementErrorByModule(module string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorsByModule[module]++

	if m.promMetrics != nil {
		m.promMetrics.ErrorsTotal.WithLabelValues("module", module).Inc()
	}
}

// IncrementErrorByType increments error counter for an error type
func (m *Metrics) IncrementErrorByType(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorsByType[errorType]++

	if m.promMetrics != nil {
		m.promMetrics.ErrorsTotal.WithLabelValues(errorType, "").Inc()
	}
}

// IncrementCacheHits increments cache hits counter
func (m *Metrics) IncrementCacheHits() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheHits++

	// Update Prometheus cache hit rate
	if m.promMetrics != nil {
		m.updateCacheHitRate()
	}
}

// IncrementCacheMisses increments cache misses counter
func (m *Metrics) IncrementCacheMisses() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheMisses++

	// Update Prometheus cache hit rate
	if m.promMetrics != nil {
		m.updateCacheHitRate()
	}
}

// updateCacheHitRate updates the Prometheus cache hit rate gauge (must be called with lock held)
func (m *Metrics) updateCacheHitRate() {
	total := m.CacheHits + m.CacheMisses
	if total > 0 {
		hitRate := float64(m.CacheHits) / float64(total) * 100.0
		m.promMetrics.CacheHitRate.Set(hitRate)
	}
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := &Metrics{
		PlaybooksExecuted:  m.PlaybooksExecuted,
		PlaysExecuted:      m.PlaysExecuted,
		TasksExecuted:      m.TasksExecuted,
		TasksSucceeded:     m.TasksSucceeded,
		TasksFailed:        m.TasksFailed,
		TasksSkipped:       m.TasksSkipped,
		TasksChanged:       m.TasksChanged,
		TotalExecutionTime: m.TotalExecutionTime,
		AverageTaskTime:    m.AverageTaskTime,
		CacheHits:          m.CacheHits,
		CacheMisses:        m.CacheMisses,
		StartTime:          m.StartTime,
		ModuleUsage:        make(map[string]int64),
		ErrorsByModule:     make(map[string]int64),
		ErrorsByType:       make(map[string]int64),
	}

	// Copy maps
	for k, v := range m.ModuleUsage {
		snapshot.ModuleUsage[k] = v
	}
	for k, v := range m.ErrorsByModule {
		snapshot.ErrorsByModule[k] = v
	}
	for k, v := range m.ErrorsByType {
		snapshot.ErrorsByType[k] = v
	}

	return snapshot
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PlaybooksExecuted = 0
	m.PlaysExecuted = 0
	m.TasksExecuted = 0
	m.TasksSucceeded = 0
	m.TasksFailed = 0
	m.TasksSkipped = 0
	m.TasksChanged = 0
	m.TotalExecutionTime = 0
	m.AverageTaskTime = 0
	m.CacheHits = 0
	m.CacheMisses = 0
	m.StartTime = time.Now()

	// Clear maps
	m.ModuleUsage = make(map[string]int64)
	m.ErrorsByModule = make(map[string]int64)
	m.ErrorsByType = make(map[string]int64)
}

// GetSuccessRate returns the success rate as a percentage
func (m *Metrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.TasksExecuted == 0 {
		return 0.0
	}

	return float64(m.TasksSucceeded) / float64(m.TasksExecuted) * 100.0
}

// GetCacheHitRate returns the cache hit rate as a percentage
func (m *Metrics) GetCacheHitRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.CacheHits + m.CacheMisses
	if total == 0 {
		return 0.0
	}

	return float64(m.CacheHits) / float64(total) * 100.0
}

// GetUptime returns the uptime since metrics started
func (m *Metrics) GetUptime() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return time.Since(m.StartTime)
}

// GetSummary returns a comprehensive metrics summary
func (m *Metrics) GetSummary() *MetricsSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate top modules
	topModules := make([]ModuleUsage, 0, len(m.ModuleUsage))
	for name, count := range m.ModuleUsage {
		topModules = append(topModules, ModuleUsage{Name: name, Count: count})
	}
	sort.Slice(topModules, func(i, j int) bool {
		return topModules[i].Count > topModules[j].Count
	})
	if len(topModules) > 10 {
		topModules = topModules[:10]
	}

	// Calculate error rates by module
	errorRates := make(map[string]float64)
	for module, errors := range m.ErrorsByModule {
		if usage, exists := m.ModuleUsage[module]; exists && usage > 0 {
			errorRates[module] = float64(errors) / float64(usage) * 100.0
		}
	}

	// Calculate slowest hosts
	slowestHosts := make([]HostPerformance, 0, len(m.HostExecutionTime))
	for host, time := range m.HostExecutionTime {
		slowestHosts = append(slowestHosts, HostPerformance{Host: host, Time: time})
	}
	sort.Slice(slowestHosts, func(i, j int) bool {
		return slowestHosts[i].Time > slowestHosts[j].Time
	})
	if len(slowestHosts) > 10 {
		slowestHosts = slowestHosts[:10]
	}

	// Calculate total errors
	totalErrors := int64(0)
	for _, count := range m.ErrorsByType {
		totalErrors += count
	}

	return &MetricsSummary{
		Overview: &OverviewMetrics{
			PlaybooksExecuted: m.PlaybooksExecuted,
			PlaysExecuted:     m.PlaysExecuted,
			TasksExecuted:     m.TasksExecuted,
			TasksSucceeded:    m.TasksSucceeded,
			TasksFailed:       m.TasksFailed,
			TasksSkipped:      m.TasksSkipped,
			TasksChanged:      m.TasksChanged,
			SuccessRate:       m.GetSuccessRate(),
			Uptime:            time.Since(m.StartTime),
		},
		Performance: &PerformanceMetrics{
			TotalExecutionTime:     m.TotalExecutionTime,
			AverageTaskTime:        m.AverageTaskTime,
			MinTaskTime:            m.MinTaskTime,
			MaxTaskTime:            m.MaxTaskTime,
			MaxConcurrentTasks:     m.MaxConcurrentTasks,
			CurrentConcurrentTasks: m.CurrentConcurrentTasks,
		},
		Modules: &ModuleMetrics{
			Usage:      m.ModuleUsage,
			TopModules: topModules,
			ErrorRates: errorRates,
		},
		Errors: &ErrorMetrics{
			ByModule: m.ErrorsByModule,
			ByType:   m.ErrorsByType,
			Total:    totalErrors,
		},
		Hosts: &HostMetrics{
			Connected:      m.HostsConnected,
			Unreachable:    m.HostsUnreachable,
			ExecutionTimes: m.HostExecutionTime,
			SlowestHosts:   slowestHosts,
		},
		Cache: &CacheMetrics{
			Hits:    m.CacheHits,
			Misses:  m.CacheMisses,
			HitRate: m.GetCacheHitRate(),
			Total:   m.CacheHits + m.CacheMisses,
		},
		ResourceUsage: &ResourceUsageMetrics{
			MemoryUsage:  m.MemoryUsage,
			CPUUsage:     m.CPUUsage,
			NetworkBytes: m.NetworkBytes,
			DiskIOBytes:  m.DiskIOBytes,
		},
		Timestamp: time.Now(),
	}
}

// GetPrometheusHandler returns HTTP handler for Prometheus metrics
func (m *Metrics) GetPrometheusHandler() http.Handler {
	if m.promRegistry == nil {
		return nil
	}
	return promhttp.HandlerFor(m.promRegistry, promhttp.HandlerOpts{})
}

// StartMetricsServer starts HTTP server for metrics endpoint
func (m *Metrics) StartMetricsServer(addr string) error {
	if m.promRegistry == nil {
		return fmt.Errorf("prometheus metrics not enabled")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", m.GetPrometheusHandler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/summary", func(w http.ResponseWriter, r *http.Request) {
		summary := m.GetSummary()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(summary)
	})

	return http.ListenAndServe(addr, mux)
}

// RecordTaskResult records the result of a task execution
func (m *Metrics) RecordTaskResult(module, host, status string, duration time.Duration, changed bool) {
	m.IncrementTasksExecuted()
	m.IncrementModuleUsage(module)
	m.AddTaskExecutionTime(module, host, duration)
	m.RecordHostExecutionTime(host, duration)

	switch strings.ToLower(status) {
	case "success", "ok":
		m.IncrementTasksSucceeded()
		if changed {
			m.IncrementTasksChanged()
		}
		if m.promMetrics != nil {
			m.promMetrics.TasksTotal.WithLabelValues("success", module).Inc()
		}
	case "failed", "error":
		m.IncrementTasksFailed()
		m.IncrementErrorByModule(module)
		if m.promMetrics != nil {
			m.promMetrics.TasksTotal.WithLabelValues("failed", module).Inc()
		}
	case "skipped":
		m.IncrementTasksSkipped()
		if m.promMetrics != nil {
			m.promMetrics.TasksTotal.WithLabelValues("skipped", module).Inc()
		}
	}
}

// GetFormattedSummary returns a human-readable formatted summary
func (m *Metrics) GetFormattedSummary() string {
	summary := m.GetSummary()

	var sb strings.Builder
	sb.WriteString("=== Onigirazu Metrics Summary ===\n\n")

	// Overview
	sb.WriteString("📊 Overview:\n")
	sb.WriteString(fmt.Sprintf("  Playbooks: %d | Plays: %d | Tasks: %d\n",
		summary.Overview.PlaybooksExecuted,
		summary.Overview.PlaysExecuted,
		summary.Overview.TasksExecuted))
	sb.WriteString(fmt.Sprintf("  Success: %d (%.1f%%) | Failed: %d | Skipped: %d | Changed: %d\n",
		summary.Overview.TasksSucceeded,
		summary.Overview.SuccessRate,
		summary.Overview.TasksFailed,
		summary.Overview.TasksSkipped,
		summary.Overview.TasksChanged))
	sb.WriteString(fmt.Sprintf("  Uptime: %v\n\n", summary.Overview.Uptime))

	// Performance
	sb.WriteString("⚡ Performance:\n")
	sb.WriteString(fmt.Sprintf("  Total Time: %v | Avg Task: %v\n",
		summary.Performance.TotalExecutionTime,
		summary.Performance.AverageTaskTime))
	sb.WriteString(fmt.Sprintf("  Min Task: %v | Max Task: %v\n",
		summary.Performance.MinTaskTime,
		summary.Performance.MaxTaskTime))
	sb.WriteString(fmt.Sprintf("  Concurrent: %d (max: %d)\n\n",
		summary.Performance.CurrentConcurrentTasks,
		summary.Performance.MaxConcurrentTasks))

	// Top modules
	if len(summary.Modules.TopModules) > 0 {
		sb.WriteString("🔧 Top Modules:\n")
		for i, module := range summary.Modules.TopModules {
			if i >= 5 {
				break
			}
			errorRate := summary.Modules.ErrorRates[module.Name]
			sb.WriteString(fmt.Sprintf("  %s: %d uses (%.1f%% errors)\n",
				module.Name, module.Count, errorRate))
		}
		sb.WriteString("\n")
	}

	// Hosts
	sb.WriteString("🖥️  Hosts:\n")
	sb.WriteString(fmt.Sprintf("  Connected: %d | Unreachable: %d\n",
		summary.Hosts.Connected, summary.Hosts.Unreachable))

	// Cache
	if summary.Cache.Total > 0 {
		sb.WriteString(fmt.Sprintf("💾 Cache: %d hits, %d misses (%.1f%% hit rate)\n",
			summary.Cache.Hits, summary.Cache.Misses, summary.Cache.HitRate))
	}

	// Resource usage
	if summary.ResourceUsage.MemoryUsage > 0 {
		sb.WriteString(fmt.Sprintf("📈 Resources: %.1f MB memory, %.1f%% CPU\n",
			float64(summary.ResourceUsage.MemoryUsage)/1024/1024,
			summary.ResourceUsage.CPUUsage))
	}

	return sb.String()
}
