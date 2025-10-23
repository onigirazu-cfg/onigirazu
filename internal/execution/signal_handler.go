package execution

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// SignalHandler manages graceful shutdown with user confirmation and cleanup
type SignalHandler struct {
	ctx              context.Context
	cancel           context.CancelFunc
	sigChan          chan os.Signal
	shutdownTimeout  time.Duration
	interruptCount   atomic.Int32
	onConfirmCancel  func(saveState bool) error
	onForceCancel    func() error
	gracefulShutdown bool
}

// NewSignalHandler creates a new signal handler with specified timeout
func NewSignalHandler(ctx context.Context, shutdownTimeout time.Duration) *SignalHandler {
	newCtx, cancel := context.WithCancel(ctx)

	sh := &SignalHandler{
		ctx:             newCtx,
		cancel:          cancel,
		sigChan:         make(chan os.Signal, 2), // Buffer for multiple signals
		shutdownTimeout: shutdownTimeout,
	}

	// Start signal listener
	signal.Notify(sh.sigChan, syscall.SIGINT, syscall.SIGTERM)
	go sh.handleSignals()

	return sh
}

// SetCancelCallbacks sets the callbacks for confirmed and forced cancellation
func (sh *SignalHandler) SetCancelCallbacks(
	onConfirm func(saveState bool) error,
	onForce func() error,
) {
	sh.onConfirmCancel = onConfirm
	sh.onForceCancel = onForce
}

// handleSignals processes incoming signals
func (sh *SignalHandler) handleSignals() {
	for sig := range sh.sigChan {
		count := sh.interruptCount.Add(1)

		switch count {
		case 1:
			// First Ctrl+C: Show confirmation prompt
			sh.handleFirstInterrupt(sig)
		case 2:
			// Second Ctrl+C: Force immediate shutdown
			sh.handleForceInterrupt(sig)
		}
	}
}

// handleFirstInterrupt prompts user for confirmation
func (sh *SignalHandler) handleFirstInterrupt(sig os.Signal) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║         Interrupt Signal Received (Ctrl+C)              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Currently running tasks will be interrupted.")
	fmt.Println()

	// Ask about state saving
	saveState := sh.promptSaveState()

	fmt.Println()
	fmt.Println("Starting graceful shutdown (max 10 seconds)...")
	fmt.Println("Press Ctrl+C again to force immediate termination.")
	fmt.Println()

	sh.gracefulShutdown = true
	sh.cancel()

	// Call the confirmation callback
	if sh.onConfirmCancel != nil {
		if err := sh.onConfirmCancel(saveState); err != nil {
			fmt.Fprintf(os.Stderr, "Error during graceful shutdown: %v\n", err)
		}
	}

	// Start timeout for graceful shutdown
	go sh.shutdownTimeout_timer()
}

// handleForceInterrupt forces immediate shutdown
func (sh *SignalHandler) handleForceInterrupt(sig os.Signal) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║     Force Shutdown Initiated (Second Ctrl+C)            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("⚠️  Terminating immediately - tasks may be interrupted!")
	fmt.Println()

	// Call the force cancel callback
	if sh.onForceCancel != nil {
		_ = sh.onForceCancel()
	}

	// Force exit
	os.Exit(130) // Standard exit code for SIGINT
}

// shutdownTimeout_timer enforces the shutdown timeout
func (sh *SignalHandler) shutdownTimeout_timer() {
	<-time.After(sh.shutdownTimeout)

	// If we're still running after timeout, force shutdown
	if sh.gracefulShutdown {
		fmt.Println()
		fmt.Println("⚠️  Graceful shutdown timeout exceeded - forcing termination!")
		fmt.Println()
		if sh.onForceCancel != nil {
			_ = sh.onForceCancel()
		}
		os.Exit(143) // Standard exit code for SIGTERM timeout
	}
}

// promptSaveState prompts user whether to save state before exiting
func (sh *SignalHandler) promptSaveState() bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Save state/audit data before exiting? [Y/n]: ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "" || response == "y" || response == "yes" {
			fmt.Println("✓ State will be saved")
			return true
		} else if response == "n" || response == "no" {
			fmt.Println("✓ State will be discarded")
			return false
		} else {
			fmt.Println("Please answer 'y' or 'n'")
		}
	}
}

// Context returns the cancellable context for the signal handler
func (sh *SignalHandler) Context() context.Context {
	return sh.ctx
}

// Cancel manually cancels the context
func (sh *SignalHandler) Cancel() {
	sh.cancel()
}

// Close closes the signal handler and releases resources
func (sh *SignalHandler) Close() {
	signal.Stop(sh.sigChan)
	close(sh.sigChan)
}

// IsGracefulShutdown returns whether a graceful shutdown was initiated
func (sh *SignalHandler) IsGracefulShutdown() bool {
	return sh.gracefulShutdown
}

// InterruptCount returns the number of interrupt signals received
func (sh *SignalHandler) InterruptCount() int32 {
	return sh.interruptCount.Load()
}
