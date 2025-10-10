package utils // import "github.com/onigirazu-cfg/onigirazu/pkg/utils"


CONSTANTS

const (
	Reset = "\033[0m"
	Bold  = "\033[1m"
	Dim   = "\033[2m"

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
    ANSI color codes


VARIABLES

var (
	// Global color scheme
	Colors *ColorScheme
	// Whether colors are enabled
	ColorsEnabled bool = true
)
var (
	SuccessSymbol = "✓"
	ErrorSymbol   = "✗"
	WarningSymbol = "⚠"
	InfoSymbol    = "ℹ"
	ChangedSymbol = "⚡"
	SkippedSymbol = "⊝"
)
    Status symbols with colors


FUNCTIONS

func Colorize(text, colorType string) string
    Colorize applies color to text based on the type

func EnableColors(enable bool)
    EnableColors enables or disables color output

func FormatPlayHeader(playName string, hostCount int) string
    FormatPlayHeader formats a play header with colors

func FormatTaskHeader(taskName, module string) string
    FormatTaskHeader formats a task header with colors

func FormatTaskResult(taskName, host, status string, changed bool) string
    FormatTaskResult formats a task result with colors

func GetStatusSymbol(status string) string
    GetStatusSymbol returns a colored status symbol

func ProgressBar(current, total int, width int) string
    ProgressBar creates a simple progress bar


TYPES

type ColorFunc func(string) string
    ColorFunc represents a function that applies color to text

type ColorScheme struct {
	Success    ColorFunc
	Error      ColorFunc
	Warning    ColorFunc
	Info       ColorFunc
	Debug      ColorFunc
	Changed    ColorFunc
	Skipped    ColorFunc
	Failed     ColorFunc
	Header     ColorFunc
	Highlight  ColorFunc
	Dim        ColorFunc
	Bold       ColorFunc
	TaskName   ColorFunc
	HostName   ColorFunc
	ModuleName ColorFunc
	PlayName   ColorFunc
}
    ColorScheme holds color functions for different types of output

func NewColorScheme() *ColorScheme
    NewColorScheme creates a new color scheme

