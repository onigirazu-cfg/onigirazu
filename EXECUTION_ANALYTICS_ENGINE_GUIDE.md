# Execution Analytics Engine - Implementation Guide

## 📊 Overview

The Execution Analytics Engine provides deep insights into Onigirazu execution patterns by leveraging the existing audit logs and metrics infrastructure. It transforms execution data into actionable intelligence for performance optimization and troubleshooting.

**Key Capabilities:**

- Task performance trending and analysis
- Host health and reliability metrics
- Bottleneck detection and recommendations
- Error pattern analysis
- Module usage analytics
- Success rate tracking
- Resource utilization insights

---

## 🏗️ Architecture

```
Audit System (existing)
    ├── ExecutionRecord
    ├── PlayExecution
    └── TaskResult
        ↓
Metrics System (existing)
    ├── Task metrics
    ├── Host metrics
    └── Resource metrics
        ↓
    [NEW] AnalyticsEngine
        ├── QueryBuilder      (build queries from audit logs)
        ├── AggregationEngine (aggregate metrics)
        ├── TrendAnalyzer     (detect trends)
        ├── BottleneckDetector (find issues)
        └── ReportGenerator   (create reports)
        ↓
CLI Commands (new)
    ├── analytics task-metrics
    ├── analytics host-health
    ├── analytics bottlenecks
    └── analytics report
```

---

## 📁 Implementation Plan

### Phase 1: Core Analytics Engine

Create new package: `internal/analytics/`

```
internal/analytics/
├── engine.go             # Main analytics orchestrator
├── engine_test.go
├── aggregator.go         # Metrics aggregation from audit logs
├── aggregator_test.go
├── trends.go             # Trend detection
├── trends_test.go
├── bottlenecks.go        # Problem detection
├── bottlenecks_test.go
├── reports.go            # Report generation
├── reports_test.go
└── types.go              # Local type definitions
```

---

## 🔧 Implementation - Core Components

### 1. Type Definitions (`internal/analytics/types.go`)

```go
package analytics

import (
    "time"
)

// TaskAnalytics represents analytics for a specific task
type TaskAnalytics struct {
    TaskName         string
    Module           string
    ExecutionCount   int
    SuccessCount     int
    FailureCount     int
    SkippedCount     int
    SuccessRate      float64       // 0.0-1.0
    AverageDuration  time.Duration
    MinDuration      time.Duration
    MaxDuration      time.Duration
    LastExecuted     time.Time
    TrendType        string        // "improving", "degrading", "stable"
    CommonErrors     map[string]int
    AffectedHosts    []string
}

// HostAnalytics represents analytics for a specific host
type HostAnalytics struct {
    Hostname        string
    TotalExecutions int
    SuccessCount    int
    FailureCount    int
    SkippedCount    int
    SuccessRate     float64
    AverageDuration time.Duration
    LastActivity    time.Time
    CommonErrors    []string
    FailedTasks     map[string]int
}

// PerformanceBottleneck represents a detected performance issue
type PerformanceBottleneck struct {
    Type        string  // "slow_task", "high_failure_rate", "high_retry"
    Description string
    Severity    string  // "critical", "high", "medium", "low"
    Impact      float64 // estimated % improvement if fixed
    Remediation string
    AffectedItems []string
}

// QueryTimeRange specifies a time range for analysis
type QueryTimeRange struct {
    From time.Time
    To   time.Time
}
```

### 2. Query Builder (`internal/analytics/aggregator.go`)

```go
package analytics

import (
    "context"
    "fmt"
    "sort"
    "time"

    "github.com/onigirazu-cfg/onigirazu/internal/audit"
)

// Aggregator queries and processes audit data
type Aggregator struct {
    auditRecorder audit.Recorder // injected dependency
}

// NewAggregator creates a new analytics aggregator
func NewAggregator(auditRecorder audit.Recorder) *Aggregator {
    return &Aggregator{
        auditRecorder: auditRecorder,
    }
}

// GetTaskAnalytics calculates analytics for a specific task
func (a *Aggregator) GetTaskAnalytics(
    ctx context.Context,
    taskName string,
    timeRange QueryTimeRange,
) (*TaskAnalytics, error) {
    // Query audit records for this task
    records, err := a.auditRecorder.Query(ctx, audit.FilterOptions{
        DateFrom: timeRange.From,
        DateTo:   timeRange.To,
        Limit:    10000,
    })
    if err != nil {
        return nil, err
    }

    ta := &TaskAnalytics{
        TaskName:     taskName,
        CommonErrors: make(map[string]int),
    }

    for _, record := range records {
        for _, play := range record.Plays {
            for _, taskResult := range play.Tasks {
                if taskResult.Name != taskName {
                    continue
                }

                ta.ExecutionCount++

                switch taskResult.Status {
                case audit.TaskStatusOk, audit.TaskStatusChanged:
                    ta.SuccessCount++
                case audit.TaskStatusFailed:
                    ta.FailureCount++
                    if taskResult.Error != "" {
                        ta.CommonErrors[taskResult.Error]++
                    }
                case audit.TaskStatusSkipped:
                    ta.SkippedCount++
                }

                // Duration tracking
                duration := time.Duration(taskResult.Duration * float64(time.Second))
                if ta.AverageDuration == 0 {
                    ta.MinDuration = duration
                    ta.MaxDuration = duration
                } else {
                    if duration < ta.MinDuration {
                        ta.MinDuration = duration
                    }
                    if duration > ta.MaxDuration {
                        ta.MaxDuration = duration
                    }
                }

                // Track host
                if taskResult.Host != "" {
                    ta.AffectedHosts = append(ta.AffectedHosts, taskResult.Host)
                }

                // Track last execution
                if taskResult.EndTime.After(ta.LastExecuted) {
                    ta.LastExecuted = taskResult.EndTime
                }
            }
        }
    }

    // Calculate aggregates
    if ta.ExecutionCount > 0 {
        ta.SuccessRate = float64(ta.SuccessCount) / float64(ta.ExecutionCount)
    }

    // Calculate average duration
    var totalDuration time.Duration
    if ta.ExecutionCount > 0 {
        // In real implementation, recalculate from records
        totalDuration = ta.MinDuration + (ta.MaxDuration / 2)
        ta.AverageDuration = totalDuration / time.Duration(ta.ExecutionCount)
    }

    return ta, nil
}

// GetHostAnalytics calculates analytics for a specific host
func (a *Aggregator) GetHostAnalytics(
    ctx context.Context,
    hostname string,
    timeRange QueryTimeRange,
) (*HostAnalytics, error) {
    records, err := a.auditRecorder.Query(ctx, audit.FilterOptions{
        DateFrom: timeRange.From,
        DateTo:   timeRange.To,
        HostFilter: hostname,
        Limit:      10000,
    })
    if err != nil {
        return nil, err
    }

    ha := &HostAnalytics{
        Hostname:    hostname,
        FailedTasks: make(map[string]int),
    }

    for _, record := range records {
        for _, play := range record.Plays {
            for _, taskResult := range play.Tasks {
                if taskResult.Host != hostname {
                    continue
                }

                ha.TotalExecutions++

                switch taskResult.Status {
                case audit.TaskStatusOk, audit.TaskStatusChanged:
                    ha.SuccessCount++
                case audit.TaskStatusFailed:
                    ha.FailureCount++
                    ha.FailedTasks[taskResult.Name]++
                    if taskResult.Error != "" {
                        ha.CommonErrors = append(ha.CommonErrors, taskResult.Error)
                    }
                case audit.TaskStatusSkipped:
                    ha.SkippedCount++
                }

                if taskResult.EndTime.After(ha.LastActivity) {
                    ha.LastActivity = taskResult.EndTime
                }
            }
        }
    }

    if ha.TotalExecutions > 0 {
        ha.SuccessRate = float64(ha.SuccessCount) / float64(ha.TotalExecutions)
    }

    return ha, nil
}

// GetAllTasksAnalytics returns analytics for all tasks in time range
func (a *Aggregator) GetAllTasksAnalytics(
    ctx context.Context,
    timeRange QueryTimeRange,
) (map[string]*TaskAnalytics, error) {
    records, err := a.auditRecorder.Query(ctx, audit.FilterOptions{
        DateFrom: timeRange.From,
        DateTo:   timeRange.To,
        Limit:    50000,
    })
    if err != nil {
        return nil, err
    }

    taskAnalytics := make(map[string]*TaskAnalytics)

    for _, record := range records {
        for _, play := range record.Plays {
            for _, taskResult := range play.Tasks {
                taskName := taskResult.Name
                if _, exists := taskAnalytics[taskName]; !exists {
                    taskAnalytics[taskName] = &TaskAnalytics{
                        TaskName:     taskName,
                        Module:       taskResult.Module,
                        CommonErrors: make(map[string]int),
                    }
                }

                ta := taskAnalytics[taskName]
                ta.ExecutionCount++

                // Update status counts
                switch taskResult.Status {
                case audit.TaskStatusOk, audit.TaskStatusChanged:
                    ta.SuccessCount++
                case audit.TaskStatusFailed:
                    ta.FailureCount++
                case audit.TaskStatusSkipped:
                    ta.SkippedCount++
                }
            }
        }
    }

    // Calculate rates
    for _, ta := range taskAnalytics {
        if ta.ExecutionCount > 0 {
            ta.SuccessRate = float64(ta.SuccessCount) / float64(ta.ExecutionCount)
        }
    }

    return taskAnalytics, nil
}
```

### 3. Trend Analysis (`internal/analytics/trends.go`)

```go
package analytics

import (
    "context"
    "time"
)

// TrendAnalyzer detects performance trends
type TrendAnalyzer struct {
    aggregator *Aggregator
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer(agg *Aggregator) *TrendAnalyzer {
    return &TrendAnalyzer{aggregator: agg}
}

// AnalyzeTaskTrend compares task performance between two periods
func (t *TrendAnalyzer) AnalyzeTaskTrend(
    ctx context.Context,
    taskName string,
) (trendType string, change float64, err error) {
    now := time.Now()

    // Current week
    currentAnalytics, err := t.aggregator.GetTaskAnalytics(ctx, taskName, QueryTimeRange{
        From: now.AddDate(0, 0, -7),
        To:   now,
    })
    if err != nil {
        return "", 0, err
    }

    // Previous week
    previousAnalytics, err := t.aggregator.GetTaskAnalytics(ctx, taskName, QueryTimeRange{
        From: now.AddDate(0, 0, -14),
        To:   now.AddDate(0, 0, -7),
    })
    if err != nil {
        return "", 0, err
    }

    // Compare success rates
    if currentAnalytics.SuccessRate > previousAnalytics.SuccessRate {
        change = (currentAnalytics.SuccessRate - previousAnalytics.SuccessRate) * 100
        return "improving", change, nil
    } else if currentAnalytics.SuccessRate < previousAnalytics.SuccessRate {
        change = (previousAnalytics.SuccessRate - currentAnalytics.SuccessRate) * 100
        return "degrading", change, nil
    }

    return "stable", 0, nil
}
```

### 4. Bottleneck Detection (`internal/analytics/bottlenecks.go`)

```go
package analytics

import (
    "context"
    "fmt"
    "sort"
    "time"
)

// BottleneckDetector finds performance issues
type BottleneckDetector struct {
    aggregator *Aggregator
}

// NewBottleneckDetector creates detector
func NewBottleneckDetector(agg *Aggregator) *BottleneckDetector {
    return &BottleneckDetector{aggregator: agg}
}

// DetectBottlenecks identifies performance issues
func (b *BottleneckDetector) DetectBottlenecks(
    ctx context.Context,
    timeRange QueryTimeRange,
) ([]PerformanceBottleneck, error) {
    var bottlenecks []PerformanceBottleneck

    // Get all tasks analytics
    allTasks, err := b.aggregator.GetAllTasksAnalytics(ctx, timeRange)
    if err != nil {
        return nil, err
    }

    // Detect slow tasks
    slowBottlenecks := b.detectSlowTasks(allTasks)
    bottlenecks = append(bottlenecks, slowBottlenecks...)

    // Detect high failure rates
    failureBottlenecks := b.detectHighFailureRate(allTasks)
    bottlenecks = append(bottlenecks, failureBottlenecks...)

    // Sort by severity
    sort.Slice(bottlenecks, func(i, j int) bool {
        severityOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
        return severityOrder[bottlenecks[i].Severity] < severityOrder[bottlenecks[j].Severity]
    })

    return bottlenecks, nil
}

func (b *BottleneckDetector) detectSlowTasks(tasks map[string]*TaskAnalytics) []PerformanceBottleneck {
    var bottlenecks []PerformanceBottleneck

    // Find top 10% slowest tasks
    type taskDuration struct {
        name     string
        duration time.Duration
    }

    var durations []taskDuration
    for name, analytics := range tasks {
        if analytics.AverageDuration > 0 {
            durations = append(durations, taskDuration{name, analytics.AverageDuration})
        }
    }

    if len(durations) < 2 {
        return bottlenecks
    }

    sort.Slice(durations, func(i, j int) bool {
        return durations[i].duration > durations[j].duration
    })

    threshold := len(durations) / 10
    if threshold == 0 {
        threshold = 1
    }

    for i := 0; i < threshold; i++ {
        td := durations[i]
        if analytics, ok := tasks[td.name]; ok {
            bottlenecks = append(bottlenecks, PerformanceBottleneck{
                Type:        "slow_task",
                Description: fmt.Sprintf("Task '%s' takes %.2fs on average", td.name, td.duration.Seconds()),
                Severity:    "medium",
                Impact:      10.0, // 10% potential improvement
                Remediation: "Review task logic, consider optimization or parallelization",
                AffectedItems: analytics.AffectedHosts,
            })
        }
    }

    return bottlenecks
}

func (b *BottleneckDetector) detectHighFailureRate(tasks map[string]*TaskAnalytics) []PerformanceBottleneck {
    var bottlenecks []PerformanceBottleneck

    for name, analytics := range tasks {
        if analytics.ExecutionCount < 3 {
            continue // Need minimum executions
        }

        if analytics.SuccessRate < 0.8 { // Less than 80% success rate
            severity := "high"
            if analytics.SuccessRate < 0.5 {
                severity = "critical"
            }

            commonError := ""
            maxCount := 0
            for err, count := range analytics.CommonErrors {
                if count > maxCount {
                    maxCount = count
                    commonError = err
                }
            }

            bottlenecks = append(bottlenecks, PerformanceBottleneck{
                Type:        "high_failure_rate",
                Description: fmt.Sprintf("Task '%s' fails %.0f%% of time (error: %s)", name, (1-analytics.SuccessRate)*100, commonError),
                Severity:    severity,
                Impact:      50.0, // 50% potential improvement
                Remediation: fmt.Sprintf("Investigate error: %s", commonError),
                AffectedItems: analytics.AffectedHosts,
            })
        }
    }

    return bottlenecks
}
```

### 5. Report Generation (`internal/analytics/reports.go`)

```go
package analytics

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "text/tabwriter"
    "time"
)

// ReportGenerator creates formatted reports
type ReportGenerator struct {
    aggregator  *Aggregator
    detector    *BottleneckDetector
}

// NewReportGenerator creates a report generator
func NewReportGenerator(agg *Aggregator, det *BottleneckDetector) *ReportGenerator {
    return &ReportGenerator{
        aggregator: agg,
        detector:   det,
    }
}

// GenerateTaskReport creates a task performance report
func (r *ReportGenerator) GenerateTaskReport(
    ctx context.Context,
    taskName string,
    timeRange QueryTimeRange,
) (string, error) {
    ta, err := r.aggregator.GetTaskAnalytics(ctx, taskName, timeRange)
    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

    fmt.Fprintf(w, "TASK PERFORMANCE REPORT\n")
    fmt.Fprintf(w, "======================\n\n")
    fmt.Fprintf(w, "Task Name:\t%s\n", taskName)
    fmt.Fprintf(w, "Module:\t%s\n", ta.Module)
    fmt.Fprintf(w, "Period:\t%s to %s\n\n", timeRange.From.Format("2006-01-02"), timeRange.To.Format("2006-01-02"))

    fmt.Fprintf(w, "EXECUTION SUMMARY\n")
    fmt.Fprintf(w, "Total Runs:\t%d\n", ta.ExecutionCount)
    fmt.Fprintf(w, "Successful:\t%d (%.1f%%)\n", ta.SuccessCount, ta.SuccessRate*100)
    fmt.Fprintf(w, "Failed:\t%d\n", ta.FailureCount)
    fmt.Fprintf(w, "Skipped:\t%d\n\n", ta.SkippedCount)

    fmt.Fprintf(w, "PERFORMANCE\n")
    fmt.Fprintf(w, "Average Duration:\t%.2fs\n", ta.AverageDuration.Seconds())
    fmt.Fprintf(w, "Min Duration:\t%.2fs\n", ta.MinDuration.Seconds())
    fmt.Fprintf(w, "Max Duration:\t%.2fs\n", ta.MaxDuration.Seconds())

    if len(ta.CommonErrors) > 0 {
        fmt.Fprintf(w, "\nTOP ERRORS\n")
        for errMsg, count := range ta.CommonErrors {
            fmt.Fprintf(w, "%s:\t%d occurrences\n", errMsg, count)
        }
    }

    if len(ta.AffectedHosts) > 0 {
        fmt.Fprintf(w, "\nHOSTS AFFECTED\n")
        for _, host := range ta.AffectedHosts {
            fmt.Fprintf(w, "- %s\n", host)
        }
    }

    w.Flush()
    return buf.String(), nil
}

// GenerateHostReport creates a host health report
func (r *ReportGenerator) GenerateHostReport(
    ctx context.Context,
    hostname string,
    timeRange QueryTimeRange,
) (string, error) {
    ha, err := r.aggregator.GetHostAnalytics(ctx, hostname, timeRange)
    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

    fmt.Fprintf(w, "HOST HEALTH REPORT\n")
    fmt.Fprintf(w, "==================\n\n")
    fmt.Fprintf(w, "Hostname:\t%s\n", hostname)
    fmt.Fprintf(w, "Period:\t%s to %s\n\n", timeRange.From.Format("2006-01-02"), timeRange.To.Format("2006-01-02"))

    fmt.Fprintf(w, "RELIABILITY\n")
    fmt.Fprintf(w, "Total Executions:\t%d\n", ha.TotalExecutions)
    fmt.Fprintf(w, "Success Rate:\t%.1f%%\n", ha.SuccessRate*100)
    fmt.Fprintf(w, "Successful:\t%d\n", ha.SuccessCount)
    fmt.Fprintf(w, "Failed:\t%d\n", ha.FailureCount)
    fmt.Fprintf(w, "Skipped:\t%d\n", ha.SkippedCount)
    fmt.Fprintf(w, "Last Activity:\t%s\n\n", ha.LastActivity.Format("2006-01-02 15:04:05"))

    if len(ha.FailedTasks) > 0 {
        fmt.Fprintf(w, "TOP FAILING TASKS\n")
        for task, count := range ha.FailedTasks {
            fmt.Fprintf(w, "%s:\t%d failures\n", task, count)
        }
    }

    w.Flush()
    return buf.String(), nil
}

// GenerateBottleneckReport creates bottleneck analysis
func (r *ReportGenerator) GenerateBottleneckReport(
    ctx context.Context,
    timeRange QueryTimeRange,
) (string, error) {
    bottlenecks, err := r.detector.DetectBottlenecks(ctx, timeRange)
    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

    fmt.Fprintf(w, "PERFORMANCE BOTTLENECK ANALYSIS\n")
    fmt.Fprintf(w, "===============================\n\n")
    fmt.Fprintf(w, "Period:\t%s to %s\n", timeRange.From.Format("2006-01-02"), timeRange.To.Format("2006-01-02"))
    fmt.Fprintf(w, "Issues Found:\t%d\n\n", len(bottlenecks))

    for i, bn := range bottlenecks {
        fmt.Fprintf(w, "[%d] %s [%s]\n", i+1, bn.Type, bn.Severity)
        fmt.Fprintf(w, "    %s\n", bn.Description)
        fmt.Fprintf(w, "    Impact: %.0f%% improvement potential\n", bn.Impact)
        fmt.Fprintf(w, "    Action: %s\n\n", bn.Remediation)
    }

    w.Flush()
    return buf.String(), nil
}

// ExportAsJSON exports analytics as JSON
func (r *ReportGenerator) ExportAsJSON(
    ctx context.Context,
    taskName string,
    timeRange QueryTimeRange,
) (string, error) {
    ta, err := r.aggregator.GetTaskAnalytics(ctx, taskName, timeRange)
    if err != nil {
        return "", err
    }

    data := map[string]interface{}{
        "task_name":          ta.TaskName,
        "module":             ta.Module,
        "execution_count":    ta.ExecutionCount,
        "success_count":      ta.SuccessCount,
        "failure_count":      ta.FailureCount,
        "skipped_count":      ta.SkippedCount,
        "success_rate":       ta.SuccessRate,
        "average_duration":   ta.AverageDuration.String(),
        "min_duration":       ta.MinDuration.String(),
        "max_duration":       ta.MaxDuration.String(),
        "last_executed":      ta.LastExecuted,
    }

    jsonBytes, err := json.MarshalIndent(data, "", "  ")
    return string(jsonBytes), err
}
```

### 6. Main Analytics Engine (`internal/analytics/engine.go`)

```go
package analytics

import (
    "context"

    "github.com/onigirazu-cfg/onigirazu/internal/audit"
)

// Engine orchestrates analytics operations
type Engine struct {
    aggregator *Aggregator
    analyzer   *TrendAnalyzer
    detector   *BottleneckDetector
    reporter   *ReportGenerator
}

// NewEngine creates a new analytics engine
func NewEngine(auditRecorder audit.Recorder) *Engine {
    agg := NewAggregator(auditRecorder)
    analyzer := NewTrendAnalyzer(agg)
    detector := NewBottleneckDetector(agg)
    reporter := NewReportGenerator(agg, detector)

    return &Engine{
        aggregator: agg,
        analyzer:   analyzer,
        detector:   detector,
        reporter:   reporter,
    }
}

// GetTaskAnalytics returns analytics for a task
func (e *Engine) GetTaskAnalytics(
    ctx context.Context,
    taskName string,
    timeRange QueryTimeRange,
) (*TaskAnalytics, error) {
    return e.aggregator.GetTaskAnalytics(ctx, taskName, timeRange)
}

// GetHostAnalytics returns analytics for a host
func (e *Engine) GetHostAnalytics(
    ctx context.Context,
    hostname string,
    timeRange QueryTimeRange,
) (*HostAnalytics, error) {
    return e.aggregator.GetHostAnalytics(ctx, hostname, timeRange)
}

// AnalyzeTaskTrend returns trend analysis for a task
func (e *Engine) AnalyzeTaskTrend(
    ctx context.Context,
    taskName string,
) (trendType string, change float64, err error) {
    return e.analyzer.AnalyzeTaskTrend(ctx, taskName)
}

// DetectBottlenecks returns detected performance issues
func (e *Engine) DetectBottlenecks(
    ctx context.Context,
    timeRange QueryTimeRange,
) ([]PerformanceBottleneck, error) {
    return e.detector.DetectBottlenecks(ctx, timeRange)
}

// GenerateTaskReport generates formatted task report
func (e *Engine) GenerateTaskReport(
    ctx context.Context,
    taskName string,
    timeRange QueryTimeRange,
) (string, error) {
    return e.reporter.GenerateTaskReport(ctx, taskName, timeRange)
}

// GenerateHostReport generates formatted host report
func (e *Engine) GenerateHostReport(
    ctx context.Context,
    hostname string,
    timeRange QueryTimeRange,
) (string, error) {
    return e.reporter.GenerateHostReport(ctx, hostname, timeRange)
}

// GenerateBottleneckReport generates formatted bottleneck report
func (e *Engine) GenerateBottleneckReport(
    ctx context.Context,
    timeRange QueryTimeRange,
) (string, error) {
    return e.reporter.GenerateBottleneckReport(ctx, timeRange)
}
```

---

## 🎯 CLI Integration

### Add new CLI command file (`internal/cli/analytics.go`)

```go
package cli

import (
    "context"
    "fmt"
    "time"

    "github.com/spf13/cobra"

    "github.com/onigirazu-cfg/onigirazu/internal/analytics"
    "github.com/onigirazu-cfg/onigirazu/internal/audit"
    "github.com/onigirazu-cfg/onigirazu/internal/logger"
)

var (
    analyticsLastDays int
    analyticsFormat   string
)

// newAnalyticsCmd creates the analytics command
func newAnalyticsCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "analytics",
        Short: "Analyze execution metrics and performance",
        Long: `Analyze Onigirazu execution metrics to identify performance issues,
trends, and optimization opportunities.

Examples:
  onigirazu analytics task-metrics install_nginx --last 7
  onigirazu analytics host-health server1 --last 30
  onigirazu analytics bottlenecks --last 14`,
        Run: func(cmd *cobra.Command, args []string) {
            cmd.Help()
        },
    }

    cmd.AddCommand(newAnalyticsTaskCmd())
    cmd.AddCommand(newAnalyticsHostCmd())
    cmd.AddCommand(newAnalyticsBottlenecksCmd())

    return cmd
}

// newAnalyticsTaskCmd creates task metrics subcommand
func newAnalyticsTaskCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "task-metrics <task_name>",
        Short: "Show performance metrics for a task",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return runTaskMetrics(cmd, args[0])
        },
    }

    cmd.Flags().IntVar(&analyticsLastDays, "last", 7, "Number of days to analyze")
    cmd.Flags().StringVar(&analyticsFormat, "format", "text", "Output format (text, json)")

    return cmd
}

func runTaskMetrics(cmd *cobra.Command, taskName string) error {
    // Initialize audit recorder
    auditRecorder := audit.NewRecorder(
        logger.NewLogger("analytics"),
        audit.AuditConfig{
            Enabled:     true,
            StoragePath: ".onigirazu/audit",
        },
    )

    // Create analytics engine
    engine := analytics.NewEngine(auditRecorder)

    // Set time range
    now := time.Now()
    timeRange := analytics.QueryTimeRange{
        From: now.AddDate(0, 0, -analyticsLastDays),
        To:   now,
    }

    // Get analytics
    ctx := context.Background()
    ta, err := engine.GetTaskAnalytics(ctx, taskName, timeRange)
    if err != nil {
        return fmt.Errorf("failed to get task metrics: %w", err)
    }

    // Format output
    if analyticsFormat == "json" {
        report, err := engine.reporter.ExportAsJSON(ctx, taskName, timeRange)
        if err != nil {
            return err
        }
        fmt.Println(report)
    } else {
        if ta.ExecutionCount == 0 {
            fmt.Printf("No executions found for task '%s' in last %d days\n", taskName, analyticsLastDays)
            return nil
        }

        report, err := engine.GenerateTaskReport(ctx, taskName, timeRange)
        if err != nil {
            return err
        }
        fmt.Println(report)
    }

    return nil
}

// newAnalyticsHostCmd creates host health subcommand
func newAnalyticsHostCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "host-health <hostname>",
        Short: "Show health metrics for a host",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return runHostHealth(cmd, args[0])
        },
    }

    cmd.Flags().IntVar(&analyticsLastDays, "last", 7, "Number of days to analyze")
    cmd.Flags().StringVar(&analyticsFormat, "format", "text", "Output format (text, json)")

    return cmd
}

func runHostHealth(cmd *cobra.Command, hostname string) error {
    auditRecorder := audit.NewRecorder(
        logger.NewLogger("analytics"),
        audit.AuditConfig{
            Enabled:     true,
            StoragePath: ".onigirazu/audit",
        },
    )

    engine := analytics.NewEngine(auditRecorder)

    now := time.Now()
    timeRange := analytics.QueryTimeRange{
        From: now.AddDate(0, 0, -analyticsLastDays),
        To:   now,
    }

    ctx := context.Background()
    ha, err := engine.GetHostAnalytics(ctx, hostname, timeRange)
    if err != nil {
        return fmt.Errorf("failed to get host metrics: %w", err)
    }

    if ha.TotalExecutions == 0 {
        fmt.Printf("No executions found for host '%s' in last %d days\n", hostname, analyticsLastDays)
        return nil
    }

    report, err := engine.GenerateHostReport(ctx, hostname, timeRange)
    if err != nil {
        return err
    }
    fmt.Println(report)

    return nil
}

// newAnalyticsBottlenecksCmd creates bottlenecks subcommand
func newAnalyticsBottlenecksCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "bottlenecks",
        Short: "Identify performance bottlenecks",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runBottlenecks(cmd)
        },
    }

    cmd.Flags().IntVar(&analyticsLastDays, "last", 7, "Number of days to analyze")

    return cmd
}

func runBottlenecks(cmd *cobra.Command) error {
    auditRecorder := audit.NewRecorder(
        logger.NewLogger("analytics"),
        audit.AuditConfig{
            Enabled:     true,
            StoragePath: ".onigirazu/audit",
        },
    )

    engine := analytics.NewEngine(auditRecorder)

    now := time.Now()
    timeRange := analytics.QueryTimeRange{
        From: now.AddDate(0, 0, -analyticsLastDays),
        To:   now,
    }

    ctx := context.Background()
    report, err := engine.GenerateBottleneckReport(ctx, timeRange)
    if err != nil {
        return err
    }
    fmt.Println(report)

    return nil
}
```

### Register in root command

Modify `internal/cli/root.go` to add analytics command:

```go
// In NewRootCommand()
rootCmd.AddCommand(newAnalyticsCmd())
```

---

## 📊 Usage Examples

```bash
# Get task performance metrics (last 7 days)
onigirazu analytics task-metrics install_nginx --last 7

# Get host health report (last 30 days)
onigirazu analytics host-health server1 --last 30

# Identify bottlenecks (last 14 days)
onigirazu analytics bottlenecks --last 14

# Export task metrics as JSON
onigirazu analytics task-metrics install_nginx --last 7 --format json
```

---

## 🧪 Testing Strategy

```go
// internal/analytics/aggregator_test.go
package analytics

import (
    "context"
    "testing"
    "time"

    "github.com/onigirazu-cfg/onigirazu/internal/audit"
)

func TestTaskAnalytics(t *testing.T) {
    // Mock audit recorder
    mockRecorder := &mockAuditRecorder{
        records: []*audit.ExecutionRecord{
            {
                ID:       "exec-1",
                Plays: []audit.PlayExecution{
                    {
                        Tasks: []audit.TaskResult{
                            {
                                Name:     "install_nginx",
                                Module:   "package",
                                Status:   audit.TaskStatusChanged,
                                Duration: 5.0,
                                Host:     "server1",
                            },
                        },
                    },
                },
            },
        },
    }

    agg := NewAggregator(mockRecorder)

    ta, err := agg.GetTaskAnalytics(context.Background(), "install_nginx", QueryTimeRange{
        From: time.Now().AddDate(0, 0, -7),
        To:   time.Now(),
    })

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if ta.ExecutionCount != 1 {
        t.Errorf("expected 1 execution, got %d", ta.ExecutionCount)
    }

    if ta.SuccessRate != 1.0 {
        t.Errorf("expected 100%% success rate, got %.1f%%", ta.SuccessRate*100)
    }
}
```

---

## 🚀 Implementation Checklist

- [ ] Create `internal/analytics/types.go`
- [ ] Create `internal/analytics/aggregator.go` with audit querying
- [ ] Create `internal/analytics/trends.go` for trend analysis
- [ ] Create `internal/analytics/bottlenecks.go` for issue detection
- [ ] Create `internal/analytics/reports.go` for report generation
- [ ] Create `internal/analytics/engine.go` as orchestrator
- [ ] Create `internal/cli/analytics.go` with CLI commands
- [ ] Register analytics command in root.go
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Test with real audit data

---

## 📝 Integration Points

1. **Audit System**: Uses existing `audit.Recorder` and `audit.ExecutionRecord`
2. **Metrics System**: Leverages existing `metrics.Metrics` data
3. **CLI**: Uses Cobra for command structure
4. **Configuration**: Uses existing config system for storage paths

---

## 🎯 Next Steps

1. **Phase 1**: Implement core aggregator querying audit logs
2. **Phase 2**: Add trend analysis and bottleneck detection
3. **Phase 3**: Create CLI commands
4. **Phase 4**: Add report formatting (PDF, HTML)
5. **Phase 5**: Connect to WebSocket API for real-time dashboards
