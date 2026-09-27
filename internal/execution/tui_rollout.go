package execution

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// OnRolloutBatch is told how a batch of a checked rollout goes
// (engine.RolloutObserver)
func (m *EnhancedTUIModel) OnRolloutBatch(phase string, report types.BatchReport) {
	r := report
	m.send(ExecutionEvent{Type: "rollout", Phase: phase, Batch: &r, Timestamp: time.Now()})
}

// batchState is how a batch of the rollout shows: a mark and a word
func batchState(b types.BatchReport, phase string) (string, string) {
	switch {
	case b.RolledBack:
		return "↺", "rolled back"
	case phase == "rolling_back":
		return "↺", "rolling back"
	case phase == "canary_pause":
		return "⏳", "canary soak"
	case phase == "checking":
		return "◎", "checking health"
	case phase == "start":
		return "◌", "running"
	case !b.Healthy:
		return "✗", "unhealthy"
	}
	return "✓", "healthy"
}

func batchColor(mark string) lipgloss.Color {
	switch mark {
	case "✓":
		return lipgloss.Color("46")
	case "✗":
		return lipgloss.Color("196")
	case "↺":
		return lipgloss.Color("208")
	}
	return lipgloss.Color("226")
}

// processRollout records a rollout event and logs it
func (m *EnhancedTUIModel) processRollout(phase string, b *types.BatchReport) {
	if b == nil {
		return
	}
	if phase == "checking" { // the batch that runs now
		if n := len(m.rollout); n > 0 {
			m.rolloutPhases[n-1] = "checking"
		}
		return
	}
	idx := -1
	for i := range m.rollout {
		if m.rollout[i].Play == b.Play && m.rollout[i].Batch == b.Batch {
			idx = i
		}
	}
	if idx < 0 {
		m.rollout = append(m.rollout, *b)
		m.rolloutPhases = append(m.rolloutPhases, phase)
		idx = len(m.rollout) - 1
	} else {
		m.rollout[idx] = *b
		m.rolloutPhases[idx] = phase
	}

	name := fmt.Sprintf("batch %d/%d", b.Batch, b.Batches)
	if b.Canary {
		name = "canary " + name
	}
	hosts := strings.Join(b.Hosts, ", ")
	switch phase {
	case "start":
		m.addLog(LogEntry{Level: "INFO", Message: fmt.Sprintf("▷ %s: %s", name, hosts)})
	case "canary_pause":
		m.addLog(LogEntry{Level: "INFO", Message: fmt.Sprintf("⏳ %s healthy; soak, then check again", name)})
	case "healthy":
		m.addLog(LogEntry{Level: "INFO", Message: fmt.Sprintf("✓ %s healthy", name)})
	case "unhealthy":
		m.addLog(LogEntry{Level: "ERROR", Message: fmt.Sprintf("✗ %s unhealthy (%s): %s", name, b.Reason, strings.Join(b.UnhealthyHosts, ", "))})
	case "rolling_back":
		m.status = "rolling back"
		m.addLog(LogEntry{Level: "WARN", Message: fmt.Sprintf("↺ rolling back %s", name)})
	case "rolled_back":
		m.status = "running"
		msg := fmt.Sprintf("↺ %s rolled back: %d change(s) undone", name, b.Undone)
		if b.HealthyAfterRollback != nil {
			if *b.HealthyAfterRollback {
				msg += ", healthy again"
			} else {
				msg += ", STILL UNHEALTHY"
			}
		}
		m.addLog(LogEntry{Level: "WARN", Message: msg})
		for _, f := range b.RollbackErrors {
			m.addLog(LogEntry{Level: "ERROR", Message: "    not undone: " + f})
		}
	case "stopped":
		m.addLog(LogEntry{Level: "ERROR", Message: fmt.Sprintf("■ %s unhealthy: rollout stopped", name)})
	}
}

// rolloutLines are the stats panel's rollout lines: the batches of the
// current play as marks
func (m *EnhancedTUIModel) rolloutLines(width int) []string {
	if len(m.rollout) == 0 {
		return nil
	}
	last := m.rollout[len(m.rollout)-1]
	var marks []string
	for i, b := range m.rollout {
		if b.Play != last.Play {
			continue
		}
		mark, _ := batchState(b, m.rolloutPhases[i])
		marks = append(marks, lipgloss.NewStyle().Foreground(batchColor(mark)).Render(mark))
	}
	_, state := batchState(last, m.rolloutPhases[len(m.rollout)-1])
	lines := []string{"", fmt.Sprintf("Rollout: %d/%d", last.Batch, last.Batches)}
	if len(marks) > width-4 {
		marks = marks[len(marks)-(width-4):]
	}
	lines = append(lines, "  "+strings.Join(marks, ""), "  "+truncateString(state, width-2))
	return lines
}

// renderRolloutModal lists every batch with its outcome
func (m *EnhancedTUIModel) renderRolloutModal() string {
	width := m.width - 6
	var all []string
	if len(m.rollout) == 0 {
		all = append(all, "  no checked rollout in this run (health_check, --canary or --auto-rollback)")
	}
	play := ""
	for i, b := range m.rollout {
		if b.Play != play {
			play = b.Play
			all = append(all, "", lipgloss.NewStyle().Bold(true).Render("Play: "+play))
		}
		mark, state := batchState(b, m.rolloutPhases[i])
		name := fmt.Sprintf("batch %d/%d", b.Batch, b.Batches)
		if b.Canary {
			name = "canary " + name
		}
		style := lipgloss.NewStyle().Foreground(batchColor(mark))
		all = append(all, style.Render(fmt.Sprintf("%s %s: %s", mark, name, state)))
		all = append(all, "    hosts: "+truncateString(strings.Join(b.Hosts, ", "), width-12))
		if b.Reason != "" {
			all = append(all, "    reason: "+truncateString(b.Reason, width-13))
		}
		if len(b.UnhealthyHosts) > 0 {
			all = append(all, "    unhealthy: "+truncateString(strings.Join(b.UnhealthyHosts, ", "), width-16))
		}
		if b.RolledBack {
			line := fmt.Sprintf("    undone: %d change(s)", b.Undone)
			if b.HealthyAfterRollback != nil {
				line += map[bool]string{true: ", healthy again", false: ", STILL UNHEALTHY"}[*b.HealthyAfterRollback]
			}
			all = append(all, line)
			for _, k := range b.Irreversible {
				all = append(all, "    kept: "+truncateString(k, width-11))
			}
			for _, f := range b.RollbackErrors {
				all = append(all, lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("    NOT undone: "+truncateString(f, width-17)))
			}
		}
	}
	rows := m.detailRows() - 2
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
	lines := []string{lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("ROLLOUT (%d batches)", len(m.rollout)))}
	lines = append(lines, all[offset:end]...)
	lines = append(lines, "", "↑↓ PgUp PgDn scroll · Esc back")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		m.modalStyle(width, "208").Render(strings.Join(lines, "\n")))
}

// handleRolloutKeypress scrolls the rollout view or closes it
func (m *EnhancedTUIModel) handleRolloutKeypress(key string) {
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
	case "esc", "q", "b":
		m.activeModal = ""
	}
	if m.detailOffset < 0 {
		m.detailOffset = 0
	}
}
