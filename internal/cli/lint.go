package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// LintIssue represents a linting issue
type LintIssue struct {
	Severity string // "error", "warning", "info"
	Rule     string
	Message  string
	File     string
	Line     int
	Play     string
	Task     string
}

// LintResult holds all linting issues
type LintResult struct {
	Issues []LintIssue
	Errors int
	Warns  int
	Infos  int
}

func newLintCmd() *cobra.Command {
	var (
		strict     bool
		noWarnings bool
		rules      []string
		skipRules  []string
		recursive  bool
	)

	cmd := &cobra.Command{
		Use:   "lint [file|directory...]",
		Short: "Lint playbook files for errors and best practices",
		Long: `Lint playbook files to check for:
  - YAML syntax errors
  - Playbook structure issues
  - Missing required fields
  - Invalid module usage
  - Security concerns
  - Best practice violations

Exit codes:
  0 - No issues found
  1 - Warnings found (only with --strict)
  2 - Errors found`,
		Example: `  # Lint a single playbook
  onigirazu lint playbook.yml

  # Lint multiple files
  onigirazu lint play1.yml play2.yml

  # Lint all playbooks in directory
  onigirazu lint --recursive playbooks/

  # Strict mode (warnings cause non-zero exit)
  onigirazu lint --strict playbook.yml

  # Only show errors
  onigirazu lint --no-warnings playbook.yml

  # Run specific rules
  onigirazu lint --rules=syntax,required-fields playbook.yml

  # Skip specific rules
  onigirazu lint --skip-rules=task-name playbook.yml`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Collect all files to lint
			var files []string
			for _, arg := range args {
				info, err := os.Stat(arg)
				if err != nil {
					return fmt.Errorf("cannot access %s: %w", arg, err)
				}

				if info.IsDir() {
					if recursive {
						err := filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
							if err != nil {
								return err
							}
							if !info.IsDir() && isYAMLFile(path) {
								files = append(files, path)
							}
							return nil
						})
						if err != nil {
							return fmt.Errorf("error walking directory %s: %w", arg, err)
						}
					} else {
						return fmt.Errorf("%s is a directory (use --recursive to lint directories)", arg)
					}
				} else {
					if !isYAMLFile(arg) {
						fmt.Fprintf(os.Stderr, "Warning: %s does not appear to be a YAML file, skipping\n", arg)
						continue
					}
					files = append(files, arg)
				}
			}

			if len(files) == 0 {
				return fmt.Errorf("no YAML files found to lint")
			}

			// Parse rule filters
			enabledRules := parseRules(rules)
			disabledRules := parseRules(skipRules)

			// Lint all files
			totalResult := &LintResult{}
			for _, file := range files {
				result := lintFile(file, enabledRules, disabledRules)
				totalResult.Issues = append(totalResult.Issues, result.Issues...)
				totalResult.Errors += result.Errors
				totalResult.Warns += result.Warns
				totalResult.Infos += result.Infos
			}

			// Display results
			displayLintResults(totalResult, noWarnings)

			// Determine exit code
			if totalResult.Errors > 0 {
				return fmt.Errorf("linting failed with %d error(s)", totalResult.Errors)
			}
			if strict && totalResult.Warns > 0 {
				return fmt.Errorf("linting failed with %d warning(s) in strict mode", totalResult.Warns)
			}

			// Success message
			if totalResult.Warns == 0 && totalResult.Infos == 0 {
				fmt.Println("✓ All checks passed!")
			} else {
				fmt.Printf("✓ No errors found (%d warning(s), %d info)\n", totalResult.Warns, totalResult.Infos)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Treat warnings as errors")
	cmd.Flags().BoolVar(&noWarnings, "no-warnings", false, "Only show errors")
	cmd.Flags().StringSliceVar(&rules, "rules", nil, "Only run specific rules (comma-separated)")
	cmd.Flags().StringSliceVar(&skipRules, "skip-rules", nil, "Skip specific rules (comma-separated)")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively lint all YAML files in directories")

	return cmd
}

// lintFile performs linting on a single file
func lintFile(filename string, enabledRules, disabledRules map[string]bool) *LintResult {
	result := &LintResult{}

	// Read file
	// #nosec G304 - Reading user-provided playbook files is the intended functionality of the lint command
	data, err := os.ReadFile(filename)
	if err != nil {
		result.addError("file-read", fmt.Sprintf("Cannot read file: %v", err), filename, 0, "", "")
		return result
	}

	// Check YAML syntax
	if shouldRunRule("syntax", enabledRules, disabledRules) {
		var rawYAML interface{}
		if err := yaml.Unmarshal(data, &rawYAML); err != nil {
			result.addError("syntax", fmt.Sprintf("YAML syntax error: %v", err), filename, 0, "", "")
			return result // Cannot continue if YAML is invalid
		}
	}

	// Parse as playbook
	var playbook types.Playbook
	if err := yaml.Unmarshal(data, &playbook); err != nil {
		result.addError("parse", fmt.Sprintf("Cannot parse playbook: %v", err), filename, 0, "", "")
		return result
	}

	// Run linting rules
	if shouldRunRule("required-fields", enabledRules, disabledRules) {
		checkRequiredFields(&playbook, filename, result)
	}
	if shouldRunRule("task-name", enabledRules, disabledRules) {
		checkTaskNames(&playbook, filename, result)
	}
	if shouldRunRule("module-args", enabledRules, disabledRules) {
		checkModuleArgs(&playbook, filename, result)
	}
	if shouldRunRule("deprecated", enabledRules, disabledRules) {
		checkDeprecated(&playbook, filename, result)
	}
	if shouldRunRule("security", enabledRules, disabledRules) {
		checkSecurity(&playbook, filename, result)
	}
	if shouldRunRule("best-practices", enabledRules, disabledRules) {
		checkBestPractices(&playbook, filename, result)
	}

	return result
}

// Linting rule implementations

func checkRequiredFields(playbook *types.Playbook, filename string, result *LintResult) {
	// Check playbook has plays
	if len(playbook.Plays) == 0 {
		result.addError("required-fields", "Playbook must contain at least one play", filename, 0, "", "")
		return
	}

	for i, play := range playbook.Plays {
		playName := play.Name
		if playName == "" {
			playName = fmt.Sprintf("Play #%d", i+1)
		}

		// Check play has name
		if play.Name == "" {
			result.addWarning("required-fields", "Play should have a name", filename, 0, playName, "")
		}

		// Check play has hosts
		if play.Hosts == "" {
			result.addError("required-fields", "Play must specify hosts", filename, 0, playName, "")
		}

		// Check play has tasks
		if len(play.Tasks) == 0 && len(play.PreTasks) == 0 && len(play.PostTasks) == 0 {
			result.addWarning("required-fields", "Play has no tasks", filename, 0, playName, "")
		}
	}
}

func checkTaskNames(playbook *types.Playbook, filename string, result *LintResult) {
	for _, play := range playbook.Plays {
		playName := play.Name
		if playName == "" {
			playName = "unnamed play"
		}

		checkTaskListNames(play.PreTasks, "pre_tasks", playName, filename, result)
		checkTaskListNames(play.Tasks, "tasks", playName, filename, result)
		checkTaskListNames(play.PostTasks, "post_tasks", playName, filename, result)
		checkTaskListNames(play.Handlers, "handlers", playName, filename, result)
	}
}

func checkTaskListNames(tasks []types.Task, section, playName, filename string, result *LintResult) {
	for i, task := range tasks {
		if task.Name == "" {
			taskRef := fmt.Sprintf("%s[%d] (%s)", section, i, task.Module)
			result.addWarning("task-name", "Task should have a name for better readability", filename, 0, playName, taskRef)
		}
	}
}

func checkModuleArgs(playbook *types.Playbook, filename string, result *LintResult) {
	knownModules := map[string][]string{
		// Core modules
		"shell":    {"cmd", "chdir", "creates", "removes"},
		"command":  {"cmd", "chdir", "creates", "removes"},
		"copy":     {"src", "dest", "content", "mode", "owner", "group", "backup"},
		"template": {"src", "dest", "mode", "owner", "group", "backup"},
		"file":     {"path", "state", "mode", "owner", "group", "recurse"},
		"package":  {"name", "state", "version"},
		"service":  {"name", "state", "enabled"},
		"user":     {"name", "state", "uid", "group", "groups", "home", "shell", "password"},
		"group":    {"name", "state", "gid"},
		"debug":    {"msg", "var"},
		"set_fact": {"cacheable"},
		"fetch":    {"src", "dest", "flat"},
		"stat":     {"path", "follow"},
		"get_url":  {"url", "dest", "mode", "checksum"},

		// File search and manipulation
		"find":       {"path", "pattern", "type", "limit"},
		"lineinfile": {"path", "line", "regexp", "insertbefore", "insertafter", "state", "backup"},

		// System modules
		"ping":     {},
		"systemd":  {"name", "state", "enabled", "daemon_reload"},
		"cron":     {"name", "minute", "hour", "day", "month", "weekday", "job", "state", "user", "special_time"},
		"firewall": {"service", "state", "permanent", "immediate"},
		"config":   {"path", "key", "value", "state"},

		// Git module
		"git": {"repo", "dest", "version", "depth", "recursive", "force", "key_file", "accept_hostkey"},

		// Docker/Container modules
		"docker_container": {"name", "image", "state", "command", "ports", "volumes", "env", "networks"},
		"docker_image":     {"name", "state", "build", "path", "pull", "force_pull", "tag"},
		"docker_compose":   {"project_src", "state", "files", "services"},
		"podman":           {"name", "image", "state", "command", "ports", "volumes", "env"},

		// Database modules
		"mysql_db":        {"name", "login_user", "login_password", "login_host", "state", "collation", "encoding"},
		"mysql_user":      {"name", "password", "login_user", "login_password", "login_host", "state", "host", "priv"},
		"postgresql_db":   {"name", "login_user", "login_password", "login_host", "state", "encoding", "locale", "owner"},
		"postgresql_user": {"name", "password", "login_user", "login_password", "login_host", "state", "groups", "role_attr_flags"},
		"mongodb":         {"name", "login_user", "login_password", "login_host", "state", "database"},
	}

	for _, play := range playbook.Plays {
		playName := play.Name
		if playName == "" {
			playName = "unnamed play"
		}

		checkTaskListModules(play.PreTasks, playName, filename, knownModules, result)
		checkTaskListModules(play.Tasks, playName, filename, knownModules, result)
		checkTaskListModules(play.PostTasks, playName, filename, knownModules, result)
		checkTaskListModules(play.Handlers, playName, filename, knownModules, result)
	}
}

func checkTaskListModules(tasks []types.Task, playName, filename string, knownModules map[string][]string, result *LintResult) {
	for _, task := range tasks {
		taskName := task.Name
		if taskName == "" {
			taskName = fmt.Sprintf("unnamed task (%s)", task.Module)
		}

		// Check if module is known
		validArgs, known := knownModules[task.Module]
		if !known {
			result.addInfo("module-args", fmt.Sprintf("Unknown module '%s' - cannot validate arguments", task.Module), filename, 0, playName, taskName)
			continue
		}

		// Check for unknown arguments
		for arg := range task.Args {
			found := false
			for _, validArg := range validArgs {
				if arg == validArg {
					found = true
					break
				}
			}
			if !found && len(validArgs) > 0 {
				result.addWarning("module-args", fmt.Sprintf("Unknown argument '%s' for module '%s'", arg, task.Module), filename, 0, playName, taskName)
			}
		}

		// Module-specific validations
		switch task.Module {
		case "copy":
			if _, hasSrc := task.Args["src"]; !hasSrc {
				if _, hasContent := task.Args["content"]; !hasContent {
					result.addError("module-args", "copy module requires either 'src' or 'content'", filename, 0, playName, taskName)
				}
			}
			if _, hasDest := task.Args["dest"]; !hasDest {
				result.addError("module-args", "copy module requires 'dest' argument", filename, 0, playName, taskName)
			}

		case "template":
			if _, hasSrc := task.Args["src"]; !hasSrc {
				result.addError("module-args", "template module requires 'src' argument", filename, 0, playName, taskName)
			}
			if _, hasDest := task.Args["dest"]; !hasDest {
				result.addError("module-args", "template module requires 'dest' argument", filename, 0, playName, taskName)
			}

		case "file":
			if _, hasPath := task.Args["path"]; !hasPath {
				result.addError("module-args", "file module requires 'path' argument", filename, 0, playName, taskName)
			}

		case "package", "service", "user", "group":
			if _, hasName := task.Args["name"]; !hasName {
				result.addError("module-args", fmt.Sprintf("%s module requires 'name' argument", task.Module), filename, 0, playName, taskName)
			}

		case "debug":
			if _, hasMsg := task.Args["msg"]; !hasMsg {
				if _, hasVar := task.Args["var"]; !hasVar {
					result.addWarning("module-args", "debug module should have either 'msg' or 'var'", filename, 0, playName, taskName)
				}
			}

		case "stat":
			if _, hasPath := task.Args["path"]; !hasPath {
				result.addError("module-args", "stat module requires 'path' argument", filename, 0, playName, taskName)
			}

		case "get_url":
			if _, hasURL := task.Args["url"]; !hasURL {
				result.addError("module-args", "get_url module requires 'url' argument", filename, 0, playName, taskName)
			}
			if _, hasDest := task.Args["dest"]; !hasDest {
				result.addError("module-args", "get_url module requires 'dest' argument", filename, 0, playName, taskName)
			}

		case "find":
			// find module doesn't require specific args (uses defaults), but warn if both path and pattern are missing
			if _, hasPath := task.Args["path"]; !hasPath {
				if _, hasPattern := task.Args["pattern"]; !hasPattern {
					result.addInfo("module-args", "find module with no 'path' or 'pattern' will search current directory with '*' pattern", filename, 0, playName, taskName)
				}
			}

		case "lineinfile":
			if _, hasPath := task.Args["path"]; !hasPath {
				result.addError("module-args", "lineinfile module requires 'path' argument", filename, 0, playName, taskName)
			}
			if _, hasLine := task.Args["line"]; !hasLine {
				if _, hasRegexp := task.Args["regexp"]; !hasRegexp {
					result.addWarning("module-args", "lineinfile module should have either 'line' or 'regexp'", filename, 0, playName, taskName)
				}
			}

		case "git":
			if _, hasRepo := task.Args["repo"]; !hasRepo {
				result.addError("module-args", "git module requires 'repo' argument", filename, 0, playName, taskName)
			}
			if _, hasDest := task.Args["dest"]; !hasDest {
				result.addError("module-args", "git module requires 'dest' argument", filename, 0, playName, taskName)
			}

		case "cron":
			if _, hasName := task.Args["name"]; !hasName {
				result.addError("module-args", "cron module requires 'name' argument", filename, 0, playName, taskName)
			}
			if _, hasJob := task.Args["job"]; !hasJob {
				if stateVal, hasState := task.Args["state"]; hasState {
					state := stateVal
					if state != "absent" {
						result.addWarning("module-args", "cron module should have 'job' argument when state is not 'absent'", filename, 0, playName, taskName)
					}
				}
			}

		case "docker_container":
			if _, hasName := task.Args["name"]; !hasName {
				result.addError("module-args", "docker_container module requires 'name' argument", filename, 0, playName, taskName)
			}
			if _, hasImage := task.Args["image"]; !hasImage {
				if stateVal, hasState := task.Args["state"]; hasState {
					state := stateVal
					if state != "absent" {
						result.addWarning("module-args", "docker_container module should have 'image' argument when state is not 'absent'", filename, 0, playName, taskName)
					}
				}
			}

		case "docker_image":
			if _, hasName := task.Args["name"]; !hasName {
				result.addError("module-args", "docker_image module requires 'name' argument", filename, 0, playName, taskName)
			}

		case "mysql_db", "postgresql_db":
			if _, hasName := task.Args["name"]; !hasName {
				result.addError("module-args", fmt.Sprintf("%s module requires 'name' argument", task.Module), filename, 0, playName, taskName)
			}

		case "mysql_user", "postgresql_user":
			if _, hasName := task.Args["name"]; !hasName {
				result.addError("module-args", fmt.Sprintf("%s module requires 'name' argument", task.Module), filename, 0, playName, taskName)
			}
		}
	}
}

func checkDeprecated(playbook *types.Playbook, filename string, result *LintResult) {
	// Check for deprecated syntax or features
	for _, play := range playbook.Plays {
		playName := play.Name
		if playName == "" {
			playName = "unnamed play"
		}

		// Check all task lists
		allTasks := append([]types.Task{}, play.PreTasks...)
		allTasks = append(allTasks, play.Tasks...)
		allTasks = append(allTasks, play.PostTasks...)
		allTasks = append(allTasks, play.Handlers...)

		for _, task := range allTasks {
			taskName := task.Name
			if taskName == "" {
				taskName = fmt.Sprintf("unnamed task (%s)", task.Module)
			}

			// Warn about using 'include' (deprecated in favor of 'include_tasks')
			if task.Include != "" {
				result.addWarning("deprecated", "Using 'include' is deprecated, consider using 'include_tasks' or 'import_tasks'", filename, 0, playName, taskName)
			}
		}
	}
}

func checkSecurity(playbook *types.Playbook, filename string, result *LintResult) {
	for _, play := range playbook.Plays {
		playName := play.Name
		if playName == "" {
			playName = "unnamed play"
		}

		allTasks := append([]types.Task{}, play.PreTasks...)
		allTasks = append(allTasks, play.Tasks...)
		allTasks = append(allTasks, play.PostTasks...)
		allTasks = append(allTasks, play.Handlers...)

		for _, task := range allTasks {
			taskName := task.Name
			if taskName == "" {
				taskName = fmt.Sprintf("unnamed task (%s)", task.Module)
			}

			// Check for shell/command modules without proper safeguards
			if task.Module == "shell" || task.Module == "command" {
				if task.When == "" && !task.IgnoreErrors {
					result.addInfo("security", fmt.Sprintf("Consider using 'when' condition or 'creates'/'removes' with %s module", task.Module), filename, 0, playName, taskName)
				}

				// Check for potentially dangerous commands
				if cmd, ok := task.Args["cmd"].(string); ok {
					if strings.Contains(cmd, "rm -rf") {
						result.addWarning("security", "Potentially dangerous command 'rm -rf' detected", filename, 0, playName, taskName)
					}
					if strings.Contains(cmd, "curl") && strings.Contains(cmd, "bash") {
						result.addWarning("security", "Piping curl to bash is potentially dangerous", filename, 0, playName, taskName)
					}
				}
			}

			// Check for hardcoded passwords
			for key, value := range task.Args {
				if strings.Contains(strings.ToLower(key), "password") {
					if strVal, ok := value.(string); ok && strVal != "" && !strings.HasPrefix(strVal, "{{") {
						result.addWarning("security", "Hardcoded password detected, consider using variables or vault", filename, 0, playName, taskName)
					}
				}
			}

			// Check for become without become_user
			if task.Become && task.BecomeUser == "" {
				result.addInfo("security", "Using 'become' without specifying 'become_user' (defaults to root)", filename, 0, playName, taskName)
			}
		}
	}
}

func checkBestPractices(playbook *types.Playbook, filename string, result *LintResult) {
	// Check playbook has a name
	if playbook.Name == "" {
		result.addInfo("best-practices", "Playbook should have a name", filename, 0, "", "")
	}

	for _, play := range playbook.Plays {
		playName := play.Name
		if playName == "" {
			playName = "unnamed play"
		}

		allTasks := append([]types.Task{}, play.PreTasks...)
		allTasks = append(allTasks, play.Tasks...)
		allTasks = append(allTasks, play.PostTasks...)

		for _, task := range allTasks {
			taskName := task.Name
			if taskName == "" {
				taskName = fmt.Sprintf("unnamed task (%s)", task.Module)
			}

			// Check loop syntax
			if task.Loop != nil {
				if len(task.Loop.Items) == 0 && task.Loop.Range == "" {
					result.addWarning("best-practices", "Loop defined but has no items or range", filename, 0, playName, taskName)
				}
				if task.Loop.Range != "" {
					// Validate range syntax (should be like "1-100" or "start-end")
					if !isValidRange(task.Loop.Range) {
						result.addWarning("best-practices", fmt.Sprintf("Loop range '%s' may have invalid format, expected 'start-end' (e.g., '1-100')", task.Loop.Range), filename, 0, playName, taskName)
					}
				}
			}

			// Recommend using package module instead of apt/yum
			if task.Module == "apt" || task.Module == "yum" || task.Module == "dnf" {
				result.addInfo("best-practices", fmt.Sprintf("Consider using 'package' module instead of '%s' for better portability", task.Module), filename, 0, playName, taskName)
			}

			// Check for tasks that should use changed_when
			if task.Module == "shell" || task.Module == "command" {
				if task.ChangedWhen == "" && task.Register == "" {
					result.addInfo("best-practices", "Consider using 'changed_when' with shell/command modules", filename, 0, playName, taskName)
				}
			}

			// Recommend handlers for service restarts
			if task.Module == "service" {
				if state, ok := task.Args["state"].(string); ok && (state == "restarted" || state == "reloaded") {
					if len(task.Notify) == 0 {
						result.addInfo("best-practices", "Consider using handlers for service restarts", filename, 0, playName, taskName)
					}
				}
			}
		}
	}
}

// Helper functions

func (r *LintResult) addError(rule, message, file string, line int, play, task string) {
	r.Issues = append(r.Issues, LintIssue{
		Severity: "error",
		Rule:     rule,
		Message:  message,
		File:     file,
		Line:     line,
		Play:     play,
		Task:     task,
	})
	r.Errors++
}

func (r *LintResult) addWarning(rule, message, file string, line int, play, task string) {
	r.Issues = append(r.Issues, LintIssue{
		Severity: "warning",
		Rule:     rule,
		Message:  message,
		File:     file,
		Line:     line,
		Play:     play,
		Task:     task,
	})
	r.Warns++
}

func (r *LintResult) addInfo(rule, message, file string, line int, play, task string) {
	r.Issues = append(r.Issues, LintIssue{
		Severity: "info",
		Rule:     rule,
		Message:  message,
		File:     file,
		Line:     line,
		Play:     play,
		Task:     task,
	})
	r.Infos++
}

func displayLintResults(result *LintResult, noWarnings bool) {
	if len(result.Issues) == 0 {
		return
	}

	// Group issues by file
	fileIssues := make(map[string][]LintIssue)
	for _, issue := range result.Issues {
		if noWarnings && issue.Severity == "warning" {
			continue
		}
		fileIssues[issue.File] = append(fileIssues[issue.File], issue)
	}

	// Display issues
	for file, issues := range fileIssues {
		fmt.Printf("\n%s:\n", file)
		for _, issue := range issues {
			icon := "ℹ"
			color := ""
			switch issue.Severity {
			case "error":
				icon = "✗"
				color = "\033[31m" // Red
			case "warning":
				icon = "⚠"
				color = "\033[33m" // Yellow
			case "info":
				icon = "ℹ"
				color = "\033[36m" // Cyan
			}

			location := ""
			if issue.Play != "" {
				location = fmt.Sprintf(" [%s", issue.Play)
				if issue.Task != "" {
					location += fmt.Sprintf(" → %s", issue.Task)
				}
				location += "]"
			}

			fmt.Printf("  %s%s [%s]%s %s\033[0m\n", color, icon, issue.Rule, location, issue.Message)
		}
	}

	// Summary
	fmt.Printf("\n")
	if result.Errors > 0 {
		fmt.Printf("\033[31m✗ %d error(s)\033[0m", result.Errors)
	}
	if result.Warns > 0 {
		if result.Errors > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("\033[33m%d warning(s)\033[0m", result.Warns)
	}
	if result.Infos > 0 {
		if result.Errors > 0 || result.Warns > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("\033[36m%d info\033[0m", result.Infos)
	}
	fmt.Printf("\n")
}

func parseRules(rules []string) map[string]bool {
	if len(rules) == 0 {
		return nil
	}
	result := make(map[string]bool)
	for _, rule := range rules {
		result[strings.TrimSpace(rule)] = true
	}
	return result
}

func shouldRunRule(rule string, enabledRules, disabledRules map[string]bool) bool {
	// If rule is explicitly disabled, don't run it
	if disabledRules != nil && disabledRules[rule] {
		return false
	}
	// If specific rules are enabled, only run those
	if enabledRules != nil {
		return enabledRules[rule]
	}
	// Otherwise, run all rules
	return true
}

// isValidRange validates loop range syntax (e.g., "1-100", "0-50")
func isValidRange(rangeStr string) bool {
	parts := strings.Split(strings.TrimSpace(rangeStr), "-")
	if len(parts) != 2 {
		return false
	}

	// Check if both parts are numeric
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return false
		}
		// Check if it's a valid number
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}
