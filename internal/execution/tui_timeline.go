package execution

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// statusMark is the one-character mark of a task status
func statusMark(status string) string {
	return map[string]string{"ok": "✓", "changed": "⟳", "failed": "✗", "skipped": "⊘", "ignored": "✗"}[status]
}

// timelineHosts are the hosts that have results, failed ones first
func (m *EnhancedTUIModel) timelineHosts() []string {
	failed := map[string]bool{}
	seen := map[string]bool{}
	var hosts []string
	for _, r := range m.results {
		if !seen[r.Host] {
			seen[r.Host] = true
			hosts = append(hosts, r.Host)
		}
		if r.Failed && !r.Ignored {
			failed[r.Host] = true
		}
	}
	sort.Slice(hosts, func(i, j int) bool {
		if failed[hosts[i]] != failed[hosts[j]] {
			return failed[hosts[i]]
		}
		return hosts[i] < hosts[j]
	})
	return hosts
}

func (m *EnhancedTUIModel) hostResults(host string) []types.TaskResult {
	var out []types.TaskResult
	for _, r := range m.results {
		if r.Host == host {
			out = append(out, r)
		}
	}
	return out
}

// handleTimelineKeypress moves between hosts and opens a host's tasks
func (m *EnhancedTUIModel) handleTimelineKeypress(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	n := len(m.timelineHosts())
	switch key {
	case "up", "k":
		m.timelineCursor--
	case "down", "j":
		m.timelineCursor++
	case "home":
		m.timelineCursor = 0
	case "end":
		m.timelineCursor = n - 1
	case "enter":
		if n > 0 {
			m.activeModal = "hosttimeline"
			m.detailOffset = 0
		}
	case "esc", "q", "l":
		m.activeModal = ""
	}
	if m.timelineCursor >= n {
		m.timelineCursor = n - 1
	}
	if m.timelineCursor < 0 {
		m.timelineCursor = 0
	}
}

// handleHostTimelineKeypress scrolls a host's tasks or goes back
func (m *EnhancedTUIModel) handleHostTimelineKeypress(key string) {
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
		m.activeModal = "timeline"
		return
	}
	if hosts := m.timelineHosts(); m.timelineCursor < len(hosts) {
		if max := len(m.hostResults(hosts[m.timelineCursor])) - (m.detailRows() - 2); m.detailOffset > max {
			m.detailOffset = max
		}
	}
	if m.detailOffset < 0 {
		m.detailOffset = 0
	}
}

// renderTimelineModal shows every host as a row of task marks in run order
func (m *EnhancedTUIModel) renderTimelineModal() string {
	width := m.width - 6
	hosts := m.timelineHosts()
	nameWidth := 12
	for _, h := range hosts {
		if len(h) > nameWidth {
			nameWidth = len(h)
		}
	}
	if nameWidth > 30 {
		nameWidth = 30
	}
	marksWidth := width - nameWidth - 20
	if marksWidth < 10 {
		marksWidth = 10
	}
	lines := []string{
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("TIMELINE (%d hosts)", len(hosts))),
		"✓ ok  ⟳ changed  ✗ failed  ⊘ skipped, oldest first",
		"",
	}
	if len(hosts) == 0 {
		lines = append(lines, "  no results yet")
	}
	rows := m.height - 10
	if rows < 3 {
		rows = 3
	}
	start := m.timelineCursor - rows/2
	if start > len(hosts)-rows {
		start = len(hosts) - rows
	}
	if start < 0 {
		start = 0
	}
	for i := start; i < len(hosts) && i < start+rows; i++ {
		results := m.hostResults(hosts[i])
		var marks []string
		var total time.Duration
		for _, r := range results {
			status := resultStatus(r)
			marks = append(marks, lipgloss.NewStyle().Foreground(statusColor(status)).Render(statusMark(status)))
			total += r.Duration
		}
		if len(marks) > marksWidth { // the newest ones
			marks = append([]string{"…"}, marks[len(marks)-marksWidth+1:]...)
		}
		name := fmt.Sprintf("%-*s", nameWidth, truncateString(hosts[i], nameWidth))
		if i == m.timelineCursor {
			name = lipgloss.NewStyle().Reverse(true).Render(name)
		}
		lines = append(lines, fmt.Sprintf("%s %s  %d tasks %s", name, strings.Join(marks, ""), len(results),
			total.Round(time.Second)))
	}
	lines = append(lines, "", "↑↓ host · Enter its tasks over time · Esc back")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		m.modalStyle(width, "51").Render(strings.Join(lines, "\n")))
}

// renderHostTimelineModal shows one host's tasks as bars on the run's time
// axis: where a task started and how long it took
func (m *EnhancedTUIModel) renderHostTimelineModal() string {
	width := m.width - 6
	hosts := m.timelineHosts()
	if m.timelineCursor >= len(hosts) {
		return m.renderTimelineModal()
	}
	host := hosts[m.timelineCursor]
	results := m.hostResults(host)

	runStart, runEnd := m.startTime, time.Now()
	for _, r := range m.results {
		if r.Timestamp.Before(runStart) {
			runStart = r.Timestamp
		}
		if end := r.Timestamp.Add(r.Duration); end.After(runEnd) {
			runEnd = end
		}
	}
	if !m.running() && len(m.results) > 0 {
		runEnd = runStart
		for _, r := range m.results {
			if end := r.Timestamp.Add(r.Duration); end.After(runEnd) {
				runEnd = end
			}
		}
	}
	span := runEnd.Sub(runStart)
	if span <= 0 {
		span = time.Second
	}
	barWidth := width - 50
	if barWidth < 10 {
		barWidth = 10
	}

	var rowsText []string
	for _, r := range results {
		status := resultStatus(r)
		from := int(float64(r.Timestamp.Sub(runStart)) / float64(span) * float64(barWidth))
		length := int(float64(r.Duration) / float64(span) * float64(barWidth))
		if length < 1 {
			length = 1
		}
		if from < 0 {
			from = 0
		}
		if from >= barWidth {
			from = barWidth - 1
		}
		if from+length > barWidth {
			length = barWidth - from
		}
		bar := strings.Repeat("·", from) +
			lipgloss.NewStyle().Foreground(statusColor(status)).Render(strings.Repeat("█", length)) +
			strings.Repeat("·", barWidth-from-length)
		rowsText = append(rowsText, fmt.Sprintf("+%-6s %s %7s %s %s",
			r.Timestamp.Sub(runStart).Round(time.Second), bar, r.Duration.Round(10*time.Millisecond),
			statusMark(status), truncateString(r.TaskName, 30)))
	}

	rows := m.detailRows() - 2
	offset := m.detailOffset
	if offset > len(rowsText)-rows {
		offset = len(rowsText) - rows
	}
	if offset < 0 {
		offset = 0
	}
	end := offset + rows
	if end > len(rowsText) {
		end = len(rowsText)
	}
	lines := []string{
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%s: %d tasks over %s", host, len(results), span.Round(time.Second))),
		"",
	}
	lines = append(lines, rowsText[offset:end]...)
	footer := "Esc back"
	if len(rowsText) > rows {
		footer = fmt.Sprintf("tasks %d-%d of %d · ↑↓ PgUp PgDn scroll · Esc back", offset+1, end, len(rowsText))
	}
	lines = append(lines, "", footer)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		m.modalStyle(width, "51").Render(strings.Join(lines, "\n")))
}
