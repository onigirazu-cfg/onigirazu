package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

// LogMode represents the output verbosity level
type LogMode int

const (
	LogModeNormal LogMode = iota
	LogModeVerbose
	LogModeDebug
)

// DisplayFormatter formats output for different verbosity modes
type DisplayFormatter interface {
	FormatInitialization(config InitConfig) string
	FormatInventoryLoaded(inventory InventoryInfo) string
	FormatPlaybookLoaded(playbook PlaybookInfo) string
	FormatExecutionStart() string
	FormatPlayStart(playName string, playIndex int, hostCount int) string
	FormatTaskStart(taskName, host string) string
	FormatTaskEnd(task TaskResult) string
	FormatExecutionEnd(summary ExecutionSummary) string
	FormatError(err string, context map[string]interface{}) string
}

// InitConfig holds initialization phase data
type InitConfig struct {
	StateBackend   string
	MaxConcurrency int
	SSHStrictMode  bool
	ConfigPath     string
	LogLevel       string
	ColorOutput    bool
	Plugins        []PluginInfo
	AuditEnabled   bool
	ExecutionID    string
}

// PluginInfo holds plugin metadata
type PluginInfo struct {
	Name    string
	Version string
}

// InventoryInfo holds inventory data
type InventoryInfo struct {
	Path       string
	GroupCount int
	HostCount  int
	Details    map[string]interface{} // For verbose/debug modes
}

// PlaybookInfo holds playbook data
type PlaybookInfo struct {
	Path      string
	PlayCount int
	Name      string
	Details   map[string]interface{} // For verbose/debug modes
}

// TaskResult holds task execution result
type TaskResult struct {
	Name     string
	Host     string
	Status   string // OK, CHANGED, FAILED, SKIPPED
	Duration time.Duration
	Output   map[string]interface{}
	Error    string
	Changed  bool
	Details  map[string]interface{} // For verbose/debug modes
}

// HostResultSummary holds per-host execution summary
type HostResultSummary struct {
	Hostname     string
	TasksOK      int
	TasksChanged int
	TasksFailed  int
	TasksSkipped int
	Duration     time.Duration
	Status       string // OK, CHANGED, FAILED
}

// ExecutionSummary holds execution summary data
type ExecutionSummary struct {
	TotalDuration time.Duration
	PlayCount     int
	TaskCount     int
	SuccessCount  int
	FailedCount   int
	ChangedCount  int
	SkippedCount  int
	Stats         map[string]interface{}
	HostResults   map[string]HostResultSummary // Per-host breakdown
}

// NormalFormatter - Clean, user-friendly output
type NormalFormatter struct {
	useColors bool
	output    io.Writer
}

// NewNormalFormatter creates a formatter for normal mode
func NewNormalFormatter(useColors bool, output io.Writer) *NormalFormatter {
	return &NormalFormatter{useColors: useColors, output: output}
}

func (f *NormalFormatter) FormatInitialization(config InitConfig) string {
	var sb strings.Builder

	if f.useColors {
		sb.WriteString(utils.Colors.Header("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
		sb.WriteString(utils.Colors.Header("📋 INITIALIZATION\n"))
		sb.WriteString(utils.Colors.Header("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
	} else {
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("📋 INITIALIZATION\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	}

	sb.WriteString(fmt.Sprintf("✓ State backend: %s\n", config.StateBackend))
	sb.WriteString(fmt.Sprintf("✓ Max concurrency: %d\n", config.MaxConcurrency))
	if config.SSHStrictMode {
		sb.WriteString("✓ SSH strict mode: enabled\n")
	} else {
		sb.WriteString("✓ SSH strict mode: disabled\n")
	}

	if config.AuditEnabled && config.ExecutionID != "" {
		sb.WriteString(fmt.Sprintf("✓ Audit recording: enabled (ID: %s)\n", config.ExecutionID))
	}

	if len(config.Plugins) > 0 {
		sb.WriteString(fmt.Sprintf("✓ Plugins: %d loaded\n", len(config.Plugins)))
	}

	sb.WriteString("\n")
	return sb.String()
}

func (f *NormalFormatter) FormatInventoryLoaded(inventory InventoryInfo) string {
	if f.useColors {
		return fmt.Sprintf("✓ Inventory: %s (%d groups, %d hosts)\n\n",
			inventory.Path, inventory.GroupCount, inventory.HostCount)
	}
	return fmt.Sprintf("✓ Inventory: %s (%d groups, %d hosts)\n\n",
		inventory.Path, inventory.GroupCount, inventory.HostCount)
}

func (f *NormalFormatter) FormatPlaybookLoaded(playbook PlaybookInfo) string {
	if f.useColors {
		return fmt.Sprintf("✓ Playbook: %s (%d plays)\n\n",
			playbook.Path, playbook.PlayCount)
	}
	return fmt.Sprintf("✓ Playbook: %s (%d plays)\n\n",
		playbook.Path, playbook.PlayCount)
}

func (f *NormalFormatter) FormatExecutionStart() string {
	var sb strings.Builder
	if f.useColors {
		sb.WriteString(utils.Colors.Header("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
		sb.WriteString(utils.Colors.Header("🚀 EXECUTION START\n"))
		sb.WriteString(utils.Colors.Header("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
	} else {
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("🚀 EXECUTION START\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

func (f *NormalFormatter) FormatPlayStart(playName string, playIndex int, hostCount int) string {
	emoji := "📌"
	if f.useColors {
		return fmt.Sprintf("\n%s Play %d: %s (%d hosts)\n", emoji, playIndex+1, playName, hostCount)
	}
	return fmt.Sprintf("\nPlay %d: %s (%d hosts)\n", playIndex+1, playName, hostCount)
}

func (f *NormalFormatter) FormatTaskStart(taskName, host string) string {
	// In normal mode, we don't output task start (too verbose)
	return ""
}

func (f *NormalFormatter) FormatTaskEnd(task TaskResult) string {
	var icon string
	switch task.Status {
	case "OK":
		icon = "✓"
	case "CHANGED":
		icon = "⚡"
	case "FAILED":
		icon = "✗"
	case "SKIPPED":
		icon = "⊘"
	default:
		icon = "?"
	}

	return fmt.Sprintf("  %s %-40s %-20s\n", icon, task.Name, task.Host)
}

func (f *NormalFormatter) FormatExecutionEnd(summary ExecutionSummary) string {
	var sb strings.Builder

	if f.useColors {
		sb.WriteString("\n" + utils.Colors.Header("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
		sb.WriteString(utils.Colors.Header("✅ EXECUTION COMPLETE\n"))
		sb.WriteString(utils.Colors.Header("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
	} else {
		sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("EXECUTION COMPLETE\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	}

	sb.WriteString(fmt.Sprintf("⏱️  Total Duration: %v\n", summary.TotalDuration))
	sb.WriteString(fmt.Sprintf("📋 Plays: %d\n", summary.PlayCount))

	sb.WriteString(fmt.Sprintf("📦 Tasks: %d total", summary.TaskCount))
	if summary.SuccessCount > 0 {
		sb.WriteString(fmt.Sprintf(" | ✓ %d ok", summary.SuccessCount))
	}
	if summary.ChangedCount > 0 {
		sb.WriteString(fmt.Sprintf(" | ⚡ %d changed", summary.ChangedCount))
	}
	if summary.FailedCount > 0 {
		sb.WriteString(fmt.Sprintf(" | ✗ %d failed", summary.FailedCount))
	}
	if summary.SkippedCount > 0 {
		sb.WriteString(fmt.Sprintf(" | ⊘ %d skipped", summary.SkippedCount))
	}
	sb.WriteString("\n")

	// Per-host breakdown
	if len(summary.HostResults) > 0 {
		sb.WriteString("\nPer-Host Results:\n")

		// Sort hosts for consistent output
		hosts := make([]string, 0, len(summary.HostResults))
		for hostname := range summary.HostResults {
			hosts = append(hosts, hostname)
		}
		sort.Strings(hosts)

		for _, hostname := range hosts {
			hostResult := summary.HostResults[hostname]
			status := "✅"
			if hostResult.Status == "FAILED" {
				status = "❌"
			} else if hostResult.Status == "CHANGED" {
				status = "🔄"
			}

			sb.WriteString(fmt.Sprintf("  %s %-25s OK:%d CHANGED:%d FAILED:%d SKIPPED:%d\n",
				status, hostname,
				hostResult.TasksOK,
				hostResult.TasksChanged,
				hostResult.TasksFailed,
				hostResult.TasksSkipped,
			))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (f *NormalFormatter) FormatError(err string, context map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString("\n")
	if f.useColors {
		sb.WriteString(utils.Colors.Error("❌ ERROR\n"))
		sb.WriteString(utils.Colors.Error("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
	} else {
		sb.WriteString("ERROR\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	}
	sb.WriteString(fmt.Sprintf("%s\n", err))
	sb.WriteString("\n")
	return sb.String()
}

// VerboseFormatter - Detailed output with more context
type VerboseFormatter struct {
	useColors bool
	output    io.Writer
}

// NewVerboseFormatter creates a formatter for verbose mode
func NewVerboseFormatter(useColors bool, output io.Writer) *VerboseFormatter {
	return &VerboseFormatter{useColors: useColors, output: output}
}

func (f *VerboseFormatter) FormatInitialization(config InitConfig) string {
	var sb strings.Builder

	if f.useColors {
		sb.WriteString(utils.Colors.Header("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
		sb.WriteString(utils.Colors.Header("📋 INITIALIZATION\n"))
		sb.WriteString(utils.Colors.Header("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
	} else {
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("INITIALIZATION\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	}

	sb.WriteString(fmt.Sprintf("✓ State backend: %s\n", config.StateBackend))
	sb.WriteString(fmt.Sprintf("✓ Max concurrency: %d\n", config.MaxConcurrency))
	sb.WriteString(fmt.Sprintf("✓ SSH strict mode: %v\n", config.SSHStrictMode))
	sb.WriteString(fmt.Sprintf("✓ Log level: %s\n", config.LogLevel))
	sb.WriteString(fmt.Sprintf("✓ Color output: %v\n", config.ColorOutput))

	if config.ConfigPath != "" {
		sb.WriteString(fmt.Sprintf("✓ Config file: %s\n", config.ConfigPath))
	}

	if config.AuditEnabled && config.ExecutionID != "" {
		sb.WriteString("✓ Audit recording: enabled\n")
		sb.WriteString(fmt.Sprintf("  └─ Execution ID: %s\n", config.ExecutionID))
	}

	if len(config.Plugins) > 0 {
		sb.WriteString(fmt.Sprintf("✓ Plugins (%d loaded):\n", len(config.Plugins)))
		for _, p := range config.Plugins {
			sb.WriteString(fmt.Sprintf("  ├─ %s (v%s)\n", p.Name, p.Version))
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

func (f *VerboseFormatter) FormatInventoryLoaded(inventory InventoryInfo) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✓ Inventory: %s\n", inventory.Path))
	sb.WriteString(fmt.Sprintf("  ├─ Groups: %d\n", inventory.GroupCount))
	sb.WriteString(fmt.Sprintf("  └─ Hosts: %d\n\n", inventory.HostCount))

	if len(inventory.Details) > 0 {
		if groups, ok := inventory.Details["groups"].(map[string]interface{}); ok {
			for groupName, hosts := range groups {
				if hostList, ok := hosts.([]interface{}); ok {
					sb.WriteString(fmt.Sprintf("  Group: %s (%d hosts)\n", groupName, len(hostList)))
					for _, host := range hostList {
						sb.WriteString(fmt.Sprintf("    - %v\n", host))
					}
				}
			}
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

func (f *VerboseFormatter) FormatPlaybookLoaded(playbook PlaybookInfo) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✓ Playbook: %s\n", playbook.Path))
	sb.WriteString(fmt.Sprintf("  ├─ Name: %s\n", playbook.Name))
	sb.WriteString(fmt.Sprintf("  └─ Plays: %d\n\n", playbook.PlayCount))

	if len(playbook.Details) > 0 {
		if plays, ok := playbook.Details["plays"].([]interface{}); ok {
			for i, play := range plays {
				if playMap, ok := play.(map[string]interface{}); ok {
					if name, ok := playMap["name"].(string); ok {
						sb.WriteString(fmt.Sprintf("  Play %d: %s\n", i+1, name))
					}
				}
			}
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

func (f *VerboseFormatter) FormatExecutionStart() string {
	var sb strings.Builder
	if f.useColors {
		sb.WriteString(utils.Colors.Header("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
		sb.WriteString(utils.Colors.Header("🚀 EXECUTION START\n"))
		sb.WriteString(utils.Colors.Header("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
	} else {
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("EXECUTION START\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	}
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	return sb.String()
}

func (f *VerboseFormatter) FormatPlayStart(playName string, playIndex int, hostCount int) string {
	return fmt.Sprintf("\n📌 Play %d: %s\n  Hosts: %d\n", playIndex+1, playName, hostCount)
}

func (f *VerboseFormatter) FormatTaskStart(taskName, host string) string {
	return fmt.Sprintf("  ➜ Starting: %s on %s\n", taskName, host)
}

func (f *VerboseFormatter) FormatTaskEnd(task TaskResult) string {
	var icon string
	switch task.Status {
	case "OK":
		icon = "✓"
	case "CHANGED":
		icon = "⚡"
	case "FAILED":
		icon = "✗"
	case "SKIPPED":
		icon = "⊘"
	default:
		icon = "?"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s %-40s on %-15s [%v]\n", icon, task.Name, task.Host, task.Duration))

	if len(task.Details) > 0 {
		if params, ok := task.Details["params"].(map[string]interface{}); ok && len(params) > 0 {
			sb.WriteString("    Parameters: ")
			for k, v := range params {
				sb.WriteString(fmt.Sprintf("%s=%v ", k, v))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (f *VerboseFormatter) FormatExecutionEnd(summary ExecutionSummary) string {
	var sb strings.Builder

	if f.useColors {
		sb.WriteString("\n" + utils.Colors.Header("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
		sb.WriteString(utils.Colors.Header("✅ EXECUTION COMPLETE\n"))
		sb.WriteString(utils.Colors.Header("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
	} else {
		sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("EXECUTION COMPLETE\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	}

	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Total Duration: %v\n\n", summary.TotalDuration))

	sb.WriteString("Summary:\n")
	sb.WriteString(fmt.Sprintf("  ├─ Plays: %d\n", summary.PlayCount))
	sb.WriteString(fmt.Sprintf("  ├─ Tasks: %d total\n", summary.TaskCount))
	sb.WriteString(fmt.Sprintf("  ├─ ✓ OK: %d\n", summary.SuccessCount))
	sb.WriteString(fmt.Sprintf("  ├─ ⚡ Changed: %d\n", summary.ChangedCount))
	sb.WriteString(fmt.Sprintf("  ├─ ✗ Failed: %d\n", summary.FailedCount))
	sb.WriteString(fmt.Sprintf("  └─ ⊘ Skipped: %d\n", summary.SkippedCount))

	// Per-host breakdown
	if len(summary.HostResults) > 0 {
		sb.WriteString("\nPer-Host Details:\n")

		// Sort hosts for consistent output
		hosts := make([]string, 0, len(summary.HostResults))
		for hostname := range summary.HostResults {
			hosts = append(hosts, hostname)
		}
		sort.Strings(hosts)

		for i, hostname := range hosts {
			hostResult := summary.HostResults[hostname]
			isLast := i == len(hosts)-1
			prefix := "  ├─"
			if isLast {
				prefix = "  └─"
			}

			status := "✅"
			if hostResult.Status == "FAILED" {
				status = "❌"
			} else if hostResult.Status == "CHANGED" {
				status = "🔄"
			}

			sb.WriteString(fmt.Sprintf("%s %s %-20s OK:%d CHANGED:%d FAILED:%d SKIPPED:%d\n",
				prefix, status, hostname,
				hostResult.TasksOK,
				hostResult.TasksChanged,
				hostResult.TasksFailed,
				hostResult.TasksSkipped,
			))
		}
	}

	sb.WriteString("\n")

	return sb.String()
}

func (f *VerboseFormatter) FormatError(err string, context map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString("\n")
	if f.useColors {
		sb.WriteString(utils.Colors.Error("❌ ERROR\n"))
		sb.WriteString(utils.Colors.Error("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"))
	} else {
		sb.WriteString("ERROR\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	}
	sb.WriteString(fmt.Sprintf("Message: %s\n", err))

	if len(context) > 0 {
		sb.WriteString("Context:\n")
		for k, v := range context {
			sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

// DebugFormatter - Full JSON output with all data
type DebugFormatter struct {
	output io.Writer
}

// NewDebugFormatter creates a formatter for debug mode
func NewDebugFormatter(output io.Writer) *DebugFormatter {
	return &DebugFormatter{output: output}
}

func (f *DebugFormatter) FormatInitialization(config InitConfig) string {
	data := map[string]interface{}{
		"phase":           "INITIALIZATION",
		"timestamp":       time.Now().Format(time.RFC3339Nano),
		"state_backend":   config.StateBackend,
		"max_concurrency": config.MaxConcurrency,
		"ssh_strict_mode": config.SSHStrictMode,
		"log_level":       config.LogLevel,
		"color_output":    config.ColorOutput,
		"config_path":     config.ConfigPath,
		"audit_enabled":   config.AuditEnabled,
		"execution_id":    config.ExecutionID,
		"plugins_count":   len(config.Plugins),
		"plugins":         config.Plugins,
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ""
	}
	return string(jsonBytes) + "\n\n"
}

func (f *DebugFormatter) FormatInventoryLoaded(inventory InventoryInfo) string {
	data := map[string]interface{}{
		"phase":       "LOAD_INVENTORY",
		"timestamp":   time.Now().Format(time.RFC3339Nano),
		"path":        inventory.Path,
		"group_count": inventory.GroupCount,
		"host_count":  inventory.HostCount,
		"details":     inventory.Details,
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ""
	}
	return string(jsonBytes) + "\n\n"
}

func (f *DebugFormatter) FormatPlaybookLoaded(playbook PlaybookInfo) string {
	data := map[string]interface{}{
		"phase":      "LOAD_PLAYBOOK",
		"timestamp":  time.Now().Format(time.RFC3339Nano),
		"path":       playbook.Path,
		"name":       playbook.Name,
		"play_count": playbook.PlayCount,
		"details":    playbook.Details,
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ""
	}
	return string(jsonBytes) + "\n\n"
}

func (f *DebugFormatter) FormatExecutionStart() string {
	data := map[string]interface{}{
		"phase":     "EXECUTION_START",
		"timestamp": time.Now().Format(time.RFC3339Nano),
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ""
	}
	return string(jsonBytes) + "\n\n"
}

func (f *DebugFormatter) FormatPlayStart(playName string, playIndex int, hostCount int) string {
	data := map[string]interface{}{
		"phase":      "PLAY_START",
		"timestamp":  time.Now().Format(time.RFC3339Nano),
		"play_index": playIndex,
		"play_name":  playName,
		"host_count": hostCount,
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ""
	}
	return string(jsonBytes) + "\n"
}

func (f *DebugFormatter) FormatTaskStart(taskName, host string) string {
	data := map[string]interface{}{
		"phase":     "TASK_START",
		"timestamp": time.Now().Format(time.RFC3339Nano),
		"task_name": taskName,
		"host":      host,
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ""
	}
	return string(jsonBytes) + "\n"
}

func (f *DebugFormatter) FormatTaskEnd(task TaskResult) string {
	data := map[string]interface{}{
		"phase":     "TASK_END",
		"timestamp": time.Now().Format(time.RFC3339Nano),
		"task_name": task.Name,
		"host":      task.Host,
		"status":    task.Status,
		"duration":  task.Duration.String(),
		"changed":   task.Changed,
		"error":     task.Error,
		"output":    task.Output,
		"details":   task.Details,
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ""
	}
	return string(jsonBytes) + "\n"
}

func (f *DebugFormatter) FormatExecutionEnd(summary ExecutionSummary) string {
	data := map[string]interface{}{
		"phase":          "EXECUTION_END",
		"timestamp":      time.Now().Format(time.RFC3339Nano),
		"total_duration": summary.TotalDuration.String(),
		"play_count":     summary.PlayCount,
		"task_count":     summary.TaskCount,
		"success_count":  summary.SuccessCount,
		"failed_count":   summary.FailedCount,
		"changed_count":  summary.ChangedCount,
		"skipped_count":  summary.SkippedCount,
		"stats":          summary.Stats,
		"host_results":   summary.HostResults,
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ""
	}
	return string(jsonBytes) + "\n\n"
}

func (f *DebugFormatter) FormatError(err string, context map[string]interface{}) string {
	data := map[string]interface{}{
		"phase":     "ERROR",
		"timestamp": time.Now().Format(time.RFC3339Nano),
		"message":   err,
		"context":   context,
	}

	jsonBytes, jsonErr := json.MarshalIndent(data, "", "  ")
	if jsonErr != nil {
		return ""
	}
	return string(jsonBytes) + "\n\n"
}
