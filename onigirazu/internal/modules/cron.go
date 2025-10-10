package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// CronModule implements cron job management
type CronModule struct {
	BaseModule
	executor *executor.CommandExecutor
}

// NewCronModule creates a new cron module
func NewCronModule() *CronModule {
	return &CronModule{
		BaseModule: BaseModule{
			name:        "cron",
			description: "Manage cron jobs and crontab files",
		},
	}
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

	// Initialize executor
	if m.executor == nil {
		exec, err := executor.NewCommandExecutor(host)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("failed to create executor: %v", err))
		}
		m.executor = exec
	}

	// Get operation type
	operation := getStringArg(args, "operation", "job")

	switch operation {
	case "job":
		return m.handleJob(ctx, host, args, result)
	case "file":
		return m.handleFile(ctx, host, args, result)
	case "system":
		return m.handleSystem(ctx, host, args, result)
	case "list":
		return m.handleList(ctx, host, args, result)
	default:
		return m.failResult(result, fmt.Sprintf("unknown operation: %s", operation))
	}
}

// handleJob manages individual cron jobs in user crontab
func (m *CronModule) handleJob(ctx context.Context, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
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
	currentCrontab, err := m.getCrontab(user)
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

	// Write crontab if changed
	if changed {
		newCrontab := m.buildCrontab(jobs)
		if err := m.setCrontab(user, newCrontab); err != nil {
			return m.failResult(result, fmt.Sprintf("failed to set crontab: %v", err))
		}
	}

	result.Changed = changed
	result.Output["jobs_count"] = len(jobs)
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleFile manages crontab files directly
func (m *CronModule) handleFile(ctx context.Context, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
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
		currentCrontab, _ := m.getCrontab(user)

		if currentCrontab != content {
			// Backup if requested
			if backup && currentCrontab != "" {
				timestamp := time.Now().Format("20060102-150405")
				backupFile := fmt.Sprintf("/tmp/crontab.%s.%s.backup", user, timestamp)
				if _, err := m.executor.Execute("sh", "-c", fmt.Sprintf("echo '%s' > %s", currentCrontab, backupFile)); err != nil {
					result.Output["backup_warning"] = fmt.Sprintf("failed to create backup: %v", err)
				} else {
					result.Output["backup_file"] = backupFile
				}
			}

			// Set new crontab
			if err := m.setCrontab(user, content); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to set crontab: %v", err))
			}

			changed = true
			result.Output["action"] = "crontab_updated"
		}
	} else if state == "absent" {
		// Remove crontab
		if _, err := m.executor.Execute("crontab", "-r", "-u", user); err != nil {
			if !strings.Contains(err.Error(), "no crontab") {
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
func (m *CronModule) handleSystem(ctx context.Context, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
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

	cronFile := fmt.Sprintf("%s/%s", cronDir, name)

	if state == "present" {
		if content == "" {
			return m.failResult(result, "content parameter is required when state is present")
		}

		// Check if file exists and compare content
		currentContent, err := m.executor.Execute("cat", cronFile)
		fileExists := err == nil

		if !fileExists || currentContent != content {
			// Write content to temp file
			tmpFile := fmt.Sprintf("/tmp/%s", name)
			if _, err := m.executor.Execute("sh", "-c", fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", tmpFile, content)); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to write temp file: %v", err))
			}

			// Move to cron directory
			if _, err := m.executor.Execute("mv", tmpFile, cronFile); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to move file: %v", err))
			}

			// Set permissions
			if cronType == "d" {
				if _, err := m.executor.Execute("chmod", "644", cronFile); err != nil {
					return m.failResult(result, fmt.Sprintf("failed to set permissions: %v", err))
				}
			} else {
				if _, err := m.executor.Execute("chmod", "755", cronFile); err != nil {
					return m.failResult(result, fmt.Sprintf("failed to set permissions: %v", err))
				}
			}

			changed = true
			result.Output["action"] = "cron_file_created"
			result.Output["file"] = cronFile
		}
	} else if state == "absent" {
		// Check if file exists
		_, err := m.executor.Execute("test", "-f", cronFile)
		if err == nil {
			// Remove file
			if _, err := m.executor.Execute("rm", "-f", cronFile); err != nil {
				return m.failResult(result, fmt.Sprintf("failed to remove file: %v", err))
			}

			changed = true
			result.Output["action"] = "cron_file_removed"
			result.Output["file"] = cronFile
		}
	}

	result.Changed = changed
	result.Duration = time.Since(result.Timestamp)
	return result, nil
}

// handleList lists cron jobs
func (m *CronModule) handleList(ctx context.Context, host types.Host, args map[string]interface{}, result types.TaskResult) (types.TaskResult, error) {
	user := getStringArg(args, "user", "root")

	// Get crontab
	crontab, err := m.getCrontab(user)
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
func (m *CronModule) getCrontab(user string) (string, error) {
	output, err := m.executor.Execute("crontab", "-l", "-u", user)
	if err != nil {
		return "", err
	}
	return output, nil
}

func (m *CronModule) setCrontab(user string, content string) error {
	// Write content to temp file
	tmpFile := fmt.Sprintf("/tmp/crontab.%s.tmp", user)
	if _, err := m.executor.Execute("sh", "-c", fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", tmpFile, content)); err != nil {
		return fmt.Errorf("failed to write temp file: %v", err)
	}

	// Install crontab
	if _, err := m.executor.Execute("crontab", "-u", user, tmpFile); err != nil {
		return fmt.Errorf("failed to install crontab: %v", err)
	}

	// Clean up temp file (ignore errors as it's just cleanup)
	_, _ = m.executor.Execute("rm", "-f", tmpFile)

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
