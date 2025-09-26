package modules

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// FactsModule implements system facts gathering
type FactsModule struct {
	BaseModule
}

// NewFactsModule creates a new facts module
func NewFactsModule() *FactsModule {
	return &FactsModule{
		BaseModule: BaseModule{
			name:        "facts",
			description: "Gather system facts and information",
		},
	}
}

// Execute gathers system facts
func (m *FactsModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "gather_facts",
		Host:      host.Name,
		Module:    m.name,
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Gather basic system facts
	facts := make(map[string]interface{})

	// Operating system information
	facts["os_family"] = runtime.GOOS
	facts["architecture"] = runtime.GOARCH
	facts["num_cpus"] = runtime.NumCPU()
	facts["go_version"] = runtime.Version()

	// Process information
	facts["pid"] = os.Getpid()
	facts["ppid"] = os.Getppid()

	// Environment variables (filtered)
	envVars := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			// Filter sensitive environment variables
			if !m.isSensitiveEnvVar(key) {
				envVars[key] = value
			}
		}
	}
	facts["environment"] = envVars

	// Working directory
	if wd, err := os.Getwd(); err == nil {
		facts["working_directory"] = wd
	}

	// Hostname
	if hostname, err := os.Hostname(); err == nil {
		facts["hostname"] = hostname
	}

	// User information
	if uid := os.Getuid(); uid >= 0 {
		facts["uid"] = uid
	}
	if gid := os.Getgid(); gid >= 0 {
		facts["gid"] = gid
	}

	// Memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	facts["memory"] = map[string]interface{}{
		"alloc":         memStats.Alloc,
		"total_alloc":   memStats.TotalAlloc,
		"sys":           memStats.Sys,
		"heap_alloc":    memStats.HeapAlloc,
		"heap_sys":      memStats.HeapSys,
		"heap_idle":     memStats.HeapIdle,
		"heap_inuse":    memStats.HeapInuse,
		"heap_released": memStats.HeapReleased,
		"heap_objects":  memStats.HeapObjects,
		"stack_inuse":   memStats.StackInuse,
		"stack_sys":     memStats.StackSys,
		"gc_runs":       memStats.NumGC,
		"gc_pause_ns":   memStats.PauseNs,
	}

	// Goroutine information
	facts["goroutines"] = runtime.NumGoroutine()

	// Platform-specific facts
	platformFacts := m.gatherPlatformFacts()
	for k, v := range platformFacts {
		facts[k] = v
	}

	// Network facts (basic)
	networkFacts := m.gatherNetworkFacts()
	facts["network"] = networkFacts

	// File system facts
	fsFacts := m.gatherFileSystemFacts()
	facts["filesystem"] = fsFacts

	// Custom facts from arguments
	if customFacts, exists := args["custom_facts"]; exists {
		if customMap, ok := customFacts.(map[string]interface{}); ok {
			facts["custom"] = customMap
		}
	}

	// Add timestamp
	facts["fact_timestamp"] = time.Now().Unix()
	facts["fact_time"] = time.Now().Format(time.RFC3339)

	result.Output["ansible_facts"] = facts
	result.Output["facts"] = facts
	result.Duration = time.Since(startTime)

	return result, nil
}

// Validate validates facts module arguments
func (m *FactsModule) Validate(args map[string]interface{}) error {
	// Facts module accepts optional custom_facts parameter
	if customFacts, exists := args["custom_facts"]; exists {
		if _, ok := customFacts.(map[string]interface{}); !ok {
			return fmt.Errorf("custom_facts must be a map")
		}
	}

	return nil
}

// isSensitiveEnvVar checks if an environment variable is sensitive
func (m *FactsModule) isSensitiveEnvVar(key string) bool {
	sensitiveKeys := []string{
		"PASSWORD", "PASSWD", "SECRET", "TOKEN", "KEY", "PRIVATE",
		"API_KEY", "AUTH", "CREDENTIAL", "CERT", "SSH_KEY",
	}

	upperKey := strings.ToUpper(key)
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(upperKey, sensitive) {
			return true
		}
	}

	return false
}

// gatherPlatformFacts gathers platform-specific facts
func (m *FactsModule) gatherPlatformFacts() map[string]interface{} {
	facts := make(map[string]interface{})

	switch runtime.GOOS {
	case "darwin":
		facts["platform"] = "Darwin"
		facts["os_family"] = "Darwin"
		// Add macOS-specific facts
		if version := m.getMacOSVersion(); version != "" {
			facts["os_version"] = version
		}

	case "linux":
		facts["platform"] = "Linux"
		facts["os_family"] = "RedHat" // Default, could be detected more precisely
		// Add Linux-specific facts
		if distro := m.getLinuxDistribution(); distro != "" {
			facts["distribution"] = distro
		}

	case "windows":
		facts["platform"] = "Win32NT"
		facts["os_family"] = "Windows"
		// Add Windows-specific facts

	default:
		facts["platform"] = runtime.GOOS
	}

	return facts
}

// gatherNetworkFacts gathers basic network information
func (m *FactsModule) gatherNetworkFacts() map[string]interface{} {
	facts := make(map[string]interface{})

	// This is a simplified implementation
	// In a real implementation, you would use network libraries to get:
	// - Network interfaces
	// - IP addresses
	// - Routing table
	// - DNS configuration

	facts["interfaces"] = []string{"lo0", "en0"} // Placeholder
	facts["default_ipv4"] = map[string]interface{}{
		"address": "127.0.0.1",
		"gateway": "192.168.1.1",
	}

	return facts
}

// gatherFileSystemFacts gathers file system information
func (m *FactsModule) gatherFileSystemFacts() map[string]interface{} {
	facts := make(map[string]interface{})

	// This is a simplified implementation
	// In a real implementation, you would gather:
	// - Mounted file systems
	// - Disk usage
	// - File system types

	facts["mounts"] = []map[string]interface{}{
		{
			"mount":  "/",
			"device": "/dev/disk1s1",
			"fstype": "apfs",
			"size":   "500GB",
		},
	}

	return facts
}

// getMacOSVersion gets macOS version (simplified)
func (m *FactsModule) getMacOSVersion() string {
	// This would typically read from system files or use system calls
	return "14.0" // Placeholder
}

// getLinuxDistribution gets Linux distribution (simplified)
func (m *FactsModule) getLinuxDistribution() string {
	// This would typically read from /etc/os-release or similar
	return "Ubuntu" // Placeholder
}

// GetSystemLoad returns current system load information
func (m *FactsModule) GetSystemLoad() map[string]interface{} {
	load := make(map[string]interface{})

	// Get memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	load["memory_usage"] = memStats.Alloc
	load["goroutines"] = runtime.NumGoroutine()
	load["gc_runs"] = memStats.NumGC

	return load
}

// GetProcessInfo returns information about the current process
func (m *FactsModule) GetProcessInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["pid"] = os.Getpid()
	info["ppid"] = os.Getppid()
	info["uid"] = os.Getuid()
	info["gid"] = os.Getgid()

	if wd, err := os.Getwd(); err == nil {
		info["working_directory"] = wd
	}

	return info
}

// GetRuntimeInfo returns Go runtime information
func (m *FactsModule) GetRuntimeInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["go_version"] = runtime.Version()
	info["go_os"] = runtime.GOOS
	info["go_arch"] = runtime.GOARCH
	info["num_cpu"] = runtime.NumCPU()
	info["num_goroutine"] = runtime.NumGoroutine()

	return info
}

// GetEnvironmentInfo returns filtered environment information
func (m *FactsModule) GetEnvironmentInfo() map[string]interface{} {
	info := make(map[string]interface{})
	envVars := make(map[string]string)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			if !m.isSensitiveEnvVar(key) {
				envVars[key] = value
			}
		}
	}

	info["variables"] = envVars

	if hostname, err := os.Hostname(); err == nil {
		info["hostname"] = hostname
	}

	return info
}

// GetMemoryInfo returns detailed memory information
func (m *FactsModule) GetMemoryInfo() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"alloc":           memStats.Alloc,
		"total_alloc":     memStats.TotalAlloc,
		"sys":             memStats.Sys,
		"heap_alloc":      memStats.HeapAlloc,
		"heap_sys":        memStats.HeapSys,
		"heap_idle":       memStats.HeapIdle,
		"heap_inuse":      memStats.HeapInuse,
		"heap_released":   memStats.HeapReleased,
		"heap_objects":    memStats.HeapObjects,
		"stack_inuse":     memStats.StackInuse,
		"stack_sys":       memStats.StackSys,
		"gc_runs":         memStats.NumGC,
		"gc_pause_total":  memStats.PauseTotalNs,
		"gc_pause_recent": memStats.PauseNs[(memStats.NumGC+255)%256],
		"mallocs":         memStats.Mallocs,
		"frees":           memStats.Frees,
		"lookups":         memStats.Lookups,
	}
}
