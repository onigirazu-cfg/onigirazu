package modules

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CronModule implements cron job management
type CronModule struct {
	*BaseExecutorModule
}

// NewCronModule creates a new cron module
func NewCronModule() *CronModule {
	return &CronModule{
		BaseExecutorModule: NewBaseExecutorModule("cron"),
	}
}

// GetDescription returns the module description
func (m *CronModule) GetDescription() string {
	return "Manage cron jobs and crontab files"
}

// Execute manages cron operations
func (m *CronModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "cron",
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Use WithExecutor to get fresh executor for this host
	var execResult types.TaskResult = result

	execErr := m.WithExecutor(host, func(exec *executor.CommandExecutor) error {
		var err error
		execResult, err = m.executeCron(ctx, exec, host, args)
		return err
	})

	if execErr != nil {
		return result, execErr
	}

	return execResult, nil
}

// executeCron performs the actual cron operations
func (m *CronModule) executeCron(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	result := types.TaskResult{
		TaskName:  "cron",
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	// Get operation type
	operation := getStringArg(args, "operation", "job")

	switch operation {
	case "job":
		return m.handleJob(ctx, exec, host, args, result)
	case "file":
		return m.handleFile(ctx, exec, host, args, result)
	case "system":
		return m.handleSystem(ctx, exec, host, args, result)
	case "list":
		return m.handleList(ctx, exec, host, args, result)
	default:
		return m.failResult(result, fmt.Sprintf("unknown operation: %s", operation))
	}
}

// handleJob manages individual cron jobs in user crontab
func (m *CronModule) handleJob(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	name := getStringArg(args, "name", "")
	job := getStringArg(args, "job", "")
	minute := getStringArg(args, "minute", "*")
	hour := getStringArg(args, "hour", "*")
	day := getStringArg(args, "day", "*")
	month := getStringArg(args, "month", "*")
	weekday := getStringArg(args, "weekday", "*")
	user := getStringArg(args, "user", "root")
	state := getStringArg(args, "state", "present")
	special_time := getStringArg(args, "special_time", "")

	if name == "" {
		return m.failResult(result, "name parameter is required")
	}

	changed := false

	// Get current crontab
	currentCrontab, err := m.getCrontab(exec, user)
	if err != nil && !strings.Contains(err.Error(), "no crontab") {
		return m.failResult(result, fmt.Sprintf("failed to get crontab: %v", err))
	}

	// Parse current jobs
	jobs := m.parseCrontab(currentCrontab)

	if state == "present" {
		if job == "" {
			return m.failResult(result, "job parameter is required when state is present")
		}

		// Build cron line
		var cronLine string
		if special_time != "" {
			cronLine = fmt.Sprintf("@%s %s", special_time, job)
		} else {
			cronLine = fmt.Sprintf("%s %s %s %s %s %s", minute, hour, day, month, weekday, job)
		}

		// Check if job already exists
		existingJob, exists := jobs[name]
		if !exists || existingJob != cronLine {
			jobs[name] = cronLine
			changed = true
			result.Output["action"] = "job_added"
		}
	} else if state == "absent" {
		if _, exists := jobs[name]; exists {
			delete(jobs, name)
			changed = true
			result.Output["action"] = "job_removed"
		}
	}

	// Write crontab if changed; check mode stops here
	if changed && !inCheckMode(args) {
		newCrontab := m.buildCrontab(jobs)
		if err := m.setCrontab(exec, user, newCrontab); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to set crontab: %v", err))
		}
	}

	result.Changed = changed
	result.Output["jobs_count"] = len(jobs)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleFile manages crontab files directly
func (m *CronModule) handleFile(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	user := getStringArg(args, "user", "root")
	content := getStringArg(args, "content", "")
	backup := getBoolArg(args, "backup", true)
	state := getStringArg(args, "state", "present")

	changed := false

	if state == "present" {
		if content == "" {
			return m.failResult(result, "content parameter is required when state is present")
		}

		// Get current crontab for comparison
		currentCrontab, _ := m.getCrontab(exec, user)

		if currentCrontab != content && inCheckMode(args) {
			changed = true
			result.Output["action"] = "crontab_would_be_updated"
		} else if currentCrontab != content {
			// Backup if requested
			if backup && currentCrontab != "" {
				timestamp := time.Now().Format("20060102-150405")
				backupFile := fmt.Sprintf("/root/crontab.%s.%s.backup", user, timestamp)
				if err := writeHostFile(ctx, host, args, backupFile, []byte(currentCrontab), 0o600); err != nil {
					result.Output["backup_warning"] = fmt.Sprintf("failed to create backup: %v", err)
				} else {
					result.Output["backup_file"] = backupFile
				}
			}

			// Set new crontab
			if err := m.setCrontab(exec, user, content); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to set crontab: %v", err))
			}

			changed = true
			result.Output["action"] = "crontab_updated"
		}
	} else if state == "absent" {
		current, _ := m.getCrontab(exec, user)
		if strings.TrimSpace(current) == "" {
			result.Duration = time.Since(result.Timestamp)
			return result, nil
		}
		if inCheckMode(args) {
			result.Changed = true
			result.Output["action"] = "crontab_would_be_removed"
			result.Duration = time.Since(result.Timestamp)
			return result, nil
		}
		// Remove crontab
		if out, err := exec.Execute("crontab", "-r", "-u", user); err != nil {
			if !strings.Contains(out, "no crontab") {
				return m.failResult(result, fmt.Sprintf("failed to remove crontab: %v", err))
			}
		} else {
			changed = true
			result.Output["action"] = "crontab_removed"
		}
	}

	result.Changed = changed
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleSystem manages system cron files (cron.d, cron.daily, etc.)
func (m *CronModule) handleSystem(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	name := getStringArg(args, "name", "")
	content := getStringArg(args, "content", "")
	cronType := getStringArg(args, "cron_type", "d") // d, daily, hourly, weekly, monthly
	state := getStringArg(args, "state", "present")

	if name == "" {
		return m.failResult(result, "name parameter is required")
	}

	changed := false

	// Determine cron directory
	var cronDir string
	switch cronType {
	case "d":
		cronDir = "/etc/cron.d"
	case "daily":
		cronDir = "/etc/cron.daily"
	case "hourly":
		cronDir = "/etc/cron.hourly"
	case "weekly":
		cronDir = "/etc/cron.weekly"
	case "monthly":
		cronDir = "/etc/cron.monthly"
	default:
		return m.failResult(result, fmt.Sprintf("invalid cron_type: %s", cronType))
	}

	if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
		return m.failResult(result, fmt.Sprintf("invalid name %q: a file name in %s", name, cronDir))
	}
	cronFile := cronDir + "/" + name
	mode := os.FileMode(0o755)
	if cronType == "d" {
		mode = 0o644
	}

	current, exists, err := readHostFile(ctx, host, args, cronFile)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to read %s: %v", cronFile, err))
	}

	if state == "present" {
		if content == "" {
			return m.failResult(result, "content parameter is required when state is present")
		}
		if !strings.HasSuffix(content, "\n") {
			content += "\n" // cron ignores a last line without a newline
		}
		if !exists || string(current) != content {
			changed = true
			result.Output["action"] = "cron_file_written"
			result.Output["file"] = cronFile
			if !inCheckMode(args) {
				if err := writeHostFile(ctx, host, args, cronFile, []byte(content), mode); err != nil {
					return m.failResult(result, fmt.Sprintf("failed to write %s: %v", cronFile, err))
				}
			}
		}
	} else if state == "absent" && exists {
		changed = true
		result.Output["action"] = "cron_file_removed"
		result.Output["file"] = cronFile
		if !inCheckMode(args) {
			if _, err := runOnHost(ctx, host, args, "rm", "-f", cronFile); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to remove file: %v", err))
			}
		}
	}

	result.Changed = changed
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleList lists cron jobs
func (m *CronModule) handleList(ctx context.Context, exec *executor.CommandExecutor, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	user := getStringArg(args, "user", "root")

	// Get crontab
	crontab, err := m.getCrontab(exec, user)
	if err != nil {
		if strings.Contains(err.Error(), "no crontab") {
			result.Output["jobs"] = []string{}
			result.Output["message"] = "no crontab for user"
		} else {
			return m.failResult(result, fmt.Sprintf("failed to get crontab: %v", err))
		}
	} else {
		jobs := m.parseCrontab(crontab)
		result.Output["jobs"] = jobs
		result.Output["jobs_count"] = len(jobs)
		result.Output["raw_crontab"] = crontab
	}

	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// Helper methods
func (m *CronModule) getCrontab(exec *executor.CommandExecutor, user string) (string, error) {
	output, err := exec.Execute("crontab -l -u " + shellQuote(user))
	if err != nil {
		// crontab exits 1 and prints "no crontab for <user>" when there is none
		if strings.Contains(output, "no crontab") {
			return "", nil
		}
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(output))
	}
	return output, nil
}

func (m *CronModule) setCrontab(exec *executor.CommandExecutor, user string, content string) error {
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	// One shell command so quoting survives SSH and sudo; no temp file
	script := fmt.Sprintf("printf '%%s' %s | crontab -u %s -", shellQuote(content), shellQuote(user))
	if output, err := exec.Execute("sh -c " + shellQuote(script)); err != nil {
		return fmt.Errorf("failed to install crontab: %w: %s", err, strings.TrimSpace(output))
	}
	return nil
}

func (m *CronModule) parseCrontab(crontab string) map[string]string {
	jobs := make(map[string]string)
	lines := strings.Split(crontab, "\n")

	var currentName string
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for name comment
		if strings.HasPrefix(line, "# Ansible:") || strings.HasPrefix(line, "# Onigirazu:") {
			currentName = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "# Ansible:"), "# Onigirazu:"))
			continue
		}

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// If we have a name, associate this line with it
		if currentName != "" {
			jobs[currentName] = line
			currentName = ""
		}
	}

	return jobs
}

func (m *CronModule) buildCrontab(jobs map[string]string) string {
	var lines []string

	lines = append(lines, "# Managed by Onigirazu")
	lines = append(lines, "")

	for name, job := range jobs {
		lines = append(lines, fmt.Sprintf("# Onigirazu: %s", name))
		lines = append(lines, job)
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// Validate validates cron module arguments
func (m *CronModule) Validate(args map[string]interface{}) error {
	operation := getStringArg(args, "operation", "job")

	switch operation {
	case "job":
		if _, exists := args["name"]; !exists {
			return fmt.Errorf("name parameter is required")
		}
		state := getStringArg(args, "state", "present")
		if state == "present" {
			if _, exists := args["job"]; !exists {
				return fmt.Errorf("job parameter is required when state is present")
			}
		}
	case "file":
		state := getStringArg(args, "state", "present")
		if state == "present" {
			if _, exists := args["content"]; !exists {
				return fmt.Errorf("content parameter is required when state is present")
			}
		}
	case "system":
		if _, exists := args["name"]; !exists {
			return fmt.Errorf("name parameter is required")
		}
		state := getStringArg(args, "state", "present")
		if state == "present" {
			if _, exists := args["content"]; !exists {
				return fmt.Errorf("content parameter is required when state is present")
			}
		}
	case "list":
		// No additional validation needed
	default:
		return fmt.Errorf("invalid operation: %s", operation)
	}

	return nil
}
