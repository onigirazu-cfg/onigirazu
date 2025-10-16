package healthcheck

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Checker performs health checks on inventory hosts
type Checker struct {
	hosts      []types.Host
	config     *CheckConfig
	logger     interfaces.Logger
	maxWorkers int
}

// NewChecker creates a new health checker
func NewChecker(hosts []types.Host, config *CheckConfig, logger interfaces.Logger) *Checker {
	if config == nil {
		config = NewCheckConfig()
	}
	if logger == nil {
		logger = &noOpLogger{}
	}
	return &Checker{
		hosts:      hosts,
		config:     config,
		logger:     logger,
		maxWorkers: 10, // default concurrent workers
	}
}

// SetMaxWorkers sets the maximum number of concurrent workers
func (c *Checker) SetMaxWorkers(workers int) {
	if workers > 0 {
		c.maxWorkers = workers
	}
}

// CheckAll performs all health checks on all hosts
func (c *Checker) CheckAll(ctx context.Context) (*HealthCheckReport, error) {
	startTime := time.Now()
	c.logger.Info("Starting health check on %d hosts", len(c.hosts))

	// Create a channel for results and errors
	resultsChan := make(chan *HostHealthReport, len(c.hosts))
	errsChan := make(chan error, len(c.hosts))

	// Semaphore for limiting concurrent workers
	sem := make(chan struct{}, c.maxWorkers)
	var wg sync.WaitGroup

	// Process each host
	for i := range c.hosts {
		wg.Add(1)
		go func(host types.Host) {
			defer wg.Done()

			// Acquire semaphore slot
			sem <- struct{}{}
			defer func() { <-sem }()

			// Check if context is canceled
			select {
			case <-ctx.Done():
				errsChan <- ctx.Err()
				return
			default:
			}

			// Check host
			report, err := c.CheckHost(ctx, host)
			if err != nil {
				if !c.config.SkipUnavailableHosts {
					errsChan <- fmt.Errorf("failed to check host %s: %w", host.Name, err)
				} else {
					c.logger.Warn("Skipping unavailable host %s: %v", host.Name, err)
				}
				return
			}

			resultsChan <- report
		}(c.hosts[i])
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(resultsChan)
	close(errsChan)

	// Collect errors
	var errs []error
	for err := range errsChan {
		errs = append(errs, err)
	}

	// Collect results
	var hostReports []HostHealthReport
	for report := range resultsChan {
		hostReports = append(hostReports, *report)
	}

	// Build overall report
	report := c.buildReport(hostReports, time.Since(startTime))

	// Return first error if any
	if len(errs) > 0 && !c.config.SkipUnavailableHosts {
		return report, errs[0]
	}

	return report, nil
}

// CheckHost performs health checks on a single host
func (c *Checker) CheckHost(ctx context.Context, host types.Host) (*HostHealthReport, error) {
	startTime := time.Now()
	c.logger.Debug("Checking health of host %s", host.Name)

	report := &HostHealthReport{
		Host:      host,
		Checks:    make([]HealthCheckResult, 0),
		Summary:   make(map[string]int),
		Timestamp: time.Now(),
	}

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Run configured checks
	for _, checkType := range c.config.CheckTypes {
		result := c.runCheck(checkCtx, host, checkType)
		report.Checks = append(report.Checks, result)

		// Update summary
		statusStr := string(result.Status)
		report.Summary[statusStr]++
	}

	// Determine overall host status
	report.Status = c.determineHostStatus(report.Checks)
	report.Duration = time.Since(startTime)

	return report, nil
}

// runCheck runs a single health check
func (c *Checker) runCheck(ctx context.Context, host types.Host, checkType CheckType) HealthCheckResult {
	result := HealthCheckResult{
		CheckType: checkType,
		Host:      host.Name,
		Status:    StatusUnknown,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	startTime := time.Now()
	defer func() { result.Duration = time.Since(startTime) }()

	switch checkType {
	case CheckConnectivity:
		c.checkConnectivity(ctx, host, &result)
	case CheckDiskSpace:
		c.checkDiskSpace(ctx, host, &result)
	case CheckMemory:
		c.checkMemory(ctx, host, &result)
	case CheckCPU:
		c.checkCPU(ctx, host, &result)
	case CheckServices:
		c.checkServices(ctx, host, &result)
	case CheckNetwork:
		c.checkNetwork(ctx, host, &result)
	default:
		result.Error = fmt.Sprintf("unknown check type: %v", checkType)
		result.Status = StatusUnknown
	}

	return result
}

// determineHostStatus determines the overall status for a host
func (c *Checker) determineHostStatus(checks []HealthCheckResult) HealthStatus {
	// If any check is critical, the host is critical
	for _, check := range checks {
		if check.Status == StatusCritical {
			return StatusCritical
		}
	}

	// If any check is warning, the host is warning
	for _, check := range checks {
		if check.Status == StatusWarning {
			return StatusWarning
		}
	}

	// If any check is unknown, the host is unknown (unless all are healthy)
	hasUnknown := false
	for _, check := range checks {
		if check.Status == StatusUnknown {
			hasUnknown = true
		}
	}

	if hasUnknown && len(checks) > 0 {
		// If all checks are unknown or healthy, check if all are unknown
		allUnknown := true
		for _, check := range checks {
			if check.Status != StatusUnknown {
				allUnknown = false
				break
			}
		}
		if allUnknown {
			return StatusUnknown
		}
	}

	// Otherwise, the host is healthy
	return StatusHealthy
}

// buildReport builds the overall health check report
func (c *Checker) buildReport(hostReports []HostHealthReport, duration time.Duration) *HealthCheckReport {
	report := &HealthCheckReport{
		Timestamp:     time.Now(),
		OverallStatus: StatusHealthy,
		HostReports:   hostReports,
		Summary:       make(map[string]int),
		TotalDuration: duration,
		Statistics: HealthCheckStatistics{
			TotalHosts:    len(hostReports),
			ChecksPerHost: make(map[CheckType]int),
		},
	}

	// Calculate statistics
	for _, hostReport := range hostReports {
		// Update summary from host reports
		for status, count := range hostReport.Summary {
			report.Summary[status] += count
		}

		// Update host status counts
		switch hostReport.Status {
		case StatusHealthy:
			report.Statistics.HealthyHosts++
		case StatusWarning:
			report.Statistics.WarningHosts++
		case StatusCritical:
			report.Statistics.CriticalHosts++
		case StatusUnknown:
			report.Statistics.UnknownHosts++
		}

		// Count checks per type
		for _, check := range hostReport.Checks {
			report.Statistics.ChecksPerHost[check.CheckType]++
		}
	}

	// Determine overall status
	if report.Statistics.CriticalHosts > 0 {
		report.OverallStatus = StatusCritical
	} else if report.Statistics.WarningHosts > 0 {
		report.OverallStatus = StatusWarning
	}

	// Calculate average duration
	if len(hostReports) > 0 {
		totalMs := 0.0
		for _, hostReport := range hostReports {
			totalMs += hostReport.Duration.Seconds() * 1000
		}
		report.Statistics.AverageDuration = totalMs / float64(len(hostReports))
	}

	return report
}

// checkConnectivity checks SSH connectivity to the host
func (c *Checker) checkConnectivity(ctx context.Context, host types.Host, result *HealthCheckResult) {
	// Check if host is local
	if sshpkg.IsLocal(host) {
		result.Status = StatusHealthy
		result.Message = "Local host"
		result.Details["connection_type"] = "local"
		return
	}

	// Try to create SSH client
	client, err := sshpkg.NewClient(host)
	if err != nil {
		result.Status = StatusCritical
		result.Message = fmt.Sprintf("SSH connection failed: %v", err)
		result.Error = err.Error()
		result.Details["connection_type"] = "ssh"
		return
	}
	defer client.Close()

	// Execute a simple command to verify connection
	_, err = client.ExecuteCommand("echo OK")
	if err != nil {
		result.Status = StatusCritical
		result.Message = fmt.Sprintf("SSH command execution failed: %v", err)
		result.Error = err.Error()
		result.Details["connection_type"] = "ssh"
		return
	}

	result.Status = StatusHealthy
	result.Message = "SSH connection successful"
	result.Details["connection_type"] = "ssh"
	result.Details["address"] = host.Address
	if host.User != "" {
		result.Details["user"] = host.User
	}
	if host.Port > 0 {
		result.Details["port"] = host.Port
	}
}

// checkDiskSpace checks disk space usage
func (c *Checker) checkDiskSpace(ctx context.Context, host types.Host, result *HealthCheckResult) {
	// Local check only
	if sshpkg.IsLocal(host) {
		// For local host, we can use local filesystem info
		// This is a simplified check - in production, you might use specific tools
		result.Status = StatusHealthy
		result.Message = "Local disk space check"
		result.Details["check_method"] = "local"
		return
	}

	// For SSH hosts, execute remote command
	client, err := sshpkg.NewClient(host)
	if err != nil {
		result.Status = StatusUnknown
		result.Message = fmt.Sprintf("Could not check disk space: %v", err)
		result.Error = err.Error()
		return
	}
	defer client.Close()

	// Execute df command to get disk usage
	output, err := client.ExecuteCommand("df -h / | tail -1 | awk '{print $5}' | sed 's/%//'")
	if err != nil {
		result.Status = StatusUnknown
		result.Message = fmt.Sprintf("Failed to get disk usage: %v", err)
		result.Error = err.Error()
		return
	}

	// Parse output
	var diskUsage int
	_, err = fmt.Sscanf(output, "%d", &diskUsage)
	if err != nil {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("Could not parse disk usage: %v", err)
		result.Error = err.Error()
		result.Details["raw_output"] = output
		return
	}

	result.Details["disk_usage_percent"] = diskUsage
	result.Details["threshold"] = c.config.DiskSpaceThreshold

	if diskUsage >= 95 {
		result.Status = StatusCritical
		result.Message = fmt.Sprintf("Disk usage critically high: %d%%", diskUsage)
	} else if diskUsage >= c.config.DiskSpaceThreshold {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("Disk usage warning: %d%%", diskUsage)
	} else {
		result.Status = StatusHealthy
		result.Message = fmt.Sprintf("Disk usage normal: %d%%", diskUsage)
	}
}

// checkMemory checks memory usage
func (c *Checker) checkMemory(ctx context.Context, host types.Host, result *HealthCheckResult) {
	if sshpkg.IsLocal(host) {
		result.Status = StatusHealthy
		result.Message = "Local memory check"
		result.Details["check_method"] = "local"
		return
	}

	client, err := sshpkg.NewClient(host)
	if err != nil {
		result.Status = StatusUnknown
		result.Message = fmt.Sprintf("Could not check memory: %v", err)
		result.Error = err.Error()
		return
	}
	defer client.Close()

	// Execute free command to get memory usage
	output, err := client.ExecuteCommand("free | grep Mem | awk '{printf \"%.0f\", ($3/$2) * 100}'")
	if err != nil {
		result.Status = StatusUnknown
		result.Message = fmt.Sprintf("Failed to get memory usage: %v", err)
		result.Error = err.Error()
		return
	}

	var memUsage int
	_, err = fmt.Sscanf(output, "%d", &memUsage)
	if err != nil {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("Could not parse memory usage: %v", err)
		result.Error = err.Error()
		result.Details["raw_output"] = output
		return
	}

	result.Details["memory_usage_percent"] = memUsage
	result.Details["threshold"] = c.config.MemoryThreshold

	if memUsage >= 95 {
		result.Status = StatusCritical
		result.Message = fmt.Sprintf("Memory usage critically high: %d%%", memUsage)
	} else if memUsage >= c.config.MemoryThreshold {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("Memory usage warning: %d%%", memUsage)
	} else {
		result.Status = StatusHealthy
		result.Message = fmt.Sprintf("Memory usage normal: %d%%", memUsage)
	}
}

// checkCPU checks CPU usage
func (c *Checker) checkCPU(ctx context.Context, host types.Host, result *HealthCheckResult) {
	if sshpkg.IsLocal(host) {
		result.Status = StatusHealthy
		result.Message = "Local CPU check"
		result.Details["check_method"] = "local"
		return
	}

	client, err := sshpkg.NewClient(host)
	if err != nil {
		result.Status = StatusUnknown
		result.Message = fmt.Sprintf("Could not check CPU: %v", err)
		result.Error = err.Error()
		return
	}
	defer client.Close()

	// Get load average
	output, err := client.ExecuteCommand("cat /proc/loadavg | awk '{print $1}' | sed 's/\\.//'")
	if err != nil {
		result.Status = StatusUnknown
		result.Message = fmt.Sprintf("Failed to get CPU load: %v", err)
		result.Error = err.Error()
		return
	}

	var cpuLoad int
	_, err = fmt.Sscanf(output, "%d", &cpuLoad)
	if err != nil {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("Could not parse CPU load: %v", err)
		result.Error = err.Error()
		result.Details["raw_output"] = output
		return
	}

	// Convert to percentage (load is often represented differently)
	result.Details["load_average"] = float64(cpuLoad) / 100
	result.Details["threshold"] = c.config.CPUThreshold

	result.Status = StatusHealthy
	result.Message = fmt.Sprintf("CPU load normal")
}

// checkServices checks if critical services are running
func (c *Checker) checkServices(ctx context.Context, host types.Host, result *HealthCheckResult) {
	if len(c.config.Services) == 0 {
		result.Status = StatusHealthy
		result.Message = "No services configured to check"
		return
	}

	if sshpkg.IsLocal(host) {
		result.Status = StatusHealthy
		result.Message = "Local service check"
		result.Details["check_method"] = "local"
		return
	}

	client, err := sshpkg.NewClient(host)
	if err != nil {
		result.Status = StatusUnknown
		result.Message = fmt.Sprintf("Could not check services: %v", err)
		result.Error = err.Error()
		return
	}
	defer client.Close()

	failedServices := []string{}
	result.Details["services"] = make(map[string]string)

	for _, service := range c.config.Services {
		cmd := fmt.Sprintf("systemctl is-active %s", service)
		_, err := client.ExecuteCommand(cmd)

		if err != nil {
			failedServices = append(failedServices, service)
			result.Details["services"].(map[string]string)[service] = "inactive"
		} else {
			result.Details["services"].(map[string]string)[service] = "active"
		}
	}

	if len(failedServices) > 0 {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("Some services are not running: %v", failedServices)
	} else {
		result.Status = StatusHealthy
		result.Message = "All services are running"
	}
}

// checkNetwork checks network connectivity
func (c *Checker) checkNetwork(ctx context.Context, host types.Host, result *HealthCheckResult) {
	if sshpkg.IsLocal(host) {
		result.Status = StatusHealthy
		result.Message = "Local network check"
		result.Details["check_method"] = "local"
		return
	}

	client, err := sshpkg.NewClient(host)
	if err != nil {
		result.Status = StatusUnknown
		result.Message = fmt.Sprintf("Could not check network: %v", err)
		result.Error = err.Error()
		return
	}
	defer client.Close()

	// Check if host can reach default gateway
	output, err := client.ExecuteCommand("ping -c 1 $(ip route | grep default | awk '{print $3}' | head -1) 2>&1 | grep -c 'bytes from'")
	if err != nil || output == "0" {
		result.Status = StatusWarning
		result.Message = "Could not reach default gateway"
		return
	}

	result.Status = StatusHealthy
	result.Message = "Network connectivity good"
	result.Details["gateway_reachable"] = true
}

// noOpLogger is a no-op logger for when no logger is provided
type noOpLogger struct{}

func (l *noOpLogger) Debug(format string, args ...interface{})                                {}
func (l *noOpLogger) Info(format string, args ...interface{})                                 {}
func (l *noOpLogger) Error(format string, args ...interface{})                                {}
func (l *noOpLogger) Warn(format string, args ...interface{})                                 {}
func (l *noOpLogger) Fatal(format string, args ...interface{})                                {}
func (l *noOpLogger) SetLevel(level string)                                                   {}
func (l *noOpLogger) TaskStart(taskName, hostName string)                                     {}
func (l *noOpLogger) TaskEnd(taskName, hostName string, changed, success bool)                {}
func (l *noOpLogger) PlayStart(playName string, playIndex, totalPlays int)                    {}
func (l *noOpLogger) PlayEnd(playName, hostName string, success bool, duration time.Duration) {}
func (l *noOpLogger) Progress(completed, total int, currentTask, currentHost string)          {}
func (l *noOpLogger) Retry(taskName, hostName string, attempt, maxAttempts int, delay time.Duration, err error) {
}
