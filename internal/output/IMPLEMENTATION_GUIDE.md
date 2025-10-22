# Console Output Improvements - Implementation Guide

## Overview

This document describes how to use the new console output components that have been implemented to provide enhanced UX comfort and better information organization.

## Components

### 1. Result Aggregator (`aggregator.go`)

**Purpose**: Groups and organizes execution results by status

**Key Types**:

- `ResultStatus`: Represents result status (success, failed, changed, skipped)
- `AggregatedHost`: A single host result with metadata
- `AggregatedResult`: A grouped set of results by status
- `ResultAggregator`: Main aggregator that collects and groups results

**Usage**:

```go
// Create aggregator
agg := output.NewResultAggregator()

// Add results
agg.Add(output.AggregatedHost{
    Name:     "host1",
    Status:   output.StatusSuccess,
    Duration: 100 * time.Millisecond,
})

// Get aggregated results grouped by status
results := agg.Aggregate()

// Get metrics
metrics := agg.GetMetrics()
```

**Converting from Types**:

```go
// From PlayResult
aggregator := output.FromPlayResult(playResult)

// From TaskResults
aggregator := output.FromTaskResults(taskResults)
```

### 2. Console Formatter (`console_formatter.go`)

**Purpose**: Formats aggregated results into beautiful console output

**Key Types**:

- `ConsoleFormatter`: Main formatter with color support

**Usage**:

```go
// Create formatter (true = no color, false = with color)
formatter := output.NewConsoleFormatter(false)

// Format results
output := formatter.FormatAggregatedResults(aggregated, metrics)

fmt.Println(output)
```

**Output Structure**:

```
EXECUTION RESULTS
=================

✓ SUCCESSFUL (2 hosts, 66.7%)
  ├─ host1                  [100ms]
  └─ host2                  [150ms] ⚡ Changed

✗ FAILED (1 hosts, 33.3%)
  └─ host3                  [200ms]
     │
     └─ Error: Connection refused
        Suggestions:
        • Check if the host is reachable
        • Verify SSH port is open (default: 22)

PERFORMANCE METRICS
===================

  Total hosts:        3
  Successful:         2
  Failed:             1
  Changed:            1
  Total duration:     450ms
  Average per host:   150ms
  Fastest:            100ms
  Slowest:            200ms

──────────────────────────────────────────────
```

### 3. Progress Renderer (`internal/progress/renderer.go`)

**Purpose**: Renders real-time execution progress with host and task details

**Key Types**:

- `ProgressRenderer`: Main renderer
- `HostTaskInfo`: Information about currently running tasks

**Usage**:

```go
// Create renderer
renderer := progress.NewProgressRenderer(false)

// Render batch progress
tasks := []progress.HostTaskInfo{
    {
        Host:      "web1.example.com",
        Task:      "Install nginx",
        StartTime: time.Now().Add(-5 * time.Second),
        Status:    "running",
    },
}

output := renderer.RenderBatchProgress(8, 20, tasks, 30*time.Second)
fmt.Println(output)

// Render summary
summary := renderer.RenderSummaryLine(20, 18, 1, 1, 2*time.Minute)
fmt.Println(summary)
```

**Output**:

```
[████████░░░░░░░░░░░░░░░░░░░░░░] 8/20 (40.0%) | 30s elapsed

⏳ Currently Running:
   • web1.example.com    : Install nginx [5s]
   • web2.example.com    : Configure server [3s]

Summary: 20 total | 18 success | 1 failed | 1 changed | 2m
```

### 4. Error Analyzer (`error_analyzer.go`)

**Purpose**: Categorizes errors and provides helpful suggestions

**Key Types**:

- `ErrorType`: Categorizes errors (connection, timeout, auth, permission, etc.)
- `ErrorAnalyzer`: Main analyzer
- `AnalyzedError`: An error with categorization and suggestions

**Usage**:

```go
// Create analyzer
analyzer := output.NewErrorAnalyzer()

// Analyze a single error
errType, suggestions := analyzer.AnalyzeError("connection refused")

// Analyze error for a host
analyzed := analyzer.AnalyzeHostError("host1", "connection refused")
fmt.Println(analyzed.Type)          // ErrorTypeConnection
fmt.Println(analyzed.Suggestions)    // List of helpful suggestions
fmt.Println(analyzed.RetryAdvice)    // Retry recommendation

// Summarize multiple errors
summary := analyzer.SummarizeErrors([]string{
    "connection refused",
    "connection refused",
    "permission denied",
})

// Group errors by type
grouped := analyzer.GroupErrorsByType(analyzedErrors)
```

### 5. Integration Helper (`integration.go`)

**Purpose**: Provides convenience methods for using all components together

**Usage**:

```go
// Create integration helper
helper := output.NewIntegrationHelper(false) // false = with color

// Process PlayResult
output, metrics := helper.ProcessPlayResult(playResult)
fmt.Println(output)

// Process TaskResults
output, metrics := helper.ProcessTaskResults(taskResults)

// Process HostResults
output, metrics := helper.FormatHostResults(hostResults, "My Playbook")

// Generate error summary report
report := helper.ErrorSummaryReport(errors)
fmt.Println(report)
```

## Integration with Existing Code

### Using with Adhoc Formatter

The new components are designed to work alongside existing output formatters:

```go
// In adhoc execution handler
results := executeAdhocTasks(hosts, tasks)

// Use new formatter for enhanced output
helper := output.NewIntegrationHelper(noColorFlag)
output, metrics := helper.ProcessTaskResults(results)
fmt.Println(output)

// Still support existing formats if needed
if format == "json" {
    // Use existing JSON formatter
}
```

### Using with Playbook Execution

```go
// After playbook execution
for _, playResult := range playResults {
    helper := output.NewIntegrationHelper(noColorFlag)
    output, metrics := helper.ProcessPlayResult(playResult)
    fmt.Println(output)
}
```

## Color Support

Both `ConsoleFormatter` and `ProgressRenderer` respect the `--no-color` flag:

```go
// Disable colors
formatter := output.NewConsoleFormatter(true)

// Enable colors (if terminal supports them)
formatter := output.NewConsoleFormatter(false)
```

Colors are automatically disabled when:

- `NO_COLOR` environment variable is set
- Output is not a TTY
- Running on Windows with unsupported terminal

## Status Hierarchy

Results are displayed in this order:

1. **SUCCESSFUL** - All tasks completed without changes
2. **CHANGED** - Tasks completed with state changes
3. **FAILED** - One or more tasks failed
4. **SKIPPED** - All tasks were skipped

## Error Categories

The error analyzer recognizes and categorizes:

1. **Connection** - Connection refused, connection reset
2. **Timeout** - Operation timed out, timeout
3. **Command** - Command not found, exit code errors
4. **Permission** - Permission denied, access issues
5. **Network** - Network unreachable, no route, DNS issues
6. **Auth** - Authentication failed, SSH key issues
7. **Unknown** - Uncategorized errors

## Performance Metrics

The aggregator calculates:

- Total hosts and task counts
- Success/failure/change statistics
- Percentages for each status group
- Total execution duration
- Average duration per host
- Fastest and slowest execution times

## Testing

Comprehensive tests are included for all components:

```bash
# Run all tests
go test ./internal/output/...
go test ./internal/progress/...

# Run specific test
go test -run TestResultAggregator_Add_And_Aggregate ./internal/output/...
```

## Examples

### Example 1: Basic Usage

```go
package main

import (
    "fmt"
    "time"
    "github.com/onigirazu-cfg/onigirazu/internal/output"
)

func main() {
    agg := output.NewResultAggregator()

    agg.Add(output.AggregatedHost{
        Name:     "web1",
        Status:   output.StatusSuccess,
        Duration: 100 * time.Millisecond,
    })

    formatter := output.NewConsoleFormatter(false)
    results := agg.Aggregate()
    metrics := agg.GetMetrics()

    formatted := formatter.FormatAggregatedResults(results, metrics)
    fmt.Println(formatted)
}
```

### Example 2: Error Analysis

```go
package main

import (
    "fmt"
    "github.com/onigirazu-cfg/onigirazu/internal/output"
)

func main() {
    analyzer := output.NewErrorAnalyzer()

    errors := []string{
        "connection refused",
        "connection refused",
        "permission denied",
        "timeout",
    }

    summary := analyzer.SummarizeErrors(errors)
    fmt.Printf("Total errors: %d\n", summary.Total)
    fmt.Printf("Most common: %s (x%d)\n", summary.MostCommonError, summary.MostCommonCount)
}
```

### Example 3: Full Integration

```go
package main

import (
    "fmt"
    "github.com/onigirazu-cfg/onigirazu/internal/output"
)

func main() {
    helper := output.NewIntegrationHelper(false)

    // Process results
    output, metrics := helper.ProcessTaskResults(taskResults)
    fmt.Println(output)

    // Error summary
    report := helper.ErrorSummaryReport(errors)
    fmt.Println(report)
}
```

## Migration Guide

### Before (Old Way)

```go
for _, result := range results {
    fmt.Printf("%s | %s => %v\n", result.Host, result.Status, result.Msg)
}
```

### After (New Way)

```go
agg := output.FromTaskResults(results)
formatter := output.NewConsoleFormatter(false)
fmt.Println(formatter.FormatAggregatedResults(agg.Aggregate(), agg.GetMetrics()))
```

## Best Practices

1. **Always use helper**: Use `IntegrationHelper` for most use cases
2. **Handle errors**: Error analysis provides actionable suggestions
3. **Color support**: Let users control color output with flags
4. **Performance**: Components are optimized for minimal overhead
5. **Testing**: Write tests for custom status formatters

## Troubleshooting

### Colors not appearing

- Check `NO_COLOR` environment variable
- Verify terminal supports ANSI colors
- Use `--no-color` flag if colors cause issues

### Missing metrics

- Ensure all hosts have Duration set
- Check for zero duration hosts
- Metrics calculate average even with zero durations

### Truncated hostnames

- Progress renderer truncates names over 20 chars
- Formatter truncates names over 25 chars
- Consider shorter hostname aliases in inventory
