package profiler

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"
)

// ProfileManager handles CPU and memory profiling for Onigirazu execution
type ProfileManager struct {
	cpuFile      *os.File
	memFile      *os.File
	traceFile    *os.File
	outputDir    string
	enabled      bool
	startTime    time.Time
	cpuProfile   bool
	memProfile   bool
	traceProfile bool
}

// Config holds profiling configuration
type Config struct {
	Enabled   bool
	OutputDir string
	CPU       bool
	Memory    bool
	Trace     bool
}

// NewProfileManager creates a new ProfileManager with the given configuration
func NewProfileManager(config Config) (*ProfileManager, error) {
	if !config.Enabled {
		return &ProfileManager{
			enabled: false,
		}, nil
	}

	// Create output directory if it doesn't exist
	if config.OutputDir == "" {
		config.OutputDir = "."
	}

	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create profiling directory: %w", err)
	}

	pm := &ProfileManager{
		outputDir:    config.OutputDir,
		enabled:      true,
		startTime:    time.Now(),
		cpuProfile:   config.CPU,
		memProfile:   config.Memory,
		traceProfile: config.Trace,
	}

	return pm, nil
}

// StartProfiling starts CPU and memory profiling
func (pm *ProfileManager) StartProfiling() error {
	if !pm.enabled {
		return nil
	}

	// Start CPU profiling
	if pm.cpuProfile {
		cpuPath := filepath.Join(pm.outputDir, fmt.Sprintf("cpu_%d.prof", time.Now().Unix()))
		f, err := os.Create(cpuPath)
		if err != nil {
			return fmt.Errorf("failed to create CPU profile file: %w", err)
		}
		pm.cpuFile = f

		if err := pprof.StartCPUProfile(f); err != nil {
			f.Close()
			return fmt.Errorf("failed to start CPU profile: %w", err)
		}
	}

	// Start trace profiling
	if pm.traceProfile {
		tracePath := filepath.Join(pm.outputDir, fmt.Sprintf("trace_%d.out", time.Now().Unix()))
		f, err := os.Create(tracePath)
		if err != nil {
			return fmt.Errorf("failed to create trace file: %w", err)
		}
		pm.traceFile = f

		if err := trace.Start(f); err != nil {
			f.Close()
			return fmt.Errorf("failed to start trace: %w", err)
		}
	}

	return nil
}

// StopProfiling stops CPU and memory profiling and writes the results
func (pm *ProfileManager) StopProfiling() error {
	if !pm.enabled {
		return nil
	}

	var errors []error

	// Stop CPU profiling
	if pm.cpuFile != nil {
		pprof.StopCPUProfile()
		pm.cpuFile.Close()
	}

	// Stop trace profiling
	if pm.traceFile != nil {
		trace.Stop()
		pm.traceFile.Close()
	}

	// Write memory profile if enabled
	if pm.memProfile {
		memPath := filepath.Join(pm.outputDir, fmt.Sprintf("mem_%d.prof", time.Now().Unix()))
		f, err := os.Create(memPath)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to create memory profile file: %w", err))
		} else {
			defer f.Close()
			runtime.GC() // Force garbage collection to get accurate memory stats
			if err := pprof.WriteHeapProfile(f); err != nil {
				errors = append(errors, fmt.Errorf("failed to write memory profile: %w", err))
			}
		}
	}

	// Write goroutine profile for debugging
	goroutinePath := filepath.Join(pm.outputDir, fmt.Sprintf("goroutine_%d.prof", time.Now().Unix()))
	f, err := os.Create(goroutinePath)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to create goroutine profile file: %w", err))
	} else {
		defer f.Close()
		if err := pprof.Lookup("goroutine").WriteTo(f, 0); err != nil {
			errors = append(errors, fmt.Errorf("failed to write goroutine profile: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during profiling cleanup: %v", errors)
	}

	return nil
}

// GenerateReport generates a profiling analysis report
func (pm *ProfileManager) GenerateReport() (string, error) {
	if !pm.enabled {
		return "", nil
	}

	duration := time.Since(pm.startTime)
	report := fmt.Sprintf("=== PROFILING REPORT ===\n")
	report += fmt.Sprintf("Duration: %v\n", duration)
	report += fmt.Sprintf("Output Directory: %s\n", pm.outputDir)
	report += fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339))

	if pm.cpuProfile && pm.cpuFile != nil {
		report += fmt.Sprintf("✅ CPU Profile: %s\n", pm.cpuFile.Name())
		report += "   Usage: go tool pprof <binary> <profile>\n"
		report += "   Example: go tool pprof ./bin/onigirazu cpu_*.prof\n\n"
	}

	if pm.memProfile {
		report += fmt.Sprintf("✅ Memory Profile: %s/mem_*.prof\n", pm.outputDir)
		report += "   Usage: go tool pprof <binary> <profile>\n"
		report += "   Example: go tool pprof ./bin/onigirazu mem_*.prof\n\n"
	}

	if pm.traceProfile && pm.traceFile != nil {
		report += fmt.Sprintf("✅ Trace Profile: %s\n", pm.traceFile.Name())
		report += "   Usage: go tool trace <profile>\n"
		report += "   Example: go tool trace trace_*.out\n\n"
	}

	report += fmt.Sprintf("✅ Goroutine Profile: %s/goroutine_*.prof\n", pm.outputDir)
	report += "   Usage: go tool pprof <binary> <profile>\n"
	report += "   Example: go tool pprof ./bin/onigirazu goroutine_*.prof\n\n"

	return report, nil
}

// IsEnabled returns whether profiling is enabled
func (pm *ProfileManager) IsEnabled() bool {
	return pm.enabled
}

// GetOutputDir returns the profiling output directory
func (pm *ProfileManager) GetOutputDir() string {
	return pm.outputDir
}

// GetStartTime returns when profiling started
func (pm *ProfileManager) GetStartTime() time.Time {
	return pm.startTime
}
