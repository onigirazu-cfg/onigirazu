package execution

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/term"
)

// InteractiveMode handles interactive execution with real-time verbosity toggle
type InteractiveMode struct {
	mode              DisplayMode
	useColors         bool
	currentExecution  *ExecutionResult
	taskBuffer        chan TaskResult
	errorBuffer       chan TaskError
	mutex             sync.RWMutex
	shouldExit        bool
	paused            bool
	pausedChan        chan bool
	displayUpdateChan chan DisplayMode
}

// TaskError represents a task execution error
type TaskError struct {
	TaskName string
	Hostname string
	Error    string
	ExitCode int
	Time     time.Time
}

// NewInteractiveMode creates a new interactive mode handler
func NewInteractiveMode(useColors bool) *InteractiveMode {
	return &InteractiveMode{
		mode:              DisplayNormal,
		useColors:         useColors,
		currentExecution:  &ExecutionResult{},
		taskBuffer:        make(chan TaskResult, 100),
		errorBuffer:       make(chan TaskError, 100),
		pausedChan:        make(chan bool, 1),
		displayUpdateChan: make(chan DisplayMode, 1),
	}
}

// Start begins interactive mode with keyboard input handling
func (im *InteractiveMode) Start() {
	// Save terminal state
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enable interactive mode: %v\n", err)
		return
	}
	defer func() {
		term.Restore(int(os.Stdin.Fd()), oldState)
	}()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		im.Stop()
	}()

	// Start keyboard input handler
	go im.handleKeyboardInput()

	// Start display updater
	go im.handleDisplayUpdates()

	// Print help
	im.printHelp()
}

// Stop ends interactive mode
func (im *InteractiveMode) Stop() {
	im.mutex.Lock()
	defer im.mutex.Unlock()
	im.shouldExit = true
}

// handleKeyboardInput reads keyboard input and toggles modes
func (im *InteractiveMode) handleKeyboardInput() {
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
func (im *InteractiveMode) handleKeypress(key byte) {
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

	case 'E', 'e':
		fmt.Println("\n[Expanding error details...]")
		// Trigger full error display

	case 'C', 'c':
		fmt.Println("\n[Collapsing error details...]")
		// Trigger collapsed error display

	case 'H', 'h':
		im.printHelp()

	case 'Q', 'q':
		fmt.Println("\n[Quit: execution continues in background]")
		im.shouldExit = true

	case 27: // ESC key - also exit
		im.shouldExit = true
	}
}

// handleDisplayUpdates processes display mode changes
func (im *InteractiveMode) handleDisplayUpdates() {
	for {
		select {
		case mode := <-im.displayUpdateChan:
			im.mutex.RLock()
			if im.currentExecution != nil && im.currentExecution.ExecutionID != "" {
				displayer := NewDisplayer(mode, im.useColors)
				displayer.DisplayExecution(im.currentExecution)
				im.printControlsFooter()
			}
			im.mutex.RUnlock()
		}
	}
}

// UpdateExecution updates the current execution result
func (im *InteractiveMode) UpdateExecution(result *ExecutionResult) {
	im.mutex.Lock()
	defer im.mutex.Unlock()
	im.currentExecution = result
}

// AddTaskResult adds a completed task result
func (im *InteractiveMode) AddTaskResult(task TaskResult) {
	select {
	case im.taskBuffer <- task:
	default:
		// Buffer full, drop oldest
		select {
		case <-im.taskBuffer:
			im.taskBuffer <- task
		default:
		}
	}
}

// AddError adds an error result
func (im *InteractiveMode) AddError(err TaskError) {
	select {
	case im.errorBuffer <- err:
	default:
		// Buffer full, drop oldest
		select {
		case <-im.errorBuffer:
			im.errorBuffer <- err
		default:
		}
	}
}

// IsPaused returns whether execution is paused
func (im *InteractiveMode) IsPaused() bool {
	im.mutex.RLock()
	defer im.mutex.RUnlock()
	return im.paused
}

// printHelp prints keyboard controls help
func (im *InteractiveMode) printHelp() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║ Interactive Execution Controls                                ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Println("║ V - Toggle VERBOSE mode      D - Toggle DEBUG mode            ║")
	fmt.Println("║ N - Return to NORMAL mode    P - Pause/Resume execution      ║")
	fmt.Println("║ E - Expand errors            C - Collapse errors             ║")
	fmt.Println("║ H - Show this help           Q/ESC - Quit (keep running)    ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	im.printControlsFooter()
}

// printControlsFooter prints the control hint at the bottom
func (im *InteractiveMode) printControlsFooter() {
	im.mutex.RLock()
	modeStr := "NORMAL"
	switch im.mode {
	case DisplayVerbose:
		modeStr = "VERBOSE"
	case DisplayDebug:
		modeStr = "DEBUG"
	}
	im.mutex.RUnlock()

	fmt.Printf("\n[Mode: %s | Press V/D/N to change | H for help]\n", modeStr)
}

// WaitForExit blocks until user signals to exit
func (im *InteractiveMode) WaitForExit() {
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
func (im *InteractiveMode) GetCurrentMode() DisplayMode {
	im.mutex.RLock()
	defer im.mutex.RUnlock()
	return im.mode
}
