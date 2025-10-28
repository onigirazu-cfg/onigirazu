package execution

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// TmuxManager handles tmux session creation and management
type TmuxManager struct {
	sessionName     string
	isTmuxAvailable bool
}

// NewTmuxManager creates a new tmux manager
func NewTmuxManager() *TmuxManager {
	tm := &TmuxManager{
		sessionName: fmt.Sprintf("onigirazu-%d", time.Now().Unix()),
	}
	tm.checkTmuxAvailability()
	return tm
}

// checkTmuxAvailability checks if tmux is installed
func (tm *TmuxManager) checkTmuxAvailability() {
	_, err := exec.LookPath("tmux")
	tm.isTmuxAvailable = err == nil
}

// IsTmuxAvailable returns whether tmux is available
func (tm *TmuxManager) IsTmuxAvailable() bool {
	return tm.isTmuxAvailable
}

// Start initializes a tmux session with dual panes (does NOT attach)
func (tm *TmuxManager) Start() (string, error) {
	if !tm.isTmuxAvailable {
		return "", fmt.Errorf("tmux is not installed. Install it with: brew install tmux (macOS) or apt-get install tmux (Linux)")
	}

	// Create session
	cmd := exec.Command("tmux", "new-session", "-d", "-s", tm.sessionName)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Create vertical split (main pane on left, control pane on right)
	cmd = exec.Command("tmux", "split-window", "-h", "-t", tm.sessionName, "-l", "30")
	if err := cmd.Run(); err != nil {
		tm.Kill()
		return "", fmt.Errorf("failed to split tmux window: %w", err)
	}

	// Set main pane (left) as focus
	cmd = exec.Command("tmux", "select-pane", "-t", tm.sessionName+":0.0")
	cmd.Run() // Ignore error

	// Add help text to right pane
	tm.updateControlPane()

	// Return without attaching - caller will send command and then attach
	return tm.sessionName, nil
}

// Attach connects to the tmux session for user interaction (blocking)
func (tm *TmuxManager) Attach() error {
	if !tm.isTmuxAvailable {
		return fmt.Errorf("tmux not available")
	}

	cmd := exec.Command("tmux", "attach-session", "-t", tm.sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// updateControlPane updates the control panel with instructions
func (tm *TmuxManager) updateControlPane() {
	controlText := `
╔════════════════════════════════════╗
║   Onigirazu Execution Controls    ║
╠════════════════════════════════════╣
║                                    ║
║  Main Window: Execution Progress   ║
║                                    ║
║  Key Bindings:                     ║
║  ─────────────────────────────     ║
║  Ctrl+B then:                      ║
║                                    ║
║    V  → Verbose mode              ║
║    D  → Debug mode                ║
║    N  → Normal mode               ║
║    P  → Pause/Resume              ║
║    E  → Expand errors             ║
║    C  → Collapse errors           ║
║    H  → Show this help            ║
║    Q  → Quit (keep running)       ║
║                                    ║
║  Pane Navigation:                  ║
║    Ctrl+B then O → Next pane       ║
║    Ctrl+B then ; → Last pane       ║
║                                    ║
╚════════════════════════════════════╝
`

	// Write to right pane
	cmd := exec.Command("tmux", "send-keys", "-t", tm.sessionName+":0.1", "cat << 'EOF'\n"+controlText+"\nEOF\n", "Enter")
	cmd.Run() // Ignore error
}

// SendCommand sends a command to the main pane
func (tm *TmuxManager) SendCommand(command string) error {
	if !tm.isTmuxAvailable {
		return fmt.Errorf("tmux not available")
	}

	cmd := exec.Command("tmux", "send-keys", "-t", tm.sessionName+":0.0", command, "Enter")
	return cmd.Run()
}

// Kill closes the tmux session
func (tm *TmuxManager) Kill() error {
	if !tm.isTmuxAvailable {
		return nil
	}

	cmd := exec.Command("tmux", "kill-session", "-t", tm.sessionName)
	return cmd.Run()
}

// GetSessionName returns the tmux session name
func (tm *TmuxManager) GetSessionName() string {
	return tm.sessionName
}

// CheckTmuxInstallation checks if tmux is installed and provides installation instructions
func CheckTmuxInstallation() (bool, string) {
	_, err := exec.LookPath("tmux")
	if err == nil {
		return true, ""
	}

	var instructions string
	if os.Getenv("DARWIN") != "" || strings.Contains(os.Getenv("OSTYPE"), "darwin") {
		instructions = "Installation: brew install tmux"
	} else if _, err := exec.LookPath("apt-get"); err == nil {
		instructions = "Installation: sudo apt-get install tmux"
	} else if _, err := exec.LookPath("yum"); err == nil {
		instructions = "Installation: sudo yum install tmux"
	} else if _, err := exec.LookPath("pacman"); err == nil {
		instructions = "Installation: sudo pacman -S tmux"
	} else {
		instructions = "Installation: Visit https://github.com/tmux/tmux/wiki/Installing"
	}

	return false, instructions
}

// GetFallbackInstructions provides fallback instructions when tmux is not available
func GetFallbackInstructions() string {
	return `
⚠️  Tmux is not installed. Falling back to interactive mode (--interactive).

You can still:
  • Press V to see verbose output
  • Press D to see debug output
  • Press P to pause execution
  • Press H to see help

To install tmux:
  macOS:  brew install tmux
  Linux:  apt-get install tmux

For more info: https://github.com/tmux/tmux/wiki/Installing
`
}
