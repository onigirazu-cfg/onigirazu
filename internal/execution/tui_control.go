package execution

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/onigirazu-cfg/onigirazu/internal/diffview"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// RunController pauses and stops the run between tasks
type RunController interface {
	Pause()
	Resume()
	Stop()
	Paused() bool
}

// SetRunController lets P pause the run and G / Q stop it
func (m *EnhancedTUIModel) SetRunController(c RunController) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.controller = c
}

// running tells whether the run has not ended yet
func (m *EnhancedTUIModel) running() bool {
	switch m.status {
	case "completed", "failed", "stopped":
		return false
	}
	return true
}

// togglePause pauses or resumes the run (tasks already running finish)
func (m *EnhancedTUIModel) togglePause() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if !m.running() {
		return
	}
	if m.controller == nil {
		m.addLog(LogEntry{Level: "WARN", Message: "This run cannot be paused"})
		return
	}
	if m.controller.Paused() {
		m.controller.Resume()
		m.paused = false
		m.addLog(LogEntry{Level: "INFO", Message: "Run resumed"})
	} else {
		m.controller.Pause()
		m.paused = true
		m.addLog(LogEntry{Level: "WARN", Message: "Run paused: running tasks finish, no new task starts (P resumes)"})
	}
}

// askConfirm opens the confirm modal for "stop" or "quit"
func (m *EnhancedTUIModel) askConfirm(action string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.activeModal = "confirm"
	m.confirmAction = action
}

// confirm carries out the confirmed action
func (m *EnhancedTUIModel) confirm() tea.Cmd {
	m.mutex.Lock()
	action := m.confirmAction
	m.activeModal = ""
	m.confirmAction = ""
	if m.running() {
		m.gracefulReq = true
		m.status = "stopping"
		m.paused = false
		if m.controller != nil {
			m.controller.Stop()
		}
		m.addLog(LogEntry{Level: "WARN", Message: "Stopping: running tasks finish, no new task starts"})
	}
	callback := m.stopCallback
	m.mutex.Unlock()
	if callback != nil {
		go func() { _ = callback() }()
	}
	if action == "quit" {
		m.markClosed()
		return tea.Quit
	}
	return nil
}

// recordResult keeps a finished task for the results browser
func (m *EnhancedTUIModel) recordResult(r types.TaskResult) {
	m.results = append(m.results, r)
	if len(m.results) > m.maxLogs {
		m.results = append([]types.TaskResult(nil), m.results[len(m.results)-m.maxLogs:]...)
	}
}

// shownResults are the results the browser lists (failed only with F)
func (m *EnhancedTUIModel) shownResults() []types.TaskResult {
	if !m.failedOnly {
		return m.results
	}
	var out []types.TaskResult
	for _, r := range m.results {
		if r.Failed {
			out = append(out, r)
		}
	}
	return out
}

func resultStatus(r types.TaskResult) string {
	switch {
	case r.Failed && r.Ignored:
		return "ignored"
	case r.Failed:
		return "failed"
	case r.Skipped:
		return "skipped"
	case r.Changed:
		return "changed"
	}
	return "ok"
}

func statusColor(status string) lipgloss.Color {
	switch status {
	case "failed":
		return lipgloss.Color("196")
	case "changed":
		return lipgloss.Color("226")
	case "skipped", "ignored":
		return lipgloss.Color("243")
	}
	return lipgloss.Color("46")
}

// handleResultsKeypress moves in the results browser and opens a result
func (m *EnhancedTUIModel) handleResultsKeypress(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	n := len(m.shownResults())
	switch key {
	case "up", "k":
		m.resultsCursor--
	case "down", "j":
		m.resultsCursor++
	case "pgup":
		m.resultsCursor -= 10
	case "pgdown":
		m.resultsCursor += 10
	case "home":
		m.resultsCursor = 0
	case "end":
		m.resultsCursor = n - 1
	case "f":
		m.failedOnly = !m.failedOnly
		m.resultsCursor = len(m.shownResults()) - 1
	case "enter":
		if n > 0 {
			m.activeModal = "detail"
			m.detailOffset = 0
		}
	case "esc", "q", "r":
		m.activeModal = ""
	}
	n = len(m.shownResults())
	if m.resultsCursor >= n {
		m.resultsCursor = n - 1
	}
	if m.resultsCursor < 0 {
		m.resultsCursor = 0
	}
}

func (m *EnhancedTUIModel) modalStyle(width int, color string) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(color)).
		Padding(0, 1).
		Width(width)
}

// renderResultsModal lists the finished tasks, newest last
func (m *EnhancedTUIModel) renderResultsModal() string {
	width := m.width - 6
	rows := m.height - 8
	if rows < 3 {
		rows = 3
	}
	shown := m.shownResults()
	title := fmt.Sprintf("TASK RESULTS (%d)", len(shown))
	if m.failedOnly {
		title += " - failed only"
	}
	lines := []string{lipgloss.NewStyle().Bold(true).Render(title), ""}
	if len(shown) == 0 {
		lines = append(lines, "  no results yet")
	}
	start := m.resultsCursor - rows/2
	if start > len(shown)-rows {
		start = len(shown) - rows
	}
	if start < 0 {
		start = 0
	}
	for i := start; i < len(shown) && i < start+rows; i++ {
		r := shown[i]
		status := resultStatus(r)
		line := fmt.Sprintf("%-8s %-20s %s", status, truncateString(r.Host, 20), r.TaskName)
		line = truncateString(line, width-4)
		style := lipgloss.NewStyle().Foreground(statusColor(status))
		if i == m.resultsCursor {
			style = style.Reverse(true)
		}
		lines = append(lines, style.Render(line))
	}
	lines = append(lines, "", "↑↓ select · Enter details · F failed only · Esc back")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		m.modalStyle(width, "51").Render(strings.Join(lines, "\n")))
}

// detailLines is everything about one task result, diff included
func detailLines(r types.TaskResult, width int) []string {
	status := resultStatus(r)
	lines := []string{
		lipgloss.NewStyle().Bold(true).Render(r.TaskName),
		"",
		"Host:     " + r.Host,
		"Module:   " + r.Module,
		"Status:   " + lipgloss.NewStyle().Foreground(statusColor(status)).Render(status),
		"Duration: " + r.Duration.Round(time.Millisecond).String(),
	}
	if r.Error != "" {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Error:"))
		lines = append(lines, wrapLines(r.Error, width-4)...)
	}
	keys := make([]string, 0, len(r.Output))
	for k := range r.Output {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := r.Output[k]
		if v == nil || fmt.Sprint(v) == "" || k == "diff" {
			continue
		}
		text := fmt.Sprint(v)
		if strings.Contains(text, "\n") {
			lines = append(lines, "", k+":")
			lines = append(lines, wrapLines(text, width-6)...)
		} else {
			lines = append(lines, truncateString(fmt.Sprintf("%s: %s", k, text), width-4))
		}
	}
	if diff := diffview.TaskDiff(r); diff != "" {
		lines = append(lines, "", lipgloss.NewStyle().Bold(true).Render("Diff:"))
		for _, line := range strings.Split(strings.TrimRight(diff, "\n"), "\n") {
			style := lipgloss.NewStyle()
			switch {
			case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
				style = style.Bold(true)
			case strings.HasPrefix(line, "+"):
				style = style.Foreground(lipgloss.Color("46"))
			case strings.HasPrefix(line, "-"):
				style = style.Foreground(lipgloss.Color("196"))
			case strings.HasPrefix(line, "@@"):
				style = style.Foreground(lipgloss.Color("51"))
			}
			lines = append(lines, style.Render(truncateString(line, width-4)))
		}
	}
	return lines
}

// renderDetailModal shows one task result, scrolled by detailOffset
func (m *EnhancedTUIModel) renderDetailModal() string {
	width := m.width - 6
	shown := m.shownResults()
	if m.resultsCursor >= len(shown) {
		return m.renderResultsModal()
	}
	r := shown[m.resultsCursor]
	all := detailLines(r, width)
	rows := m.detailRows()
	offset := m.detailOffset
	if offset > len(all)-rows {
		offset = len(all) - rows
	}
	if offset < 0 {
		offset = 0
	}
	end := offset + rows
	if end > len(all) {
		end = len(all)
	}
	lines := append([]string{}, all[offset:end]...)
	footer := "Esc back"
	if len(all) > rows {
		footer = fmt.Sprintf("lines %d-%d of %d · ↑↓ PgUp PgDn Home End scroll · Esc back", offset+1, end, len(all))
	}
	lines = append(lines, "", footer)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		m.modalStyle(width, string(statusColor(resultStatus(r)))).Render(strings.Join(lines, "\n")))
}

// detailRows is the number of content lines the detail view shows
func (m *EnhancedTUIModel) detailRows() int {
	rows := m.height - 6
	if rows < 3 {
		rows = 3
	}
	return rows
}

// handleDetailKeypress scrolls the detail view or goes back to the list
func (m *EnhancedTUIModel) handleDetailKeypress(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	switch key {
	case "up", "k":
		m.detailOffset--
	case "down", "j":
		m.detailOffset++
	case "pgup":
		m.detailOffset -= m.detailRows()
	case "pgdown":
		m.detailOffset += m.detailRows()
	case "home":
		m.detailOffset = 0
	case "end":
		m.detailOffset = 1 << 30
	case "esc", "q", "backspace", "enter":
		m.activeModal = "results"
		return
	}
	shown := m.shownResults()
	if m.resultsCursor < len(shown) {
		if max := len(detailLines(shown[m.resultsCursor], m.width-6)) - m.detailRows(); m.detailOffset > max {
			m.detailOffset = max
		}
	}
	if m.detailOffset < 0 {
		m.detailOffset = 0
	}
}

// wrapLines splits text into lines of at most width characters
func wrapLines(text string, width int) []string {
	if width < 10 {
		width = 10
	}
	var out []string
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		for len(line) > width {
			out = append(out, "  "+line[:width])
			line = line[width:]
		}
		out = append(out, "  "+line)
	}
	return out
}

// resultLines are the log lines of a finished task: one status line, and in
// VERBOSE its message and first output lines; an error always shows
func resultLines(r *types.TaskResult) []LogEntry {
	status := resultStatus(*r)
	mark := map[string]string{"ok": "✓", "changed": "⟳", "failed": "✗", "skipped": "⊘", "ignored": "✗"}[status]
	level := "TASK_END"
	if status == "failed" {
		level = "ERROR"
	}
	entries := []LogEntry{{Level: level, Message: fmt.Sprintf("%s %s: %s [%s]", mark, r.Host, r.TaskName, status)}}
	if r.Error != "" && status == "failed" {
		entries = append(entries, LogEntry{Level: "ERROR", Message: "    " + firstLine(r.Error)})
	}
	if msg, ok := r.Output["msg"].(string); ok && msg != "" {
		entries = append(entries, LogEntry{Level: "TASK_OUT", Tier: 1, Message: "    " + firstLine(msg)})
	}
	for _, key := range []string{"stdout", "stderr"} {
		text, _ := r.Output[key].(string)
		lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
		for i, line := range lines {
			if line == "" {
				continue
			}
			if i == 5 {
				entries = append(entries, LogEntry{Level: "TASK_OUT", Tier: 1, Message: fmt.Sprintf("    … %d more %s lines (R for details)", len(lines)-5, key)})
				break
			}
			entries = append(entries, LogEntry{Level: "TASK_OUT", Tier: 1, Message: "    " + key + ": " + line})
		}
	}
	return entries
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
