# Implementation Guide: --list-tags and --list-tasks Features

## Overview

This document provides a technical guide for implementing the `--list-tags` and `--list-tasks` CLI flags in Onigirazu.

## Feature Requirements

### `--list-tags` Flag

**Purpose**: List all available tags in a playbook without executing any tasks.

**Requirements**:

1. Parse the playbook YAML file
2. Extract all unique tags from all tasks
3. Count tag usage (how many tasks use each tag)
4. Identify special tags (`always`, `never`)
5. Display results in human-readable format
6. Support multiple output formats (text, JSON, YAML, CSV)
7. Exit code 0 on success, non-zero on error

**Input**:

- Playbook path (required, e.g., `playbook.yml`)
- Optional flags: `--inventory`, `--config`, `--output`

**Output Example**:

```
Available tags in playbook.yml:

  Tag Name          Count  Details
  ─────────────────────────────────────────────────────────
  setup             3      Used in: Install Base Packages, Configure Firewall
  security          2      Used in: Configure Firewall, Create User
  deployment        2      Used in: Deploy Latest Code, Install Dependencies
  always            1      Special tag: Always runs
  never             1      Special tag: Never runs by default

Summary:
  - 4 unique tags
  - 7 tasks total
  - 1 always task
  - 1 never task
```

### `--list-tasks` Flag

**Purpose**: Preview which tasks would execute with current tag filters.

**Requirements**:

1. Parse the playbook YAML file
2. Apply tag filtering logic (same as apply command)
3. For each play, show which tasks would execute
4. Indicate why tasks are skipped
5. Support `--tags` and `--skip-tags` filters
6. Support `--verbose` for detailed information
7. Support multiple output formats (text, JSON, YAML, CSV)
8. Display execution order
9. Exit code 0 on success, non-zero on error

**Input**:

- Playbook path (required)
- Optional flags:
  - `--tags TAG1,TAG2,...`
  - `--skip-tags TAG1,TAG2,...`
  - `--inventory`
  - `--config`
  - `--output` (text, json, yaml, csv)
  - `--verbose`

**Output Example**:

```
Tasks that would execute:

Play 1: Infrastructure Setup (Hosts: all)
  ✓ [setup, packages] Install Base Packages
  ✓ [setup, security] Configure Firewall
  ✓ [always] Health Check [ALWAYS TAG]

Play 2: Application Deployment (Hosts: webservers)
  ✓ [deployment] Deploy Latest Code
  ✗ [debug, never] Run Debug Info [SKIPPED: NEVER TAG]

Summary:
  - 4 would execute
  - 1 skipped (never tag)
  - 2 hosts targeted
```

## Implementation Steps

### Step 1: Modify CLI Flag Definitions

**File**: `internal/cli/apply.go`

Add new boolean flags to the `NewApplyCommand` function:

```go
var (
    // ... existing flags ...
    listTags  bool  // NEW
    listTasks bool  // NEW
)

cmd.Flags().BoolVar(&listTags, "list-tags", false,
    "List all available tags in the playbook")
cmd.Flags().BoolVar(&listTasks, "list-tasks", false,
    "List tasks that would execute with current filters")
```

### Step 2: Create Tag Discovery Service

**File**: `internal/tagdiscovery/discovery.go` (NEW)

```go
package tagdiscovery

import (
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TagInfo contains information about a tag
type TagInfo struct {
    Name      string
    Count     int
    IsSpecial bool  // true for 'always', 'never'
    Details   []string
}

// ListTagsResult contains all tags in a playbook
type ListTagsResult struct {
    Tags     map[string]*TagInfo
    Special  map[string]*TagInfo  // 'always', 'never'
    Summary  TagsSummary
}

type TagsSummary struct {
    TotalTags      int
    TotalTasks     int
    AlwaysTasks    int
    NeverTasks     int
    TaggedTasks    int
    UntaggedTasks  int
}

// DiscoverTags extracts all unique tags from a playbook
func DiscoverTags(playbook *types.Playbook) (*ListTagsResult, error) {
    result := &ListTagsResult{
        Tags:    make(map[string]*TagInfo),
        Special: make(map[string]*TagInfo),
    }

    taskCount := 0

    for _, play := range playbook.Plays {
        for _, task := range play.Tasks {
            taskCount++

            if len(task.Tags) == 0 {
                result.Summary.UntaggedTasks++
            } else {
                result.Summary.TaggedTasks++
            }

            for _, tag := range task.Tags {
                normalizedTag := strings.ToLower(tag)

                if normalizedTag == "always" || normalizedTag == "never" {
                    if result.Special[normalizedTag] == nil {
                        result.Special[normalizedTag] = &TagInfo{
                            Name:      normalizedTag,
                            IsSpecial: true,
                        }
                    }
                    result.Special[normalizedTag].Count++
                    if normalizedTag == "always" {
                        result.Summary.AlwaysTasks++
                    } else {
                        result.Summary.NeverTasks++
                    }
                } else {
                    if result.Tags[normalizedTag] == nil {
                        result.Tags[normalizedTag] = &TagInfo{
                            Name:    normalizedTag,
                            Details: []string{},
                        }
                    }
                    result.Tags[normalizedTag].Count++
                    result.Tags[normalizedTag].Details = append(
                        result.Tags[normalizedTag].Details,
                        task.Name,
                    )
                }
            }
        }
    }

    result.Summary.TotalTags = len(result.Tags) + len(result.Special)
    result.Summary.TotalTasks = taskCount

    return result, nil
}
```

### Step 3: Create Task Preview Service

**File**: `internal/taskpreview/preview.go` (NEW)

```go
package taskpreview

import (
    "github.com/onigirazu-cfg/onigirazu/internal/tagfilter"
    "github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TaskExecutionInfo contains info about a task's execution status
type TaskExecutionInfo struct {
    Name          string
    Tags          []string
    Would Execute bool
    SkipReason    string
    Module        string
    Hosts         []string
    Conditions    string
}

// PlayExecutionInfo contains execution info for a play
type PlayExecutionInfo struct {
    Name     string
    Hosts    []string
    Tasks    []*TaskExecutionInfo
    Summary  PlaySummary
}

type PlaySummary struct {
    Total      int
    Executing  int
    Skipped    int
}

// ListTasksResult contains preview of task execution
type ListTasksResult struct {
    Plays   []*PlayExecutionInfo
    Summary ExecutionSummary
}

type ExecutionSummary struct {
    Total          int
    Executing      int
    Skipped        int
    SkipReasons    map[string]int
    AffectedHosts  int
}

// PreviewTasks returns which tasks would execute with given filters
func PreviewTasks(
    playbook *types.Playbook,
    tags string,
    skipTags string,
) (*ListTasksResult, error) {
    result := &ListTasksResult{
        Plays:   make([]*PlayExecutionInfo, 0),
        Summary: ExecutionSummary{
            SkipReasons: make(map[string]int),
        },
    }

    // Parse tag filters
    filter := tagfilter.NewFilter(tags, skipTags)

    for _, play := range playbook.Plays {
        playInfo := &PlayExecutionInfo{
            Name:  play.Name,
            Hosts: play.Hosts,
            Tasks: make([]*TaskExecutionInfo, 0),
        }

        for _, task := range play.Tasks {
            taskInfo := &TaskExecutionInfo{
                Name:   task.Name,
                Tags:   task.Tags,
                Module: task.Module,
                Hosts:  play.Hosts,
            }

            // Apply tag filtering
            shouldRun, reason := filter.ShouldRun(task.Tags)
            taskInfo.Would Execute = shouldRun
            taskInfo.SkipReason = reason

            if shouldRun {
                result.Summary.Executing++
                playInfo.Summary.Executing++
            } else {
                result.Summary.Skipped++
                playInfo.Summary.Skipped++
                result.Summary.SkipReasons[reason]++
            }

            result.Summary.Total++
            playInfo.Summary.Total++

            playInfo.Tasks = append(playInfo.Tasks, taskInfo)
        }

        result.Plays = append(result.Plays, playInfo)
    }

    return result, nil
}
```

### Step 4: Create Output Formatters

**File**: `internal/output/formatters.go` (NEW)

```go
package output

import (
    "encoding/csv"
    "encoding/json"
    "fmt"
    "io"
    "strings"

    "github.com/onigirazu-cfg/onigirazu/internal/tagdiscovery"
    "github.com/onigirazu-cfg/onigirazu/internal/taskpreview"
)

// FormatTagsText formats tag list as human-readable text
func FormatTagsText(result *tagdiscovery.ListTagsResult, w io.Writer) error {
    fmt.Fprintf(w, "Available tags:\n\n")

    // Regular tags
    for name, info := range result.Tags {
        fmt.Fprintf(w, "  %s (%d tasks)\n", name, info.Count)
        for _, detail := range info.Details {
            fmt.Fprintf(w, "    - %s\n", detail)
        }
    }

    // Special tags
    if len(result.Special) > 0 {
        fmt.Fprintf(w, "\n  Special tags:\n")
        for name, info := range result.Special {
            fmt.Fprintf(w, "    %s (%d tasks)\n", name, info.Count)
        }
    }

    // Summary
    fmt.Fprintf(w, "\nSummary:\n")
    fmt.Fprintf(w, "  Total tags: %d\n", result.Summary.TotalTags)
    fmt.Fprintf(w, "  Total tasks: %d\n", result.Summary.TotalTasks)
    fmt.Fprintf(w, "  Always tasks: %d\n", result.Summary.AlwaysTasks)
    fmt.Fprintf(w, "  Never tasks: %d\n", result.Summary.NeverTasks)

    return nil
}

// FormatTasksText formats task list as human-readable text
func FormatTasksText(result *taskpreview.ListTasksResult, w io.Writer) error {
    fmt.Fprintf(w, "Tasks:\n\n")

    for _, play := range result.Plays {
        fmt.Fprintf(w, "Play: %s (Hosts: %s)\n", play.Name,
            strings.Join(play.Hosts, ", "))

        for _, task := range play.Tasks {
            status := "✓"
            if !task.WouldExecute {
                status = "✗"
            }

            tagsStr := strings.Join(task.Tags, ", ")
            fmt.Fprintf(w, "  %s [%s] %s\n", status, tagsStr, task.Name)

            if !task.WouldExecute && task.SkipReason != "" {
                fmt.Fprintf(w, "      Reason: %s\n", task.SkipReason)
            }
        }
        fmt.Fprintf(w, "\n")
    }

    // Summary
    fmt.Fprintf(w, "Summary:\n")
    fmt.Fprintf(w, "  Total tasks: %d\n", result.Summary.Total)
    fmt.Fprintf(w, "  Would execute: %d\n", result.Summary.Executing)
    fmt.Fprintf(w, "  Skipped: %d\n", result.Summary.Skipped)

    return nil
}

// FormatTagsJSON formats tag list as JSON
func FormatTagsJSON(result *tagdiscovery.ListTagsResult, w io.Writer) error {
    encoder := json.NewEncoder(w)
    encoder.SetIndent("", "  ")
    return encoder.Encode(result)
}

// FormatTasksJSON formats task list as JSON
func FormatTasksJSON(result *taskpreview.ListTasksResult, w io.Writer) error {
    encoder := json.NewEncoder(w)
    encoder.SetIndent("", "  ")
    return encoder.Encode(result)
}

// FormatTagsCSV formats tag list as CSV
func FormatTagsCSV(result *tagdiscovery.ListTagsResult, w io.Writer) error {
    writer := csv.NewWriter(w)
    defer writer.Flush()

    writer.Write([]string{"Tag", "Count", "Details"})

    for name, info := range result.Tags {
        writer.Write([]string{
            name,
            fmt.Sprintf("%d", info.Count),
            strings.Join(info.Details, "; "),
        })
    }

    return nil
}

// FormatTasksCSV formats task list as CSV
func FormatTasksCSV(result *taskpreview.ListTasksResult, w io.Writer) error {
    writer := csv.NewWriter(w)
    defer writer.Flush()

    writer.Write([]string{"Play", "Task", "Tags", "Executes", "Reason"})

    for _, play := range result.Plays {
        for _, task := range play.Tasks {
            executes := "yes"
            if !task.WouldExecute {
                executes = "no"
            }

            writer.Write([]string{
                play.Name,
                task.Name,
                strings.Join(task.Tags, ";"),
                executes,
                task.SkipReason,
            })
        }
    }

    return nil
}
```

### Step 5: Modify apply.go RunE Function

**File**: `internal/cli/apply.go`

In the `RunE` function, add logic to handle the new flags:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    playbookPath := args[0]

    // ... existing setup code ...

    // Load playbook
    playbook, err := parser.ParsePlaybook(playbookPath)
    if err != nil {
        return fmt.Errorf("failed to parse playbook: %w", err)
    }

    // Handle --list-tags
    if listTags {
        result, err := tagdiscovery.DiscoverTags(playbook)
        if err != nil {
            return fmt.Errorf("failed to discover tags: %w", err)
        }

        format := outputFormat // from global flags
        if format == "" {
            format = "text"
        }

        switch format {
        case "json":
            return output.FormatTagsJSON(result, os.Stdout)
        case "csv":
            return output.FormatTagsCSV(result, os.Stdout)
        default:
            return output.FormatTagsText(result, os.Stdout)
        }
    }

    // Handle --list-tasks
    if listTasks {
        result, err := taskpreview.PreviewTasks(playbook, tags, skipTags)
        if err != nil {
            return fmt.Errorf("failed to preview tasks: %w", err)
        }

        format := outputFormat
        if format == "" {
            format = "text"
        }

        switch format {
        case "json":
            return output.FormatTasksJSON(result, os.Stdout)
        case "csv":
            return output.FormatTasksCSV(result, os.Stdout)
        default:
            return output.FormatTasksText(result, os.Stdout)
        }
    }

    // ... existing playbook execution logic ...
},
```

### Step 6: Handle Mutual Exclusivity

Ensure `--list-tags` and `--list-tasks` cannot be used with execution flags:

```go
// In the RunE function validation section
if (listTags || listTasks) && (check || dryRun || diff) {
    return fmt.Errorf(
        "--list-tags and --list-tasks cannot be used with "+
        "--check, --dry-run, or --diff flags",
    )
}

if listTags && listTasks {
    return fmt.Errorf(
        "--list-tags and --list-tasks cannot be used together",
    )
}
```

## Testing Strategy

### Unit Tests

1. **Test tag discovery**
   - Empty playbook
   - Playbook with no tags
   - Playbook with duplicate tags
   - Special tags (always, never)

2. **Test task preview**
   - Default behavior (all tasks)
   - With `--tags` filter
   - With `--skip-tags` filter
   - Combined filters
   - Case-insensitive matching

3. **Test output formatting**
   - Text format
   - JSON format
   - YAML format
   - CSV format

### Integration Tests

1. **Real playbooks**
   - Run with real playbook files
   - Verify output accuracy
   - Check performance with large playbooks

2. **CLI flag combinations**
   - `--list-tags` with various flags
   - `--list-tasks` with various flags
   - Error cases (incompatible flags)

### Example Test Cases

```go
func TestDiscoverTags(t *testing.T) {
    playbook := &types.Playbook{
        Plays: []types.Play{
            {
                Tasks: []types.Task{
                    {Name: "Task 1", Tags: []string{"setup", "packages"}},
                    {Name: "Task 2", Tags: []string{"setup", "security"}},
                    {Name: "Task 3", Tags: []string{"always"}},
                },
            },
        },
    }

    result, err := tagdiscovery.DiscoverTags(playbook)

    assert.NoError(t, err)
    assert.Equal(t, 3, result.Summary.TotalTags)
    assert.Equal(t, 3, result.Summary.TotalTasks)
    assert.Equal(t, 1, result.Summary.AlwaysTasks)
}

func TestPreviewTasks(t *testing.T) {
    playbook := &types.Playbook{
        Plays: []types.Play{
            {
                Tasks: []types.Task{
                    {Name: "Task 1", Tags: []string{"setup"}},
                    {Name: "Task 2", Tags: []string{"deployment"}},
                    {Name: "Task 3", Tags: []string{"always"}},
                },
            },
        },
    }

    result, err := taskpreview.PreviewTasks(playbook, "setup", "")

    assert.NoError(t, err)
    assert.Equal(t, 2, result.Summary.Executing) // setup + always
    assert.Equal(t, 1, result.Summary.Skipped)   // deployment
}
```

## Performance Considerations

1. **Memory Usage**: For large playbooks with many tasks/tags, consider:
   - Lazy loading of task details
   - Streaming output for large result sets
   - Caching tag discovery results

2. **Parsing Performance**: Optimize playbook parsing:
   - Use existing parser code
   - Avoid re-parsing if cached

3. **Output Formatting**: For JSON/CSV:
   - Stream output for large playbooks
   - Avoid building entire result in memory

## Future Enhancements

1. **Additional output formats**: Markdown, HTML, XML
2. **Tag statistics**: Most used tags, tag frequency analysis
3. **Tag dependencies**: Show relationships between tags
4. **Diff mode**: `--list-tasks --diff` to show expected changes
5. **Filtering**: `--list-tags --filter pattern` to find specific tags
6. **Import/export**: Export task list for documentation generation

## Related Files

- Tag filtering logic: `internal/tagfilter/`
- Parser: `internal/parser/`
- CLI: `internal/cli/`
- Output formatting: `internal/output/`
- Types: `pkg/types/`

## See Also

- [Tag Filtering Guide](TAG_FILTERING.md)
- [Tag and Task Discovery Guide](LIST_TAGS_TASKS_GUIDE.md)
- [CLI Reference](../README.md#cli-commands)
