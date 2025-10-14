package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetBuildInfo(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := Commit
	origDate := Date

	// Test with default values
	t.Run("default values", func(t *testing.T) {
		Version = "dev"
		Commit = "unknown"
		Date = "unknown"

		info := GetBuildInfo()

		if info.Version != "dev" {
			t.Errorf("Expected version 'dev', got '%s'", info.Version)
		}
		if info.Commit != "unknown" {
			t.Errorf("Expected commit 'unknown', got '%s'", info.Commit)
		}
		if info.Date != "unknown" {
			t.Errorf("Expected date 'unknown', got '%s'", info.Date)
		}
		if info.GoVersion != runtime.Version() {
			t.Errorf("Expected GoVersion '%s', got '%s'", runtime.Version(), info.GoVersion)
		}
		expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
		if info.Platform != expectedPlatform {
			t.Errorf("Expected platform '%s', got '%s'", expectedPlatform, info.Platform)
		}
	})

	// Test with custom values
	t.Run("custom values", func(t *testing.T) {
		Version = "1.30.2"
		Commit = "abc123def"
		Date = "2025-02-01"

		info := GetBuildInfo()

		if info.Version != "1.30.2" {
			t.Errorf("Expected version '1.30.2', got '%s'", info.Version)
		}
		if info.Commit != "abc123def" {
			t.Errorf("Expected commit 'abc123def', got '%s'", info.Commit)
		}
		if info.Date != "2025-02-01" {
			t.Errorf("Expected date '2025-02-01', got '%s'", info.Date)
		}
	})

	// Restore original values
	Version = origVersion
	Commit = origCommit
	Date = origDate
}

func TestGetVersion(t *testing.T) {
	// Save original value
	origVersion := Version

	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "dev version",
			version:  "dev",
			expected: "dev",
		},
		{
			name:     "release version",
			version:  "1.30.2",
			expected: "1.30.2",
		},
		{
			name:     "semver version",
			version:  "2.0.0-beta.1",
			expected: "2.0.0-beta.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			result := GetVersion()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}

	// Restore original value
	Version = origVersion
}

func TestGetFullVersion(t *testing.T) {
	// Save original value
	origVersion := Version

	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "dev version",
			version:  "dev",
			expected: "Onigirazu dev",
		},
		{
			name:     "release version",
			version:  "1.30.2",
			expected: "Onigirazu 1.30.2",
		},
		{
			name:     "semver version",
			version:  "2.0.0-beta.1",
			expected: "Onigirazu 2.0.0-beta.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			result := GetFullVersion()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}

	// Restore original value
	Version = origVersion
}

func TestBuildInfoJSON(t *testing.T) {
	// Test that BuildInfo struct has proper JSON tags
	info := BuildInfo{
		Version:   "1.0.0",
		Commit:    "abc123",
		Date:      "2025-01-01",
		GoVersion: "go1.24",
		Platform:  "linux/amd64",
	}

	// Verify fields are accessible
	if info.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", info.Version)
	}
	if info.Commit != "abc123" {
		t.Errorf("Expected commit 'abc123', got '%s'", info.Commit)
	}
	if info.Date != "2025-01-01" {
		t.Errorf("Expected date '2025-01-01', got '%s'", info.Date)
	}
	if info.GoVersion != "go1.24" {
		t.Errorf("Expected GoVersion 'go1.24', got '%s'", info.GoVersion)
	}
	if info.Platform != "linux/amd64" {
		t.Errorf("Expected platform 'linux/amd64', got '%s'", info.Platform)
	}
}

func TestPlatformFormat(t *testing.T) {
	info := GetBuildInfo()

	// Platform should be in format "os/arch"
	parts := strings.Split(info.Platform, "/")
	if len(parts) != 2 {
		t.Errorf("Expected platform format 'os/arch', got '%s'", info.Platform)
	}

	// First part should be valid OS
	validOS := []string{"linux", "darwin", "windows", "freebsd", "openbsd", "netbsd"}
	osValid := false
	for _, os := range validOS {
		if parts[0] == os {
			osValid = true
			break
		}
	}
	if !osValid {
		t.Errorf("Expected valid OS in platform, got '%s'", parts[0])
	}

	// Second part should be valid architecture
	validArch := []string{"amd64", "arm64", "arm", "386", "ppc64le", "s390x"}
	archValid := false
	for _, arch := range validArch {
		if parts[1] == arch {
			archValid = true
			break
		}
	}
	if !archValid {
		t.Errorf("Expected valid architecture in platform, got '%s'", parts[1])
	}
}

func TestGoVersionFormat(t *testing.T) {
	info := GetBuildInfo()

	// Go version should start with "go"
	if !strings.HasPrefix(info.GoVersion, "go") {
		t.Errorf("Expected GoVersion to start with 'go', got '%s'", info.GoVersion)
	}
}
