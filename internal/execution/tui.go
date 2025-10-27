package execution

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"golang.org/x/term"
)

// ============================================================================
// GLANCE-STYLE TUI - Complete Widget-Based Dashboard
// ============================================================================

// EnhancedTUIModel - the main Bubble Tea model for glance-style dashboard
type EnhancedTUIModel struct {
	// Basic state
	program      *tea.Program
	ctx          context.Context
	cancel       context.CancelFunc
	width        int
	height       int
	lastTickTime time.Time

	// Display state
	mode         DisplayMode // Normal, Verbose, Debug
	activeModal  string      // "", "help", "stats", "confirm"
	scrollOffset int         // for main log view
	logBuffer    []LogEntry  // circular log buffer
	logIndex     int         // current position in circular buffer
	maxLogs      int

	// Execution state
	playbookName     string
	playCount        int
	currentPlayIndex int
	currentTaskName  string
	currentHost      string
	startTime        time.Time
	elapsedTime      time.Duration
	status           string // "running", "paused", "stopped", "completed"

	// Statistics
	taskStats map[string]*DetailedTaskStats
	hostStats map[string]*DetailedHostStats
	playStats map[string]*DetailedPlayStats

	// Control state
	paused      bool
	shouldExit  bool
	gracefulReq bool

	// Filter & Search state (Phase 2)
	filterMode         bool   // Is filter mode active?
	searchMode         bool   // Is search mode active?
	searchQuery        string // Current search query
	filterShowErrors   bool   // Show only ERROR level logs
	filterShowWarnings bool   // Show only WARN level logs
	filterShowTasks    bool   // Show only task-related logs
	filteredIndices    []int  // Indices of logs matching current filter
	searchIndex        int    // Current search result position

	// Synchronization
	mutex        sync.RWMutex
	eventChan    chan ExecutionEvent
	stopChan     chan struct{}
	tickerChan   chan time.Time
	readyChan    chan struct{} // Signal when TUI is ready to accept logs
	stopCallback func() error
	playbookFunc func(context.Context) error
}

// DetailedTaskStats extends basic TaskStats with more info
type DetailedTaskStats struct {
	Name        string
	Success     int
	Failed      int
	Changed     int
	Skipped     int
	LastUpdate  time.Time
	Duration    time.Duration
	AvgDuration time.Duration
}

// DetailedHostStats extends basic HostStats with more info
type DetailedHostStats struct {
	Name          string
	TaskCount     int
	SuccessCount  int
	FailedCount   int
	ChangedCount  int
	SkippedCount  int
	LastTaskTime  time.Time
	TotalDuration time.Duration
}

// DetailedPlayStats extends basic PlayStats
type DetailedPlayStats struct {
	Name      string
	Index     int
	Total     int
	Completed int
	Failed    int
	Duration  time.Duration
	StartTime time.Time
}

// LogEntry represents a single log line
type LogEntry struct {
	Timestamp time.Time
	Level     string // INFO, WARN, ERROR, DEBUG, TASK_START, TASK_END, etc.
	Message   string
}

// NewEnhancedTUIModel creates a new enhanced TUI model
func NewEnhancedTUIModel() *EnhancedTUIModel {
	ctx, cancel := context.WithCancel(context.Background())

	return &EnhancedTUIModel{
		ctx:             ctx,
		cancel:          cancel,
		mode:            DisplayNormal,
		status:          "initializing",
		startTime:       time.Now(),
		logBuffer:       make([]LogEntry, 1000), // pre-allocate circular buffer
		maxLogs:         1000,
		taskStats:       make(map[string]*DetailedTaskStats),
		hostStats:       make(map[string]*DetailedHostStats),
		playStats:       make(map[string]*DetailedPlayStats),
		eventChan:       make(chan ExecutionEvent, 100),
		stopChan:        make(chan struct{}),
		tickerChan:      make(chan time.Time),
		readyChan:       make(chan struct{}),
		activeModal:     "",
		shouldExit:      false,
		paused:          false,
		gracefulReq:     false,
		filterMode:      false,
		searchMode:      false,
		searchQuery:     "",
		filteredIndices: make([]int, 0),
		searchIndex:     0,
	}
}

// Start runs the TUI (blocking call)
func (m *EnhancedTUIModel) Start() error {
	// Verify stdin is a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("interactive mode requires a terminal")
	}

	// Get initial terminal dimensions
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil {
		m.width = w
		m.height = h
	} else {
		// Default fallback size
		m.width = 120
		m.height = 30
	}

	// Create and run the Bubble Tea program with alt screen
	m.program = tea.NewProgram(m, tea.WithAltScreen())

	// Start event processor goroutine
	go m.eventProcessor()

	// Start ticker for updates
	go m.tickerLoop()

	// Run the TUI (blocking)
	if _, err := m.program.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// Stop stops the TUI
func (m *EnhancedTUIModel) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.shouldExit = true

	if m.cancel != nil {
		m.cancel()
	}

	close(m.stopChan)

	if m.program != nil {
		m.program.Quit()
	}
}

// GetLogWriter returns an io.Writer that captures logs into the TUI's log buffer
// This allows the logger to send its output to the TUI instead of stdout
func (m *EnhancedTUIModel) GetLogWriter() io.Writer {
	return &tuiLogWriter{tui: m}
}

// WaitForReady blocks until the TUI is ready to accept logs
// This ensures the alt screen is active before playbook execution starts
func (m *EnhancedTUIModel) WaitForReady(timeout time.Duration) error {
	select {
	case <-m.readyChan:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("TUI failed to initialize within %v", timeout)
	}
}

// tuiLogWriter implements io.Writer and captures logs for the TUI
type tuiLogWriter struct {
	tui            *EnhancedTUIModel
	incompleteLine string // buffer for incomplete lines from partial writes
	mu             sync.Mutex
}

// Write implements io.Writer
func (w *tuiLogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Prepend any incomplete line from previous write
	data := w.incompleteLine + string(p)
	w.incompleteLine = ""

	// Process complete lines
	lines := strings.Split(data, "\n")

	// The last element is either empty (if data ends with \n) or incomplete line
	// All other elements are complete lines
	for i := 0; i < len(lines)-1; i++ {
		line := lines[i]
		if line != "" {
			w.tui.AddLog(line)
		}
	}

	// Keep the last element as incomplete line for next write
	if len(lines) > 0 {
		w.incompleteLine = lines[len(lines)-1]
	}

	return len(p), nil
}

// AddLog adds a log message to the TUI's buffer
func (m *EnhancedTUIModel) AddLog(message string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO", // default level, could be parsed from message
		Message:   message,
	}

	// Add to circular buffer
	m.logBuffer[m.logIndex] = entry
	m.logIndex = (m.logIndex + 1) % m.maxLogs
}

// Init initializes the model (Bubble Tea interface)
func (m *EnhancedTUIModel) Init() tea.Cmd {
	// Signal that TUI is ready to accept logs (non-blocking)
	go func() {
		select {
		case m.readyChan <- struct{}{}:
		default:
		}
	}()

	return tea.Batch(
		m.listenForEvents(),
		m.listenForTicks(),
	)
}

// Update processes messages (Bubble Tea interface)
func (m *EnhancedTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeypress(msg)

	case ExecutionEvent:
		m.processEvent(msg)
		return m, m.listenForEvents()

	case tickMsg:
		m.updateElapsedTime()
		return m, m.listenForTicks()

	case tea.QuitMsg:
		return m, tea.Quit
	}

	return m, nil
}

// View renders the complete UI (Bubble Tea interface)
func (m *EnhancedTUIModel) View() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Render based on active modal
	if m.activeModal == "help" {
		return m.renderHelpModal()
	}
	if m.activeModal == "stats" {
		return m.renderStatsModal()
	}
	if m.activeModal == "confirm" {
		return m.renderConfirmModal()
	}

	// Render main dashboard
	return m.renderDashboard()
}

// ============================================================================
// RENDERING FUNCTIONS
// ============================================================================

// renderDashboard renders the main glance-style dashboard
func (m *EnhancedTUIModel) renderDashboard() string {
	if m.width < 80 || m.height < 24 {
		return "Terminal too small. Minimum: 80x24\n"
	}

	// Calculate layout dimensions - CRITICAL: leave room for vertical separator
	borderWidth := 3 // 1 for each border + 1 for separator
	statsWidth := 28 // Fixed width for stats panel content
	logsWidth := m.width - statsWidth - borderWidth
	contentHeight := m.height - 4 // -4 for header and footer

	// Build sections
	header := m.renderHeader()
	logs := m.renderLogsPanel(logsWidth, contentHeight)
	stats := m.renderStatsPanel(statsWidth, contentHeight)
	footer := m.renderFooter()

	// Layout: header / (logs | stats) / footer using proper Lipgloss join
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		logs,
		stats,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

// renderHeader renders the top header bar with progress
func (m *EnhancedTUIModel) renderHeader() string {
	mode := m.getModeDisplay()
	status := m.getStatusDisplay()

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("238")).
		Bold(true).
		Padding(0, 1)

	left := fmt.Sprintf("onigirazu - %s", m.playbookName)
	middle := fmt.Sprintf("[%s] %s | ⏱ %s", status, mode, formatDuration(m.elapsedTime))
	right := formatTime(time.Now())

	// Calculate spacing to center
	totalContent := len(left) + len(middle) + len(right) + 4
	if totalContent > m.width {
		return headerStyle.Width(m.width).Render(left + " ... " + status)
	}

	spacing1 := (m.width - len(left) - len(middle)) / 2
	spacing2 := m.width - len(left) - spacing1 - len(middle) - len(right)

	header := left + strings.Repeat(" ", spacing1) + middle + strings.Repeat(" ", spacing2) + right

	headerLine := headerStyle.Width(m.width).Render(header[:m.width])

	// Add progress bar on a second header line if there's space and we have progress data
	_, playTotal := m.getPlayProgress()
	if m.height >= 26 && playTotal > 0 {
		progBarStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("242")).
			Padding(0, 1)

		_, failedTasks, _, changedTasks := m.getTaskProgress()
		successTasks, _, _, _ := m.getTaskProgress()

		statusStr := fmt.Sprintf("Tasks: %s%d success %s%d failed %s%d changed",
			lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("✓"),
			successTasks,
			lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗"),
			failedTasks,
			lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render("⟳"),
			changedTasks,
		)

		progressLine := progBarStyle.Render(statusStr + strings.Repeat(" ", m.width-len(statusStr)-2))
		return headerLine + "\n" + progressLine
	}

	return headerLine
}

// renderLogsPanel renders the scrollable logs section
func (m *EnhancedTUIModel) renderLogsPanel(width int, height int) string {
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(width).
		Height(height).
		Padding(1)

	// Reserve space: border (1) + padding (1) = 2 on each side
	contentWidth := width - 4
	contentHeight := height - 4

	// Absolute safety: ensure contentHeight is at least 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Get visible logs
	logs := m.getVisibleLogs(contentHeight)
	logLines := make([]string, 0, contentHeight)

	for _, entry := range logs {
		line := m.formatLogLine(entry, contentWidth)
		// SAFETY: Ensure no line exceeds contentWidth
		if len(line) > contentWidth {
			line = line[:contentWidth]
		}
		logLines = append(logLines, line)
	}

	// Truncate to exact height FIRST (before padding)
	// Always show the most recent logs
	if len(logLines) > contentHeight {
		logLines = logLines[len(logLines)-contentHeight:]
	}

	// Pad to exact height - NO overflow allowed
	for len(logLines) < contentHeight {
		logLines = append(logLines, "")
	}

	// CRITICAL: Join with exactly contentHeight lines
	content := strings.Join(logLines, "\n")

	// Extra safety: verify line count
	lineCount := strings.Count(content, "\n") + 1
	if lineCount > contentHeight {
		// Something went wrong, truncate aggressively
		contentLines := strings.Split(content, "\n")
		if len(contentLines) > contentHeight {
			contentLines = contentLines[len(contentLines)-contentHeight:]
		}
		content = strings.Join(contentLines, "\n")
	}

	return panelStyle.Render(content)
}

// renderStatsPanel renders the statistics panel on the right
func (m *EnhancedTUIModel) renderStatsPanel(width int, height int) string {
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(width).
		Height(height).
		Padding(1)

	// Reserve space: border (1) + padding (1) = 2 on each side
	contentWidth := width - 4
	contentHeight := height - 4

	// Absolute safety: ensure contentHeight is at least 1
	if contentHeight < 1 {
		contentHeight = 1
	}

	var lines []string

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51"))
	lines = append(lines, titleStyle.Render("📊 Statistics"))
	lines = append(lines, "")

	// Play progress with bar
	playCurrent, playTotal := m.getPlayProgress()
	if playTotal > 0 {
		playBar := renderProgressBar(playCurrent, playTotal, contentWidth-3)
		lines = append(lines, "Plays:")
		lines = append(lines, "  "+playBar)
	} else {
		lines = append(lines, "Plays: -/-")
	}

	// Task stats summary with indicators and progress
	successTasks, failedTasks, skippedTasks, changedTasks := m.getTaskProgress()
	totalTasks := successTasks + failedTasks + skippedTasks + changedTasks

	lines = append(lines, "")
	lines = append(lines, "Tasks:")

	// Task progress bar
	if totalTasks > 0 {
		taskBar := renderProgressBar(successTasks+changedTasks, totalTasks, contentWidth-3)
		lines = append(lines, "  "+taskBar)
	}

	// Task breakdown with indicators
	lines = append(lines, fmt.Sprintf("  %s %d | %s %d", renderTaskStatusIndicator("SUCCESS"), successTasks, renderTaskStatusIndicator("FAILED"), failedTasks))
	lines = append(lines, fmt.Sprintf("  %s %d | %s %d", renderTaskStatusIndicator("CHANGED"), changedTasks, renderTaskStatusIndicator("SKIPPED"), skippedTasks))

	// Host count
	lines = append(lines, "")
	hostCountStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("45"))
	lines = append(lines, hostCountStyle.Render(fmt.Sprintf("Hosts: %d", len(m.hostStats))))

	// Current execution info (if there's space)
	if len(lines) < contentHeight-4 && m.currentTaskName != "" {
		lines = append(lines, "")
		lines = append(lines, "Executing:")
		taskStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Italic(true)
		taskName := truncateString(m.currentTaskName, contentWidth-2)
		lines = append(lines, "  "+taskStyle.Render(taskName))

		if len(lines) < contentHeight-2 && m.currentHost != "" {
			hostStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
			hostName := truncateString(m.currentHost, contentWidth-2)
			lines = append(lines, "  "+hostStyle.Render(hostName))
		}
	}

	// Performance metrics (if there's space)
	if len(lines) < contentHeight-1 && m.elapsedTime > 0 && totalTasks > 0 {
		minutesElapsed := m.elapsedTime.Minutes()
		if minutesElapsed > 0 {
			tasksPerMin := float64(totalTasks) / minutesElapsed
			lines = append(lines, "")
			perfStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
			lines = append(lines, perfStyle.Render(fmt.Sprintf("%.1f tasks/min", tasksPerMin)))
		}
	}

	// Truncate to exact height FIRST (before padding)
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}

	// Pad to exact height - NEVER overflow
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")

	// Extra safety: verify line count
	lineCount := strings.Count(content, "\n") + 1
	if lineCount > contentHeight {
		// Something went wrong, truncate aggressively
		contentLines := strings.Split(content, "\n")
		if len(contentLines) > contentHeight {
			contentLines = contentLines[:contentHeight]
		}
		content = strings.Join(contentLines, "\n")
	}

	return panelStyle.Render(content)
}

// renderFooter renders the bottom footer with key hints
func (m *EnhancedTUIModel) renderFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1)

	hints := []string{
		"H: Help",
		"S: Stats",
		"F: Filter",
		"/: Search",
		"V: Verbose",
		"D: Debug",
		"Q: Quit",
	}

	if m.paused {
		hints = append([]string{"⏸ PAUSED"}, hints...)
	}

	footer := strings.Join(hints, " │ ")

	// Add filter status if active
	filterStatus := m.getFilterStatus()
	if filterStatus != "" {
		footer = fmt.Sprintf("%s  %s", footer, filterStatus)
	}

	// Add search mode indicator
	if m.searchMode {
		searchPrompt := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render(fmt.Sprintf(" / %s_", m.searchQuery))
		footer = fmt.Sprintf("%s%s", footer, searchPrompt)
	}

	return footerStyle.Render(footer)
}

// renderHelpModal renders the help modal overlay
func (m *EnhancedTUIModel) renderHelpModal() string {
	// Render dashboard as background
	dashboard := m.renderDashboard()

	// Create modal content
	var helpLines []string
	helpLines = append(helpLines, "")
	helpLines = append(helpLines, "  KEYBOARD SHORTCUTS")
	helpLines = append(helpLines, "  ──────────────────")
	helpLines = append(helpLines, "")
	helpLines = append(helpLines, "  DISPLAY & NAVIGATION:")
	helpLines = append(helpLines, "    H - Help (this screen)")
	helpLines = append(helpLines, "    S - Stats modal")
	helpLines = append(helpLines, "    ↑↓ - Scroll logs | PgUp/PgDn - Page scroll")
	helpLines = append(helpLines, "")
	helpLines = append(helpLines, "  FILTERING & SEARCH (Phase 2):")
	helpLines = append(helpLines, "    F - Toggle filter mode")
	helpLines = append(helpLines, "    / - Start search")
	helpLines = append(helpLines, "    When filtering: E=errors W=warnings T=tasks C=clear")
	helpLines = append(helpLines, "")
	helpLines = append(helpLines, "  MODES:")
	helpLines = append(helpLines, "    V - Toggle VERBOSE  |  D - Toggle DEBUG  |  N - NORMAL")
	helpLines = append(helpLines, "    P - Pause/Resume    |  G - Graceful stop  |  Q - Quit")
	helpLines = append(helpLines, "")

	// Pad to minimum height
	for len(helpLines) < 15 {
		helpLines = append(helpLines, "")
	}

	helpText := strings.Join(helpLines, "\n")

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("51")).
		Background(lipgloss.Color("16")).
		Foreground(lipgloss.Color("15")).
		Width(60).
		Height(len(helpLines)+2).
		Padding(0, 1)

	// Render overlay
	modal := overlayStyle.Render(helpText)

	// Position modal in center using dashboard as base
	// For now, just overlay on dashboard
	return lipgloss.JoinVertical(lipgloss.Center, dashboard, modal)
}

// renderStatsModal renders the detailed statistics modal
func (m *EnhancedTUIModel) renderStatsModal() string {
	// Render dashboard as background
	dashboard := m.renderDashboard()

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  DETAILED STATISTICS")
	lines = append(lines, "  ──────────────────")
	lines = append(lines, "")

	// Task statistics (limit to 8 entries)
	if len(m.taskStats) > 0 {
		lines = append(lines, "  TASKS:")
		taskCount := 0
		for name, stats := range m.taskStats {
			if taskCount >= 8 {
				break
			}
			taskLine := fmt.Sprintf("    %s", truncateString(name, 45))
			taskLine += fmt.Sprintf(" S:%d F:%d C:%d", stats.Success, stats.Failed, stats.Changed)
			lines = append(lines, taskLine)
			taskCount++
		}
		lines = append(lines, "")
	}

	// Host statistics (limit to 5 entries)
	if len(m.hostStats) > 0 {
		lines = append(lines, "  HOSTS:")
		hostCount := 0
		for hostname, stats := range m.hostStats {
			if hostCount >= 5 {
				break
			}
			hostLine := fmt.Sprintf("    %s", truncateString(hostname, 40))
			hostLine += fmt.Sprintf(" T:%d S:%d F:%d", stats.TaskCount, stats.SuccessCount, stats.FailedCount)
			lines = append(lines, hostLine)
			hostCount++
		}
		lines = append(lines, "")
	}

	lines = append(lines, "  Press S or ESC to close")
	lines = append(lines, "")

	// Pad to minimum height
	for len(lines) < 18 {
		lines = append(lines, "")
	}

	statsText := strings.Join(lines, "\n")

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("226")).
		Background(lipgloss.Color("16")).
		Foreground(lipgloss.Color("15")).
		Width(65).
		Height(len(lines)+2).
		Padding(0, 1)

	// Render overlay
	modal := overlayStyle.Render(statsText)

	// Position modal below dashboard
	return lipgloss.JoinVertical(lipgloss.Center, dashboard, modal)
}

// renderConfirmModal renders a confirmation modal
func (m *EnhancedTUIModel) renderConfirmModal() string {
	// Render dashboard as background
	dashboard := m.renderDashboard()

	var confirmLines []string
	confirmLines = append(confirmLines, "")
	confirmLines = append(confirmLines, "  GRACEFUL SHUTDOWN")
	confirmLines = append(confirmLines, "  ─────────────────")
	confirmLines = append(confirmLines, "")
	confirmLines = append(confirmLines, "  Stop execution gracefully?")
	confirmLines = append(confirmLines, "")
	confirmLines = append(confirmLines, "  Running tasks will complete.")
	confirmLines = append(confirmLines, "  No new tasks will start.")
	confirmLines = append(confirmLines, "")
	confirmLines = append(confirmLines, "  [Y] Continue with shutdown")
	confirmLines = append(confirmLines, "  [N] Cancel")
	confirmLines = append(confirmLines, "")

	confirmText := strings.Join(confirmLines, "\n")

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("226")).
		Background(lipgloss.Color("16")).
		Foreground(lipgloss.Color("15")).
		Width(55).
		Height(len(confirmLines)+2).
		Padding(0, 1)

	// Render overlay
	modal := overlayStyle.Render(confirmText)

	// Position modal below dashboard
	return lipgloss.JoinVertical(lipgloss.Center, dashboard, modal)
}

// ============================================================================
// HELPER RENDERING FUNCTIONS
// ============================================================================

// formatLogLine formats a single log entry for display
func (m *EnhancedTUIModel) formatLogLine(entry LogEntry, maxWidth int) string {
	var prefix string

	switch entry.Level {
	case "ERROR":
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗")
	case "WARN":
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render("⚠")
	case "TASK_START":
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Render("▶")
	case "TASK_END":
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("✓")
	case "DEBUG":
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render("◆")
	default:
		prefix = "·"
	}

	timeStr := entry.Timestamp.Format("15:04:05")
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	msg := truncateString(entry.Message, maxWidth-len(timeStr)-3)
	return timeStyle.Render(timeStr) + " " + prefix + " " + msg
}

// getVisibleLogs returns the logs to display
func (m *EnhancedTUIModel) getVisibleLogs(count int) []LogEntry {
	var result []LogEntry

	// Use filtered logs if any filter is active, otherwise use all logs
	var indices []int
	if m.filterShowErrors || m.filterShowWarnings || m.filterShowTasks || m.searchQuery != "" {
		// Any filter active: use filtered indices (even if empty = "no matches")
		indices = m.filteredIndices
	} else {
		// No filters active: collect all non-empty log entries
		for i, entry := range m.logBuffer {
			if entry.Timestamp.Unix() > 0 {
				indices = append(indices, i)
			}
		}
	}

	// Collect log entries from indices
	for _, idx := range indices {
		if idx >= 0 && idx < len(m.logBuffer) {
			if m.logBuffer[idx].Timestamp.Unix() > 0 {
				result = append(result, m.logBuffer[idx])
			}
		}
	}

	// Sort by timestamp (latest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	// Return last `count` entries
	if len(result) > count {
		result = result[:count]
	}

	// Reverse to show chronologically (oldest first)
	for i := len(result)/2 - 1; i >= 0; i-- {
		opp := len(result) - 1 - i
		result[i], result[opp] = result[opp], result[i]
	}

	return result
}

// ============================================================================
// PROGRESS & STATUS RENDERING ENHANCEMENTS
// ============================================================================

// renderProgressBar renders a visual progress bar with percentage
func renderProgressBar(current, total, width int) string {
	if width < 10 || total <= 0 {
		return ""
	}

	percentage := 0
	if total > 0 {
		percentage = (current * 100) / total
	}

	// Calculate filled bars (account for brackets and percentage text)
	barWidth := width - 10 // Leave room for [, ], space, and percentage
	if barWidth < 5 {
		barWidth = 5
	}

	filled := (barWidth * current) / total
	if current > 0 && filled == 0 {
		filled = 1 // Show at least 1 if current > 0
	}

	// Determine color based on percentage
	var barStyle lipgloss.Style
	switch {
	case percentage >= 80:
		barStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46")) // Bright green
	case percentage >= 50:
		barStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82")) // Green
	case percentage >= 25:
		barStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226")) // Yellow
	default:
		barStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	return fmt.Sprintf("[%s] %3d%%", barStyle.Render(bar), percentage)
}

// getPlayProgress returns the current play and total plays with estimated ETA
func (m *EnhancedTUIModel) getPlayProgress() (current, total int) {
	return m.currentPlayIndex, m.playCount
}

// getTaskProgress returns overall task statistics
func (m *EnhancedTUIModel) getTaskProgress() (success, failed, skipped, changed int) {
	for _, stats := range m.taskStats {
		success += stats.Success
		failed += stats.Failed
		skipped += stats.Skipped
		changed += stats.Changed
	}
	return
}

// renderTaskStatusIndicator renders a task status with emoji and color
func renderTaskStatusIndicator(level string) string {
	styles := map[string]lipgloss.Style{
		"SUCCESS":     lipgloss.NewStyle().Foreground(lipgloss.Color("46")),  // Bright green
		"FAILED":      lipgloss.NewStyle().Foreground(lipgloss.Color("196")), // Red
		"SKIPPED":     lipgloss.NewStyle().Foreground(lipgloss.Color("242")), // Gray
		"CHANGED":     lipgloss.NewStyle().Foreground(lipgloss.Color("226")), // Yellow
		"IN_PROGRESS": lipgloss.NewStyle().Foreground(lipgloss.Color("51")),  // Cyan
	}

	indicators := map[string]string{
		"SUCCESS":     "✓",
		"FAILED":      "✗",
		"SKIPPED":     "⊘",
		"CHANGED":     "⟳",
		"IN_PROGRESS": "◉",
	}

	style := styles[level]
	indicator := indicators[level]
	if indicator == "" {
		indicator = "·"
	}

	return style.Render(indicator)
}

// renderDetailedStatus renders enhanced status line with performance info
func (m *EnhancedTUIModel) renderDetailedStatus() string {
	success, failed, skipped, changed := m.getTaskProgress()
	total := success + failed + skipped + changed

	var parts []string

	// Overall progress
	if m.playCount > 0 {
		parts = append(parts, fmt.Sprintf("Plays: %d/%d", m.currentPlayIndex, m.playCount))
	}

	// Task summary with indicators
	taskSummary := fmt.Sprintf("Tasks: %s%d %s%d %s%d %s%d",
		renderTaskStatusIndicator("SUCCESS"), success,
		renderTaskStatusIndicator("FAILED"), failed,
		renderTaskStatusIndicator("CHANGED"), changed,
		renderTaskStatusIndicator("SKIPPED"), skipped,
	)
	parts = append(parts, taskSummary)

	// Execution rate (tasks per minute)
	if m.elapsedTime > 0 && total > 0 {
		minutesElapsed := m.elapsedTime.Minutes()
		if minutesElapsed > 0 {
			tasksPerMin := float64(total) / minutesElapsed
			parts = append(parts, fmt.Sprintf("%.1f tasks/min", tasksPerMin))
		}
	}

	// Hosts count
	if len(m.hostStats) > 0 {
		parts = append(parts, fmt.Sprintf("Hosts: %d", len(m.hostStats)))
	}

	return strings.Join(parts, " | ")
}

// ============================================================================
// FILTERING & SEARCH (Phase 2)
// ============================================================================

// applyFilter builds the list of filtered log indices based on current filter settings
func (m *EnhancedTUIModel) applyFilter() {
	m.filteredIndices = make([]int, 0)

	// If no filters are active, include all logs
	if !m.filterShowErrors && !m.filterShowWarnings && !m.filterShowTasks && m.searchQuery == "" {
		for i := 0; i < len(m.logBuffer); i++ {
			if m.logBuffer[i].Message != "" {
				m.filteredIndices = append(m.filteredIndices, i)
			}
		}
		return
	}

	// Apply filters
	for i := 0; i < len(m.logBuffer); i++ {
		entry := m.logBuffer[i]
		if entry.Message == "" {
			continue
		}

		// Check error filter
		if m.filterShowErrors && entry.Level != "ERROR" {
			continue
		}

		// Check warning filter
		if m.filterShowWarnings && entry.Level != "WARN" {
			continue
		}

		// Check task filter
		if m.filterShowTasks {
			if !strings.Contains(entry.Level, "TASK") && entry.Level != "ok" && entry.Level != "failed" && entry.Level != "changed" {
				continue
			}
		}

		// Check search query
		if m.searchQuery != "" {
			if !strings.Contains(strings.ToLower(entry.Message), strings.ToLower(m.searchQuery)) {
				continue
			}
		}

		m.filteredIndices = append(m.filteredIndices, i)
	}

	// Reset search index when filter changes
	m.searchIndex = 0
}

// toggleFilterMode toggles filter mode on/off
func (m *EnhancedTUIModel) toggleFilterMode() {
	m.filterMode = !m.filterMode
	if !m.filterMode {
		m.filterShowErrors = false
		m.filterShowWarnings = false
		m.filterShowTasks = false
		m.searchQuery = ""
		m.applyFilter()
	}
}

// cycleErrorFilter toggles the error filter
func (m *EnhancedTUIModel) cycleErrorFilter() {
	m.filterShowErrors = !m.filterShowErrors
	m.filterShowWarnings = false
	m.filterShowTasks = false
	m.searchQuery = ""
	m.applyFilter()
}

// cycleWarningFilter toggles the warning filter
func (m *EnhancedTUIModel) cycleWarningFilter() {
	m.filterShowWarnings = !m.filterShowWarnings
	m.filterShowErrors = false
	m.filterShowTasks = false
	m.searchQuery = ""
	m.applyFilter()
}

// cycleTaskFilter toggles the task filter
func (m *EnhancedTUIModel) cycleTaskFilter() {
	m.filterShowTasks = !m.filterShowTasks
	m.filterShowErrors = false
	m.filterShowWarnings = false
	m.searchQuery = ""
	m.applyFilter()
}

// startSearchMode enters search mode
func (m *EnhancedTUIModel) startSearchMode() {
	m.searchMode = true
	m.searchQuery = ""
	m.applyFilter()
}

// exitSearchMode exits search mode
func (m *EnhancedTUIModel) exitSearchMode() {
	m.searchMode = false
}

// addSearchChar adds a character to the search query
func (m *EnhancedTUIModel) addSearchChar(ch string) {
	if len(m.searchQuery) < 100 { // Limit search query length
		m.searchQuery += ch
		m.applyFilter()
	}
}

// removeSearchChar removes the last character from search query
func (m *EnhancedTUIModel) removeSearchChar() {
	if len(m.searchQuery) > 0 {
		m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		m.applyFilter()
	}
}

// clearFilters clears all filters and search
func (m *EnhancedTUIModel) clearFilters() {
	m.filterMode = false
	m.searchMode = false
	m.searchQuery = ""
	m.filterShowErrors = false
	m.filterShowWarnings = false
	m.filterShowTasks = false
	m.applyFilter()
	m.addLog(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Filters cleared",
	})
}

// getFilterStatus returns a string describing current filter status
func (m *EnhancedTUIModel) getFilterStatus() string {
	var status []string

	if m.filterShowErrors {
		status = append(status, "ERRORS")
	}
	if m.filterShowWarnings {
		status = append(status, "WARNINGS")
	}
	if m.filterShowTasks {
		status = append(status, "TASKS")
	}
	if m.searchQuery != "" {
		status = append(status, fmt.Sprintf(`SEARCH: "%s"`, m.searchQuery))
	}

	if len(status) == 0 {
		return ""
	}
	return fmt.Sprintf("[%s]", strings.Join(status, " | "))
}

// ============================================================================
// KEYBOARD HANDLING
// ============================================================================

// handleKeypress handles keyboard input
func (m *EnhancedTUIModel) handleKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If a modal is active, handle modal-specific keys
	if m.activeModal != "" {
		return m.handleModalKeypress(msg)
	}

	// If search mode is active, handle search input differently
	if m.searchMode {
		return m.handleSearchKeypress(msg)
	}

	// Handle global keys
	// Normalize key input: convert "shift+X" to "x" for case-insensitive handling
	keyStr := msg.String()
	if strings.HasPrefix(keyStr, "shift+") {
		keyStr = strings.TrimPrefix(keyStr, "shift+")
	}
	keyStr = strings.ToLower(keyStr)

	switch keyStr {
	case "ctrl+c", "q":
		m.shouldExit = true
		m.mutex.Lock()
		m.activeModal = ""
		m.mutex.Unlock()
		return m, tea.Quit

	case "h":
		m.mutex.Lock()
		if m.activeModal == "help" {
			m.activeModal = ""
		} else {
			m.activeModal = "help"
		}
		m.mutex.Unlock()

	case "s":
		m.mutex.Lock()
		if m.activeModal == "stats" {
			m.activeModal = ""
		} else {
			m.activeModal = "stats"
		}
		m.mutex.Unlock()

	case "v":
		m.mutex.Lock()
		if m.mode == DisplayVerbose {
			m.mode = DisplayNormal
		} else {
			m.mode = DisplayVerbose
		}
		m.addLog(LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   fmt.Sprintf("Mode: %s", m.getModeDisplay()),
		})
		m.mutex.Unlock()

	case "d":
		m.mutex.Lock()
		if m.mode == DisplayDebug {
			m.mode = DisplayNormal
		} else {
			m.mode = DisplayDebug
		}
		m.addLog(LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   fmt.Sprintf("Mode: %s", m.getModeDisplay()),
		})
		m.mutex.Unlock()

	case "n":
		m.mutex.Lock()
		if m.mode != DisplayNormal {
			m.mode = DisplayNormal
			m.addLog(LogEntry{
				Timestamp: time.Now(),
				Level:     "INFO",
				Message:   "Mode: NORMAL",
			})
		}
		m.mutex.Unlock()

	case "p":
		m.mutex.Lock()
		m.paused = !m.paused
		statusMsg := "Execution PAUSED"
		if !m.paused {
			statusMsg = "Execution RESUMED"
		}
		m.addLog(LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   statusMsg,
		})
		m.mutex.Unlock()

	case "g":
		m.mutex.Lock()
		m.activeModal = "confirm"
		m.mutex.Unlock()

	case "f":
		m.mutex.Lock()
		m.toggleFilterMode()
		if m.filterMode {
			m.addLog(LogEntry{
				Timestamp: time.Now(),
				Level:     "INFO",
				Message:   "Filter mode ON. Press: E=errors, W=warnings, T=tasks, /=search, C=clear",
			})
		}
		m.mutex.Unlock()

	case "e":
		if m.filterMode {
			m.mutex.Lock()
			m.cycleErrorFilter()
			m.mutex.Unlock()
		}

	case "w":
		if m.filterMode {
			m.mutex.Lock()
			m.cycleWarningFilter()
			m.mutex.Unlock()
		}

	case "t":
		if m.filterMode {
			m.mutex.Lock()
			m.cycleTaskFilter()
			m.mutex.Unlock()
		}

	case "/":
		m.mutex.Lock()
		m.startSearchMode()
		m.mutex.Unlock()

	case "c":
		if m.filterMode || m.searchMode {
			m.mutex.Lock()
			m.clearFilters()
			m.mutex.Unlock()
		}

	case "up":
		m.mutex.Lock()
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
		m.mutex.Unlock()

	case "down":
		m.mutex.Lock()
		m.scrollOffset++
		m.mutex.Unlock()

	case "pageup":
		m.mutex.Lock()
		m.scrollOffset -= m.height / 2
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
		m.mutex.Unlock()

	case "pagedown":
		m.mutex.Lock()
		m.scrollOffset += m.height / 2
		m.mutex.Unlock()
	}

	return m, nil
}

// handleModalKeypress handles keys while a modal is open
func (m *EnhancedTUIModel) handleModalKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "escape", "q":
		m.mutex.Lock()
		m.activeModal = ""
		m.mutex.Unlock()

	case "h":
		m.mutex.Lock()
		if m.activeModal == "help" {
			m.activeModal = ""
		}
		m.mutex.Unlock()

	case "s":
		m.mutex.Lock()
		if m.activeModal == "stats" {
			m.activeModal = ""
		}
		m.mutex.Unlock()

	case "y":
		if m.activeModal == "confirm" {
			m.mutex.Lock()
			m.gracefulReq = true
			m.activeModal = ""
			m.status = "stopping"
			m.mutex.Unlock()

			// Trigger graceful stop
			if m.stopCallback != nil {
				go m.stopCallback()
			}
		}

	case "n":
		if m.activeModal == "confirm" {
			m.mutex.Lock()
			m.activeModal = ""
			m.mutex.Unlock()
		}
	}

	return m, nil
}

// handleSearchKeypress handles keys while in search mode
func (m *EnhancedTUIModel) handleSearchKeypress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch msg.String() {
	case "escape":
		m.exitSearchMode()
	case "enter":
		// Search confirmed, exit search mode but keep the filter active
		m.exitSearchMode()
	case "backspace":
		m.removeSearchChar()
	case "ctrl+c":
		m.exitSearchMode()
		m.clearFilters()
		m.shouldExit = true
		return m, tea.Quit
	default:
		// Add printable characters to search
		if len(msg.String()) == 1 && msg.Runes[0] >= 32 && msg.Runes[0] < 127 {
			m.addSearchChar(msg.String())
		}
	}

	return m, nil
}

// ============================================================================
// EVENT PROCESSING
// ============================================================================

// addLog adds a log entry to the circular buffer
func (m *EnhancedTUIModel) addLog(entry LogEntry) {
	if entry.Timestamp.Unix() == 0 {
		entry.Timestamp = time.Now()
	}

	m.logBuffer[m.logIndex] = entry
	m.logIndex = (m.logIndex + 1) % m.maxLogs

	// Reapply filters if any are active to include the new log
	if m.filterShowErrors || m.filterShowWarnings || m.filterShowTasks || m.searchQuery != "" {
		m.applyFilter()
	}
}

// processEvent processes an execution event
func (m *EnhancedTUIModel) processEvent(event ExecutionEvent) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	level := "INFO"
	msg := event.Message

	switch event.Type {
	case "execution_start":
		m.playbookName = event.PlayName
		m.playCount = event.PlayIndex
		m.status = "running"
		m.startTime = time.Now()
		level = "INFO"
		msg = fmt.Sprintf("Starting playbook: %s", event.PlayName)

	case "play_start":
		m.currentPlayIndex = event.PlayIndex
		level = "INFO"

	case "play_end":
		level = "INFO"

	case "task_start":
		m.currentTaskName = event.TaskName
		m.currentHost = event.HostName
		level = "TASK_START"

	case "task_end":
		level = "TASK_END"
		// Update stats
		if stats, exists := m.taskStats[event.TaskName]; exists {
			stats.LastUpdate = time.Now()
		} else {
			m.taskStats[event.TaskName] = &DetailedTaskStats{
				Name:       event.TaskName,
				LastUpdate: time.Now(),
			}
		}

	case "error":
		level = "ERROR"

	case "debug":
		if m.mode == DisplayDebug {
			level = "DEBUG"
		} else {
			return // Skip debug messages in non-debug mode
		}
	}

	m.addLog(LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
	})
}

// eventProcessor processes events from the execution
func (m *EnhancedTUIModel) eventProcessor() {
	for {
		select {
		case event := <-m.eventChan:
			if m.program != nil {
				m.program.Send(event)
			}
		case <-m.stopChan:
			return
		}
	}
}

// ============================================================================
// BUBBLE TEA MESSAGE TYPES
// ============================================================================

type tickMsg time.Time

// listenForTicks returns a command that listens for ticker updates
func (m *EnhancedTUIModel) listenForTicks() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// listenForEvents returns a command that listens for execution events
func (m *EnhancedTUIModel) listenForEvents() tea.Cmd {
	return func() tea.Msg {
		select {
		case event := <-m.eventChan:
			return event
		case <-m.ctx.Done():
			return nil
		}
	}
}

// tickerLoop sends periodic tick messages
func (m *EnhancedTUIModel) tickerLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if m.program != nil {
				m.program.Send(tickMsg(time.Now()))
			}
		case <-m.stopChan:
			return
		}
	}
}

// updateElapsedTime updates the elapsed time display
func (m *EnhancedTUIModel) updateElapsedTime() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.status == "running" && !m.paused {
		m.elapsedTime = time.Since(m.startTime)
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// getModeDisplay returns the display string for current mode
func (m *EnhancedTUIModel) getModeDisplay() string {
	switch m.mode {
	case DisplayVerbose:
		return "VERBOSE"
	case DisplayDebug:
		return "DEBUG"
	default:
		return "NORMAL"
	}
}

// getStatusDisplay returns the display string for current status
func (m *EnhancedTUIModel) getStatusDisplay() string {
	status := strings.ToUpper(m.status)
	if m.paused {
		return "⏸ " + status
	}
	return status
}

// formatDuration formats a duration nicely
func formatDuration(d time.Duration) string {
	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	s := (d % time.Minute) / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// formatTime formats a time nicely
func formatTime(t time.Time) string {
	return t.Format("15:04:05")
}

// truncateString truncates a string to a maximum width
func truncateString(s string, maxWidth int) string {
	if len(s) > maxWidth {
		if maxWidth > 3 {
			return s[:maxWidth-3] + "..."
		}
		return s[:maxWidth]
	}
	return s
}

// padString pads a string to a certain width
func padString(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// ============================================================================
// EXECUTION OBSERVER INTERFACE IMPLEMENTATION
// ============================================================================

// OnExecutionStart is called when execution begins
func (m *EnhancedTUIModel) OnExecutionStart(playbookName string, playCount int) {
	m.eventChan <- ExecutionEvent{
		Type:      "execution_start",
		PlayName:  playbookName,
		PlayIndex: playCount,
		Message:   fmt.Sprintf("Starting playbook: %s (%d plays)", playbookName, playCount),
		Timestamp: time.Now(),
	}
}

// OnPlayStart is called when a play starts
func (m *EnhancedTUIModel) OnPlayStart(playName string, playIndex int, totalPlays int) {
	m.eventChan <- ExecutionEvent{
		Type:      "play_start",
		PlayName:  playName,
		PlayIndex: playIndex,
		Message:   fmt.Sprintf("[%d/%d] Play: %s", playIndex, totalPlays, playName),
		Timestamp: time.Now(),
	}
}

// OnPlayEnd is called when a play ends
func (m *EnhancedTUIModel) OnPlayEnd(playName string, playIndex int, success bool, duration time.Duration) {
	status := "✓"
	if !success {
		status = "✗"
	}
	m.eventChan <- ExecutionEvent{
		Type:      "play_end",
		PlayName:  playName,
		PlayIndex: playIndex,
		Message:   fmt.Sprintf("%s Play completed: %s (%v)", status, playName, duration.Round(time.Millisecond)),
		Timestamp: time.Now(),
	}
}

// OnTaskStart is called when a task starts
func (m *EnhancedTUIModel) OnTaskStart(taskName string, hostName string) {
	m.eventChan <- ExecutionEvent{
		Type:      "task_start",
		TaskName:  taskName,
		HostName:  hostName,
		Message:   fmt.Sprintf("▶ %s on %s", taskName, hostName),
		Timestamp: time.Now(),
	}
}

// OnTaskEnd is called when a task ends
func (m *EnhancedTUIModel) OnTaskEnd(taskResult *types.TaskResult) {
	if taskResult == nil {
		return
	}

	status := "✓"
	if taskResult.Failed {
		status = "✗"
	} else if taskResult.Changed {
		status = "⟳"
	}

	m.eventChan <- ExecutionEvent{
		Type:      "task_end",
		TaskName:  taskResult.TaskName,
		HostName:  taskResult.Host,
		Message:   fmt.Sprintf("%s %s", status, taskResult.TaskName),
		Timestamp: time.Now(),
	}

	// Update stats
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if stats, exists := m.taskStats[taskResult.TaskName]; exists {
		if taskResult.Failed {
			stats.Failed++
		} else if taskResult.Changed {
			stats.Changed++
		} else if taskResult.Skipped {
			stats.Skipped++
		} else {
			stats.Success++
		}
		stats.LastUpdate = time.Now()
	} else {
		newStats := &DetailedTaskStats{Name: taskResult.TaskName}
		if taskResult.Failed {
			newStats.Failed = 1
		} else if taskResult.Changed {
			newStats.Changed = 1
		} else if taskResult.Skipped {
			newStats.Skipped = 1
		} else {
			newStats.Success = 1
		}
		newStats.LastUpdate = time.Now()
		m.taskStats[taskResult.TaskName] = newStats
	}

	// Update host stats
	if hostStats, exists := m.hostStats[taskResult.Host]; exists {
		hostStats.TaskCount++
		if taskResult.Failed {
			hostStats.FailedCount++
		} else {
			hostStats.SuccessCount++
		}
		hostStats.LastTaskTime = time.Now()
	} else {
		newHostStats := &DetailedHostStats{
			Name:      taskResult.Host,
			TaskCount: 1,
		}
		if taskResult.Failed {
			newHostStats.FailedCount = 1
		} else {
			newHostStats.SuccessCount = 1
		}
		newHostStats.LastTaskTime = time.Now()
		m.hostStats[taskResult.Host] = newHostStats
	}
}

// OnExecutionEnd is called when execution ends
func (m *EnhancedTUIModel) OnExecutionEnd(result *types.PlaybookResult, duration time.Duration) {
	status := "✓ COMPLETED"
	if result != nil && !result.Success {
		status = "✗ FAILED"
	}

	m.eventChan <- ExecutionEvent{
		Type:      "execution_end",
		Message:   fmt.Sprintf("%s in %v", status, duration.Round(time.Millisecond)),
		Timestamp: time.Now(),
	}

	m.mutex.Lock()
	m.status = "completed"
	m.mutex.Unlock()
}

// OnError is called when an error occurs
func (m *EnhancedTUIModel) OnError(taskName string, hostName string, errMsg string) {
	m.eventChan <- ExecutionEvent{
		Type:      "error",
		TaskName:  taskName,
		HostName:  hostName,
		Message:   fmt.Sprintf("ERROR: %s", errMsg),
		Timestamp: time.Now(),
	}
}

// SetStopCallback sets the callback for stop requests
func (m *EnhancedTUIModel) SetStopCallback(callback func() error) {
	m.stopCallback = callback
}

// WaitForExit blocks until the TUI signals exit
func (m *EnhancedTUIModel) WaitForExit() {
	for {
		m.mutex.RLock()
		shouldExit := m.shouldExit
		m.mutex.RUnlock()

		if shouldExit {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
}
