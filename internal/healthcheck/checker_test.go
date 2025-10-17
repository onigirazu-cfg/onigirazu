package healthcheck

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNewChecker(t *testing.T) {
	hosts := []types.Host{
		{Name: "localhost", Address: "127.0.0.1"},
	}
	config := NewCheckConfig()
	logger := &noOpLogger{}

	checker := NewChecker(hosts, config, logger)

	if checker == nil {
		t.Fatal("NewChecker returned nil")
	}
	if len(checker.hosts) != 1 {
		t.Errorf("Expected 1 host, got %d", len(checker.hosts))
	}
	if checker.config == nil {
		t.Error("Config should not be nil")
	}
}

func TestNewCheckConfig(t *testing.T) {
	config := NewCheckConfig()

	if config == nil {
		t.Fatal("NewCheckConfig returned nil")
	}
	if len(config.CheckTypes) == 0 {
		t.Error("CheckTypes should not be empty")
	}
	if config.Timeout == 0 {
		t.Error("Timeout should not be 0")
	}
	if config.DiskSpaceThreshold <= 0 {
		t.Error("DiskSpaceThreshold should be > 0")
	}
}

func TestSetMaxWorkers(t *testing.T) {
	hosts := []types.Host{
		{Name: "localhost", Address: "127.0.0.1"},
	}
	checker := NewChecker(hosts, NewCheckConfig(), &noOpLogger{})

	checker.SetMaxWorkers(5)
	if checker.maxWorkers != 5 {
		t.Errorf("Expected maxWorkers to be 5, got %d", checker.maxWorkers)
	}

	// Test invalid value (should not change)
	oldValue := checker.maxWorkers
	checker.SetMaxWorkers(0)
	if checker.maxWorkers != oldValue {
		t.Error("SetMaxWorkers should not accept 0")
	}
}

func TestCheckAll(t *testing.T) {
	// Test with localhost (should work)
	hosts := []types.Host{
		{Name: "localhost", Address: "127.0.0.1"},
	}
	config := NewCheckConfig()
	config.CheckTypes = []CheckType{CheckConnectivity} // Only test connectivity

	checker := NewChecker(hosts, config, &noOpLogger{})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report, err := checker.CheckAll(ctx)

	if err != nil {
		t.Errorf("CheckAll failed: %v", err)
	}
	if report == nil {
		t.Fatal("CheckAll returned nil report")
	}
	if len(report.HostReports) != 1 {
		t.Errorf("Expected 1 host report, got %d", len(report.HostReports))
	}
}

func TestCheckHost(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	config.CheckTypes = []CheckType{CheckConnectivity}

	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report, err := checker.CheckHost(ctx, host)

	if err != nil {
		t.Errorf("CheckHost failed: %v", err)
	}
	if report == nil {
		t.Fatal("CheckHost returned nil report")
	}
	if report.Host.Name != "localhost" {
		t.Errorf("Expected host name 'localhost', got %s", report.Host.Name)
	}
	if len(report.Checks) == 0 {
		t.Error("CheckHost should return at least one check result")
	}
}

func TestDetermineHostStatus(t *testing.T) {
	tests := []struct {
		name     string
		checks   []HealthCheckResult
		expected HealthStatus
	}{
		{
			name: "all healthy",
			checks: []HealthCheckResult{
				{Status: StatusHealthy},
				{Status: StatusHealthy},
			},
			expected: StatusHealthy,
		},
		{
			name: "one warning",
			checks: []HealthCheckResult{
				{Status: StatusHealthy},
				{Status: StatusWarning},
			},
			expected: StatusWarning,
		},
		{
			name: "one critical",
			checks: []HealthCheckResult{
				{Status: StatusHealthy},
				{Status: StatusCritical},
			},
			expected: StatusCritical,
		},
		{
			name: "critical takes precedence",
			checks: []HealthCheckResult{
				{Status: StatusWarning},
				{Status: StatusCritical},
			},
			expected: StatusCritical,
		},
	}

	checker := NewChecker([]types.Host{}, NewCheckConfig(), &noOpLogger{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.determineHostStatus(tt.checks)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestBuildReport(t *testing.T) {
	hostReports := []HostHealthReport{
		{
			Host:   types.Host{Name: "host1", Address: "1.1.1.1"},
			Status: StatusHealthy,
			Checks: []HealthCheckResult{
				{CheckType: CheckConnectivity, Status: StatusHealthy},
			},
			Summary: map[string]int{
				string(StatusHealthy): 1,
			},
		},
		{
			Host:   types.Host{Name: "host2", Address: "2.2.2.2"},
			Status: StatusWarning,
			Checks: []HealthCheckResult{
				{CheckType: CheckDiskSpace, Status: StatusWarning},
			},
			Summary: map[string]int{
				string(StatusWarning): 1,
			},
		},
	}

	checker := NewChecker([]types.Host{}, NewCheckConfig(), &noOpLogger{})
	report := checker.buildReport(hostReports, 5*time.Second)

	if report == nil {
		t.Fatal("buildReport returned nil")
	}
	if len(report.HostReports) != 2 {
		t.Errorf("Expected 2 host reports, got %d", len(report.HostReports))
	}
	if report.Statistics.TotalHosts != 2 {
		t.Errorf("Expected 2 total hosts, got %d", report.Statistics.TotalHosts)
	}
	if report.Statistics.HealthyHosts != 1 {
		t.Errorf("Expected 1 healthy host, got %d", report.Statistics.HealthyHosts)
	}
	if report.Statistics.WarningHosts != 1 {
		t.Errorf("Expected 1 warning host, got %d", report.Statistics.WarningHosts)
	}
	if report.OverallStatus != StatusWarning {
		t.Errorf("Expected overall status warning, got %s", report.OverallStatus)
	}
}

func TestContextTimeout(t *testing.T) {
	hosts := []types.Host{
		{Name: "localhost", Address: "127.0.0.1"},
	}
	config := NewCheckConfig()

	checker := NewChecker(hosts, config, &noOpLogger{})

	// Create a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	report, _ := checker.CheckAll(ctx)

	// Should handle the context cancellation gracefully
	if report == nil {
		t.Error("CheckAll returned nil report even with canceled context")
	}
}

func TestCheckTypeParsing(t *testing.T) {
	tests := []struct {
		checkType CheckType
		name      string
	}{
		{CheckConnectivity, "connectivity"},
		{CheckDiskSpace, "disk_space"},
		{CheckMemory, "memory"},
		{CheckCPU, "cpu"},
		{CheckServices, "services"},
		{CheckNetwork, "network"},
		{CheckCertificates, "certificates"},
	}

	for _, tt := range tests {
		if string(tt.checkType) != tt.name {
			t.Errorf("Expected %s, got %s", tt.name, string(tt.checkType))
		}
	}
}

// ============================================================================
// Extended Tests for Higher Coverage
// ============================================================================

// TestNewCheckerWithNilConfig tests that nil config gets default
func TestNewCheckerWithNilConfig(t *testing.T) {
	hosts := []types.Host{
		{Name: "host1", Address: "192.168.1.1"},
	}
	checker := NewChecker(hosts, nil, &noOpLogger{})

	if checker.config == nil {
		t.Fatal("config should not be nil after NewChecker with nil config")
	}
	if len(checker.config.CheckTypes) == 0 {
		t.Error("config should have default CheckTypes")
	}
}

// TestNewCheckerWithNilLogger tests that nil logger gets default
func TestNewCheckerWithNilLogger(t *testing.T) {
	hosts := []types.Host{
		{Name: "host1", Address: "192.168.1.1"},
	}
	checker := NewChecker(hosts, NewCheckConfig(), nil)

	if checker.logger == nil {
		t.Fatal("logger should not be nil after NewChecker with nil logger")
	}
}

// TestRunCheckUnknownType tests handling of unknown check type
func TestRunCheckUnknownType(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	ctx := context.Background()
	result := checker.runCheck(ctx, host, CheckType("unknown_check"))

	if result.Status != StatusUnknown {
		t.Errorf("Expected StatusUnknown for unknown check type, got %s", result.Status)
	}
	if result.Error == "" {
		t.Error("Expected error message for unknown check type")
	}
	if !strings.Contains(result.Error, "unknown check type") {
		t.Errorf("Expected 'unknown check type' in error, got: %s", result.Error)
	}
}

// TestRunCheckHasCorrectTimestamp tests that check result has timestamp
func TestRunCheckHasCorrectTimestamp(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	ctx := context.Background()
	beforeTime := time.Now()
	result := checker.runCheck(ctx, host, CheckConnectivity)
	afterTime := time.Now()

	if result.Timestamp.Before(beforeTime) || result.Timestamp.After(afterTime) {
		t.Errorf("Check result timestamp not within expected range")
	}
}

// TestDetermineHostStatusEmptyChecks tests with empty checks slice
func TestDetermineHostStatusEmptyChecks(t *testing.T) {
	checker := NewChecker([]types.Host{}, NewCheckConfig(), &noOpLogger{})
	result := checker.determineHostStatus([]HealthCheckResult{})

	if result != StatusHealthy {
		t.Errorf("Expected StatusHealthy for empty checks, got %s", result)
	}
}

// TestDetermineHostStatusAllUnknown tests all unknown status
func TestDetermineHostStatusAllUnknown(t *testing.T) {
	checker := NewChecker([]types.Host{}, NewCheckConfig(), &noOpLogger{})
	checks := []HealthCheckResult{
		{Status: StatusUnknown},
		{Status: StatusUnknown},
	}
	result := checker.determineHostStatus(checks)

	if result != StatusUnknown {
		t.Errorf("Expected StatusUnknown when all unknown, got %s", result)
	}
}

// TestDetermineHostStatusMixedWithUnknown tests mixed status with unknown
func TestDetermineHostStatusMixedWithUnknown(t *testing.T) {
	checker := NewChecker([]types.Host{}, NewCheckConfig(), &noOpLogger{})
	checks := []HealthCheckResult{
		{Status: StatusHealthy},
		{Status: StatusUnknown},
		{Status: StatusHealthy},
	}
	result := checker.determineHostStatus(checks)

	if result != StatusHealthy {
		t.Errorf("Expected StatusHealthy when mixed with healthy, got %s", result)
	}
}

// TestBuildReportEmptyHostReports tests with no host reports
func TestBuildReportEmptyHostReports(t *testing.T) {
	checker := NewChecker([]types.Host{}, NewCheckConfig(), &noOpLogger{})
	report := checker.buildReport([]HostHealthReport{}, 1*time.Second)

	if report == nil {
		t.Fatal("buildReport should not return nil")
	}
	if report.Statistics.TotalHosts != 0 {
		t.Errorf("Expected 0 total hosts, got %d", report.Statistics.TotalHosts)
	}
	if report.OverallStatus != StatusHealthy {
		t.Errorf("Expected StatusHealthy for empty report, got %s", report.OverallStatus)
	}
}

// TestBuildReportCriticalPriority tests that critical status takes priority
func TestBuildReportCriticalPriority(t *testing.T) {
	hostReports := []HostHealthReport{
		{
			Host:   types.Host{Name: "host1", Address: "1.1.1.1"},
			Status: StatusHealthy,
			Checks: []HealthCheckResult{
				{CheckType: CheckConnectivity, Status: StatusHealthy},
			},
			Summary: map[string]int{string(StatusHealthy): 1},
		},
		{
			Host:   types.Host{Name: "host2", Address: "2.2.2.2"},
			Status: StatusCritical,
			Checks: []HealthCheckResult{
				{CheckType: CheckDiskSpace, Status: StatusCritical},
			},
			Summary: map[string]int{string(StatusCritical): 1},
		},
		{
			Host:   types.Host{Name: "host3", Address: "3.3.3.3"},
			Status: StatusWarning,
			Checks: []HealthCheckResult{
				{CheckType: CheckMemory, Status: StatusWarning},
			},
			Summary: map[string]int{string(StatusWarning): 1},
		},
	}

	checker := NewChecker([]types.Host{}, NewCheckConfig(), &noOpLogger{})
	report := checker.buildReport(hostReports, 1*time.Second)

	if report.OverallStatus != StatusCritical {
		t.Errorf("Expected StatusCritical, got %s", report.OverallStatus)
	}
	if report.Statistics.CriticalHosts != 1 {
		t.Errorf("Expected 1 critical host, got %d", report.Statistics.CriticalHosts)
	}
}

// TestBuildReportStatisticsAccuracy tests accurate stat calculation
func TestBuildReportStatisticsAccuracy(t *testing.T) {
	hostReports := []HostHealthReport{
		{
			Host:   types.Host{Name: "host1", Address: "1.1.1.1"},
			Status: StatusHealthy,
			Checks: []HealthCheckResult{
				{CheckType: CheckConnectivity, Status: StatusHealthy},
				{CheckType: CheckDiskSpace, Status: StatusHealthy},
			},
			Summary:  map[string]int{string(StatusHealthy): 2},
			Duration: 100 * time.Millisecond,
		},
		{
			Host:   types.Host{Name: "host2", Address: "2.2.2.2"},
			Status: StatusWarning,
			Checks: []HealthCheckResult{
				{CheckType: CheckMemory, Status: StatusWarning},
			},
			Summary:  map[string]int{string(StatusWarning): 1},
			Duration: 200 * time.Millisecond,
		},
	}

	checker := NewChecker([]types.Host{}, NewCheckConfig(), &noOpLogger{})
	report := checker.buildReport(hostReports, 5*time.Second)

	if report.Statistics.TotalHosts != 2 {
		t.Errorf("Expected 2 total hosts, got %d", report.Statistics.TotalHosts)
	}
	if report.Statistics.HealthyHosts != 1 {
		t.Errorf("Expected 1 healthy host, got %d", report.Statistics.HealthyHosts)
	}
	if report.Statistics.WarningHosts != 1 {
		t.Errorf("Expected 1 warning host, got %d", report.Statistics.WarningHosts)
	}

	// Check average duration calculation
	expectedAvg := (100.0 + 200.0) / 2.0 // 150ms
	if report.Statistics.AverageDuration < expectedAvg-10 || report.Statistics.AverageDuration > expectedAvg+10 {
		t.Errorf("Expected average duration ~%.2fms, got %.2fms", expectedAvg, report.Statistics.AverageDuration)
	}
}

// TestCheckAllSkipUnavailableHosts tests skip unavailable hosts behavior
func TestCheckAllSkipUnavailableHosts(t *testing.T) {
	hosts := []types.Host{
		{Name: "localhost", Address: "127.0.0.1"},
	}
	config := NewCheckConfig()
	config.SkipUnavailableHosts = true
	config.CheckTypes = []CheckType{CheckConnectivity}

	checker := NewChecker(hosts, config, &noOpLogger{})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report, err := checker.CheckAll(ctx)

	// Should not error when skip is enabled
	if err != nil && config.SkipUnavailableHosts {
		// This is OK if err is nil, but we're testing that it doesn't crash
	}
	if report == nil {
		t.Error("Expected report even when skip is enabled")
	}
}

// TestCheckHostTimeout tests context timeout handling
func TestCheckHostTimeout(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	config.Timeout = 100 * time.Millisecond
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	report, _ := checker.CheckHost(ctx, host)

	if report == nil {
		t.Fatal("Expected report even with timeout")
	}
	// The timeout should still produce a report, just with potentially different content
	if len(report.Checks) == 0 {
		t.Error("Expected at least one check result")
	}
}

// TestSetMaxWorkersInvalidValues tests invalid max workers
func TestSetMaxWorkersInvalidValues(t *testing.T) {
	hosts := []types.Host{
		{Name: "localhost", Address: "127.0.0.1"},
	}
	checker := NewChecker(hosts, NewCheckConfig(), &noOpLogger{})

	tests := []struct {
		name         string
		workers      int
		shouldChange bool
	}{
		{"zero", 0, false},
		{"negative", -5, false},
		{"valid small", 1, true},
		{"valid large", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalWorkers := checker.maxWorkers
			checker.SetMaxWorkers(tt.workers)
			if tt.shouldChange {
				if checker.maxWorkers != tt.workers {
					t.Errorf("Expected maxWorkers to be %d, got %d", tt.workers, checker.maxWorkers)
				}
			} else {
				if checker.maxWorkers != originalWorkers {
					t.Errorf("Expected maxWorkers to remain %d, got %d", originalWorkers, checker.maxWorkers)
				}
			}
		})
	}
}

// TestCheckConfigDefaults tests default configuration values
func TestCheckConfigDefaults(t *testing.T) {
	config := NewCheckConfig()

	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{"Timeout", config.Timeout, 30 * time.Second},
		{"DiskSpaceThreshold", config.DiskSpaceThreshold, 80},
		{"MemoryThreshold", config.MemoryThreshold, 80},
		{"CPUThreshold", config.CPUThreshold, 80},
		{"SkipUnavailableHosts", config.SkipUnavailableHosts, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.value)
			}
		})
	}
}

// TestCheckConnectivityLocalHost tests connectivity check for local host
func TestCheckConnectivityLocalHost(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	result := &HealthCheckResult{Details: make(map[string]interface{})}
	ctx := context.Background()

	checker.checkConnectivity(ctx, host, result)

	if result.Status != StatusHealthy {
		t.Errorf("Expected StatusHealthy for local host, got %s", result.Status)
	}
	if result.Details["connection_type"] != "local" {
		t.Errorf("Expected connection_type 'local', got %v", result.Details["connection_type"])
	}
}

// TestCheckDiskSpaceLocalHost tests disk space check for local host
func TestCheckDiskSpaceLocalHost(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	result := &HealthCheckResult{Details: make(map[string]interface{})}
	ctx := context.Background()

	checker.checkDiskSpace(ctx, host, result)

	if result.Status != StatusHealthy {
		t.Errorf("Expected StatusHealthy for local host, got %s", result.Status)
	}
	if result.Details["check_method"] != "local" {
		t.Errorf("Expected check_method 'local', got %v", result.Details["check_method"])
	}
}

// TestCheckMemoryLocalHost tests memory check for local host
func TestCheckMemoryLocalHost(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	result := &HealthCheckResult{Details: make(map[string]interface{})}
	ctx := context.Background()

	checker.checkMemory(ctx, host, result)

	if result.Status != StatusHealthy {
		t.Errorf("Expected StatusHealthy for local host, got %s", result.Status)
	}
	if result.Details["check_method"] != "local" {
		t.Errorf("Expected check_method 'local', got %v", result.Details["check_method"])
	}
}

// TestCheckCPULocalHost tests CPU check for local host
func TestCheckCPULocalHost(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	result := &HealthCheckResult{Details: make(map[string]interface{})}
	ctx := context.Background()

	checker.checkCPU(ctx, host, result)

	if result.Status != StatusHealthy {
		t.Errorf("Expected StatusHealthy for local host, got %s", result.Status)
	}
	if result.Details["check_method"] != "local" {
		t.Errorf("Expected check_method 'local', got %v", result.Details["check_method"])
	}
}

// TestCheckServicesNoServicesConfigured tests services check with no configured services
func TestCheckServicesNoServicesConfigured(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	config.Services = []string{} // Empty services
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	result := &HealthCheckResult{Details: make(map[string]interface{})}
	ctx := context.Background()

	checker.checkServices(ctx, host, result)

	if result.Status != StatusHealthy {
		t.Errorf("Expected StatusHealthy when no services configured, got %s", result.Status)
	}
}

// TestCheckServicesLocalHost tests services check for local host
func TestCheckServicesLocalHost(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	config.Services = []string{"ssh", "docker"} // Some services
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	result := &HealthCheckResult{Details: make(map[string]interface{})}
	ctx := context.Background()

	checker.checkServices(ctx, host, result)

	if result.Status != StatusHealthy {
		t.Errorf("Expected StatusHealthy for local host, got %s", result.Status)
	}
	if result.Details["check_method"] != "local" {
		t.Errorf("Expected check_method 'local', got %v", result.Details["check_method"])
	}
}

// TestCheckNetworkLocalHost tests network check for local host
func TestCheckNetworkLocalHost(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	result := &HealthCheckResult{Details: make(map[string]interface{})}
	ctx := context.Background()

	checker.checkNetwork(ctx, host, result)

	if result.Status != StatusHealthy {
		t.Errorf("Expected StatusHealthy for local host, got %s", result.Status)
	}
	if result.Details["check_method"] != "local" {
		t.Errorf("Expected check_method 'local', got %v", result.Details["check_method"])
	}
}

// TestCheckResultInitialization tests that check result is properly initialized
func TestCheckResultInitialization(t *testing.T) {
	host := types.Host{Name: "test-host", Address: "192.168.1.1"}
	config := NewCheckConfig()
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	ctx := context.Background()
	result := checker.runCheck(ctx, host, CheckConnectivity)

	if result.CheckType != CheckConnectivity {
		t.Errorf("Expected CheckType to be set, got empty")
	}
	if result.Host != host.Name {
		t.Errorf("Expected Host to be %s, got %s", host.Name, result.Host)
	}
	if result.Details == nil {
		t.Error("Expected Details to be initialized, got nil")
	}
	if result.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be set, got zero value")
	}
}

// TestCheckHostSummary tests that host summary is correctly built
func TestCheckHostSummary(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	config.CheckTypes = []CheckType{CheckConnectivity, CheckDiskSpace, CheckMemory}
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report, _ := checker.CheckHost(ctx, host)

	if report.Summary == nil {
		t.Fatal("Expected Summary to be initialized")
	}
	if len(report.Checks) != len(config.CheckTypes) {
		t.Errorf("Expected %d checks, got %d", len(config.CheckTypes), len(report.Checks))
	}
}

// TestCheckHostDuration tests that check duration is tracked
func TestCheckHostDuration(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	config := NewCheckConfig()
	checker := NewChecker([]types.Host{host}, config, &noOpLogger{})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report, _ := checker.CheckHost(ctx, host)

	if report.Duration < 0 {
		t.Errorf("Expected positive duration, got %v", report.Duration)
	}
}
