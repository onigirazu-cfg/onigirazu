package utils

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// ANSI color codes
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"

	// Foreground colors
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	// Bright foreground colors
	BrightBlack   = "\033[90m"
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// Background colors
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// ColorFunc represents a function that applies color to text
type ColorFunc func(string) string

// ColorScheme holds color functions for different types of output
type ColorScheme struct {
	Success     ColorFunc
	Error       ColorFunc
	Warning     ColorFunc
	Info        ColorFunc
	Debug       ColorFunc
	Changed     ColorFunc
	Skipped     ColorFunc
	Failed      ColorFunc
	Header      ColorFunc
	Highlight   ColorFunc
	Dim         ColorFunc
	Bold        ColorFunc
	TaskName    ColorFunc
	HostName    ColorFunc
	ModuleName  ColorFunc
	PlayName    ColorFunc
}

var (
	// Global color scheme
	Colors *ColorScheme
	// Whether colors are enabled
	ColorsEnabled bool = true
)

// init initializes the color scheme
func init() {
	Colors = NewColorScheme()
	ColorsEnabled = shouldEnableColors()
}

// NewColorScheme creates a new color scheme
func NewColorScheme() *ColorScheme {
	return &ColorScheme{
		Success:    makeColorFunc(Green),
		Error:      makeColorFunc(Red),
		Warning:    makeColorFunc(Yellow),
		Info:       makeColorFunc(Blue),
		Debug:      makeColorFunc(BrightBlack),
		Changed:    makeColorFunc(Yellow),
		Skipped:    makeColorFunc(Cyan),
		Failed:     makeColorFunc(Red),
		Header:     makeColorFunc(Bold + Blue),
		Highlight:  makeColorFunc(BrightWhite),
		Dim:        makeColorFunc(Dim),
		Bold:       makeColorFunc(Bold),
		TaskName:   makeColorFunc(BrightCyan),
		HostName:   makeColorFunc(BrightGreen),
		ModuleName: makeColorFunc(BrightMagenta),
		PlayName:   makeColorFunc(Bold + BrightBlue),
	}
}

// makeColorFunc creates a color function
func makeColorFunc(colorCode string) ColorFunc {
	return func(text string) string {
		if !ColorsEnabled {
			return text
		}
		return colorCode + text + Reset
	}
}

// shouldEnableColors determines if colors should be enabled
func shouldEnableColors() bool {
	// Check if NO_COLOR environment variable is set
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if FORCE_COLOR is set
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	// On Windows, colors might not work well in older terminals
	if runtime.GOOS == "windows" {
		// Check if we're in a modern terminal that supports colors
		term := os.Getenv("TERM")
		if term == "" {
			return false
		}
	}

	// Check if stdout is a terminal
	return isTerminal()
}

// isTerminal checks if stdout is connected to a terminal
func isTerminal() bool {
	// Simple check - in a real implementation you might want to use
	// a library like golang.org/x/term
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// EnableColors enables or disables color output
func EnableColors(enable bool) {
	ColorsEnabled = enable
}

// Colorize applies color to text based on the type
func Colorize(text, colorType string) string {
	if !ColorsEnabled {
		return text
	}

	switch strings.ToLower(colorType) {
	case "success", "ok":
		return Colors.Success(text)
	case "error", "failed":
		return Colors.Error(text)
	case "warning", "warn":
		return Colors.Warning(text)
	case "info":
		return Colors.Info(text)
	case "debug":
		return Colors.Debug(text)
	case "changed":
		return Colors.Changed(text)
	case "skipped":
		return Colors.Skipped(text)
	case "header":
		return Colors.Header(text)
	case "highlight":
		return Colors.Highlight(text)
	case "dim":
		return Colors.Dim(text)
	case "bold":
		return Colors.Bold(text)
	case "task":
		return Colors.TaskName(text)
	case "host":
		return Colors.HostName(text)
	case "module":
		return Colors.ModuleName(text)
	case "play":
		return Colors.PlayName(text)
	default:
		return text
	}
}

// Status symbols with colors
var (
	SuccessSymbol = "✓"
	ErrorSymbol   = "✗"
	WarningSymbol = "⚠"
	InfoSymbol    = "ℹ"
	ChangedSymbol = "⚡"
	SkippedSymbol = "⊝"
)

// GetStatusSymbol returns a colored status symbol
func GetStatusSymbol(status string) string {
	switch strings.ToLower(status) {
	case "success", "ok":
		return Colors.Success(SuccessSymbol)
	case "error", "failed":
		return Colors.Error(ErrorSymbol)
	case "warning", "warn":
		return Colors.Warning(WarningSymbol)
	case "info":
		return Colors.Info(InfoSymbol)
	case "changed":
		return Colors.Changed(ChangedSymbol)
	case "skipped":
		return Colors.Skipped(SkippedSymbol)
	default:
		return status
	}
}

// FormatTaskResult formats a task result with colors
func FormatTaskResult(taskName, host, status string, changed bool) string {
	symbol := GetStatusSymbol(status)
	hostColored := Colors.HostName(host)
	taskColored := Colors.TaskName(taskName)

	result := fmt.Sprintf("%s [%s] %s", symbol, hostColored, taskColored)

	if changed && status == "success" {
		result += " " + Colors.Changed("(changed)")
	}

	return result
}

// FormatPlayHeader formats a play header with colors
func FormatPlayHeader(playName string, hostCount int) string {
	playColored := Colors.PlayName(playName)
	hostInfo := Colors.Dim(fmt.Sprintf("(%d hosts)", hostCount))
	return fmt.Sprintf("\n%s %s %s", Colors.Header("PLAY"), playColored, hostInfo)
}

// FormatTaskHeader formats a task header with colors
func FormatTaskHeader(taskName, module string) string {
	taskColored := Colors.TaskName(taskName)
	moduleColored := Colors.ModuleName(fmt.Sprintf("[%s]", module))
	return fmt.Sprintf("\n%s %s %s", Colors.Header("TASK"), taskColored, moduleColored)
}

// ProgressBar creates a simple progress bar
func ProgressBar(current, total int, width int) string {
	if !ColorsEnabled || width <= 0 {
		return fmt.Sprintf("[%d/%d]", current, total)
	}

	percentage := float64(current) / float64(total)
	filled := int(percentage * float64(width))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("[%s%s%s] %d/%d (%.1f%%)",
		BrightGreen, bar, Reset, current, total, percentage*100)
}
