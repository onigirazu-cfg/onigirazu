package adhoc

import (
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CommandFormat represents the format of the ad-hoc command
type CommandFormat int

const (
	// FormatAnsibleLike represents Ansible-like syntax: -m module key=value
	FormatAnsibleLike CommandFormat = iota
	// FormatNaturalLanguage represents natural language: "install nginx"
	FormatNaturalLanguage
	// FormatJSON represents JSON syntax: {"module":"package","args":{"name":"nginx"}}
	FormatJSON
	// FormatYAML represents YAML syntax
	FormatYAML
	// FormatSimple represents simple shell command: "uptime"
	FormatSimple
)

// Options represents execution options for ad-hoc commands
type Options struct {
	// Execution options
	Check    bool          // Check mode (dry-run)
	Diff     bool          // Show differences
	Timeout  time.Duration // Execution timeout
	Parallel int           // Number of parallel executions

	// Output options
	Output  string // Output format (text, json, yaml, table)
	Verbose bool   // Verbose output
	NoColor bool   // Disable colored output

	// Variables
	Variables map[string]interface{} // Extra variables
}

// Command represents a parsed ad-hoc command
type Command struct {
	Module string                 // Module name
	Args   map[string]interface{} // Module arguments
	Format CommandFormat          // Detected format
	Raw    string                 // Original command string
}

// Result represents the result of ad-hoc command execution
type Result struct {
	Host     types.Host        // Target host
	Task     *types.Task       // Executed task
	Result   *types.TaskResult // Task result
	Duration time.Duration     // Execution duration
	Error    error             // Error if any
}

// Summary represents execution summary
type Summary struct {
	Total    int           // Total hosts
	Success  int           // Successful executions
	Failed   int           // Failed executions
	Changed  int           // Changed hosts
	Skipped  int           // Skipped hosts
	Duration time.Duration // Total duration
	Results  []*Result     // Individual results
}
