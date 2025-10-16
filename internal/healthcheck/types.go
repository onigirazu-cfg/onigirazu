package healthcheck

import (
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	StatusHealthy  HealthStatus = "healthy"
	StatusWarning  HealthStatus = "warning"
	StatusCritical HealthStatus = "critical"
	StatusUnknown  HealthStatus = "unknown"
)

// CheckType represents the type of health check
type CheckType string

const (
	CheckConnectivity CheckType = "connectivity"
	CheckDiskSpace    CheckType = "disk_space"
	CheckMemory       CheckType = "memory"
	CheckCPU          CheckType = "cpu"
	CheckServices     CheckType = "services"
	CheckNetwork      CheckType = "network"
	CheckCertificates CheckType = "certificates"
)

// HealthCheckResult contains the result of a single health check
type HealthCheckResult struct {
	CheckType CheckType              `json:"check_type"`
	Host      string                 `json:"host"`
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
	Error     string                 `json:"error,omitempty"`
}

// HostHealthReport contains the health report for a single host
type HostHealthReport struct {
	Host      types.Host          `json:"host"`
	Status    HealthStatus        `json:"status"`
	Checks    []HealthCheckResult `json:"checks"`
	Summary   map[string]int      `json:"summary"` // count of each status
	Timestamp time.Time           `json:"timestamp"`
	Duration  time.Duration       `json:"duration"`
}

// HealthCheckReport contains the overall health report
type HealthCheckReport struct {
	Timestamp     time.Time             `json:"timestamp"`
	OverallStatus HealthStatus          `json:"overall_status"`
	HostReports   []HostHealthReport    `json:"host_reports"`
	Summary       map[string]int        `json:"summary"` // count of each status
	TotalDuration time.Duration         `json:"total_duration"`
	Statistics    HealthCheckStatistics `json:"statistics"`
}

// HealthCheckStatistics contains statistics about the health checks
type HealthCheckStatistics struct {
	TotalHosts      int               `json:"total_hosts"`
	HealthyHosts    int               `json:"healthy_hosts"`
	WarningHosts    int               `json:"warning_hosts"`
	CriticalHosts   int               `json:"critical_hosts"`
	UnknownHosts    int               `json:"unknown_hosts"`
	ChecksPerHost   map[CheckType]int `json:"checks_per_host"`
	AverageDuration float64           `json:"average_duration_ms"`
}

// CheckConfig contains configuration for health checks
type CheckConfig struct {
	CheckTypes           []CheckType   `json:"check_types"`
	Timeout              time.Duration `json:"timeout"`
	DiskSpaceThreshold   int           `json:"disk_space_threshold"` // percentage
	MemoryThreshold      int           `json:"memory_threshold"`     // percentage
	CPUThreshold         int           `json:"cpu_threshold"`        // percentage
	SkipUnavailableHosts bool          `json:"skip_unavailable_hosts"`
	Services             []string      `json:"services"` // services to check
}

// NewCheckConfig creates a new check configuration with defaults
func NewCheckConfig() *CheckConfig {
	return &CheckConfig{
		CheckTypes: []CheckType{
			CheckConnectivity,
			CheckDiskSpace,
			CheckMemory,
		},
		Timeout:              30 * time.Second,
		DiskSpaceThreshold:   80,
		MemoryThreshold:      80,
		CPUThreshold:         80,
		SkipUnavailableHosts: false,
	}
}
