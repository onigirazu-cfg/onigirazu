package healthcheck

import (
	"context"
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
		t.Error("NewChecker returned nil")
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
		t.Error("NewCheckConfig returned nil")
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
		t.Error("CheckAll returned nil report")
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
		t.Error("CheckHost returned nil report")
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
		t.Error("buildReport returned nil")
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

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	report, _ := checker.CheckAll(ctx)

	// Should handle the context cancellation gracefully
	if report == nil {
		t.Error("CheckAll returned nil report even with cancelled context")
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
