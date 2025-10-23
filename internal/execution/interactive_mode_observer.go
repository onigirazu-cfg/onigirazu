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
	taskStats         map[string]*TaskStats
	hostStats         map[string]*HostStats
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
	}
}

// Start begins interactive mode with keyboard input handling
func (im *InteractiveModeObserver) Start() error {
	// Save terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to enable interactive mode: %w", err)
	}

	// NOTE: Signal handling is done by the main signal handler (signal_handler.go)
	// Don't set up duplicate signal handlers here

	// Start keyboard input handler
	go im.handleKeyboardInput()

	// Start event processor
	go im.processEvents()

	// Start display updater
	go im.handleDisplayUpdates()

	// Print help
	im.printHelp()

	// Wait for exit signal, then restore terminal
	im.WaitForExit()
	return term.Restore(int(os.Stdin.Fd()), oldState)
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
	im.mutex.RUnlock()

	// Format timestamp
	timeStr := event.Timestamp.Format("15:04:05")

	switch mode {
	case DisplayNormal:
		if event.Type == "error" || event.Type == "execution_end" {
			fmt.Printf("[%s] %s\n", timeStr, event.Message)
		} else if event.Type == "play_start" {
			fmt.Printf("[%s] %s\n", timeStr, event.Message)
		}
	case DisplayVerbose:
		fmt.Printf("[%s] %s\n", timeStr, event.Message)
	case DisplayDebug:
		fmt.Printf("[%s] DEBUG %s (type=%s)\n", timeStr, event.Message, event.Type)
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
	im.mutex.Lock()
	defer im.mutex.Unlock()

	switch key {
	case 'V', 'v':
		if im.mode == DisplayVerbose {
			im.mode = DisplayNormal
			fmt.Println("\n[Mode changed to: NORMAL]")
		} else {
			im.mode = DisplayVerbose
			fmt.Println("\n[Mode changed to: VERBOSE]")
		}
		im.displayUpdateChan <- im.mode

	case 'D', 'd':
		if im.mode == DisplayDebug {
			im.mode = DisplayNormal
			fmt.Println("\n[Mode changed to: NORMAL]")
		} else {
			im.mode = DisplayDebug
			fmt.Println("\n[Mode changed to: DEBUG]")
		}
		im.displayUpdateChan <- im.mode

	case 'N', 'n':
		if im.mode != DisplayNormal {
			im.mode = DisplayNormal
			fmt.Println("\n[Mode changed to: NORMAL]")
			im.displayUpdateChan <- im.mode
		}

	case 'P', 'p':
		im.paused = !im.paused
		status := "PAUSED"
		if !im.paused {
			status = "RESUMED"
		}
		fmt.Printf("\n[Execution %s]\n", status)
		im.pausedChan <- im.paused

	case 'S', 's':
		im.displayStats()

	case 'H', 'h':
		im.printHelp()

	case 'Q', 'q':
		fmt.Println("\n[Quit: execution continues in background]")
		im.shouldExit = true

	case 27: // ESC key
		im.shouldExit = true
	}
}

// displayStats displays current execution statistics
func (im *InteractiveModeObserver) displayStats() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║ Execution Statistics                                           ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")

	// Host stats
	fmt.Println("║ Hosts:                                                         ║")
	for _, stats := range im.hostStats {
		fmt.Printf("║   %s: %d tasks\n", stats.Name, stats.TaskCount)
	}

	// Task stats
	fmt.Println("║ Tasks:                                                         ║")
	for _, stats := range im.taskStats {
		total := stats.Success + stats.Failed + stats.Changed + stats.Skipped
		fmt.Printf("║   %s: %d (✓%d ✗%d ⟳%d ⊝%d)\n",
			stats.Name, total, stats.Success, stats.Failed, stats.Changed, stats.Skipped)
	}

	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
}

// handleDisplayUpdates processes display mode changes
func (im *InteractiveModeObserver) handleDisplayUpdates() {
	for mode := range im.displayUpdateChan {
		im.mutex.Lock()
		fmt.Printf("\n[Display mode changed to: %v]\n", mode)
		im.mutex.Unlock()
	}
}

// printHelp prints keyboard controls help
func (im *InteractiveModeObserver) printHelp() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║ Interactive Execution Controls                                ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Println("║ V - Toggle VERBOSE mode      D - Toggle DEBUG mode            ║")
	fmt.Println("║ N - Return to NORMAL mode    P - Pause/Resume execution      ║")
	fmt.Println("║ S - Show statistics          H - Show this help              ║")
	fmt.Println("║ Q/ESC - Quit (keep running)                                   ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
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
