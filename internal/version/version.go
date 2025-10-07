package version

import (
	"fmt"
	"runtime"
)

// Build information. Populated at build-time via ldflags.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// BuildInfo contains build information
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetBuildInfo returns build information
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// GetVersion returns the version string
func GetVersion() string {
	if Version == "dev" {
		return "dev"
	}
	return Version
}

// GetFullVersion returns a simple version string
func GetFullVersion() string {
	if Version == "dev" {
		return "Onigirazu dev"
	}
	return fmt.Sprintf("Onigirazu %s", Version)
}
