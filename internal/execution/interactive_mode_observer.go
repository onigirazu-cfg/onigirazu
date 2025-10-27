package execution

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"golang.org/x/term"
)

// InteractiveModeObserver handles interactive execution with live output and keyboard controls
type InteractiveModeObserver struct {
	mode              DisplayMode
	useColors         bool
	currentExecution  *ExecutionResult
	executionEvents   chan ExecutionEvent
	shouldExit        bool
	paused            bool
	pausedChan        chan bool
	displayUpdateChan chan DisplayMode
	mutex             sync.RWMutex
	outputMutex       sync.Mutex // Protects stdout access
	taskStats         map[string]*TaskStats
	hostStats         map[string]*HostStats
	terminalState     *term.State // Saved terminal state for restoration
	showingOverlay    bool        // Whether an overlay (help/stats) is currently displayed
}

// TaskStats tracks per-task statistics
type TaskStats struct {
	Name       string
	Success    int
	Failed     int
	Changed    int
	Skipped    int
	LastUpdate time.Time
}

// HostStats tracks per-host statistics
type HostStats struct {
	Name         string
	TaskCount    int
	SuccessCount int
	FailedCount  int
	LastTaskTime time.Time
}

// ExecutionEvent represents an execution event
type ExecutionEvent struct {
	Type      string // "execution_start", "play_start", "task_end", "execution_end", "error"
	PlayName  string
	PlayIndex int
	TaskName  string
	HostName  string
	Message   string
	Timestamp time.Time
}

// NewInteractiveModeObserver creates a new interactive mode observer
func NewInteractiveModeObserver(useColors bool) *InteractiveModeObserver {
	return &InteractiveModeObserver{
		mode:              DisplayNormal,
		useColors:         useColors,
		currentExecution:  &ExecutionResult{},
		executionEvents:   make(chan ExecutionEvent, 100),
		pausedChan:        make(chan bool, 1),
		displayUpdateChan: make(chan DisplayMode, 1),
		taskStats:         make(map[string]*TaskStats),
		hostStats:         make(map[string]*HostStats),
		showingOverlay:    false,
	}
}

// Screen management escape sequences
const (
	clearScreen   = "\033[2J" // Clear entire screen
	cursorHome    = "\033[H"  // Move cursor to home (0,0)
	cursorSave    = "\033[s"  // Save cursor position
	cursorRestore = "\033[u"  // Restore cursor position
	clearLine     = "\033[K"  // Clear from cursor to end of line
)

// clearAndHome clears the screen and moves cursor to home position
func (im *InteractiveModeObserver) clearAndHome() {
	fmt.Print(clearScreen + cursorHome)
}

// saveCursor saves the current cursor position
func (im *InteractiveModeObserver) saveCursor() {
	fmt.Print(cursorSave)
}

// restoreCursor restores the saved cursor position
func (im *InteractiveModeObserver) restoreCursor() {
	fmt.Print(cursorRestore)
}

// Start begins interactive mode with keyboard input handling
func (im *InteractiveModeObserver) Start() error {
	// Check if stdin is a TTY
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("failed to enable interactive mode: stdin is not a terminal (use --interactive only in interactive shells)")
	}

	// Set terminal to raw mode to read single keypresses without waiting for Enter
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set terminal raw mode: %w", err)
	}
	im.terminalState = oldState

	// Ensure terminal is restored on exit
	defer func() {
		if im.terminalState != nil {
			term.Restore(int(os.Stdin.Fd()), im.terminalState)
		}
	}()

	// NOTE: Signal handling is done by the main signal handler (signal_handler.go)
	// Don't set up duplicate signal handlers here

	// Start keyboard input handler
	go im.handleKeyboardInput()

	// Start event processor
	go im.processEvents()

	// Start display updater
	go im.handleDisplayUpdates()

	// Print initial help message (non-blocking)
	im.outputMutex.Lock()
	// In raw mode, we must use \r\n for proper line endings
	fmt.Print("═══════════════════════════════════════════════════════════════\r\n")
	fmt.Print("Interactive Mode Enabled\r\n")
	fmt.Print("Press 'H' for help, 'Q' to quit\r\n")
	fmt.Print("═══════════════════════════════════════════════════════════════\r\n")
	im.outputMutex.Unlock()

	// Wait for exit signal
	im.WaitForExit()
	return nil
}

// Stop ends interactive mode
func (im *InteractiveModeObserver) Stop() {
	im.mutex.Lock()
	defer im.mutex.Unlock()
	im.shouldExit = true
}

// Implement ExecutionObserverI interface

// OnExecutionStart is called when execution begins
func (im *InteractiveModeObserver) OnExecutionStart(playbookName string, playCount int) {
	im.executionEvents <- ExecutionEvent{
		Type:      "execution_start",
		Message:   fmt.Sprintf("Starting execution of %s (%d plays)", playbookName, playCount),
		Timestamp: time.Now(),
	}

	im.mutex.Lock()
	im.currentExecution = &ExecutionResult{
		ExecutionID:  fmt.Sprintf("exec-%d", time.Now().Unix()),
		PlaybookName: playbookName,
		StartTime:    time.Now(),
		Status:       "running",
	}
	im.mutex.Unlock()
}

// OnPlayStart is called when a play starts
func (im *InteractiveModeObserver) OnPlayStart(playName string, playIndex int, totalPlays int) {
	im.executionEvents <- ExecutionEvent{
		Type:      "play_start",
		PlayName:  playName,
		Message:   fmt.Sprintf("[%d/%d] Starting play: %s", playIndex, totalPlays, playName),
		Timestamp: time.Now(),
	}
}

// OnPlayEnd is called when a play ends
func (im *InteractiveModeObserver) OnPlayEnd(playName string, playIndex int, success bool, duration time.Duration) {
	status := "✓"
	if !success {
		status = "✗"
	}
	im.executionEvents <- ExecutionEvent{
		Type:      "play_end",
		PlayName:  playName,
		Message:   fmt.Sprintf("%s Play %s completed in %v", status, playName, duration.Round(time.Millisecond)),
		Timestamp: time.Now(),
	}
}

// OnTaskStart is called when a task starts
func (im *InteractiveModeObserver) OnTaskStart(taskName string, hostName string) {
	im.executionEvents <- ExecutionEvent{
		Type:      "task_start",
		TaskName:  taskName,
		HostName:  hostName,
		Message:   fmt.Sprintf("  ⟳ Running task '%s' on %s", taskName, hostName),
		Timestamp: time.Now(),
	}
}

// OnTaskEnd is called when a task ends
func (im *InteractiveModeObserver) OnTaskEnd(taskResult *types.TaskResult) {
	if taskResult == nil {
		return
	}

	status := "✓"
	if taskResult.Failed {
		status = "✗"
	} else if taskResult.Changed {
		status = "⟳"
	}

	im.executionEvents <- ExecutionEvent{
		Type:      "task_end",
		TaskName:  taskResult.TaskName,
		HostName:  taskResult.Host,
		Message:   fmt.Sprintf("%s Task '%s' on %s", status, taskResult.TaskName, taskResult.Host),
		Timestamp: time.Now(),
	}

	// Update task stats
	im.mutex.Lock()
	stats, exists := im.taskStats[taskResult.TaskName]
	if !exists {
		stats = &TaskStats{Name: taskResult.TaskName}
		im.taskStats[taskResult.TaskName] = stats
	}

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

	// Update host stats
	hostStats, exists := im.hostStats[taskResult.Host]
	if !exists {
		hostStats = &HostStats{Name: taskResult.Host}
		im.hostStats[taskResult.Host] = hostStats
	}
	hostStats.TaskCount++
	hostStats.LastTaskTime = time.Now()

	im.mutex.Unlock()
}

// OnExecutionEnd is called when execution completes
func (im *InteractiveModeObserver) OnExecutionEnd(result *types.PlaybookResult, duration time.Duration) {
	status := "✓ success"
	if result.Failed {
		status = "✗ failed"
	}

	im.executionEvents <- ExecutionEvent{
		Type:      "execution_end",
		Message:   fmt.Sprintf("Execution completed: %s (%v)", status, duration.Round(time.Millisecond)),
		Timestamp: time.Now(),
	}

	im.mutex.Lock()
	im.currentExecution.Status = "completed"
	if result.Failed {
		im.currentExecution.Status = "failed"
	}
	im.currentExecution.EndTime = time.Now()
	im.currentExecution.Duration = duration
	im.mutex.Unlock()
}

// OnError is called when an error occurs
func (im *InteractiveModeObserver) OnError(taskName string, hostName string, error string) {
	im.executionEvents <- ExecutionEvent{
		Type:      "error",
		TaskName:  taskName,
		HostName:  hostName,
		Message:   fmt.Sprintf("✗ Error in %s on %s: %s", taskName, hostName, error),
		Timestamp: time.Now(),
	}
}

// processEvents processes execution events
func (im *InteractiveModeObserver) processEvents() {
	for event := range im.executionEvents {
		im.mutex.RLock()
		if im.shouldExit {
			im.mutex.RUnlock()
			return
		}
		im.mutex.RUnlock()

		// Display event based on display mode
		im.displayEvent(event)
	}
}

// displayEvent displays an event based on current display mode
func (im *InteractiveModeObserver) displayEvent(event ExecutionEvent) {
	im.mutex.RLock()
	mode := im.mode
	showingOverlay := im.showingOverlay
	im.mutex.RUnlock()

	// Skip printing events while an overlay (help/stats) is showing
	if showingOverlay {
		return
	}

	// Format timestamp
	timeStr := event.Timestamp.Format("15:04:05")

	// Protect stdout from concurrent writes
	im.outputMutex.Lock()
	defer im.outputMutex.Unlock()

	switch mode {
	case DisplayNormal:
		if event.Type == "error" || event.Type == "execution_end" {
			fmt.Printf("[%s] %s\r\n", timeStr, event.Message)
		} else if event.Type == "play_start" {
			fmt.Printf("[%s] %s\r\n", timeStr, event.Message)
		}
	case DisplayVerbose:
		fmt.Printf("[%s] %s\r\n", timeStr, event.Message)
	case DisplayDebug:
		fmt.Printf("[%s] DEBUG %s (type=%s)\r\n", timeStr, event.Message, event.Type)
	}
}

// handleKeyboardInput reads keyboard input and toggles modes
func (im *InteractiveModeObserver) handleKeyboardInput() {
	buf := make([]byte, 1)
	for {
		im.mutex.RLock()
		if im.shouldExit {
			im.mutex.RUnlock()
			return
		}
		im.mutex.RUnlock()

		_, err := os.Stdin.Read(buf)
		if err != nil {
			continue
		}

		im.handleKeypress(buf[0])
	}
}

// handleKeypress processes a single keypress
func (im *InteractiveModeObserver) handleKeypress(key byte) {
	// Protect output with mutex
	im.outputMutex.Lock()

	im.mutex.Lock()
	switch key {
	case 'V', 'v':
		if im.mode == DisplayVerbose {
			im.mode = DisplayNormal
			fmt.Print("\r\n[Mode changed to: NORMAL]\r\n")
		} else {
			im.mode = DisplayVerbose
			fmt.Print("\r\n[Mode changed to: VERBOSE]\r\n")
		}
		im.displayUpdateChan <- im.mode

	case 'D', 'd':
		if im.mode == DisplayDebug {
			im.mode = DisplayNormal
			fmt.Print("\r\n[Mode changed to: NORMAL]\r\n")
		} else {
			im.mode = DisplayDebug
			fmt.Print("\r\n[Mode changed to: DEBUG]\r\n")
		}
		im.displayUpdateChan <- im.mode

	case 'N', 'n':
		if im.mode != DisplayNormal {
			im.mode = DisplayNormal
			fmt.Print("\r\n[Mode changed to: NORMAL]\r\n")
			im.displayUpdateChan <- im.mode
		}

	case 'P', 'p':
		im.paused = !im.paused
		status := "PAUSED"
		if !im.paused {
			status = "RESUMED"
		}
		fmt.Printf("\r\n[Execution %s]\r\n", status)
		im.pausedChan <- im.paused

	case 'S', 's':
		// Note: displayStats will be called after releasing output lock to avoid deadlock
		im.mutex.Unlock()
		im.outputMutex.Unlock()
		im.displayStats()
		return

	case 'H', 'h':
		// Note: printHelp will be called after releasing output lock to avoid deadlock
		im.mutex.Unlock()
		im.outputMutex.Unlock()
		im.printHelp()
		return

	case 'Q', 'q':
		fmt.Print("\r\n[Quit: execution continues in background]\r\n")
		im.shouldExit = true

	case 27: // ESC key
		im.shouldExit = true
	}

	im.mutex.Unlock()
	im.outputMutex.Unlock()
}

// displayStats displays current execution statistics
func (im *InteractiveModeObserver) displayStats() {
	im.mutex.Lock()
	im.showingOverlay = true
	im.mutex.Unlock()

	im.outputMutex.Lock()

	// In raw mode, use \r\n for proper line endings
	fmt.Print("\r\n\r\n")
	fmt.Print("╔════════════════════════════════════════════════════════════════╗\r\n")
	fmt.Print("║            Execution Statistics                               ║\r\n")
	fmt.Print("╠════════════════════════════════════════════════════════════════╣\r\n")

	// Host stats
	fmt.Print("║ Hosts:                                                         ║\r\n")
	im.mutex.RLock()
	for _, stats := range im.hostStats {
		fmt.Printf("║   %s: %d tasks\r\n", stats.Name, stats.TaskCount)
	}

	// Task stats
	fmt.Print("║ Tasks:                                                         ║\r\n")
	for _, stats := range im.taskStats {
		total := stats.Success + stats.Failed + stats.Changed + stats.Skipped
		fmt.Printf("║   %s: %d (✓%d ✗%d ⟳%d ⊝%d)\r\n",
			stats.Name, total, stats.Success, stats.Failed, stats.Changed, stats.Skipped)
	}
	im.mutex.RUnlock()

	fmt.Print("║                                                                ║\r\n")
	fmt.Print("║  Press any key to dismiss this statistics overlay...           ║\r\n")
	fmt.Print("╚════════════════════════════════════════════════════════════════╝\r\n")
	fmt.Print("\r\n")

	im.outputMutex.Unlock()

	// Wait for any key press to dismiss (with 5 second timeout)
	buf := make([]byte, 1)
	done := make(chan bool, 1)

	go func() {
		os.Stdin.Read(buf)
		done <- true
	}()

	select {
	case <-done:
		// Key pressed - dismiss overlay
	case <-time.After(5 * time.Second):
		// Timeout - dismiss overlay
	}

	im.mutex.Lock()
	im.showingOverlay = false
	im.mutex.Unlock()

	im.outputMutex.Lock()
	fmt.Print("\n") // Add spacing after stats
	im.outputMutex.Unlock()
}

// handleDisplayUpdates processes display mode changes
func (im *InteractiveModeObserver) handleDisplayUpdates() {
	for mode := range im.displayUpdateChan {
		im.mutex.RLock()
		showingOverlay := im.showingOverlay
		im.mutex.RUnlock()

		// Skip display updates while an overlay is showing
		if showingOverlay {
			continue
		}

		im.outputMutex.Lock()
		fmt.Printf("\n[Display mode changed to: %v]\n", mode)
		im.outputMutex.Unlock()
	}
}

// printHelp prints keyboard controls help and waits for key press
func (im *InteractiveModeObserver) printHelp() {
	im.mutex.Lock()
	im.showingOverlay = true
	im.mutex.Unlock()

	im.outputMutex.Lock()

	// In raw mode, use \r\n for proper line endings
	fmt.Print("\r\n\r\n")
	fmt.Print("╔═══════════════════════════════════════════════════════════════╗\r\n")
	fmt.Print("║                Interactive Execution Controls                 ║\r\n")
	fmt.Print("╠═══════════════════════════════════════════════════════════════╣\r\n")
	fmt.Print("║                                                               ║\r\n")
	fmt.Print("║  V - Toggle VERBOSE mode        D - Toggle DEBUG mode        ║\r\n")
	fmt.Print("║  N - Return to NORMAL mode      P - Pause/Resume execution   ║\r\n")
	fmt.Print("║  S - Show statistics            H - Show this help           ║\r\n")
	fmt.Print("║  Q/ESC - Quit (keep running)                                 ║\r\n")
	fmt.Print("║                                                               ║\r\n")
	fmt.Print("║  Press any key to dismiss this help overlay...               ║\r\n")
	fmt.Print("║                                                               ║\r\n")
	fmt.Print("╚═══════════════════════════════════════════════════════════════╝\r\n")
	fmt.Print("\r\n")

	im.outputMutex.Unlock()

	// Wait for any key press to dismiss (with 5 second timeout)
	buf := make([]byte, 1)
	done := make(chan bool, 1)

	go func() {
		os.Stdin.Read(buf)
		done <- true
	}()

	select {
	case <-done:
		// Key pressed - dismiss overlay
	case <-time.After(5 * time.Second):
		// Timeout - dismiss overlay
	}

	im.mutex.Lock()
	im.showingOverlay = false
	im.mutex.Unlock()

	im.outputMutex.Lock()
	fmt.Print("\n") // Add spacing after help
	im.outputMutex.Unlock()
}

// WaitForExit blocks until user signals to exit
func (im *InteractiveModeObserver) WaitForExit() {
	for {
		im.mutex.RLock()
		if im.shouldExit {
			im.mutex.RUnlock()
			break
		}
		im.mutex.RUnlock()
		time.Sleep(100 * time.Millisecond)
	}
}

// GetCurrentMode returns the current display mode
func (im *InteractiveModeObserver) GetCurrentMode() DisplayMode {
	im.mutex.RLock()
	defer im.mutex.RUnlock()
	return im.mode
}
