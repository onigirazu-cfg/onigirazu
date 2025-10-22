package progress

import (
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/utils"
)

// ProgressRenderer provides enhanced progress rendering
type ProgressRenderer struct {
	noColor     bool
	width       int
	showDetails bool
}

// NewProgressRenderer creates a new progress renderer
func NewProgressRenderer(noColor bool) *ProgressRenderer {
	return &ProgressRenderer{
		noColor:     noColor,
		width:       50,
		showDetails: true,
	}
}

// RenderBatchProgress renders progress for a batch of hosts
func (pr *ProgressRenderer) RenderBatchProgress(
	completed int,
	total int,
	currentTasks []HostTaskInfo,
	duration time.Duration,
) string {
	var sb strings.Builder

	// Progress bar
	percentage := float64(completed) / float64(total) * 100
	filled := int(float64(pr.width) * percentage / 100)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", pr.width-filled)

	if pr.noColor {
		sb.WriteString(fmt.Sprintf("[%s] %d/%d (%.1f%%) | %v elapsed\n",
			bar, completed, total, percentage, duration.Round(time.Second)))
	} else {
		sb.WriteString(fmt.Sprintf("[%s%s%s] %d/%d (%.1f%%) | %v elapsed\n",
			utils.Colors.Success("["), bar, utils.Colors.Success("]"),
			completed, total, percentage, duration.Round(time.Second)))
	}

	// Current tasks
	if pr.showDetails && len(currentTasks) > 0 {
		sb.WriteString(pr.renderCurrentTasks(currentTasks))
	}

	return sb.String()
}

// HostTaskInfo represents info about a currently running task
type HostTaskInfo struct {
	Host      string
	Task      string
	StartTime time.Time
	Status    string // running, waiting, etc.
}

// renderCurrentTasks renders currently running tasks
func (pr *ProgressRenderer) renderCurrentTasks(tasks []HostTaskInfo) string {
	var sb strings.Builder

	sb.WriteString("\n⏳ Currently Running:\n")

	for _, t := range tasks {
		elapsed := time.Since(t.StartTime).Round(time.Millisecond)

		host := t.Host
		if len(host) > 20 {
			host = host[:17] + "..."
		}

		if pr.noColor {
			sb.WriteString(fmt.Sprintf("   • %-20s: %s [%v]\n", host, t.Task, elapsed))
		} else {
			sb.WriteString(fmt.Sprintf("   • %s: %s %s\n",
				utils.Colors.HostName(fmt.Sprintf("%-20s", host)),
				utils.Colors.TaskName(t.Task),
				utils.Colors.Dim(fmt.Sprintf("[%v]", elapsed))))
		}
	}

	return sb.String()
}

// RenderSummaryLine renders final summary line
func (pr *ProgressRenderer) RenderSummaryLine(
	total int,
	success int,
	failed int,
	changed int,
	duration time.Duration,
) string {
	if pr.noColor {
		return fmt.Sprintf("\nSummary: %d total | %d success | %d failed | %d changed | %v\n",
			total, success, failed, changed, duration.Round(time.Millisecond))
	}

	return fmt.Sprintf("\nSummary: %d total | %s | %s | %s | %v\n",
		total,
		utils.Colors.Success(fmt.Sprintf("%d success", success)),
		utils.Colors.Error(fmt.Sprintf("%d failed", failed)),
		utils.Colors.Changed(fmt.Sprintf("%d changed", changed)),
		duration.Round(time.Millisecond))
}
